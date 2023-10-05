// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/upstream/mvd"
	"go.mondoo.com/cnquery/v9/shared"
	"go.mondoo.com/cnspec/v9/cli/components"
	"go.mondoo.com/cnspec/v9/cli/components/advisories"
)

// advisoryPrintable is a snapshot of the fields that get exported
// when doing things like JSON output
type advisoryPrintable struct {
	Score    string `json:"score,omitempty"`
	Advisory string `json:"advisory,omitempty"`
	Current  string `json:"current,omitempty"`
	Fixed    string `json:"fixed,omitempty"`
	Patch    string `json:"patch,omitempty"`
}

type statsPrintable struct {
	Total    int32 `json:"total"`
	Critical int32 `json:"critical"`
	High     int32 `json:"high"`
	Medium   int32 `json:"medium"`
	Low      int32 `json:"low"`
	None     int32 `json:"none"`
	Unknown  int32 `json:"unknown"`
}

type packagePrintable struct {
	Score     float32  `json:"score"`
	Name      string   `json:"package"`
	Installed string   `json:"installed"`
	Fixed     string   `json:"vulnerable"`
	Available string   `json:"available"`
	Advisory  string   `json:"advisory"`
	Cves      []string `json:"cves"`
}

func VulnReportToJSON(target string, data *mvd.VulnReport, out shared.OutputHelper) error {
	if data == nil {
		return nil
	}

	out.WriteString(
		"{" +
			"\"target\": ")
	out.WriteString("\"" + target + "\"")

	out.WriteString("," +
		"\"stats\":" +
		"{")
	out.WriteString(renderVulnerabilityStatsAsJson(data))
	out.WriteString("}")
	out.WriteString("," +
		"\"vulnerabilities\":")
	out.WriteString(renderVulnerabilitiesAsJson(data))
	out.WriteString("}")

	return nil
}

func renderVulnerabilityStatsAsJson(vulnReport *mvd.VulnReport) string {
	if vulnReport == nil || vulnReport.Stats == nil {
		return ""
	}

	var b strings.Builder

	// summary graph
	stats := vulnReport.Stats

	advisoryStats := statsPrintable{
		Total:    stats.Advisories.Total,
		Critical: stats.Advisories.Critical,
		High:     stats.Advisories.High,
		Medium:   stats.Advisories.Medium,
		Low:      stats.Advisories.Low,
		None:     stats.Advisories.None,
		Unknown:  stats.Advisories.Unknown,
	}
	jsonStats, err := json.Marshal(advisoryStats)
	if err != nil {
		log.Error().Err(err).Msg("failed to marshal advisory stats")
	}
	b.WriteString("\"advisories\":")
	b.Write(jsonStats)

	packageStats := statsPrintable{
		Total:    stats.Packages.Total,
		Critical: stats.Packages.Critical,
		High:     stats.Packages.High,
		Medium:   stats.Packages.Medium,
		Low:      stats.Packages.Low,
		None:     stats.Packages.None,
		Unknown:  stats.Packages.Unknown,
	}
	jsonStats, err = json.Marshal(packageStats)
	if err != nil {
		log.Error().Err(err).Msg("failed to marshal advisory stats")
	}
	b.WriteString(",")
	b.WriteString("\"packages\":")
	b.Write(jsonStats)

	return b.String()
}

func renderVulnerabilitiesAsJson(r *mvd.VulnReport) string {
	if r == nil {
		return ""
	}

	// platform advisories
	platformAdvisory := []*mvd.Advisory{}
	for i := range r.Advisories {
		advisory := r.Advisories[i]
		if len(advisory.FixedPlatforms) == 0 {
			continue
		}
		platformAdvisory = append(platformAdvisory, advisory)
	}

	printAdvisories := []*advisoryPrintable{}
	if len(platformAdvisory) > 0 {
		// sort advisories by score
		sort.Sort(sort.Reverse(mvd.BySeverity(platformAdvisory)))

		for i := range platformAdvisory {
			advisory := platformAdvisory[i]
			score := components.IntScore2Float(advisory.Score)

			currentVersion := r.Platform.Release
			if len(r.Platform.Build) > 0 {
				currentVersion += "/" + r.Platform.Build
			}

			fixedVersion := ""
			patch := ""
			// TODO: find the correct fixed platform entry
			if len(advisory.FixedPlatforms) > 0 {
				fixedVersion = advisory.FixedPlatforms[0].Release
				if len(advisory.FixedPlatforms[0].Build) > 0 {
					fixedVersion += "/" + advisory.FixedPlatforms[0].Build
				}
				patch = advisory.FixedPlatforms[0].PatchName
			}

			outAdvisory := &advisoryPrintable{
				Score:    fmt.Sprintf("%v", score),
				Advisory: advisory.ID,
				Current:  currentVersion,
				Fixed:    fixedVersion,
				Patch:    patch,
			}
			printAdvisories = append(printAdvisories, outAdvisory)
		}
	}

	// packages advisories
	var packages []*advisories.ReportFindingRow
	var printPkgs []*packagePrintable
	if r.Stats != nil && r.Stats.Packages != nil {
		packages = advisories.ReportAffectedPackages(r, advisories.RowWriterOpts{AdvisoryDetails: true})
		for i := range packages {
			pkg := packages[i]
			outPkg := &packagePrintable{
				Score:     components.IntScore2Float(pkg.Score),
				Name:      pkg.Name,
				Installed: pkg.Installed,
				Fixed:     pkg.Fixed,
				Available: pkg.Available,
				Advisory:  pkg.Advisory,
				Cves:      pkg.Cves,
			}
			printPkgs = append(printPkgs, outPkg)
		}
	}

	advisoriesJson, err := json.Marshal(printAdvisories)
	if err != nil {
		log.Error().Err(err).Msg("failed to marshal json")
	}

	packagesJson, err := json.Marshal(printPkgs)
	if err != nil {
		log.Error().Err(err).Msg("failed to marshal json")
	}

	out := "{" +
		"\"platform\": " +
		string(advisoriesJson) +
		"," +
		"\"packages\": " +
		string(packagesJson) +
		"}"

	return out
}
