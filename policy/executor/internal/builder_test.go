// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/types"
	"go.mondoo.com/cnspec/policy"
)

func TestPrioritizeNode(t *testing.T) {
	// ┌──────┐┌───┐
	// │eq1   ││eq2│
	// └┬────┬┘└──┬┘
	// ┌▽──┐┌▽──┐┌▽──┐
	// │dp1││dp2││dp3│
	// └─┬┬┘└─┬┬┘└┬─┬┘
	//   ││   ││  │ │
	//  ┌││───││──┘ │
	//  ││└──┐││    │
	//  ││┌──│┘│    │
	// ┌▽▽▽┐┌▽─▽┐┌──▽┐
	// │dps││rj1││rj2│
	// └───┘└┬──┘└┬──┘
	// ┌─────▽────▽┐
	// │scores     │
	// └───────────┘

	nodes := map[NodeID]*Node{
		"eq1":    {id: "eq1"},
		"eq2":    {id: "eq1"},
		"dp1":    {id: "eq1"},
		"dp2":    {id: "eq1"},
		"dp3":    {id: "eq1"},
		"rj1":    {id: "rj1"},
		"rj2":    {id: "rj2"},
		"scores": {id: "scores"},
		"dps":    {id: "dps"},
	}
	edges := map[NodeID][]NodeID{
		"eq1": {"dp1", "dp2"},
		"eq2": {"dp3"},
		"dp1": {"rj1", "dps"},
		"dp2": {"rj1", "dps"},
		"dp3": {"rj2", "dps"},
		"rj1": {"scores"},
		"rj2": {"scores"},
	}

	priorityMap := map[NodeID]int{}
	for n := range nodes {
		prioritizeNode(nodes, edges, priorityMap, n)
	}

	require.Equal(t, len(nodes), len(priorityMap))
	assert.Equal(t, 1, priorityMap["scores"])
	assert.Equal(t, 1, priorityMap["dps"])
	assert.Greater(t, priorityMap["eq1"], priorityMap["dp1"])
	assert.Greater(t, priorityMap["eq1"], priorityMap["dp2"])
	assert.Greater(t, priorityMap["eq2"], priorityMap["dp3"])
	assert.Greater(t, priorityMap["dp1"], priorityMap["dps"])
	assert.Greater(t, priorityMap["dp1"], priorityMap["rj1"])
	assert.Greater(t, priorityMap["dp2"], priorityMap["dps"])
	assert.Greater(t, priorityMap["dp2"], priorityMap["rj1"])
	assert.Greater(t, priorityMap["dp3"], priorityMap["dps"])
	assert.Greater(t, priorityMap["dp3"], priorityMap["rj2"])
	assert.Greater(t, priorityMap["rj1"], priorityMap["scores"])
	assert.Greater(t, priorityMap["rj2"], priorityMap["scores"])
}

func TestBuilder(t *testing.T) {
	b := NewBuilder()

	b.AddQuery(
		&llx.CodeBundle{
			CodeV2: &llx.CodeV2{
				Id: "propertyquery",
				Blocks: []*llx.Block{{
					Datapoints:  []uint64{1},
					Entrypoints: []uint64{2},
				}},
				Checksums: map[uint64]string{1: "checksum1", 2: "pqep"},
			},
		}, nil, nil)

	b.AddQuery(
		&llx.CodeBundle{
			CodeV2: &llx.CodeV2{
				Id: "query1",
				Blocks: []*llx.Block{{
					Entrypoints: []uint64{1},
				}},
				Checksums: map[uint64]string{1: "checksum2"},
			},
		}, map[string]string{"prop": "checksum1"}, nil)

	b.AddQuery(
		&llx.CodeBundle{
			CodeV2: &llx.CodeV2{
				Id: "query2",
				Blocks: []*llx.Block{{
					Datapoints:  []uint64{1},
					Entrypoints: []uint64{2},
				}},
				Checksums: map[uint64]string{1: "checksum3", 2: "checksum4"},
			},
		}, nil, map[string]*llx.Primitive{
			"resolvedprop": llx.StringPrimitive("hello"),
		})
	b.AddDatapointType("checksum3", string(types.Bool))

	b.AddQuery(
		&llx.CodeBundle{
			CodeV2: &llx.CodeV2{
				Id: "query3",
				Blocks: []*llx.Block{{
					Datapoints: []uint64{1},
				}},
				Checksums: map[uint64]string{1: "checksum5"},
			},
		}, nil, nil)
	b.CollectDatapoint("checksum5")

	b.AddQuery(
		&llx.CodeBundle{
			CodeV2: &llx.CodeV2{
				Id: "query4",
				Blocks: []*llx.Block{{
					Datapoints: []uint64{1},
				}},
				Checksums: map[uint64]string{1: "checksum6"},
			},
		}, nil, nil)

	b.AddQuery(
		&llx.CodeBundle{
			CodeV2: &llx.CodeV2{
				Id: "query5",
				Blocks: []*llx.Block{{
					Datapoints: []uint64{1, 2},
				}},
				Checksums: map[uint64]string{1: "checksum5", 2: "checksum7"},
			},
			MinMondooVersion: "9999.9999.9999",
		}, nil, nil)

	b.AddReportingJob(&policy.ReportingJob{
		QrId:   "query1",
		Uuid:   "query1rj",
		Notify: []string{"policyrj"},
	})

	b.AddReportingJob(&policy.ReportingJob{
		QrId:       "query2",
		Uuid:       "query2rj",
		Datapoints: map[string]bool{"checksum3": true},
		Notify:     []string{"policyrj"},
	})

	b.AddReportingJob(&policy.ReportingJob{
		QrId: "policyqr",
		Uuid: "policyrj",
	})

	b.WithMondooVersion("100.0.0")

	asset := &asset.Asset{
		Mrn:         "assetMrn",
		PlatformIds: []string{"platformId"},
	}
	ge, err := b.Build(nil, nil, asset.Mrn)
	require.NoError(t, err)

	hasNode(t, ge, "execution_query/propertyquery", ExecutionQueryNodeType)
	hasOutEdges(t, ge, "execution_query/propertyquery", "checksum1", "pqep")

	hasNode(t, ge, "execution_query/query1", ExecutionQueryNodeType)
	hasOutEdges(t, ge, "execution_query/query1", "checksum2")

	hasNode(t, ge, "execution_query/query2", ExecutionQueryNodeType)
	hasOutEdges(t, ge, "execution_query/query2", "checksum3", "checksum4")

	hasNode(t, ge, "execution_query/query3", ExecutionQueryNodeType)
	hasOutEdges(t, ge, "execution_query/query3", "checksum5")

	hasNode(t, ge, "execution_query/query4", ExecutionQueryNodeType)
	hasOutEdges(t, ge, "execution_query/query4", "checksum6")

	assert.NotContains(t, ge.nodes, "execution_query/query5")
	assert.Nil(t, ge.nodes["checksum5"].data.(*DatapointNodeData).res)
	if assert.NotNil(t, ge.nodes["checksum7"].data.(*DatapointNodeData).res) {
		assert.Error(t, ge.nodes["checksum7"].data.(*DatapointNodeData).res.Data.Error)
	}

	hasNode(t, ge, "propertyquery", ReportingQueryNodeType)
	hasOutEdges(t, ge, "propertyquery")

	hasNode(t, ge, "query1", ReportingQueryNodeType)
	hasOutEdges(t, ge, "query1", "query1rj")

	hasNode(t, ge, "query2", ReportingQueryNodeType)
	hasOutEdges(t, ge, "query2", "query2rj")

	hasNode(t, ge, "query3", ReportingQueryNodeType)
	hasOutEdges(t, ge, "query3")

	hasNode(t, ge, "query4", ReportingQueryNodeType)
	hasOutEdges(t, ge, "query4")

	hasNode(t, ge, "query5", ReportingQueryNodeType)
	hasOutEdges(t, ge, "query5")

	hasNode(t, ge, "pqep", DatapointNodeType)
	hasOutEdges(t, ge, "pqep", "propertyquery", CollectionFinisherID)

	hasNode(t, ge, "checksum1", DatapointNodeType)
	hasOutEdges(t, ge, "checksum1", "execution_query/query1", CollectionFinisherID)

	hasNode(t, ge, "checksum2", DatapointNodeType)
	hasOutEdges(t, ge, "checksum2", "query1", CollectionFinisherID)

	hasNode(t, ge, "checksum3", DatapointNodeType)
	hasOutEdges(t, ge, "checksum3", "query2rj", DatapointCollectorID, CollectionFinisherID)

	hasNode(t, ge, "checksum4", DatapointNodeType)
	hasOutEdges(t, ge, "checksum4", "query2", CollectionFinisherID)

	hasNode(t, ge, "checksum5", DatapointNodeType)
	hasOutEdges(t, ge, "checksum5", DatapointCollectorID, CollectionFinisherID)

	hasNode(t, ge, "checksum6", DatapointNodeType)
	hasOutEdges(t, ge, "checksum6", CollectionFinisherID)

	hasNode(t, ge, "query1rj", ReportingJobNodeType)
	hasOutEdges(t, ge, "query1rj", "policyrj", ScoreCollectorID)

	hasNode(t, ge, "query2rj", ReportingJobNodeType)
	hasOutEdges(t, ge, "query2rj", "policyrj", ScoreCollectorID)

	hasNode(t, ge, "policyrj", ReportingJobNodeType)
	hasOutEdges(t, ge, "policyrj", ScoreCollectorID)
}

func hasNode(t *testing.T, ge *GraphExecutor, nodeID NodeID, nodeType NodeType) {
	t.Helper()
	if assert.Contains(t, ge.nodes, nodeID) {
		assert.Equal(t, nodeType, ge.nodes[nodeID].nodeType)
	}
}

func hasOutEdges(t *testing.T, ge *GraphExecutor, nodeID NodeID, edges ...NodeID) {
	t.Helper()
	require.Len(t, ge.edges[nodeID], len(edges))
	assert.ElementsMatch(t, ge.edges[nodeID], edges)
}
