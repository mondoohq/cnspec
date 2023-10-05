// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package components

import (
	"bufio"
	"strconv"
	"strings"

	"github.com/muesli/termenv"
	"go.mondoo.com/cnquery/v9/cli/theme/colors"
	"go.mondoo.com/cnquery/v9/utils/stringx"
	"go.mondoo.com/cnspec/v9/policy"
)

const (
	scoreCardTitleBarCharAnsi  string = `▄`
	scoreCardIndicatorAnsi     string = `▄▄`
	scoreCardTitleBarCharAscii string = `-`
	scoreCardIndicatorAscii    string = `==`

	scoreA string = "   _\n" +
		"  /_\\\n" +
		" / _ \\\n" +
		"/_/ \\_\\"

	scoreB string = "  ___\n" +
		" | _ )\n" +
		" | _ \\\n" +
		" |___/"

	scoreC string = "  ___\n" +
		" / __|\n" +
		"| (__\n" +
		" \\___|"

	scoreD string = " ___\n" +
		"|   \\\n" +
		"| |) |\n" +
		"|___/"

	scoreU string = " _   _\n" +
		"| | | |\n" +
		"| |_| |\n" +
		" \\___/"

	scoreF string = " ___\n" +
		"| __|\n" +
		"| _|\n" +
		"|_|"

	scoreX string = "__  __\n" +
		"\\ \\/ /\n" +
		">  <\n" +
		"/_/\\_\\"

	scoreS string = " ___\n" +
		"/ __|\n" +
		"\\__ \\\n" +
		"|___/"
)

type ScoreCard struct {
	width   int
	height  int
	details bool
}

func NewScoreCard() *ScoreCard {
	return &ScoreCard{
		width:   30,
		height:  6,
		details: true,
	}
}

func NewMiniScoreCard() *ScoreCard {
	return &ScoreCard{
		width:   11,
		height:  6,
		details: false,
	}
}

func (m ScoreCard) Render(score *policy.Score) string {
	rating := score.Rating()
	ratingColor := DefaultRatingColors.Color(rating)
	achievementIndicators := 0

	const plusIndicator = "+"
	const minusIndicator = "-"
	subcategoryIndicator := ""

	scoreLetter := scoreU
	switch rating {
	case policy.ScoreRating_unrated:
		scoreLetter = scoreU
		achievementIndicators = 0
		subcategoryIndicator = ""
	case policy.ScoreRating_skip:
		scoreLetter = scoreS
		achievementIndicators = 0
		subcategoryIndicator = ""
	case policy.ScoreRating_aPlus:
		scoreLetter = scoreA
		achievementIndicators = 4
		subcategoryIndicator = plusIndicator
	case policy.ScoreRating_a:
		scoreLetter = scoreA
		achievementIndicators = 4
	case policy.ScoreRating_aMinus:
		scoreLetter = scoreA
		achievementIndicators = 4
		subcategoryIndicator = minusIndicator
	case policy.ScoreRating_bPlus:
		scoreLetter = scoreB
		achievementIndicators = 3
		subcategoryIndicator = plusIndicator
	case policy.ScoreRating_b:
		scoreLetter = scoreB
		achievementIndicators = 3
	case policy.ScoreRating_bMinus:
		scoreLetter = scoreB
		achievementIndicators = 3
		subcategoryIndicator = minusIndicator
	case policy.ScoreRating_cPlus:
		scoreLetter = scoreC
		achievementIndicators = 2
		subcategoryIndicator = plusIndicator
	case policy.ScoreRating_c:
		scoreLetter = scoreC
		achievementIndicators = 2
	case policy.ScoreRating_cMinus:
		scoreLetter = scoreC
		achievementIndicators = 2
		subcategoryIndicator = minusIndicator
	case policy.ScoreRating_dPlus:
		scoreLetter = scoreD
		achievementIndicators = 1
		subcategoryIndicator = plusIndicator
	case policy.ScoreRating_d:
		scoreLetter = scoreD
		achievementIndicators = 1
	case policy.ScoreRating_dMinus:
		scoreLetter = scoreD
		achievementIndicators = 1
		subcategoryIndicator = minusIndicator
	case policy.ScoreRating_failed:
		scoreLetter = scoreF
		achievementIndicators = 1
	case policy.ScoreRating_error:
		scoreLetter = scoreX
		achievementIndicators = 1
	}

	var layers []string

	scoreCardTitleBarChar := scoreCardTitleBarCharAnsi
	scoreCardIndicator := scoreCardIndicatorAnsi

	// fallback to ascii, important for Windows and Putty with remote connections to Linux
	if colors.Profile == termenv.Ascii {
		scoreCardTitleBarChar = scoreCardTitleBarCharAscii
		scoreCardIndicator = scoreCardIndicatorAscii
	}

	if m.details {
		layers = []string{
			// print big letter
			"\n" + stringx.Indent(2, scoreLetter),
			// add + or - to letter
			"\n\n" + strings.Repeat(" ", 8) + subcategoryIndicator,
			// determine indicator headerline
			stringx.Indent(1, strings.Repeat(scoreCardTitleBarChar, m.width-2)),
			// label for rating + score
			"\n\n" + strings.Repeat(" ", 11) + rating.CategoryLabel() + " " + strconv.Itoa(int(score.Value)) + "/100",
			// completion
			"\n\n\n" + strings.Repeat(" ", 11) + strconv.Itoa(int(score.Completion())) + "% complete",
			// achievement indicator
			"\n\n\n\n" + strings.Repeat(" ", 11) + strings.Repeat(scoreCardIndicator+" ", achievementIndicators),
		}
	} else {
		layers = []string{
			// print big letter
			"\n" + stringx.Indent(2, scoreLetter),
			// add + or - to letter
			"\n\n" + strings.Repeat(" ", 8) + subcategoryIndicator,
			// determine indicator headerline
			stringx.Indent(1, strings.Repeat(scoreCardTitleBarChar, m.width-2)),
		}
	}

	renderedCard := stringx.Overlay(
		NewPaper().Render(m.width, m.height), // render paper
		layers...,
	)

	// now we need to colorize the card
	b := strings.Builder{}

	// we need to colorize each line
	scanner := bufio.NewScanner(strings.NewReader(renderedCard))
	for scanner.Scan() {
		line := scanner.Text()
		colored := termenv.String(line).Foreground(ratingColor).String()
		b.WriteString(colored)
		b.WriteString("\n")
	}

	return b.String()
}

func NewMicroScoreCard() *MicroScoreCard {
	return &MicroScoreCard{}
}

type MicroScoreCard struct{}

func (m MicroScoreCard) Render(score *policy.Score) string {
	rating := score.Rating()
	ratingColor := DefaultRatingColors.Color(rating)

	// category code can be 1-2 chars, ensure we always render 4 chars
	cc := rating.Letter()
	if len(cc) == 1 {
		cc = cc + " "
	}
	cc = " " + cc + " "

	// return colored micro scorecard
	return termenv.String(cc).Foreground(ratingColor).String()
}
