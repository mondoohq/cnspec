// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"cmp"
	"context"
	"encoding/json"
	"strconv"
	"strings"
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

func TestRiskFactor_AdjustRiskScoreMultiple2(t *testing.T) {
	testCases := []struct {
		name             string
		scoredRiskInfos1 []*ScoredRiskInfo
		scoredRiskInfos2 []*ScoredRiskInfo
		baseScore        uint32
		expectedScore    uint32
	}{
		{
			name:             "no risk factors",
			scoredRiskInfos1: []*ScoredRiskInfo{},
			scoredRiskInfos2: []*ScoredRiskInfo{},
			baseScore:        30,
			expectedScore:    30,
		},
		{
			name: "one risk factor",
			scoredRiskInfos1: []*ScoredRiskInfo{
				{
					RiskFactor:       &RiskFactor{Mrn: "a", Magnitude: &RiskMagnitude{Value: -0.1}},
					ScoredRiskFactor: &ScoredRiskFactor{IsDetected: true},
				},
			},
			scoredRiskInfos2: []*ScoredRiskInfo{},
			baseScore:        30,
			expectedScore:    37,
		},
		{
			name: "two risk factors",
			scoredRiskInfos1: []*ScoredRiskInfo{
				{
					RiskFactor:       &RiskFactor{Mrn: "a", Magnitude: &RiskMagnitude{Value: -0.1}},
					ScoredRiskFactor: &ScoredRiskFactor{IsDetected: true},
				},
			},
			scoredRiskInfos2: []*ScoredRiskInfo{
				{
					RiskFactor:       &RiskFactor{Mrn: "b", Magnitude: &RiskMagnitude{Value: -0.1}},
					ScoredRiskFactor: &ScoredRiskFactor{IsDetected: true},
				},
			},
			baseScore:     30,
			expectedScore: 43,
		},
		{
			name: "mixed toxic and non-toxic",
			scoredRiskInfos1: []*ScoredRiskInfo{
				{
					RiskFactor:       &RiskFactor{Mrn: "a", Magnitude: &RiskMagnitude{Value: -1, IsToxic: true}},
					ScoredRiskFactor: &ScoredRiskFactor{IsDetected: true},
				},
			},
			scoredRiskInfos2: []*ScoredRiskInfo{
				{
					RiskFactor:       &RiskFactor{Mrn: "b", Magnitude: &RiskMagnitude{Value: -0.1, IsToxic: false}},
					ScoredRiskFactor: &ScoredRiskFactor{IsDetected: true},
				},
			},
			baseScore:     30,
			expectedScore: 100,
		},
		{
			name: "test toxic sorted 1",
			scoredRiskInfos1: []*ScoredRiskInfo{
				{
					RiskFactor:       &RiskFactor{Mrn: "a", Magnitude: &RiskMagnitude{Value: -1, IsToxic: true}},
					ScoredRiskFactor: &ScoredRiskFactor{IsDetected: true},
				},
				{
					RiskFactor:       &RiskFactor{Mrn: "b", Magnitude: &RiskMagnitude{Value: 1, IsToxic: true}},
					ScoredRiskFactor: &ScoredRiskFactor{IsDetected: true},
				},
				{
					RiskFactor:       &RiskFactor{Mrn: "c", Magnitude: &RiskMagnitude{Value: -1, IsToxic: true}},
					ScoredRiskFactor: &ScoredRiskFactor{IsDetected: true},
				},
			},
			baseScore:     30,
			expectedScore: 0, // applied a c b
		},
		{
			name: "test toxic sorted 2",
			scoredRiskInfos1: []*ScoredRiskInfo{
				{
					RiskFactor:       &RiskFactor{Mrn: "a", Magnitude: &RiskMagnitude{Value: -0.2, IsToxic: true}},
					ScoredRiskFactor: &ScoredRiskFactor{IsDetected: true},
				},
				{
					RiskFactor:       &RiskFactor{Mrn: "b", Magnitude: &RiskMagnitude{Value: 0.1, IsToxic: true}},
					ScoredRiskFactor: &ScoredRiskFactor{IsDetected: true},
				},
				{
					RiskFactor:       &RiskFactor{Mrn: "c", Magnitude: &RiskMagnitude{Value: -0.1, IsToxic: true}},
					ScoredRiskFactor: &ScoredRiskFactor{IsDetected: true},
				},
			},
			scoredRiskInfos2: []*ScoredRiskInfo{
				{
					RiskFactor:       &RiskFactor{Mrn: "d", Magnitude: &RiskMagnitude{Value: -0.1}},
					ScoredRiskFactor: &ScoredRiskFactor{IsDetected: true},
				},
			},
			baseScore:     30,
			expectedScore: 57, // applied d a c b
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			score := &Score{Value: tc.baseScore}
			SortScoredRiskInfo(tc.scoredRiskInfos1)
			SortScoredRiskInfo(tc.scoredRiskInfos2)
			AdjustRiskScore(score, tc.scoredRiskInfos1, tc.scoredRiskInfos2)
			assert.EqualValues(t, int(tc.expectedScore), int(score.RiskScore))
		})
	}

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
func TestCmpRiskFactors(t *testing.T) {
	testCases := []struct {
		name     string
		ri       *RiskFactor
		rj       *RiskFactor
		expected int
	}{
		{
			name:     "both nil magnitudes",
			ri:       &RiskFactor{Mrn: "a"},
			rj:       &RiskFactor{Mrn: "b"},
			expected: strings.Compare("a", "b"),
		},
		{
			name:     "one nil magnitude",
			ri:       &RiskFactor{Mrn: "a", Magnitude: &RiskMagnitude{Value: 0.5}},
			rj:       &RiskFactor{Mrn: "b"},
			expected: 1,
		},
		{
			name:     "different toxicity",
			ri:       &RiskFactor{Mrn: "a", Magnitude: &RiskMagnitude{Value: 0.5, IsToxic: true}},
			rj:       &RiskFactor{Mrn: "b", Magnitude: &RiskMagnitude{Value: 0.5, IsToxic: false}},
			expected: 1,
		},
		{
			name:     "different values",
			ri:       &RiskFactor{Mrn: "a", Magnitude: &RiskMagnitude{Value: 0.5}},
			rj:       &RiskFactor{Mrn: "b", Magnitude: &RiskMagnitude{Value: 0.7}},
			expected: cmp.Compare(0.5, 0.7),
		},
		{
			name:     "same values",
			ri:       &RiskFactor{Mrn: "a", Magnitude: &RiskMagnitude{Value: 0.5}},
			rj:       &RiskFactor{Mrn: "b", Magnitude: &RiskMagnitude{Value: 0.5}},
			expected: strings.Compare("a", "b"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := cmpRiskFactors(tc.ri, tc.rj)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestMergeSorted(t *testing.T) {
	// Test case 1: Merging two sorted slices of integers
	t.Run("mergeSorted integers", func(t *testing.T) {
		slice1 := []int{1, 3, 5}
		slice2 := []int{2, 4, 6}
		expected := []int{1, 2, 3, 4, 5, 6}

		var result []int
		seq := mergeSorted(cmp.Compare, slice1, slice2)

		seq(func(val int) bool {
			result = append(result, val)
			return true
		})

		assert.Equal(t, expected, result)
	})

	// Test case 2: Merging slices with duplicates
	t.Run("mergeSorted with duplicates", func(t *testing.T) {
		slice1 := []int{1, 3, 5}
		slice2 := []int{1, 3, 5}
		expected := []int{1, 1, 3, 3, 5, 5}

		var result []int
		seq := mergeSorted(cmp.Compare, slice1, slice2)

		seq(func(val int) bool {
			result = append(result, val)
			return true
		})

		assert.Equal(t, expected, result)
	})

	// Test case 3: Merging slices of strings
	t.Run("mergeSorted strings", func(t *testing.T) {
		slice1 := []string{"apple", "orange"}
		slice2 := []string{"banana", "pear"}
		expected := []string{"apple", "banana", "orange", "pear"}

		var result []string
		seq := mergeSorted(cmp.Compare, slice1, slice2)

		seq(func(val string) bool {
			result = append(result, val)
			return true
		})

		assert.Equal(t, expected, result)
	})

	// Test case 4: Merging with empty slices
	t.Run("mergeSorted with empty slices", func(t *testing.T) {
		slice1 := []int{}
		slice2 := []int{1, 2, 3}
		expected := []int{1, 2, 3}

		var result []int
		seq := mergeSorted(func(i, j int) int {
			return cmp.Compare(i, j)
		}, slice1, slice2)

		seq(func(val int) bool {
			result = append(result, val)
			return true
		})

		assert.Equal(t, expected, result)
	})

	// Test case 5: Merging multiple slices
	t.Run("mergeSorted multiple slices", func(t *testing.T) {
		slice1 := []int{1, 4, 7}
		slice2 := []int{2, 5, 8}
		slice3 := []int{3, 6, 9}
		expected := []int{1, 2, 3, 4, 5, 6, 7, 8, 9}

		var result []int
		seq := mergeSorted(cmp.Compare, slice1, slice2, slice3)

		seq(func(val int) bool {
			result = append(result, val)
			return true
		})

		assert.Equal(t, expected, result)
	})
}
