// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/parquet-go/parquet-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/cli/reporter/ocsf"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/providers-sdk/v1/upstream/mvd"
	"go.mondoo.com/mql/providers-sdk/v1/upstream/mvd/cvss"
)

// sampleAssetMrn is the asset of sampleReportCollection.
const sampleAssetMrn = "//assets.api.mondoo.app/spaces/dazzling-golick-767384/assets/2DRZ1cCWFyTYCArycAXHwvn1oU2"

// fixedScanTime keeps the converter's clock out of the assertions.
var fixedScanTime = time.Date(2026, 8, 22, 10, 0, 0, 0, time.UTC)

func toOcsf(t *testing.T, r *policy.ReportCollection) *ocsf.Events {
	t.Helper()
	events, err := convertToOCSF(r, ocsfConfig{version: ocsf.DefaultVersion, findings: OcsfFindingsCompliance}, fixedScanTime)
	require.NoError(t, err)
	return events
}

func findingByUID(events *ocsf.Events, uid string) *ocsf.ComplianceFinding {
	for i := range events.ComplianceFindings {
		if events.ComplianceFindings[i].FindingInfo.UID == uid {
			return &events.ComplianceFindings[i]
		}
	}
	return nil
}

func TestOcsfConverter(t *testing.T) {
	events := toOcsf(t, sampleReportCollection())

	require.Len(t, events.ComplianceFindings, 3, "one finding per reporting check")
	require.Len(t, events.InventoryInfos, 1, "one inventory event per asset")
	require.Empty(t, events.VulnerabilityFindings,
		"the affected package of this fixture has no advisory and therefore no CVE, "+
			"which OCSF cannot express as a vulnerability")

	for _, finding := range events.ComplianceFindings {
		assert.Equal(t, ocsf.ClassUIDComplianceFinding, finding.ClassUID)
		assert.Equal(t, ocsf.CategoryUIDComplianceFinding, finding.CategoryUID)
		assert.EqualValues(t, ocsf.ComplianceFindingTypeUIDCreate, finding.TypeUID)
		assert.Equal(t, fixedScanTime.UnixMilli(), finding.Time)
		assert.Equal(t, string(ocsf.Version130), finding.Metadata.Version)
		assert.Equal(t, "cnspec", finding.Metadata.Product.Name)
		assert.NotEmpty(t, finding.Compliance.Standards, "standards is a required attribute")
		require.Len(t, finding.Resources, 1)
		assert.Equal(t, "X1", finding.Resources[0].Name)
	}

	// a passing check is informational and compliant
	pass := findingByUID(events, "mondoo-linux-security-snmp-server-is-not-enabled")
	require.NotNil(t, pass)
	assert.Equal(t, ocsf.SeverityInformational, pass.SeverityID)
	assert.Equal(t, ocsf.ComplianceStatusPass, pass.Compliance.StatusID)
	assert.Equal(t, "Pass", pass.Compliance.Status)
	assert.Equal(t, ocsf.StatusNew, pass.StatusID)
	assert.Contains(t, pass.Message, "PASS")

	// an errored check is reported as a coverage gap, not as a compliance verdict
	errored := findingByUID(events, "mondoo-kubernetes-security-kubelet-event-record-qps")
	require.NotNil(t, errored)
	assert.Equal(t, ocsf.SeverityMedium, errored.SeverityID)
	assert.Equal(t, ocsf.ComplianceStatusUnknown, errored.Compliance.StatusID)
	assert.Equal(t, ocsf.StatusOther, errored.StatusID)

	// a skipped check is suppressed
	skipped := findingByUID(events, "mondoo-kubernetes-security-secure-scheduler_conf")
	require.NotNil(t, skipped)
	assert.Equal(t, ocsf.StatusSuppressed, skipped.StatusID)
	assert.Equal(t, ocsf.ComplianceStatusOther, skipped.Compliance.StatusID)

	// the asset shows up as a device, even though it is not a cloud asset
	inv := events.InventoryInfos[0]
	assert.Equal(t, ocsf.ClassUIDInventoryInfo, inv.ClassUID)
	assert.EqualValues(t, ocsf.InventoryInfoTypeUIDCollect, inv.TypeUID)
	assert.Equal(t, "X1", inv.Device.Name)
	assert.Equal(t, ocsf.DeviceTypeServer, inv.Device.TypeID, "kind baremetal is a server")
	require.NotNil(t, inv.Device.OS)
	assert.Equal(t, ocsf.OSTypeLinux, inv.Device.OS.TypeID)
	assert.Equal(t, "22.04", inv.Device.OS.Version)
	require.NotNil(t, inv.Device.HwInfo)
	assert.Equal(t, "amd64", inv.Device.HwInfo.CPUType, "1.3 carries the architecture in cpu_type")
	assert.Nil(t, inv.Cloud)
	assert.Empty(t, inv.Metadata.Profiles, "the cloud profile is only set on cloud assets")

}

func TestOcsfVulnerabilityFindings(t *testing.T) {
	events := toOcsf(t, advisoryReportCollection())
	require.Len(t, events.VulnerabilityFindings, 1, "one finding per advisory")

	vuln := events.VulnerabilityFindings[0]
	assert.Equal(t, ocsf.ClassUIDVulnerabilityFinding, vuln.ClassUID)
	assert.EqualValues(t, ocsf.VulnerabilityFindingTypeUIDCreate, vuln.TypeUID)
	assert.Equal(t, ocsf.SeverityCritical, vuln.SeverityID, "an advisory score of 95 is critical")
	assert.Equal(t, "USN-1234-1", vuln.FindingInfo.UID)

	require.Len(t, vuln.Vulnerabilities, 1, "one entry per CVE of the advisory")
	require.NotNil(t, vuln.Vulnerabilities[0].CVE)
	assert.Equal(t, "CVE-2023-0286", vuln.Vulnerabilities[0].CVE.UID)
	require.Len(t, vuln.Vulnerabilities[0].CVE.CVSS, 1)
	assert.Equal(t, "3.1", vuln.Vulnerabilities[0].CVE.CVSS[0].Version)
	assert.InDelta(t, 7.4, vuln.Vulnerabilities[0].CVE.CVSS[0].BaseScore, 0.001)

	require.Len(t, vuln.Vulnerabilities[0].AffectedPackages, 1)
	pkg := vuln.Vulnerabilities[0].AffectedPackages[0]
	assert.Equal(t, "libssl1.1", pkg.Name)
	assert.Equal(t, "1.1.1f-3ubuntu2.19", pkg.Version)
	assert.Equal(t, "1.1.1f-3ubuntu2.20", pkg.FixedInVersion)
	assert.True(t, vuln.Vulnerabilities[0].IsFixAvailable)

	// an advisory without CVEs still has to identify itself
	noCVE := advisoryReportCollection()
	noCVE.VulnReports[sampleAssetMrn].Advisories[0].Cves = nil

	v13 := toOcsf(t, noCVE)
	require.Len(t, v13.VulnerabilityFindings, 1)
	require.NotNil(t, v13.VulnerabilityFindings[0].Vulnerabilities[0].CVE,
		"OCSF 1.3 has no advisory attribute, so the advisory id goes in cve.uid")
	assert.Equal(t, "USN-1234-1", v13.VulnerabilityFindings[0].Vulnerabilities[0].CVE.UID)

	events19, err := convertToOCSF(noCVE, ocsfConfig{version: ocsf.Version190, findings: OcsfFindingsCompliance}, fixedScanTime)
	require.NoError(t, err)
	v19 := events19.VulnerabilityFindings[0].Vulnerabilities[0]
	assert.Nil(t, v19.CVE, "1.9 allows just one of advisory, cve and cwe")
	require.NotNil(t, v19.Advisory)
	assert.Equal(t, "USN-1234-1", v19.Advisory.UID)
}

func TestOcsfConverterNilReport(t *testing.T) {
	var report *policy.ReportCollection
	events, err := convertToOCSF(report, ocsfConfig{version: ocsf.DefaultVersion, findings: OcsfFindingsCompliance}, fixedScanTime)
	require.NoError(t, err)
	assert.Equal(t, 0, events.Len())
	assert.Empty(t, events.Classes())
}

func TestOcsfConverterIsDeterministic(t *testing.T) {
	first := bytes.Buffer{}
	second := bytes.Buffer{}
	require.NoError(t, toOcsf(t, sampleReportCollection()).WriteJSON(&first))
	require.NoError(t, toOcsf(t, sampleReportCollection()).WriteJSON(&second))
	assert.Equal(t, first.String(), second.String(), "two runs over the same report must match byte for byte")
}

func TestOcsfComplianceMappings(t *testing.T) {
	report := detailedReportCollection()
	report.Bundle.Queries[0].Tags = map[string]string{
		"compliance/cis-aws-foundations-benchmark-1.5.0": "1.4",
		"compliance/iso-27001-2022":                      "a-8-24",
		"compliance/disabled-framework":                  "false",
		"mondoo.com/platform":                            "aws",
	}

	events := toOcsf(t, report)
	require.Len(t, events.ComplianceFindings, 1)
	finding := events.ComplianceFindings[0]

	assert.Equal(t, []string{"cis-aws-foundations-benchmark-1.5.0", "iso-27001-2022"}, finding.Compliance.Standards,
		"framework tags become standards, disabled ones are dropped")
	assert.Equal(t, []string{"1.4", "a-8-24"}, finding.Compliance.Requirements)
	assert.Equal(t, "1.4", finding.Compliance.Control)
	assert.Equal(t, ocsf.ComplianceStatusFail, finding.Compliance.StatusID)
	assert.Equal(t, ocsf.SeverityCritical, finding.SeverityID, "a score of 0 is a risk of 100")

	// the check documentation travels with the finding
	assert.Equal(t, "Root login over SSH should be disabled.", finding.FindingInfo.Desc)
	assert.Equal(t, "https://example.com/cis", finding.FindingInfo.SrcURL)
	require.NotNil(t, finding.Remediation)
	assert.Contains(t, finding.Remediation.Desc, "Set PermitRootLogin to no in your TF config.")
	assert.NotContains(t, finding.Remediation.Desc, "Use the AWS console to fix it.",
		"remediation stays filtered to the asset's platform")
	assert.Equal(t, []string{"https://example.com/cis"}, finding.Remediation.References)
	assert.Contains(t, finding.Unmapped["mql"], "PermitRootLogin")
	assert.Equal(t, "0", finding.Unmapped["score"])
	assert.Equal(t, "100", finding.Unmapped["risk"])
}

func TestOcsfComplianceStandardsFallback(t *testing.T) {
	// A check with no compliance mapping still needs the required standards
	// attribute, so the policy it belongs to stands in for the framework.
	report := detailedReportCollection()
	report.Bundle.Policies = []*policy.Policy{
		{
			Mrn:  "//policy.api.mondoo.app/policies/test-policy",
			Name: "Test Policy",
			Uid:  "test-policy",
			Groups: []*policy.PolicyGroup{
				{Checks: []*policy.Mquery{{Mrn: "//policy.api.mondoo.app/queries/test-check"}}},
			},
		},
	}

	events := toOcsf(t, report)
	require.Len(t, events.ComplianceFindings, 1)
	assert.Equal(t, []string{"Test Policy"}, events.ComplianceFindings[0].Compliance.Standards)

	// with neither mapping nor policy there is still a value
	bare := toOcsf(t, detailedReportCollection())
	assert.Equal(t, []string{"Mondoo Policy"}, bare.ComplianceFindings[0].Compliance.Standards)
}

func TestOcsfVersionSelection(t *testing.T) {
	report := detailedReportCollection()

	v13, err := convertToOCSF(report, ocsfConfig{version: ocsf.Version130, findings: OcsfFindingsCompliance}, fixedScanTime)
	require.NoError(t, err)
	assert.Equal(t, "1.3.0", v13.ComplianceFindings[0].Metadata.Version)
	assert.Empty(t, v13.ComplianceFindings[0].Compliance.Desc,
		"compliance.desc does not exist before OCSF 1.9")

	v19, err := convertToOCSF(report, ocsfConfig{version: ocsf.Version190, findings: OcsfFindingsCompliance}, fixedScanTime)
	require.NoError(t, err)
	assert.Equal(t, "1.9.0", v19.ComplianceFindings[0].Metadata.Version)
	assert.Equal(t, "Root login over SSH should be disabled.", v19.ComplianceFindings[0].Compliance.Desc)
}

func TestOcsfAssetError(t *testing.T) {
	report := sampleReportCollection()
	assetMrn := "//assets.api.mondoo.app/spaces/dazzling-golick-767384/assets/2DRZ1cCWFyTYCArycAXHwvn1oU2"
	report.Errors = map[string]string{assetMrn: "could not connect to the asset"}

	events := toOcsf(t, report)
	errFinding := findingByUID(events, "asset-error")
	require.NotNil(t, errFinding, "an asset that failed to scan must not look clean")
	assert.Equal(t, ocsf.SeverityHigh, errFinding.SeverityID)
	assert.Equal(t, ocsf.StatusOther, errFinding.StatusID)
	assert.Equal(t, ocsf.ComplianceStatusUnknown, errFinding.Compliance.StatusID)
	assert.Contains(t, errFinding.StatusDetail, "could not connect")
}

func TestOcsfCloudAsset(t *testing.T) {
	report := sampleReportCollection()
	for _, asset := range report.Assets {
		asset.PlatformIds = []string{"arn:aws:ec2:us-east-1:123456789012:instance/i-abc"}
		asset.Platform = &inventory.Platform{
			Name: "amazonlinux", Runtime: "aws-ec2-instance", Kind: "virtualmachine",
			Family: []string{"linux", "unix", "os"}, Version: "2023",
		}
	}

	events := toOcsf(t, report)
	inv := events.InventoryInfos[0]
	require.NotNil(t, inv.Cloud)
	assert.Equal(t, "AWS", inv.Cloud.Provider)
	assert.Equal(t, "us-east-1", inv.Cloud.Region)
	require.NotNil(t, inv.Cloud.Account)
	assert.Equal(t, "123456789012", inv.Cloud.Account.UID)
	assert.Equal(t, []string{"cloud"}, inv.Metadata.Profiles)
	assert.Equal(t, ocsf.DeviceTypeVirtual, inv.Device.TypeID)

	finding := events.ComplianceFindings[0]
	assert.Equal(t, "aws", finding.Resources[0].CloudPartition)
	assert.Equal(t, "us-east-1", finding.Resources[0].Region)
}

// TestOcsfRequiredAttributes pins the attributes OCSF marks as required for the
// classes cnspec emits. Dropping one of them makes a lake reject the record, and
// nothing else in the test suite would notice.
func TestOcsfRequiredAttributes(t *testing.T) {
	report := erroredReportCollection()

	buf := bytes.Buffer{}
	require.NoError(t, toOcsf(t, report).WriteJSON(&buf))

	required := map[float64][]string{
		ocsf.ClassUIDComplianceFinding:    {"activity_id", "category_uid", "class_uid", "type_uid", "time", "severity_id", "metadata", "compliance", "finding_info"},
		ocsf.ClassUIDVulnerabilityFinding: {"activity_id", "category_uid", "class_uid", "type_uid", "time", "severity_id", "metadata", "finding_info", "vulnerabilities"},
		ocsf.ClassUIDInventoryInfo:        {"activity_id", "category_uid", "class_uid", "type_uid", "time", "severity_id", "metadata", "device"},
	}

	seen := map[float64]bool{}
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		var event map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &event))

		classUID, ok := event["class_uid"].(float64)
		require.True(t, ok, "every event names its class")
		seen[classUID] = true

		for _, attr := range required[classUID] {
			assert.Contains(t, event, attr, "class %v is missing the required attribute %q", classUID, attr)
		}

		// finding_info.uid and .title identify the finding downstream
		if info, ok := event["finding_info"].(map[string]any); ok {
			assert.NotEmpty(t, info["uid"])
			assert.NotEmpty(t, info["title"])
		}
		if compliance, ok := event["compliance"].(map[string]any); ok {
			assert.NotEmpty(t, compliance["standards"])
		}
	}

	assert.Len(t, seen, 3, "the fixture must exercise all three event classes")
}

func TestOcsfFileHandlerDirectory(t *testing.T) {
	for _, tc := range []struct {
		format Format
		ext    string
	}{
		{FormatOcsfJson, ".jsonl"},
		{FormatOcsfParquet, ".parquet"},
	} {
		dir := filepath.Join(t.TempDir(), "out")
		conf := defaultPrintConfig()
		conf.format = tc.format

		handler := &ocsfFileHandler{target: dir, conf: conf}
		require.NoError(t, handler.WriteReport(t.Context(), advisoryReportCollection()))

		for _, class := range []string{ocsf.ClassComplianceFinding, ocsf.ClassVulnerabilityFinding, ocsf.ClassInventoryInfo} {
			path := filepath.Join(dir, class+tc.ext)
			info, err := os.Stat(path)
			require.NoError(t, err, "one file per event class")
			assert.NotZero(t, info.Size())
		}
	}
}

func TestOcsfFileHandlerSingleFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "report.jsonl")
	conf := defaultPrintConfig()
	conf.format = FormatOcsfJson

	handler := &ocsfFileHandler{target: "file://" + path, conf: conf}
	require.NoError(t, handler.WriteReport(t.Context(), advisoryReportCollection()))

	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	assert.Len(t, lines, 5, "3 checks + 1 vulnerability + 1 inventory event, all in one stream")
}

func TestOcsfMillis(t *testing.T) {
	assert.Equal(t, int64(0), ocsfMillis(0))
	assert.Equal(t, int64(1700000000000), ocsfMillis(1700000000), "seconds are scaled up")
	assert.Equal(t, int64(1700000000000), ocsfMillis(1700000000000), "milliseconds are left alone")
}

func TestOcsfParseTime(t *testing.T) {
	assert.Equal(t, int64(0), ocsfParseTime(""))
	assert.Equal(t, int64(0), ocsfParseTime("not a date"), "an unparsable date must not become the year 1")
	assert.Equal(t, time.Date(2023, 1, 2, 3, 4, 5, 0, time.UTC).UnixMilli(), ocsfParseTime("2023-01-02T03:04:05Z"))
	assert.Equal(t, time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC).UnixMilli(), ocsfParseTime("2023-01-02"))
}

func TestVulnReportToOCSFJSON(t *testing.T) {
	report := advisoryReportCollection().VulnReports[sampleAssetMrn]

	buf := bytes.Buffer{}
	require.NoError(t, VulnReportToOCSFJSON("X1", report, ocsf.DefaultVersion, &buf))

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	require.Len(t, lines, 1, "the package is covered by the advisory, so it is not reported twice")

	var event map[string]any
	require.NoError(t, json.Unmarshal([]byte(lines[0]), &event))
	assert.EqualValues(t, ocsf.ClassUIDVulnerabilityFinding, event["class_uid"])
	assert.Equal(t, "Critical", event["severity"])
	assert.Equal(t, "X1", event["resources"].([]any)[0].(map[string]any)["name"])

	vulns := event["vulnerabilities"].([]any)
	require.Len(t, vulns, 1)
	vuln := vulns[0].(map[string]any)
	assert.Equal(t, "CVE-2023-0286", vuln["cve"].(map[string]any)["uid"])
	assert.Equal(t, true, vuln["is_fix_available"])
	assert.Contains(t, vuln["references"], "https://ubuntu.com/security/notices/USN-1234-1")
}

// cloudAssetReportCollection is the sample scan with the asset re-cast as an EC2
// instance, so the cloud profile and the ARN-derived fields get validated too.
func cloudAssetReportCollection() *policy.ReportCollection {
	report := sampleReportCollection()
	for _, asset := range report.Assets {
		asset.PlatformIds = []string{"arn:aws:ec2:us-east-1:123456789012:instance/i-abc"}
		asset.Platform = &inventory.Platform{
			Name: "amazonlinux", Runtime: "aws-ec2-instance", Kind: "virtualmachine",
			Family: []string{"linux", "unix", "os"}, Version: "2023", Arch: "arm64",
		}
	}
	return report
}

// erroredReportCollection is a scan where the asset could not be reached.
func erroredReportCollection() *policy.ReportCollection {
	report := advisoryReportCollection()
	report.Errors = map[string]string{sampleAssetMrn: "could not connect to the asset"}
	return report
}

// advisoryReportCollection is the sample scan whose vulnerability report carries
// a full advisory, which is the shape the vulnerability API actually returns:
// every affected package is accounted for by an advisory, and every advisory
// names its CVEs.
func advisoryReportCollection() *policy.ReportCollection {
	report := sampleReportCollection()
	report.VulnReports[sampleAssetMrn].Advisories = []*mvd.Advisory{
		{
			ID:          "USN-1234-1",
			Title:       "OpenSSL vulnerabilities",
			Description: "Several security issues were fixed in OpenSSL.",
			Score:       95,
			Published:   "2023-03-01T00:00:00Z",
			Affected:    []*mvd.Package{{Name: "libssl1.1", Version: "1.1.1f-3ubuntu2.19"}},
			Cves: []*mvd.CVE{{
				ID:      "CVE-2023-0286",
				Summary: "X.400 address type confusion",
				Url:     "https://nvd.nist.gov/vuln/detail/CVE-2023-0286",
				Cvss:    []*cvss.Cvss{{Vector: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:N/A:N", Score: 7.4}},
			}},
			Refs: []*mvd.Reference{{Url: "https://ubuntu.com/security/notices/USN-1234-1"}},
		},
	}
	return report
}

func TestOcsfDetectionFindings(t *testing.T) {
	report := detailedReportCollection()
	report.Bundle.Queries[0].Tags = map[string]string{"compliance/cis-aws-foundations-benchmark-1.5.0": "1.4"}
	report.Bundle.Queries[0].Impact = &policy.Impact{Value: &policy.ImpactValue{Value: 80}}

	events, err := convertToOCSF(report,
		ocsfConfig{version: ocsf.DefaultVersion, findings: OcsfFindingsDetection}, fixedScanTime)
	require.NoError(t, err)

	assert.Empty(t, events.ComplianceFindings, "detection mode reports checks as class 2004 only")
	require.Len(t, events.DetectionFindings, 1)

	finding := events.DetectionFindings[0]
	assert.Equal(t, ocsf.ClassUIDDetectionFinding, finding.ClassUID)
	assert.EqualValues(t, ocsf.DetectionFindingTypeUIDCreate, finding.TypeUID)
	assert.Equal(t, ocsf.SeverityCritical, finding.SeverityID)

	// the risk and impact attributes 2004 has, which 2003 does not
	assert.Equal(t, 100, finding.RiskScore, "a score of 0 is a risk of 100")
	assert.Equal(t, ocsf.RiskLevelCritical, finding.RiskLevelID)
	assert.Equal(t, "Critical", finding.RiskLevel)
	assert.Equal(t, 80, finding.ImpactScore)
	assert.Equal(t, ocsf.ImpactHigh, finding.ImpactID)

	// the check itself is the analytic that produced the finding
	require.NotNil(t, finding.FindingInfo.Analytic)
	assert.Equal(t, ocsf.AnalyticTypeRule, finding.FindingInfo.Analytic.TypeID)
	assert.Equal(t, "Ensure the thing is configured", finding.FindingInfo.Analytic.Name)
	assert.Contains(t, finding.FindingInfo.Analytic.Desc, "PermitRootLogin")

	// class 2004 has no compliance object, so the mappings travel in unmapped
	assert.Equal(t, "cis-aws-foundations-benchmark-1.5.0", finding.Unmapped["compliance_standards"])
	assert.Equal(t, "1.4", finding.Unmapped["compliance_controls"])

	require.NotNil(t, finding.Remediation)
	assert.Contains(t, finding.Remediation.Desc, "PermitRootLogin")

	// Every check is reported, passing ones included, with the outcome in
	// status_code. That is what Prowler does, and it keeps a detection-only
	// stream a complete record of the scan.
	all, err := convertToOCSF(sampleReportCollection(),
		ocsfConfig{version: ocsf.DefaultVersion, findings: OcsfFindingsDetection}, fixedScanTime)
	require.NoError(t, err)
	assert.Empty(t, all.ComplianceFindings, "a check is reported in one class, never two")
	require.Len(t, all.DetectionFindings, 3, "every check is a detection, whatever its outcome")

	codes := map[string]bool{}
	for _, finding := range all.DetectionFindings {
		codes[finding.StatusCode] = true
	}
	assert.Equal(t, map[string]bool{"PASS": true, "ERROR": true, "SKIPPED": true}, codes,
		"the outcome is what status_code carries")
}

func TestTruncateAssessment(t *testing.T) {
	assert.Equal(t, "short", truncateAssessment("short"))

	long := strings.Repeat("a", maxAssessmentBytes+100)
	res := truncateAssessment(long)
	assert.Less(t, len(res), len(long))
	assert.Contains(t, res, "assessment truncated")

	// a multi-byte rune straddling the cut must not be split
	runes := strings.Repeat("ü", maxAssessmentBytes)
	res = truncateAssessment(runes)
	assert.True(t, utf8.ValidString(res), "the cut has to land on a rune boundary")
}

// TestOcsfStreamedParquetAcrossAssets covers the streamed writer path: several
// assets are written into one Parquet file per class through repeated writes,
// and the file has to hold every row and still be readable after Close.
func TestOcsfStreamedParquetAcrossAssets(t *testing.T) {
	dir := t.TempDir()
	conf := defaultPrintConfig()
	conf.format = FormatOcsfParquet

	const assets, checks = 5, 4
	handler := &ocsfFileHandler{target: dir, conf: conf}
	require.NoError(t, handler.WriteReport(t.Context(), largeReportCollection(assets, checks)))

	raw, err := os.ReadFile(filepath.Join(dir, ocsf.ClassComplianceFinding+".parquet"))
	require.NoError(t, err)

	rows, err := parquet.Read[ocsf.ComplianceFinding](bytes.NewReader(raw), int64(len(raw)))
	require.NoError(t, err)
	assert.Len(t, rows, assets*checks, "every asset's rows land in the one file per class")

	seen := map[string]bool{}
	for _, row := range rows {
		require.Len(t, row.Resources, 1)
		seen[row.Resources[0].UID] = true
	}
	assert.Len(t, seen, assets, "each asset is represented")

	inventory, err := os.Stat(filepath.Join(dir, ocsf.ClassInventoryInfo+".parquet"))
	require.NoError(t, err)
	assert.NotZero(t, inventory.Size())

	// classes the scan did not produce must not leave empty files behind
	_, err = os.Stat(filepath.Join(dir, ocsf.ClassDetectionFinding+".parquet"))
	assert.True(t, os.IsNotExist(err), "no file for a class with no events")
}
