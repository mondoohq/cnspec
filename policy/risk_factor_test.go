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

func TestRiskFactor_AdjustRiskScore(t *testing.T) {
	tests := []struct {
		risk     RiskFactor
		score    Score
		onDetect Score
		onFail   Score
	}{
		// Relative, increase risk
		{
			risk:     RiskFactor{Magnitude: 0.4},
			score:    Score{RiskScore: 40},
			onDetect: Score{RiskScore: 40, RiskFactors: risks(&ScoredRiskFactor{Risk: 0.4})},
			onFail:   Score{RiskScore: 64, RiskFactors: risks(&ScoredRiskFactor{Risk: -0.4})},
		},
		{
			risk:     RiskFactor{Mrn: "internet-facing", Magnitude: 0.4},
			score:    Score{RiskScore: 10},
			onDetect: Score{RiskScore: 10, RiskFactors: risks(&ScoredRiskFactor{Mrn: "internet-facing", Risk: 0.4})},
			onFail:   Score{RiskScore: 45, RiskFactors: risks(&ScoredRiskFactor{Mrn: "internet-facing", Risk: -0.4})},
		},
		{
			risk:     RiskFactor{Magnitude: 0.4},
			score:    Score{RiskScore: 90},
			onDetect: Score{RiskScore: 90, RiskFactors: risks(&ScoredRiskFactor{Risk: 0.4})},
			onFail:   Score{RiskScore: 94, RiskFactors: risks(&ScoredRiskFactor{Risk: -0.4})},
		},
		// Absolute, decrease risk
		{
			risk:     RiskFactor{Magnitude: -0.4},
			score:    Score{RiskScore: 40},
			onDetect: Score{RiskScore: 64, RiskFactors: risks(&ScoredRiskFactor{Risk: -0.4})},
			onFail:   Score{RiskScore: 40},
		},
		{
			risk:     RiskFactor{Magnitude: -0.4},
			score:    Score{RiskScore: 10},
			onDetect: Score{RiskScore: 45, RiskFactors: risks(&ScoredRiskFactor{Risk: -0.4})},
			onFail:   Score{RiskScore: 10},
		},
		{
			risk:     RiskFactor{Magnitude: -0.4},
			score:    Score{RiskScore: 90},
			onDetect: Score{RiskScore: 94, RiskFactors: risks(&ScoredRiskFactor{Risk: -0.4})},
			onFail:   Score{RiskScore: 90},
		},
		// Absolute, increase risk
		{
			risk:     RiskFactor{Magnitude: 0.2, IsAbsolute: true},
			score:    Score{RiskScore: 40},
			onDetect: Score{RiskScore: 20, RiskFactors: risks(&ScoredRiskFactor{Risk: 0.2, IsAbsolute: true})},
			onFail:   Score{RiskScore: 40},
		},
		{
			risk:     RiskFactor{Magnitude: 0.2, IsAbsolute: true},
			score:    Score{RiskScore: 10},
			onDetect: Score{RiskScore: 0, RiskFactors: risks(&ScoredRiskFactor{Risk: 0.2, IsAbsolute: true})},
			onFail:   Score{RiskScore: 10},
		},
		{
			risk:     RiskFactor{Magnitude: 0.2, IsAbsolute: true},
			score:    Score{RiskScore: 90},
			onDetect: Score{RiskScore: 70, RiskFactors: risks(&ScoredRiskFactor{Risk: 0.2, IsAbsolute: true})},
			onFail:   Score{RiskScore: 90},
		},
		// Absolute, decrease risk
		{
			risk:     RiskFactor{Magnitude: -0.2, IsAbsolute: true},
			score:    Score{RiskScore: 40},
			onDetect: Score{RiskScore: 60, RiskFactors: risks(&ScoredRiskFactor{Risk: -0.2, IsAbsolute: true})},
			onFail:   Score{RiskScore: 40},
		},
		{
			risk:     RiskFactor{Magnitude: -0.2, IsAbsolute: true},
			score:    Score{RiskScore: 10},
			onDetect: Score{RiskScore: 30, RiskFactors: risks(&ScoredRiskFactor{Risk: -0.2, IsAbsolute: true})},
			onFail:   Score{RiskScore: 10},
		},
		{
			risk:     RiskFactor{Magnitude: -0.2, IsAbsolute: true},
			score:    Score{RiskScore: 90},
			onDetect: Score{RiskScore: 100, RiskFactors: risks(&ScoredRiskFactor{Risk: -0.2, IsAbsolute: true})},
			onFail:   Score{RiskScore: 90},
		},
	}

	for i := range tests {
		t.Run("test#"+strconv.Itoa(i), func(t *testing.T) {
			cur := tests[i]

			okScore := cur.score
			cur.risk.AdjustRiskScore(&okScore, true)
			assert.Equal(t, cur.onDetect, okScore, "ok scores match")

			nokScore := cur.score
			cur.risk.AdjustRiskScore(&nokScore, false)
			assert.Equal(t, cur.onFail, nokScore, "fail scores match")
		})
	}
}
