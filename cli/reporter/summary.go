// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"bytes"
	"strconv"
	"strings"

	"github.com/muesli/termenv"
	"go.mondoo.com/cnspec/cli/components"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/mql/cli/printer"
	"go.mondoo.com/mql/cli/theme/colors"
)

// TODO: ================== vv CLEAN UP vv ========================

type summaryPrinter struct {
	print *printer.Printer
}

func NewSummaryRenderer(print *printer.Printer) *summaryPrinter {
	return &summaryPrinter{
		print: print,
	}
}

// SummaryStats is the per-asset and per-policy view of a report collection: the
// score of every asset (synthesized for assets that errored or were never
// reported on) and, per policy, one score per asset. It is what the summary
// renderer draws and what the report viewer filters on.
type SummaryStats struct {
	// AssetScores maps an asset MRN to its overall score.
	AssetScores map[string]*policy.Score
	// AssetNames maps an asset MRN to its human-readable name.
	AssetNames map[string]string
	// PolicyStats maps a policy MRN to the score it got on each asset, in the
	// order the assets were visited.
	PolicyStats map[string][]*policy.Score
	// PolicyNames maps a policy MRN to its human-readable name.
	PolicyNames map[string]string
}

// GenerateStats extracts the per-asset and per-policy scores from a report
// collection. An asset with no report is not an asset without findings: it gets
// a synthesized ScoreType_Error score carrying the scan error, or a
// ScoreType_Unknown score when there is no error either.
func GenerateStats(report *policy.ReportCollection) SummaryStats {
	// stats data
	stats := SummaryStats{
		AssetScores: map[string]*policy.Score{},
		AssetNames:  map[string]string{},
		PolicyStats: map[string][]*policy.Score{},
		PolicyNames: map[string]string{},
	}

	// extract statistics from scan report
	//
	// A collection can arrive without a bundle: a scan where every asset failed
	// to connect never resolves policies, so there is nothing to attach. That is
	// the case a caller most wants stats for -- it is the difference between
	// "no findings" and "nothing ran" -- so it must not be the case that panics.
	// An empty map rather than a nil one: the policy loop below then simply
	// finds nothing, instead of every caller needing its own guard.
	pbm := &policy.PolicyBundleMap{}
	if report.Bundle != nil {
		pbm = report.Bundle.ToMap()
	}
	for assetMrn := range report.Assets {
		stats.AssetNames[assetMrn] = report.Assets[assetMrn].Name
		assetReport, ok := report.Reports[assetMrn]
		if !ok {
			if errMsg := report.Errors[assetMrn]; errMsg != "" {
				stats.AssetScores[assetMrn] = &policy.Score{
					QrId:    assetMrn,
					Type:    policy.ScoreType_Error,
					Message: errMsg,
				}
			} else {
				stats.AssetScores[assetMrn] = &policy.Score{
					QrId: assetMrn,
					Type: policy.ScoreType_Unknown,
				}
			}
			continue
		} else {
			stats.AssetScores[assetMrn] = assetReport.Scores[assetMrn]
		}
		// stats.AssetNames[assetMrn] = report.Assets[assetMrn].Name

		// iterate over each policy to get the score results per assets
		for k := range pbm.Policies {
			p := pbm.Policies[k]
			stats.PolicyNames[k] = p.Name

			score := assetReport.Scores[k]
			if stats.PolicyStats[k] == nil {
				stats.PolicyStats[k] = []*policy.Score{}
			}
			stats.PolicyStats[k] = append(stats.PolicyStats[k], score)
		}
	}

	return stats
}

func (s *summaryPrinter) Render(report *policy.ReportCollection) string {
	stats := GenerateStats(report)

	var res bytes.Buffer
	res.WriteString(s.print.H1("Summary"))

	// render asset stats
	res.WriteString(s.print.Primary("Asset Overview"))
	res.WriteString(NewLineCharacter + NewLineCharacter)

	// render policy list
	microScoreCard := components.NewMicroScoreCard()
	for k := range stats.AssetScores {
		score := stats.AssetScores[k]
		res.WriteString("■ ")
		res.WriteString(microScoreCard.Render(score))
		res.WriteString(" ")
		res.WriteString(stats.AssetNames[k])
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

	if len(stats.PolicyStats) > 0 {

		entries := []components.StackBarDataEntry{}
		ratings := []map[string]int{}

		for k := range stats.PolicyNames {
			// We are looking for MRNs that are policies only. Everything else
			// may be filtered
			if err := policy.IsPolicyMrn(k); err != nil {
				continue
			}

			scores := stats.PolicyStats[k]
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
				Key:    stats.PolicyNames[k],
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
