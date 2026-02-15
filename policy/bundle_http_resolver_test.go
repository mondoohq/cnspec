// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package policy_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/v12/policy"
)

func TestHTTPBundleResolver_IsApplicable(t *testing.T) {
	resolver := policy.NewHTTPBundleResolver(nil)

	assert.True(t, resolver.IsApplicable("http://example.com/policy.mql.yaml"))
	assert.True(t, resolver.IsApplicable("https://example.com/policy.mql.yaml"))
	assert.True(t, resolver.IsApplicable("https://raw.githubusercontent.com/org/repo/main/policy.mql.yaml"))
	assert.False(t, resolver.IsApplicable("s3://bucket/policy.mql.yaml"))
	assert.False(t, resolver.IsApplicable("/path/to/policy.mql.yaml"))
	assert.False(t, resolver.IsApplicable("relative/path.mql.yaml"))
	assert.False(t, resolver.IsApplicable(""))
}

func TestBundleFromHTTP(t *testing.T) {
	data, err := os.ReadFile("../examples/example.mql.yaml")
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	}))
	defer server.Close()

	loader := policy.NewBundleLoader(policy.NewHTTPBundleResolver(server.Client()))
	bundle, err := loader.BundleFromPaths(server.URL + "/example.mql.yaml")

	require.NoError(t, err)
	require.NotNil(t, bundle)
	assert.Len(t, bundle.Queries, 1)
	require.Len(t, bundle.Policies, 1)
	require.Len(t, bundle.Policies[0].Groups, 1)
	assert.Len(t, bundle.Policies[0].Groups[0].Checks, 3)
	assert.Len(t, bundle.Policies[0].Groups[0].Queries, 2)
}

func TestBundleFromHTTP_400Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	loader := policy.NewBundleLoader(policy.NewHTTPBundleResolver(server.Client()))
	bundle, err := loader.BundleFromPaths(server.URL + "/policy.mql.yaml")

	require.Error(t, err)
	assert.Nil(t, bundle)
	assert.Contains(t, err.Error(), "400 Bad Request")
	assert.Contains(t, err.Error(), "malformed")
}

func TestBundleFromHTTP_403Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	loader := policy.NewBundleLoader(policy.NewHTTPBundleResolver(server.Client()))
	bundle, err := loader.BundleFromPaths(server.URL + "/policy.mql.yaml")

	require.Error(t, err)
	assert.Nil(t, bundle)
	assert.Contains(t, err.Error(), "403 Forbidden")
	assert.Contains(t, err.Error(), "access denied")
}

func TestBundleFromHTTP_404Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	loader := policy.NewBundleLoader(policy.NewHTTPBundleResolver(server.Client()))
	bundle, err := loader.BundleFromPaths(server.URL + "/policy.mql.yaml")

	require.Error(t, err)
	assert.Nil(t, bundle)
	assert.Contains(t, err.Error(), "404 Not Found")
	assert.Contains(t, err.Error(), "resource not found")
}

func TestBundleFromHTTP_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	loader := policy.NewBundleLoader(policy.NewHTTPBundleResolver(server.Client()))
	bundle, err := loader.BundleFromPaths(server.URL + "/policy.mql.yaml")

	require.Error(t, err)
	assert.Nil(t, bundle)
	assert.Contains(t, err.Error(), "500")
}

func TestBundleFromMixedSources_HTTPAndLocal(t *testing.T) {
	data, err := os.ReadFile("../examples/directory/example1.mql.yaml")
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	}))
	defer server.Close()

	loader := policy.NewBundleLoader(
		policy.NewHTTPBundleResolver(server.Client()),
		policy.NewFileBundleResolver(),
	)
	bundle, err := loader.BundleFromPaths(
		server.URL+"/example1.mql.yaml",
		"../examples/directory/example2.mql.yaml",
	)

	require.NoError(t, err)
	require.NotNil(t, bundle)
	assert.Len(t, bundle.Policies, 2)
}
