// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"encoding/json"

	"go.mondoo.com/mql/providers-sdk/v1/upstream/fex"
	"go.mondoo.com/mql/utils/iox"
)

// vulnJSONRow is the serialized form of a single VEX vulnerability row. It
// carries strictly more than the legacy package view — the PURL, the resolved
// severity label, reference URLs, and a remediation hint — while preserving the
// package/installed/fixed columns of the previous output.
type vulnJSONRow struct {
	Advisory    string   `json:"advisory"`
	Severity    string   `json:"severity"`
	Package     string   `json:"package"`
	Installed   string   `json:"installed"`
	Fixed       string   `json:"fixed,omitempty"`
	Purl        string   `json:"purl,omitempty"`
	Summary     string   `json:"summary,omitempty"`
	References  []string `json:"references,omitempty"`
	Remediation string   `json:"remediation,omitempty"`
}

// vulnJSONReport is the top-level document emitted for the JSON/YAML formats.
type vulnJSONReport struct {
	Target          string        `json:"target"`
	Stats           severityCount `json:"stats"`
	Vulnerabilities []vulnJSONRow `json:"vulnerabilities"`
}

// VulnReportToJSON renders the VEX rows for a target as JSON. The YAML formats
// are derived from this output by the caller.
func VulnReportToJSON(target string, rows []fex.VulnRow, out iox.OutputHelper) error {
	doc := vulnJSONReport{
		Target:          target,
		Stats:           countSeverities(rows),
		Vulnerabilities: make([]vulnJSONRow, 0, len(rows)),
	}
	for i := range rows {
		row := rows[i]
		doc.Vulnerabilities = append(doc.Vulnerabilities, vulnJSONRow{
			Advisory:    row.ID,
			Severity:    severityOrNone(row.Severity),
			Package:     row.AffectedName,
			Installed:   row.AffectedVersion,
			Fixed:       row.FixedVersion,
			Purl:        row.AffectedPurl,
			Summary:     row.Summary,
			References:  row.References,
			Remediation: row.RemediationHint,
		})
	}

	data, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	return out.WriteString(string(data))
}
