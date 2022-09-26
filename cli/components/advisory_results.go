package components

import (
	"fmt"
	"io"
	"net/url"
	"sort"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/olekukonko/tablewriter"
	"go.mondoo.com/cnquery/resources/packs/core/vadvisor"
	cvss_proto "go.mondoo.com/cnquery/resources/packs/core/vadvisor/cvss"
)

func NewAdvisoryResultTable() AdvisoryResultTable {
	return AdvisoryResultTable{
		DetailedPackageRisks: false,
		ScoreAscending:       false,
	}
}

// AdvisoryResultTable is a component to help print a list of vulnerable packages
type AdvisoryResultTable struct {
	DetailedPackageRisks bool
	ScoreAscending       bool
}

// renderReportCli prints one vuln report on CLI
func (a AdvisoryResultTable) Render(r *vadvisor.VulnReport) (string, error) {
	b := &strings.Builder{}
	if r == nil {
		return "", errors.New("report cannot be empty")
	}
	indicator := NewCvssIndicator()
	// severity := r.Severity()

	// platform advisories
	platformAdvisory := []*vadvisor.Advisory{}
	for i := range r.Advisories {
		advisory := r.Advisories[i]
		if len(advisory.FixedPlatforms) == 0 {
			continue
		}
		platformAdvisory = append(platformAdvisory, advisory)
	}

	// TODO: double-iteration
	if len(platformAdvisory) > 0 {
		// sort advisories by score
		if a.ScoreAscending {
			sort.Sort(vadvisor.BySeverity(platformAdvisory))
		} else {
			sort.Sort(sort.Reverse(vadvisor.BySeverity(platformAdvisory)))
		}

		// render platform advisories
		table := tablewriter.NewWriter(b)
		table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
		table.SetBorder(false)
		table.SetHeaderLine(false)
		table.SetRowLine(false)
		table.SetColumnSeparator("")
		table.SetAlignment(tablewriter.ALIGN_LEFT)

		header := []string{"■", "score", "advisory", "current", "fixed", "patch"}
		table.SetHeader(header)

		for i := range platformAdvisory {
			advisory := platformAdvisory[i]
			score := IntScore2Float(advisory.Score)
			severity := cvss_proto.Rating(score)
			icon := indicator.Render(severity)

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

			line := []string{icon, fmt.Sprintf("%v", score), advisory.ID, currentVersion, fixedVersion, patch}
			table.Append(line)
		}
		table.Render()
	}

	// packages advisories
	if r.Stats != nil && r.Stats.Packages != nil && r.Stats.Packages.Affected > 0 {
		// FIXME: the remaining code here needs to be ported
		// reportWriter := NewCliTableWriter(b, a.DetailedPackageRisks)
		// err := r.RenderReport(reportWriter, vadvisor.RowWriterOpts{
		// 	AdvisoryDetails: a.DetailedPackageRisks,
		// 	ScoreAscending:  a.ScoreAscending,
		// })
		// if err != nil {
		// 	return "", err
		// }
	}

	return b.String(), nil
}

func NewCliTableWriter(writer io.Writer, detailedPackageRisks bool) *CliTableWriter {
	table := tablewriter.NewWriter(writer)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetBorder(false)
	table.SetHeaderLine(false)
	table.SetRowLine(false)
	table.SetColumnSeparator("")
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	// table.SetAutoMergeCells(true)

	return &CliTableWriter{
		table:                table,
		maxSamePkg:           0,
		detailedPackageRisks: detailedPackageRisks,
	}
}

type CliTableWriter struct {
	table                *tablewriter.Table
	lastEntry            *ReportFindingRow
	pkgCount             int
	maxSamePkg           int
	detailedPackageRisks bool
}

func (c *CliTableWriter) WriteHeader() error {
	header := []string{"■", "score", "package", "installed", "fixed", "available"}

	if c.detailedPackageRisks {
		header = append(header, "advisory")
	}

	c.table.SetHeader(header)
	return nil
}

func (c *CliTableWriter) renderRow(row *ReportFindingRow, overrideIndicator string) {
	if row == nil {
		return
	}

	icon := ""

	score := IntScore2Float(row.Score)
	severity := cvss_proto.Rating(score)
	// eg. if no cve was attached to the advisory the score will be -1
	reportScore := ""
	if score >= float32(0.0) {
		reportScore = fmt.Sprintf("%v", score)

		icon = NewCvssIndicator().Render(severity)
		if len(overrideIndicator) > 0 {
			icon = overrideIndicator
		}
	}

	record := []string{
		reportScore,
		row.Name,
		// installed package
		row.Installed,
		// fixed packages
		row.Fixed,
		row.Available,
	}

	if c.detailedPackageRisks {
		record = append(record, row.Advisory)
	}

	line := append([]string{icon}, record...)

	c.table.Append(line)
}

func (c *CliTableWriter) Write(row ReportFindingRow) error {
	lastIcon := ""
	if len(row.Advisory) > 0 {
		row.Advisory = getAdvisoryUrl(row.Advisory)
	}

	// render previous entry
	if c.lastEntry != nil {
		// determine the icon for the previous entry
		if c.lastEntry.Name == row.Name {
			if c.pkgCount > 1 {
				lastIcon = "├─"
			}

			// print previous row with the same package name, only if we have not reached the max limit
			if c.maxSamePkg == 0 || c.pkgCount < c.maxSamePkg {
				c.renderRow(c.lastEntry, lastIcon)
			}

			c.pkgCount++
			c.lastEntry = &row
		} else {
			// once the name switched, the icon for the previous one is the last
			if c.pkgCount > 1 {
				lastIcon = "╰─"
			}

			if c.maxSamePkg > 1 {
				// check if we got more items than we rendered
				if c.maxSamePkg > 0 && c.pkgCount > c.maxSamePkg {
					c.lastEntry = &ReportFindingRow{
						Score:     c.lastEntry.Score,
						Name:      c.lastEntry.Name,
						Installed: c.lastEntry.Installed,
						Available: "",
						Fixed:     "...",
						Advisory:  fmt.Sprintf("%d more advisories", c.pkgCount-c.maxSamePkg),
					}
				}
			}

			// print previous row
			c.renderRow(c.lastEntry, lastIcon)

			// reset counter
			c.pkgCount = 1
			c.lastEntry = &row
		}
	} else {
		// only the very first entry should reach here
		// we do not render the entry immediatly to determine the row indicator
		c.pkgCount = 1
		c.lastEntry = &row
	}

	return nil
}

func (c *CliTableWriter) Flush() {
	// we need to print the last row in cache
	if c.lastEntry != nil {
		lastIcon := ""
		if c.pkgCount > 1 {
			lastIcon = "╰─"
		}
		c.renderRow(c.lastEntry, lastIcon)
	}

	// and render the table
	c.table.Render()
}

func IntScore2Float(score int32) float32 {
	return float32(score) / 10
}

// TODO: remove hardcoded old references
func getAdvisoryUrl(id string) string {
	path := url.QueryEscape(id)
	return fmt.Sprintf("https://console.mondoo.com/vuln/%s", path)
}
