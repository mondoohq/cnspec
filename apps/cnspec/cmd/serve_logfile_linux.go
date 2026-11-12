// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build linux

package cmd

import (
	"github.com/cockroachdb/errors"
	"golang.org/x/sys/unix"
)

// redirectFD points file descriptor fd at the log file at path, so everything
// the process writes to fd, through any writer, lands in the file. Dup3 is
// used because linux/arm64 has no dup2 system call.
func redirectFD(path string, fd int) error {
	f, err := openLogFile(path)
	if err != nil {
		return err
	}
	// fd keeps its own reference to the open file description.
	defer f.Close()

	if err := unix.Dup3(int(f.Fd()), fd, 0); err != nil {
		return errors.Wrap(err, "could not redirect output to log file")
	}
	return nil
}
