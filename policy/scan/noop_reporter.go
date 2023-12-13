// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/upstream/gql"
)

type NoOpReporter struct{}

func NewNoOpReporter() Reporter {
	return &NoOpReporter{}
}

func (r *NoOpReporter) AddReport(asset *inventory.Asset, results *AssetReport) {
}

func (r *NoOpReporter) AddVulnReport(asset *inventory.Asset, vulnReport *gql.VulnReport) {
}

func (r *NoOpReporter) AddScanError(asset *inventory.Asset, err error) {
}

func (r *NoOpReporter) Reports() *ScanResult {
	return &ScanResult{
		Result: &ScanResult_None{},
	}
}
