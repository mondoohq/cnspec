// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRiskFactor_AdjustScore(t *testing.T) {
	tests := []struct {
		risk          RiskFactor
		score         Score
		onDetect      *ScoredRiskFactor
		onDetectScore Score
		onFail        *ScoredRiskFactor
		onFailScore   Score
	}{
		// Relative, increase risk
		{
			risk:          RiskFactor{Magnitude: 0.4},
			score:         Score{Value: 40},
			onDetect:      &ScoredRiskFactor{Risk: 0.4},
			onDetectScore: Score{Value: 40},
			onFail:        &ScoredRiskFactor{Risk: -0.4},
			onFailScore:   Score{Value: 64},
		},
		{
			risk:          RiskFactor{Magnitude: 0.4},
			score:         Score{Value: 10},
			onDetect:      &ScoredRiskFactor{Risk: 0.4},
			onDetectScore: Score{Value: 10},
			onFail:        &ScoredRiskFactor{Risk: -0.4},
			onFailScore:   Score{Value: 45},
		},
		{
			risk:          RiskFactor{Magnitude: 0.4},
			score:         Score{Value: 90},
			onDetect:      &ScoredRiskFactor{Risk: 0.4},
			onDetectScore: Score{Value: 90},
			onFail:        &ScoredRiskFactor{Risk: -0.4},
			onFailScore:   Score{Value: 94},
		},
		// Absolute, decrease risk
		{
			risk:          RiskFactor{Magnitude: -0.4},
			score:         Score{Value: 40},
			onDetect:      &ScoredRiskFactor{Risk: -0.4},
			onDetectScore: Score{Value: 64},
			onFail:        nil,
			onFailScore:   Score{Value: 40},
		},
		{
			risk:          RiskFactor{Magnitude: -0.4},
			score:         Score{Value: 10},
			onDetect:      &ScoredRiskFactor{Risk: -0.4},
			onDetectScore: Score{Value: 45},
			onFail:        nil,
			onFailScore:   Score{Value: 10},
		},
		{
			risk:          RiskFactor{Magnitude: -0.4},
			score:         Score{Value: 90},
			onDetect:      &ScoredRiskFactor{Risk: -0.4},
			onDetectScore: Score{Value: 94},
			onFail:        nil,
			onFailScore:   Score{Value: 90},
		},
		// Absolute, increase risk
		{
			risk:          RiskFactor{Magnitude: 0.2, IsAbsolute: true},
			score:         Score{Value: 40},
			onDetect:      &ScoredRiskFactor{Risk: 0.2, IsAbsolute: true},
			onDetectScore: Score{Value: 20},
			onFail:        nil,
			onFailScore:   Score{Value: 40},
		},
		{
			risk:          RiskFactor{Magnitude: 0.2, IsAbsolute: true},
			score:         Score{Value: 10},
			onDetect:      &ScoredRiskFactor{Risk: 0.2, IsAbsolute: true},
			onDetectScore: Score{Value: 0},
			onFail:        nil,
			onFailScore:   Score{Value: 10},
		},
		{
			risk:          RiskFactor{Magnitude: 0.2, IsAbsolute: true},
			score:         Score{Value: 90},
			onDetect:      &ScoredRiskFactor{Risk: 0.2, IsAbsolute: true},
			onDetectScore: Score{Value: 70},
			onFail:        nil,
			onFailScore:   Score{Value: 90},
		},
		// Absolute, decrease risk
		{
			risk:          RiskFactor{Magnitude: -0.2, IsAbsolute: true},
			score:         Score{Value: 40},
			onDetect:      &ScoredRiskFactor{Risk: -0.2, IsAbsolute: true},
			onDetectScore: Score{Value: 60},
			onFail:        nil,
			onFailScore:   Score{Value: 40},
		},
		{
			risk:          RiskFactor{Magnitude: -0.2, IsAbsolute: true},
			score:         Score{Value: 10},
			onDetect:      &ScoredRiskFactor{Risk: -0.2, IsAbsolute: true},
			onDetectScore: Score{Value: 30},
			onFail:        nil,
			onFailScore:   Score{Value: 10},
		},
		{
			risk:          RiskFactor{Magnitude: -0.2, IsAbsolute: true},
			score:         Score{Value: 90},
			onDetect:      &ScoredRiskFactor{Risk: -0.2, IsAbsolute: true},
			onDetectScore: Score{Value: 100},
			onFail:        nil,
			onFailScore:   Score{Value: 90},
		},
	}

	for i := range tests {
		t.Run("test#"+strconv.Itoa(i), func(t *testing.T) {
			cur := tests[i]

			okScore := cur.score
			okRF := cur.risk.AdjustScore(&okScore, true)
			assert.Equal(t, cur.onDetect, okRF, "ok risk factors match")
			assert.Equal(t, cur.onDetectScore, okScore, "ok scores match")

			nokScore := cur.score
			nokRF := cur.risk.AdjustScore(&nokScore, false)
			assert.Equal(t, cur.onFail, nokRF, "fail risk factors match")
			assert.Equal(t, cur.onFailScore, nokScore, "fail scores match")
		})
	}
}
