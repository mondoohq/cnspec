// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package components

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"go.mondoo.com/cnquery/v11/cli/theme/colors"
	"go.mondoo.com/cnspec/v11/policy"
)

var DefaultRatingColors = NewRating(colors.DefaultColorTheme)

// Initializes a Rating
func NewRating(theme colors.Theme) Rating {
	policyRatingColorMapping := map[policy.ScoreRating]termenv.Color{
		policy.ScoreRating_unrated: theme.Unknown,
		policy.ScoreRating_aPlus:   theme.Good,
		policy.ScoreRating_a:       theme.Good,
		policy.ScoreRating_aMinus:  theme.Good,
		policy.ScoreRating_bPlus:   theme.Low,
		policy.ScoreRating_b:       theme.Low,
		policy.ScoreRating_bMinus:  theme.Low,
		policy.ScoreRating_cPlus:   theme.Medium,
		policy.ScoreRating_c:       theme.Medium,
		policy.ScoreRating_cMinus:  theme.Medium,
		policy.ScoreRating_dPlus:   theme.High,
		policy.ScoreRating_d:       theme.High,
		policy.ScoreRating_dMinus:  theme.High,
		policy.ScoreRating_failed:  theme.Critical,
		policy.ScoreRating_error:   theme.Critical,
	}

	return Rating{
		PolicyRatingColorMapping: policyRatingColorMapping,
	}
}

type Rating struct {
	// colors for policy ratings
	PolicyRatingColorMapping map[policy.ScoreRating]termenv.Color
}

func (t Rating) Color(r policy.ScoreRating) termenv.Color {
	c, ok := t.PolicyRatingColorMapping[r]
	if ok {
		return c
	}
	return t.PolicyRatingColorMapping[policy.ScoreRating_unrated]
}

// These are the new Score Rating labels and colors, once we move away
// from A-F scores, we can delete the above code.

var DefaultScoreRatingColors = NewScoreRating(colors.DefaultColorTheme)

// Initializes a Score Rating
func NewScoreRating(theme colors.Theme) ScoreRating {
	return ScoreRating{
		ScoreRatingColorMapping: map[string]termenv.Color{
			policy.ScoreRatingTextUnrated:  theme.Unknown,
			policy.ScoreRatingTextNone:     theme.Good,
			policy.ScoreRatingTextLow:      theme.Low,
			policy.ScoreRatingTextMedium:   theme.Medium,
			policy.ScoreRatingTextHigh:     theme.High,
			policy.ScoreRatingTextCritical: theme.Critical,
			policy.ScoreRatingTextError:    theme.Critical,
		},
		// TODO @afiune this should live in "go.mondoo.com/cnquery/v11/cli/theme/colors"
		ScoreRatingLipglossColorMapping: map[string]lipgloss.Color{
			policy.ScoreRatingTextUnrated:  lipgloss.Color("231"),
			policy.ScoreRatingTextNone:     lipgloss.Color("78"),
			policy.ScoreRatingTextLow:      lipgloss.Color("117"),
			policy.ScoreRatingTextMedium:   lipgloss.Color("75"),
			policy.ScoreRatingTextHigh:     lipgloss.Color("212"),
			policy.ScoreRatingTextCritical: lipgloss.Color("204"),
			policy.ScoreRatingTextError:    lipgloss.Color("210"),
		},
	}
}

type ScoreRating struct {
	// colors for policy score ratings
	ScoreRatingColorMapping         map[string]termenv.Color
	ScoreRatingLipglossColorMapping map[string]lipgloss.Color
}

func (t ScoreRating) Color(scoreRating string) termenv.Color {
	c, ok := t.ScoreRatingColorMapping[scoreRating]
	if ok {
		return c
	}
	return t.ScoreRatingColorMapping[policy.ScoreRatingTextUnrated]
}

func (t ScoreRating) LipglossColor(scoreRating string) lipgloss.Color {
	c, ok := t.ScoreRatingLipglossColorMapping[scoreRating]
	if ok {
		return c
	}
	return t.ScoreRatingLipglossColorMapping[policy.ScoreRatingTextUnrated]
}

func (t ScoreRating) LipglossStyle(scoreRating string) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.LipglossColor(scoreRating))
}
