// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package internal

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/v13/policy"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/types"
)

func placeholderRes(checksum, origin, msg string) *llx.RawResult {
	return &llx.RawResult{
		CodeID: checksum,
		Data: &llx.RawData{
			Error: &queryRunError{originCodeID: origin, err: errors.New(msg)},
		},
	}
}

func realBoolRes(checksum string, v bool) *llx.RawResult {
	return &llx.RawResult{CodeID: checksum, Data: llx.BoolData(v)}
}

func executedNilRes(checksum string) *llx.RawResult {
	return &llx.RawResult{
		CodeID: checksum,
		Data:   &llx.RawData{Type: types.String, Value: nil, Error: nil},
	}
}

// A broadcast placeholder from a query that never ran must not survive on a
// shared datapoint once a healthy query reports an executed result.
func TestDatapointNode_PlaceholderUpgradedByExecutedResult(t *testing.T) {
	t.Run("real value overrides placeholder", func(t *testing.T) {
		nodeData := &DatapointNodeData{}
		nodeData.initialize()
		nodeData.consume("", &envelope{res: placeholderRes("cs", "otherquery", "property p errored")})
		nodeData.consume("", &envelope{res: realBoolRes("cs", true)})
		require.NotNil(t, nodeData.res)
		require.Nil(t, nodeData.res.Data.Error)
		assert.Equal(t, true, nodeData.res.Data.Value)
	})

	t.Run("executed nil overrides placeholder", func(t *testing.T) {
		nodeData := &DatapointNodeData{}
		nodeData.initialize()
		nodeData.consume("", &envelope{res: placeholderRes("cs", "otherquery", "property p errored")})
		nodeData.consume("", &envelope{res: executedNilRes("cs")})
		require.NotNil(t, nodeData.res)
		assert.Nil(t, nodeData.res.Data.Error)
	})

	t.Run("placeholder never overrides executed result", func(t *testing.T) {
		nodeData := &DatapointNodeData{}
		nodeData.initialize()
		nodeData.consume("", &envelope{res: realBoolRes("cs", true)})
		nodeData.consume("", &envelope{res: placeholderRes("cs", "otherquery", "property p errored")})
		require.Nil(t, nodeData.res.Data.Error)
		assert.Equal(t, true, nodeData.res.Data.Value)
	})

	t.Run("real executed error is kept (not a placeholder)", func(t *testing.T) {
		nodeData := &DatapointNodeData{}
		nodeData.initialize()
		nodeData.consume("", &envelope{res: &llx.RawResult{
			CodeID: "cs",
			Data:   &llx.RawData{Error: errors.New("real execution error")},
		}})
		nodeData.consume("", &envelope{res: realBoolRes("cs", true)})
		// a genuinely executed error is real data; it is not upgraded away
		require.NotNil(t, nodeData.res.Data.Error)
	})
}

// A healthy query whose shared statement checksum was poisoned by another
// query's broadcast placeholder must score from its own executed results.
func TestReportingQueryNode_PoisonedSharedChecksumRecovers(t *testing.T) {
	nodeData := &ReportingQueryNodeData{
		queryID: "healthyquery",
		results: map[string]*DataResult{
			"shared-cs": {checksum: "shared-cs"},
		},
	}
	nodeData.initialize()

	// another query that could not run broadcasts its placeholder first
	nodeData.consume(NodeID("shared-cs"), &envelope{res: placeholderRes("shared-cs", "brokenquery", "property p errored")})
	s := nodeData.score()
	require.NotNil(t, s)
	assert.Equal(t, policy.ScoreType_Error, s.Type, "before the upgrade the poisoned score is an error")

	// the healthy query executes and reports the real result
	nodeData.consume(NodeID("shared-cs"), &envelope{res: realBoolRes("shared-cs", true)})
	nodeData.invalidated = true
	s = nodeData.score()
	require.NotNil(t, s)
	assert.Equal(t, policy.ScoreType_Result, s.Type, "executed result must override the placeholder")
	assert.Equal(t, uint32(100), s.Value)
	assert.Empty(t, s.Message)
}

// An erroring query must report its OWN property error, not the error of a
// different query that happened to poison a shared statement checksum first.
func TestReportingQueryNode_OwnErrorPreferredOverForeign(t *testing.T) {
	nodeData := &ReportingQueryNodeData{
		queryID: "queryA",
		results: map[string]*DataResult{
			"shared-cs": {checksum: "shared-cs"},
			"own-cs":    {checksum: "own-cs"},
		},
	}
	nodeData.initialize()

	// queryB failed first and poisoned the shared checksum
	nodeData.consume(NodeID("shared-cs"), &envelope{res: placeholderRes("shared-cs", "queryB", "property propB errored: bad B")})
	// queryA's own broadcast for its exclusive checksum
	nodeData.consume(NodeID("own-cs"), &envelope{res: placeholderRes("own-cs", "queryA", "property propA errored: bad A")})

	s := nodeData.score()
	require.NotNil(t, s)
	assert.Equal(t, policy.ScoreType_Error, s.Type)
	assert.Contains(t, s.Message, "propA", "own error must be reported")
	assert.NotContains(t, s.Message, "propB", "foreign query's error must not leak into this query's message")
}

// If the only errors available are foreign placeholders, they are still used
// as the message (better than an empty error).
func TestReportingQueryNode_ForeignErrorFallback(t *testing.T) {
	nodeData := &ReportingQueryNodeData{
		queryID: "queryA",
		results: map[string]*DataResult{
			"shared-cs": {checksum: "shared-cs"},
		},
	}
	nodeData.initialize()
	nodeData.consume(NodeID("shared-cs"), &envelope{res: placeholderRes("shared-cs", "queryB", "property propB errored: bad B")})

	s := nodeData.score()
	require.NotNil(t, s)
	assert.Equal(t, policy.ScoreType_Error, s.Type)
	assert.True(t, strings.Contains(s.Message, "propB"), "fallback to the only available message")
}

// A completed reporting job must reopen when a poisoned datapoint is upgraded
// with the executed result, and a placeholder must never clobber an executed
// result.
func TestReportingJobNode_PlaceholderUpgradeReopens(t *testing.T) {
	newJob := func() *ReportingJobNodeData {
		return &ReportingJobNodeData{
			queryID:      "job",
			forwardScore: true,
			childScores:  map[NodeID]*reportingJobResult{},
			datapoints: map[NodeID]*reportingJobDatapoint{
				"cs": {},
			},
		}
	}

	t.Run("real result reopens completed job", func(t *testing.T) {
		nodeData := newJob()
		nodeData.consume(NodeID("cs"), &envelope{res: placeholderRes("cs", "otherquery", "property p errored")})
		nodeData.completed = true

		nodeData.consume(NodeID("cs"), &envelope{res: realBoolRes("cs", true)})
		assert.False(t, nodeData.completed, "upgrade must reopen the job for rescoring")
		require.NotNil(t, nodeData.datapoints[NodeID("cs")].res)
		assert.Nil(t, nodeData.datapoints[NodeID("cs")].res.Data.Error)
	})

	t.Run("placeholder does not clobber executed result", func(t *testing.T) {
		nodeData := newJob()
		nodeData.consume(NodeID("cs"), &envelope{res: realBoolRes("cs", true)})
		nodeData.consume(NodeID("cs"), &envelope{res: placeholderRes("cs", "otherquery", "property p errored")})
		require.NotNil(t, nodeData.datapoints[NodeID("cs")].res)
		assert.Nil(t, nodeData.datapoints[NodeID("cs")].res.Data.Error)
	})
}

// A property fed by a poisoned datapoint must pick up the executed value.
func TestExecutionQueryNode_PlaceholderPropUpgraded(t *testing.T) {
	nodeData, q := nilPropNode(t)

	nodeData.consume(NodeID("prop-checksum"), &envelope{res: placeholderRes("prop-checksum", "otherquery", "property p errored")})
	nodeData.recalculate()

	nodeData.consume(NodeID("prop-checksum"), &envelope{res: &llx.RawResult{
		CodeID: "prop-checksum",
		Data:   llx.StringData("Success and Failure"),
	}})
	nodeData.recalculate()

	select {
	case item := <-q:
		got := item.props()["prop1"]
		require.NotNil(t, got)
		require.Empty(t, got.Error, "placeholder error must be upgraded by the real value")
		assert.Equal(t, llx.StringPrimitive("Success and Failure").Value, got.Data.Value)
	default:
		t.Fatal("query never queued")
	}
}
