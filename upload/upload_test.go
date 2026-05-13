// Copyright Mondoo, Inc. 2026
// SPDX-License-Identifier: BUSL-1.1

package upload

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeTempFile(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "payload.bin")
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600))
	return path
}

// TestUploadFileWithClient_UsesSuppliedClient asserts the supplied HTTP client
// is the one that issues the request. This is the seam the bucket-upload code
// uses to inject a proxy-aware client, so it must not be bypassed.
func TestUploadFileWithClient_UsesSuppliedClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, "application/octet-stream", r.Header.Get("Content-Type"))
		assert.Equal(t, "header-value", r.Header.Get("X-Test-Header"))
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.Equal(t, "hello-world", string(body))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	var transportHits atomic.Int32
	client := &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		transportHits.Add(1)
		return http.DefaultTransport.RoundTrip(req)
	})}

	resp, err := UploadFileWithClient(
		context.Background(),
		server.URL,
		map[string]string{"X-Test-Header": "header-value"},
		writeTempFile(t, "hello-world"),
		"application/octet-stream",
		client,
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, int32(1), transportHits.Load(), "expected the supplied client's transport to be used")
}

// TestUploadFileWithClient_RoutesThroughProxyTransport models the real fix:
// callers can inject a transport with a custom Proxy func (as the SQLite
// scan-db upload does to honor api_proxy from mondoo.yml). We verify the
// transport's Proxy hook is consulted for the outbound request.
func TestUploadFileWithClient_RoutesThroughProxyTransport(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	var proxyCalls atomic.Int32
	tr := &http.Transport{
		Proxy: func(*http.Request) (*url.URL, error) {
			proxyCalls.Add(1)
			// Returning nil means "do not use a proxy" — that's fine for the
			// test; we only care that the Proxy hook is invoked, which is what
			// the production fix relies on.
			return nil, nil
		},
	}
	client := &http.Client{Transport: tr}

	resp, err := UploadFileWithClient(
		context.Background(),
		server.URL,
		nil,
		writeTempFile(t, "payload"),
		"application/octet-stream",
		client,
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Greater(t, proxyCalls.Load(), int32(0), "transport.Proxy must be consulted for the upload request")
}

// TestUploadFile_DefaultClient sanity-checks the back-compat wrapper still
// PUTs the file when no explicit client is provided.
func TestUploadFile_DefaultClient(t *testing.T) {
	var received string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		received = string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	resp, err := UploadFile(
		context.Background(),
		server.URL,
		nil,
		writeTempFile(t, "default-client-payload"),
		"application/octet-stream",
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "default-client-payload", received)
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }
