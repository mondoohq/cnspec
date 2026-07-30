// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package upload

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClassifyFailure(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want FailureKind
	}{
		{"deadline exceeded", context.DeadlineExceeded, FailureTimeout},
		{"wrapped deadline", fmt.Errorf("put failed: %w", context.DeadlineExceeded), FailureTimeout},
		{"canceled", context.Canceled, FailureContextCanceled},
		{"net timeout", &net.OpError{Op: "dial", Err: &timeoutErr{}}, FailureTimeout},
		{"connection reset", &net.OpError{Op: "write", Err: syscall.ECONNRESET}, FailureConnectionReset},
		// Distinct from reset on purpose: refused is the signature of a host
		// that cannot reach the ingest destination at all.
		{"connection refused", &net.OpError{Op: "dial", Err: syscall.ECONNREFUSED}, FailureConnectionRefused},
		{"broken pipe", &net.OpError{Op: "write", Err: syscall.EPIPE}, FailureBrokenPipe},
		{"dns", &net.DNSError{Err: "no such host", Name: "ingest.example.com"}, FailureDNS},
		{"tls", &tls.CertificateVerificationError{}, FailureTLS},
		{"unknown", errors.New("something odd"), FailureOther},
		{"nil", nil, FailureOther},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, ClassifyFailure(tc.err))
		})
	}
}

// timeoutErr is a net.Error that reports Timeout() == true.
type timeoutErr struct{}

func (*timeoutErr) Error() string   { return "i/o timeout" }
func (*timeoutErr) Timeout() bool   { return true }
func (*timeoutErr) Temporary() bool { return true }
