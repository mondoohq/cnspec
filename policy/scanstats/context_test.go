// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scanstats

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestContextWithCollector_RoundTrip(t *testing.T) {
	c := New()
	ctx := ContextWithCollector(context.Background(), c)
	got := CollectorFromContext(ctx)
	require.Same(t, c, got, "CollectorFromContext must return the same Collector stored by ContextWithCollector")
}

func TestCollectorFromContext_Absent(t *testing.T) {
	got := CollectorFromContext(context.Background())
	require.Nil(t, got, "CollectorFromContext must return nil when no Collector is in ctx")
}

func TestRecordKindCounts_SevenMetrics(t *testing.T) {
	c := New()
	k := KindCounts{
		Checks:             10,
		DataQueries:        3,
		Policies:           2,
		Controls:           4,
		Frameworks:         1,
		ChecksErrored:      2,
		DataQueriesErrored: 1,
	}
	RecordKindCounts(c, k)

	stats := c.ToProto()
	require.NotNil(t, stats)
	require.Len(t, stats.Metrics, 7)

	byName := map[string]int64{}
	for _, m := range stats.Metrics {
		byName[m.Name] = m.GetIntValue()
	}

	require.Equal(t, int64(10), byName[MetricChecks])
	require.Equal(t, int64(3), byName[MetricDataQueries])
	require.Equal(t, int64(2), byName[MetricPolicies])
	require.Equal(t, int64(4), byName[MetricControls])
	require.Equal(t, int64(1), byName[MetricFrameworks])
	require.Equal(t, int64(2), byName[MetricChecksErrored])
	require.Equal(t, int64(1), byName[MetricDataQueriesErrored])
}

func TestRecordKindCounts_NilCollector(t *testing.T) {
	// Must not panic
	RecordKindCounts(nil, KindCounts{Checks: 5})
}
