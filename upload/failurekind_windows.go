// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build windows

package upload

import (
	"errors"

	"golang.org/x/sys/windows"
)

// classifyPlatform maps Windows socket errors onto FailureKind.
//
// This exists because the POSIX names do not work here. Go defines
// syscall.ECONNRESET, ECONNREFUSED and friends on Windows as synthetic
// APPLICATION_ERROR+iota constants, whereas Winsock returns its own numbering
// (WSAECONNRESET = 10054, WSAECONNREFUSED = 10061, ...). An errors.Is check
// against the POSIX constant therefore never matches a real Windows socket
// failure, and every one of them was landing in FailureOther. Windows is a
// first-class scanning target, so this is not a rare path.
//
// golang.org/x/sys/windows is used rather than syscall because syscall only
// exposes four WSAE* constants (WSAEACCES, WSAENOPROTOOPT, WSAECONNABORTED,
// WSAECONNRESET); the rest, including WSAECONNREFUSED, live in x/sys.
func classifyPlatform(err error) (FailureKind, bool) {
	switch {
	case errors.Is(err, windows.WSAECONNABORTED):
		// "An established connection was aborted by the software in your host
		// machine" — the local side killed it, not the peer. Its own kind; see
		// FailureConnectionAbortedLocally.
		return FailureConnectionAbortedLocally, true
	case errors.Is(err, windows.WSAECONNRESET):
		return FailureConnectionReset, true
	case errors.Is(err, windows.WSAECONNREFUSED):
		return FailureConnectionRefused, true
	case errors.Is(err, windows.WSAETIMEDOUT):
		return FailureTimeout, true
	case errors.Is(err, windows.WSAEHOSTUNREACH), errors.Is(err, windows.WSAENETUNREACH):
		// No route to the destination at all. Grouped with refused because the
		// operational question is the same: this host cannot reach that
		// endpoint, as distinct from reaching it and being dropped.
		return FailureConnectionRefused, true
	case errors.Is(err, windows.WSAHOST_NOT_FOUND), errors.Is(err, windows.WSATRY_AGAIN):
		return FailureDNS, true
	}
	return "", false
}
