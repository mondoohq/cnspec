// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/policy"
)

// A multi-statement asset filter compiles to one query with one entrypoint per
// statement, and score() combines the statements with a logical and: the query
// scores 100 only if every entrypoint is true. But score() SKIPS nil-valued
// entrypoints instead of failing them, so while a statement is still nil the
// query is scored on the subset that did resolve — and it reports
// ScoreCompletion 100 while doing so.
//
// This is reachable because datapoint checksums are content-addressed and
// shared across queries: an entrypoint of this query can be resolved by a
// different query that short-circuited the same statement, before this query
// has executed it. The Debian 8 and Debian 9 policy filters are identical apart
// from their version line, so `asset.platform == "debian"` and
// `asset.kind != "container-image"` arrive from a sibling filter while
// `asset.version == /^8\./` is still outstanding.
//
// Consumers of these scores must therefore wait for execution to finish rather
// than latch the first passing score they see (see filterScoreTracker in
// policy/executor/graph.go).
func TestReportingQueryNode_NilEntrypointScoresProvisionally(t *testing.T) {
	nodeData := &ReportingQueryNodeData{
		queryID: "debian8-filter",
		results: map[string]*DataResult{
			"platform-cs": {checksum: "platform-cs"},
			"version-cs":  {checksum: "version-cs"},
			"kind-cs":     {checksum: "kind-cs"},
		},
	}
	nodeData.initialize()

	// The sibling Debian 9/10/11 filters compile these two statements to the
	// same checksums and report them first.
	nodeData.consume(NodeID("platform-cs"), &envelope{res: realBoolRes("platform-cs", true)})
	nodeData.consume(NodeID("kind-cs"), &envelope{res: realBoolRes("kind-cs", true)})

	// Nothing is scored yet: the version statement is still outstanding.
	assert.Nil(t, nodeData.score(), "no score before every entrypoint is resolved")

	// A nil lands on the shared version checksum — a branch some other query
	// short-circuited and never evaluated.
	nodeData.consume(NodeID("version-cs"), &envelope{res: executedNilRes("version-cs")})

	s := nodeData.score()
	require.NotNil(t, s)
	assert.Equal(t, policy.ScoreType_Result, s.Type)
	assert.Equal(t, uint32(100), s.Value,
		"the nil entrypoint is skipped, so the filter provisionally passes on the other two statements")
	assert.Equal(t, uint32(100), s.ScoreCompletion,
		"the provisional score is indistinguishable from a final one by completion alone")

	// This query then executes the version statement itself. A real result
	// upgrades the nil and the node recalculates to the truth: no match.
	nodeData.consume(NodeID("version-cs"), &envelope{res: realBoolRes("version-cs", false)})
	nodeData.invalidated = true

	s = nodeData.score()
	require.NotNil(t, s)
	assert.Equal(t, policy.ScoreType_Result, s.Type)
	assert.Equal(t, uint32(0), s.Value,
		"the executed result must override the short-circuit nil")
}

// Deciding a filter from its final score is only sound if the correction
// actually reaches the reporting query: the datapoint node has already
// forwarded the nil once, and it must forward the upgrade too rather than treat
// itself as done. Without this the reporting query would keep the nil forever
// and its last score would still be the provisional one.
func TestDatapointNode_NilUpgradeIsForwardedAgain(t *testing.T) {
	nodeData := &DatapointNodeData{}
	nodeData.initialize()

	nodeData.consume("", &envelope{res: executedNilRes("version-cs")})
	require.NotNil(t, nodeData.recalculate(), "the short-circuit nil is forwarded downstream")
	require.Nil(t, nodeData.recalculate(), "nothing new to forward")

	nodeData.consume("", &envelope{res: realBoolRes("version-cs", false)})
	out := nodeData.recalculate()
	require.NotNil(t, out, "the upgrade must be forwarded so consumers can re-score")
	require.NotNil(t, out.res)
	assert.Equal(t, false, out.res.Data.Value)
}
