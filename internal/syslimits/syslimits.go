// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Package syslimits keeps cnspec a good citizen inside resource-constrained,
// critical environments. Go does not, on its own, respect the CPU and memory
// limits imposed by a Linux cgroup: GOMAXPROCS defaults to the number of host
// CPUs (not the container's CPU quota) and the garbage collector only reacts to
// heap growth, not to the container's memory limit. On a small container packed
// onto a large host this leads to two failure modes that can destabilize the
// whole node:
//
//   - CPU: the Go scheduler spins up one OS thread per host CPU. When the cgroup
//     only grants a fraction of those CPUs the kernel aggressively throttles the
//     process, causing latency spikes for co-located, business-critical
//     workloads.
//   - Memory: without a memory target the GC lets the heap grow until the cgroup
//     limit is hit and the kernel OOM-kills the process — and, under memory
//     pressure, potentially neighbours too.
//
// This package detects the cgroup CPU quota and memory limit (cgroup v2 and v1)
// and applies matching GOMAXPROCS and GOMEMLIMIT values so cnspec stays within
// the budget it was given. It always defers to limits the operator set
// explicitly (the GOMAXPROCS / GOMEMLIMIT environment variables) and can be
// disabled entirely via MONDOO_DISABLE_AUTO_RESOURCE_LIMITS.
package syslimits

import (
	"math"
	"os"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

const (
	// EnvDisable turns off all automatic resource limiting when set to a
	// truthy value ("1", "true").
	EnvDisable = "MONDOO_DISABLE_AUTO_RESOURCE_LIMITS"
	// EnvMemoryHeadroom overrides the fraction of the cgroup memory limit that
	// GOMEMLIMIT is set to (default 0.9). Accepts a value in (0, 1].
	EnvMemoryHeadroom = "MONDOO_MEMORY_LIMIT_HEADROOM"

	// defaultMemoryHeadroom leaves 10% of the cgroup memory limit as a buffer
	// between the GC's soft target and the hard limit the kernel enforces. The
	// GC treats GOMEMLIMIT as a soft limit and works hard to keep total memory
	// below it, so the buffer absorbs non-heap allocations (stacks, provider
	// gRPC buffers) and the GC's own overshoot before the kernel OOM-kills us.
	defaultMemoryHeadroom = 0.9

	// unlimitedThreshold treats astronomically large cgroup limits as "no
	// limit". cgroup v1 reports an unset memory limit as a value close to
	// math.MaxInt64 (commonly 0x7FFFFFFFFFFFF000); anything at or above 2^62
	// bytes (4 EiB) is not a real budget.
	unlimitedThreshold uint64 = 1 << 62
)

// Limits describes the CPU and memory budget detected for the current process.
// A zero value in either field means "no limit detected".
type Limits struct {
	// CPUQuota is the number of CPUs the cgroup grants (e.g. 2.5), or 0 when
	// no CPU quota is configured.
	CPUQuota float64
	// CPUSource records where CPUQuota came from, for logging.
	CPUSource string
	// MemoryLimitBytes is the cgroup memory limit in bytes, or 0 when no
	// (finite) limit is configured.
	MemoryLimitBytes uint64
	// MemorySource records where MemoryLimitBytes came from, for logging.
	MemorySource string
}

// Result records the changes Apply made so callers can log or inspect them.
type Result struct {
	Limits Limits

	GOMAXPROCSChanged bool
	GOMAXPROCS        int

	MemoryLimitChanged bool
	MemoryLimitBytes   int64
}

// Apply detects cgroup CPU/memory limits and constrains the Go runtime to match
// them by setting GOMAXPROCS and GOMEMLIMIT. It is safe to call once at startup
// and is a no-op when:
//   - MONDOO_DISABLE_AUTO_RESOURCE_LIMITS is truthy,
//   - the platform is not Linux, or
//   - no cgroup limits are detected.
//
// Operator-provided GOMAXPROCS / GOMEMLIMIT settings always win; Apply never
// raises a limit above what the host or operator allows.
func Apply() Result {
	var res Result

	if isTruthy(os.Getenv(EnvDisable)) {
		log.Debug().Str("env", EnvDisable).Msg("automatic resource limiting disabled")
		return res
	}

	limits := Detect()
	res.Limits = limits

	applyCPU(limits, &res)
	applyMemory(limits, &res)

	return res
}

func applyCPU(limits Limits, res *Result) {
	if limits.CPUQuota <= 0 {
		return
	}
	// Respect an explicit operator override.
	if _, ok := os.LookupEnv("GOMAXPROCS"); ok {
		log.Debug().Msg("GOMAXPROCS set by operator, not overriding")
		return
	}

	procs := procsForQuota(limits.CPUQuota, runtime.NumCPU())
	current := runtime.GOMAXPROCS(0)
	if procs >= current {
		// Never raise GOMAXPROCS above the host default — we only ever want to
		// shrink our footprint to fit the cgroup.
		return
	}

	runtime.GOMAXPROCS(procs)
	res.GOMAXPROCSChanged = true
	res.GOMAXPROCS = procs
	log.Info().
		Int("gomaxprocs", procs).
		Int("previous", current).
		Float64("cpu_quota", limits.CPUQuota).
		Str("source", limits.CPUSource).
		Msg("limiting CPU usage to cgroup quota")
}

func applyMemory(limits Limits, res *Result) {
	if limits.MemoryLimitBytes == 0 {
		return
	}
	// Respect an explicit operator override. GOMEMLIMIT set via the environment
	// is reflected in the runtime's current soft limit, which otherwise defaults
	// to math.MaxInt64.
	if _, ok := os.LookupEnv("GOMEMLIMIT"); ok {
		log.Debug().Msg("GOMEMLIMIT set by operator, not overriding")
		return
	}

	headroom := memoryHeadroom()
	target := memLimitWithHeadroom(limits.MemoryLimitBytes, headroom)
	if target <= 0 {
		return
	}

	current := debug.SetMemoryLimit(-1)
	if current != math.MaxInt64 && current <= target {
		// A tighter limit is already in effect (operator or a previous call).
		return
	}

	debug.SetMemoryLimit(target)
	res.MemoryLimitChanged = true
	res.MemoryLimitBytes = target
	log.Info().
		Int64("gomemlimit_bytes", target).
		Uint64("cgroup_limit_bytes", limits.MemoryLimitBytes).
		Float64("headroom", headroom).
		Str("source", limits.MemorySource).
		Msg("limiting memory usage to cgroup limit")
}

// Detect reads the cgroup CPU quota and memory limit for the current process.
// It returns a zero-valued Limits on non-Linux platforms or when nothing is
// configured. Detection never fails — unreadable or malformed cgroup files are
// treated as "no limit".
func Detect() Limits {
	var l Limits
	if runtime.GOOS != "linux" {
		return l
	}

	if quota, source, ok := detectCPU(); ok {
		l.CPUQuota = quota
		l.CPUSource = source
	}
	if bytes, source, ok := detectMemory(); ok {
		l.MemoryLimitBytes = bytes
		l.MemorySource = source
	}
	return l
}

// procsForQuota converts a fractional CPU quota into a GOMAXPROCS value. It
// floors the quota (so we never claim more CPU than granted), enforces a
// minimum of 1, and never exceeds the number of host CPUs.
func procsForQuota(quota float64, numCPU int) int {
	procs := int(math.Floor(quota))
	if procs < 1 {
		procs = 1
	}
	if numCPU > 0 && procs > numCPU {
		procs = numCPU
	}
	return procs
}

// memLimitWithHeadroom applies the headroom fraction to a cgroup memory limit,
// returning the soft GOMEMLIMIT target in bytes. It returns 0 for an unlimited
// or nonsensical input.
func memLimitWithHeadroom(bytes uint64, headroom float64) int64 {
	if bytes == 0 || bytes >= unlimitedThreshold {
		return 0
	}
	if headroom <= 0 || headroom > 1 {
		headroom = defaultMemoryHeadroom
	}
	target := float64(bytes) * headroom
	// Guard against overflow when casting back to int64.
	if target >= float64(math.MaxInt64) {
		return 0
	}
	return int64(target)
}

func memoryHeadroom() float64 {
	if v := os.Getenv(EnvMemoryHeadroom); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 && f <= 1 {
			return f
		}
		log.Warn().Str("env", EnvMemoryHeadroom).Str("value", v).
			Msg("invalid memory headroom, using default")
	}
	return defaultMemoryHeadroom
}

func isTruthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
