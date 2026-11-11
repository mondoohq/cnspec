// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build !linux

package cmd

import "github.com/cockroachdb/errors"

// redirectFD is Linux-only: the setting exists to keep the service out of the
// systemd journal and rsyslog. On Windows the service logs to the Event Log.
func redirectFD(path string, fd int) error {
	return errors.New(logFileKey + " is supported only on Linux")
}
