// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"go.mondoo.com/cnspec/cli/progress"
	"go.mondoo.com/cnspec/cli/reporter"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
)

// These tests fork a real child, and that is the point of them.
//
// Everything this package is for only exists once a process has actually been
// started: the descriptor the progress stream is inherited on, an exit status
// that says failure while the report says otherwise, a cancel that has
// something to kill. A test that asserted the assembled arguments look right
// would prove none of it -- and the three bugs worth having a test for (a pipe
// that never reaches EOF, a scan treated as broken because it found problems,
// an abandoned child left running) are exactly the three it would miss.
//
// The child is this test binary, re-executed with -test.run pointed at
// TestScanChild, which is the pattern os/exec's own tests use. Binary is the
// seam. It writes what it emits rather than replaying a recording, because what
// is under test here is the transport and the lifecycle, not the shape of a
// real scan's output: the launcher's own tests, which drive a Model, are the
// ones that run against recorded fixtures.

const (
	childEnv     = "CNSPEC_SCAN_TEST_CHILD"
	childModeEnv = "CNSPEC_SCAN_TEST_CHILD_MODE"
	childExitEnv = "CNSPEC_SCAN_TEST_CHILD_EXIT"

	// childModeReport writes a report and a progress stream: what a scan that
	// worked looks like.
	childModeReport = "report"
	// childModeSilent writes progress but no report: a scan that fell over
	// before it had anything to say.
	childModeSilent = "silent"
	// childModeHang never exits on its own, so only a kill ends it.
	childModeHang = "hang"
)

// childAsset is what the stand-in child claims to have scanned, on both the
// progress stream and in the report, so a test can tell one end from the other.
const (
	childAssetIndex = "//platformid.api.mondoo.app/hostname/scan-test-host"
	childAssetName  = "scan-test-host"
	childAssetScore = "HIGH"
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

	writeChildProgress(os.Getenv(progress.StreamEnvVar))

	if os.Getenv(childModeEnv) == childModeReport {
		if format := childFlag("-o"); format != ReportFormat {
			childDie(96, "the child was asked for format "+strconv.Quote(format))
		}
		writeChildReport(childFlag("--output-target"))
	}

	if os.Getenv(childModeEnv) == childModeHang {
		childSay("the child is waiting to be killed")
		time.Sleep(2 * time.Minute)
	}

	code, _ := strconv.Atoi(os.Getenv(childExitEnv))
	os.Exit(code)
}

func childSay(line string) { _, _ = fmt.Fprintln(os.Stderr, line) }

func childDie(code int, line string) {
	childSay(line)
	os.Exit(code)
}

// writeChildReport writes a json-full artifact through the same writer the real
// scan uses, so what comes back is a collection the loader accepts rather than
// a shape this test invented.
func writeChildReport(path string) {
	if path == "" {
		childDie(94, "no output target")
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		childDie(94, err.Error())
	}
	defer func() { _ = f.Close() }()

	collection := &policy.ReportCollection{
		Assets: map[string]*inventory.Asset{
			childAssetIndex: {
				Mrn:  childAssetIndex,
				Name: childAssetName,
			},
		},
	}
	if err := reporter.WriteCollection(collection, f); err != nil {
		childDie(93, err.Error())
	}
}

// writeChildProgress emits an NDJSON stream to wherever the parent asked for
// it, through both destination forms the real writer honors.
func writeChildProgress(target string) {
	if target == "" {
		return
	}

	var out *os.File
	if rest, ok := strings.CutPrefix(target, "fd:"); ok {
		fd, err := strconv.Atoi(rest)
		if err != nil {
			childDie(92, "unreadable descriptor "+target)
		}
		out = os.NewFile(uintptr(fd), target)
	} else {
		f, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
		if err != nil {
			childDie(91, err.Error())
		}
		out = f
	}
	if out == nil {
		childDie(90, "no progress stream")
	}
	defer func() { _ = out.Close() }()

	events := []progress.StreamEvent{
		{Event: progress.EventScanStart, Version: progress.StreamVersion},
		{Event: progress.EventDiscovered, Count: 1, Discovered: 1},
		{Event: progress.EventAssetAdded, Index: childAssetIndex, Name: childAssetName, Platform: "debian", Num: 1},
		{Event: progress.EventAssetProgress, Index: childAssetIndex, Num: 1, Percent: 0.5},
		// A line the reader cannot fold has to be skipped rather than end the
		// stream, so one goes down the wire on purpose; see below.
		{Event: "a_future_event_this_launcher_has_never_heard_of", Index: childAssetIndex},
		{Event: progress.EventAssetScore, Index: childAssetIndex, Num: 1, Score: childAssetScore},
		{Event: progress.EventAssetDone, Index: childAssetIndex, Num: 1, State: progress.StateCompleted, Score: childAssetScore, Done: 1, Total: 1},
		{Event: progress.EventScanDone, Total: 1, Done: 1, Completed: 1, Discovered: 1},
	}

	enc := json.NewEncoder(out)
	for i, ev := range events {
		ev.Seq = uint64(i)
		ev.Time = time.Now().UTC().Format(time.RFC3339Nano)
		if err := enc.Encode(ev); err != nil {
			childDie(89, err.Error())
		}
	}
	// Not JSON at all: a corrupt line is a display problem and must not stop
	// the parent watching a scan that is running perfectly well.
	if _, err := fmt.Fprintln(out, "{this is not json"); err != nil {
		childDie(88, err.Error())
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

// childRequest builds a request that runs the stand-in child instead of cnspec.
func childRequest(t *testing.T, mode string, exit int) Request {
	t.Helper()

	real := Binary
	Binary = func() string { return os.Args[0] }
	t.Cleanup(func() { Binary = real })

	return Request{
		// Everything after -- is left for the child to read off os.Args, which
		// is also where the reporting flags the session appends land.
		Args: []string{"-test.run=TestScanChild", "--", "scan", "local"},
		Env: []string{
			childEnv + "=1",
			childModeEnv + "=" + mode,
			childExitEnv + "=" + strconv.Itoa(exit),
		},
		TrackTemp: func(fn func()) func() { return fn },
	}
}

// runSession forks the child and waits for it, returning its outcome.
func runSession(t *testing.T, s *Session) Outcome {
	t.Helper()
	if err := s.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	select {
	case out := <-s.Exit():
		return out
	case <-time.After(30 * time.Second):
		t.Fatal("the child never exited")
		return Outcome{}
	}
}

// drained closes when the progress reader has seen the end of the stream.
func drained(s *Session) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		for range s.Wake() { //revive:disable-line:empty-block
		}
	}()
	return done
}

// A scan that found failing checks exits non-zero. That is the ordinary outcome
// of a useful scan, and the caller tells it apart from a scan that broke by
// reading the report rather than the exit status.
func TestAFailingScanStillProducesAReadableReport(t *testing.T) {
	s := NewSession(childRequest(t, childModeReport, 1))
	defer s.Release()

	out := runSession(t, s)
	if out.Err == nil {
		t.Fatal("the child was supposed to exit non-zero")
	}

	report, err := s.LoadReport()
	if err != nil {
		t.Fatalf("a non-zero exit must not stop the report being read: %v", err)
	}
	if len(report.GetAssets()) == 0 {
		t.Fatal("the report carries no assets")
	}
}

// A scan that wrote nothing is the only failure, and what there is to say about
// it is on the child's own output: the loader's error is about a file and
// explains nothing on its own.
func TestAScanWithNoReportKeepsTheChildsLastWords(t *testing.T) {
	s := NewSession(childRequest(t, childModeSilent, 3))
	defer s.Release()

	out := runSession(t, s)
	if out.Err == nil {
		t.Fatal("the child was supposed to exit non-zero")
	}
	if _, err := s.LoadReport(); err == nil {
		t.Fatal("a scan that wrote no report must fail to load one")
	}

	lines := LastLines(out.Output, 1)
	if len(lines) != 1 || !strings.Contains(lines[0], "MONDOO_AUTO_UPDATE") {
		t.Errorf("the child's last words did not survive: %q", out.Output)
	}
}

// The pin exists because the auto-update cache can hold a release that has
// never heard of json-full and would write no report at all. The child reports
// what it was given, so this asserts the value that actually reached it.
func TestTheChildIsPinnedToThisBinary(t *testing.T) {
	s := NewSession(childRequest(t, childModeSilent, 0))
	defer s.Release()

	out := runSession(t, s)
	if !strings.Contains(out.Output, "MONDOO_AUTO_UPDATE=false") {
		t.Errorf("the child was not pinned to this binary; it saw:\n%s", out.Output)
	}
}

// The progress stream has to arrive over the descriptor the child inherited,
// and the reader has to see the end of it. A pipe whose parent end is left open
// never reaches EOF, which would hang the reader for the life of the launcher.
func TestProgressArrivesOverTheInheritedDescriptor(t *testing.T) {
	s := NewSession(childRequest(t, childModeReport, 1))
	defer s.Release()

	runSession(t, s)

	select {
	case <-drained(s):
	case <-time.After(10 * time.Second):
		t.Fatal("the progress stream never ended: the parent's write end is still open")
	}

	snap := s.Progress().Snapshot()
	if snap.Total != 1 || snap.Done != 1 || snap.Completed != 1 {
		t.Fatalf("the stream describes one completed asset, got %+v", snap)
	}
	if snap.Assets[0].Score != childAssetScore {
		t.Errorf("the asset's score did not survive the stream: %q", snap.Assets[0].Score)
	}
	if snap.Assets[0].Percent != 1 {
		t.Errorf("a finished asset is at 100%%, got %v", snap.Assets[0].Percent)
	}
	if snap.Assets[0].Label() != childAssetName {
		t.Errorf("the asset is not named: %q", snap.Assets[0].Label())
	}
	if !snap.ScanDone {
		t.Error("the summary event never arrived")
	}
	// The child sent one event this launcher has never heard of and one line
	// that is not JSON. Neither is a reason to stop watching, and the summary
	// above arriving after both is what proves it.
}

// A scan the user abandons has to die. The child here would sleep for two
// minutes; if cancelling did not kill it, this test would sit there for two
// minutes and then fail.
func TestCancellingKillsTheChild(t *testing.T) {
	s := NewSession(childRequest(t, childModeHang, 0))

	if err := s.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	reportDir := filepath.Dir(s.ReportPath())

	s.Abort()

	select {
	case out := <-s.Exit():
		if out.Err == nil {
			t.Error("a killed child does not exit cleanly")
		}
	case <-time.After(15 * time.Second):
		t.Fatal("cancel did not kill the child")
	}

	// And the abort releases what the child was reading, once it is gone.
	deadline := time.Now().Add(10 * time.Second)
	for {
		if _, err := os.Stat(reportDir); os.IsNotExist(err) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("the report directory outlived the cancelled scan: %s", reportDir)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// A session ends exactly one way whatever happened. A fork that never happened
// still has to close the progress stream and publish an outcome, or the caller
// waits on two channels that will never answer.
func TestAStartThatNeverForkedStillEndsTheSession(t *testing.T) {
	real := Binary
	Binary = func() string { return filepath.Join(t.TempDir(), "no-such-binary") }
	t.Cleanup(func() { Binary = real })

	s := NewSession(Request{Args: []string{"scan", "local"}})
	defer s.Release()

	if err := s.Start(); err == nil {
		t.Fatal("starting a binary that is not there must fail")
	}

	select {
	case out := <-s.Exit():
		if out.Err == nil {
			t.Error("the failure was not published as the outcome")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("a fork that never happened never ended the session")
	}

	select {
	case <-drained(s):
	case <-time.After(5 * time.Second):
		t.Fatal("the progress stream was left open with nothing to write to it")
	}
}

// The plan's own temp files are released with the session, so that there is one
// thing that ends a scan rather than two -- and a credential written for a
// command that was cancelled does not outlive the launcher.
func TestTheRequestsCleanupIsReleasedWithTheSession(t *testing.T) {
	dir := t.TempDir()
	secret := filepath.Join(dir, "inventory.yml")
	if err := os.WriteFile(secret, []byte("password: <PLACEHOLDER-not-a-real-secret>\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	tracked := 0
	req := childRequest(t, childModeSilent, 0)
	req.Cleanup = func() { _ = os.Remove(secret) }
	req.TrackTemp = func(fn func()) func() { tracked++; return fn }

	s := NewSession(req)
	runSession(t, s)

	reportDir := filepath.Dir(s.ReportPath())
	s.Release()

	if _, err := os.Stat(secret); !os.IsNotExist(err) {
		t.Error("the credential the command was given outlived the scan")
	}
	if _, err := os.Stat(reportDir); !os.IsNotExist(err) {
		t.Error("the report directory outlived the scan")
	}
	if tracked != 1 {
		t.Errorf("the report directory was not registered with the caller's temp registry (%d)", tracked)
	}
}
