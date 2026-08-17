// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scanstats

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

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
