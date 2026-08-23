// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package generate

import (
	"strings"
	"sync"

	"github.com/cockroachdb/errors"
	"go.mondoo.com/mql"
	"go.mondoo.com/mql/llx"
	"go.mondoo.com/mql/mqlc"
	"go.mondoo.com/mql/providers"
	"go.mondoo.com/mql/providers-sdk/v1/resources"
	"go.mondoo.com/mql/types"
)

// ValidationRequest is one query to check, with the context the compiler needs
// to judge it. It exists because a query is not self-contained: `props.<name>`
// only compiles when the validator is told which props the check declares — the
// prompt asks the agent to use props, so a validator that cannot see them
// rejects the correct answer.
type ValidationRequest struct {
	// MQL is the query to validate.
	MQL string
	// Props are the check's parameterized properties. Without them every
	// props.<name> reference is an unknown symbol.
	Props []Prop
	// Provider is the target provider, used to report honestly when its schema
	// is not installed rather than blaming the query for an unknown resource.
	Provider string
}

// Validator checks a generated MQL query. Compilation is the guaranteed gate;
// richer checks (execute-and-assert against a fixture) implement the same
// interface. A nil error means the query passed.
type Validator interface {
	Validate(req ValidationRequest) error
}

// ProviderChecker is the optional half of a Validator: it reports up front
// whether it can validate queries for a provider at all. The generator asks
// before it spends an agent invocation, because "the aws provider is not
// installed" must not be discovered three agent calls later disguised as three
// compile errors about an unknown resource named `aws`.
type ProviderChecker interface {
	// CheckProvider returns nil when queries for this provider can be validated.
	// An empty provider means "unknown", which is never an error.
	CheckProvider(provider string) error
}

// ErrValidationUnavailable marks the errors that mean "this query was not
// judged", as opposed to "this query is wrong". Callers use errors.Is to tell
// the two apart, because retrying the agent cannot fix the first kind.
var ErrValidationUnavailable = errors.New("MQL validation is unavailable")

// compileValidator validates by compiling the query in-process against the
// provider schema, exactly as `cnspec run --ast` and the bundle linter do.
// Compilation is necessary but not sufficient — it catches unknown resources,
// unknown fields, and syntax errors, but not the semantic traps documented in
// cnspec/CLAUDE.md (null three-valued logic, dotted-path husks). Those are the
// job of the execute-and-assert gate and expert review.
type compileValidator struct {
	schema   resources.ResourcesSchema
	features mql.Features

	// installed is the set of provider names with a locally installed schema,
	// resolved once and reused. The compiler cannot distinguish "you named a
	// resource that does not exist" from "the provider that defines it is not
	// installed", so we answer that question before compiling.
	installedOnce sync.Once
	installed     map[string]bool
	installedErr  error
}

// NewCompileValidator builds a validator against the default runtime schema
// (every locally installed provider plus core). It mirrors the linter's
// compiler configuration so a query that passes here passes lint.
func NewCompileValidator() (Validator, error) {
	runtime := providers.DefaultRuntime()
	if runtime == nil {
		return nil, errors.New("no default runtime available for validation")
	}
	schema := runtime.Schema()
	if schema == nil {
		return nil, errors.New("no schema available for validation")
	}
	features := mql.DefaultFeatures
	features = append(features, byte(mql.FailIfNoEntryPoints))
	return &compileValidator{schema: schema, features: features}, nil
}

func (v *compileValidator) config() mqlc.CompilerConfig {
	return mqlc.NewConfig(v.schema, v.features)
}

// CheckProvider reports whether this validator can resolve resources for the
// named provider. Constructing the validator cannot answer this: the default
// runtime always hands out a schema, even one holding nothing but core, so on a
// machine with no providers installed the validator builds happily and then
// rejects every query with "cannot find resource for identifier 'aws'" — an
// error that reads as a defect in the generated query rather than a missing
// install.
func (v *compileValidator) CheckProvider(provider string) error {
	provider = strings.TrimSpace(provider)
	if provider == "" {
		return nil // no target resolved; nothing to promise either way
	}

	v.installedOnce.Do(func() {
		v.installed, v.installedErr = installedProviderNames()
	})
	if v.installedErr != nil {
		return errors.Mark(
			errors.Wrapf(v.installedErr, "cannot tell whether the %q provider is installed, so generated MQL cannot be compile-checked", provider),
			ErrValidationUnavailable)
	}
	if v.installed[provider] {
		return nil
	}
	return errors.Mark(
		errors.Newf("the %q provider is not installed, so its resources cannot be compile-checked; install it with `cnspec providers install %s`, or re-run with --no-validate to generate without validation", provider, provider),
		ErrValidationUnavailable)
}

func (v *compileValidator) Validate(req ValidationRequest) error {
	query := strings.TrimSpace(req.MQL)
	if query == "" {
		return errors.New("empty query")
	}
	if err := v.CheckProvider(req.Provider); err != nil {
		return err
	}

	conf := v.config()
	bundle, err := mqlc.Compile(query, newPropsHandler(req.Props, conf), conf)
	if err != nil {
		return errors.Wrap(err, "MQL does not compile")
	}
	return assertVerdict(bundle)
}

// installedProviderNames returns the provider names available locally, builtin
// ones included.
func installedProviderNames() (map[string]bool, error) {
	names := map[string]bool{}
	for _, name := range providers.GetBuiltinProviderNames() {
		names[name] = true
	}
	active, err := providers.ListActive()
	if err != nil {
		return nil, err
	}
	for _, p := range active {
		if p != nil && p.Provider != nil {
			names[p.Name] = true
		}
	}
	return names, nil
}

// assertVerdict rejects a query that compiles but does not answer pass/fail. A
// scored check has to produce a verdict, and the cheapest way for generation to
// go wrong is a truncated answer: `aws.s3.buckets` instead of
// `aws.s3.buckets.all(...)` compiles, scores off whatever boolean fields the
// resource happens to expose, and reads as a working check.
//
// Measured against every compiling check in content/ (1374 boolean, 135 block,
// 44 list-of-block, 2 score): all 1555 pass this gate, while the truncation
// shapes (`aws.s3.buckets`, `os.hostname`, `sshd.config`) are rejected.
func assertVerdict(bundle *llx.CodeBundle) error {
	if bundle == nil || bundle.CodeV2 == nil {
		return errors.New("MQL compiled to no executable code")
	}
	code := bundle.CodeV2
	entrypoints := code.Entrypoints()
	if len(entrypoints) == 0 {
		return errors.New("MQL produces no result to score")
	}

	// The last entrypoint is the query's verdict, the same convention the
	// execute gate uses to read a result.
	last := code.Chunk(entrypoints[len(entrypoints)-1])
	if last == nil {
		return errors.New("MQL produces no result to score")
	}
	resultType := types.Type(last.DereferencedTypeV2(code))

	switch {
	case resultType == types.Bool:
		return nil
	case resultType == types.Score:
		// the `switch { case ...: score(N) }` form scores itself
		return nil
	case resultType == types.Block || (resultType.IsArray() && resultType.Child() == types.Block):
		// A block scores on the assertions inside it. A block that only reads
		// fields is a data query, not a check — and a bare resource compiles
		// into exactly that shape, since the compiler fills the block with the
		// resource's default fields.
		if blockAsserts(code) {
			return nil
		}
		return errors.New("MQL returns a block of field values, not a pass/fail verdict; assert on the fields (e.g. `resource { field == \"x\" }`) instead of listing them")
	default:
		return errors.Newf("MQL returns %s, not a pass/fail verdict; a scored check must evaluate to true (compliant) or false", resultType.Label())
	}
}

// blockAsserts reports whether any block entrypoint is a boolean produced by an
// assertion — a comparison, a logical operator, or a list predicate — rather
// than a plain field read. Every block-form check in content/ has one; a bare
// resource has only field reads, even when some of those fields are booleans.
func blockAsserts(code *llx.CodeV2) bool {
	for i := range code.Blocks {
		for _, ref := range code.Blocks[i].Entrypoints {
			chunk := code.Chunk(ref)
			if chunk == nil {
				continue
			}
			t := types.Type(chunk.DereferencedTypeV2(code))
			if !isBoolish(t) {
				continue
			}
			if isAssertionID(chunk.Id) {
				return true
			}
		}
	}
	return false
}

// isBoolish reports whether a result is a verdict or a list of verdicts.
func isBoolish(t types.Type) bool {
	return t == types.Bool || (t.IsArray() && t.Child() == types.Bool)
}

// isAssertionID reports whether a chunk id names an assertion rather than a
// field. Operators are punctuation (`==`, `!=`, `&&`) and the compiler's list
// predicates are `$`-prefixed (`$all`, `$any`, `$none`); a field read is a plain
// identifier.
func isAssertionID(id string) bool {
	if id == "" {
		return false
	}
	if strings.HasPrefix(id, "$") {
		return true
	}
	for _, r := range id {
		isIdentifierRune := r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
		if !isIdentifierRune {
			return true
		}
	}
	return false
}

// NoopValidator accepts every query. Used when validation is explicitly disabled
// or a schema cannot be loaded; callers should warn when falling back to it.
type NoopValidator struct{}

func (NoopValidator) Validate(ValidationRequest) error { return nil }

// QueryRunner executes an MQL query against a target asset and reports whether it
// resolved to a concrete boolean verdict.
type QueryRunner interface {
	// Run executes the query. resolved is false when the query ran but produced
	// null (data did not resolve) rather than a true/false verdict; err is set
	// when execution itself failed.
	Run(req ValidationRequest) (value bool, resolved bool, err error)
}

// executeValidator is the strongest gate: it compiles the query AND runs it
// against a real target, requiring a concrete true/false verdict. This is what
// catches the semantic traps a compile check cannot — a query that resolves to
// null (a null-unsafe access, an unresolved field, or a dotted path that
// compiles to an empty resource husk) is rejected here even though it compiled.
type executeValidator struct {
	compile Validator
	runner  QueryRunner
}

// NewExecuteValidator wraps a compile validator with execution against runner. If
// compile is nil, a standard compile validator is created.
func NewExecuteValidator(runner QueryRunner, compile Validator) (Validator, error) {
	if runner == nil {
		return nil, errors.New("execute validation requires a query runner")
	}
	if compile == nil {
		var err error
		compile, err = NewCompileValidator()
		if err != nil {
			return nil, err
		}
	}
	return &executeValidator{compile: compile, runner: runner}, nil
}

// CheckProvider defers to the compile stage, so a missing provider is still
// reported before an agent call is spent.
func (v *executeValidator) CheckProvider(provider string) error {
	if c, ok := v.compile.(ProviderChecker); ok {
		return c.CheckProvider(provider)
	}
	return nil
}

func (v *executeValidator) Validate(req ValidationRequest) error {
	if err := v.compile.Validate(req); err != nil {
		return err
	}
	_, resolved, err := v.runner.Run(req)
	if err != nil {
		return errors.Wrap(err, "query failed to execute against the test target")
	}
	if !resolved {
		return errors.New("query executed but did not resolve to a concrete true/false verdict (result was null); likely a null-unsafe access, an unresolved field, or a dotted path that compiles to an empty resource")
	}
	return nil
}
