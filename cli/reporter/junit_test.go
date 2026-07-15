// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/v13/policy"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/upstream/mvd"
	"go.mondoo.com/mql/v13/utils/iox"
)

func sampleReportCollection() *policy.ReportCollection {
	return &policy.ReportCollection{
		Assets: map[string]*inventory.Asset{
			"//assets.api.mondoo.app/spaces/dazzling-golick-767384/assets/2DRZ1cCWFyTYCArycAXHwvn1oU2": {
				Name:        "X1",
				PlatformIds: []string{"//platformid.api.mondoo.app/hostname/X1"},
				State:       inventory.State_STATE_ONLINE,
				Platform: &inventory.Platform{
					Name:    "ubuntu",
					Arch:    "amd64",
					Kind:    "baremetal",
					Version: "22.04",
					Family:  []string{"debian", "linux", "unix", "os"},
				},
			},
		},
		ResolvedPolicies: map[string]*policy.ResolvedPolicy{
			"//assets.api.mondoo.app/spaces/dazzling-golick-767384/assets/2DRZ1cCWFyTYCArycAXHwvn1oU2": {
				CollectorJob: &policy.CollectorJob{
					ReportingQueries: map[string]*policy.StringArray{
						"+u6doYoYG5E=": nil,
						"057itYF8s30=": nil,
						"GyJVAziB/tU=": nil,
					},
				},
			},
		},
		Bundle: &policy.Bundle{
			Policies: nil, // not needed for this test since junit does not sort by policy
			Queries: []*policy.Mquery{
				{
					Mrn:    "//policy.api.mondoo.app/queries/mondoo-linux-security-snmp-server-is-not-enabled",
					CodeId: "+u6doYoYG5E=",
					Title:  "Ensure SNMP server is stopped and not enabled",
				},
				{
					Mrn:    "//policy.api.mondoo.app/queries/mondoo-kubernetes-security-kubelet-event-record-qps",
					CodeId: "057itYF8s30=",
					Title:  "Configure kubelet to capture all event creation",
				},
				{
					Mrn:    "//policy.api.mondoo.app/queries/mondoo-kubernetes-security-secure-scheduler_conf",
					CodeId: "GyJVAziB/tU=",
					Title:  "Set secure file permissions on the scheduler.conf file",
				},
			},
		},
		Reports: map[string]*policy.Report{
			"//assets.api.mondoo.app/spaces/dazzling-golick-767384/assets/2DRZ1cCWFyTYCArycAXHwvn1oU2": {
				ScoringMrn: "//assets.api.mondoo.app/spaces/dazzling-golick-767384/assets/2DRZ1cCWFyTYCArycAXHwvn1oU2",
				EntityMrn:  "//assets.api.mondoo.app/spaces/dazzling-golick-767384/assets/2DRZ1cCWFyTYCArycAXHwvn1oU2",
				Score: &policy.Score{
					Value:           29,
					ScoreCompletion: 100,
					DataCompletion:  100,
				},
				// add passed, failed and skipped test
				Scores: map[string]*policy.Score{
					"+u6doYoYG5E=": {
						Type:  2, // result
						Value: 100,
					},
					"057itYF8s30=": {
						Type:  4, // error
						Value: 0,
					},
					"GyJVAziB/tU=": {
						Type:  8, // skip
						Value: 0,
					},
				},
			},
		},
		VulnReports: map[string]*mvd.VulnReport{
			"//assets.api.mondoo.app/spaces/dazzling-golick-767384/assets/2DRZ1cCWFyTYCArycAXHwvn1oU2": {
				Packages: []*mvd.Package{
					{
						Name:      "libssl1.1",
						Version:   "1.1.1f-3ubuntu2.19",
						Affected:  true,
						Score:     100,
						Available: "1.1.1f-3ubuntu2.20",
					},
				},
				Stats: &mvd.ReportStats{
					Packages: &mvd.ReportStatsPackages{
						Total:    1,
						Critical: 1,
						Affected: 1,
					},
				},
			},
		},
	}
}

func TestJunitConverter(t *testing.T) {
	yr := sampleReportCollection()
	buf := bytes.Buffer{}
	writer := iox.IOWriter{Writer: &buf}
	err := ConvertToJunit(yr, &writer, false)
	require.NoError(t, err)

	junitReport := buf.String()
	assert.Contains(t, junitReport, "name=\"Policy Report for X1\"")
	assert.Contains(t, junitReport, "<testcase name=\"Ensure SNMP server is stopped and not enabled\" classname=\"score\"></testcase>")
	assert.Contains(t, junitReport, "<testcase name=\"Configure kubelet to capture all event creation\" classname=\"score\">\n\t\t\t<failure message=\"\" type=\"error\"></failure>\n\t\t</testcase>")
	assert.Contains(t, junitReport, "<testcase name=\"Set secure file permissions on the scheduler.conf file\" classname=\"score\">\n\t\t\t<skipped message=\"skipped\"></skipped>\n\t\t</testcase>")
	assert.Contains(t, junitReport, "<testsuite name=\"Vulnerability Report for")
	assert.Contains(t, junitReport, "<property name=\"report.packages.total\" value=\"1\"></property>")
	assert.Contains(t, junitReport, "<property name=\"report.packages.critical\" value=\"1\"></property>")
	assert.Contains(t, junitReport, "<testcase name=\"libssl1.1\" classname=\"vulnerability\">")
	assert.Contains(t, junitReport, "<failure message=\"Update libssl1.1 to 1.1.1f-3ubuntu2.20\"><![CDATA[libssl1.1 with version 1.1.1f-3ubuntu2.19 has known vulnerabilities (score 10)]]></failure>")
}

func TestJunitNilReport(t *testing.T) {
	var yr *policy.ReportCollection

	buf := bytes.Buffer{}
	writer := iox.IOWriter{Writer: &buf}
	err := ConvertToJunit(yr, &writer, false)
	require.NoError(t, err)

	assert.Equal(t, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<testsuites></testsuites>\n", buf.String())
}

// detailedReportCollection builds a minimal collection with a single failing
// check that carries a description, MQL, remediation, and references. It has no
// ExecutionJob, so the assessment section is intentionally absent here (that path
// is covered by TestJunitConverterDetailedAssessment).
func detailedReportCollection() *policy.ReportCollection {
	assetMrn := "//assets.api.mondoo.app/spaces/test/assets/abc"
	codeID := "abc123=="
	return &policy.ReportCollection{
		Assets: map[string]*inventory.Asset{
			assetMrn: {
				Name:     "X1",
				Platform: &inventory.Platform{Name: "terraform-hcl", Family: []string{"terraform"}},
			},
		},
		ResolvedPolicies: map[string]*policy.ResolvedPolicy{
			assetMrn: {
				CollectorJob: &policy.CollectorJob{
					ReportingQueries: map[string]*policy.StringArray{
						codeID: nil,
					},
				},
			},
		},
		Bundle: &policy.Bundle{
			Queries: []*policy.Mquery{
				{
					Mrn:    "//policy.api.mondoo.app/queries/test-check",
					CodeId: codeID,
					Title:  "Ensure the thing is configured",
					Mql:    "sshd.config.params['PermitRootLogin'] == \"no\"",
					Docs: &policy.MqueryDocs{
						Desc: "Root login over SSH should be disabled.",
						Remediation: &policy.Remediation{
							Items: []*policy.TypedDoc{
								{Id: "console", Desc: "Use the AWS console to fix it."},
								{Id: "terraform", Desc: "Set PermitRootLogin to no in your TF config."},
								{Id: "cloudformation", Desc: "Use CloudFormation to fix it."},
							},
						},
						Refs: []*policy.MqueryRef{
							{Title: "CIS Benchmark", Url: "https://example.com/cis"},
						},
					},
				},
			},
		},
		Reports: map[string]*policy.Report{
			assetMrn: {
				ScoringMrn: assetMrn,
				EntityMrn:  assetMrn,
				Scores: map[string]*policy.Score{
					codeID: {Type: policy.ScoreType_Result, Value: 0},
				},
			},
		},
	}
}

func TestJunitConverterDetailed(t *testing.T) {
	yr := detailedReportCollection()

	buf := bytes.Buffer{}
	writer := iox.IOWriter{Writer: &buf}
	require.NoError(t, ConvertToJunit(yr, &writer, true))
	out := buf.String()

	// the failed check testcase carries a rich body
	assert.Contains(t, out, "name=\"Ensure the thing is configured\"")
	assert.Contains(t, out, "Root login over SSH should be disabled.")
	assert.Contains(t, out, "Query:")
	assert.Contains(t, out, "PermitRootLogin")
	assert.Contains(t, out, "Remediation:")
	// remediation is filtered to the terraform platform (family match); the
	// console/cloudformation variants are dropped as noise
	assert.Contains(t, out, "[terraform] Set PermitRootLogin to no in your TF config.")
	assert.NotContains(t, out, "Use the AWS console to fix it.")
	assert.NotContains(t, out, "Use CloudFormation to fix it.")
	assert.Contains(t, out, "References:")
	assert.Contains(t, out, "CIS Benchmark: https://example.com/cis")

	// the default (lean) output must be unchanged: no body, generic message
	leanBuf := bytes.Buffer{}
	leanWriter := iox.IOWriter{Writer: &leanBuf}
	require.NoError(t, ConvertToJunit(yr, &leanWriter, false))
	lean := leanBuf.String()
	assert.NotContains(t, lean, "Root login over SSH should be disabled.")
	assert.NotContains(t, lean, "Remediation:")
	assert.Contains(t, lean, "message=\"results do not match\"")
}

// TestJunitConverterDetailedAssessment exercises the GetCodeBundle ->
// Query2Assessment -> Assessment path against a real report fixture that carries
// a compiled execution job and failing assertion checks.
func TestJunitConverterDetailedAssessment(t *testing.T) {
	raw, err := os.ReadFile("./testdata/report-ubuntu.json")
	require.NoError(t, err)
	yr := &policy.ReportCollection{}
	require.NoError(t, json.Unmarshal(raw, yr))

	buf := bytes.Buffer{}
	writer := iox.IOWriter{Writer: &buf}
	require.NoError(t, ConvertToJunit(yr, &writer, true))
	out := buf.String()

	// failing checks render the query and the expected-vs-actual assessment
	assert.Contains(t, out, "Query:")
	assert.Contains(t, out, "Result:")
	// no ANSI color escapes should leak into the XML
	assert.NotContains(t, out, "\x1b[")
}

func TestQueryRemediationPlatformFilter(t *testing.T) {
	mkQuery := func(items ...*policy.TypedDoc) *policy.Mquery {
		return &policy.Mquery{Docs: &policy.MqueryDocs{Remediation: &policy.Remediation{Items: items}}}
	}
	console := &policy.TypedDoc{Id: "console", Desc: "console fix"}
	tf := &policy.TypedDoc{Id: "terraform", Desc: "terraform fix"}
	def := &policy.TypedDoc{Id: "default", Desc: "generic fix"}

	tfKeys := platformRemediationKeys(&inventory.Platform{Name: "terraform-hcl", Family: []string{"terraform"}})

	// family match ("terraform" via terraform-hcl) keeps only the terraform item
	out := queryRemediation(mkQuery(console, tf), tfKeys)
	assert.Contains(t, out, "[terraform] terraform fix")
	assert.NotContains(t, out, "console fix")

	// no platform-specific match -> fall back to all items (never drop remediation)
	out = queryRemediation(mkQuery(console), tfKeys)
	assert.Contains(t, out, "[console] console fix")

	// platform-agnostic "default" is kept and shown without a label
	out = queryRemediation(mkQuery(def, console), tfKeys)
	assert.Contains(t, out, "generic fix")
	assert.NotContains(t, out, "[default]")
	assert.NotContains(t, out, "console fix")

	// nil docs / nil remediation are safe
	assert.Equal(t, "", queryRemediation(&policy.Mquery{}, tfKeys))
}
