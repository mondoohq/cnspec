// Copyright Mondoo, Inc. 2026
// SPDX-License-Identifier: BUSL-1.1

package executor

import (
	"sync"

	"go.mondoo.com/cnspec/v13/policy"
	"go.mondoo.com/cnspec/v13/policy/scanstats"
	"go.mondoo.com/mql/v13/llx"
)

// countingCollector accumulates per-kind scan counts as scores and data results
// flow through the graph executor. Executed counts come from the resolved
// policy (via scanstats.PolicyKinds); errored counts are tracked with
// last-write-wins semantics per qr_id / data id, because a node's score/result
// can be re-emitted (e.g. error -> result) as datapoints arrive during
// execution. It implements the executor's ScoreCollector and DatapointCollector.
type countingCollector struct {
	mu            sync.Mutex
	pk            *scanstats.PolicyKinds
	erroredChecks map[string]struct{} // check qr_ids currently in error
	erroredData   map[string]struct{} // data ids currently in error
}

func newCountingCollector(rp *policy.ResolvedPolicy) *countingCollector {
	return &countingCollector{
		pk:            scanstats.NewPolicyKinds(rp),
		erroredChecks: map[string]struct{}{},
		erroredData:   map[string]struct{}{},
	}
}

func (c *countingCollector) SinkScore(scores []*policy.Score) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, s := range scores {
		if s == nil || !c.pk.IsCheckQrId(s.GetQrId()) {
			continue
		}
		if s.Type == policy.ScoreType_Error {
			c.erroredChecks[s.GetQrId()] = struct{}{}
		} else {
			delete(c.erroredChecks, s.GetQrId()) // upgraded away from error
		}
	}
}

// SinkData tracks data queries that returned a genuine query error. Note this
// counts real result errors on the raw stream; it does not count datapoints the
// downstream store later drops or replaces (e.g. an oversized payload swapped
// for a synthetic "datafield removed" marker), so the count reflects actual
// query failures rather than storage-side substitutions.
func (c *countingCollector) SinkData(results []*llx.RawResult) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, r := range results {
		if r == nil {
			continue
		}
		if r.Data != nil && r.Data.Error != nil {
			c.erroredData[r.CodeID] = struct{}{}
		} else {
			delete(c.erroredData, r.CodeID)
		}
	}
}

// recordTo writes the final per-kind counts to the scan-stats collector.
func (c *countingCollector) recordTo(stats *scanstats.Collector) {
	c.mu.Lock()
	counts := c.pk.Counts
	counts.ChecksErrored = int64(len(c.erroredChecks))
	counts.DataQueriesErrored = int64(len(c.erroredData))
	c.mu.Unlock()
	scanstats.RecordKindCounts(stats, counts)
}
