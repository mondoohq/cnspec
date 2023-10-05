// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"bytes"
	"sort"
	"strconv"
	"strings"

	"github.com/muesli/termenv"
	"go.mondoo.com/cnquery/v9/cli/printer"
	"go.mondoo.com/cnquery/v9/cli/theme/colors"
	"go.mondoo.com/cnquery/v9/mrn"
	"go.mondoo.com/cnspec/v9/cli/components"
	"go.mondoo.com/cnspec/v9/policy"
)

type policyScore struct {
	score *policy.Score
	title string
}

func policyScores(report *policy.Report, bundle *policy.PolicyBundleMap) []policyScore {
	scores := []policyScore{}

	for id, score := range report.Scores {
		pol, ok := bundle.Policies[id]
		if !ok {
			continue
		}

		// We only keep queries and policies in printing. Normal queries will typically
		// not be a MRN. Everything except for policies and queries can be skipped
		if m, err := mrn.NewMRN(id); err == nil {
			rid, _ := m.ResourceID(policy.MRN_RESOURCE_POLICY)
			qid, _ := m.ResourceID(policy.MRN_RESOURCE_QUERY)
			if rid == "" && qid == "" {
				continue
			}
		}

		x := policyScore{
			score: score,
			title: pol.Name,
		}
		if x.title == "" {
			x.title = id
		}
		scores = append(scores, x)
	}

	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score.Value < scores[j].score.Value
	})

	return scores
}

// TODO: ================== vv CLEAN UP vv ========================

type summaryPrinter struct {
	print *printer.Printer
}

func NewSummaryRenderer(print *printer.Printer) *summaryPrinter {
	return &summaryPrinter{
		print: print,
	}
}

type summaryStats struct {
	assetScores map[string]*policy.Score
	assetNames  map[string]string
	policyStats map[string][]*policy.Score
	policyNames map[string]string
}

func (s *summaryPrinter) GenerateStats(report *policy.ReportCollection) summaryStats {
	// stats data
	stats := summaryStats{
		assetScores: map[string]*policy.Score{},
		assetNames:  map[string]string{},
		policyStats: map[string][]*policy.Score{},
		policyNames: map[string]string{},
	}

	// extract statistics from scan report
	pbm := report.Bundle.ToMap()
	for assetMrn := range report.Assets {
		stats.assetNames[assetMrn] = report.Assets[assetMrn].Name
		assetReport, ok := report.Reports[assetMrn]
		if !ok {
			if errMsg := report.Errors[assetMrn]; errMsg != "" {
				stats.assetScores[assetMrn] = &policy.Score{
					QrId:    assetMrn,
					Type:    policy.ScoreType_Error,
					Message: errMsg,
				}
			} else {
				stats.assetScores[assetMrn] = &policy.Score{
					QrId: assetMrn,
					Type: policy.ScoreType_Unknown,
				}
			}
			continue
		} else {
			stats.assetScores[assetMrn] = assetReport.Scores[assetMrn]
		}
		// stats.assetNames[assetMrn] = report.Assets[assetMrn].Name

		// iterate over each policy to get the score results per assets
		for k := range pbm.Policies {
			p := pbm.Policies[k]
			stats.policyNames[k] = p.Name

			score := assetReport.Scores[k]
			if stats.policyStats[k] == nil {
				stats.policyStats[k] = []*policy.Score{}
			}
			stats.policyStats[k] = append(stats.policyStats[k], score)
		}
	}

	return stats
}

func (s *summaryPrinter) Render(report *policy.ReportCollection) string {
	summaryStats := s.GenerateStats(report)

	var res bytes.Buffer
	res.WriteString(s.print.H1("Summary"))

	// render asset stats
	res.WriteString(s.print.Primary("Asset Overview"))
	res.WriteString(NewLineCharacter + NewLineCharacter)

	// render policy list
	mircoScoreCard := components.NewMicroScoreCard()
	for k := range summaryStats.assetScores {
		score := summaryStats.assetScores[k]
		res.WriteString("■ ")
		res.WriteString(mircoScoreCard.Render(score))
		res.WriteString(" ")
		res.WriteString(summaryStats.assetNames[k])
		res.WriteString(NewLineCharacter)
	}
	res.WriteString(NewLineCharacter)

	// render policy stats
	res.WriteString(s.print.Primary("Aggregated Policy Overview"))
	res.WriteString(NewLineCharacter + NewLineCharacter)
	data := components.StackBarData{
		Title: "Stacked Data",
		Color: []termenv.Color{
			colors.DefaultColorTheme.Good,
			colors.DefaultColorTheme.Low,
			colors.DefaultColorTheme.Medium,
			colors.DefaultColorTheme.High,
			colors.DefaultColorTheme.Critical,
			colors.DefaultColorTheme.Unknown,
		},
		Labels:  []string{"A", "B", "C", "D", "F", "U"},
		Entries: []components.StackBarDataEntry{},
	}

	if len(summaryStats.policyStats) > 0 {

		entries := []components.StackBarDataEntry{}
		ratings := []map[string]int{}

		for k := range summaryStats.policyNames {
			// We are looking for MRNs that are policies only. Everything else
			// may be filtered
			if err := policy.IsPolicyMrn(k); err != nil {
				continue
			}

			scores := summaryStats.policyStats[k]
			total := 0
			r := map[string]int{}
			for i := range scores {
				s := scores[i]
				total++
				switch s.Rating() {
				case policy.ScoreRating_aPlus, policy.ScoreRating_a, policy.ScoreRating_aMinus:
					r["a"]++
				case policy.ScoreRating_bPlus, policy.ScoreRating_b, policy.ScoreRating_bMinus:
					r["b"]++
				case policy.ScoreRating_cPlus, policy.ScoreRating_c, policy.ScoreRating_cMinus:
					r["c"]++
				case policy.ScoreRating_dPlus, policy.ScoreRating_d, policy.ScoreRating_dMinus:
					r["d"]++
				case policy.ScoreRating_failed:
					r["f"]++
				case policy.ScoreRating_skip:
					r["u"]++
				case policy.ScoreRating_unrated:
					r["u"]++
				}
			}

			// skip 100% unrated policies from the result list
			if r["u"] == total {
				continue
			}

			ratings = append(ratings, r)

			entry := components.StackBarDataEntry{
				Key:    summaryStats.policyNames[k],
				Values: []float64{0, 0, 0, 0, 0, 0},
			}

			if total > 0 {
				entry.Values = []float64{
					float64(r["a"]) / float64(total),
					float64(r["b"]) / float64(total),
					float64(r["c"]) / float64(total),
					float64(r["d"]) / float64(total),
					float64(r["f"]) / float64(total),
					float64(r["u"]) / float64(total),
				}
			}

			entries = append(entries, entry)
		}
		data.Entries = entries

		chart := components.NewStackBarChart(func(idx int, total float64, datapoints []float64, labels []string) string {
			return ratingString(ratings[idx])
		})
		res.WriteString(chart.Render(data))
	}

	return res.String()
}

func ratingString(r map[string]int) string {
	res := &bytes.Buffer{}
	res.WriteString(" ")
	for k, v := range r {
		if v == 0 {
			continue
		}
		res.WriteString(strings.ToUpper(k) + ": " + strconv.Itoa(v) + " ")
	}
	return res.String()
}
