// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	"sort"
	"strings"
	"testing"
)

// The curated Cloud & Virtualization forms, and what a screen for one has to
// get right.
//
// TestEverySpecNamesRealFlags already proves generically, over the whole
// registry, that every flag these specs name exists. What it cannot prove is
// the half that decides whether a curated credential section produces a usable
// screen: that the secret reaches the provider and reaches nothing else. That
// is caught here, at authoring time, rather than in a demo.
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

// The gate over this file's six connectors.
//
// TestEverySpecNamesRealFlags already proves that every flag named here exists,
// generically over the whole registry. What it cannot prove is the half that
// decides whether a curated credential section produces a usable screen: that
// the secret reaches the provider and reaches nothing else. That is caught
// here, at authoring time, rather than in a demo.

// cloudAConnectors are the six this file curates.
var cloudAConnectors = filedHere(
	"alicloud", "digitalocean", "equinix", "hcp", "hetzner", "nutanix",
)

func TestCloudASpecsAreRegistered(t *testing.T) {
	for _, name := range cloudAConnectors {
		if _, ok := formSpecs[name]; !ok {
			t.Errorf("%s has no registered spec", name)
		}
	}
}

// Every secret a curated credential section offers reaches the provider, and
// none of them reaches the command line. One secret is filled at a time,
// because that is how a form is actually used.
func TestCloudACredentialsReachTheProvider(t *testing.T) {
	const secret = "<PLACEHOLDER-not-a-real-secret>"

	for _, name := range cloudAConnectors {
		spec := formSpecs[name]
		c, base := formFor(t, name)

		for _, flag := range spec.Credential {
			i := base.IndexOfFlag(flag)
			if i < 0 {
				t.Errorf("%s: --%s is in Credential but built no field", name, flag)
				continue
			}
			if !base.Fields()[i].Secret {
				// A non-secret in CREDENTIAL is deliberate -- an AccessKey id
				// belongs beside the secret it pairs with -- and carries no
				// delivery question.
				continue
			}

			f := base
			f.SetFields(append([]field(nil), base.Fields()...))
			f.Fields()[i].SetValue(secret)

			assertCredentialReachesTheProvider(t, c, f, flag, secret)
			if strings.Contains(strings.Join(f.Args(), " "), secret) {
				t.Errorf("%s: --%s reached argv: %v", name, flag, f.Args())
			}
		}
	}
}

// nutanix's two credentials both travel, and a typed password is no longer
// swapped for a prompt.
//
// It used to be: --ask-pass is declared, so a typed password took the prompt
// route, the typed value was dropped and the child asked for it again. The
// toggle is still on the form -- ticking it is the best route there is, because
// the value never exists outside the process that uses it -- but it is now the
// user's choice rather than a substitution made on their behalf.
func TestNutanixCarriesEitherCredentialAndStillOffersThePrompt(t *testing.T) {
	c, base := formFor(t, "nutanix")

	fill := func(flag string) form {
		f := base
		f.SetFields(append([]field(nil), base.Fields()...))
		i := f.IndexOfFlag(flag)
		if i < 0 {
			t.Fatalf("nutanix built no --%s field", flag)
		}
		f.Fields()[i].SetValue("<PLACEHOLDER-not-a-real-secret>")
		return f
	}

	for _, flag := range []string{"password", "api-key"} {
		f := fill(flag)
		if got := deliveryFor(f); got != deliverInventory {
			t.Errorf("--%s routes as %v, want the inventory", flag, got)
		}
		assertCredentialReachesTheProvider(t, c, f, flag, "<PLACEHOLDER-not-a-real-secret>")
	}

	// Ticked with nothing typed: an ordinary flag on an ordinary command line.
	ask := base
	ask.SetFields(append([]field(nil), base.Fields()...))
	i := ask.IndexOfFlag("ask-pass")
	if i < 0 {
		t.Fatal("nutanix lost its --ask-pass toggle")
	}
	ask.Fields()[i].SetOn(true)
	if got := deliveryFor(ask); got != deliverPlain {
		t.Errorf("--ask-pass on its own routes as %v, want the command line", got)
	}
	if !strings.Contains(strings.Join(ask.Args(), " "), "--ask-pass") {
		t.Errorf("--ask-pass did not reach the command line: %v", ask.Args())
	}
}

// nutanix's --endpoint is the connector's one required flag. A required field
// left in sectionOptions sits behind the "more" fold, where focusFirstMissing
// parks a cursor that moveFocus cannot reach.
func TestCloudARequiredFieldsAreReachable(t *testing.T) {
	for _, name := range cloudAConnectors {
		_, f := formFor(t, name)
		for _, fd := range f.Fields() {
			if fd.Required && fd.Section == sectionOptions {
				t.Errorf("%s: %q is required but sits behind the options fold",
					name, fd.Label)
			}
		}
	}
}

// equinix takes `org <id>` or `project <id>`: two positional arguments, which
// the usage string only hints at and MinArgs=2 records. The words are the
// connector's own, and both have to reach the command line in order.
func TestEquinixEmitsItsSubcommandPair(t *testing.T) {
	_, f := formFor(t, "equinix")

	kind := fieldByLabel(t, f, "kind")
	if kind.Kind != fieldChoice || !kind.Strict {
		t.Errorf("the kind field is %v (strict=%v), want a closed choice",
			kind.Kind, kind.Strict)
	}
	kind.SetValue("project")
	fieldByLabel(t, f, "id").SetValue("proj-123")

	got := strings.Join(f.Args(), " ")
	if got != "project proj-123" {
		t.Errorf("args = %q, want %q", got, "project proj-123")
	}
}

// The connectors whose account *is* the target lead with --discover, because
// otherwise TARGET is empty and the one question worth asking sits behind the
// "more" fold.
func TestAccountWideConnectorsLeadWithDiscovery(t *testing.T) {
	for _, name := range []string{"digitalocean", "hetzner"} {
		_, f := formFor(t, name)
		i := f.IndexOfFlag("discover")
		if i < 0 {
			t.Errorf("%s built no --discover field", name)
			continue
		}
		if f.Fields()[i].Section != sectionTarget {
			t.Errorf("%s: --discover sits in %q, want TARGET",
				name, f.Fields()[i].Section)
		}
	}
}

// The ambient credential widgets belong to source_ambient.go, and this file
// curates around them rather than duplicating them. What a spec is allowed to
// change is the label and the section; what it must not do is turn the paste
// box back into an ordinary text field or take the readout away.
func TestAmbientWidgetsSurviveTheseSpecs(t *testing.T) {
	withAmbientEnv(t, nil)
	for _, name := range []string{"digitalocean", "hetzner"} {
		_, f := formFor(t, name)

		i := f.IndexOfFlag("token")
		if i < 0 {
			t.Errorf("%s built no --token field", name)
			continue
		}
		if f.Fields()[i].Kind != fieldPaste {
			t.Errorf("%s: --token is a %v, want the paste box", name, f.Fields()[i].Kind)
		}
		if f.Fields()[i].Section != sectionCredential {
			t.Errorf("%s: --token sits in %q", name, f.Fields()[i].Section)
		}
		if f.IndexOfSpecial(specialCredentialState) < 0 {
			t.Errorf("%s: the credential readout is gone", name)
		}
	}

	// digitalocean's second, report-only readout is part of the same
	// arrangement and must not have picked up a flag from this file.
	_, f := formFor(t, "digitalocean")
	if i := f.IndexOfSpecial(specialCredentialState + ".spaces"); i < 0 {
		t.Error("the digitalocean spaces readout is gone")
	} else if f.Fields()[i].Flag != "" {
		t.Errorf("the spaces readout acquired --%s", f.Fields()[i].Flag)
	}
}

// The command each curated form assembles, spelled out.
//
// Every case that carries a credential now reads the same, because there is one
// route and the bar names the file it will write rather than a command with a
// variable in front of it. What is still worth asserting is which cases those
// are: a connector that grew a credential, or lost one, changes a line here.
//
// The old expectations were transcripts -- each was run against the installed
// provider while writing this file and got past its credential gate -- and what
// they were checking was that the launcher had picked the right one of four
// routes. There is no choice left to get wrong.
func TestCloudAAssemblesTheVerifiedCommands(t *testing.T) {
	withAmbientEnv(t, nil)

	cases := []struct {
		connector string
		fill      map[string]string
		positions []string
		want      string
	}{{
		connector: "alicloud",
		fill: map[string]string{
			"region": "cn-hangzhou", "access-key-id": "AK", "access-key-secret": "SK",
		},
		want: "cnspec scan --inventory-file <generated, 0600>",
	}, {
		connector: "digitalocean",
		fill:      map[string]string{"token": "dop_v1_x"},
		want:      "cnspec scan --inventory-file <generated, 0600>",
	}, {
		connector: "equinix",
		fill:      map[string]string{"token": "eq-token"},
		positions: []string{"org", "org-123"},
		want:      "cnspec scan --inventory-file <generated, 0600>",
	}, {
		connector: "hcp",
		fill:      map[string]string{"client-id": "cid", "client-secret": "csec"},
		want:      "cnspec scan --inventory-file <generated, 0600>",
	}, {
		connector: "hetzner",
		fill:      map[string]string{"token": "hz-token"},
		want:      "cnspec scan --inventory-file <generated, 0600>",
	}, {
		// Basic auth: the child prompts, so nothing is carried at all.
		connector: "nutanix",
		fill: map[string]string{
			"endpoint": "pc.example.com", "user": "admin", "password": "pw",
		},
		want: "cnspec scan --inventory-file <generated, 0600>",
	}, {
		// Both nutanix credentials go in the file, as every credential does.
		connector: "nutanix",
		fill:      map[string]string{"endpoint": "pc.example.com", "api-key": "key"},
		want:      "cnspec scan --inventory-file <generated, 0600>",
	}}

	for _, tc := range cases {
		c, f := formFor(t, tc.connector)
		for flag, v := range tc.fill {
			i := f.IndexOfFlag(flag)
			if i < 0 {
				t.Errorf("%s built no --%s field", tc.connector, flag)
				continue
			}
			f.Fields()[i].SetValue(v)
		}
		for _, fd := range positionalFields(&f) {
			if fd.Pos < len(tc.positions) {
				fd.SetValue(tc.positions[fd.Pos])
			}
		}

		if got := (launchRequest{form: f}).preview(c, scanAction()); got != tc.want {
			t.Errorf("%s preview:\n got %q\nwant %q", tc.connector, got, tc.want)
		}
		// Whatever the route, none of it may be on the command line.
		for _, fd := range f.Secrets() {
			if strings.Contains(strings.Join(f.Args(), " "), fd.Value()) {
				t.Errorf("%s: the secret reached argv: %v", tc.connector, f.Args())
			}
		}
	}
}

// alicloud declares MinArgs=0 and MaxArgs=0, so anything this form emitted as a
// bare argument would make the command unrunnable -- `cnspec shell alicloud
// staging` answers `unknown command "staging"`.
//
// The profile is attached now, and this is the assertion that says how: it is a
// launcher-owned field, it contributes ALIBABA_CLOUD_PROFILE, and it puts
// nothing on the command line. The earlier version of this test locked the
// workaround -- "the spec declares no positional fields at all" -- because
// there was no way to have one without the other. There is now; see
// PositionalSpec.Special, and the note at the bottom of cli/launcher/forms/forms_cloud.go for
// what was tried before it.
func TestAlicloudProfileTravelsWithoutABareArgument(t *testing.T) {
	snap, ok := snapshotByName(t)["alicloud"]
	if !ok {
		t.Fatal("alicloud is not in the connector snapshot")
	}
	if snap.MaxArgs != 0 {
		t.Fatalf("alicloud now takes %d arguments; the profile could travel in one",
			snap.MaxArgs)
	}

	c := snap.connector()
	f := newForm(c)
	profile := fieldByLabel(t, f, "profile")
	if profile.Special != specialAlicloudProfile {
		t.Fatalf("the profile field is %q, want the launcher-owned marker %q",
			profile.Special, specialAlicloudProfile)
	}
	if profile.Source() != srcAlicloudProfile {
		t.Errorf("the profile field has source %q, want %q", profile.Source(), srcAlicloudProfile)
	}

	// Nothing on this form is a plain positional, and filling everything in
	// still emits no bare argument.
	for _, fd := range positionalFields(&f) {
		t.Errorf("alicloud built a positional field %q", fd.Label)
	}
	for i := range f.Fields() {
		f.Fields()[i].SetValue("filled")
	}
	for _, a := range f.Args() {
		if !strings.HasPrefix(a, "-") && !isFlagValue(f, a) {
			t.Errorf("alicloud emitted the bare argument %q", a)
		}
	}

	// And the profile does reach the child, as a variable with a value -- the
	// other half of the trap, since an Emit map suppressing the argument would
	// have produced ALIBABA_CLOUD_PROFILE= instead.
	f = newForm(c)
	fieldByLabel(t, f, "profile").SetValue("staging")
	r := launchRequest{form: f}
	env, cleanup, err := r.environment()
	if cleanup != nil {
		defer cleanup()
	}
	if err != nil {
		t.Fatal(err)
	}
	if !containsString(env, alicloudProfileEnv+"=staging") {
		t.Errorf("the chosen profile did not reach the child: %v", env)
	}

	// An unset profile contributes nothing at all, rather than an empty
	// variable the SDK would read as an explicit empty profile.
	r = launchRequest{form: newForm(c)}
	if env, _, _ := r.environment(); len(env) != 0 {
		t.Errorf("an untouched form contributed %v", env)
	}
}

// isFlagValue reports whether an emitted word is the value following a flag
// rather than a positional argument. args() emits "--flag value" pairs, so the
// check is simply whether any flag field holds it.
func isFlagValue(f form, word string) bool {
	for _, fd := range f.Fields() {
		if fd.Flag != "" && fd.Emitted() == word {
			return true
		}
	}
	return false
}

// cloudBConnectors are the six cloud connectors that were curated second: three cloud APIs and three hypervisors.
var cloudBConnectors = filedHere(
	"oci", "openstack", "proxmox", "stackit", "vcd", "vsphere",
)

// All six are registered, and by this file rather than by accident: a spec that
// silently failed to register leaves the connector on the generic screen, which
// looks fine and is not what was curated.
func TestCloudBRegistersItsConnectors(t *testing.T) {
	for _, name := range cloudBConnectors {
		if _, ok := formSpecs[name]; !ok {
			t.Errorf("%s has no registered form spec", name)
		}
	}
}

// The heart of it: a credential section that cannot deliver produces a form
// that refuses to launch at the end of filling it in. Every secret field these
// specs put in front of a user is filled here and the delivery checked.
//
// oci --key-secret is the deliberate exception and is asserted separately
// below, so that removing its exclusion without finding it a route fails here
// rather than in a demo.
func TestCloudBCredentialsCanBeDelivered(t *testing.T) {
	for _, name := range cloudBConnectors {
		c, f := formFor(t, name)

		var secretFlags []string
		for _, fd := range f.Fields() {
			if fd.Secret {
				secretFlags = append(secretFlags, fd.Flag)
			}
		}
		sort.Strings(secretFlags)

		for _, flag := range secretFlags {
			if name == "oci" && flag == "key-secret" {
				continue
			}
			// One secret at a time: the routes that carry exactly one are the
			// ones that exist, and a form with two credentials filled in is not
			// something any of these connectors asks for.
			_, filled := formFor(t, name)
			i := filled.IndexOfFlag(flag)
			if i < 0 {
				t.Errorf("%s: --%s vanished between two builds of the same form", name, flag)
				continue
			}
			filled.Fields()[i].SetValue("<PLACEHOLDER-not-a-real-secret>")

			if deliveryFor(filled) == deliverPlain {
				t.Errorf("%s: --%s is not being treated as a secret at all", name, flag)
				continue
			}
			assertCredentialReachesTheProvider(t, c, filled, flag, "<PLACEHOLDER-not-a-real-secret>")
			if strings.Contains(strings.Join(filled.Args(), " "), "<PLACEHOLDER-not-a-real-secret>") {
				t.Errorf("%s: --%s reached argv: %v", name, flag, filled.Args())
			}
		}

		if len(secretFlags) == 0 {
			t.Errorf("%s: no secret field was built, so this proved nothing", name)
		}
	}
}

// oci's --key-secret is a private key passphrase and travels like every other
// credential.
//
// Its history is the argument for this whole change. It was first a refusal --
// oci declares no --ask-key-secret, reads no environment variable of its own,
// and the launcher's own inventory builder could not produce the private_key
// credential carrying key and passphrase together that NewOciConnection
// requires. Then it was MONDOO_KEY_SECRET, a variable derived from the flag
// name that works only because mql fills any flag with no config mapping from
// one. Both were the launcher reasoning about a provider it could have asked:
// oci's own ParseCLI builds exactly the private_key credential it wants, which
// is what the round trip now finds there.
func TestOCIKeySecretIsCarriedAsACredential(t *testing.T) {
	c, f := formFor(t, "oci")
	i := f.IndexOfFlag("key-secret")
	if i < 0 {
		t.Fatal("oci no longer declares --key-secret")
	}
	if !f.Fields()[i].Secret {
		t.Error("oci --key-secret is a private key passphrase and must be classified as a secret")
	}
	f.Fields()[i].SetValue("<PLACEHOLDER-not-a-real-secret>")

	assertCredentialReachesTheProvider(t, c, f, "key-secret", "<PLACEHOLDER-not-a-real-secret>")
}

// stackit --service-account-key is a JSON key blob whose name the shared
// classifier does not read as a credential: it ends in "-key", which is neither
// a strong secret word nor a reference. Without the spec's Secret override the
// whole key goes on the command line.
func TestStackitServiceAccountKeyIsASecret(t *testing.T) {
	c, f := formFor(t, "stackit")

	i := f.IndexOfFlag("service-account-key")
	if i < 0 {
		t.Fatal("stackit no longer declares --service-account-key")
	}
	if !f.Fields()[i].Secret {
		t.Fatal("--service-account-key is not marked secret; the spec's Secret override is not taking effect")
	}
	if f.Fields()[i].Section != sectionCredential {
		t.Errorf("--service-account-key is in %q, want %q", f.Fields()[i].Section, sectionCredential)
	}

	f.Fields()[i].SetValue(`{"client_id":"x","privateKey":"y"}`)
	assertCredentialReachesTheProvider(t, c, f, "service-account-key",
		`{"client_id":"x","privateKey":"y"}`)

	// The two path flags name where a key lives rather than holding one, and
	// have to stay on the command line: that is how the provider receives them.
	for _, flag := range []string{"service-account-key-path", "private-key-path"} {
		j := f.IndexOfFlag(flag)
		if j < 0 {
			t.Errorf("stackit no longer declares --%s", flag)
			continue
		}
		if f.Fields()[j].Secret {
			t.Errorf("--%s names a file and must not be treated as a secret", flag)
		}
	}
}

// proxmox's token reaches the provider, and the inventory the launcher writes
// is the provider's own reading of the form rather than the launcher's guess at
// it.
//
// The guess happened to be right here -- Connect reads conf.Options["host"] and
// conf.Options["token"], which is what a secret with no accompanying --user
// produced -- and that is exactly why it is worth replacing: being right was
// something a person had checked once, in a comment, against a provider free to
// change.
func TestProxmoxInventoryIsTheProvidersOwnReading(t *testing.T) {
	c, f := formFor(t, "proxmox")

	set := func(flag, value string) {
		i := f.IndexOfFlag(flag)
		if i < 0 {
			t.Fatalf("proxmox no longer declares --%s", flag)
		}
		f.Fields()[i].SetValue(value)
	}
	set("host", "https://pve.example.com:8006")
	set("token", "PVEAPIToken=root@pam!ui=<PLACEHOLDER-not-a-real-secret>")

	if got := deliveryFor(f); got != deliverInventory {
		t.Fatalf("delivery = %v, want deliverInventory", got)
	}
	p := withParser(t, &fakeParser{secretFlag: "token"})
	plan, err := (launchRequest{form: f}).plan(c, scanAction())
	if plan.cleanup != nil {
		defer plan.cleanup()
	}
	if err != nil {
		t.Fatalf("the launch was refused: %v", err)
	}
	if got, _ := p.sentValue("host"); got != "https://pve.example.com:8006" {
		t.Errorf("the host reached the provider as %q", got)
	}
	if got, _ := p.sentValue("token"); got != "PVEAPIToken=root@pam!ui=<PLACEHOLDER-not-a-real-secret>" {
		t.Errorf("the token reached the provider as %q", got)
	}
	if strings.Contains(strings.Join(plan.args, " "), "<PLACEHOLDER-not-a-real-secret>") {
		t.Errorf("the token reached argv: %v", plan.args)
	}
}

// vsphere is addressed by one positional argument in the user@realm@host shape,
// not by a flag. ParseCLI prepends a scheme and hands it to url.Parse, so the
// user half is everything before the last @ -- there is nowhere else for the
// realm to go, and no flag a spec could map it onto.
func TestVsphereTargetIsOnePositional(t *testing.T) {
	c, f := formFor(t, "vsphere")
	if c.MinArgs != 1 || c.MaxArgs != 1 {
		t.Fatalf("vsphere declares %d..%d args, expected exactly one", c.MinArgs, c.MaxArgs)
	}

	var positional []field
	for _, fd := range f.Fields() {
		if fd.Flag == "" {
			positional = append(positional, fd)
		}
	}
	if len(positional) != 1 {
		t.Fatalf("expected one positional field, got %d", len(positional))
	}
	p := positional[0]
	if !p.Required {
		t.Error("the vsphere target is required; MinArgs is 1")
	}
	if p.Section != sectionTarget {
		t.Errorf("the target field is in %q, want %q", p.Section, sectionTarget)
	}
	if !strings.Contains(p.Label, "@") {
		t.Errorf("label %q does not say the value is a user@realm@host pair", p.Label)
	}

	// The realm is part of the user half and must not have been invented as a
	// flag of its own.
	if i := f.IndexOfFlag("realm"); i >= 0 {
		t.Error("vsphere declares no --realm; the realm belongs in the positional argument")
	}

	// Filling it in has to produce the argument verbatim, ahead of any flag.
	f.Fields()[0].SetValue("chris@vsphere.local@vcenter.example.com")
	args := f.Args()
	if len(args) == 0 || args[0] != "chris@vsphere.local@vcenter.example.com" {
		t.Errorf("args = %v, want the target as the leading argument", args)
	}
}

// vcd marks --user and --host required, and a required field left in OPTIONS
// sits behind the "more" fold, where the cursor focusFirstMissing parks cannot
// be reached. Both have to be in TARGET.
func TestVCDRequiredFieldsAreReachable(t *testing.T) {
	_, f := formFor(t, "vcd")
	for _, fd := range f.Fields() {
		if fd.Required && fd.Section == sectionOptions {
			t.Errorf("--%s is required but sits in %q", fd.Flag, sectionOptions)
		}
	}
	for _, flag := range []string{"host", "user"} {
		section, ok := sectionOf(f, flag)
		if !ok {
			t.Errorf("vcd no longer declares --%s", flag)
			continue
		}
		if section != sectionTarget {
			t.Errorf("--%s is in %q, want %q", flag, section, sectionTarget)
		}
	}
}

// The two hypervisors declare an --ask-pass partner, and it stays a toggle
// rather than a substitution.
//
// It used to be the route: a typed password was dropped, --ask-pass went on the
// command line in its place, and the child asked for the password the user had
// already given it. Prompting is still the strongest thing available -- the
// value never exists outside the process that uses it -- and it is still one
// keystroke away.
func TestAskPassStaysAToggleForVCDAndVsphere(t *testing.T) {
	for _, name := range []string{"vcd", "vsphere"} {
		c, f := formFor(t, name)
		i := f.IndexOfFlag("password")
		if i < 0 {
			t.Errorf("%s no longer declares --password", name)
			continue
		}
		f.Fields()[i].SetValue("<PLACEHOLDER-not-a-real-secret>")

		if got := deliveryFor(f); got != deliverInventory {
			t.Errorf("%s: a typed password routes as %v, want the inventory", name, got)
		}
		assertCredentialReachesTheProvider(t, c, f, "password", "<PLACEHOLDER-not-a-real-secret>")

		// Ticked with nothing typed: an ordinary flag on an ordinary command
		// line.
		_, ask := formFor(t, name)
		j := ask.IndexOfFlag("ask-pass")
		if j < 0 {
			t.Errorf("%s no longer offers --ask-pass", name)
			continue
		}
		ask.Fields()[j].SetOn(true)
		if got := deliveryFor(ask); got != deliverPlain {
			t.Errorf("%s: --ask-pass on its own routes as %v, want the command line", name, got)
		}
		if !strings.Contains(strings.Join(ask.Args(), " "), "--ask-pass") {
			t.Errorf("%s: --ask-pass did not reach the command line: %v", name, ask.Args())
		}
	}
}

// oci's --profile is the one picker in this file. The default section in
// ~/.oci/config is the literal DEFAULT rather than the lowercase spelling the
// AWS convention would suggest, which is what srcOCIProfile knows and a
// hand-written option list would get wrong.
func TestOCIProfileHasThePicker(t *testing.T) {
	_, f := formFor(t, "oci")
	i := f.IndexOfFlag("profile")
	if i < 0 {
		t.Fatal("oci no longer declares --profile")
	}
	if f.Fields()[i].Source() != srcOCIProfile {
		t.Errorf("--profile source = %q, want %q", f.Fields()[i].Source(), srcOCIProfile)
	}
	if f.Fields()[i].Kind != fieldChoice {
		t.Errorf("--profile is kind %v, want a picker", f.Fields()[i].Kind)
	}

	// --tenancy is scope, not a discovery output: oci has no way to list
	// tenancies before it is authenticated to one, and --filters takes regions,
	// compartments and tags rather than an account. Nothing may quietly attach
	// a picker to it.
	if j := f.IndexOfFlag("tenancy"); j >= 0 {
		if f.Fields()[j].Source() != "" || f.Fields()[j].LiveSource != "" {
			t.Errorf("--tenancy has a picker attached (%q/%q); it is required input, not a discovery result",
				f.Fields()[j].Source(), f.Fields()[j].LiveSource)
		}
	}
	// The same holds for stackit's project id.
	_, sf := formFor(t, "stackit")
	if j := sf.IndexOfFlag("project-id"); j >= 0 {
		if sf.Fields()[j].Source() != "" || sf.Fields()[j].LiveSource != "" {
			t.Errorf("stackit --project-id has a picker attached (%q/%q); it is required input",
				sf.Fields()[j].Source(), sf.Fields()[j].LiveSource)
		}
	}
}

// Every option list offered has to be one the provider accepts. auth-method is
// the strict one -- oci's ParseCLI rejects anything outside its own
// SupportedAuthMethods before the scan starts -- so a stale entry here is a
// hard error rather than a bad suggestion.
func TestOCIChoicesAreAccepted(t *testing.T) {
	_, f := formFor(t, "oci")

	i := f.IndexOfFlag("auth-method")
	if i < 0 {
		t.Fatal("oci no longer declares --auth-method")
	}
	want := map[string]bool{
		"api-key": true, "instance-principal": true, "resource-principal": true,
		"workload-identity": true, "security-token": true,
	}
	if len(f.Fields()[i].Options) != len(want) {
		t.Errorf("auth-method offers %v, want %d entries", f.Fields()[i].Options, len(want))
	}
	for _, opt := range f.Fields()[i].Options {
		if !want[opt] {
			t.Errorf("auth-method offers %q, which the provider rejects", opt)
		}
	}

	// Regions are a suggestion list, not a validation list: a region newer than
	// this binary still has to be typeable.
	j := f.IndexOfFlag("region")
	if j < 0 {
		t.Fatal("oci no longer declares --region")
	}
	if len(f.Fields()[j].Options) == 0 {
		t.Error("no regions are offered")
	}
	if f.Fields()[j].Strict {
		t.Error("the region list is strict, so a region newer than this binary cannot be entered")
	}
	for _, r := range f.Fields()[j].Options {
		if !strings.Contains(r, "-") {
			t.Errorf("%q does not look like an OCI region identifier", r)
		}
	}
}

// openstack's scope and its credentials are two different things and the form
// has to keep them apart: the username and the domain that qualifies it belong
// with the password, the project and the region are the target.
func TestOpenstackSectionsSplitScopeFromCredentials(t *testing.T) {
	_, f := formFor(t, "openstack")

	inTarget := []string{"cloud", "auth-url", "region", "project-name", "project-id"}
	inCredential := []string{"username", "password", "application-credential-secret"}

	for _, flag := range inTarget {
		if section, ok := sectionOf(f, flag); !ok || section != sectionTarget {
			t.Errorf("--%s is in %q, want %q", flag, section, sectionTarget)
		}
	}
	for _, flag := range inCredential {
		if section, ok := sectionOf(f, flag); !ok || section != sectionCredential {
			t.Errorf("--%s is in %q, want %q", flag, section, sectionCredential)
		}
	}

	// --cloud leads, because choosing a clouds.yaml entry answers the auth URL,
	// the project and the region at once.
	for _, fd := range f.Fields() {
		if fd.Section != sectionTarget {
			continue
		}
		if fd.Flag != "cloud" {
			t.Errorf("the first TARGET field is --%s, want --cloud", fd.Flag)
		}
		break
	}
}
