// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
)

// Run shows the interactive launcher and blocks until the user quits. Commands
// the user picks are run from inside the launcher (see launchCmd), so a session
// can run several of them.
func Run() error {
	// The last line of defence for the generated inventory, which holds a
	// plaintext credential when the OS keychain was unavailable. This covers
	// every way out of Run: a quit, an error, a panic unwinding through here,
	// and -- because bubbletea turns both into a quit -- SIGINT and SIGTERM.
	defer cleanupTempFiles()
	stopWatching := cleanupTempFilesOnHangup()
	defer stopWatching()

	m := NewModel(BuildCatalog())
	_, err := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion()).Run()
	return err
}

// cleanupTempFilesOnHangup removes what the launcher wrote if the terminal goes
// away, and returns the func that uninstalls it.
//
// SIGHUP is the one signal worth handling here, and the only one. bubbletea
// already listens for SIGINT and SIGTERM and turns them into a quit
// (Program.handleSignals), so Run returns normally and the deferred cleanup
// above does the work with the terminal properly restored -- installing our own
// handler for those would race with that restore for no gain. Nothing listens
// for SIGHUP, whose default disposition is to terminate, and which is exactly
// what arrives when a terminal window is closed on a launcher sitting at a form
// with a credential in it.
//
// Exiting rather than re-raising is deliberate: SIGHUP means the terminal has
// already gone, so there is no screen state left to hand back and nothing to
// restore it for.
func cleanupTempFilesOnHangup() func() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP)
	done := make(chan struct{})

	go func() {
		select {
		case <-sig:
			cleanupTempFiles()
			os.Exit(1)
		case <-done:
		}
	}()

	return func() {
		signal.Stop(sig)
		close(done)
	}
}

// launchDoneMsg reports that a launched command finished.
type launchDoneMsg struct{ err error }

// launchCmd hands the terminal to this same binary with the assembled
// arguments. Re-executing (rather than calling into cobra directly) means the
// chosen command goes through the exact same startup path as if the user had
// typed it: provider auto-install, flag parsing, discovery, and all.
//
// This is now the exception rather than the rule. A scan runs as a background
// child and comes back as a report the launcher renders (see scan.go); what is
// left here is `shell`, which is a genuinely interactive REPL, produces no
// report, and therefore has to own the terminal for as long as it runs.
// tea.Exec hands it over and takes it back on exit.
func launchCmd(args []string, extraEnv []string, warn string) tea.Cmd {
	self, err := os.Executable()
	if err != nil {
		self = os.Args[0]
	}
	c := exec.Command(self, args...)
	c.Env = append(os.Environ(), extraEnv...)
	// The launcher is already the current binary; letting the child hand off to
	// whatever release sits in the auto-update cache would put the user in a
	// different version of the shell than the one they are running.
	c.Env = append(c.Env, "MONDOO_AUTO_UPDATE=false")
	return tea.Exec(&handoverCmd{Cmd: c, args: args, warn: warn}, func(err error) tea.Msg {
		return launchDoneMsg{err: err}
	})
}

// handoverCmd announces the command before running it, so the user sees what
// they are about to be dropped into.
//
// It used to also read a line from the shared stdin afterwards, to stop the
// launcher repainting over a scan report the user was still reading. That pause
// went with the scan: a report is now rendered inside the TUI, and reading a
// line off the terminal that bubbletea is about to take back is how type-ahead
// gets swallowed. A shell has nothing left on screen worth pausing for.
type handoverCmd struct {
	*exec.Cmd
	args []string
	warn string
}

func (c *handoverCmd) SetStdin(r io.Reader) {
	if c.Stdin == nil {
		c.Stdin = r
	}
}

func (c *handoverCmd) SetStdout(w io.Writer) {
	if c.Stdout == nil {
		c.Stdout = w
	}
}

func (c *handoverCmd) SetStderr(w io.Writer) {
	if c.Stderr == nil {
		c.Stderr = w
	}
}

func (c *handoverCmd) Run() error {
	out := c.Stdout
	if out == nil {
		out = os.Stdout
	}
	if c.warn != "" {
		// Printed where the user is looking when the command starts, not only
		// in the launcher they are about to leave.
		fmt.Fprintf(out, "\n! %s\n", c.warn)
	}
	fmt.Fprintf(out, "\n$ cnspec %s\n\n", shellJoin(c.args))

	err := c.Cmd.Run()
	if err != nil {
		fmt.Fprintf(out, "\ncommand exited with an error: %v\n", err)
	}
	// The exit status is the command's business, not the launcher's: a failing
	// command is a normal outcome and must not look like the launcher broke.
	return nil
}

// shellJoin renders args for display, quoting the ones that need it.
func shellJoin(args []string) string {
	out := make([]string, 0, len(args))
	for _, a := range args {
		if a == "" || strings.ContainsAny(a, " \t\"'") {
			out = append(out, strconv.Quote(a))
			continue
		}
		out = append(out, a)
	}
	return strings.Join(out, " ")
}

// tokenize splits a user-entered argument string into individual args,
// honoring double and single quotes so values like -c "aws.regions" survive as
// a single token. This is intentionally small: it is a convenience for the
// launcher, not a full shell parser.
func tokenize(s string) []string {
	var out []string
	var cur []rune
	var quote rune
	inToken := false

	flush := func() {
		if inToken {
			out = append(out, string(cur))
			cur = cur[:0]
			inToken = false
		}
	}

	for _, r := range s {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				cur = append(cur, r)
			}
			inToken = true
		case r == '"' || r == '\'':
			quote = r
			inToken = true
		case r == ' ' || r == '\t':
			flush()
		default:
			cur = append(cur, r)
			inToken = true
		}
	}
	flush()
	return out
}
