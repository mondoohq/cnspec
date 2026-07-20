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

	// The report-level generated date is used as first-seen for every finding.
	ts := xss.GetFirstSeenAt()
	if ts == nil {
		t.Fatal("expected FirstSeenAt from the report generated date")
	}
	if got := ts.AsTime().UTC().Format("2006-01-02 15:04:05"); got != "2025-01-01 12:00:00" {
		t.Errorf("first-seen = %q, want 2025-01-01 12:00:00", got)
	}
	if docs[1].GetFex().GetFirstSeenAt() == nil {
		t.Error("expected FirstSeenAt on the second finding too")
	}
}

func TestConvertUnparsableGeneratedLeavesFirstSeenUnset(t *testing.T) {
	// An unrecognized generated date must not fall back to time.Now(); leave unset.
	xml := []byte(`<OWASPZAPReport generated="not a date"><site name="s"><alerts><alertitem>
	  <pluginid>1</pluginid><name>x</name><riskcode>2</riskcode><desc>d</desc>
	  <instances><instance><uri>https://a/x</uri><method>GET</method></instance></instances>
	</alertitem></alerts></site></OWASPZAPReport>`)
	docs, err := zap.Convert(xml)
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	if docs[0].GetFex().GetFirstSeenAt() != nil {
		t.Errorf("expected FirstSeenAt unset for an unparsable date, got %v", docs[0].GetFex().GetFirstSeenAt())
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

func TestConvertMultiSiteDistinctIDs(t *testing.T) {
	// The same plugin fires on two sites; the findings must get distinct ids
	// (previously the bare plugin id collided across sites).
	xml := []byte(`<OWASPZAPReport>
	  <site name="https://a.example.com"><alerts><alertitem>
	    <pluginid>40012</pluginid><name>XSS</name><riskcode>3</riskcode><desc>d</desc>
	    <instances><instance><uri>https://a.example.com/x</uri><method>GET</method></instance></instances>
	  </alertitem></alerts></site>
	  <site name="https://b.example.com"><alerts><alertitem>
	    <pluginid>40012</pluginid><name>XSS</name><riskcode>3</riskcode><desc>d</desc>
	    <instances><instance><uri>https://b.example.com/x</uri><method>GET</method></instance></instances>
	  </alertitem></alerts></site>
	</OWASPZAPReport>`)
	docs, err := zap.Convert(xml)
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	if len(docs) != 2 {
		t.Fatalf("want 2 findings, got %d", len(docs))
	}
	id0, id1 := docs[0].GetFex().GetId(), docs[1].GetFex().GetId()
	if id0 == id1 {
		t.Errorf("ids collided across sites: %q", id0)
	}
	// Ref still carries the raw plugin id.
	if docs[0].GetFex().GetRef() != "40012" {
		t.Errorf("Ref = %q, want 40012", docs[0].GetFex().GetRef())
	}
}
