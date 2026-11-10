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

// checkReport is the AssetReport of an asset with one check whose score is
// checkScore, under an asset score of assetScore. The two differ on a real
// asset whenever other policies average the asset score up.
func checkReport(assetMrn string, assetScore, checkScore uint32) *AssetReport {
	checkMrn := assetMrn + "/checks/c"
	return &AssetReport{
		Mrn: assetMrn,
		ResolvedPolicy: &policy.ResolvedPolicy{CollectorJob: &policy.CollectorJob{
			ReportingJobs: map[string]*policy.ReportingJob{
				"check": {Uuid: "check", QrId: checkMrn, Type: policy.ReportingJob_CHECK},
			},
		}},
		Report: &policy.Report{
			Score: &policy.Score{Type: policy.ScoreType_Result, Value: assetScore, ScoreCompletion: 100},
			Scores: map[string]*policy.Score{
				checkMrn: {QrId: checkMrn, Type: policy.ScoreType_Result, Value: checkScore, ScoreCompletion: 100},
			},
		},
	}
}

// scanResultFor runs the same sequence of reporter calls through the reporter
// each report type uses, so the test judges exactly the ScanResult the scan
// command sees for that --report-type. Each asset has one check scored
// checkScores[i] and an asset score of 100, so only the check can fail it.
func scanResultFor(t *testing.T, reportType ReportType, checkScores []uint32, scanErr error) *ScanResult {
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
	for i, v := range checkScores {
		asset := &inventory.Asset{Mrn: "//assets/" + string(rune('a'+i)), Name: "asset"}
		r.AddReport(asset, checkReport(asset.Mrn, 100, v))
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
			name:      "worst check at the threshold fails",
			scores:    []uint32{90, 60}, // risk 40
			threshold: 40,
			want:      true,
		},
		{
			name:      "worst check below the threshold passes",
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

func TestScanResultFailed_JudgesChecksNotTheAssetScore(t *testing.T) {
	// One critical finding next to a passing policy averages the asset to 50
	// (risk 50). The threshold must still trip on the finding's risk of 100.
	for _, r := range []Reporter{NewAggregateReporter(), NewErrorReporter(), NewNoOpReporter()} {
		r.AddReport(&inventory.Asset{Mrn: "//assets/a"}, checkReport("//assets/a", 50, 0))
		res := r.Reports()
		assert.Equal(t, uint32(50), res.WorstScoreValue())
		assert.Equal(t, uint32(0), res.WorstCheckScoreValue())
		assert.True(t, res.Failed(70))
		assert.True(t, res.Failed(100))
	}

	// The reverse: a low asset score with no check over the threshold passes.
	for _, r := range []Reporter{NewAggregateReporter(), NewErrorReporter(), NewNoOpReporter()} {
		r.AddReport(&inventory.Asset{Mrn: "//assets/a"}, checkReport("//assets/a", 10, 80))
		res := r.Reports()
		assert.False(t, res.Failed(21))
		assert.True(t, res.Failed(20))
	}
}

func TestScanResultFailed_IgnoresUnratedAssets(t *testing.T) {
	// An unrated asset carries Value 0 but must not read as risk 100, the
	// same way ReportCollection.GetWorstScore skips it.
	unrated := &policy.Score{Type: policy.ScoreType_Unscored, ScoreCompletion: 100}
	add := func(r Reporter) *ScanResult {
		r.AddReport(&inventory.Asset{Mrn: "//assets/rated"}, checkReport("//assets/rated", 95, 95))
		r.AddReport(&inventory.Asset{Mrn: "//assets/unrated"}, &AssetReport{Report: &policy.Report{Score: unrated}})
		return r.Reports()
	}
	for _, r := range []Reporter{NewAggregateReporter(), NewErrorReporter(), NewNoOpReporter()} {
		res := add(r)
		assert.Equal(t, uint32(95), res.WorstScoreValue())
		assert.Equal(t, uint32(95), res.WorstCheckScoreValue())
		assert.False(t, res.Failed(50))
	}
}

func TestScanResultWorstScoreValue_NoRatedScore(t *testing.T) {
	assert.Equal(t, uint32(100), (&ScanResult{}).WorstScoreValue())
	assert.Equal(t, uint32(100), (&ScanResult{WorstScore: &policy.Score{Type: policy.ScoreType_Unscored, ScoreCompletion: 100}}).WorstScoreValue())
	assert.Equal(t, uint32(0), (&ScanResult{WorstScore: &policy.Score{Type: policy.ScoreType_Result, Value: 0, ScoreCompletion: 100}}).WorstScoreValue())
}

func TestScanResultWorstCheckScoreValue_NoRatedScore(t *testing.T) {
	assert.Equal(t, uint32(100), (&ScanResult{}).WorstCheckScoreValue())
	assert.Equal(t, uint32(100), (&ScanResult{WorstCheckScore: &policy.Score{Type: policy.ScoreType_Result, Value: 0}}).WorstCheckScoreValue())
	assert.Equal(t, uint32(0), (&ScanResult{WorstCheckScore: &policy.Score{Type: policy.ScoreType_Result, Value: 0, ScoreCompletion: 100}}).WorstCheckScoreValue())
}

func TestScanResultFailed_Nil(t *testing.T) {
	var res *ScanResult
	assert.False(t, res.Failed(defaultRiskThreshold))
	assert.Equal(t, uint32(100), res.WorstScoreValue())
	assert.Equal(t, uint32(100), res.WorstCheckScoreValue())
}
