// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package sqlite

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/v13/upload"
)

func TestUploadFailureTags_TransportError(t *testing.T) {
	tags := uploadFailureTags(
		"sess-1",
		"//assets.api.mondoo.app/spaces/s1/assets/a1",
		upload.FailureTimeout,
		0,
		upload.Result{BytesSent: 1_441_792, Duration: 30_857 * time.Millisecond},
		3_621_572,
	)

	assert.Equal(t, "upload_failure", tags["reportKind"])
	assert.Equal(t, "sess-1", tags["uploadSessionId"])
	assert.Equal(t, "timeout", tags["failureKind"])
	assert.Equal(t, "1441792", tags["bytesSent"])
	assert.Equal(t, "3621572", tags["bytesTotal"])
	assert.Equal(t, "30857", tags["durationMs"])
	assert.Equal(t, "//assets.api.mondoo.app/spaces/s1/assets/a1", tags["assetMrn"])
	// No response => no httpStatus tag at all, rather than "0".
	_, ok := tags["httpStatus"]
	require.False(t, ok)
}

func TestUploadFailureTags_HTTPStatusIncluded(t *testing.T) {
	tags := uploadFailureTags("sess-2", "//assets/a2", upload.FailureHTTPStatus, 408,
		upload.Result{BytesSent: 3_621_572, Duration: 30_900 * time.Millisecond}, 3_621_572)

	assert.Equal(t, "408", tags["httpStatus"])
	assert.Equal(t, "http_status", tags["failureKind"])
}

func TestUploadFailureTags_EmptySessionOmitted(t *testing.T) {
	// url_request_failed happens before a session exists.
	tags := uploadFailureTags("", "//assets/a3", upload.FailureURLRequest, 0, upload.Result{}, 0)

	_, ok := tags["uploadSessionId"]
	assert.False(t, ok, "no session id exists yet for url_request_failed")
	assert.Equal(t, "url_request_failed", tags["failureKind"])
}

func TestUploadOutcomeTags_FailureCarriesAttempts(t *testing.T) {
	out := scanUploadOutcome{
		sessionID:  "sess-9",
		attempts:   5,
		lastKind:   upload.FailureConnectionReset,
		lastRes:    upload.Result{BytesSent: 229376, Duration: 19403 * time.Millisecond},
		bytesTotal: 1855488,
	}

	tags := uploadOutcomeTags(uploadReportKindFailure, out, "//assets/a1")

	assert.Equal(t, "upload_failure", tags["reportKind"])
	assert.Equal(t, "5", tags["attempts"])
	assert.Equal(t, "connection_reset", tags["failureKind"])
	assert.Equal(t, "229376", tags["bytesSent"])
}

func TestUploadOutcomeTags_RecoveryUsesItsOwnReportKind(t *testing.T) {
	out := scanUploadOutcome{
		sessionID:  "sess-9",
		attempts:   3,
		lastKind:   upload.FailureConnectionReset,
		bytesTotal: 1855488,
	}

	tags := uploadOutcomeTags(uploadReportKindRecovered, out, "//assets/a1")

	assert.Equal(t, "upload_recovered", tags["reportKind"],
		"a recovery must not be reported as upload_failure — that means a lost scan")
	assert.Equal(t, "3", tags["attempts"])
	assert.Equal(t, "connection_reset", tags["failureKind"],
		"the recovery still says what it was fighting")
}
