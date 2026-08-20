// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"bytes"
	"sync/atomic"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/v13/policy/scanstats"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
)

func TestScanDispatcher_InFlightReflectsHeldScanSlots(t *testing.T) {
	d := &scanDispatcher{scanSem: make(chan struct{}, 4)}
	require.Equal(t, 0, d.inFlight())

	d.scanSem <- struct{}{}
	d.scanSem <- struct{}{}
	require.Equal(t, 2, d.inFlight())

	<-d.scanSem
	require.Equal(t, 1, d.inFlight())
}

func TestScanDispatcher_InFlightZeroWhenNoSemaphore(t *testing.T) {
	d := &scanDispatcher{}
	require.Equal(t, 0, d.inFlight())
}

func TestLogResourceStats_NoTrackerDoesNotPanic(t *testing.T) {
	// DEBUG_PROVIDER_MEMORY can be set on a scan whose dispatcher was built
	// without a tracker (unit tests, embedded callers).
	d := &scanDispatcher{scannedAssets: &atomic.Int64{}}
	require.NotPanics(t, func() {
		d.logResourceStats(&inventory.Asset{Name: "test-asset"})
	})
}

func TestLogResourceStats_UsesTrackerPeak(t *testing.T) {
	tr := scanstats.NewResourceTracker(scanstats.ResourceTrackerConfig{
		RunID:  "run-abc",
		Sample: func() scanstats.Sample { return scanstats.Sample{RuntimeBytes: 4242, Goroutines: 17} },
	})
	tr.SetInFlightFunc(func() int { return 5 })
	tr.Observe()

	d := &scanDispatcher{scannedAssets: &atomic.Int64{}, resourceTracker: tr}

	// logResourceStats writes through the package-level zerolog logger, so
	// capture that global for the duration of the test (not parallel-safe)
	// and assert on the emitted line, proving the logged values came from
	// the tracker rather than an independent reader.
	buf := &bytes.Buffer{}
	orig := log.Logger
	log.Logger = zerolog.New(buf)
	defer func() { log.Logger = orig }()

	d.logResourceStats(&inventory.Asset{Name: "test-asset"})

	out := buf.String()
	// goroutines_peak and in_flight_at_peak come only from the tracker;
	// runtime_peak_mb would be zero either way (4242 bytes truncates under
	// MB division), so it can't discriminate a working implementation.
	require.Contains(t, out, `"goroutines_peak":17`, out)
	require.Contains(t, out, `"in_flight_at_peak":5`, out)
}
