// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build integration

package integration

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// result is one completed cnspec invocation.
type result struct {
	args     []string
	stdout   []byte
	stderr   []byte
	exitCode int
	duration time.Duration
	timedOut bool
}

// run executes the binary under test and returns once it exits or the timeout
// elapses.
//
// The suite has its own runner rather than using mql's test.NewCliTestRunner
// because that one builds its command with exec.Command and has no timeout: a
// provider subprocess that wedges would hang until `go test -timeout` killed
// the whole test binary, which reports as "panic: test timed out" with no
// indication of which target was being scanned. Here a stuck scan fails by
// name, with its output attached.
func run(t *testing.T, timeout time.Duration, args ...string) *result {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, cnspecBin, args...)
	cmd.Env = childEnv()
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	// Kill the process tree if it ignores the context cancellation. Provider
	// plugins are separate processes; without this the parent can exit while
	// children hold the pipes open and Wait never returns.
	cmd.WaitDelay = 15 * time.Second

	start := time.Now()
	err := cmd.Run()
	res := &result{
		args:     args,
		stdout:   stdout.Bytes(),
		stderr:   stderr.Bytes(),
		duration: time.Since(start),
		timedOut: errors.Is(ctx.Err(), context.DeadlineExceeded),
	}

	var exitErr *exec.ExitError
	switch {
	case err == nil:
		res.exitCode = 0
	case errors.As(err, &exitErr):
		res.exitCode = exitErr.ExitCode()
	default:
		t.Fatalf("could not run cnspec %s: %v", strings.Join(args, " "), err)
	}

	if res.timedOut {
		t.Fatalf("cnspec %s timed out after %s\n%s",
			strings.Join(args, " "), timeout, res.dump())
	}
	t.Logf("cnspec %s -> exit %d in %s", strings.Join(args, " "), res.exitCode, res.duration.Round(time.Millisecond))
	return res
}

// dump renders the invocation for a failure message. Both streams, because the
// interesting half is rarely the one you guessed: cnspec writes the report to
// stdout and everything explaining it -- including recovered provider panics --
// to stderr.
func (r *result) dump() string {
	var b strings.Builder
	fmt.Fprintf(&b, "--- args: %s\n", strings.Join(r.args, " "))
	fmt.Fprintf(&b, "--- exit: %d after %s\n", r.exitCode, r.duration.Round(time.Millisecond))
	fmt.Fprintf(&b, "--- stdout (%d bytes):\n%s\n", len(r.stdout), truncate(r.stdout, 4000))
	fmt.Fprintf(&b, "--- stderr (%d bytes):\n%s\n", len(r.stderr), truncate(r.stderr, 8000))
	return b.String()
}

func truncate(b []byte, max int) string {
	if len(b) <= max {
		return string(b)
	}
	return string(b[:max]) + fmt.Sprintf("\n... (%d more bytes)", len(b)-max)
}

// saveArtifact writes a failing scenario's output next to the tests so CI can
// upload it. An assertion that a check floor was missed is not diagnosable from
// the message alone; the report is.
func (r *result) saveArtifact(t *testing.T, name string) {
	t.Helper()
	safe := strings.NewReplacer("/", "-", " ", "-").Replace(name)
	path := filepath.Join(artifactDir, safe+".txt")
	if err := os.WriteFile(path, []byte(r.dump()), 0o644); err != nil {
		t.Logf("could not save artifact %s: %v", path, err)
		return
	}
	t.Logf("saved failing output to %s", path)
}
