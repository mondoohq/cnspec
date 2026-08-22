// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package ocsf

import (
	"cmp"
	"slices"
)

// Events is one scan's worth of OCSF events, grouped by event class. Consumers
// that expect a single class per stream (Amazon Security Lake, an AWS Glue
// crawler) read them one class at a time; consumers that take a mixed stream
// read them all at once.
type Events struct {
	ComplianceFindings    []ComplianceFinding
	VulnerabilityFindings []VulnerabilityFinding
	InventoryInfos        []InventoryInfo
}

// classOrder is the order classes are written in, so a mixed stream and a set
// of per-class files always come out the same way.
var classOrder = []string{ClassComplianceFinding, ClassVulnerabilityFinding, ClassInventoryInfo}

// Len is the total number of events across all classes.
func (e *Events) Len() int {
	return len(e.ComplianceFindings) + len(e.VulnerabilityFindings) + len(e.InventoryInfos)
}

// Classes lists the event classes that carry at least one event, in write order.
func (e *Events) Classes() []string {
	var res []string
	for _, class := range classOrder {
		if e.count(class) > 0 {
			res = append(res, class)
		}
	}
	return res
}

func (e *Events) count(class string) int {
	switch class {
	case ClassComplianceFinding:
		return len(e.ComplianceFindings)
	case ClassVulnerabilityFinding:
		return len(e.VulnerabilityFindings)
	case ClassInventoryInfo:
		return len(e.InventoryInfos)
	}
	return 0
}

// Sort orders every class by event time, which is what Security Lake asks for,
// and breaks ties on the finding uid and the asset so two runs over the same
// report produce byte-identical output.
func (e *Events) Sort() {
	slices.SortStableFunc(e.ComplianceFindings, func(a, b ComplianceFinding) int {
		return cmp.Or(
			cmp.Compare(a.Time, b.Time),
			cmp.Compare(a.FindingInfo.UID, b.FindingInfo.UID),
			cmp.Compare(resourceUID(a.Resources), resourceUID(b.Resources)),
		)
	})
	slices.SortStableFunc(e.VulnerabilityFindings, func(a, b VulnerabilityFinding) int {
		return cmp.Or(
			cmp.Compare(a.Time, b.Time),
			cmp.Compare(a.FindingInfo.UID, b.FindingInfo.UID),
			cmp.Compare(resourceUID(a.Resources), resourceUID(b.Resources)),
		)
	})
	slices.SortStableFunc(e.InventoryInfos, func(a, b InventoryInfo) int {
		return cmp.Or(
			cmp.Compare(a.Time, b.Time),
			cmp.Compare(a.Device.UID, b.Device.UID),
		)
	})
}

func resourceUID(resources []ResourceDetails) string {
	if len(resources) == 0 {
		return ""
	}
	return resources[0].UID
}
