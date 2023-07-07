package policy_test

import (
	"context"
	"strings"
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
		{asset: "asset1", policies: []string{policyMrn("policy-active"), policyMrn("policy-ignored")}},
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

func TestResolve_Frameworks(t *testing.T) {
	b := parseBundle(t, `
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
- uid: parent-framework
  dependencies:
  - mrn: `+frameworkMrn("framework1")+`

framework_maps:
- uid: framework-map1
  framework_owner: framework1
  policy_dependencies:
  - uid: policy1
  controls:
  - uid: control1
    checks:
    - uid: check-pass-1
  - uid: control2
    checks:
    - uid: check-pass-2
    - uid: check-fail
`)

	srv := initResolver(t, []*testAsset{
		{asset: "asset1", policies: []string{policyMrn("policy1")}, frameworks: []string{frameworkMrn("parent-framework")}},
	}, []*policy.Bundle{b})

	t.Run("resolve with correct filters", func(t *testing.T) {
		bundle, err := srv.GetBundle(context.Background(), &policy.Mrn{Mrn: "asset1"})
		require.NoError(t, err)

		bundleMap, err := bundle.Compile(context.Background(), nil)
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

		require.Len(t, rp.ExecutionJob.Queries, 3)

		rjTester := frameworkReportingJobTester{
			t:                     t,
			queryIdToReportingJob: map[string]*policy.ReportingJob{},
			rjIdToReportingJob:    rp.CollectorJob.ReportingJobs,
		}

		for _, rj := range rjTester.rjIdToReportingJob {
			rjTester.queryIdToReportingJob[rj.QrId] = rj
		}

		// control3 had no checks, so it should not have a reporting job.
		// TODO: is that the desired behavior?
		require.Nil(t, rjTester.queryIdToReportingJob[controlMrn("control3")])
		rjTester.requireReportsTo(mrnToQueryId[queryMrn("check-pass-1")], controlMrn("control1"))
		rjTester.requireReportsTo(mrnToQueryId[queryMrn("check-pass-2")], controlMrn("control2"))
		rjTester.requireReportsTo(mrnToQueryId[queryMrn("check-fail")], controlMrn("control2"))
		rjTester.requireReportsTo(controlMrn("control1"), frameworkMrn("framework1"))
		rjTester.requireReportsTo(controlMrn("control2"), frameworkMrn("framework1"))
		rjTester.requireReportsTo(frameworkMrn("framework1"), frameworkMrn("parent-framework"))
		rjTester.requireReportsTo(frameworkMrn("parent-framework"), "root")
	})
}

type frameworkReportingJobTester struct {
	t                     *testing.T
	queryIdToReportingJob map[string]*policy.ReportingJob
	rjIdToReportingJob    map[string]*policy.ReportingJob
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
	} else if isPolicy(childQueryId) {
		require.Equal(tester.t, policy.ReportingJob_POLICY, childRj.Type)
	} else {
		require.Equal(tester.t, policy.ReportingJob_CHECK, childRj.Type)
	}
}

func TestResolve_CheckValidUntil(t *testing.T) {
	stillValid := policy.CheckValidUntil(time.Now().Format(time.RFC3339), "test123")
	require.False(t, stillValid)
	stillValid = policy.CheckValidUntil(time.Now().Add(time.Hour*1).Format(time.RFC3339), "test123")
	require.True(t, stillValid)
	// wrong format as input, should return false
	stillValid = policy.CheckValidUntil(time.Now().Format(time.RFC1123), "test123")
	require.False(t, stillValid)
}

func TestResolve_Exceptions(t *testing.T) {
	b := parseBundle(t, `
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
      framework_owner: mondoo-ucf
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
`)

	_, srv, err := inmemory.NewServices(nil)
	require.NoError(t, err)

	t.Run("resolve with ignored control", func(t *testing.T) {
		_, err = srv.SetBundle(context.Background(), b)
		require.NoError(t, err)

		_, err = srv.Assign(context.Background(), &policy.PolicyAssignment{
			AssetMrn:      "asset1",
			PolicyMrns:    []string{policyMrn("ssh-policy")},
			FrameworkMrns: []string{"//test.sth/framework/mondoo-ucf"},
		})
		require.NoError(t, err)

		filters, err := srv.GetPolicyFilters(context.Background(), &policy.Mrn{Mrn: "asset1"})
		require.NoError(t, err)
		assetPolicy, err := srv.GetPolicy(context.Background(), &policy.Mrn{Mrn: "asset1"})
		require.NoError(t, err)

		err = srv.DataLake.SetPolicy(context.Background(), assetPolicy, filters.Items)
		require.NoError(t, err)

		rp, err := srv.Resolve(context.Background(), &policy.ResolveReq{
			PolicyMrn:    "asset1",
			AssetFilters: []*explorer.Mquery{{Mql: "true"}},
		})
		require.NoError(t, err)
		require.NotNil(t, rp)
		require.Len(t, rp.CollectorJob.ReportingJobs, 9)
		frameworkJob := rp.CollectorJob.ReportingJobs["NkTWThBaLqc="]
		require.Equal(t, frameworkJob.Type, policy.ReportingJob_FRAMEWORK)
		require.Equal(t, explorer.ScoringSystem_IGNORE_SCORE, frameworkJob.ChildJobs["Bf7vb8/h2YM="].Scoring)
		require.Len(t, frameworkJob.ChildJobs, 3)
	})

	t.Run("resolve with ignored control and validUntil", func(t *testing.T) {
		b.Frameworks[0].Groups[1].EndDate = time.Now().Add(time.Hour).Unix()
		_, err = srv.SetBundle(context.Background(), b)
		require.NoError(t, err)

		_, err = srv.Assign(context.Background(), &policy.PolicyAssignment{
			AssetMrn:      "asset1",
			PolicyMrns:    []string{policyMrn("ssh-policy")},
			FrameworkMrns: []string{"//test.sth/framework/mondoo-ucf"},
		})
		require.NoError(t, err)

		filters, err := srv.GetPolicyFilters(context.Background(), &policy.Mrn{Mrn: "asset1"})
		require.NoError(t, err)
		assetPolicy, err := srv.GetPolicy(context.Background(), &policy.Mrn{Mrn: "asset1"})
		require.NoError(t, err)

		err = srv.DataLake.SetPolicy(context.Background(), assetPolicy, filters.Items)
		require.NoError(t, err)

		rp, err := srv.Resolve(context.Background(), &policy.ResolveReq{
			PolicyMrn:    "asset1",
			AssetFilters: []*explorer.Mquery{{Mql: "true"}},
		})
		require.NoError(t, err)
		require.NotNil(t, rp)
		require.Len(t, rp.CollectorJob.ReportingJobs, 9)
		frameworkJob := rp.CollectorJob.ReportingJobs["NkTWThBaLqc="]
		require.Equal(t, frameworkJob.Type, policy.ReportingJob_FRAMEWORK)
		require.Equal(t, explorer.ScoringSystem_IGNORE_SCORE, frameworkJob.ChildJobs["Bf7vb8/h2YM="].Scoring)
		require.Len(t, frameworkJob.ChildJobs, 3)
	})

	t.Run("resolve with expired validUntil", func(t *testing.T) {
		b.Frameworks[0].Groups[1].EndDate = time.Now().Add(-time.Hour).Unix()
		_, err = srv.SetBundle(context.Background(), b)
		require.NoError(t, err)

		_, err = srv.Assign(context.Background(), &policy.PolicyAssignment{
			AssetMrn:      "asset1",
			PolicyMrns:    []string{policyMrn("ssh-policy")},
			FrameworkMrns: []string{"//test.sth/framework/mondoo-ucf"},
		})
		require.NoError(t, err)

		filters, err := srv.GetPolicyFilters(context.Background(), &policy.Mrn{Mrn: "asset1"})
		require.NoError(t, err)
		assetPolicy, err := srv.GetPolicy(context.Background(), &policy.Mrn{Mrn: "asset1"})
		require.NoError(t, err)

		err = srv.DataLake.SetPolicy(context.Background(), assetPolicy, filters.Items)
		require.NoError(t, err)

		rp, err := srv.Resolve(context.Background(), &policy.ResolveReq{
			PolicyMrn:    "asset1",
			AssetFilters: []*explorer.Mquery{{Mql: "true"}},
		})
		require.NoError(t, err)
		require.NotNil(t, rp)
		require.Len(t, rp.CollectorJob.ReportingJobs, 9)
		frameworkJob := rp.CollectorJob.ReportingJobs["NkTWThBaLqc="]
		require.Equal(t, frameworkJob.Type, policy.ReportingJob_FRAMEWORK)
		require.Equal(t, explorer.ScoringSystem_SCORING_UNSPECIFIED, frameworkJob.ChildJobs["Bf7vb8/h2YM="].Scoring)
		require.Len(t, frameworkJob.ChildJobs, 3)
	})

	t.Run("resolve with disabled control", func(t *testing.T) {
		b.Frameworks[0].Groups[1].Type = 5
		_, err = srv.SetBundle(context.Background(), b)
		require.NoError(t, err)

		_, err = srv.Assign(context.Background(), &policy.PolicyAssignment{
			AssetMrn:      "asset1",
			PolicyMrns:    []string{policyMrn("ssh-policy")},
			FrameworkMrns: []string{"//test.sth/framework/mondoo-ucf"},
		})
		require.NoError(t, err)

		filters, err := srv.GetPolicyFilters(context.Background(), &policy.Mrn{Mrn: "asset1"})
		require.NoError(t, err)
		assetPolicy, err := srv.GetPolicy(context.Background(), &policy.Mrn{Mrn: "asset1"})
		require.NoError(t, err)

		err = srv.DataLake.SetPolicy(context.Background(), assetPolicy, filters.Items)
		require.NoError(t, err)

		rp, err := srv.Resolve(context.Background(), &policy.ResolveReq{
			PolicyMrn:    "asset1",
			AssetFilters: []*explorer.Mquery{{Mql: "true"}},
		})
		require.NoError(t, err)
		require.NotNil(t, rp)
		require.Len(t, rp.CollectorJob.ReportingJobs, 8)
		frameworkJob := rp.CollectorJob.ReportingJobs["ym15u8kWL9c="]
		require.Equal(t, frameworkJob.Type, policy.ReportingJob_FRAMEWORK)
		require.Len(t, frameworkJob.ChildJobs, 2)
	})

	t.Run("resolve with rejected disable exception", func(t *testing.T) {
		b.Frameworks[0].Groups[1].Type = 5
		b.Frameworks[0].Groups[1].Rejected = true
		_, err = srv.SetBundle(context.Background(), b)
		require.NoError(t, err)

		_, err = srv.Assign(context.Background(), &policy.PolicyAssignment{
			AssetMrn:      "asset1",
			PolicyMrns:    []string{policyMrn("ssh-policy")},
			FrameworkMrns: []string{"//test.sth/framework/mondoo-ucf"},
		})
		require.NoError(t, err)

		filters, err := srv.GetPolicyFilters(context.Background(), &policy.Mrn{Mrn: "asset1"})
		require.NoError(t, err)
		assetPolicy, err := srv.GetPolicy(context.Background(), &policy.Mrn{Mrn: "asset1"})
		require.NoError(t, err)

		err = srv.DataLake.SetPolicy(context.Background(), assetPolicy, filters.Items)
		require.NoError(t, err)

		rp, err := srv.Resolve(context.Background(), &policy.ResolveReq{
			PolicyMrn:    "asset1",
			AssetFilters: []*explorer.Mquery{{Mql: "true"}},
		})
		require.NoError(t, err)
		require.NotNil(t, rp)
		require.Len(t, rp.CollectorJob.ReportingJobs, 9)
		frameworkJob := rp.CollectorJob.ReportingJobs["ym15u8kWL9c="]
		require.Equal(t, frameworkJob.Type, policy.ReportingJob_FRAMEWORK)
		require.Len(t, frameworkJob.ChildJobs, 3)
	})
}
