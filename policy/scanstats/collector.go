// Copyright Mondoo, Inc. 2026
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
	MetricScanDuration    = "cnspec.scan.duration"         // unit: ms
	MetricQueriesExecuted = "cnspec.scan.queries_executed" // unit: count
	MetricQueriesErrored  = "cnspec.scan.queries_errored"  // unit: count
	MetricUploadSize      = "cnspec.scan.upload_size"      // unit: bytes

	MetricChecks             = "cnspec.scan.checks"               // unit: count
	MetricDataQueries        = "cnspec.scan.data_queries"         // unit: count
	MetricPolicies           = "cnspec.scan.policies"             // unit: count
	MetricControls           = "cnspec.scan.controls"             // unit: count
	MetricFrameworks         = "cnspec.scan.frameworks"           // unit: count
	MetricChecksErrored      = "cnspec.scan.checks_errored"       // unit: count
	MetricDataQueriesErrored = "cnspec.scan.data_queries_errored" // unit: count
)

// Collector accumulates scan metrics. It is safe for concurrent use.
type Collector struct {
	mu      sync.Mutex
	metrics []*policy.Metric
}

// New returns an empty Collector.
func New() *Collector { return &Collector{} }

func (c *Collector) add(m *policy.Metric) {
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
