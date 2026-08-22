// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	"errors"
	"strings"
	"testing"

	tuiform "go.mondoo.com/cnspec/cli/tui/form"

	"github.com/charmbracelet/x/ansi"
	"go.mondoo.com/mql/providers-sdk/v1/vault"

	"go.mondoo.com/mql/providers"
	"go.mondoo.com/mql/providers-sdk/v1/plugin"
)

const sentinel = "S3CR3T-must-never-reach-argv"

// This is the test that matters most in the package.
//
// The launcher runs commands by re-executing cnspec. Every argument of that
// re-execution is world-readable through `ps auxww`, so a credential that
// reaches argv is disclosed to every user on the machine. Nothing in the form
// engine may put one there, for any connector, ever.
func TestNoSecretEverReachesArgv(t *testing.T) {
	catalog := BuildCatalog()
	if len(catalog) == 0 {
		t.Skip("no catalog available")
	}

	checked := 0
	for _, c := range catalog {
		if !c.HasFormData() {
			continue
		}
		f := newForm(c)

		// Fill every field: secrets with the sentinel, everything else with
		// something plausible, so nothing is skipped for being empty.
		secretFields := 0
		for i := range f.Fields() {
			fd := &f.Fields()[i]
			switch {
			case fd.Secret:
				fd.SetValue(sentinel)
				secretFields++
			case fd.Kind == fieldBool:
				fd.SetOn(true)
			case fd.Kind == fieldMultiChoice:
				if len(fd.Options) > 0 {
					fd.SetPicks(map[string]bool{fd.Options[0]: true})
				}
			default:
				fd.SetValue("placeholder")
			}
		}
		if secretFields == 0 {
			continue
		}
		checked++

		// The package-wide stand-in provider answers here; see TestMain. What
		// is being checked is the launcher's own assembly over every connector
		// in the catalog, not any one provider's mapping -- that is
		// TestEveryCredentialFieldRoundTrips, which spends a subprocess per
		// connector to do it properly.
		r := launchRequest{form: f}
		plan, err := r.plan(c, scanAction())
		args, env, cleanup := plan.args, plan.env, plan.cleanup
		if cleanup != nil {
			defer cleanup()
		}
		if err != nil {
			// A provider that keeps the credential nowhere refuses, which is
			// the point: better a clear refusal than a scan that runs without
			// the credential the user supplied.
			t.Errorf("%s: %v", c.Name, err)
			continue
		}

		for _, a := range args {
			if strings.Contains(a, sentinel) {
				t.Errorf("%s: secret reached the command line: %q", c.Name, strings.Join(args, " "))
			}
		}
		for _, e := range env {
			if strings.Contains(e, sentinel) {
				t.Errorf("%s: secret reached the child's environment: %q", c.Name, e)
			}
		}
	}

	if checked == 0 {
		t.Skip("no installed connector declares a secret-carrying flag")
	}
	t.Logf("checked %d connectors carrying secrets", checked)
}

// A password typed into a form is the password that gets used.
//
// This was the opposite assertion for a release. ssh declares --ask-pass, so a
// typed password took the prompt route: the launcher dropped the value, put
// --ask-pass on the command line in its place, and the child asked the user for
// the password they had just typed. The whole command was
// `cnspec scan ssh chris@10.0.0.4 --ask-pass`.
//
// Prompting is genuinely the strongest route -- the value never exists outside
// the process that uses it -- and it is still available as a toggle on the
// form. What is not available is substituting it for what somebody typed.
func TestATypedPasswordIsTheOneThatTravels(t *testing.T) {
	c := sshConnector()
	f := newForm(c)
	fieldByLabel(t, f, "user@host").SetValue("chris@10.0.0.4")
	fieldByLabel(t, f, "password").SetValue(sentinel)

	p := withParser(t, &fakeParser{secretFlag: "password"})
	plan, err := (launchRequest{form: f}).plan(c, scanAction())
	if plan.cleanup != nil {
		defer plan.cleanup()
	}
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := p.sentValue("password"); got != sentinel {
		t.Fatalf("the typed password reached the provider as %q", got)
	}
	if containsString(plan.args, "--ask-pass") {
		t.Fatalf("the typed password was replaced by a prompt: %v", plan.args)
	}
	if strings.Contains(strings.Join(plan.args, " ")+strings.Join(plan.env, " "), sentinel) {
		t.Fatal("the secret reached the command line or the environment")
	}
}

// A credential reaches the provider, and nothing else.
func TestTheCredentialReachesTheProviderAndNotArgv(t *testing.T) {
	c := githubConnector()
	f := newForm(c)
	fieldByLabel(t, f, "kind").SetValue("org")
	fieldByLabel(t, f, "name").SetValue("mondoohq")
	fieldByLabel(t, f, "personal access token").SetValue(sentinel)

	p := withParser(t, &fakeParser{secretFlag: "token"})
	plan, err := (launchRequest{form: f}).plan(c, scanAction())
	if plan.cleanup != nil {
		defer plan.cleanup()
	}
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.Join(plan.args, " "), sentinel) {
		t.Fatalf("secret on the command line: %v", plan.args)
	}
	if len(plan.env) != 0 {
		t.Fatalf("the inventory route hands the child no environment, got %v", plan.env)
	}
	// The target used to travel on argv beside the credential's variable. It
	// travels in the inventory now, which means this is a question about what
	// the provider was told rather than about the words the child was given.
	if got := p.args; len(got) != 2 || got[0] != "org" || got[1] != "mondoohq" {
		t.Fatalf("the target was dropped along with the secret: %v", got)
	}
}

// A bool flag named like a prompt is not a prompt, and the form must not offer
// it as one.
//
// mql acts on exactly two kinds. --ask-pass it reads by name and turns into
// --password; anything carrying FlagOption_AskInput it collects into an
// "ask-flags" annotation and prompts for. A third kind exists in the wild --
// clickhousecloud's --ask-secret and weaviate's --ask-api-key are declared as
// plain bools -- and it does nothing at all.
//
// Verified against the shipped binary rather than read off the enum:
//
//	cnspec shell ssh chris@127.0.0.1 --ask-pass </dev/null
//	→ FTL failed to get password error="asking input is only supported
//	  when used with an interactive terminal (TTY)"
//
//	cnspec shell weaviate --host 127.0.0.1 --ask-api-key </dev/null
//	→ status code: 404, error: 404 page not found
//
// The first is a prompt refusing a pipe. The second never asked for anything. A
// toggle labelled "prompt for API key" that leads to an unauthenticated scan is
// worse than an absent row, so the row is absent.
//
// This used to matter twice over: the inert flag also made the launcher take
// the prompt route, leave the credential off argv, and connect with nothing.
// The route is gone; the row is still a lie and is still not drawn.
func TestAPromptFlagNothingActsOnIsNotOffered(t *testing.T) {
	for _, tc := range []struct {
		name    string
		option  plugin.FlagOption
		offered bool
	}{
		{name: "inert bool", option: 0, offered: false},
		{name: "declared AskInput", option: plugin.FlagOption_AskInput, offered: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := Connector{
				Provider: "test", Name: "test-ask", Use: "test", Installed: true,
				Flags: []plugin.Flag{
					{Long: "api-key", Type: plugin.FlagType_String, ConfigEntry: "-"},
					{Long: "ask-api-key", Type: plugin.FlagType_Bool, Option: tc.option},
				},
			}
			f := newForm(c)
			if got := f.IndexOfFlag("ask-api-key") >= 0; got != tc.offered {
				t.Errorf("--ask-api-key offered = %v, want %v", got, tc.offered)
			}
			// The credential itself is still on the form either way, and still
			// classified as a secret: what an inert toggle costs it is a
			// partner that was never going to arrive.
			i := f.IndexOfFlag("api-key")
			if i < 0 {
				t.Fatal("the api-key field was not built")
			}
			f.Fields()[i].Secret = true
			f.Fields()[i].SetValue(sentinel)
			if got := deliveryFor(f); got != deliverInventory {
				t.Errorf("delivery = %v, want the inventory", got)
			}
		})
	}

	// --ask-pass is the exception and stays one: mql reads it by name, so the
	// providers that declare it as a plain bool -- which is all of them -- keep
	// working.
	c := Connector{
		Provider: "test", Name: "test-ask-pass", Use: "test", Installed: true,
		Flags: []plugin.Flag{
			{Long: "password", Type: plugin.FlagType_String, ConfigEntry: "-"},
			{Long: "ask-pass", Type: plugin.FlagType_Bool},
		},
	}
	if newForm(c).IndexOfFlag("ask-pass") < 0 {
		t.Error("--ask-pass is not offered, so nothing can ask the child to prompt")
	}
}

// Without a secret the launcher must stay on the plain command line: routing
// everything through a file would be a pointless regression.
func TestNoSecretMeansNoInventoryFile(t *testing.T) {
	c := awsConnector()
	f := newForm(c)
	fieldByLabel(t, f, "profile").SetValue("prod")

	r := launchRequest{form: f}
	plan, err := r.plan(c, scanAction())
	args, cleanup := plan.args, plan.cleanup
	if cleanup != nil {
		defer cleanup()
	}
	if err != nil {
		t.Fatal(err)
	}
	if containsString(args, "--inventory-file") {
		t.Fatalf("expected a plain command line, got %v", args)
	}
	if strings.Join(args, " ") != "scan aws --profile prod" {
		t.Fatalf("args = %v", args)
	}
}

// The command bar has to show what will really run, so a user can confirm for
// themselves that no secret is on it.
func TestCommandPreviewHidesSecrets(t *testing.T) {
	c := sshConnector()
	f := newForm(c)
	fieldByLabel(t, f, "user@host").SetValue("chris@host")
	fieldByLabel(t, f, "password").SetValue(sentinel)

	preview := (launchRequest{form: f}).preview(c, scanAction())
	if strings.Contains(preview, sentinel) {
		t.Fatalf("command preview leaked the secret: %q", preview)
	}
	// The bar names the file the launch will write, not a path that does not
	// exist yet.
	if !strings.Contains(preview, "--inventory-file") {
		t.Fatalf("preview should show the inventory route, got %q", preview)
	}
}

// A secret field must render masked, never in the clear.
func TestSecretFieldsRenderMasked(t *testing.T) {
	fd := valued(tuiform.Decl{Kind: fieldText, Secret: true}, sentinel)
	if strings.Contains(fd.Display(), sentinel) {
		t.Fatalf("secret field displayed in the clear: %q", fd.Display())
	}
}

// FlagOption_Password is set by only a handful of providers, so the classifier
// carries the weight. These are real flags from the installed set that leave
// Option unset.
func TestClassifierCatchesUnmarkedSecrets(t *testing.T) {
	unmarked := []struct{ connector, flag string }{
		{"azure", "client-secret"},
		{"azure", "certificate-secret"},
		{"github", "token"},
		{"github", "app-private-key-content"},
		{"neon", "token"},
		{"okta", "token"},
		{"alicloud", "access-key-secret"},
		{"cloudflare", "token"},
		{"openstack", "application-credential-secret"},
	}
	for _, u := range unmarked {
		fl := plugin.Flag{Long: u.flag, Type: plugin.FlagType_String}
		if !tuiform.IsSecretFlag(fl) {
			t.Errorf("%s --%s not classified as a secret", u.connector, u.flag)
		}
	}

	// Flags that name where a credential lives, or that toggle a prompt, must
	// stay on the command line -- that is how the provider expects them.
	//
	// Descriptions are carried verbatim here because for one of these the name
	// alone is genuinely ambiguous: github's --app-private-key holds a path,
	// and only its description says so. A provider that ships that flag with no
	// description gets it classified as a secret instead, which fails safe
	// (the value moves into the inventory) but would not connect.
	safe := []struct {
		flag, desc string
		typ        plugin.FlagType
	}{
		{"identity-file", "Select a file from which to read the identity", plugin.FlagType_String},
		{"app-private-key", "GitHub App private key file path", plugin.FlagType_String},
		{"certificate-path", "Path (in PKCS #12/PFX or PEM format) to the certificate", plugin.FlagType_String},
		{"credentials-path", "The path to the service account credentials", plugin.FlagType_String},
		{"federated-token-file", "Path to a file containing an OIDC token", plugin.FlagType_String},
		{"auth-method", "Sign-in methods to use", plugin.FlagType_String},
		{"application-credential-name", "Application credential name", plugin.FlagType_String},
		{"tls-ca", "Path to the trusted CA certificate", plugin.FlagType_String},
		{"sslrootcert", "Path to the trusted CA certificate for TLS", plugin.FlagType_String},
		{"ask-pass", "Prompt for connection password", plugin.FlagType_Bool},
		{"ask-api-key", "Prompt for the API key", plugin.FlagType_Bool},
		{"ask-secret", "Prompt for the secret", plugin.FlagType_Bool},
	}
	for _, s := range safe {
		if tuiform.IsSecretFlag(plugin.Flag{Long: s.flag, Desc: s.desc, Type: s.typ}) {
			t.Errorf("--%s should not be classified as a secret value", s.flag)
		}
	}
}

// A sweep over everything installed, so a provider added later that names a
// flag in the usual way is covered without anyone editing a list.
func TestEverySecretShapedFlagIsClassified(t *testing.T) {
	if _, err := providers.ListActive(); err != nil {
		t.Skip("cannot read the installed provider set")
	}
	misses := 0
	for _, c := range BuildCatalog() {
		for _, fl := range c.Flags {
			if fl.Option&plugin.FlagOption_Hidden != 0 || fl.Type != plugin.FlagType_String {
				continue
			}
			name := strings.ToLower(fl.Long)
			looksSecret := strings.HasSuffix(name, "password") ||
				strings.HasSuffix(name, "token") ||
				strings.HasSuffix(name, "secret") ||
				strings.HasSuffix(name, "api-key")
			if looksSecret && !tuiform.IsSecretFlag(fl) {
				t.Errorf("%s --%s looks like a secret but is not classified", c.Name, fl.Long)
				misses++
			}
		}
	}
	if misses == 0 {
		t.Log("every secret-shaped flag in the installed set is classified")
	}
}

// launcherOwnedFields are the `special` markers a form is allowed to carry.
//
// `special` means "a field the launcher owns rather than one the provider
// declared", and there are legitimate ones -- the ambient-credential readout
// and its paste box are exactly that. What must never come back is the keychain
// toggle: it asked the user to choose between the operating system's protection
// and a temporary file, and it appeared even for connectors whose credential
// the launcher cannot carry at all, which made it doubly misleading.
//
// So this is an allowlist rather than a ban. A new launcher-owned field is
// added here deliberately, by someone who has read the paragraph above.
var launcherOwnedFields = map[string]bool{
	// The ambient-credential readout, and digitalocean's second one for its
	// env-only Spaces keys. Both hold the *name* of the variable a credential
	// came from and never the credential; see source_ambient.go.
	"credential-state":        true,
	"credential-state.spaces": true,

	// The two targets that have no flag to travel in, so they reach the child
	// through its environment instead. Neither holds a credential: a profile
	// name and a docker context name are both public, and what they select is
	// which credentials the child finds for itself. See PositionalSpec.Special.
	specialAlicloudProfile: true,
	specialDockerContext:   true,
}

func TestNoKeychainToggleIsOffered(t *testing.T) {
	for _, c := range BuildCatalog() {
		if !c.HasFormData() {
			continue
		}
		for _, fd := range newForm(c).Fields() {
			if strings.Contains(strings.ToLower(fd.Label), "keychain") {
				t.Fatalf("%s still asks about the keychain: %q", c.Name, fd.Label)
			}
			if fd.Special != "" && !launcherOwnedFields[fd.Special] {
				t.Errorf("%s has an undeclared launcher-owned field %q; "+
					"add it to launcherOwnedFields if it is meant to be there", c.Name, fd.Special)
			}
		}
	}
}

// The keychain being unavailable must not lose the credential or put it on the
// command line; it falls back to the inventory, which is still off argv.
func TestKeychainFailureFallsBackToTheInventory(t *testing.T) {
	c := sshConnector()
	f := newForm(c)
	fieldByLabel(t, f, "user@host").SetValue("chris@10.0.0.4")
	fieldByLabel(t, f, "password").SetValue(sentinel)
	withParser(t, &fakeParser{secretFlag: "password"})

	r := launchRequest{form: f}
	plan, err := r.plan(c, scanAction())
	args, env, cleanup := plan.args, plan.env, plan.cleanup
	if cleanup != nil {
		defer cleanup()
	}
	if err != nil {
		t.Fatal(err)
	}
	// Whichever route was taken, the secret is not on the command line.
	if strings.Contains(strings.Join(args, " "), sentinel) {
		t.Fatalf("secret reached argv: %v", args)
	}
	_ = env
}

// When the keychain cannot be used the scan still runs, the secret still stays
// off the command line, and the user is told the guarantee is weaker.
func TestKeychainFailureWarnsAndContinues(t *testing.T) {
	orig := storeCredentialFn
	storeCredentialFn = func(id string, cred *vault.Credential) error {
		return errors.New("keyring unavailable")
	}
	defer func() { storeCredentialFn = orig }()

	c := sshConnector()
	f := newForm(c)
	fieldByLabel(t, f, "user@host").SetValue("chris@10.0.0.4")
	fieldByLabel(t, f, "password").SetValue(sentinel)
	withParser(t, &fakeParser{secretFlag: "password"})

	r := launchRequest{form: f}
	plan, err := r.plan(c, scanAction())
	if plan.cleanup != nil {
		defer plan.cleanup()
	}
	if err != nil {
		t.Fatalf("a keychain failure must not stop the scan: %v", err)
	}
	if plan.warn == "" {
		t.Fatal("falling back to a file must warn")
	}
	for _, want := range []string{"keychain", "0600", "removed after"} {
		if !strings.Contains(plan.warn, want) {
			t.Errorf("the warning should mention %q: %q", want, plan.warn)
		}
	}
	// The point of the fallback: still not on the command line.
	if strings.Contains(strings.Join(plan.args, " "), sentinel) {
		t.Fatalf("secret reached argv: %v", plan.args)
	}
	if !containsString(plan.args, "--inventory-file") {
		t.Fatalf("expected the inventory route, got %v", plan.args)
	}
}

// The warning reaches the screen, not just the struct.
func TestKeychainWarningIsShown(t *testing.T) {
	m := sized(newTestModel(), 140, 30)
	m.lastWarn = "could not use the OS keychain (test) — written to a 0600 file"
	if out := ansi.Strip(m.View()); !strings.Contains(out, "could not use the OS keychain") {
		t.Errorf("the warning should be on screen:\n%s", out)
	}
}
