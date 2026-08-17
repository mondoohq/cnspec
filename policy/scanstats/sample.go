// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scanstats

import (
	"runtime"
	"runtime/metrics"
)

// Sample is one observation of this process's resource state.
type Sample struct {
	// RuntimeBytes is the Go runtime's memory footprint — the quantity the
	// runtime accounts against GOMEMLIMIT, and the one the OOM killer acts
	// on. Deliberately not MemStats.Alloc, which counts only live heap
	// objects and so understates the real footprint.
	RuntimeBytes uint64
	Goroutines   int
	CPU          CPUSample
}

// CPUSample is the CPU half of an observation. Every seconds field is
// cumulative since process start, so a scan's usage is the difference between
// two samples rather than a high-water mark — unlike the memory fields, which
// are instantaneous and tracked as peaks.
type CPUSample struct {
	// Valid reports whether the runtime/metrics CPU names resolved.
	//
	// This deliberately does NOT key on "value > 0" the way the memory
	// footprint's presence flag does. A live process always has a non-zero
	// footprint, so zero there means "did not resolve"; but a scan short
	// enough legitimately burns ~0 CPU-seconds, so zero here is a real
	// measurement. Keying on value would silently drop fast scans.
	Valid bool

	// TotalSeconds is CPU *available* to the process, not CPU consumed: it
	// includes IdleSeconds. CPU actually burned is TotalSeconds-IdleSeconds.
	TotalSeconds    float64
	IdleSeconds     float64
	UserSeconds     float64
	GCSeconds       float64
	ScavengeSeconds float64

	// GOMAXPROCS is a gauge, not a counter, so it is never differenced. On
	// Linux the runtime derives it from the cgroup CPU quota and updates it
	// as that quota changes, which makes it a usable stand-in for "how much
	// CPU was this process actually allowed" — unless the GOMAXPROCS
	// environment variable is set, which disables the automatic default.
	GOMAXPROCS int
}

// Busy returns CPU-seconds actually consumed: total available minus idle.
func (c CPUSample) Busy() float64 {
	if c.TotalSeconds > c.IdleSeconds {
		return c.TotalSeconds - c.IdleSeconds
	}
	return 0
}

// SampleFunc returns a current Sample. Injected so the tracker's logic is
// testable without a live runtime.
type SampleFunc func() Sample

// runtime/metrics names read on every sample. The memory pair's difference is
// the Go runtime's memory footprint (the quantity accounted against
// GOMEMLIMIT); the /cpu/classes/* counters are cumulative CPU-seconds.
const (
	metricTotalBytes        = "/memory/classes/total:bytes"
	metricHeapReleasedBytes = "/memory/classes/heap/released:bytes"

	metricCPUTotalSeconds    = "/cpu/classes/total:cpu-seconds"
	metricCPUIdleSeconds     = "/cpu/classes/idle:cpu-seconds"
	metricCPUUserSeconds     = "/cpu/classes/user:cpu-seconds"
	metricCPUGCSeconds       = "/cpu/classes/gc/total:cpu-seconds"
	metricCPUScavengeSeconds = "/cpu/classes/scavenge/total:cpu-seconds"
	metricGOMAXPROCS         = "/sched/gomaxprocs:threads"
)

// defaultSample reads the live process resource state.
//
// It uses runtime/metrics rather than runtime.ReadMemStats for two reasons:
// ReadMemStats stops the world, which would perturb the very scan being
// measured; and MemStats.Alloc counts only live heap objects, excluding
// stacks and memory the runtime has retained but not returned to the OS, so
// it systematically understates what the OOM killer acts on. Reading the CPU
// classes in the same call makes them nearly free.
func defaultSample() Sample {
	// A fresh slice per call: metrics.Read writes into it, so a shared one
	// would need its own lock and this is called once a second.
	s := []metrics.Sample{
		{Name: metricTotalBytes},
		{Name: metricHeapReleasedBytes},
		{Name: metricCPUTotalSeconds},
		{Name: metricCPUIdleSeconds},
		{Name: metricCPUUserSeconds},
		{Name: metricCPUGCSeconds},
		{Name: metricCPUScavengeSeconds},
		{Name: metricGOMAXPROCS},
	}
	metrics.Read(s)

	var total, released uint64
	if s[0].Value.Kind() == metrics.KindUint64 {
		total = s[0].Value.Uint64()
	}
	if s[1].Value.Kind() == metrics.KindUint64 {
		released = s[1].Value.Uint64()
	}

	var footprint uint64
	if total > released {
		footprint = total - released
	}

	return Sample{
		RuntimeBytes: footprint,
		Goroutines:   runtime.NumGoroutine(),
		CPU:          cpuFromMetrics(s[2:]),
	}
}

// cpuFromMetrics builds a CPUSample from the CPU slice of a metrics read.
// Valid is set only when every cpu-seconds name resolved to a float, so a
// renamed or dropped runtime metric yields an absent CPU reading rather than
// a plausible-looking zero.
func cpuFromMetrics(s []metrics.Sample) CPUSample {
	var cpu CPUSample

	seconds := make([]float64, 5)
	for i := range seconds {
		if s[i].Value.Kind() != metrics.KindFloat64 {
			return cpu
		}
		seconds[i] = s[i].Value.Float64()
	}

	cpu.Valid = true
	cpu.TotalSeconds = seconds[0]
	cpu.IdleSeconds = seconds[1]
	cpu.UserSeconds = seconds[2]
	cpu.GCSeconds = seconds[3]
	cpu.ScavengeSeconds = seconds[4]

	// GOMAXPROCS is reported separately from the cpu-seconds set: it is a
	// gauge of a different kind, and its absence should not invalidate an
	// otherwise good CPU reading.
	if s[5].Value.Kind() == metrics.KindUint64 {
		cpu.GOMAXPROCS = int(s[5].Value.Uint64())
	}

	return cpu
}
