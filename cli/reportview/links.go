// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reportview

import (
	"bytes"
	"net/url"
	"os/exec"
	"runtime"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cockroachdb/errors"
	"github.com/muesli/termenv"
	"go.mondoo.com/cnspec/cli/tui"
)

// Clickable URLs, for the REFERENCES section of a check.
//
// A reference is a link and nothing else -- a CIS benchmark PDF, a CVE, a
// vendor advisory -- and printing it as eighty characters of plain text asks
// the reader to retype it, or to leave the viewer, find the report again and
// select it with the mouse. Both of the affordances below exist because either
// one alone leaves somebody stuck.
//
// # OSC 8, for the terminal
//
// termenv.Hyperlink wraps the text in an OSC 8 sequence, which is how iTerm2,
// WezTerm, Kitty, foot, GNOME Terminal, Windows Terminal and modern VTE
// terminals turn text into something you can click or ctrl-click. Two
// properties make it safe inside a pane that is measured in exact cells:
//
//   - it is zero width. ansi.StringWidth returns the same number for the bare
//     URL and the wrapped one, so a row that fits before wrapping still fits.
//   - ansi.Truncate cuts the *visible* text and keeps the sequences either side
//     of it, so a URL wider than the pane comes out as a shortened, still-valid
//     link rather than as an escape sequence leaking into the row below.
//
// TestHyperlinkIsZeroWidth and TestATruncatedLinkIsStillALink hold both to it,
// with the raw bytes in the failure message.
//
// A terminal that does not understand OSC 8 -- Terminal.app, older xterm, the
// Linux console -- skips the sequence and draws the text, so nothing degrades
// except the clicking. Nothing is printed twice and no marker is left behind.
//
// # A click zone, for this program
//
// OSC 8 is not enough on its own here, and the reason is specific to a TUI: the
// viewer runs with mouse tracking on, so the terminal hands the click to the
// application instead of acting on it. The link is live to a ctrl-click or a
// right-click-and-open, and dead to a plain one -- which is the click a reader
// will try first.
//
// So a link is also a Zone, exactly the way a COPY button is: a rect on the row
// it was drawn on, tagged, carrying the index of the URL it names. The detail
// pane turns a ClickMsg on one into a command, and the command hands the URL to
// the platform opener.
//
// # Where links are, and are not
//
// The REFERENCES section only. It is the one place the viewer prints a bare URL
// as a thing to go and read.
//
// Descriptions, audit steps and remediation prose are markdown, and they go
// through glamour, which already gives a link its own color and underline
// (styles.DarkStyleConfig.Link). Adding OSC 8 there would mean finding the URLs
// again inside a wall of styled rows that glamour has already word-wrapped,
// matching them back to the source by eye, and hoping the mapping holds -- the
// same guessing game markdown.Blocks exists to avoid for code fences. The row
// provenance simply is not there for prose, and a link zone on the wrong cells
// is worse than no link at all.
//
// # Opening it
//
// The platform opener: `open` on macOS, `rundll32 url.dll,FileProtocolHandler`
// on Windows, `xdg-open` elsewhere. No new module -- github.com/pkg/browser is
// in the graph but only as an indirect dependency, and promoting it would edit
// go.mod for four lines of exec.
//
// Three things it does that a naive exec does not:
//
//   - it refuses anything that is not http or https. A reference URL comes out
//     of a policy bundle, which is content, and `open` will cheerfully hand
//     file://, a custom scheme or an app URL to whatever claims it. The same
//     check gates whether the row is drawn as a link at all, so the viewer never
//     offers a link it would then refuse.
//   - it keeps the child off the terminal. The viewer owns an alt-screen, and a
//     helper writing "gio: no handler" onto it corrupts the frame. stdout and
//     stderr go to a buffer, which is also where the reason for a failure comes
//     from.
//   - it waits, and reports. A click that appears to do nothing is the complaint
//     this package keeps getting, so the outcome comes back as a message and the
//     footer says either what was opened or why it was not. Waiting is what makes
//     the failing cases visible at all: a missing xdg-open fails at start, but an
//     xdg-open with no handler only fails on exit. It costs nothing the user is
//     waiting on -- this runs as a tea.Cmd, off the event loop, so the viewer
//     keeps repainting whatever the opener does.

// linkZoneTag marks a Zone as a URL. The zone's Idx is the index of the URL in
// the pane's list.
const linkZoneTag = "link"

// linkable reports whether a string is a web address this viewer will present as
// a link and hand to the platform opener.
//
// It is deliberately narrow. Everything else -- a relative path, a mailto:, a
// scheme nobody recognizes, a bundle field that was never a URL -- is drawn as
// the plain text it is, which is what the section did before any of this.
func linkable(raw string) bool {
	if raw == "" {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

// hyperlink wraps text in an OSC 8 sequence pointing at url. The result is the
// same number of cells wide as text.
func hyperlink(url, text string) string {
	return termenv.Hyperlink(url, text)
}

// openDoneMsg is the outcome of opening a URL, on its way back to the frame.
//
// Unexported, like copyDoneMsg and unlike ExportDoneMsg: handing a URL to a
// helper is a subprocess that returns, not a render that can outlive the view,
// so there is no case where the viewer is gone and the result still matters.
type openDoneMsg struct {
	URL string
	Err error
}

// openURLCmd opens a URL, as a command.
func openURLCmd(raw string) tea.Cmd {
	return func() tea.Msg {
		return openDoneMsg{URL: raw, Err: openURL(raw)}
	}
}

// openURL hands a URL to the platform opener, having first checked that it is
// one worth handing over.
func openURL(raw string) error {
	if !linkable(raw) {
		return errors.Newf("%q is not an http or https address", raw)
	}
	return urlOpener(raw)
}

// urlOpener is what actually launches the browser.
//
// It is a variable for the same reason clipboardWrite is: a test that called the
// real one would open a browser window on the developer's machine and would fail
// on a build box that has no xdg-open, for a reason that has nothing to do with
// this viewer.
var urlOpener = runOpener

// runOpener runs the platform's opener and waits for it.
func runOpener(raw string) error {
	name, args := openerFor(runtime.GOOS, raw)
	cmd := exec.Command(name, args...)
	// The viewer owns an alt-screen buffer; a helper printing a diagnostic onto
	// it would corrupt the frame. The buffer catches that diagnostic instead,
	// which is the most useful half of the failure message.
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	if err := cmd.Run(); err != nil {
		if msg := tui.OneLine(out.String()); msg != "" {
			return errors.Wrap(err, msg)
		}
		return errors.Wrapf(err, "%s failed", name)
	}
	return nil
}

// openerFor is the command that opens a URL on an operating system, split out
// from running it so a test can check the three of them anywhere.
//
// The Windows form is rundll32 rather than `cmd /c start`, because start is a
// shell builtin and would put the URL through cmd's parser -- where & and ^ mean
// something -- for no gain.
func openerFor(goos, raw string) (string, []string) {
	switch goos {
	case "darwin":
		return "open", []string{raw}
	case "windows":
		return "rundll32", []string{"url.dll,FileProtocolHandler", raw}
	default:
		return "xdg-open", []string{raw}
	}
}

// openNotice is the one line the footer shows afterwards. A success says what
// was opened, because "opened" alone on a page of six references does not say
// which one; a failure says why, in the words of whatever refused.
func openNotice(msg openDoneMsg) string {
	if msg.Err != nil {
		return "could not open " + msg.URL + ": " + tui.OneLine(msg.Err.Error())
	}
	return "opened " + msg.URL
}

// openDone puts the outcome in the footer. Like a copy, this needs no box: the
// click was one action and the reason it failed fits on the line.
func (m Model) openDone(msg openDoneMsg) (tea.Model, tea.Cmd) {
	m.state.Notice = openNotice(msg)
	return m, nil
}
