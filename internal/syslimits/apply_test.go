// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package syslimits

import (
	"math"
	"os"
	"runtime/debug"
	"testing"

	"github.com/stretchr/testify/assert"
)

// clearGomemlimit removes GOMEMLIMIT for the duration of a test and restores it
// afterwards, so the "unconstrained" cases are deterministic regardless of the
// caller's environment.
func clearGomemlimit(t *testing.T) {
	t.Helper()
	prev, had := os.LookupEnv("GOMEMLIMIT")
	os.Unsetenv("GOMEMLIMIT")
	t.Cleanup(func() {
		if had {
			os.Setenv("GOMEMLIMIT", prev)
		} else {
			os.Unsetenv("GOMEMLIMIT")
		}
	})
}

func TestMemoryHeadroom(t *testing.T) {
	t.Run("valid override", func(t *testing.T) {
		t.Setenv(EnvMemoryHeadroom, "0.75")
		assert.Equal(t, 0.75, memoryHeadroom())
	})
	t.Run("full at 1.0", func(t *testing.T) {
		t.Setenv(EnvMemoryHeadroom, "1.0")
		assert.Equal(t, 1.0, memoryHeadroom())
	})
	for _, bad := range []string{"0", "-0.5", "1.5", "abc", ""} {
		t.Run("invalid falls back: "+bad, func(t *testing.T) {
			t.Setenv(EnvMemoryHeadroom, bad)
			assert.Equal(t, defaultMemoryHeadroom, memoryHeadroom())
		})
	}
}

func TestApplyMemory(t *testing.T) {
	const gib = int64(1) << 30

	// applyMemory mutates the process-global soft memory limit; save and restore
	// it so these tests don't leak into others.
	orig := debug.SetMemoryLimit(-1)
	t.Cleanup(func() { debug.SetMemoryLimit(orig) })

	t.Run("sets headroom target when unconstrained", func(t *testing.T) {
		clearGomemlimit(t)
		debug.SetMemoryLimit(math.MaxInt64) // simulate "no limit"

		var res Result
		applyMemory(Limits{MemoryLimitBytes: uint64(gib), MemorySource: "test"}, &res)

		want := memLimitWithHeadroom(uint64(gib), defaultMemoryHeadroom)
		assert.True(t, res.MemoryLimitChanged)
		assert.Equal(t, want, res.MemoryLimitBytes)
		assert.Equal(t, want, debug.SetMemoryLimit(-1))
	})

	t.Run("respects GOMEMLIMIT env override", func(t *testing.T) {
		t.Setenv("GOMEMLIMIT", "512MiB")
		debug.SetMemoryLimit(math.MaxInt64)

		var res Result
		applyMemory(Limits{MemoryLimitBytes: uint64(gib)}, &res)

		assert.False(t, res.MemoryLimitChanged)
		assert.Equal(t, int64(math.MaxInt64), debug.SetMemoryLimit(-1))
	})

	t.Run("never raises an existing tighter limit", func(t *testing.T) {
		clearGomemlimit(t)
		tighter := int64(100) << 20 // 100 MiB already in effect
		debug.SetMemoryLimit(tighter)

		var res Result
		applyMemory(Limits{MemoryLimitBytes: uint64(gib)}, &res) // 0.9 GiB > 100 MiB

		assert.False(t, res.MemoryLimitChanged)
		assert.Equal(t, tighter, debug.SetMemoryLimit(-1))
	})

	t.Run("no detected limit is a no-op", func(t *testing.T) {
		clearGomemlimit(t)
		debug.SetMemoryLimit(math.MaxInt64)

		var res Result
		applyMemory(Limits{}, &res)

		assert.False(t, res.MemoryLimitChanged)
		assert.Equal(t, int64(math.MaxInt64), debug.SetMemoryLimit(-1))
	})

	t.Run("limit below floor is skipped to avoid GC thrash", func(t *testing.T) {
		clearGomemlimit(t)
		debug.SetMemoryLimit(math.MaxInt64)

		var res Result
		applyMemory(Limits{MemoryLimitBytes: minMemoryLimitBytes - 1, MemorySource: "test"}, &res)

		assert.False(t, res.MemoryLimitChanged)
		assert.Equal(t, int64(math.MaxInt64), debug.SetMemoryLimit(-1))
	})

	t.Run("limit at floor is applied", func(t *testing.T) {
		clearGomemlimit(t)
		debug.SetMemoryLimit(math.MaxInt64)

		var res Result
		applyMemory(Limits{MemoryLimitBytes: minMemoryLimitBytes, MemorySource: "test"}, &res)

		assert.True(t, res.MemoryLimitChanged)
		assert.Equal(t, memLimitWithHeadroom(minMemoryLimitBytes, defaultMemoryHeadroom), debug.SetMemoryLimit(-1))
	})
}
