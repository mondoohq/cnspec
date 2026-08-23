// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reportdoc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnspec/policy"
)

// TestOutcomeOf pins the collapse of every policy.ScoreType, because this is the
// one place it happens and each report format now trusts it rather than looking
// at the score type again for itself.
//
// Every constant of the enum is listed, plus the two values that are not
// constants: a score type cnspec does not know, and no score at all. Both have to
// read as Unknown - a format that treated them as a verdict would report a check
// that never answered as passing or failing.
func TestOutcomeOf(t *testing.T) {
	tests := []struct {
		name  string
		score *policy.Score
		want  Outcome
		label string
	}{
		{"result with a full score", &policy.Score{Type: policy.ScoreType_Result, Value: 100}, OutcomePass, "PASS"},
		{"result below a full score", &policy.Score{Type: policy.ScoreType_Result, Value: 99}, OutcomeFail, "FAIL"},
		{"result with no score", &policy.Score{Type: policy.ScoreType_Result, Value: 0}, OutcomeFail, "FAIL"},
		{"error", &policy.Score{Type: policy.ScoreType_Error}, OutcomeError, "ERROR"},
		{"skip", &policy.Score{Type: policy.ScoreType_Skip}, OutcomeSkipped, "SKIPPED"},
		{"out of scope", &policy.Score{Type: policy.ScoreType_OutOfScope}, OutcomeSkipped, "SKIPPED"},
		{"disabled", &policy.Score{Type: policy.ScoreType_Disabled}, OutcomeSkipped, "SKIPPED"},
		{"unscored", &policy.Score{Type: policy.ScoreType_Unscored}, OutcomeUnscored, "UNSCORED"},
		{"unknown", &policy.Score{Type: policy.ScoreType_Unknown}, OutcomeUnknown, "UNKNOWN"},
		{"a score type cnspec does not know", &policy.Score{Type: 1 << 20}, OutcomeUnknown, "UNKNOWN"},
		{"no score at all", nil, OutcomeUnknown, "UNKNOWN"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			outcome := OutcomeOf(tc.score)
			assert.Equal(t, tc.want, outcome)
			assert.Equal(t, tc.label, outcome.Label())
		})
	}
}

// TestOutcomeZeroValueIsUnknown pins that the zero Outcome is Unknown. A struct
// field left unset must not read as a passing check.
func TestOutcomeZeroValueIsUnknown(t *testing.T) {
	var zero Outcome
	assert.Equal(t, OutcomeUnknown, zero)
	assert.Equal(t, "UNKNOWN", zero.Label())
}
