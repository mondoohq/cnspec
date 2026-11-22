// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/mql/cli/printer"
	"go.mondoo.com/mql/cli/theme/colors"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/providers-sdk/v1/upstream/mvd"
	"go.mondoo.com/mql/utils/iox"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const vulnTestAssetMrn = "//assets.api.mondoo.app/spaces/s/assets/a"

func renderVulns(t *testing.T, data *policy.ReportCollection) string {
	t.Helper()
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
	rr.printVulns(vulnTestAssetMrn)
	return buf.String()
}

func vulnTestData(assessment *policy.VulnAssessment, report *mvd.VulnReport) *policy.ReportCollection {
	data := &policy.ReportCollection{
		Assets: map[string]*inventory.Asset{
			vulnTestAssetMrn: {Mrn: vulnTestAssetMrn, Name: "vulntest", Url: "https://console.mondoo.com/asset"},
		},
		VulnReports:     map[string]*mvd.VulnReport{},
		VulnAssessments: map[string]*policy.VulnAssessment{},
	}
	if assessment != nil {
		data.VulnAssessments[vulnTestAssetMrn] = assessment
	}
	if report != nil {
		data.VulnReports[vulnTestAssetMrn] = report
	}
	return data
}

// No data must never read as zero findings: that is how a scan reported
// "No advisories found (passed)" and CVSS 0.0 for a container with 48 CVEs.
func TestPrintVulnsWithoutReportIsNotAPass(t *testing.T) {
	cases := map[string]struct {
		assessment *policy.VulnAssessment
		contains   []string
		noLink     bool
	}{
		"timed out": {
			assessment: &policy.VulnAssessment{
				State:   policy.VulnResultState_VULN_RESULT_STATE_PENDING,
				Message: "Mondoo Platform did not finish analyzing vulnerabilities within 2m0s",
			},
			contains: []string{"Not assessed: Mondoo Platform did not finish analyzing vulnerabilities within 2m0s"},
		},
		"failed": {
			assessment: &policy.VulnAssessment{
				State:   policy.VulnResultState_VULN_RESULT_STATE_FAILED,
				Message: "connection reset",
			},
			contains: []string{"Not assessed: could not get vulnerability results: connection reset"},
		},
		"platform not supported": {
			assessment: &policy.VulnAssessment{
				State:   policy.VulnResultState_VULN_RESULT_STATE_NOT_ASSESSED,
				Message: "platform vulnerabilities for terraform are not supported",
			},
			contains: []string{"Not assessed: platform vulnerabilities for terraform are not supported"},
			noLink:   true,
		},
		"ready without stats": {
			assessment: &policy.VulnAssessment{State: policy.VulnResultState_VULN_RESULT_STATE_READY},
			contains:   []string{"Not assessed: the vulnerability report has no statistics"},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var report *mvd.VulnReport
			if name == "ready without stats" {
				report = &mvd.VulnReport{}
			}
			out := renderVulns(t, vulnTestData(tc.assessment, report))
			for _, s := range tc.contains {
				assert.Contains(t, out, s)
			}
			assert.NotContains(t, out, "passed")
			assert.NotContains(t, out, "No advisories found")
			assert.NotContains(t, out, "CVSS")
			if tc.noLink {
				assert.NotContains(t, out, "https://console.mondoo.com/asset")
			} else {
				assert.Contains(t, out, "See the results in Mondoo Platform: https://console.mondoo.com/asset")
			}
		})
	}
}

func TestPrintVulnsZeroFindingsIsAPass(t *testing.T) {
	report := &mvd.VulnReport{Stats: &mvd.ReportStats{Advisories: &mvd.ReportStatsAdvisories{}}}
	out := renderVulns(t, vulnTestData(&policy.VulnAssessment{State: policy.VulnResultState_VULN_RESULT_STATE_READY}, report))
	assert.Contains(t, out, "No advisories found")
	assert.Contains(t, out, "Overall CVSS score: 0.0")
	assert.NotContains(t, out, "Not assessed")
}

func TestPrintVulnsNewerScan(t *testing.T) {
	newer := time.Date(2026, 11, 21, 10, 0, 0, 0, time.UTC)
	report := &mvd.VulnReport{Stats: &mvd.ReportStats{Advisories: &mvd.ReportStatsAdvisories{}}}
	out := renderVulns(t, vulnTestData(&policy.VulnAssessment{
		State:         policy.VulnResultState_VULN_RESULT_STATE_READY,
		NewerScan:     true,
		NewerScanTime: timestamppb.New(newer),
	}, report))
	assert.Contains(t, out, "Results from a newer scan of this asset ("+newer.Local().Format(time.RFC3339)+")")
}

func TestPrintVulnsNothingToSay(t *testing.T) {
	assert.Empty(t, renderVulns(t, vulnTestData(nil, nil)))
}

func TestJunitVulnsNotAssessedIsSkipped(t *testing.T) {
	data := vulnTestData(&policy.VulnAssessment{
		State:   policy.VulnResultState_VULN_RESULT_STATE_PENDING,
		Message: "timed out",
	}, nil)
	ts := assetMvdTests(data, vulnTestAssetMrn, data.Assets[vulnTestAssetMrn])
	require.NotNil(t, ts)
	assert.Equal(t, 1, ts.Skipped)
	assert.Equal(t, 0, ts.Failures)
	require.Len(t, ts.Testcases, 1)
	require.NotNil(t, ts.Testcases[0].Skipped)
	assert.Equal(t, "timed out", ts.Testcases[0].Skipped.Message)
}
