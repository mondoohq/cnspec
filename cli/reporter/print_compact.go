// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/muesli/termenv"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/explorer"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/upstream/mvd"
	"go.mondoo.com/cnquery/v11/utils/stringx"
	cnspecComponents "go.mondoo.com/cnspec/v11/cli/components"
	"go.mondoo.com/cnspec/v11/policy"
)

type assetMrnName struct {
	Mrn  string
	Name string
}

type defaultReporter struct {
	*Reporter
	output io.Writer
	data   *policy.ReportCollection

	// indicates if the StoreResourcesData cnquery feature is enabled
	isStoreResourcesEnabled bool

	// vv the items below will be automatically filled
	bundle *policy.PolicyBundleMap
}

func (r *defaultReporter) out(s string) {
	_, _ = r.output.Write([]byte(s))
}

func (r *defaultReporter) print() error {
	// catch case where the scan was not successful and no bundle was fetched from server
	if r.data == nil {
		log.Debug().Msg("report does not contain any data")
		return nil
	}

	// this case can happen when we have only assets with errors, eg. where no valid policy was found
	if r.data.Bundle != nil {
		r.bundle = r.data.Bundle.ToMap()
	}

	// sort assets by name, to make it more intuitive
	i := 0
	orderedAssets := make([]assetMrnName, len(r.data.Assets))
	for assetMrn, asset := range r.data.Assets {
		orderedAssets[i] = assetMrnName{
			Mrn:  assetMrn,
			Name: asset.Name,
		}
		i++
	}
	sort.Slice(orderedAssets, func(i, j int) bool {
		return orderedAssets[i].Name < orderedAssets[j].Name
	})

	if r.Conf.printContents() {
		r.printAssetSections(orderedAssets)
	}

	r.printSummary(orderedAssets)
	return nil
}

func (r *defaultReporter) printSummary(orderedAssets []assetMrnName) {
	var (
		assetUrl         = ""
		projectId        = ""
		assetsByPlatform = make(map[string][]*inventory.Asset)
		assetsByScore    = make(map[string]int)
	)
	for _, assetMrnName := range orderedAssets {
		assetMrn := assetMrnName.Mrn
		asset := r.data.Assets[assetMrn]
		if asset.Url != "" {
			assetUrl = asset.Url
		}
		if val, ok := asset.Labels["mondoo.com/project-id"]; ok {
			projectId = val
		}
		platformName := getPlatformNameForAsset(asset)
		if platformName != "" {
			assetsByPlatform[platformName] = append(assetsByPlatform[platformName], asset)
		}
		if _, ok := r.data.Reports[assetMrn]; ok {
			assetScore := r.data.Reports[assetMrn].Score.Rating().Text()
			assetsByScore[assetScore]++
		}
	}

	if len(r.data.Errors) > 0 {
		assetsByScore[policy.ScoreRatingTextError] += len(r.data.Errors)
	}

	if len(assetsByScore) > 0 {
		assetsString := "asset"
		if len(r.data.Assets) > 1 {
			assetsString = "assets"
		}
		header := fmt.Sprintf("Scanned %d %s", len(r.data.Assets), assetsString)
		r.out(termenv.String(header + NewLineCharacter).Foreground(r.Colors.Primary).String())
	}

	// print assets by platform
	r.printAssetsByPlatform(assetsByPlatform)

	// print distributions
	if len(orderedAssets) > 1 {
		r.printSummaryHeader()
		r.out(cnspecComponents.NewDistributions(assetsByScore, assetsByPlatform).View())
		r.out(NewLineCharacter)
	}

	if r.Conf.isCompact {
		r.out(NewLineCharacter)
		if !r.IsIncognito && assetUrl != "" {
			url := ""
			if len(orderedAssets) > 1 {
				// we do not have a space url, so we extract it form the asset url
				// https://console.mondoo.com/space/inventory/2JtqGyVTZULTW0uwQ5YxXW4nh6Y?spaceId=dazzling-golick-767384
				// an individual asset url wouldn't make sense here
				// when runnin inside cicd, we create an url for the cicd project
				spaceUrlRegexp := regexp.MustCompile(`^(http.*)/inventory/[a-zA-Z0-9-]+(\?.+)$`)
				m := spaceUrlRegexp.FindStringSubmatch(assetUrl)
				if len(m) > 0 {
					if projectId != "" {
						url = m[1] + "/cicd/jobs" + m[2] + "&projectId=" + projectId
					} else {
						url = m[1] + "/inventory" + m[2]
					}
				}

			} else {
				url = assetUrl
			}

			r.out("See more scan results and asset relationships on the Mondoo Console: ")
			r.out(url + NewLineCharacter)

			if len(orderedAssets) == 1 && orderedAssets[0].Mrn != "" && r.isStoreResourcesEnabled {
				r.out("Asset MRN: " + orderedAssets[0].Mrn + NewLineCharacter)
			}
		}
	}
}

func (r *defaultReporter) printSummaryHeader() {
	summaryHeader := "Summary"
	summaryDivider := strings.Repeat("=", utf8.RuneCountInString(summaryHeader))
	r.out(NewLineCharacter)
	r.out(termenv.
		String(summaryHeader + NewLineCharacter + summaryDivider + NewLineCharacter).
		Foreground(r.Colors.Primary).
		String())
	r.out(NewLineCharacter)
}

func (r *defaultReporter) printAssetsByPlatform(assetsByPlatform map[string][]*inventory.Asset) {
	availablePlatforms := make([]string, 0, len(assetsByPlatform))
	for k := range assetsByPlatform {
		availablePlatforms = append(availablePlatforms, k)
	}
	sort.Strings(availablePlatforms)

	for _, platform := range availablePlatforms {
		r.out(NewLineCharacter + platform + NewLineCharacter)
		for i := range assetsByPlatform[platform] {
			assetScore := ""
			assetScoreRating := policy.ScoreRating_unrated
			if r.data.Reports[assetsByPlatform[platform][i].Mrn] != nil {
				score := r.data.Reports[assetsByPlatform[platform][i].Mrn].Score
				assetScoreRating = score.Rating()
				assetScore = "[" + strconv.Itoa(int(score.Value)) + "/100]"
			} else {
				assetScoreRating = policy.ScoreRating_error
				assetScore = string(policy.ScoreRatingTextError)
			}

			paddedAssetScore := fmt.Sprintf("%-9s", assetScore)
			scoreColor := cnspecComponents.DefaultRatingColors.Color(assetScoreRating)
			output := fmt.Sprintf("    %s   %s", termenv.String(paddedAssetScore).Foreground(scoreColor), assetsByPlatform[platform][i].Name)
			r.out(output + NewLineCharacter)
		}
	}
}

func (r *defaultReporter) printAssetSections(orderedAssets []assetMrnName) {
	if len(orderedAssets) == 0 {
		return
	}

	var queries map[string]*explorer.Mquery
	var controls map[string]*policy.Control
	if r.bundle != nil {
		queries = r.bundle.QueryMap()
		controls = r.bundle.ControlsMap()
	}

	for _, assetMrnName := range orderedAssets {
		assetMrn := assetMrnName.Mrn
		asset := r.data.Assets[assetMrn]
		target := asset.Name
		if target == "" {
			target = assetMrn
		}

		r.out(r.Printer.H2("Asset: (" + getPlatformNameForAsset(asset) + ") " + target))

		errorMsg, ok := r.data.Errors[assetMrn]
		if ok {
			r.out(r.Printer.Error(errorMsg))
			r.out(NewLineCharacter + NewLineCharacter)
			continue
		}

		report, ok := r.data.Reports[assetMrn]
		if !ok {
			// nothing to do, we get an error message in the summary code
			continue
		}
		if report == nil {
			// the asset didn't match any policy, so no report was generated
			continue
		}

		resolved, ok := r.data.ResolvedPolicies[assetMrn]
		if !ok {
			// nothing to do, we get an additional error message in the summary code
			continue
		}

		if r.Conf.printControls {
			r.printAssetControls(resolved, report, controls, assetMrn, asset)
		}

		if r.Conf.printData || r.Conf.printChecks {
			r.printAssetQueries(resolved, report, queries, assetMrn, asset)
		}

		if r.Conf.printRisks {
			r.printAssetRisks(resolved, report, assetMrn, asset)
		}
		r.out(NewLineCharacter)

		if r.Conf.printVulnerabilities {
			// TODO: we should re-use the report results
			r.printVulns(report, assetMrn)
		}

	}
	r.out(NewLineCharacter)
}

// TODO: this should be done during the execution, as queries come in, not at the end!
// Remove all this code and migrate it to tap or something
// ============================= vv ============================================

func (r *defaultReporter) printAssetControls(resolved *policy.ResolvedPolicy, report *policy.Report, controls map[string]*policy.Control, assetMrn string, asset *inventory.Asset) {
	var scores []*policy.Score
	for _, rj := range resolved.CollectorJob.ReportingJobs {
		if rj.Type != policy.ReportingJob_CONTROL {
			continue
		}

		score, ok := report.Scores[rj.QrId]
		if !ok {
			log.Warn().Str("control", rj.QrId).Msg("missing score for control")
		}

		scores = append(scores, score)
	}

	if len(scores) == 0 {
		return
	}

	sort.Slice(scores, func(i, j int) bool {
		return scores[i].QrId < scores[j].QrId
	})

	r.out("Compliance controls:" + NewLineCharacter)

	for i := range scores {
		score := scores[i]
		control, ok := controls[score.QrId]
		if !ok {
			r.out("Couldn't find any controls for " + score.QrId)
			r.out(NewLineCharacter)
			continue
		}

		r.printControl(score, control, resolved, report)
	}

	r.out(NewLineCharacter)
}

func (r *defaultReporter) printControl(score *policy.Score, control *policy.Control, resolved *policy.ResolvedPolicy, report *policy.Report) {
	title := control.Title
	if title == "" {
		title = control.Mrn
	}

	switch score.Type {
	case policy.ScoreType_Error:
		r.out(termenv.String("! Error:      ").Foreground(r.Colors.Error).String())
		r.out(title)
		r.out(NewLineCharacter)
		if !r.Conf.isCompact {
			errorMessage := strings.ReplaceAll(score.Message, "\n", NewLineCharacter)
			r.out(termenv.String("  Message:    " + errorMessage).Foreground(r.Colors.Error).String())
			r.out(NewLineCharacter)
		}
	case policy.ScoreType_Unknown, policy.ScoreType_Unscored:
		r.out(termenv.String(". Unknown:    ").Foreground(r.Colors.Disabled).String())
		r.out(title)
		r.out(NewLineCharacter)

	case policy.ScoreType_Skip:
		r.out(termenv.String(". Skipped:    ").Foreground(r.Colors.Disabled).String())
		r.out(title)
		r.out(NewLineCharacter)

	case policy.ScoreType_Result:
		var passfail string
		if score.Value == 100 {
			passfail = termenv.String("✓ Pass:  ").Foreground(r.Colors.Success).String()
		} else {
			passfail = termenv.String("✕ Fail:  ").Foreground(r.Colors.High).String()
		}

		r.out(passfail + title + NewLineCharacter)

	default:
		r.out("unknown result for " + title + NewLineCharacter)
	}
}

// FIXME v12: This is a temporary workaround to deal with the fact that scores don't carry information about success or failure
type simpleScore struct {
	Value   uint32
	Type    uint32
	Message string
	Success bool
	Rating  policy.ScoreRating
}

func (r *defaultReporter) printAssetQueries(resolved *policy.ResolvedPolicy, report *policy.Report, queries map[string]*explorer.Mquery, assetMrn string, asset *inventory.Asset) {
	results := report.RawResults()

	if r.Conf.printData {
		dataQueriesOutput := ""
		resolved.WithDataQueries(func(id string, query *policy.ExecutionQuery) {
			data := query.Code.FilterResults(results)
			result := r.Reporter.Printer.Datas(query.Code, data)
			if result == "" {
				return
			}
			if r.Conf.isCompact {
				result = stringx.MaxLines(10, result)
			}
			dataQueriesOutput += result + NewLineCharacter
		})

		if len(dataQueriesOutput) > 0 {
			r.out("Data queries:" + NewLineCharacter)
			r.out(dataQueriesOutput)
			r.out(NewLineCharacter)
		}
	}

	if r.Conf.printChecks {
		foundChecks := map[string]simpleScore{}
		sortedPassed := []string{}
		sortedWarnings := []string{}
		sortedFailed := []string{}

		for id, score := range report.Scores {
			_, ok := resolved.CollectorJob.ReportingQueries[id]
			if !ok {
				continue
			}

			query, ok := queries[id]
			if !ok {
				r.out("Couldn't find any queries for score of " + id)
				continue
			}

			// FIXME v12: this is only a workaround for a deeper bug with the score value
			if query.Impact != nil && query.Impact.Value != nil {
				floor := 100 - uint32(query.Impact.Value.Value)
				if floor > score.Value {
					score.Value = floor
				}
			}

			score := simpleScore{
				Value:   score.Value,
				Type:    score.Type,
				Message: score.Message,
				Rating:  score.Rating(),
				// FIXME v12: this is incorrect because the score value is 100 for failing checks whose impact is 0
				Success: score.Value == 100,
			}
			foundChecks[id] = score

			if score.Success {
				sortedPassed = append(sortedPassed, id)
			} else if score.Value >= uint32(r.ScoreThreshold) {
				sortedWarnings = append(sortedWarnings, id)
			} else {
				sortedFailed = append(sortedFailed, id)
			}
		}

		if r.ScoreThreshold == 0 {
			sortedFailed = append(sortedFailed, sortedWarnings...)
			sortedWarnings = []string{}
		}

		sort.Slice(sortedPassed, func(i, j int) bool {
			return queries[sortedPassed[i]].Title < queries[sortedPassed[j]].Title
		})

		sort.Slice(sortedWarnings, func(i, j int) bool {
			ida := sortedWarnings[i]
			idb := sortedWarnings[j]
			a := foundChecks[ida].Value
			b := foundChecks[idb].Value
			if a == b {
				return queries[ida].Title < queries[idb].Title
			}
			return a > b
		})

		sort.Slice(sortedFailed, func(i, j int) bool {
			ida := sortedFailed[i]
			idb := sortedFailed[j]
			a := foundChecks[ida].Value
			b := foundChecks[idb].Value
			if a == b {
				return queries[ida].Title < queries[idb].Title
			}
			return a > b
		})

		prevPrinted := false
		if len(sortedPassed) != 0 {
			r.out("Passing:" + NewLineCharacter)
			for _, id := range sortedPassed {
				r.printCheck(foundChecks[id], queries[id], resolved, report, results)
			}
			prevPrinted = true
		}

		if len(sortedWarnings) != 0 {
			if prevPrinted {
				r.out(NewLineCharacter)
			}
			// FIXME v12: rename to risk threshold
			r.out("Warning - above score threshold:" + NewLineCharacter)
			for _, id := range sortedWarnings {
				r.printCheck(foundChecks[id], queries[id], resolved, report, results)
			}
			prevPrinted = true
		}

		if len(sortedFailed) != 0 {
			if prevPrinted {
				r.out(NewLineCharacter)
			}
			if r.ScoreThreshold > 0 {
				// FIXME v12: rename to risk threshold
				r.out("Failing - below score threshold:" + NewLineCharacter)
			} else {
				r.out("Failing:" + NewLineCharacter)
			}
			for _, id := range sortedFailed {
				r.printCheck(foundChecks[id], queries[id], resolved, report, results)
			}
		}

	}
}

func (r *defaultReporter) printAssetRisks(resolved *policy.ResolvedPolicy, report *policy.Report, assetMrn string, asset *inventory.Asset) {
	if report.Risks == nil || len(report.Risks.Items) == 0 {
		return
	}

	if len(r.bundle.RiskFactors) == 0 {
		log.Warn().Msg("found risk factors in report, but none are in the bundle for printing")
		return
	}

	allowedRiskMrns := map[string]bool{}
	for _, s := range report.Scores {
		for _, r := range s.RiskFactors.GetItems() {
			allowedRiskMrns[r.Mrn] = true
		}
	}

	// TODO: we need to get the risk factors that apply for vulnerabilities
	// This is currently not supported

	var res strings.Builder
	for i := range report.Risks.Items {
		risk := report.Risks.Items[i]
		if !risk.IsDetected {
			continue
		}
		if !allowedRiskMrns[risk.Mrn] {
			continue
		}

		var text string

		riskInfo, ok := r.bundle.RiskFactors[risk.Mrn]
		if !ok {
			text = risk.Mrn
		} else {
			text = riskInfo.Title
		}

		if risk.Risk > 0 {
			text = termenv.String("✕ " + text).Foreground(r.Colors.High).String()
		} else {
			text = termenv.String("✓ " + text).Foreground(r.Colors.Success).String()
		}

		res.WriteString(text + NewLineCharacter)
	}
	out := res.String()

	r.out(NewLineCharacter + "Risks / Preventive Controls:" + NewLineCharacter)
	if out != "" {
		r.out(out)
	} else {
		r.out(termenv.String("✓ no downgrading risks detected" + NewLineCharacter).Foreground(r.Colors.Disabled).String())
	}
}

// only works with type == policy.ScoreType_Result
func (r *defaultReporter) printScore(title string, score simpleScore, query *explorer.Mquery) string {
	color := cnspecComponents.DefaultRatingColors.Color(score.Rating)

	var passfail string
	if score.Success {
		passfail = termenv.String("✓ ").Foreground(r.Colors.Success).String()
	} else {
		scoreIndicator := ""
		if query.Impact != nil {
			scoreIndicator = " (" + strconv.Itoa(int(score.Value)) + ")"
		}
		scoreSymbol := "✕"
		if score.Value > uint32(r.ScoreThreshold) {
			scoreSymbol = "!"
		}
		passfail = termenv.String(fmt.Sprintf("%s %-17s", scoreSymbol, score.Rating.Text()+scoreIndicator+":")).Foreground(color).String()
	}

	return passfail + title + NewLineCharacter
}

func (r *defaultReporter) printCheck(score simpleScore, query *explorer.Mquery, resolved *policy.ResolvedPolicy, report *policy.Report, results map[string]*llx.RawResult) {
	title := query.Title
	if title == "" {
		title = query.Mrn
	}

	switch score.Type {
	case policy.ScoreType_Error:
		r.out(termenv.String("! Error:      ").Foreground(r.Colors.Error).String())
		r.out(title)
		r.out(NewLineCharacter)
		if !r.Conf.isCompact {
			errorMessage := strings.ReplaceAll(score.Message, "\n", NewLineCharacter)
			r.out(termenv.String("  Message:    " + errorMessage).Foreground(r.Colors.Error).String())
			r.out(NewLineCharacter)
		}
	case policy.ScoreType_Unknown, policy.ScoreType_Unscored:
		r.out(termenv.String(". Unknown:    ").Foreground(r.Colors.Disabled).String())
		r.out(title)
		r.out(NewLineCharacter)

	case policy.ScoreType_Skip:
		r.out(termenv.String(". Skipped:    ").Foreground(r.Colors.Disabled).String())
		r.out(title)
		r.out(NewLineCharacter)

	case policy.ScoreType_Result:
		r.out(r.printScore(title, score, query))

		// additional information about the failed query
		if !r.Conf.isCompact && score.Value != 100 {
			queryString := strings.ReplaceAll(stringx.Indent(4, query.Query), "\n", NewLineCharacter)
			r.out("  Query:" + NewLineCharacter + queryString)
			r.out(NewLineCharacter)

			codeBundle := resolved.GetCodeBundle(query)
			if codeBundle == nil {
				r.out(r.Reporter.Printer.Error("failed to find code bundle for query '" + query.Mrn + "' in bundle"))
			} else {
				r.out("  Result:" + NewLineCharacter)
				assessment := policy.Query2Assessment(codeBundle, report)
				if assessment != nil {
					assessmentString := stringx.Indent(4, r.Printer.Assessment(codeBundle, assessment))
					assessmentString = strings.ReplaceAll(assessmentString, "\n", NewLineCharacter)
					r.out(assessmentString)
				} else {
					data := codeBundle.FilterResults(results)
					result := stringx.Indent(4, r.Reporter.Printer.Results(codeBundle, data))
					result = strings.ReplaceAll(result, "\n", NewLineCharacter)
					r.out(result)
				}
			}
			r.out(NewLineCharacter)
		}
	default:
		r.out("unknown result for " + title + NewLineCharacter)
	}
}

// ============================= ^^ ============================================

func (r *defaultReporter) printVulns(report *policy.Report, assetMrn string) {
	print := r.Printer

	vulnReport := r.data.VulnReports[assetMrn]

	if vulnReport == nil {
		return
	}
	r.out(print.Primary("Vulnerabilities:" + NewLineCharacter))

	score := report.Scores[advisoryPolicyMrn]
	_ = score

	r.printVulnList(vulnReport)
	r.printVulnSummary(vulnReport)
}

func (r *defaultReporter) printVulnList(report *mvd.VulnReport) {
	if report.GetStats() == nil || report.Stats.Advisories.Total == 0 {
		color := cnspecComponents.DefaultRatingColors.Color(policy.ScoreRating_aPlus)
		indicatorChar := '■'
		title := "No advisories found"
		state := "(passed)"
		r.out(termenv.String(string(indicatorChar), title, state).Foreground(color).String())
		r.out(NewLineCharacter + NewLineCharacter)
		return
	}
	r.out(RenderVulnReport(report))
}

func (r *defaultReporter) printVulnSummary(report *mvd.VulnReport) {
	if report.GetStats() == nil {
		return
	}
	cvss := cnspecComponents.IntScore2Float(report.Stats.Score)

	// TODO: the CVSS score is not equal to the advisory policy score above.
	// So we need to grab it and translate it, to get to the right color
	vulnScore := &policy.Score{
		Value:           uint32(100 - report.Stats.Score),
		Type:            policy.ScoreType_Result,
		ScoreCompletion: 100,
		DataCompletion:  100,
	}

	r.out(r.scoreColored(vulnScore.Rating(), fmt.Sprintf("Overall CVSS score: %.1f%s%s", cvss, NewLineCharacter, NewLineCharacter)))
}
