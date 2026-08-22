// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import "testing"

// The curated Infrastructure as Code forms.
//
// All three are path-shaped: the whole input is a file or a directory, there is
// no credential, and what a curated form owes them is an argument slot that
// says what the argument is. They arrived in forms_misc_test.go, whose list
// spanned five categories at once; see forms_filing_test.go for the rule that
// keeps them here.
//
// terraform is curated in cli/launcher/forms/forms_iac.go and is exercised in
// parsecli_test.go, where its three sub-command shapes are the point.

// iacConnectors are the Infrastructure as Code connectors this file covers.
var iacConnectors = filedHere("ansible", "bicep", "cloudformation")

// Every connector this file claims has a spec, and exactly one.
func TestIaCSpecsAreRegisteredExactlyOnce(t *testing.T) {
	for _, name := range iacConnectors {
		if _, ok := formSpecs[name]; !ok {
			t.Errorf("%s has no registered spec", name)
		}
		if containsString(duplicateSpecs, name) {
			t.Errorf("%s was registered twice, so two files claim it", name)
		}
	}
}

// Each of these asks for one path, and says what the path should point at. See
// assertPathShapedConnector for what "says" means and why the derived slot is
// not enough.
func TestIaCConnectorsNameTheirPath(t *testing.T) {
	for _, name := range iacConnectors {
		t.Run(name, func(t *testing.T) {
			assertPathShapedConnector(t, name)
		})
	}
}
