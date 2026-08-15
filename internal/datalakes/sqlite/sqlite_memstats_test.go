// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package sqlite

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/v13/policy/scanstats"
)

func TestRecordMemStats_WritesTrackerIntoCollector(t *testing.T) {
	tr := scanstats.NewMemTracker(scanstats.MemTrackerConfig{
		RunID: "run-abc",
		// Avoid reading the real /sys/fs/cgroup default on a containerized
		// CI runner.
		CgroupRoot: filepath.Join(t.TempDir(), "absent"),
	})
	ctx := scanstats.ContextWithMemTracker(context.Background(), tr)

	c := scanstats.New()
	recordMemStats(ctx, c)

	stats := c.ToProto()
	require.NotNil(t, stats)

	names := map[string]bool{}
	for _, m := range stats.Metrics {
		names[m.Name] = true
	}
	require.True(t, names[scanstats.MetricRunID])
	require.True(t, names[scanstats.MetricMemRuntimePeak])
	require.True(t, names[scanstats.MetricConcurrencyParallelism])
}

func TestRecordMemStats_NoTrackerOnContextIsNoop(t *testing.T) {
	// Scans that never installed a tracker must still upload cleanly.
	c := scanstats.New()
	require.NotPanics(t, func() { recordMemStats(context.Background(), c) })
	require.Nil(t, c.ToProto())
}
