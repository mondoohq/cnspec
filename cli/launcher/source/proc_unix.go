// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build !windows

package source

import (
	"os/exec"
	"syscall"
)

// A picker's child is often a shell, and a shell that forks rather than execs
// leaves the real work running when only the shell is killed. `sh -c "sleep
// 30"` is the case that proved it: macOS execs the sleep, so killing the child
// ends it, while Linux forks, so the sleep outlived its context and kept the
// stdout pipe open -- and Wait, which waits for the pipe rather than for the
// process, blocked for the full thirty seconds.
//
// Putting the child in its own process group makes the whole tree killable by
// one signal, so cancelling stops the work rather than only stopping the wait.

func setProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// killTree signals the child's whole group. The negative pid is the group.
func killTree(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err != nil {
		// The group is gone if the child already exited, which is not a
		// failure to cancel. Fall back to the process itself in case the
		// group was never established.
		return cmd.Process.Kill()
	}
	return nil
}
