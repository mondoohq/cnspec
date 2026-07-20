// Copyright Mondoo, Inc. 2026
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
