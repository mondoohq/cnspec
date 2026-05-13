// Copyright Mondoo, Inc. 2026
// SPDX-License-Identifier: BUSL-1.1

package upload

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeTempFile(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "payload.bin")
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600))
	return path
}

// unsetEnv removes an env var for the duration of the test, restoring whatever
// value (or unset state) was there before. t.Setenv is not a fit here: setting
// to "" leaves the var present-but-empty, which config.GetAPIProxy treats as
// "env is configured" and short-circuits ahead of the viper lookup.
func unsetEnv(t *testing.T, key string) {
	t.Helper()
	prev, had := os.LookupEnv(key)
	require.NoError(t, os.Unsetenv(key))
	t.Cleanup(func() {
		if had {
			_ = os.Setenv(key, prev)
		} else {
			_ = os.Unsetenv(key)
		}
	})
}

// TestUploadFile_PUTsContent is a smoke test for the round-trip: PUT method,
// content-type header, request body, and custom headers all reach the server.
func TestUploadFile_PUTsContent(t *testing.T) {
	var received string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, "application/octet-stream", r.Header.Get("Content-Type"))
		assert.Equal(t, "header-value", r.Header.Get("X-Test-Header"))
		body, _ := io.ReadAll(r.Body)
		received = string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	resp, err := UploadFile(
		context.Background(),
		server.URL,
		map[string]string{"X-Test-Header": "header-value"},
		writeTempFile(t, "hello-world"),
		"application/octet-stream",
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "hello-world", received)
}

// TestNewHTTPClient_HonorsAPIProxy is the regression test for the
// bucket-upload-ignores-api_proxy bug. When mondoo.yml sets api_proxy, the
// scan-db upload client must route through that proxy URL — previously the
// bare http.Client used here ignored config-file proxy settings entirely.
func TestNewHTTPClient_HonorsAPIProxy(t *testing.T) {
	// Isolate from the host environment so this test only exercises the viper
	// (mondoo.yml) path that the bug report covers.
	unsetEnv(t, "MONDOO_API_PROXY")
	unsetEnv(t, "HTTPS_PROXY")
	unsetEnv(t, "https_proxy")

	const proxyURL = "http://proxy.test.invalid:3128"
	prev := viper.GetString("api_proxy")
	viper.Set("api_proxy", proxyURL)
	t.Cleanup(func() { viper.Set("api_proxy", prev) })

	client, err := newHTTPClient()
	require.NoError(t, err)
	require.NotNil(t, client)

	tr, ok := client.Transport.(*http.Transport)
	require.True(t, ok, "expected *http.Transport, got %T", client.Transport)
	require.NotNil(t, tr.Proxy, "transport.Proxy must be set when api_proxy is configured")

	req := httptest.NewRequest(http.MethodPut, "https://storage.googleapis.com/anything", nil)
	got, err := tr.Proxy(req)
	require.NoError(t, err)
	require.NotNil(t, got, "Proxy func returned nil URL — request would bypass the proxy")
	assert.Equal(t, proxyURL, got.String())
}

// TestNewHTTPClient_NoProxyConfigured exercises the unconfigured path: no
// api_proxy, no MONDOO_API_PROXY, no HTTPS_PROXY. We should still get a
// usable client; we don't assert on transport since the default client uses
// http.DefaultTransport's ProxyFromEnvironment.
func TestNewHTTPClient_NoProxyConfigured(t *testing.T) {
	unsetEnv(t, "MONDOO_API_PROXY")
	unsetEnv(t, "HTTPS_PROXY")
	unsetEnv(t, "https_proxy")

	prev := viper.GetString("api_proxy")
	viper.Set("api_proxy", "")
	t.Cleanup(func() { viper.Set("api_proxy", prev) })

	client, err := newHTTPClient()
	require.NoError(t, err)
	require.NotNil(t, client)
}
