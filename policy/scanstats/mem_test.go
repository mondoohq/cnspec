// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scanstats

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// fakeSampler returns each sample in turn, repeating the last one forever.
// Mutex-guarded because Start calls the sampler from its own goroutine while
// the test reads from the main one — an unguarded counter here fails -race.
func fakeSampler(samples ...Sample) SampleFunc {
	var mu sync.Mutex
	i := 0
	return func() Sample {
		mu.Lock()
		defer mu.Unlock()
		s := samples[i]
		if i < len(samples)-1 {
			i++
		}
		return s
	}
}

func TestMemTracker_TracksHighWaterNotLatest(t *testing.T) {
	tr := NewMemTracker(MemTrackerConfig{
		Sample: fakeSampler(
			Sample{RuntimeBytes: 100, Goroutines: 10},
			Sample{RuntimeBytes: 900, Goroutines: 90},
			Sample{RuntimeBytes: 300, Goroutines: 30},
		),
	})

	tr.Observe()
	tr.Observe()
	tr.Observe()

	// Peak is the maximum ever seen, not the most recent value.
	require.Equal(t, uint64(900), tr.peakRuntime)
	require.Equal(t, 90, tr.peakGoroutines)
	// The latest value is tracked separately.
	require.Equal(t, uint64(300), tr.lastRuntime)
}

func TestMemTracker_InFlightCapturedAtPeakNotAtLastSample(t *testing.T) {
	inFlight := 0
	tr := NewMemTracker(MemTrackerConfig{
		Sample: fakeSampler(
			Sample{RuntimeBytes: 100},
			Sample{RuntimeBytes: 900}, // peak happens here
			Sample{RuntimeBytes: 300},
		),
	})
	tr.SetInFlightFunc(func() int { return inFlight })

	inFlight = 2
	tr.Observe()
	inFlight = 48 // busy when the peak is set
	tr.Observe()
	inFlight = 1 // quiet again by the last sample
	tr.Observe()

	// The denominator must describe the moment of the peak, not the moment
	// of the final sample — otherwise the peak is uninterpretable.
	require.Equal(t, 48, tr.inFlightAtPeak)
}

func TestMemTracker_NilReceiverIsSafe(t *testing.T) {
	var tr *MemTracker
	require.NotPanics(t, func() {
		tr.Observe()
		tr.SetInFlightFunc(func() int { return 1 })
	})
}

func TestMemTracker_NoInFlightFuncDoesNotPanic(t *testing.T) {
	tr := NewMemTracker(MemTrackerConfig{
		Sample: fakeSampler(Sample{RuntimeBytes: 100}),
	})
	require.NotPanics(t, func() { tr.Observe() })
	require.Equal(t, 0, tr.inFlightAtPeak)
}
