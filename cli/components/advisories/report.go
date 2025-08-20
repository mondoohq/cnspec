// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package advisories

import (
	"fmt"
	"sort"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/upstream/mvd"
	"go.mondoo.com/cnquery/v12/providers/core/resources/versions/generic"
)

type RowWriter interface {
	WriteHeader() error
	Write(row ReportFindingRow) error
	Flush()
}

type RowWriterOpts struct {
	AdvisoryDetails bool
	ScoreAscending  bool
}

type ReportFindingRow struct {
	Score     int32    `json:"score"`
	Name      string   `json:"package"`
	Installed string   `json:"installed"`
	Fixed     string   `json:"vulnerable"`
	Available string   `json:"available"`
	Advisory  string   `json:"advisory"`
	Cves      []string `json:"cves"`
}

func RenderReport(report *mvd.VulnReport, writer RowWriter, opts RowWriterOpts) error {
	if report == nil || len(report.Advisories) == 0 {
		return nil
	}

	defer writer.Flush()

	err := writer.WriteHeader()
	if err != nil {
		return err
	}

	rows := ReportAffectedPackages(report, opts)
	for i := range rows {
		err = writer.Write(*rows[i])
		if err != nil {
			return err
		}
	}

	return nil
}

func ReportAffectedPackages(report *mvd.VulnReport, opts RowWriterOpts) []*ReportFindingRow {
	rows := make([]*ReportFindingRow, 0)
	advisoriesByPackage := advisoriesByPackage(report.Advisories)

	// iterate over all affected packages, therefore we filter the package list by affected
	affectedPkgs := mvd.FilterByAffected(report.Packages)
	// sort packages by score
	if opts.ScoreAscending {
		sort.Sort(mvd.ByPkgSeverity(affectedPkgs))
	} else {
		sort.Sort(sort.Reverse(mvd.ByPkgSeverity(affectedPkgs)))
	}

	for i := range affectedPkgs {
		pkg := affectedPkgs[i]

		// iterate over each advisory per package
		id := pkgID(pkg)

		advisories := advisoriesByPackage[id].Advisories
		if opts.ScoreAscending {
			sort.Sort(mvd.BySeverity(advisories))
		} else {
			sort.Sort(sort.Reverse(mvd.BySeverity(advisories)))
		}

		if len(advisories) > 0 {
			pkgWithMaxFixed := advisoriesByPackage[id].Pkg
			row := ReportFindingRow{
				Score:     pkgWithMaxFixed.Score,
				Name:      pkgWithMaxFixed.Name,
				Installed: pkgWithMaxFixed.Version,
				Fixed:     filterZeroEpochPrefix(advisoriesByPackage[id].MaxVulnerable),
				Available: pkgWithMaxFixed.Available,
				Advisory:  "",
				Cves:      nil,
			}
			rows = append(rows, &row)

			if opts.AdvisoryDetails {
				// iterate over all advisories and print details
				for j := range advisories {
					advisory := advisories[j]

					var vulnerableVersion string
					vulnerable := findVulnerablePackageWithoutNamespace(advisory, pkg)
					if vulnerable != nil {
						vulnerableVersion = vulnerable.Version
					}

					row := ReportFindingRow{
						Score:     advisory.Score,
						Name:      pkg.Name,
						Installed: pkg.Version,
						Fixed:     filterZeroEpochPrefix(vulnerableVersion),
						Available: pkg.Available,
						Advisory:  advisory.ID,
						Cves:      cvesToString(advisory.Cves),
					}

					rows = append(rows, &row)
				}
			}
		} else {
			log.Warn().Str("name", pkg.Name).Msg("pkg is missing advisory information, please contact support.")
			row := ReportFindingRow{
				Score:     0.0,
				Name:      pkg.Name,
				Installed: pkg.Version,
				Fixed:     "",
				Available: pkg.Available,
				Advisory:  "",
				Cves:      nil,
			}

			rows = append(rows, &row)
		}
	}
	return rows
}

// filters a leading 0: for rpm versions
func filterZeroEpochPrefix(version string) string {
	return strings.TrimPrefix(version, "0:")
}

func cvesToString(cves []*mvd.CVE) []string {
	list := make([]string, len(cves))
	for i := range cves {
		list[i] = cves[i].ID
	}
	return list
}

type Entry struct {
	Pkg           *mvd.Package
	Advisories    []*mvd.Advisory
	MaxVulnerable string
}

func pkgID(pkg *mvd.Package) string {
	return fmt.Sprintf("%s/%s/%s/%s", pkg.Namespace, pkg.Name, pkg.Version, pkg.Arch)
}

// AdvisoriesByPackage iterates over advisories and store all advisories per package
// The entry.Pkg.Fixed will state the latest version that fixes all advisories
func advisoriesByPackage(advisories []*mvd.Advisory) map[string]Entry {
	advisoriesByPackage := map[string]Entry{}
	for i := range advisories {
		advisory := advisories[i]
		// iterate over all affected packages per advisory
		for j := range advisory.Affected {

			pkg := advisory.Affected[j]
			fixed := findVulnerablePackageWithoutNamespace(advisory, pkg)

			maxVulnerable := ""
			if fixed != nil {
				maxVulnerable = fixed.Version
			}

			pID := pkgID(pkg)
			pkgEntry, ok := advisoriesByPackage[pID]
			if !ok {
				advisoriesByPackage[pID] = Entry{
					Pkg:           pkg,
					Advisories:    []*mvd.Advisory{advisory},
					MaxVulnerable: maxVulnerable,
				}
			} else {
				if fixed != nil {
					// determine max version of required fix for advisory
					cmp, err := generic.Compare(pkgEntry.Pkg.Format, pkgEntry.MaxVulnerable, maxVulnerable)
					if err == nil && cmp < 0 {
						pkgEntry.MaxVulnerable = maxVulnerable
					}
				}

				pkgEntry.Advisories = append(pkgEntry.Advisories, advisory)
				advisoriesByPackage[pID] = pkgEntry
			}
		}
	}
	return advisoriesByPackage
}

func findVulnerablePackageWithoutNamespace(advisory *mvd.Advisory, installedPkg *mvd.Package) *mvd.Package {
	var match *mvd.Package
	for i := range advisory.Fixed {
		if advisory.Fixed[i].Name == installedPkg.Name || advisory.Fixed[i].Name == installedPkg.Origin {
			// This currently works under the assumption, that the highest version is the last one in the list
			// To not re-apply all the version comparison here, we ensure the ordering in the upstream data
			match = advisory.Fixed[i]
		}
	}
	return match
}
