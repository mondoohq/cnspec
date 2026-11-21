// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build !windows

package generate

import (
	"os/exec"
	"syscall"
)

// A coding agent is a process tree, not a process: it spawns tool subprocesses
// and helpers, and those keep the stdout pipe open after the agent itself is
// signalled. os/exec's Wait waits for the *pipe*, not for the process, so
// killing only the direct child leaves Wait blocked on a grandchild — an agent
// that forked a helper ran for 601 seconds against a 180-second timeout, and
// the orphan outlived the run.
//
// Putting the agent in its own process group makes the whole tree killable with
// one signal, so a timeout stops the work instead of only stopping the wait.
// This mirrors cli/launcher/source/proc_unix.go, which was written for the same
// failure.

func setProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// killTree signals the agent's whole group. The negative pid is the group.
func killTree(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err != nil {
		// The group is gone if the agent already exited, which is not a failure
		// to cancel. Fall back to the process itself in case the group was
		// never established.
		return cmd.Process.Kill()
	}
	return nil
}
