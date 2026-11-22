// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package generate

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"
)

// TestHarness_RealAgent runs the golden intent→MQL cases against a real coding
// agent (Claude Code, Codex, Kimi, DeepSeek) so you can measure how well a given
// agent/model generates MQL. It is skipped by default so CI stays offline and
// deterministic; opt in with environment variables:
//
//	# evaluate Claude Code
//	CNSPEC_HARNESS_AGENT=claude go test ./internal/generate -run TestHarness_RealAgent -v
//
//	# evaluate Codex with a specific model, requiring at least 60% pass
//	CNSPEC_HARNESS_AGENT=codex CNSPEC_HARNESS_MODEL=... \
//	  CNSPEC_HARNESS_MIN_RATE=0.6 go test ./internal/generate -run TestHarness_RealAgent -v
//
// It reports the pass rate and every miss. By default it only fails if the agent
// is unavailable or generation errors on every case (clearly broken); set
// CNSPEC_HARNESS_MIN_RATE to enforce a quality floor.
func TestHarness_RealAgent(t *testing.T) {
	agent := os.Getenv("CNSPEC_HARNESS_AGENT")
	if agent == "" {
		t.Skip("set CNSPEC_HARNESS_AGENT=claude|codex|kimi|deepseek to run the real-agent harness")
	}

	backend, err := Backend(agent)
	if err != nil {
		t.Fatalf("agent %q not usable: %v", agent, err)
	}

	// Validate by compiling when a schema is available; otherwise report without
	// validation rather than failing the whole run.
	var validator Validator = NoopValidator{}
	if v, verr := NewCompileValidator(); verr == nil {
		validator = v
	} else {
		t.Logf("compile validation unavailable (%v); scoring generated MQL without it", verr)
	}

	cases, err := LoadCases("testdata/harness_cases.yaml")
	if err != nil {
		t.Fatalf("LoadCases: %v", err)
	}

	var corpus *Corpus
	for _, p := range []string{"../../content", "content"} {
		if c, cerr := LoadCorpus(p); cerr == nil && c.Size() > 0 {
			corpus = c
			t.Logf("grounding on %d examples from %s", c.Size(), p)
			break
		}
	}

	gen, err := New(Config{
		Backend:   backend,
		Corpus:    corpus,
		Validator: validator,
		Model:     os.Getenv("CNSPEC_HARNESS_MODEL"),
		Timeout:   3 * time.Minute,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	results := RunCases(context.Background(), gen, cases)
	rate, passed, total := PassRate(results)
	t.Logf("agent=%s model=%q pass=%d/%d (%.0f%%)", agent, os.Getenv("CNSPEC_HARNESS_MODEL"), passed, total, rate*100)
	for _, r := range results {
		if r.Pass {
			t.Logf("  PASS %s", r.Case.Name)
		} else {
			t.Logf("  FAIL %s: %s (got: %q)", r.Case.Name, r.Reason, r.Got)
		}
	}

	if passed == 0 {
		t.Fatalf("agent %q generated nothing usable across %d cases", agent, total)
	}
	if min := envFloat("CNSPEC_HARNESS_MIN_RATE", 0); rate < min {
		t.Fatalf("pass rate %.2f below required %.2f", rate, min)
	}
}

func envFloat(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}
