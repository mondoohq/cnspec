// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package generate

import (
	"strings"

	"github.com/cockroachdb/errors"
	"go.mondoo.com/mql/v13"
	"go.mondoo.com/mql/v13/mqlc"
	"go.mondoo.com/mql/v13/providers"
	"go.mondoo.com/mql/v13/providers-sdk/v1/resources"
)

// Validator checks a generated MQL string. Compilation is the guaranteed gate;
// richer checks (execute-and-assert against a fixture) can implement the same
// interface. A nil error means the query passed.
type Validator interface {
	Validate(mql string) error
}

// compileValidator validates by compiling the query in-process against the
// provider schema, exactly as `cnspec run --ast` and the bundle linter do.
// Compilation is necessary but not sufficient — it catches unknown resources,
// unknown fields, and syntax errors, but not the semantic traps documented in
// cnspec/CLAUDE.md (null three-valued logic, dotted-path husks). Those are the
// job of the execute-and-assert gate and expert review.
type compileValidator struct {
	schema   resources.ResourcesSchema
	features mql.Features
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

func (v *compileValidator) Validate(query string) error {
	query = strings.TrimSpace(query)
	if query == "" {
		return errors.New("empty query")
	}
	_, err := mqlc.Compile(query, nil, mqlc.NewConfig(v.schema, v.features))
	if err != nil {
		return errors.Wrap(err, "MQL does not compile")
	}
	return nil
}

// NoopValidator accepts every query. Used when validation is explicitly disabled
// or a schema cannot be loaded; callers should warn when falling back to it.
type NoopValidator struct{}

func (NoopValidator) Validate(string) error { return nil }

// QueryRunner executes an MQL query against a target asset and reports whether it
// resolved to a concrete boolean verdict.
type QueryRunner interface {
	// Run executes the query. resolved is false when the query ran but produced
	// null (data did not resolve) rather than a true/false verdict; err is set
	// when execution itself failed.
	Run(query string) (value bool, resolved bool, err error)
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

func (v *executeValidator) Validate(query string) error {
	if err := v.compile.Validate(query); err != nil {
		return err
	}
	_, resolved, err := v.runner.Run(query)
	if err != nil {
		return errors.Wrap(err, "query failed to execute against the test target")
	}
	if !resolved {
		return errors.New("query executed but did not resolve to a concrete true/false verdict (result was null); likely a null-unsafe access, an unresolved field, or a dotted path that compiles to an empty resource")
	}
	return nil
}
