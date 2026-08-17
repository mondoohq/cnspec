// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scanstats

import (
	"math"
	"sync"
	"time"
)

// ResourceTrackerConfig configures a ResourceTracker. A zero Sample or
// CgroupRoot falls back to the real process defaults.
type ResourceTrackerConfig struct {
	RunID          string
	Parallelism    int
	MaxConnections int
	Sample         SampleFunc
	CgroupRoot     string
}

// ResourceTracker holds resource usage for one scan run: memory high-water
// marks and cumulative CPU. A tracker is created per scan run (per
// distributeJob call), not per process: for the CLI these coincide, but a
// long-lived serve/queue process handling multiple jobs gets a fresh tracker
// per job. Usage is process-wide while scanstats Collectors are per-asset, so
// a single tracker is shared across every asset scanned by its run; RunID is
// what lets the resulting per-asset records be grouped back into the run they
// came from.
//
// Memory and CPU are tracked differently because they are different kinds of
// quantity. Memory is instantaneous, so the tracker keeps peaks. CPU counters
// are cumulative since process start, so the tracker keeps a baseline taken
// at the run's first observation and reports the difference.
//
// Every exported method is safe on a nil receiver: telemetry must never
// panic a scan.
type ResourceTracker struct {
	mu             sync.Mutex
	peakRuntime    uint64
	peakGoroutines int
	// inFlightAtPeak is sampled when the peak is set, not when the scan
	// ends. A peak without the concurrency it occurred at cannot be
	// compared against any other peak.
	inFlightAtPeak int
	lastRuntime    uint64
	// hasSample is true once at least one observation has measured a
	// non-zero runtime footprint. Guards the runtime-derived metrics in
	// Record: a tracker that never resolved a real sample must omit them
	// rather than report zero, which would be indistinguishable downstream
	// from a real measurement.
	hasSample bool

	// cpuBaseline is the first valid CPU reading of the run; lastCPU is the
	// most recent. Their difference is the run's CPU usage. hasCPUBaseline
	// tracks resolution rather than a non-zero value, because zero
	// CPU-seconds is a legitimate reading for a very short scan.
	cpuBaseline    CPUSample
	lastCPU        CPUSample
	hasCPUBaseline bool

	inFlight func() int

	cfg ResourceTrackerConfig

	stopOnce sync.Once
	stop     chan struct{}
	// wg tracks the sampling goroutine so Stop can wait for it to exit.
	// Without it Stop returns while a tick may still be in flight, which
	// matters for a long-lived process (cnspec serve) that creates one
	// tracker per job: samplers from consecutive jobs could otherwise overlap.
	wg sync.WaitGroup
}

// NewResourceTracker returns a tracker. Sampling does not start until Start.
func NewResourceTracker(cfg ResourceTrackerConfig) *ResourceTracker {
	if cfg.Sample == nil {
		cfg.Sample = defaultSample
	}
	if cfg.CgroupRoot == "" {
		cfg.CgroupRoot = defaultCgroupRoot
	}
	return &ResourceTracker{cfg: cfg, stop: make(chan struct{})}
}

// SetInFlightFunc registers an accessor for the number of assets currently
// scanning. Registered after construction because the scan dispatcher that
// owns the semaphore is built after the tracker.
//
// f is invoked while the tracker's mutex is held, so it must not block, do
// I/O, or call back into the tracker: the mutex is not reentrant, so a
// callback into Peaks or Snapshot would self-deadlock.
func (t *ResourceTracker) SetInFlightFunc(f func() int) {
	if t == nil {
		return
	}
	t.mu.Lock()
	t.inFlight = f
	t.mu.Unlock()
}

// Observe takes one sample, folds the memory readings into the high-water
// marks, and advances the CPU counters.
func (t *ResourceTracker) Observe() {
	if t == nil {
		return
	}
	s := t.cfg.Sample()

	t.mu.Lock()
	defer t.mu.Unlock()

	t.lastRuntime = s.RuntimeBytes
	if s.RuntimeBytes > 0 {
		t.hasSample = true
	}
	if s.RuntimeBytes > t.peakRuntime {
		t.peakRuntime = s.RuntimeBytes
		if t.inFlight != nil {
			t.inFlightAtPeak = t.inFlight()
		}
	}
	if s.Goroutines > t.peakGoroutines {
		t.peakGoroutines = s.Goroutines
	}

	if s.CPU.Valid {
		if !t.hasCPUBaseline {
			t.cpuBaseline = s.CPU
			t.hasCPUBaseline = true
		}
		t.lastCPU = s.CPU
	}
}

// Peaks returns the current high-water marks: runtime footprint bytes,
// goroutine count, and the in-flight asset count at the moment the footprint
// peak was set. Safe on a nil tracker, which reports zeroes.
func (t *ResourceTracker) Peaks() (runtimeBytes uint64, goroutines int, inFlightAtPeak int) {
	if t == nil {
		return 0, 0, 0
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.peakRuntime, t.peakGoroutines, t.inFlightAtPeak
}

// ResourceSnapshot is a point-in-time view for diagnostics: the latest
// observed runtime footprint and CPU usage so far, plus best-effort cgroup
// readings. Presence flags follow the same absent-never-zero rule as the
// recorded metrics.
type ResourceSnapshot struct {
	RuntimeBytes     uint64
	HasRuntime       bool
	CgroupCurrent    uint64
	HasCgroupCurrent bool
	CgroupMax        uint64
	HasCgroupMax     bool
	CPUBusySeconds   float64
	HasCPU           bool
}

// Snapshot returns the current diagnostic view. Safe on a nil tracker, which
// reports everything absent.
func (t *ResourceTracker) Snapshot() ResourceSnapshot {
	if t == nil {
		return ResourceSnapshot{}
	}

	t.mu.Lock()
	last, hasSample := t.lastRuntime, t.hasSample
	cpu, hasCPU := t.cpuUsageLocked()
	t.mu.Unlock()

	// File I/O outside the lock: reading cgroup files under the tracker
	// mutex would block the sampler goroutine.
	cg := readCgroup(t.cfg.CgroupRoot)

	return ResourceSnapshot{
		RuntimeBytes:     last,
		HasRuntime:       hasSample,
		CgroupCurrent:    cg.current,
		HasCgroupCurrent: cg.hasCurrent,
		CgroupMax:        cg.max,
		HasCgroupMax:     cg.hasMax,
		CPUBusySeconds:   cpu.busy,
		HasCPU:           hasCPU,
	}
}

// cpuUsage is the run's CPU consumption: the difference between the latest
// observation and the baseline taken at the run's first observation.
type cpuUsage struct {
	busy       float64
	available  float64
	user       float64
	gc         float64
	scavenge   float64
	gomaxprocs int
}

// cpuUsageLocked computes the run's CPU deltas. Callers must hold t.mu.
//
// Each delta is clamped at zero. The counters are monotonic within a process
// so a negative difference should be impossible, but reporting a negative
// CPU-seconds figure would be worse than reporting none.
func (t *ResourceTracker) cpuUsageLocked() (cpuUsage, bool) {
	if !t.hasCPUBaseline {
		return cpuUsage{}, false
	}

	delta := func(now, base float64) float64 {
		if now > base {
			return now - base
		}
		return 0
	}

	return cpuUsage{
		busy:       delta(t.lastCPU.Busy(), t.cpuBaseline.Busy()),
		available:  delta(t.lastCPU.TotalSeconds, t.cpuBaseline.TotalSeconds),
		user:       delta(t.lastCPU.UserSeconds, t.cpuBaseline.UserSeconds),
		gc:         delta(t.lastCPU.GCSeconds, t.cpuBaseline.GCSeconds),
		scavenge:   delta(t.lastCPU.ScavengeSeconds, t.cpuBaseline.ScavengeSeconds),
		gomaxprocs: t.lastCPU.GOMAXPROCS,
	}, true
}

// Start begins sampling every interval until Stop. It takes one sample
// immediately so a scan shorter than a single tick still reports a value —
// and so the CPU baseline is taken at the start of the run rather than a
// tick into it.
func (t *ResourceTracker) Start(interval time.Duration) {
	if t == nil {
		return
	}
	t.Observe()

	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-t.stop:
				return
			case <-ticker.C:
				t.Observe()
			}
		}
	}()
}

// Stop ends sampling and waits for the sampling goroutine to exit, so that
// once it returns no further observations can occur. Safe to call more than
// once, and on a nil tracker.
func (t *ResourceTracker) Stop() {
	if t == nil {
		return
	}
	t.stopOnce.Do(func() { close(t.stop) })
	t.wg.Wait()
}

// addBytes records a byte-count metric, omitting it when the value cannot be
// represented as an int64. A uint64 at or above 1<<63 would wrap negative in
// the cast, and a negative byte count downstream is worse than an absent one —
// the same rule the cgroup presence flags follow. Clamping was considered and
// rejected: a fabricated ceiling reads as a real measurement.
func addBytes(c *Collector, name string, v uint64) {
	if v > math.MaxInt64 {
		return
	}
	c.AddInt(name, "bytes", int64(v))
}

// Record writes the tracker's state into c. Called once per asset, so every
// asset's upload carries the process state as of that asset's completion.
//
// Values that could not be measured are omitted rather than recorded as
// zero: a zero cannot be told apart from a real measurement downstream.
func (t *ResourceTracker) Record(c *Collector) {
	if t == nil || c == nil {
		return
	}

	// Refresh before reading: without this, a sample can be up to a full
	// tick stale, and every asset that finishes inside one sampling window
	// would report an identical "at finish" value. Observe takes the lock
	// itself, so this must happen before the mutex block below.
	t.Observe()

	t.mu.Lock()
	peak, last, hasSample := t.peakRuntime, t.lastRuntime, t.hasSample
	goroutines, inFlight := t.peakGoroutines, t.inFlightAtPeak
	cpu, hasCPU := t.cpuUsageLocked()
	t.mu.Unlock()

	if t.cfg.RunID != "" {
		c.AddString(MetricRunID, t.cfg.RunID)
	}

	// Absent, not zero: a tracker that never resolved a real runtime sample
	// must omit these rather than report zero, which would be
	// indistinguishable downstream from a real measurement.
	if hasSample {
		addBytes(c, MetricMemRuntimePeak, peak)
		addBytes(c, MetricMemRuntimeAtFinish, last)
		c.AddInt(MetricMemGoroutinesPeak, "count", int64(goroutines))
	}

	if hasCPU {
		c.AddDouble(MetricCPUBusySeconds, "s", cpu.busy)
		c.AddDouble(MetricCPUAvailableSeconds, "s", cpu.available)
		c.AddDouble(MetricCPUUserSeconds, "s", cpu.user)
		c.AddDouble(MetricCPUGCSeconds, "s", cpu.gc)
		c.AddDouble(MetricCPUScavengeSeconds, "s", cpu.scavenge)

		// Ratios are omitted rather than reported as zero when their
		// denominator is zero: a fabricated 0.0 fraction reads downstream as
		// "no time in GC" when the truth is "not enough CPU to say".
		if cpu.busy > 0 {
			c.AddDouble(MetricCPUGCFraction, "", cpu.gc/cpu.busy)
		}
		if cpu.available > 0 {
			c.AddDouble(MetricCPUUtilization, "", cpu.busy/cpu.available)
		}
		if cpu.gomaxprocs > 0 {
			c.AddInt(MetricCPUGOMAXPROCS, "count", int64(cpu.gomaxprocs))
		}
	}

	c.AddInt(MetricConcurrencyInFlightAtPeak, "count", int64(inFlight))
	c.AddInt(MetricConcurrencyParallelism, "count", int64(t.cfg.Parallelism))
	c.AddInt(MetricConcurrencyMaxConnections, "count", int64(t.cfg.MaxConnections))

	cg := readCgroup(t.cfg.CgroupRoot)
	if cg.hasCurrent {
		addBytes(c, MetricMemCgroupCurrent, cg.current)
	}
	if cg.hasPeak {
		addBytes(c, MetricMemCgroupPeak, cg.peak)
	}
	if cg.hasMax {
		addBytes(c, MetricMemCgroupMax, cg.max)
	}
}
