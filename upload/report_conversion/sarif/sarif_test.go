// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package sarif_test

import (
	"testing"

	rc "go.mondoo.com/cnspec/v13/upload/report_conversion"
	"go.mondoo.com/cnspec/v13/upload/report_conversion/sarif"
	"go.mondoo.com/mql/v13/providers-sdk/v1/upstream/fex"
)

func TestConvert(t *testing.T) {
	docs := rc.AssertClean(t, sarif.Convert, "testdata/basic.sarif")
	if len(docs) != 2 {
		t.Fatalf("want 2 documents, got %d", len(docs))
	}

	f := docs[0].GetFex()
	if f == nil {
		t.Fatal("expected a FEX finding")
	}
	if f.GetSource().GetName() != "semgrep" {
		t.Errorf("source = %q, want semgrep", f.GetSource().GetName())
	}
	if f.GetDetails().GetCategory() != fex.FindingDetail_CATEGORY_SECURITY {
		t.Errorf("category = %v, want SECURITY", f.GetDetails().GetCategory())
	}
	if got := f.GetDetails().GetSeverity().GetRating(); got != fex.SeverityRating_SEVERITY_RATING_HIGH {
		t.Errorf("severity = %v, want HIGH (from sarif level=error)", got)
	}
	if len(f.GetAffects()) != 1 {
		t.Fatalf("want 1 affected component, got %d", len(f.GetAffects()))
	}
	// The SARIF region line is captured on the file component.
	if got := f.GetAffects()[0].GetComponent().GetFile().GetStartLine(); got != 42 {
		t.Errorf("start_line = %d, want 42 (from sarif region)", got)
	}
}

// TestConvertEmptyFields checks that results with neither a ruleId nor a message
// still produce clean, distinctly-identified findings (no collapse, no empty
// summary).
func TestConvertEmptyFields(t *testing.T) {
	report := []byte(`{
	  "version": "2.1.0",
	  "runs": [{
	    "tool": {"driver": {"name": "toolx"}},
	    "results": [
	      {"locations": [{"physicalLocation": {"artifactLocation": {"uri": "a.go"}}}]},
	      {"locations": [{"physicalLocation": {"artifactLocation": {"uri": "b.go"}}}]}
	    ]
	  }]
	}`)
	docs, err := sarif.Convert(report)
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	if len(docs) != 2 {
		t.Fatalf("want 2 documents, got %d", len(docs))
	}
	ids := map[string]bool{}
	for i, d := range docs {
		if err := rc.Validate(d); err != nil {
			t.Errorf("document %d not clean: %v", i, err)
		}
		id := d.GetFex().GetId()
		if ids[id] {
			t.Errorf("duplicate id %q — empty-field results collapsed", id)
		}
		ids[id] = true
	}
}

// A single result with multiple locations in the same file at different lines
// must keep each distinct code location; only exact duplicates are deduped.
func TestConvertSameFileDifferentLines(t *testing.T) {
	docs := rc.AssertClean(t, sarif.Convert, "testdata/same_file_lines.sarif")
	if len(docs) != 1 {
		t.Fatalf("want 1 document, got %d", len(docs))
	}

	affects := docs[0].GetFex().GetAffects()
	if len(affects) != 2 {
		t.Fatalf("want 2 affected locations (line 42 and 100, exact dup dropped), got %d", len(affects))
	}

	lines := map[int32]bool{}
	for _, a := range affects {
		lines[a.GetComponent().GetFile().GetStartLine()] = true
	}
	if !lines[42] || !lines[100] {
		t.Errorf("want lines 42 and 100 preserved, got %v", lines)
	}
}

func TestConvertRuleEnrichmentAndSuppression(t *testing.T) {
	report := []byte(`{
	  "version": "2.1.0",
	  "runs": [{
	    "tool": {"driver": {
	      "name": "codeql",
	      "rules": [{
	        "id": "rule1",
	        "name": "SQL Injection",
	        "shortDescription": {"text": "SQL injection vulnerability."},
	        "helpUri": "https://example.com/rules/rule1",
	        "defaultConfiguration": {"level": "error"},
	        "properties": {"precision": "high", "tags": ["security", "external/cwe/cwe-89"]}
	      }]
	    }},
	    "results": [
	      {"ruleId": "rule1", "locations": [{"physicalLocation": {"artifactLocation": {"uri": "a.go"}, "region": {"startLine": 5}}}]},
	      {"ruleId": "rule1", "message": {"text": "suppressed"}, "suppressions": [{"kind": "inSource"}], "locations": [{"physicalLocation": {"artifactLocation": {"uri": "b.go"}, "region": {"startLine": 9}}}]}
	    ]
	  }]
	}`)
	docs, err := sarif.Convert(report)
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	if len(docs) != 2 {
		t.Fatalf("want 2 documents, got %d", len(docs))
	}

	// First result has no message/level: description, severity, confidence and
	// references are all enriched from the rule definition.
	f := docs[0].GetFex()
	if got := f.GetDetails().GetDescription(); got != "SQL injection vulnerability." {
		t.Errorf("description = %q, want rule short description", got)
	}
	if got := f.GetDetails().GetSeverity().GetRating(); got != fex.SeverityRating_SEVERITY_RATING_HIGH {
		t.Errorf("severity = %v, want HIGH (from rule default level error)", got)
	}
	if got := f.GetDetails().GetConfidence(); got != fex.Confidence_CONFIDENCE_HIGH {
		t.Errorf("confidence = %v, want HIGH (from precision)", got)
	}
	var haveHelp, haveCWE bool
	for _, r := range f.GetDetails().GetReferences() {
		if r.GetUrl() == "https://example.com/rules/rule1" {
			haveHelp = true
		}
		if r.GetType() == "CWE" && r.GetName() == "CWE-89" {
			haveCWE = true
		}
	}
	if !haveHelp || !haveCWE {
		t.Errorf("expected helpUri + CWE-89 references, got %+v", f.GetDetails().GetReferences())
	}
	if f.GetStatus() != fex.Status_STATUS_AFFECTED {
		t.Errorf("status = %v, want AFFECTED", f.GetStatus())
	}

	// Second result is suppressed → NOT_AFFECTED.
	if got := docs[1].GetFex().GetStatus(); got != fex.Status_STATUS_NOT_AFFECTED {
		t.Errorf("suppressed status = %v, want NOT_AFFECTED", got)
	}
}

func TestConvertNoneLevelKeepsSeverity(t *testing.T) {
	report := []byte(`{
	  "version": "2.1.0",
	  "runs": [{
	    "tool": {"driver": {"name": "toolx"}},
	    "results": [
	      {"ruleId": "r1", "level": "none", "message": {"text": "informational"},
	       "locations": [{"physicalLocation": {"artifactLocation": {"uri": "a.go"}}}]}
	    ]
	  }]
	}`)
	docs, err := sarif.Convert(report)
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("want 1 document, got %d", len(docs))
	}
	sev := docs[0].GetFex().GetDetails().GetSeverity()
	if sev == nil || sev.GetRating() != fex.SeverityRating_SEVERITY_RATING_NONE {
		t.Errorf("level none → %v, want SEVERITY_RATING_NONE (not dropped)", sev.GetRating())
	}
}
