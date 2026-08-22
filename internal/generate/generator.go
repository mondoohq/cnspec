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

// Validate runs the configured validator against a query (e.g. MQL a user edited
// by hand in the interactive flow). Returns nil when no validator is configured.
func (g *Generator) Validate(query string) error {
	if g.cfg.Validator == nil {
		return nil
	}
	return g.cfg.Validator.Validate(query)
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
	for _, c := range checks {
		if ctx.Err() != nil {
			break
		}
		res := g.GenerateCheck(ctx, c)
		results = append(results, res)
		if progress != nil {
			progress(res)
		}
	}
	return results
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

		lastMQL = gen.MQL
		if v := g.cfg.Validator; v != nil {
			if verr := v.Validate(gen.MQL); verr != nil {
				lastErr = verr
				continue
			}
		}

		res.Action = ActionGenerated
		res.MQL = gen.MQL
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
