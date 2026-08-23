// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reportmodel

import (
	"encoding/json"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/internal/reportfixture"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
)

// The fixtures are named rather than pathed, because the two no longer come from
// the same place. report-ubuntu is one asset that scanned, and it is shared with
// every reporter through internal/reportfixture; report-k8s is fifteen that did
// not, and it still lives with the reporter tests.
const (
	fixtureUbuntu = "report-ubuntu"
	fixtureK8s    = "report-k8s"
)

func loadCollection(t *testing.T, fixture string) *policy.ReportCollection {
	t.Helper()
	if fixture == fixtureUbuntu {
		res, err := reportfixture.UbuntuScan()
		require.NoError(t, err)
		return res
	}

	raw, err := os.ReadFile("../reporter/testdata/" + fixture + ".json")
	require.NoError(t, err)

	res := &policy.ReportCollection{}
	require.NoError(t, json.Unmarshal(raw, res))
	return res
}

func TestNilCollection(t *testing.T) {
	report := New(nil)
	require.NotNil(t, report)
	assert.Empty(t, report.Assets)
	assert.Nil(t, report.Collection())
	assert.Nil(t, report.Asset("//no/such/asset"))
}

func TestEmptyCollection(t *testing.T) {
	report := New(&policy.ReportCollection{})
	require.NotNil(t, report)
	assert.Empty(t, report.Assets)
	assert.Equal(t, 0, report.AssetCounts.Total)
	assert.Equal(t, 0, report.CheckCounts.Total)
}

// The happy path: one asset, one report, no errors.
func TestUbuntuAsset(t *testing.T) {
	report := New(loadCollection(t, fixtureUbuntu))

	require.Len(t, report.Assets, 1)
	asset := report.Assets[0]

	assert.Equal(t, "ubuntu:24.04", asset.Name)
	assert.True(t, asset.Scanned())
	assert.Empty(t, asset.ScanError)
	require.NotNil(t, asset.Platform)
	assert.Equal(t, "ubuntu", asset.Platform.Name)
	assert.Equal(t, "Ubuntu 24.04.2 LTS", asset.PlatformName)

	require.NotNil(t, asset.Score)
	assert.Equal(t, policy.ScoreType_Result, asset.Score.Type)
	assert.Equal(t, StatusFail, asset.Status)

	assert.Same(t, asset, report.Asset(asset.Mrn))

	// The checks come from the reporting jobs, not from Report.Scores. The
	// score map is much larger than the check set because it also holds policy,
	// control and execution-query scores.
	require.NotEmpty(t, asset.Checks)
	rawScores := len(report.Collection().Reports[asset.Mrn].Scores)
	assert.Greater(t, rawScores, len(asset.Checks),
		"Report.Scores holds more than the asset's checks; the model must not equate them")

	// Every check resolved to a real query, ran on this asset and is only
	// listed once.
	resolved := report.Collection().ResolvedPolicies[asset.Mrn]
	require.NotNil(t, resolved)
	seen := map[string]bool{}
	for _, check := range asset.Checks {
		require.NotNil(t, check.Query, "check %q has no query", check.Title)
		assert.NotEmpty(t, check.CodeId)
		assert.Contains(t, resolved.CollectorJob.ReportingQueries, check.CodeId,
			"check %q did not run on this asset", check.Title)
		assert.False(t, seen[check.Mrn], "check %q listed twice", check.Mrn)
		seen[check.Mrn] = true
		assert.NotEmpty(t, check.Policies, "check %q is not reachable from any policy", check.Title)
	}

	assert.Equal(t, len(asset.Checks), asset.Counts.Total)
	assert.Equal(t, asset.Counts, report.CheckCounts)
	assert.Equal(t, 1, report.AssetCounts.Total)
	assert.Equal(t, 1, report.AssetCounts.Failed)
}

// The reporting-job traversal must agree with the reference implementation in
// print_compact.go: CHECK / CHECK_AND_DATA_QUERY jobs, resolved through the
// bundle and confirmed against ReportingQueries.
func TestChecksMatchReportingJobs(t *testing.T) {
	collection := loadCollection(t, fixtureUbuntu)
	report := New(collection)
	require.Len(t, report.Assets, 1)
	asset := report.Assets[0]

	resolved := collection.ResolvedPolicies[asset.Mrn]
	require.NotNil(t, resolved)
	bundle := collection.Bundle.ToMap()

	expected := map[string]bool{}
	for _, job := range resolved.CollectorJob.ReportingJobs {
		if job.Type != policy.ReportingJob_CHECK && job.Type != policy.ReportingJob_CHECK_AND_DATA_QUERY {
			continue
		}
		query, ok := bundle.Queries[job.QrId]
		if !ok || query == nil || query.CodeId == "" {
			continue
		}
		if _, ok := resolved.CollectorJob.ReportingQueries[query.CodeId]; !ok {
			continue
		}
		expected[query.Mrn] = true
	}
	require.NotEmpty(t, expected)

	got := map[string]bool{}
	for _, check := range asset.Checks {
		got[check.Mrn] = true
	}
	assert.Equal(t, expected, got)
}

// Failed and errored are two outcomes, not one: the ubuntu fixture has both,
// plus passes. (Skipped and unscored checks are covered by
// TestFourScoreStatesAreDistinguished, which builds them explicitly - the ubuntu
// scan has skipped and unscored *scores*, but they belong to policies and data
// queries, not to any of its 24 checks.)
func TestUbuntuCheckStates(t *testing.T) {
	report := New(loadCollection(t, fixtureUbuntu))
	require.Len(t, report.Assets, 1)
	asset := report.Assets[0]

	byStatus := map[Status][]*Check{}
	for _, check := range asset.Checks {
		byStatus[check.Status] = append(byStatus[check.Status], check)
	}

	for _, status := range []Status{StatusPass, StatusFail, StatusError} {
		assert.NotEmpty(t, byStatus[status], "no check with status %s in the fixture", status)
	}

	// A failing check is a result score below 100 - not a score type of its own.
	for _, check := range byStatus[StatusFail] {
		require.NotNil(t, check.Score)
		assert.Equal(t, policy.ScoreType_Result, check.Score.Type)
		assert.NotEqual(t, uint32(100), check.Score.Value)
	}
	for _, check := range byStatus[StatusPass] {
		require.NotNil(t, check.Score)
		assert.Equal(t, policy.ScoreType_Result, check.Score.Type)
		assert.Equal(t, uint32(100), check.Score.Value)
	}
	// An errored check has a score type of its own, so it can never be mistaken
	// for a pass - the score value of an error is 0, which alone would look
	// like a total failure rather than a check that proved nothing.
	for _, check := range byStatus[StatusError] {
		assert.Equal(t, policy.ScoreType_Error, check.Score.Type)
	}

	assert.Equal(t, len(byStatus[StatusPass]), asset.Counts.Passed)
	assert.Equal(t, len(byStatus[StatusFail]), asset.Counts.Failed)
	assert.Equal(t, len(byStatus[StatusError]), asset.Counts.Errored)
	assert.Equal(t, len(byStatus[StatusSkipped]), asset.Counts.Skipped)
	assert.Equal(t, len(byStatus[StatusUnscored]), asset.Counts.Unscored)
	assert.Equal(t, len(byStatus[StatusFail])+len(byStatus[StatusError]), asset.Counts.Findings())

	// A skipped or unscored check is not a finding; a failed or errored one is.
	assert.True(t, StatusFail.IsFinding())
	assert.True(t, StatusError.IsFinding())
	assert.False(t, StatusSkipped.IsFinding())
	assert.False(t, StatusUnscored.IsFinding())
	assert.False(t, StatusPass.IsFinding())
	assert.False(t, StatusUnknown.IsFinding())
}

func TestPolicyTree(t *testing.T) {
	report := New(loadCollection(t, fixtureUbuntu))
	require.Len(t, report.Assets, 1)
	asset := report.Assets[0]

	require.NotEmpty(t, asset.Policies)

	names := []string{}
	inTree := map[string]bool{}
	total := 0
	for _, p := range asset.Policies {
		names = append(names, p.Name)
		assert.NotEmpty(t, p.Checks, "policy %q has no checks", p.Name)
		assert.Equal(t, len(p.Checks), p.Counts.Total)
		total += len(p.Checks)
		for _, check := range p.Checks {
			inTree[check.Mrn] = true
		}
		// A policy of a scanned asset carries its own per-asset score.
		if p.Mrn != "" {
			assert.NotNil(t, p.Score, "policy %q has no score", p.Name)
		}
	}

	assert.Contains(t, names, "Mondoo Linux Security")
	assert.Contains(t, names, "Mondoo SSH Server Security")
	// The space is a policy in the bundle too, but a check is attributed to the
	// policy that declares it, never to an ancestor of that policy.
	assert.NotContains(t, names, "test-infallible-taussig-796596")

	// Every check of the asset is reachable through the tree, and the tree
	// introduces none the asset does not have.
	assert.Len(t, inTree, len(asset.Checks))
	assert.GreaterOrEqual(t, total, len(asset.Checks))
	for _, check := range asset.Checks {
		assert.True(t, inTree[check.Mrn], "check %q unreachable in the policy tree", check.Title)
	}
}

func TestSeverityAndRisk(t *testing.T) {
	report := New(loadCollection(t, fixtureUbuntu))
	asset := report.Assets[0]

	withImpact := 0
	for _, check := range asset.Checks {
		assert.Contains(t,
			[]string{
				policy.ScoreRatingTextCritical, policy.ScoreRatingTextHigh,
				policy.ScoreRatingTextMedium, policy.ScoreRatingTextLow,
				policy.ScoreRatingTextNone,
			},
			check.Severity, "check %q has an unexpected severity", check.Title)

		if check.HasImpact {
			withImpact++
			assert.NotEqual(t, policy.ScoreRatingTextNone, check.Severity)
		}

		// Risk is the realized risk of the score, not the declared severity.
		if check.Status == StatusPass {
			assert.Equal(t, int32(0), check.Risk)
		}
	}
	assert.NotZero(t, withImpact, "no check in the fixture declares an impact")
}

func TestDeterministicOrder(t *testing.T) {
	collection := loadCollection(t, fixtureUbuntu)

	first := New(collection)
	for i := 0; i < 5; i++ {
		other := New(collection)
		require.Len(t, other.Assets, len(first.Assets))
		for a := range first.Assets {
			assert.Equal(t, first.Assets[a].Mrn, other.Assets[a].Mrn)
			assert.Equal(t, checkTitles(first.Assets[a].Checks), checkTitles(other.Assets[a].Checks))
			assert.Equal(t, policyNames(first.Assets[a].Policies), policyNames(other.Assets[a].Policies))
		}
	}
}

func TestDetail(t *testing.T) {
	report := New(loadCollection(t, fixtureUbuntu))
	asset := report.Assets[0]

	var failing, errored *Check
	for _, check := range asset.Checks {
		if failing == nil && check.Status == StatusFail {
			failing = check
		}
		if errored == nil && check.Status == StatusError {
			errored = check
		}
	}
	require.NotNil(t, failing, "the fixture has no failing check")
	require.NotNil(t, errored, "the fixture has no errored check")

	detail := failing.Detail()
	assert.Equal(t, failing.Title, detail.Title)
	assert.Equal(t, StatusFail, detail.Status)
	assert.Equal(t, policy.ScoreRatingTextCritical, detail.Severity)
	assert.NotEmpty(t, detail.Description, "check %q has no description", failing.Title)
	assert.NotEmpty(t, detail.Mql)
	assert.NotEmpty(t, detail.Remediation, "check %q has no remediation", failing.Title)
	assert.NotEmpty(t, detail.Policies)
	assert.Empty(t, detail.Error, "a failing check is not an errored one")

	// An errored check explains itself through the score message.
	errDetail := errored.Detail()
	assert.Equal(t, StatusError, errDetail.Status)
	assert.NotEmpty(t, errDetail.Error, "errored check %q has no error message", errored.Title)

	// Remediation, references and the assessment are exposed structurally, not
	// as one formatted string.
	assessed, remediated, referenced, audited := 0, 0, 0, 0
	for _, check := range asset.Checks {
		d := check.Detail()
		if d.Assessment != "" {
			assessed++
		}
		if d.Audit != "" {
			audited++
		}
		for _, item := range d.Remediation {
			assert.NotEmpty(t, item.Desc)
			remediated++
		}
		for _, ref := range d.References {
			assert.NotEmpty(t, ref.Url)
			referenced++
		}
		for k := range d.Compliance {
			assert.True(t, strings.HasPrefix(k, "compliance/"))
		}
		for _, s := range []string{d.Description, d.Mql, d.Assessment, d.Audit, d.Error} {
			assert.NotContains(t, s, "\r", "detail of %q carries a carriage return into the renderer", check.Title)
		}
	}
	assert.Equal(t, 15, assessed, "expected 15 of the 24 checks to have an assessment")
	assert.Equal(t, 2, audited, "expected 2 checks with manual audit steps")
	assert.NotZero(t, remediated, "no check in the fixture has remediation")
	assert.NotZero(t, referenced, "no check in the fixture has references")
}

func TestDetailOfNilCheck(t *testing.T) {
	var check *Check
	assert.Equal(t, CheckDetail{Status: StatusUnknown}, check.Detail())
}

// The model must not rewrite the scan's scores. print_compact.go raises the
// value of a low-impact failing check in place as a documented workaround; doing
// that here would leak a mutated collection back to whoever loaded it.
func TestScoresAreNotMutated(t *testing.T) {
	collection := loadCollection(t, fixtureUbuntu)

	before := map[string]uint32{}
	for mrn, score := range collection.Reports {
		for k, v := range score.Scores {
			before[mrn+"/"+k] = v.Value
		}
	}

	report := New(collection)
	for _, asset := range report.Assets {
		for _, check := range asset.Checks {
			_ = check.Detail()
		}
	}

	for mrn, score := range collection.Reports {
		for k, v := range score.Scores {
			assert.Equal(t, before[mrn+"/"+k], v.Value, "score %s/%s was mutated", mrn, k)
		}
	}
}

// Fifteen assets, no reports, fifteen errors. An asset that failed to scan is
// not an asset without findings.
func TestErroredAssets(t *testing.T) {
	collection := loadCollection(t, fixtureK8s)
	require.Len(t, collection.Assets, 15)
	require.Empty(t, collection.Reports)
	require.Len(t, collection.Errors, 15)
	require.Nil(t, collection.Bundle, "the k8s fixture has no bundle; the model must not require one")

	report := New(collection)
	require.Len(t, report.Assets, 15)

	assert.Equal(t, 15, report.AssetCounts.Total)
	assert.Equal(t, 15, report.AssetCounts.Errored)
	assert.Equal(t, 0, report.AssetCounts.Passed)
	assert.Equal(t, 0, report.AssetCounts.Failed)
	assert.Equal(t, 0, report.CheckCounts.Total)

	for _, asset := range report.Assets {
		assert.Equal(t, StatusError, asset.Status, "asset %q should be errored", asset.Name)
		assert.True(t, asset.Status.IsFinding(), "an errored asset is something to act on")
		assert.NotEmpty(t, asset.ScanError, "asset %q lost its scan error", asset.Name)
		assert.False(t, asset.Scanned())

		// The synthesized score carries the failure, exactly like the summary
		// renderer's.
		require.NotNil(t, asset.Score)
		assert.Equal(t, policy.ScoreType_Error, asset.Score.Type)
		assert.Equal(t, asset.ScanError, asset.Score.Message)
		assert.Equal(t, asset.Mrn, asset.Score.QrId)

		assert.Empty(t, asset.Checks)
		assert.Empty(t, asset.Policies)
		assert.Equal(t, 0, asset.Counts.Total)
	}

	// Names and platforms survive even though nothing scanned, so the viewer can
	// tell the user which assets failed, and the assets come out sorted.
	assert.Equal(t, "K8s Cluster minikube", report.Assets[0].Name)
	assert.Equal(t, "kube-system/coredns", report.Assets[1].Name)
	assert.True(t, sort.SliceIsSorted(report.Assets, func(i, j int) bool {
		return report.Assets[i].Name < report.Assets[j].Name
	}), "assets are not sorted by name")
	for _, asset := range report.Assets {
		assert.NotEqual(t, asset.Mrn, asset.Name, "asset %q lost its name", asset.Mrn)
		assert.NotEmpty(t, asset.PlatformName)
	}
}

// An asset that has neither a report nor an error is unknown, not passing.
func TestAssetWithNeitherReportNorError(t *testing.T) {
	report := New(&policy.ReportCollection{
		Assets: map[string]*inventory.Asset{
			"//assets/silent": {Mrn: "//assets/silent", Name: "silent"},
		},
	})

	require.Len(t, report.Assets, 1)
	asset := report.Assets[0]
	assert.Equal(t, "silent", asset.Name)
	assert.Equal(t, StatusUnknown, asset.Status)
	assert.Empty(t, asset.ScanError)
	require.NotNil(t, asset.Score)
	assert.Equal(t, policy.ScoreType_Unknown, asset.Score.Type)
	assert.Equal(t, 1, report.AssetCounts.Unknown)
	assert.Equal(t, 0, report.AssetCounts.Passed)
}

func TestAssetOnlyKnownByItsError(t *testing.T) {
	// An error for an MRN that is not in the inventory still produces an asset:
	// dropping it would report a failed scan as a scan of nothing.
	report := New(&policy.ReportCollection{
		Errors: map[string]string{"//assets/ghost": "could not connect"},
	})

	require.Len(t, report.Assets, 1)
	asset := report.Assets[0]
	assert.Equal(t, "//assets/ghost", asset.Mrn)
	assert.Equal(t, "//assets/ghost", asset.Name)
	assert.Equal(t, StatusError, asset.Status)
	assert.Equal(t, "could not connect", asset.ScanError)
}

func checkTitles(checks []*Check) []string {
	res := make([]string, 0, len(checks))
	for _, check := range checks {
		res = append(res, check.Title)
	}
	return res
}

func policyNames(policies []*Policy) []string {
	res := make([]string, 0, len(policies))
	for _, p := range policies {
		res = append(res, p.Name)
	}
	return res
}
