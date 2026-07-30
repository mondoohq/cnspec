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
	FailureTimeout FailureKind = "timeout"
	// FailureConnectionRefused means the peer never accepted the connection, so
	// nothing was established and nothing was sent. Deliberately distinct from
	// FailureConnectionReset: refused points at "cannot reach this destination
	// at all" (egress policy, firewall, DNS pointing somewhere dead), reset at
	// "the connection existed and was dropped". For a client that can reach the
	// API host but not the ingest host, this is the kind that says so.
	FailureConnectionRefused FailureKind = "connection_refused"
	// FailureConnectionReset is an established connection dropped by the peer or
	// the network (ECONNRESET).
	FailureConnectionReset FailureKind = "connection_reset"
	// FailureBrokenPipe is a local write to a connection the peer had already
	// closed (EPIPE) — typically mid-body, after some bytes went out.
	FailureBrokenPipe FailureKind = "broken_pipe"
	// FailureConnectionAbortedLocally is Windows WSAECONNABORTED: "an
	// established connection was aborted by the software in your host machine".
	// Deliberately NOT folded into FailureConnectionReset — the peer did not
	// drop this connection, something on the client host did, which in practice
	// means antivirus, endpoint security, or a local proxy agent. "Our own
	// machine killed it" and "the far end dropped it" lead to completely
	// different investigations, so they get different kinds.
	FailureConnectionAbortedLocally FailureKind = "connection_aborted_locally"
	FailureDNS                      FailureKind = "dns"
	FailureTLS                      FailureKind = "tls"
	FailureContextCanceled          FailureKind = "context_canceled"
	FailureHTTPStatus               FailureKind = "http_status"
	FailureReportRPC                FailureKind = "report_rpc_failed"
	FailureURLRequest               FailureKind = "url_request_failed"
	// FailureOther is the catch-all. An unrecognised error is still reported —
	// with its message intact — rather than dropped. A nil error also maps here;
	// see ClassifyFailure.
	FailureOther FailureKind = "other"
)

// ClassifyFailure maps a transport error to a FailureKind. It inspects the
// error tree with errors.Is/As rather than matching strings, so it survives
// message changes in the standard library.
//
// Order matters: context.Canceled is checked before the net.Error timeout
// branch because a canceled request often surfaces as both, and the DNS/TLS
// cases are checked before it because those errors are also net.Errors.
//
// A nil error maps to FailureOther. Callers only classify on a failure, so nil
// here means a caller bug; FailureOther keeps that from inventing a wire value
// that downstream queries would have to know about, and the accompanying error
// message will be empty, which makes the mistake obvious in the data.
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

	if errors.Is(err, syscall.ECONNREFUSED) {
		return FailureConnectionRefused
	}
	if errors.Is(err, syscall.ECONNRESET) {
		return FailureConnectionReset
	}
	if errors.Is(err, syscall.EPIPE) {
		return FailureBrokenPipe
	}

	// Windows socket errors do NOT satisfy the checks above: Go defines
	// syscall.ECONNRESET and friends on Windows as synthetic
	// APPLICATION_ERROR+iota constants, while the kernel returns WSAE* values
	// (WSAECONNRESET = 10054). errors.Is against the POSIX names therefore never
	// matches there, and every Windows connection failure fell through to
	// FailureOther until this hook existed. See failurekind_windows.go.
	if kind, ok := classifyPlatform(err); ok {
		return kind
	}

	// Checked after the specific cases above: a DNS or TLS failure is also a
	// net.Error, and a timeout is the least specific useful answer.
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return FailureTimeout
	}

	return FailureOther
}
