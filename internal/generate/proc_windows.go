// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build windows

package generate

import "os/exec"

// Windows has no process group to signal the way Unix does; killing a tree
// needs a job object, which is more machinery than this call site justifies.
// Killing the agent process itself is what we can do portably, and the
// WaitDelay in the backend is the backstop that keeps a helper still holding
// the output pipe from blocking Wait forever.

func setProcessGroup(cmd *exec.Cmd) {}

func killTree(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	return cmd.Process.Kill()
}
