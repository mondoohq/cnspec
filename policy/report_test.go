package policy_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/policy"
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

	require.Equal(t, uint32(3), scoreDist.GetA())
	require.Equal(t, uint32(3), scoreDist.GetB())
	require.Equal(t, uint32(3), scoreDist.GetC())
	require.Equal(t, uint32(3), scoreDist.GetD())
	require.Equal(t, uint32(1), scoreDist.GetF())
	require.Equal(t, uint32(1), scoreDist.GetError())
	require.Equal(t, uint32(1), scoreDist.GetUnrated())

	require.Equal(t, uint32(15), scoreDist.GetTotal())
}

func TestScoreDistributionRemove(t *testing.T) {
	scoreDist := &policy.ScoreDistribution{
		Total:   18,
		A:       3,
		B:       3,
		C:       3,
		D:       3,
		F:       2,
		Error:   2,
		Unrated: 2,
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
	require.Equal(t, uint32(0), scoreDist.GetA())
	require.Equal(t, uint32(0), scoreDist.GetB())
	require.Equal(t, uint32(0), scoreDist.GetC())
	require.Equal(t, uint32(0), scoreDist.GetD())
	require.Equal(t, uint32(1), scoreDist.GetF())
	require.Equal(t, uint32(1), scoreDist.GetError())
	require.Equal(t, uint32(1), scoreDist.GetUnrated())
	require.Equal(t, uint32(3), scoreDist.GetTotal())
}
