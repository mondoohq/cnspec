// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"encoding/csv"
	"strings"

	"go.mondoo.com/mql/providers-sdk/v1/upstream/fex"
	"go.mondoo.com/mql/utils/iox"
)

type csvStruct struct {
	Severity    string
	Advisory    string
	Package     string
	Installed   string
	Fixed       string
	Purl        string
	References  string
	Remediation string
}

func (c csvStruct) toSlice() []string {
	return []string{c.Severity, c.Advisory, c.Package, c.Installed, c.Fixed, c.Purl, c.References, c.Remediation}
}

// VulnReportToCSV writes the VEX rows for a target as CSV.
func VulnReportToCSV(rows []fex.VulnRow, out iox.OutputHelper) error {
	w := csv.NewWriter(out)

	// write header
	err := w.Write(csvStruct{
		Severity:    "Severity",
		Advisory:    "Advisory",
		Package:     "Package",
		Installed:   "Installed",
		Fixed:       "Fixed",
		Purl:        "PURL",
		References:  "References",
		Remediation: "Remediation",
	}.toSlice())
	if err != nil {
		return err
	}

	for i := range rows {
		row := rows[i]
		rec := csvStruct{
			Severity:    severityOrNone(row.Severity),
			Advisory:    row.ID,
			Package:     row.AffectedName,
			Installed:   row.AffectedVersion,
			Fixed:       row.FixedVersion,
			Purl:        row.AffectedPurl,
			References:  strings.Join(row.References, " "),
			Remediation: row.RemediationHint,
		}
		if err := w.Write(escapeCSVRow(rec.toSlice())); err != nil {
			return err
		}
	}

	w.Flush()
	return w.Error()
}

// escapeCSVRow neutralizes any cell that a spreadsheet application would
// interpret as a formula. Package fields come from the scanned target, so a
// value such as "=HYPERLINK(...)" must not execute when the report is opened in
// Excel, LibreOffice, or Sheets. See OWASP "CSV Injection".
func escapeCSVRow(row []string) []string {
	for i := range row {
		row[i] = escapeCSVCell(row[i])
	}
	return row
}

func escapeCSVCell(value string) string {
	if value == "" {
		return value
	}
	switch value[0] {
	case '=', '+', '-', '@', '\t', '\r':
		return "'" + value
	}
	return value
}
