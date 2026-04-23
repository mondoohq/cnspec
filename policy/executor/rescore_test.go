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

// makeRP builds a ResolvedPolicy from reporting jobs. It creates dummy
// ExecutionJob queries for CHECK/DATA_QUERY so ReportingQueryNodes are wired.
func makeRP(rjs map[string]*policy.ReportingJob) *policy.ResolvedPolicy {
	return &policy.ResolvedPolicy{
		CollectorJob: &policy.CollectorJob{ReportingJobs: rjs},
	}
}

func s(qrId string, value uint32) *policy.Score {
	return &policy.Score{QrId: qrId, Type: policy.ScoreType_Result, Value: value, ScoreCompletion: 100, Weight: 1}
}

func sr(qrId string, value, riskScore uint32) *policy.Score {
	return &policy.Score{QrId: qrId, Type: policy.ScoreType_Result, Value: value, RiskScore: riskScore, ScoreCompletion: 100, Weight: 1}
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
	assert.Equal(t, uint32(40), collected["control1"].Value, "ctrl = avg(error→0, 80) = 40")
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

func TestRescore_ValueAndRiskScore(t *testing.T) {
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
	require.NoError(t, RescoreResolvedPolicy("", rp, map[string]*policy.Score{
		"q1": sr("q1", 100, 80), "q2": sr("q2", 60, 20),
	}, collector))

	assert.Equal(t, uint32(80), collected["policy1"].Value, "Value = avg(100,60) = 80")
	assert.Equal(t, uint32(50), collected["policy1"].RiskScore, "RiskScore = avg(80,20) = 50")
}

func TestRescore_FullHierarchyRiskAndValue(t *testing.T) {
	rp := makeRP(map[string]*policy.ReportingJob{
		"c1-rj": {Uuid: "c1-rj", QrId: "q1", Type: policy.ReportingJob_CHECK, Notify: []string{"pol1"}},
		"c2-rj": {Uuid: "c2-rj", QrId: "q2", Type: policy.ReportingJob_CHECK, Notify: []string{"pol1"}},
		"c3-rj": {Uuid: "c3-rj", QrId: "q3", Type: policy.ReportingJob_CHECK, Notify: []string{"pol1"}},
		"c4-rj": {Uuid: "c4-rj", QrId: "q4", Type: policy.ReportingJob_CHECK, Notify: []string{"pol2"}},
		"c5-rj": {Uuid: "c5-rj", QrId: "q5", Type: policy.ReportingJob_CHECK, Notify: []string{"pol2"}},
		"c6-rj": {Uuid: "c6-rj", QrId: "q6", Type: policy.ReportingJob_CHECK, Notify: []string{"pol3"}},
		"c7-rj": {Uuid: "c7-rj", QrId: "q7", Type: policy.ReportingJob_CHECK, Notify: []string{"pol3"}},
		"pol1": {
			Uuid: "pol1", QrId: "policy1", Type: policy.ReportingJob_POLICY,
			ScoringSystem: policy.ScoringSystem_WORST, Notify: []string{"ctrl1"},
			ChildJobs: map[string]*policy.Impact{"c1-rj": {Value: &policy.ImpactValue{Value: 100}}, "c2-rj": {Value: &policy.ImpactValue{Value: 100}}, "c3-rj": {Value: &policy.ImpactValue{Value: 100}}},
		},
		"pol2": {
			Uuid: "pol2", QrId: "policy2", Type: policy.ReportingJob_POLICY,
			ScoringSystem: policy.ScoringSystem_AVERAGE, Notify: []string{"ctrl1"},
			ChildJobs: map[string]*policy.Impact{"c4-rj": {Value: &policy.ImpactValue{Value: 100}}, "c5-rj": {Value: &policy.ImpactValue{Value: 100}}},
		},
		"pol3": {
			Uuid: "pol3", QrId: "policy3", Type: policy.ReportingJob_POLICY,
			ScoringSystem: policy.ScoringSystem_AVERAGE, Notify: []string{"ctrl2"},
			ChildJobs: map[string]*policy.Impact{"c6-rj": {Value: &policy.ImpactValue{Value: 100}}, "c7-rj": {Value: &policy.ImpactValue{Value: 100}}},
		},
		"ctrl1": {
			Uuid: "ctrl1", QrId: "control1", Type: policy.ReportingJob_CONTROL,
			ScoringSystem: policy.ScoringSystem_AVERAGE, Notify: []string{"fw1"},
			ChildJobs: map[string]*policy.Impact{"pol1": {Value: &policy.ImpactValue{Value: 100}}, "pol2": {Value: &policy.ImpactValue{Value: 100}}},
		},
		"ctrl2": {
			Uuid: "ctrl2", QrId: "control2", Type: policy.ReportingJob_CONTROL,
			ScoringSystem: policy.ScoringSystem_AVERAGE, Notify: []string{"fw1"},
			ChildJobs: map[string]*policy.Impact{"pol3": {Value: &policy.ImpactValue{Value: 100}}},
		},
		"fw1": {
			Uuid: "fw1", QrId: "framework1", Type: policy.ReportingJob_FRAMEWORK,
			ScoringSystem: policy.ScoringSystem_WORST, Notify: []string{"root-uuid"},
			ChildJobs: map[string]*policy.Impact{"ctrl1": {Value: &policy.ImpactValue{Value: 100}}, "ctrl2": {Value: &policy.ImpactValue{Value: 100}}},
		},
		"root-uuid": {
			Uuid: "root-uuid", QrId: "root", Type: policy.ReportingJob_POLICY,
			ScoringSystem: policy.ScoringSystem_AVERAGE,
			ChildJobs:     map[string]*policy.Impact{"fw1": {Value: &policy.ImpactValue{Value: 100}}},
		},
	})
	collector, collected := newScoreCollector()
	require.NoError(t, RescoreResolvedPolicy("//asset/1", rp, map[string]*policy.Score{
		"q1": sr("q1", 100, 90), "q2": sr("q2", 80, 30), "q3": sr("q3", 60, 70),
		"q4": sr("q4", 50, 40), "q5": sr("q5", 90, 10),
		"q6": st("q6", policy.ScoreType_Skip), "q7": sr("q7", 70, 60),
	}, collector))

	assert.Equal(t, uint32(60), collected["policy1"].Value)
	assert.Equal(t, uint32(30), collected["policy1"].RiskScore)
	assert.Equal(t, uint32(70), collected["policy2"].Value)
	assert.Equal(t, uint32(25), collected["policy2"].RiskScore)
	assert.Equal(t, uint32(70), collected["policy3"].Value)
	assert.Equal(t, uint32(60), collected["policy3"].RiskScore)
	assert.Equal(t, uint32(65), collected["control1"].Value)
	assert.Equal(t, uint32(27), collected["control1"].RiskScore)
	assert.Equal(t, uint32(70), collected["control2"].Value)
	assert.Equal(t, uint32(60), collected["control2"].RiskScore)
	assert.Equal(t, uint32(65), collected["framework1"].Value)
	assert.Equal(t, uint32(27), collected["framework1"].RiskScore)
	assert.Equal(t, uint32(65), collected["//asset/1"].Value)
	assert.Equal(t, uint32(27), collected["//asset/1"].RiskScore)
}

func TestRescore_WeightedRiskScoring(t *testing.T) {
	// WEIGHTED scoring with different weights. Value and RiskScore
	// should be weighted independently.
	//
	// q1: V=100, R=80, weight=3  ──┐
	// q2: V=0,   R=20, weight=1  ──┘──> pol1 (WEIGHTED)
	//
	// Value:     (100*3 + 0*1) / (3+1) = 75
	// RiskScore: (80*3 + 20*1) / (3+1) = 65
	rp := makeRP(map[string]*policy.ReportingJob{
		"c1-rj": {Uuid: "c1-rj", QrId: "q1", Type: policy.ReportingJob_CHECK, Notify: []string{"pol1"}},
		"c2-rj": {Uuid: "c2-rj", QrId: "q2", Type: policy.ReportingJob_CHECK, Notify: []string{"pol1"}},
		"pol1": {
			Uuid: "pol1", QrId: "policy1", Type: policy.ReportingJob_POLICY,
			ScoringSystem: policy.ScoringSystem_WEIGHTED,
			ChildJobs: map[string]*policy.Impact{
				"c1-rj": {Value: &policy.ImpactValue{Value: 100}, Weight: 3},
				"c2-rj": {Value: &policy.ImpactValue{Value: 100}, Weight: 1},
			},
		},
	})
	collector, collected := newScoreCollector()
	require.NoError(t, RescoreResolvedPolicy("", rp, map[string]*policy.Score{
		"q1": sr("q1", 100, 80), "q2": sr("q2", 0, 20),
	}, collector))

	assert.Equal(t, uint32(75), collected["policy1"].Value, "Value = weighted(100*3, 0*1) = 75")
	assert.Equal(t, uint32(65), collected["policy1"].RiskScore, "RiskScore = weighted(80*3, 20*1) = 65")
}

func TestRescore_MultiLevelWeightedRiskScoring(t *testing.T) {
	// Multi-level with WEIGHTED at policy level and WORST at root.
	// Both Value and RiskScore should propagate independently.
	//
	// q1: V=100, R=90, w=3  ──┐
	// q2: V=60,  R=30, w=1  ──┘──> pol1 (WEIGHTED) ──┐
	//                                                  ├──> root (WORST)
	// q3: V=80,  R=40        ──> pol2 (AVERAGE)   ──┘
	//
	// pol1 Value:     (100*3 + 60*1) / 4 = 90
	// pol1 RiskScore: (90*3 + 30*1) / 4 = 75
	// pol2 Value:     avg(80) = 80
	// pol2 RiskScore: avg(40) = 40
	// root Value:     worst(90, 80) = 80
	// root RiskScore: worst(75, 40) = 40
	rp := makeRP(map[string]*policy.ReportingJob{
		"c1-rj": {Uuid: "c1-rj", QrId: "q1", Type: policy.ReportingJob_CHECK, Notify: []string{"pol1"}},
		"c2-rj": {Uuid: "c2-rj", QrId: "q2", Type: policy.ReportingJob_CHECK, Notify: []string{"pol1"}},
		"c3-rj": {Uuid: "c3-rj", QrId: "q3", Type: policy.ReportingJob_CHECK, Notify: []string{"pol2"}},
		"pol1": {
			Uuid: "pol1", QrId: "policy1", Type: policy.ReportingJob_POLICY,
			ScoringSystem: policy.ScoringSystem_WEIGHTED, Notify: []string{"root-uuid"},
			ChildJobs: map[string]*policy.Impact{
				"c1-rj": {Value: &policy.ImpactValue{Value: 100}, Weight: 3},
				"c2-rj": {Value: &policy.ImpactValue{Value: 100}, Weight: 1},
			},
		},
		"pol2": {
			Uuid: "pol2", QrId: "policy2", Type: policy.ReportingJob_POLICY,
			ScoringSystem: policy.ScoringSystem_AVERAGE, Notify: []string{"root-uuid"},
			ChildJobs: map[string]*policy.Impact{"c3-rj": {Value: &policy.ImpactValue{Value: 100}}},
		},
		"root-uuid": {
			Uuid: "root-uuid", QrId: "root", Type: policy.ReportingJob_POLICY,
			ScoringSystem: policy.ScoringSystem_WORST,
			ChildJobs: map[string]*policy.Impact{
				"pol1": {Value: &policy.ImpactValue{Value: 100}},
				"pol2": {Value: &policy.ImpactValue{Value: 100}},
			},
		},
	})
	collector, collected := newScoreCollector()
	require.NoError(t, RescoreResolvedPolicy("//asset/1", rp, map[string]*policy.Score{
		"q1": sr("q1", 100, 90), "q2": sr("q2", 60, 30), "q3": sr("q3", 80, 40),
	}, collector))

	assert.Equal(t, uint32(90), collected["policy1"].Value, "pol1 V = weighted(100*3,60*1) = 90")
	assert.Equal(t, uint32(75), collected["policy1"].RiskScore, "pol1 R = weighted(90*3,30*1) = 75")
	assert.Equal(t, uint32(80), collected["policy2"].Value, "pol2 V = avg(80) = 80")
	assert.Equal(t, uint32(40), collected["policy2"].RiskScore, "pol2 R = avg(40) = 40")
	assert.Equal(t, uint32(80), collected["//asset/1"].Value, "root V = worst(90,80) = 80")
	assert.Equal(t, uint32(40), collected["//asset/1"].RiskScore, "root R = worst(75,40) = 40")
}

func TestRescore_RealResolvedPolicy(t *testing.T) {
	rpData, err := os.ReadFile("internal/testdata/resolved_policy.json")
	require.NoError(t, err)

	rp := &policy.ResolvedPolicy{}
	require.NoError(t, json.Unmarshal(rpData, rp))
	require.NotNil(t, rp.CollectorJob)

	// Load real server-side scores directly into policy.Score
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

	// Every collected score must have a valid type
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

	// Compare rolled-up Value against server scores for leaf entries.
	// Value should match since we feed the same leaf scores.
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
		// Leaf score types should be preserved through forward-score nodes
		if rolled.Type != ss.Type {
			t.Logf("Type differs for %s: rollup=%d server=%d (expected for aggregate nodes)", qrId, rolled.Type, ss.Type)
		}
	}
	assert.Equal(t, 0, valueMismatches, "all Value scores should match server")
}
