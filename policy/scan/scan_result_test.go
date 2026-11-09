// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
)

// defaultRiskThreshold mirrors the CLI default (cli/reporter.DEFAULT_RISK_THRESHOLD),
// which no score can meet, so only asset errors fail a scan.
const defaultRiskThreshold = 101

// scanResultFor runs the same sequence of reporter calls through the reporter
// each report type uses, so the test judges exactly the ScanResult the scan
// command sees for that --report-type.
func scanResultFor(t *testing.T, reportType ReportType, scores []uint32, scanErr error) *ScanResult {
	t.Helper()
	var r Reporter
	switch reportType {
	case ReportType_FULL:
		r = NewAggregateReporter()
	case ReportType_ERROR:
		r = NewErrorReporter()
	case ReportType_NONE:
		r = NewNoOpReporter()
	default:
		t.Fatalf("unknown report type %v", reportType)
	}
	for i, v := range scores {
		asset := &inventory.Asset{Mrn: "//assets/" + string(rune('a'+i)), Name: "asset"}
		r.AddReport(asset, &AssetReport{
			Mrn:    asset.Mrn,
			Report: &policy.Report{Score: &policy.Score{Type: policy.ScoreType_Result, Value: v, ScoreCompletion: 100}},
		})
	}
	if scanErr != nil {
		r.AddScanError(&inventory.Asset{Mrn: "//assets/broken", Name: "broken"}, scanErr)
	}
	return r.Reports()
}

func TestScanResultFailed_SameForEveryReportType(t *testing.T) {
	reportTypes := []ReportType{ReportType_FULL, ReportType_ERROR, ReportType_NONE}

	cases := []struct {
		name      string
		scores    []uint32
		scanErr   error
		threshold int
		want      bool
	}{
		{
			name:      "clean scan passes",
			scores:    []uint32{90, 75},
			threshold: defaultRiskThreshold,
			want:      false,
		},
		{
			name:      "asset error fails",
			scores:    []uint32{100},
			scanErr:   errors.New("dial tcp 127.0.0.1:1: connect: connection refused"),
			threshold: defaultRiskThreshold,
			want:      true,
		},
		{
			name:      "only errors and no reports fails",
			scanErr:   errors.New("connection refused"),
			threshold: defaultRiskThreshold,
			want:      true,
		},
		{
			name:      "worst score at the threshold fails",
			scores:    []uint32{90, 60}, // risk 40
			threshold: 40,
			want:      true,
		},
		{
			name:      "worst score below the threshold passes",
			scores:    []uint32{90, 61}, // risk 39
			threshold: 40,
			want:      false,
		},
		{
			name:      "no reports and default threshold passes",
			threshold: defaultRiskThreshold,
			want:      false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, rt := range reportTypes {
				res := scanResultFor(t, rt, tc.scores, tc.scanErr)
				assert.Equal(t, tc.want, res.Failed(tc.threshold), "report type %s", rt)
			}
		})
	}
}

func TestScanResultFailed_MatchesFullCollection(t *testing.T) {
	// A full result carries the report collection, whose GetWorstScore and
	// Errors decided the exit status before WorstScore and Ok were used for
	// every report type. Both views must agree on the same result.
	for _, tc := range []struct {
		name    string
		scores  []uint32
		scanErr error
	}{
		{name: "clean", scores: []uint32{90, 75}},
		{name: "rated zero", scores: []uint32{100, 0}},
		{name: "asset error", scores: []uint32{100}, scanErr: errors.New("connection refused")},
		{name: "no reports"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res := scanResultFor(t, ReportType_FULL, tc.scores, tc.scanErr)
			full := res.GetFull()
			assert.Equal(t, full.GetWorstScore(), res.WorstScoreValue())
			assert.Equal(t, len(full.Errors) > 0, !res.Ok)
			for _, th := range []int{1, 50, 100, defaultRiskThreshold} {
				want := len(full.Errors) > 0 || (100-full.GetWorstScore()) >= uint32(th)
				assert.Equal(t, want, res.Failed(th), "threshold %d", th)
			}
		})
	}
}

func TestScanResultFailed_IgnoresUnratedAssets(t *testing.T) {
	// An unrated asset carries Value 0 but must not read as risk 100, the
	// same way ReportCollection.GetWorstScore skips it.
	unrated := &policy.Score{Type: policy.ScoreType_Unscored, ScoreCompletion: 100}
	add := func(r Reporter) *ScanResult {
		r.AddReport(&inventory.Asset{Mrn: "//assets/rated"}, &AssetReport{Report: &policy.Report{Score: &policy.Score{Type: policy.ScoreType_Result, Value: 95, ScoreCompletion: 100}}})
		r.AddReport(&inventory.Asset{Mrn: "//assets/unrated"}, &AssetReport{Report: &policy.Report{Score: unrated}})
		return r.Reports()
	}
	for _, r := range []Reporter{NewAggregateReporter(), NewErrorReporter(), NewNoOpReporter()} {
		res := add(r)
		assert.Equal(t, uint32(95), res.WorstScoreValue())
		assert.False(t, res.Failed(50))
	}
}

func TestScanResultWorstScoreValue_NoRatedScore(t *testing.T) {
	assert.Equal(t, uint32(100), (&ScanResult{}).WorstScoreValue())
	assert.Equal(t, uint32(100), (&ScanResult{WorstScore: &policy.Score{Type: policy.ScoreType_Unscored, ScoreCompletion: 100}}).WorstScoreValue())
	assert.Equal(t, uint32(0), (&ScanResult{WorstScore: &policy.Score{Type: policy.ScoreType_Result, Value: 0, ScoreCompletion: 100}}).WorstScoreValue())
}

func TestScanResultFailed_Nil(t *testing.T) {
	var res *ScanResult
	assert.False(t, res.Failed(defaultRiskThreshold))
	assert.Equal(t, uint32(100), res.WorstScoreValue())
}
