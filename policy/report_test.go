// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package policy_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/v10/policy"
)

func TestScoreDistributionAdd(t *testing.T) {
	scoreDist := &policy.ScoreDistribution{}
	scoreDist.Add(&policy.Score{Value: 100, Type: policy.ScoreType_Result, ScoreCompletion: 100})
	scoreDist.Add(&policy.Score{Value: 85, Type: policy.ScoreType_Result, ScoreCompletion: 100})
	scoreDist.Add(&policy.Score{Value: 80, Type: policy.ScoreType_Result, ScoreCompletion: 100})
	scoreDist.Add(&policy.Score{Value: 75, Type: policy.ScoreType_Result, ScoreCompletion: 100})
	scoreDist.Add(&policy.Score{Value: 65, Type: policy.ScoreType_Result, ScoreCompletion: 100})
	scoreDist.Add(&policy.Score{Value: 60, Type: policy.ScoreType_Result, ScoreCompletion: 100})
	scoreDist.Add(&policy.Score{Value: 50, Type: policy.ScoreType_Result, ScoreCompletion: 100})
	scoreDist.Add(&policy.Score{Value: 40, Type: policy.ScoreType_Result, ScoreCompletion: 100})
	scoreDist.Add(&policy.Score{Value: 30, Type: policy.ScoreType_Result, ScoreCompletion: 100})
	scoreDist.Add(&policy.Score{Value: 29, Type: policy.ScoreType_Result, ScoreCompletion: 100})
	scoreDist.Add(&policy.Score{Value: 15, Type: policy.ScoreType_Result, ScoreCompletion: 100})
	scoreDist.Add(&policy.Score{Value: 10, Type: policy.ScoreType_Result, ScoreCompletion: 100})
	scoreDist.Add(&policy.Score{Value: 5, Type: policy.ScoreType_Result, ScoreCompletion: 100})
	scoreDist.Add(&policy.Score{Type: policy.ScoreType_Error})
	scoreDist.Add(&policy.Score{Type: policy.ScoreType_Unscored})
	scoreDist.Add(nil)

	require.Equal(t, uint32(3), scoreDist.GetA())
	require.Equal(t, uint32(3), scoreDist.GetB())
	require.Equal(t, uint32(3), scoreDist.GetC())
	require.Equal(t, uint32(3), scoreDist.GetD())
	require.Equal(t, uint32(1), scoreDist.GetF())
	require.Equal(t, uint32(1), scoreDist.GetError())
	require.Equal(t, uint32(2), scoreDist.GetUnrated())

	require.Equal(t, uint32(16), scoreDist.GetTotal())
}

func TestScoreDistributionAddCount(t *testing.T) {
	scoreDist := &policy.ScoreDistribution{}
	scoreDist.AddCount(&policy.Score{Value: 100, Type: policy.ScoreType_Result, ScoreCompletion: 100}, 3)
	scoreDist.AddCount(&policy.Score{Value: 75, Type: policy.ScoreType_Result, ScoreCompletion: 100}, 4)
	scoreDist.AddCount(&policy.Score{Value: 50, Type: policy.ScoreType_Result, ScoreCompletion: 100}, 5)
	scoreDist.AddCount(&policy.Score{Type: policy.ScoreType_Error}, 6)
	scoreDist.AddCount(&policy.Score{Type: policy.ScoreType_Unscored}, 7)

	require.Equal(t, uint32(3), scoreDist.GetA())
	require.Equal(t, uint32(4), scoreDist.GetB())
	require.Equal(t, uint32(5), scoreDist.GetC())
	require.Equal(t, uint32(6), scoreDist.GetError())
	require.Equal(t, uint32(7), scoreDist.GetUnrated())

	require.Equal(t, uint32(25), scoreDist.GetTotal())
}

func TestScoreDistributionRemove(t *testing.T) {
	scoreDist := &policy.ScoreDistribution{
		Total:   19,
		A:       3,
		B:       3,
		C:       3,
		D:       3,
		F:       2,
		Error:   2,
		Unrated: 3,
	}

	scoreDist.Remove(&policy.Score{Value: 100, Type: policy.ScoreType_Result, ScoreCompletion: 100})
	scoreDist.Remove(&policy.Score{Value: 85, Type: policy.ScoreType_Result, ScoreCompletion: 100})
	scoreDist.Remove(&policy.Score{Value: 80, Type: policy.ScoreType_Result, ScoreCompletion: 100})
	scoreDist.Remove(&policy.Score{Value: 75, Type: policy.ScoreType_Result, ScoreCompletion: 100})
	scoreDist.Remove(&policy.Score{Value: 65, Type: policy.ScoreType_Result, ScoreCompletion: 100})
	scoreDist.Remove(&policy.Score{Value: 60, Type: policy.ScoreType_Result, ScoreCompletion: 100})
	scoreDist.Remove(&policy.Score{Value: 50, Type: policy.ScoreType_Result, ScoreCompletion: 100})
	scoreDist.Remove(&policy.Score{Value: 40, Type: policy.ScoreType_Result, ScoreCompletion: 100})
	scoreDist.Remove(&policy.Score{Value: 30, Type: policy.ScoreType_Result, ScoreCompletion: 100})
	scoreDist.Remove(&policy.Score{Value: 29, Type: policy.ScoreType_Result, ScoreCompletion: 100})
	scoreDist.Remove(&policy.Score{Value: 15, Type: policy.ScoreType_Result, ScoreCompletion: 100})
	scoreDist.Remove(&policy.Score{Value: 10, Type: policy.ScoreType_Result, ScoreCompletion: 100})
	scoreDist.Remove(&policy.Score{Value: 5, Type: policy.ScoreType_Result, ScoreCompletion: 100})
	scoreDist.Remove(&policy.Score{Type: policy.ScoreType_Error})
	scoreDist.Remove(&policy.Score{Type: policy.ScoreType_Unscored})
	scoreDist.Remove(nil)

	require.Equal(t, uint32(0), scoreDist.GetA())
	require.Equal(t, uint32(0), scoreDist.GetB())
	require.Equal(t, uint32(0), scoreDist.GetC())
	require.Equal(t, uint32(0), scoreDist.GetD())
	require.Equal(t, uint32(1), scoreDist.GetF())
	require.Equal(t, uint32(1), scoreDist.GetError())
	require.Equal(t, uint32(1), scoreDist.GetUnrated())
	require.Equal(t, uint32(3), scoreDist.GetTotal())
}

func TestScoreDistributionAddScoreDist(t *testing.T) {
	a := &policy.ScoreDistribution{
		Total:   18,
		A:       3,
		B:       3,
		C:       3,
		D:       3,
		F:       2,
		Error:   2,
		Unrated: 2,
	}

	b := &policy.ScoreDistribution{
		Total:   10,
		A:       3,
		B:       3,
		C:       3,
		Unrated: 1,
	}
	c := a.AddScoreDistribution(b)

	require.Equal(t, uint32(28), c.GetTotal())
	require.Equal(t, uint32(6), c.GetA())
	require.Equal(t, uint32(6), c.GetB())
	require.Equal(t, uint32(6), c.GetC())
	require.Equal(t, uint32(3), c.GetD())
	require.Equal(t, uint32(2), c.GetF())
	require.Equal(t, uint32(2), c.GetError())
	require.Equal(t, uint32(3), c.GetUnrated())
}
