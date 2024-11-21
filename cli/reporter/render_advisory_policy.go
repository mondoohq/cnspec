// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"bytes"
	"sort"
	"strconv"
	"strings"

	mapstructure "github.com/go-viper/mapstructure/v2"
	"github.com/muesli/termenv"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/cli/printer"
	"go.mondoo.com/cnquery/v11/cli/theme/colors"
	"go.mondoo.com/cnquery/v11/providers"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/upstream/mvd"
	"go.mondoo.com/cnquery/v11/utils/stringx"
	"go.mondoo.com/cnspec/v11/cli/components"
	"go.mondoo.com/cnspec/v11/policy"
)

// TODO: re-use the structure without importing all os resources
type KernelVersion struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Running bool   `json:"running"`
}

func renderAdvisoryPolicy(print *printer.Printer, policyObj *policy.Policy, report *policy.Report, bundle *policy.PolicyBundleMap, resolvedPolicy *policy.ResolvedPolicy, scoringData []reportRow) string {
	var b bytes.Buffer

	b.WriteString(print.H2(policyObj.Name))

	// render mini score card
	score := report.Scores[policyObj.Mrn]

	results := report.Data
	value, _ := getVulnReport(results)
	if value == nil || value.Data == nil {
		return b.String()
	}
	if value.Error != "" {
		b.Write([]byte(print.Error("Could not load the advisory report: "+value.Error) + NewLineCharacter + NewLineCharacter))
		return b.String()
	}

	rawData := value.Data.RawData().Value

	var vulnReport mvd.VulnReport
	cfg := &mapstructure.DecoderConfig{
		Metadata: nil,
		Result:   &vulnReport,
		TagName:  "json",
	}
	decoder, _ := mapstructure.NewDecoder(cfg)
	if err := decoder.Decode(rawData); err != nil {
		b.WriteString(print.Error("could not decode advisory report" + NewLineCharacter + NewLineCharacter))
		return b.String()
	}

	cvssScore := ""
	if vulnReport.Stats != nil {
		v := float64(vulnReport.Stats.Score) / 10
		cvssScore = print.Primary("CVSS:    ") + strconv.FormatFloat(v, 'f', 1, 32)
	}

	// render policy headline
	box1 := components.NewMiniScoreCard().Render(score)
	box2 := NewLineCharacter + stringx.Indent(2, print.Primary("Policy:  ")+policyObj.Name+NewLineCharacter+print.Primary("Version: ")+policyObj.Version+NewLineCharacter+print.Primary("Score:   ")+score.HumanStatus()+NewLineCharacter+cvssScore)
	b.WriteString(stringx.MergeSideBySide(
		box1,
		box2,
	))

	// summary graph
	if vulnReport.Stats != nil {
		b.WriteString(RenderVulnerabilityStats(&vulnReport))
		b.WriteString(RenderVulnReport(&vulnReport))
	}

	// render additional information
	schema := providers.DefaultRuntime().Schema()
	kernelInstalledChecksum, err := defaultChecksum(kernelInstalled, schema)
	if err != nil {
		log.Debug().Err(err).Msg("could not determine installed kernel checksum")
	}

	kernelDataValue, ok := results[kernelInstalledChecksum]
	if ok && kernelDataValue.Data != nil {
		if kernelDataValue.Error != "" {
			b.WriteString(print.Error(kernelDataValue.Error + NewLineCharacter))
		} else {
			rawData := kernelDataValue.Data.RawData().Value

			kernelVersions := []KernelVersion{}

			cfg := &mapstructure.DecoderConfig{
				Metadata: nil,
				Result:   &kernelVersions,
				TagName:  "json",
			}
			decoder, _ := mapstructure.NewDecoder(cfg)
			err := decoder.Decode(rawData)
			if err != nil {
				b.WriteString(print.Error("could not decode kernel versions" + NewLineCharacter))
			} else {
				b.WriteString("Installed Kernel Versions:" + NewLineCharacter)

				// sort the kernel version
				// NOTE: this is poor man's version since the versions can vary a lot and comparison is more complicated
				sort.SliceStable(kernelVersions, func(i, j int) bool {
					return kernelVersions[i].Version > kernelVersions[j].Version
				})

				// print kernel versions
				for i := range kernelVersions {
					kv := kernelVersions[i]
					if kv.Running {
						b.WriteString(print.Secondary(" * " + kv.Version + " (running)"))
					} else {
						b.WriteString(print.Disabled(" * " + kv.Version + " (not running)"))
					}
					b.WriteString(NewLineCharacter)
				}
			}
		}
		b.WriteString(NewLineCharacter)
	}

	// TODO: iterate over all other scoring queries that are not covered within the screen above
	b.WriteString("Additional Checks:" + NewLineCharacter)
	scoreQueries := map[string]struct{}{}
	for i := range policyObj.Groups {
		group := policyObj.Groups[i]
		for i := range group.Checks {
			check := group.Checks[i]
			scoreQueries[check.Mrn] = struct{}{}
		}
	}

	ignoreList := []string{"no-platform-advisories", "no-platform-cves", "installed-kernels"}
	ignore := func(k string) bool {
		// skip query its already included
		for j := range ignoreList {
			if strings.HasSuffix(k, ignoreList[j]) {
				return true
			}
		}
		return false
	}

	for k := range scoreQueries {
		if ignore(k) {
			continue
		}

		q, ok := bundle.Queries[k]
		if ok {
			state := print.Disabled("(unscored)")
			score, sok := report.Scores[q.CodeId]
			if sok {
				if score.Value == 100 {
					state = print.Success("(passed) ")
				} else {
					state = print.Failed("(failed)")
				}
			}

			b.WriteString(scoreIndicator(score))
			severity := scoreRating(score)
			color := components.DefaultRatingColors.Color(severity)
			b.WriteString(termenv.String(" " + q.Title + " " + state).Foreground(color).String())
			b.WriteString(NewLineCharacter)
		}
	}

	b.WriteString(NewLineCharacter)
	return b.String()
}

func scoreIndicator(score *policy.Score) string {
	char := '■'
	color := components.DefaultRatingColors.Color(scoreRating(score))
	return termenv.String(string(char)).Foreground(color).String()
}

func RenderVulnerabilityStats(vulnReport *mvd.VulnReport) string {
	if vulnReport == nil || vulnReport.Stats == nil {
		return ""
	}

	var b bytes.Buffer

	// summary graph
	stats := vulnReport.Stats

	// only render if we have advisories
	if stats.Advisories.Total > 0 {
		total := stats.Advisories.Total
		colorMap := []termenv.Color{
			colors.DefaultColorTheme.Critical,
			colors.DefaultColorTheme.High,
			colors.DefaultColorTheme.Medium,
			colors.DefaultColorTheme.Low,
			colors.DefaultColorTheme.Good,
		}
		labels := []string{"Critical", "High", "Medium", "Low", "None"}
		datapoints := []float64{
			(float64(stats.Advisories.Critical) / float64(total)),
			(float64(stats.Advisories.High) / float64(total)),
			(float64(stats.Advisories.Medium) / float64(total)),
			(float64(stats.Advisories.Low) / float64(total)),
			(float64(stats.Advisories.None) / float64(total)),
		}

		// only add unknown if it really happened
		if vulnReport.Stats.Advisories.Unknown > 0 {
			colorMap = append(colorMap, colors.DefaultColorTheme.Unknown)
			labels = append(labels, "Unknown")
			datapoints = append(datapoints, (float64(vulnReport.Stats.Advisories.Unknown) / float64(total)))
		}

		// render advisories bar chart
		advisoriesBarChart := components.NewBarChart(
			components.WithBarChartBorder(),
			components.WithBarChartTitle("Advisories"),
			components.WithBarChartLabelFunc(components.BarChartPercentageLabelFunc),
		)
		b.WriteString(advisoriesBarChart.Render(datapoints, colorMap, labels))
		b.WriteString(NewLineCharacter)
	}

	// only render if we have packages scanned, not the case for vmware ESXi
	if stats.Packages.Total > 0 {
		pkgTotal := stats.Packages.Total
		pkgColorMap := []termenv.Color{
			colors.DefaultColorTheme.Unknown,
			colors.DefaultColorTheme.Critical,
			colors.DefaultColorTheme.High,
			colors.DefaultColorTheme.Medium,
			colors.DefaultColorTheme.Low,
		}
		pkgLabels := []string{"Total", "Critical", "High", "Medium", "Low"}

		max := stats.Packages.Critical
		if stats.Packages.High > max {
			max = stats.Packages.High
		}
		if stats.Packages.Medium > max {
			max = stats.Packages.Medium
		}
		if stats.Packages.Low > max {
			max = stats.Packages.Low
		}
		pkgDatapoints := []float64{
			float64(1.0), // number of packages is always 100%
			(float64(stats.Packages.Critical) / float64(max)),
			(float64(stats.Packages.High) / float64(max)),
			(float64(stats.Packages.Medium) / float64(max)),
			(float64(stats.Packages.Low) / float64(max)),
		}

		// values for datapoints
		valueLabels := []int32{pkgTotal, stats.Packages.Critical, stats.Packages.High, stats.Packages.Medium, stats.Packages.Low}

		// render packages bar chart
		packagesBarChart := components.NewBarChart(
			components.WithBarChartBorder(),
			components.WithBarChartTitle("Packages"),
			components.WithBarChartLabelFunc(func(i int, datapoints []float64) string {
				return strconv.FormatInt(int64(valueLabels[i]), 10)
			}),
		)
		b.WriteString(packagesBarChart.Render(pkgDatapoints, pkgColorMap, pkgLabels))
		b.WriteString(NewLineCharacter)
	}

	return b.String()
}

func RenderVulnReport(vulnReport *mvd.VulnReport) string {
	return RenderVulnReportDetailed(vulnReport, false)
}

func RenderVulnReportDetailed(vulnReport *mvd.VulnReport, detailed bool) string {
	var b bytes.Buffer
	if vulnReport == nil || vulnReport.Stats == nil || vulnReport.Stats.Advisories.Total == 0 {
		color := components.DefaultRatingColors.Color(policy.ScoreRating_aPlus)
		indicatorChar := '■'
		title := "No advisories found"
		state := "(passed)"
		b.WriteString(termenv.String(string(indicatorChar)).Foreground(color).String())
		b.WriteString(termenv.String(" " + title + " " + state).Foreground(color).String())
		b.WriteString(NewLineCharacter + NewLineCharacter)
	} else {
		// render advisory table
		renderer := components.NewAdvisoryResultTable()
		if detailed {
			renderer.DetailedPackageRisks = true
		}
		output, err := renderer.Render(vulnReport)
		if err != nil {
			return err.Error()
		}
		b.WriteString(output)
		b.WriteString(NewLineCharacter)
	}

	return b.String()
}
