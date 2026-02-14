// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"go.mondoo.com/cnspec/v13/policy"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
)

type NoOpReporter struct{}

func NewNoOpReporter() Reporter {
	return &NoOpReporter{}
}

func (r *NoOpReporter) AddBundle(bundle *policy.Bundle) {}

func (r *NoOpReporter) AddReport(asset *inventory.Asset, results *AssetReport) {
}

func (r *NoOpReporter) AddScanError(asset *inventory.Asset, err error) {
}

func (r *NoOpReporter) Reports() *ScanResult {
	return &ScanResult{
		Result: &ScanResult_None{},
	}
}
