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
