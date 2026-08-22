// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reportmodel

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
)

const stateAssetMrn = "//assets/states"

// stateCase is one check of the hand-built collection below, together with the
// outcome the model must report for it.
type stateCase struct {
	name  string
	score *policy.Score
	want  Status
}

// stateCollection builds a collection with one check per outcome. The ubuntu
// fixture proves the traversal against real data but only carries passing,
// failing and errored checks; skipped and unscored checks need a scan whose
// filters excluded a query, so they are constructed here.
func stateCollection(cases []stateCase) *policy.ReportCollection {
	bundle := &policy.Bundle{}
	group := &policy.PolicyGroup{}
	jobs := map[string]*policy.ReportingJob{
		"policy-job": {
			Uuid: "policy-job",
			QrId: "//policies/states",
			Type: policy.ReportingJob_POLICY,
		},
	}
	reportingQueries := map[string]*policy.StringArray{}
	scores := map[string]*policy.Score{}

	for _, c := range cases {
		mrn := "//queries/" + c.name
		codeID := "code-" + c.name

		bundle.Queries = append(bundle.Queries, &policy.Mquery{
			Mrn:    mrn,
			CodeId: codeID,
			Title:  c.name,
			Mql:    "true",
			Impact: &policy.Impact{Value: &policy.ImpactValue{Value: 80}},
		})
		group.Checks = append(group.Checks, &policy.Mquery{Mrn: mrn})

		jobs["job-"+c.name] = &policy.ReportingJob{
			Uuid:   "job-" + c.name,
			QrId:   mrn,
			Type:   policy.ReportingJob_CHECK,
			Notify: []string{"policy-job"},
		}
		reportingQueries[codeID] = &policy.StringArray{Items: []string{"job-" + c.name}}

		if c.score != nil {
			c.score.QrId = mrn
			scores[mrn] = c.score
		}
	}

	bundle.Policies = []*policy.Policy{{
		Mrn:    "//policies/states",
		Name:   "States",
		Groups: []*policy.PolicyGroup{group},
	}}

	return &policy.ReportCollection{
		Assets: map[string]*inventory.Asset{
			stateAssetMrn: {
				Mrn:      stateAssetMrn,
				Name:     "states",
				Platform: &inventory.Platform{Name: "ubuntu", Title: "Ubuntu", Family: []string{"linux"}},
			},
		},
		Bundle: bundle,
		Reports: map[string]*policy.Report{
			stateAssetMrn: {
				EntityMrn: stateAssetMrn,
				Score:     &policy.Score{QrId: stateAssetMrn, Type: policy.ScoreType_Result, Value: 40},
				Scores:    scores,
			},
		},
		ResolvedPolicies: map[string]*policy.ResolvedPolicy{
			stateAssetMrn: {
				CollectorJob: &policy.CollectorJob{
					ReportingJobs:    jobs,
					ReportingQueries: reportingQueries,
				},
			},
		},
	}
}

// The point of the model: failed, errored, skipped and unscored are four
// distinct outcomes. Collapsing any of them into "not passing" - or worse into
// "passing", which a bare score value of 100 on an unscored check invites -
// would tell the user something untrue.
func TestFourScoreStatesAreDistinguished(t *testing.T) {
	cases := []stateCase{
		{"pass", &policy.Score{Type: policy.ScoreType_Result, Value: 100}, StatusPass},
		{"fail", &policy.Score{Type: policy.ScoreType_Result, Value: 0}, StatusFail},
		{"partial", &policy.Score{Type: policy.ScoreType_Result, Value: 99}, StatusFail},
		{"error", &policy.Score{Type: policy.ScoreType_Error, Message: "boom"}, StatusError},
		{"skip", &policy.Score{Type: policy.ScoreType_Skip}, StatusSkipped},
		{"outofscope", &policy.Score{Type: policy.ScoreType_OutOfScope}, StatusSkipped},
		{"disabled", &policy.Score{Type: policy.ScoreType_Disabled}, StatusSkipped},
		{"unscored", &policy.Score{Type: policy.ScoreType_Unscored, Value: 100}, StatusUnscored},
		{"missing", nil, StatusUnknown},
	}

	report := New(stateCollection(cases))
	require.Len(t, report.Assets, 1)
	asset := report.Assets[0]
	require.Len(t, asset.Checks, len(cases))

	got := map[string]Status{}
	for _, check := range asset.Checks {
		got[check.Title] = check.Status
	}
	for _, c := range cases {
		assert.Equal(t, c.want, got[c.name], "check %q", c.name)
	}

	assert.Equal(t, Counts{
		Total:    9,
		Passed:   1,
		Failed:   2,
		Errored:  1,
		Skipped:  3,
		Unscored: 1,
		Unknown:  1,
	}, asset.Counts)
	assert.Equal(t, 3, asset.Counts.Findings())
	assert.Equal(t, asset.Counts, report.CheckCounts)

	// The four are never the same value, so a viewer switching on Status cannot
	// accidentally merge them.
	distinct := map[Status]bool{}
	for _, s := range []Status{StatusPass, StatusFail, StatusError, StatusSkipped, StatusUnscored, StatusUnknown} {
		assert.False(t, distinct[s])
		distinct[s] = true
	}
	assert.Len(t, distinct, 6)
}

// A check whose code id is not in ReportingQueries did not run on this asset -
// the bundle holds queries for platforms other than the one scanned - and must
// not show up as a check of it.
func TestChecksThatDidNotRunAreDropped(t *testing.T) {
	collection := stateCollection([]stateCase{
		{"ran", &policy.Score{Type: policy.ScoreType_Result, Value: 100}, StatusPass},
		{"other-platform", &policy.Score{Type: policy.ScoreType_Result, Value: 0}, StatusFail},
	})
	delete(collection.ResolvedPolicies[stateAssetMrn].CollectorJob.ReportingQueries, "code-other-platform")

	report := New(collection)
	require.Len(t, report.Assets, 1)
	require.Len(t, report.Assets[0].Checks, 1)
	assert.Equal(t, "ran", report.Assets[0].Checks[0].Title)
}

// Only CHECK and CHECK_AND_DATA_QUERY jobs are checks. Report.Scores also holds
// policy, control and execution-query scores, which is why iterating it instead
// of the reporting jobs is wrong.
func TestOnlyCheckJobsBecomeChecks(t *testing.T) {
	collection := stateCollection([]stateCase{
		{"check", &policy.Score{Type: policy.ScoreType_Result, Value: 100}, StatusPass},
		{"data", &policy.Score{Type: policy.ScoreType_Unscored}, StatusUnscored},
		{"control", &policy.Score{Type: policy.ScoreType_Result, Value: 0}, StatusFail},
	})
	jobs := collection.ResolvedPolicies[stateAssetMrn].CollectorJob.ReportingJobs
	jobs["job-data"].Type = policy.ReportingJob_DATA_QUERY
	jobs["job-control"].Type = policy.ReportingJob_CONTROL

	report := New(collection)
	require.Len(t, report.Assets, 1)
	require.Len(t, report.Assets[0].Checks, 1)
	assert.Equal(t, "check", report.Assets[0].Checks[0].Title)

	// A CHECK_AND_DATA_QUERY job is still a check.
	jobs["job-data"].Type = policy.ReportingJob_CHECK_AND_DATA_QUERY
	report = New(collection)
	assert.Len(t, report.Assets[0].Checks, 2)
}

// A check whose reporting job leads to no policy in the bundle is still
// reachable in the tree, under the synthetic node.
func TestChecksWithoutAPolicyAreStillReachable(t *testing.T) {
	collection := stateCollection([]stateCase{
		{"orphan", &policy.Score{Type: policy.ScoreType_Result, Value: 0}, StatusFail},
	})
	collection.ResolvedPolicies[stateAssetMrn].CollectorJob.ReportingJobs["job-orphan"].Notify = nil

	report := New(collection)
	asset := report.Assets[0]
	require.Len(t, asset.Checks, 1)
	require.Len(t, asset.Policies, 1)
	assert.Equal(t, UngroupedPolicyName, asset.Policies[0].Name)
	assert.Empty(t, asset.Policies[0].Mrn)
	assert.Len(t, asset.Policies[0].Checks, 1)
}

// The score of a check can be keyed by MRN or by code id depending on how the
// scan collected it; both have to resolve.
func TestScoreKeyedByCodeID(t *testing.T) {
	collection := stateCollection([]stateCase{
		{"byCode", &policy.Score{Type: policy.ScoreType_Result, Value: 0}, StatusFail},
	})
	report := collection.Reports[stateAssetMrn]
	score := report.Scores["//queries/byCode"]
	delete(report.Scores, "//queries/byCode")
	report.Scores["code-byCode"] = score

	model := New(collection)
	require.Len(t, model.Assets[0].Checks, 1)
	assert.Equal(t, StatusFail, model.Assets[0].Checks[0].Status)
}

// A check that appears in two policies is one check, listed under both.
func TestCheckSharedByTwoPolicies(t *testing.T) {
	collection := stateCollection([]stateCase{
		{"shared", &policy.Score{Type: policy.ScoreType_Result, Value: 0}, StatusFail},
	})
	collection.Bundle.Policies = append(collection.Bundle.Policies, &policy.Policy{
		Mrn:  "//policies/second",
		Name: "Second",
		Groups: []*policy.PolicyGroup{{
			Checks: []*policy.Mquery{{Mrn: "//queries/shared"}},
		}},
	})
	jobs := collection.ResolvedPolicies[stateAssetMrn].CollectorJob.ReportingJobs
	jobs["second-policy-job"] = &policy.ReportingJob{
		Uuid: "second-policy-job",
		QrId: "//policies/second",
		Type: policy.ReportingJob_POLICY,
	}
	jobs["job-shared"].Notify = append(jobs["job-shared"].Notify, "second-policy-job")

	report := New(collection)
	asset := report.Assets[0]

	require.Len(t, asset.Checks, 1, "a check in two policies is still one check")
	assert.Equal(t, 1, asset.Counts.Total, "a shared check must not be counted twice")

	require.Len(t, asset.Policies, 2)
	assert.Equal(t, []string{"Second", "States"}, policyNames(asset.Policies))
	for _, p := range asset.Policies {
		require.Len(t, p.Checks, 1)
		assert.Same(t, asset.Checks[0], p.Checks[0])
	}
	assert.Len(t, asset.Checks[0].Policies, 2)
}
