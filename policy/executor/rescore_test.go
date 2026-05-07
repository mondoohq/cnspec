// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package executor

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/v13/policy"
	"go.mondoo.com/cnspec/v13/policy/executor/internal"
)

func newScoreCollector() (ScoreCollector, map[string]*policy.Score) {
	collected := map[string]*policy.Score{}
	return &internal.FuncCollector{
		SinkScoreFunc: func(scores []*policy.Score) {
			for _, s := range scores {
				collected[s.QrId] = s
			}
		},
	}, collected
}

// makeRP builds a minimal ResolvedPolicy from reporting jobs.
func makeRP(rjs map[string]*policy.ReportingJob) *policy.ResolvedPolicy {
	return &policy.ResolvedPolicy{
		CollectorJob: &policy.CollectorJob{ReportingJobs: rjs},
	}
}

func s(qrId string, value uint32) *policy.Score {
	return &policy.Score{QrId: qrId, Type: policy.ScoreType_Result, Value: value, ScoreCompletion: 100, Weight: 1}
}

func st(qrId string, typ uint32) *policy.Score {
	return &policy.Score{QrId: qrId, Type: typ, Value: 0, ScoreCompletion: 100, Weight: 1}
}

func TestRescore_SimpleAverage(t *testing.T) {
	rp := makeRP(map[string]*policy.ReportingJob{
		"c1-rj": {Uuid: "c1-rj", QrId: "q1", Type: policy.ReportingJob_CHECK, Notify: []string{"pol1"}},
		"c2-rj": {Uuid: "c2-rj", QrId: "q2", Type: policy.ReportingJob_CHECK, Notify: []string{"pol1"}},
		"pol1": {
			Uuid: "pol1", QrId: "policy1", Type: policy.ReportingJob_POLICY,
			ScoringSystem: policy.ScoringSystem_AVERAGE,
			ChildJobs:     map[string]*policy.Impact{"c1-rj": {Value: &policy.ImpactValue{Value: 100}}, "c2-rj": {Value: &policy.ImpactValue{Value: 100}}},
		},
	})
	collector, collected := newScoreCollector()
	require.NoError(t, RescoreResolvedPolicy("", rp, map[string]*policy.Score{"q1": s("q1", 100), "q2": s("q2", 60)}, collector))
	assert.Equal(t, uint32(80), collected["policy1"].Value, "avg(100,60) = 80")
}

func TestRescore_WorstScoring(t *testing.T) {
	rp := makeRP(map[string]*policy.ReportingJob{
		"c1-rj": {Uuid: "c1-rj", QrId: "q1", Type: policy.ReportingJob_CHECK, Notify: []string{"pol1"}},
		"c2-rj": {Uuid: "c2-rj", QrId: "q2", Type: policy.ReportingJob_CHECK, Notify: []string{"pol1"}},
		"pol1": {
			Uuid: "pol1", QrId: "policy1", Type: policy.ReportingJob_POLICY,
			ScoringSystem: policy.ScoringSystem_WORST,
			ChildJobs:     map[string]*policy.Impact{"c1-rj": {Value: &policy.ImpactValue{Value: 100}}, "c2-rj": {Value: &policy.ImpactValue{Value: 100}}},
		},
	})
	collector, collected := newScoreCollector()
	require.NoError(t, RescoreResolvedPolicy("", rp, map[string]*policy.Score{"q1": s("q1", 90), "q2": s("q2", 30)}, collector))
	assert.Equal(t, uint32(30), collected["policy1"].Value, "worst(90,30) = 30")
}

func TestRescore_MultiLevel(t *testing.T) {
	rp := makeRP(map[string]*policy.ReportingJob{
		"c1-rj": {Uuid: "c1-rj", QrId: "q1", Type: policy.ReportingJob_CHECK, Notify: []string{"pol1"}},
		"c2-rj": {Uuid: "c2-rj", QrId: "q2", Type: policy.ReportingJob_CHECK, Notify: []string{"pol1"}},
		"c3-rj": {Uuid: "c3-rj", QrId: "q3", Type: policy.ReportingJob_CHECK, Notify: []string{"pol2"}},
		"pol1": {
			Uuid: "pol1", QrId: "policy1", Type: policy.ReportingJob_POLICY,
			ScoringSystem: policy.ScoringSystem_AVERAGE, Notify: []string{"ctrl1"},
			ChildJobs: map[string]*policy.Impact{"c1-rj": {Value: &policy.ImpactValue{Value: 100}}, "c2-rj": {Value: &policy.ImpactValue{Value: 100}}},
		},
		"pol2": {
			Uuid: "pol2", QrId: "policy2", Type: policy.ReportingJob_POLICY,
			ScoringSystem: policy.ScoringSystem_AVERAGE, Notify: []string{"ctrl1"},
			ChildJobs: map[string]*policy.Impact{"c3-rj": {Value: &policy.ImpactValue{Value: 100}}},
		},
		"ctrl1": {
			Uuid: "ctrl1", QrId: "control1", Type: policy.ReportingJob_CONTROL,
			ScoringSystem: policy.ScoringSystem_AVERAGE, Notify: []string{"fw1"},
			ChildJobs: map[string]*policy.Impact{"pol1": {Value: &policy.ImpactValue{Value: 100}}, "pol2": {Value: &policy.ImpactValue{Value: 100}}},
		},
		"fw1": {
			Uuid: "fw1", QrId: "framework1", Type: policy.ReportingJob_FRAMEWORK,
			ScoringSystem: policy.ScoringSystem_WORST, Notify: []string{"root-uuid"},
			ChildJobs: map[string]*policy.Impact{"ctrl1": {Value: &policy.ImpactValue{Value: 100}}},
		},
		"root-uuid": {
			Uuid: "root-uuid", QrId: "root", Type: policy.ReportingJob_POLICY,
			ScoringSystem: policy.ScoringSystem_AVERAGE,
			ChildJobs:     map[string]*policy.Impact{"fw1": {Value: &policy.ImpactValue{Value: 100}}},
		},
	})
	collector, collected := newScoreCollector()
	require.NoError(t, RescoreResolvedPolicy("//asset/1", rp, map[string]*policy.Score{
		"q1": s("q1", 100), "q2": s("q2", 60), "q3": s("q3", 40),
	}, collector))

	assert.Equal(t, uint32(80), collected["policy1"].Value, "pol1 = avg(100,60) = 80")
	assert.Equal(t, uint32(40), collected["policy2"].Value, "pol2 = avg(40) = 40")
	assert.Equal(t, uint32(60), collected["control1"].Value, "ctrl1 = avg(80,40) = 60")
	assert.Equal(t, uint32(60), collected["framework1"].Value, "fw1 = worst(60) = 60")
	assert.Equal(t, uint32(60), collected["//asset/1"].Value, "root = avg(60) = 60")
}

func TestRescore_ExceptionTypes(t *testing.T) {
	rp := makeRP(map[string]*policy.ReportingJob{
		"c1-rj": {Uuid: "c1-rj", QrId: "q1", Type: policy.ReportingJob_CHECK, Notify: []string{"pol1"}},
		"c2-rj": {Uuid: "c2-rj", QrId: "q2", Type: policy.ReportingJob_CHECK, Notify: []string{"pol1"}},
		"c3-rj": {Uuid: "c3-rj", QrId: "q3", Type: policy.ReportingJob_CHECK, Notify: []string{"pol1"}},
		"c4-rj": {Uuid: "c4-rj", QrId: "q4", Type: policy.ReportingJob_CHECK, Notify: []string{"pol1"}},
		"pol1": {
			Uuid: "pol1", QrId: "policy1", Type: policy.ReportingJob_POLICY,
			ScoringSystem: policy.ScoringSystem_AVERAGE,
			ChildJobs: map[string]*policy.Impact{
				"c1-rj": {Value: &policy.ImpactValue{Value: 100}}, "c2-rj": {Value: &policy.ImpactValue{Value: 100}},
				"c3-rj": {Value: &policy.ImpactValue{Value: 100}}, "c4-rj": {Value: &policy.ImpactValue{Value: 100}},
			},
		},
	})
	collector, collected := newScoreCollector()
	require.NoError(t, RescoreResolvedPolicy("", rp, map[string]*policy.Score{
		"q1": s("q1", 100), "q2": st("q2", policy.ScoreType_Skip),
		"q3": st("q3", policy.ScoreType_Disabled), "q4": s("q4", 40),
	}, collector))
	assert.Equal(t, uint32(70), collected["policy1"].Value, "avg(100,40) = 70, skip+disabled excluded")
}

func TestRescore_ControlRemapping(t *testing.T) {
	rp := makeRP(map[string]*policy.ReportingJob{
		"c1-rj": {Uuid: "c1-rj", QrId: "q1", Type: policy.ReportingJob_CHECK, Notify: []string{"ctrl1"}},
		"c2-rj": {Uuid: "c2-rj", QrId: "q2", Type: policy.ReportingJob_CHECK, Notify: []string{"ctrl1"}},
		"ctrl1": {
			Uuid: "ctrl1", QrId: "control1", Type: policy.ReportingJob_CONTROL,
			ScoringSystem: policy.ScoringSystem_AVERAGE,
			ChildJobs:     map[string]*policy.Impact{"c1-rj": {Value: &policy.ImpactValue{Value: 100}}, "c2-rj": {Value: &policy.ImpactValue{Value: 100}}},
		},
	})
	collector, collected := newScoreCollector()
	require.NoError(t, RescoreResolvedPolicy("", rp, map[string]*policy.Score{
		"q1": st("q1", policy.ScoreType_Error), "q2": s("q2", 80),
	}, collector))
	assert.Equal(t, uint32(40), collected["control1"].Value, "ctrl = avg(error->0, 80) = 40")
	assert.Equal(t, policy.ScoreType_Result, collected["control1"].Type)
}

func TestRescore_ImpactFloor(t *testing.T) {
	rp := makeRP(map[string]*policy.ReportingJob{
		"c1-rj": {Uuid: "c1-rj", QrId: "q1", Type: policy.ReportingJob_CHECK, Notify: []string{"pol1"}},
		"pol1": {
			Uuid: "pol1", QrId: "policy1", Type: policy.ReportingJob_POLICY,
			ScoringSystem: policy.ScoringSystem_AVERAGE,
			ChildJobs:     map[string]*policy.Impact{"c1-rj": {Value: &policy.ImpactValue{Value: 30}}},
		},
	})
	collector, collected := newScoreCollector()
	require.NoError(t, RescoreResolvedPolicy("", rp, map[string]*policy.Score{"q1": s("q1", 50)}, collector))
	assert.Equal(t, uint32(70), collected["policy1"].Value, "impact floor raises 50 to 70")
}

func TestRescore_UpdatedScores(t *testing.T) {
	rp := makeRP(map[string]*policy.ReportingJob{
		"c1-rj": {Uuid: "c1-rj", QrId: "q1", Type: policy.ReportingJob_CHECK, Notify: []string{"pol1"}},
		"c2-rj": {Uuid: "c2-rj", QrId: "q2", Type: policy.ReportingJob_CHECK, Notify: []string{"pol1"}},
		"pol1": {
			Uuid: "pol1", QrId: "policy1", Type: policy.ReportingJob_POLICY,
			ScoringSystem: policy.ScoringSystem_AVERAGE,
			ChildJobs:     map[string]*policy.Impact{"c1-rj": {Value: &policy.ImpactValue{Value: 100}}, "c2-rj": {Value: &policy.ImpactValue{Value: 100}}},
		},
	})

	c1, col1 := newScoreCollector()
	require.NoError(t, RescoreResolvedPolicy("", rp, map[string]*policy.Score{"q1": s("q1", 80), "q2": s("q2", 40)}, c1))
	assert.Equal(t, uint32(60), col1["policy1"].Value, "first run: avg(80,40) = 60")

	c2, col2 := newScoreCollector()
	require.NoError(t, RescoreResolvedPolicy("", rp, map[string]*policy.Score{"q1": s("q1", 100), "q2": s("q2", 60)}, c2))
	assert.Equal(t, uint32(80), col2["policy1"].Value, "second run: avg(100,60) = 80")
}

func TestRescore_RealResolvedPolicy(t *testing.T) {
	rpData, err := os.ReadFile("internal/testdata/resolved_policy.json")
	require.NoError(t, err)

	rp := &policy.ResolvedPolicy{}
	require.NoError(t, json.Unmarshal(rpData, rp))
	require.NotNil(t, rp.CollectorJob)

	scoresData, err := os.ReadFile("internal/testdata/scores.json")
	require.NoError(t, err)

	var scoresList []*policy.Score
	require.NoError(t, json.Unmarshal(scoresData, &scoresList))

	scores := map[string]*policy.Score{}
	for _, s := range scoresList {
		scores[s.QrId] = s
	}

	rjs := rp.CollectorJob.ReportingJobs
	t.Logf("Real RP: %d reporting jobs, %d scores", len(rjs), len(scores))

	collector, collected := newScoreCollector()
	require.NoError(t, RescoreResolvedPolicy("//asset/test", rp, scores, collector))
	t.Logf("Collected %d scores", len(collected))

	validTypes := map[uint32]bool{
		policy.ScoreType_Result:     true,
		policy.ScoreType_Error:      true,
		policy.ScoreType_Skip:       true,
		policy.ScoreType_Unscored:   true,
		policy.ScoreType_OutOfScope: true,
		policy.ScoreType_Disabled:   true,
	}
	for qrId, s := range collected {
		assert.True(t, validTypes[s.Type], "invalid score type %d for %s", s.Type, qrId)
	}

	// Leaf scores supplied in the scores map should round-trip unchanged
	// through the static-node path.
	valueMismatches := 0
	for qrId, rolled := range collected {
		ss, ok := scores[qrId]
		if !ok {
			continue
		}
		if rolled.Value != ss.Value {
			t.Errorf("Value mismatch for %s: rollup=%d server=%d (type=%d)", qrId, rolled.Value, ss.Value, rolled.Type)
			valueMismatches++
		}
	}
	assert.Equal(t, 0, valueMismatches, "all Value scores should match server")
}
