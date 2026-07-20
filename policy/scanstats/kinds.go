// Copyright Mondoo, Inc. 2026
// SPDX-License-Identifier: BUSL-1.1

package scanstats

import (
	"go.mondoo.com/cnspec/v13/policy"
	"go.mondoo.com/mql/v13/llx"
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

// CountByKind classifies a scan's scored entities by kind using the resolved
// policy's reporting jobs. Executed counts come from the reporting-job types
// (deduplicated by qr_id, excluding the asset-root aggregate). Errored counts
// come from the actual results: errored checks are error-typed scores mapped
// back to a CHECK reporting job; errored data queries are data results with a
// non-empty error.
func CountByKind(rp *policy.ResolvedPolicy, scores []*policy.Score, dataResults []*llx.Result) KindCounts {
	var c KindCounts
	if rp == nil || rp.GetCollectorJob() == nil {
		return c
	}

	// qr_id -> kind, deduped and excluding the asset-root aggregate node.
	kindByQrid := make(map[string]policy.ReportingJob_Type)
	for _, job := range rp.GetCollectorJob().GetReportingJobs() {
		qr := job.GetQrId()
		if qr == "" || qr == "root" {
			continue
		}
		kindByQrid[qr] = job.GetType()
	}

	for _, t := range kindByQrid {
		switch t {
		case policy.ReportingJob_CHECK, policy.ReportingJob_CHECK_AND_DATA_QUERY:
			c.Checks++
		case policy.ReportingJob_DATA_QUERY:
			c.DataQueries++
		case policy.ReportingJob_POLICY:
			c.Policies++
		case policy.ReportingJob_CONTROL:
			c.Controls++
		case policy.ReportingJob_FRAMEWORK:
			c.Frameworks++
		}
	}

	for _, s := range scores {
		if s == nil || s.Type != policy.ScoreType_Error {
			continue
		}
		switch kindByQrid[s.GetQrId()] {
		case policy.ReportingJob_CHECK, policy.ReportingJob_CHECK_AND_DATA_QUERY:
			c.ChecksErrored++
		}
	}

	for _, r := range dataResults {
		if r != nil && r.GetError() != "" {
			c.DataQueriesErrored++
		}
	}

	return c
}
