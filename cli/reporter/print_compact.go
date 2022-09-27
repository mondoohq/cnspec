package reporter

import (
	"fmt"
	io "io"

	"github.com/mitchellh/mapstructure"
	"github.com/muesli/termenv"
	"go.mondoo.com/cnquery/cli/components"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/resources/packs/core/vadvisor"
	"go.mondoo.com/cnquery/stringx"
	cnspecComponents "go.mondoo.com/cnspec/cli/components"
	"go.mondoo.com/cnspec/policy"
)

type defaultReporter struct {
	*Reporter
	isCompact bool
	isSummary bool
	out       io.Writer
	data      *policy.ReportCollection

	// vv the items below will be automatically filled
	bundle *policy.PolicyBundleMap
}

func (r *defaultReporter) print() error {
	r.bundle = r.data.Bundle.ToMap()

	if !r.isSummary {
		r.printQueries()
	}

	r.printSummary()
	return nil
}

func (r *defaultReporter) printSummary() {
	for mrn, asset := range r.data.Assets {
		r.printAssetSummary(mrn, asset)
	}
}

func printCompactScoreSummary(score *policy.Score) string {
	return fmt.Sprintf("%s   %3d/100     (%d%% completed)",
		score.Rating().Letter(),
		score.Value, score.Completion())
}

func (r *defaultReporter) printAssetSummary(assetMrn string, asset *policy.Asset) {
	target := asset.Name
	if target == "" {
		target = assetMrn
	}

	report, ok := r.data.Reports[assetMrn]
	if !ok {
		// If scanning the asset has failed, there will be no report, we should first look if there's an error for that target.
		if err, ok := r.data.Errors[assetMrn]; ok {
			r.out.Write([]byte(termenv.String(fmt.Sprintf(
				`✕ Error for asset %s: %s`,
				target, err,
			)).Foreground(r.Colors.Error).String()))
		} else {
			r.out.Write([]byte(fmt.Sprintf(
				`✕ Could not find asset %s`,
				target,
			)))
		}
		return
	}

	resolved, ok := r.data.ResolvedPolicies[assetMrn]
	if !ok {
		r.out.Write([]byte(fmt.Sprintf(
			`✕ Could not find resolved policy for asset %s`,
			target,
		)))
		return
	}

	// TODO: we should re-use the report results
	r.printVulns(resolved, report, report.RawResults())

	score := printCompactScoreSummary(report.Score)
	report.ComputeStats(resolved)

	r.out.Write([]byte(termenv.String(`Summary
========================

`).Foreground(r.Colors.Secondary).String()))
	r.out.Write([]byte(termenv.String(fmt.Sprintf("Target:     %s\n", target)).Foreground(r.Colors.Primary).String()))

	if report.Stats == nil || report.Stats.Total == 0 {
		r.out.Write([]byte(fmt.Sprintf("Datapoints: %d\n", len(report.Data))))
	} else {
		passCnt := report.Stats.Passed.Total
		passPct := float32(passCnt) / float32(report.Stats.Total) * 100
		passProgress := components.Hbar(15, passPct)
		failCnt := report.Stats.Failed.Total
		failPct := float32(failCnt) / float32(report.Stats.Total) * 100
		failProgress := components.Hbar(15, failPct)
		errCnt := report.Stats.Errors.Total
		errPct := float32(errCnt) / float32(report.Stats.Total) * 100
		errProgress := components.Hbar(15, errPct)
		skipCnt := report.Stats.Skipped + report.Stats.Unknown
		skipPct := float32(skipCnt) / float32(report.Stats.Total) * 100
		skipProgress := components.Hbar(15, skipPct)

		r.out.Write([]byte(r.scoreColored(report.Score.Rating(), fmt.Sprintf("Score:      %s\n", score))))
		r.out.Write([]byte(termenv.String(fmt.Sprintf("✓ Passed:   %s%.0f%% (%d)\n", passProgress, passPct, passCnt)).Foreground(r.Colors.Success).String()))
		r.out.Write([]byte(termenv.String(fmt.Sprintf("✕ Failed:   %s%.0f%% (%d)\n", failProgress, failPct, failCnt)).Foreground(r.Colors.Critical).String()))
		r.out.Write([]byte(termenv.String(fmt.Sprintf("! Errors:   %s%.0f%% (%d)\n", errProgress, errPct, errCnt)).Foreground(r.Colors.Error).String()))
		r.out.Write([]byte(termenv.String(fmt.Sprintf("» Skipped:  %s%.0f%% (%d)\n", skipProgress, skipPct, skipCnt)).Foreground(r.Colors.Disabled).String()))

	}

	r.out.Write([]byte("\nPolicies:\n"))
	scores := policyScores(report, r.bundle)
	for i := range scores {
		x := scores[i]
		switch x.score.Type {
		case policy.ScoreType_Error:
			r.out.Write([]byte(termenv.String("E  EE  " + x.title).Foreground(r.Colors.Error).String()))
			r.out.Write([]byte{'\n'})
		case policy.ScoreType_Unknown, policy.ScoreType_Unscored, policy.ScoreType_Skip:
			r.out.Write([]byte(".  ..  " + x.title))
			r.out.Write([]byte{'\n'})
		case policy.ScoreType_Result:
			rating := x.score.Rating()
			line := fmt.Sprintf(
				"%s %3d  %s\n",
				rating.Letter(), x.score.Value, x.title,
			)
			r.out.Write([]byte(r.scoreColored(rating, line)))
		default:
			r.out.Write([]byte("?  ..  " + x.title))
			r.out.Write([]byte{'\n'})
		}
	}
	if len(scores) > 0 {
		r.out.Write([]byte{'\n'})
	}

	if !r.IsIncognito {
		panic("PROVIDE UPSTREAM URL")
	} else {
		if r.isCompact {
			r.out.Write([]byte("To get more information, please run this scan with \"-o full\".\n"))
		}
	}
}

// TODO: this should be done during the execution, as queries come in, not at the end!
// Remove all this code and migrate it to tap or something
// ============================= vv ============================================

func (r *defaultReporter) printQueries() {
	r.out.Write([]byte{'\n'})
	queries := r.bundle.QueryMap()
	for mrn, asset := range r.data.Assets {
		r.printAssetQueries(queries, mrn, asset)
		r.out.Write([]byte{'\n'})
	}
	r.out.Write([]byte{'\n'})
}

func (r *defaultReporter) printAssetQueries(queries map[string]*policy.Mquery, assetMrn string, asset *policy.Asset) {
	report, ok := r.data.Reports[assetMrn]
	if !ok {
		// nothing to do, we get an error message in the summary code
		return
	}

	resolved, ok := r.data.ResolvedPolicies[assetMrn]
	if !ok {
		// nothing to do, we get an additional error message in the summary code
		return
	}

	results := report.RawResults()

	r.out.Write([]byte("Data queries:\n"))
	resolved.WithDataQueries(func(id string, query *policy.ExecutionQuery) {
		data := query.Code.FilterResults(results)
		result := r.Reporter.Printer.Results(query.Code, data, report.ResolvedPolicyVersion == "v2")
		if result == "" {
			return
		}
		if r.isCompact {
			result = stringx.MaxLines(10, result)
		}
		r.out.Write([]byte(result))
		r.out.Write([]byte{'\n'})
	})
	r.out.Write([]byte("\n"))

	r.out.Write([]byte("Controls:\n"))
	for id, score := range report.Scores {
		_, ok := resolved.CollectorJob.ReportingQueries[id]
		if !ok {
			continue
		}

		query, ok := queries[id]
		if !ok {
			r.out.Write([]byte("Couldn't find any queries for incoming value for " + id))
			continue
		}

		r.printControl(score, query, asset, resolved, report, results)
	}
}

// only works with type == policy.ScoreType_Result
func (r *defaultReporter) printScore(title string, score *policy.Score, query *policy.Mquery) string {
	// FIXME: this is only a workaround for a deeper bug with the score value
	if query.Severity != nil {
		score.Value = 100 - uint32(query.Severity.Value)
	}
	rating := score.Rating()
	color := cnspecComponents.DefaultRatingColors.Color(rating)

	var passfail string
	if score.Value == 100 {
		passfail = termenv.String("✓ Pass:  ").Foreground(r.Colors.Success).String()
	} else {
		passfail = termenv.String("✕ Fail:  ").Foreground(color).String()
	}

	var suffix string
	if query.Severity != nil && score.Value != 100 {
		suffix = termenv.String(
			fmt.Sprintf(" %s %d", rating.Letter(), score.Value),
		).Foreground(color).String()
	}

	return passfail + title + suffix + "\n"
}

func (r *defaultReporter) printControl(score *policy.Score, query *policy.Mquery, asset *policy.Asset, resolved *policy.ResolvedPolicy, report *policy.Report, results map[string]*llx.RawResult) {
	title := query.Title
	if title == "" {
		title = query.Mrn
	}

	switch score.Type {
	case policy.ScoreType_Error:
		r.out.Write([]byte(termenv.String("! Error: ").Foreground(r.Colors.Error).String()))
		r.out.Write([]byte(title))
		r.out.Write([]byte{'\n'})
		if !r.isCompact {
			r.out.Write([]byte(termenv.String("  Message: " + score.Message).Foreground(r.Colors.Error).String()))
			r.out.Write([]byte{'\n'})
		}
	case policy.ScoreType_Unknown, policy.ScoreType_Unscored:
		r.out.Write([]byte(termenv.String(". Unknown: ").Foreground(r.Colors.Disabled).String()))
		r.out.Write([]byte(title))
		r.out.Write([]byte{'\n'})

	case policy.ScoreType_Skip:
		r.out.Write([]byte(termenv.String(". Skipped: ").Foreground(r.Colors.Disabled).String()))
		r.out.Write([]byte(title))
		r.out.Write([]byte{'\n'})

	case policy.ScoreType_Result:
		r.out.Write([]byte(r.printScore(title, score, query)))

		// additional information about the failed query
		if !r.isCompact && score.Value != 100 {
			r.out.Write([]byte("  Query:\n" + stringx.Indent(4, query.Query)))
			r.out.Write([]byte{'\n'})

			codeBundle := resolved.GetCodeBundle(query)
			if codeBundle == nil {
				r.out.Write([]byte(r.Reporter.Printer.Error("failed to find code bundle for query '" + query.Mrn + "' in bundle")))
			} else {
				useV2Code := report.ResolvedPolicyVersion == "v2"

				r.out.Write([]byte("  Result:\n"))
				assessment := policy.Query2Assessment(codeBundle, report)
				if assessment != nil {
					r.out.Write([]byte(stringx.Indent(4, r.Printer.Assessment(codeBundle, assessment, useV2Code))))
				} else {
					data := codeBundle.FilterResults(results)
					result := r.Reporter.Printer.Results(codeBundle, data, useV2Code)
					r.out.Write([]byte(stringx.Indent(4, result)))
				}
			}
			r.out.Write([]byte{'\n'})
		}
	default:
		r.out.Write([]byte("unknown result for " + title + "\n"))
	}
}

// ============================= ^^ ============================================

func (r *defaultReporter) printVulns(resolved *policy.ResolvedPolicy, report *policy.Report, results map[string]*llx.RawResult) {
	print := r.Printer

	value, ok := results[vulnReportDatapointChecksum]
	if !ok {
		return
	}

	r.out.Write([]byte(print.Primary("Vulnerabilities:\n")))

	if value == nil || value.Data == nil {
		r.out.Write([]byte(print.Error("Could not find the vulnerability report.") + "\n"))
		return
	}
	if value.Data.Error != nil {
		r.out.Write([]byte(print.Error("Could not load the vulnerability report: "+value.Data.Error.Error()) + "\n"))
		return
	}

	score := report.Scores[advisoryPolicyMrn]
	_ = score

	rawData := value.Data.Value

	var vulnReport vadvisor.VulnReport
	cfg := &mapstructure.DecoderConfig{
		Metadata: nil,
		Result:   &vulnReport,
		TagName:  "json",
	}
	decoder, _ := mapstructure.NewDecoder(cfg)
	err := decoder.Decode(rawData)
	if err != nil {
		r.out.Write([]byte(print.Error("could not decode advisory report\n\n")))
		return
	}

	r.printVulnList(&vulnReport)
	r.printVulnSummary(&vulnReport)
}

func (r *defaultReporter) printVulnList(report *vadvisor.VulnReport) {
	if report.GetStats() == nil || report.Stats.Advisories.Total == 0 {
		color := cnspecComponents.DefaultRatingColors.Color(policy.ScoreRating_aPlus)
		indicatorChar := '■'
		title := "No advisories found"
		state := "(passed)"
		r.out.Write([]byte(termenv.String(string(indicatorChar), title, state).Foreground(color).String()))
		r.out.Write([]byte("\n\n"))
		return
	}

	// FIXME: print advisory results
	// renderer := components.NewAdvisoryResultTable()
	// renderer.ScoreAscending = true
	// output, err := renderer.Render(report)
	// if err != nil {
	// 	r.out.Write([]byte(r.Printer.Error(err.Error() + "\n\n")))
	// 	return
	// }
	// r.out.Write([]byte(output))
	// r.out.Write([]byte("\n"))
}

func (r *defaultReporter) printVulnSummary(report *vadvisor.VulnReport) {
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

	r.out.Write([]byte(r.scoreColored(vulnScore.Rating(), fmt.Sprintf("Overall CVSS score: %.1f\n\n\n", cvss))))
}
