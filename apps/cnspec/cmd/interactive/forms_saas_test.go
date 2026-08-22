// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	"strings"
	"testing"
)

// The curated SaaS forms: the accounts an organisation holds with someone else.
//
// This file used to be three -- forms_saas_a_test.go, _b and _c -- and the
// letter recorded which contributor wrote which third rather than anything
// about the connectors. Two of those three also carried Identity connectors,
// because the person who curated atlassian also curated activedirectory, so
// "the SaaS tests" and "the tests one person wrote" were the same list and only
// one of them was a category. They are filed by what the launcher shows now;
// see forms_filing_test.go for the rule and what it can and cannot enforce.
//
// Every test below drives the real form -- newForm over the connector rebuilt
// from internal/connectors/connectors.json -- rather than inspecting the
// FormSpec literal, because a spec that names a flag the connector does not
// declare is silently dropped by applySpec. Reading the spec back would agree
// with itself; building the form is what proves the flag survived. That is also
// why they build from the recorded artifact rather than from the machine: CI
// runs with an empty provider set, and a check that skipped itself there would
// prove nothing exactly where the proof is needed.

// saasConnectors is every connector the launcher files under SaaS that this
// file curates.
//
// The package-wide gates check the specs that exist. What they cannot check is
// a connector nobody registered, which looks exactly like a connector nobody
// was assigned -- so the list is written down, and filedHere is what proves
// every name on it really is a SaaS connector rather than one that drifted in.
var saasConnectors = filedHere(
	"atlassian", "cloudflare", "databricks", "datadog", "dropbox", "gitlab",
	"grafana", "iru", "jamf", "mondoo", "mongodbatlas", "netlify", "nextdns",
	"slack", "snowflake", "tailscale", "vercel", "zoom",
)

// Every connector this file claims has a spec, and exactly one.
//
// Both halves matter and they fail differently. registerSpec is first-wins, so
// a sibling file that also registered one of these would take it and the only
// visible sign would be an entry in duplicateSpecs; that is the second check.
// The first is the opposite failure -- a connector named here whose init was
// removed, which leaves it on the generic flag-derived screen. That screen
// looks like a form and says nothing about being uncurated.
//
// The environment routes this used to pin as well -- JAMF_CLIENT_SECRET,
// MONGODB_ATLAS_PRIVATE_KEY, TAILSCALE_API_KEY and a dozen more -- are gone
// with the registry that held them. Three of them were the interesting ones:
// jamf routes its credential on cred.User carrying the client id, mongodbatlas
// on the labels "private-key" and "client-secret", and okta's key wants a
// private_key credential rather than a password. Pinning a variable name never
// checked any of that.
func TestSaaSSpecsAreRegisteredExactlyOnce(t *testing.T) {
	for _, name := range saasConnectors {
		if _, ok := formSpecs[name]; !ok {
			t.Errorf("%s has no registered spec", name)
		}
		if containsString(duplicateSpecs, name) {
			t.Errorf("%s was registered twice, so two files claim it", name)
		}
	}
}

// saasTargetLeads is what each SaaS form's TARGET section leads with, by the
// label the spec gives it, and "" for the connectors whose whole form is the
// credential.
//
// This was three tables in three files asking three different questions of
// three thirds of the list: one checked a label was in TARGET and visible, one
// checked the first field carried a particular flag, one checked which label
// led TARGET. They are the same claim -- TARGET is the first section on the
// pane, so what leads it is what the launcher says this connector is addressed
// by -- and none of the three covered the whole category. One table does, and
// every entry is a decision that is not obvious from the flag list: snowflake
// leads with the account rather than the user, tailscale with a positional the
// usage string does not even name, atlassian with a selector that is not a flag
// at all.
var saasTargetLeads = map[string]string{
	"atlassian":    "product",
	"databricks":   "connect to",
	"datadog":      "site",
	"gitlab":       "group",
	"grafana":      "instance URL",
	"iru":          "tenant subdomain",
	"jamf":         "Jamf Pro URL",
	"mongodbatlas": "organization id",
	"netlify":      "account (optional)",
	"slack":        "team ID",
	"snowflake":    "account identifier",
	"tailscale":    "tailnet",
	"vercel":       "team (slug or ID)",
	"zoom":         "account ID",

	// The four with nothing to target. A TARGET section that holds an API key
	// is a lie about what the field is, so nothing may be promoted into it to
	// fill the space: cloudflare and dropbox declare one flag each and it is
	// the credential, nextdns the same, and mondoo reads neither a flag nor an
	// argument -- its ParseCLI builds a config from the connector name and
	// returns.
	"cloudflare": "",
	"dropbox":    "",
	"nextdns":    "",
	"mondoo":     "",
}

// The field the connector cannot run without has to be on the screen, in
// TARGET, and visible before anything else is chosen. A curated form that
// buries it -- or gates it behind a selector value that is not the default --
// is worse than the generic screen it replaced.
func TestSaaSTargetsLeadTheirForms(t *testing.T) {
	for _, name := range saasConnectors {
		label, declared := saasTargetLeads[name]
		if !declared {
			t.Errorf("%s is curated here but saasTargetLeads does not say what its "+
				"form is addressed by; add the label, or \"\" if it has no target", name)
			continue
		}
		t.Run(name, func(t *testing.T) {
			_, f := hermeticFormFor(t, name)

			if label == "" {
				for _, fd := range f.Fields() {
					if fd.Section == sectionTarget {
						t.Errorf("%q is in TARGET, but %s has nothing to target",
							fd.Label, name)
					}
				}
				return
			}

			at := -1
			for i := range f.Fields() {
				if f.Fields()[i].Label == label {
					at = i
				}
			}
			if at < 0 {
				t.Fatalf("no field labelled %q; the form asks %v", label, fieldLabels(f))
			}
			if got := f.Fields()[at].Section; got != sectionTarget {
				t.Errorf("%q is in %s, want %s", label, got, sectionTarget)
			}
			if !f.Visible(at) {
				t.Errorf("%q is hidden on a form nobody has touched yet", label)
			}
			// It leads its section: nothing else in TARGET comes before it.
			for i, fd := range f.Fields() {
				if fd.Section == sectionTarget && i < at {
					t.Errorf("%q comes before the target %q", fd.Label, label)
				}
			}
			// And TARGET is the first section, so the target is the first thing
			// the user meets rather than something below a credential.
			if at != 0 {
				t.Errorf("the form leads with %q, want the target %q",
					f.Fields()[0].Label, label)
			}
		})
	}
}

// The target flags, named as flags rather than as the labels the lead table
// uses. mongodbatlas asks for a project inside the organization and neither is
// the lead; jamf and netlify are named here because the flag behind the label
// is the part that has to survive -- a spec that relabelled --instance-domain
// onto a different flag would still lead with "Jamf Pro URL".
//
// All of these are what is being scanned rather than who is scanning it, and a
// target left in OPTIONS is below the credential, which reverses the order the
// connector is actually used in.
func TestSaaSSecondaryTargetsStayInTheTargetSection(t *testing.T) {
	targets := map[string][]string{
		"jamf":         {"instance-domain"},
		"mongodbatlas": {"org-id", "project-id"},
		"netlify":      {"account"},
	}
	for name, flags := range targets {
		t.Run(name, func(t *testing.T) {
			_, f := hermeticFormFor(t, name)
			for _, flag := range flags {
				i := f.IndexOfFlag(flag)
				if i < 0 {
					t.Errorf("--%s is not on the form; fields are %v", flag, fieldLabels(f))
					continue
				}
				if got := f.Fields()[i].Section; got != sectionTarget {
					t.Errorf("--%s is in %s, want %s", flag, got, sectionTarget)
				}
			}
		})
	}
}

// A credential a user can fill in has to reach the provider, and must not reach
// a command line every account on the machine can read. The target it was
// filled in beside has to survive being separated from it.
//
// This used to name the environment variable each one travelled in, and the
// list was long: ATLASSIAN_USER_TOKEN, DATABRICKS_TOKEN, DD_APP_KEY, nine in
// all. Every one was a claim about a provider that a person had checked by
// running it, and every one could go stale in silence. There is one route now
// and the provider decides what it carries.
//
// Each secret is filled on its own form rather than all at once. Both
// mongodbatlas and atlassian declare alternative credentials only one of which
// is ever used, so the shapes are listed as a user would fill them -- not
// because the route could only carry one, which it no longer is, but because
// several providers pick a different authentication path when they see the
// second and legitimately drop the first.
func TestSaaSCredentialsReachTheProvider(t *testing.T) {
	for _, tc := range []struct {
		name      string
		connector string
		// flag is the credential field to fill; fill names the fields that have
		// to be filled beside it for the form to make sense, and is checked
		// afterwards to prove the target survived.
		flag string
		fill map[string]string
		// byLabel fills a field the connector has no flag for -- a selector, or
		// a positional the spec named.
		byLabel map[string]string
		// validates marks the shapes that are complete as listed, so the form
		// must accept them. It is not every row: several of these fill one
		// credential onto a form that still has a required target empty, which
		// is how a user meets it rather than how they leave it.
		validates bool
	}{
		{name: "atlassian jira", connector: "atlassian", flag: "user-token",
			byLabel: map[string]string{"product": "jira"}},
		{name: "atlassian admin", connector: "atlassian", flag: "admin-token",
			byLabel: map[string]string{"product": "admin"}},
		{name: "atlassian scim", connector: "atlassian", flag: "scim-token",
			byLabel: map[string]string{"product": "scim"}},
		{name: "databricks workspace", connector: "databricks", flag: "token",
			byLabel: map[string]string{"connect to": "workspace"}},
		{name: "databricks account console", connector: "databricks", flag: "client-secret",
			byLabel: map[string]string{"connect to": "account console"}},
		{name: "datadog API key", connector: "datadog", flag: "api-key"},
		{name: "datadog application key", connector: "datadog", flag: "app-key"},
		{name: "grafana token", connector: "grafana", flag: "token"},
		{name: "cloudflare token", connector: "cloudflare", flag: "token"},
		{name: "gitlab token", connector: "gitlab", flag: "token"},
		{name: "jamf API client", connector: "jamf", flag: "client-secret", validates: true,
			fill: map[string]string{"instance-domain": "https://acme.jamfcloud.com", "client-id": "abc"}},
		{name: "atlas programmatic API key", connector: "mongodbatlas", flag: "private-key", validates: true,
			fill: map[string]string{"org-id": "5f1", "public-key": "pub"}},
		{name: "atlas service account", connector: "mongodbatlas", flag: "client-secret", validates: true,
			fill: map[string]string{"org-id": "5f1", "client-id": "mdb_sa_id"}},
		{name: "netlify personal access token", connector: "netlify", flag: "token", validates: true,
			fill: map[string]string{"account": "acme"}},
		{name: "nextdns API key", connector: "nextdns", flag: "api-key", validates: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, f := hermeticFormFor(t, tc.connector)
			for label, value := range tc.byLabel {
				fieldByLabel(t, f, label).SetValue(value)
			}
			for flag, value := range tc.fill {
				setFlag(t, &f, flag, value)
			}

			i := f.IndexOfFlag(tc.flag)
			if i < 0 {
				t.Fatalf("no --%s field; the form asks %v", tc.flag, fieldLabels(f))
			}
			if !f.Fields()[i].Secret {
				t.Fatalf("--%s is not marked secret, so its value would reach argv", tc.flag)
			}
			if got := f.Fields()[i].Section; got != sectionCredential {
				t.Errorf("--%s is in %s, want %s", tc.flag, got, sectionCredential)
			}
			f.Fields()[i].SetValue(sentinel)
			resolveSources(&f)
			if tc.validates {
				if err := f.Validate(); err != nil {
					t.Fatalf("the filled form does not validate: %v", err)
				}
			}

			p := assertCredentialReachesTheProvider(t, c, f, tc.flag, sentinel)

			// The target has to survive being separated from the credential.
			for flag, value := range tc.fill {
				if got, _ := p.sentValue(flag); got != value {
					t.Errorf("--%s reached the provider as %q, want %q", flag, got, value)
				}
			}
		})
	}
}

// The one that matters most, over every SaaS connector: fill every box and
// check that nothing a user typed into a credential comes back out on a command
// line every account on the machine can read.
//
// This was two sweeps in two files that had drifted apart. One filled a picker
// with its first option and the other with the word "placeholder"; one resolved
// the value sources afterwards and the other did not; one checked f.Args()
// before planning and the other only checked the plan; one failed a connector
// whose form offered no secret at all and the other returned quietly. Every one
// of those differences was an accident of which file the author had open, and
// the strict side of each is the one that is kept here.
func TestNoSaaSSecretReachesArgv(t *testing.T) {
	// mondoo is the one SaaS form with no credential to fill, and it is not an
	// omission: its ParseCLI reads neither a flag nor an argument and the
	// credential comes from the workstation's own registration. Every other
	// connector here must offer one, because a curated CREDENTIAL section that
	// built no field is a screen that cannot be completed.
	noCredential := map[string]bool{"mondoo": true}

	for _, name := range saasConnectors {
		t.Run(name, func(t *testing.T) {
			c, f := hermeticFormFor(t, name)
			secrets := fillEveryField(&f, sentinel)
			resolveSources(&f)

			if secrets == 0 && !noCredential[name] {
				t.Fatalf("no field on the %s form is marked secret", name)
			}
			if secrets > 0 && noCredential[name] {
				t.Fatalf("%s now offers a credential; it is listed as having none", name)
			}

			for _, a := range f.Args() {
				if strings.Contains(a, sentinel) {
					t.Fatalf("a secret reached the command line: %v", f.Args())
				}
			}

			r := launchRequest{form: f}
			plan, err := r.plan(c, scanAction())
			if plan.cleanup != nil {
				defer plan.cleanup()
			}
			if err != nil {
				// A refusal is a correct outcome only when the provider kept
				// the credential nowhere -- which is what a form holding two
				// alternative credentials at once produces, since several
				// providers pick one method and legitimately drop the other --
				// and only when the message says which flag.
				if !strings.Contains(err.Error(), "did not keep") {
					t.Fatalf("unexpected launch failure: %v", err)
				}
				return
			}
			if strings.Contains(strings.Join(plan.args, " "), sentinel) {
				t.Fatalf("a secret reached argv: %v", plan.args)
			}
			// The preview is what tells the user the secret is not on the
			// command line, so it must not contradict itself.
			if preview := (launchRequest{form: f}).preview(c, scanAction()); strings.Contains(preview, sentinel) {
				t.Fatalf("the command preview shows a secret: %q", preview)
			}
		})
	}
}

// A form is navigated by label and args() tracks visibility by label, so two
// fields sharing one would make a hidden field visible.
func TestSaaSLabelsAreUnique(t *testing.T) {
	for _, name := range saasConnectors {
		_, f := hermeticFormFor(t, name)
		seen := map[string]int{}
		for _, fd := range f.Fields() {
			seen[fd.Label]++
		}
		for label, n := range seen {
			if n > 1 {
				t.Errorf("%s: %d fields labelled %q", name, n, label)
			}
		}
	}
}

// The credential fields have to be in CREDENTIAL, which is not automatic for
// the ones that are not secrets: a client id, a user name and the identity file
// a key is read from are all read as ordinary flags and would sit in the
// collapsed OPTIONS row, away from the secret they only mean anything beside.
func TestSaaSPairsIdentifiersWithTheirSecrets(t *testing.T) {
	want := map[string][]string{
		"snowflake": {"user", "identity-file"},
		"tailscale": {"client-id"},
		"zoom":      {"client-id"},
	}

	for name, flags := range want {
		f := newForm(connectorFor(t, name))
		for _, flag := range flags {
			i := f.IndexOfFlag(flag)
			if i < 0 {
				t.Errorf("%s: --%s is not on the form", name, flag)
				continue
			}
			if got := f.Fields()[i].Section; got != sectionCredential {
				t.Errorf("%s: --%s is in %s, want %s", name, flag, got, sectionCredential)
			}
			// These are identifiers, not secrets. Marking one secret would
			// take it off the command line where the provider expects to be
			// shown it, and would put a value that needs no protection into the
			// OS keychain.
			if f.Fields()[i].Secret {
				t.Errorf("%s: --%s is classified as a secret, which it is not", name, flag)
			}
		}
	}
}

// mondoo reads neither a flag nor an argument: its ParseCLI builds a config
// from the connector name and returns, and the credential comes from the
// workstation's registration. The empty spec exists to suppress the box that
// MaxArgs=4 would otherwise produce, so the form has to be empty and the
// launch has to be the bare command.
func TestMondooAsksForNothing(t *testing.T) {
	c, f := hermeticFormFor(t, "mondoo")
	if len(f.Fields()) != 0 {
		t.Fatalf("mondoo asks for %v, but the connector reads nothing", fieldLabels(f))
	}
	if err := f.Validate(); err != nil {
		t.Fatalf("an empty form should be launchable: %v", err)
	}

	r := launchRequest{form: f}
	plan, err := r.plan(c, scanAction())
	if err != nil {
		t.Fatal(err)
	}
	if plan.cleanup != nil {
		defer plan.cleanup()
	}
	if got := strings.Join(plan.args, " "); got != "scan mondoo" {
		t.Fatalf("mondoo launch = %q, want %q", got, "scan mondoo")
	}
}

// atlassian's four products need four different credentials, and the flat
// screen offered all five flags at once -- four of them always wrong. The
// product chosen has to decide what is asked, and the word it emits has to be
// the one the connector's own ParseCLI accepts.
func TestAtlassianAsksOnlyForTheChosenProduct(t *testing.T) {
	for _, tc := range []struct {
		product string
		want    []string // credential labels that must be visible
		gone    []string // and those that must not
		args    []string
	}{
		{product: "jira", want: []string{"site URL", "account email", "API token"},
			gone: []string{"admin API token", "SCIM API token", "directory id"},
			args: []string{"jira"}},
		{product: "confluence", want: []string{"site URL", "account email", "API token"},
			gone: []string{"admin API token", "SCIM API token"},
			args: []string{"confluence"}},
		{product: "admin", want: []string{"admin API token"},
			gone: []string{"API token", "SCIM API token", "site URL", "directory id"},
			args: []string{"admin"}},
		{product: "scim", want: []string{"SCIM API token", "directory id"},
			gone: []string{"API token", "admin API token", "site URL"},
			args: []string{"scim", "dir-1"}},
	} {
		t.Run(tc.product, func(t *testing.T) {
			_, f := hermeticFormFor(t, "atlassian")
			fieldByLabel(t, f, "product").SetValue(tc.product)
			resolveSources(&f)

			visible := map[string]bool{}
			for _, i := range f.VisibleIndices() {
				visible[f.Fields()[i].Label] = true
			}
			for _, label := range tc.want {
				if !visible[label] {
					t.Errorf("%q is hidden for %s", label, tc.product)
				}
			}
			for _, label := range tc.gone {
				if visible[label] {
					t.Errorf("%q is offered for %s, which does not use it", label, tc.product)
				}
			}

			if tc.product == "scim" {
				fieldByLabel(t, f, "directory id").SetValue("dir-1")
			}
			got := f.Args()
			if len(got) != len(tc.args) {
				t.Fatalf("args = %v, want %v", got, tc.args)
			}
			for i := range got {
				if got[i] != tc.args[i] {
					t.Fatalf("args = %v, want %v", got, tc.args)
				}
			}
		})
	}
}

// scim is the one product whose second argument the connector insists on --
// "scim requires a directory id" -- and the only one where the field appears.
func TestAtlassianScimRequiresItsDirectory(t *testing.T) {
	_, f := hermeticFormFor(t, "atlassian")
	fieldByLabel(t, f, "product").SetValue("scim")
	resolveSources(&f)
	if err := f.Validate(); err == nil {
		t.Fatal("scim with no directory id validated")
	}
	fieldByLabel(t, f, "directory id").SetValue("dir-1")
	if err := f.Validate(); err != nil {
		t.Fatalf("scim with a directory id did not validate: %v", err)
	}
}

// databricks declares MaxArgs=0, so the plane selector is a UI distinction and
// nothing more: a stray word on the command line would be rejected by cnspec
// before the connector ever saw it.
func TestDatabricksPlaneSelectorPutsNoWordOnTheCommandLine(t *testing.T) {
	c := connectorFor(t, "databricks")
	if c.MaxArgs != 0 {
		t.Fatalf("databricks now declares MaxArgs=%d; the selector may need to emit a word", c.MaxArgs)
	}

	for _, plane := range []string{"account console", "workspace"} {
		_, f := hermeticFormFor(t, "databricks")
		fieldByLabel(t, f, "connect to").SetValue(plane)
		fieldByLabel(t, f, "host").SetValue("example.cloud.databricks.com")
		resolveSources(&f)
		for _, a := range f.Args() {
			if a == plane {
				t.Errorf("%q reached the command line: %v", plane, f.Args())
			}
		}
	}
}

// The account id routes to the account console and a personal access token is
// documented as workspace-only, so neither is offered where it does nothing.
func TestDatabricksAsksWhatThePlaneNeeds(t *testing.T) {
	for _, tc := range []struct{ plane, want, gone string }{
		{plane: "account console", want: "account id", gone: "personal access token"},
		{plane: "workspace", want: "personal access token", gone: "account id"},
	} {
		t.Run(tc.plane, func(t *testing.T) {
			_, f := hermeticFormFor(t, "databricks")
			fieldByLabel(t, f, "connect to").SetValue(tc.plane)
			resolveSources(&f)

			visible := map[string]bool{}
			for _, i := range f.VisibleIndices() {
				visible[f.Fields()[i].Label] = true
			}
			if !visible[tc.want] {
				t.Errorf("%q is hidden for a %s connection", tc.want, tc.plane)
			}
			if visible[tc.gone] {
				t.Errorf("%q is offered for a %s connection", tc.gone, tc.plane)
			}
			// OAuth M2M works on both planes, so it is never gated.
			if !visible["OAuth client secret"] {
				t.Errorf("OAuth is hidden for a %s connection", tc.plane)
			}
		})
	}
}

// The shared classifier reads --api-key as a secret and --app-key as ordinary
// text, because "app-key" ends in no strong secret word. A Datadog application
// key is a credential, so the spec says so for this one flag rather than
// widening a word list every other connector shares.
func TestDatadogApplicationKeyIsASecret(t *testing.T) {
	_, f := hermeticFormFor(t, "datadog")
	for _, flag := range []string{"api-key", "app-key"} {
		i := f.IndexOfFlag(flag)
		if i < 0 {
			t.Fatalf("no --%s field", flag)
		}
		if !f.Fields()[i].Secret {
			t.Errorf("--%s is not marked secret, so its value would reach argv", flag)
		}
		if got := f.Fields()[i].Section; got != sectionCredential {
			t.Errorf("--%s is in %s, want %s", flag, got, sectionCredential)
		}
	}

	// Both filled is the normal case for datadog rather than an impossible one:
	// the API key says which account, the application key says which user, and
	// the API refuses most endpoints without the pair. This was a refusal once,
	// because delivery resolved one route for the whole form, and then it was
	// two variables. It is one call now, and how many secrets it carries is not
	// a question the launcher has to answer.
	f.Fields()[f.IndexOfFlag("api-key")].SetValue(sentinel)
	f.Fields()[f.IndexOfFlag("app-key")].SetValue(sentinel + "-app")
	c := connectorFor(t, "datadog")

	p := withParser(t, &fakeParser{secretFlag: "api-key"})
	plan, err := (launchRequest{form: f}).plan(c, scanAction())
	if plan.cleanup != nil {
		defer plan.cleanup()
	}
	if err != nil {
		t.Fatalf("both keys were refused: %v", err)
	}
	if strings.Contains(strings.Join(plan.args, " "), sentinel) {
		t.Fatalf("a key reached argv: %v", plan.args)
	}
	for flag, want := range map[string]string{
		"api-key": sentinel,
		"app-key": sentinel + "-app",
	} {
		if got, _ := p.sentValue(flag); got != want {
			t.Errorf("--%s reached the provider as %q, want %q", flag, got, want)
		}
	}
}

// The site list suggests without constraining: a Datadog region newer than
// this binary still has to be typeable.
func TestDatadogSiteIsASuggestion(t *testing.T) {
	_, f := hermeticFormFor(t, "datadog")
	fd := fieldByLabel(t, f, "site")
	if fd.Kind != fieldChoice {
		t.Fatalf("site is a %v, want a picker", fd.Kind)
	}
	if fd.Strict {
		t.Error("the site list is strict, so a new Datadog region could not be typed")
	}
	if len(fd.Options) == 0 {
		t.Error("the site picker offers nothing")
	}
}

// gitlab's group and project are enumerable, but only through cnspec's own
// discovery, which needs the token first -- so they are live pickers rather
// than lists fetched while the screen is being drawn.
func TestGitLabTargetsUseDiscovery(t *testing.T) {
	_, f := hermeticFormFor(t, "gitlab")
	for _, tc := range []struct{ label, source string }{
		{"group", srcDiscoverGitLabGroups},
		{"project", srcDiscoverGitLabProjects},
	} {
		fd := fieldByLabel(t, f, tc.label)
		if fd.LiveSource != tc.source {
			t.Errorf("%s live source = %q, want %q", tc.label, fd.LiveSource, tc.source)
		}
		if fd.Kind != fieldChoice {
			t.Errorf("%s is a %v, want a picker", tc.label, fd.Kind)
		}
		if _, ok := sourceByID(tc.source); !ok {
			t.Errorf("%s names %q, which is not a registered source", tc.label, tc.source)
		}
	}

	// The ambient widgets stay whatever source_ambient.go made them: this file
	// curates the target around the credential, never over it.
	if got := fieldByLabel(t, f, "token").Kind; got != fieldPaste {
		t.Errorf("the token field is a %v, want the ambient paste box", got)
	}
}

// cloudflare declares one flag and it is the credential, so there is no target
// to curate -- only where the token belongs and what to call it.
func TestCloudflareIsCredentialOnly(t *testing.T) {
	_, f := hermeticFormFor(t, "cloudflare")
	token := fieldByLabel(t, f, "API token")
	if token.Flag != "token" || !token.Secret {
		t.Errorf("the API token row is %+v, want the secret --token flag", token)
	}
	if token.Kind != fieldPaste {
		t.Errorf("the token field is a %v, want the ambient paste box", token.Kind)
	}
}

// A value picker that names a source nobody registered leaves the field an
// empty box with no explanation, which is the failure the source registry
// exists to prevent. TestEverySourceNamedByASpecExists covers every spec at
// once; this one names the pickers this file attaches and the flag each belongs
// to, so a source retargeted at a different flag is visible here.
func TestSaaSPickersNameRegisteredSources(t *testing.T) {
	cases := []struct{ connector, flag, source string }{
		{"mongodbatlas", "project-id", srcDiscoverAtlasProjects},
		{"netlify", "account", discoverSourceID("netlify", "accounts")},
	}
	for _, tc := range cases {
		t.Run(tc.connector+" --"+tc.flag, func(t *testing.T) {
			s, ok := sourceByID(tc.source)
			if !ok {
				t.Fatalf("%q is not a registered source", tc.source)
			}
			// Both are cnspec's own discovery, which connects into every asset
			// it finds. That has to stay deferred: running it when the form
			// opens would spend a round trip on a field nobody looked at.
			if s.Cost != CostRemote {
				t.Errorf("%s runs at cost %v, want CostRemote", tc.source, s.Cost)
			}

			_, f := hermeticFormFor(t, tc.connector)
			i := f.IndexOfFlag(tc.flag)
			if i < 0 {
				t.Fatalf("--%s is not on the form; fields are %v", tc.flag, fieldLabels(f))
			}
			if f.Fields()[i].Source() != tc.source {
				t.Fatalf("--%s draws from %q, want %q", tc.flag, f.Fields()[i].Source(), tc.source)
			}
			// applySources only merges a live source into a base source's
			// values, so a picker attached through LiveSources alone would
			// never show anything. This one has to be the base.
			if f.Fields()[i].Kind != fieldChoice {
				t.Errorf("--%s is a %v, want a picker", tc.flag, f.Fields()[i].Kind)
			}
		})
	}
}

// vercel's team picker has to be reachable.
//
// applySources only fills a field whose `source` is set, so a field carrying
// nothing but a liveSource loads its values and never shows them -- it spins,
// answers, and the picker stays empty. That is why the discovery source is
// attached through Sources here, and this is the assertion that keeps someone
// from "fixing" it into LiveSources later.
func TestVercelTeamPickerIsAttachedSoItsValuesLand(t *testing.T) {
	f := newForm(connectorFor(t, "vercel"))
	i := f.IndexOfFlag("team")
	if i < 0 {
		t.Fatal("vercel has no --team field")
	}
	fd := f.Fields()[i]

	if fd.Source() != discoverSourceID("vercel", "teams") {
		t.Errorf("--team source = %q, want the registered vercel teams discovery", fd.Source())
	}
	if fd.LiveSource != "" && fd.Source() == "" {
		t.Error("--team carries only a live source, whose values applySources never applies")
	}
	if fd.Kind != fieldChoice {
		t.Errorf("--team kind = %v, want a picker", fd.Kind)
	}
	if !deferredSource(fd.Source()) {
		t.Error("the vercel teams picker is not deferred, so it would run for every form the user passes through")
	}
}

// snowflake's --token was the launcher's last standing refusal, and it is not
// one any more.
//
// The refusal was sound about the provider and wrong about the conclusion.
// snowflake declares --token and --password with ConfigEntry "-", which tells
// mql to read them from cobra alone and consult no variable -- confirmed
// against the shipped binary, `MONDOO_TOKEN=... MONDOO_PASSWORD=... cnspec shell
// snowflake --account ... --user ...` still answered "missing credentials for
// snowflake connection". Every word of that is about how a *flag value* gets
// into the process. It is not how the value gets in any more: it goes into
// req.Flags, which is the same place cobra would have put it.
func TestSnowflakeTokenTravels(t *testing.T) {
	c := connectorFor(t, "snowflake")
	f := newForm(c)
	i := f.IndexOfFlag("token")
	if i < 0 {
		t.Fatal("snowflake no longer declares --token")
	}
	f.Fields()[i].SetValue("<PLACEHOLDER-not-a-real-secret>")

	assertCredentialReachesTheProvider(t, c, f, "token", "<PLACEHOLDER-not-a-real-secret>")

	// --ask-pass is still declared and still the strongest route for the
	// password, and it is still a toggle the user can tick.
	if j := f.IndexOfFlag("ask-pass"); j < 0 {
		t.Error("snowflake no longer offers --ask-pass, so nothing can prompt for its password")
	}
}

// snowflake's account picker offers the account identifiers from
// connections.toml, which is the only thing in that file the connector can use:
// no flag and no variable takes a connection name. Attaching it anywhere but
// --account would put a connection name on the command line as an account.
func TestSnowflakeAccountPickerIsOnTheAccountFlag(t *testing.T) {
	f := newForm(connectorFor(t, "snowflake"))
	i := f.IndexOfFlag("account")
	if i < 0 {
		t.Fatal("snowflake has no --account field")
	}
	if got := f.Fields()[i].Source(); got != srcSnowflakeConnection {
		t.Errorf("--account source = %q, want %q", got, srcSnowflakeConnection)
	}
	for _, fd := range f.Fields() {
		if fd.Flag != "account" && fd.Source() == srcSnowflakeConnection {
			t.Errorf("the connections.toml picker is also on --%s, which does not take an account identifier", fd.Flag)
		}
	}
}

// tailscale's tailnet is a positional argument the usage string does not name,
// so without the spec it renders as "argument 1". It reaches the provider as
// req.Args[0] and needs no environment route of its own.
//
// tailscale is also the credential that could never have been written down as a
// variable name: it reads the same secret as the API token or as the OAuth
// client secret depending on whether --client-id is set.
func TestTailscaleNamesItsTailnet(t *testing.T) {
	f := newForm(connectorFor(t, "tailscale"))
	if len(f.Fields()) == 0 {
		t.Fatal("tailscale built an empty form")
	}
	lead := f.Fields()[0]
	if lead.Flag != "" {
		t.Fatalf("tailscale leads with --%s, not with its positional argument", lead.Flag)
	}
	if lead.Label != "tailnet" {
		t.Errorf("the positional is labelled %q, want %q", lead.Label, "tailnet")
	}
	if lead.Required {
		t.Error("the tailnet is optional -- the provider defaults to the token's own tailnet")
	}

	// It travels as an argument, so nothing should have given it a variable.
	if lead.Env != "" {
		t.Errorf("the tailnet carries env %q, but it reaches the provider as req.Args[0]", lead.Env)
	}
}

// iru takes no argument: the tenant is a flag and the token is a flag the
// provider marks as a password. Iru is Kandji renamed, and the launcher files
// it beside jamf because they are the same job -- which is why it is here and
// not under the "Other" heading a stale comment in forms_misc_test.go gave it.
func TestIruAsksForItsTenantAndKeepsTheTokenOffArgv(t *testing.T) {
	c, f := formFor(t, "iru")

	if pos := positionalFields(&f); len(pos) != 0 {
		t.Errorf("iru takes no argument but the form asks for %d: %v",
			len(pos), fieldLabels(f))
	}

	subdomain := fieldByLabel(t, f, "tenant subdomain")
	if subdomain.Section != sectionTarget {
		t.Errorf("the tenant sits in %q rather than TARGET", subdomain.Section)
	}
	if subdomain.Secret {
		t.Error("the tenant subdomain is not a secret and must stay on the command line")
	}

	token := fieldByLabel(t, f, "API token")
	if !token.Secret || token.Section != sectionCredential {
		t.Fatalf("the API token is secret=%v in %q", token.Secret, token.Section)
	}

	subdomain.SetValue("mondoo")
	token.SetValue(sentinel)

	p := assertCredentialReachesTheProvider(t, c, f, "token", sentinel)
	if got, ok := p.sentValue("subdomain"); !ok || got != "mondoo" {
		t.Errorf("the tenant reached the provider as %q (present=%v)", got, ok)
	}
}
