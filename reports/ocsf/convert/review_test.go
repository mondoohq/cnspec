// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package convert

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/internal/reportfixture"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/cnspec/reports/ocsf"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
)

// TestOcsfSeverityFollowsDeclaredImpact pins that a failing check carries the
// severity its author gave it.
//
// A leaf check scores exactly 100 or 0 -- policy/executor scores it as one or the
// other, never in between -- so its risk is 100 whenever it fails, and a severity
// derived from that risk is Critical for every failing check in every scan. The
// declared impact is the only thing that tells one failing check from another,
// and it is what the SARIF reporter has always used.
//
// Asserting over four impacts in one scan is the point: a single failing check
// would still pass with the risk mapping, since risk 100 and impact 95 both land
// in the Critical band.
func TestOcsfSeverityFollowsDeclaredImpact(t *testing.T) {
	impacts := map[string]struct {
		impact       int32
		wantSeverity int
	}{
		"critical": {95, ocsf.SeverityCritical},
		"high":     {80, ocsf.SeverityHigh},
		"medium":   {50, ocsf.SeverityMedium},
		"low":      {20, ocsf.SeverityLow},
	}

	report := failingChecksWithImpacts(impacts)

	t.Run("compliance", func(t *testing.T) {
		events := toOcsf(t, report)
		require.Len(t, events.ComplianceFindings, len(impacts))
		for _, finding := range events.ComplianceFindings {
			want := impacts[finding.Compliance.Control].wantSeverity
			assert.Equal(t, want, finding.SeverityID,
				"check %q declares impact %d", finding.Compliance.Control,
				impacts[finding.Compliance.Control].impact)
			assert.Equal(t, ocsf.SeverityName(want), finding.Severity)
		}
	})

	t.Run("detection agrees with impact_id", func(t *testing.T) {
		events, err := convertAt(report,
			Options{Version: ocsf.DefaultVersion, Findings: ocsf.FindingsDetection}, fixedScanTime)
		require.NoError(t, err)
		require.Len(t, events.DetectionFindings, len(impacts))

		for _, finding := range events.DetectionFindings {
			// Class 2004 carries both severity_id and impact_id. They run on the
			// same bands, so one event saying Critical severity and High impact was
			// the event contradicting itself.
			assert.Equal(t, impactLevel(int32(finding.ImpactScore)), finding.ImpactID)
			assert.Equal(t, severityFromRisk(int32(finding.ImpactScore)), finding.SeverityID,
				"severity and impact must not disagree inside one event")
		}
	})
}

// TestOcsfSeverityFallsBackToRisk covers a check with no declared impact, where
// the risk is the only signal there is.
func TestOcsfSeverityFallsBackToRisk(t *testing.T) {
	events := toOcsf(t, reportfixture.Detailed())
	require.Len(t, events.ComplianceFindings, 1)
	assert.Equal(t, ocsf.SeverityCritical, events.ComplianceFindings[0].SeverityID)
}

// failingChecksWithImpacts is a scan of one asset where every check fails and
// each declares a different impact. The check's compliance control is its name,
// which is what the assertions key on.
func failingChecksWithImpacts(impacts map[string]struct {
	impact       int32
	wantSeverity int
},
) *policy.ReportCollection {
	const assetMrn = "//assets.api.mondoo.app/spaces/test/assets/impacts"

	res := &policy.ReportCollection{
		Assets: map[string]*inventory.Asset{
			assetMrn: {Name: "X1", Platform: &inventory.Platform{Name: "ubuntu"}},
		},
		ResolvedPolicies: map[string]*policy.ResolvedPolicy{
			assetMrn: {CollectorJob: &policy.CollectorJob{ReportingQueries: map[string]*policy.StringArray{}}},
		},
		Bundle:  &policy.Bundle{},
		Reports: map[string]*policy.Report{assetMrn: {Scores: map[string]*policy.Score{}}},
	}

	for name, tc := range impacts {
		codeID := "code-" + name
		res.ResolvedPolicies[assetMrn].CollectorJob.ReportingQueries[codeID] = nil
		res.Bundle.Queries = append(res.Bundle.Queries, &policy.Mquery{
			Mrn:    "//policy.api.mondoo.app/queries/" + name,
			CodeId: codeID,
			Title:  "Check " + name,
			Impact: &policy.Impact{Value: &policy.ImpactValue{Value: tc.impact}},
			Tags:   map[string]string{"compliance/test": name},
		})
		// Value 0 is what a failing leaf check scores; that is the whole point.
		res.Reports[assetMrn].Scores[codeID] = &policy.Score{Type: policy.ScoreType_Result, Value: 0}
	}
	return res
}

// TestOcsfFindingUIDIsPerAsset pins that finding_info.uid identifies a check on
// an asset rather than the check alone.
//
// With the rule id alone there, count(DISTINCT finding_info.uid) over a fleet
// scan returns the number of checks however many assets were scanned, and any
// consumer keyed on the uid collapses every asset's copy onto one row.
func TestOcsfFindingUIDIsPerAsset(t *testing.T) {
	const assets, checks = 3, 2
	events := toOcsf(t, largeReportCollection(assets, checks))
	require.Len(t, events.ComplianceFindings, assets*checks)

	uids := map[string]bool{}
	for _, finding := range events.ComplianceFindings {
		uids[finding.FindingInfo.UID] = true
		require.Len(t, finding.Resources, 1)
		rule := strings.TrimPrefix(finding.Unmapped["query_mrn"], "//policy.api.mondoo.app/queries/")
		assert.Equal(t, rule+"/"+finding.Resources[0].UID, finding.FindingInfo.UID,
			"the uid is the rule and the asset, and both stay recoverable from it")
	}
	assert.Len(t, uids, assets*checks, "one uid per check per asset")

	// The rule id itself stays where a consumer looks for the rule.
	detection, err := convertAt(largeReportCollection(assets, checks),
		Options{Findings: ocsf.FindingsDetection}, fixedScanTime)
	require.NoError(t, err)
	analytics := map[string]bool{}
	for _, finding := range detection.DetectionFindings {
		require.NotNil(t, finding.FindingInfo.Analytic)
		analytics[finding.FindingInfo.Analytic.UID] = true
	}
	assert.Len(t, analytics, checks, "analytic.uid is the rule, so it is shared across assets")
}

// TestOcsfAssetErrorUIDIsPerAsset covers the finding that stands in for a whole
// failed scan. A fleet scan where a hundred assets were unreachable is a hundred
// findings, not one repeated -- but every one of them used to carry the literal
// "asset-error".
func TestOcsfAssetErrorUIDIsPerAsset(t *testing.T) {
	report := largeReportCollection(2, 1)
	report.Errors = map[string]string{}
	for mrn := range report.Assets {
		report.Errors[mrn] = "could not connect to the asset"
	}

	events := toOcsf(t, report)

	uids := map[string]bool{}
	for _, finding := range events.ComplianceFindings {
		if finding.FindingInfo.Title == "Asset scan error" {
			uids[finding.FindingInfo.UID] = true
		}
	}
	assert.Len(t, uids, 2, "each unreachable asset is its own finding")
}

// TestOcsfGcpAsset pins the mapping of a GCP asset's project id.
//
// project_uid is deprecated in OCSF 1.9 in favor of account.uid, and it was the
// only place cnspec put a GCP project -- so at 1.9 the validator warned on every
// event of a GCP asset, and dropping the deprecated attribute alone would have
// lost the project id entirely.
func TestOcsfGcpAsset(t *testing.T) {
	report := gcpAssetReportCollection()

	v13 := toOcsf(t, report)
	cloud := v13.InventoryInfos[0].Cloud
	require.NotNil(t, cloud)
	assert.Equal(t, "GCP", cloud.Provider)
	require.NotNil(t, cloud.Account, "the project is the account of a GCP asset")
	assert.Equal(t, "mondoo-dev-262313", cloud.Account.UID)
	assert.Equal(t, "mondoo-dev-262313", cloud.ProjectUID,
		"1.3 still has project_uid, and a 1.3 consumer reads it there")

	events19, err := convertAt(report, Options{Version: ocsf.Version190}, fixedScanTime)
	require.NoError(t, err)
	cloud19 := events19.InventoryInfos[0].Cloud
	require.NotNil(t, cloud19)
	require.NotNil(t, cloud19.Account)
	assert.Equal(t, "mondoo-dev-262313", cloud19.Account.UID)
	assert.Empty(t, cloud19.ProjectUID,
		"project_uid is deprecated at 1.9; the validator warns on every event that carries it")
}

// gcpAssetReportCollection is the sample scan re-cast as a GCP compute instance,
// so the deprecation gate on cloud.project_uid gets exercised. The only cloud
// fixture before it was an EC2 instance, which has no project id at all -- so the
// schema validator ran green over a mapping it never saw.
func gcpAssetReportCollection() *policy.ReportCollection {
	report := reportfixture.Sample()
	for _, asset := range report.Assets {
		asset.PlatformIds = []string{
			"//platformid.api.mondoo.app/runtime/gcp/projects/mondoo-dev-262313/instances/1234567890",
		}
		asset.Platform = &inventory.Platform{
			Name: "ubuntu", Runtime: "gcp-compute-instance", Kind: "virtualmachine",
			Family: []string{"debian", "linux", "unix", "os"}, Version: "22.04", Arch: "amd64",
		}
	}
	return report
}

// TestOcsfVulnMarksANonCVEIdentifier covers an advisory with no CVEs at 1.3.
//
// cve.uid is specified as the CVE-YYYY-NNNNN form, and 1.3 has no advisory object
// to put a USN or RHSA in instead -- so the id stays in cve.uid, where a 1.3
// consumer looks for it, and unmapped says it is not a CVE. Without the marker an
// NVD join over cve.uid drops the row and reports nothing.
func TestOcsfVulnMarksANonCVEIdentifier(t *testing.T) {
	report := advisoryReportCollection()
	report.VulnReports[reportfixture.AssetMrn].Advisories[0].Cves = nil

	v13 := toOcsf(t, report)
	require.Len(t, v13.VulnerabilityFindings, 1)
	finding := v13.VulnerabilityFindings[0]
	require.NotNil(t, finding.Vulnerabilities[0].CVE)
	assert.Equal(t, "USN-1234-1", finding.Vulnerabilities[0].CVE.UID)
	assert.Equal(t, "USN-1234-1", finding.Unmapped["non_cve_uid"],
		"a consumer joining cve.uid against NVD has to be able to tell this apart from a CVE")

	// A real CVE carries no marker.
	withCVE := toOcsf(t, advisoryReportCollection())
	require.Len(t, withCVE.VulnerabilityFindings, 1)
	assert.NotContains(t, withCVE.VulnerabilityFindings[0].Unmapped, "non_cve_uid")

	// 1.9 has the advisory object, so cve.uid is not misused there at all.
	events19, err := convertAt(report, Options{Version: ocsf.Version190}, fixedScanTime)
	require.NoError(t, err)
	assert.NotContains(t, events19.VulnerabilityFindings[0].Unmapped, "non_cve_uid")
}

// TestOcsfVulnCvssScoreIsACvssScore pins the scale of unmapped.cvss_score.
//
// cnspec scores an advisory 0-100, so a CVSS 9.5 advisory carried "95" under a
// key named cvss_score: every advisory in the report matched a `cvss_score > 7`
// filter, and nothing about the value said it was on another scale.
func TestOcsfVulnCvssScoreIsACvssScore(t *testing.T) {
	events := toOcsf(t, advisoryReportCollection())
	require.Len(t, events.VulnerabilityFindings, 1)
	assert.Equal(t, "9.5", events.VulnerabilityFindings[0].Unmapped["cvss_score"],
		"the advisory scores 95 on cnspec's 0-100 scale, which is CVSS 9.5")

	// A whole number keeps its decimal. Shortest-round-trip formatting renders
	// 100 as "10", which reads like a different scale again next to "9.5", and
	// CVSS is published with one decimal everywhere.
	//
	// Only non-zero scores: a zero advisory score falls back to the package's,
	// so it cannot be used to probe the formatting.
	for score, want := range map[int32]string{100: "10.0", 70: "7.0", 55: "5.5"} {
		r := advisoryReportCollection()
		r.VulnReports[reportfixture.AssetMrn].Advisories[0].Score = score
		got := toOcsf(t, r).VulnerabilityFindings[0].Unmapped["cvss_score"]
		assert.Equal(t, want, got, "cnspec score %d", score)
	}
}

// TestConvertToDirEmptyScanKeepsPreviousOutput covers a conversion that produces
// no events at all -- a scan that discovered no assets, with a bundle that loaded
// fine, so the "no policy bundle found" guard does not fire.
//
// The stale sweep deletes every filename cnspec knows, so running it after an
// empty conversion emptied the directory of a previous good run and returned nil:
// the report was gone and the scan exited 0, leaving "no findings" and "all
// clean" indistinguishable downstream.
func TestConvertToDirEmptyScanKeepsPreviousOutput(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "out")

	written, err := ConvertToDir(advisoryReportCollection(), dir, Options{}, EncodingJSON)
	require.NoError(t, err)
	require.NotEmpty(t, written, "the first run writes a good report")

	// A scan that discovered nothing: no assets, and a bundle that loaded.
	empty := &policy.ReportCollection{Bundle: &policy.Bundle{}}
	written, err = ConvertToDir(empty, dir, Options{}, EncodingJSON)
	require.NoError(t, err)
	assert.Empty(t, written)

	for _, class := range []string{
		ocsf.ClassComplianceFinding, ocsf.ClassVulnerabilityFinding, ocsf.ClassInventoryInfo,
	} {
		assert.FileExists(t, filepath.Join(dir, class+".jsonl"),
			"an empty conversion must not delete the previous run's report")
	}
}

// TestConvertToDirDoesNotFollowASymlink covers a symlink pre-placed at one of the
// eight fixed, public class filenames.
//
// os.Create follows it and truncates whatever it points at, so anyone who can
// write the output directory can redirect a scan -- frequently running as root --
// onto a file of their choosing and have it overwritten with the findings.
func TestConvertToDirDoesNotFollowASymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("O_NOFOLLOW has no Windows equivalent; see nofollow_windows.go")
	}

	root := t.TempDir()
	dir := filepath.Join(root, "out")
	require.NoError(t, os.MkdirAll(dir, 0o700))

	victim := filepath.Join(root, "victim")
	require.NoError(t, os.WriteFile(victim, []byte("do not overwrite me"), 0o600))
	require.NoError(t, os.Symlink(victim, filepath.Join(dir, ocsf.ClassComplianceFinding+".jsonl")))

	_, err := ConvertToDir(advisoryReportCollection(), dir, Options{}, EncodingJSON)
	require.Error(t, err, "the open has to fail rather than follow the link")

	content, readErr := os.ReadFile(victim)
	require.NoError(t, readErr)
	assert.Equal(t, "do not overwrite me", string(content),
		"the symlink target must not be truncated and rewritten with findings")
}

// TestConvertToDirWritesPrivateFiles pins the permissions of the output.
//
// These files carry cloud account ids and ARNs, the MQL source of every check,
// rendered assessments showing the observed values, and under IncludeData the raw
// output of every data query. A scan is routinely run as root, and 0755/0644
// hands all of it to every local account on the host.
func TestConvertToDirWritesPrivateFiles(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permission bits do not apply on Windows")
	}

	dir := filepath.Join(t.TempDir(), "out")
	written, err := ConvertToDir(advisoryReportCollection(), dir, Options{}, EncodingJSON)
	require.NoError(t, err)
	require.NotEmpty(t, written)

	info, err := os.Stat(dir)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o700), info.Mode().Perm(), "the output directory is not world-readable")

	for _, path := range written {
		info, err := os.Stat(path)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm(), "%s is not world-readable", path)
	}
}

// TestConvertToDirSurvivesAnUnremovableStaleFile covers a leftover the sweep
// cannot delete -- a root-owned file from an earlier run is the ordinary way to
// get one; the test uses a non-empty directory, which os.Remove rejects the same
// way.
//
// Every file of this run is already written and closed by the time the sweep
// runs, so propagating the failure turned a complete scan into a non-zero exit
// (log.Fatal, in apps/cnspec/cmd/scan.go) over work that had already succeeded.
func TestConvertToDirSurvivesAnUnremovableStaleFile(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "out")
	require.NoError(t, os.MkdirAll(dir, 0o700))

	// A stale name this run will not write, that os.Remove cannot delete.
	stale := filepath.Join(dir, ocsf.ClassDetectionFinding+".jsonl")
	require.NoError(t, os.Mkdir(stale, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(stale, "child"), []byte("x"), 0o600))

	written, err := ConvertToDir(advisoryReportCollection(), dir, Options{}, EncodingJSON)
	require.NoError(t, err, "the scan wrote every one of its files; cleanup is not its verdict")
	assert.NotEmpty(t, written)
	assert.FileExists(t, filepath.Join(dir, ocsf.ClassComplianceFinding+".jsonl"))
}
