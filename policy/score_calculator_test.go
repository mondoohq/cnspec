// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v9/explorer"
)

type scoreTest struct {
	in  []*Score
	out *Score
}

func testScoring(t *testing.T, init func() ScoreCalculator, tests []scoreTest) {
	for i := range tests {
		test := tests[i]
		t.Run("idx"+strconv.Itoa(i), func(t *testing.T) {
			calc := init()
			for j := range test.in {
				calc.Add(test.in[j])
			}
			res := calc.Calculate()

			assert.Equal(t, int(test.out.DataCompletion), int(res.DataCompletion), "data completion")
			assert.Equal(t, int(test.out.ScoreCompletion), int(res.ScoreCompletion), "score completion")
			assert.Equal(t, int(test.out.Value), int(res.Value), "value")
			assert.Equal(t, int(test.out.Weight), int(res.Weight), "weight")
			assert.Equal(t, test.out.Type, res.Type)
		})
	}
}

func TestEmptyScore(t *testing.T) {
	testScoring(t, func() ScoreCalculator {
		res := averageScoreCalculator{}
		res.Init()
		return &res
	}, []scoreTest{
		{
			in:  []*Score{},
			out: &Score{ScoreCompletion: 100, DataCompletion: 100, Type: ScoreType_Unscored},
		},
	})
}

func TestAverageScores(t *testing.T) {
	testScoring(t, func() ScoreCalculator {
		res := averageScoreCalculator{}
		res.Init()
		return &res
	}, []scoreTest{
		{
			in:  []*Score{},
			out: &Score{Value: 0, ScoreCompletion: 100, DataCompletion: 100, Type: ScoreType_Unscored},
		},
		{
			in: []*Score{
				{Value: 0, ScoreCompletion: 0, DataCompletion: 80, DataTotal: 5, Weight: 1, Type: ScoreType_Result},
				{Value: 20, ScoreCompletion: 20, DataCompletion: 50, DataTotal: 2, Weight: 2, Type: ScoreType_Result},
				{Value: 100, ScoreCompletion: 100, DataCompletion: 33, DataTotal: 3, Weight: 3, Type: ScoreType_Result},
			},
			out: &Score{Value: 60, ScoreCompletion: 40, DataCompletion: 59, DataTotal: 10, Weight: 6, Type: ScoreType_Result},
		},
		{
			in: []*Score{
				{Value: 0, ScoreCompletion: 100, DataCompletion: 100, DataTotal: 1, Weight: 1, Type: ScoreType_Unscored},
			},
			out: &Score{Value: 0, ScoreCompletion: 100, DataCompletion: 100, DataTotal: 1, Weight: 0, Type: ScoreType_Unscored},
		},
		{
			in: []*Score{
				{Value: 0, ScoreCompletion: 100, DataCompletion: 100, DataTotal: 1, Weight: 1, Type: ScoreType_Unscored},
				{Value: 100, ScoreCompletion: 100, DataCompletion: 100, DataTotal: 1, Weight: 1, Type: ScoreType_Result},
			},
			out: &Score{Value: 100, ScoreCompletion: 100, DataCompletion: 100, DataTotal: 1, Weight: 1, Type: ScoreType_Result},
		},
		{
			in: []*Score{
				{Value: 0, ScoreCompletion: 100, DataCompletion: 100, DataTotal: 1, Weight: 1, Type: ScoreType_Unscored},
				{ScoreCompletion: 100, DataCompletion: 100, Weight: 1, Type: ScoreType_Error},
			},
			out: &Score{ScoreCompletion: 100, DataCompletion: 100, DataTotal: 1, Type: ScoreType_Error},
		},
	})
}

func TestWeightedScores(t *testing.T) {
	testScoring(t, func() ScoreCalculator {
		res := weightedScoreCalculator{}
		res.Init()
		return &res
	}, []scoreTest{
		{
			in:  []*Score{},
			out: &Score{Value: 0, ScoreCompletion: 100, DataCompletion: 100, Type: ScoreType_Unscored},
		},
		{
			in: []*Score{
				{Value: 0, ScoreCompletion: 0, DataCompletion: 80, DataTotal: 5, Weight: 1, Type: ScoreType_Result},
				{Value: 20, ScoreCompletion: 20, DataCompletion: 50, DataTotal: 2, Weight: 2, Type: ScoreType_Result},
				{Value: 100, ScoreCompletion: 100, DataCompletion: 33, DataTotal: 3, Weight: 3, Type: ScoreType_Result},
			},
			out: &Score{Value: 68, ScoreCompletion: 40, DataCompletion: 59, Weight: 6, Type: ScoreType_Result},
		},
		{
			in: []*Score{
				{Value: 0, ScoreCompletion: 100, DataCompletion: 100, DataTotal: 1, Weight: 1, Type: ScoreType_Unscored},
			},
			out: &Score{Value: 0, ScoreCompletion: 100, DataCompletion: 100, DataTotal: 1, Weight: 0, Type: ScoreType_Unscored},
		},
		{
			in: []*Score{
				{Value: 0, ScoreCompletion: 100, DataCompletion: 100, DataTotal: 1, Weight: 1, Type: ScoreType_Unscored},
				{Value: 100, ScoreCompletion: 100, DataCompletion: 100, DataTotal: 1, Weight: 1, Type: ScoreType_Result},
			},
			out: &Score{Value: 100, ScoreCompletion: 100, DataCompletion: 100, DataTotal: 1, Weight: 1, Type: ScoreType_Result},
		},
		{
			in: []*Score{
				{Value: 0, ScoreCompletion: 100, DataCompletion: 100, DataTotal: 1, Weight: 1, Type: ScoreType_Unscored},
				{ScoreCompletion: 100, DataCompletion: 100, Weight: 1, Type: ScoreType_Error},
			},
			out: &Score{ScoreCompletion: 100, DataCompletion: 100, DataTotal: 1, Type: ScoreType_Error},
		},
	})
}

func TestWorstScores(t *testing.T) {
	testScoring(t, func() ScoreCalculator {
		res := worstScoreCalculator{}
		res.Init()
		return &res
	}, []scoreTest{
		{
			in:  []*Score{},
			out: &Score{Value: 0, ScoreCompletion: 100, DataCompletion: 100, Type: ScoreType_Unscored},
		},
		{
			in: []*Score{
				{Value: 0, ScoreCompletion: 0, DataCompletion: 80, DataTotal: 5, Weight: 1, Type: ScoreType_Result},
				{Value: 20, ScoreCompletion: 20, DataCompletion: 50, DataTotal: 2, Weight: 2, Type: ScoreType_Result},
				{Value: 100, ScoreCompletion: 100, DataCompletion: 33, DataTotal: 3, Weight: 3, Type: ScoreType_Result},
			},
			out: &Score{Value: 20, ScoreCompletion: 40, DataCompletion: 59, Weight: 6, Type: ScoreType_Result},
		},
		{
			in: []*Score{
				{Value: 0, ScoreCompletion: 100, DataCompletion: 100, DataTotal: 1, Weight: 1, Type: ScoreType_Unscored},
			},
			out: &Score{Value: 0, ScoreCompletion: 100, DataCompletion: 100, DataTotal: 1, Weight: 0, Type: ScoreType_Unscored},
		},
		{
			in: []*Score{
				{Value: 0, ScoreCompletion: 100, DataCompletion: 100, DataTotal: 1, Weight: 1, Type: ScoreType_Unscored},
				{Value: 100, ScoreCompletion: 100, DataCompletion: 100, DataTotal: 1, Weight: 1, Type: ScoreType_Result},
			},
			out: &Score{Value: 100, ScoreCompletion: 100, DataCompletion: 100, DataTotal: 1, Weight: 1, Type: ScoreType_Result},
		},
		{
			in: []*Score{
				{Value: 0, ScoreCompletion: 100, DataCompletion: 100, DataTotal: 1, Weight: 1, Type: ScoreType_Unscored},
				{ScoreCompletion: 100, DataCompletion: 100, Weight: 1, Type: ScoreType_Error},
			},
			out: &Score{ScoreCompletion: 100, DataCompletion: 100, DataTotal: 1, Type: ScoreType_Error},
		},
	})
}

func TestDataOnly(t *testing.T) {
	testScoring(t, func() ScoreCalculator {
		res := averageScoreCalculator{}
		res.Init()
		return &res
	}, []scoreTest{
		{
			in: []*Score{
				{DataCompletion: 80, DataTotal: 5, Type: ScoreType_Unscored},
				{DataCompletion: 40, DataTotal: 5, Type: ScoreType_Unscored},
			},
			out: &Score{ScoreCompletion: 100, DataCompletion: 60, DataTotal: 10, Type: ScoreType_Unscored},
		},
	})
}

func TestDataScoreMix(t *testing.T) {
	testScoring(t, func() ScoreCalculator {
		res := weightedScoreCalculator{}
		res.Init()
		return &res
	}, []scoreTest{
		{
			in: []*Score{
				{DataCompletion: 80, DataTotal: 5, Type: ScoreType_Unscored},
				{Value: 20, ScoreCompletion: 40, Weight: 2, Type: ScoreType_Result},
				{Value: 60, ScoreCompletion: 80, Weight: 2, Type: ScoreType_Result},
			},
			out: &Score{Value: 40, ScoreCompletion: 60, Weight: 4, DataCompletion: 80, DataTotal: 10, Type: ScoreType_Result},
		},
	})
}

func TestImpact(t *testing.T) {
	t.Run("with impact 0", func(t *testing.T) {
		calc := &worstScoreCalculator{}
		calc.Init()

		AddSpecdScore(calc, &Score{
			Type:            ScoreType_Result,
			Value:           90,
			ScoreCompletion: 100,
			Weight:          1,
		}, true, nil)

		AddSpecdScore(calc, &Score{
			Type:            ScoreType_Result,
			Value:           0,
			ScoreCompletion: 100,
			Weight:          1,
		}, true, &explorer.Impact{
			Value: &explorer.ImpactValue{},
		})

		s := calc.Calculate()
		require.EqualValues(t, 90, int(s.Value))
	})

	t.Run("does not modify success", func(t *testing.T) {
		calc := &worstScoreCalculator{}
		calc.Init()

		AddSpecdScore(calc, &Score{
			Type:            ScoreType_Result,
			Value:           100,
			ScoreCompletion: 100,
			Weight:          1,
		}, true, &explorer.Impact{
			Value:  &explorer.ImpactValue{Value: 20},
			Weight: 1,
		})

		s := calc.Calculate()
		require.EqualValues(t, 100, int(s.Value))
	})

	t.Run("severity is treated as a floor", func(t *testing.T) {
		calc := &worstScoreCalculator{}
		calc.Init()

		AddSpecdScore(calc, &Score{
			Type:            ScoreType_Result,
			Value:           60,
			ScoreCompletion: 100,
			Weight:          1,
		}, true, &explorer.Impact{
			Value:  &explorer.ImpactValue{Value: 20},
			Weight: 1,
		})

		s := calc.Calculate()
		require.EqualValues(t, 80, int(s.Value))
	})
}
