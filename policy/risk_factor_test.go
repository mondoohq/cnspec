// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v11"
	"go.mondoo.com/cnquery/v11/explorer"
	"go.mondoo.com/cnquery/v11/mqlc"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/testutils"
)

func risks(risks ...*ScoredRiskFactor) *ScoredRiskFactors {
	return &ScoredRiskFactors{Items: risks}
}

func genRiskFactor1() RiskFactor {
	return RiskFactor{
		Mrn:   "//long/mrn",
		Title: "Some title",
		Docs: &RiskFactorDocs{
			Active:   "Does exist",
			Inactive: "Does not exist",
		},
		Filters: &explorer.Filters{
			Items: map[string]*explorer.Mquery{
				"//filters/mrn": {
					Mrn: "//filters/mrn",
					Mql: "true",
				},
			},
		},
		Checks: []*explorer.Mquery{
			{
				Mrn: "//some/check/1",
				Mql: "1 == 1",
			},
			{
				Mrn: "//some/check/2",
				Mql: "2 == 2",
			},
		},
		Scope: ScopeType_SOFTWARE_AND_RESOURCE,
		Magnitude: &RiskMagnitude{
			Value:   0.5,
			IsToxic: true,
		},
		Software: []*SoftwareSelector{{
			Name:    "mypackage",
			Version: "1.2.3",
		}},
		Resources: []*ResourceSelector{{
			Name: "mondoo",
		}},
	}
}

func TestRiskFactor_Checksums(t *testing.T) {
	base := genRiskFactor1()

	coreSchema := testutils.MustLoadSchema(testutils.SchemaProvider{Provider: "core"})
	conf := mqlc.NewConfig(coreSchema, cnquery.DefaultFeatures)

	ctx := context.Background()
	baseEsum, baseCsum, err := base.RefreshChecksum(ctx, conf)
	require.NoError(t, err)
	require.NotEqual(t, baseEsum, baseCsum)

	noChanges := []func(RiskFactor) RiskFactor{
		func(rf RiskFactor) RiskFactor {
			return rf
		},
		func(rf RiskFactor) RiskFactor {
			rf.Magnitude.Value = 0.5
			return rf
		},
	}

	for i := range noChanges {
		t.Run("noChanges/"+strconv.Itoa(i), func(t *testing.T) {
			test := noChanges[i]
			mod := test(genRiskFactor1())
			esum, csum, err := mod.RefreshChecksum(ctx, conf)
			assert.NoError(t, err)
			assert.Equal(t, baseEsum, esum)
			assert.Equal(t, baseCsum, csum)
		})
	}

	contentChanges := []func(RiskFactor) RiskFactor{
		// 0
		func(rf RiskFactor) RiskFactor {
			rf.Title = ""
			return rf
		},
		// 1
		func(rf RiskFactor) RiskFactor {
			rf.Docs = nil
			return rf
		},
	}

	for i := range contentChanges {
		t.Run("contentChange/"+strconv.Itoa(i), func(t *testing.T) {
			test := contentChanges[i]
			mod := test(genRiskFactor1())
			esum, csum, err := mod.RefreshChecksum(ctx, conf)
			assert.NoError(t, err)
			assert.Equal(t, baseEsum, esum)
			assert.NotEqual(t, baseCsum, csum)
		})
	}

	executionChanges := []func(RiskFactor) RiskFactor{
		// 0
		func(rf RiskFactor) RiskFactor {
			rf.Checks = rf.Checks[0:1]
			return rf
		},
		// 1
		func(rf RiskFactor) RiskFactor {
			rf.Checks[0].Mql = "0 != 1"
			return rf
		},
		// 2
		func(rf RiskFactor) RiskFactor {
			rf.Resources[0].Name = "asset"
			return rf
		},
		// 3
		func(rf RiskFactor) RiskFactor {
			rf.Software[0].Name = "mondoo"
			return rf
		},
		// 4
		func(rf RiskFactor) RiskFactor {
			rf.Magnitude.IsToxic = false
			return rf
		},
		// 5
		func(rf RiskFactor) RiskFactor {
			rf.Magnitude.Value = 0.7
			return rf
		},
		// 6
		func(rf RiskFactor) RiskFactor {
			rf.Action = explorer.Action_DEACTIVATE
			return rf
		},
	}

	for i := range executionChanges {
		t.Run("executionChanges/"+strconv.Itoa(i), func(t *testing.T) {
			test := executionChanges[i]
			mod := test(genRiskFactor1())
			esum, csum, err := mod.RefreshChecksum(ctx, conf)
			assert.NoError(t, err)
			assert.NotEqual(t, baseEsum, esum)
			assert.NotEqual(t, baseCsum, csum)
		})
	}
}

func TestRiskFactor_AdjustRiskScoreMultiple(t *testing.T) {
	rfs := []*RiskFactor{
		{Magnitude: &RiskMagnitude{Value: 0.2}},
		{Magnitude: &RiskMagnitude{Value: 0.3}},
		{Magnitude: &RiskMagnitude{Value: 0.4}},
	}
	a := &Score{RiskScore: 30}
	rfs[0].AdjustRiskScore(a, false)
	rfs[1].AdjustRiskScore(a, false)
	rfs[2].AdjustRiskScore(a, false)

	b := &Score{RiskScore: 30}
	rfs[2].AdjustRiskScore(b, false)
	rfs[1].AdjustRiskScore(b, false)
	rfs[0].AdjustRiskScore(b, false)

	a.RiskFactors = nil
	b.RiskFactors = nil
	assert.Equal(t, uint32(76), a.RiskScore)
	assert.Equal(t, a, b)
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
			risk:     RiskFactor{Magnitude: &RiskMagnitude{Value: 0.4}},
			score:    Score{RiskScore: 40},
			onDetect: Score{RiskScore: 40, RiskFactors: risks(&ScoredRiskFactor{Risk: 0.4, IsDetected: true})},
			onFail:   Score{RiskScore: 64, RiskFactors: risks(&ScoredRiskFactor{Risk: -0.4})},
		},
		{
			risk:     RiskFactor{Mrn: "internet-facing", Magnitude: &RiskMagnitude{Value: 0.4}},
			score:    Score{RiskScore: 10},
			onDetect: Score{RiskScore: 10, RiskFactors: risks(&ScoredRiskFactor{Mrn: "internet-facing", Risk: 0.4, IsDetected: true})},
			onFail:   Score{RiskScore: 45, RiskFactors: risks(&ScoredRiskFactor{Mrn: "internet-facing", Risk: -0.4})},
		},
		{
			risk:     RiskFactor{Magnitude: &RiskMagnitude{Value: 0.4}},
			score:    Score{RiskScore: 90},
			onDetect: Score{RiskScore: 90, RiskFactors: risks(&ScoredRiskFactor{Risk: 0.4, IsDetected: true})},
			onFail:   Score{RiskScore: 94, RiskFactors: risks(&ScoredRiskFactor{Risk: -0.4})},
		},
		// Absolute, decrease risk
		{
			risk:     RiskFactor{Magnitude: &RiskMagnitude{Value: -0.4}},
			score:    Score{RiskScore: 40},
			onDetect: Score{RiskScore: 64, RiskFactors: risks(&ScoredRiskFactor{Risk: -0.4, IsDetected: true})},
			onFail:   Score{RiskScore: 40},
		},
		{
			risk:     RiskFactor{Magnitude: &RiskMagnitude{Value: -0.4}},
			score:    Score{RiskScore: 10},
			onDetect: Score{RiskScore: 45, RiskFactors: risks(&ScoredRiskFactor{Risk: -0.4, IsDetected: true})},
			onFail:   Score{RiskScore: 10},
		},
		{
			risk:     RiskFactor{Magnitude: &RiskMagnitude{Value: -0.4}},
			score:    Score{RiskScore: 90},
			onDetect: Score{RiskScore: 94, RiskFactors: risks(&ScoredRiskFactor{Risk: -0.4, IsDetected: true})},
			onFail:   Score{RiskScore: 90},
		},
		// Absolute, increase risk
		{
			risk:     RiskFactor{Magnitude: &RiskMagnitude{Value: 0.2, IsToxic: true}},
			score:    Score{RiskScore: 40},
			onDetect: Score{RiskScore: 20, RiskFactors: risks(&ScoredRiskFactor{Risk: 0.2, IsToxic: true, IsDetected: true})},
			onFail:   Score{RiskScore: 40},
		},
		{
			risk:     RiskFactor{Magnitude: &RiskMagnitude{Value: 0.2, IsToxic: true}},
			score:    Score{RiskScore: 10},
			onDetect: Score{RiskScore: 0, RiskFactors: risks(&ScoredRiskFactor{Risk: 0.2, IsToxic: true, IsDetected: true})},
			onFail:   Score{RiskScore: 10},
		},
		{
			risk:     RiskFactor{Magnitude: &RiskMagnitude{Value: 0.2, IsToxic: true}},
			score:    Score{RiskScore: 90},
			onDetect: Score{RiskScore: 70, RiskFactors: risks(&ScoredRiskFactor{Risk: 0.2, IsToxic: true, IsDetected: true})},
			onFail:   Score{RiskScore: 90},
		},
		// Absolute, decrease risk
		{
			risk:     RiskFactor{Magnitude: &RiskMagnitude{Value: -0.2, IsToxic: true}},
			score:    Score{RiskScore: 40},
			onDetect: Score{RiskScore: 60, RiskFactors: risks(&ScoredRiskFactor{Risk: -0.2, IsToxic: true, IsDetected: true})},
			onFail:   Score{RiskScore: 40},
		},
		{
			risk:     RiskFactor{Magnitude: &RiskMagnitude{Value: -0.2, IsToxic: true}},
			score:    Score{RiskScore: 10},
			onDetect: Score{RiskScore: 30, RiskFactors: risks(&ScoredRiskFactor{Risk: -0.2, IsToxic: true, IsDetected: true})},
			onFail:   Score{RiskScore: 10},
		},
		{
			risk:     RiskFactor{Magnitude: &RiskMagnitude{Value: -0.2, IsToxic: true}},
			score:    Score{RiskScore: 90},
			onDetect: Score{RiskScore: 100, RiskFactors: risks(&ScoredRiskFactor{Risk: -0.2, IsToxic: true, IsDetected: true})},
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

func TestScoredRiskFactors_Add(t *testing.T) {
	risks := &ScoredRiskFactors{}
	risks.Add(&ScoredRiskFactors{
		Items: []*ScoredRiskFactor{
			{Mrn: "//mrn1", Risk: -0.2},
			{Mrn: "//mrn2", Risk: -0.4},
		},
	})
	risks.Add(&ScoredRiskFactors{
		Items: []*ScoredRiskFactor{
			{Mrn: "//mrn1", Risk: -0.6},
			{Mrn: "//mrn3", Risk: -0.9},
		},
	})

	assert.Equal(t, []*ScoredRiskFactor{
		{Mrn: "//mrn1", Risk: -0.6},
		{Mrn: "//mrn2", Risk: -0.4},
		{Mrn: "//mrn3", Risk: -0.9},
	}, risks.Items)
}

func TestUnmarshal(t *testing.T) {
	testCases := []struct {
		json string
		risk RiskFactor
	}{
		{
			json: `{"magnitude": 0.5}`,
			risk: RiskFactor{Magnitude: &RiskMagnitude{Value: 0.5}},
		},
		{
			json: `{"magnitude": 0.5, "is_absolute": true}`,
			risk: RiskFactor{Magnitude: &RiskMagnitude{Value: 0.5, IsToxic: true}},
		},
		{
			json: `{"magnitude": 0.5, "is_absolute": false}`,
			risk: RiskFactor{Magnitude: &RiskMagnitude{Value: 0.5, IsToxic: false}},
		},
		{
			json: `{"magnitude": {"value": 0.5, "is_toxic": true}}`,
			risk: RiskFactor{Magnitude: &RiskMagnitude{Value: 0.5, IsToxic: true}},
		},
		{
			json: `{"magnitude": {"value": 0.5, "is_toxic": false}}`,
			risk: RiskFactor{Magnitude: &RiskMagnitude{Value: 0.5, IsToxic: false}},
		},
		{
			json: `{"magnitude": {"value": 0.5}, "is_absolute": true}`,
			risk: RiskFactor{Magnitude: &RiskMagnitude{Value: 0.5, IsToxic: true}},
		},
	}

	for i := range testCases {
		t.Run("test#"+strconv.Itoa(i), func(t *testing.T) {
			var risk RiskFactor
			err := json.Unmarshal([]byte(testCases[i].json), &risk)
			require.NoError(t, err)
			assert.Equal(t, &testCases[i].risk, &risk)
		})
	}
}
