// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scanstats

import (
	"os"
	"path/filepath"
	"runtime"
	"runtime/metrics"
	"strconv"
	"strings"
	"sync"
)

// Sample is one observation of this process's memory state.
type Sample struct {
	// RuntimeBytes is the Go runtime's memory footprint — the quantity the
	// runtime accounts against GOMEMLIMIT, and the one the OOM killer acts
	// on. Deliberately not MemStats.Alloc, which counts only live heap
	// objects and so understates the real footprint.
	RuntimeBytes uint64
	Goroutines   int
}

// SampleFunc returns a current Sample. Injected so the high-water logic is
// testable without a live runtime.
type SampleFunc func() Sample

// MemTrackerConfig configures a MemTracker. A zero Sample or CgroupRoot
// falls back to the real process defaults.
type MemTrackerConfig struct {
	RunID          string
	Parallelism    int
	MaxConnections int
	Sample         SampleFunc
	CgroupRoot     string
}

// MemTracker holds process-wide memory high-water marks for one scan run.
// Memory is per-process while scanstats Collectors are per-asset, so this is
// shared across every asset scanned by a run and makes no per-asset claim.
//
// Every exported method is safe on a nil receiver: telemetry must never
// panic a scan.
type MemTracker struct {
	mu             sync.Mutex
	peakRuntime    uint64
	peakGoroutines int
	// inFlightAtPeak is sampled when the peak is set, not when the scan
	// ends. A peak without the concurrency it occurred at cannot be
	// compared against any other peak.
	inFlightAtPeak int
	lastRuntime    uint64

	inFlight func() int

	cfg MemTrackerConfig
}

// NewMemTracker returns a tracker. Sampling does not start until Start.
func NewMemTracker(cfg MemTrackerConfig) *MemTracker {
	if cfg.Sample == nil {
		cfg.Sample = defaultSample
	}
	if cfg.CgroupRoot == "" {
		cfg.CgroupRoot = defaultCgroupRoot
	}
	return &MemTracker{cfg: cfg}
}

// SetInFlightFunc registers an accessor for the number of assets currently
// scanning. Registered after construction because the scan dispatcher that
// owns the semaphore is built after the tracker.
func (t *MemTracker) SetInFlightFunc(f func() int) {
	if t == nil {
		return
	}
	t.mu.Lock()
	t.inFlight = f
	t.mu.Unlock()
}

// Observe takes one sample and folds it into the high-water marks.
func (t *MemTracker) Observe() {
	if t == nil {
		return
	}
	s := t.cfg.Sample()

	t.mu.Lock()
	defer t.mu.Unlock()

	t.lastRuntime = s.RuntimeBytes
	if s.RuntimeBytes > t.peakRuntime {
		t.peakRuntime = s.RuntimeBytes
		if t.inFlight != nil {
			t.inFlightAtPeak = t.inFlight()
		}
	}
	if s.Goroutines > t.peakGoroutines {
		t.peakGoroutines = s.Goroutines
	}
}

// defaultCgroupRoot is the cgroup v2 unified hierarchy mount point.
const defaultCgroupRoot = "/sys/fs/cgroup"

// cgroupStats holds cgroup v2 memory readings. Each value carries a
// presence flag: a value that could not be read is absent, never zero.
// Reporting zero would be indistinguishable from a real measurement and
// would corrupt any aggregate computed across a mixed fleet.
type cgroupStats struct {
	current, peak, max          uint64
	hasCurrent, hasPeak, hasMax bool
}

// readCgroup reads cgroup v2 memory values from root, best-effort. Every
// file is optional: non-Linux hosts have no cgroup at all, cgroup v1 has a
// different layout, and memory.peak requires Linux 5.19 or later.
func readCgroup(root string) cgroupStats {
	var cg cgroupStats
	cg.current, cg.hasCurrent = readCgroupValue(root, "memory.current")
	cg.peak, cg.hasPeak = readCgroupValue(root, "memory.peak")
	cg.max, cg.hasMax = readCgroupValue(root, "memory.max")
	return cg
}

// readCgroupValue reads a single unsigned integer from a cgroup file. The
// literal "max" means no limit and is reported as absent rather than as a
// sentinel number.
func readCgroupValue(root, name string) (uint64, bool) {
	raw, err := os.ReadFile(filepath.Join(root, name))
	if err != nil {
		return 0, false
	}
	s := strings.TrimSpace(string(raw))
	if s == "" || s == "max" {
		return 0, false
	}
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

// runtimeFootprintMetrics are the runtime/metrics names whose difference is
// the Go runtime's memory footprint — the same quantity the runtime accounts
// against GOMEMLIMIT.
const (
	metricTotalBytes        = "/memory/classes/total:bytes"
	metricHeapReleasedBytes = "/memory/classes/heap/released:bytes"
)

// defaultSample reads the live process memory state.
//
// It uses runtime/metrics rather than runtime.ReadMemStats for two reasons:
// ReadMemStats stops the world, which would perturb the very scan being
// measured; and MemStats.Alloc counts only live heap objects, excluding
// stacks and memory the runtime has retained but not returned to the OS, so
// it systematically understates what the OOM killer acts on.
func defaultSample() Sample {
	// A fresh slice per call: metrics.Read writes into it, so a shared one
	// would need its own lock and this is called once a second.
	s := []metrics.Sample{
		{Name: metricTotalBytes},
		{Name: metricHeapReleasedBytes},
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
	}
}
