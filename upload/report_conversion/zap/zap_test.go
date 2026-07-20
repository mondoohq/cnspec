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
	// Affected URL.
	if len(xss.GetAffects()) != 1 {
		t.Fatalf("want 1 affected URL, got %d", len(xss.GetAffects()))
	}
	if got := xss.GetAffects()[0].GetComponent().GetId(); got != "https://example.com/search?q=test" {
		t.Errorf("affected uri = %q", got)
	}
	// Request context is first-class HttpRequest evidence.
	if len(xss.GetEvidences()) != 1 {
		t.Fatalf("want 1 evidence, got %d", len(xss.GetEvidences()))
	}
	hr := xss.GetEvidences()[0].GetHttpRequest()
	if hr == nil {
		t.Fatal("expected HttpRequest evidence")
	}
	if hr.GetParam() != "q" || hr.GetMethod() != "GET" {
		t.Errorf("http evidence = method %q param %q, want GET/q", hr.GetMethod(), hr.GetParam())
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

func TestConvertPreservesDistinctInstances(t *testing.T) {
	// Same URL, different method/param → distinct Affects (not collapsed);
	// a truly identical instance is deduped.
	xml := []byte(`<OWASPZAPReport><site name="s"><alerts><alertitem>
	  <pluginid>1</pluginid><name>x</name><riskcode>2</riskcode>
	  <desc>d</desc>
	  <instances>
	    <instance><uri>https://a/x</uri><method>GET</method><param>q</param></instance>
	    <instance><uri>https://a/x</uri><method>POST</method><param>q</param></instance>
	    <instance><uri>https://a/x</uri><method>GET</method><param>q</param></instance>
	  </instances>
	</alertitem></alerts></site></OWASPZAPReport>`)
	docs, err := zap.Convert(xml)
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	affects := docs[0].GetFex().GetAffects()
	if len(affects) != 2 {
		t.Fatalf("want 2 distinct instances (GET+POST; duplicate GET deduped), got %d", len(affects))
	}
}

func TestConvertNotZAP(t *testing.T) {
	_, err := zap.Convert([]byte(`<something/>`))
	if err == nil {
		t.Fatal("expected error for non-ZAP XML")
	}
}
