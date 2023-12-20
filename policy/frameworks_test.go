// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveFramework_Exceptions(t *testing.T) {
	frameworks := map[string]*Framework{
		"framework-mrn1": {
			Mrn: "framework-mrn1",
			Groups: []*FrameworkGroup{
				{
					Controls: []*Control{
						{Mrn: "control-mrn1"},
						{Mrn: "control-mrn2"},
						{Mrn: "control-mrn3"},
						{Mrn: "control-mrn4"},
					},
				},
				{
					Type: GroupType_DISABLE,
					Controls: []*Control{
						{Mrn: "control-mrn1"},
					},
				},
				{
					Type: GroupType_OUT_OF_SCOPE,
					Controls: []*Control{
						{Mrn: "control-mrn2"},
					},
				},
				{
					Type:         GroupType_DISABLE,
					ReviewStatus: ReviewStatus_REJECTED,
					Controls: []*Control{
						{Mrn: "control-mrn3"},
					},
				},
				{
					Type:         GroupType_OUT_OF_SCOPE,
					ReviewStatus: ReviewStatus_REJECTED,
					Controls: []*Control{
						{Mrn: "control-mrn4"},
					},
				},
			},
			FrameworkMaps: []*FrameworkMap{
				{
					Controls: []*ControlMap{
						{Mrn: "control-mrn1"},
						{Mrn: "control-mrn2"},
						{Mrn: "control-mrn3"},
						{Mrn: "control-mrn4"},
					},
				},
			},
		},
	}

	resolved := ResolveFramework("framework-mrn1", frameworks)
	assert.NotNil(t, resolved)
	assert.Len(t, resolved.ReportTargets, 2)
	assert.NotContains(t, resolved.ReportTargets, "control-mrn1")
	assert.NotContains(t, resolved.ReportTargets, "control-mrn2")
	assert.Contains(t, resolved.ReportTargets, "control-mrn3")
	assert.Contains(t, resolved.ReportTargets, "control-mrn4")
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
