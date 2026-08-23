// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reportview

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"go.mondoo.com/cnspec/cli/reportmodel"
	"go.mondoo.com/cnspec/cli/tui"
	"go.mondoo.com/cnspec/policy"
)

// Every color a finding is drawn in comes from here, and everything here comes
// from tui.Ratings -- which is components.DefaultScoreRatingColors, the one
// rating palette in this repo that is already expressed as lipgloss colors. A
// pane must not pick its own color for a status or a severity:
// cli/progress/todolist.go hand-copied that map once already and the launcher
// picked its three by eye, and three disagreeing ideas of what "high" looks
// like is two too many.
//
// The chrome the frame is drawn in -- the accent, the three text weights, the
// bands, the border -- lives in cli/tui and is shared with the launcher, so the
// two programs look like one. What is left here is the half that is about a
// report: how an outcome and a severity turn into a color, a glyph and a badge.

// StatusRating maps a check outcome onto the rating label whose color it should
// be drawn in. It is the only place the two vocabularies meet.
//
// The mapping keeps all six outcomes apart, because they mean different things:
// a check that could not run is not a check that passed, and a check nobody
// scored is not a check that failed.
func StatusRating(s reportmodel.Status) string {
	switch s {
	case reportmodel.StatusPass:
		return policy.ScoreRatingTextNone
	case reportmodel.StatusFail:
		return policy.ScoreRatingTextCritical
	case reportmodel.StatusError:
		return policy.ScoreRatingTextError
	case reportmodel.StatusSkipped:
		return policy.ScoreRatingTextSkip
	default: // UNSCORED, UNKNOWN
		return policy.ScoreRatingTextUnrated
	}
}

// StatusStyle is the lipgloss style a status is drawn in.
func StatusStyle(s reportmodel.Status) lipgloss.Style {
	return tui.Ratings.LipglossStyle(StatusRating(s))
}

// SeverityStyle is the lipgloss style a severity label is drawn in. The label is
// one of the policy.ScoreRatingText* values, which is what
// reportmodel.Check.Severity carries.
func SeverityStyle(severity string) lipgloss.Style {
	return tui.Ratings.LipglossStyle(severity)
}

// StatusIcon is the emoji for a status, from the model so the viewer and the
// reporter show the same glyph. It is two cells wide for every status.
//
// That width is why no pane of this viewer draws it. A rendered line here is
// exactly one terminal row of a known width, and an emoji is the one thing that
// makes ansi.StringWidth and the terminal disagree: ✅ and ❌ are
// Emoji_Presentation and measure two, ⚠️ and ⏭️ carry a variation selector that
// some terminals honor and some ignore, and a glyph that is two cells here and
// one cell there is a pane that is one column too wide on somebody else's
// machine. StatusGlyph is what a row uses; this stays because the emoji is the
// vocabulary of the reporter's markdown and json output, where width is nobody's
// problem, and a caller wanting to match those should have it from one place.
func StatusIcon(s reportmodel.Status) string {
	return s.Icon()
}

// StatusGlyph renders a status as a single colored cell, which is what a list
// of hundreds of rows can afford to spend on an outcome.
//
// # Why a mark for two of them and punctuation for the rest
//
// The six glyphs are not six arbitrary picks. A check either reached a verdict
// or it did not, and the shape says which:
//
//	✓  PASS      the check ran and the answer was yes
//	✗  FAIL      the check ran and the answer was no
//	!  ERROR     it could not run, and that needs a human
//	»  SKIPPED   it was stepped over on purpose
//	i  UNSCORED  it reported, and nothing scores it
//	?  UNKNOWN   nobody can say
//
// A tick and a cross are verdicts. The other four are punctuation because none
// of them is a verdict -- an errored check has *proved* nothing, which is
// exactly what reportmodel says about it -- and a reader who learns that one
// rule can read a row they have never seen before. Colour then separates the
// four further, but the shape alone already says "no answer here", which is the
// thing a monochrome terminal has to be able to tell.
//
// The six stay six. They are six different facts and collapsing them into a
// tick and a cross would say a check nobody scored is a check that failed.
//
// # Why not ⚠ for the error
//
// Because it is not reliably one cell. U+26A0 has Emoji_Presentation=No, so
// ansi.StringWidth calls it narrow and the geometry tests would pass, but a
// terminal that renders it in emoji presentation anyway draws it two cells wide
// and every row below shifts. The same goes for ✔ and ✘ (U+2714/U+2718), which
// are the emoji-presentation cousins of the two marks used here. ASCII cannot
// be widened by anybody's font, so the three glyphs that carry the most risk of
// being drawn as emoji are ASCII.
func StatusGlyph(s reportmodel.Status) string {
	return StatusStyle(s).Render(statusGlyph(s))
}

// statusGlyph is the uncolored cell, which is what the width and the
// monochrome-readability tests measure.
func statusGlyph(s reportmodel.Status) string {
	switch s {
	case reportmodel.StatusPass:
		return "✓"
	case reportmodel.StatusFail:
		return "✗"
	case reportmodel.StatusError:
		return "!"
	case reportmodel.StatusSkipped:
		return "»"
	case reportmodel.StatusUnscored:
		return "i"
	default: // UNKNOWN, and anything reportmodel grows later
		return "?"
	}
}

// StatusGlyphWidth is the rendered width of every glyph. It is one, and
// TestStatusGlyphsAreOneCell holds it there: the pane's geometry budgets for a
// column of these, and a two-cell glyph would push a row past the rect it was
// measured for.
const StatusGlyphWidth = 1

// StatusLabel renders a status as a fixed-width colored word, so a column of
// them lines up: "PASS    ", "ERROR   ", "UNSCORED".
//
// The detail pane and the plain-text fallback still spell the outcome out. They
// are pages being read rather than lists being scanned, the word names the
// outcome exactly, and it survives a grep of a captured session -- see the note
// on statusRow in detail.go.
func StatusLabel(s reportmodel.Status) string {
	return StatusStyle(s).Render(tui.PadRight(string(s), statusLabelWidth))
}

// statusLabelWidth is the width of the longest status, UNSCORED.
const statusLabelWidth = 8

// StatusMark is the glyph and the word together: the cell a list row spends on
// an outcome, with the word that says what it meant printed next to it.
//
// It is what the detail pane's status rows draw, and it is the only place in the
// viewer where the two vocabularies meet on purpose. A reader arrives at that
// row from a tree full of "✗" and leaves it knowing that "✗" is FAIL; a row that
// carried only the word would send them back no wiser, and one that carried only
// the glyph would leave the tree unexplained anywhere.
func StatusMark(s reportmodel.Status) string {
	return StatusGlyph(s) + " " + StatusLabel(s)
}

// StatusMarkWidth is the rendered width of every StatusMark, so a row built out
// of one starts its next field in the same column whatever the outcome was.
const StatusMarkWidth = StatusGlyphWidth + 1 + statusLabelWidth

// SeverityMark is SeverityDots and SeverityBadge together, the severity half of
// the same idea: the tree's four dots with the band named beside them.
//
// severity is one of the policy.ScoreRatingText* labels and empty for a check
// that states none, realized is Status.IsFinding -- both exactly as SeverityDots
// takes them, so the dots here and the dots in the tree cannot drift apart. An
// empty severity draws blank on both halves rather than "NONE": nobody rating a
// check is not somebody rating it harmless.
func SeverityMark(severity string, realized bool) string {
	return SeverityDots(severity, realized) + " " + SeverityBadge(severity)
}

// SeverityMarkWidth is the rendered width of every SeverityMark.
const SeverityMarkWidth = SeverityDotsWidth + 1 + severityBadgeWidth

// SeverityBadge renders a severity as a short colored tag: "CRIT", "HIGH",
// "MED", "LOW", "NONE". It is always severityBadgeWidth cells wide, and blank
// for a check with no severity at all.
func SeverityBadge(severity string) string {
	short := map[string]string{
		policy.ScoreRatingTextCritical: "CRIT",
		policy.ScoreRatingTextHigh:     "HIGH",
		policy.ScoreRatingTextMedium:   "MED",
		policy.ScoreRatingTextLow:      "LOW",
		policy.ScoreRatingTextNone:     "NONE",
	}[severity]
	if short == "" {
		return strings.Repeat(" ", severityBadgeWidth)
	}
	return SeverityStyle(severity).Render(tui.PadRight(short, severityBadgeWidth))
}

// severityBadgeWidth is the width of the longest severity badge.
const severityBadgeWidth = 4
