// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scanstats

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCollector_Empty_ToProtoNil(t *testing.T) {
	var c *Collector
	require.Nil(t, c.ToProto())     // nil receiver is safe
	require.Nil(t, New().ToProto()) // no metrics -> nil
}

func TestCollector_Adders(t *testing.T) {
	c := New()
	c.AddDuration(MetricScanDuration, 4200*time.Millisecond)
	c.AddInt(MetricChecks, "count", 128)
	c.AddDouble("cnspec.scan.avg_latency", "ms", 3.5)
	c.AddBool("cnspec.scan.truncated", true)

	stats := c.ToProto()
	require.Len(t, stats.Metrics, 4)

	require.Equal(t, MetricScanDuration, stats.Metrics[0].Name)
	require.Equal(t, "ms", stats.Metrics[0].Unit)
	require.Equal(t, int64(4200), stats.Metrics[0].GetIntValue())

	require.Equal(t, int64(128), stats.Metrics[1].GetIntValue())
	require.Equal(t, "count", stats.Metrics[1].Unit)

	require.Equal(t, 3.5, stats.Metrics[2].GetDoubleValue())
	require.Equal(t, true, stats.Metrics[3].GetBoolValue())
}

func TestCollector_ErroredAndUploadSizeConstants(t *testing.T) {
	c := New()
	c.AddInt(MetricChecksErrored, "count", 3)
	c.AddInt(MetricUploadSize, "bytes", 4096)
	stats := c.ToProto()
	require.Len(t, stats.Metrics, 2)
	require.Equal(t, "cnspec.scan.checks_errored", stats.Metrics[0].Name)
	require.Equal(t, int64(3), stats.Metrics[0].GetIntValue())
	require.Equal(t, "cnspec.scan.upload_size", stats.Metrics[1].Name)
	require.Equal(t, int64(4096), stats.Metrics[1].GetIntValue())
}

// A nil collector must not panic: add() is nil-tolerant like ToProto.
func TestCollector_NilReceiverDoesNotPanic(t *testing.T) {
	var c *Collector
	require.NotPanics(t, func() {
		c.AddInt(MetricUploadSize, "bytes", 1)
		c.AddDuration(MetricUploadDuration, time.Second)
		c.AddDouble(MetricUploadThroughput, "bps", 1.5)
		c.AddBool("x", true)
	})
	require.Nil(t, c.ToProto())
}

func TestCollector_AddString(t *testing.T) {
	c := New()
	c.AddString(MetricRunID, "1e8f7a0c-0000-4000-8000-000000000000")

	stats := c.ToProto()
	require.Len(t, stats.Metrics, 1)
	require.Equal(t, MetricRunID, stats.Metrics[0].Name)
	require.Equal(t, "1e8f7a0c-0000-4000-8000-000000000000", stats.Metrics[0].GetStringValue())
}

func TestCollector_AddStringNilReceiverIsSafe(t *testing.T) {
	var c *Collector
	require.NotPanics(t, func() { c.AddString(MetricRunID, "x") })
	require.Nil(t, c.ToProto())
}

func TestMemMetricNamesMatchWireContract(t *testing.T) {
	// These names are a wire contract (ADR-0004). They may be added to, but
	// never renamed or repurposed — downstream queries key on them.
	require.Equal(t, "cnspec.scan.run_id", MetricRunID)
	require.Equal(t, "cnspec.scan.mem.runtime_peak_bytes", MetricMemRuntimePeak)
	require.Equal(t, "cnspec.scan.mem.runtime_at_finish_bytes", MetricMemRuntimeAtFinish)
	require.Equal(t, "cnspec.scan.mem.goroutines_peak", MetricMemGoroutinesPeak)
	require.Equal(t, "cnspec.scan.mem.cgroup_current_bytes", MetricMemCgroupCurrent)
	require.Equal(t, "cnspec.scan.mem.cgroup_peak_bytes", MetricMemCgroupPeak)
	require.Equal(t, "cnspec.scan.mem.cgroup_max_bytes", MetricMemCgroupMax)
	require.Equal(t, "cnspec.scan.concurrency.in_flight_at_peak", MetricConcurrencyInFlightAtPeak)
	require.Equal(t, "cnspec.scan.concurrency.parallelism", MetricConcurrencyParallelism)
	require.Equal(t, "cnspec.scan.concurrency.max_connections", MetricConcurrencyMaxConnections)
}
