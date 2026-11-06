// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"sync"

	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
)

// NoOpReporter keeps no asset reports and no errors, so a scan with
// ReportType_NONE prints nothing. It still tracks whether every asset
// scanned and the worst score it saw, because the CLI exit status is derived
// from ScanResult.Ok and ScanResult.WorstScore for every report type.
type NoOpReporter struct {
	mu         sync.Mutex
	failed     bool
	worstScore *policy.Score
}

func NewNoOpReporter() Reporter {
	return &NoOpReporter{}
}

func (r *NoOpReporter) AddBundle(bundle *policy.Bundle) {}

func (r *NoOpReporter) AddReport(asset *inventory.Asset, results *AssetReport) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.worstScore = lowerScore(r.worstScore, results.Report.GetScore())
}

func (r *NoOpReporter) AddScanError(asset *inventory.Asset, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.failed = true
}

func (r *NoOpReporter) Reports() *ScanResult {
	r.mu.Lock()
	defer r.mu.Unlock()
	return &ScanResult{
		Ok:         !r.failed,
		WorstScore: r.worstScore,
		Result:     &ScanResult_None{None: &Empty{}},
	}
}
