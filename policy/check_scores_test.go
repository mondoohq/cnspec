// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorstCheckScore(t *testing.T) {
	rated := func(v uint32) *Score {
		return &Score{Type: ScoreType_Result, Value: v, ScoreCompletion: 100}
	}

	// root averages two policies; each policy holds checks. The policy and
	// asset scores are deliberately better than the worst check, the way an
	// average reads on a real asset.
	resolved := &ResolvedPolicy{CollectorJob: &CollectorJob{ReportingJobs: map[string]*ReportingJob{
		"root":     {Uuid: "root", QrId: "root", Type: ReportingJob_POLICY, ChildJobs: map[string]*Impact{"p1": nil, "p2": nil}},
		"p1":       {Uuid: "p1", QrId: "//policies/p1", Type: ReportingJob_POLICY, ChildJobs: map[string]*Impact{"medium": nil, "excepted": {Scoring: ScoringSystem_IGNORE_SCORE, Action: Action_IGNORE}}},
		"p2":       {Uuid: "p2", QrId: "//policies/p2", Type: ReportingJob_POLICY, ChildJobs: map[string]*Impact{"errored": nil, "data": nil, "pending": nil, "disabled": nil}},
		"medium":   {Uuid: "medium", QrId: "//checks/medium", Type: ReportingJob_CHECK},
		"excepted": {Uuid: "excepted", QrId: "//checks/excepted", Type: ReportingJob_CHECK},
		"errored":  {Uuid: "errored", QrId: "//checks/errored", Type: ReportingJob_CHECK_AND_DATA_QUERY},
		"data":     {Uuid: "data", QrId: "//queries/data", Type: ReportingJob_DATA_QUERY},
		"pending":  {Uuid: "pending", QrId: "//checks/pending", Type: ReportingJob_CHECK},
		"disabled": {Uuid: "disabled", QrId: "//checks/disabled", Type: ReportingJob_CHECK, ChildJobs: map[string]*Impact{"dq": {Scoring: ScoringSystem_DISABLED}}},
		"dq":       {Uuid: "dq", QrId: "disabled-code", Type: ReportingJob_EXECUTION_QUERY},
	}}}
	report := &Report{
		Score: rated(90),
		Scores: map[string]*Score{
			"root":              rated(90),
			"//policies/p1":     rated(80),
			"//policies/p2":     rated(100),
			"//checks/medium":   rated(55),
			"//checks/excepted": rated(0),
			"//checks/errored":  {Type: ScoreType_Error, Value: 0, ScoreCompletion: 100},
			"//queries/data":    rated(0),
			"//checks/pending":  {Type: ScoreType_Result, Value: 0, ScoreCompletion: 0},
			"//checks/disabled": {Type: ScoreType_Disabled, Value: 0, ScoreCompletion: 100},
		},
	}

	worst := WorstCheckScore(resolved, report)
	require.NotNil(t, worst)
	assert.Equal(t, uint32(55), worst.Value, "only the medium check is a rated, scored finding")

	excepted := resolved.ExceptedQrIds()
	assert.Contains(t, excepted, "//checks/excepted")
	assert.Contains(t, excepted, "disabled-code")
	assert.NotContains(t, excepted, "//checks/medium")

	t.Run("lowest of several checks", func(t *testing.T) {
		report.Scores["//checks/pending"] = rated(10)
		defer func() { report.Scores["//checks/pending"] = &Score{Type: ScoreType_Result, ScoreCompletion: 0} }()
		assert.Equal(t, uint32(10), WorstCheckScore(resolved, report).Value)
	})

	t.Run("no rated check", func(t *testing.T) {
		assert.Nil(t, WorstCheckScore(resolved, &Report{Score: rated(100)}))
	})

	t.Run("missing inputs", func(t *testing.T) {
		assert.Nil(t, WorstCheckScore(nil, report))
		assert.Nil(t, WorstCheckScore(resolved, nil))
		assert.Nil(t, WorstCheckScore(&ResolvedPolicy{}, report))
		assert.Nil(t, (*ResolvedPolicy)(nil).ExceptedQrIds())
	})
}
