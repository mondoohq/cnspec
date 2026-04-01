// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"fmt"
	"sort"
	"strings"

	"github.com/owenrumney/go-sarif/v2/sarif"
	"go.mondoo.com/cnspec/v13/policy"
	"go.mondoo.com/mql/v13/cli/printer"
	"go.mondoo.com/mql/v13/mrn"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/utils/iox"
)

const sarifAssetErrorRuleID = "asset-error"

// ConvertToSarif converts a ReportCollection into a SARIF 2.1.0 report.
// Each scanned asset is represented as a separate SARIF run.
func ConvertToSarif(r *policy.ReportCollection, out iox.OutputHelper) error {
	report, err := sarif.New(sarif.Version210)
	if err != nil {
		return err
	}

	if r == nil {
		return writeSarif(report, out)
	}

	if r.Bundle == nil {
		return fmt.Errorf("no policy bundle found")
	}

	bundle := r.Bundle.ToMap()
	queries := bundle.QueryMap()

	// Create one run per asset (deterministic order via sorted keys)
	assetMrns := sortedKeys(r.Assets)
	for _, assetMrn := range assetMrns {
		assetObj := r.Assets[assetMrn]
		run := newAssetRun(assetObj)

		// Register the asset-error rule if this asset has an error
		if _, hasErr := r.Errors[assetMrn]; hasErr {
			run.AddRule(sarifAssetErrorRuleID).
				WithName("Asset scan error").
				WithDescription("The asset could not be scanned successfully")
		}

		// Register reporting queries applicable to this asset as SARIF rules
		registerAssetRules(run, r, assetMrn, queries)

		// Emit results for this asset
		addAssetErrors(run, r, assetMrn, assetObj)
		addAssetResults(run, r, assetMrn, assetObj, queries)

		report.AddRun(run)
	}

	return writeSarif(report, out)
}

// newAssetRun creates a new SARIF run for a given asset
func newAssetRun(asset *inventory.Asset) *sarif.Run {
	run := sarif.NewRunWithInformationURI("cnspec", "https://cnspec.io")
	// Tag the run with asset metadata so consumers can identify which asset it covers
	props := sarif.Properties{"asset": asset.Name}
	if asset.Platform != nil {
		platformName := getPlatformNameForAsset(asset)
		if platformName != "" {
			props["platform"] = platformName
		}
	}
	run.Properties = props
	return run
}

// registerAssetRules registers the reporting queries for a single asset as SARIF rules on the run
func registerAssetRules(run *sarif.Run, r *policy.ReportCollection, assetMrn string, queries map[string]*policy.Mquery) {
	resolved, ok := r.ResolvedPolicies[assetMrn]
	if !ok || resolved.CollectorJob == nil {
		return
	}
	queryIDs := sortedKeys(resolved.CollectorJob.ReportingQueries)
	for _, id := range queryIDs {
		query, ok := queries[id]
		if !ok {
			continue
		}

		ruleID := queryRuleID(query)
		rb := run.AddRule(ruleID)
		if query.Title != "" {
			rb.WithName(query.Title)
		}
		desc := queryDescription(query)
		if desc != "" {
			rb.WithDescription(desc)
		}
		if query.Impact != nil && query.Impact.Value != nil {
			rb.WithProperties(sarif.Properties{
				"impact": query.Impact.Value.GetValue(),
			})
		}
	}
}

func addAssetErrors(run *sarif.Run, r *policy.ReportCollection, assetMrn string, assetObj *inventory.Asset) {
	errMsg, ok := r.Errors[assetMrn]
	if !ok {
		return
	}
	result := sarif.NewRuleResult(sarifAssetErrorRuleID).
		WithLevel("error").
		WithMessage(sarif.NewTextMessage(fmt.Sprintf("Asset %s: %s", assetObj.Name, errMsg)))
	run.AddResult(result)
}

func addAssetResults(run *sarif.Run, r *policy.ReportCollection, assetMrn string, assetObj *inventory.Asset, queries map[string]*policy.Mquery) {
	report, ok := r.Reports[assetMrn]
	if !ok {
		return
	}

	resolved, ok := r.ResolvedPolicies[assetMrn]
	if !ok || resolved.CollectorJob == nil {
		return
	}

	// Sort score IDs for deterministic output
	scoreIDs := sortedKeys(report.Scores)
	for _, id := range scoreIDs {
		score := report.Scores[id]

		if _, ok := resolved.CollectorJob.ReportingQueries[id]; !ok {
			continue
		}

		query, ok := queries[id]
		if !ok {
			continue
		}

		ruleID := queryRuleID(query)
		level := scoreToSarifLevel(score)

		msg := query.Title
		if msg == "" {
			msg = ruleID
		}
		if score != nil && score.Message != "" {
			msg += ": " + score.MessageLine()
		}

		// Render the assessment (expected vs actual) for failed checks
		assessmentText := renderAssessment(query, report, resolved)
		if assessmentText != "" {
			msg += "\n\n" + assessmentText
		}

		result := sarif.NewRuleResult(ruleID).
			WithLevel(level).
			WithMessage(sarif.NewTextMessage(msg))

		// Add asset information as a logical location
		logicalLoc := sarif.NewLogicalLocation().
			WithName(assetObj.Name).
			WithKind("asset")
		if assetObj.Platform != nil {
			platformName := getPlatformNameForAsset(assetObj)
			if platformName != "" {
				logicalLoc.WithFullyQualifiedName(assetObj.Name + " (" + platformName + ")")
			}
		}
		result.WithLocations([]*sarif.Location{
			sarif.NewLocation().WithLogicalLocations([]*sarif.LogicalLocation{logicalLoc}),
		})

		run.AddResult(result)
	}
}

// renderAssessment renders the assessment (expected vs actual values) for a query as plain text.
func renderAssessment(query *policy.Mquery, report *policy.Report, resolved *policy.ResolvedPolicy) string {
	if resolved.ExecutionJob == nil {
		return ""
	}

	codeBundle := resolved.GetCodeBundle(query)
	if codeBundle == nil {
		return ""
	}

	assessment := policy.Query2Assessment(codeBundle, report)
	if assessment == nil {
		return ""
	}

	return strings.TrimSpace(printer.PlainNoColorPrinter.Assessment(codeBundle, assessment))
}

// scoreToSarifLevel maps a cnspec Score to a SARIF level using cnspec's
// severity rating system:
//
//	100        → "none"    (pass)
//	61 .. 99   → "note"    (Low severity)
//	31 .. 60   → "warning" (Medium severity)
//	 0 .. 30   → "error"   (High/Critical severity)
func scoreToSarifLevel(score *policy.Score) string {
	if score == nil {
		return "none"
	}

	switch score.Type {
	case policy.ScoreType_Error:
		return "error"
	case policy.ScoreType_Skip, policy.ScoreType_Unscored, policy.ScoreType_OutOfScope, policy.ScoreType_Disabled:
		return "none"
	case policy.ScoreType_Unknown:
		return "none"
	case policy.ScoreType_Result:
		if score.Value == 100 {
			return "none" // pass
		}
		if score.Value >= 61 {
			return "note" // Low severity
		}
		if score.Value >= 31 {
			return "warning" // Medium severity
		}
		return "error" // High/Critical severity
	default:
		return "none"
	}
}

// queryRuleID returns a stable, human-readable rule ID for a query.
// It prefers the UID, then extracts the resource name from the MRN
// (stripping prefixes like //local.cnspec.io/run/local-execution/queries/),
// and falls back to the code ID.
func queryRuleID(query *policy.Mquery) string {
	if query.Uid != "" {
		return query.Uid
	}
	if query.Mrn != "" {
		if name, err := mrn.GetResource(query.Mrn, policy.MRN_RESOURCE_QUERY); err == nil {
			return name
		}
		return query.Mrn
	}
	return query.CodeId
}

// queryDescription extracts a description from a query
func queryDescription(query *policy.Mquery) string {
	if query.Docs != nil && query.Docs.Desc != "" {
		return query.Docs.Desc
	}
	if query.Desc != "" {
		return query.Desc
	}
	return ""
}

func writeSarif(report *sarif.Report, out iox.OutputHelper) error {
	return report.Write(out)
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
