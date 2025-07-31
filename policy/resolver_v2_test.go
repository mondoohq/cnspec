// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package policy_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v11/explorer"
	"go.mondoo.com/cnquery/v11/mqlc"
	"go.mondoo.com/cnquery/v11/providers"
	"go.mondoo.com/cnspec/v11/internal/datalakes/inmemory"
	"go.mondoo.com/cnspec/v11/policy"
)

func collectQueriesFromRiskFactors(p *policy.Policy, query map[string]*explorer.Mquery) {
	for _, rf := range p.RiskFactors {
		for _, c := range rf.Checks {
			query[c.Mrn] = c
		}
	}
}

func newResolvedPolicyTester(bundle *policy.Bundle, conf mqlc.CompilerConfig) *resolvedPolicyTester {
	m := bundle.ToMap()
	for _, p := range m.Policies {
		collectQueriesFromRiskFactors(p, m.Queries)
	}

	return &resolvedPolicyTester{
		bundleMap: m,
		items:     []resolvedPolicyTesterItem{},
		conf:      conf,
	}
}

type resolvedPolicyTesterItem interface {
	testIt(t *testing.T, resolvedPolicy *policy.ResolvedPolicy)
}

type resolvedPolicyTester struct {
	bundleMap *policy.PolicyBundleMap
	items     []resolvedPolicyTesterItem
	conf      mqlc.CompilerConfig
}

func (r *resolvedPolicyTester) doTest(t *testing.T, rp *policy.ResolvedPolicy) {
	for _, item := range r.items {
		item.testIt(t, rp)
	}
}

func (r *resolvedPolicyTester) ExecutesQuery(mrn string) *resolvedPolicyTesterExecutesQueryBuilder {
	item := &resolvedPolicyTesterExecutesQueryBuilder{tester: r, mrn: mrn}
	r.items = append(r.items, item)
	return item
}

func (r *resolvedPolicyTester) DoesNotExecutesQuery(mrn string) {
	item := &resolvedPolicyTesterExecutesQueryBuilder{tester: r, mrn: mrn, doesNotExecute: true}
	r.items = append(r.items, item)
}

type resolvedPolicyTesterExecutesQueryBuilder struct {
	tester         *resolvedPolicyTester
	mrn            string
	datapoints     *[]string
	props          *map[string]string
	doesNotExecute bool
}

func (r *resolvedPolicyTesterExecutesQueryBuilder) WithProps(props map[string]string) *resolvedPolicyTesterExecutesQueryBuilder {
	r.props = &props
	return r
}

func (r *resolvedPolicyTesterExecutesQueryBuilder) testIt(t *testing.T, rp *policy.ResolvedPolicy) {
	q := r.tester.bundleMap.Queries[r.mrn]
	require.NotNilf(t, q, "query not found in bundle: %s", r.mrn)
	codeId := q.CodeId
	require.NotEmptyf(t, codeId, "query %s doesn't have code id", r.mrn)

	eq := rp.ExecutionJob.Queries[codeId]
	if r.doesNotExecute {
		require.Nil(t, eq, "query %s should not be executed", r.mrn)
		return
	}
	require.NotNilf(t, eq, "query %s not found in ExecutionJob", r.mrn)

	if r.datapoints != nil {
		require.ElementsMatchf(t, *r.datapoints, eq.Datapoints, "datapoints mismatch for query %q", r.mrn)
	}

	if r.props != nil {
		require.Lenf(t, eq.Properties, len(*r.props), "properties mismatch for query %q", r.mrn)
		for propName, mql := range *r.props {
			// Compile the property
			codeBundle, err := mqlc.Compile(mql, nil, r.tester.conf)
			require.NoErrorf(t, err, "failed to compile property %q for query %q", propName, r.mrn)
			propCodeId := codeBundle.CodeV2.Id
			require.NotEmptyf(t, propCodeId, "property %s doesn't have code id", propName)
			propEq := rp.ExecutionJob.Queries[propCodeId]
			require.NotNilf(t, propEq, "property %q not found in ExecutionJob with code id %q", propName, propCodeId)
			require.Lenf(t, propEq.Datapoints, 1, "property %q should have exactly one datapoint", propName)
			propDatapoint := propEq.Datapoints[0]
			require.Equalf(t, eq.Properties[propName], propDatapoint, "property %q value mismatch", propName)
		}
	}
}

type resolvedPolicyTesterReportingJobNotifiesBuilder struct {
	rjTester          *resolvedPolicyTesterReportingJobBuilder
	childMrn          string
	childMrnForCodeId string
	parent            string
	impact            *explorer.Impact
	impactSet         bool
}

func (r *resolvedPolicyTesterReportingJobNotifiesBuilder) WithImpact(impact *explorer.Impact) *resolvedPolicyTesterReportingJobNotifiesBuilder {
	r.impact = impact
	r.impactSet = true
	return r
}

func findReportingJobByQrId(rp *policy.ResolvedPolicy, qrId string) *policy.ReportingJob {
	for _, rj := range rp.CollectorJob.ReportingJobs {
		if rj.QrId == qrId {
			return rj
		}
	}
	return nil
}

func (r *resolvedPolicyTesterReportingJobNotifiesBuilder) testIt(t *testing.T, rp *policy.ResolvedPolicy) {
	var qrId string
	var extraInfo string
	mrnsMatchesQrId := false
	if r.childMrn != "" {
		qrId = r.childMrn
		mrnsMatchesQrId = true
	} else {
		q := r.rjTester.tester.bundleMap.Queries[r.childMrnForCodeId]
		require.NotNilf(t, q, "query not found in bundle: %s", r.childMrnForCodeId)
		require.NotEmptyf(t, q.CodeId, "query %s doesn't have code id", r.childMrnForCodeId)
		qrId = q.CodeId
		extraInfo = " (" + r.childMrnForCodeId + ")"
	}
	childRj := findReportingJobByQrId(rp, qrId)
	require.NotNilf(t, childRj, "child reporting job %s%s not found", qrId, extraInfo)

	if mrnsMatchesQrId {
		require.Equalf(t, []string{qrId}, childRj.Mrns, "child reporting job %s%s mrns mismatch", qrId, extraInfo)
	}

	parentRj := findReportingJobByQrId(rp, r.parent)
	require.NotNilf(t, parentRj, "parent reporting job %s not found", r.parent)
	require.Containsf(t, childRj.Notify, parentRj.Uuid, "child reporting job %s%s doesn't notify parent reporting job %s", qrId, extraInfo, r.parent)

	require.Containsf(t, parentRj.ChildJobs, childRj.Uuid, "parent reporting job %s doesn't have child reporting job %s%s", r.parent, qrId, extraInfo)
	if r.impactSet {
		require.EqualExportedValuesf(t, r.impact, parentRj.ChildJobs[childRj.Uuid], "impact mismatch for child reporting job %s%s", qrId, extraInfo)
	}

}

type resolvedPolicyTesterReportingJobBuilder struct {
	tester        *resolvedPolicyTester
	mrn           string
	mrnForCodeId  string
	typ           *policy.ReportingJob_Type
	scoringSystem *explorer.ScoringSystem
	notifies      []*resolvedPolicyTesterReportingJobNotifiesBuilder
	notifiesSet   bool
	doesNotExist  bool
}

func (r *resolvedPolicyTester) CodeIdReportingJobForMrn(mrn string) *resolvedPolicyTesterReportingJobBuilder {
	var item *resolvedPolicyTesterReportingJobBuilder
	for _, existing := range r.items {
		if existingItem, ok := existing.(*resolvedPolicyTesterReportingJobBuilder); ok && existingItem.mrnForCodeId == mrn {
			item = existingItem
			break
		}
	}
	if item == nil {
		item = &resolvedPolicyTesterReportingJobBuilder{tester: r, mrnForCodeId: mrn}
		r.items = append(r.items, item)
	}

	return item
}

func (r *resolvedPolicyTester) ReportingJobByMrn(mrn string) *resolvedPolicyTesterReportingJobBuilder {
	var item *resolvedPolicyTesterReportingJobBuilder
	for _, existing := range r.items {
		if existingItem, ok := existing.(*resolvedPolicyTesterReportingJobBuilder); ok && existingItem.mrn == mrn {
			item = existingItem
			break
		}
	}
	if item == nil {
		item = &resolvedPolicyTesterReportingJobBuilder{tester: r, mrn: mrn}
		r.items = append(r.items, item)
	}
	return item
}

func (r *resolvedPolicyTesterReportingJobBuilder) DoesNotExist() {
	r.doesNotExist = true
}

func (r *resolvedPolicyTesterReportingJobBuilder) WithType(typ policy.ReportingJob_Type) *resolvedPolicyTesterReportingJobBuilder {
	r.typ = &typ
	return r
}

func (r *resolvedPolicyTesterReportingJobBuilder) Notifies(qrId string) *resolvedPolicyTesterReportingJobNotifiesBuilder {
	n := &resolvedPolicyTesterReportingJobNotifiesBuilder{rjTester: r, childMrnForCodeId: r.mrnForCodeId, childMrn: r.mrn, parent: qrId}
	r.notifies = append(r.notifies, n)
	r.notifiesSet = true
	return n
}

func (r *resolvedPolicyTesterReportingJobBuilder) WithScoringSystem(scoringSystem explorer.ScoringSystem) *resolvedPolicyTesterReportingJobBuilder {
	r.scoringSystem = &scoringSystem
	return r
}

func (r *resolvedPolicyTesterReportingJobBuilder) testIt(t *testing.T, rp *policy.ResolvedPolicy) {
	var qrId string
	var extraInfo string
	if r.mrn != "" {
		qrId = r.mrn
	} else {
		q := r.tester.bundleMap.Queries[r.mrnForCodeId]
		require.NotNilf(t, q, "query not found in bundle: %s", r.mrnForCodeId)
		require.NotEmptyf(t, q.CodeId, "query %s doesn't have code id", r.mrnForCodeId)
		qrId = q.CodeId
		extraInfo = " (" + r.mrnForCodeId + ")"
	}

	rj := findReportingJobByQrId(rp, qrId)
	if r.doesNotExist {
		require.Nilf(t, rj, "reporting job %s%s should not exist", qrId, extraInfo)
		return
	}
	require.NotNilf(t, rj, "reporting job %s%s not found", qrId, extraInfo)

	if r.typ != nil {
		require.Equalf(t, *r.typ, rj.Type, "reporting job %s%s type mismatch", qrId, extraInfo)
	}

	if r.scoringSystem != nil {
		require.Equalf(t, *r.scoringSystem, rj.ScoringSystem, "reporting job %s%s scoring system mismatch", qrId, extraInfo)
	}

	if r.notifiesSet {
		for _, n := range r.notifies {
			n.testIt(t, rp)
		}
		require.Len(t, rj.Notify, len(r.notifies), "reporting job uuid=%s qrId=%s%s notify mismatch", rj.Uuid, qrId, extraInfo)
	}
}

func TestResolveV2_EmptyPolicy(t *testing.T) {
	ctx := context.Background()
	b := parseBundle(t, `
owner_mrn: //test.sth
policies:
- uid: policy1
`)

	srv := initResolver(t, []*testAsset{
		{asset: "asset1", policies: []string{policyMrn("policy1")}},
	}, []*policy.Bundle{b})

	t.Run("resolve w/o filters", func(t *testing.T) {
		_, err := srv.Resolve(ctx, &policy.ResolveReq{
			PolicyMrn: policyMrn("policy1"),
		})
		assert.EqualError(t, err, "rpc error: code = InvalidArgument desc = asset doesn't support any policies")
	})

	t.Run("resolve with empty filters", func(t *testing.T) {
		_, err := srv.Resolve(ctx, &policy.ResolveReq{
			PolicyMrn:    policyMrn("policy1"),
			AssetFilters: []*explorer.Mquery{{}},
		})
		assert.EqualError(t, err, "failed to compile query: failed to compile query '': query is not implemented ''")
	})

	t.Run("resolve with random filters", func(t *testing.T) {
		_, err := srv.Resolve(ctx, &policy.ResolveReq{
			PolicyMrn:    policyMrn("policy1"),
			AssetFilters: []*explorer.Mquery{{Mql: "true"}},
		})
		assert.EqualError(t, err,
			"rpc error: code = InvalidArgument desc = asset isn't supported by any policies\n"+
				"policies didn't provide any filters\n"+
				"asset supports: true\n")
	})
}

func TestResolveV2_SimplePolicy(t *testing.T) {
	ctx := context.Background()
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
		rp, err := srv.Resolve(ctx, &policy.ResolveReq{
			PolicyMrn:    policyMrn("policy1"),
			AssetFilters: []*explorer.Mquery{{Mql: "true"}},
		})
		require.NoError(t, err)
		require.NotNil(t, rp)
		require.Len(t, rp.ExecutionJob.Queries, 3)
		require.Len(t, rp.Filters, 1)
		require.Len(t, rp.CollectorJob.ReportingJobs, 5)

		rpTester := newResolvedPolicyTester(b, srv.NewCompilerConfig())
		rpTester.ExecutesQuery(queryMrn("query1"))
		rpTester.
			ExecutesQuery(queryMrn("check1")).
			WithProps(map[string]string{"name": `return "definitely not the asset name"`})
		rpTester.CodeIdReportingJobForMrn(queryMrn("check1")).Notifies(queryMrn("check1"))
		rpTester.CodeIdReportingJobForMrn(queryMrn("query1")).Notifies(queryMrn("query1"))
		rpTester.ReportingJobByMrn(queryMrn("check1")).Notifies("root")
		rpTester.ReportingJobByMrn(queryMrn("query1")).Notifies("root")

		rpTester.doTest(t, rp)
	})

	t.Run("resolve with many filters (one is correct)", func(t *testing.T) {
		rp, err := srv.Resolve(ctx, &policy.ResolveReq{
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
		_, err := srv.Resolve(ctx, &policy.ResolveReq{
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

func TestResolveV2_PolicyWithImpacts(t *testing.T) {
	// For impacts, we always find the worst impact specified for a query in a policy bundle.
	// All instances of the query use that impact
	ctx := context.Background()
	b := parseBundle(t, `
owner_mrn: //test.sth
policies:
- owner_mrn: //test.sth
  mrn: //test.sth
  groups:
  - policies:
    - uid: policy1
    - uid: policy2
      action: 4
- uid: policy1
  groups:
  - type: chapter
    filters: "true"
    checks:
    - uid: check1
    - uid: check2
      impact: 10
    - uid: check3
      impact: 60
    queries:
    - uid: query1
- uid: policy2
  groups:
  - type: chapter
    filters: "true"
    checks:
    - uid: check2
      impact: 5
    - uid: check3
      impact: 80
queries:
- uid: check1
  mql: asset.name == props.name
  props:
  - uid: name
    mql: return "definitely not the asset name"
- uid: check2
  mql: true == false
  impact: 70
- uid: check3
  mql: true == true
  impact: 9
- uid: query1
  mql: asset{*}
`)

	srv := initResolver(t, []*testAsset{
		{asset: "asset1", policies: []string{policyMrn("policy1"), policyMrn("policy2")}},
	}, []*policy.Bundle{b})

	t.Run("resolve with correct filters", func(t *testing.T) {
		rp, err := srv.Resolve(ctx, &policy.ResolveReq{
			PolicyMrn:    "//test.sth",
			AssetFilters: []*explorer.Mquery{{Mql: "true"}},
		})
		require.NoError(t, err)
		require.NotNil(t, rp)

		rpTester := newResolvedPolicyTester(b, srv.NewCompilerConfig())
		rpTester.ExecutesQuery(queryMrn("query1"))
		rpTester.
			ExecutesQuery(queryMrn("check1")).
			WithProps(map[string]string{"name": `return "definitely not the asset name"`})
		rpTester.ExecutesQuery(queryMrn("check2"))
		rpTester.CodeIdReportingJobForMrn(queryMrn("check1")).Notifies(queryMrn("check1"))
		rpTester.CodeIdReportingJobForMrn(queryMrn("check2")).Notifies(queryMrn("check2")).WithImpact(&explorer.Impact{Value: &explorer.ImpactValue{Value: 70}})
		rpTester.CodeIdReportingJobForMrn(queryMrn("check3")).Notifies(queryMrn("check3")).WithImpact(&explorer.Impact{Value: &explorer.ImpactValue{Value: 80}})
		rpTester.CodeIdReportingJobForMrn(queryMrn("query1")).Notifies(queryMrn("query1"))
		rpTester.ReportingJobByMrn(queryMrn("check1")).Notifies(policyMrn("policy1"))
		rpTester.ReportingJobByMrn(queryMrn("check2")).Notifies(policyMrn("policy1"))
		rpTester.ReportingJobByMrn(queryMrn("check3")).Notifies(policyMrn("policy1"))
		rpTester.ReportingJobByMrn(queryMrn("query1")).Notifies(policyMrn("policy1"))
		rpTester.ReportingJobByMrn(queryMrn("check2")).Notifies(policyMrn("policy2"))
		rpTester.ReportingJobByMrn(queryMrn("check3")).Notifies(policyMrn("policy2"))

		rpTester.doTest(t, rp)
	})
}

func TestResolveV2_PolicyWithScoringSystem(t *testing.T) {
	ctx := context.Background()
	b := parseBundle(t, `
owner_mrn: //test.sth
policies:
- owner_mrn: //test.sth
  mrn: //test.sth
  groups:
  - policies:
    - uid: policy1
- uid: policy1
  scoring_system: highest impact
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
		rp, err := srv.Resolve(ctx, &policy.ResolveReq{
			PolicyMrn:    "//test.sth",
			AssetFilters: []*explorer.Mquery{{Mql: "true"}},
		})
		require.NoError(t, err)
		require.NotNil(t, rp)

		rpTester := newResolvedPolicyTester(b, srv.NewCompilerConfig())
		rpTester.ExecutesQuery(queryMrn("query1"))
		rpTester.
			ExecutesQuery(queryMrn("check1")).
			WithProps(map[string]string{"name": `return "definitely not the asset name"`})
		rpTester.CodeIdReportingJobForMrn(queryMrn("check1")).Notifies(queryMrn("check1"))
		rpTester.CodeIdReportingJobForMrn(queryMrn("query1")).Notifies(queryMrn("query1"))
		rpTester.ReportingJobByMrn(queryMrn("check1")).Notifies(policyMrn("policy1"))
		rpTester.ReportingJobByMrn(queryMrn("query1")).Notifies(policyMrn("policy1"))
		rpTester.ReportingJobByMrn(policyMrn("policy1")).WithScoringSystem(explorer.ScoringSystem_WORST).Notifies("root")

		rpTester.doTest(t, rp)
	})
}

func TestResolveV2_PolicyWithScoringSystemOverride(t *testing.T) {
	ctx := context.Background()
	b := parseBundle(t, `
owner_mrn: //test.sth
policies:
- owner_mrn: //test.sth
  mrn: //test.sth
  groups:
  - policies:
    - uid: policy1
      scoring_system: banded
- uid: policy1
  scoring_system: highest impact
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
		rp, err := srv.Resolve(ctx, &policy.ResolveReq{
			PolicyMrn:    "//test.sth",
			AssetFilters: []*explorer.Mquery{{Mql: "true"}},
		})
		require.NoError(t, err)
		require.NotNil(t, rp)

		rpTester := newResolvedPolicyTester(b, srv.NewCompilerConfig())
		rpTester.ExecutesQuery(queryMrn("query1"))
		rpTester.
			ExecutesQuery(queryMrn("check1")).
			WithProps(map[string]string{"name": `return "definitely not the asset name"`})
		rpTester.CodeIdReportingJobForMrn(queryMrn("check1")).Notifies(queryMrn("check1"))
		rpTester.CodeIdReportingJobForMrn(queryMrn("query1")).Notifies(queryMrn("query1"))
		rpTester.ReportingJobByMrn(queryMrn("check1")).Notifies(policyMrn("policy1"))
		rpTester.ReportingJobByMrn(queryMrn("query1")).Notifies(policyMrn("policy1"))
		rpTester.ReportingJobByMrn(policyMrn("policy1")).WithScoringSystem(explorer.ScoringSystem_BANDED).Notifies("root")

		rpTester.doTest(t, rp)
	})
}

func TestResolveV2_PolicyActionIgnore(t *testing.T) {
	ctx := context.Background()
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
		rp, err := srv.Resolve(ctx, &policy.ResolveReq{
			PolicyMrn:    "//test.sth",
			AssetFilters: []*explorer.Mquery{{Mql: "true"}},
		})
		require.NoError(t, err)
		require.NotNil(t, rp)

		rpTester := newResolvedPolicyTester(b, srv.NewCompilerConfig())
		rpTester.ExecutesQuery(queryMrn("query1"))
		rpTester.ExecutesQuery(queryMrn("check1"))
		rpTester.CodeIdReportingJobForMrn(queryMrn("check1")).Notifies(queryMrn("check1"))
		rpTester.CodeIdReportingJobForMrn(queryMrn("query1")).Notifies(queryMrn("query1"))
		rpTester.ReportingJobByMrn(queryMrn("check1")).Notifies(policyMrn("policy-active"))
		rpTester.ReportingJobByMrn(queryMrn("query1")).Notifies(policyMrn("policy-active"))
		rpTester.ReportingJobByMrn(queryMrn("check1")).Notifies(policyMrn("policy-ignored"))
		rpTester.ReportingJobByMrn(queryMrn("query1")).Notifies(policyMrn("policy-ignored"))
		rpTester.ReportingJobByMrn(policyMrn("policy-active")).Notifies("root")
		rpTester.ReportingJobByMrn(policyMrn("policy-ignored")).Notifies("root").WithImpact(&explorer.Impact{Scoring: explorer.ScoringSystem_IGNORE_SCORE, Action: explorer.Action_IGNORE})

		rpTester.doTest(t, rp)
	})
}

func TestResolveV2_PolicyActionScoringSystem(t *testing.T) {
	ctx := context.Background()
	b := parseBundle(t, `
owner_mrn: //test.sth
policies:
- owner_mrn: //test.sth
  mrn: //test.sth
  groups:
  - policies:
    - uid: policy-active
      scoring_system: 6
    - uid: policy-ignored
      action: 4
- uid: policy-active
  owner_mrn: //test.sth
  scoring_system: 2
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

	t.Run("resolve with scoring system", func(t *testing.T) {
		rp, err := srv.Resolve(ctx, &policy.ResolveReq{
			PolicyMrn:    "//test.sth",
			AssetFilters: []*explorer.Mquery{{Mql: "true"}},
		})
		require.NoError(t, err)
		require.NotNil(t, rp)

		rpTester := newResolvedPolicyTester(b, srv.NewCompilerConfig())
		rpTester.ExecutesQuery(queryMrn("query1"))
		rpTester.ExecutesQuery(queryMrn("check1"))
		rpTester.CodeIdReportingJobForMrn(queryMrn("check1")).Notifies(queryMrn("check1"))
		rpTester.CodeIdReportingJobForMrn(queryMrn("query1")).Notifies(queryMrn("query1"))
		rpTester.ReportingJobByMrn(queryMrn("check1")).Notifies(policyMrn("policy-active"))
		rpTester.ReportingJobByMrn(queryMrn("query1")).Notifies(policyMrn("policy-active"))
		rpTester.ReportingJobByMrn(queryMrn("check1")).Notifies(policyMrn("policy-ignored"))
		rpTester.ReportingJobByMrn(queryMrn("query1")).Notifies(policyMrn("policy-ignored"))
		rpTester.ReportingJobByMrn(policyMrn("policy-active")).WithScoringSystem(explorer.ScoringSystem_BANDED).Notifies("root")
		rpTester.ReportingJobByMrn(policyMrn("policy-ignored")).Notifies("root").WithImpact(&explorer.Impact{Scoring: explorer.ScoringSystem_IGNORE_SCORE, Action: explorer.Action_IGNORE})

		rpTester.doTest(t, rp)
	})
}

func TestResolveV2_IgnoredQuery(t *testing.T) {
	ctx := context.Background()
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

	_, err = srv.SetBundle(ctx, b)
	require.NoError(t, err)

	rp, err := srv.Resolve(ctx, &policy.ResolveReq{
		PolicyMrn:    "asset1",
		AssetFilters: []*explorer.Mquery{{Mql: "true"}},
	})

	require.NoError(t, err)
	require.NotNil(t, rp)

	rpTester := newResolvedPolicyTester(b, srv.NewCompilerConfig())
	rpTester.ExecutesQuery(queryMrn("check1"))
	rpTester.CodeIdReportingJobForMrn(queryMrn("check1")).Notifies(queryMrn("check1"))
	rpTester.CodeIdReportingJobForMrn(queryMrn("check1")).Notifies("policy-1").WithImpact(&explorer.Impact{Scoring: explorer.ScoringSystem_IGNORE_SCORE, Action: explorer.Action_IGNORE})
	rpTester.ReportingJobByMrn(policyMrn("policy-1")).Notifies("root")
}

func TestResolveV2_Frameworks(t *testing.T) {
	ctx := context.Background()
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
    - uid: check-overlap
      title: overlaps with check
      mql: 1 == 1
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

		bundle, err := srv.GetBundle(ctx, &policy.Mrn{Mrn: "asset1"})
		require.NoError(t, err)

		bundleMap, err := bundle.Compile(ctx, conf.Schema, nil)
		require.NoError(t, err)

		mrnToQueryId := map[string]string{}
		for _, q := range bundleMap.Queries {
			mrnToQueryId[q.Mrn] = q.CodeId
		}

		rp, err := srv.Resolve(ctx, &policy.ResolveReq{
			PolicyMrn:    "asset1",
			AssetFilters: []*explorer.Mquery{{Mql: "true"}},
		})
		require.NoError(t, err)
		require.NotNil(t, rp)

		rpTester := newResolvedPolicyTester(b, srv.NewCompilerConfig())
		rpTester.ExecutesQuery(queryMrn("check-fail"))
		rpTester.ExecutesQuery(queryMrn("check-pass-1"))
		rpTester.ExecutesQuery(queryMrn("check-pass-2"))
		rpTester.ExecutesQuery(queryMrn("active-query"))
		rpTester.ExecutesQuery(queryMrn("active-query-2"))
		rpTester.ExecutesQuery(queryMrn("check-overlap"))

		rpTester.CodeIdReportingJobForMrn(queryMrn("check-fail")).Notifies(queryMrn("check-fail"))
		rpTester.CodeIdReportingJobForMrn(queryMrn("check-fail")).Notifies(controlMrn("control2"))
		rpTester.ReportingJobByMrn(queryMrn("check-fail")).Notifies(policyMrn("policy1"))

		rpTester.CodeIdReportingJobForMrn(queryMrn("check-pass-1")).Notifies(queryMrn("check-pass-1"))
		// This is a limitation of the test framework. We lookup the code id from check-pass-1 because
		// we need 1 tester that has all the notifies
		rpTester.CodeIdReportingJobForMrn(queryMrn("check-pass-1")).Notifies(queryMrn("check-overlap"))
		rpTester.CodeIdReportingJobForMrn(queryMrn("check-pass-1")).Notifies(controlMrn("control1"))
		rpTester.ReportingJobByMrn(queryMrn("check-pass-1")).Notifies(policyMrn("policy1"))

		rpTester.CodeIdReportingJobForMrn(queryMrn("check-pass-2")).Notifies(queryMrn("check-pass-2"))
		rpTester.CodeIdReportingJobForMrn(queryMrn("check-pass-2")).Notifies(controlMrn("control2"))
		rpTester.ReportingJobByMrn(queryMrn("check-pass-2")).Notifies(policyMrn("policy1"))

		rpTester.CodeIdReportingJobForMrn(queryMrn("active-query")).Notifies(queryMrn("active-query"))
		rpTester.CodeIdReportingJobForMrn(queryMrn("active-query")).Notifies(controlMrn("control1")).WithImpact(&explorer.Impact{Scoring: explorer.ScoringSystem_IGNORE_SCORE, Action: explorer.Action_IGNORE})
		rpTester.ReportingJobByMrn(queryMrn("active-query")).Notifies(policyMrn("policy1"))

		rpTester.CodeIdReportingJobForMrn(queryMrn("active-query-2")).Notifies(queryMrn("active-query-2"))
		rpTester.CodeIdReportingJobForMrn(queryMrn("active-query-2")).Notifies(controlMrn("control1")).WithImpact(&explorer.Impact{Scoring: explorer.ScoringSystem_IGNORE_SCORE, Action: explorer.Action_IGNORE})
		rpTester.ReportingJobByMrn(queryMrn("active-query-2")).Notifies(policyMrn("policy1"))

		rpTester.ReportingJobByMrn(queryMrn("check-overlap")).Notifies(policyMrn("policy1"))

		rpTester.ReportingJobByMrn(controlMrn("control1")).Notifies(controlMrn("control4"))
		rpTester.ReportingJobByMrn(controlMrn("control1")).Notifies(controlMrn("control5"))
		rpTester.ReportingJobByMrn(controlMrn("control1")).Notifies(frameworkMrn("framework1"))
		rpTester.ReportingJobByMrn(controlMrn("control2")).Notifies(frameworkMrn("framework1"))
		rpTester.ReportingJobByMrn(controlMrn("control4")).Notifies(frameworkMrn("framework1"))

		rpTester.ReportingJobByMrn(policyMrn("policy1")).Notifies("root")
		rpTester.ReportingJobByMrn(frameworkMrn("framework1")).Notifies(frameworkMrn("parent-framework"))
		rpTester.ReportingJobByMrn(frameworkMrn("parent-framework")).Notifies("root")

		rpTester.doTest(t, rp)
	})

	t.Run("test resolving with inactive data queries", func(t *testing.T) {
		// test that creating a bundle with inactive data queries  (where the packs/policies are inactive)
		// will still end up in a successfully resolved policy for the asset
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

		bundle, err := srv.GetBundle(ctx, &policy.Mrn{Mrn: "asset1"})
		require.NoError(t, err)

		bundleMap, err := bundle.Compile(ctx, conf.Schema, nil)
		require.NoError(t, err)

		mrnToQueryId := map[string]string{}
		for _, q := range bundleMap.Queries {
			mrnToQueryId[q.Mrn] = q.CodeId
		}

		rp, err := srv.Resolve(ctx, &policy.ResolveReq{
			PolicyMrn:    "asset1",
			AssetFilters: []*explorer.Mquery{{Mql: "true"}},
		})
		require.NoError(t, err)
		require.NotNil(t, rp)

		rpTester := newResolvedPolicyTester(b, srv.NewCompilerConfig())
		rpTester.ExecutesQuery(queryMrn("active-query"))

		rpTester.CodeIdReportingJobForMrn(queryMrn("active-query")).Notifies(queryMrn("active-query"))
		rpTester.CodeIdReportingJobForMrn(queryMrn("active-query")).Notifies(controlMrn("control1"))
		rpTester.ReportingJobByMrn(queryMrn("active-query")).Notifies(policyMrn("policy1"))

		rpTester.ReportingJobByMrn(controlMrn("control1")).Notifies(frameworkMrn("framework1"))

		rpTester.ReportingJobByMrn(frameworkMrn("framework1")).Notifies(frameworkMrn("parent-framework"))
		rpTester.ReportingJobByMrn(frameworkMrn("parent-framework")).Notifies("root")
		rpTester.ReportingJobByMrn(policyMrn("policy1")).Notifies("root")

		rpTester.doTest(t, rp)
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

		bundle, err := srv.GetBundle(ctx, &policy.Mrn{Mrn: "asset1"})
		require.NoError(t, err)

		bundleMap, err := bundle.Compile(ctx, conf.Schema, nil)
		require.NoError(t, err)

		mrnToQueryId := map[string]string{}
		for _, q := range bundleMap.Queries {
			mrnToQueryId[q.Mrn] = q.CodeId
		}

		rp, err := srv.Resolve(ctx, &policy.ResolveReq{
			PolicyMrn:    "asset1",
			AssetFilters: []*explorer.Mquery{{Mql: "true"}},
		})
		require.NoError(t, err)
		require.NotNil(t, rp)

		rpTester := newResolvedPolicyTester(b, srv.NewCompilerConfig())
		rpTester.ExecutesQuery(queryMrn("query-2"))

		rpTester.ReportingJobByMrn(queryMrn("query-1")).DoesNotExist()

		rpTester.CodeIdReportingJobForMrn(queryMrn("query-2")).Notifies(queryMrn("query-2"))
		rpTester.CodeIdReportingJobForMrn(queryMrn("query-2")).Notifies(controlMrn("control1"))
		rpTester.ReportingJobByMrn(queryMrn("query-2")).Notifies(policyMrn("policy2"))

		rpTester.ReportingJobByMrn(controlMrn("control1")).Notifies(frameworkMrn("framework1"))

		rpTester.ReportingJobByMrn(frameworkMrn("framework1")).Notifies(frameworkMrn("parent-framework"))
		rpTester.ReportingJobByMrn(frameworkMrn("parent-framework")).Notifies("root")
		rpTester.ReportingJobByMrn(policyMrn("policy2")).Notifies("root")

		rpTester.doTest(t, rp)
	})

	t.Run("test checksumming", func(t *testing.T) {
		bInitial := parseBundle(t, bundleStr)

		srv := initResolver(t, []*testAsset{
			{asset: "asset1", policies: []string{policyMrn("policy1")}, frameworks: []string{frameworkMrn("parent-framework")}},
		}, []*policy.Bundle{bInitial})

		rpInitial, err := srv.Resolve(ctx, &policy.ResolveReq{
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

		rpFrameworkUpdate, err := srv.Resolve(ctx, &policy.ResolveReq{
			PolicyMrn:    "asset1",
			AssetFilters: []*explorer.Mquery{{Mql: "true"}},
		})
		require.NoError(t, err)
		require.NotNil(t, rpFrameworkUpdate)

		require.NotEqual(t, rpInitial.GraphExecutionChecksum, rpFrameworkUpdate.GraphExecutionChecksum)
	})
}

// TestResolve_PoliciesMatchingAgainstIncorrectPlatform tests that policies are not matched against
// assets that do not match the asset filter. It was possible that the reporting structure had
// a node for the policy, but no actual reporting job for it. To the user, this could look
// like the policy was executed. The issue was that a policy was considered matching if either
// the groups or any of its queries filters matched. This tests to ensure that if the policies
// group filtered it out, it doesn't show up in the reporting structure
func TestResolveV2_PoliciesMatchingAgainstIncorrectPlatform(t *testing.T) {
	ctx := context.Background()
	b := parseBundle(t, `
owner_mrn: //test.sth
policies:
- uid: policy1
  groups:
  - type: chapter
    filters: "true"
    checks:
    - uid: check1
- uid: policy2
  groups:
  - type: chapter
    filters: "false"
    checks:
    - uid: check2
- uid: pack1
  groups:
  - type: chapter
    filters: "true"
    queries:
    - uid: dataquery1
- uid: pack2
  groups:
  - type: chapter
    filters: "false"
    queries:
    - uid: dataquery2

queries:
- uid: check1
  title: check1
  mql: true
- uid: check2
  title: check2
  filters: |
    true
  mql: |
    1 == 1
- uid: dataquery1
  title: dataquery1
  mql: |
    asset.name
- uid: dataquery2
  title: dataquery2
  filters: |
    true
  mql: |
    asset.version
`)

	srv := initResolver(t, []*testAsset{
		{asset: "asset1", policies: []string{policyMrn("policy1"), policyMrn("policy2"), policyMrn("pack1"), policyMrn("pack2")}},
	}, []*policy.Bundle{b})

	t.Run("resolve with correct filters", func(t *testing.T) {
		rp, err := srv.Resolve(ctx, &policy.ResolveReq{
			PolicyMrn:    "asset1",
			AssetFilters: []*explorer.Mquery{{Mql: "true"}},
		})
		require.NoError(t, err)
		require.NotNil(t, rp)

		rpTester := newResolvedPolicyTester(b, srv.NewCompilerConfig())

		rpTester.ReportingJobByMrn(policyMrn("policy2")).DoesNotExist()
		rpTester.ReportingJobByMrn(policyMrn("pack2")).DoesNotExist()

		rpTester.doTest(t, rp)
	})
}

func TestResolveV2_NeverPruneRoot(t *testing.T) {
	ctx := context.Background()
	b := parseBundle(t, `
owner_mrn: //test.sth
policies:
- uid: policy1
  groups:
  - type: chapter
    filters: "false"
    checks:
    - uid: check1

queries:
- uid: check1
  title: check1
  filters: |
    true
  mql: |
    1 == 1
`)

	srv := initResolver(t, []*testAsset{
		{asset: "asset1", policies: []string{policyMrn("policy1")}},
	}, []*policy.Bundle{b})

	rp, err := srv.Resolve(ctx, &policy.ResolveReq{
		PolicyMrn:    "asset1",
		AssetFilters: []*explorer.Mquery{{Mql: "true"}},
	})
	require.NoError(t, err)
	require.NotNil(t, rp)

}

func TestResolveV2_PoliciesMatchingFilters(t *testing.T) {
	ctx := context.Background()
	b := parseBundle(t, `
owner_mrn: //test.sth
policies:
- uid: policy1
  groups:
  - type: chapter
    checks:
    - uid: check1
    - uid: check2
queries:
- uid: check1
  title: check1
  filters:
  - mql: asset.name == "asset1"
  - mql: asset.name == "asset2"
  mql: |
    asset.version
- uid: check2
  title: check2
  filters:
  - mql: |
      asset.name == "asset1"
      asset.name == "asset2"
  mql: |
    asset.platform
`)

	srv := initResolver(t, []*testAsset{
		{asset: "asset1", policies: []string{policyMrn("policy1")}},
	}, []*policy.Bundle{b})

	t.Run("resolve with correct filters", func(t *testing.T) {
		rp, err := srv.Resolve(ctx, &policy.ResolveReq{
			PolicyMrn:    "asset1",
			AssetFilters: []*explorer.Mquery{{Mql: "asset.name == \"asset1\""}},
		})
		require.NoError(t, err)
		require.NotNil(t, rp)

		rpTester := newResolvedPolicyTester(b, srv.NewCompilerConfig())

		rpTester.ExecutesQuery(queryMrn("check1"))
		rpTester.DoesNotExecutesQuery(queryMrn("check2"))

		rpTester.doTest(t, rp)
	})
}

func TestResolveV2_TwoMrns(t *testing.T) {
	ctx := context.Background()
	b := parseBundle(t, `
owner_mrn: //test.sth
policies:
- uid: policy1
  groups:
  - filters:
    - mql: asset.name == "asset1"
    checks:
    - uid: check1
      mql: asset.name == props.name
      props:
      - uid: name
        mql: return "definitely not the asset name"
    - uid: check2
      mql: asset.name == props.name
      props:
      - uid: name
        mql: return "definitely not the asset name"
`)

	srv := initResolver(t, []*testAsset{
		{asset: "asset1", policies: []string{policyMrn("policy1")}},
	}, []*policy.Bundle{b})

	t.Run("resolve two MRNs to one codeID matching filter", func(t *testing.T) {
		rp, err := srv.Resolve(ctx, &policy.ResolveReq{
			PolicyMrn:    policyMrn("policy1"),
			AssetFilters: []*explorer.Mquery{{Mql: "asset.name == \"asset1\""}},
		})
		require.NoError(t, err)
		require.NotNil(t, rp)

		rpTester := newResolvedPolicyTester(b, srv.NewCompilerConfig())
		rpTester.ExecutesQuery(queryMrn("check1"))
		rpTester.ExecutesQuery(queryMrn("check2"))

		rpTester.CodeIdReportingJobForMrn(queryMrn("check1")).Notifies(queryMrn("check1"))
		// This is a limitation of the test framework. We lookup the code id from check-pass-1 because
		// we need 1 tester that has all the notifies
		rpTester.CodeIdReportingJobForMrn(queryMrn("check1")).Notifies(queryMrn("check2"))

		rpTester.ReportingJobByMrn(queryMrn("check1")).Notifies("root")
		rpTester.ReportingJobByMrn(queryMrn("check2")).Notifies("root")

		rpTester.doTest(t, rp)
	})
}

func TestResolveV2_TwoMrns_FilterMismatch(t *testing.T) {
	ctx := context.Background()
	b := parseBundle(t, `
owner_mrn: //test.sth
policies:
- uid: policy1
  groups:
  - checks:
    - uid: check1
      mql: asset.name == props.name
      props:
      - uid: name
        mql: return "definitely not the asset name"
      filters:
      - mql: asset.name == "asset1"
    - uid: check2
      mql: asset.name == props.name
      props:
      - uid: name
        mql: return "definitely not the asset name"
      filters:
      - mql: asset.name == "asset2"
`)

	srv := initResolver(t, []*testAsset{
		{asset: "asset1", policies: []string{policyMrn("policy1")}},
	}, []*policy.Bundle{b})

	t.Run("resolve two MRNs to one codeID matching filter", func(t *testing.T) {
		rp, err := srv.Resolve(ctx, &policy.ResolveReq{
			PolicyMrn:    policyMrn("policy1"),
			AssetFilters: []*explorer.Mquery{{Mql: "asset.name == \"asset1\""}},
		})
		require.NoError(t, err)
		require.NotNil(t, rp)

		rpTester := newResolvedPolicyTester(b, srv.NewCompilerConfig())
		rpTester.ExecutesQuery(queryMrn("check1"))

		rpTester.CodeIdReportingJobForMrn(queryMrn("check1")).Notifies(queryMrn("check1"))
		rpTester.ReportingJobByMrn(queryMrn("check1")).Notifies("root")
		rpTester.ReportingJobByMrn(queryMrn("check2")).DoesNotExist()

		rpTester.doTest(t, rp)
	})
}

func TestResolveV2_TwoMrns_DataQueries(t *testing.T) {
	ctx := context.Background()
	b := parseBundle(t, `
owner_mrn: //test.sth
policies:
- uid: policy1
  groups:
  - filters:
    - mql: asset.name == "asset1"
    checks:
    - uid: check1
      mql: asset.name == props.name
      props:
      - uid: name
        mql: return "definitely not the asset name"
  - queries:
    - uid: active-query
      title: users
      mql: users
    - uid: active-query-2
      title: users length
      mql: users
`)

	srv := initResolver(t, []*testAsset{
		{asset: "asset1", policies: []string{policyMrn("policy1")}},
	}, []*policy.Bundle{b})

	t.Run("resolve two MRNs to one codeID matching filter", func(t *testing.T) {
		rp, err := srv.Resolve(ctx, &policy.ResolveReq{
			PolicyMrn:    policyMrn("policy1"),
			AssetFilters: []*explorer.Mquery{{Mql: "asset.name == \"asset1\""}},
		})
		require.NoError(t, err)
		require.NotNil(t, rp)

		rpTester := newResolvedPolicyTester(b, srv.NewCompilerConfig())
		rpTester.ExecutesQuery(queryMrn("check1"))
		rpTester.ExecutesQuery(queryMrn("active-query"))
		rpTester.ExecutesQuery(queryMrn("active-query-2"))

		rpTester.CodeIdReportingJobForMrn(queryMrn("check1")).Notifies(queryMrn("check1"))
		rpTester.CodeIdReportingJobForMrn(queryMrn("active-query")).Notifies(queryMrn("active-query"))
		// This is a limitation of the test framework. We lookup the code id from active-query because
		// we need 1 tester that has all the notifies
		rpTester.CodeIdReportingJobForMrn(queryMrn("active-query")).Notifies(queryMrn("active-query-2"))

		rpTester.ReportingJobByMrn(queryMrn("check1")).Notifies("root")
		rpTester.ReportingJobByMrn(queryMrn("active-query")).Notifies("root")
		rpTester.ReportingJobByMrn(queryMrn("active-query-2")).Notifies("root")

		rpTester.doTest(t, rp)
	})
}

func TestResolveV2_TwoMrns_Variants(t *testing.T) {
	ctx := context.Background()
	b := parseBundle(t, `
owner_mrn: //test.sth
policies:
- uid: policy1
  groups:
  - checks:
    - uid: check-variants
queries:
  - uid: check-variants
    variants:
      - uid: variant1
      - uid: variant2
  - uid: variant1
    mql: asset.name == "test1"
    filters: asset.family.contains("unix")
  - uid: variant2
    mql: asset.name == "test1"
    filters: asset.name == "asset1"
`)

	srv := initResolver(t, []*testAsset{
		{asset: "asset1", policies: []string{policyMrn("policy1")}},
	}, []*policy.Bundle{b})

	t.Run("resolve two variants to different codeIDs matching filter", func(t *testing.T) {
		rp, err := srv.Resolve(ctx, &policy.ResolveReq{
			PolicyMrn: policyMrn("policy1"),
			AssetFilters: []*explorer.Mquery{
				{Mql: "asset.name == \"asset1\""},
				{Mql: "asset.family.contains(\"unix\")"},
			},
		})
		require.NoError(t, err)
		require.NotNil(t, rp)

		rpTester := newResolvedPolicyTester(b, srv.NewCompilerConfig())
		rpTester.ExecutesQuery(queryMrn("variant1"))
		rpTester.DoesNotExecutesQuery(queryMrn("variant2"))

		rpTester.CodeIdReportingJobForMrn(queryMrn("variant1")).Notifies(queryMrn("variant1"))
		rpTester.ReportingJobByMrn(queryMrn("variant1")).Notifies("check-variants")
		rpTester.ReportingJobByMrn(queryMrn("variant2")).DoesNotExist()
		rpTester.ReportingJobByMrn("check-variants").Notifies("root")
	})
}

func TestResolveV2_Variants(t *testing.T) {
	ctx := context.Background()
	b := parseBundle(t, `
owner_mrn: //test.sth
policies:
  - uid: example2
    name: Another policy
    version: "1.0.0"
    groups:
      # Additionally it defines some queries of its own
      - type: chapter
        title: Some uname infos
        queries:
          # In this case, we are using a shared query that is defined below
          - uid: uname
        checks:
          - uid: check-os
            variants:
              - uid: check-os-unix
              - uid: check-os-windows

queries:
  # This is a composed query which has two variants: one for unix type systems
  # and one for windows, where we don't run the additional argument.
  # If you run the "uname" query, it will pick matching sub-queries for you.
  - uid: uname
    title: Collect uname info
    variants:
      - uid: unix-uname
      - uid: windows-uname
  - uid: unix-uname
    mql: command("uname -a").stdout
    filters: asset.family.contains("unix")
  - uid: windows-uname
    mql: command("uname").stdout
    filters: asset.family.contains("windows")

  - uid: check-os-unix
    filters: asset.family.contains("unix")
    title: A check only run on Linux/macOS
    mql: users.contains(name == "root")
  - uid: check-os-windows
    filters: asset.family.contains("windows")
    title: A check only run on Windows
    mql: users.contains(name == "Administrator")`)

	srv := initResolver(t, []*testAsset{
		{asset: "asset1", policies: []string{policyMrn("example2")}},
	}, []*policy.Bundle{b})

	_, err := srv.SetBundle(ctx, b)
	require.NoError(t, err)

	_, err = b.Compile(ctx, conf.Schema, nil)
	require.NoError(t, err)

	rp, err := srv.Resolve(ctx, &policy.ResolveReq{
		PolicyMrn:    policyMrn("example2"),
		AssetFilters: []*explorer.Mquery{{Mql: "asset.family.contains(\"windows\")"}},
	})

	require.NoError(t, err)
	require.NotNil(t, rp)

	rpTester := newResolvedPolicyTester(b, srv.NewCompilerConfig())

	rpTester.ExecutesQuery(queryMrn("windows-uname"))
	rpTester.ExecutesQuery(queryMrn("check-os-windows"))
	rpTester.DoesNotExecutesQuery(queryMrn("unix-uname"))
	rpTester.DoesNotExecutesQuery(queryMrn("check-os-unix"))

	rpTester.CodeIdReportingJobForMrn(queryMrn("windows-uname")).Notifies(queryMrn("windows-uname"))
	rpTester.ReportingJobByMrn(queryMrn("windows-uname")).Notifies(queryMrn("uname"))
	rpTester.ReportingJobByMrn(queryMrn("uname")).Notifies("root")

	rpTester.CodeIdReportingJobForMrn(queryMrn("check-os-windows")).Notifies(queryMrn("check-os-windows"))
	rpTester.ReportingJobByMrn(queryMrn("check-os-windows")).Notifies(queryMrn("check-os"))
	rpTester.ReportingJobByMrn(queryMrn("check-os")).Notifies("root")

	rpTester.doTest(t, rp)
}

func TestResolveV2_RiskFactors(t *testing.T) {
	ctx := context.Background()
	b := parseBundle(t, `
owner_mrn: //test.sth
queries:
- uid: query-1
  title: query-1
  mql: 3 == 3
- uid: query-2
  title: query-2
  mql: 1 == 2
policies:
  - name: testpolicy1
    uid: testpolicy1
    risk_factors:
    - uid: sshd-service
      magnitude: 0.9
    - uid: sshd-service-na
      action: 2
    groups:
    - filters: asset.name == "asset1"
      checks:
      - uid: query-1
      - uid: query-2
      policies:
      - uid: risk-factors-security
  - uid: risk-factors-security
    name: Mondoo Risk Factors analysis
    version: "1.0.0"
    risk_factors:
      - uid: sshd-service
        title: SSHd Service running
        indicator: asset-in-use
        magnitude: 0.6
        filters:
          - mql: |
              asset.name == "asset1"
        checks:
          - uid: sshd-service-running
            mql: 1 == 1
      - uid: sshd-service-na
        title: SSHd Service running
        indicator: asset-in-use
        magnitude: 0.5
        filters:
          - mql: |
              asset.name == "asset1"
        checks:
          - uid: sshd-service-running-na
            mql: 1 == 7
      - uid: not-matching
        title: Not Matching
        indicator: asset-in-use
        magnitude: 0.5
        filters:
          - mql: |
              asset.name == "asset2"
        checks:
          - uid: not-matching
            mql: true == false
`)

	srv := initResolver(t, []*testAsset{
		{asset: "asset1", policies: []string{policyMrn("testpolicy1")}},
	}, []*policy.Bundle{b})

	rp, err := srv.Resolve(ctx, &policy.ResolveReq{
		PolicyMrn:    "asset1",
		AssetFilters: []*explorer.Mquery{{Mql: "asset.name == \"asset1\""}},
	})
	require.NoError(t, err)
	require.NotNil(t, rp)

	rpTester := newResolvedPolicyTester(b, srv.NewCompilerConfig())

	rpTester.ExecutesQuery(queryMrn("query-1"))
	rpTester.ExecutesQuery(queryMrn("query-2"))
	rpTester.ExecutesQuery(queryMrn("sshd-service-running"))
	rpTester.DoesNotExecutesQuery(queryMrn("sshd-service-running-na"))
	rpTester.DoesNotExecutesQuery(queryMrn("not-matching"))
	rpTester.ReportingJobByMrn(queryMrn("sshd-service-running-na")).DoesNotExist()
	rpTester.ReportingJobByMrn(queryMrn("not-matching")).DoesNotExist()
	rpTester.ReportingJobByMrn(riskFactorMrn("not-matching")).DoesNotExist()

	rpTester.CodeIdReportingJobForMrn(queryMrn("query-1")).Notifies(queryMrn("query-1"))
	rpTester.CodeIdReportingJobForMrn(queryMrn("query-2")).Notifies(queryMrn("query-2"))

	// rpTester.CodeIdReportingJobForMrn(queryMrn("sshd-service-running")).Notifies(queryMrn("sshd-service-running"))
	rpTester.CodeIdReportingJobForMrn(queryMrn("sshd-service-running")).Notifies(riskFactorMrn("sshd-service")).WithImpact(&explorer.Impact{Scoring: explorer.ScoringSystem_IGNORE_SCORE, Action: explorer.Action_IGNORE})
	rpTester.ReportingJobByMrn(riskFactorMrn("sshd-service")).WithType(policy.ReportingJob_RISK_FACTOR).Notifies(policyMrn("risk-factors-security"))

	rpTester.ReportingJobByMrn(queryMrn("query-1")).Notifies(policyMrn("testpolicy1"))
	rpTester.ReportingJobByMrn(queryMrn("query-2")).Notifies(policyMrn("testpolicy1"))

	rpTester.ReportingJobByMrn(policyMrn("testpolicy1")).Notifies("root")

	rpTester.doTest(t, rp)

	require.Equal(t, float32(0.9), rp.CollectorJob.RiskFactors[riskFactorMrn("sshd-service")].Magnitude.GetValue())
}

func TestResolveV2_FrameworkExceptions(t *testing.T) {
	ctx := context.Background()
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
				frameworks: []string{frameworkMrn("mondoo-ucf")},
			},
		}, []*policy.Bundle{b})

		rp, err := srv.Resolve(ctx, &policy.ResolveReq{
			PolicyMrn:    "asset1",
			AssetFilters: []*explorer.Mquery{{Mql: "true"}},
		})
		require.NoError(t, err)
		require.NotNil(t, rp)

		rpTester := newResolvedPolicyTester(b, srv.NewCompilerConfig())

		rpTester.ReportingJobByMrn(controlMrn("mondoo-ucf-02")).Notifies(frameworkMrn("mondoo-ucf")).WithImpact(&explorer.Impact{Scoring: explorer.ScoringSystem_IGNORE_SCORE, Action: explorer.Action_IGNORE})

		rpTester.doTest(t, rp)
	})

	t.Run("resolve with ignored control and validUntil", func(t *testing.T) {
		b := parseBundle(t, bundleString)
		b.Frameworks[0].Groups[1].EndDate = time.Now().Add(time.Hour).Unix()

		srv = initResolver(t, []*testAsset{
			{
				asset:      "asset1",
				policies:   []string{policyMrn("ssh-policy")},
				frameworks: []string{frameworkMrn("mondoo-ucf")},
			},
		}, []*policy.Bundle{b})

		rp, err := srv.Resolve(ctx, &policy.ResolveReq{
			PolicyMrn:    "asset1",
			AssetFilters: []*explorer.Mquery{{Mql: "true"}},
		})
		require.NoError(t, err)
		require.NotNil(t, rp)

		rpTester := newResolvedPolicyTester(b, srv.NewCompilerConfig())

		rpTester.ReportingJobByMrn(controlMrn("mondoo-ucf-02")).Notifies(frameworkMrn("mondoo-ucf")).WithImpact(&explorer.Impact{Scoring: explorer.ScoringSystem_IGNORE_SCORE, Action: explorer.Action_IGNORE})

		rpTester.doTest(t, rp)
	})

	t.Run("resolve with expired validUntil", func(t *testing.T) {
		b := parseBundle(t, bundleString)
		b.Frameworks[0].Groups[1].EndDate = time.Now().Add(-time.Hour).Unix()

		srv = initResolver(t, []*testAsset{
			{
				asset:      "asset1",
				policies:   []string{policyMrn("ssh-policy")},
				frameworks: []string{frameworkMrn("mondoo-ucf")},
			},
		}, []*policy.Bundle{b})

		rp, err := srv.Resolve(ctx, &policy.ResolveReq{
			PolicyMrn:    "asset1",
			AssetFilters: []*explorer.Mquery{{Mql: "true"}},
		})
		require.NoError(t, err)
		require.NotNil(t, rp)

		require.NoError(t, err)
		require.NotNil(t, rp)

		rpTester := newResolvedPolicyTester(b, srv.NewCompilerConfig())

		rpTester.ReportingJobByMrn(controlMrn("mondoo-ucf-02")).Notifies(frameworkMrn("mondoo-ucf")).WithImpact(nil)

		rpTester.doTest(t, rp)
	})

	t.Run("resolve with expired validUntil", func(t *testing.T) {
		b := parseBundle(t, bundleString)
		b.Frameworks[0].Groups[1].EndDate = time.Now().Add(time.Hour).Unix()
		b.Frameworks[0].Groups[1].Type = policy.GroupType_DISABLE
		b.Frameworks[0].Groups[1].ReviewStatus = policy.ReviewStatus_REJECTED

		srv = initResolver(t, []*testAsset{
			{
				asset:      "asset1",
				policies:   []string{policyMrn("ssh-policy")},
				frameworks: []string{frameworkMrn("mondoo-ucf")},
			},
		}, []*policy.Bundle{b})

		rp, err := srv.Resolve(ctx, &policy.ResolveReq{
			PolicyMrn:    "asset1",
			AssetFilters: []*explorer.Mquery{{Mql: "true"}},
		})
		require.NoError(t, err)
		require.NotNil(t, rp)

		require.NoError(t, err)
		require.NotNil(t, rp)

		rpTester := newResolvedPolicyTester(b, srv.NewCompilerConfig())

		rpTester.ReportingJobByMrn(controlMrn("mondoo-ucf-02")).Notifies(frameworkMrn("mondoo-ucf")).WithImpact(nil)

		rpTester.doTest(t, rp)
	})

	t.Run("resolve with disabled control", func(t *testing.T) {
		b := parseBundle(t, bundleString)
		b.Frameworks = append(b.Frameworks, &policy.Framework{
			Mrn: frameworkMrn("test"),
			Dependencies: []*policy.FrameworkRef{
				{
					Mrn:    frameworkMrn("mondoo-ucf"),
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
				frameworks: []string{frameworkMrn("mondoo-ucf"), frameworkMrn("test")},
			},
		}, []*policy.Bundle{b})

		rp, err := srv.Resolve(context.Background(), &policy.ResolveReq{
			PolicyMrn:    "asset1",
			AssetFilters: []*explorer.Mquery{{Mql: "true"}},
		})
		require.NoError(t, err)
		require.NotNil(t, rp)

		rpTester := newResolvedPolicyTester(b, srv.NewCompilerConfig())

		rpTester.ReportingJobByMrn(controlMrn("mondoo-ucf-01")).DoesNotExist()

		rpTester.doTest(t, rp)
	})
}

func TestResolveV2_PolicyExceptionIgnored(t *testing.T) {
	ctx := context.Background()
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
- owner_mrn: //test.sth
  mrn: //test.sth
  groups:
  - policies:
    - uid: policy1
  - type: ignored
    uid: "exceptions-1"
    checks:
    - uid: check1
`)

	srv := initResolver(t, []*testAsset{
		{asset: "asset1", policies: []string{policyMrn("policy1")}},
	}, []*policy.Bundle{b})

	t.Run("resolve with correct filters", func(t *testing.T) {
		rp, err := srv.Resolve(ctx, &policy.ResolveReq{
			PolicyMrn:    "//test.sth",
			AssetFilters: []*explorer.Mquery{{Mql: "true"}},
		})
		require.NoError(t, err)
		require.NotNil(t, rp)

		rpTester := newResolvedPolicyTester(b, srv.NewCompilerConfig())
		rpTester.ExecutesQuery(queryMrn("query1"))
		rpTester.
			ExecutesQuery(queryMrn("check1")).
			WithProps(map[string]string{"name": `return "definitely not the asset name"`})
		rpTester.CodeIdReportingJobForMrn(queryMrn("check1")).Notifies(queryMrn("check1"))
		rpTester.CodeIdReportingJobForMrn(queryMrn("query1")).Notifies(queryMrn("query1"))
		rpTester.ReportingJobByMrn(queryMrn("check1")).Notifies(policyMrn("policy1")).WithImpact(&explorer.Impact{Scoring: explorer.ScoringSystem_IGNORE_SCORE, Action: explorer.Action_IGNORE})
		rpTester.ReportingJobByMrn(queryMrn("query1")).Notifies(policyMrn("policy1"))
		rpTester.ReportingJobByMrn(policyMrn("policy1")).Notifies("root")

		rpTester.doTest(t, rp)
	})
}

func TestResolveV2_PolicyExceptionDisabled(t *testing.T) {
	ctx := context.Background()
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
- owner_mrn: //test.sth
  mrn: //test.sth
  groups:
  - policies:
    - uid: policy1
  - type: disable
    uid: "exceptions-1"
    checks:
    - uid: query1
`)

	srv := initResolver(t, []*testAsset{
		{asset: "asset1", policies: []string{policyMrn("policy1")}},
	}, []*policy.Bundle{b})

	t.Run("resolve with correct filters", func(t *testing.T) {
		rp, err := srv.Resolve(ctx, &policy.ResolveReq{
			PolicyMrn:    "//test.sth",
			AssetFilters: []*explorer.Mquery{{Mql: "true"}},
		})
		require.NoError(t, err)
		require.NotNil(t, rp)

		rpTester := newResolvedPolicyTester(b, srv.NewCompilerConfig())
		rpTester.DoesNotExecutesQuery(queryMrn("query1"))
		rpTester.
			ExecutesQuery(queryMrn("check1")).
			WithProps(map[string]string{"name": `return "definitely not the asset name"`})
		rpTester.CodeIdReportingJobForMrn(queryMrn("check1")).Notifies(queryMrn("check1"))
		rpTester.ReportingJobByMrn(queryMrn("check1")).Notifies(policyMrn("policy1"))
		rpTester.ReportingJobByMrn(policyMrn("policy1")).Notifies("root")

		rpTester.doTest(t, rp)
	})
}

func TestResolveV2_PropsDefinedAtPolicy(t *testing.T) {
	ctx := context.Background()
	b := parseBundle(t, `
owner_mrn: //test.sth
policies:
- uid: policy1
  props:
  - uid: name
    mql: return "definitely not the asset name"
  groups:
  - type: chapter
    filters: "true"
    checks:
    - uid: check1
      mql: asset.name == props.name
      props:
      - uid: name
    queries:
    - uid: query1
      mql: asset{*}
`)

	srv := initResolver(t, []*testAsset{
		{asset: "asset1", policies: []string{policyMrn("policy1")}},
	}, []*policy.Bundle{b})

	t.Run("resolve with correct filters", func(t *testing.T) {
		rp, err := srv.Resolve(ctx, &policy.ResolveReq{
			PolicyMrn:    policyMrn("policy1"),
			AssetFilters: []*explorer.Mquery{{Mql: "true"}},
		})
		require.NoError(t, err)
		require.NotNil(t, rp)
		require.Len(t, rp.ExecutionJob.Queries, 3)
		require.Len(t, rp.Filters, 1)
		require.Len(t, rp.CollectorJob.ReportingJobs, 5)

		rpTester := newResolvedPolicyTester(b, srv.NewCompilerConfig())
		rpTester.ExecutesQuery(queryMrn("query1"))
		rpTester.
			ExecutesQuery(queryMrn("check1")).
			WithProps(map[string]string{"name": `return "definitely not the asset name"`})
		rpTester.CodeIdReportingJobForMrn(queryMrn("check1")).Notifies(queryMrn("check1"))
		rpTester.CodeIdReportingJobForMrn(queryMrn("query1")).Notifies(queryMrn("query1"))
		rpTester.ReportingJobByMrn(queryMrn("check1")).Notifies("root")
		rpTester.ReportingJobByMrn(queryMrn("query1")).Notifies("root")

		rpTester.doTest(t, rp)
	})
}

func TestResolveV2_ValidUntil(t *testing.T) {
	ctx := context.Background()
	bYaml := `
owner_mrn: //test.sth
policies:
  - uid: example1
    name: Example policy 1
    groups:
      - filters:
          - mql: "true"
        checks:
          - uid: check-01
            mql: |
              1 == 2
            impact: 95

      - type: override
        title: Exception 1
        valid:
          until: 2025-09-01
        checks:
          - uid: check-01
            action: preview
`

	t.Run("now is before validUntil", func(t *testing.T) {
		b := parseBundle(t, bYaml)
		srv := initResolver(t, []*testAsset{
			{asset: "asset1", policies: []string{policyMrn("example1")}},
		}, []*policy.Bundle{b})
		srv.NowProvider = func() time.Time {
			return time.Date(2024, 9, 1, 0, 0, 0, 0, time.UTC)
		}
		rp, err := srv.Resolve(ctx, &policy.ResolveReq{
			PolicyMrn:    policyMrn("example1"),
			AssetFilters: []*explorer.Mquery{{Mql: "true"}},
		})
		require.NoError(t, err)
		require.NotNil(t, rp)

		rpTester := newResolvedPolicyTester(b, srv.NewCompilerConfig())
		rpTester.ExecutesQuery(queryMrn("check-01"))
		rpTester.CodeIdReportingJobForMrn(queryMrn("check-01")).
			Notifies(queryMrn("check-01")).
			WithImpact(&explorer.Impact{Scoring: explorer.ScoringSystem_IGNORE_SCORE, Action: explorer.Action_IGNORE, Value: &explorer.ImpactValue{Value: 95}})
		rpTester.ReportingJobByMrn(queryMrn("check-01")).Notifies("root")
		rpTester.doTest(t, rp)
	})

	t.Run("now is after validUntil", func(t *testing.T) {
		b := parseBundle(t, bYaml)
		srv := initResolver(t, []*testAsset{
			{asset: "asset1", policies: []string{policyMrn("example1")}},
		}, []*policy.Bundle{b})
		srv.NowProvider = func() time.Time {
			return time.Date(2025, 9, 2, 0, 0, 0, 0, time.UTC)
		}
		rp, err := srv.Resolve(ctx, &policy.ResolveReq{
			PolicyMrn:    policyMrn("example1"),
			AssetFilters: []*explorer.Mquery{{Mql: "true"}},
		})
		require.NoError(t, err)
		require.NotNil(t, rp)

		rpTester := newResolvedPolicyTester(b, srv.NewCompilerConfig())
		rpTester.ExecutesQuery(queryMrn("check-01"))
		rpTester.CodeIdReportingJobForMrn(queryMrn("check-01")).
			Notifies(queryMrn("check-01")).
			WithImpact(&explorer.Impact{Value: &explorer.ImpactValue{Value: 95}})
		rpTester.ReportingJobByMrn(queryMrn("check-01")).Notifies("root")
		rpTester.doTest(t, rp)
	})
}
