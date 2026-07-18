// Copyright Mondoo, Inc. 2026
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
		for _, result := range run.Results {
			docs = append(docs, fex.FexToDocument(convertResult(result, source, runTime, index)))
			index++
		}
	}
	return docs, nil
}

func convertResult(result *gosarif.Result, source *fex.Source, runTime *time.Time, index int) *fex.FindingExchange {
	ruleID := deref(result.RuleID)
	message := ""
	if result.Message.Text != nil {
		message = *result.Message.Text
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

	f := &fex.FindingExchange{
		Id:      id,
		Ref:     ruleID,
		Summary: summary,
		Source:  source,
		Status:  fex.Status_STATUS_AFFECTED,
		Details: &fex.FindingDetail{
			Category:    fex.FindingDetail_CATEGORY_SECURITY,
			Description: message,
			Severity:    convertSeverity(result.Level),
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
		rating = fex.SeverityRating_SEVERITY_RATING_HIGH
	case "warning":
		rating = fex.SeverityRating_SEVERITY_RATING_MEDIUM
	case "note":
		rating = fex.SeverityRating_SEVERITY_RATING_LOW
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
		if uri == nil || seen[*uri] {
			continue
		}
		seen[*uri] = true
		out = append(out, &fex.Affects{Component: &fex.Component{
			Id:      shortHash(*uri),
			Details: &fex.Component_File{File: &fex.FileComponent{Path: *uri}},
		}})
	}
	return out
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
