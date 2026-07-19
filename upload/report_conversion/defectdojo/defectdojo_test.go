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
