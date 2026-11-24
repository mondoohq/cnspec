// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Package ohdf converts an OASIS Heimdall Data Format (OHDF, formerly HDF)
// document into Mondoo FEX findings. OHDF is the normalized result schema of the
// MITRE Security Automation Framework and the schema InSpec emits as exec-json.
//
// It reaches further than one tool. The MITRE SAF CLI converts the output of
// roughly thirty scanners into OHDF, so a report that reaches this converter may
// have started as Nessus, Trivy, Prisma Cloud, Checkov, Snyk, Fortify, Veracode,
// an XCCDF result or a DISA checklist. An InSpec run needs no conversion at all.
//
// cnspec also writes this format (reports/hdf), so a cnspec scan exported to OHDF
// converts back through here.
package ohdf

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	rc "go.mondoo.com/cnspec/upload/report_conversion"
	"go.mondoo.com/mql/providers-sdk/v1/upstream/fex"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func init() { rc.Register("hdf", Convert) }

// The document types below carry only the fields this converter reads. They are
// deliberately separate from the types in reports/hdf, which describe what cnspec
// emits: a reader has to tolerate whatever the thirty-odd upstream converters
// produce, including absent sections and fields cnspec never writes, so the two
// sides have different obligations and sharing the structs would constrain both.
type document struct {
	Platform   platform  `json:"platform"`
	Version    string    `json:"version"`
	Profiles   []profile `json:"profiles"`
	Statistics struct {
		Duration *float64 `json:"duration"`
	} `json:"statistics"`
}

type platform struct {
	Name     string `json:"name"`
	Release  string `json:"release"`
	TargetID string `json:"target_id"`
}

type profile struct {
	Name     string    `json:"name"`
	Title    *string   `json:"title"`
	Version  *string   `json:"version"`
	Controls []control `json:"controls"`
}

type control struct {
	ID           string            `json:"id"`
	Title        *string           `json:"title"`
	Desc         *string           `json:"desc"`
	Descriptions []description     `json:"descriptions"`
	Impact       float64           `json:"impact"`
	Refs         []ref             `json:"refs"`
	Tags         map[string]any    `json:"tags"`
	Code         string            `json:"code"`
	Results      []result          `json:"results"`
	Source       map[string]string `json:"-"`
}

type description struct {
	Label string `json:"label"`
	Data  string `json:"data"`
}

type ref struct {
	Ref string `json:"ref"`
	URL string `json:"url"`
}

type result struct {
	Status      string `json:"status"`
	CodeDesc    string `json:"code_desc"`
	Message     string `json:"message"`
	SkipMessage string `json:"skip_message"`
	Resource    string `json:"resource"`
	StartTime   string `json:"start_time"`
}

// OHDF result statuses.
const (
	statusPassed  = "passed"
	statusFailed  = "failed"
	statusSkipped = "skipped"
	statusError   = "error"
)

// Convert parses an OHDF document and returns one FEX document per control that
// reports a problem.
//
// A control becomes a finding when at least one of its results failed or errored.
// Passing and skipped controls are dropped, matching the JUnit converter: a
// compliance profile is mostly passing checks, and importing those as findings
// would bury the ones that need attention. An impact of 0 marks a control the run
// deliberately excluded, which is not a finding either.
//
// The unit is the control rather than the result. A control carries the identity a
// finding needs (id, title, description, remediation, compliance tags) while its
// results are the evidence for one run, and a control that fails on many resources
// would otherwise repeat all of that per resource under ids that OHDF does not
// give it.
func Convert(data []byte) ([]*fex.FindingDocument, error) {
	var doc document
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse ohdf report: %w", err)
	}
	if len(doc.Profiles) == 0 {
		return nil, fmt.Errorf("ohdf report has no profiles")
	}

	affects := affectedComponent(doc.Platform)

	// Control ids are unique within a profile, not across a document: an overlay
	// profile restates the controls it amends, and merged reports repeat them
	// outright. The tally is keyed on the emitted id rather than on the profile,
	// so a repeat is caught wherever in the document it comes from.
	seen := map[string]int{}

	var docs []*fex.FindingDocument
	for i := range doc.Profiles {
		p := &doc.Profiles[i]
		source := &fex.Source{Name: profileSource(p)}
		for j := range p.Controls {
			c := &p.Controls[j]
			failing := failingResults(c)
			if len(failing) == 0 {
				continue
			}

			f := toFex(c, failing, source, affects)
			// The first control to claim an id keeps it; later ones get a stable
			// derived id so none of them overwrite another.
			key := f.Id
			if n := seen[key]; n > 0 {
				f.Id = shortHash(source.Name + "\x00" + key + "#" + strconv.Itoa(n))
			}
			seen[key]++
			docs = append(docs, fex.FexToDocument(f))
		}
	}
	return docs, nil
}

// failingResults returns the results that report a problem. A control whose impact
// is 0 reports none: OHDF reserves that value for a control the run excluded, and
// Heimdall renders it "Not Applicable".
func failingResults(c *control) []result {
	if c.Impact == 0 {
		return nil
	}

	var out []result
	for _, r := range c.Results {
		switch r.Status {
		case statusFailed, statusError:
			out = append(out, r)
		}
	}
	return out
}

func toFex(c *control, failing []result, source *fex.Source, affects []*fex.Affects) *fex.FindingExchange {
	title := deref(c.Title)
	summary := title
	if summary == "" {
		summary = c.ID
	}
	if summary == "" {
		summary = "Finding from " + source.Name
	}

	// Id must be stable and non-empty. The control id is the natural key; a
	// control without one falls back to a hash of what identifies it.
	id := c.ID
	if id == "" {
		id = shortHash(source.Name + "\x00" + summary)
	}

	f := &fex.FindingExchange{
		Id:      id,
		Ref:     c.ID,
		Summary: summary,
		Source:  source,
		Status:  status(failing),
		Details: &fex.FindingDetail{
			Category:    fex.FindingDetail_CATEGORY_SECURITY,
			Description: describe(c, failing),
			Severity:    severity(c.Impact),
			References:  references(c),
			Properties:  properties(c),
		},
		Affects: affects,
	}
	if ts := firstSeen(failing); ts != nil {
		f.FirstSeenAt = timestamppb.New(*ts)
	}
	if rem := remediations(c); len(rem) > 0 {
		f.Remediations = rem
	}
	return f
}

// status reports what the failing results mean. An errored check did not reach a
// verdict, so it is reported as still under investigation rather than as a
// confirmed failure.
func status(failing []result) fex.Status {
	for _, r := range failing {
		if r.Status == statusFailed {
			return fex.Status_STATUS_AFFECTED
		}
	}
	return fex.Status_STATUS_UNDER_INVESTIGATION
}

// severity maps the OHDF impact (0.0-1.0) onto a severity rating using the bands
// Heimdall and the SAF CLI apply. A control with no impact at all has already been
// dropped, so no rating means the document carried a value outside the scale.
func severity(impact float64) *fex.Severity {
	var rating fex.SeverityRating
	switch {
	case impact > 1:
		return nil
	case impact >= 0.9:
		rating = fex.SeverityRating_SEVERITY_RATING_CRITICAL
	case impact >= 0.7:
		rating = fex.SeverityRating_SEVERITY_RATING_HIGH
	case impact >= 0.4:
		rating = fex.SeverityRating_SEVERITY_RATING_MEDIUM
	case impact >= 0.1:
		rating = fex.SeverityRating_SEVERITY_RATING_LOW
	case impact > 0:
		rating = fex.SeverityRating_SEVERITY_RATING_NONE
	default:
		return nil
	}
	return &fex.Severity{Rating: rating}
}

// describe renders what the check is and how it failed. The control's own
// description says what it checks; the failing results say what was found, which
// is the part that differs between two runs of the same control.
func describe(c *control, failing []result) string {
	var b strings.Builder
	if desc := strings.TrimSpace(deref(c.Desc)); desc != "" {
		b.WriteString(desc)
	}

	if check := descriptionData(c, "check"); check != "" {
		writeSection(&b, "Check", check)
	}
	if code := strings.TrimSpace(c.Code); code != "" {
		writeSection(&b, "Code", code)
	}

	var results []string
	for _, r := range failing {
		if line := resultLine(r); line != "" {
			results = append(results, line)
		}
	}
	if len(results) > 0 {
		writeSection(&b, "Results", strings.Join(results, "\n"))
	}
	return b.String()
}

// resultLine renders one failing result: what was tested, and what happened.
func resultLine(r result) string {
	parts := make([]string, 0, 3)
	if desc := strings.TrimSpace(r.CodeDesc); desc != "" {
		parts = append(parts, desc)
	}
	if msg := strings.TrimSpace(r.Message); msg != "" {
		parts = append(parts, msg)
	}
	if len(parts) == 0 && r.Status == statusError {
		// An errored result often carries neither, so say that it errored rather
		// than dropping it and leaving the finding with no evidence at all.
		parts = append(parts, "check errored")
	}
	return strings.Join(parts, ": ")
}

func writeSection(b *strings.Builder, title, body string) {
	if b.Len() > 0 {
		b.WriteString("\n\n")
	}
	b.WriteString(title)
	b.WriteString(":\n")
	b.WriteString(body)
}

// remediations carries the control's fix text. OHDF labels it in the descriptions
// array, which is where every SAF converter puts it.
func remediations(c *control) []*fex.Remediation {
	fix := descriptionData(c, "fix")
	if fix == "" {
		return nil
	}
	return []*fex.Remediation{{
		Category: fex.Remediation_Fix,
		Summary:  fix,
	}}
}

// descriptionData returns the first description carrying a label, comparing case
// insensitively since converters disagree on casing.
func descriptionData(c *control, label string) string {
	for _, d := range c.Descriptions {
		if strings.EqualFold(d.Label, label) {
			if data := strings.TrimSpace(d.Data); data != "" {
				return data
			}
		}
	}
	return ""
}

func references(c *control) []*fex.Reference {
	var out []*fex.Reference
	for _, r := range c.Refs {
		url := strings.TrimSpace(r.URL)
		if url == "" {
			continue
		}
		name := strings.TrimSpace(r.Ref)
		if name == "" {
			name = url
		}
		out = append(out, &fex.Reference{Name: name, Url: url})
	}
	return out
}

// properties carries the control's tags. The compliance mappings live there -
// `nist` in particular, which is what the SAF ecosystem keys its 800-53 views off -
// and they are the part of an OHDF control that has no home elsewhere in FEX.
func properties(c *control) map[string]string {
	if len(c.Tags) == 0 {
		return nil
	}

	out := make(map[string]string, len(c.Tags))
	for k, v := range c.Tags {
		if s := tagValue(v); s != "" {
			out[k] = s
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// tagValue flattens a tag. OHDF tags are free-form JSON: `nist` is a list of
// control ids, `severity` a string, and converters also emit numbers and booleans.
func tagValue(v any) string {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case bool:
		return strconv.FormatBool(t)
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case []any:
		parts := make([]string, 0, len(t))
		for _, item := range t {
			if s := tagValue(item); s != "" {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, ", ")
	case map[string]any:
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		parts := make([]string, 0, len(keys))
		for _, k := range keys {
			if s := tagValue(t[k]); s != "" {
				parts = append(parts, k+"="+s)
			}
		}
		return strings.Join(parts, ", ")
	default:
		return ""
	}
}

// affectedComponent identifies what was scanned. OHDF describes a single target
// per document, and `target_id` is whatever the producing tool chose to call it -
// a hostname, an image digest, an account id - so it is carried as the component
// id rather than parsed into a shape it may not have.
func affectedComponent(p platform) []*fex.Affects {
	id := strings.TrimSpace(p.TargetID)
	if id == "" {
		id = strings.TrimSpace(p.Name)
	}
	if id == "" {
		return nil
	}

	identifiers := map[string]string{}
	if name := strings.TrimSpace(p.Name); name != "" {
		identifiers["platform"] = name
	}
	if release := strings.TrimSpace(p.Release); release != "" {
		identifiers["release"] = release
	}
	if target := strings.TrimSpace(p.TargetID); target != "" {
		identifiers["target_id"] = target
	}

	return []*fex.Affects{{Component: &fex.Component{
		Id:          id,
		Identifiers: identifiers,
	}}}
}

// profileSource names the tool that produced the findings. The profile title is
// what a converter sets to the scanner's name; the profile name is InSpec's own
// identifier and is the fallback.
func profileSource(p *profile) string {
	if title := strings.TrimSpace(deref(p.Title)); title != "" {
		return title
	}
	if name := strings.TrimSpace(p.Name); name != "" {
		return name
	}
	return "OHDF"
}

// firstSeen is the earliest time any failing result ran, so re-uploading an old
// report does not reset first-seen to the upload time. It is left unset when the
// document carries no parsable timestamp.
func firstSeen(failing []result) *time.Time {
	var earliest *time.Time
	for _, r := range failing {
		raw := strings.TrimSpace(r.StartTime)
		if raw == "" {
			continue
		}
		ts, err := parseTime(raw)
		if err != nil {
			continue
		}
		if earliest == nil || ts.Before(*earliest) {
			t := ts
			earliest = &t
		}
	}
	return earliest
}

// parseTime accepts the timestamp shapes OHDF documents carry in practice. InSpec
// writes a Ruby time, and converters emit RFC 3339 with and without a zone.
func parseTime(raw string) (time.Time, error) {
	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05 -0700",
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		if ts, err := time.Parse(layout, raw); err == nil {
			return ts.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized timestamp %q", raw)
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func shortHash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])[:16]
}
