// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/owenrumney/go-sarif/v2/sarif"
	"go.mondoo.com/cnspec/v13/policy"
	"go.mondoo.com/mql/v13/cli/printer"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/mrn"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/utils/iox"
)

const sarifAssetErrorRuleID = "asset-error"

// sarifFingerprintKey is the key under which cnspec stores its stable
// per-finding fingerprint in SARIF partialFingerprints. It uses a URI-style
// namespace we own so it can't collide with well-known keys used by other
// tooling (e.g. GitHub code scanning's primaryLocationLineHash). The trailing
// version lets us evolve the fingerprint algorithm without remapping old runs.
const sarifFingerprintKey = "https://cnspec.io/fingerprint/v1"

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

	// The asset is recorded as a logical location on every result so consumers
	// can still tell which asset a finding belongs to.
	logicalLoc := assetLogicalLocation(assetObj)

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

		// Build the assessment once and reuse it for both the human-readable
		// detail and the structured source locations.
		var assessment *llx.Assessment
		var codeBundle *llx.CodeBundle
		if resolved.ExecutionJob != nil {
			codeBundle = resolved.GetCodeBundle(query)
			if codeBundle != nil {
				assessment = policy.Query2Assessment(codeBundle, report)
			}
		}

		// Source locations of the failing resources. This covers terraform and
		// any resource that carries @context data; it is empty for scalar checks
		// or resources without context.
		var locations []llx.SourceContext
		if assessment != nil && codeBundle != nil {
			for _, sc := range codeBundle.FailingResourceContexts(assessment) {
				if sc.Path != "" {
					locations = append(locations, sc)
				}
			}
		}

		if len(locations) == 0 {
			// No source context: anchor a single result to the asset and keep the
			// full assessment detail (expected vs actual) in the message.
			detail := msg
			if assessment != nil {
				if text := strings.TrimSpace(printer.PlainNoColorPrinter.Assessment(codeBundle, assessment)); text != "" {
					detail += "\n\n" + text
				}
			}
			result := sarif.NewRuleResult(ruleID).
				WithLevel(level).
				WithMessage(sarif.NewTextMessage(detail)).
				WithLocations([]*sarif.Location{
					sarif.NewLocation().WithLogicalLocations([]*sarif.LogicalLocation{logicalLoc}),
				})
			run.AddResult(result)
			continue
		}

		// One result per failing resource, each pointing at its exact source.
		// The code snippet travels in the region; the message stays concise.
		for i := range locations {
			loc := sarif.NewLocationWithPhysicalLocation(physicalLocationFromContext(locations[i])).
				WithLogicalLocations([]*sarif.LogicalLocation{logicalLoc})
			result := sarif.NewRuleResult(ruleID).
				WithLevel(level).
				WithMessage(sarif.NewTextMessage(msg)).
				WithLocations([]*sarif.Location{loc}).
				WithPartialFingerPrints(map[string]interface{}{
					sarifFingerprintKey: sarifLocationFingerprint(ruleID, locations[i]),
				})
			run.AddResult(result)
		}
	}
}

// assetLogicalLocation builds the SARIF logical location that identifies an asset.
func assetLogicalLocation(assetObj *inventory.Asset) *sarif.LogicalLocation {
	logicalLoc := sarif.NewLogicalLocation().
		WithName(assetObj.Name).
		WithKind("asset")
	if assetObj.Platform != nil {
		platformName := getPlatformNameForAsset(assetObj)
		if platformName != "" {
			logicalLoc.WithFullyQualifiedName(assetObj.Name + " (" + platformName + ")")
		}
	}
	return logicalLoc
}

// physicalLocationFromContext maps an MQL source context (path + range + content)
// to a SARIF physical location with a region and, when available, a code snippet.
func physicalLocationFromContext(ctx llx.SourceContext) *sarif.PhysicalLocation {
	pl := sarif.NewPhysicalLocation().
		WithArtifactLocation(sarif.NewSimpleArtifactLocation(ctx.Path))

	startLine, startCol, endLine, endCol, hasCols, ok := ctx.Range.Bounds()
	var region *sarif.Region
	if ok && startLine >= 1 {
		region = sarif.NewRegion().
			WithStartLine(int(startLine)).
			WithEndLine(int(endLine))
		if hasCols {
			region.WithStartColumn(int(startCol)).WithEndColumn(int(endCol))
		}
	}
	if ctx.Content != "" {
		if region == nil {
			region = sarif.NewRegion()
		}
		region.WithSnippet(sarif.NewArtifactContent().WithText(ctx.Content))
	}
	if region != nil {
		pl.WithRegion(region)
	}
	return pl
}

// sarifLocationFingerprint produces a stable fingerprint for a (rule, location)
// pair so code-scanning consumers can dedup the same finding across runs.
func sarifLocationFingerprint(ruleID string, ctx llx.SourceContext) string {
	h := sha256.Sum256([]byte(ruleID + "\n" + ctx.Path + "#" + ctx.Range.String()))
	return hex.EncodeToString(h[:])
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
