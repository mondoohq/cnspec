// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnspec/policy"
)

type AssetReport struct {
	Mrn            string
	ResolvedPolicy *policy.ResolvedPolicy
	Bundle         *policy.Bundle
	Report         *policy.Report
}

type Reporter interface {
	AddReport(asset *asset.Asset, results *AssetReport)
	AddScanError(asset *asset.Asset, err error)
	Reports() *ScanResult
}
