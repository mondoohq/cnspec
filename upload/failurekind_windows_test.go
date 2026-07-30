// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build windows

package upload

import (
	"fmt"
	"net"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/sys/windows"
)

// TestClassifyFailure_WindowsSocketErrors covers the gap that put every Windows
// connection failure into FailureOther: Go's syscall.ECONNRESET on Windows is a
// synthetic APPLICATION_ERROR+iota value, not Winsock's 10054, so errors.Is
// against the POSIX names never matched here.
//
// The errors are shaped the way net/http delivers them — a *net.OpError
// wrapping the raw Errno — because that wrapping is what errors.Is has to
// traverse.
func TestClassifyFailure_WindowsSocketErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want FailureKind
	}{
		{
			// Seen in the field as: "An established connection was aborted by
			// the software in your host machine."
			name: "WSAECONNABORTED is local interference, not a peer reset",
			err:  &net.OpError{Op: "read", Err: windows.WSAECONNABORTED},
			want: FailureConnectionAbortedLocally,
		},
		{
			// Seen in the field as: "An existing connection was forcibly closed
			// by the remote host."
			name: "WSAECONNRESET",
			err:  &net.OpError{Op: "read", Err: windows.WSAECONNRESET},
			want: FailureConnectionReset,
		},
		{name: "WSAECONNREFUSED", err: &net.OpError{Op: "dial", Err: windows.WSAECONNREFUSED}, want: FailureConnectionRefused},
		{name: "WSAETIMEDOUT", err: &net.OpError{Op: "dial", Err: windows.WSAETIMEDOUT}, want: FailureTimeout},
		{name: "WSAEHOSTUNREACH", err: &net.OpError{Op: "dial", Err: windows.WSAEHOSTUNREACH}, want: FailureConnectionRefused},
		{name: "WSAENETUNREACH", err: &net.OpError{Op: "dial", Err: windows.WSAENETUNREACH}, want: FailureConnectionRefused},
		{name: "WSAHOST_NOT_FOUND", err: &net.OpError{Op: "dial", Err: windows.WSAHOST_NOT_FOUND}, want: FailureDNS},
		{name: "wrapped", err: fmt.Errorf("upload failed: %w", &net.OpError{Op: "write", Err: windows.WSAECONNRESET}), want: FailureConnectionReset},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, ClassifyFailure(tc.err))
		})
	}
}

// TestClassifyFailure_WindowsPOSIXNamesDoNotMatch is the regression guard for
// the root cause. If a future Go release ever makes syscall.ECONNRESET on
// Windows equal Winsock's value, this test fails and the platform hook can be
// simplified — but until then it documents WHY the hook has to exist.
func TestClassifyFailure_WindowsPOSIXNamesDoNotMatch(t *testing.T) {
	assert.NotEqual(t, syscall.Errno(windows.WSAECONNRESET), syscall.ECONNRESET,
		"syscall.ECONNRESET now equals WSAECONNRESET — the platform hook may be redundant")
}
