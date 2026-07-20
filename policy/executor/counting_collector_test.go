// Copyright Mondoo, Inc. 2026
// SPDX-License-Identifier: BUSL-1.1

package executor

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/v13/policy"
	"go.mondoo.com/cnspec/v13/policy/scanstats"
	"go.mondoo.com/mql/v13/llx"
)

// buildTestResolvedPolicy builds a ResolvedPolicy that exercises every
// ReportingJob kind (mirroring the fixture in kinds_test.go).
func buildTestResolvedPolicy() *policy.ResolvedPolicy {
	return &policy.ResolvedPolicy{
		CollectorJob: &policy.CollectorJob{
			ReportingJobs: map[string]*policy.ReportingJob{
				"u0": {QrId: "root", Type: policy.ReportingJob_POLICY},                    // ignored (root)
				"u1": {QrId: "//q/check1", Type: policy.ReportingJob_CHECK},               // check
				"u2": {QrId: "//q/check2", Type: policy.ReportingJob_CHECK},               // check
				"u3": {QrId: "//q/cadq1", Type: policy.ReportingJob_CHECK_AND_DATA_QUERY}, // check
				"u4": {QrId: "//q/data1", Type: policy.ReportingJob_DATA_QUERY},           // data query
				"u5": {QrId: "//p/pol1", Type: policy.ReportingJob_POLICY},                // policy
				"u6": {QrId: "//c/ctrl1", Type: policy.ReportingJob_CONTROL},              // control
				"u7": {QrId: "//f/fw1", Type: policy.ReportingJob_FRAMEWORK},              // framework
				"u8": {QrId: "//r/risk1", Type: policy.ReportingJob_RISK_FACTOR},          // ignored
			},
		},
	}
}

func TestCountingCollector_ScoreErroredLastWriteWins(t *testing.T) {
	rp := buildTestResolvedPolicy()
	c := newCountingCollector(rp)

	// Sink an error score on check1 — should be counted.
	c.SinkScore([]*policy.Score{
		{QrId: "//q/check1", Type: policy.ScoreType_Error},
	})
	// Sink an ok result on check2 — should NOT be counted.
	c.SinkScore([]*policy.Score{
		{QrId: "//q/check2", Type: policy.ScoreType_Result},
	})
	// Sink error then result on cadq1 — last-write-wins: final state is NOT errored.
	c.SinkScore([]*policy.Score{
		{QrId: "//q/cadq1", Type: policy.ScoreType_Error},
	})
	c.SinkScore([]*policy.Score{
		{QrId: "//q/cadq1", Type: policy.ScoreType_Result},
	})
	// An error on a non-check qr_id (a policy) must NOT count.
	c.SinkScore([]*policy.Score{
		{QrId: "//p/pol1", Type: policy.ScoreType_Error},
	})

	stats := scanstats.New()
	c.recordTo(stats)

	proto := stats.ToProto()
	require.NotNil(t, proto)

	byName := map[string]int64{}
	for _, m := range proto.Metrics {
		byName[m.Name] = m.GetIntValue()
	}

	// Executed counts: 3 checks (check1, check2, cadq1), 1 data query, 1 policy, 1 control, 1 framework.
	require.Equal(t, int64(3), byName[scanstats.MetricChecks])
	require.Equal(t, int64(1), byName[scanstats.MetricDataQueries])
	require.Equal(t, int64(1), byName[scanstats.MetricPolicies])
	require.Equal(t, int64(1), byName[scanstats.MetricControls])
	require.Equal(t, int64(1), byName[scanstats.MetricFrameworks])

	// Only check1 remains errored (cadq1 was upgraded to Result).
	require.Equal(t, int64(1), byName[scanstats.MetricChecksErrored])
	// No data errors were sinked.
	require.Equal(t, int64(0), byName[scanstats.MetricDataQueriesErrored])
}

func TestCountingCollector_DataErroredLastWriteWins(t *testing.T) {
	rp := buildTestResolvedPolicy()
	c := newCountingCollector(rp)

	// Sink a data result with an error — should be counted.
	c.SinkData([]*llx.RawResult{
		{CodeID: "code-a", Data: &llx.RawData{Error: errors.New("boom")}},
	})
	// Sink a clean data result — should NOT be counted.
	c.SinkData([]*llx.RawResult{
		{CodeID: "code-b", Data: &llx.RawData{}},
	})
	// Same id: error then clean — last-write-wins: final state is NOT errored.
	c.SinkData([]*llx.RawResult{
		{CodeID: "code-c", Data: &llx.RawData{Error: errors.New("first")}},
	})
	c.SinkData([]*llx.RawResult{
		{CodeID: "code-c", Data: &llx.RawData{}},
	})

	stats := scanstats.New()
	c.recordTo(stats)

	proto := stats.ToProto()
	require.NotNil(t, proto)

	byName := map[string]int64{}
	for _, m := range proto.Metrics {
		byName[m.Name] = m.GetIntValue()
	}

	// Only code-a remains errored (code-c was upgraded to clean).
	require.Equal(t, int64(1), byName[scanstats.MetricDataQueriesErrored])
	require.Equal(t, int64(0), byName[scanstats.MetricChecksErrored])
}

func TestCountingCollector_NilInputsSafe(t *testing.T) {
	rp := buildTestResolvedPolicy()
	c := newCountingCollector(rp)

	// Must not panic on nil entries.
	c.SinkScore([]*policy.Score{nil})
	c.SinkData([]*llx.RawResult{nil})

	stats := scanstats.New()
	c.recordTo(stats)

	proto := stats.ToProto()
	require.NotNil(t, proto)

	byName := map[string]int64{}
	for _, m := range proto.Metrics {
		byName[m.Name] = m.GetIntValue()
	}
	require.Equal(t, int64(0), byName[scanstats.MetricChecksErrored])
	require.Equal(t, int64(0), byName[scanstats.MetricDataQueriesErrored])
}
