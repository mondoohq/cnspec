// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/upstream/gql"
	"go.mondoo.com/cnspec/v9/policy"
)

type AssetReport struct {
	Mrn            string
	ResolvedPolicy *policy.ResolvedPolicy
	Report         *policy.Report
}

type Reporter interface {
	AddReport(asset *inventory.Asset, results *AssetReport)
	AddVulnReport(asset *inventory.Asset, vulnReport *gql.VulnReport)
	AddScanError(asset *inventory.Asset, err error)
	Reports() *ScanResult
}
