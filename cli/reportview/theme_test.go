// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reportview

import (
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/cli/components"
	"go.mondoo.com/cnspec/cli/reportmodel"
	"go.mondoo.com/cnspec/policy"
)

// The one rule the palette exists for: a status or severity color comes from
// components.DefaultScoreRatingColors, the rating palette the CLI already uses.
// Picking one by hand is how cli/progress and the launcher ended up disagreeing
// about what "high" looks like.
func TestColorsComeFromTheRatingPalette(t *testing.T) {
	for _, sev := range AllSeverities {
		require.Equal(t,
			components.DefaultScoreRatingColors.LipglossColor(sev),
			SeverityStyle(sev).GetForeground(),
			"severity %q", sev)
	}

	for _, s := range AllStatuses {
		require.Equal(t,
			components.DefaultScoreRatingColors.LipglossColor(StatusRating(s)),
			StatusStyle(s).GetForeground(),
			"status %q", s)
	}
}

// All six outcomes have to stay distinguishable. Pass, fail, error and skip get
// four different colors; unscored and unknown share the unrated color, which is
// right -- both mean "nobody scored this" -- but neither is ever painted as a
// pass.
func TestStatusColorsDistinguishOutcomes(t *testing.T) {
	require.Equal(t, policy.ScoreRatingTextNone, StatusRating(reportmodel.StatusPass))
	require.Equal(t, policy.ScoreRatingTextCritical, StatusRating(reportmodel.StatusFail))
	require.Equal(t, policy.ScoreRatingTextError, StatusRating(reportmodel.StatusError))
	require.Equal(t, policy.ScoreRatingTextSkip, StatusRating(reportmodel.StatusSkipped))
	require.Equal(t, policy.ScoreRatingTextUnrated, StatusRating(reportmodel.StatusUnscored))
	require.Equal(t, policy.ScoreRatingTextUnrated, StatusRating(reportmodel.StatusUnknown))

	pass := StatusStyle(reportmodel.StatusPass).GetForeground()
	for _, s := range []reportmodel.Status{
		reportmodel.StatusFail, reportmodel.StatusError,
		reportmodel.StatusSkipped, reportmodel.StatusUnscored, reportmodel.StatusUnknown,
	} {
		require.NotEqual(t, pass, StatusStyle(s).GetForeground(), "%q must not look like a pass", s)
	}
}

// Badges are fixed width so that a column of them lines up.
func TestBadgeWidths(t *testing.T) {
	for _, s := range AllStatuses {
		require.Equal(t, statusLabelWidth, ansi.StringWidth(ansi.Strip(StatusLabel(s))), "status %q", s)
	}
	for _, sev := range AllSeverities {
		require.Equal(t, severityBadgeWidth, ansi.StringWidth(ansi.Strip(SeverityBadge(sev))), "severity %q", sev)
	}
	require.Equal(t, severityBadgeWidth, ansi.StringWidth(SeverityBadge("")), "an unknown severity is blank, not narrow")
}

// Every status glyph is exactly one cell. This is not a style rule: a pane of
// this viewer assumes one rendered line is one terminal row of a known width, so
// a glyph the terminal draws two cells wide pushes the row past the rect it was
// measured for and the frame drifts.
//
// It is measured rather than eyeballed because the runes that look right are
// exactly the ones that are not safe. An emoji-presentation glyph -- ✅, ❌, and
// any of the emoji sequences reportmodel.Status.Icon returns -- is two cells,
// and ⚠, ✔ and ✘ are one cell by the standard and two on a terminal that
// renders them as emoji anyway. The glyphs used here are two text-presentation
// marks and three ASCII characters, none of which any font can widen.
func TestStatusGlyphsAreOneCell(t *testing.T) {
	for _, s := range AllStatuses {
		require.Equal(t, StatusGlyphWidth, ansi.StringWidth(statusGlyph(s)), "status %q", s)
		require.Equal(t, StatusGlyphWidth, ansi.StringWidth(ansi.Strip(StatusGlyph(s))), "status %q", s)
	}
	// A status nobody has heard of still gets its one cell rather than none.
	require.Equal(t, StatusGlyphWidth, ansi.StringWidth(statusGlyph(reportmodel.Status("WAT"))))

	// The emoji this replaced, for the record: two cells, which is why it is
	// not what a row draws.
	require.Equal(t, 2, ansi.StringWidth(StatusIcon(reportmodel.StatusPass)))
}

// The six outcomes stay six. Pass, fail, error, skipped, unscored and unknown
// mean different things -- a check that could not run is not a check that
// passed, and a check nobody scored is not a check that failed -- so no two of
// them may share a glyph. reportmodel's own emoji already spends ℹ️ on two of
// them, which is one reason the pane does not use it.
func TestTheStatusGlyphKeepsTheOutcomesApart(t *testing.T) {
	seen := map[string]reportmodel.Status{}
	for _, s := range AllStatuses {
		g := statusGlyph(s)
		require.NotContains(t, seen, g, "%q and %q would both draw %q", seen[g], s, g)
		seen[g] = s
	}
	require.Len(t, seen, len(AllStatuses))

	// And the shape says which side of the line the outcome is on before the
	// color does: a mark is a verdict, punctuation is the absence of one.
	require.Equal(t, "✓", statusGlyph(reportmodel.StatusPass))
	require.Equal(t, "✗", statusGlyph(reportmodel.StatusFail))
	for _, s := range []reportmodel.Status{
		reportmodel.StatusError, reportmodel.StatusSkipped,
		reportmodel.StatusUnscored, reportmodel.StatusUnknown,
	} {
		require.NotContains(t, "✓✗", statusGlyph(s), "%s reached no verdict", s)
	}
}

// The glyph is colored out of the shared rating palette, exactly as the word it
// replaced was. A pane that picked its own color for an outcome would be the
// fourth idea in this repo of what "high" looks like; see the note at the top of
// theme.go.
func TestTheStatusGlyphTakesItsColorFromTheStatusStyle(t *testing.T) {
	for _, s := range AllStatuses {
		require.Equal(t, StatusStyle(s).Render(statusGlyph(s)), StatusGlyph(s), "status %q", s)
	}
}
