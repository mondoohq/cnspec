// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// TestNewFailureReporter_Gating verifies that no reporter (a no-op) is created
// when there is no upstream to report to, so incognito/offline scans never
// attempt failure telemetry.
func TestNewFailureReporter_Gating(t *testing.T) {
	assert.Nil(t, newFailureReporter(nil, ""), "nil upstream")
	assert.Nil(t, newFailureReporter(&upstream.UpstreamConfig{Incognito: true, ApiEndpoint: "https://x"}, ""), "incognito")
	assert.Nil(t, newFailureReporter(&upstream.UpstreamConfig{ApiEndpoint: ""}, ""), "no endpoint")

	fr := newFailureReporter(&upstream.UpstreamConfig{ApiEndpoint: "https://x", SpaceMrn: "//spaces/abc"}, "")
	if assert.NotNil(t, fr, "valid upstream should produce a reporter") {
		assert.Equal(t, "//spaces/abc", fr.spaceMrn, "spaceMrn falls back to upstream config")
	}
}

// TestFailureReporter_NilSafe ensures a nil reporter (the incognito/offline
// case) is safe to use everywhere.
func TestFailureReporter_NilSafe(t *testing.T) {
	var fr *failureReporter
	assert.NotPanics(t, func() {
		fr.report(&inventory.Asset{Mrn: "//assets/1"}, assert.AnError)
		fr.close()
	})
}

// TestFailureReporter_AdmissionBounds checks that admission is bounded by both
// the in-flight concurrency limit and the per-scan total cap, dropping the rest
// instead of blocking. Slots are never released here, so exactly
// maxConcurrentFailureReports admissions succeed and everything else is dropped.
func TestFailureReporter_AdmissionBounds(t *testing.T) {
	fr := newFailureReporter(&upstream.UpstreamConfig{ApiEndpoint: "https://x"}, "//spaces/abc")
	if !assert.NotNil(t, fr) {
		return
	}

	attempts := maxFailureReportsPerScan + 50
	admitted := 0
	for i := 0; i < attempts; i++ {
		if fr.tryAdmit() {
			admitted++
		}
	}

	// Only the concurrency slots can be admitted since none are ever released.
	assert.Equal(t, maxConcurrentFailureReports, admitted)
	assert.Equal(t, int64(attempts), fr.submitted.Load())
	assert.Equal(t, int64(attempts-maxConcurrentFailureReports), fr.dropped.Load())
	// At least the calls beyond the per-scan cap must have been dropped.
	assert.GreaterOrEqual(t, fr.dropped.Load(), int64(attempts-maxFailureReportsPerScan))
}

// TestFailureReporter_DispatchesAsync verifies that report() delivers the
// failure through the send hook off the caller's goroutine, with the expected
// message and asset/space tags, and that close() drains outstanding sends.
func TestFailureReporter_DispatchesAsync(t *testing.T) {
	fr := newFailureReporter(&upstream.UpstreamConfig{ApiEndpoint: "https://x"}, "//spaces/abc")
	require.NotNil(t, fr)

	type call struct {
		msg  string
		tags map[string]string
	}
	var mu sync.Mutex
	var calls []call
	fr.send = func(msg string, tags map[string]string) {
		mu.Lock()
		defer mu.Unlock()
		calls = append(calls, call{msg: msg, tags: tags})
	}

	asset := &inventory.Asset{
		Mrn:         "//assets/1",
		Name:        "web",
		PlatformIds: []string{"//p/1"},
		Platform:    &inventory.Platform{Name: "ubuntu", Version: "22.04"},
	}
	fr.report(asset, fmt.Errorf("boom"))
	fr.report(asset, fmt.Errorf("bang"))

	fr.close() // drains outstanding sends

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, calls, 2)
	msgs := map[string]bool{}
	for _, c := range calls {
		msgs[c.msg] = true
		assert.Equal(t, "//assets/1", c.tags["assetMrn"])
		assert.Equal(t, "//spaces/abc", c.tags["spaceMrn"])
		assert.Equal(t, "ubuntu", c.tags["assetPlatform"])
	}
	assert.True(t, msgs["scan failure: boom"], "expected boom message")
	assert.True(t, msgs["scan failure: bang"], "expected bang message")
	assert.Zero(t, fr.dropped.Load())
}
