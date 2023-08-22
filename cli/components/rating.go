// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package components

import (
	"github.com/muesli/termenv"
	"go.mondoo.com/cnquery/cli/theme/colors"
	"go.mondoo.com/cnspec/policy"
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
