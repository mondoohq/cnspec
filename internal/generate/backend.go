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
	"time"
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
	// maxStdout overrides the stdout cap; zero means maxAgentStdout. Only tests
	// set it, so a runaway agent can be reproduced without producing megabytes.
	maxStdout int
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

// Output limits for one agent invocation. An answer is a fenced JSON object of
// a few hundred bytes; anything past these caps is a runaway, not a response.
// Reading an agent's stdout into an unbounded buffer cost 1.59 GB of RSS for
// 400 MB of output.
const (
	maxAgentStdout = 8 << 20   // 8 MiB
	maxAgentStderr = 256 << 10 // 256 KiB
)

func (b *cliBackend) Generate(ctx context.Context, t GenTask) (GenResult, error) {
	if !b.Available() {
		return GenResult{}, errors.Newf("agent %q is not available: %q not found on PATH (set %s to override)", b.name, b.binary(), b.binEnv)
	}

	argv := append(b.args(t.Model), t.Prompt)

	limit := b.maxStdout
	if limit <= 0 {
		limit = maxAgentStdout
	}
	stdout := &cappedBuffer{max: limit}
	stderr := &cappedBuffer{max: maxAgentStderr}
	cmd := exec.CommandContext(ctx, b.binary(), argv...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	// The agent gets a deliberate environment, not ours: see agentEnv. It never
	// gets our stdin; headless invocations must not block waiting for input.
	cmd.Env = agentEnviron()
	cmd.Stdin = nil
	// An agent is a process tree. Cancelling has to stop the tree, not just stop
	// waiting for it: os/exec's Wait returns when the stdout pipe closes, and a
	// helper the agent forked holds that pipe open long after the timeout. The
	// process group makes the tree killable with one signal (see proc_unix.go),
	// and WaitDelay bounds the case where something still holds the pipe after
	// the group is gone.
	setProcessGroup(cmd)
	cmd.Cancel = func() error { return killTree(cmd) }
	cmd.WaitDelay = 2 * time.Second

	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return GenResult{}, errors.Wrapf(ctx.Err(), "agent %q timed out or was cancelled", b.name)
		}
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = strings.TrimSpace(stdout.String())
		}
		// The agent's own output goes into an error a human reads on a
		// terminal, so it is sanitized like any other model-authored text.
		return GenResult{}, errors.Wrapf(err, "agent %q failed: %s", b.name, SanitizeModelText(truncate(detail, 500)))
	}

	// A truncated response must not be parsed: the tail of a fenced block is
	// where the answer ends, so half a response can parse into a query that was
	// never returned. Report the runaway instead.
	if stdout.Overflowed() {
		return GenResult{}, errors.Newf("agent %q produced more than %d bytes of output (%d and counting); refusing to parse a truncated response",
			b.name, stdout.max, stdout.Total())
	}

	raw := stdout.String()
	res := parseResponse(raw)
	res.Raw = raw
	if res.MQL == "" {
		return res, errors.Newf("agent %q returned no MQL; raw output: %s", b.name, SanitizeModelText(truncate(strings.TrimSpace(raw), 500)))
	}
	return res, nil
}

// cappedBuffer collects at most max bytes and counts everything else. It never
// fails a write: returning an error would stop os/exec draining the pipe, and a
// child blocked writing to a full pipe hangs until the timeout instead of
// exiting. Discarding the overflow keeps memory bounded while letting the agent
// finish, and Overflowed() lets the caller refuse the result outright.
type cappedBuffer struct {
	max   int
	buf   bytes.Buffer
	total int64
}

func (c *cappedBuffer) Write(p []byte) (int, error) {
	// n is what we report: an io.Writer that reports fewer bytes than it was
	// given is a short write, which os/exec turns into an error on Wait.
	n := len(p)
	c.total += int64(n)
	if room := c.max - c.buf.Len(); room > 0 {
		if len(p) > room {
			p = p[:room]
		}
		c.buf.Write(p)
	}
	return n, nil
}

func (c *cappedBuffer) String() string { return c.buf.String() }

func (c *cappedBuffer) Total() int64 { return c.total }

func (c *cappedBuffer) Overflowed() bool { return c.total > int64(c.max) }

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
