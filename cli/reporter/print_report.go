// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"bytes"
	"fmt"
	io "io"
	"sort"
	"strings"

	"github.com/muesli/termenv"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/cli/printer"
	"go.mondoo.com/cnquery/cli/theme/colors"
	"go.mondoo.com/cnquery/explorer"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/utils/stringx"
	"go.mondoo.com/cnspec/cli/components"
	"go.mondoo.com/cnspec/policy"
)

const (
	assetOverviewPolicyMrn = "//policy.api.mondoo.app/policies/asset-overview"
	advisoryPolicyMrn      = "//policy.api.mondoo.app/policies/platform-vulnerability"
)

type policyRenderer func(print *printer.Printer, policyObj *policy.Policy, report *policy.Report, bundle *policy.PolicyBundleMap, resolvedPolicy *policy.ResolvedPolicy, data []reportRow) string

type reportRenderer struct {
	printer *printer.Printer
	out     io.Writer
	data    *policy.ReportCollection
}

func (r *reportRenderer) print() error {
	// TODO: sort assets by reverse score
	var res bytes.Buffer
	var scanSummary string

	// print summaryPrinter
	s := NewSummaryRenderer(r.printer)
	if len(r.data.Reports) > 0 {
		scanSummary = s.Render(r.data)
		res.WriteString(scanSummary)
	}

	// render asset name/summaryPrinter + policy results
	for assetMrn := range r.data.Assets {
		renderedAsset, err := r.printAsset(assetMrn)
		if err != nil {
			log.Error().Err(err).Send()
		} else {
			res.WriteString(renderedAsset)
		}
		res.WriteString(NewLineCharacter)
	}

	// print errors
	if len(r.data.Errors) > 0 {
		res.WriteString(r.printer.Primary("Scan Failures" + NewLineCharacter + NewLineCharacter))

		for name, errMsg := range r.data.Errors {
			// TODO: most likely we need to fetch the asset name here
			assetLine := termenv.String(fmt.Sprintf("■ Asset: %s%s", name, NewLineCharacter)).
				Foreground(colors.DefaultColorTheme.Critical).String()
			res.WriteString(assetLine)
			errLine := termenv.String(stringx.Indent(2, fmt.Sprintf("Error: %s%s", errMsg, NewLineCharacter))).
				Foreground(colors.DefaultColorTheme.Critical).String()
			errLine = strings.ReplaceAll(errLine, "\n", NewLineCharacter)
			res.WriteString(errLine)
		}
	}

	fmt.Fprintln(r.out, res.String())
	return nil
}

func (r *reportRenderer) printAsset(assetMrn string) (string, error) {
	assetObj := r.data.Assets[assetMrn]
	report := r.data.Reports[assetMrn]
	resolvedPolicy := r.data.ResolvedPolicies[assetMrn]

	if report == nil {
		return `✕ Could not find report for asset ` + assetObj.Mrn, nil
	}

	var res bytes.Buffer
	assetScore := report.Scores[assetObj.Mrn]
	overview := r.assetSummary(assetObj, assetScore)
	res.WriteString(overview)

	// render scanReport
	bundle := r.data.Bundle.ToMap()
	policies, err := resolvedPolicy.RootBundlePolicies(bundle, assetMrn)
	if err != nil {
		return "", err
	}

	for i := range policies {
		reportOutput, err := r.renderPolicyReport(policies[i], report, bundle, resolvedPolicy, nil)
		if err != nil {
			return "", err
		}
		res.WriteString(reportOutput)
	}

	return res.String(), nil
}

func (r *reportRenderer) assetSummary(assetObj *inventory.Asset, score *policy.Score) string {
	var res bytes.Buffer

	if assetObj == nil {
		return res.String()
	}

	res.WriteString(NewLineCharacter)

	// header with asset name
	res.WriteString(r.printer.H1(assetObj.Name))
	res.WriteString(components.NewScoreCard().Render(score))

	res.WriteString(NewLineCharacter)
	return res.String()
}

// policyModActions tracks the remove and modify actions to pass them to childs for proper rendering of ignores
type policyModActions map[string]explorer.Action

func (p policyModActions) Clone() policyModActions {
	res := policyModActions{}
	for k := range p {
		res[k] = p[k]
	}
	return res
}

func (r *reportRenderer) renderPolicyReport(policyObj *policy.Policy, report *policy.Report, bundle *policy.PolicyBundleMap, resolved *policy.ResolvedPolicy, parentQueryActions policyModActions) (string, error) {
	var res bytes.Buffer
	var queryActionsForChildren policyModActions

	log.Trace().Str("mrn", policyObj.Mrn).Msgf("match policy assetfilter: %v, pfilter: %v", resolved.Filters, policyObj.ComputedFilters)
	filters, err := policy.MatchingAssetFilters(policyObj.Mrn, resolved.Filters, policyObj)
	if err != nil {
		return r.printer.Error(err.Error), nil
	}

	// print policy details
	// NOTE: asset policies and space policies have no filter set but we want to render them too
	if len(filters) > 0 || policyObj.ComputedFilters == nil || len(policyObj.ComputedFilters.Items) == 0 {

		// determine renderer for policy
		var render policyRenderer
		switch policyObj.Mrn {
		case assetOverviewPolicyMrn:
			render = renderAssetOverview
		case advisoryPolicyMrn:
			render = renderAdvisoryPolicy
		default:
			render = renderPolicy
		}

		// use meta renderer for asset and space policies
		if strings.HasPrefix(policyObj.Mrn, "//assets.api.mondoo.app") || strings.HasPrefix(policyObj.Mrn, "//captain.api.mondoo.app") {
			render = renderMetaPolicy
		}

		var scoringData []reportRow
		scoringData, queryActionsForChildren = r.generateScoringResults(policyObj, report, bundle, resolved, parentQueryActions)
		result := render(r.printer, policyObj, report, bundle, resolved, scoringData)
		res.WriteString(result)
	}

	policies := r.policyReportChildren(policyObj, bundle)

	sort.Slice(policies, func(i, j int) bool {
		return policies[i].Name < policies[j].Name
	})

	// render sub-policies
	for i := range policies {
		// NOTE: do not pass the filtered asset filter, eg. a space policy may not include a filter but its child's
		// if we pass-through the filtered queries, child's policies matching the original asset filter are not rendered
		x, err := r.renderPolicyReport(policies[i], report, bundle, resolved, queryActionsForChildren.Clone())
		if err != nil {
			return "", err
		}
		res.WriteString(x)
	}

	return res.String(), nil
}

func (r *reportRenderer) policyReportChildren(policyObj *policy.Policy, bundle *policy.PolicyBundleMap) []*policy.Policy {
	policies := map[string]struct{}{}
	for i := range policyObj.Groups {
		group := policyObj.Groups[i]
		for i := range group.Policies {
			policies[group.Policies[i].Mrn] = struct{}{}
		}
	}

	policyRefs := []*policy.Policy{}
	if len(policies) > 0 {
		for id := range policies {
			curPolicy := bundle.Policies[id]
			if curPolicy == nil {
				continue
			}
			policyRefs = append(policyRefs, curPolicy)
		}
	}
	return policyRefs
}

func (r *reportRenderer) generateScoringResults(policyObj *policy.Policy, report *policy.Report, bundle *policy.PolicyBundleMap, resolved *policy.ResolvedPolicy, parentQueryActions policyModActions) ([]reportRow, policyModActions) {
	checks := map[string]*explorer.Mquery{}
	for i := range policyObj.Groups {
		group := policyObj.Groups[i]
		for i := range group.Checks {
			check := group.Checks[i]
			checks[check.Mrn] = check
		}
	}

	data := []reportRow{}

	// we are passing all actions that are not handled here through to the childs
	// - Add is handled directly within this policy, no need to pass through
	// - Remove is passed-through to the first child that adds or readds a query
	// - Modify is not-handled yet but would work similar to Remove
	actionsForChilds := policyModActions{}

	// extract queries for scores that are in the bundle
	for qid, check := range checks {
		action := check.Action

		// we only render query additions, all others need to be passed-through to the child policy
		// NOTE: we need to copy the map when we pass eg. Remove to Children, since multiple children can add the same query
		// FIXME: DEPRECATED, remove in v9.0 vv
		// Remove Action_UNSPECIFIED in v9.0
		if action != explorer.Action_ACTIVATE && action != explorer.Action_UNSPECIFIED {
			// ^^
			actionsForChilds[qid] = check.Action
			continue
		}

		// overwrite action if we got a modify from parent
		modifiedAction, isActionModified := parentQueryActions[qid]
		if isActionModified {
			action = modifiedAction
			// modifies parent actions so
			delete(parentQueryActions, qid)
		}

		query := bundle.Queries[qid]
		if query == nil {
			log.Warn().Msg("failed to find query '" + qid + "' in bundle")
			continue
		}

		codeBundle := resolved.GetCodeBundle(query)
		if codeBundle == nil {
			log.Debug().Msg("failed to find code bundle for query '" + qid + "' in bundle")
			continue
		}

		var score *policy.Score
		var assessment *llx.Assessment
		if report != nil {
			score = report.Scores[query.CodeId]
			assessment = policy.Query2Assessment(codeBundle, report)
		}

		data = append(data, reportRow{
			Query:      query,
			Bundle:     codeBundle,
			Score:      score,
			Action:     action,
			Impact:     check.Impact,
			Assessment: assessment,
		})
	}

	// merge unhandled actions from parent with new actions for childs
	for k := range parentQueryActions {
		actionsForChilds[k] = parentQueryActions[k]
	}

	// sort by severity and title
	sort.Sort(rowByScoreAndAction(data))

	return data, actionsForChilds
}

type reportRow struct {
	Query      *explorer.Mquery
	Bundle     *llx.CodeBundle
	Score      *policy.Score
	Action     explorer.Action
	Impact     *explorer.Impact
	Assessment *llx.Assessment
}

func (row reportRow) Indicator() string {
	char := '■'
	color := components.DefaultRatingColors.Color(scoreRating(row.Score))

	if row.Score != nil {

		if row.Score.Type == policy.ScoreType_Error {
			color = colors.DefaultColorTheme.Error
			char = '×'
		}

		if row.Action == explorer.Action_DEACTIVATE {
			color = colors.DefaultColorTheme.Disabled
			char = '×'
		}

		if row.Action == explorer.Action_MODIFY && row.Impact != nil && row.Impact.Weight == 0 {
			color = colors.DefaultColorTheme.Secondary
			char = '»'
		}

		if row.Score.Type == policy.ScoreType_Skip {
			color = colors.DefaultColorTheme.Disabled
			char = '»'
		}
	}

	return termenv.String(string(char)).Foreground(color).String()
}

func colorizeRow(row reportRow, text string) string {
	explain := ""
	severity := scoreRating(row.Score)
	color := components.DefaultRatingColors.Color(severity)

	if severity == policy.ScoreRating_aPlus {
		explain = "(passed)"
	} else if severity == policy.ScoreRating_unrated {
		explain = ""
	} else {
		explain = "(failed)"
	}

	if row.Score != nil {
		if row.Score.Type == policy.ScoreType_Error {
			color = colors.DefaultColorTheme.Error
			explain = "(error)"
		}

		if row.Score.Type == policy.ScoreType_Unknown {
			color = colors.DefaultColorTheme.Unknown
			explain = "(unknown)"
		}

		if row.Score.Type == policy.ScoreType_Unscored {
			color = colors.DefaultColorTheme.Unknown
			explain = "(unscored)"
		}

		if row.Action == explorer.Action_DEACTIVATE {
			color = colors.DefaultColorTheme.Disabled
			explain = "(removed)"
		}

		if row.Action == explorer.Action_MODIFY && row.Impact != nil && row.Impact.Weight == 0 {
			color = colors.DefaultColorTheme.Low
			explain = "(modified)"
		}

		if row.Score.Type == policy.ScoreType_Skip {
			color = colors.DefaultColorTheme.Disabled
			explain = "(skipped)"
		}
	} else {
		explain = "(unscored)"
	}

	return termenv.String(text + " " + explain).Foreground(color).String()
}

type rowByScoreAndAction []reportRow

func (data rowByScoreAndAction) Len() int { return len(data) }

var sortedScores = map[uint32]uint32{
	policy.ScoreType_Error:   0,
	policy.ScoreType_Result:  1,
	policy.ScoreType_Skip:    2,
	policy.ScoreType_Unknown: 3,
}

// sort by severity and title
func (data rowByScoreAndAction) Less(i, j int) bool {
	if data[i].Action != data[j].Action {
		if data[i].Action == explorer.Action_DEACTIVATE {
			return false
		}
		if data[j].Action == explorer.Action_DEACTIVATE {
			return true
		}
	}

	if data[i].Score != nil && data[j].Score == nil {
		return true
	}
	if data[i].Score != nil && data[j].Score != nil {
		// we want to sort by score type error, result, skip, unknown
		itype := data[i].Score.Type
		jtype := data[j].Score.Type
		if itype != jtype {
			return sortedScores[itype] < sortedScores[jtype]
		}

		if data[i].Score.Value == data[j].Score.Value {
			return data[i].Query.Title < data[j].Query.Title
		}
		return data[i].Score.Value < data[j].Score.Value
	}
	if data[i].Score == nil && data[j].Score == nil {
		return data[i].Query.Title < data[j].Query.Title
	}
	return false
}

func (data rowByScoreAndAction) Swap(i, j int) {
	data[i], data[j] = data[j], data[i]
}

func scoreRating(score *policy.Score) policy.ScoreRating {
	severity := policy.ScoreRating_unrated
	if score != nil {
		if score.Type == policy.ScoreType_Result {
			if score.Value == 100 {
				severity = policy.ScoreRating_aPlus
			} else {
				severity = policy.ScoreRating_failed
			}
		}

		if score.Type == policy.ScoreType_Error {
			severity = policy.ScoreRating_failed
		}
	}
	return severity
}
