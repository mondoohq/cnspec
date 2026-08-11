// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package sqlite

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/v13/policy"
	"go.mondoo.com/ranger-rpc/codes"
	"go.mondoo.com/ranger-rpc/status"
)

// fakeResolver stands in for *policy.Services so the upload handshake can be
// driven without credentials. urlErrs is consumed one entry per call, so a test
// can make the first N GetUploadURL calls fail and the rest succeed.
type fakeResolver struct {
	uploadURL string

	urlCalls int
	urlErr   error

	completedCalls int
	completedErrs  []error
}

func (f *fakeResolver) GetUploadURL(ctx context.Context, in *policy.GetUploadURLReq) (*policy.GetUploadURLResp, error) {
	f.urlCalls++
	if f.urlErr != nil {
		return nil, f.urlErr
	}
	return &policy.GetUploadURLResp{
		UploadSessionId: "sess-1",
		UploadUrl:       &policy.UploadURL{Url: f.uploadURL},
	}, nil
}

func (f *fakeResolver) ReportUploadCompleted(ctx context.Context, in *policy.ReportUploadCompletedReq) (*policy.Empty, error) {
	f.completedCalls++
	if f.completedCalls <= len(f.completedErrs) {
		if err := f.completedErrs[f.completedCalls-1]; err != nil {
			return nil, err
		}
	}
	return &policy.Empty{}, nil
}

func testScanDataFile(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "scan.db")
	require.NoError(t, os.WriteFile(p, []byte("scan-database-contents"), 0o600))
	return p
}

// flakyPUT returns a server whose first failCount PUTs fail with status, then
// succeed. status 0 means "hijack the connection and drop it" — a transport
// failure rather than an HTTP error.
func flakyPUT(t *testing.T, failCount int, status int) *httptest.Server {
	t.Helper()
	var n int
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n++
		if n <= failCount {
			if status == 0 {
				hj, ok := w.(http.Hijacker)
				require.True(t, ok)
				conn, _, err := hj.Hijack()
				require.NoError(t, err)
				conn.Close()
				return
			}
			w.WriteHeader(status)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
}

func TestDoScanUpload_RetriesTransportErrorThenSucceeds(t *testing.T) {
	srv := flakyPUT(t, 1, 0)
	defer srv.Close()
	f := &fakeResolver{uploadURL: srv.URL}

	out, err := doScanUpload(context.Background(), f, testScanDataFile(t), "//assets/a1", nil)

	require.NoError(t, err)
	assert.Equal(t, 2, out.attempts, "one failed attempt then success")
	assert.NotEmpty(t, out.lastKind, "the failure that was recovered from must be recorded")
}

func TestDoScanUpload_RetriesRetryableHTTPStatus(t *testing.T) {
	srv := flakyPUT(t, 1, http.StatusServiceUnavailable)
	defer srv.Close()
	f := &fakeResolver{uploadURL: srv.URL}

	out, err := doScanUpload(context.Background(), f, testScanDataFile(t), "//assets/a1", nil)

	require.NoError(t, err)
	assert.Equal(t, 2, out.attempts)
	assert.Equal(t, http.StatusServiceUnavailable, out.lastStatus)
}

func TestDoScanUpload_FetchesFreshURLPerAttempt(t *testing.T) {
	// Signed URLs carry X-Goog-Expires=299, so a URL reused across a backoff
	// window would PUT against an expired URL and 403.
	srv := flakyPUT(t, 2, 0)
	defer srv.Close()
	f := &fakeResolver{uploadURL: srv.URL}

	out, err := doScanUpload(context.Background(), f, testScanDataFile(t), "//assets/a1", nil)

	require.NoError(t, err)
	assert.Equal(t, 3, out.attempts)
	assert.Equal(t, 3, f.urlCalls, "GetUploadURL must be called once per attempt, not once total")
}

func TestDoScanUpload_ExhaustsBudgetAndFails(t *testing.T) {
	srv := flakyPUT(t, 99, 0)
	defer srv.Close()
	f := &fakeResolver{uploadURL: srv.URL}

	out, err := doScanUpload(context.Background(), f, testScanDataFile(t), "//assets/a1", nil)

	require.Error(t, err)
	assert.Equal(t, 5, out.attempts, "the full retry budget is spent before giving up")
	// The exact kind is platform-dependent (upload.ClassifyFailure differs on
	// Windows), so assert only that one was classified.
	assert.NotEmpty(t, out.lastKind)
}

func TestDoScanUpload_PermanentRPCErrorFailsFast(t *testing.T) {
	srv := flakyPUT(t, 0, 0)
	defer srv.Close()
	// Ranger status, not grpc status: upstream.RetryableRPCError classifies with
	// go.mondoo.com/ranger-rpc/status, and a grpc error read through it comes
	// back as Unknown — which IS retryable. Production errors arrive from a
	// ranger client, so this is the shape that matters.
	f := &fakeResolver{
		uploadURL: srv.URL,
		urlErr:    status.Error(codes.Unauthenticated, "request permission unauthenticated"),
	}

	out, err := doScanUpload(context.Background(), f, testScanDataFile(t), "//assets/a1", nil)

	require.Error(t, err)
	assert.Equal(t, 1, out.attempts, "a permanent error must not burn the retry budget")
	assert.Equal(t, 1, f.urlCalls)
}

func TestDoScanUpload_RetriesReportUploadCompleted(t *testing.T) {
	srv := flakyPUT(t, 0, 0)
	defer srv.Close()
	f := &fakeResolver{
		uploadURL:     srv.URL,
		completedErrs: []error{status.Error(codes.Unavailable, "backend hiccup")},
	}

	out, err := doScanUpload(context.Background(), f, testScanDataFile(t), "//assets/a1", nil)

	require.NoError(t, err)
	assert.Equal(t, 2, f.completedCalls, "a transient completion RPC failure is retried")
	assert.Equal(t, 1, out.attempts, "the PUT itself never failed")
}

func TestDoScanUpload_CancelledContextReturnsPromptly(t *testing.T) {
	srv := flakyPUT(t, 99, 0)
	defer srv.Close()
	f := &fakeResolver{uploadURL: srv.URL}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	_, err := doScanUpload(ctx, f, testScanDataFile(t), "//assets/a1", nil)

	require.Error(t, err)
	assert.Less(t, time.Since(start), 5*time.Second, "must abort rather than sleep out the budget")
}

// trackedBody records whether the transport-facing contract was met: a body
// read to EOF and closed is the only state from which Go's transport will
// recycle a connection.
type trackedBody struct {
	io.Reader
	closed   bool
	readToIO bool
}

func (b *trackedBody) Read(p []byte) (int, error) {
	n, err := b.Reader.Read(p)
	if err == io.EOF {
		b.readToIO = true
	}
	return n, err
}

func (b *trackedBody) Close() error {
	b.closed = true
	return nil
}

func TestDrainAndClose_ReadsToEOFThenCloses(t *testing.T) {
	// Close alone is not enough: with no api_proxy set UploadFile runs on
	// http.DefaultTransport's process-wide pool, and a body left undrained
	// costs a pooled connection on every scan.
	body := &trackedBody{Reader: strings.NewReader("<?xml version='1.0'?><PostResponse/>")}

	drainAndClose(body)

	assert.True(t, body.readToIO, "body must be read to EOF so the connection can be recycled")
	assert.True(t, body.closed, "body must still be closed")
}

func TestDrainAndClose_ClosesEvenWhenReadFails(t *testing.T) {
	body := &trackedBody{Reader: iotest.ErrReader(errors.New("connection reset"))}

	drainAndClose(body)

	assert.True(t, body.closed, "a read error must not leak the body")
}
