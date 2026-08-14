// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scanstats

import (
	"os"
	"path/filepath"
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

func writeCgroupFile(t *testing.T, dir, name, contents string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(contents), 0o600))
}

func TestReadCgroup_AllPresent(t *testing.T) {
	dir := t.TempDir()
	writeCgroupFile(t, dir, "memory.current", "3402137600\n")
	writeCgroupFile(t, dir, "memory.peak", "3500000000\n")
	writeCgroupFile(t, dir, "memory.max", "4294967296\n")

	cg := readCgroup(dir)
	require.True(t, cg.hasCurrent)
	require.Equal(t, uint64(3402137600), cg.current)
	require.True(t, cg.hasPeak)
	require.Equal(t, uint64(3500000000), cg.peak)
	require.True(t, cg.hasMax)
	require.Equal(t, uint64(4294967296), cg.max)
}

func TestReadCgroup_AbsentDirectoryReportsNothing(t *testing.T) {
	// Non-Linux hosts and cgroup v1 systems: every value must be absent,
	// never zero. A zero would be read downstream as a real measurement.
	cg := readCgroup(filepath.Join(t.TempDir(), "does-not-exist"))
	require.False(t, cg.hasCurrent)
	require.False(t, cg.hasPeak)
	require.False(t, cg.hasMax)
}

func TestReadCgroup_UnlimitedMaxIsAbsent(t *testing.T) {
	// "max" means no limit. Reporting it as a number would invent a ceiling
	// that does not exist and make headroom ratios meaningless.
	dir := t.TempDir()
	writeCgroupFile(t, dir, "memory.current", "1024\n")
	writeCgroupFile(t, dir, "memory.max", "max\n")

	cg := readCgroup(dir)
	require.True(t, cg.hasCurrent)
	require.False(t, cg.hasMax)
}

func TestReadCgroup_MissingPeakOnOlderKernel(t *testing.T) {
	// memory.peak requires Linux 5.19+; the others must still be reported.
	dir := t.TempDir()
	writeCgroupFile(t, dir, "memory.current", "2048\n")
	writeCgroupFile(t, dir, "memory.max", "8192\n")

	cg := readCgroup(dir)
	require.True(t, cg.hasCurrent)
	require.False(t, cg.hasPeak)
	require.True(t, cg.hasMax)
}

func TestReadCgroup_MalformedValueIsAbsent(t *testing.T) {
	dir := t.TempDir()
	writeCgroupFile(t, dir, "memory.current", "not-a-number\n")

	cg := readCgroup(dir)
	require.False(t, cg.hasCurrent)
}
