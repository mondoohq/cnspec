// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/internal/reportfixture"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/cnspec/reports/reportdoc"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/utils/iox"
)

func TestJunitConverter(t *testing.T) {
	yr := reportfixture.Sample()
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

func TestJunitConverterDetailed(t *testing.T) {
	yr := reportfixture.Detailed()

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
	raw := reportfixture.UbuntuScanJSON()
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

	tfKeys := reportdoc.PlatformRemediationKeys(&inventory.Platform{Name: "terraform-hcl", Family: []string{"terraform"}})

	// family match ("terraform" via terraform-hcl) keeps only the terraform item
	out := reportdoc.QueryRemediation(mkQuery(console, tf), tfKeys)
	assert.Contains(t, out, "[terraform] terraform fix")
	assert.NotContains(t, out, "console fix")

	// no platform-specific match -> fall back to all items (never drop remediation)
	out = reportdoc.QueryRemediation(mkQuery(console), tfKeys)
	assert.Contains(t, out, "[console] console fix")

	// platform-agnostic "default" is kept and shown without a label
	out = reportdoc.QueryRemediation(mkQuery(def, console), tfKeys)
	assert.Contains(t, out, "generic fix")
	assert.NotContains(t, out, "[default]")
	assert.NotContains(t, out, "console fix")

	// nil docs / nil remediation are safe
	assert.Equal(t, "", reportdoc.QueryRemediation(&policy.Mquery{}, tfKeys))
}
