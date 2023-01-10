package reporter

import (
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/mitchellh/mapstructure"
	"github.com/muesli/ansi"
	"github.com/muesli/termenv"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/cli/components"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/stringx"
	"go.mondoo.com/cnquery/upstream/mvd"
	cnspecComponents "go.mondoo.com/cnspec/cli/components"
	"go.mondoo.com/cnspec/policy"
)

type assetMrnName struct {
	Mrn  string
	Name string
}

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

	r.out.Write([]byte(NewLineCharacter))
	if !r.isSummary {
		r.printAssetSections(orderedAssets)
	}

	r.printSummary(orderedAssets)
	return nil
}

func (r *defaultReporter) printSummary(orderedAssets []assetMrnName) {
	assetUrl := ""
	assetsByPlatform := make(map[string][]*policy.Asset)
	assetsByScore := make(map[string]int)
	for _, assetMrnName := range orderedAssets {
		assetMrn := assetMrnName.Mrn
		asset := r.data.Assets[assetMrn]
		if asset.Url != "" {
			assetUrl = asset.Url
		}
		if asset.PlatformName != "" {
			assetsByPlatform[asset.PlatformName] = append(assetsByPlatform[asset.PlatformName], asset)
		}
		if _, ok := r.data.Reports[assetMrn]; ok {
			assetScore := r.data.Reports[assetMrn].Score.Rating().Letter()
			assetsByScore[assetScore]++
		}
	}

	if len(assetsByScore) > 0 {
		header := fmt.Sprintf("Scanned %d assets", len(r.data.Assets))
		r.out.Write([]byte(termenv.String(header + NewLineCharacter).Foreground(r.Colors.Primary).String()))
	}

	// print assets by platform
	r.printAssetsByPlatform(assetsByPlatform)

	// print distributions
	if len(orderedAssets) > 1 {
		summaryHeader := fmt.Sprintf("Summary")
		summaryDivider := strings.Repeat("=", utf8.RuneCountInString(summaryHeader))
		r.out.Write([]byte(NewLineCharacter))
		r.out.Write([]byte(termenv.String(summaryHeader + NewLineCharacter + summaryDivider + NewLineCharacter).Foreground(r.Colors.Primary).String()))
		r.out.Write([]byte(NewLineCharacter))

		scoreHeader := "Score Distribution"
		assetHeader := "Asset Distribution"
		header := scoreHeader + "\t\t" + assetHeader
		headerDivider := strings.Repeat("-", utf8.RuneCountInString(scoreHeader)) + "\t\t" + strings.Repeat("-", utf8.RuneCountInString(assetHeader))

		r.out.Write([]byte(header + NewLineCharacter))
		r.out.Write([]byte(headerDivider + NewLineCharacter))

		scores := r.getScoreDistribution(assetsByScore)
		assets := r.getAssetDistribution(assetsByPlatform)

		maxIndex := 0
		if len(scores) > len(assets) {
			maxIndex = len(scores)
		} else {
			maxIndex = len(assets)
		}
		// I also gave the tablewriter a try, but it didn't generate a nice output
		for i := 0; i < maxIndex; i++ {
			row := ""
			addedScore := false
			if i < len(scores) {
				row = scores[i]
				addedScore = true
			}
			if i < len(assets) {
				if !addedScore {
					row += strings.Repeat(" ", utf8.RuneCountInString(scoreHeader))
				} else {
					visibleScoreWidth := ansi.PrintableRuneWidth(scores[i])
					spacing := utf8.RuneCountInString(scoreHeader) - visibleScoreWidth
					row += strings.Repeat(" ", spacing)
				}
				row += "\t\t"
				row += assets[i]
			}
			row += NewLineCharacter
			r.out.Write([]byte(row))
		}
	}

	if r.isCompact {
		r.out.Write([]byte(NewLineCharacter))
		if !r.IsIncognito && assetUrl != "" {
			url := ""
			if len(orderedAssets) > 1 {
				// we do not have a space url, so we extract it form the asset url
				// https://console.mondoo.com/space/fleet/2JtqGyVTZULTW0uwQ5YxXW4nh6Y?spaceId=dazzling-golick-767384
				// an individual asset url wouldn't make sense here
				spaceUrlRegexp := regexp.MustCompile(`^(http.*)/[a-zA-Z0-9-]+(\?.+)$`)
				m := spaceUrlRegexp.FindStringSubmatch(assetUrl)
				url = m[1] + m[2]
			} else {
				url = assetUrl
			}

			r.out.Write([]byte("See detailed report information in the Mondoo Console: "))
			r.out.Write([]byte(url + NewLineCharacter))
		}
		if len(assetsByScore) > 0 {
			r.out.Write([]byte("To get more information on the CLI, please run this scan with \"-o full\"." + NewLineCharacter))
		}
	}
}

func (r *defaultReporter) getScoreDistribution(assetsByScore map[string]int) []string {
	scores := []string{}
	for _, score := range []string{"A", "B", "C", "D", "F", "U", "X"} {
		scoreColor := r.Colors.Unknown
		switch score {
		case "A":
			scoreColor = cnspecComponents.DefaultRatingColors.Color(policy.ScoreRating_a)
		case "B":
			scoreColor = cnspecComponents.DefaultRatingColors.Color(policy.ScoreRating_b)
		case "C":
			scoreColor = cnspecComponents.DefaultRatingColors.Color(policy.ScoreRating_c)
		case "D":
			scoreColor = cnspecComponents.DefaultRatingColors.Color(policy.ScoreRating_d)
		case "F":
			scoreColor = cnspecComponents.DefaultRatingColors.Color(policy.ScoreRating_failed)
		case "X":
			scoreColor = cnspecComponents.DefaultRatingColors.Color(policy.ScoreRating_error)
		}
		coloredScore := termenv.String(score).Foreground(scoreColor).String()
		output := fmt.Sprintf("%s %3d assets", coloredScore, assetsByScore[score])
		if score == "X" || score == "U" {
			if _, ok := assetsByScore[score]; !ok {
				continue
			}
		}
		scores = append(scores, output)
	}
	return scores
}

func (r *defaultReporter) getAssetDistribution(assetsByPlatform map[string][]*policy.Asset) []string {
	assets := []string{}

	maxPlatformLength := 0
	for platform := range assetsByPlatform {
		if len(platform) > maxPlatformLength {
			maxPlatformLength = len(platform)
		}
	}

	for platform := range assetsByPlatform {
		spacing := strings.Repeat(" ", maxPlatformLength-len(platform))
		output := fmt.Sprintf("%s %s%3d", platform, spacing, len(assetsByPlatform[platform]))
		assets = append(assets, output)
	}

	return assets
}

func (r *defaultReporter) printAssetsByPlatform(assetsByPlatform map[string][]*policy.Asset) {
	availablePlatforms := make([]string, 0, len(assetsByPlatform))
	for k := range assetsByPlatform {
		availablePlatforms = append(availablePlatforms, k)
	}
	sort.Strings(availablePlatforms)

	for _, platform := range availablePlatforms {
		r.out.Write([]byte(NewLineCharacter + platform + NewLineCharacter))
		for i := range assetsByPlatform[platform] {
			assetScoreRating := r.data.Reports[assetsByPlatform[platform][i].Mrn].Score.Rating()
			assetScore := assetScoreRating.Letter()
			scoreColor := cnspecComponents.DefaultRatingColors.Color(assetScoreRating)
			output := fmt.Sprintf("    %s %s", termenv.String(assetScore).Foreground(scoreColor), assetsByPlatform[platform][i].Name)
			r.out.Write([]byte(output + NewLineCharacter))
		}
	}
}

func printCompactScoreSummary(score *policy.Score) string {
	return fmt.Sprintf("%s   %3d/100     (%d%% completed)",
		score.Rating().Letter(),
		score.Value, score.Completion())
}

func failureHbar(stats *policy.Stats) string {
	var res string

	if stats.Failed.A > 0 {
		c := cnspecComponents.DefaultRatingColors.Color(policy.ScoreRating_a)
		pct := float32(stats.Failed.A) / float32(stats.Total) * 100
		res += termenv.String(components.Hbar(15, pct)).Foreground(c).String()
	}
	if stats.Failed.B > 0 {
		c := cnspecComponents.DefaultRatingColors.Color(policy.ScoreRating_b)
		pct := float32(stats.Failed.B) / float32(stats.Total) * 100
		res += termenv.String(components.Hbar(15, pct)).Foreground(c).String()
	}
	if stats.Failed.C > 0 {
		c := cnspecComponents.DefaultRatingColors.Color(policy.ScoreRating_c)
		pct := float32(stats.Failed.C) / float32(stats.Total) * 100
		res += termenv.String(components.Hbar(15, pct)).Foreground(c).String()
	}
	if stats.Failed.D > 0 {
		c := cnspecComponents.DefaultRatingColors.Color(policy.ScoreRating_d)
		pct := float32(stats.Failed.D) / float32(stats.Total) * 100
		res += termenv.String(components.Hbar(15, pct)).Foreground(c).String()
	}
	if stats.Failed.F > 0 {
		c := cnspecComponents.DefaultRatingColors.Color(policy.ScoreRating_failed)
		pct := float32(stats.Failed.F) / float32(stats.Total) * 100
		res += termenv.String(components.Hbar(15, pct)).Foreground(c).String()
	}

	return res
}

func addSpace(s string) string {
	if s == "" {
		return s
	}
	return s + " "
}

func (r *defaultReporter) printAssetSections(orderedAssets []assetMrnName) {
	if len(orderedAssets) == 0 || len(r.data.Errors) == len(orderedAssets) {
		return
	}

	r.out.Write([]byte(NewLineCharacter))

	var queries map[string]*policy.Mquery
	if r.bundle != nil {
		queries = r.bundle.QueryMap()
	}

	for _, assetMrnName := range orderedAssets {
		assetMrn := assetMrnName.Mrn
		asset := r.data.Assets[assetMrn]
		target := asset.Name
		if target == "" {
			target = assetMrn
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
		assetString := fmt.Sprintf("Asset: %s", target)
		assetDivider := strings.Repeat("=", utf8.RuneCountInString(assetString))
		r.out.Write([]byte(termenv.String("Asset: ").Foreground(r.Colors.Secondary).String()))
		r.out.Write([]byte(termenv.String(fmt.Sprintf("%s%s", target, NewLineCharacter)).Foreground(r.Colors.Primary).String()))
		r.out.Write([]byte(termenv.String(assetDivider).Foreground(r.Colors.Secondary).String()))
		r.out.Write([]byte(NewLineCharacter))

		resolved, ok := r.data.ResolvedPolicies[assetMrn]
		if !ok {
			// nothing to do, we get an additional error message in the summary code
			continue
		}

		r.printAssetQueries(resolved, report, queries, assetMrn, asset)
		r.out.Write([]byte(NewLineCharacter))
		// TODO: we should re-use the report results
		r.printVulns(resolved, report, report.RawResults())

	}
	r.out.Write([]byte(NewLineCharacter))
}

// TODO: this should be done during the execution, as queries come in, not at the end!
// Remove all this code and migrate it to tap or something
// ============================= vv ============================================

func (r *defaultReporter) printAssetQueries(resolved *policy.ResolvedPolicy, report *policy.Report, queries map[string]*policy.Mquery, assetMrn string, asset *policy.Asset) {
	results := report.RawResults()

	dataQueriesOutput := ""
	resolved.WithDataQueries(func(id string, query *policy.ExecutionQuery) {
		data := query.Code.FilterResults(results)
		result := r.Reporter.Printer.Results(query.Code, data)
		if result == "" {
			return
		}
		if r.isCompact {
			result = stringx.MaxLines(10, result)
		}
		dataQueriesOutput += result + NewLineCharacter
	})

	if len(dataQueriesOutput) > 0 {
		r.out.Write([]byte("Data queries:" + NewLineCharacter))
		r.out.Write([]byte(dataQueriesOutput))
		r.out.Write([]byte(NewLineCharacter))
	}

	r.out.Write([]byte("Controls:" + NewLineCharacter))
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
		floor := 100 - uint32(query.Severity.Value)
		if floor > score.Value {
			score.Value = floor
		}
	}
	rating := score.Rating()
	color := cnspecComponents.DefaultRatingColors.Color(rating)

	var passfail string
	if score.Value == 100 {
		passfail = termenv.String("✓ Pass:  ").Foreground(r.Colors.Success).String()
	} else {
		passfail = termenv.String("✕ Fail:  ").Foreground(color).String()
	}

	var scoreIndicator string
	if query.Severity != nil && score.Value != 100 {
		scoreIndicator = termenv.String(
			fmt.Sprintf("%s %3d  ", rating.Letter(), score.Value),
		).Foreground(color).String()
	}

	return passfail + scoreIndicator + title + NewLineCharacter
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
		r.out.Write([]byte(NewLineCharacter))
		if !r.isCompact {
			errorMessage := strings.ReplaceAll(score.Message, "\n", NewLineCharacter)
			r.out.Write([]byte(termenv.String("  Message: " + errorMessage).Foreground(r.Colors.Error).String()))
			r.out.Write([]byte(NewLineCharacter))
		}
	case policy.ScoreType_Unknown, policy.ScoreType_Unscored:
		r.out.Write([]byte(termenv.String(". Unknown: ").Foreground(r.Colors.Disabled).String()))
		r.out.Write([]byte(title))
		r.out.Write([]byte(NewLineCharacter))

	case policy.ScoreType_Skip:
		r.out.Write([]byte(termenv.String(". Skipped: ").Foreground(r.Colors.Disabled).String()))
		r.out.Write([]byte(title))
		r.out.Write([]byte(NewLineCharacter))

	case policy.ScoreType_Result:
		r.out.Write([]byte(r.printScore(title, score, query)))

		// additional information about the failed query
		if !r.isCompact && score.Value != 100 {
			queryString := strings.ReplaceAll(stringx.Indent(4, query.Query), "\n", NewLineCharacter)
			r.out.Write([]byte("  Query:" + NewLineCharacter + queryString))
			r.out.Write([]byte(NewLineCharacter))

			codeBundle := resolved.GetCodeBundle(query)
			if codeBundle == nil {
				r.out.Write([]byte(r.Reporter.Printer.Error("failed to find code bundle for query '" + query.Mrn + "' in bundle")))
			} else {
				r.out.Write([]byte("  Result:" + NewLineCharacter))
				assessment := policy.Query2Assessment(codeBundle, report)
				if assessment != nil {
					assessmentString := stringx.Indent(4, r.Printer.Assessment(codeBundle, assessment))
					assessmentString = strings.ReplaceAll(assessmentString, "\n", NewLineCharacter)
					r.out.Write([]byte(assessmentString))
				} else {
					data := codeBundle.FilterResults(results)
					result := stringx.Indent(4, r.Reporter.Printer.Results(codeBundle, data))
					result = strings.ReplaceAll(result, "\n", NewLineCharacter)
					r.out.Write([]byte(result))
				}
			}
			r.out.Write([]byte(NewLineCharacter))
		}
	default:
		r.out.Write([]byte("unknown result for " + title + NewLineCharacter))
	}
}

// ============================= ^^ ============================================

func (r *defaultReporter) printVulns(resolved *policy.ResolvedPolicy, report *policy.Report, results map[string]*llx.RawResult) {
	print := r.Printer

	value, ok := results[vulnReportDatapointChecksum]
	if !ok {
		return
	}

	r.out.Write([]byte(print.Primary("Vulnerabilities:" + NewLineCharacter)))

	if value == nil || value.Data == nil {
		r.out.Write([]byte(print.Error("Could not find the vulnerability report.") + NewLineCharacter))
		return
	}
	if value.Data.Error != nil {
		r.out.Write([]byte(print.Error("Could not load the vulnerability report: "+value.Data.Error.Error()) + NewLineCharacter))
		return
	}

	score := report.Scores[advisoryPolicyMrn]
	_ = score

	rawData := value.Data.Value

	var vulnReport mvd.VulnReport
	cfg := &mapstructure.DecoderConfig{
		Metadata: nil,
		Result:   &vulnReport,
		TagName:  "json",
	}
	decoder, _ := mapstructure.NewDecoder(cfg)
	err := decoder.Decode(rawData)
	if err != nil {
		r.out.Write([]byte(print.Error("could not decode advisory report" + NewLineCharacter + NewLineCharacter)))
		return
	}

	r.printVulnList(&vulnReport)
	r.printVulnSummary(&vulnReport)
}

func (r *defaultReporter) printVulnList(report *mvd.VulnReport) {
	if report.GetStats() == nil || report.Stats.Advisories.Total == 0 {
		color := cnspecComponents.DefaultRatingColors.Color(policy.ScoreRating_aPlus)
		indicatorChar := '■'
		title := "No advisories found"
		state := "(passed)"
		r.out.Write([]byte(termenv.String(string(indicatorChar), title, state).Foreground(color).String()))
		r.out.Write([]byte(NewLineCharacter + NewLineCharacter))
		return
	}
	r.out.Write([]byte(RenderVulnReport(report)))
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

	r.out.Write([]byte(r.scoreColored(vulnScore.Rating(), fmt.Sprintf("Overall CVSS score: %.1f%s%s", cvss, NewLineCharacter, NewLineCharacter))))
}
