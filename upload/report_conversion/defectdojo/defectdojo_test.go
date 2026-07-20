// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package defectdojo_test

import (
	"testing"

	rc "go.mondoo.com/cnspec/v13/upload/report_conversion"
	"go.mondoo.com/cnspec/v13/upload/report_conversion/defectdojo"
	"go.mondoo.com/mql/v13/providers-sdk/v1/upstream/fex"
)

func TestConvertJSON(t *testing.T) {
	docs := rc.AssertClean(t, defectdojo.Convert, "testdata/basic.json")
	if len(docs) != 2 {
		t.Fatalf("want 2 documents, got %d", len(docs))
	}

	// First finding (no CVE) → FEX.
	f := docs[0].GetFex()
	if f == nil {
		t.Fatal("finding 0: expected FEX")
	}
	if f.GetSource().GetName() != "manual-pentest" {
		t.Errorf("source = %q", f.GetSource().GetName())
	}
	if f.GetDetails().GetSeverity().GetRating() != fex.SeverityRating_SEVERITY_RATING_HIGH {
		t.Errorf("severity = %v, want HIGH", f.GetDetails().GetSeverity().GetRating())
	}
	if f.GetId() != "PENTEST-001" {
		t.Errorf("id = %q, want PENTEST-001", f.GetId())
	}

	// Second finding (has CVE) → VEX.
	v := docs[1].GetVex()
	if v == nil {
		t.Fatal("finding 1: expected VEX (has CVE)")
	}
	if v.GetId() != "CVE-2024-0001" {
		t.Errorf("vex id = %q", v.GetId())
	}
}

func TestConvertCSV(t *testing.T) {
	docs := rc.AssertClean(t, defectdojo.Convert, "testdata/basic.csv")
	if len(docs) != 2 {
		t.Fatalf("want 2 documents, got %d", len(docs))
	}
	// Row without a CVE → FEX; row with a CVE → VEX.
	if docs[0].GetFex() == nil {
		t.Error("row 0: expected FEX")
	}
	if v := docs[1].GetVex(); v == nil || v.GetId() != "CVE-2024-1234" {
		t.Errorf("row 1: expected VEX CVE-2024-1234, got %+v", v)
	}
}

func TestConvertMissingRequired(t *testing.T) {
	_, err := defectdojo.Convert([]byte(`{"findings":[{"title":"x"}]}`))
	if err == nil {
		t.Fatal("expected error for missing severity/description")
	}
}

func TestConvertStatusAndMappings(t *testing.T) {
	docs := rc.AssertClean(t, defectdojo.Convert, "testdata/extended.json")
	// 6 findings in the fixture, but the DefectDojo-flagged duplicate is skipped.
	if len(docs) != 5 {
		t.Fatalf("want 5 documents (duplicate skipped), got %d", len(docs))
	}

	// A false_p finding maps to STATUS_FALSE_POSITIVE (not AFFECTED).
	fp := docs[0].GetFex()
	if fp == nil {
		t.Fatal("finding 0: expected FEX")
	}
	if fp.GetStatus() != fex.Status_STATUS_FALSE_POSITIVE {
		t.Errorf("false_p status = %v, want STATUS_FALSE_POSITIVE", fp.GetStatus())
	}

	// A CVE finding becomes VEX with Ratings (severity + CVSSv3) and References.
	v := docs[1].GetVex()
	if v == nil {
		t.Fatal("finding 1: expected VEX")
	}
	if v.GetStatus() != fex.Status_STATUS_AFFECTED {
		t.Errorf("vex status = %v, want STATUS_AFFECTED", v.GetStatus())
	}
	if len(v.GetRatings()) == 0 {
		t.Fatal("vex: expected Ratings to be set")
	}
	rating := v.GetRatings()[0]
	if rating.GetSeverity() != "high" {
		t.Errorf("vex rating severity = %q, want high", rating.GetSeverity())
	}
	if rating.GetScore() != 7.5 {
		t.Errorf("vex rating score = %v, want 7.5", rating.GetScore())
	}
	if rating.GetMethod() != fex.ScoringMethod_SCOREMETHOD_CVSSv3 {
		t.Errorf("vex rating method = %v, want CVSSv3", rating.GetMethod())
	}
	if rating.GetVector() == "" {
		t.Error("vex rating vector should be set")
	}
	if len(v.GetReferences()) == 0 {
		t.Error("vex: expected References to be set")
	}

	// file_path + line populate FileComponent.StartLine.
	fc := docs[2].GetFex()
	if fc == nil {
		t.Fatal("finding 2: expected FEX")
	}
	if len(fc.GetAffects()) == 0 {
		t.Fatal("finding 2: expected an affected component")
	}
	file := fc.GetAffects()[0].GetComponent().GetFile()
	if file == nil {
		t.Fatal("finding 2: expected a FileComponent")
	}
	if file.GetPath() != "config/app.yaml" {
		t.Errorf("file path = %q", file.GetPath())
	}
	if file.GetStartLine() != 12 {
		t.Errorf("file start line = %d, want 12", file.GetStartLine())
	}
	// verified:true maps to HIGH confidence.
	if fc.GetDetails().GetConfidence() != fex.Confidence_CONFIDENCE_HIGH {
		t.Errorf("verified finding confidence = %v, want HIGH", fc.GetDetails().GetConfidence())
	}

	// The DefectDojo-flagged duplicate must not appear in the output.
	for _, d := range docs {
		if d.GetFex().GetSummary() == "Duplicate of the hardcoded secret finding" {
			t.Error("duplicate finding should have been skipped")
		}
	}

	// Two findings with the same title/description but different files get
	// distinct ids (the fallback hash includes the file path).
	a, b := docs[3].GetFex(), docs[4].GetFex()
	if a == nil || b == nil {
		t.Fatal("findings 3/4: expected FEX")
	}
	if a.GetId() == "" || a.GetId() == b.GetId() {
		t.Errorf("expected distinct ids, got %q and %q", a.GetId(), b.GetId())
	}
}
