// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package loadtest

import (
	"context"
	"net/http"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

// Metrics is the typed instrument wrapper the runner uses to record loadtest
// activity. We expose via Prometheus by registering the OTel Prometheus
// exporter with a SDK MeterProvider — same metrics surface, scrape format
// the platform team is already wired up for.
type Metrics struct {
	provider *sdkmetric.MeterProvider

	syncDuration    metric.Float64Histogram
	resolveDuration metric.Float64Histogram
	uploadDuration  metric.Float64Histogram

	syncCounter    metric.Int64Counter
	resolveCounter metric.Int64Counter
	uploadCounter  metric.Int64Counter

	inFlight metric.Int64UpDownCounter
}

// NewMetrics builds a Prometheus-exporting MeterProvider, wires up the
// loadtest instruments, and returns the http.Handler that exposes /metrics.
// Caller is responsible for calling Shutdown when the run finishes.
func NewMetrics(ctx context.Context) (*Metrics, http.Handler, error) {
	exporter, err := prometheus.New()
	if err != nil {
		return nil, nil, errors.Wrap(err, "create prometheus exporter")
	}

	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(exporter))
	meter := provider.Meter("go.mondoo.com/cnspec/loadtest")

	syncDuration, err := meter.Float64Histogram(
		"cnspec.loadtest.sync.duration",
		metric.WithDescription("Duration of SynchronizeAssets calls."),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, nil, err
	}
	resolveDuration, err := meter.Float64Histogram(
		"cnspec.loadtest.resolve.duration",
		metric.WithDescription("Duration of ResolveAndUpdateJobs calls."),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, nil, err
	}
	uploadDuration, err := meter.Float64Histogram(
		"cnspec.loadtest.upload.duration",
		metric.WithDescription("Duration of scan-db upload (GetUploadURL → PUT → ReportUploadCompleted)."),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, nil, err
	}

	syncCounter, err := meter.Int64Counter(
		"cnspec.loadtest.sync.calls",
		metric.WithDescription("SynchronizeAssets call count."),
	)
	if err != nil {
		return nil, nil, err
	}
	resolveCounter, err := meter.Int64Counter(
		"cnspec.loadtest.resolve.calls",
		metric.WithDescription("ResolveAndUpdateJobs call count."),
	)
	if err != nil {
		return nil, nil, err
	}
	uploadCounter, err := meter.Int64Counter(
		"cnspec.loadtest.upload.calls",
		metric.WithDescription("Scan-db upload count."),
	)
	if err != nil {
		return nil, nil, err
	}

	inFlight, err := meter.Int64UpDownCounter(
		"cnspec.loadtest.scans.in_flight",
		metric.WithDescription("Scans currently being processed by a worker."),
	)
	if err != nil {
		return nil, nil, err
	}

	return &Metrics{
		provider:        provider,
		syncDuration:    syncDuration,
		resolveDuration: resolveDuration,
		uploadDuration:  uploadDuration,
		syncCounter:     syncCounter,
		resolveCounter:  resolveCounter,
		uploadCounter:   uploadCounter,
		inFlight:        inFlight,
	}, promhttp.Handler(), nil
}

// Shutdown flushes any buffered metric data and releases provider resources.
// Safe to call on a nil receiver so the runner can defer it unconditionally.
func (m *Metrics) Shutdown(ctx context.Context) error {
	if m == nil || m.provider == nil {
		return nil
	}
	return m.provider.Shutdown(ctx)
}

func statusAttr(err error) attribute.KeyValue {
	if err != nil {
		return attribute.String("status", "error")
	}
	return attribute.String("status", "ok")
}

// recordSync records the result of a SynchronizeAssets call. dur is the
// wall-clock time the RPC took; err is non-nil on failure. Safe on a nil
// receiver — callers don't have to check before recording.
func (m *Metrics) recordSync(ctx context.Context, dur time.Duration, err error) {
	if m == nil {
		return
	}
	m.syncDuration.Record(ctx, dur.Seconds(), metric.WithAttributes(statusAttr(err)))
	m.syncCounter.Add(ctx, 1, metric.WithAttributes(statusAttr(err)))
}

func (m *Metrics) recordResolve(ctx context.Context, dur time.Duration, err error) {
	if m == nil {
		return
	}
	m.resolveDuration.Record(ctx, dur.Seconds(), metric.WithAttributes(statusAttr(err)))
	m.resolveCounter.Add(ctx, 1, metric.WithAttributes(statusAttr(err)))
}

func (m *Metrics) recordUpload(ctx context.Context, dur time.Duration, err error) {
	if m == nil {
		return
	}
	m.uploadDuration.Record(ctx, dur.Seconds(), metric.WithAttributes(statusAttr(err)))
	m.uploadCounter.Add(ctx, 1, metric.WithAttributes(statusAttr(err)))
}

func (m *Metrics) inFlightAdd(ctx context.Context, delta int64) {
	if m == nil {
		return
	}
	m.inFlight.Add(ctx, delta)
}
