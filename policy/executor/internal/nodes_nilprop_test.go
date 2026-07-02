// Copyright Mondoo, Inc. 2026
// SPDX-License-Identifier: BUSL-1.1

package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/types"
)

func nilPropNode(t *testing.T) (*ExecutionQueryNodeData, chan runQueueItem) {
	t.Helper()
	q := make(chan runQueueItem, 1)
	nodeData := &ExecutionQueryNodeData{
		queryID:  "checkquery",
		runQueue: q,
		codeBundle: &llx.CodeBundle{
			CodeV2: &llx.CodeV2{Id: "checkquery"},
		},
		requiredProperties: map[string]*executionQueryProperty{
			"prop1": {
				name:     "prop1",
				checksum: "prop-checksum",
			},
		},
	}
	nodeData.initialize()
	require.Nil(t, nodeData.recalculate())
	return nodeData, q
}

func nilRes(checksum string) *llx.RawResult {
	return &llx.RawResult{
		CodeID: checksum,
		Data:   &llx.RawData{Type: types.String, Value: nil, Error: nil},
	}
}

// A transient nil (short-circuit evaluation reports nil for unevaluated
// branches; see isNilResult) can arrive for a property checksum BEFORE the
// real value. The query must execute with the real value, not the nil.
// Otherwise every comparison against the property evaluates to false and the
// check reports a false FAIL with correct-looking data.
func TestExecutionQueryNode_NilPropUpgradedBeforeDequeue(t *testing.T) {
	nodeData, q := nilPropNode(t)

	// 1. transient nil arrives first: query becomes runnable and is queued
	nodeData.consume(NodeID("prop-checksum"), &envelope{res: nilRes("prop-checksum")})
	nodeData.recalculate()

	// 2. real value arrives while the item waits in the run queue
	nodeData.consume(NodeID("prop-checksum"), &envelope{res: &llx.RawResult{
		CodeID: "prop-checksum",
		Data:   llx.StringData("Success and Failure"),
	}})
	nodeData.recalculate()

	select {
	case item := <-q:
		props := item.props()
		got := props["prop1"]
		require.NotNil(t, got)
		require.Empty(t, got.Error)
		require.NotNil(t, got.Data)
		assert.Equal(t, llx.StringPrimitive("Success and Failure").Value, got.Data.Value,
			"query must execute with the real value, not the transient nil")
	default:
		t.Fatal("query never queued")
	}

	// the nil->real upgrade must not queue the query a second time
	select {
	case <-q:
		t.Fatal("query queued twice")
	default:
	}
}

// A property may legitimately resolve to null (e.g. a missing map key).
// If no real value ever arrives, the query still runs with the null value.
func TestExecutionQueryNode_LegitimateNullPropStillRuns(t *testing.T) {
	nodeData, q := nilPropNode(t)

	nodeData.consume(NodeID("prop-checksum"), &envelope{res: nilRes("prop-checksum")})
	nodeData.recalculate()

	select {
	case item := <-q:
		props := item.props()
		got := props["prop1"]
		require.NotNil(t, got)
		assert.Equal(t, types.Type(got.Data.Type), types.Nil, "null property value is preserved")
	default:
		t.Fatal("query never queued")
	}
}

// A real value must never be downgraded by a later nil for the same checksum.
func TestExecutionQueryNode_RealPropNotDowngradedByNil(t *testing.T) {
	nodeData, q := nilPropNode(t)

	nodeData.consume(NodeID("prop-checksum"), &envelope{res: &llx.RawResult{
		CodeID: "prop-checksum",
		Data:   llx.StringData("Success"),
	}})
	nodeData.recalculate()

	nodeData.consume(NodeID("prop-checksum"), &envelope{res: nilRes("prop-checksum")})
	nodeData.recalculate()

	select {
	case item := <-q:
		got := item.props()["prop1"]
		require.NotNil(t, got)
		assert.Equal(t, llx.StringPrimitive("Success").Value, got.Data.Value)
	default:
		t.Fatal("query never queued")
	}
}

// An error result is real data: it must resolve the property (and win over a
// transient nil), so the execution manager can surface the property error.
func TestExecutionQueryNode_ErrorPropOverridesNil(t *testing.T) {
	nodeData, q := nilPropNode(t)

	nodeData.consume(NodeID("prop-checksum"), &envelope{res: nilRes("prop-checksum")})
	nodeData.recalculate()

	nodeData.consume(NodeID("prop-checksum"), &envelope{res: &llx.RawResult{
		CodeID: "prop-checksum",
		Data:   &llx.RawData{Error: assert.AnError},
	}})
	nodeData.recalculate()

	select {
	case item := <-q:
		got := item.props()["prop1"]
		require.NotNil(t, got)
		assert.NotEmpty(t, got.Error)
	default:
		t.Fatal("query never queued")
	}
}
