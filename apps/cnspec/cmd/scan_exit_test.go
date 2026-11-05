// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnspec/cli/reporter"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/cnspec/policy/scan"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
)

// scanResultFor runs the same sequence of reporter calls through the reporter
// each report type uses, so the test judges exactly the ScanResult the scan
// command sees for that --report-type.
func scanResultFor(t *testing.T, reportType scan.ReportType, scores []uint32, scanErr error) *scan.ScanResult {
	t.Helper()
	var r scan.Reporter
	switch reportType {
	case scan.ReportType_FULL:
		r = scan.NewAggregateReporter()
	case scan.ReportType_ERROR:
		r = scan.NewErrorReporter()
	case scan.ReportType_NONE:
		r = scan.NewNoOpReporter()
	default:
		t.Fatalf("unknown report type %v", reportType)
	}
	for i, v := range scores {
		asset := &inventory.Asset{Mrn: "//assets/" + string(rune('a'+i)), Name: "asset"}
		r.AddReport(asset, &scan.AssetReport{
			Mrn:    asset.Mrn,
			Report: &policy.Report{Score: &policy.Score{Type: policy.ScoreType_Result, Value: v, ScoreCompletion: 100}},
		})
	}
	if scanErr != nil {
		r.AddScanError(&inventory.Asset{Mrn: "//assets/broken", Name: "broken"}, scanErr)
	}
	return r.Reports()
}

func TestScanExitCode_SameForEveryReportType(t *testing.T) {
	reportTypes := []scan.ReportType{scan.ReportType_FULL, scan.ReportType_ERROR, scan.ReportType_NONE}

	cases := []struct {
		name      string
		scores    []uint32
		scanErr   error
		threshold int
		want      int
	}{
		{
			name:      "clean scan exits 0",
			scores:    []uint32{90, 75},
			threshold: reporter.DEFAULT_RISK_THRESHOLD,
			want:      0,
		},
		{
			name:      "asset error exits 1",
			scores:    []uint32{100},
			scanErr:   errors.New("dial tcp 127.0.0.1:1: connect: connection refused"),
			threshold: reporter.DEFAULT_RISK_THRESHOLD,
			want:      1,
		},
		{
			name:      "only errors and no reports exits 1",
			scanErr:   errors.New("connection refused"),
			threshold: reporter.DEFAULT_RISK_THRESHOLD,
			want:      1,
		},
		{
			name:      "worst score at the threshold exits 1",
			scores:    []uint32{90, 60}, // risk 40
			threshold: 40,
			want:      1,
		},
		{
			name:      "worst score below the threshold exits 0",
			scores:    []uint32{90, 61}, // risk 39
			threshold: 40,
			want:      0,
		},
		{
			name:      "no reports and default threshold exits 0",
			threshold: reporter.DEFAULT_RISK_THRESHOLD,
			want:      0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, rt := range reportTypes {
				res := scanResultFor(t, rt, tc.scores, tc.scanErr)
				assert.Equal(t, tc.want, scanExitCode(res, tc.threshold), "report type %s", rt)
			}
		})
	}
}

func TestScanExitCode_FullIgnoresUnratedAssets(t *testing.T) {
	// The full report is the reference: an unrated asset carries Value 0 but
	// must not read as risk 100. Error and none agree.
	unrated := &policy.Score{Type: policy.ScoreType_Unscored, ScoreCompletion: 100}
	add := func(r scan.Reporter) *scan.ScanResult {
		r.AddReport(&inventory.Asset{Mrn: "//assets/rated"}, &scan.AssetReport{Report: &policy.Report{Score: &policy.Score{Type: policy.ScoreType_Result, Value: 95, ScoreCompletion: 100}}})
		r.AddReport(&inventory.Asset{Mrn: "//assets/unrated"}, &scan.AssetReport{Report: &policy.Report{Score: unrated}})
		return r.Reports()
	}
	for _, r := range []scan.Reporter{scan.NewAggregateReporter(), scan.NewErrorReporter(), scan.NewNoOpReporter()} {
		assert.Equal(t, 0, scanExitCode(add(r), 50))
	}
}

func TestScanExitCode_NilResult(t *testing.T) {
	assert.Equal(t, 0, scanExitCode(nil, reporter.DEFAULT_RISK_THRESHOLD))
}
