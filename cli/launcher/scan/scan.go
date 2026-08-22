// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Package scan runs one `cnspec scan` as a background child and reports on it.
//
// It owns the process, the progress stream it emits, the report it writes and
// the single cleanup that removes all of it. It owns none of the presentation:
// there is no bubbletea message and no rendering in here, because what a scan
// *is* does not change with the screen that happens to be watching it.
package scan

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cockroachdb/errors"
	"go.mondoo.com/cnspec/cli/progress"
	"go.mondoo.com/cnspec/cli/reporter"
	"go.mondoo.com/cnspec/policy"
)

// A scan started from the launcher is a background child, not a handover.
//
// The launcher used to release the terminal to `cnspec scan` (tea.Exec) and
// take it back when the child was done, which made it a menu that got out of
// the way. It is now a place you stay: the child runs detached from the
// terminal, its progress arrives as NDJSON on a descriptor the launcher holds,
// and the report it writes is read back and rendered in the TUI.
//
// Three properties are load-bearing, and each one is a bug that was possible
// without it:
//
//   - The child writes a full-fidelity report. `-o json-full --output-target`
//     writes the whole policy.ReportCollection, which is the only artifact the
//     viewer can be built from; every other output format is a reduction that
//     has already thrown away the titles, docs and remediation.
//   - The child is pinned to this binary. Without MONDOO_AUTO_UPDATE=false it
//     hands off to whatever release sits in the auto-update cache -- a binary
//     that may never have heard of json-full, which would write nothing and
//     leave the viewer opening on a file that is not there.
//   - The child is killable. exec.CommandContext gets the context the UI
//     cancels, so a scan the user walks away from dies rather than carrying on
//     against a cloud account nobody is watching.

// ReportFormat is the output format the child is asked for, and ReportFile is
// what the artifact is called inside its private directory.
const (
	ReportFormat = "json-full"
	ReportFile   = "report.json"
)

// KillDelay is how long a cancelled child gets to exit on its own before
// os/exec kills it outright.
const KillDelay = 5 * time.Second

// OutputTail is how much of the child's output is kept for the error path.
const OutputTail = 16 << 10

// Binary names what a scan re-executes: this same binary, so the child goes
// through the exact same startup path as if the user had typed the command.
//
// It is a variable for one reason, and the reason is that the alternative is
// untestable. Everything that makes this phase worth having -- the inherited
// descriptor the progress stream arrives on, a non-zero exit that still carries
// a report, a cancel that actually kills something -- only exists once a real
// process has been forked. Substituting the child is what lets a test prove
// those against a real fork instead of asserting that the arguments look right.
var Binary = func() string {
	self, err := os.Executable()
	if err != nil {
		return os.Args[0]
	}
	return self
}

// Outcome is how the child ended: the error from Wait (nil for a clean exit)
// and whatever it wrote to its output streams.
//
// A non-nil Err is emphatically not "the scan failed". `cnspec scan` exits 1
// when any asset errored *or* when the worst score crosses the risk threshold
// (apps/cnspec/cmd/scan.go), so a scan that worked perfectly and found two
// failing checks lands here with an ExitError. Only the report file decides
// which of the two happened -- see LoadReport, and the launcher's scanExited.
type Outcome struct {
	Err    error
	Output string
}

// Request is what a scan has to be handed to run: an argv, the environment it
// needs on top of this process's own, and the release for whatever the caller
// wrote to disk to make the command possible.
//
// It is deliberately not the launcher's own assembly type. A launch plan also
// carries the warning line the scanning screen shows, which is a fact about a
// pane rather than about a process, and depending on it here would point the
// scan engine back at the layer that decides what to run. Naming what a scan
// needs is what leaves this package with no edge to the launcher at all.
type Request struct {
	// Args is the command line as the user sees it. The reporting flags the
	// launcher adds for itself are not in here and are never shown: they are
	// how the caller reads the result, not part of what was asked for.
	Args []string
	// Env is what the caller asked for, on top of this process's environment.
	Env []string
	// Cleanup releases whatever the caller wrote for this command -- a
	// generated inventory holding a credential, a rewritten kubeconfig. It is
	// released with the rest of the session, so there is one thing that ends a
	// scan rather than two.
	Cleanup func()
	// TrackTemp registers a cleanup in the caller's process-wide registry and
	// returns it wrapped, so that every exit the process can observe -- a
	// signal, a panic unwinding, quitting -- removes what this session wrote.
	//
	// A nil TrackTemp is honoured rather than rejected, but it means the report
	// directory is only removed on the paths this package reaches itself. The
	// launcher always supplies one; see the shim in apps/cnspec/cmd/interactive.
	TrackTemp func(func()) func()
}

// Session is one background scan: the child, the temp artifact it writes, the
// progress stream it emits, and the single cleanup that removes all of it.
//
// It is created on the caller's event loop -- which is why the context and the
// channels exist before the process does, so a user who presses esc while the
// fork is still in flight has something to cancel -- and started from a command
// that runs off it.
type Session struct {
	ctx    context.Context
	cancel context.CancelFunc

	args      []string
	env       []string
	trackTemp func(func()) func()

	// reportPath is where the child writes the json-full artifact. It is set
	// before the fork and read only after the child has exited.
	reportPath string

	// prog is the aggregate of the NDJSON progress stream, written by the
	// reader goroutine and read by the renderer. It carries its own lock.
	prog *Progress
	// wake carries one token per progress event and is closed when the stream
	// ends. It is a notification, not a queue: the state lives in prog, so a
	// coalesced wake loses nothing and a UI that fell behind never becomes
	// back-pressure on the child's writes.
	wake chan struct{}

	// exit carries the child's outcome exactly once.
	exit chan Outcome
	// done is closed once that outcome has been published, so an abort can wait
	// for the child to be reaped before removing the files it was reading.
	done chan struct{}

	startedAt time.Time

	cleanupMu sync.Mutex
	cleanups  []func()
	released  bool
}

// NewSession builds the session for a prepared request. Nothing is forked here:
// this runs on the caller's event loop, and the whole point of it is that the
// cancel function exists from the moment the launcher switches screens.
func NewSession(req Request) *Session {
	ctx, cancel := context.WithCancel(context.Background())
	s := &Session{
		ctx:       ctx,
		cancel:    cancel,
		args:      req.Args,
		env:       req.Env,
		trackTemp: req.TrackTemp,
		prog:      NewProgress(),
		wake:      make(chan struct{}, 1),
		exit:      make(chan Outcome, 1),
		done:      make(chan struct{}),
		startedAt: time.Now(),
	}
	if s.trackTemp == nil {
		s.trackTemp = func(fn func()) func() { return fn }
	}
	s.AddCleanup(req.Cleanup)
	return s
}

// Progress is the live aggregate of the child's progress stream.
func (s *Session) Progress() *Progress { return s.prog }

// Wake yields one token per progress event and is closed when the stream ends.
func (s *Session) Wake() <-chan struct{} { return s.wake }

// Exit carries the child's outcome exactly once.
func (s *Session) Exit() <-chan Outcome { return s.exit }

// ReportPath is where the child was told to write its artifact. It is empty
// until Start has run.
func (s *Session) ReportPath() string { return s.reportPath }

// Cancel asks the child to stop. It is safe to call more than once and before
// the fork, which is the point of it existing from construction.
func (s *Session) Cancel() { s.cancel() }

// AddCleanup registers something to release when the scan is over.
func (s *Session) AddCleanup(fn func()) {
	if fn == nil {
		return
	}
	s.cleanupMu.Lock()
	if s.released {
		// Registering after the session ended -- a fork that lost a race with
		// a cancel -- must still release, or the file it names outlives the UI.
		s.cleanupMu.Unlock()
		fn()
		return
	}
	s.cleanups = append(s.cleanups, fn)
	s.cleanupMu.Unlock()
}

// Release runs every cleanup exactly once, whichever path reaches it first.
func (s *Session) Release() {
	s.cleanupMu.Lock()
	if s.released {
		s.cleanupMu.Unlock()
		return
	}
	s.released = true
	fns := s.cleanups
	s.cleanups = nil
	s.cleanupMu.Unlock()

	for _, fn := range fns {
		fn()
	}
}

// Abort kills the child and releases everything once it has been reaped.
//
// Waiting matters: the child may still be reading the generated inventory, and
// removing that out from under a process that has not died yet turns a
// cancelled scan into a confusing connection error instead of a cancelled scan.
func (s *Session) Abort() {
	s.cancel()
	go func() {
		<-s.done
		s.Release()
	}()
}

// finish publishes the outcome and marks the session reaped. It is called
// exactly once, by whichever goroutine owns the child.
func (s *Session) finish(out Outcome) {
	s.exit <- out
	close(s.done)
}

// Elapsed is how long the scan has been running.
func (s *Session) Elapsed() time.Duration { return time.Since(s.startedAt) }

// Start forks the child and wires up its progress stream and its exit.
//
// Everything after Start runs in goroutines: one folding the NDJSON stream into
// the aggregate, one waiting for the process. The caller gets control back as
// soon as the fork succeeds.
//
// A Start that fails ends the session on the way out -- the progress stream is
// closed and the failure is published as the outcome -- because nothing was
// forked and so no goroutine will ever do it. That is what makes a session end
// exactly one way whatever happened, and it is why the caller has nothing to do
// with the error beyond showing it.
func (s *Session) Start() error {
	if err := s.start(); err != nil {
		close(s.wake)
		s.finish(Outcome{Err: err})
		return err
	}
	return nil
}

func (s *Session) start() error {
	self := Binary()

	// A private directory, not just a private file, the same way the generated
	// inventory does it: the report is not a secret, but it names every asset
	// and every failing check on them.
	dir, err := os.MkdirTemp("", "cnspec-ui-report-")
	if err != nil {
		return errors.Wrap(err, "cannot create a directory for the report")
	}
	s.reportPath = filepath.Join(dir, ReportFile)
	// Registered before the child can write a byte, in the caller's registry --
	// the same one the generated inventory holding a credential goes in -- so
	// every way out of this process -- quitting, SIGHUP, a panic unwinding --
	// removes it. Cancelling goes in the same cleanup: an abnormal exit must
	// kill the child rather than leave it writing into a directory that is
	// already gone.
	s.AddCleanup(s.trackTemp(func() {
		s.cancel()
		_ = os.RemoveAll(dir)
	}))

	sink, err := newProgressSink(dir)
	if err != nil {
		return err
	}

	// The reporting flags go on the end and are deliberately not stored back on
	// the session: they are how the launcher reads the answer, not part of the
	// command the user asked for, and the scanning screen shows the latter.
	args := append(append([]string{}, s.args...),
		"-o", ReportFormat, "--output-target", s.reportPath)

	cmd := exec.CommandContext(s.ctx, self, args...)
	cmd.Env = append(os.Environ(), s.env...)
	// os/exec keeps the last entry for a repeated key, so both of these win
	// over anything inherited.
	cmd.Env = append(cmd.Env,
		"MONDOO_AUTO_UPDATE=false",
		progress.StreamEnvVar+"="+sink.target)
	cmd.ExtraFiles = sink.extraFiles

	// Cancel is a request first and a kill second: cnspec closes its provider
	// connections on an interrupt. WaitDelay is what makes cancel mean cancel
	// anyway when it does not.
	cmd.Cancel = func() error {
		if err := cmd.Process.Signal(os.Interrupt); err != nil {
			return cmd.Process.Kill()
		}
		return nil
	}
	cmd.WaitDelay = KillDelay

	// The child gets no terminal: stdin is /dev/null and both output streams
	// are captured. os/exec promises at most one goroutine writes when Stdout
	// and Stderr are the same comparable writer. The tail is bounded because a
	// debug-level scan produces megabytes of log lines and only the last
	// screenful of them has ever explained anything.
	out := &tailWriter{limit: OutputTail}
	cmd.Stdout = out
	cmd.Stderr = out

	if err := cmd.Start(); err != nil {
		sink.closeAll()
		return errors.Wrapf(err, "cannot start %s", filepath.Base(self))
	}

	// The parent's copy of the write end is closed here, once, on purpose:
	// while it stays open the reader below never sees EOF. It is also why the
	// pipe is held as an *os.File all the way through Start rather than reduced
	// to a descriptor number. A file whose Fd was handed elsewhere and which
	// then becomes collectable is closed a second time by the runtime's own
	// cleanup, on a number the process has since reused -- and because os.File
	// uses runtime.AddCleanup now, SetFinalizer(f, nil) does not prevent it.
	sink.childStarted()

	go func() {
		defer sink.closeAll()
		s.ReadProgress(sink.reader)
	}()

	go func() {
		err := cmd.Wait()
		s.finish(Outcome{Err: err, Output: out.String()})
	}()

	return nil
}

// ReadProgress folds the child's NDJSON stream into the aggregate, waking the
// caller once per event. It returns when the stream ends, which on the pipe is
// when the child has exited and the parent's write end is closed.
func (s *Session) ReadProgress(r io.Reader) {
	defer close(s.wake)
	if r == nil {
		return
	}

	sc := bufio.NewScanner(r)
	// One event is one line and an asset name can be long. The ceiling stops a
	// corrupt stream from growing the buffer without bound.
	sc.Buffer(make([]byte, 0, 8<<10), 1<<20)

	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var ev progress.StreamEvent
		if err := json.Unmarshal(line, &ev); err != nil {
			// A line the launcher cannot read is a display problem, never a
			// reason to stop watching a scan that is running perfectly well.
			continue
		}
		s.prog.apply(ev)
		select {
		case s.wake <- struct{}{}:
		default:
		}
	}
}

// LoadReport reads the artifact the child wrote back into memory. On a real
// scan it is a multi-megabyte JSON document, which is not something to parse
// between two keystrokes -- callers run this off their event loop.
func (s *Session) LoadReport() (*policy.ReportCollection, error) {
	if s.reportPath == "" {
		return nil, errors.New("the scan wrote no report")
	}
	return reporter.LoadCollectionFile(s.reportPath)
}

// --- the progress destination ------------------------------------------------

// progressSink is the parent's end of the child's progress stream.
//
// There are two shapes because there have to be. A pipe is what a live consumer
// wants -- the reader blocks and events arrive as they happen -- but it reaches
// the child as an inherited descriptor, and exec.Cmd.ExtraFiles is not
// supported on Windows. There the stream goes to a file in the same private
// directory as the report, and is tailed.
type progressSink struct {
	// target is the MONDOO_PROGRESS_STREAM value handed to the child.
	target string
	// extraFiles are the descriptors the child inherits, if any.
	extraFiles []*os.File
	// reader yields the NDJSON stream to the parent.
	reader io.Reader

	// write is the parent's copy of the pipe's write end, closed as soon as the
	// child owns its own.
	write *os.File
	// closers release what is left when the stream is done.
	closers []io.Closer
	once    sync.Once
}

// progressFD is the descriptor the first entry of ExtraFiles lands on in the
// child; 0, 1 and 2 are the standard streams.
const progressFD = 3

func newProgressSink(dir string) (*progressSink, error) {
	if runtime.GOOS == "windows" {
		return newFileSink(dir)
	}
	return newPipeSink()
}

func newPipeSink() (*progressSink, error) {
	r, w, err := os.Pipe()
	if err != nil {
		return nil, errors.Wrap(err, "cannot open the progress stream")
	}
	return &progressSink{
		target:     "fd:" + strconv.Itoa(progressFD),
		extraFiles: []*os.File{w},
		reader:     r,
		write:      w,
		closers:    []io.Closer{r},
	}, nil
}

func newFileSink(dir string) (*progressSink, error) {
	path := filepath.Join(dir, "progress.ndjson")
	// Created here rather than left to the child, because the tail opens it
	// before the child has necessarily written anything.
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return nil, errors.Wrap(err, "cannot open the progress stream")
	}
	t := &tailReader{f: f}
	return &progressSink{
		target:  path,
		reader:  t,
		closers: []io.Closer{t, f},
	}, nil
}

// childStarted releases the parent's copy of anything the child now owns.
func (s *progressSink) childStarted() {
	if s.write != nil {
		_ = s.write.Close()
		s.write = nil
	}
}

// closeAll releases everything the sink holds, including a write end the child
// never got because the fork failed.
func (s *progressSink) closeAll() {
	s.once.Do(func() {
		s.childStarted()
		for _, c := range s.closers {
			_ = c.Close()
		}
	})
}

// tailInterval is how often the Windows fallback looks for new events.
const tailInterval = 100 * time.Millisecond

// tailReader turns a file being appended to by another process into a stream
// that ends when the reader is closed.
//
// It exists only for the platform that cannot inherit a pipe. Read polls rather
// than returning (0, nil), which an io.Reader caller is entitled to treat as a
// broken reader.
type tailReader struct {
	f      *os.File
	closed atomic.Bool
}

func (t *tailReader) Read(p []byte) (int, error) {
	for {
		n, err := t.f.Read(p)
		if n > 0 {
			return n, nil
		}
		if err != nil && !errors.Is(err, io.EOF) {
			return 0, err
		}
		if t.closed.Load() {
			return 0, io.EOF
		}
		time.Sleep(tailInterval)
	}
}

func (t *tailReader) Close() error {
	t.closed.Store(true)
	return nil
}

// --- the child's output ------------------------------------------------------

// tailWriter keeps the last limit bytes written to it. It is what the error
// path shows when a scan produced no readable report, because the reason it
// did not is on the child's stderr and nowhere else.
type tailWriter struct {
	mu    sync.Mutex
	limit int
	buf   []byte
}

func (w *tailWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.buf = append(w.buf, p...)
	if w.limit > 0 && len(w.buf) > w.limit {
		w.buf = w.buf[len(w.buf)-w.limit:]
	}
	return len(p), nil
}

func (w *tailWriter) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return string(w.buf)
}

// LastLines returns the final n non-empty lines of the child's output, which is
// what there is room for in a footer.
func LastLines(s string, n int) []string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		out = append(out, line)
	}
	if len(out) > n {
		out = out[len(out)-n:]
	}
	return out
}
