// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build !windows

package reportfile

import "syscall"

// oNoFollow makes an open fail on a symlink instead of following it.
const oNoFollow = syscall.O_NOFOLLOW
