// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	"context"
	"io"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	launcherscan "go.mondoo.com/cnspec/cli/launcher/scan"
	"go.mondoo.com/cnspec/policy"
)

// What a scan *is* now lives in cli/launcher/scan: the child process, the
// NDJSON progress stream it emits, the report it writes, and the single cleanup
// that removes all of it. What is left here is the launcher's view of one --
// the bubbletea messages the event loop switches on, the commands that produce
// them, and the three display-only facts (the command line as typed, the
// weaker-guarantee warning, the target's name) that a scanning screen renders
// and a scan has no use for.
//
// The split is worth stating because it is not arbitrary: the engine has no
// tea.Msg in it and the shim forks nothing. That is what lets the engine be
// tested against a real fork without a Model, and the screen be rendered
// without one.

// scanSession is one background scan as the launcher sees it.
//
// It embeds the engine's session rather than reimplementing it, and adds only
// what the scanning screen draws. Args are held here as well as in the engine
// because they are two different facts that happen to coincide: the engine
// holds an argv to exec, this holds the command line to show, and the reporting
// flags the engine appends are deliberately in the first and not the second.
type scanSession struct {
	*launcherscan.Session

	// cancel asks the child to stop. It exists from construction, before the
	// fork, so an esc pressed while the fork is still in flight has something
	// to act on.
	cancel context.CancelFunc

	// args is the command line as the user sees it.
	args []string
	// warn is the weaker-guarantee note from the plan, shown while the scan
	// runs rather than only in the pane the user has just left.
	warn string
	// label names the target, for the scanning screen's title.
	label string
	// prog is the live aggregate the scanning screen reads each frame.
	prog scanProgressView
}

// scanProgressView is the renderer's handle on the engine's aggregate.
type scanProgressView struct{ *launcherscan.Progress }

func (p scanProgressView) snapshot() scanSnapshot { return p.Snapshot() }

// scanSnapshot and scanAsset are the engine's types under the names the views
// already use, so that moving the engine did not become a rename of every
// screen that draws one.
type (
	scanSnapshot = launcherscan.Snapshot
	scanAsset    = launcherscan.Asset
)

// scanOutcome is how the child ended: the error from Wait (nil for a clean
// exit) and whatever it wrote to its output streams.
//
// A non-nil err is emphatically not "the scan failed". `cnspec scan` exits 1
// when any asset errored *or* when the worst score crosses the risk threshold
// (apps/cnspec/cmd/scan.go), so a scan that worked perfectly and found two
// failing checks lands here with an ExitError. Only the report file decides
// which of the two happened -- see Model.scanExited.
type scanOutcome struct {
	err    error
	output string
}

// newScanSession builds the session for a prepared plan. Nothing is forked
// here: this runs on the event loop, and the whole point of it is that the
// cancel function exists from the moment the launcher switches screens.
func newScanSession(plan launchPlan, label string) *scanSession {
	// The plan's own temp files -- a generated inventory holding a credential,
	// a rewritten kubeconfig -- are released with the rest of the session, so
	// there is one thing that ends a scan rather than two. trackTemp is handed
	// in rather than reached for, because it is this process's registry of what
	// must be removed on any exit at all and the engine has no opinion on where
	// that lives.
	s := launcherscan.NewSession(launcherscan.Request{
		Args:      plan.args,
		Env:       plan.env,
		Cleanup:   plan.cleanup,
		TrackTemp: trackTemp,
	})
	return &scanSession{
		Session: s,
		cancel:  s.Cancel,
		args:    plan.args,
		warn:    plan.warn,
		label:   label,
		prog:    scanProgressView{s.Progress()},
	}
}

func (s *scanSession) release()                 { s.Release() }
func (s *scanSession) abort()                   { s.Abort() }
func (s *scanSession) elapsed() time.Duration   { return s.Elapsed() }
func (s *scanSession) readProgress(r io.Reader) { s.ReadProgress(r) }

// scanState is the launcher's grip on the background scan it is watching.
//
// The three fields move together and are meaningless apart: loading says
// something only while a session exists, and out is read exactly once, by the
// report that session's exit produced. They were three fields on Model, and the
// order in which each of the three endings had to clear them was written out
// three times.
type scanState struct {
	// session is the background child currently running, if any. It is a
	// pointer so that cancelling one from a copied Model still cancels the real
	// thing.
	session *scanSession
	// loading is true between the child exiting and its report being read; on a
	// large scan that parse is not instant and a still screen would look like a
	// hang.
	loading bool
	// out is how the last child ended, kept only long enough to explain a scan
	// that produced no readable report.
	out scanOutcome
}

// active reports whether a scan is attached.
func (s scanState) active() bool { return s.session != nil }

// current reports whether an arriving message is about the scan on screen.
//
// Every handler asks this first, and it is the whole of the staleness rule: an
// event from a scan the user has already cancelled must not reanimate the
// screen they left. The cancelled session releases itself.
func (s scanState) current(session *scanSession) bool { return session == s.session }

// begin attaches a session for a prepared plan and returns the command that
// forks it.
//
// The session is built here, on the event loop, rather than inside the command:
// its cancel function has to exist before the screen that offers to cancel is
// drawn, or an esc pressed during the fork would have nothing to act on.
func (s *scanState) begin(plan launchPlan, label string) tea.Cmd {
	*s = scanState{session: newScanSession(plan, label)}
	return startScanCmd(s.session)
}

// running wires up the two streams the child now has.
func (s scanState) running(msg scanRunningMsg) tea.Cmd {
	if !s.current(msg.session) {
		return nil
	}
	return tea.Batch(waitProgressCmd(msg.session), waitScanCmd(msg.session))
}

// progressed repaints and asks for the next event.
func (s scanState) progressed(msg scanProgressMsg) tea.Cmd {
	if !s.current(msg.session) || msg.done {
		return nil
	}
	return waitProgressCmd(msg.session)
}

// exited turns the child's exit into the one question that matters: is there a
// report to read?
//
// The exit status deliberately does not decide that. `cnspec scan` exits 1 when
// any asset errored or when the worst score crosses the risk threshold, so the
// ordinary outcome of a useful scan -- it found things -- is a non-zero status.
// The launcher therefore always tries to read the artifact, and only treats the
// run as failed when there is nothing readable there.
func (s *scanState) exited(msg scanExitedMsg) tea.Cmd {
	if !s.current(msg.session) {
		return nil
	}
	s.loading = true
	s.out = msg.outcome
	return loadReportCmd(msg.session)
}

// finish releases the session and hands back how the child ended, leaving no
// scan attached. It reports false for a message from a scan that is no longer
// the one on screen.
//
// The outcome is handed over rather than kept: it is up to 16 KiB of the
// child's log output, and there is no reason for it to live as long as the
// session does.
func (s *scanState) finish(msg scanReportMsg) (scanOutcome, bool) {
	if !s.current(msg.session) {
		return scanOutcome{}, false
	}
	// The artifact has been read into memory, so the file has no reason to
	// exist any longer -- and neither has the generated inventory the child was
	// reading, nor the child itself.
	msg.session.release()
	out := s.out
	*s = scanState{}
	return out, true
}

// abort kills the child and forgets it.
//
// It returns immediately rather than waiting for the process to die: the
// session reaps it and releases what it was reading in the background, and a UI
// that froze for five seconds on cancel would be the thing this whole phase was
// written to avoid.
func (s *scanState) abort() {
	if s.session != nil {
		s.session.abort()
	}
	*s = scanState{}
}

// cancel stops a scan on the way out of the program.
//
// A scan the user quit out of must not outlive the launcher. A started
// session's tracked cleanup cancels it too, so cleanupTempFiles covers the
// running case; this covers the one where the fork has not happened yet and
// there is nothing in the registry to cancel.
func (s *scanState) cancel() {
	if s.session != nil {
		s.session.cancel()
	}
	*s = scanState{}
}

// lastLines returns the final n non-empty lines of the child's output, which is
// what there is room for in a footer.
func lastLines(s string, n int) []string { return launcherscan.LastLines(s, n) }

// --- messages and commands ----------------------------------------------------

// scanRunningMsg reports that the child is alive and its streams are wired up.
type scanRunningMsg struct{ session *scanSession }

// scanProgressMsg reports that the aggregate moved, or -- with done set -- that
// the progress stream has ended.
type scanProgressMsg struct {
	session *scanSession
	done    bool
}

// scanExitedMsg carries the child's outcome.
type scanExitedMsg struct {
	session *scanSession
	outcome scanOutcome
}

// scanReportMsg carries the artifact the child wrote, or the reason it could
// not be read back.
type scanReportMsg struct {
	session *scanSession
	report  *policy.ReportCollection
	err     error
}

// startScanCmd forks the child off the event loop.
//
// A Start that fails has already ended the session -- the progress stream is
// closed and the failure published as the outcome -- so that a session ends
// exactly one way whatever happened. All that is left here is to say so.
func startScanCmd(s *scanSession) tea.Cmd {
	return func() tea.Msg {
		if err := s.Start(); err != nil {
			return scanExitedMsg{session: s, outcome: scanOutcome{err: err}}
		}
		return scanRunningMsg{session: s}
	}
}

// waitProgressCmd blocks until the aggregate moves.
func waitProgressCmd(s *scanSession) tea.Cmd {
	return func() tea.Msg {
		_, ok := <-s.Wake()
		return scanProgressMsg{session: s, done: !ok}
	}
}

// waitScanCmd blocks until the child exits.
func waitScanCmd(s *scanSession) tea.Cmd {
	return func() tea.Msg {
		out := <-s.Exit()
		return scanExitedMsg{session: s, outcome: scanOutcome{err: out.Err, output: out.Output}}
	}
}

// loadReportCmd reads the artifact back off the event loop. On a real scan it
// is a multi-megabyte JSON document, which is not something to parse between
// two keystrokes.
func loadReportCmd(s *scanSession) tea.Cmd {
	return func() tea.Msg {
		collection, err := s.LoadReport()
		if err != nil {
			return scanReportMsg{session: s, err: err}
		}
		return scanReportMsg{session: s, report: collection}
	}
}
