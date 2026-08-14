// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/v13/policy/scanstats"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
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

func TestLogMemoryStats_NoTrackerDoesNotPanic(t *testing.T) {
	// DEBUG_PROVIDER_MEMORY can be set on a scan whose dispatcher was built
	// without a tracker (unit tests, embedded callers).
	d := &scanDispatcher{scannedAssets: &atomic.Int64{}}
	require.NotPanics(t, func() {
		d.logMemoryStats(&inventory.Asset{Name: "test-asset"})
	})
}

func TestLogMemoryStats_UsesTrackerPeak(t *testing.T) {
	tr := scanstats.NewMemTracker(scanstats.MemTrackerConfig{
		RunID:  "run-abc",
		Sample: func() scanstats.Sample { return scanstats.Sample{RuntimeBytes: 4242, Goroutines: 9} },
	})
	tr.Observe()

	d := &scanDispatcher{scannedAssets: &atomic.Int64{}, memTracker: tr}
	require.NotPanics(t, func() {
		d.logMemoryStats(&inventory.Asset{Name: "test-asset"})
	})

	// The tracker, not a second independent reader, is the source of truth.
	peak, _, _ := tr.Peaks()
	require.Equal(t, uint64(4242), peak)
}
