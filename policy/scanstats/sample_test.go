// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scanstats

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultSample_ReturnsLiveValues(t *testing.T) {
	s := defaultSample()

	// A running Go process always has a non-zero footprint and at least
	// this test's own goroutine. The stub returned a zero Sample.
	require.Greater(t, s.RuntimeBytes, uint64(0))
	require.Greater(t, s.Goroutines, 0)
}

// metricByName indexes a ScanStatistics for assertion by metric name.

func TestDefaultSample_ReturnsLiveCPUValues(t *testing.T) {
	// The /cpu/classes/* counters are only recomputed at a GC boundary, so a
	// process that has not collected yet reads exactly 0 for every one of
	// them. Measured on go1.26: at startup total=0.000000000, and immediately
	// after a forced collection total=0.016285328.
	//
	// This package allocates almost nothing, so its tests routinely run
	// before the first GC and the TotalSeconds assertion below loses the race
	// -- which is how it failed CI on the v14.0.0-rc.5 release bump. Force
	// the collection so the counters are populated rather than asserting a
	// timing coincidence.
	//
	// Note this is a property of the *test*, not of CPUSample: zero
	// CPU-seconds is a legitimate reading for a short scan, which is exactly
	// why Valid does not key on value > 0.
	runtime.GC()

	s := defaultSample()

	require.True(t, s.CPU.Valid, "every /cpu/classes name should resolve on a supported Go runtime")
	require.Greater(t, s.CPU.TotalSeconds, 0.0, "a collection has happened, so the CPU counters are populated")
	require.GreaterOrEqual(t, s.CPU.TotalSeconds, s.CPU.IdleSeconds,
		"total is CPU available and includes idle, so it can never be the smaller of the two")
	require.Greater(t, s.CPU.GOMAXPROCS, 0)
}

func TestCPUSample_BusyExcludesIdle(t *testing.T) {
	// The trap this guards: /cpu/classes/total is CPU *available*, not CPU
	// consumed. Reporting it as usage would overstate every scan by the idle
	// share, which on a mostly-waiting scan is nearly all of it.
	c := CPUSample{TotalSeconds: 10, IdleSeconds: 7}
	require.Equal(t, 3.0, c.Busy())
}

func TestCPUSample_BusyNeverNegative(t *testing.T) {
	c := CPUSample{TotalSeconds: 4, IdleSeconds: 9}
	require.Equal(t, 0.0, c.Busy())
}
