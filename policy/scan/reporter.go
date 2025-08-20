// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/upstream/gql"
	"go.mondoo.com/cnspec/v12/policy"
)

type AssetReport struct {
	Mrn            string
	ResolvedPolicy *policy.ResolvedPolicy
	Report         *policy.Report
}

type VulnReporter interface {
	// AddVulnReport adds the vulnerability scan results to the reporter
	AddVulnReport(asset *inventory.Asset, vulnReport *gql.VulnReport)
}

type Reporter interface {
	// AddBundle adds the policy bundle to the reporter which includes more information about the policies
	AddBundle(bundle *policy.Bundle)
	// AddReport adds the scan results to the reporter
	AddReport(asset *inventory.Asset, results *AssetReport)
	// AddScanError adds the scan error to the reporter
	AddScanError(asset *inventory.Asset, err error)
	// Reports returns the scan results
	Reports() *ScanResult
}
