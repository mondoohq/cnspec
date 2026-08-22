// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/require"
)

// A carriage return inside a rendered line makes ansi.StringWidth disagree with
// the terminal, which is how a panel ends up one column wider than the layout
// believes. reporter.NewLineCharacter is "\r\n" on Windows, so this is the
// everyday case there, not an exotic one.
func TestCleanRemovesCarriageReturns(t *testing.T) {
	require.Equal(t, "a\nb", Clean("a\r\nb"))
	require.Equal(t, "a\nb", Clean("a\rb"))
	require.Equal(t, "a    b", Clean("a\tb"))
	require.Equal(t, "plain", Clean("plain"))

	for _, ln := range Lines("one\r\ntwo\r\n") {
		require.NotContains(t, ln, "\r")
	}
}

func TestLines(t *testing.T) {
	require.Empty(t, Lines(""))
	require.Equal(t, []string{"one"}, Lines("one"))
	require.Len(t, Lines("a\nb\nc"), 3)
}

// Wrap must never return a line wider than the limit, whatever it is given: a
// long word, a URL, an MRN, or a paragraph that already has newlines in it.
func TestWrapNeverExceedsWidth(t *testing.T) {
	inputs := []string{
		"the quick brown fox jumps over the lazy dog",
		"//policy.api.mondoo.com/policies/mondoo-linux-security/queries/mondoo-linux-security-ensure-secure-permissions",
		"first line\r\nsecond line that is quite a lot longer than the first one",
		strings.Repeat("x", 300),
		"",
	}
	for _, in := range inputs {
		for _, w := range []int{1, 5, 20, 40, 80} {
			for i, ln := range Wrap(in, w) {
				require.LessOrEqual(t, ansi.StringWidth(ln), w,
					"width %d, line %d of %q", w, i, in)
				require.NotContains(t, ln, "\n")
			}
		}
	}
}

func TestFitIsExact(t *testing.T) {
	require.Len(t, Fit([]string{"a", "b", "c"}, 5), 5)
	require.Len(t, Fit([]string{"a", "b", "c"}, 2), 2)
	require.Empty(t, Fit([]string{"a"}, 0))
	require.Empty(t, Fit(nil, -3))
	require.Equal(t, []string{"", ""}, Fit(nil, 2))
}

func TestTruncate(t *testing.T) {
	require.Equal(t, "", Truncate("hello", 0))
	require.Equal(t, "hello", Truncate("hello", 10))
	require.LessOrEqual(t, ansi.StringWidth(Truncate("hello world", 5)), 5)
}

func TestScrollClamps(t *testing.T) {
	var s Scroll
	s.Move(10, 3, 10)
	require.Equal(t, 0, s.Off, "everything fits, so there is nothing to scroll")

	s = Scroll{}
	s.Move(100, 50, 10)
	require.Equal(t, 40, s.Off)
	s.Move(-1000, 50, 10)
	require.Equal(t, 0, s.Off)

	// A window that grows past the end of the list scrolls back rather than
	// leaving a view of nothing.
	s = Scroll{Off: 40}
	require.Equal(t, 0, s.Apply(50, 50))
}

func TestScrollEnsureVisible(t *testing.T) {
	s := Scroll{}
	s.EnsureVisible(30, 100, 10)
	require.Equal(t, 21, s.Off, "the row must be the last visible one")
	s.EnsureVisible(30, 100, 10)
	require.Equal(t, 21, s.Off, "a row already in view does not scroll")
	s.EnsureVisible(5, 100, 10)
	require.Equal(t, 5, s.Off, "scrolling up puts the row at the top")
	s.EnsureVisible(99, 100, 10)
	require.Equal(t, 90, s.Off)
}

func TestPosition(t *testing.T) {
	require.Equal(t, "0", Position(0, 0, 10))
	require.Equal(t, "7", Position(3, 7, 10), "everything fits, so only the total is worth saying")
	require.Equal(t, "4/70", Position(3, 70, 10))
}

// WrapWords is the plain-prose wrapper. It breaks between words and never
// inside one, so every line it returns is within the limit unless a single word
// was already over it -- which is the right trade for the prose it is for, and
// the reason Wrap (which hard-breaks a URL or an MRN) is a separate function.
func TestWrapWordsBreaksBetweenWords(t *testing.T) {
	inputs := []string{
		"Findings are uploaded, so they appear in the console, feed compliance reports, and are visible to your team.",
		"Nothing\nleaves\tthis   computer.",
		"",
	}
	for _, in := range inputs {
		for _, w := range []int{8, 20, 40, 80} {
			for i, ln := range WrapWords(in, w) {
				require.NotContains(t, ln, "\n", "width %d, line %d of %q", w, i, in)
				if words := strings.Fields(ln); len(words) > 1 {
					require.LessOrEqual(t, ansi.StringWidth(ln), w,
						"width %d, line %d of %q", w, i, in)
				}
			}
		}
	}

	// A word of its own on a line, over the limit, is left over the limit
	// rather than cut: the caller truncates to the pane, and a mid-word break
	// here would be a second place deciding that.
	require.Equal(t, []string{strings.Repeat("y", 200)}, WrapWords(strings.Repeat("y", 200), 20))

	// A width below the floor is raised to it rather than producing a column of
	// single letters.
	require.Equal(t, WrapWords("a bb ccc dddd", 8), WrapWords("a bb ccc dddd", 1))
}

// The two wrappers are not interchangeable, and the difference is the reason
// both exist: Wrap keeps the newlines a string already has, WrapWords collapses
// them. Report text needs the first; a sentence written in Go source needs the
// second.
func TestWrapKeepsNewlinesAndWrapWordsDoesNot(t *testing.T) {
	require.Equal(t, []string{"one", "two"}, Wrap("one\ntwo", 20))
	require.Equal(t, []string{"one two"}, WrapWords("one\ntwo", 20))
}

// OneLine is what stands between an error string somebody else wrote and a
// fixed-height layout. Truncating to a width does not remove a newline, which is
// how a three-line error made a 24-row terminal render 26 rows.
func TestOneLineFlattensEverything(t *testing.T) {
	require.Equal(t, "a b c", OneLine("a\nb\nc"))
	require.Equal(t, "a b", OneLine("a\r\n\r\nb"))
	require.Equal(t, "a b", OneLine("  a\t\tb  "))
	require.Equal(t, "", OneLine("\n\n"))
	require.Equal(t, "plain", OneLine("plain"))

	// Clean on its own is not enough, and that is the trap: it normalizes a
	// newline rather than removing one.
	require.Contains(t, Clean("a\r\nb"), "\n")
	require.NotContains(t, OneLine("a\r\nb"), "\n")
}
