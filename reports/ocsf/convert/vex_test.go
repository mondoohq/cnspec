// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package convert

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/internal/reportfixture"
	"go.mondoo.com/cnspec/reports/ocsf"
)

// vexEvents renders the fixture at a given OCSF version and parses the
// newline-delimited result back, keyed by finding title.
func vexEvents(t *testing.T, version ocsf.Version) map[string]map[string]any {
	t.Helper()
	buf := bytes.Buffer{}
	require.NoError(t, ConvertVexReport(
		reportfixture.VexTarget, reportfixture.VexRows(), version, EncodingJSON, &buf))

	out := map[string]map[string]any{}
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		var event map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &event))
		title := event["finding_info"].(map[string]any)["title"].(string)
		out[title] = event
	}
	return out
}

// Rows arrive one per (vulnerability, package); a finding is one advisory. The
// two openssl rows are therefore one finding carrying two packages, not two
// findings -- which is also how the mvd path reports the same advisory.
func TestOcsfVexGroupsRowsByAdvisory(t *testing.T) {
	events := vexEvents(t, ocsf.DefaultVersion)
	assert.Len(t, events, 5, "six rows, but USN-1234-1 accounts for two of them")

	usn := events["USN-1234-1"]
	require.NotNil(t, usn)
	vulns := usn["vulnerabilities"].([]any)
	require.Len(t, vulns, 1)

	pkgs := vulns[0].(map[string]any)["affected_packages"].([]any)
	require.Len(t, pkgs, 2)
	names := []string{
		pkgs[0].(map[string]any)["name"].(string),
		pkgs[1].(map[string]any)["name"].(string),
	}
	assert.ElementsMatch(t, []string{"libssl3", "openssl"}, names)

	assert.EqualValues(t, ocsf.ClassUIDVulnerabilityFinding, usn["class_uid"])
	assert.Equal(t, "Critical", usn["severity"])
	assert.Equal(t, "New", usn["status"])
	assert.Equal(t, reportfixture.VexTarget,
		usn["resources"].([]any)[0].(map[string]any)["name"])
}

// cve.uid is specified as a CVE identifier. A distro advisory id put there joins
// against nothing in NVD, so below 1.9 it has to be marked; at 1.9 the advisory
// object is where it belongs. Same rule as the mvd path.
func TestOcsfVexAdvisoryIdIsNotPassedOffAsACve(t *testing.T) {
	v13 := vexEvents(t, ocsf.Version130)

	cve := v13["CVE-2023-0286"]["vulnerabilities"].([]any)[0].(map[string]any)
	assert.Equal(t, "CVE-2023-0286", cve["cve"].(map[string]any)["uid"])
	assert.NotContains(t, v13["CVE-2023-0286"]["unmapped"], "non_cve_uid",
		"a real CVE needs no marker")

	usn := v13["USN-1234-1"]["vulnerabilities"].([]any)[0].(map[string]any)
	assert.Equal(t, "USN-1234-1", usn["cve"].(map[string]any)["uid"],
		"1.3 has nowhere else to put it")
	assert.Equal(t, "USN-1234-1", v13["USN-1234-1"]["unmapped"].(map[string]any)["non_cve_uid"],
		"so it must be marked as not a CVE")

	v19 := vexEvents(t, ocsf.Version190)
	usn19 := v19["USN-1234-1"]["vulnerabilities"].([]any)[0].(map[string]any)
	assert.NotContains(t, usn19, "cve", "1.9 has the advisory object")
	assert.Equal(t, "USN-1234-1", usn19["advisory"].(map[string]any)["uid"])
	assert.NotContains(t, v19["USN-1234-1"]["unmapped"], "non_cve_uid")
}

// The VEX path knows the purl, which the mvd path never did. Losing it here
// would throw away the one identifier that resolves a package exactly.
func TestOcsfVexCarriesPurlAndFixState(t *testing.T) {
	events := vexEvents(t, ocsf.DefaultVersion)

	fixed := events["CVE-2023-0286"]["vulnerabilities"].([]any)[0].(map[string]any)
	assert.Equal(t, true, fixed["is_fix_available"])
	pkg := fixed["affected_packages"].([]any)[0].(map[string]any)
	assert.Equal(t, "3.0.2-0ubuntu1.12", pkg["fixed_in_version"])
	assert.Equal(t, "deb", pkg["package_manager"])
	assert.Contains(t, pkg["purl"], "pkg:deb/ubuntu/libssl3@")

	// is_fix_available is an optional bool, so false is omitted rather than
	// emitted -- absent is how the OCSF type says "no fix", and the mvd path
	// reports the same case the same way.
	unfixed := events["CVE-2024-9999"]["vulnerabilities"].([]any)[0].(map[string]any)
	assert.NotContains(t, unfixed, "is_fix_available")
	assert.NotContains(t, unfixed["affected_packages"].([]any)[0].(map[string]any), "fixed_in_version")
}

// cvss_score is a 0.0-10.0 value. Emitting cnspec's 0-100 risk there made every
// finding match a `cvss_score > 7` filter; see vulnerability.go.
func TestOcsfVexCvssScoreIsACvssScore(t *testing.T) {
	events := vexEvents(t, ocsf.DefaultVersion)
	for title, want := range map[string]string{
		"USN-1234-1":    "9.0",
		"CVE-2023-0286": "7.0",
		"CVE-2024-9999": "4.0",
		"CVE-2024-7777": "1.0",
	} {
		got := events[title]["unmapped"].(map[string]any)["cvss_score"]
		assert.Equal(t, want, got, "cvss_score of %s", title)
	}
}

// A severity label cnspec does not know must not be promoted; it reads as the
// lowest band, the same as none.
func TestOcsfVexUnknownSeverityIsNotSevere(t *testing.T) {
	events := vexEvents(t, ocsf.DefaultVersion)
	finding := events["GHSA-xxxx-yyyy-zzzz"]
	require.NotNil(t, finding)
	assert.Equal(t, ocsf.SeverityName(ocsf.SeverityInformational), finding["severity"])
	assert.Equal(t, "0.0", finding["unmapped"].(map[string]any)["cvss_score"])
}

// Parquet is the same events in a different encoding, and it is a real file with
// a schema rather than JSON with a different extension.
func TestOcsfVexParquetEncoding(t *testing.T) {
	buf := bytes.Buffer{}
	require.NoError(t, ConvertVexReport(
		reportfixture.VexTarget, reportfixture.VexRows(), ocsf.DefaultVersion, EncodingParquet, &buf))
	require.NotEmpty(t, buf.Bytes())
	assert.Equal(t, "PAR1", string(buf.Bytes()[:4]), "a parquet file starts with its magic")
}

// An empty report writes nothing rather than an empty JSON object, which is what
// newline-delimited JSON means by "no events".
func TestOcsfVexNoRows(t *testing.T) {
	buf := bytes.Buffer{}
	require.NoError(t, ConvertVexReport("target", nil, ocsf.DefaultVersion, EncodingJSON, &buf))
	assert.Empty(t, strings.TrimSpace(buf.String()))
}
