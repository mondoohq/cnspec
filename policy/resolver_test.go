// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package policy_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v9/explorer"
	"go.mondoo.com/cnquery/v9/mrn"
	"go.mondoo.com/cnquery/v9/providers"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/testutils"
	"go.mondoo.com/cnspec/v9/internal/datalakes/inmemory"
	"go.mondoo.com/cnspec/v9/policy"
)

type testAsset struct {
	asset      string
	policies   []string
	frameworks []string
}

func parseBundle(t *testing.T, data string) *policy.Bundle {
	res, err := policy.BundleFromYAML([]byte(data))
	require.NoError(t, err)
	return res
}

func initResolver(t *testing.T, assets []*testAsset, bundles []*policy.Bundle) *policy.LocalServices {
	runtime := testutils.LinuxMock()
	_, srv, err := inmemory.NewServices(runtime, nil)
	require.NoError(t, err)

	for i := range bundles {
		bundle := bundles[i]
		_, err := srv.SetBundle(context.Background(), bundle)
		require.NoError(t, err)
	}

	for i := range assets {
		asset := assets[i]
		_, err := srv.Assign(context.Background(), &policy.PolicyAssignment{
			AssetMrn:      asset.asset,
			PolicyMrns:    asset.policies,
			FrameworkMrns: asset.frameworks,
		})
		require.NoError(t, err)
	}

	return srv
}

func policyMrn(uid string) string {
	return "//test.sth/policies/" + uid
}

func frameworkMrn(uid string) string {
	return "//test.sth/frameworks/" + uid
}

func controlMrn(uid string) string {
	return "//test.sth/controls/" + uid
}

func queryMrn(uid string) string {
	return "//test.sth/queries/" + uid
}

func TestResolve_EmptyPolicy(t *testing.T) {
	b := parseBundle(t, `
owner_mrn: //test.sth
policies:
- uid: policy1
`)

	srv := initResolver(t, []*testAsset{
		{asset: "asset1", policies: []string{policyMrn("policy1")}},
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
		{asset: "asset1", policies: []string{policyMrn("policy1")}},
	}, []*policy.Bundle{b})

	t.Run("resolve with correct filters", func(t *testing.T) {
		rp, err := srv.Resolve(context.Background(), &policy.ResolveReq{
			PolicyMrn:    policyMrn("policy1"),
			AssetFilters: []*explorer.Mquery{{Mql: "true"}},
		})
		require.NoError(t, err)
		require.NotNil(t, rp)
		require.Len(t, rp.ExecutionJob.Queries, 3)
		require.Len(t, rp.Filters, 1)
		require.Len(t, rp.CollectorJob.ReportingJobs, 3)

		qrIdToRj := map[string]*policy.ReportingJob{}
		for _, rj := range rp.CollectorJob.ReportingJobs {
			qrIdToRj[rj.QrId] = rj
		}
		// scoring queries report by code id
		require.NotNil(t, qrIdToRj[b.Queries[1].CodeId])
		// data queries report by mrn
		require.NotNil(t, qrIdToRj[queryMrn("query1")])

		require.Len(t, qrIdToRj[b.Queries[1].CodeId].Datapoints, 3)
		require.Len(t, qrIdToRj[queryMrn("query1")].Datapoints, 1)
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
      mql: asset.arch
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
      mql: asset.arch
`)

	srv := initResolver(t, []*testAsset{
		{asset: "asset1", policies: []string{policyMrn("policy-active"), policyMrn("policy-ignored")}},
	}, []*policy.Bundle{b})

	t.Run("resolve with ignored policy", func(t *testing.T) {
		rp, err := srv.Resolve(context.Background(), &policy.ResolveReq{
			PolicyMrn:    "//test.sth",
			AssetFilters: []*explorer.Mquery{{Mql: "true"}},
		})
		require.NoError(t, err)
		require.NotNil(t, rp)
		require.Len(t, rp.CollectorJob.ReportingJobs, 5)
		ignoreJob := rp.CollectorJob.ReportingJobs["lTbmPQz/DwA="]
		require.NotNil(t, ignoreJob)
		childJob := ignoreJob.ChildJobs["DmPNGpL6IXo="]
		require.NotNil(t, childJob)
		require.Equal(t, explorer.ScoringSystem_IGNORE_SCORE, childJob.Scoring)
	})
}

func TestResolve_DisabledQuery(t *testing.T) {
	b := parseBundle(t, `
owner_mrn: //test.sth
policies:
- uid: policy-1
  owner_mrn: //test.sth
  groups:
  - type: chapter
    filters: "true"
    checks:
    - uid: check1
      mql: 1 == 1
      action: 2
`)

	srv := initResolver(t, []*testAsset{
		{asset: "asset1", policies: []string{policyMrn("policy-1")}},
	}, []*policy.Bundle{b})

	rp, err := srv.Resolve(context.Background(), &policy.ResolveReq{
		PolicyMrn:    "asset1",
		AssetFilters: []*explorer.Mquery{{Mql: "true"}},
	})
	require.NoError(t, err)
	require.NotNil(t, rp)
	require.Len(t, rp.CollectorJob.ReportingJobs, 2)
	for _, rj := range rp.CollectorJob.ReportingJobs {
		if rj.Type == policy.ReportingJob_CHECK {
			require.Fail(t, "expected no check reporting job")
		}
	}
}

func TestResolve_IgnoredQuery(t *testing.T) {
	b := parseBundle(t, `
owner_mrn: //test.sth
policies:
- uid: policy-1
  owner_mrn: //test.sth
  groups:
  - type: chapter
    filters: "true"
    checks:
    - uid: check1
      mql: 1 == 1
- mrn: asset1
  owner_mrn: //test.sth
  groups:
  - policies:
    - uid: policy-1
  - checks:
    - uid: check1
      action: 4
`)

	_, srv, err := inmemory.NewServices(providers.DefaultRuntime(), nil)
	require.NoError(t, err)

	ctx := context.Background()
	_, err = srv.SetBundle(ctx, b)
	require.NoError(t, err)

	bundleMap, err := b.Compile(context.Background(), schema, nil)
	require.NoError(t, err)

	rp, err := srv.Resolve(context.Background(), &policy.ResolveReq{
		PolicyMrn:    "asset1",
		AssetFilters: []*explorer.Mquery{{Mql: "true"}},
	})

	require.NoError(t, err)
	require.NotNil(t, rp)
	require.Len(t, rp.CollectorJob.ReportingJobs, 3)

	mrnToQueryId := map[string]string{}
	for _, q := range bundleMap.Queries {
		mrnToQueryId[q.Mrn] = q.CodeId
	}

	rjTester := frameworkReportingJobTester{
		t:                     t,
		queryIdToReportingJob: map[string]*policy.ReportingJob{},
		rjIdToReportingJob:    rp.CollectorJob.ReportingJobs,
		rjIdToDatapointJob:    rp.CollectorJob.Datapoints,
		dataQueriesMrns:       map[string]struct{}{},
	}

	for _, rj := range rjTester.rjIdToReportingJob {
		_, ok := rjTester.queryIdToReportingJob[rj.QrId]
		require.False(t, ok)
		rjTester.queryIdToReportingJob[rj.QrId] = rj
	}

	queryRj := rjTester.queryIdToReportingJob[mrnToQueryId[queryMrn("check1")]]
	// we ensure that even though ignored, theres an RJ for the query
	require.NotNil(t, queryRj)
	parent := queryRj.Notify[0]
	parentRj := rjTester.rjIdToReportingJob[parent]
	require.NotNil(t, parentRj)
	require.Equal(t, explorer.ScoringSystem_IGNORE_SCORE, parentRj.ChildJobs[queryRj.Uuid].Scoring)
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

	_, srv, err := inmemory.NewServices(providers.DefaultRuntime(), nil)
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
		assetPolicy.UpdateChecksums(context.Background(), srv.DataLake.GetRawPolicy, srv.DataLake.GetQuery, nil, schema)

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
		assetPolicy.UpdateChecksums(context.Background(), srv.DataLake.GetRawPolicy, srv.DataLake.GetQuery, nil, schema)

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

func TestResolve_Frameworks(t *testing.T) {
	bundleStr := `
owner_mrn: //test.sth
policies:
- uid: policy1
  groups:
  - filters: "true"
    checks:
    - uid: check-fail
      mql: 1 == 2
    - uid: check-pass-1
      mql: 1 == 1
    - uid: check-pass-2
      mql: 2 == 2
    queries:
    - uid: active-query
      title: users
      mql: users
    - uid: active-query-2
      title: users length
      mql: users.length
- uid: policy-inactive
  groups:
  - filters: "false"
    checks:
    - uid: inactive-fail
      mql: 1 == 2
    - uid: inactive-pass
      mql: 1 == 1
    - uid: inactive-pass-2
      mql: 2 == 2
    queries:
    - uid: inactive-query
      title: users group
      mql: users { group}
frameworks:
- uid: framework1
  name: framework1
  groups:
  - title: group1
    controls:
    - uid: control1
      title: control1
    - uid: control2
      title: control2
    - uid: control3
      title: control3
    - uid: control4
      title: control4
    - uid: control5
      title: control5
- uid: framework2
  name: framework2
  groups:
  - title: group1
    controls:
    - uid: control1
      title: control1
    - uid: control2
      title: control2
- uid: parent-framework
  dependencies:
  - mrn: ` + frameworkMrn("framework1") + `

framework_maps:
- uid: framework-map1
  framework_owner:
    uid: framework1
  policy_dependencies:
  - uid: policy1
  controls:
  - uid: control1
    checks:
    - uid: check-pass-1
    queries:
    - uid: active-query
    - uid: active-query-2
  - uid: control2
    checks:
    - uid: check-pass-2
    - uid: check-fail
  - uid: control4
    controls:
    - uid: control1
- uid: framework-map2
  framework_owner:
    uid: framework1
  policy_dependencies:
  - uid: policy1
  controls:
  - uid: control4
    controls:
    - uid: control1  
  - uid: control5
    controls:
    - uid: control1  
`

	t.Run("resolve with correct filters", func(t *testing.T) {
		b := parseBundle(t, bundleStr)

		srv := initResolver(t, []*testAsset{
			{asset: "asset1", policies: []string{policyMrn("policy1"), policyMrn("policy-inactive")}, frameworks: []string{frameworkMrn("parent-framework")}},
		}, []*policy.Bundle{b})

		bundle, err := srv.GetBundle(context.Background(), &policy.Mrn{Mrn: "asset1"})
		require.NoError(t, err)

		bundleMap, err := bundle.Compile(context.Background(), schema, nil)
		require.NoError(t, err)

		mrnToQueryId := map[string]string{}
		for _, q := range bundleMap.Queries {
			mrnToQueryId[q.Mrn] = q.CodeId
		}

		rp, err := srv.Resolve(context.Background(), &policy.ResolveReq{
			PolicyMrn:    "asset1",
			AssetFilters: []*explorer.Mquery{{Mql: "true"}},
		})
		require.NoError(t, err)
		require.NotNil(t, rp)

		// Check that there are no duplicates in the reporting job's notify list
		for _, rj := range rp.CollectorJob.ReportingJobs {
			requireUnique(t, rj.Notify)
		}

		require.Len(t, rp.ExecutionJob.Queries, 5)

		rjTester := frameworkReportingJobTester{
			t:                     t,
			queryIdToReportingJob: map[string]*policy.ReportingJob{},
			rjIdToReportingJob:    rp.CollectorJob.ReportingJobs,
			rjIdToDatapointJob:    rp.CollectorJob.Datapoints,
			dataQueriesMrns:       map[string]struct{}{},
		}

		for _, p := range bundleMap.Policies {
			for _, g := range p.Groups {
				for _, q := range g.Queries {
					rjTester.dataQueriesMrns[q.Mrn] = struct{}{}
				}
			}
		}

		for _, rj := range rjTester.rjIdToReportingJob {
			_, ok := rjTester.queryIdToReportingJob[rj.QrId]
			require.False(t, ok)
			rjTester.queryIdToReportingJob[rj.QrId] = rj
		}

		// control3 had no checks, so it should not have a reporting job.
		// TODO: is that the desired behavior?
		require.Nil(t, rjTester.queryIdToReportingJob[controlMrn("control3")])
		rjTester.requireReportsTo(mrnToQueryId[queryMrn("check-pass-1")], queryMrn("check-pass-1"))
		rjTester.requireReportsTo(mrnToQueryId[queryMrn("check-pass-2")], queryMrn("check-pass-2"))
		rjTester.requireReportsTo(mrnToQueryId[queryMrn("check-fail")], queryMrn("check-fail"))

		queryJob1 := rjTester.queryIdToReportingJob[queryMrn("active-query")]
		require.Equal(t, 1, len(queryJob1.Datapoints))

		queryJob2 := rjTester.queryIdToReportingJob[queryMrn("active-query-2")]
		require.Equal(t, 1, len(queryJob2.Datapoints))

		// scoring queries
		rjTester.requireReportsTo(queryMrn("check-pass-1"), controlMrn("control1"))
		rjTester.requireReportsTo(queryMrn("check-pass-2"), controlMrn("control2"))
		rjTester.requireReportsTo(queryMrn("check-fail"), controlMrn("control2"))
		// note: data queries RJs are reporting by MRN, not code id
		rjTester.requireReportsTo(queryMrn("active-query"), controlMrn("control1"))
		rjTester.requireReportsTo(queryMrn("active-query-2"), controlMrn("control1"))

		rjTester.requireReportsTo(controlMrn("control1"), frameworkMrn("framework1"))
		rjTester.requireReportsTo(controlMrn("control1"), controlMrn("control4"))
		rjTester.requireReportsTo(controlMrn("control2"), frameworkMrn("framework1"))
		rjTester.requireReportsTo(controlMrn("control4"), frameworkMrn("framework1"))
		rjTester.requireReportsTo(controlMrn("control5"), frameworkMrn("framework1"))
		rjTester.requireReportsTo(frameworkMrn("framework1"), frameworkMrn("parent-framework"))
		rjTester.requireReportsTo(frameworkMrn("parent-framework"), "root")

		require.Nil(t, rjTester.queryIdToReportingJob[queryMrn("inactive-fail")])
		require.Nil(t, rjTester.queryIdToReportingJob[queryMrn("inactive-pass")])
		require.Nil(t, rjTester.queryIdToReportingJob[queryMrn("inactive-pass-2")])

		require.Nil(t, rjTester.queryIdToReportingJob[queryMrn("inactive-query")])
	})

	t.Run("test resolving with inactive data queries", func(t *testing.T) {
		// test that creating a bundle with inactive data queries  (where the packs/policies are inactive)
		// will still end up in a succesfully resolved policy for the asset
		bundleStr := `
owner_mrn: //test.sth
policies:
- uid: policy1
  groups:
  - filters: "true"
    queries:
    - uid: active-query
      title: users
      mql: users
- uid: policy-inactive
  groups:
  - filters: "false"
    queries:
    - uid: inactive-query
      title: users group
      mql: users { group}
frameworks:
- uid: framework1
  name: framework1
  groups:
  - title: group1
    controls:
    - uid: control1
      title: control1
    - uid: control2
      title: control2
- uid: parent-framework
  dependencies:
  - mrn: ` + frameworkMrn("framework1") + `

framework_maps:
- uid: framework-map1
  framework_owner:
    uid: framework1
  policy_dependencies:
  - uid: policy1
  - uid: policy-inactive
  controls:
  - uid: control1
    queries:
    - uid: active-query
  - uid: control2
    queries:
    - uid: inactive-query
`
		b := parseBundle(t, bundleStr)

		// we do not activate policy-inactive, which means that its query should not get executed
		srv := initResolver(t, []*testAsset{
			{asset: "asset1", policies: []string{policyMrn("policy1")}, frameworks: []string{frameworkMrn("parent-framework")}},
		}, []*policy.Bundle{b})

		bundle, err := srv.GetBundle(context.Background(), &policy.Mrn{Mrn: "asset1"})
		require.NoError(t, err)

		bundleMap, err := bundle.Compile(context.Background(), schema, nil)
		require.NoError(t, err)

		mrnToQueryId := map[string]string{}
		for _, q := range bundleMap.Queries {
			mrnToQueryId[q.Mrn] = q.CodeId
		}

		rp, err := srv.Resolve(context.Background(), &policy.ResolveReq{
			PolicyMrn:    "asset1",
			AssetFilters: []*explorer.Mquery{{Mql: "true"}},
		})
		require.NoError(t, err)
		require.NotNil(t, rp)

		// Check that there are no duplicates in the reporting job's notify list
		for _, rj := range rp.CollectorJob.ReportingJobs {
			requireUnique(t, rj.Notify)
		}

		require.Len(t, rp.ExecutionJob.Queries, 1)

		rjTester := frameworkReportingJobTester{
			t:                     t,
			queryIdToReportingJob: map[string]*policy.ReportingJob{},
			rjIdToReportingJob:    rp.CollectorJob.ReportingJobs,
			rjIdToDatapointJob:    rp.CollectorJob.Datapoints,
			dataQueriesMrns:       map[string]struct{}{},
		}

		for _, p := range bundleMap.Policies {
			for _, g := range p.Groups {
				for _, q := range g.Queries {
					rjTester.dataQueriesMrns[q.Mrn] = struct{}{}
				}
			}
		}

		for _, rj := range rjTester.rjIdToReportingJob {
			_, ok := rjTester.queryIdToReportingJob[rj.QrId]
			require.False(t, ok)
			rjTester.queryIdToReportingJob[rj.QrId] = rj
		}

		queryJob1 := rjTester.queryIdToReportingJob[queryMrn("active-query")]
		require.Equal(t, 1, len(queryJob1.Datapoints))

		// queries
		rjTester.requireReportsTo(queryMrn("active-query"), controlMrn("control1"))
		require.Nil(t, rjTester.queryIdToReportingJob[queryMrn("inactive-query")])

		rjTester.requireReportsTo(controlMrn("control1"), frameworkMrn("framework1"))
		// the data query here is disabled, control2 has no rj
		require.Nil(t, rjTester.queryIdToReportingJob[controlMrn("control2")])
		rjTester.requireReportsTo(frameworkMrn("framework1"), frameworkMrn("parent-framework"))
		rjTester.requireReportsTo(frameworkMrn("parent-framework"), "root")
	})

	t.Run("test resolving with non-matching data queries", func(t *testing.T) {
		// test that creating a bundle with active data queries that do not match the asset, based on the
		// policy asset filters, will still create a resolved policy for the asset
		bundleStr := `
owner_mrn: //test.sth
policies:
- uid: policy1
  groups:
  - filters: "false"
    queries:
    - uid: query-1
      title: users
      mql: users
- uid: policy2
  groups:
  - filters: "true"
    queries:
    - uid: query-2
      title: users length
      mql: users.length

frameworks:
- uid: framework1
  name: framework1
  groups:
  - title: group1
    controls:
    - uid: control1
      title: control1
- uid: parent-framework
  dependencies:
  - mrn: ` + frameworkMrn("framework1") + `

framework_maps:
- uid: framework-map1
  framework_owner:
    uid: framework1
  policy_dependencies:
  - uid: policy1
  - uid: policy2
  controls:
  - uid: control1
    queries:
    - uid: query-1
    - uid: query-2
`
		b := parseBundle(t, bundleStr)

		srv := initResolver(t, []*testAsset{
			{asset: "asset1", policies: []string{policyMrn("policy1"), policyMrn("policy2")}, frameworks: []string{frameworkMrn("parent-framework")}},
		}, []*policy.Bundle{b})

		bundle, err := srv.GetBundle(context.Background(), &policy.Mrn{Mrn: "asset1"})
		require.NoError(t, err)

		bundleMap, err := bundle.Compile(context.Background(), schema, nil)
		require.NoError(t, err)

		mrnToQueryId := map[string]string{}
		for _, q := range bundleMap.Queries {
			mrnToQueryId[q.Mrn] = q.CodeId
		}

		rp, err := srv.Resolve(context.Background(), &policy.ResolveReq{
			PolicyMrn:    "asset1",
			AssetFilters: []*explorer.Mquery{{Mql: "true"}},
		})
		require.NoError(t, err)
		require.NotNil(t, rp)

		// Check that there are no duplicates in the reporting job's notify list
		for _, rj := range rp.CollectorJob.ReportingJobs {
			requireUnique(t, rj.Notify)
		}

		require.Len(t, rp.ExecutionJob.Queries, 1)

		rjTester := frameworkReportingJobTester{
			t:                     t,
			queryIdToReportingJob: map[string]*policy.ReportingJob{},
			rjIdToReportingJob:    rp.CollectorJob.ReportingJobs,
			rjIdToDatapointJob:    rp.CollectorJob.Datapoints,
			dataQueriesMrns:       map[string]struct{}{},
		}

		for _, p := range bundleMap.Policies {
			for _, g := range p.Groups {
				for _, q := range g.Queries {
					rjTester.dataQueriesMrns[q.Mrn] = struct{}{}
				}
			}
		}

		for _, rj := range rjTester.rjIdToReportingJob {
			_, ok := rjTester.queryIdToReportingJob[rj.QrId]
			require.False(t, ok)
			rjTester.queryIdToReportingJob[rj.QrId] = rj
		}

		queryJob1 := rjTester.queryIdToReportingJob[queryMrn("query-2")]
		require.Equal(t, 1, len(queryJob1.Datapoints))

		rjTester.requireReportsTo(queryMrn("query-2"), controlMrn("control1"))
		// query-1 is part of the policy that does not match the asset (even though it's active)
		// there should be no rjs for it
		require.Nil(t, rjTester.queryIdToReportingJob[queryMrn("query-1")])
		rjTester.requireReportsTo(controlMrn("control1"), frameworkMrn("framework1"))
		rjTester.requireReportsTo(frameworkMrn("framework1"), frameworkMrn("parent-framework"))
		rjTester.requireReportsTo(frameworkMrn("parent-framework"), "root")
	})

	t.Run("test checksumming", func(t *testing.T) {
		bInitial := parseBundle(t, bundleStr)

		srv := initResolver(t, []*testAsset{
			{asset: "asset1", policies: []string{policyMrn("policy1")}, frameworks: []string{frameworkMrn("parent-framework")}},
		}, []*policy.Bundle{bInitial})

		rpInitial, err := srv.Resolve(context.Background(), &policy.ResolveReq{
			PolicyMrn:    "asset1",
			AssetFilters: []*explorer.Mquery{{Mql: "true"}},
		})
		require.NoError(t, err)
		require.NotNil(t, rpInitial)

		bFrameworkUpdate := parseBundle(t, bundleStr)
		bFrameworkUpdate.Frameworks[0].Groups[0].Controls = bFrameworkUpdate.Frameworks[0].Groups[0].Controls[:2]

		srv = initResolver(t, []*testAsset{
			{asset: "asset1", policies: []string{policyMrn("policy1")}, frameworks: []string{frameworkMrn("parent-framework")}},
		}, []*policy.Bundle{bFrameworkUpdate})

		rpFrameworkUpdate, err := srv.Resolve(context.Background(), &policy.ResolveReq{
			PolicyMrn:    "asset1",
			AssetFilters: []*explorer.Mquery{{Mql: "true"}},
		})
		require.NoError(t, err)
		require.NotNil(t, rpFrameworkUpdate)

		require.NotEqual(t, rpInitial.GraphExecutionChecksum, rpFrameworkUpdate.GraphExecutionChecksum)
	})
}

type frameworkReportingJobTester struct {
	t                     *testing.T
	queryIdToReportingJob map[string]*policy.ReportingJob
	rjIdToDatapointJob    map[string]*policy.DataQueryInfo
	rjIdToReportingJob    map[string]*policy.ReportingJob
	dataQueriesMrns       map[string]struct{}
}

func isFramework(queryId string) bool {
	return strings.Contains(queryId, "/frameworks/")
}

func isControl(queryId string) bool {
	return strings.Contains(queryId, "/controls/")
}

func isPolicy(queryId string) bool {
	return strings.Contains(queryId, "/policies/")
}

func (tester *frameworkReportingJobTester) requireReportsTo(childQueryId string, parentQueryId string) {
	tester.t.Helper()

	childRj, ok := tester.queryIdToReportingJob[childQueryId]
	require.True(tester.t, ok)

	parentRj, ok := tester.queryIdToReportingJob[parentQueryId]
	require.True(tester.t, ok)

	require.Contains(tester.t, parentRj.ChildJobs, childRj.Uuid)
	require.Contains(tester.t, childRj.Notify, parentRj.Uuid)

	if isFramework(parentQueryId) {
		require.Equal(tester.t, policy.ReportingJob_FRAMEWORK, parentRj.Type)
		require.Equal(tester.t, explorer.ScoringSystem_AVERAGE, parentRj.ScoringSystem)
	} else if isControl(parentQueryId) {
		require.Equal(tester.t, policy.ReportingJob_CONTROL, parentRj.Type)
	} else if isPolicy(parentQueryId) || parentQueryId == "root" {
		require.Equal(tester.t, policy.ReportingJob_POLICY, parentRj.Type)
		// The root/asset reporting job is not a framework, but a policy
		childImpact := parentRj.ChildJobs[childRj.Uuid]
		require.Equal(tester.t, explorer.ScoringSystem_IGNORE_SCORE, childImpact.Scoring)
	} else {
		require.Equal(tester.t, policy.ReportingJob_CHECK, parentRj.Type)
	}

	if isControl(childQueryId) {
		require.Equal(tester.t, policy.ReportingJob_CONTROL, childRj.Type)
	} else if isFramework(childQueryId) {
		require.Equal(tester.t, policy.ReportingJob_FRAMEWORK, childRj.Type)
		require.Equal(tester.t, explorer.ScoringSystem_AVERAGE, childRj.ScoringSystem)
	} else if isPolicy(childQueryId) {
		require.Equal(tester.t, policy.ReportingJob_POLICY, childRj.Type)
	} else {
		_, isData := tester.dataQueriesMrns[childQueryId]
		if isData {
			require.Equal(tester.t, policy.ReportingJob_DATA_QUERY, childRj.Type)
		} else {
			require.Equal(tester.t, policy.ReportingJob_CHECK, childRj.Type)
		}
	}
}

func TestResolve_CheckValidUntil(t *testing.T) {
	stillValid := policy.CheckValidUntil(time.Now().Unix(), "test123")
	require.False(t, stillValid)
	stillValid = policy.CheckValidUntil(time.Now().Add(time.Hour*1).Unix(), "test123")
	require.True(t, stillValid)
	// forever
	stillValid = policy.CheckValidUntil(0, "test123")
	require.True(t, stillValid)
	// expired
	stillValid = policy.CheckValidUntil(time.Now().Add(-time.Hour*1).Unix(), "test123")
	require.False(t, stillValid)
}

func TestResolve_Exceptions(t *testing.T) {
	bundleString := `
owner_mrn: //test.sth
policies:
- uid: ssh-policy
  name: SSH Policy
  groups:
  - filters: "true"
    checks:
    - uid: sshd-ciphers-01
      title: Prevent weaker CBC ciphers from being used
      mql: sshd.config.ciphers.none( /cbc/ )
      impact: 60
    - uid: sshd-ciphers-02
      title: Do not allow ciphers with few bits
      mql: sshd.config.ciphers.none( /128/ )
      impact: 60
    - uid: sshd-config-permissions
      title: SSH config editing should be limited to admins
      mql: sshd.config.file.permissions.mode == 0644
      impact: 100

frameworks:
- uid: mondoo-ucf
  mrn: //test.sth/framework/mondoo-ucf
  name: Unified Compliance Framework
  groups:
  - title: System hardening
    controls:
    - uid: mondoo-ucf-01
      title: Only use strong ciphers
    - uid: mondoo-ucf-02
      title: Limit access to system configuration
    - uid: mondoo-ucf-03
      title: Only use ciphers with sufficient bits
  - title: exception-1
    type: 4
    controls:
    - uid: mondoo-ucf-02

framework_maps:
    - uid: compliance-to-ssh-policy
      mrn: //test.sth/framework/compliance-to-ssh-policy
      framework_owner:
        uid: mondoo-ucf
      policy_dependencies:
      - uid: ssh-policy
      controls:
      - uid: mondoo-ucf-01
        checks:
        - uid: sshd-ciphers-01
        - uid: sshd-ciphers-02
      - uid: mondoo-ucf-02
        checks:
        - uid: sshd-config-permissions
      - uid: mondoo-ucf-03
        checks:
        - uid: sshd-ciphers-02
`

	_, srv, err := inmemory.NewServices(providers.DefaultRuntime(), nil)
	require.NoError(t, err)

	t.Run("resolve with ignored control", func(t *testing.T) {
		b := parseBundle(t, bundleString)

		srv = initResolver(t, []*testAsset{
			{
				asset:      "asset1",
				policies:   []string{policyMrn("ssh-policy")},
				frameworks: []string{"//test.sth/framework/mondoo-ucf"},
			},
		}, []*policy.Bundle{b})

		rp, err := srv.Resolve(context.Background(), &policy.ResolveReq{
			PolicyMrn:    "asset1",
			AssetFilters: []*explorer.Mquery{{Mql: "true"}},
		})
		require.NoError(t, err)
		require.NotNil(t, rp)
		require.Len(t, rp.CollectorJob.ReportingJobs, 12)
		var frameworkJob *policy.ReportingJob
		for _, rj := range rp.CollectorJob.ReportingJobs {
			if rj.QrId == "//test.sth/framework/mondoo-ucf" {
				frameworkJob = rj
				break
			}
		}
		require.NotNil(t, frameworkJob)
		require.Equal(t, frameworkJob.Type, policy.ReportingJob_FRAMEWORK)
		var childJob *explorer.Impact
		for uuid, j := range frameworkJob.ChildJobs {
			if rp.CollectorJob.ReportingJobs[uuid].QrId == "//test.sth/controls/mondoo-ucf-02" {
				childJob = j
				break
			}
		}
		require.NotNil(t, childJob)
		require.Equal(t, explorer.ScoringSystem_IGNORE_SCORE, childJob.Scoring)
		require.Len(t, frameworkJob.ChildJobs, 3)
	})

	t.Run("resolve with ignored control and validUntil", func(t *testing.T) {
		b := parseBundle(t, bundleString)
		b.Frameworks[0].Groups[1].EndDate = time.Now().Add(time.Hour).Unix()

		srv = initResolver(t, []*testAsset{
			{
				asset:      "asset1",
				policies:   []string{policyMrn("ssh-policy")},
				frameworks: []string{"//test.sth/framework/mondoo-ucf"},
			},
		}, []*policy.Bundle{b})

		rp, err := srv.Resolve(context.Background(), &policy.ResolveReq{
			PolicyMrn:    "asset1",
			AssetFilters: []*explorer.Mquery{{Mql: "true"}},
		})
		require.NoError(t, err)
		require.NotNil(t, rp)
		require.Len(t, rp.CollectorJob.ReportingJobs, 12)
		var frameworkJob *policy.ReportingJob
		for _, rj := range rp.CollectorJob.ReportingJobs {
			if rj.QrId == "//test.sth/framework/mondoo-ucf" {
				frameworkJob = rj
				break
			}
		}
		require.Equal(t, frameworkJob.Type, policy.ReportingJob_FRAMEWORK)
		var childJob *explorer.Impact
		for uuid, j := range frameworkJob.ChildJobs {
			if rp.CollectorJob.ReportingJobs[uuid].QrId == "//test.sth/controls/mondoo-ucf-02" {
				childJob = j
				break
			}
		}
		require.Equal(t, explorer.ScoringSystem_IGNORE_SCORE, childJob.Scoring)
		require.Len(t, frameworkJob.ChildJobs, 3)
	})

	t.Run("resolve with expired validUntil", func(t *testing.T) {
		b := parseBundle(t, bundleString)
		b.Frameworks[0].Groups[1].EndDate = time.Now().Add(-time.Hour).Unix()

		srv = initResolver(t, []*testAsset{
			{
				asset:      "asset1",
				policies:   []string{policyMrn("ssh-policy")},
				frameworks: []string{"//test.sth/framework/mondoo-ucf"},
			},
		}, []*policy.Bundle{b})

		rp, err := srv.Resolve(context.Background(), &policy.ResolveReq{
			PolicyMrn:    "asset1",
			AssetFilters: []*explorer.Mquery{{Mql: "true"}},
		})
		require.NoError(t, err)
		require.NotNil(t, rp)
		require.Len(t, rp.CollectorJob.ReportingJobs, 12)
		var frameworkJob *policy.ReportingJob
		for _, rj := range rp.CollectorJob.ReportingJobs {
			if rj.QrId == "//test.sth/framework/mondoo-ucf" {
				frameworkJob = rj
				break
			}
		}
		require.Equal(t, frameworkJob.Type, policy.ReportingJob_FRAMEWORK)
		var childJob *explorer.Impact
		for uuid, j := range frameworkJob.ChildJobs {
			if rp.CollectorJob.ReportingJobs[uuid].QrId == "//test.sth/controls/mondoo-ucf-02" {
				childJob = j
				break
			}
		}
		require.Equal(t, explorer.ScoringSystem_SCORING_UNSPECIFIED, childJob.Scoring)
		require.Len(t, frameworkJob.ChildJobs, 3)
	})

	t.Run("resolve with disabled control", func(t *testing.T) {
		b := parseBundle(t, bundleString)
		b.Frameworks = append(b.Frameworks, &policy.Framework{
			Mrn: "//test.sth/framework/test",
			Dependencies: []*policy.FrameworkRef{
				{
					Mrn:    b.Frameworks[0].Mrn,
					Action: explorer.Action_ACTIVATE,
				},
			},
			Groups: []*policy.FrameworkGroup{
				{
					Uid:  "test",
					Type: policy.GroupType_DISABLE,
					Controls: []*policy.Control{
						{Uid: b.Frameworks[0].Groups[0].Controls[0].Uid},
					},
				},
			},
		})

		srv = initResolver(t, []*testAsset{
			{
				asset:      "asset1",
				policies:   []string{policyMrn("ssh-policy")},
				frameworks: []string{"//test.sth/framework/mondoo-ucf", "//test.sth/framework/test"},
			},
		}, []*policy.Bundle{b})

		rp, err := srv.Resolve(context.Background(), &policy.ResolveReq{
			PolicyMrn:    "asset1",
			AssetFilters: []*explorer.Mquery{{Mql: "true"}},
		})
		require.NoError(t, err)
		require.NotNil(t, rp)
		require.Len(t, rp.CollectorJob.ReportingJobs, 12)
		var frameworkJob *policy.ReportingJob
		for _, rj := range rp.CollectorJob.ReportingJobs {
			if rj.QrId == "//test.sth/framework/mondoo-ucf" {
				frameworkJob = rj
				break
			}
		}
		require.NotNil(t, frameworkJob)
		require.Equal(t, frameworkJob.Type, policy.ReportingJob_FRAMEWORK)
		require.Len(t, frameworkJob.ChildJobs, 2)
	})

	t.Run("resolve with out of scope control", func(t *testing.T) {
		b := parseBundle(t, bundleString)
		b.Frameworks = append(b.Frameworks, &policy.Framework{
			Mrn: "//test.sth/framework/test",
			Dependencies: []*policy.FrameworkRef{
				{
					Mrn:    b.Frameworks[0].Mrn,
					Action: explorer.Action_ACTIVATE,
				},
			},
			Groups: []*policy.FrameworkGroup{
				{
					Uid:  "test",
					Type: policy.GroupType_OUT_OF_SCOPE,
					Controls: []*policy.Control{
						{Uid: b.Frameworks[0].Groups[0].Controls[0].Uid},
					},
				},
			},
		})

		srv = initResolver(t, []*testAsset{
			{
				asset:      "asset1",
				policies:   []string{policyMrn("ssh-policy")},
				frameworks: []string{"//test.sth/framework/mondoo-ucf", "//test.sth/framework/test"},
			},
		}, []*policy.Bundle{b})

		rp, err := srv.Resolve(context.Background(), &policy.ResolveReq{
			PolicyMrn:    "asset1",
			AssetFilters: []*explorer.Mquery{{Mql: "true"}},
		})
		require.NoError(t, err)
		require.NotNil(t, rp)
		require.Len(t, rp.CollectorJob.ReportingJobs, 12)
		var frameworkJob *policy.ReportingJob
		for _, rj := range rp.CollectorJob.ReportingJobs {
			if rj.QrId == "//test.sth/framework/mondoo-ucf" {
				frameworkJob = rj
				break
			}
		}
		require.NotNil(t, frameworkJob)
		require.Equal(t, frameworkJob.Type, policy.ReportingJob_FRAMEWORK)
		require.Len(t, frameworkJob.ChildJobs, 2)
	})

	t.Run("resolve with rejected disable exception", func(t *testing.T) {
		b := parseBundle(t, bundleString)
		b.Frameworks[0].Groups[1].Type = policy.GroupType_DISABLE
		b.Frameworks[0].Groups[1].ReviewStatus = policy.ReviewStatus_REJECTED

		srv = initResolver(t, []*testAsset{
			{
				asset:      "asset1",
				policies:   []string{policyMrn("ssh-policy")},
				frameworks: []string{"//test.sth/framework/mondoo-ucf"},
			},
		}, []*policy.Bundle{b})

		rp, err := srv.Resolve(context.Background(), &policy.ResolveReq{
			PolicyMrn:    "asset1",
			AssetFilters: []*explorer.Mquery{{Mql: "true"}},
		})
		require.NoError(t, err)
		require.NotNil(t, rp)
		require.Len(t, rp.CollectorJob.ReportingJobs, 12)
		var frameworkJob *policy.ReportingJob
		for _, rj := range rp.CollectorJob.ReportingJobs {
			if rj.QrId == "//test.sth/framework/mondoo-ucf" {
				frameworkJob = rj
				break
			}
		}
		require.Equal(t, frameworkJob.Type, policy.ReportingJob_FRAMEWORK)
		require.Len(t, frameworkJob.ChildJobs, 3)
	})
}

func requireUnique(t *testing.T, items []string) {
	seen := make(map[string]bool)
	for _, item := range items {
		if seen[item] {
			t.Errorf("duplicate item found: %s", item)
		}
		seen[item] = true
	}
}
