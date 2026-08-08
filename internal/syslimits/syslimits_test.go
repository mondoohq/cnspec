// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package syslimits

import (
	"math"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseCPUMaxV2(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    float64
		ok      bool
	}{
		{"two cpus", "200000 100000", 2.0, true},
		{"half a cpu", "50000 100000", 0.5, true},
		{"unlimited", "max 100000", 0, false},
		{"trailing newline", "150000 100000\n", 1.5, true},
		{"empty", "", 0, false},
		{"single field", "100000", 0, false},
		{"zero quota", "0 100000", 0, false},
		{"zero period", "100000 0", 0, false},
		{"garbage", "abc def", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseCPUMaxV2(tt.content)
			assert.Equal(t, tt.ok, ok)
			if tt.ok {
				assert.InDelta(t, tt.want, got, 0.0001)
			}
		})
	}
}

func TestParseCPUV1(t *testing.T) {
	tests := []struct {
		name   string
		quota  string
		period string
		want   float64
		ok     bool
	}{
		{"two cpus", "200000", "100000", 2.0, true},
		{"unlimited", "-1", "100000", 0, false},
		{"quarter cpu", "25000", "100000", 0.25, true},
		{"newlines", "300000\n", "100000\n", 3.0, true},
		{"zero period", "100000", "0", 0, false},
		{"garbage", "x", "y", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseCPUV1(tt.quota, tt.period)
			assert.Equal(t, tt.ok, ok)
			if tt.ok {
				assert.InDelta(t, tt.want, got, 0.0001)
			}
		})
	}
}

func TestParseMemoryLimit(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    uint64
		ok      bool
	}{
		{"512MiB", "536870912", 536870912, true},
		{"v2 max", "max", 0, false},
		{"trailing newline", "1073741824\n", 1073741824, true},
		{"v1 unlimited near maxint", "9223372036854771712", 0, false},
		{"zero", "0", 0, false},
		{"empty", "", 0, false},
		{"garbage", "lots", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseMemoryLimit(tt.content)
			assert.Equal(t, tt.ok, ok)
			if tt.ok {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestMemLimitWithHeadroom(t *testing.T) {
	gib := float64(uint64(1) << 30)
	tests := []struct {
		name     string
		bytes    uint64
		headroom float64
		want     int64
	}{
		{"90pct of 1GiB", 1 << 30, 0.9, int64(gib * 0.9)},
		{"full at 1.0", 1 << 30, 1.0, 1 << 30},
		{"unlimited zero", 0, 0.9, 0},
		{"unlimited huge", math.MaxUint64, 0.9, 0},
		{"invalid headroom falls back", 1 << 30, 0, int64(gib * defaultMemoryHeadroom)},
		{"headroom over 1 falls back", 1 << 30, 1.5, int64(gib * defaultMemoryHeadroom)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, memLimitWithHeadroom(tt.bytes, tt.headroom))
		})
	}
}

func TestCandidateDirs(t *testing.T) {
	got := candidateDirs("/sys/fs/cgroup", "/kubepods/pod123/container456")
	want := []string{
		filepath.FromSlash("/sys/fs/cgroup/kubepods/pod123/container456"),
		filepath.FromSlash("/sys/fs/cgroup/kubepods/pod123"),
		filepath.FromSlash("/sys/fs/cgroup/kubepods"),
		filepath.FromSlash("/sys/fs/cgroup"),
	}
	assert.Equal(t, want, got)

	// Root path yields just the mount.
	got = candidateDirs("/sys/fs/cgroup", "/")
	assert.Equal(t, []string{filepath.FromSlash("/sys/fs/cgroup")}, got)

	// Empty path is treated as root.
	got = candidateDirs("/sys/fs/cgroup", "")
	assert.Equal(t, []string{filepath.FromSlash("/sys/fs/cgroup")}, got)
}

func TestIsTruthy(t *testing.T) {
	for _, v := range []string{"1", "true", "TRUE", "yes", "on", " true "} {
		assert.True(t, isTruthy(v), v)
	}
	for _, v := range []string{"", "0", "false", "no", "off", "nope"} {
		assert.False(t, isTruthy(v), v)
	}
}
