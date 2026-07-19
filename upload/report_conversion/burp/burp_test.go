// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package burp_test

import (
	"strings"
	"testing"

	rc "go.mondoo.com/cnspec/v13/upload/report_conversion"
	"go.mondoo.com/cnspec/v13/upload/report_conversion/burp"
	"go.mondoo.com/mql/v13/providers-sdk/v1/upstream/fex"
)

func TestConvert(t *testing.T) {
	docs := rc.AssertClean(t, burp.Convert, "testdata/basic.xml")
	if len(docs) != 2 {
		t.Fatalf("want 2 documents, got %d", len(docs))
	}

	xss := docs[0].GetFex()
	if xss == nil {
		t.Fatal("expected FEX")
	}
	if xss.GetId() != "123456789" {
		t.Errorf("id = %q, want serialNumber 123456789", xss.GetId())
	}
	if xss.GetSource().GetName() != "burp" {
		t.Errorf("source = %q", xss.GetSource().GetName())
	}
	if xss.GetDetails().GetSeverity().GetRating() != fex.SeverityRating_SEVERITY_RATING_HIGH {
		t.Errorf("severity High → %v, want HIGH", xss.GetDetails().GetSeverity().GetRating())
	}
	if xss.GetDetails().GetConfidence() != fex.Confidence_CONFIDENCE_HIGH {
		t.Errorf("confidence Certain → %v, want HIGH", xss.GetDetails().GetConfidence())
	}
	if got := xss.GetDetails().GetDescription(); got == "" || containsHTMLTag(got) {
		t.Errorf("description not cleaned: %q", got)
	}
	// Affected URL = host+path, with the location parameter captured.
	if len(xss.GetAffects()) != 1 {
		t.Fatalf("want 1 affected URL, got %d", len(xss.GetAffects()))
	}
	comp := xss.GetAffects()[0].GetComponent()
	if comp.GetId() != "https://example.com/search" {
		t.Errorf("affected url = %q", comp.GetId())
	}
	if comp.GetProperties()["location"] != "q parameter" {
		t.Errorf("location property = %q, want 'q parameter'", comp.GetProperties()["location"])
	}
	// CWE reference + remediation.
	if len(xss.GetDetails().GetReferences()) != 1 || xss.GetDetails().GetReferences()[0].GetName() != "CWE-79" {
		t.Errorf("expected CWE-79 reference, got %+v", xss.GetDetails().GetReferences())
	}
	if len(xss.GetRemediations()) != 1 {
		t.Errorf("expected a remediation, got %d", len(xss.GetRemediations()))
	}

	// Second issue: Low + Firm.
	hsts := docs[1].GetFex()
	if hsts.GetDetails().GetSeverity().GetRating() != fex.SeverityRating_SEVERITY_RATING_LOW {
		t.Errorf("severity Low → %v, want LOW", hsts.GetDetails().GetSeverity().GetRating())
	}
	if hsts.GetDetails().GetConfidence() != fex.Confidence_CONFIDENCE_MEDIUM {
		t.Errorf("confidence Firm → %v, want MEDIUM", hsts.GetDetails().GetConfidence())
	}
}

func TestConvertNotBurp(t *testing.T) {
	_, err := burp.Convert([]byte(`<OWASPZAPReport/>`))
	if err == nil {
		t.Fatal("expected error for non-Burp XML")
	}
}

func containsHTMLTag(s string) bool {
	return strings.Contains(s, "<") && strings.Contains(s, ">")
}
