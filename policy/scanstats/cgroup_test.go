// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scanstats

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

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
