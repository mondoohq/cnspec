// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package upload

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"syscall"
)

// FailureKind is a closed taxonomy describing why an upload attempt failed.
// It is sent to the platform as the "failureKind" tag on a health.SendError
// report, so the values are part of the wire contract: add to them freely, but
// do not rename or repurpose an existing one.
type FailureKind string

const (
	FailureTimeout         FailureKind = "timeout"
	FailureConnectionReset FailureKind = "connection_reset"
	FailureDNS             FailureKind = "dns"
	FailureTLS             FailureKind = "tls"
	FailureContextCanceled FailureKind = "context_canceled"
	FailureHTTPStatus      FailureKind = "http_status"
	FailureReportRPC       FailureKind = "report_rpc_failed"
	FailureURLRequest      FailureKind = "url_request_failed"
	// FailureOther is the catch-all. An unrecognised error is still reported —
	// with its message intact — rather than dropped.
	FailureOther FailureKind = "other"
)

// ClassifyFailure maps a transport error to a FailureKind. It inspects the
// error tree with errors.Is/As rather than matching strings, so it survives
// message changes in the standard library.
//
// Order matters: context.Canceled is checked before the net.Error timeout
// branch because a canceled request often surfaces as both, and the DNS/TLS
// cases are checked before it because those errors are also net.Errors.
func ClassifyFailure(err error) FailureKind {
	if err == nil {
		return FailureOther
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return FailureTimeout
	}
	if errors.Is(err, context.Canceled) {
		return FailureContextCanceled
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return FailureDNS
	}

	var certErr *tls.CertificateVerificationError
	if errors.As(err, &certErr) {
		return FailureTLS
	}
	var recordErr tls.RecordHeaderError
	if errors.As(err, &recordErr) {
		return FailureTLS
	}

	if errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.ECONNREFUSED) ||
		errors.Is(err, syscall.EPIPE) {
		return FailureConnectionReset
	}

	// Checked after the specific cases above: a DNS or TLS failure is also a
	// net.Error, and a timeout is the least specific useful answer.
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return FailureTimeout
	}

	return FailureOther
}
