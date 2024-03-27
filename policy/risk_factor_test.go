// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func risks(risks ...*ScoredRiskFactor) *ScoredRiskFactors {
	return &ScoredRiskFactors{Items: risks}
}

func TestRiskFactor_AdjustScore(t *testing.T) {
	tests := []struct {
		risk     RiskFactor
		score    Score
		onDetect Score
		onFail   Score
	}{
		// Relative, increase risk
		{
			risk:     RiskFactor{Magnitude: 0.4},
			score:    Score{Value: 40},
			onDetect: Score{Value: 40, RiskFactors: risks(&ScoredRiskFactor{Risk: 0.4})},
			onFail:   Score{Value: 64, RiskFactors: risks(&ScoredRiskFactor{Risk: -0.4})},
		},
		{
			risk:     RiskFactor{Mrn: "internet-facing", Magnitude: 0.4},
			score:    Score{Value: 10},
			onDetect: Score{Value: 10, RiskFactors: risks(&ScoredRiskFactor{Mrn: "internet-facing", Risk: 0.4})},
			onFail:   Score{Value: 45, RiskFactors: risks(&ScoredRiskFactor{Mrn: "internet-facing", Risk: -0.4})},
		},
		{
			risk:     RiskFactor{Magnitude: 0.4},
			score:    Score{Value: 90},
			onDetect: Score{Value: 90, RiskFactors: risks(&ScoredRiskFactor{Risk: 0.4})},
			onFail:   Score{Value: 94, RiskFactors: risks(&ScoredRiskFactor{Risk: -0.4})},
		},
		// Absolute, decrease risk
		{
			risk:     RiskFactor{Magnitude: -0.4},
			score:    Score{Value: 40},
			onDetect: Score{Value: 64, RiskFactors: risks(&ScoredRiskFactor{Risk: -0.4})},
			onFail:   Score{Value: 40},
		},
		{
			risk:     RiskFactor{Magnitude: -0.4},
			score:    Score{Value: 10},
			onDetect: Score{Value: 45, RiskFactors: risks(&ScoredRiskFactor{Risk: -0.4})},
			onFail:   Score{Value: 10},
		},
		{
			risk:     RiskFactor{Magnitude: -0.4},
			score:    Score{Value: 90},
			onDetect: Score{Value: 94, RiskFactors: risks(&ScoredRiskFactor{Risk: -0.4})},
			onFail:   Score{Value: 90},
		},
		// Absolute, increase risk
		{
			risk:     RiskFactor{Magnitude: 0.2, IsAbsolute: true},
			score:    Score{Value: 40},
			onDetect: Score{Value: 20, RiskFactors: risks(&ScoredRiskFactor{Risk: 0.2, IsAbsolute: true})},
			onFail:   Score{Value: 40},
		},
		{
			risk:     RiskFactor{Magnitude: 0.2, IsAbsolute: true},
			score:    Score{Value: 10},
			onDetect: Score{Value: 0, RiskFactors: risks(&ScoredRiskFactor{Risk: 0.2, IsAbsolute: true})},
			onFail:   Score{Value: 10},
		},
		{
			risk:     RiskFactor{Magnitude: 0.2, IsAbsolute: true},
			score:    Score{Value: 90},
			onDetect: Score{Value: 70, RiskFactors: risks(&ScoredRiskFactor{Risk: 0.2, IsAbsolute: true})},
			onFail:   Score{Value: 90},
		},
		// Absolute, decrease risk
		{
			risk:     RiskFactor{Magnitude: -0.2, IsAbsolute: true},
			score:    Score{Value: 40},
			onDetect: Score{Value: 60, RiskFactors: risks(&ScoredRiskFactor{Risk: -0.2, IsAbsolute: true})},
			onFail:   Score{Value: 40},
		},
		{
			risk:     RiskFactor{Magnitude: -0.2, IsAbsolute: true},
			score:    Score{Value: 10},
			onDetect: Score{Value: 30, RiskFactors: risks(&ScoredRiskFactor{Risk: -0.2, IsAbsolute: true})},
			onFail:   Score{Value: 10},
		},
		{
			risk:     RiskFactor{Magnitude: -0.2, IsAbsolute: true},
			score:    Score{Value: 90},
			onDetect: Score{Value: 100, RiskFactors: risks(&ScoredRiskFactor{Risk: -0.2, IsAbsolute: true})},
			onFail:   Score{Value: 90},
		},
	}

	for i := range tests {
		t.Run("test#"+strconv.Itoa(i), func(t *testing.T) {
			cur := tests[i]

			okScore := cur.score
			cur.risk.AdjustScore(&okScore, true)
			assert.Equal(t, cur.onDetect, okScore, "ok scores match")

			nokScore := cur.score
			cur.risk.AdjustScore(&nokScore, false)
			assert.Equal(t, cur.onFail, nokScore, "fail scores match")
		})
	}
}
