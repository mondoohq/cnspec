// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"bytes"
	"encoding/csv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/internal/reportfixture"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/utils/iox"
)

func renderCSV(t *testing.T, r *policy.ReportCollection) [][]string {
	t.Helper()
	buf := bytes.Buffer{}
	writer := iox.IOWriter{Writer: &buf}
	require.NoError(t, ConvertToCSV(r, &writer))

	records, err := csv.NewReader(strings.NewReader(buf.String())).ReadAll()
	require.NoError(t, err, "output is not valid CSV:\n%s", buf.String())
	return records
}

// rowsByCheck indexes data rows by the Check column.
func rowsByCheck(t *testing.T, records [][]string) map[string][]string {
	t.Helper()
	require.NotEmpty(t, records)
	out := map[string][]string{}
	for _, rec := range records[1:] {
		out[rec[4]] = rec
	}
	return out
}

func TestConvertToCSV(t *testing.T) {
	records := renderCSV(t, reportfixture.Sample())

	assert.Equal(t, []string{
		"Asset", "Asset MRN", "Platform", "Platform Version",
		"Check", "Title", "Status", "Score", "Impact", "Message",
	}, records[0])

	// Three checks on one asset. The asset's own aggregate score and the
	// per-policy scores live in the same map and must not become rows.
	require.Len(t, records, 4, "want a header and one row per check")

	byCheck := rowsByCheck(t, records)

	pass := byCheck["//policy.api.mondoo.app/queries/mondoo-linux-security-snmp-server-is-not-enabled"]
	require.NotNil(t, pass, "passing check is missing")
	assert.Equal(t, "X1", pass[0])
	assert.Equal(t, "ubuntu", pass[2])
	assert.Equal(t, "22.04", pass[3])
	assert.Equal(t, "Ensure SNMP server is stopped and not enabled", pass[5])
	assert.Equal(t, "pass", pass[6])
	assert.Equal(t, "100", pass[7])

	errored := byCheck["//policy.api.mondoo.app/queries/mondoo-kubernetes-security-kubelet-event-record-qps"]
	require.NotNil(t, errored)
	assert.Equal(t, "error", errored[6])
	// An errored check has no score. Writing 0 would sort it next to a real
	// failure and make a broken check look like a finding.
	assert.Empty(t, errored[7], "errored check must not carry a score")

	skipped := byCheck["//policy.api.mondoo.app/queries/mondoo-kubernetes-security-secure-scheduler_conf"]
	require.NotNil(t, skipped)
	assert.Equal(t, "skip", skipped[6])
	assert.Empty(t, skipped[7], "skipped check must not carry a score")
}

// TestCSVStatusMatchesJSON pins the two exports to one vocabulary.
//
// Both derive status through gatherScoreValue. If CSV grew its own names, two
// exports of the same scan would disagree about what happened and whichever one
// the reader had open would look authoritative.
//
// Run against the recorded scan, not the hand-built fixture: the two formats
// select their checks from different places -- the JSON report walks
// ExecutionJob.Queries, CSV walks CollectorJob.ReportingQueries -- and only a
// real scan has both populated. The assertion is therefore on the statuses they
// agree to report, not on the two sets being identical.
func TestCSVStatusMatchesJSON(t *testing.T) {
	report, err := reportfixture.UbuntuScan()
	require.NoError(t, err)

	proto, err := ConvertToProto(report)
	require.NoError(t, err)

	byCheck := rowsByCheck(t, renderCSV(t, report))
	require.NotEmpty(t, byCheck, "the recorded scan produced no CSV rows")

	var compared int
	for assetMrn, scores := range proto.GetScores() {
		for id, sv := range scores.GetValues() {
			if id == assetMrn {
				continue // the asset's aggregate has no CSV row
			}
			row, ok := byCheck[id]
			if !ok {
				continue
			}
			assert.Equal(t, sv.GetStatus(), row[6], "status disagrees for %s", id)
			compared++
		}
	}
	require.NotZero(t, compared, "no check appeared in both exports; the comparison proved nothing")
	t.Logf("compared %d checks present in both exports", compared)
}

// TestCSVIncludesAssetErrors covers the asset that never scanned.
//
// It gets a row of its own. Left out, a scan where half the fleet failed to
// connect would be indistinguishable in a spreadsheet from one where it all
// passed -- the exact failure the exit code also cannot show.
func TestCSVIncludesAssetErrors(t *testing.T) {
	const brokenMrn = "//assets.api.mondoo.app/spaces/x/assets/broken"
	report := reportfixture.Sample()
	report.Assets[brokenMrn] = &inventory.Asset{
		Name:     "unreachable-host",
		Platform: &inventory.Platform{Name: "unknown"},
	}
	report.Errors = map[string]string{brokenMrn: "rpc error: could not connect"}

	records := renderCSV(t, report)
	require.Len(t, records, 5, "want the header, the error row and three checks")

	errRow := records[1] // error rows are written first
	assert.Equal(t, "unreachable-host", errRow[0])
	assert.Equal(t, brokenMrn, errRow[1])
	assert.Equal(t, "error", errRow[6])
	assert.Equal(t, "rpc error: could not connect", errRow[9])
}

func TestCSVNilReport(t *testing.T) {
	records := renderCSV(t, nil)
	// A header and nothing else: the export ran and found no assets, which a
	// consumer can tell apart from a truncated file.
	require.Len(t, records, 1)
	assert.Equal(t, "Asset", records[0][0])
}

func TestCSVIsDeterministic(t *testing.T) {
	first := renderCSV(t, reportfixture.Sample())
	for range 8 {
		// Go randomises map iteration and these files get diffed and committed,
		// so every map walked by the converter is sorted.
		assert.Equal(t, first, renderCSV(t, reportfixture.Sample()))
	}
}

// TestCSVEscapesFormulas covers CSV injection through fields that come from the
// scanned target and from policy content, not just from the vuln path that
// escapeCSVCell was originally written for. See OWASP "CSV Injection".
func TestCSVEscapesFormulas(t *testing.T) {
	report := reportfixture.Sample()
	for _, a := range report.Assets {
		a.Name = "=HYPERLINK(\"http://evil\",\"click\")"
	}
	for _, q := range report.Bundle.Queries {
		q.Title = "+SUM(A1:A2)"
	}

	records := renderCSV(t, report)
	require.Greater(t, len(records), 1)
	for _, rec := range records[1:] {
		assert.True(t, strings.HasPrefix(rec[0], "'="), "asset name not neutralized: %q", rec[0])
		assert.True(t, strings.HasPrefix(rec[5], "'+"), "title not neutralized: %q", rec[5])
	}
}

// TestConvertersSurviveMissingJobs covers a ResolvedPolicy that carries only a
// CollectorJob.
//
// Both converters used to reach through it with direct field access --
// resolved.ExecutionJob.Queries in ConvertToProto, and the CollectorJob lookup
// here -- which is a nil pointer dereference, not an error. On the `-o json`
// path that is a panic partway through writing a report, so the user gets a
// stack trace and a truncated document. Found by TestCSVStatusMatchesJSON
// against the shared fixture, which has no ExecutionJob.
func TestConvertersSurviveMissingJobs(t *testing.T) {
	report := reportfixture.Sample()
	for _, resolved := range report.ResolvedPolicies {
		require.Nil(t, resolved.ExecutionJob, "fixture is expected to have no ExecutionJob")
	}

	require.NotPanics(t, func() {
		_, err := ConvertToProto(report)
		require.NoError(t, err)
	}, "ConvertToProto must not panic on a report without an ExecutionJob")

	require.NotPanics(t, func() {
		renderCSV(t, report)
	}, "ConvertToCSV must not panic on a report without an ExecutionJob")

	// And with no CollectorJob either: no checks resolve, so the file is a
	// header and any asset errors, rather than a crash.
	for _, resolved := range report.ResolvedPolicies {
		resolved.CollectorJob = nil
	}
	records := renderCSV(t, report)
	assert.Len(t, records, 1)
}

func TestCSVImpactValue(t *testing.T) {
	assert.Empty(t, impactValue(&policy.Mquery{}), "an undeclared impact is not impact 0")
	assert.Empty(t, impactValue(&policy.Mquery{Impact: &policy.Impact{}}))
	assert.Equal(t, "80", impactValue(&policy.Mquery{
		Impact: &policy.Impact{Value: &policy.ImpactValue{Value: 80}},
	}))
}

func TestCSVCheckIdentifier(t *testing.T) {
	// The UID is what the check is called in the policy YAML, so it is what a
	// reader can search for; an upstream-resolved bundle has only MRNs.
	assert.Equal(t, "my-uid", checkIdentifier(&policy.Mquery{Uid: "my-uid", Mrn: "//m"}, "score-id"))
	assert.Equal(t, "//m", checkIdentifier(&policy.Mquery{Mrn: "//m"}, "score-id"))
	assert.Equal(t, "score-id", checkIdentifier(&policy.Mquery{}, "score-id"))
}
