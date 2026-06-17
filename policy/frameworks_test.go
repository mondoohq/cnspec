// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateChecksums_Group_ReviewStatus(t *testing.T) {
	loader := DefaultBundleLoader()
	bundle, err := loader.BundleFromPaths("scan/testdata/compliance-bundle.mql.yaml")
	require.NoError(t, err)

	ctx := context.Background()
	framework := bundle.Frameworks[0]
	err = framework.UpdateChecksums(ctx, nil, nil, bundle.ToMap())
	require.NoError(t, err)

	oldChecksum := framework.GraphExecutionChecksum

	framework.Groups[0].ReviewStatus = ReviewStatus_REJECTED
	framework.ClearExecutionChecksums()

	err = framework.UpdateChecksums(ctx, nil, nil, bundle.ToMap())
	require.NoError(t, err)

	// Make sure the execution checksum changes when the review status changed.
	assert.NotEqual(t, oldChecksum, framework.GraphExecutionChecksum)
}

func TestResolvedFrameworkTopologicalSort(t *testing.T) {
	framework := &ResolvedFramework{
		ReportTargets: map[string]ResolvedFrameworkReferenceSet{},
		ReportSources: map[string]ResolvedFrameworkReferenceSet{},
		Nodes:         map[string]ResolvedFrameworkNode{},
	}

	framework.addReportLink(ResolvedFrameworkNode{Mrn: "z"}, ResolvedFrameworkNode{Mrn: "c"})
	framework.addReportLink(ResolvedFrameworkNode{Mrn: "y"}, ResolvedFrameworkNode{Mrn: "x"})
	framework.addReportLink(ResolvedFrameworkNode{Mrn: "a"}, ResolvedFrameworkNode{Mrn: "b"})
	framework.addReportLink(ResolvedFrameworkNode{Mrn: "b"}, ResolvedFrameworkNode{Mrn: "c"})
	framework.addReportLink(ResolvedFrameworkNode{Mrn: "c"}, ResolvedFrameworkNode{Mrn: "d"})
	framework.addReportLink(ResolvedFrameworkNode{Mrn: "c"}, ResolvedFrameworkNode{Mrn: "e"})
	framework.addReportLink(ResolvedFrameworkNode{Mrn: "b"}, ResolvedFrameworkNode{Mrn: "e"})

	sorted := framework.TopologicalSort()

	requireComesAfter(t, sorted, "z", "c")
	requireComesAfter(t, sorted, "y", "x")
	requireComesAfter(t, sorted, "a", "b")
	requireComesAfter(t, sorted, "b", "c")
	requireComesAfter(t, sorted, "c", "d")
	requireComesAfter(t, sorted, "c", "e")
	requireComesAfter(t, sorted, "b", "e")
}

func requireComesAfter(t *testing.T, sorted []string, a, b string) {
	t.Helper()
	aIdx := -1
	bIdx := -1
	for i, v := range sorted {
		if v == a {
			aIdx = i
		}
		if v == b {
			bIdx = i
		}
	}
	if aIdx == -1 {
		t.Errorf("Expected %s to be in sorted list", a)
	}
	if bIdx == -1 {
		t.Errorf("Expected %s to be in sorted list", b)
	}
	if aIdx < bIdx {
		t.Errorf("Expected %s to come after %s", a, b)
	}
}

// A framework's graph checksums are an order-sensitive rolling hash over its
// framework maps. The maps are loaded without a stable ORDER BY, so the
// checksum must not depend on the order they happen to arrive in — otherwise
// the same framework yields different graph_content_checksum values across
// reads, which makes the PolicyBundle rebuild persist CAS never converge.
func TestFrameworkGraphChecksum_StableAcrossFrameworkMapOrder(t *testing.T) {
	ctx := context.Background()

	mk := func(maps []*FrameworkMap) *Framework {
		return &Framework{
			Mrn:                    "//framework.api.mondoo.app/frameworks/test",
			LocalContentChecksum:   "local-content",
			LocalExecutionChecksum: "local-exec",
			FrameworkMaps:          maps,
		}
	}
	a := &FrameworkMap{Mrn: "//fm/a", LocalContentChecksum: "ca", LocalExecutionChecksum: "ea"}
	b := &FrameworkMap{Mrn: "//fm/b", LocalContentChecksum: "cb", LocalExecutionChecksum: "eb"}

	f1 := mk([]*FrameworkMap{a, b})
	f2 := mk([]*FrameworkMap{b, a}) // same maps, reversed load order

	require.NoError(t, f1.UpdateChecksums(ctx, nil, nil, nil))
	require.NoError(t, f2.UpdateChecksums(ctx, nil, nil, nil))

	assert.Equal(t, f1.GraphContentChecksum, f2.GraphContentChecksum,
		"graph_content_checksum must not depend on FrameworkMap load order")
	assert.Equal(t, f1.GraphExecutionChecksum, f2.GraphExecutionChecksum,
		"graph_execution_checksum must not depend on FrameworkMap load order")
}
