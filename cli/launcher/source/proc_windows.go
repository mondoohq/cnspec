// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build windows

package source

import "os/exec"

// Windows has no process groups to signal the way Unix does, and killing a
// tree needs a job object. No picker source spawns a shell on Windows -- the
// file-backed ones read files and the rest run a vendor CLI directly -- so the
// default, killing the process itself, is what these callers need. WaitDelay
// in execRunner is the backstop that keeps a held pipe from blocking Wait.

func setProcessGroup(cmd *exec.Cmd) {}

func killTree(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	return cmd.Process.Kill()
}
