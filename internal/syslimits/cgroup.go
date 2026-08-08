// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package syslimits

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	cgroupV2Mount        = "/sys/fs/cgroup"
	cgroupV1CPUMount     = "/sys/fs/cgroup/cpu"
	cgroupV1CPUAcctMount = "/sys/fs/cgroup/cpu,cpuacct"
	cgroupV1MemoryMount  = "/sys/fs/cgroup/memory"
	procSelfCgroup       = "/proc/self/cgroup"
)

// isCgroupV2 reports whether the host uses the cgroup v2 unified hierarchy.
// The presence of cgroup.controllers at the mount root is the canonical signal.
func isCgroupV2() bool {
	_, err := os.Stat(filepath.Join(cgroupV2Mount, "cgroup.controllers"))
	return err == nil
}

func detectCPU() (float64, string, bool) {
	if isCgroupV2() {
		if q, ok := detectCPUV2(); ok {
			return q, "cgroupv2 cpu.max", true
		}
		return 0, "", false
	}
	if q, ok := detectCPUV1(); ok {
		return q, "cgroupv1 cpu.cfs_quota_us", true
	}
	return 0, "", false
}

func detectMemory() (uint64, string, bool) {
	if isCgroupV2() {
		if b, ok := detectMemoryV2(); ok {
			return b, "cgroupv2 memory.max", true
		}
		return 0, "", false
	}
	if b, ok := detectMemoryV1(); ok {
		return b, "cgroupv1 memory.limit_in_bytes", true
	}
	return 0, "", false
}

// candidateDirs returns the cgroup directories to inspect, from the process's
// own (leaf) cgroup up to the mount root. A limit may be set on the leaf or on
// any ancestor; the effective limit is the tightest one along the path, so
// callers scan every candidate and keep the minimum.
func candidateDirs(mount, relPath string) []string {
	relPath = strings.TrimSpace(relPath)
	if relPath == "" {
		relPath = "/"
	}
	var dirs []string
	cur := relPath
	for {
		dirs = append(dirs, filepath.Join(mount, cur))
		if cur == "/" || cur == "." || cur == "" {
			break
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		cur = parent
	}
	return dirs
}

// ---- cgroup v2 ----

// cgroupV2RelPath reads the process's cgroup v2 path from /proc/self/cgroup.
// The v2 entry has the form "0::/some/path". Falls back to the root.
func cgroupV2RelPath() string {
	data, err := os.ReadFile(procSelfCgroup)
	if err != nil {
		return "/"
	}
	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.SplitN(line, ":", 3)
		if len(parts) == 3 && parts[0] == "0" && parts[1] == "" {
			return parts[2]
		}
	}
	return "/"
}

func detectCPUV2() (float64, bool) {
	var best float64
	found := false
	for _, dir := range candidateDirs(cgroupV2Mount, cgroupV2RelPath()) {
		content, err := os.ReadFile(filepath.Join(dir, "cpu.max"))
		if err != nil {
			continue
		}
		if q, ok := parseCPUMaxV2(string(content)); ok {
			if !found || q < best {
				best = q
				found = true
			}
		}
	}
	return best, found
}

// parseCPUMaxV2 parses a cgroup v2 cpu.max value ("<quota> <period>", or
// "max <period>" for unlimited) into the number of CPUs it grants.
func parseCPUMaxV2(content string) (float64, bool) {
	fields := strings.Fields(content)
	if len(fields) < 2 {
		return 0, false
	}
	if fields[0] == "max" {
		return 0, false
	}
	quota, err := strconv.ParseFloat(fields[0], 64)
	if err != nil || quota <= 0 {
		return 0, false
	}
	period, err := strconv.ParseFloat(fields[1], 64)
	if err != nil || period <= 0 {
		return 0, false
	}
	return quota / period, true
}

func detectMemoryV2() (uint64, bool) {
	var best uint64
	found := false
	for _, dir := range candidateDirs(cgroupV2Mount, cgroupV2RelPath()) {
		content, err := os.ReadFile(filepath.Join(dir, "memory.max"))
		if err != nil {
			continue
		}
		if b, ok := parseMemoryLimit(string(content)); ok {
			if !found || b < best {
				best = b
				found = true
			}
		}
	}
	return best, found
}

// ---- cgroup v1 ----

// cgroupV1RelPath reads the process's cgroup path for a given v1 controller
// from /proc/self/cgroup. Entries have the form "N:controller-list:/path".
func cgroupV1RelPath(controller string) string {
	data, err := os.ReadFile(procSelfCgroup)
	if err != nil {
		return "/"
	}
	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.SplitN(line, ":", 3)
		if len(parts) != 3 {
			continue
		}
		for _, c := range strings.Split(parts[1], ",") {
			if c == controller {
				return parts[2]
			}
		}
	}
	return "/"
}

func detectCPUV1() (float64, bool) {
	mount := cgroupV1CPUMount
	if _, err := os.Stat(mount); err != nil {
		mount = cgroupV1CPUAcctMount
	}
	relPath := cgroupV1RelPath("cpu")
	var best float64
	found := false
	for _, dir := range candidateDirs(mount, relPath) {
		quotaRaw, err1 := os.ReadFile(filepath.Join(dir, "cpu.cfs_quota_us"))
		periodRaw, err2 := os.ReadFile(filepath.Join(dir, "cpu.cfs_period_us"))
		if err1 != nil || err2 != nil {
			continue
		}
		if q, ok := parseCPUV1(string(quotaRaw), string(periodRaw)); ok {
			if !found || q < best {
				best = q
				found = true
			}
		}
	}
	return best, found
}

// parseCPUV1 converts cgroup v1 cpu.cfs_quota_us / cpu.cfs_period_us into the
// number of CPUs granted. A quota of -1 (or non-positive) means unlimited.
func parseCPUV1(quotaStr, periodStr string) (float64, bool) {
	quota, err := strconv.ParseInt(strings.TrimSpace(quotaStr), 10, 64)
	if err != nil || quota <= 0 {
		return 0, false
	}
	period, err := strconv.ParseInt(strings.TrimSpace(periodStr), 10, 64)
	if err != nil || period <= 0 {
		return 0, false
	}
	return float64(quota) / float64(period), true
}

func detectMemoryV1() (uint64, bool) {
	relPath := cgroupV1RelPath("memory")
	var best uint64
	found := false
	for _, dir := range candidateDirs(cgroupV1MemoryMount, relPath) {
		content, err := os.ReadFile(filepath.Join(dir, "memory.limit_in_bytes"))
		if err != nil {
			continue
		}
		if b, ok := parseMemoryLimit(string(content)); ok {
			if !found || b < best {
				best = b
				found = true
			}
		}
	}
	return best, found
}

// parseMemoryLimit parses a cgroup memory limit value. It handles the cgroup v2
// "max" sentinel and the cgroup v1 near-MaxInt64 "unlimited" value, returning
// ok=false for both.
func parseMemoryLimit(content string) (uint64, bool) {
	s := strings.TrimSpace(content)
	if s == "" || s == "max" {
		return 0, false
	}
	b, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, false
	}
	if b == 0 || b >= unlimitedThreshold {
		return 0, false
	}
	return b, true
}
