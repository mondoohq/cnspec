// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Package defectdojo converts an OWASP DefectDojo "Generic Findings Import"
// report into Mondoo FEX/VEX. It accepts both encodings of that format: JSON (an
// object with a "findings" array) and CSV (a header row with Title, Description,
// Severity, Date, …). Each finding requires a title, severity, and description.
// This lets manual/pentest findings and arbitrary tools be imported without a
// tool-specific converter.
//
// Format reference: https://documentation.defectdojo.com (Generic Findings Import).
package defectdojo

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	rc "go.mondoo.com/cnspec/v13/upload/report_conversion"
	"go.mondoo.com/mql/v13/providers-sdk/v1/upstream/fex"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func init() { rc.Register("defectdojo", Convert) }

type report struct {
	Name     string    `json:"name"`
	Findings []finding `json:"findings"`
}

type finding struct {
	Title            string `json:"title"`
	Severity         string `json:"severity"`
	Description      string `json:"description"`
	Mitigation       string `json:"mitigation"`
	Impact           string `json:"impact"`
	CVE              string `json:"cve"`
	CWE              int    `json:"cwe"`
	FilePath         string `json:"file_path"`
	Line             int    `json:"line"`
	ComponentName    string `json:"component_name"`
	ComponentVersion string `json:"component_version"`
	UniqueIDFromTool string `json:"unique_id_from_tool"`
	References       string `json:"references"`
	Date             string `json:"date"` // RFC3339 or YYYY-MM-DD
}

// Convert parses an OWASP DefectDojo Generic Findings Import report (JSON or CSV,
// auto-detected) and returns one document per finding. A finding with a CVE
// becomes VEX; everything else becomes FEX.
func Convert(data []byte) ([]*fex.FindingDocument, error) {
	name, findings, err := parse(data)
	if err != nil {
		return nil, err
	}
	source := &fex.Source{Name: name}

	docs := make([]*fex.FindingDocument, 0, len(findings))
	for i, f := range findings {
		if f.Title == "" || f.Severity == "" || f.Description == "" {
			return nil, fmt.Errorf("finding %d: title, severity, and description are required", i)
		}
		if f.CVE != "" {
			docs = append(docs, fex.VexToDocument(toVex(f, source)))
		} else {
			docs = append(docs, fex.FexToDocument(toFex(f, source, i)))
		}
	}
	return docs, nil
}

// parse auto-detects JSON vs CSV and returns the source name and findings.
func parse(data []byte) (string, []finding, error) {
	if trimmed := bytes.TrimSpace(data); len(trimmed) > 0 && trimmed[0] == '{' {
		var r report
		if err := json.Unmarshal(data, &r); err != nil {
			return "", nil, fmt.Errorf("parse DefectDojo JSON: %w", err)
		}
		return sourceName(r.Name), r.Findings, nil
	}
	findings, err := parseCSV(data)
	if err != nil {
		return "", nil, err
	}
	return "defectdojo", findings, nil
}

// parseCSV maps the DefectDojo CSV columns (by header name) onto findings.
func parseCSV(data []byte) ([]finding, error) {
	r := csv.NewReader(bytes.NewReader(data))
	r.FieldsPerRecord = -1 // rows may carry different optional-column counts
	header, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("parse DefectDojo CSV header: %w", err)
	}
	col := map[string]int{}
	for i, h := range header {
		col[strings.TrimSpace(h)] = i
	}
	get := func(row []string, name string) string {
		if i, ok := col[name]; ok && i < len(row) {
			return strings.TrimSpace(row[i])
		}
		return ""
	}

	var findings []finding
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("parse DefectDojo CSV row: %w", err)
		}
		f := finding{
			Title:            get(row, "Title"),
			Severity:         get(row, "Severity"),
			Description:      get(row, "Description"),
			Mitigation:       get(row, "Mitigation"),
			Impact:           get(row, "Impact"),
			CVE:              get(row, "CVE"),
			References:       firstNonEmpty(get(row, "References"), get(row, "Url")),
			Date:             get(row, "Date"),
			ComponentName:    get(row, "component_name"),
			ComponentVersion: get(row, "component_version"),
			FilePath:         get(row, "file_path"),
		}
		if cwe := get(row, "CweId"); cwe != "" {
			f.CWE, _ = strconv.Atoi(cwe)
		}
		findings = append(findings, f)
	}
	return findings, nil
}

func toFex(f finding, source *fex.Source, index int) *fex.FindingExchange {
	id := f.UniqueIDFromTool
	if id == "" {
		id = fmt.Sprintf("finding-%d", index)
	}
	fx := &fex.FindingExchange{
		Id:      id,
		Ref:     f.UniqueIDFromTool,
		Summary: f.Title,
		Source:  source,
		Status:  fex.Status_STATUS_AFFECTED,
		Details: &fex.FindingDetail{
			Category:    fex.FindingDetail_CATEGORY_SECURITY,
			Description: joinDetail(f),
			Severity:    severity(f.Severity),
			References:  references(f),
		},
		Affects: affects(f),
	}
	if ts := parseTime(f.Date); ts != nil {
		fx.FirstSeenAt = ts
	}
	if r := remediation(f); r != nil {
		fx.Remediations = []*fex.Remediation{r}
	}
	return fx
}

func toVex(f finding, source *fex.Source) *fex.VulnerabilityExchange {
	vx := &fex.VulnerabilityExchange{
		Id:      f.CVE,
		Ref:     f.UniqueIDFromTool,
		Summary: f.Title,
		Source:  source,
		Status:  fex.Status_STATUS_AFFECTED,
		Details: &fex.VulnerabilityDetails{
			Details:        joinDetail(f),
			Recommendation: f.Mitigation,
		},
		Affects: affects(f),
	}
	if ts := parseTime(f.Date); ts != nil {
		vx.FirstSeen = ts
	}
	return vx
}

func affects(f finding) []*fex.Affects {
	switch {
	case f.ComponentName != "":
		ids := map[string]string{}
		if f.ComponentVersion != "" {
			ids["version"] = f.ComponentVersion
		}
		return []*fex.Affects{{Component: &fex.Component{Id: f.ComponentName, Identifiers: ids}}}
	case f.FilePath != "":
		return []*fex.Affects{{Component: &fex.Component{
			Id:      f.FilePath,
			Details: &fex.Component_File{File: &fex.FileComponent{Path: f.FilePath}},
		}}}
	default:
		return nil
	}
}

func references(f finding) []*fex.Reference {
	var out []*fex.Reference
	if f.CWE > 0 {
		out = append(out, &fex.Reference{Type: "CWE", Name: fmt.Sprintf("CWE-%d", f.CWE)})
	}
	if f.References != "" {
		out = append(out, &fex.Reference{Name: "reference", Url: f.References})
	}
	return out
}

func remediation(f finding) *fex.Remediation {
	if f.Mitigation == "" {
		return nil
	}
	return &fex.Remediation{Summary: "Mitigation", Details: f.Mitigation}
}

// joinDetail combines description and impact into the detail text.
func joinDetail(f finding) string {
	if f.Impact == "" {
		return f.Description
	}
	return f.Description + "\n\nImpact: " + f.Impact
}

func severity(s string) *fex.Severity {
	var rating fex.SeverityRating
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "critical":
		rating = fex.SeverityRating_SEVERITY_RATING_CRITICAL
	case "high":
		rating = fex.SeverityRating_SEVERITY_RATING_HIGH
	case "medium":
		rating = fex.SeverityRating_SEVERITY_RATING_MEDIUM
	case "low":
		rating = fex.SeverityRating_SEVERITY_RATING_LOW
	case "info", "informational", "none":
		rating = fex.SeverityRating_SEVERITY_RATING_NONE
	default:
		return nil
	}
	return &fex.Severity{Rating: rating}
}

func sourceName(name string) string {
	if name == "" {
		return "defectdojo"
	}
	return name
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func parseTime(s string) *timestamppb.Timestamp {
	if s == "" {
		return nil
	}
	for _, layout := range []string{"2006-01-02", time.RFC3339} {
		if t, err := time.Parse(layout, s); err == nil {
			return timestamppb.New(t)
		}
	}
	return nil
}
