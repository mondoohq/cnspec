// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reportdoc

import "go.mondoo.com/cnspec/policy"

// ScoreRisk is the inverse of a score value: 0 (no risk) to 100 (critical).
func ScoreRisk(score *policy.Score) int32 {
	if score == nil {
		return 0
	}
	return 100 - int32(score.Value)
}

// RiskSeverityLabel maps a cnspec risk value (0-100, the inverse of a score value)
// to the severity label cnspec uses everywhere else. The bands mirror
// policy.ScoreRatingsText:
//
//	90 .. 100 → CRITICAL
//	70 ..  89 → HIGH
//	40 ..  69 → MEDIUM
//	 1 ..  39 → LOW
//	        0 → NONE
func RiskSeverityLabel(risk int32) string {
	switch {
	case risk >= 90:
		return policy.ScoreRatingTextCritical
	case risk >= 70:
		return policy.ScoreRatingTextHigh
	case risk >= 40:
		return policy.ScoreRatingTextMedium
	case risk >= 1:
		return policy.ScoreRatingTextLow
	default:
		return policy.ScoreRatingTextNone
	}
}
