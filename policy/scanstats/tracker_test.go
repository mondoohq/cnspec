// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scanstats

import (
	"math"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/policy"
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

func TestResourceTracker_TracksHighWaterNotLatest(t *testing.T) {
	tr := NewResourceTracker(ResourceTrackerConfig{
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

func TestResourceTracker_InFlightCapturedAtPeakNotAtLastSample(t *testing.T) {
	inFlight := 0
	tr := NewResourceTracker(ResourceTrackerConfig{
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

func TestResourceTracker_NilReceiverIsSafe(t *testing.T) {
	var tr *ResourceTracker
	require.NotPanics(t, func() {
		tr.Observe()
		tr.SetInFlightFunc(func() int { return 1 })
	})
}

func TestResourceTracker_NoInFlightFuncDoesNotPanic(t *testing.T) {
	tr := NewResourceTracker(ResourceTrackerConfig{
		Sample: fakeSampler(Sample{RuntimeBytes: 100}),
	})
	require.NotPanics(t, func() { tr.Observe() })
	require.Equal(t, 0, tr.inFlightAtPeak)
}

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

func TestResourceTracker_RecordWritesPeaksAndRunConstants(t *testing.T) {
	dir := t.TempDir()
	writeCgroupFile(t, dir, "memory.current", "3402137600\n")
	writeCgroupFile(t, dir, "memory.peak", "3500000000\n")
	writeCgroupFile(t, dir, "memory.max", "4294967296\n")

	tr := NewResourceTracker(ResourceTrackerConfig{
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

func TestResourceTracker_RecordOmitsCgroupMetricsWhenUnavailable(t *testing.T) {
	tr := NewResourceTracker(ResourceTrackerConfig{
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

func TestResourceTracker_RecordOmitsRunIDWhenUnset(t *testing.T) {
	tr := NewResourceTracker(ResourceTrackerConfig{
		CgroupRoot: filepath.Join(t.TempDir(), "absent"),
		Sample:     fakeSampler(Sample{RuntimeBytes: 100}),
	})
	tr.Observe()

	c := New()
	tr.Record(c)
	require.NotContains(t, metricByName(t, c), MetricRunID)
}

func TestResourceTracker_RecordNilSafe(t *testing.T) {
	var tr *ResourceTracker
	c := New()
	require.NotPanics(t, func() { tr.Record(c) })
	require.Nil(t, c.ToProto())

	real := NewResourceTracker(ResourceTrackerConfig{Sample: fakeSampler(Sample{RuntimeBytes: 1})})
	require.NotPanics(t, func() { real.Record(nil) })
}

func TestResourceTracker_StartStopObservesAndIsIdempotent(t *testing.T) {
	tr := NewResourceTracker(ResourceTrackerConfig{
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

	var nilTr *ResourceTracker
	require.NotPanics(t, func() {
		nilTr.Start(time.Millisecond)
		nilTr.Stop()
	})
}

func TestResourceTracker_RecordOmitsRuntimeMetricsWhenNoSampleEverResolved(t *testing.T) {
	// A zero-byte sample simulates every runtime/metrics name failing to
	// resolve: defaultSample falls back to 0 rather than a real footprint.
	tr := NewResourceTracker(ResourceTrackerConfig{
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

func TestResourceTracker_RecordRefreshesAtFinishRatherThanUsingStaleSample(t *testing.T) {
	tr := NewResourceTracker(ResourceTrackerConfig{
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

func TestResourceTracker_SnapshotReportsPresentAndAbsentCgroupValues(t *testing.T) {
	dir := t.TempDir()
	writeCgroupFile(t, dir, "memory.current", "1048576\n")
	writeCgroupFile(t, dir, "memory.max", "2097152\n")

	tr := NewResourceTracker(ResourceTrackerConfig{
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

func TestResourceTracker_SnapshotReportsAbsentWhenCgroupRootMissing(t *testing.T) {
	tr := NewResourceTracker(ResourceTrackerConfig{
		CgroupRoot: filepath.Join(t.TempDir(), "absent"),
		Sample:     fakeSampler(Sample{RuntimeBytes: 0, Goroutines: 0}),
	})
	tr.Observe()

	snap := tr.Snapshot()
	require.False(t, snap.HasRuntime)
	require.False(t, snap.HasCgroupCurrent)
	require.False(t, snap.HasCgroupMax)
}

func TestResourceTracker_SnapshotNilReceiverIsSafe(t *testing.T) {
	var tr *ResourceTracker
	var snap ResourceSnapshot
	require.NotPanics(t, func() { snap = tr.Snapshot() })
	require.Equal(t, ResourceSnapshot{}, snap)
}

func TestResourceTracker_StartTakesAnImmediateSample(t *testing.T) {
	// A scan shorter than one tick must still report something.
	tr := NewResourceTracker(ResourceTrackerConfig{
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

	tr := NewResourceTracker(ResourceTrackerConfig{
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

	tr := NewResourceTracker(ResourceTrackerConfig{
		CgroupRoot: dir,
		Sample:     fakeSampler(Sample{RuntimeBytes: 100, Goroutines: 1}),
	})
	tr.Observe()

	c := New()
	tr.Record(c)
	require.Equal(t, int64(math.MaxInt64), metricByName(t, c)[MetricMemCgroupCurrent].GetIntValue())
}

func TestResourceTracker_StopWaitsForSamplerToExit(t *testing.T) {
	var samples atomic.Int64
	tr := NewResourceTracker(ResourceTrackerConfig{
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

// cpuAt builds a Sample carrying a resolved CPU reading. The memory side is
// held non-zero so the runtime metrics stay present and do not interfere with
// what the CPU assertions are measuring.
func cpuAt(total, idle, user, gc, scavenge float64) Sample {
	return Sample{
		RuntimeBytes: 1000,
		Goroutines:   5,
		CPU: CPUSample{
			Valid:           true,
			TotalSeconds:    total,
			IdleSeconds:     idle,
			UserSeconds:     user,
			GCSeconds:       gc,
			ScavengeSeconds: scavenge,
			GOMAXPROCS:      8,
		},
	}
}

func TestResourceTracker_RecordReportsCPUDeltaNotAbsoluteCounters(t *testing.T) {
	// The counters are cumulative since PROCESS start, so a scan that begins
	// in an already-warm process must not be charged for CPU burned before it
	// started. The baseline is taken at the first observation.
	tr := NewResourceTracker(ResourceTrackerConfig{
		CgroupRoot: filepath.Join(t.TempDir(), "absent"),
		Sample: fakeSampler(
			cpuAt(100, 60, 30, 8, 2),  // baseline: 40 busy already burned
			cpuAt(130, 75, 45, 12, 3), // now: 55 busy => this run used 15
		),
	})
	tr.Observe() // baseline
	tr.Observe() // advances to the second sample

	c := New()
	tr.Record(c) // Record observes again; sampler repeats the last value
	m := metricByName(t, c)

	require.InDelta(t, 15.0, m[MetricCPUBusySeconds].GetDoubleValue(), 0.0001)
	require.InDelta(t, 30.0, m[MetricCPUAvailableSeconds].GetDoubleValue(), 0.0001)
	require.InDelta(t, 15.0, m[MetricCPUUserSeconds].GetDoubleValue(), 0.0001)
	require.InDelta(t, 4.0, m[MetricCPUGCSeconds].GetDoubleValue(), 0.0001)
	require.InDelta(t, 1.0, m[MetricCPUScavengeSeconds].GetDoubleValue(), 0.0001)
	require.Equal(t, int64(8), m[MetricCPUGOMAXPROCS].GetIntValue())
}

func TestResourceTracker_RecordComputesCPUFractions(t *testing.T) {
	tr := NewResourceTracker(ResourceTrackerConfig{
		CgroupRoot: filepath.Join(t.TempDir(), "absent"),
		Sample: fakeSampler(
			cpuAt(0, 0, 0, 0, 0),
			cpuAt(100, 60, 30, 10, 0), // 40 busy of 100 available, 10 in GC
		),
	})
	tr.Observe()
	tr.Observe()

	c := New()
	tr.Record(c)
	m := metricByName(t, c)

	require.InDelta(t, 0.25, m[MetricCPUGCFraction].GetDoubleValue(), 0.0001)  // 10 / 40
	require.InDelta(t, 0.40, m[MetricCPUUtilization].GetDoubleValue(), 0.0001) // 40 / 100
}

func TestResourceTracker_RecordKeepsZeroCPUForVeryShortScan(t *testing.T) {
	// The distinction that separates CPU from memory: a live process always
	// has a non-zero memory footprint, so zero there means "did not resolve"
	// and is omitted. But a scan short enough legitimately burns ~0
	// CPU-seconds, so a zero CPU delta is a REAL measurement and must be
	// reported. Omitting it would silently drop every fast scan.
	tr := NewResourceTracker(ResourceTrackerConfig{
		CgroupRoot: filepath.Join(t.TempDir(), "absent"),
		Sample:     fakeSampler(cpuAt(100, 100, 0, 0, 0)),
	})
	tr.Observe()

	c := New()
	tr.Record(c)
	m := metricByName(t, c)

	require.Contains(t, m, MetricCPUBusySeconds)
	require.Equal(t, 0.0, m[MetricCPUBusySeconds].GetDoubleValue())
	// The ratios, by contrast, are omitted: their denominator is zero, and a
	// fabricated 0.0 would read as "no time in GC" rather than "cannot say".
	require.NotContains(t, m, MetricCPUGCFraction)
	require.NotContains(t, m, MetricCPUUtilization)
}

func TestResourceTracker_RecordOmitsCPUWhenNeverResolved(t *testing.T) {
	// CPU.Valid false stands for a runtime that renamed or dropped one of the
	// /cpu/classes names. Absent, never zero.
	tr := NewResourceTracker(ResourceTrackerConfig{
		CgroupRoot: filepath.Join(t.TempDir(), "absent"),
		Sample:     fakeSampler(Sample{RuntimeBytes: 1000, Goroutines: 5}),
	})
	tr.Observe()

	c := New()
	tr.Record(c)
	m := metricByName(t, c)

	for _, name := range []string{
		MetricCPUBusySeconds, MetricCPUAvailableSeconds, MetricCPUUserSeconds,
		MetricCPUGCSeconds, MetricCPUScavengeSeconds, MetricCPUGCFraction,
		MetricCPUUtilization, MetricCPUGOMAXPROCS,
	} {
		require.NotContains(t, m, name)
	}
	// The memory side is unaffected by CPU being unavailable.
	require.Contains(t, m, MetricMemRuntimePeak)
}

func TestResourceTracker_SnapshotReportsCPUBusySeconds(t *testing.T) {
	tr := NewResourceTracker(ResourceTrackerConfig{
		CgroupRoot: filepath.Join(t.TempDir(), "absent"),
		Sample: fakeSampler(
			cpuAt(10, 5, 3, 1, 0),
			cpuAt(30, 15, 10, 3, 0), // busy goes 5 -> 15, so the run used 10
		),
	})
	tr.Observe()
	tr.Observe()

	snap := tr.Snapshot()
	require.True(t, snap.HasCPU)
	require.InDelta(t, 10.0, snap.CPUBusySeconds, 0.0001)
}

func TestResourceTracker_SnapshotReportsCPUAbsentBeforeAnyObservation(t *testing.T) {
	tr := NewResourceTracker(ResourceTrackerConfig{
		CgroupRoot: filepath.Join(t.TempDir(), "absent"),
		Sample:     fakeSampler(cpuAt(10, 5, 3, 1, 0)),
	})

	snap := tr.Snapshot()
	require.False(t, snap.HasCPU)
	require.Equal(t, 0.0, snap.CPUBusySeconds)
}
