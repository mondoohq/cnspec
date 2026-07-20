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
	// id is a deterministic hash of the issue identity, NOT Burp's volatile
	// serialNumber (which is regenerated each scan and would cause duplicates).
	if xss.GetId() == "" || xss.GetId() == "123456789" {
		t.Errorf("id = %q, want a deterministic hash, not the serialNumber", xss.GetId())
	}
	// Ref keeps the Burp type/plugin id.
	if xss.GetRef() != "2097936" {
		t.Errorf("ref = %q, want burp type 2097936", xss.GetRef())
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
	// Affected URL = host+path.
	if len(xss.GetAffects()) != 1 {
		t.Fatalf("want 1 affected URL, got %d", len(xss.GetAffects()))
	}
	comp := xss.GetAffects()[0].GetComponent()
	if got := comp.GetId(); got != "https://example.com/search" {
		t.Errorf("affected url = %q", got)
	}
	// Host IP is carried as a component identifier.
	if got := comp.GetIdentifiers()["ip"]; got != "93.184.216.34" {
		t.Errorf("component ip identifier = %q, want 93.184.216.34", got)
	}
	// Request context is first-class HttpRequest evidence.
	if len(xss.GetEvidences()) != 1 {
		t.Fatalf("want 1 evidence, got %d", len(xss.GetEvidences()))
	}
	hr := xss.GetEvidences()[0].GetHttpRequest()
	if hr == nil || hr.GetUrl() != "https://example.com/search" {
		t.Errorf("http evidence url = %+v", hr)
	}
	if hr.GetParam() != "q parameter" {
		t.Errorf("http evidence param = %q, want 'q parameter'", hr.GetParam())
	}
	// The captured request/response are base64-decoded from Burp's XML.
	if got := hr.GetRequest(); got != "GET /search?q=test" {
		t.Errorf("http evidence request = %q, want decoded 'GET /search?q=test'", got)
	}
	if got := hr.GetResponse(); got != "HTTP/1.1 200 OK" {
		t.Errorf("http evidence response = %q, want decoded 'HTTP/1.1 200 OK'", got)
	}
	// CWE reference (with mitre URL) + remediation.
	if len(xss.GetDetails().GetReferences()) != 1 || xss.GetDetails().GetReferences()[0].GetName() != "CWE-79" {
		t.Errorf("expected CWE-79 reference, got %+v", xss.GetDetails().GetReferences())
	}
	if got := xss.GetDetails().GetReferences()[0].GetUrl(); got != "https://cwe.mitre.org/data/definitions/79.html" {
		t.Errorf("CWE reference url = %q", got)
	}
	// Both the general remediation (remediationBackground) and the
	// instance-specific one (remediationDetail) are surfaced.
	rems := xss.GetRemediations()
	if len(rems) != 2 {
		t.Fatalf("expected 2 remediations (background + detail), got %d", len(rems))
	}
	if got := rems[0].GetDetails(); got != "Encode output." {
		t.Errorf("remediation[0] = %q, want the background", got)
	}
	if got := rems[1].GetDetails(); got != "HTML-encode the q parameter before echoing it." {
		t.Errorf("remediation[1] = %q, want the instance-specific detail", got)
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

func TestConvertDeterministicID(t *testing.T) {
	// The same Burp export converted twice must produce the same finding ids,
	// even though Burp regenerates serialNumbers per scan.
	first := rc.AssertClean(t, burp.Convert, "testdata/basic.xml")
	second := rc.AssertClean(t, burp.Convert, "testdata/basic.xml")
	if len(first) != len(second) {
		t.Fatalf("doc count differs: %d vs %d", len(first), len(second))
	}
	for i := range first {
		a, b := first[i].GetFex().GetId(), second[i].GetFex().GetId()
		if a == "" {
			t.Fatalf("doc %d has empty id", i)
		}
		if a != b {
			t.Errorf("doc %d id not deterministic: %q vs %q", i, a, b)
		}
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
