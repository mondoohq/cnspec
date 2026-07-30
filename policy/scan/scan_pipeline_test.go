// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/v13/cli/progress"
	"go.mondoo.com/mql/v13/discovery"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
)

// Assets are submitted to the dispatcher before a worker slot is free, so they
// can be closed while they wait: the explorer shuts down when the scan returns
// early, and dedup evicts assets that turn out to be a subset of a newly
// connected one. Both clear the runtime, which used to segfault in
// EnsureProvidersConnected as soon as the queued scan started.
func TestScanSingleAssetSkipsClosedAsset(t *testing.T) {
	d := &scanDispatcher{multiprogress: progress.NoopMultiProgress{}}

	t.Run("runtime cleared", func(t *testing.T) {
		tracked := &discovery.TrackedAsset{
			Asset: &inventory.Asset{Name: "closed", PlatformIds: []string{"//platformid/1"}},
			State: discovery.AssetClosed,
		}
		require.NotPanics(t, func() {
			d.scanSingleAsset(context.Background(), tracked)
		})
	})

	t.Run("asset and runtime cleared", func(t *testing.T) {
		require.NotPanics(t, func() {
			d.scanSingleAsset(context.Background(), &discovery.TrackedAsset{State: discovery.AssetClosed})
		})
	})
}

// recoverAssetPanic must recover the panic itself. It previously delegated to
// health.ReportPanic, whose recover() sat one frame too deep to catch anything,
// so a single bad asset crashed the whole scan process.
func TestRecoverAssetPanic(t *testing.T) {
	t.Run("records the asset as failed", func(t *testing.T) {
		reporter := NewAggregateReporter()
		d := &scanDispatcher{
			multiprogress: progress.NoopMultiProgress{},
			// A zero-value explorer tracks no assets, so CloseAsset reports
			// the asset as untracked instead of touching a live connection.
			explorer: &discovery.AssetExplorer{},
			reporter: reporter,
			spaceMrn: "//captain.api.mondoo.app/spaces/test",
		}
		assetMrn := "//assets.api.mondoo.app/spaces/test/assets/1"
		tracked := &discovery.TrackedAsset{
			Asset: &inventory.Asset{
				Name:        "panicky",
				Mrn:         assetMrn,
				PlatformIds: []string{"//platformid/1"},
			},
		}

		require.NotPanics(t, func() {
			defer d.recoverAssetPanic(tracked)
			panic("boom")
		})

		assert.Nil(t, tracked.Asset, "asset reference should be released")
		res := reporter.Reports()
		assert.False(t, res.Ok)
		assert.Contains(t, res.GetFull().Errors[assetMrn], "boom")
	})

	t.Run("without an asset", func(t *testing.T) {
		d := &scanDispatcher{multiprogress: progress.NoopMultiProgress{}}
		require.NotPanics(t, func() {
			defer d.recoverAssetPanic(nil)
			panic("boom")
		})
	})

	t.Run("no panic", func(t *testing.T) {
		d := &scanDispatcher{multiprogress: progress.NoopMultiProgress{}}
		require.NotPanics(t, func() {
			defer d.recoverAssetPanic(nil)
		})
	})
}
