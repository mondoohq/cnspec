// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"encoding/csv"
	"fmt"
	"strings"

	"go.mondoo.com/cnquery/shared"
	"go.mondoo.com/cnquery/providers-sdk/v1/upstream/mvd"
	"go.mondoo.com/cnspec/cli/components"
	"go.mondoo.com/cnspec/cli/components/advisories"
)

type csvStruct struct {
	Name      string
	Score     string
	Installed string
	Fixed     string
	Available string
	Advisory  string
	Cves      string
}

func (c csvStruct) toSlice() []string {
	return []string{c.Name, c.Score, c.Installed, c.Fixed, c.Available, c.Advisory, c.Cves}
}

// ReportCollectionToCSV writes the given report collection to the given output directory
func VulnReportToCSV(data *mvd.VulnReport, out shared.OutputHelper) error {
	w := csv.NewWriter(out)

	// write header
	err := w.Write(csvStruct{
		"Package Name",
		"Score",
		"Installed",
		"Fixed",
		"Available",
		"Advisory",
		"CVEs",
	}.toSlice())
	if err != nil {
		return err
	}

	pkgs := renderVulnerabilitiesAsCSV(data)

	for i := range pkgs {
		pkg := pkgs[i]
		err := w.Write(pkg.toSlice())
		if err != nil {
			return err
		}
	}

	w.Flush()
	return w.Error()
}

func renderVulnerabilitiesAsCSV(r *mvd.VulnReport) []*csvStruct {
	if r == nil {
		return []*csvStruct{}
	}

	// packages advisories
	var packages []*advisories.ReportFindingRow
	var printPkgs []*csvStruct
	if r.Stats != nil && r.Stats.Packages != nil {
		packages = advisories.ReportAffectedPackages(r, advisories.RowWriterOpts{AdvisoryDetails: true})
		for i := range packages {
			pkg := packages[i]
			outPkg := &csvStruct{
				Score:     fmt.Sprintf("%v", components.IntScore2Float(pkg.Score)),
				Name:      pkg.Name,
				Installed: pkg.Installed,
				Fixed:     pkg.Fixed,
				Available: pkg.Available,
				Advisory:  pkg.Advisory,
				Cves:      strings.Join(pkg.Cves, " "),
			}
			printPkgs = append(printPkgs, outPkg)
		}
	}

	return printPkgs
}
