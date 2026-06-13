// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/v13/policy"
	"go.mondoo.com/mql/v13/cli/printer"
	"go.mondoo.com/mql/v13/cli/theme/colors"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/upstream/mvd"
	"go.mondoo.com/mql/v13/utils/iox"
)

func TestCompactReporter(t *testing.T) {
	reportCollectionRaw, err := os.ReadFile("./testdata/report-ubuntu.json")
	require.NoError(t, err)

	yr := &policy.ReportCollection{}
	err = json.Unmarshal(reportCollectionRaw, yr)
	require.NoError(t, err)

	buf := bytes.Buffer{}
	writer := iox.IOWriter{Writer: &buf}

	r := &Reporter{
		Conf:    defaultPrintConfig(),
		Printer: &printer.DefaultPrinter,
		Colors:  &colors.DefaultColorTheme,
	}
	rr := &defaultReporter{
		Reporter: r,
		output:   &writer,
		data:     yr,
	}
	rr.print()

	strData := buf.String()
	assert.Contains(t, strData, "! Error:          Set")
	assert.Contains(t, strData, "✓ Ensure ")
	assert.Contains(t, strData, "✕ CRITICAL (100): Ensure")
}

// TestCompactReporterNoChecksSpacing guards against the regression in
// https://github.com/mondoohq/mql/issues/6390: an asset with no checks left a
// double blank line between the asset header and the first printed section.
// Each non-empty body section must be separated from the header (and from the
// previous section) by exactly one blank line.
func TestCompactReporterNoChecksSpacing(t *testing.T) {
	assetMrn := "//assets.api.mondoo.app/spaces/test/assets/no-checks"
	riskMrn := "//policy.api.mondoo.app/risks/example"

	data := &policy.ReportCollection{
		Assets: map[string]*inventory.Asset{
			assetMrn: {
				Mrn:      assetMrn,
				Name:     "pdx-fortigate-01",
				Platform: &inventory.Platform{Name: "fortios", Title: "FortiOS"},
			},
		},
		Reports: map[string]*policy.Report{
			assetMrn: {
				ScoringMrn: assetMrn,
				Score:      &policy.Score{},
				Scores:     map[string]*policy.Score{},
				Data:       map[string]*llx.Result{},
				// A risk factor is present but not detected, so the risks
				// section prints its "no downgrading risks detected" line.
				Risks: &policy.ScoredRiskFactors{
					Items: []*policy.ScoredRiskFactor{{Mrn: riskMrn, IsDetected: false}},
				},
			},
		},
		// No CHECK reporting jobs => no checks are printed for this asset.
		ResolvedPolicies: map[string]*policy.ResolvedPolicy{
			assetMrn: {
				ExecutionJob: &policy.ExecutionJob{Queries: map[string]*policy.ExecutionQuery{}},
				CollectorJob: &policy.CollectorJob{
					ReportingJobs:    map[string]*policy.ReportingJob{},
					ReportingQueries: map[string]*policy.StringArray{},
				},
			},
		},
		VulnReports: map[string]*mvd.VulnReport{
			assetMrn: {Stats: &mvd.ReportStats{Advisories: &mvd.ReportStatsAdvisories{Total: 0}}},
		},
		Bundle: &policy.Bundle{
			Policies: []*policy.Policy{{
				Mrn:         "//policy.api.mondoo.app/policies/example",
				RiskFactors: []*policy.RiskFactor{{Mrn: riskMrn, Title: "Example risk"}},
			}},
		},
	}

	buf := bytes.Buffer{}
	writer := iox.IOWriter{Writer: &buf}
	rr := &defaultReporter{
		Reporter: &Reporter{
			Conf:    defaultPrintConfig(),
			Printer: &printer.DefaultPrinter,
			Colors:  &colors.DefaultColorTheme,
		},
		output: &writer,
		data:   data,
	}
	rr.print()

	lines := strings.Split(buf.String(), NewLineCharacter)

	// Locate the first section that follows the asset header. With no checks,
	// the risks section is the first body section.
	risksIdx := -1
	for i, l := range lines {
		if strings.Contains(l, "Risks / Preventive Controls:") {
			risksIdx = i
			break
		}
	}
	require.GreaterOrEqual(t, risksIdx, 2, "expected a Risks section preceded by the asset header")

	// Exactly one blank line separates the header underline from the section:
	// the line right before "Risks" is blank, the one before that is not.
	assert.Equal(t, "", lines[risksIdx-1], "expected a single blank line before the Risks section")
	assert.NotEqual(t, "", lines[risksIdx-2], "expected exactly one blank line (no double gap) after the asset header")

	// And there should be a single blank line between sections too.
	vulnsIdx := -1
	for i, l := range lines {
		if strings.Contains(l, "Vulnerabilities:") {
			vulnsIdx = i
			break
		}
	}
	require.Greater(t, vulnsIdx, risksIdx, "expected a Vulnerabilities section after Risks")
	assert.Equal(t, "", lines[vulnsIdx-1], "expected a blank line before the Vulnerabilities section")
	assert.NotEqual(t, "", lines[vulnsIdx-2], "expected exactly one blank line between Risks and Vulnerabilities")
}

func TestVulnReporter(t *testing.T) {
	reportRaw, err := os.ReadFile("./testdata/mondoo-debug-vulnReport.json")
	require.NoError(t, err)

	report := &mvd.VulnReport{}
	err = json.Unmarshal(reportRaw, report)
	require.NoError(t, err)

	buf := bytes.Buffer{}
	writer := iox.IOWriter{Writer: &buf}
	target := "index.docker.io/library/ubuntu@669e010b58ba"

	t.Run("format=summary", func(t *testing.T) {
		conf := defaultPrintConfig().setFormat(FormatSummary)
		r := NewReporter(conf, false)
		r.out = &writer
		require.NoError(t, err)
		err = r.PrintVulns(report, target)
		require.NoError(t, err)
	})

	t.Run("format=compact", func(t *testing.T) {
		conf := defaultPrintConfig().setFormat(FormatCompact)
		r := NewReporter(conf, false)
		r.out = &writer
		err = r.PrintVulns(report, target)
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "5.5    libblkid1       2.34-0.1ubuntu9.1")
		assert.NotContains(t, buf.String(), "USN-5279-1")
	})

	t.Run("format=full", func(t *testing.T) {
		conf := defaultPrintConfig().setFormat(FormatFull)
		r := NewReporter(conf, false)
		r.out = &writer
		require.NoError(t, err)
		err = r.PrintVulns(report, target)
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "5.5    libblkid1       2.34-0.1ubuntu9.1")
		assert.Contains(t, buf.String(), "USN-5279-1")
	})

	t.Run("format=yaml", func(t *testing.T) {
		conf := defaultPrintConfig().setFormat(FormatYAMLv1)
		r := NewReporter(conf, false)
		r.out = &writer
		require.NoError(t, err)
		err = r.PrintVulns(report, target)
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "score: 5.5")
		assert.Contains(t, buf.String(), "package: libblkid1")
		assert.Contains(t, buf.String(), "installed: 2.34-0.1ubuntu9.1")
		assert.Contains(t, buf.String(), "advisory: USN-5279-1")
	})
}
