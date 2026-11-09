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
	// ExceptionDecisions accounts for every exception read from config for
	// the asset (ADR-0006); nil when there were none.
	ExceptionDecisions *policy.ExceptionDecisions
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

// lowerScore returns the lower of the two scores for ScanResult.WorstScore.
// Unrated scores are skipped: their Value is 0, which would otherwise read as
// the worst possible score. This mirrors ReportCollection.GetWorstScore, so a
// scan exits the same way whichever report type it ran with.
func lowerScore(current, next *policy.Score) *policy.Score {
	if next == nil || next.Rating() == policy.ScoreRating_unrated {
		return current
	}
	if current == nil || next.Value < current.Value {
		return next
	}
	return current
}
