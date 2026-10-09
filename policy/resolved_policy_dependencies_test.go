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
// before it decides what to admit, so the resolved policy is shaped by "leaker"
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
queries:
- uid: shared-check
  mql: 1 == 1
  impact: 20
- uid: filtered-out-check
  mql: 2 == 2
- uid: leaker-check
  mql: 3 == 3
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
			policies:   []string{policyMrn("admitted"), policyMrn("filtered-out"), policyMrn("leaker")},
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

		// Dependencies don't see it. HasOverrides is what flags leaker, so
		// that a change to it invalidates every resolved policy in its spaces.
		assert.NotContains(t, rp.Dependencies, policyMrn("leaker"))
		leaker, err := srv.DataLake.GetValidatedPolicy(ctx, policyMrn("leaker"))
		require.NoError(t, err)
		assert.True(t, leaker.HasOverrides())
	})

	t.Run("only admitted policies and frameworks", func(t *testing.T) {
		assert.Equal(t, []string{
			frameworkMrn("framework1"),
			policyMrn("admitted"),
			"asset1",
		}, rp.Dependencies)
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
}

func TestPolicy_HasOverrides(t *testing.T) {
	ctx := context.Background()
	b := parseBundle(t, `
owner_mrn: //test.sth
policies:
- uid: mondoo-style
  scoring_system: highest impact
  groups:
  - filters: "true"
    checks:
    - uid: inline-check
      mql: 1 == 1
      impact: 80
    - uid: defined-check
    queries:
    - uid: inline-query
      mql: asset.name
  - filters: "false"
    policies:
    - uid: child
  risk_factors:
  - uid: own-risk
    magnitude:
      value: 0.5
    checks:
    - uid: own-risk-check
      mql: 2 == 2
- uid: child
  groups:
  - checks:
    - uid: child-check
      mql: 3 == 3
- uid: override-group
  groups:
  - type: override
    checks:
    - uid: defined-check
      action: modify
      impact: 95
- uid: disable-group
  groups:
  - type: disable
    checks:
    - uid: defined-check
- uid: ignore-group
  groups:
  - type: ignored
    checks:
    - uid: defined-check
- uid: out-of-scope-group
  groups:
  - type: out_of_scope_group
    checks:
    - uid: defined-check
- uid: reference-impact
  groups:
  - checks:
    - uid: defined-check
      impact: 95
- uid: entry-action
  groups:
  - checks:
    - uid: defined-check
      action: deactivate
- uid: policy-ref-action
  groups:
  - policies:
    - uid: child
      action: deactivate
- uid: policy-ref-impact
  groups:
  - policies:
    - uid: child
      impact: 95
- uid: policy-ref-scoring
  groups:
  - policies:
    - uid: child
      scoring_system: average
- uid: risk-reference
  risk_factors:
  - uid: own-risk
    magnitude:
      value: 0.9
queries:
- uid: defined-check
  mql: 4 == 4
  impact: 20
`)
	srv := initResolver(t, nil, []*policy.Bundle{b})

	tests := []struct {
		policy string
		want   bool
	}{
		// Inline definitions with their own impact, a reference without an
		// override, filtered groups, a policy reference without an override
		// and a risk factor's magnitude on its own definition.
		{policy: "mondoo-style", want: false},
		{policy: "child", want: false},
		{policy: "override-group", want: true},
		{policy: "disable-group", want: true},
		{policy: "ignore-group", want: true},
		{policy: "out-of-scope-group", want: true},
		{policy: "reference-impact", want: true},
		{policy: "entry-action", want: true},
		{policy: "policy-ref-action", want: true},
		{policy: "policy-ref-impact", want: true},
		{policy: "policy-ref-scoring", want: true},
		{policy: "risk-reference", want: true},
	}
	for _, tc := range tests {
		t.Run(tc.policy, func(t *testing.T) {
			p, err := srv.DataLake.GetValidatedPolicy(ctx, policyMrn(tc.policy))
			require.NoError(t, err)
			assert.Equal(t, tc.want, p.HasOverrides())
		})
	}
}
