// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/require"
)

func ansiStrip(s string) string { return ansi.Strip(s) }

// termSizes covers the sizes that actually break things: the 80x24 default, a
// window too narrow for two panes, and one short enough that the header, the
// body and the footer cannot all have the room they want.
var termSizes = []struct{ w, h int }{
	{80, 24}, {100, 30}, {120, 40}, {200, 60}, {76, 24}, {70, 24}, {60, 15}, {40, 10}, {30, 6}, {20, 3},
}

// Panel is the promise the whole layout rests on: exactly h lines of exactly w
// cells, whatever it is handed. Content that is too short, too long, too wide or
// full of escape sequences all have to come out the same shape, because the
// caller has already budgeted the rows.
func TestPanelIsExactlyTheSizeItWasAsked(t *testing.T) {
	contents := [][]string{
		nil,
		{"one"},
		{"one", "two", "three", "four", "five", "six", "seven", "eight", "nine", "ten", "eleven", "twelve"},
		{strings.Repeat("x", 400)},
		{StyleAccent.Render("styled") + " " + StyleDim.Render(strings.Repeat("wide ", 50))},
	}
	for _, content := range contents {
		for _, s := range termSizes {
			out := Panel(StyleAccent.Render("Title"), StyleFaint.Render("9/99"), content, s.w, s.h, ColAccent)
			lines := strings.Split(out, "\n")
			require.Len(t, lines, s.h, "%dx%d", s.w, s.h)
			for i, ln := range lines {
				require.LessOrEqual(t, Width(ln), s.w, "%dx%d line %d", s.w, s.h, i)
			}
		}
	}
}

// The box has a floor, and it is the border. A rounded box cannot be drawn in
// fewer than three lines or two columns, so below that Panel draws the smallest
// box there is rather than something that is not a box. Nothing asks it to: Dims
// clamps BodyH at 3 and a pane is never narrower than a third of a terminal.
// The floor is asserted so that a caller that does ask knows what it gets.
func TestPanelHasAFloorOfOneBorderedRow(t *testing.T) {
	for _, s := range []struct{ w, h int }{{1, 1}, {2, 2}, {0, 0}} {
		lines := strings.Split(Panel("t", "s", []string{"x"}, s.w, s.h, ColAccent), "\n")
		require.Len(t, lines, BorderLines+1, "%dx%d", s.w, s.h)
	}
}

// A title too long for the top edge must not widen the box; the status is
// dropped first, and then the title is, so the frame stays a frame.
func TestPanelTopSurvivesAnOversizedTitle(t *testing.T) {
	bs := lipgloss.NewStyle().Foreground(ColAccentD)
	for _, w := range []int{2, 4, 8, 20, 80} {
		out := PanelTop(strings.Repeat("title ", 20), strings.Repeat("status ", 5), w, bs)
		require.LessOrEqual(t, Width(out), w, "width %d", w)
	}
}

func TestBarIsExactlyOneLineOfTheGivenWidth(t *testing.T) {
	for _, w := range []int{1, 5, 40} {
		out := Bar("a row that is definitely longer than five cells", w, BandSelected)
		require.NotContains(t, out, "\n")
		require.Equal(t, w, Width(ansi.Strip(out)), "width %d", w)
	}
}

func TestPadRight(t *testing.T) {
	require.Equal(t, "ab  ", PadRight("ab", 4))
	require.Equal(t, "abcd", PadRight("abcd", 2), "a string already that wide is left alone")
	require.Equal(t, 4, Width(ansi.Strip(PadRight(StyleDim.Render("ab"), 4))),
		"padding measures cells, not bytes")
}

func TestInnerDimensionsNeverGoBelowOne(t *testing.T) {
	for _, n := range []int{-5, 0, 1, 2, 3, 4, 5, 40} {
		require.GreaterOrEqual(t, InnerWidth(n), 1, "InnerWidth(%d)", n)
		require.GreaterOrEqual(t, InnerHeight(n), 1, "InnerHeight(%d)", n)
	}
	require.Equal(t, 36, InnerWidth(40))
	require.Equal(t, 38, InnerHeight(40))
}

// Dims is the only place a terminal size becomes pane sizes, so the invariants
// the renderers assume have to hold for every size, not just the comfortable
// ones.
func TestDimsKeepsThePanesInsideTheTerminal(t *testing.T) {
	for _, s := range termSizes {
		l := Dims(s.w, s.h)
		require.GreaterOrEqual(t, l.ListH, 1, "%dx%d", s.w, s.h)
		require.GreaterOrEqual(t, l.BodyH, 3, "%dx%d", s.w, s.h)
		if l.TwoPane {
			require.Equal(t, s.w, l.LeftW+1+l.RightW, "%dx%d: the two panes and the gutter are the width", s.w, s.h)
			require.Equal(t, l.LeftW+1, l.RightX, "%dx%d", s.w, s.h)
			require.GreaterOrEqual(t, s.w, MinTwoPaneWidth)
		} else {
			require.Equal(t, s.w, l.LeftW, "%dx%d: one pane is the whole width", s.w, s.h)
			require.Less(t, s.w, MinTwoPaneWidth)
		}
	}
}

// A click has to land on the thing that was drawn there. Zones are appended in
// render order, so the last one wins where two overlap.
func TestHitZoneTakesTheTopmost(t *testing.T) {
	l := Layout{Zones: []Zone{
		{Rect: Rect{X: 0, Y: 0, W: 10, H: 3}, Kind: ZoneEntry, Idx: 1},
		{Rect: Rect{X: 2, Y: 1, W: 4, H: 1}, Kind: ZoneField, Idx: 7},
	}}
	require.Equal(t, ZoneField, l.HitZone(3, 1).Kind)
	require.Equal(t, 7, l.HitZone(3, 1).Idx)
	require.Equal(t, ZoneEntry, l.HitZone(9, 1).Kind)
	require.Equal(t, ZoneNone, l.HitZone(50, 50).Kind, "a click outside every zone does nothing")
}

func TestRectHit(t *testing.T) {
	r := Rect{X: 2, Y: 3, W: 4, H: 2}
	require.True(t, r.Hit(2, 3))
	require.True(t, r.Hit(5, 4))
	require.False(t, r.Hit(6, 4), "the right edge is exclusive")
	require.False(t, r.Hit(5, 5), "the bottom edge is exclusive")
	require.False(t, r.Hit(1, 3))
}
