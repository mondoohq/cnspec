// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// The walk over an asset's checks, and the choice of which class each one goes
// out as. Both classes report every check; what the event looks like is the
// business of compliance.go and detection.go, and neither of them holds the
// other's branch.

package convert

import (
	"maps"
	"slices"

	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/cnspec/reports/ocsf"
)

// addCheckFindings emits one finding per reporting check of an asset.
func (c *converter) addCheckFindings(events *ocsf.Events, r *policy.ReportCollection, ctx *assetContext, queries map[string]*policy.Mquery) {
	report, ok := r.Reports[ctx.assetMrn]
	if !ok {
		return
	}
	resolved, ok := r.ResolvedPolicies[ctx.assetMrn]
	if !ok || resolved.CollectorJob == nil {
		return
	}

	for _, id := range slices.Sorted(maps.Keys(report.Scores)) {
		score := report.Scores[id]
		if _, ok := resolved.CollectorJob.ReportingQueries[id]; !ok {
			continue
		}
		query, ok := queries[id]
		if !ok {
			continue
		}

		// Both classes report every check, passing ones included, with the outcome
		// in status_code. A detection stream that dropped the passes would not be
		// a complete record of the scan on its own.
		if c.detection() {
			events.DetectionFindings = append(events.DetectionFindings,
				c.detectionFinding(resolved, report, query, score, ctx))
		} else {
			events.ComplianceFindings = append(events.ComplianceFindings,
				c.complianceFinding(resolved, report, query, score, ctx))
		}
	}
}

// assetErrorUID identifies the finding that stands in for a whole failed scan.
const assetErrorUID = "asset-error"

// addAssetError reports an asset that could not be scanned at all. Without it the
// asset would silently look clean: it has no checks and therefore no findings.
func (c *converter) addAssetError(events *ocsf.Events, r *policy.ReportCollection, ctx *assetContext) {
	errMsg, ok := r.Errors[ctx.assetMrn]
	if !ok || errMsg == "" {
		return
	}

	if c.detection() {
		events.DetectionFindings = append(events.DetectionFindings, c.detectionAssetError(errMsg, ctx))
		return
	}
	events.ComplianceFindings = append(events.ComplianceFindings, c.complianceAssetError(errMsg, ctx))
}

// assetErrorInfo is the finding_info both classes give a failed scan.
func (c *converter) assetErrorInfo() ocsf.FindingInfo {
	return ocsf.FindingInfo{
		UID:         assetErrorUID,
		Title:       "Asset scan error",
		Desc:        "cnspec could not complete the scan of this asset. No policy results are available for it.",
		CreatedTime: c.now,
		Types:       []string{"Scan Error"},
		DataSources: []string{productName},
	}
}

// assetErrorUnmapped is what a failed scan carries that OCSF has no attribute
// for. There is no check behind it, so the asset is all there is to say.
func assetErrorUnmapped(ctx *assetContext) map[string]string {
	res := map[string]string{}
	if ctx.assetMrn != "" {
		res["asset_mrn"] = ctx.assetMrn
	}
	return res
}

// assetErrorSeverity is High because a scan that did not run leaves every check
// unanswered, which is worse than a single check erroring out.
const assetErrorSeverity = ocsf.SeverityHigh
