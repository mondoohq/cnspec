// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import "testing"

// The Developer & Supply Chain forms.
//
// Three connectors, and no two of them have the same shape: sbom is a path,
// depsdev is a path that is genuinely optional, and artifactory is the one
// connector here with a credential and no spec at all. They arrived in
// forms_misc_test.go, whose list spanned five categories at once; see
// forms_filing_test.go for the rule that keeps them here.

// devConnectors are the Developer & Supply Chain connectors this file covers.
var devConnectors = filedHere("artifactory", "depsdev", "sbom")

// sbom's whole input is a path, and a curated form owes it an argument slot
// that says what the argument is. sbom is the case that makes the point: its
// usage string is not in the recorded metadata, so the derived slot asks for
// "argument 1".
func TestSbomNamesItsFile(t *testing.T) {
	assertPathShapedConnector(t, "sbom")
}

// depsdev is the only connector here whose target is genuinely optional, and
// the form has to keep it that way: with no go.mod it answers questions about
// individual packages.
func TestDepsdevTargetStaysOptionalAndIsAskedOnce(t *testing.T) {
	_, f := formFor(t, "depsdev")

	pos := positionalFields(&f)
	if len(pos) != 1 {
		t.Fatalf("depsdev has %d argument slots, want one: %v", len(pos), fieldLabels(f))
	}
	if pos[0].Required {
		t.Error("the go.mod path is required, but depsdev runs without one")
	}
	if hasFlagField(f, "path") {
		t.Error("--path is offered as well as the positional; the same file is asked for twice")
	}
	if err := f.Validate(); err != nil {
		t.Errorf("depsdev refused to launch with no target: %v", err)
	}
	if got := f.Args(); len(got) != 0 {
		t.Errorf("an empty depsdev form produced %v", got)
	}
	f.Fields()[0].SetValue("./go.mod")
	if got := f.Args(); len(got) != 1 || got[0] != "./go.mod" {
		t.Errorf("args = %v, want just the path", got)
	}
}

// artifactory: no spec, but its credential still has to be deliverable, or a
// user who installs the provider gets a form that refuses to launch. The shape
// checked is what the generic layer builds from the connector's own flags, with
// nothing curated over it.
//
// This used to build the connector by hand from artifactory 13.1.0's flag list,
// because the recorded artifact did not carry it -- which was the reason it had
// no spec. The artifact carries it now, so the connector is read from there
// like every other one, and the flag names are the provider's rather than the
// test author's.
func TestArtifactoryCredentialsHaveARoute(t *testing.T) {
	if _, curated := formSpecs["artifactory"]; curated {
		t.Fatal("artifactory grew a spec; drop its registerUncurated call and let " +
			"TestEverySpecNamesRealFlags check it")
	}

	c := connectorFor(t, "artifactory")
	for _, flag := range []string{"token", "api-key"} {
		t.Run(flag, func(t *testing.T) {
			f := newForm(c)
			fd := fieldByLabel(t, f, flag)
			if !fd.Secret {
				t.Fatalf("--%s is not classified as a secret", flag)
			}
			fd.SetValue(sentinel)
			assertCredentialReachesTheProvider(t, c, f, flag, sentinel)
		})
	}
}
