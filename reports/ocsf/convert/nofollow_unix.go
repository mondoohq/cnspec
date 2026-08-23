// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build !windows

package convert

import "syscall"

// oNoFollow makes an open fail on a symlink instead of following it, which is
// what stops a pre-placed link at one of the fixed class filenames redirecting a
// scan's output onto a file outside the output directory.
const oNoFollow = syscall.O_NOFOLLOW
