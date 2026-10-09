// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package policy_test

import (
	"context"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/policy"
)

// The space-wide override shape: "leaker" never matches the asset, since its
// only filtered group is "false", but its unfiltered override group raises the
// impact of a check "admitted" also runs. The builder gathers that override
// before it decides what to admit, so the resolved policy depends on "leaker"
// without containing it.
const rpDependenciesBundle = `
owner_mrn: //test.sth
policies:
- uid: admitted
  groups:
  - filters: "true"
    checks:
    - uid: shared-check
- uid: filtered-out
  groups:
  - filters: "false"
    checks:
    - uid: filtered-out-check
- uid: leaker
  groups:
  - filters: "false"
    checks:
    - uid: leaker-check
  - type: override
    checks:
    - uid: shared-check
      action: modify
      impact: 95
- uid: bystander
  groups:
  - filters: "false"
    checks:
    - uid: bystander-check
  - type: override
    checks:
    - uid: filtered-out-check
      action: modify
      impact: 95
queries:
- uid: shared-check
  mql: 1 == 1
  impact: 20
- uid: filtered-out-check
  mql: 2 == 2
- uid: leaker-check
  mql: 3 == 3
- uid: bystander-check
  mql: 4 == 4
frameworks:
- uid: framework1
  name: framework1
  groups:
  - title: group1
    controls:
    - uid: control1
      title: control1
framework_maps:
- uid: framework-map1
  framework_owner:
    uid: framework1
  policy_dependencies:
  - uid: admitted
  controls:
  - uid: control1
    checks:
    - uid: shared-check
`

func TestResolveV2_Dependencies(t *testing.T) {
	ctx := context.Background()
	b := parseBundle(t, rpDependenciesBundle)

	srv := initResolver(t, []*testAsset{
		{
			asset:      "asset1",
			policies:   []string{policyMrn("admitted"), policyMrn("filtered-out"), policyMrn("leaker"), policyMrn("bystander")},
			frameworks: []string{frameworkMrn("framework1")},
		},
	}, []*policy.Bundle{b})

	rp, err := srv.Resolve(ctx, &policy.ResolveReq{
		PolicyMrn:    "asset1",
		AssetFilters: []*policy.Mquery{{Mql: "true"}},
	})
	require.NoError(t, err)
	require.NotNil(t, rp)

	assert.True(t, slices.IsSorted(rp.Dependencies), "dependencies are not sorted: %v", rp.Dependencies)
	assert.Len(t, slices.Compact(slices.Clone(rp.Dependencies)), len(rp.Dependencies), "dependencies have duplicates: %v", rp.Dependencies)

	t.Run("admitted policies and frameworks", func(t *testing.T) {
		assert.Contains(t, rp.Dependencies, "asset1")
		assert.Contains(t, rp.Dependencies, policyMrn("admitted"))
		assert.Contains(t, rp.Dependencies, frameworkMrn("framework1"))
	})

	t.Run("a policy whose filters don't match", func(t *testing.T) {
		assert.NotContains(t, rp.Dependencies, policyMrn("filtered-out"))
	})

	t.Run("a policy whose override reaches the resolved policy without it", func(t *testing.T) {
		// leaker is not part of the resolved policy...
		assert.Nil(t, findReportingJobByQrId(rp, policyMrn("leaker")))

		// ...but its impact is: 95 instead of the check's own 20, on the
		// check's edge to the code it runs and on its edge to "admitted".
		checkJob := findReportingJobByQrId(rp, queryMrn("shared-check"))
		require.NotNil(t, checkJob)
		require.Len(t, checkJob.ChildJobs, 1)
		for _, impact := range checkJob.ChildJobs {
			assert.Equal(t, int32(95), impact.GetValue().GetValue())
		}
		admittedJob := findReportingJobByQrId(rp, policyMrn("admitted"))
		require.NotNil(t, admittedJob)
		require.Contains(t, admittedJob.ChildJobs, checkJob.Uuid)
		assert.Equal(t, int32(95), admittedJob.ChildJobs[checkJob.Uuid].GetValue().GetValue())

		assert.Contains(t, rp.Dependencies, policyMrn("leaker"))
	})

	t.Run("a policy whose override reaches nothing in the resolved policy", func(t *testing.T) {
		// bystander overrides a check only filtered-out runs.
		assert.NotContains(t, rp.Dependencies, policyMrn("bystander"))
	})
}

func TestResolveV2_DependenciesWithoutLeak(t *testing.T) {
	ctx := context.Background()
	b := parseBundle(t, rpDependenciesBundle)

	// Without leaker the check keeps its own impact, which is what makes the
	// leaker case above a leak.
	srv := initResolver(t, []*testAsset{
		{asset: "asset1", policies: []string{policyMrn("admitted"), policyMrn("filtered-out")}},
	}, []*policy.Bundle{b})

	rp, err := srv.Resolve(ctx, &policy.ResolveReq{
		PolicyMrn:    "asset1",
		AssetFilters: []*policy.Mquery{{Mql: "true"}},
	})
	require.NoError(t, err)

	checkJob := findReportingJobByQrId(rp, queryMrn("shared-check"))
	require.NotNil(t, checkJob)
	for _, impact := range checkJob.ChildJobs {
		assert.Equal(t, int32(20), impact.GetValue().GetValue())
	}
	assert.Equal(t, []string{policyMrn("admitted"), "asset1"}, rp.Dependencies)
}

func TestPolicy_HasUngatedGlobalInfo(t *testing.T) {
	ctx := context.Background()
	b := parseBundle(t, `
owner_mrn: //test.sth
policies:
- uid: unfiltered-override
  groups:
  - filters: "false"
    checks:
    - uid: own-check
  - type: override
    checks:
    - uid: unfiltered-check
      action: modify
      impact: 95
- uid: filtered-group
  groups:
  - type: override
    filters: "false"
    checks:
    - uid: unfiltered-check
      action: modify
      impact: 95
- uid: filtered-check
  groups:
  - type: override
    checks:
    - uid: filtered-check
      action: modify
      impact: 95
- uid: plain
  groups:
  - checks:
    - uid: unfiltered-check
    - uid: own-check
queries:
- uid: unfiltered-check
  mql: 1 == 1
- uid: filtered-check
  filters: "false"
  mql: 2 == 2
- uid: own-check
  filters: "false"
  mql: 3 == 3
`)
	srv := initResolver(t, nil, []*policy.Bundle{b})

	tests := []struct {
		policy string
		want   bool
	}{
		// The leak: nothing gates the override, so it reaches resolved
		// policies that never admit the policy.
		{policy: "unfiltered-override", want: true},
		// The group's filter is one of the policy's filters: wherever the
		// override applies, the policy is admitted too.
		{policy: "filtered-group", want: false},
		// Same for a filter on the check the override targets.
		{policy: "filtered-check", want: false},
		// No overrides at all.
		{policy: "plain", want: false},
	}
	for _, tc := range tests {
		t.Run(tc.policy, func(t *testing.T) {
			p, err := srv.DataLake.GetValidatedPolicy(ctx, policyMrn(tc.policy))
			require.NoError(t, err)

			got, err := p.HasUngatedGlobalInfo(ctx, srv.DataLake.GetQuery)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}
