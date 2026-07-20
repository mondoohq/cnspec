// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package junit_test

import (
	"strings"
	"testing"

	rc "go.mondoo.com/cnspec/v13/upload/report_conversion"
	"go.mondoo.com/cnspec/v13/upload/report_conversion/junit"
	"go.mondoo.com/mql/v13/providers-sdk/v1/upstream/fex"
)

func TestConvert(t *testing.T) {
	// 3 test cases: one failure, one error, one pass → 2 findings.
	docs := rc.AssertClean(t, junit.Convert, "testdata/basic.xml")
	if len(docs) != 2 {
		t.Fatalf("want 2 documents (failure + error, pass ignored), got %d", len(docs))
	}

	f := docs[0].GetFex()
	if f == nil {
		t.Fatal("expected FEX")
	}
	if f.GetSource().GetName() != "security-checks" {
		t.Errorf("source = %q", f.GetSource().GetName())
	}
	if f.GetDetails().GetSeverity().GetRating() != fex.SeverityRating_SEVERITY_RATING_MEDIUM {
		t.Errorf("failure severity = %v, want MEDIUM", f.GetDetails().GetSeverity().GetRating())
	}
	// The error case is rated higher than a plain failure.
	if got := docs[1].GetFex().GetDetails().GetSeverity().GetRating(); got != fex.SeverityRating_SEVERITY_RATING_HIGH {
		t.Errorf("error severity = %v, want HIGH", got)
	}
}

func TestConvertDuplicateCasesGetDistinctIDs(t *testing.T) {
	// Two failing cases with the same suite + classname + name (e.g. a merged
	// report) must not collapse into one id.
	xml := []byte(`<testsuite name="s">
	  <testcase classname="c" name="t"><failure message="a"/></testcase>
	  <testcase classname="c" name="t"><failure message="b"/></testcase>
	</testsuite>`)
	docs, err := junit.Convert(xml)
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	if len(docs) != 2 {
		t.Fatalf("want 2 documents, got %d", len(docs))
	}
	if docs[0].GetFex().GetId() == docs[1].GetFex().GetId() {
		t.Errorf("duplicate cases collapsed to the same id %q", docs[0].GetFex().GetId())
	}
}

func TestConvertMalformedXML(t *testing.T) {
	// A genuine syntax error should surface as a parse error, not a misleading
	// "no testsuite element found".
	_, err := junit.Convert([]byte(`<testsuite name="s"><testcase`))
	if err == nil {
		t.Fatal("expected a parse error for malformed XML")
	}
	if strings.Contains(err.Error(), "no testsuite element found") {
		t.Errorf("got misleading structural error instead of the syntax error: %v", err)
	}
}

func TestConvertNestedSuites(t *testing.T) {
	// Aggregated reports nest <testsuite> inside <testsuite>; the nested failing
	// case must surface, its file/line must map to a FileComponent, and
	// system-out/system-err must be folded into the description.
	docs := rc.AssertClean(t, junit.Convert, "testdata/nested.xml")
	if len(docs) != 1 {
		t.Fatalf("want 1 document (the nested failure; both passing/outer cases ignored), got %d", len(docs))
	}

	f := docs[0].GetFex()
	if f == nil {
		t.Fatal("expected FEX")
	}
	if f.GetRef() != "tests.test_login.test_admin_login" {
		t.Errorf("ref = %q, want the nested case name", f.GetRef())
	}
	// The nested suite name is used as the source.
	if got := f.GetSource().GetName(); got != "nested-module" {
		t.Errorf("source = %q, want nested-module", got)
	}

	// testcase file/line → FileComponent{Path,StartLine}.
	if len(f.GetAffects()) != 1 {
		t.Fatalf("want 1 affects (file location), got %d", len(f.GetAffects()))
	}
	file := f.GetAffects()[0].GetComponent().GetFile()
	if file == nil {
		t.Fatal("expected a FileComponent for the testcase file/line")
	}
	if file.GetPath() != "tests/test_login.py" {
		t.Errorf("file path = %q", file.GetPath())
	}
	if file.GetStartLine() != 42 {
		t.Errorf("file start line = %d, want 42", file.GetStartLine())
	}

	// system-out is folded into the description alongside the failure message.
	desc := f.GetDetails().GetDescription()
	if !strings.Contains(desc, "sending POST /login") {
		t.Errorf("description missing system-out: %q", desc)
	}
	if !strings.Contains(desc, "retrying request") {
		t.Errorf("description missing system-err: %q", desc)
	}
	if !strings.Contains(desc, "admin login rejected") {
		t.Errorf("description missing failure contents: %q", desc)
	}
}

func TestConvertBareTestsuite(t *testing.T) {
	xml := []byte(`<testsuite name="s"><testcase name="t"><failure message="boom"/></testcase></testsuite>`)
	docs, err := junit.Convert(xml)
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("want 1 document, got %d", len(docs))
	}
}
