// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnspec/v13"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/upstream"
	"go.mondoo.com/mql/v13/providers-sdk/v1/upstream/health"
)

const (
	// maxConcurrentFailureReports bounds how many upstream failure reports may
	// be in flight at once. Each report is a synchronous HTTP POST inside the
	// mql health client (with its own 30s timeout), so this caps the number of
	// goroutines and sockets a burst of failures can consume.
	maxConcurrentFailureReports = 4
	// maxFailureReportsPerScan caps the total number of failure reports a single
	// scan will send upstream. A scan of a large, degraded fleet can produce
	// thousands of connect/discovery errors; reporting every one would flood the
	// platform and waste the scan's time, so we report a representative sample
	// and log how many were suppressed.
	maxFailureReportsPerScan = 100
	// failureReportDrainTimeout bounds how long we wait for outstanding reports
	// to flush at the end of a scan. The scan result is already complete at that
	// point, so we never block the user for long on telemetry.
	failureReportDrainTimeout = 10 * time.Second
)

// failureReporter sends per-asset scan failures upstream to the Mondoo Platform
// as failure reports, without ever blocking or stalling the scan itself.
//
// The previous implementation called health.ReportError synchronously at each
// error site. Because that call reads config from disk, builds a fresh HTTP
// client, and does a POST with a 30s, non-cancellable timeout, a fleet with
// many unreachable assets could serialize hundreds of such calls on the
// discovery/connect path and hang the scan before it started — the opposite of
// resilience. This type fixes that: reports are fire-and-forget with bounded
// concurrency, a per-scan cap, and drop-on-overload, so failure telemetry can
// never take down the scan it is reporting on.
//
// A nil *failureReporter is a valid no-op (used for incognito/offline scans),
// so every method is safe to call on nil.
type failureReporter struct {
	spaceMrn string

	// send performs the actual upstream delivery. It is a field so tests can
	// substitute a hook instead of hitting the network; production uses
	// health.ReportError.
	send func(msg string, tags map[string]string)

	sem       chan struct{}
	wg        sync.WaitGroup
	submitted atomic.Int64
	dropped   atomic.Int64
}

// newFailureReporter returns a reporter for upstream scans, or nil (a no-op)
// when there is no upstream to report to (nil config, incognito, or no API
// endpoint). The underlying health client additionally no-ops when no service
// account is configured.
func newFailureReporter(up *upstream.UpstreamConfig, spaceMrn string) *failureReporter {
	if up == nil || up.Incognito || up.ApiEndpoint == "" {
		return nil
	}
	if spaceMrn == "" {
		spaceMrn = up.SpaceMrn
	}
	return &failureReporter{
		spaceMrn: spaceMrn,
		sem:      make(chan struct{}, maxConcurrentFailureReports),
		send: func(msg string, tags map[string]string) {
			health.ReportError("cnspec", cnspec.Version, cnspec.Build, msg, health.WithTags(tags))
		},
	}
}

// report sends a single asset scan failure upstream asynchronously. It returns
// immediately; the actual POST happens on a bounded background goroutine. If
// the per-scan cap is reached or all report slots are busy, the report is
// dropped (and counted) rather than blocking the caller.
func (fr *failureReporter) report(asset *inventory.Asset, err error) {
	if fr == nil || asset == nil || err == nil {
		return
	}
	if !fr.tryAdmit() {
		return
	}

	// Snapshot everything the goroutine needs before returning; the caller may
	// mutate/close the asset (e.g. set tracked.Asset = nil) right after.
	tags := assetErrorTags(fr.spaceMrn, asset)
	msg := "scan failure: " + err.Error()

	fr.wg.Add(1)
	go func() {
		defer fr.wg.Done()
		defer func() { <-fr.sem }()
		// A failure in the reporting path must never crash the scan.
		defer func() {
			if r := recover(); r != nil {
				log.Error().Interface("panic", r).Msg("recovered panic while reporting scan failure upstream")
			}
		}()
		fr.send(msg, tags)
	}()
}

// tryAdmit reserves a slot for one report, enforcing both the per-scan total
// cap and the in-flight concurrency limit. It returns false (and counts a drop)
// when the cap is reached or all slots are busy. On success the caller owns the
// slot and must release it via <-fr.sem (the report goroutine does this).
func (fr *failureReporter) tryAdmit() bool {
	if fr.submitted.Add(1) > maxFailureReportsPerScan {
		fr.dropped.Add(1)
		return false
	}
	select {
	case fr.sem <- struct{}{}:
		return true
	default:
		// All report slots are busy — drop rather than stall the scan.
		fr.dropped.Add(1)
		return false
	}
}

// close waits (up to failureReportDrainTimeout) for outstanding reports to
// flush and logs how many reports were suppressed. It is safe to call on nil.
func (fr *failureReporter) close() {
	if fr == nil {
		return
	}
	done := make(chan struct{})
	go func() {
		fr.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(failureReportDrainTimeout):
		log.Warn().Msg("timed out flushing scan failure reports; continuing")
	}
	if d := fr.dropped.Load(); d > 0 {
		log.Warn().
			Int64("suppressed", d).
			Int("reported", maxFailureReportsPerScan).
			Msg("some scan failure reports were suppressed to avoid overloading the platform")
	}
}
