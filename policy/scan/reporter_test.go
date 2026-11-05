// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
)

func TestLowerScore(t *testing.T) {
	rated := func(v uint32) *policy.Score {
		return &policy.Score{Type: policy.ScoreType_Result, Value: v, ScoreCompletion: 100}
	}
	low, high, tie := rated(40), rated(90), rated(40)

	cases := []struct {
		name    string
		current *policy.Score
		next    *policy.Score
		want    *policy.Score
	}{
		{name: "both nil", want: nil},
		{name: "first rated score is taken", next: low, want: low},
		{name: "nil next keeps current", current: low, want: low},
		{name: "lower next replaces current", current: high, next: low, want: low},
		{name: "higher next keeps current", current: low, next: high, want: low},
		{name: "equal value keeps current", current: tie, next: low, want: tie},
		{name: "unscored next is skipped", current: high, next: &policy.Score{Type: policy.ScoreType_Unscored, ScoreCompletion: 100}, want: high},
		{name: "unknown next is skipped", current: high, next: &policy.Score{Type: policy.ScoreType_Unknown}, want: high},
		{name: "incomplete result is unrated and skipped", current: high, next: &policy.Score{Type: policy.ScoreType_Result, Value: 0, ScoreCompletion: 0}, want: high},
		{name: "unrated next with nil current stays nil", next: &policy.Score{Type: policy.ScoreType_Unscored}, want: nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Same(t, tc.want, lowerScore(tc.current, tc.next))
		})
	}

	t.Run("error and skip scores count by value like GetWorstScore", func(t *testing.T) {
		errScore := &policy.Score{Type: policy.ScoreType_Error, Value: 0, ScoreCompletion: 100}
		assert.Same(t, errScore, lowerScore(high, errScore))

		skip := &policy.Score{Type: policy.ScoreType_Skip, Value: 100, ScoreCompletion: 100}
		assert.Same(t, low, lowerScore(low, skip))
	})
}

// AddReport must not panic on a result without a report or without a score,
// and must leave the worst score alone. Reading the score through GetScore
// keeps a nil Report from dereferencing.
func TestReporters_AddReportWithoutScore(t *testing.T) {
	for name, r := range map[string]Reporter{
		"aggregate": NewAggregateReporter(),
		"error":     NewErrorReporter(),
		"noop":      NewNoOpReporter(),
	} {
		t.Run(name, func(t *testing.T) {
			r.AddReport(&inventory.Asset{Mrn: "//assets/rated"}, &AssetReport{Report: &policy.Report{Score: &policy.Score{Type: policy.ScoreType_Result, Value: 70, ScoreCompletion: 100}}})
			assert.NotPanics(t, func() {
				r.AddReport(&inventory.Asset{Mrn: "//assets/noreport"}, &AssetReport{})
				r.AddReport(&inventory.Asset{Mrn: "//assets/noscore"}, &AssetReport{Report: &policy.Report{}})
			})

			res := r.Reports()
			assert.True(t, res.Ok)
			if assert.NotNil(t, res.WorstScore) {
				assert.Equal(t, uint32(70), res.WorstScore.Value)
			}
		})
	}
}
