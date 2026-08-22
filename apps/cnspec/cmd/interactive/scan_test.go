// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	launcherscan "go.mondoo.com/cnspec/cli/launcher/scan"
)

// These tests fork a real child and drive the whole Model over it.
//
// What the child process itself guarantees -- the inherited descriptor, the pin
// to this binary, a cancel that kills something -- is proven in cli/launcher/scan,
// against a child that binary owns. What is left here is the launcher: that a
// scan which found things still ends in the viewer, and that one which wrote
// nothing ends back at the form saying why. Both of those are statements about
// phases and messages, and neither can be made without a real fork underneath.
//
// The child is this test binary, re-executed with -test.run pointed at
// TestScanChild, which is the pattern os/exec's own tests use.
// launcherscan.Binary is the seam. The fixtures under testdata are a real
// report and a real progress stream, both recorded from `cnspec scan local
// --incognito -f examples/example.mql.yaml -o json-full`.

const (
	childEnv     = "CNSPEC_UI_TEST_CHILD"
	childModeEnv = "CNSPEC_UI_TEST_CHILD_MODE"
	childExitEnv = "CNSPEC_UI_TEST_CHILD_EXIT"

	// childModeReport writes the recorded report and the recorded progress
	// stream: what a scan that worked looks like.
	childModeReport = "report"
	// childModeSilent writes progress but no report: a scan that fell over
	// before it had anything to say.
	childModeSilent = "silent"
)

// TestScanChild is the stand-in scan. It is not a test: it returns immediately
// unless it was started as a child, and when it was, it exits the process
// itself rather than letting the testing framework decide the status.
func TestScanChild(t *testing.T) {
	if os.Getenv(childEnv) != "1" {
		return
	}

	// Always the first thing on stderr, so a parent that captured the output
	// can see for itself which value the child was pinned to.
	childSay("MONDOO_AUTO_UPDATE=" + os.Getenv("MONDOO_AUTO_UPDATE"))

	writeChildProgress(os.Getenv("MONDOO_PROGRESS_STREAM"))

	if os.Getenv(childModeEnv) == childModeReport {
		if format := childFlag("-o"); format != launcherscan.ReportFormat {
			childDie(96, "the child was asked for format "+strconv.Quote(format))
		}
		raw, err := os.ReadFile(filepath.Join("testdata", "scan_report.json"))
		if err != nil {
			childDie(95, err.Error())
		}
		if err := os.WriteFile(childFlag("--output-target"), raw, 0o600); err != nil {
			childDie(94, err.Error())
		}
	}

	code, _ := strconv.Atoi(os.Getenv(childExitEnv))
	os.Exit(code)
}

func childSay(line string) { _, _ = fmt.Fprintln(os.Stderr, line) }

func childDie(code int, line string) {
	childSay(line)
	os.Exit(code)
}

// writeChildProgress replays the recorded NDJSON stream to wherever the parent
// asked for it, through both destination forms the real writer honors.
func writeChildProgress(target string) {
	if target == "" {
		return
	}

	var out *os.File
	if rest, ok := strings.CutPrefix(target, "fd:"); ok {
		fd, err := strconv.Atoi(rest)
		if err != nil {
			childDie(93, "unreadable descriptor "+target)
		}
		out = os.NewFile(uintptr(fd), target)
	} else {
		f, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
		if err != nil {
			childDie(92, err.Error())
		}
		out = f
	}
	if out == nil {
		childDie(91, "no progress stream")
	}
	defer func() { _ = out.Close() }()

	raw, err := os.ReadFile(filepath.Join("testdata", "scan_progress.ndjson"))
	if err != nil {
		childDie(90, err.Error())
	}
	for _, line := range strings.Split(strings.TrimRight(string(raw), "\n"), "\n") {
		if _, err := fmt.Fprintln(out, line); err != nil {
			childDie(89, err.Error())
		}
	}
}

// childFlag reads a flag's value off this process's own command line.
func childFlag(name string) string {
	for i, a := range os.Args {
		if a == name && i+1 < len(os.Args) {
			return os.Args[i+1]
		}
	}
	return ""
}

// childPlan builds a plan that runs the stand-in child instead of cnspec.
func childPlan(t *testing.T, mode string, exit int) launchPlan {
	t.Helper()

	real := launcherscan.Binary
	launcherscan.Binary = func() string { return os.Args[0] }
	t.Cleanup(func() { launcherscan.Binary = real })

	return launchPlan{
		// Everything after -- is left for the child to read off os.Args, which
		// is also where the reporting flags the session appends land.
		args: []string{"-test.run=TestScanChild", "--", "scan", "local"},
		env: []string{
			childEnv + "=1",
			childModeEnv + "=" + mode,
			childExitEnv + "=" + strconv.Itoa(exit),
		},
	}
}

// runSession forks the child and waits for it, returning its outcome.
func runSession(t *testing.T, s *scanSession) scanOutcome {
	t.Helper()
	if err := s.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	select {
	case out := <-s.Exit():
		return scanOutcome{err: out.Err, output: out.Output}
	case <-time.After(30 * time.Second):
		t.Fatal("the child never exited")
		return scanOutcome{}
	}
}

// A scan that wrote nothing is the only failure. What the user is told then is
// the child's own last word, because the loader's error is about a file and
// explains nothing on its own.
func TestAScanWithNoReportFailsWithTheChildsReason(t *testing.T) {
	s := newScanSession(childPlan(t, childModeSilent, 3), "local")
	defer s.release()

	out := runSession(t, s)
	if out.err == nil {
		t.Fatal("the child was supposed to exit non-zero")
	}

	msg, ok := loadReportCmd(s)().(scanReportMsg)
	if !ok {
		t.Fatal("expected a report message")
	}
	if msg.err == nil {
		t.Fatal("a scan that wrote no report must fail")
	}

	said := scanFailure(msg.err, out)
	if !strings.Contains(said, "no report") {
		t.Errorf("the failure does not say what went wrong: %q", said)
	}
	if strings.Contains(said, "exit status") {
		t.Errorf("an exit status is not a reason and must not be the whole message: %q", said)
	}
}

// The whole model, driven the way the program's own loop drives it: a scan
// starts, its progress arrives, its report is read and the viewer opens on it
// -- with the child exiting 1 throughout, because it found things.
func TestTheLauncherEndsUpInTheViewer(t *testing.T) {
	m := newTestModel()
	m.width, m.height = 120, 40

	model, cmd := m.startScan(childPlan(t, childModeReport, 1))
	m = model.(Model)
	if m.phase != phaseScanning {
		t.Fatal("starting a scan does not show the scanning screen")
	}
	if !strings.Contains(m.View(), "scanning") {
		t.Errorf("the scanning screen does not say what it is doing:\n%s", m.View())
	}

	m = pump(t, m, cmd, func(m Model) bool { return m.phase == phaseViewing })

	if m.phase != phaseViewing {
		t.Fatalf("the launcher never reached the viewer: %s", m.lastErr)
	}
	if !m.viewer.loaded {
		t.Error("the report was not kept")
	}
	if m.scan.active() {
		t.Error("the session outlived the scan it described")
	}
	if !strings.Contains(m.View(), "launcher-test-host") {
		t.Errorf("the viewer is not showing the scanned asset:\n%s", m.View())
	}
}

// A scan that wrote no report leaves the launcher where it started, saying why.
func TestAFailedScanReturnsToTheLauncher(t *testing.T) {
	m := newTestModel()
	m.width, m.height = 120, 40

	model, cmd := m.startScan(childPlan(t, childModeSilent, 2))
	m = pump(t, model.(Model), cmd, func(m Model) bool { return m.lastErr != "" })

	if m.phase != phaseForm {
		t.Errorf("a failed scan left the launcher in phase %d", m.phase)
	}
	if m.viewer.loaded {
		t.Error("a scan that wrote no report must not leave one behind")
	}
	if !strings.Contains(m.lastErr, "no report") {
		t.Errorf("the launcher does not say what happened: %q", m.lastErr)
	}
}

// pump runs commands the way bubbletea would, feeding each answer back into the
// model, until the condition holds or the test runs out of patience.
func pump(t *testing.T, m Model, cmd tea.Cmd, until func(Model) bool) Model {
	t.Helper()

	queue := []tea.Cmd{cmd}
	deadline := time.Now().Add(60 * time.Second)

	for len(queue) > 0 && !until(m) {
		if time.Now().After(deadline) {
			t.Fatal("the model never settled")
		}
		next := queue[0]
		queue = queue[1:]
		if next == nil {
			continue
		}

		msg := runCmd(t, next)
		if msg == nil {
			continue
		}
		// A batch is a list of commands for the runtime to run, not a message
		// the model has any use for.
		if batch, ok := msg.(tea.BatchMsg); ok {
			queue = append(queue, batch...)
			continue
		}
		// Timers would make this test take as long as they do.
		if isTimer(msg) {
			continue
		}

		model, out := m.Update(msg)
		m = model.(Model)
		queue = append(queue, out)
	}
	return m
}

// runCmd runs one command with a bound, so a command that never answers fails
// the test rather than hanging it.
func runCmd(t *testing.T, cmd tea.Cmd) tea.Msg {
	t.Helper()
	out := make(chan tea.Msg, 1)
	go func() { out <- cmd() }()
	select {
	case msg := <-out:
		return msg
	case <-time.After(30 * time.Second):
		t.Fatal("a command never answered")
		return nil
	}
}

// isTimer reports whether a message is one of bubbletea's own animations, which
// sleep for as long as they say and have nothing to contribute here.
func isTimer(msg tea.Msg) bool {
	name := strings.ToLower(fmt.Sprintf("%T", msg))
	return strings.Contains(name, "blink") || strings.Contains(name, "tick")
}
