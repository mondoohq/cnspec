// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package zap_test

import (
	"testing"

	rc "go.mondoo.com/cnspec/v13/upload/report_conversion"
	"go.mondoo.com/cnspec/v13/upload/report_conversion/zap"
	"go.mondoo.com/mql/v13/providers-sdk/v1/upstream/fex"
)

func TestConvert(t *testing.T) {
	docs := rc.AssertClean(t, zap.Convert, "testdata/basic.xml")
	if len(docs) != 2 {
		t.Fatalf("want 2 documents, got %d", len(docs))
	}

	xss := docs[0].GetFex()
	if xss == nil {
		t.Fatal("expected FEX")
	}
	if xss.GetSource().GetName() != "https://example.com" {
		t.Errorf("source = %q", xss.GetSource().GetName())
	}
	if xss.GetDetails().GetSeverity().GetRating() != fex.SeverityRating_SEVERITY_RATING_HIGH {
		t.Errorf("riskcode 3 → %v, want HIGH", xss.GetDetails().GetSeverity().GetRating())
	}
	if xss.GetDetails().GetConfidence() != fex.Confidence_CONFIDENCE_MEDIUM {
		t.Errorf("confidence 2 → %v, want MEDIUM", xss.GetDetails().GetConfidence())
	}
	// HTML in desc is stripped.
	if got := xss.GetDetails().GetDescription(); got != "Reflected XSS in the q parameter." {
		t.Errorf("description = %q (HTML not cleaned?)", got)
	}
	// Affected URL + request context.
	if len(xss.GetAffects()) != 1 {
		t.Fatalf("want 1 affected URL, got %d", len(xss.GetAffects()))
	}
	comp := xss.GetAffects()[0].GetComponent()
	if comp.GetId() != "https://example.com/search?q=test" {
		t.Errorf("affected uri = %q", comp.GetId())
	}
	if comp.GetProperties()["param"] != "q" {
		t.Errorf("param property = %q, want q", comp.GetProperties()["param"])
	}
	// CWE reference + remediation.
	if len(xss.GetDetails().GetReferences()) == 0 || xss.GetDetails().GetReferences()[0].GetName() != "CWE-79" {
		t.Errorf("expected CWE-79 reference, got %+v", xss.GetDetails().GetReferences())
	}
	if len(xss.GetRemediations()) != 1 {
		t.Errorf("expected a remediation, got %d", len(xss.GetRemediations()))
	}

	// Second alert: riskcode 1 → LOW.
	if got := docs[1].GetFex().GetDetails().GetSeverity().GetRating(); got != fex.SeverityRating_SEVERITY_RATING_LOW {
		t.Errorf("riskcode 1 → %v, want LOW", got)
	}
}

func TestConvertNotZAP(t *testing.T) {
	_, err := zap.Convert([]byte(`<something/>`))
	if err == nil {
		t.Fatal("expected error for non-ZAP XML")
	}
}
