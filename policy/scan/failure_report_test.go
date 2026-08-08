// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"testing"

	"github.com/cockroachdb/errors"
	"github.com/stretchr/testify/assert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/upstream"
)

func TestAssetErrorTags(t *testing.T) {
	t.Run("full asset", func(t *testing.T) {
		asset := &inventory.Asset{
			Mrn:         "//assets/1",
			Name:        "web-server",
			PlatformIds: []string{"//platformid/1", "//platformid/2"},
			Platform:    &inventory.Platform{Name: "ubuntu", Version: "22.04"},
		}
		tags := assetErrorTags("//spaces/abc", asset)
		assert.Equal(t, "//spaces/abc", tags["spaceMrn"])
		assert.Equal(t, "//assets/1", tags["assetMrn"])
		assert.Equal(t, "web-server", tags["assetName"])
		assert.Equal(t, "//platformid/1,//platformid/2", tags["platformIDs"])
		assert.Equal(t, "ubuntu", tags["assetPlatform"])
		assert.Equal(t, "22.04", tags["assetPlatformVersion"])
	})

	t.Run("no space mrn", func(t *testing.T) {
		tags := assetErrorTags("", &inventory.Asset{Mrn: "//assets/1", Name: "x"})
		_, ok := tags["spaceMrn"]
		assert.False(t, ok, "spaceMrn should be omitted when empty")
	})

	t.Run("nil asset", func(t *testing.T) {
		tags := assetErrorTags("//spaces/abc", nil)
		assert.Equal(t, "//spaces/abc", tags["spaceMrn"])
		_, ok := tags["assetMrn"]
		assert.False(t, ok)
	})

	t.Run("no platform", func(t *testing.T) {
		tags := assetErrorTags("", &inventory.Asset{Mrn: "//assets/1", Name: "x"})
		_, ok := tags["assetPlatform"]
		assert.False(t, ok)
		_, ok = tags["platformIDs"]
		assert.False(t, ok)
	})
}

// TestReportScanFailure_Noop verifies the gates that keep cnspec from attempting
// an upstream failure report when there is no upstream to report to. These paths
// must return before touching the network so they are safe to call for every
// asset error, including in incognito and offline scans.
func TestReportScanFailure_Noop(t *testing.T) {
	s := NewLocalScanner()
	asset := &inventory.Asset{Mrn: "//assets/1", Name: "x"}
	err := errors.New("boom")

	// None of these should panic or block; they must return via an early gate.
	assert.NotPanics(t, func() {
		s.reportScanFailure(nil, "", asset, err)                                                         // nil upstream
		s.reportScanFailure(&upstream.UpstreamConfig{Incognito: true, ApiEndpoint: "x"}, "", asset, err) // incognito
		s.reportScanFailure(&upstream.UpstreamConfig{ApiEndpoint: ""}, "", asset, err)                   // no endpoint
		s.reportScanFailure(&upstream.UpstreamConfig{ApiEndpoint: "x"}, "", nil, err)                    // nil asset
		s.reportScanFailure(&upstream.UpstreamConfig{ApiEndpoint: "x"}, "", asset, nil)                  // nil error
	})
}
