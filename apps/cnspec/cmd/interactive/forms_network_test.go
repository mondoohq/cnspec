// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	"strings"
	"testing"
)

// The curated Network & Security Devices forms.
//
// Every test below drives the real form -- newForm over the connector rebuilt
// from internal/connectors/connectors.json -- rather than inspecting the
// FormSpec literal, because a spec that names a flag the connector does not
// declare is silently dropped by applySpec. Reading the spec back would agree
// with itself; building the form is what proves the flag survived.
//
// The two connector lists below are what used to be two files, split where one
// contributor's work ended and the next began. They stay two lists because each
// test names the list it iterates, and merging them would have been a rewrite
// of every assertion in both. Both are declared through filedHere, so the thing
// the letters used to hide -- a connector from another category quietly added
// to whichever list its author had open -- fails in
// TestEveryTestedConnectorIsFiledUnderItsCategory with the name of the file it
// belongs in.

// The nine appliance connectors that address a device as a positional or through a --hostname flag.
//
// Every test below drives the real form -- newForm over the connector rebuilt
// from internal/connectors/connectors.json -- rather than inspecting the FormSpec literal,
// because a spec that names a flag the connector does not declare is silently
// dropped by applySpec. Reading the spec back would agree with itself; building
// the form is what proves the flag survived.
var networkAConnectors = filedHere(
	"arista", "bigip", "checkpoint", "ciscocatalyst", "fortios",
	"host", "ipmi", "junos", "mikrotik",
)

// All nine are registered, and each one registered here and nowhere else. Nine
// agents were curating disjoint slices of the catalog at the same time, so a
// connector this file believes it owns being absent means the registration was
// lost, and registerSpec keeping someone else's is a collision this catches
// before it reaches a user.
func TestNetworkAConnectorsAreRegistered(t *testing.T) {
	for _, name := range networkAConnectors {
		if _, ok := formSpecs[name]; !ok {
			t.Errorf("%s has no registered spec", name)
		}
	}
}

// A connector that requires an argument must ask for it somewhere the user can
// actually get to. A required field that lands in OPTIONS sits behind the fold,
// where focusFirstMissing parks a cursor the user cannot move to -- so the
// launcher would refuse to scan while showing nothing that needed filling in.
//
// This is the assertion for "the required field is present and reachable": it
// exists, it is required, it is in TARGET, and the form considers it visible.
func TestNetworkARequiredFieldsAreReachable(t *testing.T) {
	// The five connectors whose target is an argument, and the label each one
	// gives it. The labels are asserted rather than derived because the
	// snapshot cannot carry a usage string -- connectorSnapshot.connector()
	// sets Use to the connector name -- so without an explicit Positional in
	// the spec every one of these would render as "argument 1".
	want := map[string]string{
		"arista":        "user@host",
		"ciscocatalyst": "hostname",
		"host":          "host",
		"ipmi":          "user@host",
		"mikrotik":      "user@host",
	}

	for _, name := range networkAConnectors {
		c, f := formFor(t, name)
		label, needsArg := want[name]

		if !needsArg {
			if c.MinArgs > 0 {
				t.Errorf("%s: expected no required argument, but the connector declares MinArgs=%d",
					name, c.MinArgs)
			}
			continue
		}
		if c.MinArgs == 0 {
			t.Errorf("%s: expected a required argument, but the connector declares MinArgs=0", name)
			continue
		}

		fd := fieldByLabel(t, f, label)
		if !fd.Required {
			t.Errorf("%s: %q is the connector's argument but is not marked required", name, label)
		}
		if fd.Section != sectionTarget {
			t.Errorf("%s: %q is required but sits in %s, where the cursor cannot reach it",
				name, label, fd.Section)
		}
		if fd.Flag != "" {
			t.Errorf("%s: %q should be the positional argument, not --%s", name, label, fd.Flag)
		}

		// Reachable means the form is willing to show it right now, with
		// nothing else filled in.
		visible := false
		for _, i := range f.VisibleIndices() {
			if f.Fields()[i].Label == label {
				visible = true
			}
		}
		if !visible {
			t.Errorf("%s: %q is required but hidden", name, label)
		}
	}
}

// A typed password reaches the provider, and the connector's own --ask-pass is
// still on the form for a user who would rather be prompted.
//
// These seven all declare --ask-pass, and that used to mean a typed password
// was thrown away: the launcher took the prompt route, put --ask-pass on the
// command line in place of the value, and the child asked for the password the
// user had already given it. Verified before the change --
// `cnspec scan ssh chris@10.0.0.5 --ask-pass` was the entire command for a form
// with a password in it. Prompting is still the strongest route there is and it
// is still one keystroke away; it is no longer substituted for what was typed.
func TestNetworkAPasswordReachesTheProvider(t *testing.T) {
	// connector -> the positional to fill, if it has one.
	arg := map[string]string{
		"arista":        "user@host",
		"ciscocatalyst": "hostname",
		"ipmi":          "user@host",
		"mikrotik":      "user@host",
	}
	// bigip, checkpoint and junos take the device as --hostname instead.
	hostFlag := map[string]bool{"bigip": true, "checkpoint": true, "junos": true}

	for _, name := range []string{
		"arista", "bigip", "checkpoint", "ciscocatalyst", "ipmi", "junos", "mikrotik",
	} {
		c, f := formFor(t, name)

		if label, ok := arg[name]; ok {
			fieldByLabel(t, f, label).SetValue("admin@10.0.0.4")
		}
		if hostFlag[name] {
			fieldByFlag(t, f, "hostname").SetValue("10.0.0.4")
		}
		pw := fieldByFlag(t, f, "password")
		if !pw.Secret {
			t.Errorf("%s: --password is not classified as a secret", name)
		}
		if pw.Section != sectionCredential {
			t.Errorf("%s: --password is in %s, not CREDENTIAL", name, pw.Section)
		}
		pw.SetValue(sentinel)

		if got := deliveryFor(f); got != deliverInventory {
			t.Errorf("%s: a typed password routes as %v, want the inventory", name, got)
			continue
		}
		assertCredentialReachesTheProvider(t, c, f, "password", sentinel)

		// The toggle is still there, and ticking it with nothing typed is still
		// an ordinary flag on an ordinary command line.
		ask := f.IndexOfFlag("ask-pass")
		if ask < 0 {
			t.Errorf("%s: lost its --ask-pass toggle", name)
			continue
		}
		prompt := newForm(c)
		prompt.Fields()[prompt.IndexOfFlag("ask-pass")].SetOn(true)
		if got := deliveryFor(prompt); got != deliverPlain {
			t.Errorf("%s: --ask-pass on its own routes as %v, want the command line", name, got)
		}
		if !containsString(prompt.Args(), "--ask-pass") {
			t.Errorf("%s: the child would never be asked to prompt: %v", name, prompt.Args())
		}
	}
}

// host declares no credential flag at all, so it must stay on the plain command
// line. Routing a scan with no secret through an inventory file would be a
// pointless regression, and it is also the check that this connector was
// curated as the scanning target it is rather than as a login.
func TestHostIsAScanTargetNotALogin(t *testing.T) {
	c, f := formFor(t, "host")

	for _, fd := range f.Fields() {
		if fd.Secret {
			t.Errorf("host: --%s is classified as a secret; this connector takes no credential", fd.Flag)
		}
		if fd.Section == sectionCredential {
			t.Errorf("host: %q is in CREDENTIAL; this connector takes no credential", fd.Label)
		}
	}

	fieldByLabel(t, f, "host").SetValue("https://mondoo.com")

	if got := deliveryFor(f); got != deliverPlain {
		t.Errorf("host: delivery = %v, want deliverPlain", got)
	}

	r := launchRequest{form: f}
	plan, err := r.plan(c, scanAction())
	if plan.cleanup != nil {
		defer plan.cleanup()
	}
	if err != nil {
		t.Fatal(err)
	}
	if containsString(plan.args, "--inventory-file") {
		t.Errorf("host: expected a plain command line, got %v", plan.args)
	}
	if !containsString(plan.args, "https://mondoo.com") {
		t.Errorf("host: the target did not reach the command line: %v", plan.args)
	}
}

// fortios and checkpoint were the two connectors this file recorded as
// unroutable, and neither is any more.
//
// fortios --token was refused because the variable derived from its name,
// MONDOO_TOKEN, is the platform service account's own -- handing it to the child
// would have replaced the credential the scan authenticates and uploads with.
// checkpoint --api-key was allowed only because MONDOO_API_KEY happened not to
// be taken. Both facts were about a name the launcher invented; neither was
// about fortios or checkpoint. There is no derived name now, so there is nothing
// to collide.
func TestNetworkACredentialsReachTheProvider(t *testing.T) {
	for _, tc := range []struct{ connector, flag string }{
		{connector: "fortios", flag: "token"},
		{connector: "checkpoint", flag: "api-key"},
	} {
		c, f := formFor(t, tc.connector)
		fieldByFlag(t, f, "hostname").SetValue("10.0.0.4")

		fd := fieldByFlag(t, f, tc.flag)
		if !fd.Secret {
			t.Errorf("%s: --%s is not classified as a secret", tc.connector, tc.flag)
			continue
		}
		// A credential belongs in CREDENTIAL: the classifier puts it there,
		// which is why leaving it out of the spec's Credential list hides
		// nothing.
		if fd.Section != sectionCredential {
			t.Errorf("%s: --%s is in %s, not CREDENTIAL", tc.connector, tc.flag, fd.Section)
		}
		fd.SetValue(sentinel)

		assertCredentialReachesTheProvider(t, c, f, tc.flag, sentinel)
	}
}

// checkpoint takes a password or an API key, and handling one must not cost the
// other. The target has to survive both.
func TestCheckpointPasswordCarriesItsTargetToo(t *testing.T) {
	c, f := formFor(t, "checkpoint")
	fieldByFlag(t, f, "hostname").SetValue("mgmt.example.com")
	fieldByFlag(t, f, "username").SetValue("admin")
	fieldByFlag(t, f, "password").SetValue(sentinel)

	p := withParser(t, &fakeParser{secretFlag: "password"})
	plan, err := (launchRequest{form: f}).plan(c, scanAction())
	if plan.cleanup != nil {
		defer plan.cleanup()
	}
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.Join(plan.args, " "), sentinel) {
		t.Fatalf("secret reached the command line: %v", plan.args)
	}
	// The target now travels in the inventory rather than on argv, so this is
	// a question about what the provider was told rather than about the words
	// the child was given.
	if got, _ := p.sentValue("hostname"); got != "mgmt.example.com" {
		t.Fatalf("the target did not survive the credential handling: hostname = %q", got)
	}
}

// junos --identity-file names a key rather than holding one, so it has to stay
// on the command line -- that is how the provider expects to receive it -- while
// still being presented as part of the credential. ssh gets this right and
// junos is the same shape; this pins it.
func TestJunosIdentityFileIsAPathNotASecret(t *testing.T) {
	_, f := formFor(t, "junos")
	fd := fieldByFlag(t, f, "identity-file")

	if fd.Secret {
		t.Error("junos: --identity-file is a path and must not be masked or diverted off argv")
	}
	if !fd.Reference {
		t.Error("junos: --identity-file should be classified as a reference to a credential")
	}
	if fd.Section != sectionCredential {
		t.Errorf("junos: --identity-file should be shown with the credential, got %s", fd.Section)
	}
}

// The target of each flag-addressed connector is promoted to TARGET, so the
// first thing on the pane is the device rather than a TLS toggle.
func TestNetworkAFlagAddressedTargetsArePromoted(t *testing.T) {
	for _, name := range []string{"bigip", "checkpoint", "fortios", "junos"} {
		_, f := formFor(t, name)
		fd := fieldByFlag(t, f, "hostname")
		if fd.Section != sectionTarget {
			t.Errorf("%s: --hostname is in %s, want TARGET", name, fd.Section)
		}
		if f.Fields()[0].Flag != "hostname" {
			t.Errorf("%s: the first field is --%s, want --hostname", name, f.Fields()[0].Flag)
		}
	}
}

// A picker is a promise that its values work. srcSSHHost lists ~/.ssh/config
// aliases, which only mean anything to a client that reads that file -- and
// none of these nine was shown to. Six of them are not SSH clients at all
// (arista is goeapi over HTTPS, ipmi is RMCP to a BMC, mikrotik is the RouterOS
// API, bigip/checkpoint/fortios are REST APIs), host is not a login, and
// ciscocatalyst addresses a Catalyst Center appliance through its API.
//
// This is a test about honesty rather than mechanics, so it is written to fail
// loudly if someone attaches the picker without revisiting the reasoning
// recorded in the header of cli/launcher/forms/forms_network.go.
func TestNetworkAOffersNoSSHConfigHosts(t *testing.T) {
	for _, name := range networkAConnectors {
		_, f := formFor(t, name)
		for _, fd := range f.Fields() {
			if fd.Source() == srcSSHHost || fd.LiveSource == srcSSHHost {
				t.Errorf("%s: %q offers ~/.ssh/config hosts, but this connector does not "+
					"resolve ssh_config aliases; see the reasoning in cli/launcher/forms/forms_network.go",
					name, fd.Label)
			}
		}
	}
}

// fortios --enable-forti-sdk-logs is an SDK debugging switch. Hiding it is the
// only thing the spec does to that connector's options, so it is worth pinning:
// a Hide that stops matching is silent.
func TestFortiosHidesTheSDKLogSwitch(t *testing.T) {
	_, f := formFor(t, "fortios")
	if fd := findFieldByFlag(f, "enable-forti-sdk-logs"); fd != nil {
		t.Errorf("fortios: --enable-forti-sdk-logs is still shown, in %s", fd.Section)
	}
	// The rest of the connector survived the hiding.
	for _, flag := range []string{"hostname", "token", "insecure"} {
		if findFieldByFlag(f, flag) == nil {
			t.Errorf("fortios: --%s disappeared", flag)
		}
	}
}

// mikrotik's two ports are suggestions, not a validation list: RouterOS listens
// on 8728 plain and 8729 with --tls, and a device on a non-default port must
// still be reachable by typing it. Choices on a flag leaves the field free
// text, and this is what says so.
func TestMikrotikPortsAreSuggestionsNotAWhitelist(t *testing.T) {
	_, f := formFor(t, "mikrotik")
	fd := fieldByFlag(t, f, "port")

	if fd.Kind != fieldChoice {
		t.Errorf("mikrotik: --port kind = %v, want a picker", fd.Kind)
	}
	if fd.Strict {
		t.Error("mikrotik: --port is strict, so a device on a non-default port could not be reached")
	}
	if len(fd.Options) != 2 || fd.Options[0] != "8728" || fd.Options[1] != "8729" {
		t.Errorf("mikrotik: --port options = %v, want the API and API-SSL ports", fd.Options)
	}
}

// networkBConnectors are the connectors this file curates. ipinfo is on the
// list this file was written from and is deliberately not here; see
// TestIPInfoHasNothingToCurate.
var networkBConnectors = filedHere(
	"nd-ssh", "networkdiscovery", "nmap", "opcua",
	"panos", "redfish", "shodan", "unifi",
)

func TestNetworkBEveryConnectorHasASpec(t *testing.T) {
	for _, name := range networkBConnectors {
		if _, ok := formSpecs[name]; !ok {
			t.Errorf("%s has no registered spec", name)
		}
	}
}

// The assertion that catches a broken credential section at authoring time
// rather than in a demo: a credential the user can fill in reaches the
// provider, and reaches nothing that publishes it.
//
// Each case used to name the route as well -- the prompt for four of them, an
// environment variable for the other two -- and every one of those was a claim
// about the connector that a person had checked. There is one route now, and
// what it carries is the provider's business.
func TestNetworkBCredentialsCanLaunch(t *testing.T) {
	cases := []struct {
		connector string
		// selector is the value of the leading positional, for the forms whose
		// credential is behind one.
		selectorLabel string
		selector      string
		flag          string
	}{
		{connector: "nd-ssh", flag: "password"},
		{connector: "panos", flag: "password"},
		{connector: "redfish", flag: "password"},
		{connector: "shodan", flag: "token"},
		{
			connector: "unifi", selectorLabel: "sign in with", selector: "username and password",
			flag: "password",
		},
		{
			connector: "unifi", selectorLabel: "sign in with", selector: "API key",
			flag: "api-key",
		},
	}

	const secret = "<PLACEHOLDER-not-a-real-secret>"
	for _, tc := range cases {
		c, f := formFor(t, tc.connector)
		if tc.selectorLabel != "" {
			setLabelled(t, &f, tc.selectorLabel, tc.selector)
		}
		i := f.IndexOfFlag(tc.flag)
		if i < 0 {
			t.Errorf("%s: no --%s field to fill in", tc.connector, tc.flag)
			continue
		}
		if !f.Fields()[i].Secret {
			t.Errorf("%s: --%s is not classified as a secret, so it would reach argv",
				tc.connector, tc.flag)
			continue
		}
		f.Fields()[i].SetValue(secret)

		t.Run(tc.connector+"/"+tc.flag, func(t *testing.T) {
			assertCredentialReachesTheProvider(t, c, f, tc.flag, secret)
		})
	}
}

// nd-ssh declares three secret-carrying flags, and the spec leaves exactly one
// of them fillable and promotes the two prompts that do the same job more
// safely. The prompts are what make that a simplification rather than a loss:
// the child asks and sets the flag itself, so the enable password never exists
// in the launcher at all.
//
// The route no longer forces this -- one ParseCLI call carries as many secrets
// as the form holds -- but the screen is still the better one, because a
// prompted secret is stronger than a stored one and a Cisco login password and
// enable password are two boxes nobody wants to fill twice.
func TestNdSSHPromptsForWhatItCannotCarry(t *testing.T) {
	c, f := formFor(t, "nd-ssh")

	for _, flag := range []string{"enable-password", "private-key-passphrase"} {
		if f.IndexOfFlag(flag) >= 0 {
			t.Errorf("--%s is still fillable; a second secret has nowhere to travel", flag)
		}
	}
	for _, flag := range []string{"ask-pass", "ask-enable-password"} {
		if f.IndexOfFlag(flag) < 0 {
			t.Errorf("--%s is not offered, so its credential cannot be collected at all", flag)
		}
	}

	// Only one secret field survives.
	var secretFlags []string
	for _, fd := range f.Fields() {
		if fd.Secret {
			secretFlags = append(secretFlags, fd.Flag)
		}
	}
	if len(secretFlags) != 1 || secretFlags[0] != "password" {
		t.Fatalf("nd-ssh offers %v as secrets, want only --password", secretFlags)
	}

	// The realistic Cisco case: a login password and an enable password. The
	// first travels as a prompt, the second as its own prompt flag.
	setLabelled(t, &f, "user@host", "admin@switch1.example.com")
	f.Fields()[f.IndexOfFlag("password")].SetValue("<PLACEHOLDER-not-a-real-secret>")
	f.Fields()[f.IndexOfFlag("ask-enable-password")].SetOn(true)

	p := assertCredentialReachesTheProvider(t, c, f, "password", "<PLACEHOLDER-not-a-real-secret>")

	// The target and the second prompt both reach the provider. They used to be
	// checked on argv, which is where they travelled while the credential went
	// by environment variable; the inventory carries the whole invocation now,
	// so the question is what the provider was told.
	if len(p.args) != 1 || p.args[0] != "admin@switch1.example.com" {
		t.Errorf("the target did not reach the provider: %v", p.args)
	}
	if _, ok := p.sentValue("ask-enable-password"); !ok {
		t.Errorf("the enable-password prompt was not requested: %v", sortedKeys(p.flags))
	}
	if _, ok := p.sentValue("store-commands"); ok {
		t.Errorf("the debugging flag was handed to the provider: %v", sortedKeys(p.flags))
	}
}

// nmap and shodan reach their sub-commands through a selector, and the words
// it emits are the provider's own: anything else is rejected by ParseCLI with
// "invalid sub-command". A range emits no word at all, because it travels as
// --networks.
func TestNetworkBSelectorsEmitTheProviderSubcommands(t *testing.T) {
	cases := []struct {
		connector string
		label     string
		choice    string
		target    string
		wantArgs  []string
	}{
		{"nmap", "what to scan", "host", "192.168.1.1", []string{"host", "192.168.1.1"}},
		{"nmap", "what to scan", "domain", "example.com", []string{"domain", "example.com"}},
		{"nmap", "what to scan", "network range", "", nil},
		{"shodan", "what to query", "host", "192.168.1.1", []string{"host", "192.168.1.1"}},
		{"shodan", "what to query", "domain", "example.com", []string{"domain", "example.com"}},
		{"shodan", "what to query", "account", "", nil},
		{"shodan", "what to query", "network range", "", nil},
	}

	for _, tc := range cases {
		_, f := formFor(t, tc.connector)
		setLabelled(t, &f, tc.label, tc.choice)
		if tc.target != "" {
			setLabelled(t, &f, "target", tc.target)
		}
		got := f.Args()
		if strings.Join(got, " ") != strings.Join(tc.wantArgs, " ") {
			t.Errorf("%s %q emitted %v, want %v", tc.connector, tc.choice, got, tc.wantArgs)
		}
		if err := f.Validate(); err != nil {
			t.Errorf("%s %q does not validate: %v", tc.connector, tc.choice, err)
		}
	}
}

// --networks is what a scan is pointed at, not a place to put something
// discovery found. A picker on it would offer the answer as the question, and
// it is shown only for the shape that uses it.
func TestNetworkRangesAreInputNotDiscovered(t *testing.T) {
	for _, name := range []string{"nmap", "shodan"} {
		_, f := formFor(t, name)
		i := f.IndexOfFlag("networks")
		if i < 0 {
			t.Errorf("%s: no --networks field", name)
			continue
		}
		if f.Fields()[i].Source() != "" || f.Fields()[i].LiveSource != "" {
			t.Errorf("%s: --networks has a picker attached (%q/%q)",
				name, f.Fields()[i].Source(), f.Fields()[i].LiveSource)
		}
		if len(f.Fields()[i].ShowIf) == 0 {
			t.Errorf("%s: --networks is shown for every shape, including the ones that ignore it", name)
		}
	}

	// And the single-target shapes do not carry it.
	_, f := formFor(t, "nmap")
	setLabelled(t, &f, "what to scan", "host")
	setLabelled(t, &f, "target", "192.168.1.1")
	f.Fields()[f.IndexOfFlag("networks")].SetPicks(map[string]bool{"10.0.0.0/8": true})
	if strings.Contains(strings.Join(f.Args(), " "), "10.0.0.0/8") {
		t.Errorf("a leftover range travelled with a single-host scan: %v", f.Args())
	}
}

// A required field left in OPTIONS sits behind the "more" fold, where
// focusFirstMissing parks a cursor the user cannot see. opcua's --endpoint is
// the case: it is the only thing the connector takes and it carries
// FlagOption_Required.
func TestNetworkBRequiredFieldsAreReachable(t *testing.T) {
	for _, name := range networkBConnectors {
		_, f := formFor(t, name)
		for _, fd := range f.Fields() {
			if fd.Required && fd.Section == sectionOptions {
				t.Errorf("%s: %q is required but sits behind the fold in %s",
					name, fd.Label, fd.Section)
			}
		}
	}

	_, f := formFor(t, "opcua")
	i := f.IndexOfFlag("endpoint")
	if i < 0 {
		t.Fatal("opcua has no --endpoint field")
	}
	if !f.Fields()[i].Required {
		t.Error("opcua --endpoint lost its required marking")
	}
	if f.Fields()[i].Section != sectionTarget {
		t.Errorf("opcua --endpoint is in %s, want %s", f.Fields()[i].Section, sectionTarget)
	}
}

// A credential typed into a row that a later choice hid is not part of the
// command, and must not decide how the command is delivered.
//
// This is the unifi sequence: type a password, notice the controller wants an
// API key instead, switch the selector, type the key. The password row is gone
// from the screen and its --password is already absent from args() -- but
// secrets() used to walk every field regardless of visibility, so delivery saw
// two credentials on a form displaying one, and the launcher refused to run a
// screen with nothing wrong on it. Nothing named the stale value, because
// nothing could: it was not being shown.
func TestAHiddenCredentialDoesNotDecideTheRoute(t *testing.T) {
	c, f := formFor(t, "unifi")
	setLabelled(t, &f, "sign in with", "username and password")
	setLabelled(t, &f, "controller", "10.0.0.9")
	f.Fields()[f.IndexOfFlag("password")].SetValue("<PLACEHOLDER-typed-then-abandoned>")

	// The user changes their mind.
	setLabelled(t, &f, "sign in with", "API key")
	f.Fields()[f.IndexOfFlag("api-key")].SetValue("<PLACEHOLDER-the-kept-one>")

	if got := f.Secrets(); len(got) != 1 || got[0].Flag != "api-key" {
		var names []string
		for _, s := range got {
			names = append(names, s.Flag)
		}
		t.Fatalf("the form carries %v, want only the api-key the screen is showing", names)
	}

	p := withParser(t, &fakeParser{secretFlag: "api-key"})
	plan, err := (launchRequest{form: f}).plan(c, scanAction())
	if plan.cleanup != nil {
		defer plan.cleanup()
	}
	if err != nil {
		t.Fatalf("a form showing one credential was refused: %v", err)
	}
	if got, _ := p.sentValue("api-key"); got != "<PLACEHOLDER-the-kept-one>" {
		t.Errorf("the key reached the provider as %q", got)
	}
	if _, sent := p.sentValue("password"); sent {
		t.Error("the abandoned password was handed to the provider anyway")
	}
	// And the abandoned password goes nowhere at all -- not to argv, and not
	// to the child's environment either.
	for _, s := range append(append([]string{}, plan.args...), plan.env...) {
		if strings.Contains(s, "<PLACEHOLDER-typed-then-abandoned>") {
			t.Errorf("the hidden password was carried anyway: %q", s)
		}
	}
}

// unifi's two ways in are alternatives, and each choice shows exactly one
// credential.
func TestUnifiOffersOneCredentialAtATime(t *testing.T) {
	cases := map[string][]string{
		"username and password": {"username", "password", "ask-pass"},
		"API key":               {"api-key"},
	}
	all := []string{"username", "password", "ask-pass", "api-key"}

	for choice, want := range cases {
		_, f := formFor(t, "unifi")
		setLabelled(t, &f, "sign in with", choice)

		visible := map[string]bool{}
		for _, i := range f.VisibleIndices() {
			visible[f.Fields()[i].Flag] = true
		}
		shown := map[string]bool{}
		for _, flag := range want {
			shown[flag] = true
			if !visible[flag] {
				t.Errorf("unifi %q hides --%s, which that choice needs", choice, flag)
			}
		}
		for _, flag := range all {
			if !shown[flag] && visible[flag] {
				t.Errorf("unifi %q also shows --%s, which belongs to the other way in", choice, flag)
			}
		}
		// The controller is asked for either way.
		if !visible["hostname"] {
			t.Errorf("unifi %q does not ask for the controller", choice)
		}
	}
}

// ipinfo is on the list this file was written from and carries nothing to
// curate: no flags, no positional arguments, no discovery targets. A spec
// would name flags that do not exist, and applySpec would drop every one of
// them in silence -- which is the exact failure the spec tests exist to catch.
//
// Its credential is env-only and flagless too: the connection reads
// IPINFO_TOKEN directly and nothing on the command line carries it, so there is
// nothing for a form to collect and nothing for ParseCLI to be handed. The
// launcher passes its own environment through, so the variable works as the
// provider documents it.
//
// This is a drift guard rather than a formality: the day ipinfo declares a
// flag, HasFormData turns true, it appears in the snapshot, and this says so.
func TestIPInfoHasNothingToCurate(t *testing.T) {
	if _, ok := formSpecs["ipinfo"]; ok {
		t.Error("ipinfo has a spec, but declares no flags for one to name")
	}
	if _, ok := snapshotByName(t)["ipinfo"]; ok {
		t.Error("ipinfo now carries form metadata and should be curated")
	}

	// And where it is installed, confirm that is a property of the connector
	// rather than of the snapshot being stale.
	for _, c := range BuildCatalog() {
		if c.Name != "ipinfo" || !c.Installed {
			continue
		}
		if c.HasFormData() {
			t.Errorf("ipinfo declares form metadata locally (%d flags, MaxArgs=%d, discovery=%v); curate it",
				len(c.Flags), c.MaxArgs, c.Discovery)
		}
	}
}
