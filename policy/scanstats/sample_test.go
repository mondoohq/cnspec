// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scanstats

import (
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
	s := defaultSample()

	require.True(t, s.CPU.Valid, "every /cpu/classes name should resolve on a supported Go runtime")
	require.Greater(t, s.CPU.TotalSeconds, 0.0, "a running process has consumed some wall-clock CPU capacity")
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
