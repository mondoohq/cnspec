// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	"strings"
	"testing"

	tuiform "go.mondoo.com/cnspec/cli/tui/form"
)

// The curated Identity & Access forms: the systems that decide who can sign in,
// and where credentials live.
//
// These eight were scattered across the three files named for whoever wrote
// them -- activedirectory and google-workspace in forms_saas_a_test.go, ms365
// and okta in _b, auth0, bitwarden, jumpcloud and keycloak in _c -- because
// each contributor curated a slice of the catalog rather than a category. The
// launcher has shown them under Identity & Access all along, and ms365 and
// google-workspace belong here rather than under SaaS despite being
// productivity suites: what a scan of either inspects is the tenant's
// directory, its sign-in policy and its admin roles.
//
// Every test below drives the real form -- newForm over the connector rebuilt
// from internal/connectors/connectors.json -- rather than inspecting the
// FormSpec literal, because a spec that names a flag the connector does not
// declare is silently dropped by applySpec. Reading the spec back would agree
// with itself; building the form is what proves the flag survived.

// identityConnectors is every connector the launcher files under Identity &
// Access. filedHere is what proves that claim rather than restating it; see
// forms_filing_test.go.
var identityConnectors = filedHere(
	"activedirectory", "auth0", "bitwarden", "google-workspace",
	"jumpcloud", "keycloak", "ms365", "okta",
)

// Every connector this file claims has a spec, and exactly one.
//
// registerSpec is first-wins: a sibling file that also registered one of these
// would take it, and the only visible sign would be an entry in duplicateSpecs.
// The other half is a connector named here that never got registered at all
// because an init was removed, which leaves it on the generic flag-derived
// screen -- a screen that looks like a form and says nothing about being
// uncurated.
func TestIdentitySpecsAreRegisteredExactlyOnce(t *testing.T) {
	for _, name := range identityConnectors {
		if _, ok := formSpecs[name]; !ok {
			t.Errorf("%s has no registered spec", name)
		}
		if containsString(duplicateSpecs, name) {
			t.Errorf("%s was registered twice, so two files claim it", name)
		}
	}
}

// identityTargetLeads is what each form's TARGET section leads with, by the
// label the spec gives it.
//
// None of these is obvious from the flag list: keycloak leads with the server
// rather than the realm, bitwarden with a URL that only self-hosted
// installations need, jumpcloud with an organization id that only multi-tenant
// keys use. Every one is a decision about what the connector is addressed by,
// which is what the first section of the pane tells the reader.
var identityTargetLeads = map[string]string{
	"activedirectory":  "domain controller",
	"auth0":            "tenant domain",
	"bitwarden":        "API base URL (self-hosted only)",
	"google-workspace": "customer id",
	"jumpcloud":        "organization ID (multi-tenant keys only)",
	"keycloak":         "server URL",
	"ms365":            "tenant id",
	"okta":             "organization",
}

// The field the connector cannot run without has to be on the screen, in
// TARGET, and visible before anything else is chosen. A curated form that
// buries it -- or gates it behind a selector value that is not the default --
// is worse than the generic screen it replaced.
func TestIdentityTargetsLeadTheirForms(t *testing.T) {
	for _, name := range identityConnectors {
		label, declared := identityTargetLeads[name]
		if !declared {
			t.Errorf("%s is curated here but identityTargetLeads does not say what "+
				"its form is addressed by; add the label, or \"\" if it has no target", name)
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

// The target flags that are not the lead. ms365 can be scoped three ways and
// all three say what is being scanned rather than who is scanning it -- a
// target left in OPTIONS is below the credential, which reverses the order
// these connectors are actually used in.
func TestIdentitySecondaryTargetsStayInTheTargetSection(t *testing.T) {
	targets := map[string][]string{
		"ms365": {"tenant-id", "organization", "sharepoint-url"},
		"okta":  {"organization"},
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

// Every credential a curated form offers reaches the provider, and the target
// it was filled in beside survives being separated from it.
//
// Filled "on its own" is deliberate. okta declares two alternative credentials
// at once, only one of which is ever used, so the shapes are listed as the user
// would fill them rather than all at once -- not because the route could only
// carry one, which it no longer is, but because several providers pick a
// different authentication path when they see the second and legitimately drop
// the first.
//
// The route each one used to be pinned to is gone with the registry that held
// it. okta's key is the case that made the point: it wants a private_key
// credential rather than a password, and pinning a variable name never checked
// that.
func TestIdentityCredentialsReachTheProvider(t *testing.T) {
	cases := []struct {
		name      string
		connector string
		// fill is flag -> value; flag gets the sentinel.
		flag string
		fill map[string]string
	}{
		{
			// Registered in cli/launcher/forms/forms_identity.go, and checked
			// here anyway: the okta spec puts --token in CREDENTIAL, and a
			// section entry whose flag has no route is what produces a form
			// that refuses.
			name: "okta API token", connector: "okta", flag: "token",
			fill: map[string]string{"organization": "dev-123.okta.com"},
		},
		{
			name: "okta service app", connector: "okta", flag: "private-key",
			fill: map[string]string{
				"organization":   "dev-123.okta.com",
				"client-id":      "0oa1",
				"private-key-id": "kid-1",
				"scopes":         "okta.users.read",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, f := hermeticFormFor(t, tc.connector)
			for flag, value := range tc.fill {
				setFlag(t, &f, flag, value)
			}
			fd := setFlag(t, &f, tc.flag, sentinel)
			if !fd.Secret {
				t.Fatalf("--%s carries a credential but is not classified as a secret", tc.flag)
			}
			if fd.Section != sectionCredential {
				t.Errorf("--%s is in %s, want %s", tc.flag, fd.Section, sectionCredential)
			}
			if err := f.Validate(); err != nil {
				t.Fatalf("the filled form does not validate: %v", err)
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

// The blanket rule, over every Identity connector: fill every box and check
// that nothing a user typed into a credential comes back out on a command line
// every account on the machine can read.
//
// This was two sweeps in two files that had drifted apart. One filled a picker
// with its first option and the other with the word "placeholder"; one resolved
// the value sources afterwards and the other did not; one checked f.Args()
// before planning and the other only checked the plan; one failed a connector
// whose form offered no secret at all and the other returned quietly. Every one
// of those differences was an accident of which file the author had open, and
// the strict side of each is the one that is kept here.
func TestNoIdentitySecretReachesArgv(t *testing.T) {
	// google-workspace is the one form here with no credential to fill:
	// --credentials-path names a file rather than holding a secret, so the
	// whole form travels on the command line the way the provider expects.
	// Every other connector here must offer one, because a curated CREDENTIAL
	// section that built no field is a screen that cannot be completed.
	noCredential := map[string]bool{"google-workspace": true}

	for _, name := range identityConnectors {
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
func TestIdentityLabelsAreUnique(t *testing.T) {
	for _, name := range identityConnectors {
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
// the ones that are not secrets: a client id, a user name and the realm a token
// is requested from are all read as ordinary flags and would sit in the
// collapsed OPTIONS row, away from the secret they only mean anything beside.
func TestIdentityPairsIdentifiersWithTheirSecrets(t *testing.T) {
	want := map[string][]string{
		"auth0":     {"client-id"},
		"bitwarden": {"client-id"},
		"keycloak":  {"username", "client-id", "auth-realm"},
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

// keycloak declares two secrets for two alternative authentication methods,
// reads an untagged password as the *client* secret, and needs cred.User to
// mean the admin one. A user fills one; which one they filled is keycloak's to
// work out, and it does, in its own ParseCLI. Both have to be able to launch.
func TestKeycloakCarriesEitherAuthenticationMethod(t *testing.T) {
	c := connectorFor(t, "keycloak")
	for _, flag := range []string{"password", "client-secret"} {
		t.Run(flag, func(t *testing.T) {
			f := newForm(c)
			value := "must-never-reach-argv-" + flag
			setFlag(t, &f, flag, value)
			assertCredentialReachesTheProvider(t, c, f, flag, value)
			if args := strings.Join(f.Args(), " "); strings.Contains(args, value) {
				t.Errorf("--%s reached argv: %s", flag, args)
			}
		})
	}
}

// ms365's two secrets reach the provider, and the history of this one test is
// the argument for the whole change.
//
// It was first a refusal, reasoned from the provider: "its ParseCLI reads
// req.GetFlags() and calls os.Getenv for nothing, so no variable carries
// --client-secret or --certificate-secret to it". Every clause was true and the
// conclusion did not follow. Then it was MONDOO_CLIENT_SECRET and
// MONDOO_CERTIFICATE_SECRET, names derived from the flags rather than known to
// ms365, working only because mql fills a flag with no config mapping from one
// before ParseCLI runs.
//
// Reading req.GetFlags() was never the obstacle. It is the destination: the
// launcher fills it directly now, over gRPC, and ms365's own ParseCLI builds
// the pkcs12 credential for the certificate passphrase that no inventory this
// package could write would have produced.
func TestMs365SecretsReachTheProvider(t *testing.T) {
	for _, flag := range []string{"client-secret", "certificate-secret"} {
		t.Run(flag, func(t *testing.T) {
			c, f := hermeticFormFor(t, "ms365")
			setFlag(t, &f, "tenant-id", "3d8c")
			setFlag(t, &f, flag, sentinel)

			assertCredentialReachesTheProvider(t, c, f, flag, sentinel)
		})
	}
}

// The keyless sign-in shapes are what ms365's own help recommends, and they are
// what the launcher can run: no secret in the form means the plain command
// line, with the certificate path -- which names a file rather than holding a
// credential -- on it, because that is how the provider expects to receive it.
func TestMs365KeylessSignInLaunchesPlainly(t *testing.T) {
	c, f := hermeticFormFor(t, "ms365")
	setFlag(t, &f, "tenant-id", "3d8c")
	setFlag(t, &f, "client-id", "9b21")
	setFlag(t, &f, "certificate-path", "/keys/ms365.pem")
	method := setFlag(t, &f, "auth-method", "cli")
	if method.Kind != fieldChoice {
		t.Errorf("--auth-method is a %v, want a picker over the declared methods", method.Kind)
	}
	if !containsString(method.Options, "workload-identity") {
		t.Errorf("--auth-method options = %v, want the four declared methods", method.Options)
	}

	if got := deliveryFor(f); got != deliverPlain {
		t.Fatalf("delivery = %v, want the plain command line for a form with no secret", got)
	}

	r := launchRequest{form: f}
	plan, err := r.plan(c, scanAction())
	if err != nil {
		t.Fatal(err)
	}
	if plan.cleanup != nil {
		defer plan.cleanup()
	}
	line := strings.Join(plan.args, " ")
	for _, want := range []string{"--tenant-id 3d8c", "--certificate-path /keys/ms365.pem", "--auth-method cli"} {
		if !strings.Contains(line, want) {
			t.Errorf("launch = %q, missing %q", line, want)
		}
	}
}

// okta --private-key is the one flag in this file whose classification the
// shared word lists get wrong, and this is the test that keeps the correction
// honest: it asserts both that the classifier alone says "not a secret" -- so
// the override is doing real work rather than restating the default -- and that
// the form ends up treating it as one.
//
// The classifier is right about the general case. isSecretReference matches
// "path to" in a description, which is exactly what makes github's
// --app-private-key safe on the command line. okta's takes either the PEM
// itself or a path to it, so the same description covers a value that must
// never be an argument.
func TestOktaPrivateKeyIsCorrectedToASecret(t *testing.T) {
	c, f := hermeticFormFor(t, "okta")

	var declared bool
	for _, fl := range c.Flags {
		if fl.Long != "private-key" {
			continue
		}
		declared = true
		if tuiform.IsSecretFlag(fl) {
			t.Error("the shared classifier now marks okta --private-key itself; " +
				"the FormSpec.Secret override for okta is redundant and should go")
		}
	}
	if !declared {
		t.Fatal("okta no longer declares --private-key")
	}

	i := f.IndexOfFlag("private-key")
	if i < 0 {
		t.Fatalf("--private-key is not on the form; fields are %v", fieldLabels(f))
	}
	if !f.Fields()[i].Secret {
		t.Fatal("okta --private-key can hold a PEM and must never reach argv")
	}

	// A secret is masked on screen as well as kept off the command line: the
	// form is drawn in a terminal somebody else can be looking at.
	f.Fields()[i].SetValue(sentinel)
	if strings.Contains(f.Fields()[i].Display(), sentinel) {
		t.Fatalf("the private key is rendered in the clear: %q", f.Fields()[i].Display())
	}
}

// activedirectory's --password reaches the provider.
//
// Its history is the whole argument in one connector. It was refused, because
// its ParseCLI reads no variable for --password and there was no name to
// register. Then it travelled in MONDOO_PASSWORD, a name derived from the flag
// that works only because mql fills any flag with no config mapping from one --
// confirmed against the shipped binary at the time, which is exactly the kind of
// evidence that decays. Neither was a fact about activedirectory. The value now
// goes into req.Flags, which is where that provider reads it from and always
// did.
func TestActiveDirectoryPasswordReachesTheProvider(t *testing.T) {
	c, f := hermeticFormFor(t, "activedirectory")
	fieldByLabel(t, f, "domain controller").SetValue("dc1.example.com")
	fieldByLabel(t, f, "bind user").SetValue("svc@example.com")
	fieldByLabel(t, f, "password").SetValue(sentinel)

	assertCredentialReachesTheProvider(t, c, f, "password", sentinel)
}

// The Kerberos paths need no secret at all -- --keytab and --ccache name files
// -- so they launch, and they are what makes the activedirectory form useful
// today.
func TestActiveDirectoryKerberosLaunchesOnTheCommandLine(t *testing.T) {
	_, f := hermeticFormFor(t, "activedirectory")
	fieldByLabel(t, f, "domain controller").SetValue("dc1.example.com")
	fieldByLabel(t, f, "bind user").SetValue("svc@EXAMPLE.COM")
	fieldByLabel(t, f, "use Kerberos (GSSAPI)").SetOn(true)
	fieldByLabel(t, f, "Kerberos keytab").SetValue("/etc/krb5.keytab")

	if got := deliveryFor(f); got != deliverPlain {
		t.Fatalf("delivery = %v, want the plain command line", got)
	}
	args := strings.Join(f.Args(), " ")
	for _, want := range []string{"--dc dc1.example.com", "--kerberos", "--keytab /etc/krb5.keytab"} {
		if !strings.Contains(args, want) {
			t.Errorf("args %q missing %q", args, want)
		}
	}
	// --backend is hidden: it declares two values and the connection rejects
	// one of them outright, so it can only do nothing or fail.
	if strings.Contains(args, "--backend") {
		t.Errorf("--backend reached the command line: %q", args)
	}
}

// google-workspace carries no secret at all: --credentials-path names a file,
// so the whole form travels on the command line the way the provider expects.
func TestGoogleWorkspaceLaunchesOnThePlainCommandLine(t *testing.T) {
	_, f := hermeticFormFor(t, "google-workspace")
	resolveSources(&f)
	if got := deliveryFor(f); got != deliverPlain {
		t.Fatalf("delivery = %v, want the plain command line", got)
	}
}
