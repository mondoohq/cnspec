package policy_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/explorer"
	"go.mondoo.com/cnquery/mrn"
	"go.mondoo.com/cnspec/internal/datalakes/inmemory"
	"go.mondoo.com/cnspec/policy"
)

type testAsset struct {
	asset    string
	policies []string
}

func parseBundle(t *testing.T, data string) *policy.Bundle {
	res, err := policy.BundleFromYAML([]byte(data))
	require.NoError(t, err)
	return res
}

func initResolver(t *testing.T, assets []*testAsset, bundles []*policy.Bundle) *policy.LocalServices {
	_, srv, err := inmemory.NewServices(nil)
	require.NoError(t, err)

	for i := range bundles {
		bundle := bundles[i]
		_, err := srv.SetBundle(context.Background(), bundle)
		require.NoError(t, err)
	}

	for i := range assets {
		asset := assets[i]
		_, err := srv.Assign(context.Background(), &policy.PolicyAssignment{
			AssetMrn:   asset.asset,
			PolicyMrns: asset.policies,
		})
		require.NoError(t, err)
	}

	return srv
}

func policyMrn(uid string) string {
	return "//test.sth/policies/" + uid
}

func TestResolve_EmptyPolicy(t *testing.T) {
	b := parseBundle(t, `
owner_mrn: //test.sth
policies:
- uid: policy1
`)

	srv := initResolver(t, []*testAsset{
		{"asset1", []string{policyMrn("policy1")}},
	}, []*policy.Bundle{b})

	t.Run("resolve w/o filters", func(t *testing.T) {
		_, err := srv.Resolve(context.Background(), &policy.ResolveReq{
			PolicyMrn: policyMrn("policy1"),
		})
		assert.EqualError(t, err, "rpc error: code = InvalidArgument desc = asset doesn't support any policies")
	})

	t.Run("resolve with empty filters", func(t *testing.T) {
		_, err := srv.Resolve(context.Background(), &policy.ResolveReq{
			PolicyMrn:    policyMrn("policy1"),
			AssetFilters: []*explorer.Mquery{{}},
		})
		assert.EqualError(t, err, "failed to compile query: failed to compile query '': query is not implemented ''")
	})

	t.Run("resolve with random filters", func(t *testing.T) {
		_, err := srv.Resolve(context.Background(), &policy.ResolveReq{
			PolicyMrn:    policyMrn("policy1"),
			AssetFilters: []*explorer.Mquery{{Mql: "true"}},
		})
		assert.EqualError(t, err,
			"rpc error: code = InvalidArgument desc = asset isn't supported by any policies\n"+
				"policies didn't provide any filters\n"+
				"asset supports: true\n")
	})
}

func TestResolve_SimplePolicy(t *testing.T) {
	b := parseBundle(t, `
owner_mrn: //test.sth
policies:
- uid: policy1
  groups:
  - type: chapter
    filters: "true"
    checks:
    - uid: check1
      mql: asset.name == props.name
      props:
      - uid: name
        mql: return "definitely not the asset name"
    queries:
    - uid: query1
      mql: asset{*}
`)

	srv := initResolver(t, []*testAsset{
		{"asset1", []string{policyMrn("policy1")}},
	}, []*policy.Bundle{b})

	checkResolvedPolicy := func(t *testing.T, rp *policy.ResolvedPolicy) {
		require.Len(t, rp.ExecutionJob.Queries, 3)
		require.Len(t, rp.Filters, 1)
	}

	t.Run("resolve with correct filters", func(t *testing.T) {
		rp, err := srv.Resolve(context.Background(), &policy.ResolveReq{
			PolicyMrn:    policyMrn("policy1"),
			AssetFilters: []*explorer.Mquery{{Mql: "true"}},
		})
		require.NoError(t, err)
		require.NotNil(t, rp)
		checkResolvedPolicy(t, rp)
	})

	t.Run("resolve with many filters (one is correct)", func(t *testing.T) {
		rp, err := srv.Resolve(context.Background(), &policy.ResolveReq{
			PolicyMrn: policyMrn("policy1"),
			AssetFilters: []*explorer.Mquery{
				{Mql: "asset.family.contains(\"linux\")"},
				{Mql: "true"},
				{Mql: "asset.family.contains(\"windows\")"},
			},
		})
		require.NoError(t, err)
		require.NotNil(t, rp)
	})

	t.Run("resolve with incorrect filters", func(t *testing.T) {
		_, err := srv.Resolve(context.Background(), &policy.ResolveReq{
			PolicyMrn: policyMrn("policy1"),
			AssetFilters: []*explorer.Mquery{
				{Mql: "asset.family.contains(\"linux\")"},
				{Mql: "false"},
				{Mql: "asset.family.contains(\"windows\")"},
			},
		})
		assert.EqualError(t, err,
			"rpc error: code = InvalidArgument desc = asset isn't supported by any policies\n"+
				"policies support: true\n"+
				"asset supports: asset.family.contains(\"linux\"), asset.family.contains(\"windows\"), false\n")
	})
}

func TestResolve_PolicyActionIgnore(t *testing.T) {
	b := parseBundle(t, `
owner_mrn: //test.sth
policies:
- owner_mrn: //test.sth
  mrn: //test.sth
  groups:
  - policies:
    - uid: policy-active
    - uid: policy-ignored
      action: 4
- uid: policy-active
  owner_mrn: //test.sth
  groups:
  - type: chapter
    filters: "true"
    checks:
    - uid: check1
      mql: asset.name == "definitely not the asset name"
    queries:
    - uid: query1
      mql: asset{*}
- uid: policy-ignored
  owner_mrn: //test.sth
  groups:
  - type: chapter
    filters: "true"
    checks:
    - uid: check1
      mql: asset.name == "definitely not the asset name"
    queries:
    - uid: query1
      mql: asset{*}
`)

	srv := initResolver(t, []*testAsset{
		{"asset1", []string{policyMrn("policy-active"), policyMrn("policy-ignored")}},
	}, []*policy.Bundle{b})

	t.Run("resolve with ignored policy", func(t *testing.T) {
		rp, err := srv.Resolve(context.Background(), &policy.ResolveReq{
			PolicyMrn:    "//test.sth",
			AssetFilters: []*explorer.Mquery{{Mql: "true"}},
		})
		require.NoError(t, err)
		require.NotNil(t, rp)
		require.Len(t, rp.CollectorJob.ReportingJobs, 4)
		ignoreJob := rp.CollectorJob.ReportingJobs["jGWUFIvetOg="]
		require.Equal(t, explorer.ScoringSystem_IGNORE_SCORE, ignoreJob.ChildJobs["lgJDqBZEz+M="].Scoring)
	})
}

func TestResolve_ExpiredGroups(t *testing.T) {
	b := parseBundle(t, `
owner_mrn: //test.sth
policies:
- uid: policy1
  groups:
  - type: chapter
    filters: "true"
    checks:
    - uid: check1
      mql: "1 == 1"
    - uid: check2
      mql: "1 == 2"
`)

	_, srv, err := inmemory.NewServices(nil)
	require.NoError(t, err)

	_, err = srv.SetBundle(context.Background(), b)
	require.NoError(t, err)

	_, err = srv.Assign(context.Background(), &policy.PolicyAssignment{
		AssetMrn:   "asset1",
		PolicyMrns: []string{policyMrn("policy1")},
	})
	require.NoError(t, err)

	filters, err := srv.GetPolicyFilters(context.Background(), &policy.Mrn{Mrn: "asset1"})
	require.NoError(t, err)
	assetPolicy, err := srv.GetPolicy(context.Background(), &policy.Mrn{Mrn: "asset1"})
	require.NoError(t, err)

	err = srv.DataLake.SetPolicy(context.Background(), assetPolicy, filters.Items)
	require.NoError(t, err)

	t.Run("resolve with single group", func(t *testing.T) {
		rp, err := srv.Resolve(context.Background(), &policy.ResolveReq{
			PolicyMrn:    "asset1",
			AssetFilters: []*explorer.Mquery{{Mql: "true"}},
		})
		require.NoError(t, err)
		require.NotNil(t, rp)
		require.Len(t, rp.ExecutionJob.Queries, 2)
	})

	t.Run("resolve with end dates", func(t *testing.T) {
		assetPolicy, err := srv.GetPolicy(context.Background(), &policy.Mrn{Mrn: "asset1"})
		require.NoError(t, err)
		m, err := mrn.NewChildMRN(b.OwnerMrn, explorer.MRN_RESOURCE_QUERY, "check2")
		require.NoError(t, err)

		// Add a group with an end date in the future. This group deactivates a check
		assetPolicy.Groups = append(assetPolicy.Groups, &policy.PolicyGroup{
			Uid:     "not-expired",
			EndDate: time.Now().Add(time.Hour).Unix(),
			Checks: []*explorer.Mquery{
				{
					Mrn:    m.String(),
					Action: explorer.Action_DEACTIVATE,
					Impact: &explorer.Impact{
						Action: explorer.Action_DEACTIVATE,
					},
				},
			},
		})

		// Recompute the checksums so that the resolved policy is invalidated
		assetPolicy.InvalidateAllChecksums()
		assetPolicy.UpdateChecksums(context.Background(), srv.DataLake.GetRawPolicy, srv.DataLake.GetQuery, nil)

		// Set the asset policy
		err = srv.DataLake.SetPolicy(context.Background(), assetPolicy, filters.Items)
		require.NoError(t, err)

		rp, err := srv.Resolve(context.Background(), &policy.ResolveReq{
			PolicyMrn:    "asset1",
			AssetFilters: []*explorer.Mquery{{Mql: "true"}},
		})
		require.NoError(t, err)
		require.NotNil(t, rp)
		require.Len(t, rp.ExecutionJob.Queries, 1)

		// Set the end date of the group to the past. This group deactivates a check,
		// but it should not be taken into account because it is expired
		assetPolicy.Groups[1].EndDate = time.Now().Add(-time.Hour).Unix()

		// Recompute the checksums so that the resolved policy is invalidated
		assetPolicy.InvalidateAllChecksums()
		assetPolicy.UpdateChecksums(context.Background(), srv.DataLake.GetRawPolicy, srv.DataLake.GetQuery, nil)

		// Set the asset policy
		err = srv.DataLake.SetPolicy(context.Background(), assetPolicy, filters.Items)
		require.NoError(t, err)

		rp, err = srv.Resolve(context.Background(), &policy.ResolveReq{
			PolicyMrn:    "asset1",
			AssetFilters: []*explorer.Mquery{{Mql: "true"}},
		})
		require.NoError(t, err)
		require.NotNil(t, rp)
		require.Len(t, rp.ExecutionJob.Queries, 2)
	})
}
