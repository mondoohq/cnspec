// Copyright Mondoo, Inc. 2026
// SPDX-License-Identifier: BUSL-1.1

package sqlite

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

// TestNewUploadHTTPClient_HonorsAPIProxy is the regression test for the
// bucket-upload-ignores-api_proxy bug. When mondoo.yml sets api_proxy, the
// scan-db upload client must route through that proxy URL.
func TestNewUploadHTTPClient_HonorsAPIProxy(t *testing.T) {
	// Isolate from the host environment so this test only exercises the viper
	// (mondoo.yml) path that the bug report covers.
	unsetEnv(t, "MONDOO_API_PROXY")
	unsetEnv(t, "HTTPS_PROXY")
	unsetEnv(t, "https_proxy")

	const proxyURL = "http://proxy.test.invalid:3128"
	prev := viper.GetString("api_proxy")
	viper.Set("api_proxy", proxyURL)
	t.Cleanup(func() { viper.Set("api_proxy", prev) })

	client, err := newUploadHTTPClient()
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

// TestNewUploadHTTPClient_NoProxyConfigured exercises the unconfigured path:
// no api_proxy, no MONDOO_API_PROXY, no HTTPS_PROXY. We should still get a
// usable client; we don't assert on transport since the default client uses
// http.DefaultTransport's ProxyFromEnvironment.
func TestNewUploadHTTPClient_NoProxyConfigured(t *testing.T) {
	unsetEnv(t, "MONDOO_API_PROXY")
	unsetEnv(t, "HTTPS_PROXY")
	unsetEnv(t, "https_proxy")

	prev := viper.GetString("api_proxy")
	viper.Set("api_proxy", "")
	t.Cleanup(func() { viper.Set("api_proxy", prev) })

	client, err := newUploadHTTPClient()
	require.NoError(t, err)
	require.NotNil(t, client)
}
