// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build windows

package convert

// oNoFollow has no Windows equivalent: the syscall package defines no
// O_NOFOLLOW there and CreateFile has no flag for it. The exposure is also
// smaller -- creating a symlink on Windows needs SeCreateSymbolicLinkPrivilege,
// which an unprivileged account does not have by default -- so this is 0 rather
// than a refusal to write.
const oNoFollow = 0
