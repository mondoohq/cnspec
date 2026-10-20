// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package policy_test

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/policy"
)

type recordedResolverEvents struct {
	mu     sync.Mutex
	misses []*policy.ResolvedPolicyCacheMiss
}

func (r *recordedResolverEvents) ResolvedPolicyCacheMiss(_ context.Context, ev *policy.ResolvedPolicyCacheMiss) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.misses = append(r.misses, ev)
}

func TestResolverEvents_ResolvedPolicyCacheMiss(t *testing.T) {
	b := parseBundle(t, `
owner_mrn: //test.sth
policies:
- uid: policy1
  groups:
  - type: chapter
    filters: asset.family.contains("linux")
    checks:
    - uid: check1
      mql: asset.name != ""
`)

	t.Run("no events in context is a no-op", func(t *testing.T) {
		srv := initResolver(t, []*testAsset{
			{asset: "asset1", policies: []string{policyMrn("policy1")}},
		}, []*policy.Bundle{b})

		_, err := srv.ResolveAndUpdateJobs(context.Background(), &policy.UpdateAssetJobsReq{
			AssetMrn:     "asset1",
			AssetFilters: []*policy.Mquery{{Mql: `asset.family.contains("linux")`}},
		})
		require.NoError(t, err)
	})

	t.Run("miss with matching keys, then hit", func(t *testing.T) {
		srv := initResolver(t, []*testAsset{
			{asset: "asset1", policies: []string{policyMrn("policy1")}},
		}, []*policy.Bundle{b})
		rec := &recordedResolverEvents{}
		ctx := policy.WithResolverEvents(context.Background(), rec)

		for range 2 {
			_, err := srv.ResolveAndUpdateJobs(ctx, &policy.UpdateAssetJobsReq{
				AssetMrn:     "asset1",
				AssetFilters: []*policy.Mquery{{Mql: `asset.family.contains("linux")`}},
			})
			require.NoError(t, err)
		}

		require.Len(t, rec.misses, 1, "the second resolve must hit the cache")
		miss := rec.misses[0]
		assert.Equal(t, "asset1", miss.PolicyMrn)
		assert.NotEmpty(t, miss.LookupFiltersChecksum)
		assert.Equal(t, miss.LookupFiltersChecksum, miss.StoredFiltersChecksum)
		assert.Equal(t, 1, miss.AssetFilters)
		assert.Equal(t, 1, miss.MatchingFilters)
		assert.Empty(t, miss.DroppedFilters)
	})

	t.Run("miss that stores under a different key reports the dropped filter", func(t *testing.T) {
		srv := initResolver(t, []*testAsset{
			{asset: "asset1", policies: []string{policyMrn("policy1")}},
		}, []*policy.Bundle{b})
		rec := &recordedResolverEvents{}
		ctx := policy.WithResolverEvents(context.Background(), rec)

		for range 2 {
			_, err := srv.ResolveAndUpdateJobs(ctx, &policy.UpdateAssetJobsReq{
				AssetMrn: "asset1",
				AssetFilters: []*policy.Mquery{
					{Mql: `asset.family.contains("linux")`},
					{Mql: `asset.platform == "arch"`, CodeId: "sent-code-id"},
				},
			})
			require.NoError(t, err)
		}

		// the row is stored under the matching key, which the lookup never asks
		// for, so every resolve misses
		require.Len(t, rec.misses, 2)
		miss := rec.misses[1]
		assert.NotEqual(t, miss.LookupFiltersChecksum, miss.StoredFiltersChecksum)
		assert.Equal(t, 2, miss.AssetFilters)
		assert.Equal(t, 1, miss.MatchingFilters)
		require.Len(t, miss.DroppedFilters, 1)
		dropped := miss.DroppedFilters[0]
		assert.Equal(t, `asset.platform == "arch"`, dropped.Mql)
		assert.Equal(t, "sent-code-id", dropped.SentCodeId)
		assert.NotEmpty(t, dropped.CodeId)
		assert.NotEqual(t, "sent-code-id", dropped.CodeId)
		assert.Empty(t, dropped.PolicyCodeId, "the policy has no filter with this MQL")
	})

	t.Run("no matching filters reports every filter as dropped", func(t *testing.T) {
		srv := initResolver(t, []*testAsset{
			{asset: "asset1", policies: []string{policyMrn("policy1")}},
		}, []*policy.Bundle{b})
		rec := &recordedResolverEvents{}
		ctx := policy.WithResolverEvents(context.Background(), rec)

		_, err := srv.ResolveAndUpdateJobs(ctx, &policy.UpdateAssetJobsReq{
			AssetMrn:     "asset1",
			AssetFilters: []*policy.Mquery{{Mql: `asset.family.contains("windows")`}},
		})
		require.Error(t, err)

		require.Len(t, rec.misses, 1)
		miss := rec.misses[0]
		assert.Empty(t, miss.StoredFiltersChecksum)
		assert.Equal(t, 0, miss.MatchingFilters)
		require.Len(t, miss.DroppedFilters, 1)
		assert.Equal(t, `asset.family.contains("windows")`, miss.DroppedFilters[0].Mql)
	})
}
