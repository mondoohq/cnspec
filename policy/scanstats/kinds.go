// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scanstats

import (
	"go.mondoo.com/cnspec/v13/policy"
)

// KindCounts holds per-kind counts of scored entities in a scan.
type KindCounts struct {
	Checks             int64
	DataQueries        int64
	Policies           int64
	Controls           int64
	Frameworks         int64
	ChecksErrored      int64
	DataQueriesErrored int64
}

// PolicyKinds classifies a scan's scored entities by kind from the resolved
// policy's reporting jobs. It holds the executed per-kind counts and can tell
// whether a score's qr_id belongs to a check (used to classify errored scores).
type PolicyKinds struct {
	Counts     KindCounts          // executed counts; the *Errored fields stay zero
	checkQrIds map[string]struct{} // qr_ids of CHECK / CHECK_AND_DATA_QUERY jobs
}

// NewPolicyKinds computes executed per-kind counts (deduped by qr_id, excluding
// the asset-root aggregate "root" and risk factors) and records the set of
// check qr_ids for later errored-score classification.
func NewPolicyKinds(rp *policy.ResolvedPolicy) *PolicyKinds {
	pk := &PolicyKinds{checkQrIds: map[string]struct{}{}}
	if rp == nil || rp.GetCollectorJob() == nil {
		return pk
	}
	kindByQrid := make(map[string]policy.ReportingJob_Type)
	for _, job := range rp.GetCollectorJob().GetReportingJobs() {
		qr := job.GetQrId()
		if qr == "" || qr == "root" {
			continue
		}
		kindByQrid[qr] = job.GetType()
	}
	for qr, t := range kindByQrid {
		switch t {
		case policy.ReportingJob_CHECK, policy.ReportingJob_CHECK_AND_DATA_QUERY:
			pk.Counts.Checks++
			pk.checkQrIds[qr] = struct{}{}
		case policy.ReportingJob_DATA_QUERY:
			pk.Counts.DataQueries++
		case policy.ReportingJob_POLICY:
			pk.Counts.Policies++
		case policy.ReportingJob_CONTROL:
			pk.Counts.Controls++
		case policy.ReportingJob_FRAMEWORK:
			pk.Counts.Frameworks++
		}
	}
	return pk
}

// IsCheckQrId reports whether qrid maps to a CHECK / CHECK_AND_DATA_QUERY job.
func (pk *PolicyKinds) IsCheckQrId(qrid string) bool {
	_, ok := pk.checkQrIds[qrid]
	return ok
}

// RecordKindCounts records the per-kind counts as the well-known scan metrics
// on the collector. No-op if c is nil.
func RecordKindCounts(c *Collector, k KindCounts) {
	if c == nil {
		return
	}
	c.AddInt(MetricChecks, "count", k.Checks)
	c.AddInt(MetricDataQueries, "count", k.DataQueries)
	c.AddInt(MetricPolicies, "count", k.Policies)
	c.AddInt(MetricControls, "count", k.Controls)
	c.AddInt(MetricFrameworks, "count", k.Frameworks)
	c.AddInt(MetricChecksErrored, "count", k.ChecksErrored)
	c.AddInt(MetricDataQueriesErrored, "count", k.DataQueriesErrored)
}
