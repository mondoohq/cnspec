// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package source

import (
	"testing"
)

// The environment is read through the injected lookup, never directly, or a
// test would be reading the developer's own tokens.
func TestAmbientStateReadsOnlyWhatItIsGiven(t *testing.T) {
	a := AmbientCredential{Env: []string{"A", "B"}}
	if got := a.present(mapEnv(map[string]string{"B": "x"})); len(got) != 1 || got[0] != "B" {
		t.Errorf("present = %v, want [B]", got)
	}
	// A variable that is set but empty is not a credential.
	if got := a.present(mapEnv(map[string]string{"A": "   "})); len(got) != 0 {
		t.Errorf("a blank variable read as present: %v", got)
	}
	if got := a.present(mapEnv(nil)); len(got) != 0 {
		t.Errorf("present = %v with nothing set", got)
	}
}

// A partial set is not a credential either: with All, the provider needs every
// variable, and reporting "found" off the first one would be confidently wrong.
func TestAllOrNothingCredentialsNeedEveryVariable(t *testing.T) {
	a := AmbientCredential{Env: []string{"A", "B"}, All: true}
	if got := a.present(mapEnv(map[string]string{"A": "x"})); len(got) != 0 {
		t.Errorf("half a credential read as present: %v", got)
	}
	got := a.present(mapEnv(map[string]string{"A": "x", "B": "y"}))
	if len(got) != 2 {
		t.Fatalf("present = %v, want both", got)
	}
	if chosen := a.chosen(got); chosen != "A + B" {
		t.Errorf("chosen = %q, want both named", chosen)
	}
}

// okta's ParseCLI composes OKTA_ORG_NAME + "." + OKTA_BASE_URL and then guards
// with `if organization == ""`. With both unset the composition is the literal
// ".", the guard never fires, and the connector proceeds with an organization
// of "." -- so the launcher special-cases both-empty rather than mirroring the
// composition faithfully. A half-composed value *is* mirrored, because that one
// the provider really will use.
func TestOktaOrganizationIsComposedFromTheEnvironment(t *testing.T) {
	for _, tc := range []struct {
		env  map[string]string
		want string
	}{
		{nil, ""},
		{map[string]string{"OKTA_ORG_NAME": "", "OKTA_BASE_URL": ""}, ""},
		{map[string]string{"OKTA_ORG_NAME": "  ", "OKTA_BASE_URL": "\t"}, ""},
		{map[string]string{"OKTA_ORG_NAME": "dev-123", "OKTA_BASE_URL": "okta.com"}, "dev-123.okta.com"},
		{map[string]string{"OKTA_ORG_NAME": " dev-123 ", "OKTA_BASE_URL": "okta.com"}, "dev-123.okta.com"},
		{map[string]string{"OKTA_ORG_NAME": "dev-123"}, "dev-123."},
		{map[string]string{"OKTA_BASE_URL": "okta.com"}, ".okta.com"},
	} {
		if got := oktaOrganization(mapEnv(tc.env)); got != tc.want {
			t.Errorf("oktaOrganization(%v) = %q, want %q", tc.env, got, tc.want)
		}
	}
}

// A paste box needs a flag to paste into.
//
// This used to check a second thing as well -- that the variable a pasted value
// would travel in was one the provider actually reads -- because a pasted token
// was delivered by exporting a variable, and inventing the name sent it
// somewhere nothing read it. A pasted value now travels the same way a typed
// one does, through the connector's own ParseCLI, so the only claim left is
// that there is a flag for it to be the value of.
func TestEveryPasteBoxHasAFlagToPasteInto(t *testing.T) {
	checked := 0
	for _, a := range ambientCredentials {
		if a.All {
			// A readout over a set of variables collects nothing: there is no
			// paste box, deliberately. See the digitalocean Spaces note.
			continue
		}
		checked++
		if a.Flag == "" {
			t.Errorf("%s: offers a paste box with no flag to paste into", a.Connector)
		}
	}
	if checked == 0 {
		t.Fatal("no ambient credential was checked, so this proved nothing")
	}
}

func mapEnv(vars map[string]string) EnvLookup {
	return func(name string) (string, bool) {
		v, ok := vars[name]
		return v, ok
	}
}
