// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	"slices"
	"strings"
	"testing"

	tuiform "go.mondoo.com/cnspec/cli/tui/form"
	"go.mondoo.com/mql/providers-sdk/v1/plugin"

	"go.mondoo.com/cnspec/cli/launcher/source"
)

// Every test in this file injects the environment.
//
// The variables the ambient class reads are exactly the ones a developer
// running these tests is most likely to have exported, so a test that consulted
// the real environment would pass or fail depending on whose machine it ran on
// -- and would be reading that developer's actual tokens to do it. The
// injected lookup answers from a map and nothing else.
func withAmbientEnv(t *testing.T, vars map[string]string) {
	t.Helper()
	prev := source.SetAmbientEnv(func(name string) (string, bool) {
		v, ok := vars[name]
		return v, ok
	})
	t.Cleanup(func() { source.SetAmbientEnv(prev) })
}

// ambientConnector is a stand-in for an installed provider, so these tests say
// the same thing on a machine with no providers installed at all.
func ambientConnector(name string, flags ...plugin.Flag) Connector {
	return Connector{
		Provider: name, Name: name, Use: name, Category: catSaaS,
		Installed: true,
		Flags:     append([]plugin.Flag{{Long: "token", Type: plugin.FlagType_String}}, flags...),
	}
}

const ambientToken = "tok-<PLACEHOLDER-not-a-real-secret>"

// The one thing a credential readout exists to say is where the token came
// from. The one thing it must never say is what the token is.
func TestAmbientReadoutNamesTheVariableNeverTheToken(t *testing.T) {
	withAmbientEnv(t, map[string]string{"GITLAB_TOKEN": ambientToken})

	f := newForm(ambientConnector("gitlab"))
	fd := fieldByLabel(t, f, "credential")

	if fd.Kind != tuiform.KindCredentialState {
		t.Fatalf("the readout is a %v, want a credential-state field", fd.Kind)
	}
	if fd.Value() != "GITLAB_TOKEN" {
		t.Errorf("readout = %q, want the variable that supplied the token", fd.Value())
	}
	if !fd.IsSet() {
		t.Error("a token was found but the readout reports itself unset")
	}
	if strings.Contains(fd.Display(), ambientToken) {
		t.Fatalf("the readout displayed the token: %q", fd.Display())
	}
	// Nothing else on the form may be holding it either: the launcher reads
	// whether the variable is set, never what it holds.
	for _, g := range f.Fields() {
		if strings.Contains(g.Value(), ambientToken) {
			t.Fatalf("%q holds the token itself: %q", g.Label, g.Value())
		}
	}
}

// An empty row and a row with nothing to fill in look identical, which is the
// failure this widget exists to end.
func TestAmbientReadoutSaysWhatToExportWhenNothingIsSet(t *testing.T) {
	withAmbientEnv(t, nil)

	f := newForm(ambientConnector("cloudflare"))
	fd := fieldByLabel(t, f, "credential")
	if fd.IsSet() {
		t.Fatalf("nothing is exported but the readout says %q", fd.Value())
	}

	hint := credentialHint(*fd)
	if !strings.Contains(hint, "CLOUDFLARE_TOKEN") {
		t.Errorf("the hint should name the variable to export, got %q", hint)
	}
	if !strings.Contains(hint, "paste") {
		t.Errorf("the hint should offer the other way in, got %q", hint)
	}
}

// okta reads three levels in order -- --token, OKTA_API_TOKEN, then OKTA_TOKEN
// -- so the readout has to name the one the provider will actually reach for.
func TestOktaReadoutFollowsTheProvidersPrecedence(t *testing.T) {
	both := map[string]string{
		"OKTA_API_TOKEN": ambientToken,
		"OKTA_TOKEN":     ambientToken,
	}
	for _, tc := range []struct {
		name string
		env  map[string]string
		want string
	}{
		{"both set", both, "OKTA_API_TOKEN"},
		{"only the fallback", map[string]string{"OKTA_TOKEN": ambientToken}, "OKTA_TOKEN"},
		{"only the first", map[string]string{"OKTA_API_TOKEN": ambientToken}, "OKTA_API_TOKEN"},
		{"neither", nil, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			withAmbientEnv(t, tc.env)
			f := newForm(oktaTestConnector())
			if got := fieldByLabel(t, f, "credential").Value(); got != tc.want {
				t.Errorf("readout = %q, want %q", got, tc.want)
			}
		})
	}
}

func oktaTestConnector() Connector {
	return ambientConnector("okta",
		plugin.Flag{Long: "organization", Type: plugin.FlagType_String,
			Desc: "The domain of the Okta organization to scan"},
	)
}

// The launcher must not prefill an organization it composed out of nothing.
// oktaOrganization itself is checked in cli/launcher/source; this is the form
// honouring it.
func TestOktaOrganizationIsNotPrefilledFromNothing(t *testing.T) {
	withAmbientEnv(t, nil)
	fd := fieldByLabel(t, newForm(oktaTestConnector()), "organization")
	if fd.Value() != "" {
		t.Fatalf("the organization field was prefilled with %q", fd.Value())
	}
	if fd.Prefilled() != "" {
		t.Errorf("nothing was filled in, but it claims %q", fd.Prefilled())
	}
}

// What the provider really will use is worth showing, and an editable field is
// where a user gets to notice it.
func TestOktaOrganizationReachesTheForm(t *testing.T) {
	withAmbientEnv(t, map[string]string{"OKTA_ORG_NAME": "dev-123", "OKTA_BASE_URL": "okta.com"})
	f := newForm(oktaTestConnector())
	fd := fieldByLabel(t, f, "organization")
	if fd.Value() != "dev-123.okta.com" {
		t.Fatalf("organization = %q, want the composed domain", fd.Value())
	}
	if fd.Prefilled() == "" {
		t.Error("a value the user did not type must say where it came from")
	}
	// It is an ordinary flag, so it still travels on the command line.
	if !slices.Contains(f.Args(), "--organization") {
		t.Errorf("the composed organization did not reach the command: %v", f.Args())
	}
}

// A typed value is never overwritten by the environment.
func TestOktaOrganizationDoesNotOverwriteTheUser(t *testing.T) {
	withAmbientEnv(t, map[string]string{"OKTA_ORG_NAME": "dev-123", "OKTA_BASE_URL": "okta.com"})
	f := newForm(oktaTestConnector())
	fieldByLabel(t, f, "organization").SetValue("chosen.okta.com")
	resolveSources(&f)
	if got := fieldByLabel(t, f, "organization").Value(); got != "chosen.okta.com" {
		t.Fatalf("organization = %q, want what the user typed", got)
	}
}

// The flag beats the environment in every one of these providers, so a readout
// that named a variable while a pasted token was about to be used would be
// confidently wrong about the one fact it states.
func TestPastedTokenBeatsTheEnvironment(t *testing.T) {
	withAmbientEnv(t, map[string]string{"SLACK_TOKEN": ambientToken})

	f := newForm(ambientConnector("slack"))
	if got := fieldByLabel(t, f, "credential").Value(); got != "SLACK_TOKEN" {
		t.Fatalf("readout = %q, want the variable", got)
	}

	paste := fieldByLabel(t, f, "token")
	if paste.Kind != fieldPaste {
		t.Fatalf("the token flag is a %v, want a paste box", paste.Kind)
	}
	if !paste.Secret {
		t.Fatal("the paste box is not marked secret, so its value could reach argv")
	}
	paste.SetValue("pasted-" + ambientToken)

	// resolveSources is what the model runs after every keystroke.
	resolveSources(&f)
	if got := fieldByLabel(t, f, "credential").Value(); got != "pasted" {
		t.Errorf("readout = %q, want it to follow the pasted token", got)
	}
	if strings.Contains(fieldByLabel(t, f, "credential").Display(), ambientToken) {
		t.Fatal("the readout displayed the pasted token")
	}

	// Clearing it hands the row back to the environment.
	fieldByLabel(t, f, "token").SetValue("")
	resolveSources(&f)
	if got := fieldByLabel(t, f, "credential").Value(); got != "SLACK_TOKEN" {
		t.Errorf("readout = %q after clearing the paste box, want the variable back", got)
	}
}

// A form can be rebuilt underneath the user when a provider install lands, and
// the readout on the new form was derived before the old form's pasted token
// was carried onto it.
func TestReadoutFollowsATokenCarriedOntoARebuiltForm(t *testing.T) {
	withAmbientEnv(t, map[string]string{"GITLAB_TOKEN": ambientToken})

	old := newForm(ambientConnector("gitlab"))
	fieldByLabel(t, old, "token").SetValue("pasted-" + ambientToken)

	rebuilt := newForm(ambientConnector("gitlab"))
	carryOver(&rebuilt, old)

	if got := fieldByLabel(t, rebuilt, "credential").Value(); got != "pasted" {
		t.Errorf("readout = %q after a rebuild carried the token over, want it to follow", got)
	}
}

// The readout is a launcher-owned field with no flag, which is precisely the
// shape that used to be emitted as a positional argument.
func TestAmbientReadoutNeverReachesTheCommandLine(t *testing.T) {
	withAmbientEnv(t, map[string]string{"HCLOUD_TOKEN": ambientToken})

	c := ambientConnector("hetzner")
	f := newForm(c)
	fieldByLabel(t, f, "token").SetValue(ambientToken)

	for _, a := range f.Args() {
		if strings.Contains(a, "HCLOUD_TOKEN") || strings.Contains(a, ambientToken) {
			t.Fatalf("the credential reached the command line: %v", f.Args())
		}
	}

	assertCredentialReachesTheProvider(t, c, f, "token", ambientToken)
}

// A pasted token reaches the provider, for every ambient credential that
// offers a box to paste into.
//
// This used to check that the variable the launcher would export was one the
// provider reads, which was the right check while the launcher had to name one.
// A paste box is now indistinguishable from typing into the field it sits over,
// which is the point of the box.
func TestEveryPasteBoxReachesTheProvider(t *testing.T) {
	withAmbientEnv(t, nil)
	checked := 0
	for _, a := range source.AmbientCredentials() {
		if a.Flag == "" {
			continue
		}
		c := ambientConnector(a.Connector)
		f := newForm(c)
		i := f.IndexOfFlag(a.Flag)
		if i < 0 {
			t.Errorf("%s: no --%s field to paste into", a.Connector, a.Flag)
			continue
		}
		f.Fields()[i].SetValue(ambientToken)
		checked++

		t.Run(a.Connector+"/"+a.Flag, func(t *testing.T) {
			assertCredentialReachesTheProvider(t, c, f, a.Flag, ambientToken)
		})
	}
	if checked == 0 {
		t.Fatal("no paste box was checked, so this proved nothing")
	}
}

// digitalocean's Spaces credentials are env-only: no flag carries any of them,
// so they are reported and never collected. Every credential now reaches the
// provider as a flag value, over its own ParseCLI, so a box over a value with
// no flag would collect something with nowhere to hand it to.
func TestDigitalOceanSpacesIsReportOnly(t *testing.T) {
	withAmbientEnv(t, map[string]string{
		"DIGITALOCEAN_TOKEN":         ambientToken,
		"DIGITALOCEAN_SPACES_KEY":    "key",
		"DIGITALOCEAN_SPACES_SECRET": "secret",
	})

	c := ambientConnector("digitalocean")
	f := newForm(c)

	spaces := fieldByLabel(t, f, "spaces keys")
	if spaces.Flag != "" || spaces.Special == "" {
		t.Fatalf("the spaces readout carries a flag (%q), so it is not report-only", spaces.Flag)
	}
	if !spaces.IsSet() {
		t.Fatal("both spaces variables are set but the readout says nothing was found")
	}
	if !strings.Contains(spaces.Value(), "DIGITALOCEAN_SPACES_KEY") {
		t.Errorf("spaces readout = %q, want the variables it found", spaces.Value())
	}

	// Nothing typed, so nothing to carry.
	if got := deliveryFor(f); got != deliverPlain {
		t.Errorf("delivery = %v with nothing typed, want the plain command line", got)
	}
	_ = c
	if got := len(f.Secrets()); got != 0 {
		t.Errorf("%d secret fields hold a value, want none: the readouts are not secrets", got)
	}
}

// A partial set is not a credential: the provider needs both, and reporting
// "found" off the first one would be confidently wrong.
func TestDigitalOceanSpacesNeedsBothVariables(t *testing.T) {
	withAmbientEnv(t, map[string]string{"DIGITALOCEAN_SPACES_KEY": "key"})

	f := newForm(ambientConnector("digitalocean"))
	fd := fieldByLabel(t, f, "spaces keys")
	if fd.IsSet() {
		t.Fatalf("half the credential reads as present: %q", fd.Value())
	}
	if !strings.Contains(credentialHint(*fd), "DIGITALOCEAN_SPACES_SECRET") {
		t.Errorf("the hint should name what is missing, got %q", credentialHint(*fd))
	}
}

// Inserting the readout ahead of a field must not re-point the selectors that
// make a form guided: they hold plain indices into the field list.
func TestInsertingAReadoutKeepsSelectorsPointing(t *testing.T) {
	withAmbientEnv(t, nil)

	// github's curated form leads with a kind/name pair, and the credential
	// section comes after it.
	f := newForm(githubConnector())
	name := fieldByLabel(t, f, "name")
	kindAt := -1
	for i := range f.Fields() {
		if f.Fields()[i].Label == "kind" {
			kindAt = i
		}
	}
	if kindAt < 0 {
		t.Fatal("github lost its kind selector")
	}
	if name.DependsOn != kindAt {
		t.Fatalf("name depends on field %d, want the kind selector at %d", name.DependsOn, kindAt)
	}

	// And a synthetic case where the readout really does land ahead of a
	// dependent field.
	g := tuiform.New("", []field{
		tuiform.NewField(tuiform.Decl{Label: "a"}),
		tuiform.NewField(tuiform.Decl{Label: "b", ShowIf: []string{"x"}, DependsOn: 0}),
	})
	g.InsertField(0, tuiform.NewField(tuiform.Decl{Label: "readout", Kind: tuiform.KindCredentialState}))
	if got := g.Fields()[2].DependsOn; got != 1 {
		t.Errorf("dependsOn = %d after an insert ahead of it, want 1", got)
	}
	if g.Fields()[0].DependsOn != 0 {
		t.Errorf("the inserted field picked up a dependency: %d", g.Fields()[0].DependsOn)
	}
}

// The declarations are the whole of this class, so what they must satisfy is
// checked once over all of them rather than remembered per provider.
func TestAmbientDeclarationsAreConsistent(t *testing.T) {
	withAmbientEnv(t, nil)
	seen := map[string]bool{}

	for _, a := range source.AmbientCredentials() {
		if a.Connector == "" || len(a.Env) == 0 {
			t.Errorf("%+v: an ambient credential needs a connector and a variable", a)
			continue
		}
		key := a.Connector + "/" + a.Name
		if seen[key] {
			t.Errorf("%s: declared twice, so one readout would overwrite the other", key)
		}
		seen[key] = true

		if a.Source != "" {
			s, ok := sourceByID(a.Source)
			if !ok {
				t.Errorf("%s: source %q is not registered", key, a.Source)
			} else if s.Class != ClassAmbient {
				t.Errorf("%s: source %q is class %v, want ClassAmbient", key, a.Source, s.Class)
			}
		}

		// The readout is launcher-owned, and the allowlist in secrets_test.go
		// is where that is granted deliberately.
		if !launcherOwnedFields[a.Special()] {
			t.Errorf("%s: %q is not in launcherOwnedFields", key, a.Special())
		}

		// args() tracks field visibility by label, so two fields sharing one
		// would make a hidden field visible.
		f := newForm(ambientConnector(a.Connector))
		labels := map[string]int{}
		for _, fd := range f.Fields() {
			labels[fd.Label]++
		}
		for label, n := range labels {
			if n > 1 {
				t.Errorf("%s: %d fields labelled %q", key, n, label)
			}
		}
	}
}
