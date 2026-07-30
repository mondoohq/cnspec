// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build !windows

package upload

// classifyPlatform is the no-op counterpart to the Windows implementation.
// Everywhere except Windows the POSIX errno checks in ClassifyFailure already
// match, so there is nothing platform-specific left to recognise.
func classifyPlatform(error) (FailureKind, bool) {
	return "", false
}
