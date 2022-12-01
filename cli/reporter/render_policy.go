package reporter

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/muesli/termenv"
	"go.mondoo.com/cnquery/cli/printer"
	"go.mondoo.com/cnquery/cli/theme/colors"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/stringx"
	"go.mondoo.com/cnspec/cli/components"
	"go.mondoo.com/cnspec/policy"
)

func renderPolicy(print *printer.Printer, policyObj *policy.Policy, report *policy.Report, bundle *policy.PolicyBundleMap, resolvedPolicy *policy.ResolvedPolicy, scoringData []reportRow) string {
	var res bytes.Buffer

	res.WriteString(print.H2(policyObj.Name))

	// render mini score card
	score := report.Scores[policyObj.Mrn]
	humanScore := "N/A"
	if score != nil {
		humanScore = fmt.Sprintf("%d (completion: %d%%, via %s score)", score.Value, score.Completion(), strings.ToLower(policyObj.ScoringSystem.String()))
	}

	box1 := components.NewMiniScoreCard().Render(score)
	box2 := NewLineCharacter + stringx.Indent(2, print.Primary("Policy:  ")+policyObj.Name+NewLineCharacter+print.Primary("Version: ")+policyObj.Version+NewLineCharacter+print.Primary("Mrn:     ")+policyObj.Mrn+NewLineCharacter+print.Primary("Score:   ")+humanScore)
	res.WriteString(stringx.MergeSideBySide(
		box1,
		box2,
	))

	// print scoring queries
	renderScoringQueries(print, policyObj, report, bundle, resolvedPolicy, scoringData, &res)
	res.WriteString(NewLineCharacter)

	// print data queries
	renderDataQueries(print, policyObj, report, bundle, resolvedPolicy, &res)

	return res.String()
}

func renderDataQueries(print *printer.Printer, policyObj *policy.Policy, report *policy.Report, bundleMap *policy.PolicyBundleMap, resolvedPolicy *policy.ResolvedPolicy, res *bytes.Buffer) {
	dataQueries := map[string]policy.QueryAction{}
	for i := range policyObj.Specs {
		spec := policyObj.Specs[i]
		for qid, queryAction := range spec.DataQueries {
			dataQueries[qid] = queryAction
		}
	}

	// TODO: this highlights an internal issue when Results are converted to RawResults
	results := report.RawResults()
	if len(dataQueries) > 0 {
		res.WriteString(print.Primary("Data Queries:" + NewLineCharacter + NewLineCharacter))
	}

	// iterate over queries and render them properly
	// TODO: we need to take care of the query action
	for qid := range dataQueries {
		query := bundleMap.Queries[qid]
		if query == nil {
			res.WriteString(NewLineCharacter + print.Error("failed to find query '"+qid+"' in bundle"))
			continue
		}

		codeBundle := resolvedPolicy.GetCodeBundle(query)
		if codeBundle == nil {
			res.WriteString(NewLineCharacter + print.Error("failed to find code bundle for query '"+qid+"' in bundle"))
			continue
		}

		res.WriteString("■ Title: ")
		res.WriteString(query.Title)
		res.WriteString(NewLineCharacter)
		res.WriteString("  ID:    ")
		res.WriteString(query.Mrn)
		res.WriteString(NewLineCharacter)
		writeQueryCompact(res, "  Query: ", print.Disabled(query.Query))

		// print data results
		// copy all contents where we have labels
		filteredResults := map[string]*llx.RawResult{}
		for i := range codeBundle.CodeV2.Checksums {
			checksum := codeBundle.CodeV2.Checksums[i]

			res, ok := results[checksum]
			if ok {
				filteredResults[checksum] = res
			}
		}

		result := print.Results(codeBundle, filteredResults)
		writeQueryCompact(res, "  Result:", result)

		res.WriteString(NewLineCharacter)
	}
}

// if we have a multi-line query, place the query in newline
func writeQueryCompact(res *bytes.Buffer, title string, value string) {
	res.WriteString(title)
	if strings.Contains(value, NewLineCharacter) {
		res.WriteString(NewLineCharacter)
		res.WriteString(stringx.Indent(4, value))
	} else {
		res.WriteString(value)
		res.WriteString(NewLineCharacter)
	}
}

func renderScoringQueries(print *printer.Printer, policyObj *policy.Policy, report *policy.Report, bundle *policy.PolicyBundleMap, resolvedPolicy *policy.ResolvedPolicy, data []reportRow, res *bytes.Buffer) {
	// return if we do not have any queries to render for this policy
	if len(data) == 0 {
		res.WriteString(print.Disabled(NewLineCharacter + "no scored queries" + NewLineCharacter))
		return
	}

	res.WriteString(print.Primary("Scoring Queries:" + NewLineCharacter))

	// print summary
	renderPolicySummary(res, data)

	// print report results
	renderPolicyReportTable(print, report, bundle, resolvedPolicy, res, data)
}

func renderPolicySummary(res *bytes.Buffer, data []reportRow) {
	ps := policy.Stats{
		Passed: &policy.ScoreDistribution{},
		Failed: &policy.ScoreDistribution{},
		Errors: &policy.ScoreDistribution{},
	}

	for i := range data {
		row := data[i]

		ps.Total++
		if row.Action == policy.QueryAction_DEACTIVATE {
			ps.Skipped++
		} else if row.Action == policy.QueryAction_MODIFY && row.ActionSpec != nil && row.ActionSpec.Weight == 0 {
			ps.Skipped++
		} else if row.Score != nil && row.Score.Type == policy.ScoreType_Error {
			ps.Errors.Total++
		} else if row.Score != nil && row.Score.Type == policy.ScoreType_Skip {
			ps.Skipped++
		} else if scoreRating(row.Score) == policy.ScoreRating_failed {
			ps.Failed.Total++
		} else if scoreRating(row.Score) == policy.ScoreRating_aPlus {
			ps.Passed.Total++
		} else {
			ps.Unknown++
		}
	}

	colorMap := []termenv.Color{
		colors.DefaultColorTheme.Good,
		colors.DefaultColorTheme.High,
		colors.DefaultColorTheme.Critical,
		colors.DefaultColorTheme.Unknown,
		colors.DefaultColorTheme.Unknown,
	}
	labels := []string{"Passed", "Failed", "Errors", "Ignored", "Unknown"}
	datapoints := []float64{
		(float64(ps.Passed.Total) / float64(ps.Total)),
		(float64(ps.Failed.Total) / float64(ps.Total)),
		(float64(ps.Errors.Total) / float64(ps.Total)),
		(float64(ps.Skipped) / float64(ps.Total)),
		(float64(ps.Unknown) / float64(ps.Total)),
	}

	// render bar chart
	barChart := components.NewBarChart(components.WithBarChartBorder(), components.WithBarChartLabelFunc(components.BarChartPercentageLabelFunc))
	res.WriteString(barChart.Render(datapoints, colorMap, labels))
}

func renderPolicyReportTable(print *printer.Printer, report *policy.Report, bundle *policy.PolicyBundleMap, resolvedPolicy *policy.ResolvedPolicy, res *bytes.Buffer, data []reportRow) {
	results := report.RawResults()
	res.WriteString(NewLineCharacter)
	for i := range data {
		row := data[i]

		action := ""
		if row.Action == policy.QueryAction_DEACTIVATE {
			action = "// removed by user"
		} else if row.Action == policy.QueryAction_MODIFY && (row.ActionSpec == nil || row.ActionSpec.Weight == 0) {
			action = "// ignored by user"
		} else if row.Score != nil && row.Score.Type == policy.ScoreType_Skip {
			action = "// skipped by query condition"
		}

		res.WriteString(row.Indicator())
		res.WriteString(" Title:  ")
		res.WriteString(colorizeRow(row, row.Query.Title))
		res.WriteString(" " + action)
		res.WriteString(NewLineCharacter)

		// passed scores do not need to print details
		if row.Score != nil && row.Score.Value == 100 {
			continue
		}

		// do not display the result if the score was not completed
		if row.Score == nil || row.Score.ScoreCompletion == 0 {
			continue
		}

		writeQueryCompact(res, "  Query:  ", print.Disabled(row.Query.Query))

		// print error if we got one
		errorMsg := row.Score.MessageLine()
		if errorMsg != "" {
			res.WriteString("  Error: ")
			// Print out original errorMsg so we get nice newline formatting
			res.WriteString(print.Failed(row.Score.Message))
			res.WriteString(NewLineCharacter)
		} else if row.Score == nil || row.Score.Value != 100 {
			// print more details if the test failed
			if row.Query != nil {
				if row.Assessment != nil {
					res.WriteString("  Assessment:" + NewLineCharacter)
					res.WriteString(stringx.Indent(2, print.Assessment(row.Bundle, row.Assessment)))
				} else {
					// If we don't have an assessment, print as result so we can display something
					// useful
					qid := row.Query.Mrn
					query := bundle.Queries[qid]
					if query == nil {
						res.WriteString(NewLineCharacter + print.Error("failed to find query '"+qid+"' in bundle"))
						continue
					}

					codeBundle := resolvedPolicy.GetCodeBundle(query)
					if codeBundle == nil {
						res.WriteString(NewLineCharacter + print.Error("failed to find code bundle for query '"+qid+"' in bundle"))
						continue
					}

					filteredResults := codeBundle.FilterResults(results)
					result := print.Results(codeBundle, filteredResults)
					writeQueryCompact(res, "  Result: ", result)

				}
			}
		}
		res.WriteString(NewLineCharacter)

	}
}
