// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package upload

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The shape net/http actually produces for a failed signed PUT: the signature
// lives in the query string, so it must not survive redaction.
const signedURL = "https://storage.googleapis.com/mondoo-ingest/ingest/scans/sess-1?" +
	"X-Goog-Algorithm=GOOG4-RSA-SHA256&X-Goog-Credential=svc%40proj.iam.gserviceaccount.com" +
	"&X-Goog-Signature=8a1f3c9d2b7e4f60a1b2c3d4e5f60718293a4b5c6d7e8f90"

func TestRedactError_DropsSignedURL(t *testing.T) {
	err := &url.Error{Op: "Put", URL: signedURL, Err: context.DeadlineExceeded}

	got := RedactError(err)

	assert.NotContains(t, got, "X-Goog-Signature")
	assert.NotContains(t, got, "8a1f3c9d2b7e4f60")
	assert.NotContains(t, got, "storage.googleapis.com")
	// The diagnostically useful part survives.
	assert.Contains(t, got, "Put")
	assert.Contains(t, got, "context deadline exceeded")
}

func TestRedactError_StripsQueryFromEmbeddedURL(t *testing.T) {
	// Not a *url.Error — a URL formatted into the message by another layer.
	err := fmt.Errorf("upload to %s failed", signedURL)

	got := RedactError(err)

	assert.NotContains(t, got, "X-Goog-Signature")
	assert.Contains(t, got, "<redacted>")
	// The host/path is retained in this branch; only the credential-bearing
	// query is removed.
	assert.Contains(t, got, "storage.googleapis.com")
}

func TestRedactError_Truncates(t *testing.T) {
	got := RedactError(errors.New(strings.Repeat("a", redactedErrorMax*2)))

	assert.Len(t, got, redactedErrorMax+len("…"))
	assert.True(t, strings.HasSuffix(got, "…"))
}

func TestRedactError_PassesThroughPlainError(t *testing.T) {
	assert.Equal(t, "connection reset by peer", RedactError(errors.New("connection reset by peer")))
}

func TestRedactError_Nil(t *testing.T) {
	require.Equal(t, "", RedactError(nil))
}
