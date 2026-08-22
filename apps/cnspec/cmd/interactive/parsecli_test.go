// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	deliverypkg "go.mondoo.com/cnspec/cli/launcher/delivery"
	tuiform "go.mondoo.com/cnspec/cli/tui/form"
	"go.mondoo.com/mql/llx"
	"go.mondoo.com/mql/providers"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/providers-sdk/v1/vault"
)

// Two kinds of test live here and they are answering different questions.
//
// The per-connector tests all over this package ask what the *launcher* does
// with a provider's answer, and they use the fake below. Spawning a plugin
// subprocess per case would make the suite a provider integration test that
// happens to be shaped like a unit test, and it would skip entirely wherever
// the provider is not installed -- which on CI is everywhere, since
// PROVIDERS_PATH points at an empty directory.
//
// TestEveryCredentialFieldRoundTrips asks the other question: whether the real
// providers actually answer the way this whole approach assumes. It runs
// against whatever is installed and reports what it could not check.

// fakeParser stands in for a provider. It records what it was asked and answers
// the way the well-behaved majority do: flags become connection options and the
// one marked secret becomes a typed credential.
type fakeParser struct {
	provider  string
	connector string
	args      []string
	flags     map[string]*llx.Primitive
	// secretFlag is the flag the fake turns into a credential rather than an
	// option. Empty means every flag becomes an option, which is how the four
	// providers that never read conn.Credentials behave.
	secretFlag string
	// classify decides which flags become credentials when secretFlag names
	// nothing in particular. See defaultTestParser.
	classify func(flag string) bool
	// asset overrides the answer entirely, for a test about one provider's
	// particular shape.
	asset *inventory.Asset
	err   error
}

// answer is the stand-in's ParseCLI.
func (p *fakeParser) answer(provider, connector string, args []string, flags map[string]*llx.Primitive) (*inventory.Asset, error) {
	p.provider, p.connector, p.args, p.flags = provider, connector, args, flags
	if p.err != nil {
		return nil, p.err
	}
	if p.asset != nil {
		return p.asset, nil
	}
	conn := &inventory.Config{Type: connector, Options: map[string]string{}}
	for name, prim := range flags {
		value := string(prim.Value)
		credential := name == p.secretFlag
		if p.secretFlag == "" && p.classify != nil {
			credential = p.classify(name)
		}
		if credential {
			conn.Credentials = append(conn.Credentials,
				&vault.Credential{Type: vault.CredentialType_password, Password: value})
			continue
		}
		conn.Options[name] = value
	}
	return &inventory.Asset{Connections: []*inventory.Config{conn}}, nil
}

// withParser points the launcher at a stand-in provider for the duration of one
// test.
func withParser(t *testing.T, p *fakeParser) *fakeParser {
	t.Helper()
	prev := parseCLI
	t.Cleanup(func() { parseCLI = prev })
	parseCLI = p.answer
	return p
}

// defaultTestParser is what the package runs against unless a test says
// otherwise: a stand-in that turns every credential-shaped flag into a
// credential, which is what the well-behaved majority of providers do.
//
// It is installed from TestMain rather than left to each test to remember,
// because forgetting means starting a plugin subprocess -- which is slow, and
// which answers differently on a machine with no providers installed.
func defaultTestParser() {
	p := &fakeParser{classify: func(flag string) bool {
		// The form layer's own classifier, so the stand-in agrees with the
		// screen about what a secret is.
		return tuiform.IsSecretFlag(plugin.Flag{Long: flag, Type: plugin.FlagType_String})
	}}
	parseCLI = p.answer
}

// sentValue is what the launcher handed the provider for one flag.
func (p *fakeParser) sentValue(flag string) (string, bool) {
	prim, ok := p.flags[flag]
	if !ok {
		return "", false
	}
	return string(prim.Value), true
}

// assertCredentialReachesTheProvider is the claim every curated credential
// makes, and the only one it makes.
//
// It used to be a different claim per connector -- "this token travels in
// GITHUB_TOKEN", "that one travels in DD_API_KEY" -- because the launcher had
// to name a variable and the name was a fact about the provider that a person
// had checked. There is no name to get wrong now. What is left is the invariant
// the whole package exists for: the value reaches the provider, and it does not
// reach a command line that `ps auxww` publishes.
func assertCredentialReachesTheProvider(t *testing.T, c Connector, f form, flag, value string) *fakeParser {
	t.Helper()
	p := withParser(t, &fakeParser{secretFlag: flag})

	plan, err := (launchRequest{form: f}).plan(c, scanAction())
	if plan.cleanup != nil {
		t.Cleanup(plan.cleanup)
	}
	if err != nil {
		t.Fatalf("a typed credential must have somewhere to go: %v", err)
	}

	if p.connector != c.Name || p.provider != c.Provider {
		t.Errorf("asked %s/%s, want %s/%s", p.provider, p.connector, c.Provider, c.Name)
	}
	got, ok := p.sentValue(flag)
	if !ok {
		t.Fatalf("--%s never reached the provider; it was sent %v", flag, sortedKeys(p.flags))
	}
	if got != value {
		t.Errorf("--%s reached the provider as %q, want %q", flag, got, value)
	}
	if line := strings.Join(plan.args, " "); strings.Contains(line, value) {
		t.Fatalf("the credential reached the command line: %v", plan.args)
	}
	if joined := strings.Join(plan.env, " "); strings.Contains(joined, value) {
		t.Fatalf("the credential reached the child's environment: %v", plan.env)
	}
	if len(plan.args) < 3 || plan.args[len(plan.args)-2] != "--inventory-file" {
		t.Fatalf("want a scan driven by an inventory file, got %v", plan.args)
	}
	return p
}

func sortedKeys(m map[string]*llx.Primitive) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// A launch is refused when the provider did not keep the credential.
//
// This is the check no table could make, and databricks is why it is a refusal
// rather than a warning: its credential switch has no default arm, so an
// untagged credential leaves the token empty and the Databricks SDK then
// resolves DATABRICKS_TOKEN out of the ambient environment on its own. That
// does not error. It scans whatever account that variable names.
func TestALaunchIsRefusedWhenTheProviderDropsTheCredential(t *testing.T) {
	c := Connector{Provider: "databricks", Name: "databricks"}
	f := tuiform.New("databricks", []tuiform.Field{secretField("token", "must-never-reach-argv")})

	withParser(t, &fakeParser{asset: &inventory.Asset{
		Connections: []*inventory.Config{{
			Type: "databricks", Options: map[string]string{"account-id": "123"},
		}},
	}})

	_, err := (launchRequest{form: f}).plan(c, scanAction())
	if err == nil {
		t.Fatal("the launch went ahead with a credential the provider had dropped")
	}
	if !strings.Contains(err.Error(), "--token") {
		t.Errorf("the refusal does not name the credential that was lost: %v", err)
	}
}

// A credential the provider reads as a plain connection option cannot go to the
// keychain, and the user is told so by name rather than left to assume it did.
//
// openai is one of eleven connectors this happens on -- see
// deliverypkg.PlacedOption for the measured set, which is larger than the four
// AI connectors it was first noticed on. Refusing them would take working
// connectors away for an exposure no worse than the environment route they had
// before --
// a 0600 file in a 0700 directory, removed after the run, versus
// /proc/<pid>/environ for the length of it. So it warns, and the warning names
// the flag.
func TestAPlaintextOnlyCredentialWarnsAndNamesTheFlag(t *testing.T) {
	c := Connector{Provider: "openai", Name: "openai"}
	f := tuiform.New("openai", []tuiform.Field{secretField("token", "must-never-reach-argv")})

	// secretFlag empty: every flag becomes an option, which is what these four
	// providers do.
	withParser(t, &fakeParser{})

	plan, err := (launchRequest{form: f}).plan(c, scanAction())
	if plan.cleanup != nil {
		defer plan.cleanup()
	}
	if err != nil {
		t.Fatalf("a plaintext-only credential is a warning, not a refusal: %v", err)
	}
	if !strings.Contains(plan.warn, "--token") {
		t.Errorf("the warning does not name the flag: %q", plan.warn)
	}
	if !strings.Contains(plan.warn, "plain text") {
		t.Errorf("the warning does not say what the exposure is: %q", plan.warn)
	}
	if strings.Contains(strings.Join(plan.args, " "), "must-never-reach-argv") {
		t.Fatalf("the key reached argv even so: %v", plan.args)
	}
}

// The keychain is not touched for a credential that could not be referenced
// from the file anyway.
//
// Saving one would leave a secret in the OS store that nothing reads, and on
// macOS it would raise an authentication dialog behind the launcher to put it
// there.
func TestNoKeychainWriteForACredentialTheFileCannotReference(t *testing.T) {
	c := Connector{Provider: "openai", Name: "openai"}
	f := tuiform.New("openai", []tuiform.Field{secretField("token", "must-never-reach-argv")})
	withParser(t, &fakeParser{})

	writes := 0
	prev := storeCredentialFn
	storeCredentialFn = func(id string, cred *vault.Credential) error {
		writes++
		return nil
	}
	defer func() { storeCredentialFn = prev }()

	plan, err := (launchRequest{form: f}).plan(c, scanAction())
	if plan.cleanup != nil {
		defer plan.cleanup()
	}
	if err != nil {
		t.Fatal(err)
	}
	if writes != 0 {
		t.Errorf("the keychain was written %d times for a credential it cannot hold", writes)
	}
}

// The provider is asked before the keychain is touched, so a launch that gets
// refused leaves nothing behind in the OS store.
func TestARefusedLaunchWritesNoKeychainEntry(t *testing.T) {
	c := Connector{Provider: "databricks", Name: "databricks"}
	f := tuiform.New("databricks", []tuiform.Field{secretField("token", "must-never-reach-argv")})
	withParser(t, &fakeParser{asset: &inventory.Asset{
		Connections: []*inventory.Config{{Type: "databricks"}},
	}})

	writes := 0
	prev := storeCredentialFn
	storeCredentialFn = func(id string, cred *vault.Credential) error {
		writes++
		return nil
	}
	defer func() { storeCredentialFn = prev }()

	if _, err := (launchRequest{form: f}).plan(c, scanAction()); err == nil {
		t.Fatal("want a refusal")
	}
	if writes != 0 {
		t.Errorf("a refused launch wrote %d keychain entries", writes)
	}
}

// A provider that cannot be asked is a refusal, not a scan with no credential.
func TestAProviderThatCannotBeAskedRefusesTheLaunch(t *testing.T) {
	c := Connector{Provider: "ssh", Name: "ssh"}
	f := tuiform.New("ssh", []tuiform.Field{secretField("password", "must-never-reach-argv")})
	withParser(t, &fakeParser{err: errNoProvider})

	_, err := (launchRequest{form: f}).plan(c, scanAction())
	if err == nil {
		t.Fatal("the launch went ahead without asking the provider anything")
	}
	if !strings.Contains(err.Error(), "ssh") {
		t.Errorf("the refusal does not name the connector: %v", err)
	}
}

var errNoProvider = &parseError{"cannot start the ssh provider"}

type parseError struct{ msg string }

func (e *parseError) Error() string { return e.msg }

func secretField(flag, value string) tuiform.Field {
	fd := tuiform.NewField(tuiform.Decl{
		Label: flag, Flag: flag, Kind: tuiform.KindText,
		Secret: true, Section: tuiform.SectionCredential,
	})
	fd.SetValue(value)
	return fd
}

// Whether every connector actually round-trips through ParseCLI is the entire
// basis for this approach, so it is checked against the real providers rather
// than reasoned about.
//
// For each connector the catalog knows about, and for each credential field on
// its form, the form is filled the way a user configuring one target fills it
// -- the TARGET section plus whatever the connector marks required, plus that
// one credential -- and the connector's own ParseCLI is asked what it means.
// The value is then looked for in the asset that came back.
//
// Only one outcome is a failure: the provider keeping the value nowhere at all.
// A credential is the good case and an option is the warned case, and both are
// counted and logged so a change in the split is visible.
//
// The counts are logged rather than asserted. CI points PROVIDERS_PATH at an
// empty directory, so this checks nothing there and says so; a total asserted
// here would fail on one machine and pass vacuously on the other, and the
// vacuous pass is the dangerous half.
func TestEveryCredentialFieldRoundTrips(t *testing.T) {
	if testing.Short() {
		t.Skip("starts a provider plugin per connector")
	}

	// Each entry is a connector, a flag, and why that provider keeps the value
	// nowhere the launcher can find it. Every one was reproduced against the
	// installed provider, and every one is a launch the user is refused with a
	// message naming the flag -- which is the honest answer, because the scan
	// would otherwise run without the credential.
	knownLost := map[string]map[string]string{
		// Verified: alicloud's ParseCLI keeps access-key-id, region, regions,
		// role-arn and role-session-name, and drops --sts-token entirely.
		"alicloud": {"sts-token": "the provider's ParseCLI does not read it"},

		// A passphrase with no certificate, a client secret with no client id,
		// a key passphrase with no key: each provider correctly drops half a
		// credential. The other half is a field on the same form.
		"azure": {"certificate-secret": "needs --certificate-path, which this probe leaves empty"},
		"ms365": {"certificate-secret": "needs --certificate-path, which this probe leaves empty"},
		"jamf":  {"client-secret": "needs --client-id, which is not in TARGET"},
		"oci":   {"key-secret": "needs --key-path, which this probe leaves empty"},
	}

	checked, skipped, lost := 0, 0, 0
	placements := map[deliverypkg.Placement]int{}
	connectors := map[string]bool{}

	installed := installedProviders()
	for _, c := range BuildCatalog() {
		if !c.DeclaresMetadata() || !installed[c.Provider] {
			skipped++
			continue
		}
		base := newForm(c)
		for i := range base.Fields() {
			fd := base.Fields()[i]
			if !fd.Secret || fd.Reference || fd.Flag == "" || !base.Visible(i) {
				continue
			}

			f := fillOneCredential(c, fd.Identity())
			req := deliverypkg.RequestFor(f, c.Flags)
			asset, err := deliverypkg.Parser.ParseCLI(c.Provider, c.Name, req.Args, req.Flags)
			if err != nil {
				// The provider refused these settings, which is a fact about
				// the probe's filling rather than about the round trip. The
				// launcher surfaces the same message.
				t.Logf("%s --%s: not checkable here, the provider refused the probe's settings (%v)",
					c.Name, fd.Flag, err)
				continue
			}
			checked++
			connectors[c.Name] = true

			placed := deliverypkg.Locate(f, asset)
			if len(placed) != 1 {
				t.Errorf("%s --%s: %d located secrets, want exactly one", c.Name, fd.Flag, len(placed))
				continue
			}
			placements[placed[0].Placement]++

			why, expected := knownLost[c.Name][fd.Flag]
			if placed[0].Placement == deliverypkg.PlacedNowhere {
				lost++
				if !expected {
					t.Errorf("%s --%s: the provider kept the value nowhere, so a launch "+
						"is refused. Either the flag is not a credential, or the form is "+
						"missing the field it belongs with; add it to knownLost with the reason.",
						c.Name, fd.Flag)
				}
				continue
			}
			if expected {
				t.Errorf("%s --%s now lands as %v, but is recorded as lost because %s. "+
					"Drop it from knownLost.", c.Name, fd.Flag, placed[0].Placement, why)
			}
		}
	}

	t.Logf("checked %d credential fields over %d connectors: %d reached a credential the "+
		"keychain can hold, %d reached a plaintext option, %d were kept nowhere; "+
		"skipped %d catalog entries with no metadata installed here",
		checked, len(connectors),
		placements[deliverypkg.PlacedCredential], placements[deliverypkg.PlacedOption], lost,
		skipped)
	if checked == 0 {
		t.Skipf("no provider is installed here, so nothing was checked (%d catalog entries skipped)", skipped)
	}
}

// fillOneCredential fills the form the way a user configuring one target does:
// the TARGET section, anything the connector marks required, and exactly one
// credential.
//
// The rest of the CREDENTIAL section is deliberately left empty. Filling a
// second credential makes several providers choose a different authentication
// path and legitimately drop the one under test -- github prefers its app
// credentials over --token, junos refuses a password beside an identity file --
// and a probe that provoked that would be measuring its own filling.
func fillOneCredential(c Connector, want string) form {
	f := newForm(c)
	for i := range f.Fields() {
		if !f.Visible(i) {
			continue
		}
		fd := &f.Fields()[i]
		if fd.Special != "" {
			continue
		}
		if fd.Secret {
			if fd.Identity() == want {
				fd.SetValue(probeSecretFor(fd.Flag))
			}
			continue
		}
		if !fd.Required && fd.Section != sectionTarget {
			continue
		}
		// A path-shaped flag is left alone: the provider opens it, and a path
		// that does not exist is an error about the probe.
		if strings.Contains(fd.Flag, "-path") || strings.Contains(fd.Flag, "-file") {
			continue
		}
		switch fd.Kind {
		case fieldBool, fieldCredentialState:
		case fieldChoice, fieldMultiChoice:
			if len(fd.Options) == 0 {
				if fd.Kind == fieldChoice {
					fd.SetValue("probe")
				}
				continue
			}
			if fd.Kind == fieldMultiChoice {
				fd.TogglePick(fd.Options[0])
			} else {
				fd.SetValue(fd.Options[0])
			}
		default:
			switch {
			case fd.Flag == "":
				if c.Name == "ssh" || c.Name == "winrm" {
					fd.SetValue("probe@10.0.0.5")
				} else {
					fd.SetValue("probe")
				}
			case strings.Contains(fd.Flag, "port"):
				fd.SetValue("443")
			case strings.Contains(fd.Flag, "url"), strings.Contains(fd.Flag, "endpoint"):
				fd.SetValue("https://probe.example")
			case strings.Contains(fd.Flag, "host"):
				fd.SetValue("probe.example")
			default:
				fd.SetValue("probe")
			}
		}
	}
	return f
}

// probeSecretFor is a value distinctive enough that finding it in the asset
// means the provider kept it, rather than that something else happened to be
// spelled the same way.
func probeSecretFor(flag string) string {
	return "cnspec-ui-probe-" + strings.ReplaceAll(flag, "-", "_") +
		"-" + time.Now().Format("150405")
}

// installedProviders is what is on this machine, read once.
//
// The tests that ask a real provider check this first and skip what is missing.
// ProviderParser.ParseCLI would otherwise install it, which is right for a
// launch -- a user who opened a form expects it to work -- and wrong for a test
// suite: CI points PROVIDERS_PATH at an empty directory, so every one of these
// would download a provider over the network to answer a question about the
// launcher.
var installedProviders = sync.OnceValue(func() map[string]bool {
	out := map[string]bool{}
	list, err := providers.ListActive()
	if err != nil {
		return out
	}
	for _, p := range list {
		if p.Provider != nil {
			out[p.Name] = true
		}
	}
	return out
})
