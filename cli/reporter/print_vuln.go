// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"unicode/utf8"

	"github.com/muesli/termenv"
	"go.mondoo.com/mql/cli/theme"
	"go.mondoo.com/mql/cli/theme/colors"
	"go.mondoo.com/mql/providers-sdk/v1/upstream/fex"
)

// defaultVulnReporter renders the vulnerabilities returned by the PURL-native
// SBOM scan (VEX) as human-readable text. The rows are already flattened and
// sorted (most severe first) by fex.VulnRows.
type defaultVulnReporter struct {
	*Reporter
	isCompact bool
	isSummary bool
	target    string
	out       io.Writer
	rows      []fex.VulnRow
}

func (r *defaultVulnReporter) print() error {
	header := fmt.Sprintf("\nTarget:     %s\n", r.target)
	_, _ = r.out.Write([]byte(termenv.String(header).Foreground(theme.DefaultTheme.Colors.Primary).String()))
	summaryDivider := strings.Repeat("=", utf8.RuneCountInString(header))
	_, _ = r.out.Write([]byte(termenv.String(summaryDivider + "\n\n").Foreground(theme.DefaultTheme.Colors.Secondary).String()))

	_, _ = r.out.Write([]byte(RenderVulnRowsSummary(r.rows)))
	if !r.isSummary {
		_, _ = r.out.Write([]byte(RenderVulnRowsTable(r.rows, !r.isCompact)))
	}
	return nil
}

// severityCount tallies rows by their severity label.
type severityCount struct {
	Total    int `json:"total"`
	Critical int `json:"critical"`
	High     int `json:"high"`
	Medium   int `json:"medium"`
	Low      int `json:"low"`
	None     int `json:"none"`
}

func countSeverities(rows []fex.VulnRow) severityCount {
	var c severityCount
	for i := range rows {
		c.Total++
		switch rows[i].Severity {
		case fex.SeverityCritical:
			c.Critical++
		case fex.SeverityHigh:
			c.High++
		case fex.SeverityMedium:
			c.Medium++
		case fex.SeverityLow:
			c.Low++
		default:
			c.None++
		}
	}
	return c
}

// RenderVulnRowsSummary renders a one-line, colored severity breakdown for the
// VEX rows. It is safe to call with no rows.
func RenderVulnRowsSummary(rows []fex.VulnRow) string {
	c := countSeverities(rows)
	if c.Total == 0 {
		color := colors.DefaultColorTheme.Good
		return termenv.String("■ No vulnerabilities found").Foreground(color).String() + NewLineCharacter + NewLineCharacter
	}

	var b bytes.Buffer
	fmt.Fprintf(&b, "Vulnerabilities: %d", c.Total)
	b.WriteString("  (")
	parts := []struct {
		label string
		count int
		color termenv.Color
	}{
		{"Critical", c.Critical, colors.DefaultColorTheme.Critical},
		{"High", c.High, colors.DefaultColorTheme.High},
		{"Medium", c.Medium, colors.DefaultColorTheme.Medium},
		{"Low", c.Low, colors.DefaultColorTheme.Low},
		{"None", c.None, colors.DefaultColorTheme.Unknown},
	}
	first := true
	for _, p := range parts {
		if p.count == 0 {
			continue
		}
		if !first {
			b.WriteString("  ")
		}
		first = false
		b.WriteString(termenv.String(fmt.Sprintf("%s: %d", p.label, p.count)).Foreground(p.color).String())
	}
	b.WriteString(")")
	b.WriteString(NewLineCharacter + NewLineCharacter)
	return b.String()
}

// RenderVulnRowsTable renders the VEX rows as an aligned table. When detailed is
// true, the remediation hint (and, absent one, the first reference URL) is added
// as a trailing column. Cells are left uncolored so tab alignment stays correct.
func RenderVulnRowsTable(rows []fex.VulnRow, detailed bool) string {
	if len(rows) == 0 {
		return ""
	}

	var b bytes.Buffer
	tw := tabwriter.NewWriter(&b, 0, 2, 2, ' ', 0)

	header := []string{"SEVERITY", "ADVISORY", "PACKAGE", "INSTALLED", "FIXED"}
	if detailed {
		header = append(header, "REMEDIATION")
	}
	fmt.Fprintln(tw, strings.Join(header, "\t"))

	for i := range rows {
		row := rows[i]
		installed := row.AffectedVersion
		fixed := row.FixedVersion
		if fixed == "" {
			fixed = "-"
		}
		pkg := row.AffectedName
		if pkg == "" {
			pkg = "-"
		}
		cells := []string{severityOrNone(row.Severity), row.ID, pkg, orDash(installed), fixed}
		if detailed {
			cells = append(cells, orDash(remediationForRow(row)))
		}
		fmt.Fprintln(tw, strings.Join(cells, "\t"))
	}
	_ = tw.Flush()
	b.WriteString(NewLineCharacter)
	return b.String()
}

func severityOrNone(severity string) string {
	if severity == "" {
		return fex.SeverityNone
	}
	return severity
}

func orDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return s
}

// remediationForRow prefers the authored remediation hint, then falls back to the
// first reference URL, so the detailed column always carries actionable context
// when any is available.
func remediationForRow(row fex.VulnRow) string {
	if h := strings.TrimSpace(row.RemediationHint); h != "" {
		return h
	}
	if len(row.References) > 0 {
		return row.References[0]
	}
	return ""
}
