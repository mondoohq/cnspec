// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package ohdf_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	rc "go.mondoo.com/cnspec/upload/report_conversion"
	"go.mondoo.com/cnspec/upload/report_conversion/ohdf"
	"go.mondoo.com/mql/providers-sdk/v1/upstream/fex"
)

const (
	// Produced by a SAF-style converter: a tool name in the profile title, NIST
	// and CCI tags, Ruby and RFC 3339 timestamps, a second profile repeating a
	// control id, and controls this converter has to drop.
	thirdPartyReport = "testdata/third-party-scan.hdf.json"
	// Produced by cnspec's own OHDF writer (reports/hdf), so the two halves of
	// the format meet: an exported scan has to import.
	cnspecReport = "testdata/cnspec-scan.hdf.json"
)

func findingByRef(docs []*fex.FindingDocument, ref string) *fex.FindingExchange {
	for _, d := range docs {
		if f := d.GetFex(); f != nil && f.GetRef() == ref {
			return f
		}
	}
	return nil
}

func TestConvertThirdPartyReport(t *testing.T) {
	docs := rc.AssertClean(t, ohdf.Convert, thirdPartyReport)

	// Three of the six controls report a problem: two failed, one errored. The
	// passing, the excluded (impact 0) and the skipped controls are not findings.
	if len(docs) != 3 {
		t.Fatalf("got %d findings, want 3", len(docs))
	}
	for _, ref := range []string{"51192", "19506", "90002"} {
		if f := findingByRef(docs, ref); f != nil {
			t.Errorf("control %q should not be a finding, got summary %q", ref, f.GetSummary())
		}
	}
}

func TestConvertControlMapping(t *testing.T) {
	docs := rc.AssertClean(t, ohdf.Convert, thirdPartyReport)

	f := findingByRef(docs, "10180")
	if f == nil {
		t.Fatal("expected a finding for control 10180")
	}

	if got := f.GetSummary(); got != "Ping the remote host" {
		t.Errorf("summary = %q, want the control title", got)
	}
	if got := f.GetSource().GetName(); got != "Nessus Vulnerability Scan" {
		t.Errorf("source = %q, want the profile title", got)
	}
	if got := f.GetStatus(); got != fex.Status_STATUS_AFFECTED {
		t.Errorf("status = %v, want AFFECTED for a failed control", got)
	}
	if got := f.GetDetails().GetSeverity().GetRating(); got != fex.SeverityRating_SEVERITY_RATING_MEDIUM {
		t.Errorf("rating = %v, want MEDIUM for impact 0.5", got)
	}

	// The control's own description, its check text and both failing results all
	// reach the reader; without the results the finding says what is checked but
	// never what was found.
	desc := f.GetDetails().GetDescription()
	for _, want := range []string{
		"The remote host responded to an ICMP echo request.",
		"Send an ICMP echo request",
		"ICMP echo reply from 10.0.0.12",
		"ICMP echo reply from 10.0.0.13",
	} {
		if !strings.Contains(desc, want) {
			t.Errorf("description is missing %q\ngot: %s", want, desc)
		}
	}

	rem := f.GetRemediations()
	if len(rem) != 1 {
		t.Fatalf("got %d remediations, want 1", len(rem))
	}
	if got := rem[0].GetSummary(); !strings.Contains(got, "Filter ICMP") {
		t.Errorf("remediation = %q, want the fix description", got)
	}
	if got := rem[0].GetCategory(); got != fex.Remediation_Fix {
		t.Errorf("remediation category = %v, want Fix", got)
	}

	// A ref without a URL cannot be followed, so it is dropped rather than
	// carried as an empty link.
	refs := f.GetDetails().GetReferences()
	if len(refs) != 2 {
		t.Fatalf("got %d references, want 2", len(refs))
	}
	if refs[0].GetName() != "Nessus plugin 10180" {
		t.Errorf("reference name = %q", refs[0].GetName())
	}
	if refs[1].GetName() != refs[1].GetUrl() {
		t.Errorf("a reference without a label should fall back to its URL, got %q", refs[1].GetName())
	}

	// The earliest failing result, not the upload time: the two results are the
	// same instant written two ways, one Ruby and one RFC 3339.
	if got := f.GetFirstSeenAt().AsTime().UTC().Format("2006-01-02T15:04:05Z"); got != "2026-03-04T05:06:07Z" {
		t.Errorf("first seen = %s, want the earliest result time", got)
	}
}

// TestConvertTagsBecomeProperties covers the compliance mappings. They are the
// part of an OHDF control with no home elsewhere in FEX, and `nist` in particular
// is what the SAF ecosystem keys its 800-53 views off.
func TestConvertTagsBecomeProperties(t *testing.T) {
	docs := rc.AssertClean(t, ohdf.Convert, thirdPartyReport)

	f := findingByRef(docs, "10180")
	if f == nil {
		t.Fatal("expected a finding for control 10180")
	}

	props := f.GetDetails().GetProperties()
	tests := map[string]string{
		"nist":              "SI-4, RA-5",
		"cci":               "CCI-001312",
		"severity":          "medium",
		"plugin_family":     "General",
		"risk_factor":       "5.5",
		"exploit_available": "false",
		"nested":            "a=one, b=two",
	}
	for key, want := range tests {
		if got := props[key]; got != want {
			t.Errorf("property %q = %q, want %q", key, got, want)
		}
	}
	if _, ok := props["empty_tag"]; ok {
		t.Errorf("a null tag should be dropped, got %q", props["empty_tag"])
	}
}

func TestConvertAffectedComponent(t *testing.T) {
	docs := rc.AssertClean(t, ohdf.Convert, thirdPartyReport)

	f := findingByRef(docs, "10180")
	if f == nil {
		t.Fatal("expected a finding for control 10180")
	}

	affects := f.GetAffects()
	if len(affects) != 1 {
		t.Fatalf("got %d affected components, want 1", len(affects))
	}
	c := affects[0].GetComponent()
	if got := c.GetId(); got != "i-0abc123def4567890" {
		t.Errorf("component id = %q, want the platform target_id", got)
	}
	if got := c.GetIdentifiers()["platform"]; got != "Heimdall Tools" {
		t.Errorf("platform identifier = %q", got)
	}
	if got := c.GetIdentifiers()["release"]; got != "2.13.0" {
		t.Errorf("release identifier = %q", got)
	}
}

// TestConvertErroredControl covers the control that could not reach a verdict. It
// is a finding, because a check that did not run is not a check that passed, but
// it is not reported as a confirmed failure.
func TestConvertErroredControl(t *testing.T) {
	docs := rc.AssertClean(t, ohdf.Convert, thirdPartyReport)

	f := findingByRef(docs, "90001")
	if f == nil {
		t.Fatal("expected a finding for the errored control 90001")
	}
	if got := f.GetStatus(); got != fex.Status_STATUS_UNDER_INVESTIGATION {
		t.Errorf("status = %v, want UNDER_INVESTIGATION for an errored control", got)
	}
	if got := f.GetDetails().GetSeverity().GetRating(); got != fex.SeverityRating_SEVERITY_RATING_HIGH {
		t.Errorf("rating = %v, want HIGH for impact 0.7", got)
	}
	// The result carries no code_desc and no message, so without a fallback the
	// finding would describe nothing at all.
	if !strings.Contains(f.GetDetails().GetDescription(), "check errored") {
		t.Errorf("an errored result with no text should still say so, got %q",
			f.GetDetails().GetDescription())
	}
}

// TestConvertRepeatedControlID covers a document whose profiles reuse an id.
// Merged reports do this, and two findings sharing an id collapse downstream.
func TestConvertRepeatedControlID(t *testing.T) {
	docs := rc.AssertClean(t, ohdf.Convert, thirdPartyReport)

	ids := map[string]int{}
	for _, d := range docs {
		ids[d.GetFex().GetId()]++
	}
	for id, n := range ids {
		if n > 1 {
			t.Errorf("finding id %q appears %d times", id, n)
		}
	}

	// Both controls keep the source ref; only the id is disambiguated.
	var refs int
	for _, d := range docs {
		if d.GetFex().GetRef() == "10180" {
			refs++
		}
	}
	if refs != 2 {
		t.Errorf("got %d findings for the repeated ref, want 2", refs)
	}
}

// TestConvertCnspecReport closes the loop: a scan cnspec exported as OHDF has to
// import through the same converter.
func TestConvertCnspecReport(t *testing.T) {
	docs := rc.AssertClean(t, ohdf.Convert, cnspecReport)

	// The recorded scan reports 2 failed and 4 errored checks out of 24.
	if len(docs) != 6 {
		t.Fatalf("got %d findings, want 6", len(docs))
	}

	var affected, investigating int
	for _, d := range docs {
		f := d.GetFex()
		switch f.GetStatus() {
		case fex.Status_STATUS_AFFECTED:
			affected++
		case fex.Status_STATUS_UNDER_INVESTIGATION:
			investigating++
		}
		if f.GetAffects() == nil {
			t.Errorf("finding %q has no affected component", f.GetId())
		}
		// cnspec writes a nist tag on every control, mapped or UM-1.
		if got := f.GetDetails().GetProperties()["nist"]; got == "" {
			t.Errorf("finding %q lost its nist tag", f.GetId())
		}
	}
	if affected != 2 {
		t.Errorf("got %d failed findings, want 2", affected)
	}
	if investigating != 4 {
		t.Errorf("got %d errored findings, want 4", investigating)
	}
}

func TestConvertRejectsBadInput(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{"not json", "this is not json"},
		{"no profiles", `{"platform":{"name":"x"},"version":"1","profiles":[]}`},
		{"profiles wrong type", `{"profiles":"nope"}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := ohdf.Convert([]byte(test.data)); err == nil {
				t.Error("expected an error")
			}
		})
	}
}

// TestConvertRegistered checks the converter is reachable by the name `cnspec
// upload --format` takes, which is also the name the scan reporter writes.
func TestConvertRegistered(t *testing.T) {
	conv, ok := rc.Get("hdf")
	if !ok {
		t.Fatal("no converter registered for \"hdf\"")
	}

	data, err := os.ReadFile(thirdPartyReport)
	if err != nil {
		t.Fatal(err)
	}
	docs, err := conv(data)
	if err != nil {
		t.Fatalf("registered converter failed: %v", err)
	}
	if len(docs) == 0 {
		t.Error("registered converter produced no documents")
	}
}

// TestFixtureIsValidOHDF guards the fixtures themselves: a malformed one would
// make every test above prove less than it appears to.
func TestFixtureIsValidOHDF(t *testing.T) {
	for _, path := range []string{thirdPartyReport, cnspecReport} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var doc struct {
			Platform map[string]any   `json:"platform"`
			Version  string           `json:"version"`
			Profiles []map[string]any `json:"profiles"`
		}
		if err := json.Unmarshal(data, &doc); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		if len(doc.Platform) == 0 || doc.Version == "" || len(doc.Profiles) == 0 {
			t.Errorf("%s: missing a field OHDF requires", path)
		}
	}
}
