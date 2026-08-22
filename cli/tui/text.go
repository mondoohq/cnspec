// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package tui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// Text helpers shared by every pane of both UIs, so that the rule "one element
// of a []string is exactly one terminal line" is enforced in one place rather
// than in each of them.

// Clean makes an arbitrary string safe to put in a rendered line. It normalizes
// carriage returns and expands tabs.
//
// It is also what keeps a fixed-height layout fixed. The launcher assigned
// err.Error() straight into its footer and rendered it through ansi.Truncate,
// which cuts a line to width and leaves every newline in it: a three-line
// provider error made the view 26 rows in a 24-row terminal and pushed the
// footer off the screen. Truncation does not remove newlines; this does.
//
// This is not optional for anything that came out of cli/reporter: its
// NewLineCharacter is "\r\n" on Windows, and a carriage return inside a line
// makes ansi.StringWidth disagree with what the terminal does, which shows up as
// a panel one column too wide and a layout that drifts. reportmodel already
// normalizes what it returns; anything pulled from elsewhere goes through here.
func Clean(s string) string {
	if strings.ContainsAny(s, "\r\t") {
		s = strings.ReplaceAll(s, "\r\n", "\n")
		s = strings.ReplaceAll(s, "\r", "\n")
		s = strings.ReplaceAll(s, "\t", "    ")
	}
	return s
}

// Lines splits a string into rendered lines, cleaning it first. A string with no
// newlines yields one line; an empty string yields none, so a caller can append
// the result of an empty section without leaving a blank row behind.
func Lines(s string) []string {
	s = Clean(s)
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

// Wrap breaks a string into lines no wider than w cells, preserving the
// newlines it already has. Words longer than w (a URL, an MRN) are broken rather
// than allowed to overflow.
func Wrap(s string, w int) []string {
	if w < 1 {
		w = 1
	}
	var res []string
	for _, para := range Lines(s) {
		if para == "" {
			res = append(res, "")
			continue
		}
		for _, line := range strings.Split(ansi.Wrap(para, w, ""), "\n") {
			if ansi.StringWidth(line) <= w {
				res = append(res, line)
				continue
			}
			// ansi.Wrap prefers to break at a word boundary and gives up on a
			// token it cannot fit -- an MRN or a URL at a narrow width. Those
			// get cut mid-token rather than allowed to overflow the pane.
			res = append(res, strings.Split(ansi.Hardwrap(line, w, false), "\n")...)
		}
	}
	return res
}

// Fit forces a slice of lines to be exactly n lines long, padding with blanks
// and dropping the excess. The frame uses it on every pane, which is what makes
// a miscounting pane a cosmetic bug rather than a scrolled terminal.
func Fit(lines []string, n int) []string {
	if n < 0 {
		n = 0
	}
	if len(lines) > n {
		return lines[:n]
	}
	for len(lines) < n {
		lines = append(lines, "")
	}
	return lines
}

// Width is the rendered width of a line in terminal cells. Escape sequences
// count as nothing, which is why a pane must measure with this rather than with
// len.
func Width(s string) int {
	return ansi.StringWidth(s)
}

// Truncate cuts a line to w cells, appending an ellipsis when it had to cut. It
// is ANSI-aware, so a styled line keeps its styling.
func Truncate(s string, w int) string {
	if w < 1 {
		return ""
	}
	return ansi.Truncate(s, w, "…")
}

// WrapWords breaks plain prose to width on word boundaries, collapsing the
// whitespace it finds. It is the wrapper for a sentence written in Go source --
// a modal's explanation of what an option means -- where the line breaks are
// the wrapper's business and there is no formatting to preserve.
//
// Wrap is the one to reach for otherwise: it keeps the newlines a string
// already has, which is what report text needs and what this deliberately
// throws away.
func WrapWords(s string, width int) []string {
	if width < 8 {
		width = 8
	}
	var out []string
	var line string
	for _, word := range strings.Fields(s) {
		switch {
		case line == "":
			line = word
		case Width(line)+1+Width(word) <= width:
			line += " " + word
		default:
			out = append(out, line)
			line = word
		}
	}
	if line != "" {
		out = append(out, line)
	}
	return out
}

// OneLine flattens an arbitrary string into a single rendered line: cleaned,
// with every run of whitespace -- newlines included -- collapsed to one space.
//
// It is what an error message goes through before it is drawn into a
// fixed-height layout. Truncating to a width does not remove a newline, so a
// three-line error rendered into a one-line footer is three rows, and the row
// budget the whole layout is built on is off by two. Clean alone is not enough:
// it normalizes a newline, it does not remove one.
func OneLine(s string) string {
	return strings.Join(strings.Fields(Clean(s)), " ")
}
