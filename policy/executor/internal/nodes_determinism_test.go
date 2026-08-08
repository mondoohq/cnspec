// Copyright Mondoo, Inc. 2026
// SPDX-License-Identifier: BUSL-1.1

package internal

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/types"
)

// TestReportingQueryNode_DeterministicErrorMessage guards the sub-error
// ordering of a score's message. A query with several failing datapoints
// (e.g. the macOS FileVault check shelling out to three fdesetup commands,
// all failing without root) aggregates their errors into one message; that
// aggregation used to follow Go's randomized map iteration, so the same
// three errors produced a differently-ordered message on every scan —
// churning stored score rows and scan history on content that didn't
// change. The message must be identical across runs.
func TestReportingQueryNode_DeterministicErrorMessage(t *testing.T) {
	build := func() *ReportingQueryNodeData {
		nodeData := &ReportingQueryNodeData{
			queryID: "testquery",
			results: map[string]*DataResult{
				"ep1": {checksum: "ep1"},
				"ep2": {checksum: "ep2"},
				"ep3": {checksum: "ep3"},
			},
		}
		nodeData.initialize()
		for i, ep := range []string{"ep1", "ep2", "ep3"} {
			nodeData.consume(ep, &envelope{
				res: &llx.RawResult{
					CodeID: ep,
					Data: &llx.RawData{
						Type:  types.Bool,
						Error: errors.New(fmt.Sprintf("fdesetup call %d failed: requires root", i+1)),
					},
				},
			})
		}
		return nodeData
	}

	reference := build().score()
	require.NotNil(t, reference)
	require.NotEmpty(t, reference.Message)

	// Rebuilding the node exercises fresh map iteration each time; with three
	// entries a nondeterministic order flips the message with probability 5/6
	// per attempt, so 50 identical runs make regression practically certain
	// to fail loudly.
	for i := 0; i < 50; i++ {
		s := build().score()
		require.NotNil(t, s)
		assert.Equal(t, reference.Message, s.Message, "score message must not depend on map iteration order")
	}
}
