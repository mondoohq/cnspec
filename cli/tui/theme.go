// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package tui

import (
	"github.com/charmbracelet/lipgloss"
	"go.mondoo.com/cnspec/cli/components"
	"go.mondoo.com/cnspec/policy"
)

// One palette for every terminal UI cnspec draws, in two halves.
//
// # Chrome
//
// Violet is the brand accent and marks whatever has focus; cyan is the command
// line and the key hints; the greys are the three weights of body text. These
// are frame furniture rather than meaning, so they are named here as literals
// and nowhere else. The launcher and the report viewer each carried their own
// copy of these seven colors -- identical in name and value, which is how it
// stayed unnoticed -- and two copies of a palette is one copy too many.
//
// None of the styles below carries a margin. A margin adds a line the layout
// arithmetic cannot see, and every panel here is sized in exact lines.
var (
	// ColAccent is the focus accent.
	ColAccent = lipgloss.Color("141")
	// ColAccentD is the dimmer violet an unfocused border is drawn in.
	ColAccentD = lipgloss.Color("97")
	// ColCyan is the command line and the key names in a hint.
	ColCyan = lipgloss.Color("87")
	// ColInk is text drawn on top of a colored band.
	ColInk = lipgloss.Color("16")
	// ColText, ColDim and ColFaint are the three weights of body text.
	ColText  = lipgloss.Color("252")
	ColDim   = lipgloss.Color("245")
	ColFaint = lipgloss.Color("240")
)

// Ratings is the one rating palette in this repo, and the second half of the
// answer: every color that carries *meaning* -- a severity, an outcome, an
// install state -- resolves through it rather than being picked by eye.
//
// It was picked by eye three times before this. cli/progress/todolist.go
// hand-copied the map with a comment saying it matched
// cli/components/rating.go, and the launcher chose green 78, amber 214 and red
// 203 against the palette's 78, 212 and 204 -- matching on one and missing on
// two, which is the signature of hand-picking rather than of a decision. A
// comment claiming two tables agree is an import that was not written.
var Ratings = components.DefaultScoreRatingColors

// The three semantic colors a chrome element needs: something is fine,
// something wants attention, something is wrong. They are the rating palette's
// own NONE, HIGH and CRITICAL, so a green dot in the launcher and a green
// severity in the viewer are the same green.
var (
	ColGood = Ratings.LipglossColor(policy.ScoreRatingTextNone)
	ColWarn = Ratings.LipglossColor(policy.ScoreRatingTextHigh)
	ColBad  = Ratings.LipglossColor(policy.ScoreRatingTextCritical)
)

var (
	// StyleText is the default body text of a pane.
	StyleText = lipgloss.NewStyle().Foreground(ColText)
	// StyleDim is secondary text: counts, platform names, paths.
	StyleDim = lipgloss.NewStyle().Foreground(ColDim)
	// StyleFaint is tertiary text: rules, section labels, hints.
	StyleFaint = lipgloss.NewStyle().Foreground(ColFaint)
	// StyleAccent marks the focused thing.
	StyleAccent = lipgloss.NewStyle().Foreground(ColAccent).Bold(true)
	// StyleKey renders a key name in a hint.
	StyleKey = lipgloss.NewStyle().Foreground(ColCyan)
	// StyleLabel is a small-caps section label inside a pane.
	StyleLabel = lipgloss.NewStyle().Foreground(ColFaint).Bold(true)

	// StyleGood, StyleWarn and StyleBad are the semantic colors as text styles.
	StyleGood = lipgloss.NewStyle().Foreground(ColGood)
	StyleWarn = lipgloss.NewStyle().Foreground(ColWarn)
	StyleBad  = lipgloss.NewStyle().Foreground(ColBad)

	// BandSelected is the full-width band a selected row is drawn as, in the
	// focused pane; BandInactive is the same row in a pane without focus. Use
	// them with Bar, which is what keeps the band exactly one line wide.
	BandSelected = lipgloss.NewStyle().Foreground(ColInk).Background(ColAccent).Bold(true)
	BandInactive = lipgloss.NewStyle().Foreground(ColText).Background(lipgloss.Color("237"))
)

// BorderColor is the color of a panel border, which is both UIs' primary signal
// for where the keys will land.
func BorderColor(focused bool) lipgloss.Color {
	if focused {
		return ColAccent
	}
	return ColAccentD
}

// HintSep separates two key hints in a footer.
const HintSep = "   "

// Kbd renders one key hint, e.g. `↑/↓ move`.
func Kbd(key, label string) string {
	return StyleKey.Render(key) + " " + StyleFaint.Render(label)
}
