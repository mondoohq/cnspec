// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scanstats

import (
	"math"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/v13/policy"
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

func TestDefaultSample_ReturnsLiveValues(t *testing.T) {
	s := defaultSample()

	// A running Go process always has a non-zero footprint and at least
	// this test's own goroutine. The stub returned a zero Sample.
	require.Greater(t, s.RuntimeBytes, uint64(0))
	require.Greater(t, s.Goroutines, 0)
}

// metricByName indexes a ScanStatistics for assertion by metric name.
func metricByName(t *testing.T, c *Collector) map[string]*policy.Metric {
	t.Helper()
	stats := c.ToProto()
	require.NotNil(t, stats)
	out := map[string]*policy.Metric{}
	for _, m := range stats.Metrics {
		out[m.Name] = m
	}
	return out
}

func TestMemTracker_RecordWritesPeaksAndRunConstants(t *testing.T) {
	dir := t.TempDir()
	writeCgroupFile(t, dir, "memory.current", "3402137600\n")
	writeCgroupFile(t, dir, "memory.peak", "3500000000\n")
	writeCgroupFile(t, dir, "memory.max", "4294967296\n")

	tr := NewMemTracker(MemTrackerConfig{
		RunID:          "run-abc",
		Parallelism:    16,
		MaxConnections: 50,
		CgroupRoot:     dir,
		Sample: fakeSampler(
			Sample{RuntimeBytes: 100, Goroutines: 10},
			Sample{RuntimeBytes: 900, Goroutines: 90},
			Sample{RuntimeBytes: 300, Goroutines: 30},
		),
	})
	tr.SetInFlightFunc(func() int { return 7 })
	tr.Observe()
	tr.Observe()
	tr.Observe()

	c := New()
	tr.Record(c)
	m := metricByName(t, c)

	require.Equal(t, "run-abc", m[MetricRunID].GetStringValue())
	require.Equal(t, int64(900), m[MetricMemRuntimePeak].GetIntValue())
	require.Equal(t, "bytes", m[MetricMemRuntimePeak].Unit)
	require.Equal(t, int64(300), m[MetricMemRuntimeAtFinish].GetIntValue())
	require.Equal(t, int64(90), m[MetricMemGoroutinesPeak].GetIntValue())
	require.Equal(t, int64(7), m[MetricConcurrencyInFlightAtPeak].GetIntValue())
	require.Equal(t, int64(16), m[MetricConcurrencyParallelism].GetIntValue())
	require.Equal(t, int64(50), m[MetricConcurrencyMaxConnections].GetIntValue())
	require.Equal(t, int64(3402137600), m[MetricMemCgroupCurrent].GetIntValue())
	require.Equal(t, int64(3500000000), m[MetricMemCgroupPeak].GetIntValue())
	require.Equal(t, int64(4294967296), m[MetricMemCgroupMax].GetIntValue())
}

func TestMemTracker_RecordOmitsCgroupMetricsWhenUnavailable(t *testing.T) {
	tr := NewMemTracker(MemTrackerConfig{
		RunID:      "run-abc",
		CgroupRoot: filepath.Join(t.TempDir(), "absent"),
		Sample:     fakeSampler(Sample{RuntimeBytes: 100, Goroutines: 10}),
	})
	tr.Observe()

	c := New()
	tr.Record(c)
	m := metricByName(t, c)

	// Absent, not zero — a zero would be read downstream as a real limit.
	require.NotContains(t, m, MetricMemCgroupCurrent)
	require.NotContains(t, m, MetricMemCgroupPeak)
	require.NotContains(t, m, MetricMemCgroupMax)
	// Runtime metrics are unaffected by the cgroup being unavailable.
	require.Contains(t, m, MetricMemRuntimePeak)
}

func TestMemTracker_RecordOmitsRunIDWhenUnset(t *testing.T) {
	tr := NewMemTracker(MemTrackerConfig{
		CgroupRoot: filepath.Join(t.TempDir(), "absent"),
		Sample:     fakeSampler(Sample{RuntimeBytes: 100}),
	})
	tr.Observe()

	c := New()
	tr.Record(c)
	require.NotContains(t, metricByName(t, c), MetricRunID)
}

func TestMemTracker_RecordNilSafe(t *testing.T) {
	var tr *MemTracker
	c := New()
	require.NotPanics(t, func() { tr.Record(c) })
	require.Nil(t, c.ToProto())

	real := NewMemTracker(MemTrackerConfig{Sample: fakeSampler(Sample{RuntimeBytes: 1})})
	require.NotPanics(t, func() { real.Record(nil) })
}

func TestMemTracker_StartStopObservesAndIsIdempotent(t *testing.T) {
	tr := NewMemTracker(MemTrackerConfig{
		CgroupRoot: filepath.Join(t.TempDir(), "absent"),
		Sample:     fakeSampler(Sample{RuntimeBytes: 500, Goroutines: 5}),
	})

	tr.Start(time.Millisecond)
	require.Eventually(t, func() bool {
		tr.mu.Lock()
		defer tr.mu.Unlock()
		return tr.peakRuntime == 500
	}, 2*time.Second, 5*time.Millisecond)

	tr.Stop()
	require.NotPanics(t, func() { tr.Stop() }) // double Stop must not panic

	var nilTr *MemTracker
	require.NotPanics(t, func() {
		nilTr.Start(time.Millisecond)
		nilTr.Stop()
	})
}

func TestMemTracker_RecordOmitsRuntimeMetricsWhenNoSampleEverResolved(t *testing.T) {
	// A zero-byte sample simulates every runtime/metrics name failing to
	// resolve: defaultSample falls back to 0 rather than a real footprint.
	tr := NewMemTracker(MemTrackerConfig{
		RunID:      "run-abc",
		CgroupRoot: filepath.Join(t.TempDir(), "absent"),
		Sample:     fakeSampler(Sample{RuntimeBytes: 0, Goroutines: 0}),
	})

	c := New()
	tr.Record(c)
	m := metricByName(t, c)

	// Absent, not zero: recording a zero here would be indistinguishable
	// downstream from a real measurement and corrupt fleet-wide aggregates.
	require.NotContains(t, m, MetricMemRuntimePeak)
	require.NotContains(t, m, MetricMemRuntimeAtFinish)
	require.NotContains(t, m, MetricMemGoroutinesPeak)
	// Concurrency metrics and RunID are unconditional and must still be
	// present even though no runtime sample was ever resolved.
	require.Contains(t, m, MetricRunID)
	require.Contains(t, m, MetricConcurrencyInFlightAtPeak)
	require.Contains(t, m, MetricConcurrencyParallelism)
	require.Contains(t, m, MetricConcurrencyMaxConnections)
}

func TestMemTracker_RecordRefreshesAtFinishRatherThanUsingStaleSample(t *testing.T) {
	tr := NewMemTracker(MemTrackerConfig{
		CgroupRoot: filepath.Join(t.TempDir(), "absent"),
		Sample: fakeSampler(
			Sample{RuntimeBytes: 100, Goroutines: 1},
			Sample{RuntimeBytes: 200, Goroutines: 2},
		),
	})
	tr.Observe() // simulates the periodic sampler firing once, a tick ago

	c := New()
	tr.Record(c)
	m := metricByName(t, c)

	// Record must pull a fresh sample rather than reporting the value from
	// the last periodic tick, or every asset finishing inside one sampling
	// window would report an identical, stale value.
	require.Equal(t, int64(200), m[MetricMemRuntimeAtFinish].GetIntValue())
	// The peak also reflects the fresh sample since it is the larger value.
	require.Equal(t, int64(200), m[MetricMemRuntimePeak].GetIntValue())
}

func TestMemTracker_SnapshotReportsPresentAndAbsentCgroupValues(t *testing.T) {
	dir := t.TempDir()
	writeCgroupFile(t, dir, "memory.current", "1048576\n")
	writeCgroupFile(t, dir, "memory.max", "2097152\n")

	tr := NewMemTracker(MemTrackerConfig{
		CgroupRoot: dir,
		Sample:     fakeSampler(Sample{RuntimeBytes: 555, Goroutines: 5}),
	})
	tr.Observe()

	snap := tr.Snapshot()
	require.True(t, snap.HasRuntime)
	require.Equal(t, uint64(555), snap.RuntimeBytes)
	require.True(t, snap.HasCgroupCurrent)
	require.Equal(t, uint64(1048576), snap.CgroupCurrent)
	require.True(t, snap.HasCgroupMax)
	require.Equal(t, uint64(2097152), snap.CgroupMax)
}

func TestMemTracker_SnapshotReportsAbsentWhenCgroupRootMissing(t *testing.T) {
	tr := NewMemTracker(MemTrackerConfig{
		CgroupRoot: filepath.Join(t.TempDir(), "absent"),
		Sample:     fakeSampler(Sample{RuntimeBytes: 0, Goroutines: 0}),
	})
	tr.Observe()

	snap := tr.Snapshot()
	require.False(t, snap.HasRuntime)
	require.False(t, snap.HasCgroupCurrent)
	require.False(t, snap.HasCgroupMax)
}

func TestMemTracker_SnapshotNilReceiverIsSafe(t *testing.T) {
	var tr *MemTracker
	var snap MemSnapshot
	require.NotPanics(t, func() { snap = tr.Snapshot() })
	require.Equal(t, MemSnapshot{}, snap)
}

func TestMemTracker_StartTakesAnImmediateSample(t *testing.T) {
	// A scan shorter than one tick must still report something.
	tr := NewMemTracker(MemTrackerConfig{
		CgroupRoot: filepath.Join(t.TempDir(), "absent"),
		Sample:     fakeSampler(Sample{RuntimeBytes: 777, Goroutines: 7}),
	})
	tr.Start(time.Hour) // no tick will ever fire
	defer tr.Stop()

	require.Eventually(t, func() bool {
		tr.mu.Lock()
		defer tr.mu.Unlock()
		return tr.peakRuntime == 777
	}, 2*time.Second, 5*time.Millisecond)
}

func TestRecord_OmitsByteValuesThatCannotBeRepresented(t *testing.T) {
	// A uint64 at or above 1<<63 would wrap negative in the int64 cast used by
	// the metric API. Omitting is deliberate: clamping would emit a fabricated
	// ceiling that reads downstream as a real measurement.
	dir := t.TempDir()
	writeCgroupFile(t, dir, "memory.current", "1024\n")
	writeCgroupFile(t, dir, "memory.max", "18446744073709551615\n") // 2^64-1

	tr := NewMemTracker(MemTrackerConfig{
		CgroupRoot: dir,
		Sample:     fakeSampler(Sample{RuntimeBytes: 1 << 63, Goroutines: 3}),
	})
	tr.Observe()

	c := New()
	tr.Record(c)
	m := metricByName(t, c)

	require.NotContains(t, m, MetricMemCgroupMax)
	require.NotContains(t, m, MetricMemRuntimePeak)
	// Representable values alongside them are unaffected.
	require.Equal(t, int64(1024), m[MetricMemCgroupCurrent].GetIntValue())
	require.Equal(t, int64(3), m[MetricMemGoroutinesPeak].GetIntValue())
}

func TestRecord_KeepsLargestRepresentableByteValue(t *testing.T) {
	// The boundary itself must still be reported — the guard rejects only what
	// genuinely cannot round-trip.
	dir := t.TempDir()
	writeCgroupFile(t, dir, "memory.current", "9223372036854775807\n") // 2^63-1

	tr := NewMemTracker(MemTrackerConfig{
		CgroupRoot: dir,
		Sample:     fakeSampler(Sample{RuntimeBytes: 100, Goroutines: 1}),
	})
	tr.Observe()

	c := New()
	tr.Record(c)
	require.Equal(t, int64(math.MaxInt64), metricByName(t, c)[MetricMemCgroupCurrent].GetIntValue())
}

func TestMemTracker_StopWaitsForSamplerToExit(t *testing.T) {
	var samples atomic.Int64
	tr := NewMemTracker(MemTrackerConfig{
		CgroupRoot: filepath.Join(t.TempDir(), "absent"),
		Sample: func() Sample {
			samples.Add(1)
			return Sample{RuntimeBytes: 100, Goroutines: 1}
		},
	})

	tr.Start(time.Millisecond)
	require.Eventually(t, func() bool { return samples.Load() > 1 }, 2*time.Second, 5*time.Millisecond)

	tr.Stop()
	// Stop returned, so the sampler has exited: the count must not move again.
	after := samples.Load()
	time.Sleep(20 * time.Millisecond)
	require.Equal(t, after, samples.Load())
}
