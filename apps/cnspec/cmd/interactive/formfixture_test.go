// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	"sort"
	"strings"
	"testing"

	"go.mondoo.com/mql/providers-sdk/v1/plugin"
)

// The fixtures and accessors every curated-form test is written against.
//
// There used to be one set per file: nine `xForm(t, name)` constructors that
// differed only in the words of their fatal message, two field-finders that
// differed only in whether they took a pointer, two "fill this field" helpers
// that differed only in whether they addressed it by flag or by label, and two
// positional collectors of which one sorted and the other did not. Nine copies
// of a helper is nine places a fix has to be made and eight places it can be
// forgotten -- and the sorted/unsorted pair was the version of that already
// waiting to happen, since the two disagreed about a form whose fields arrive
// out of argument order.
//
// One set, used by all of them. The two that remain distinct are distinct for a
// reason stated where they are defined.

// connectorFor rebuilds one connector from the recorded metadata rather than
// from whatever is installed, so these tests say the same thing where CI runs
// them: with PROVIDERS_PATH pointed at an empty directory.
//
// The snapshot is preferred whenever it has an entry, and today it has one for
// every connector with a spec. A connector it does not carry falls back to the
// fixtures below; see staticConnectorFixtures for what that fallback is for and
// why nothing currently reaches it.
func connectorFor(t *testing.T, name string) Connector {
	t.Helper()
	if snap, ok := snapshotByName(t)[name]; ok {
		return snap.connector()
	}
	c, ok := staticConnectorFixtures[name]
	if !ok {
		t.Fatalf("%s: no entry in %s and no fixture to build a form from",
			name, connectorSnapshotPath)
	}
	// The category is computed rather than declared, for the reason
	// connectorSnapshot gives for not recording one: categorize() owns it, and a
	// second copy is a second thing to keep in step. The fixtures below carried
	// one until Identity & Access was split out of SaaS, after which four of the
	// six claimed a category the launcher had stopped showing them under.
	c.Category = categorize(c.Provider, c.Name)
	t.Logf("%s: not in the snapshot, so this ran against the recorded fixture", name)
	return c
}

// formFor builds a connector's real curated form -- the generic layer from its
// declared metadata, then whatever cli/launcher/forms registered over it.
func formFor(t *testing.T, name string) (Connector, form) {
	t.Helper()
	c := connectorFor(t, name)
	return c, newForm(c)
}

// hermeticFormFor is formFor with the ambient environment emptied first.
//
// It is a separate call rather than the default because it is not free: a test
// that asserts what a credential readout says has to control the environment,
// and a test that asserts what the *provider* falls back to must not, or it
// asserts against a fixture instead of against the machine. Which of the two a
// test is doing is a decision, so it is made at the call site.
func hermeticFormFor(t *testing.T, name string) (Connector, form) {
	t.Helper()
	withAmbientEnv(t, nil)
	return formFor(t, name)
}

// findFieldByFlag returns the field emitting a flag, or nil when the overlay never
// put one on the form.
func findFieldByFlag(f form, flag string) *field {
	for i := range f.Fields() {
		if f.Fields()[i].Flag == flag {
			return &f.Fields()[i]
		}
	}
	return nil
}

// fieldByFlag is findFieldByFlag for the tests that cannot continue without it.
func fieldByFlag(t *testing.T, f form, flag string) *field {
	t.Helper()
	fd := findFieldByFlag(f, flag)
	if fd == nil {
		t.Fatalf("no field for --%s in %v", flag, fieldLabels(f))
	}
	return fd
}

// hasFlagField reports whether a flag reached the screen at all.
func hasFlagField(f form, flag string) bool { return findFieldByFlag(f, flag) != nil }

// sectionOf reports which section a flag's field landed in, and whether it is
// on the screen at all.
func sectionOf(f form, flag string) (string, bool) {
	if fd := findFieldByFlag(f, flag); fd != nil {
		return fd.Section, true
	}
	return "", false
}

// positionalFields are the form's argument slots, in argument order: the fields
// with no flag and no launcher-owned marker.
//
// The order is asserted rather than assumed. applySpec sorts the fields it
// built, so a form usually arrives in argument order already -- but "usually"
// is what a test must not depend on, and the unsorted copy of this helper
// agreed with the sorted one only for as long as that held.
func positionalFields(f *form) []*field {
	var out []*field
	for i := range f.Fields() {
		if f.Fields()[i].Flag == "" && f.Fields()[i].Special == "" {
			out = append(out, &f.Fields()[i])
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Pos < out[j].Pos })
	return out
}

// setFlag fills the field emitting a flag, failing if the overlay never put it
// on the form. "Reachable" is the property being asserted: a flag the overlay
// silently dropped has no field to find.
func setFlag(t *testing.T, f *form, flag, value string) *field {
	t.Helper()
	i := f.IndexOfFlag(flag)
	if i < 0 {
		t.Fatalf("--%s is not on the form; fields are %v", flag, fieldLabels(*f))
	}
	f.Fields()[i].SetValue(value)
	return &f.Fields()[i]
}

// setLabelled writes a value into the field with this label, which is how a
// positional is addressed -- it has no flag.
func setLabelled(t *testing.T, f *form, label, value string) {
	t.Helper()
	for i := range f.Fields() {
		if f.Fields()[i].Label == label {
			f.Fields()[i].SetValue(value)
			return
		}
	}
	t.Fatalf("no field labelled %q in %v", label, fieldLabels(*f))
}

// staticConnectorFixtures record what six connectors declare once their
// provider is installed, read from ~/.config/mondoo/providers/<name>/<name>.json
// after installing each into a scratch PROVIDERS_PATH.
//
// They exist because the recorded artifact did not carry these six: they
// reached the catalog only through the compiled-in static list, which strips
// Flags, so every check that needs a flag had nothing to run against. That is
// no longer true -- the artifact carries all six, connectorFor prefers it
// whenever it has an entry, and TestEverySpecIsCoveredByTheSnapshotGate is
// what proves none of them slipped back out. So nothing reaches these today.
//
// They are kept rather than deleted because the fallback is the only thing
// standing between "this connector left the artifact" and a test suite that
// cannot build a form for it at all. Be clear about what a fixture proves: it
// is written by the same hand as the spec, so it cannot prove the flag names
// are the ones the provider ships -- only the artifact can do that. What it
// proves is everything downstream of the names: that the spec's sections
// resolve, that a filled credential has a delivery route instead of a form that
// refuses to launch, and that no secret reaches argv.
var staticConnectorFixtures = map[string]Connector{
	// auth0 13.0.0
	"auth0": {
		Provider: "auth0", Name: "auth0", Use: "auth0",
		Installed: true,
		Flags: []plugin.Flag{
			{Long: "domain", Type: plugin.FlagType_String, Desc: "Auth0 tenant domain (e.g. your-tenant.us.auth0.com)"},
			{Long: "client-id", Type: plugin.FlagType_String, Desc: "Auth0 machine-to-machine application client ID"},
			{Long: "client-secret", Type: plugin.FlagType_String, Desc: "Auth0 machine-to-machine application client secret"},
		},
	},
	// bitwarden 13.0.0
	"bitwarden": {
		Provider: "bitwarden", Name: "bitwarden", Use: "bitwarden",
		Installed: true,
		Flags: []plugin.Flag{
			{Long: "client-id", Type: plugin.FlagType_String, Desc: "Bitwarden organization client ID (e.g. organization.<uuid>)"},
			{Long: "client-secret", Type: plugin.FlagType_String, Desc: "Bitwarden organization client secret"},
			{Long: "api-url", Type: plugin.FlagType_String, Desc: "Bitwarden Public API base URL, for self-hosted deployments"},
			{Long: "identity-url", Type: plugin.FlagType_String, Desc: "Bitwarden identity token URL, for self-hosted deployments"},
		},
	},
	// dropbox 13.0.1
	"dropbox": {
		Provider: "dropbox", Name: "dropbox", Use: "dropbox",
		Installed: true,
		Flags: []plugin.Flag{
			{Long: "token", Type: plugin.FlagType_String, Desc: "Dropbox Business team access token"},
		},
	},
	// jumpcloud 13.0.0
	"jumpcloud": {
		Provider: "jumpcloud", Name: "jumpcloud", Use: "jumpcloud",
		Installed: true,
		Flags: []plugin.Flag{
			{Long: "api-key", Type: plugin.FlagType_String, Desc: "JumpCloud API key"},
			{Long: "org-id", Type: plugin.FlagType_String, Desc: "JumpCloud organization id (required for multi-tenant API keys)"},
		},
	},
	// keycloak 13.0.0
	"keycloak": {
		Provider: "keycloak", Name: "keycloak", Use: "keycloak",
		Installed: true,
		Flags: []plugin.Flag{
			{Long: "url", Type: plugin.FlagType_String, Desc: "Base URL of the Keycloak server, for example https://keycloak.example.com"},
			{Long: "realm", Type: plugin.FlagType_String, Desc: "Scope the scan to a single realm instead of every realm the credentials can read"},
			{Long: "auth-realm", Type: plugin.FlagType_String, Desc: "Realm the token is requested from (defaults to master for a user, or the scanned realm for a service account)"},
			{Long: "client-id", Type: plugin.FlagType_String, Desc: "Client the token is requested for (defaults to admin-cli for a user)"},
			{Long: "client-secret", Type: plugin.FlagType_String, Desc: "Secret of a confidential client, which selects service account authentication"},
			{Long: "username", Type: plugin.FlagType_String, Desc: "Admin user to authenticate as, which selects password authentication"},
			{Long: "password", Type: plugin.FlagType_String, Desc: "Password of the admin user"},
			{Long: "ca-cert", Type: plugin.FlagType_String, Desc: "Certificate authority to trust for the server certificate, either the PEM itself or a path to it"},
		},
		Discovery: []string{"all", "auto", "realms"},
	},
	// zoom 13.0.0
	"zoom": {
		Provider: "zoom", Name: "zoom", Use: "zoom",
		Installed: true,
		Flags: []plugin.Flag{
			{Long: "account-id", Type: plugin.FlagType_String, Desc: "Zoom account ID"},
			{Long: "client-id", Type: plugin.FlagType_String, Desc: "Zoom Server-to-Server OAuth app client ID"},
			{Long: "client-secret", Type: plugin.FlagType_String, Desc: "Zoom Server-to-Server OAuth app client secret"},
		},
	},
}

// fillEveryField answers every field on a form the way a user would, writing
// secret into each credential, and returns how many credentials it found.
//
// It is shared because it was not: two category files each grew their own copy
// of this loop and the copies had drifted. One filled a picker with its first
// option and the other with the word "placeholder", which is not a value any
// strict list would accept; one skipped a launcher-owned readout by its Kind
// and the other by its Special marker, so each typed into fields the other left
// alone. Neither difference was a decision -- they are the same sweep, and this
// is the strict side of each.
//
// The launcher-owned rows are skipped by both tests: a credential readout holds
// the name of a variable rather than a value, and a Special field is the
// launcher's own question, so typing into either asserts nothing about the
// connector.
func fillEveryField(f *form, secret string) int {
	secrets := 0
	for i := range f.Fields() {
		fd := &f.Fields()[i]
		switch {
		case fd.Kind == fieldCredentialState || fd.Special != "":
			// A launcher-owned row; it is not typed into.
		case fd.Secret:
			fd.SetValue(secret)
			secrets++
		case fd.Kind == fieldBool:
			fd.SetOn(true)
		case fd.Kind == fieldMultiChoice:
			if len(fd.Options) > 0 {
				fd.SetPicks(map[string]bool{fd.Options[0]: true})
			}
		case len(fd.Options) > 0:
			fd.SetValue(fd.Options[0])
		default:
			fd.SetValue("placeholder")
		}
	}
	return secrets
}

// assertPathShapedConnector is the one thing a curated form owes a connector
// whose whole input is a path: an argument slot that says what the argument is.
//
// The derived slot spells the usage string, so an uncurated bicep asks for
// "PATH" and an uncurated sbom -- whose Use the recorded artifact does not
// carry -- asks for "argument 1". Neither tells a reader what to type.
//
// It lives here rather than in one category file because the connectors it
// applies to are in four of them: kustomize is Containers & Kubernetes, ansible
// and bicep and cloudformation are Infrastructure as Code, sbom is Developer &
// Supply Chain. They were one loop over one list while that list was named for
// the person who wrote it.
func assertPathShapedConnector(t *testing.T, name string) {
	t.Helper()
	c, f := formFor(t, name)

	pos := positionalFields(&f)
	if len(pos) != 1 {
		t.Fatalf("%s has %d argument slots, want exactly one: %v",
			name, len(pos), fieldLabels(f))
	}
	fd := pos[0]
	if fd.Section != sectionTarget {
		t.Errorf("the path sits in %q rather than TARGET", fd.Section)
	}
	if !fd.Required {
		t.Errorf("%s declares MinArgs=%d but its path is optional", name, c.MinArgs)
	}
	if fd.Label == "" || fd.Label == "PATH" || strings.HasPrefix(fd.Label, "argument ") {
		t.Errorf("the path slot is still labelled %q", fd.Label)
	}
	if fd.Desc == "" {
		t.Errorf("%s does not say what its path should point at", name)
	}

	// Empty is refused in place rather than launched into a usage error, and a
	// filled one is the whole command.
	if err := f.Validate(); err == nil {
		t.Error("an empty path launched")
	}
	f.Fields()[0].SetValue("/srv/infra")
	if err := f.Validate(); err != nil {
		t.Errorf("a filled path was still refused: %v", err)
	}
	if got := f.Args(); len(got) != 1 || got[0] != "/srv/infra" {
		t.Errorf("args = %v, want just the path", got)
	}
}
