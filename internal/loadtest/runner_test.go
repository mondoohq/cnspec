// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package loadtest

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/v13/policy"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
)

// recordingClient implements Client by appending each call into slices guarded
// by a mutex so concurrent worker goroutines stay safe.
type recordingClient struct {
	mu      sync.Mutex
	syncs   []*inventory.Asset
	stores  []*policy.StoreResultsReq
	resolve []string
}

func (r *recordingClient) SynchronizeAsset(_ context.Context, _ string, asset *inventory.Asset) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.syncs = append(r.syncs, asset)
	return "//assets.api.mondoo.com/" + asset.PlatformIds[0], nil
}

func (r *recordingClient) ResolveAndUpdateJobs(_ context.Context, mrn string, _ *inventory.Asset) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.resolve = append(r.resolve, mrn)
	return nil
}

func (r *recordingClient) StoreResults(_ context.Context, req *policy.StoreResultsReq) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stores = append(r.stores, req)
	return nil
}

func TestRunnerBaselineUsesTemplateScores(t *testing.T) {
	tpl := makeStateTemplate(5)
	tpl.Asset = &inventory.Asset{Name: "tpl", PlatformIds: []string{"x"}}
	rc := &recordingClient{}

	stats, err := Run(context.Background(), Config{
		SpaceMrn:      "//captain.api.mondoo.app/spaces/test",
		Templates:     []*Template{tpl},
		Assets:        1,
		ScansPerAsset: 1,
		ChangePct:     50,
		Seed:          1,
		TotalShards:   1,
		Workers:       1,
		Client:        rc,
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, stats.SyncCalls)
	require.EqualValues(t, 1, stats.StoreCalls)

	require.Len(t, rc.stores, 1)
	for i, s := range rc.stores[0].Scores {
		require.Equal(t, tpl.Scores[i].Value, s.Value, "baseline (scan 0) must replay template scores even when change-pct > 0")
	}
}

func TestRunnerSyncOnlyOnFirstScan(t *testing.T) {
	tpl := makeStateTemplate(3)
	tpl.Asset = &inventory.Asset{Name: "tpl", PlatformIds: []string{"x"}}
	rc := &recordingClient{}

	_, err := Run(context.Background(), Config{
		SpaceMrn:      "//captain.api.mondoo.app/spaces/test",
		Templates:     []*Template{tpl},
		Assets:        2,
		ScansPerAsset: 4,
		Seed:          1,
		TotalShards:   1,
		Workers:       1,
		Client:        rc,
	})
	require.NoError(t, err)
	require.Len(t, rc.syncs, 2, "one sync per asset")
	require.Len(t, rc.resolve, 2, "one resolve per asset")
	require.Len(t, rc.stores, 8, "store on every scan: 2 assets * 4 scans")
}

func TestRunnerSharding(t *testing.T) {
	tpl := makeStateTemplate(2)
	tpl.Asset = &inventory.Asset{Name: "tpl", PlatformIds: []string{"x"}}

	const totalAssets = 20
	allShards := map[string]int{}
	for shard := 0; shard < 4; shard++ {
		rc := &recordingClient{}
		_, err := Run(context.Background(), Config{
			SpaceMrn:      "//captain.api.mondoo.app/spaces/test",
			Templates:     []*Template{tpl},
			Assets:        totalAssets,
			ScansPerAsset: 1,
			Seed:          42,
			ShardID:       shard,
			TotalShards:   4,
			Workers:       2,
			Client:        rc,
		})
		require.NoError(t, err)
		for _, a := range rc.syncs {
			allShards[a.PlatformIds[0]]++
		}
	}
	require.Len(t, allShards, totalAssets, "shards must collectively cover every asset exactly once")
	for pid, count := range allShards {
		require.Equal(t, 1, count, "asset %s handled by multiple shards", pid)
	}
}

func TestRunnerValidate(t *testing.T) {
	tpl := makeStateTemplate(1)
	tpl.Asset = &inventory.Asset{Name: "tpl", PlatformIds: []string{"x"}}
	rc := &recordingClient{}

	_, err := Run(context.Background(), Config{
		SpaceMrn:    "",
		Templates:   []*Template{tpl},
		Assets:      1,
		TotalShards: 1,
		Workers:     1,
		Client:      rc,
	})
	require.Error(t, err)

	_, err = Run(context.Background(), Config{
		SpaceMrn:      "//x",
		Templates:     []*Template{tpl},
		Assets:        1,
		ScansPerAsset: 1,
		ShardID:       5,
		TotalShards:   2,
		Workers:       1,
		Client:        rc,
	})
	require.Error(t, err)
}
