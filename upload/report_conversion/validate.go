// Copyright Mondoo, Inc. 2026
// SPDX-License-Identifier: BUSL-1.1

package report_conversion

import (
	"fmt"

	"go.mondoo.com/mql/v13/providers-sdk/v1/upstream/fex"
)

// Validate reports whether a FindingDocument is "clean": well-formed with the
// fields the platform needs to ingest it. It is the reusable invariant behind the
// converter test harness (AssertClean) and can also gate uploads before they are
// sent. It intentionally checks only the core required fields — richer,
// converter-specific expectations (severity present, evidence attached) belong in
// each converter's own tests.
func Validate(doc *fex.FindingDocument) error {
	if doc == nil {
		return fmt.Errorf("finding document is nil")
	}
	vex, fexDoc := doc.GetVex(), doc.GetFex()
	switch {
	case vex != nil && fexDoc != nil:
		return fmt.Errorf("finding document has both vex and fex set")
	case vex != nil:
		return validateVex(vex)
	case fexDoc != nil:
		return validateFex(fexDoc)
	default:
		return fmt.Errorf("finding document has neither vex nor fex set")
	}
}

func validateVex(v *fex.VulnerabilityExchange) error {
	if v.GetId() == "" {
		return fmt.Errorf("vex: id (CVE/advisory) is required")
	}
	if v.GetSummary() == "" {
		return fmt.Errorf("vex %q: summary is required", v.GetId())
	}
	if v.GetSource().GetName() == "" {
		return fmt.Errorf("vex %q: source name is required", v.GetId())
	}
	if v.GetStatus() == fex.Status_STATUS_UNSPECIFIED {
		return fmt.Errorf("vex %q: status is required", v.GetId())
	}
	return nil
}

func validateFex(f *fex.FindingExchange) error {
	if f.GetId() == "" {
		return fmt.Errorf("fex: id is required")
	}
	if f.GetSummary() == "" {
		return fmt.Errorf("fex %q: summary is required", f.GetId())
	}
	if f.GetSource().GetName() == "" {
		return fmt.Errorf("fex %q: source name is required", f.GetId())
	}
	if f.GetStatus() == fex.Status_STATUS_UNSPECIFIED {
		return fmt.Errorf("fex %q: status is required", f.GetId())
	}
	if f.GetDetails() == nil {
		return fmt.Errorf("fex %q: details are required", f.GetId())
	}
	if f.GetDetails().GetCategory() == fex.FindingDetail_CATEGORY_UNSPECIFIED {
		return fmt.Errorf("fex %q: details.category is required", f.GetId())
	}
	return nil
}
