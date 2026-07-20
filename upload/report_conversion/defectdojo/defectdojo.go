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
	"crypto/sha256"
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
	Title            string  `json:"title"`
	Severity         string  `json:"severity"`
	Description      string  `json:"description"`
	Mitigation       string  `json:"mitigation"`
	Impact           string  `json:"impact"`
	CVE              string  `json:"cve"`
	CWE              int     `json:"cwe"`
	CVSSv3           string  `json:"cvssv3"`       // CVSSv3 vector string
	CVSSv3Score      float64 `json:"cvssv3_score"` // CVSSv3 base score
	FilePath         string  `json:"file_path"`
	Line             int     `json:"line"`
	ComponentName    string  `json:"component_name"`
	ComponentVersion string  `json:"component_version"`
	UniqueIDFromTool string  `json:"unique_id_from_tool"`
	References       string  `json:"references"`
	Date             string  `json:"date"` // RFC3339 or YYYY-MM-DD

	// Status flags. DefectDojo emits these as booleans; they are pointers so we
	// can tell "absent" from an explicit false (Active defaults to true in
	// DefectDojo). See status() for the mapping to fex.Status.
	Active       *bool `json:"active"`
	Verified     *bool `json:"verified"`
	FalseP       *bool `json:"false_p"`
	OutOfScope   *bool `json:"out_of_scope"`
	RiskAccepted *bool `json:"risk_accepted"`
	Duplicate    *bool `json:"duplicate"`
	IsMitigated  *bool `json:"is_mitigated"`
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
		// Skip fully blank rows (common as trailing empty CSV lines).
		if f.Title == "" && f.Severity == "" && f.Description == "" {
			continue
		}
		if f.Title == "" || f.Severity == "" || f.Description == "" {
			return nil, fmt.Errorf("finding %d: title, severity, and description are required", i)
		}
		if f.CVE != "" {
			docs = append(docs, fex.VexToDocument(toVex(f, source)))
		} else {
			docs = append(docs, fex.FexToDocument(toFex(f, source)))
		}
	}
	return docs, nil
}

// parse auto-detects JSON vs CSV and returns the source name and findings.
func parse(data []byte) (string, []finding, error) {
	trimmed := bytes.TrimSpace(data)
	switch {
	case len(trimmed) > 0 && trimmed[0] == '{':
		var r report
		if err := json.Unmarshal(data, &r); err != nil {
			return "", nil, fmt.Errorf("parse DefectDojo JSON: %w", err)
		}
		return sourceName(r.Name), r.Findings, nil
	case len(trimmed) > 0 && trimmed[0] == '[':
		// DefectDojo's Generic Findings Import uses a {"findings": [...]} object,
		// not a bare array — give a clear error instead of failing as CSV.
		return "", nil, fmt.Errorf("expected a DefectDojo JSON object {\"findings\": [...]}, got a JSON array")
	default:
		findings, err := parseCSV(data)
		if err != nil {
			return "", nil, err
		}
		return "defectdojo", findings, nil
	}
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
		// DefectDojo's CSV format defines TitleCase columns; component_name /
		// file_path are JSON-only fields and are intentionally not read here.
		f := finding{
			Title:       get(row, "Title"),
			Severity:    get(row, "Severity"),
			Description: get(row, "Description"),
			Mitigation:  get(row, "Mitigation"),
			Impact:      get(row, "Impact"),
			CVE:         get(row, "CVE"),
			CVSSv3:      firstNonEmpty(get(row, "CVSSV3"), get(row, "CvssV3")),
			References:  firstNonEmpty(get(row, "References"), get(row, "Url")),
			Date:        get(row, "Date"),
			// Status columns (TitleCase in the DefectDojo CSV export). Header
			// names vary slightly between exports, so accept a couple variants.
			Active:       parseCSVBool(firstNonEmpty(get(row, "Active"))),
			Verified:     parseCSVBool(firstNonEmpty(get(row, "Verified"))),
			FalseP:       parseCSVBool(firstNonEmpty(get(row, "FalsePositive"), get(row, "False Positive"))),
			OutOfScope:   parseCSVBool(firstNonEmpty(get(row, "OutOfScope"), get(row, "Out Of Scope"))),
			RiskAccepted: parseCSVBool(firstNonEmpty(get(row, "RiskAccepted"), get(row, "Risk Accepted"))),
			Duplicate:    parseCSVBool(firstNonEmpty(get(row, "Duplicate"))),
			IsMitigated:  parseCSVBool(firstNonEmpty(get(row, "IsMitigated"), get(row, "Is Mitigated"))),
		}
		if cwe := get(row, "CweId"); cwe != "" {
			f.CWE, _ = strconv.Atoi(cwe)
		}
		if s := firstNonEmpty(get(row, "CVSSV3 Score"), get(row, "CvssV3Score")); s != "" {
			f.CVSSv3Score, _ = strconv.ParseFloat(s, 64)
		}
		findings = append(findings, f)
	}
	return findings, nil
}

func toFex(f finding, source *fex.Source) *fex.FindingExchange {
	id := f.UniqueIDFromTool
	if id == "" {
		// Content-based id so it's stable across reorderings of the input. Include
		// the file/line/component so otherwise-identical findings at different
		// locations do not collapse onto the same id.
		id = shortHash(strings.Join([]string{
			f.Title, f.Description, f.FilePath, strconv.Itoa(f.Line),
			f.ComponentName, f.ComponentVersion,
		}, "\x00"))
	}
	fx := &fex.FindingExchange{
		Id:      id,
		Ref:     f.UniqueIDFromTool,
		Summary: f.Title,
		Source:  source,
		Status:  status(f),
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
		Status:  status(f),
		Details: &fex.VulnerabilityDetails{
			Details:        joinDetail(f),
			Recommendation: f.Mitigation,
		},
		Affects:    affects(f),
		Ratings:    ratings(f),
		References: references(f),
	}
	if ts := parseTime(f.Date); ts != nil {
		vx.FirstSeen = ts
	}
	return vx
}

// status maps the DefectDojo status booleans onto fex.Status. Checked in order
// of precedence; the same mapping is used for both FEX and VEX documents.
func status(f finding) fex.Status {
	switch {
	case boolVal(f.FalseP):
		return fex.Status_STATUS_FALSE_POSITIVE
	case boolVal(f.RiskAccepted):
		return fex.Status_STATUS_WONT_FIX // accepted risk
	case boolVal(f.IsMitigated):
		return fex.Status_STATUS_FIXED
	case f.Active != nil && !*f.Active:
		// Explicitly inactive: DefectDojo marks a finding inactive once it is no
		// longer present. Active defaults to true, so an absent flag stays affected.
		return fex.Status_STATUS_FIXED
	case boolVal(f.OutOfScope):
		return fex.Status_STATUS_NOT_AFFECTED
	default:
		return fex.Status_STATUS_AFFECTED
	}
}

// ratings builds a CVSSv3 rating from the severity mapping and any CVSS score or
// vector present in the finding. Returns nil when there is nothing to report.
func ratings(f finding) []*fex.Rating {
	r := &fex.Rating{}
	has := false
	if severity(f.Severity) != nil {
		r.Severity = strings.ToLower(strings.TrimSpace(f.Severity))
		has = true
	}
	if f.CVSSv3Score > 0 {
		r.Score = float32(f.CVSSv3Score)
		r.Method = fex.ScoringMethod_SCOREMETHOD_CVSSv3
		has = true
	}
	if v := strings.TrimSpace(f.CVSSv3); v != "" {
		r.Vector = v
		r.Method = fex.ScoringMethod_SCOREMETHOD_CVSSv3
		has = true
	}
	if !has {
		return nil
	}
	return []*fex.Rating{r}
}

func boolVal(b *bool) bool {
	return b != nil && *b
}

// parseCSVBool parses a CSV cell into an optional bool. Empty cells stay nil so
// an absent column does not read as false.
func parseCSVBool(s string) *bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	b, err := strconv.ParseBool(strings.ToLower(s))
	if err != nil {
		return nil
	}
	return &b
}

func affects(f finding) []*fex.Affects {
	// Build the file location (path + optional 1-based start line) up front so it
	// can be attached whether or not a component name is also present.
	var file *fex.FileComponent
	if f.FilePath != "" {
		file = &fex.FileComponent{Path: f.FilePath}
		if f.Line > 0 {
			file.StartLine = int32(f.Line)
		}
	}

	switch {
	case f.ComponentName != "":
		ids := map[string]string{}
		if f.ComponentVersion != "" {
			ids["version"] = f.ComponentVersion
		}
		comp := &fex.Component{Id: f.ComponentName, Identifiers: ids}
		// Represent both the component and the file location when we have both.
		if file != nil {
			comp.Details = &fex.Component_File{File: file}
		}
		return []*fex.Affects{{Component: comp}}
	case file != nil:
		return []*fex.Affects{{Component: &fex.Component{
			Id:      f.FilePath,
			Details: &fex.Component_File{File: file},
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

func shortHash(s string) string {
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h)[:16]
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
