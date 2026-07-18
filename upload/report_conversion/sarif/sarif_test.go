// Copyright Mondoo, Inc. 2026
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
		t.Errorf("want 1 affected component, got %d", len(f.GetAffects()))
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
