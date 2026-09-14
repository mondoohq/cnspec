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
	"go.mondoo.com/mql/utils/iox"
)

// sarifVulnDoc renders the fixture and parses it back, which is also the check
// that what we emit is JSON a consumer can read.
func sarifVulnDoc(t *testing.T) map[string]any {
	t.Helper()
	buf := bytes.Buffer{}
	writer := iox.IOWriter{Writer: &buf}
	require.NoError(t, VulnReportToSarif(reportfixture.VexTarget, reportfixture.VexRows(), &writer))

	var doc map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &doc))
	return doc
}

func sarifVulnRun(t *testing.T) map[string]any {
	t.Helper()
	runs := sarifVulnDoc(t)["runs"].([]any)
	require.Len(t, runs, 1, "a vuln report covers one target, so it is one run")
	return runs[0].(map[string]any)
}

func TestSarifVulnReportShape(t *testing.T) {
	doc := sarifVulnDoc(t)
	assert.Equal(t, "2.1.0", doc["version"])

	run := doc["runs"].([]any)[0].(map[string]any)
	driver := run["tool"].(map[string]any)["driver"].(map[string]any)
	assert.Equal(t, "cnspec", driver["name"])
	assert.Equal(t, "Mondoo", driver["organization"])
	assert.Equal(t, reportfixture.VexTarget, run["properties"].(map[string]any)["asset"])

	// One result per row; one rule per distinct advisory (USN-1234-1 covers two
	// of the six rows).
	assert.Len(t, run["results"].([]any), len(reportfixture.VexRows()))
	assert.Len(t, driver["rules"].([]any), 5)
}

// Severity has to reach SARIF as both a level and the numeric GitHub code
// scanning reads, and the two must agree with each other.
func TestSarifVulnSeverityMapping(t *testing.T) {
	run := sarifVulnRun(t)

	levels := map[string]string{}
	severities := map[string]string{}
	for _, raw := range run["results"].([]any) {
		result := raw.(map[string]any)
		props := result["properties"].(map[string]any)
		id := result["ruleId"].(string)
		levels[id] = result["level"].(string)
		severities[id] = props["security-severity"].(string)
	}

	assert.Equal(t, "error", levels["USN-1234-1"], "CRITICAL is an error")
	assert.Equal(t, "9.0", severities["USN-1234-1"])
	assert.Equal(t, "error", levels["CVE-2023-0286"], "HIGH is an error")
	assert.Equal(t, "7.0", severities["CVE-2023-0286"])
	assert.Equal(t, "warning", levels["CVE-2024-9999"], "MEDIUM is a warning")
	assert.Equal(t, "4.0", severities["CVE-2024-9999"])
	assert.Equal(t, "note", levels["CVE-2024-7777"], "LOW is a note")
	assert.Equal(t, "1.0", severities["CVE-2024-7777"])

	// An unrecognised label must not read as severe.
	assert.Equal(t, "note", levels["GHSA-xxxx-yyyy-zzzz"])
	assert.Equal(t, "0.0", severities["GHSA-xxxx-yyyy-zzzz"])
}

// The package coordinates are the reason this format is worth emitting; a result
// that names no package locates nothing.
func TestSarifVulnResultCarriesPackageCoordinates(t *testing.T) {
	run := sarifVulnRun(t)

	var found map[string]any
	for _, raw := range run["results"].([]any) {
		result := raw.(map[string]any)
		props := result["properties"].(map[string]any)
		if result["ruleId"] == "CVE-2023-0286" && props["package"] == "libssl3" {
			found = result
		}
	}
	require.NotNil(t, found, "the libssl3 row is missing")

	props := found["properties"].(map[string]any)
	assert.Equal(t, "3.0.2-0ubuntu1.10", props["installedVersion"])
	assert.Equal(t, "3.0.2-0ubuntu1.12", props["fixedVersion"])
	assert.Equal(t, "deb", props["ecosystem"])
	assert.Contains(t, props["purl"], "pkg:deb/ubuntu/libssl3@")
	assert.Equal(t, "fail", props["status"])

	locs := found["locations"].([]any)[0].(map[string]any)["logicalLocations"].([]any)
	require.Len(t, locs, 2, "the asset and the package both locate the finding")
	assert.Equal(t, reportfixture.VexTarget, locs[0].(map[string]any)["name"])
	assert.Equal(t, "libssl3", locs[1].(map[string]any)["name"])
	assert.Equal(t, "package", locs[1].(map[string]any)["kind"])
}

// A row with no fixed version must say so rather than claim a fix of "".
func TestSarifVulnUnfixedRow(t *testing.T) {
	run := sarifVulnRun(t)
	for _, raw := range run["results"].([]any) {
		result := raw.(map[string]any)
		if result["ruleId"] != "CVE-2024-9999" {
			continue
		}
		props := result["properties"].(map[string]any)
		assert.NotContains(t, props, "fixedVersion", "there is no fix to report")
		assert.Contains(t, result["message"].(map[string]any)["text"], "No fixed version is available yet.")
		return
	}
	t.Fatal("the unfixed row is missing")
}

// VulnRows emits a row for a vulnerability whose component never resolved rather
// than dropping it. It still has to render, and must not render as a package
// literally named "".
func TestSarifVulnUnresolvedComponentStillRenders(t *testing.T) {
	run := sarifVulnRun(t)
	for _, raw := range run["results"].([]any) {
		result := raw.(map[string]any)
		if result["ruleId"] != "CVE-2024-7777" {
			continue
		}
		props := result["properties"].(map[string]any)
		assert.Equal(t, "unknown package", props["package"])
		assert.NotContains(t, props, "purl")
		return
	}
	t.Fatal("the unresolved-component row is missing")
}

// Fingerprints are what lets a consumer track one finding across runs, so the
// same input must produce the same bytes.
func TestSarifVulnDeterministic(t *testing.T) {
	first := bytes.Buffer{}
	second := bytes.Buffer{}
	w1 := iox.IOWriter{Writer: &first}
	w2 := iox.IOWriter{Writer: &second}
	require.NoError(t, VulnReportToSarif(reportfixture.VexTarget, reportfixture.VexRows(), &w1))
	require.NoError(t, VulnReportToSarif(reportfixture.VexTarget, reportfixture.VexRows(), &w2))
	assert.Equal(t, first.String(), second.String())

	run := sarifVulnRun(t)
	seen := map[string]bool{}
	for _, raw := range run["results"].([]any) {
		result := raw.(map[string]any)
		fp := result["partialFingerprints"].(map[string]any)[sarifFingerprintKey].(string)
		assert.NotEmpty(t, fp)
		assert.False(t, seen[fp], "two findings share a fingerprint, so one hides the other")
		seen[fp] = true
	}
}

// An empty report is a valid report: it must still be a SARIF document rather
// than no output at all.
func TestSarifVulnNoRows(t *testing.T) {
	buf := bytes.Buffer{}
	writer := iox.IOWriter{Writer: &buf}
	require.NoError(t, VulnReportToSarif("target", nil, &writer))

	var doc map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &doc))
	run := doc["runs"].([]any)[0].(map[string]any)
	assert.Empty(t, run["results"])
}
