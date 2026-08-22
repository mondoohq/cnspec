// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Package generate turns a policy check's natural-language intent (its title and
// docs.desc) into validated MQL. cnspec drives the workflow and delegates the
// model call to a coding-agent CLI (Claude Code, Codex, Kimi, DeepSeek) that the
// user already has installed and authenticated. cnspec ships no LLM SDK and no
// API-key handling of its own; the agent brings its own model access.
//
// See docs/adr/0004-description-first-policy-authoring.md for the design.
package generate

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/cockroachdb/errors"
)

// GenTask is a single generation request handed to a backend. The prompt is
// fully rendered by the generator (see prompt.go) so backends stay dumb and
// interchangeable.
type GenTask struct {
	// Prompt is the complete instruction, including intent, schema context, and
	// grounding examples.
	Prompt string
	// Model optionally overrides the agent CLI's default model.
	Model string
}

// GenResult is the parsed outcome of a generation request.
type GenResult struct {
	// MQL is the generated query. Empty if the backend could not produce one.
	MQL string
	// Explanation is the agent's plain-language reasoning, when requested.
	Explanation string
	// Raw is the unparsed agent output, kept for diagnostics.
	Raw string
}

// AgentBackend is a pluggable generation backend. Each implementation drives one
// coding-agent CLI in headless mode. New agents are new adapters, not core
// changes.
type AgentBackend interface {
	// Name is the stable identifier used by the --agent flag.
	Name() string
	// Available reports whether the backend's CLI is installed and usable. A
	// backend that is not available is skipped by auto-selection and produces a
	// clear error when explicitly requested.
	Available() bool
	// Generate runs one task and returns the parsed result. Implementations must
	// respect ctx for cancellation and timeouts.
	Generate(ctx context.Context, t GenTask) (GenResult, error)
}

// cliBackend is the shared implementation behind every supported agent. It
// invokes a CLI in non-interactive mode, passing the prompt as the final
// argument and capturing stdout. The exact binary and flags are localized here
// so the rest of cnspec stays backend-agnostic, and every field can be
// overridden by an environment variable so the adapter survives CLI changes
// without a code release.
type cliBackend struct {
	name string
	// bin is the default executable name, looked up on PATH.
	bin string
	// binEnv is the environment variable that overrides bin (absolute path or
	// alternate name).
	binEnv string
	// args returns the arguments that precede the prompt, given an optional
	// model override.
	args func(model string) []string
}

func (b *cliBackend) Name() string { return b.name }

// binary resolves the executable, honoring the per-backend env override.
func (b *cliBackend) binary() string {
	if v := strings.TrimSpace(os.Getenv(b.binEnv)); v != "" {
		return v
	}
	return b.bin
}

func (b *cliBackend) Available() bool {
	_, err := exec.LookPath(b.binary())
	return err == nil
}

func (b *cliBackend) Generate(ctx context.Context, t GenTask) (GenResult, error) {
	if !b.Available() {
		return GenResult{}, errors.Newf("agent %q is not available: %q not found on PATH (set %s to override)", b.name, b.binary(), b.binEnv)
	}

	argv := append(b.args(t.Model), t.Prompt)

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, b.binary(), argv...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	// The agent inherits the environment (auth tokens, config) but never our
	// stdin; headless invocations must not block waiting for input.
	cmd.Stdin = nil

	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return GenResult{}, errors.Wrapf(ctx.Err(), "agent %q timed out or was cancelled", b.name)
		}
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = strings.TrimSpace(stdout.String())
		}
		return GenResult{}, errors.Wrapf(err, "agent %q failed: %s", b.name, truncate(detail, 500))
	}

	raw := stdout.String()
	res := parseResponse(raw)
	res.Raw = raw
	if res.MQL == "" {
		return res, errors.Newf("agent %q returned no MQL; raw output: %s", b.name, truncate(strings.TrimSpace(raw), 500))
	}
	return res, nil
}

// registry holds the supported backends in a stable, documented order. Claude
// Code is first so it wins auto-selection when multiple CLIs are installed.
func registry() []AgentBackend {
	return []AgentBackend{
		&cliBackend{
			name:   "claude",
			bin:    "claude",
			binEnv: "CNSPEC_AGENT_CLAUDE_BIN",
			args: func(model string) []string {
				a := []string{"-p", "--output-format", "text"}
				if model != "" {
					a = append(a, "--model", model)
				}
				return a
			},
		},
		&cliBackend{
			name:   "codex",
			bin:    "codex",
			binEnv: "CNSPEC_AGENT_CODEX_BIN",
			args: func(model string) []string {
				a := []string{"exec"}
				if model != "" {
					a = append(a, "--model", model)
				}
				return a
			},
		},
		&cliBackend{
			name:   "kimi",
			bin:    "kimi",
			binEnv: "CNSPEC_AGENT_KIMI_BIN",
			args: func(model string) []string {
				a := []string{}
				if model != "" {
					a = append(a, "--model", model)
				}
				return a
			},
		},
		&cliBackend{
			name:   "deepseek",
			bin:    "deepseek",
			binEnv: "CNSPEC_AGENT_DEEPSEEK_BIN",
			args: func(model string) []string {
				a := []string{}
				if model != "" {
					a = append(a, "--model", model)
				}
				return a
			},
		},
	}
}

// BackendNames returns the supported agent identifiers in selection order.
func BackendNames() []string {
	bs := registry()
	names := make([]string, len(bs))
	for i, b := range bs {
		names[i] = b.Name()
	}
	return names
}

// AvailableBackends returns the names of backends whose CLI is installed.
func AvailableBackends() []string {
	var names []string
	for _, b := range registry() {
		if b.Available() {
			names = append(names, b.Name())
		}
	}
	sort.Strings(names)
	return names
}

// Backend returns the named backend. An empty name auto-selects the first
// available backend in registry order. It errors when the requested backend is
// unknown or when no backend is available at all, with a message that tells the
// user how to proceed.
func Backend(name string) (AgentBackend, error) {
	bs := registry()

	if name == "" {
		for _, b := range bs {
			if b.Available() {
				return b, nil
			}
		}
		return nil, errors.Newf("no supported coding agent found on PATH; install one of %s, or set --agent and its CNSPEC_AGENT_*_BIN override", strings.Join(BackendNames(), ", "))
	}

	for _, b := range bs {
		if b.Name() == name {
			if !b.Available() {
				return nil, errors.Newf("agent %q is not installed or not on PATH (set CNSPEC_AGENT_%s_BIN to override)", name, strings.ToUpper(name))
			}
			return b, nil
		}
	}
	return nil, errors.Newf("unknown agent %q; supported agents: %s", name, strings.Join(BackendNames(), ", "))
}

// truncate shortens s to at most n bytes without splitting a multibyte rune.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	// back off to a rune boundary at or before n
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	return s[:n] + "…"
}
