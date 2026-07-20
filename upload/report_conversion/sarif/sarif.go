// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Package sarif converts SARIF (Static Analysis Results Interchange Format)
// reports into Mondoo FEX findings. SARIF is an open standard emitted by many
// SAST/IaC tools, so this one converter reaches a broad set of scanners.
package sarif

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"time"

	gosarif "github.com/owenrumney/go-sarif/v2/sarif"
	rc "go.mondoo.com/cnspec/v13/upload/report_conversion"
	"go.mondoo.com/mql/v13/providers-sdk/v1/upstream/fex"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func init() { rc.Register("sarif", Convert) }

// Convert parses a SARIF report (JSON) and returns one FEX document per result.
func Convert(data []byte) ([]*fex.FindingDocument, error) {
	report, err := gosarif.FromBytes(data)
	if err != nil {
		return nil, fmt.Errorf("parse sarif report: %w", err)
	}

	var docs []*fex.FindingDocument
	index := 0
	for _, run := range report.Runs {
		source := &fex.Source{Name: run.Tool.Driver.Name}
		runTime := runStartTime(run)
		rules := ruleIndex(run)
		for _, result := range run.Results {
			docs = append(docs, fex.FexToDocument(convertResult(result, source, runTime, rules, index)))
			index++
		}
	}
	return docs, nil
}

// ruleIndex maps a run's rule definitions by id so results can be enriched with
// the rule's description, help URI, taxa (CWE) and default severity.
func ruleIndex(run *gosarif.Run) map[string]*gosarif.ReportingDescriptor {
	out := map[string]*gosarif.ReportingDescriptor{}
	if run.Tool.Driver != nil {
		for _, r := range run.Tool.Driver.Rules {
			if r != nil {
				out[r.ID] = r
			}
		}
	}
	for _, ext := range run.Tool.Extensions {
		for _, r := range ext.Rules {
			if r != nil {
				out[r.ID] = r
			}
		}
	}
	return out
}

func convertResult(result *gosarif.Result, source *fex.Source, runTime *time.Time, rules map[string]*gosarif.ReportingDescriptor, index int) *fex.FindingExchange {
	ruleID := deref(result.RuleID)
	rule := rules[ruleID]
	message := ""
	if result.Message.Text != nil {
		message = *result.Message.Text
	}
	// Fall back to the rule's own description when the result carries no message.
	if message == "" && rule != nil {
		if d := mfmsText(rule.ShortDescription); d != "" {
			message = d
		} else if d := mfmsText(rule.FullDescription); d != "" {
			message = d
		}
	}

	// Id must be stable and non-empty. Prefer the rule id; else a hash of the
	// message; else fall back to the result index so empty-rule/empty-message
	// results don't collapse into one finding.
	id := ruleID
	if id == "" {
		if message != "" {
			id = shortHash(message)
		} else {
			id = fmt.Sprintf("finding-%d", index)
		}
	}
	// Summary must be non-empty; prefer the message, then the rule id, then a
	// generic label so Validate never rejects it.
	summary := message
	if summary == "" {
		summary = ruleID
	}
	if summary == "" {
		summary = "Finding from " + source.Name
	}

	// Severity: prefer the result level, fall back to the rule's default level.
	severity := convertSeverity(result.Level)
	if severity == nil && rule != nil && rule.DefaultConfiguration != nil {
		severity = convertSeverity(&rule.DefaultConfiguration.Level)
	}

	f := &fex.FindingExchange{
		Id:      id,
		Ref:     ruleID,
		Summary: summary,
		Source:  source,
		Status:  resultStatus(result),
		Details: &fex.FindingDetail{
			Category:    fex.FindingDetail_CATEGORY_SECURITY,
			Description: message,
			Severity:    severity,
			Confidence:  resultConfidence(result, rule),
			References:  ruleReferences(rule),
			Properties:  stringProps(result.Properties),
		},
	}
	// Prefer the scanner's own first-detection time so re-uploading an old report
	// doesn't reset first-seen to upload time; leave unset when unknown so the
	// platform can assign/preserve it.
	if ts := firstSeen(result, runTime); ts != nil {
		f.FirstSeenAt = timestamppb.New(*ts)
	}
	f.Affects = convertAffects(result)
	f.Remediations = convertRemediations(result)
	return f
}

// resultStatus maps SARIF suppressions onto the finding status. A result with any
// suppression is treated as not affected (the scanner/user has suppressed it).
func resultStatus(result *gosarif.Result) fex.Status {
	for _, s := range result.Suppressions {
		if s == nil {
			continue
		}
		// A suppression with an explicit "rejected" status is not actually
		// suppressed; anything else (accepted / under review / unset) is.
		if s.Status != nil && strings.EqualFold(*s.Status, "rejected") {
			continue
		}
		return fex.Status_STATUS_NOT_AFFECTED
	}
	return fex.Status_STATUS_AFFECTED
}

// resultConfidence maps a SARIF "precision" property (on the result, then the
// rule) onto fex confidence.
func resultConfidence(result *gosarif.Result, rule *gosarif.ReportingDescriptor) fex.Confidence {
	precision := propString(result.Properties, "precision")
	if precision == "" && rule != nil {
		precision = propString(rule.Properties, "precision")
	}
	switch strings.ToLower(precision) {
	case "very-high", "high":
		return fex.Confidence_CONFIDENCE_HIGH
	case "medium":
		return fex.Confidence_CONFIDENCE_MEDIUM
	case "low":
		return fex.Confidence_CONFIDENCE_LOW
	default:
		return fex.Confidence_CONFIDENCE_UNSPECIFIED
	}
}

// ruleReferences builds structured references from a rule's help URI and any CWE
// taxa carried in its tags (the "external/cwe/cwe-NNN" convention used by CodeQL,
// Semgrep, and others).
func ruleReferences(rule *gosarif.ReportingDescriptor) []*fex.Reference {
	if rule == nil {
		return nil
	}
	var refs []*fex.Reference
	if rule.HelpURI != nil && *rule.HelpURI != "" {
		name := "Help"
		if rule.Name != nil && *rule.Name != "" {
			name = *rule.Name
		}
		refs = append(refs, &fex.Reference{Name: name, Url: *rule.HelpURI})
	}
	for _, tag := range ruleTags(rule) {
		lower := strings.ToLower(tag)
		idx := strings.Index(lower, "cwe-")
		if idx < 0 {
			continue
		}
		num := strings.TrimSpace(tag[idx+len("cwe-"):])
		if num == "" {
			continue
		}
		refs = append(refs, &fex.Reference{
			Name: "CWE-" + num,
			Type: "CWE",
			Url:  "https://cwe.mitre.org/data/definitions/" + num + ".html",
		})
	}
	return refs
}

func ruleTags(rule *gosarif.ReportingDescriptor) []string {
	raw, ok := rule.Properties["tags"]
	if !ok {
		return nil
	}
	list, ok := raw.([]interface{})
	if !ok {
		return nil
	}
	var tags []string
	for _, v := range list {
		if s, ok := v.(string); ok {
			tags = append(tags, s)
		}
	}
	return tags
}

func mfmsText(m *gosarif.MultiformatMessageString) string {
	if m == nil || m.Text == nil {
		return ""
	}
	return *m.Text
}

func propString(p gosarif.Properties, key string) string {
	if p == nil {
		return ""
	}
	if s, ok := p[key].(string); ok {
		return s
	}
	return ""
}

// runStartTime returns the earliest invocation start time of a run, or nil.
func runStartTime(run *gosarif.Run) *time.Time {
	var earliest *time.Time
	for _, inv := range run.Invocations {
		if inv.StartTimeUTC == nil {
			continue
		}
		if earliest == nil || inv.StartTimeUTC.Before(*earliest) {
			earliest = inv.StartTimeUTC
		}
	}
	return earliest
}

// firstSeen prefers a result's provenance first-detection time, falling back to
// the run's start time. Returns nil when neither is present.
func firstSeen(result *gosarif.Result, runTime *time.Time) *time.Time {
	if result.Provenance != nil && result.Provenance.FirstDetectionTimeUTC != nil {
		return result.Provenance.FirstDetectionTimeUTC
	}
	return runTime
}

func convertSeverity(level *string) *fex.Severity {
	if level == nil {
		return nil
	}
	var rating fex.SeverityRating
	switch strings.ToLower(*level) {
	case "error":
		// SARIF "error" is the tool's highest level, but a SAST/lint error is not
		// the same as a CVSS-critical vulnerability, so map it to HIGH rather than
		// CRITICAL (reserve CRITICAL for scanners that carry a real CVSS score).
		rating = fex.SeverityRating_SEVERITY_RATING_HIGH
	case "warning":
		rating = fex.SeverityRating_SEVERITY_RATING_MEDIUM
	case "note":
		rating = fex.SeverityRating_SEVERITY_RATING_LOW
	case "none":
		rating = fex.SeverityRating_SEVERITY_RATING_NONE
	default:
		return nil
	}
	return &fex.Severity{Rating: rating}
}

func convertAffects(result *gosarif.Result) []*fex.Affects {
	var out []*fex.Affects
	seen := map[string]bool{}
	for _, loc := range result.Locations {
		if loc.PhysicalLocation == nil || loc.PhysicalLocation.ArtifactLocation == nil {
			continue
		}
		uri := loc.PhysicalLocation.ArtifactLocation.URI
		if uri == nil {
			continue
		}
		file := &fex.FileComponent{Path: *uri}
		if r := loc.PhysicalLocation.Region; r != nil {
			file.StartLine = derefInt(r.StartLine)
			file.EndLine = derefInt(r.EndLine)
			file.StartColumn = derefInt(r.StartColumn)
			file.EndColumn = derefInt(r.EndColumn)
		}
		// Dedup on the code location so distinct lines in the same file are
		// preserved rather than collapsed to the first one. Key on path + start
		// line/column only (not the end range): those are the stable anchors, so
		// the derived component Id stays consistent across re-scans while still
		// distinguishing separate findings.
		key := fmt.Sprintf("%s:%d:%d", file.Path, file.StartLine, file.StartColumn)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, &fex.Affects{Component: &fex.Component{
			Id:      shortHash(key),
			Details: &fex.Component_File{File: file},
		}})
	}
	return out
}

func derefInt(p *int) int32 {
	if p == nil {
		return 0
	}
	return int32(*p)
}

func convertRemediations(result *gosarif.Result) []*fex.Remediation {
	if len(result.Fixes) == 0 {
		return nil
	}
	var out []*fex.Remediation
	for _, fix := range result.Fixes {
		var b strings.Builder
		b.WriteString("### Automated fix\n\n")
		for _, change := range fix.ArtifactChanges {
			if change.ArtifactLocation.URI != nil {
				fmt.Fprintf(&b, "File: %s\n\n", *change.ArtifactLocation.URI)
			}
			for _, r := range change.Replacements {
				b.WriteString("```diff\n")
				if r.DeletedRegion.Snippet != nil && r.DeletedRegion.Snippet.Text != nil {
					fmt.Fprintf(&b, "- %s\n", *r.DeletedRegion.Snippet.Text)
				}
				if r.InsertedContent != nil && r.InsertedContent.Text != nil {
					fmt.Fprintf(&b, "+ %s\n", *r.InsertedContent.Text)
				}
				b.WriteString("```\n\n")
			}
		}
		out = append(out, &fex.Remediation{
			Category: fex.Remediation_Fix,
			Summary:  "Automated fix",
			FixType:  "markdown",
			Details:  b.String(),
		})
	}
	return out
}

func stringProps(p gosarif.Properties) map[string]string {
	if len(p) == 0 {
		return nil
	}
	out := map[string]string{}
	for k, v := range p {
		if s, ok := v.(string); ok {
			out[k] = s
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func shortHash(s string) string {
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h)[:16]
}
