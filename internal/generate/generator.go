// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package generate

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
)

// Config configures a Generator. Only Backend is required; the rest have safe
// defaults.
type Config struct {
	// Backend is the coding-agent CLI used for generation (required).
	Backend AgentBackend
	// Corpus provides grounding examples. Nil disables few-shot grounding.
	Corpus *Corpus
	// Validator gates generated MQL. Nil means no validation (the caller should
	// warn); use NewCompileValidator for the standard gate.
	Validator Validator
	// Model optionally overrides the agent's model.
	Model string
	// Force regenerates checks that already have MQL.
	Force bool
	// Explain requests a per-check explanation from the agent.
	Explain bool
	// SkillPaths point the agent at any discovered skill files (mql, policy-graph).
	SkillPaths []string
	// MaxAttempts bounds validation retries (total attempts, min 1). Default 3.
	MaxAttempts int
	// ExampleLimit caps grounding examples per check. Default 5.
	ExampleLimit int
	// Timeout bounds a single agent invocation. Default 180s.
	Timeout time.Duration
}

// Generator produces MQL for policy checks by driving an agent backend, grounded
// in similar existing checks and gated by validation.
type Generator struct {
	cfg Config
}

// New builds a Generator, applying defaults for unset fields.
func New(cfg Config) (*Generator, error) {
	if cfg.Backend == nil {
		return nil, errors.New("a generation backend is required")
	}
	if cfg.MaxAttempts < 1 {
		cfg.MaxAttempts = 3
	}
	if cfg.ExampleLimit <= 0 {
		cfg.ExampleLimit = 5
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 180 * time.Second
	}
	return &Generator{cfg: cfg}, nil
}

// ErrNoValidator reports that nothing checked a query. It is returned instead of
// nil so a caller cannot mistake "no validator is configured" for "this query is
// valid" — the interactive flow accepts hand-edited MQL on the strength of this
// call, and silently accepting it unchecked is the failure it is meant to catch.
// Callers that legitimately run without validation test for it with errors.Is.
var ErrNoValidator = errors.Mark(errors.New("no validator is configured; the query was not checked"), ErrValidationUnavailable)

// Validate runs the configured validator against a query (e.g. MQL a user edited
// by hand in the interactive flow), with the check's props so `props.<name>`
// resolves. It returns ErrNoValidator when no validator is configured.
func (g *Generator) Validate(query string, props ...Prop) error {
	if g.cfg.Validator == nil {
		return ErrNoValidator
	}
	return g.cfg.Validator.Validate(ValidationRequest{MQL: query, Props: props})
}

// Ground returns up to n grounding examples for an intent+provider, or nil when
// no corpus is configured. Used by the interactive flow to preview precedents.
func (g *Generator) Ground(intent, provider string, n int) []Example {
	if g.cfg.Corpus == nil {
		return nil
	}
	return g.cfg.Corpus.Search(intent, provider, n)
}

// Generate processes a batch of checks in order, invoking progress (if non-nil)
// as each result is ready so the CLI can render live status.
func (g *Generator) Generate(ctx context.Context, checks []Check, progress func(Result)) []Result {
	results := make([]Result, 0, len(checks))
	report := func(res Result) {
		results = append(results, res)
		if progress != nil {
			progress(res)
		}
	}
	for i, c := range checks {
		if err := ctx.Err(); err != nil {
			// Report the tail rather than dropping it. A caller rendering one
			// row per check (and GenerateCheck's own per-check cancel result
			// promises exactly that) would otherwise leave every remaining row
			// pending forever, with no result and no progress callback.
			for _, remaining := range checks[i:] {
				report(cancelledResult(remaining, err))
			}
			return results
		}
		report(g.GenerateCheck(ctx, c))
	}
	return results
}

// cancelledResult is the outcome recorded for a check the run never reached.
func cancelledResult(c Check, err error) Result {
	provider, _ := ResolveProvider(c)
	return Result{
		UID:      c.UID,
		Action:   ActionFailed,
		Provider: provider,
		MQL:      c.Mql,
		Reason:   "cancelled",
		Err:      err,
	}
}

// GenerateCheck produces MQL for one check. It never panics; failures are
// reported in the Result.
func (g *Generator) GenerateCheck(ctx context.Context, c Check) Result {
	res := Result{UID: c.UID}

	if c.HasVariants {
		res.Action = ActionSkipped
		res.Reason = "variant parent (mql lives in its variants)"
		res.MQL = c.Mql
		return res
	}

	if strings.TrimSpace(c.Mql) != "" && !g.cfg.Force {
		res.Action = ActionSkipped
		res.Reason = "already has mql (use --force to regenerate)"
		res.MQL = c.Mql
		return res
	}

	intent := strings.TrimSpace(c.Title + " " + c.Desc)
	if strings.TrimSpace(c.Title) == "" && strings.TrimSpace(c.Desc) == "" {
		res.Action = ActionSkipped
		res.Reason = "no title or description to generate from"
		return res
	}

	provider, _ := ResolveProvider(c)
	res.Provider = provider

	// Ask up front whether this provider can be validated at all. Discovering it
	// afterwards costs MaxAttempts agent invocations and reports "cannot find
	// resource for identifier 'aws'", which reads as a defect in the generated
	// query rather than a provider that was never installed.
	if checker, ok := g.cfg.Validator.(ProviderChecker); ok && g.cfg.Validator != nil {
		if err := checker.CheckProvider(provider); err != nil {
			res.Action = ActionFailed
			res.Err = err
			res.Reason = err.Error()
			return res
		}
	}

	var examples []Example
	if g.cfg.Corpus != nil {
		examples = g.cfg.Corpus.Search(intent, provider, g.cfg.ExampleLimit)
		// never feed the check its own prior body back as an example
		examples = filterExamples(examples, c.UID)
	}

	var lastErr error
	var lastMQL string
	for attempt := 1; attempt <= g.cfg.MaxAttempts; attempt++ {
		if ctx.Err() != nil {
			res.Action = ActionFailed
			res.Err = ctx.Err()
			res.Reason = "cancelled"
			return res
		}

		prompt := BuildPrompt(PromptData{
			Title:       c.Title,
			Desc:        c.Desc,
			Provider:    provider,
			Props:       c.Props,
			Guidance:    c.Guidance,
			Examples:    examples,
			Explain:     g.cfg.Explain,
			SkillPaths:  g.cfg.SkillPaths,
			RetryError:  errString(lastErr),
			PreviousMQL: lastMQL,
		})

		callCtx, cancel := context.WithTimeout(ctx, g.cfg.Timeout)
		gen, err := g.cfg.Backend.Generate(callCtx, GenTask{Prompt: prompt, Model: g.cfg.Model})
		cancel()
		if err != nil {
			lastErr = err
			continue
		}

		// An empty answer is not a generated check. The backend rejects one
		// today, but the contract belongs here: a Result carrying ActionGenerated
		// and no MQL would be written into a bundle as an empty check body.
		mql := strings.TrimSpace(gen.MQL)
		if mql == "" {
			lastErr = errors.New("agent returned no MQL")
			continue
		}

		lastMQL = mql
		if v := g.cfg.Validator; v != nil {
			verr := v.Validate(ValidationRequest{MQL: mql, Props: c.Props, Provider: provider})
			if verr != nil {
				// Retrying cannot fix an environment that cannot judge the
				// answer; report that, rather than a count of attempts that
				// makes it look like the agent kept getting it wrong.
				if errors.Is(verr, ErrValidationUnavailable) {
					res.Action = ActionFailed
					res.Err = verr
					res.Reason = verr.Error()
					return res
				}
				lastErr = verr
				continue
			}
		}

		res.Action = ActionGenerated
		res.MQL = mql
		res.Explanation = gen.Explanation
		return res
	}

	res.Action = ActionFailed
	res.Err = lastErr
	attempts := strconv.Itoa(g.cfg.MaxAttempts)
	if lastErr != nil {
		res.Reason = "failed after " + attempts + " attempts: " + lastErr.Error()
	} else {
		res.Reason = "failed after " + attempts + " attempts"
	}
	return res
}

func filterExamples(examples []Example, uid string) []Example {
	out := examples[:0]
	for _, ex := range examples {
		if ex.UID == uid {
			continue
		}
		out = append(out, ex)
	}
	return out
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
