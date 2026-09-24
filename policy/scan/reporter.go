// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/providers-sdk/v1/upstream/gql"
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
	// AddScanError adds the scan error to the reporter. It marks the asset as
	// having failed to scan: implementations may exclude it from a
	// successful-looking result and flip the run's exit code. Use it only
	// when the asset produced no usable report.
	AddScanError(asset *inventory.Asset, err error)
	// AddScanWarning records a non-fatal issue observed while scanning an
	// asset that still produced a report via AddReport -- e.g. a provider
	// that crashed partway through and left some fields errored. Unlike
	// AddScanError, this must never exclude the asset's report or flip the
	// run's exit code; it is visibility, not a failure signal.
	AddScanWarning(asset *inventory.Asset, warnings []string)
	// Reports returns the scan results
	Reports() *ScanResult
}
