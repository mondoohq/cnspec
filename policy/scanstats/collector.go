// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Package scanstats collects namespaced metrics about a scan and converts them
// into the policy.ScanStatistics proto attached to ReportUploadCompleted.
package scanstats

import (
	"sync"
	"time"

	"go.mondoo.com/cnspec/v13/policy"
)

// Well-known core metric names. Core cnspec metrics use the "cnspec.scan.*"
// namespace; provider-contributed metrics use "provider.<name>.*".
const (
	MetricScanDuration = "cnspec.scan.duration"    // unit: ms
	MetricUploadSize   = "cnspec.scan.upload_size" // unit: bytes
	// MetricUploadDuration / MetricUploadThroughput give the success-path
	// baseline for upload latency. Without them a slow failing upload cannot
	// be compared against the normal distribution.
	MetricUploadDuration   = "cnspec.scan.upload_duration"       // unit: ms
	MetricUploadThroughput = "cnspec.scan.upload_throughput_bps" // unit: bps

	MetricChecks             = "cnspec.scan.checks"               // unit: count
	MetricDataQueries        = "cnspec.scan.data_queries"         // unit: count
	MetricPolicies           = "cnspec.scan.policies"             // unit: count
	MetricControls           = "cnspec.scan.controls"             // unit: count
	MetricFrameworks         = "cnspec.scan.frameworks"           // unit: count
	MetricChecksErrored      = "cnspec.scan.checks_errored"       // unit: count
	MetricDataQueriesErrored = "cnspec.scan.data_queries_errored" // unit: count

	// MetricRunID identifies the scan process a record came from. The memory
	// metrics below are process-wide but emitted once per asset, so a run
	// scanning N assets produces N records carrying one process's rising
	// high-water mark. Without this, those records cannot be grouped and the
	// only correct aggregation (max per run) is inexpressible.
	MetricRunID = "cnspec.scan.run_id"

	MetricMemRuntimePeak     = "cnspec.scan.mem.runtime_peak_bytes"      // unit: bytes
	MetricMemRuntimeAtFinish = "cnspec.scan.mem.runtime_at_finish_bytes" // unit: bytes
	MetricMemGoroutinesPeak  = "cnspec.scan.mem.goroutines_peak"         // unit: count
	MetricMemCgroupCurrent   = "cnspec.scan.mem.cgroup_current_bytes"    // unit: bytes
	MetricMemCgroupPeak      = "cnspec.scan.mem.cgroup_peak_bytes"       // unit: bytes
	MetricMemCgroupMax       = "cnspec.scan.mem.cgroup_max_bytes"        // unit: bytes

	MetricConcurrencyInFlightAtPeak = "cnspec.scan.concurrency.in_flight_at_peak" // unit: count
	MetricConcurrencyParallelism    = "cnspec.scan.concurrency.parallelism"       // unit: count
	MetricConcurrencyMaxConnections = "cnspec.scan.concurrency.max_connections"   // unit: count

	// CPU metrics are cumulative over the run, not high-water marks like the
	// memory ones: each is the difference between the run's first and latest
	// observation. Note that "available" is CPU the process could have used
	// (roughly GOMAXPROCS x wall time) — "busy" is what it actually burned.
	MetricCPUBusySeconds      = "cnspec.scan.cpu.busy_seconds"      // unit: s
	MetricCPUAvailableSeconds = "cnspec.scan.cpu.available_seconds" // unit: s
	MetricCPUUserSeconds      = "cnspec.scan.cpu.user_seconds"      // unit: s
	MetricCPUGCSeconds        = "cnspec.scan.cpu.gc_seconds"        // unit: s
	MetricCPUScavengeSeconds  = "cnspec.scan.cpu.scavenge_seconds"  // unit: s
	MetricCPUGCFraction       = "cnspec.scan.cpu.gc_fraction"       // gc / busy
	MetricCPUUtilization      = "cnspec.scan.cpu.utilization"       // busy / available
	MetricCPUGOMAXPROCS       = "cnspec.scan.cpu.gomaxprocs"        // unit: count

	// Write-time scan-content checksums (feature-gated; see the sqlite
	// datalake and scandb.WithWriteTimeChecksums). Shadow mode exists to
	// measure the cost fleet-wide before comparison ships enabled, and
	// duration is that measurement: pure accumulated hashing time across
	// the scan's writes (there is no separate pass to time). Errors is the
	// count of rows whose hash failed — checksum work never fails a scan
	// (the row is stored without a checksum and the file stays unstamped,
	// fail-open), so this uploaded count is the only way a silent hashing
	// problem becomes visible fleet-wide. Per-kind row counts stay in the
	// local log only — row volume is already visible upstream through the
	// scan database itself.
	MetricChecksumDuration = "cnspec.scan.checksum.duration" // unit: ms
	MetricChecksumErrors   = "cnspec.scan.checksum.errors"   // unit: count
)

// Collector accumulates scan metrics. It is safe for concurrent use.
type Collector struct {
	mu      sync.Mutex
	metrics []*policy.Metric
}

// New returns an empty Collector.
func New() *Collector { return &Collector{} }

func (c *Collector) add(m *policy.Metric) {
	// Nil-tolerant, matching ToProto below: recording a metric should never
	// panic a scan just because a caller had no collector. Metrics added to a
	// nil collector are dropped.
	if c == nil {
		return
	}
	c.mu.Lock()
	c.metrics = append(c.metrics, m)
	c.mu.Unlock()
}

// AddInt records an integer metric (counts, byte sizes, unix timestamps).
func (c *Collector) AddInt(name, unit string, v int64) {
	c.add(&policy.Metric{Name: name, Unit: unit, Value: &policy.Metric_IntValue{IntValue: v}})
}

// AddDuration records a duration as an integer number of milliseconds.
func (c *Collector) AddDuration(name string, d time.Duration) {
	c.AddInt(name, "ms", d.Milliseconds())
}

// AddDouble records a floating-point metric (ratios, averages).
func (c *Collector) AddDouble(name, unit string, v float64) {
	c.add(&policy.Metric{Name: name, Unit: unit, Value: &policy.Metric_DoubleValue{DoubleValue: v}})
}

// AddBool records a boolean flag metric.
func (c *Collector) AddBool(name string, v bool) {
	c.add(&policy.Metric{Name: name, Value: &policy.Metric_BoolValue{BoolValue: v}})
}

// AddString records a string metric (identifiers, labels). No unit applies.
func (c *Collector) AddString(name, v string) {
	c.add(&policy.Metric{Name: name, Value: &policy.Metric_StringValue{StringValue: v}})
}

// ToProto returns the collected metrics as a ScanStatistics, or nil when the
// collector is nil or has no metrics.
func (c *Collector) ToProto() *policy.ScanStatistics {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.metrics) == 0 {
		return nil
	}
	out := make([]*policy.Metric, len(c.metrics))
	copy(out, c.metrics)
	return &policy.ScanStatistics{Metrics: out}
}
