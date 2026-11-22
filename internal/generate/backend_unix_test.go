// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build !windows

package generate

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// scriptedAgent writes an executable stand-in for an agent CLI and returns a
// backend that runs it.
func scriptedAgent(t *testing.T, body string) *cliBackend {
	t.Helper()
	path := filepath.Join(t.TempDir(), "agent.sh")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body), 0o700); err != nil {
		t.Fatalf("write script: %v", err)
	}
	return &cliBackend{
		name:   "scripted",
		bin:    path,
		binEnv: "CNSPEC_TEST_AGENT_BIN_UNSET",
		args:   func(string) []string { return nil },
	}
}

// TestBackendStopsAForkedHelper is the 601-seconds-against-a-180-second-timeout
// case. The agent exits immediately but leaves a child holding the stdout pipe,
// and os/exec's Wait waits for the pipe rather than for the process — so without
// a process group to kill and a WaitDelay to bound it, the run outlives its
// timeout by as long as the orphan lives.
func TestBackendStopsAForkedHelper(t *testing.T) {
	backend := scriptedAgent(t, "sleep 45 &\necho started\n")

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := backend.Generate(ctx, GenTask{Prompt: "irrelevant"})
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected the cancelled run to report an error")
	}
	// generous: the fix returns in about the timeout plus the 2s WaitDelay,
	// while the bug returns only when the 45s orphan exits
	if elapsed > 20*time.Second {
		t.Fatalf("Generate took %s; the agent's process tree outlived its timeout", elapsed)
	}
}

// TestBackendCapsStdout pins that a runaway agent is reported, not parsed. The
// buffer used to be unbounded (400 MB of output cost 1.59 GB of RSS), and a
// truncated capture must not be handed to the parser — half a response can parse
// into a query the agent never returned.
func TestBackendCapsStdout(t *testing.T) {
	// a valid answer, then far more output than the cap allows
	backend := scriptedAgent(t, `printf '{"mql": "aws.s3.buckets.all(encryption != empty)"}\n'
i=0
while [ $i -lt 200 ]; do
  printf 'xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx\n'
  i=$((i+1))
done
`)
	backend.maxStdout = 512

	res, err := backend.Generate(context.Background(), GenTask{Prompt: "irrelevant"})
	if err == nil {
		t.Fatalf("expected an error for a runaway agent, got MQL %q", res.MQL)
	}
	if !strings.Contains(err.Error(), "refusing to parse") {
		t.Fatalf("error should say the output was refused, got: %v", err)
	}
	if res.MQL != "" {
		t.Fatalf("a truncated capture must not yield MQL, got %q", res.MQL)
	}
}

// TestBackendDoesNotForwardCredentials is the end-to-end half of
// TestAgentEnvWithholdsCredentials: what the child process actually sees.
func TestBackendDoesNotForwardCredentials(t *testing.T) {
	t.Setenv("AWS_SECRET_ACCESS_KEY", "leaked-to-the-agent")
	t.Setenv("MONDOO_API_TOKEN", "leaked-to-the-agent")
	t.Setenv("ANTHROPIC_API_KEY", "the-agents-own-key")

	backend := scriptedAgent(t, "env\n")

	res, err := backend.Generate(context.Background(), GenTask{Prompt: "irrelevant"})
	// no MQL in `env` output, so Generate reports that; the raw output is what
	// this test is after
	if err == nil {
		t.Fatal("expected 'no MQL' from an agent that prints its environment")
	}
	if strings.Contains(res.Raw, "leaked-to-the-agent") {
		t.Errorf("the agent subprocess inherited a credential:\n%s", res.Raw)
	}
	if !strings.Contains(res.Raw, "the-agents-own-key") {
		t.Errorf("the agent must keep its own auth, got:\n%s", res.Raw)
	}
}
