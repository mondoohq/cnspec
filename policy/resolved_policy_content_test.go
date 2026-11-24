// Copyright Mondoo, Inc. 2026
// SPDX-License-Identifier: BUSL-1.1

package policy_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/internal/datalakes/inmemory"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/mql/providers-sdk/v1/testutils"
)

// contentBundle has one policy the asset's filters admit and one they do not.
// Each call varies one part of it.
func contentBundle(admittedMql, admittedImpact, otherMql, otherOverride string) string {
	return `
owner_mrn: //test.sth
policies:
  - uid: admitted
    groups:
    - filters: asset.name == "asset1"
      checks:
      - uid: admitted-check
        mql: ` + admittedMql + `
        impact: ` + admittedImpact + `
      - uid: shared-unfiltered-check
        mql: 5 == 5
  - uid: not-admitted
    groups:
    - filters: asset.name == "other"
      checks:
      - uid: not-admitted-check
        mql: ` + otherMql + `
` + otherOverride
}

func resolveContent(t *testing.T, bundle string) *policy.ResolvedPolicy {
	ctx := context.Background()
	_, srv, err := inmemory.NewServices(testutils.LinuxMock())
	require.NoError(t, err)

	_, err = srv.SetBundle(ctx, parseBundle(t, bundle))
	require.NoError(t, err)
	_, err = srv.Assign(ctx, &policy.PolicyAssignment{
		AssetMrn:   "asset1",
		PolicyMrns: []string{policyMrn("admitted"), policyMrn("not-admitted")},
	})
	require.NoError(t, err)

	rp, err := srv.ResolveAndUpdateJobs(ctx, &policy.UpdateAssetJobsReq{
		AssetMrn:     "asset1",
		AssetFilters: []*policy.Mquery{{Mql: `asset.name == "asset1"`}},
	})
	require.NoError(t, err)
	return rp
}

func TestResolvedPolicyContentChecksum(t *testing.T) {
	base := resolveContent(t, contentBundle("1 == 1", "50", "2 == 2", ""))
	require.NotEmpty(t, base.ContentChecksum())

	t.Run("a policy the asset's filters do not admit moves the graph checksum, not the content", func(t *testing.T) {
		rp := resolveContent(t, contentBundle("1 == 1", "50", "3 == 3", ""))
		require.NotEqual(t, base.GraphExecutionChecksum, rp.GraphExecutionChecksum, "precondition: the bundle changed")
		require.NotEqual(t, base.GetCollectorJob().GetChecksum(), rp.GetCollectorJob().GetChecksum(), "precondition: every UUID changed with it")
		assert.Equal(t, base.ContentChecksum(), rp.ContentChecksum())
	})

	t.Run("an admitted check's impact moves the content", func(t *testing.T) {
		rp := resolveContent(t, contentBundle("1 == 1", "80", "2 == 2", ""))
		assert.NotEqual(t, base.ContentChecksum(), rp.ContentChecksum())
	})

	t.Run("an admitted check's query moves the content", func(t *testing.T) {
		rp := resolveContent(t, contentBundle("1 == 2", "50", "2 == 2", ""))
		assert.NotEqual(t, base.ContentChecksum(), rp.ContentChecksum())
	})

	// The builder gathers overrides from every policy, admitted or not
	// (gatherGlobalInfoFromPolicy). A policy the asset's filters do not admit
	// can still raise the impact of an unfiltered check an admitted policy
	// shares, and that reaches the asset. The content has to move with it,
	// which a key built from the admitted policies alone would miss.
	t.Run("a non-admitted policy's impact on a shared unfiltered check moves the content", func(t *testing.T) {
		withOverride := `      - uid: shared-unfiltered-check
        impact: 90
`
		rp := resolveContent(t, contentBundle("1 == 1", "50", "2 == 2", withOverride))
		assert.NotEqual(t, base.ContentChecksum(), rp.ContentChecksum())
	})

	t.Run("deterministic", func(t *testing.T) {
		rp := resolveContent(t, contentBundle("1 == 1", "50", "2 == 2", ""))
		for range 20 {
			assert.Equal(t, base.ContentChecksum(), rp.ContentChecksum())
		}
	})
}

func TestResolvedPolicyContentChecksum_Nil(t *testing.T) {
	var rp *policy.ResolvedPolicy
	assert.Empty(t, rp.ContentChecksum())
}
