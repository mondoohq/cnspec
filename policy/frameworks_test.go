package policy

import (
	"testing"
)

func TestResolvedFrameworkTopologicalSort(t *testing.T) {
	framework := &ResolvedFramework{
		ReportTargets: map[string][]string{},
		ReportSources: map[string][]string{},
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
