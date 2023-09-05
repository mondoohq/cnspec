// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
	"go.mondoo.com/cnspec/policy"
)

type AssetReport struct {
	Mrn            string
	ResolvedPolicy *policy.ResolvedPolicy
	Bundle         *policy.Bundle
	Report         *policy.Report
}

type Reporter interface {
	AddReport(asset *inventory.Asset, results *AssetReport)
	AddScanError(asset *inventory.Asset, err error)
	Reports() *ScanResult
}
