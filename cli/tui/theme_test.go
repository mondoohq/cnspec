// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package tui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/cli/components"
	"go.mondoo.com/cnspec/policy"
)

// The rule this package exists to make enforceable: a colour that carries
// meaning comes out of the rating palette, never out of a literal.
//
// It was broken three ways before. cli/progress/todolist.go held a hand-copied
// map with a comment saying it matched cli/components/rating.go; the launcher
// picked green 78, amber 214 and red 203 against that palette's 78, 212 and 204.
// Matching on one colour and missing on two is the signature of hand-picking
// rather than of a decision, and it is exactly what a test can hold shut.
func TestSemanticColorsComeFromTheRatingPalette(t *testing.T) {
	require.Equal(t, components.DefaultScoreRatingColors, Ratings,
		"tui.Ratings must be the palette itself, not a copy of it")

	for _, tc := range []struct {
		name   string
		got    lipgloss.Color
		rating string
	}{
		{"good", ColGood, policy.ScoreRatingTextNone},
		{"warn", ColWarn, policy.ScoreRatingTextHigh},
		{"bad", ColBad, policy.ScoreRatingTextCritical},
	} {
		require.Equal(t, Ratings.LipglossColor(tc.rating), tc.got, "col%s", tc.name)
	}

	// And the three text styles are those three colours, so a caller reaching
	// for the style cannot end up somewhere else than a caller reaching for the
	// colour.
	require.Equal(t, ColGood, StyleGood.GetForeground())
	require.Equal(t, ColWarn, StyleWarn.GetForeground())
	require.Equal(t, ColBad, StyleBad.GetForeground())
}

// The three of them have to stay apart. "fine", "look at this" and "this is
// wrong" are three facts, and a palette that renders two of them the same has
// stopped carrying the third.
func TestSemanticColorsAreDistinct(t *testing.T) {
	require.NotEqual(t, ColGood, ColWarn)
	require.NotEqual(t, ColWarn, ColBad)
	require.NotEqual(t, ColGood, ColBad)
}

// No style in this package may carry a margin. A margin adds a line the layout
// arithmetic cannot see, which is how a panel that "obviously" fits ends up
// scrolling the terminal -- and the panels here are sized in exact lines.
func TestNoStyleCarriesAMargin(t *testing.T) {
	for name, st := range map[string]lipgloss.Style{
		"StyleText":    StyleText,
		"StyleDim":     StyleDim,
		"StyleFaint":   StyleFaint,
		"StyleAccent":  StyleAccent,
		"StyleKey":     StyleKey,
		"StyleLabel":   StyleLabel,
		"StyleGood":    StyleGood,
		"StyleWarn":    StyleWarn,
		"StyleBad":     StyleBad,
		"BandSelected": BandSelected,
		"BandInactive": BandInactive,
	} {
		require.Zero(t, st.GetMarginTop(), name)
		require.Zero(t, st.GetMarginBottom(), name)
		require.Zero(t, st.GetMarginLeft(), name)
		require.Zero(t, st.GetMarginRight(), name)
	}
}

func TestBorderColorFollowsFocus(t *testing.T) {
	require.Equal(t, ColAccent, BorderColor(true))
	require.Equal(t, ColAccentD, BorderColor(false))
	require.NotEqual(t, BorderColor(true), BorderColor(false),
		"the border is the only signal for where the keys will land")
}

func TestKbdPutsTheKeyFirst(t *testing.T) {
	require.Equal(t, "^r reporting", ansiStrip(Kbd("^r", "reporting")))
}
