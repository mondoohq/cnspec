// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reportdoc

import (
	"testing"

	"go.mondoo.com/cnspec/policy"
)

func TestRiskFromSeverityLabelLandsInBand(t *testing.T) {
	for _, tc := range []struct {
		label string
		want  Band
	}{
		{"CRITICAL", BandCritical}, {"HIGH", BandHigh}, {"MEDIUM", BandMedium},
		{"MODERATE", BandMedium}, {"LOW", BandLow}, {"NEGLIGIBLE", BandLow},
		{"NONE", BandZero}, {"", BandZero}, {"nonsense", BandZero},
	} {
		if got := BandOf(RiskFromSeverityLabel(tc.label)); got != tc.want {
			t.Errorf("%q -> band %v, want %v", tc.label, got, tc.want)
		}
	}
}

func TestCheckRatingUsesTheCheckScore(t *testing.T) {
	codeID := &policy.Score{Type: policy.ScoreType_Result, Value: 0}
	scores := map[string]*policy.Score{
		"code-id": codeID,
		"//check": {Type: policy.ScoreType_Result, Value: 40},
	}

	rated := CheckRating(scores, "//check", codeID)
	if rated.Value != 40 || rated.Type != policy.ScoreType_Result {
		t.Errorf("failing check rated %d (%v), want 40", rated.Value, rated.Type)
	}
	if codeID.Value != 0 {
		t.Error("the code id's score was changed")
	}

	if got := CheckRating(scores, "//missing", codeID); got != codeID {
		t.Error("a check without a score of its own should keep the code id's score")
	}

	pass := &policy.Score{Type: policy.ScoreType_Result, Value: 100}
	if got := CheckRating(scores, "//check", pass); got != pass {
		t.Error("a passing result should be kept as it is")
	}
	errored := &policy.Score{Type: policy.ScoreType_Error}
	if got := CheckRating(scores, "//check", errored); got != errored {
		t.Error("an errored result should be kept as it is")
	}
}
