// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reportview

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"go.mondoo.com/cnspec/cli/reporter"
	"go.mondoo.com/cnspec/cli/tui"
	"go.mondoo.com/cnspec/policy"
)

// The risk meter: a number and four dots, the way Mondoo's web UI draws risk.
//
//	100 ●●●●   critical
//	 78 ●●●○   high
//	 51 ●●○○   medium
//	 22 ●○○○   low
//	  0 ○○○○   none -- scored, and nothing came of it
//	  -        no score at all
//
// # The bands are cnspec's, not this file's
//
// reporter.RiskSeverityLabel already decides where one band ends and the next
// begins (>= 90 critical, >= 70 high, >= 40 medium, >= 1 low, else none), and
// sarif, the CLI and reportmodel.Check.Severity all read it. A second ladder
// here would be a viewer that disagrees with the report it is viewing about
// whether a thing is high or critical, so the band is asked for rather than
// recomputed and the only decision left in this file is how many dots each band
// is worth.
//
// The color is the same story one layer down: SeverityStyle resolves the band
// through components.DefaultScoreRatingColors, which is the palette the CLI
// draws its scorecard with. See the note at the top of theme.go.
//
// # Why the dots carry the band too
//
// The count is not decoration on top of the color -- it is the same fact said a
// second way, so that the meter still reads on a monochrome terminal, under
// NO_COLOR, in a piped `script` capture, and to a reader who cannot tell the
// red from the amber. Four dots is critical whatever color they came out. That
// is also why the number is drawn: three signals, one fact.
//
// # No score is not zero risk
//
// A score with no value is drawn as "-", never as four hollow dots. The two are
// different facts and this package keeps its outcomes apart: a check that
// errored, one that was skipped and one nobody scored have no risk to state,
// while a check that ran and came back clean has a risk and it is zero. Reading
// them as the same thing is how a scan that did not happen gets mistaken for a
// clean bill of health.
//
// The trap is in the data rather than in the drawing. policy.Score.Value is 0
// for every score type that carries no value, so the arithmetic behind
// reporter.ScoreRisk -- 100 minus the value -- turns an errored check into risk
// 100, which is a confident CRITICAL for a check that proved nothing.
// reportmodel says as much where it stores Check.Risk ("read Risk together with
// Status rather than on its own"); RiskOf is that reading, in one place.

const (
	// riskDotCount is how many dots a meter has. Four is what the web UI uses,
	// and four bands above NONE is exactly what RiskSeverityLabel defines, so
	// each band adds one dot.
	riskDotCount = 4
	// riskFilled and riskHollow are both one cell wide in every terminal that
	// draws them (they are ambiguous-width, which x/ansi resolves as narrow --
	// TestRiskMeterIsOneRowAndFixedWidth pins it).
	riskFilled = "●"
	riskHollow = "○"
	// riskNoScore is what a thing with no score to state says instead of a
	// number.
	riskNoScore = "-"
	// riskNumWidth is the room the number gets: "100" is the widest risk there
	// is.
	riskNumWidth = 3
	// RiskMeterWidth is the rendered width of every meter, whatever it says. A
	// column of them lines up, and the geometry can budget for one.
	RiskMeterWidth = riskNumWidth + 1 + riskDotCount
)

// RiskOf is the realized risk of a score -- 0 for a check that came back clean,
// 100 for one that failed outright -- and whether the score states one at all.
//
// It is false for every score that carries no value: a nil score, an error, a
// skip, an unscored or unknown type, and a result whose completion is zero (the
// same case policy.Score.Rating treats as unrated). See the package comment
// above for why that gate is not optional.
func RiskOf(score *policy.Score) (int32, bool) {
	if score == nil || score.Type != policy.ScoreType_Result || score.Completion() == 0 {
		return 0, false
	}
	risk := reporter.ScoreRisk(score)
	// A score value above 100 is not a thing cnspec produces, but it is a thing
	// a hand-edited json-full can hold, and a negative risk would render as a
	// four-cell number in a three-cell column.
	return min(max(risk, 0), 100), true
}

// RiskMeter draws a score as the number and the dots, RiskMeterWidth cells wide.
// A score with no risk to state comes out as "-".
func RiskMeter(score *policy.Score) string {
	risk, ok := RiskOf(score)
	return riskMeter(risk, ok)
}

// riskMeter is RiskMeter over an already-resolved risk, which is what the tests
// walk the bands with.
func riskMeter(risk int32, ok bool) string {
	if !ok {
		// The dash sits in the number's column so that it lines up with the
		// numbers above and below it, and the dot column is left empty rather
		// than filled with hollows -- that is the whole point of the case.
		return tui.StyleFaint.Render(padLeftTo(riskNoScore, riskNumWidth)) +
			strings.Repeat(" ", RiskMeterWidth-riskNumWidth)
	}
	style := riskStyle(risk)
	filled := riskBandDots(riskBand(risk))
	return style.Render(padLeftTo(strconv.Itoa(int(risk)), riskNumWidth)) + " " +
		style.Render(strings.Repeat(riskFilled, filled)+
			strings.Repeat(riskHollow, riskDotCount-filled))
}

// riskBand is the severity band a risk falls in. It is reporter's decision, and
// asking for it is the whole of this package's agreement with the CLI about
// where high stops and critical starts.
func riskBand(risk int32) string {
	return reporter.RiskSeverityLabel(risk)
}

// riskStyle is the color a risk is drawn in: its band, resolved through the
// shared rating palette by SeverityStyle. Nothing here picks a color.
func riskStyle(risk int32) lipgloss.Style {
	return SeverityStyle(riskBand(risk))
}

// riskBandDots is how many of the four dots a band fills. One per band above
// NONE, and none for NONE itself: a scored thing with no risk shows four hollow
// dots, which is a statement, while a thing with no score shows no dots at all.
func riskBandDots(band string) int {
	switch band {
	case policy.ScoreRatingTextCritical:
		return 4
	case policy.ScoreRatingTextHigh:
		return 3
	case policy.ScoreRatingTextMedium:
		return 2
	case policy.ScoreRatingTextLow:
		return 1
	default: // NONE, and anything the reporter grows later
		return 0
	}
}

// --- severity, drawn with the same dots -------------------------------------
//
// A severity is what a check is *worth* -- how bad it would be if it failed --
// and that is a property of the check and not of how it did. So the dots say
// two things at once and they are deliberately different things:
//
//	severity   outcome     drawn
//	CRITICAL   FAIL        ●●●● in the critical color
//	CRITICAL   PASS        ●●●● in grey
//	HIGH       ERROR       ●●●○ in the error color's band
//	LOW        PASS        ●○○○ in grey
//	NONE       PASS        ○○○○ in grey
//	(none set) ERROR       blank
//
// # The count is the band, always
//
// A passing critical check is still a critical check, so it keeps its four
// dots. Counting the band down because the check happened to pass would be
// drawing risk, which the meter above already draws and which is 0 for a pass.
// The count comes from riskBandDots, the same ladder the meter uses, so the two
// cannot drift apart about where high stops and critical starts.
//
// # The color is whether that severity was realized
//
// reportmodel.Status.IsFinding already draws the line the reader cares about --
// fail and error on one side, pass, skipped and unscored on the other -- so it
// is asked rather than re-derived. A finding gets its band's color; everything
// else goes grey. What that buys is the only reason to do any of this: a pane of
// grey with a few colored clusters in it, sized by how much they matter, is read
// in one glance, and a pane of colored badges is read one row at a time.
//
// The grey is tui.StyleFaint and not a rating color on purpose. Every color that
// *states a severity* comes from the shared palette (see theme.go), but this one
// states the opposite -- that the severity did not happen -- and the palette has
// no entry for that, so borrowing one would be claiming a rating the check does
// not have. Grey is chrome, and it comes from the chrome palette.
//
// # Filled and hollow, so the band survives without color
//
// The dots are filled up to the band and hollow after it, exactly as the meter
// draws them. Grey against red is invisible on a monochrome terminal and to a
// good number of readers on a color one, but filled against hollow is a shape,
// and three filled dots is HIGH whatever came out of the terminal. The status
// glyph beside them carries the outcome the same way. Between them the row reads
// with the color stripped entirely, which is what
// TestACheckRowReadsWithNoColorAtAll holds.
//
// # No severity is not zero severity
//
// A check with no impact configured is drawn blank, and a check configured at
// impact 0 is drawn ○○○○. They are different claims: one is "nobody said what
// this is worth", the other is "somebody said it is worth nothing", and four
// hollow dots is the second. This is the same distinction the meter makes above
// between "-" and ○○○○, for the same reason, and the trap is in the same place:
// reportmodel derives Check.Severity from Check.Impact, which is 0 when no
// impact was set, so Severity reads NONE for both. Check.HasImpact is what tells
// them apart, and checkSeverity below is that reading.

// SeverityDotsWidth is the rendered width of a severity, whatever it says --
// the same four cells as the meter's dots, so the two line up in a pane that
// draws both.
const SeverityDotsWidth = riskDotCount

// SeverityDots draws a severity band as riskDotCount dots: filled up to the
// band, hollow after it, colored when the severity was realized and grey when it
// was not. An empty severity -- a check that states none at all -- is blank.
//
// severity is one of the policy.ScoreRatingText* labels, which is what
// reportmodel.Check.Severity carries; realized is Status.IsFinding.
func SeverityDots(severity string, realized bool) string {
	if severity == "" {
		return strings.Repeat(" ", SeverityDotsWidth)
	}
	style := tui.StyleFaint
	if realized {
		style = SeverityStyle(severity)
	}
	filled := riskBandDots(severity)
	return style.Render(strings.Repeat(riskFilled, filled) +
		strings.Repeat(riskHollow, riskDotCount-filled))
}

// riskField is the meter with the word in front of it, which is what a row of
// mixed facts needs: on a line that already says "impact 100", a bare number is
// a second number nobody can name.
func riskField(score *policy.Score) string {
	return tui.StyleFaint.Render("risk") + " " + RiskMeter(score)
}

// padLeftTo right-aligns a string in w cells, which is what a column of numbers
// wants and padTo (which pads on the right) is not.
func padLeftTo(s string, w int) string {
	if d := w - tui.Width(s); d > 0 {
		return strings.Repeat(" ", d) + s
	}
	return s
}
