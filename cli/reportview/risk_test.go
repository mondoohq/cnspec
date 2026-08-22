// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reportview

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/cli/components"
	"go.mondoo.com/cnspec/cli/reporter"
	"go.mondoo.com/cnspec/cli/reportmodel"
	"go.mondoo.com/cnspec/cli/tui"
	"go.mondoo.com/cnspec/policy"
)

// meter is what a meter looks like with the colors taken off, which is also what
// it looks like on a terminal that has none.
func meter(score *policy.Score) string {
	return ansi.Strip(RiskMeter(score))
}

// result builds a scored result. Completion matters: a result with none is a
// score policy.Score.Rating itself calls unrated, and RiskOf agrees.
func result(value uint32) *policy.Score {
	return &policy.Score{
		Type: policy.ScoreType_Result, Value: value,
		DataCompletion: 100, ScoreCompletion: 100,
	}
}

// --- the bands --------------------------------------------------------------

// The bands are reporter.RiskSeverityLabel's, and the dots are one per band. The
// table is written out in full rather than computed, so that a change to either
// ladder has to be made here as well as there.
func TestRiskMeterDrawsTheBand(t *testing.T) {
	for _, tc := range []struct {
		risk int32
		want string
	}{
		{100, "100 ●●●●"},
		{92, " 92 ●●●●"},
		{90, " 90 ●●●●"},
		{89, " 89 ●●●○"},
		{78, " 78 ●●●○"},
		{70, " 70 ●●●○"},
		{69, " 69 ●●○○"},
		{51, " 51 ●●○○"},
		{40, " 40 ●●○○"},
		{39, " 39 ●○○○"},
		{22, " 22 ●○○○"},
		{5, "  5 ●○○○"},
		{1, "  1 ●○○○"},
		{0, "  0 ○○○○"},
	} {
		require.Equal(t, tc.want, ansi.Strip(riskMeter(tc.risk, true)), "risk %d", tc.risk)
	}

	require.Equal(t, "  -     ", ansi.Strip(riskMeter(0, false)),
		"a thing with no score says so, in the number's column")
}

// The dot count is not decoration on top of the color: it is the band, said
// again, so the meter survives a monochrome terminal, NO_COLOR, a piped capture
// and a reader who cannot tell the red from the amber. Every risk value from 0
// to 100 is checked against the band reporter puts it in.
func TestDotCountEncodesTheBandWithoutColor(t *testing.T) {
	dots := map[string]int{
		policy.ScoreRatingTextCritical: 4,
		policy.ScoreRatingTextHigh:     3,
		policy.ScoreRatingTextMedium:   2,
		policy.ScoreRatingTextLow:      1,
		policy.ScoreRatingTextNone:     0,
	}
	for risk := int32(0); risk <= 100; risk++ {
		band := reporter.RiskSeverityLabel(risk)
		want, ok := dots[band]
		require.True(t, ok, "risk %d fell in band %q, which has no dot count", risk, band)

		// Stripped, i.e. exactly what a terminal with no colors shows.
		plain := ansi.Strip(riskMeter(risk, true))
		require.Equal(t, want, strings.Count(plain, riskFilled),
			"risk %d is %s and must fill %d dots: %q", risk, band, want, plain)
		require.Equal(t, riskDotCount-want, strings.Count(plain, riskHollow), "risk %d", risk)
	}
}

// The color is the band's, out of components.DefaultScoreRatingColors -- the
// same palette the CLI's scorecard draws with. A fourth hand-picked severity
// palette in this repo would be one too many; see the note at the top of
// theme.go.
func TestRiskColorComesFromTheRatingPalette(t *testing.T) {
	for risk := int32(0); risk <= 100; risk++ {
		require.Equal(t,
			components.DefaultScoreRatingColors.LipglossColor(reporter.RiskSeverityLabel(risk)),
			riskStyle(risk).GetForeground(), "risk %d", risk)
	}
}

// --- unscored is not zero ---------------------------------------------------

// The distinction the whole file exists for. "Nobody scored this" and "this
// scored clean" are different facts and are drawn differently: a dash against
// four hollow dots. This package keeps pass, fail, error, skipped and unscored
// as five things, and a risk meter is not the place to quietly merge two of
// them.
func TestUnscoredAndZeroRiskAreDifferent(t *testing.T) {
	require.Equal(t, "  0 ○○○○", meter(result(100)), "a check that ran and came back clean")
	require.Equal(t, "  -     ", meter(nil), "a check nobody scored")
	require.NotEqual(t, meter(result(100)), meter(nil))
}

// Every score type that carries no value is refused, because policy.Score.Value
// is 0 for all of them and 100-minus-the-value would call each one a confident
// CRITICAL. This is the trap reportmodel warns about where it stores Check.Risk.
func TestRiskOfRefusesAScoreWithNoValue(t *testing.T) {
	for _, tc := range []struct {
		name  string
		score *policy.Score
	}{
		{"nil", nil},
		{"unknown", &policy.Score{Type: policy.ScoreType_Unknown}},
		{"error", &policy.Score{Type: policy.ScoreType_Error, DataCompletion: 100, ScoreCompletion: 100}},
		{"skip", &policy.Score{Type: policy.ScoreType_Skip, DataCompletion: 100, ScoreCompletion: 100}},
		{"unscored", &policy.Score{Type: policy.ScoreType_Unscored, DataCompletion: 100, ScoreCompletion: 100}},
		{"out of scope", &policy.Score{Type: policy.ScoreType_OutOfScope}},
		{"disabled", &policy.Score{Type: policy.ScoreType_Disabled}},
		{"a result nothing completed", &policy.Score{Type: policy.ScoreType_Result}},
	} {
		risk, ok := RiskOf(tc.score)
		require.False(t, ok, "%s carries no risk to state", tc.name)
		require.Zero(t, risk)
		require.Equal(t, "  -     ", meter(tc.score), "%s", tc.name)
	}

	risk, ok := RiskOf(result(31))
	require.True(t, ok)
	require.EqualValues(t, 69, risk, "risk is the inverse of the score value")
}

// A score value outside 0..100 is not something cnspec writes, but it is
// something a hand-edited json-full holds, and a negative risk would render as a
// four-cell number in a three-cell column.
func TestRiskIsClampedToTheScale(t *testing.T) {
	risk, ok := RiskOf(result(255))
	require.True(t, ok)
	require.Zero(t, risk)
	require.Equal(t, RiskMeterWidth, tui.Width(meter(result(255))))
}

// --- geometry ---------------------------------------------------------------

// Every meter is the same width and exactly one row, whatever it says. That is
// what lets a column of them line up and what lets rowLine budget for one.
func TestRiskMeterIsOneRowAndFixedWidth(t *testing.T) {
	cases := []string{meter(nil)}
	for risk := int32(0); risk <= 100; risk++ {
		cases = append(cases, ansi.Strip(riskMeter(risk, true)))
	}
	for _, m := range cases {
		require.Equal(t, RiskMeterWidth, tui.Width(m), "%q", m)
		require.NotContains(t, m, "\n", "%q", m)
	}
	// Both glyphs are ambiguous-width runes. x/ansi resolves them as one cell,
	// which is the assumption every width above rests on.
	require.Equal(t, 1, tui.Width(riskFilled))
	require.Equal(t, 1, tui.Width(riskHollow))
}

// --- where they land --------------------------------------------------------

// The check page's status row is the tree's row with the words attached: the
// same glyph in the same first cell, the same four dots for the severity, and
// FAIL and CRIT beside them as the legend for both. The fixture supplies each of
// the three cases: a check that failed, one that passed and one that errored
// before it could score anything.
func TestCheckStatusRowSpeaksTheTreesVocabulary(t *testing.T) {
	for _, tc := range []struct {
		title  string
		status reportmodel.Status
		want   string
	}{
		{"Ensure secure permissions on /etc/group- are set", reportmodel.StatusFail, "✗ FAIL      ●●●● CRIT"},
		{"Ensure X Window System is not installed", reportmodel.StatusPass, "✓ PASS      ●●●● CRIT"},
		// No impact configured, so no dots and no badge -- see
		// TestAnErroredCheckIsNotBadgedNone below.
		{"Only use strong Ciphers", reportmodel.StatusError, "! ERROR"},
	} {
		st, check := stateFor(t, fixtureUbuntu, tc.title)
		require.Equal(t, tc.status, check.Status, tc.title)

		_, lines := renderDetail(st, 100)
		row := ansi.Strip(lines[statusRowAt(t, lines)])
		require.True(t, strings.HasPrefix(row, tc.want),
			"%s: row is %q, want it to open with %q", tc.title, row, tc.want)
	}
}

// A passing critical check still shows four dots, exactly as it does in the
// tree: the count is what the check is *worth*, and that does not move when the
// check does well. What moves is the color -- the band's when the severity was
// realized, grey when it was not.
func TestTheStatusRowsDotsAreSeverityNotOutcome(t *testing.T) {
	styled(t)
	crit := policy.ScoreRatingTextCritical

	for _, tc := range []struct {
		title string
		want  string
	}{
		{"Ensure secure permissions on /etc/group- are set", SeverityStyle(crit).Render("●●●●")},
		{"Ensure X Window System is not installed", tui.StyleFaint.Render("●●●●")},
	} {
		st, _ := stateFor(t, fixtureUbuntu, tc.title)
		_, lines := renderDetail(st, 100)
		require.Contains(t, lines[statusRowAt(t, lines)], tc.want, tc.title)
	}
}

// The check row carries no meter, and that is the change this row was made for:
// its four dots are the severity, the way the tree's are, and a second four-dot
// group next to them meaning the realized risk is the confusion the tree exists
// to avoid.
//
// Nothing is lost by dropping it. RiskOf states a number only for a
// ScoreType_Result score, and reporter defines PASS as such a score with value
// 100 and FAIL as one below it -- so "risk 0" was precisely PASS, "risk 100" was
// precisely FAIL, and "-" was precisely every other outcome. The glyph and the
// word say all three, and say them without a number to decode.
func TestTheCheckPageCarriesNoMeter(t *testing.T) {
	st := NewState(loadReport(t, fixtureUbuntu))
	asset := st.Report.Assets[0]
	require.NotEmpty(t, asset.Checks)

	for _, check := range asset.Checks {
		st.SelectCheck(asset, nil, check)
		_, lines := renderDetail(st, 100)
		row := ansi.Strip(lines[statusRowAt(t, lines)])
		require.NotContains(t, row, "risk", check.Title)
		// One dot group, not two: at most riskDotCount dots on the row.
		dots := strings.Count(row, riskFilled) + strings.Count(row, riskHollow)
		require.LessOrEqual(t, dots, riskDotCount, "%s: %q", check.Title, row)
	}
}

// An errored check must not read as a rated one. reportmodel derives Severity
// from Impact and an unset impact is 0, which RiskSeverityLabel reads as NONE --
// so without the HasImpact gate this page would badge four checks that nobody
// rated as rated harmless, and draw four hollow dots saying so.
//
// The row also cannot reach the older trap it used to have to dodge: the raw
// Risk reportmodel stores for these checks is 100, a confident CRITICAL for a
// check that proved nothing, and the page no longer states a risk at all.
func TestAnErroredCheckIsNotBadgedNone(t *testing.T) {
	st, check := stateFor(t, fixtureUbuntu, "Only use strong Ciphers")
	require.Equal(t, reportmodel.StatusError, check.Status)
	require.False(t, check.HasImpact, "the fixture's ssh checks configure no impact")
	require.NotNil(t, check.Score)
	require.EqualValues(t, policy.ScoreType_Error, check.Score.Type)
	require.EqualValues(t, 100, check.Risk, "100 minus a value that is not there")

	_, lines := renderDetail(st, 100)
	row := ansi.Strip(lines[statusRowAt(t, lines)])
	require.Equal(t,
		"! "+tui.PadRight(string(reportmodel.StatusError), statusLabelWidth)+
			"  "+strings.Repeat(" ", SeverityMarkWidth), row,
		"the severity half of the row is spent and blank, so the impact column does not move")
	require.NotContains(t, row, riskFilled)
	require.NotContains(t, row, riskHollow)
	require.NotContains(t, row, policy.ScoreRatingTextNone)
}

// The alignment the row was asked for, stated as an equality rather than as a
// pair of literals that happen to match: for every check of the fixture, the
// glyph and the dot group the tree drew are the glyph and the dot group the
// detail page draws. One vocabulary, two panes, and neither can be changed
// without the other.
func TestTheDetailRowRepeatsTheTreeRow(t *testing.T) {
	p, st := treeFor(t, fixtureUbuntu)
	p.foldAll(st, true)

	res := p.Render(st, tui.Rect{X: 0, Y: 0, W: 100, H: 200})
	rows := p.build(st)
	require.Equal(t, len(rows), len(res.Lines))

	checks := 0
	for i, r := range rows {
		if r.kind != nodeCheck {
			continue
		}
		checks++

		treeRow := ansi.Strip(res.Lines[i])
		st.SelectCheck(r.asset, r.policy, r.check)
		_, lines := renderDetail(st, 100)
		detailRow := ansi.Strip(lines[statusRowAt(t, lines)])

		require.Equal(t, firstRune(treeRow), firstRune(detailRow),
			"%s: the tree's glyph and the page's must be the same cell", r.check.Title)
		require.Equal(t, dotRuns(treeRow), dotRuns(detailRow),
			"%s: the tree's dots and the page's must be the same dots", r.check.Title)
	}
	require.Equal(t, 24, checks, "every check of the fixture")
}

// The row is a grid: every field is fixed-width, so the impact lands in the same
// column on every check and arrowing down the tree does not make the row shuffle
// under the cursor. That is what "line up in a column" buys.
func TestTheStatusRowIsAColumn(t *testing.T) {
	st := NewState(loadReport(t, fixtureUbuntu))
	asset := st.Report.Assets[0]

	col := -1
	rated := 0
	for _, check := range asset.Checks {
		st.SelectCheck(asset, nil, check)
		_, lines := renderDetail(st, 100)
		row := ansi.Strip(lines[statusRowAt(t, lines)])

		// The two fixed groups, whatever they say.
		require.GreaterOrEqual(t, tui.Width(row), StatusMarkWidth+2+SeverityMarkWidth, check.Title)

		byteAt := strings.Index(row, "impact ")
		if !check.HasImpact {
			require.Equal(t, -1, byteAt, "%s states no impact", check.Title)
			continue
		}
		rated++
		// In cells, not bytes: the dots are three bytes each.
		at := tui.Width(row[:byteAt])
		if col < 0 {
			col = at
		}
		require.Equal(t, col, at, "%s: the impact moved column", check.Title)
	}
	require.Equal(t, 20, rated, "the fixture's twenty rated checks")
	require.Equal(t, StatusMarkWidth+2+SeverityMarkWidth+2, col,
		"the impact starts right after the two fixed groups and their separators")
}

// firstRune is the first non-space cell of a row, which on both a tree row and a
// status row is the status glyph.
func firstRune(s string) string {
	for _, r := range strings.TrimLeft(s, " ") {
		return string(r)
	}
	return ""
}

// dotRuns is every maximal run of meter or severity dots in a row, so two rows
// can be compared on what their dots say without caring where they sit.
func dotRuns(s string) []string {
	var res []string
	var cur strings.Builder
	for _, r := range s {
		if string(r) == riskFilled || string(r) == riskHollow {
			cur.WriteRune(r)
			continue
		}
		if cur.Len() > 0 {
			res = append(res, cur.String())
			cur.Reset()
		}
	}
	if cur.Len() > 0 {
		res = append(res, cur.String())
	}
	return res
}

// An asset's page states its risk, which is the number a reader wants when the
// question is how bad this one machine is. The meter stays here -- and only here
// -- because an asset row has no severity for it to collide with, and because a
// rolled-up risk really is a fact of its own rather than the status restated.
// The tree makes the same split; see TestOnlyAssetRowsCarryAMeter.
func TestAssetPageCarriesTheRisk(t *testing.T) {
	st := NewState(loadReport(t, fixtureUbuntu))
	asset := st.Report.Assets[0]
	st.SelectAsset(asset)

	out := ansi.Strip(strings.Join(detailBody(st, 100), "\n"))
	require.Contains(t, out, "✗ FAIL      risk 100 ●●●●")
}

// And an asset that never scanned has no risk to state. Fifteen of them here,
// every one a "-": there is no report to take a number from, which is what the
// section below the row goes on to say in words.
func TestAnUnscannedAssetStatesNoRisk(t *testing.T) {
	st := NewState(loadReport(t, fixtureK8s))
	require.Len(t, st.Report.Assets, 15)

	for _, asset := range st.Report.Assets {
		require.False(t, asset.Scanned())
		st.SelectAsset(asset)
		out := ansi.Strip(strings.Join(detailBody(st, 100), "\n"))
		require.Contains(t, out, "! ERROR     risk   -", asset.Name)
		require.Contains(t, out, "THIS ASSET WAS NOT SCANNED", asset.Name)
	}
}

// The tree puts the *meter* on asset rows and nowhere else. A policy row spends
// its right-hand slot on the findings tally, and a check row draws its severity
// on the left -- see treePane.parts for both.
//
// The check row now draws dots too, and this is where the pane has to keep the
// two apart. They are different facts: the meter is the realized risk of a
// scored thing and is 0 for a pass, the check's dots are what the check is worth
// and do not move when it passes. What tells them apart on screen is the number:
// a meter is "100 ●●●●" and a severity is bare dots, and they sit at opposite
// ends of the row. This pins that a check row never grows a number in front of
// its dots and that a policy row has neither.
func TestOnlyAssetRowsCarryAMeter(t *testing.T) {
	p, st := treeFor(t, fixtureUbuntu)
	p.foldAll(st, true)

	rows := p.build(st)
	lines := treeText(t, p, st)
	require.Len(t, lines, len(rows))

	assets, checks := 0, 0
	for i, r := range rows {
		switch r.kind {
		case nodeAsset:
			assets++
			require.Contains(t, lines[i], "100 ●●●●", "asset row %d", i)
		case nodeCheck:
			checks++
			// A severity is dots and nothing else. Every risk the meter can
			// state is a number followed by a space and then the dots, so a row
			// with no digit before its dots cannot be carrying a meter.
			require.NotRegexp(t, `[0-9-] [`+riskFilled+riskHollow+`]`, lines[i],
				"check row %d must not read as a risk meter", i)
		default:
			require.NotContains(t, lines[i], riskFilled, "row %d is a %s row", i, r.kind.tag())
			require.NotContains(t, lines[i], riskHollow, "row %d is a %s row", i, r.kind.tag())
		}
	}
	require.Equal(t, 1, assets)
	require.Equal(t, 24, checks)
}

// --- severity, drawn with the same dots -------------------------------------

// The count is the band and nothing but the band: four for critical, down to
// none for a check somebody rated as harmless. Whether the check passed or
// failed does not enter into it -- a passing critical check is still a critical
// check -- so every band is checked in both states and both give the same dots.
func TestSeverityDotsCountTheBand(t *testing.T) {
	for _, tc := range []struct{ severity, want string }{
		{policy.ScoreRatingTextCritical, "●●●●"},
		{policy.ScoreRatingTextHigh, "●●●○"},
		{policy.ScoreRatingTextMedium, "●●○○"},
		{policy.ScoreRatingTextLow, "●○○○"},
		{policy.ScoreRatingTextNone, "○○○○"},
	} {
		require.Equal(t, tc.want, ansi.Strip(SeverityDots(tc.severity, true)),
			"%s, realized", tc.severity)
		require.Equal(t, tc.want, ansi.Strip(SeverityDots(tc.severity, false)),
			"%s, not realized -- the band does not shrink because the check passed",
			tc.severity)
	}

	// And the count agrees with the ladder the risk meter uses, rather than
	// being a second one written out by hand above.
	for _, sev := range AllSeverities {
		require.Equal(t, riskBandDots(sev),
			strings.Count(ansi.Strip(SeverityDots(sev, true)), riskFilled), sev)
	}
}

// A check that states no severity at all is blank, and a check rated NONE is
// four hollow dots. The two are different claims -- "nobody said what this is
// worth" against "somebody said it is worth nothing" -- and this is the same
// distinction the meter makes between "-" and ○○○○, for the same reason.
func TestNoSeverityAndZeroSeverityAreDifferent(t *testing.T) {
	require.Equal(t, "○○○○", ansi.Strip(SeverityDots(policy.ScoreRatingTextNone, false)))
	require.Equal(t, "    ", ansi.Strip(SeverityDots("", false)))
	require.NotEqual(t, SeverityDots(policy.ScoreRatingTextNone, false), SeverityDots("", false))

	// reportmodel is where the trap is: an unset impact is 0, and 0 reads as
	// NONE, so Severity alone cannot tell the two apart. HasImpact can, and
	// checkSeverity is the reading of the pair.
	_, none := stateFor(t, fixtureUbuntu, "Only use strong Ciphers")
	require.False(t, none.HasImpact, "the fixture's ssh checks configure no impact")
	require.Equal(t, policy.ScoreRatingTextNone, none.Severity, "which still reads as NONE")
	require.Equal(t, "", checkSeverity(none), "and must not be drawn as a rating")

	_, rated := stateFor(t, fixtureUbuntu, "Ensure X Window System is not installed")
	require.True(t, rated.HasImpact)
	require.Equal(t, policy.ScoreRatingTextCritical, checkSeverity(rated))
}

// The color is the only thing the outcome moves, and it moves exactly where
// Status.IsFinding draws the line: a fail and an error get the band's color out
// of the shared rating palette, a pass, a skip and an unscored check go grey.
// That is what makes a screen of passes read as a wall of grey with the findings
// as the only color on it.
//
// The grey is tui.StyleFaint rather than a rating color on purpose: the palette has
// no entry for "this severity did not happen", and borrowing one would be
// claiming a rating the check does not have.
func TestSeverityColorIsTheBandOnlyWhenItWasRealized(t *testing.T) {
	styled(t)

	for _, sev := range AllSeverities {
		require.Equal(t,
			SeverityStyle(sev).Render(ansi.Strip(SeverityDots(sev, true))),
			SeverityDots(sev, true),
			"a realized %s is drawn in the palette's color for %s", sev, sev)
		require.Equal(t,
			tui.StyleFaint.Render(ansi.Strip(SeverityDots(sev, false))),
			SeverityDots(sev, false),
			"an unrealized %s is grey", sev)
	}

	// Which is a real difference on a terminal that has colors, and no
	// difference at all in what the dots say.
	crit := policy.ScoreRatingTextCritical
	require.NotEqual(t, SeverityDots(crit, true), SeverityDots(crit, false))
	require.Equal(t, ansi.Strip(SeverityDots(crit, true)), ansi.Strip(SeverityDots(crit, false)))

	// IsFinding is asked rather than re-derived, so the split is fail and error
	// against everything else, in one place, for the whole viewer.
	require.True(t, reportmodel.StatusFail.IsFinding())
	require.True(t, reportmodel.StatusError.IsFinding())
	for _, s := range []reportmodel.Status{
		reportmodel.StatusPass, reportmodel.StatusSkipped,
		reportmodel.StatusUnscored, reportmodel.StatusUnknown,
	} {
		require.False(t, s.IsFinding(), "%s is not a finding, so its severity is grey", s)
	}
}

// Every severity is the same four cells, whatever it says, so the column lines
// up and the title starts in the same place on every row.
func TestSeverityDotsAreOneRowAndFixedWidth(t *testing.T) {
	styled(t)
	for _, sev := range append(append([]string{}, AllSeverities...), "", "NOT A BAND") {
		for _, realized := range []bool{true, false} {
			d := SeverityDots(sev, realized)
			require.Equal(t, SeverityDotsWidth, tui.Width(d), "%q realized=%v", sev, realized)
			require.NotContains(t, ansi.Strip(d), "\n", "%q", sev)
		}
	}
	require.Equal(t, riskDotCount, SeverityDotsWidth,
		"a severity and a meter's dots line up, which is why they are the same count")
}

// The whole claim, end to end and in the pane: the ubuntu fixture's failing and
// passing critical checks draw the same four dots, and differ only in color.
// Counting the band down for a pass would be drawing risk, which is a different
// fact and lives in the detail pane.
func TestAPassingCriticalCheckKeepsItsFourDots(t *testing.T) {
	styled(t)
	p, st := treeFor(t, fixtureUbuntu)
	p.foldAll(st, true)

	res := p.Render(st, tui.Rect{X: 0, Y: 0, W: 74, H: 40})
	rows := p.build(st)

	var failed, passed string
	for i, r := range rows {
		if r.kind != nodeCheck || checkSeverity(r.check) != policy.ScoreRatingTextCritical {
			continue
		}
		switch r.check.Status {
		case reportmodel.StatusFail:
			failed = res.Lines[i]
		case reportmodel.StatusPass:
			passed = res.Lines[i]
		}
	}
	require.NotEmpty(t, failed)
	require.NotEmpty(t, passed)

	require.Contains(t, ansi.Strip(failed), "✗ ●●●●")
	require.Contains(t, ansi.Strip(passed), "✓ ●●●●",
		"a passing critical check is still a critical check")
	require.Contains(t, failed, SeverityStyle(policy.ScoreRatingTextCritical).Render("●●●●"))
	require.Contains(t, passed, tui.StyleFaint.Render("●●●●"), "and its dots are grey")
}

// The errored ssh checks of the fixture configure no impact at all, so their
// severity column is four spaces -- not four hollow dots, which would be the
// pane claiming somebody rated them harmless.
func TestACheckWithNoSeverityDrawsNoDots(t *testing.T) {
	p, st := treeFor(t, fixtureUbuntu)
	p.foldAll(st, true)

	res := p.Render(st, tui.Rect{X: 0, Y: 0, W: 74, H: 40})
	rows := p.build(st)

	unrated := 0
	for i, r := range rows {
		if r.kind != nodeCheck || r.check.HasImpact {
			continue
		}
		unrated++
		line := ansi.Strip(res.Lines[i])
		require.NotContains(t, line, riskFilled, "row %d", i)
		require.NotContains(t, line, riskHollow, "row %d", i)
		// The cells are still spent, so the title still starts where every
		// other title does.
		require.Contains(t, line, "! "+strings.Repeat(" ", SeverityDotsWidth)+" ", "row %d", i)
	}
	require.Equal(t, 4, unrated, "the fixture's four errored ssh checks")
}

// A check row has to be readable with the color taken off entirely -- a
// monochrome terminal, NO_COLOR, a piped `script` capture, a reader who cannot
// tell the red from the amber. Stripped, the glyph still names the outcome and
// the filled dots still count the band, for every check in the fixture.
func TestACheckRowReadsWithNoColorAtAll(t *testing.T) {
	styled(t) // colors on, so that stripping them is a real strip
	p, st := treeFor(t, fixtureUbuntu)
	p.foldAll(st, true)

	res := p.Render(st, tui.Rect{X: 0, Y: 0, W: 74, H: 40})
	rows := p.build(st)

	seen := map[reportmodel.Status]int{}
	for i, r := range rows {
		if r.kind != nodeCheck {
			continue
		}
		plain := ansi.Strip(res.Lines[i])
		require.NotContains(t, plain, "\x1b", "row %d still carries an escape", i)
		seen[r.check.Status]++

		// The outcome, from the glyph alone.
		require.Equal(t, statusGlyph(r.check.Status), strings.Fields(plain)[0],
			"row %d: %q", i, plain)
		// The band, from the filled dots alone.
		require.Equal(t, riskBandDots(checkSeverity(r.check)),
			strings.Count(plain, riskFilled), "row %d: %q", i, plain)
		// And the rest of the row is the title, in full: what the two columns
		// gave up in width went here.
		require.True(t, strings.HasSuffix(plain, r.check.Title), "row %d: %q", i, plain)
	}
	require.Equal(t, map[reportmodel.Status]int{
		reportmodel.StatusPass: 18, reportmodel.StatusFail: 2, reportmodel.StatusError: 4,
	}, seen, "the fixture's spread of outcomes")
}

// The meter is a right-aligned tag like any other, so a pane too narrow for one
// drops it rather than letting it eat the asset name. rowLine already had that
// rule; this pins that the meter obeys it.
func TestANarrowTreeDropsTheMeterBeforeTheName(t *testing.T) {
	p, st := treeFor(t, fixtureUbuntu)

	for _, w := range []int{1, 2, 8, 12, 19, 20, 21, 40, 74} {
		res := p.Render(st, tui.Rect{X: 0, Y: 0, W: w, H: 20})
		for i, ln := range res.Lines {
			require.LessOrEqual(t, tui.Width(ln), w, "w=%d row %d: %q", w, i, ansi.Strip(ln))
		}
		if w <= 20 {
			require.NotContains(t, ansi.Strip(res.Lines[0]), riskFilled,
				"w=%d has no room for a meter", w)
		}
	}
	require.Contains(t, ansi.Strip(p.Render(st, tui.Rect{X: 0, Y: 0, W: 74, H: 20}).Lines[0]), "100 ●●●●")
}

// statusRowAt is the index of a detail page's status row: the first row that
// opens with a status glyph and the word for it, which is the row directly under
// the title block on both the check page and the asset page.
func statusRowAt(t *testing.T, lines []string) int {
	t.Helper()
	for i, ln := range lines {
		s := ansi.Strip(ln)
		for _, st := range AllStatuses {
			if strings.HasPrefix(s, statusGlyph(st)+" "+string(st)) {
				return i
			}
		}
	}
	t.Fatalf("no status row in %q", lines)
	return -1
}
