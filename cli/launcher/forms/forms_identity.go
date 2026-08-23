// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package forms

// Identity & Access: activedirectory, auth0, bitwarden, google-workspace,
// jumpcloud, keycloak, ms365 and okta.
//
// These are the systems that decide who can sign in, and where credentials
// live. The catalog separates them from SaaS because "review our identity
// provider" and "review the tools we subscribe to" are different jobs, and
// ms365 and google-workspace are here rather than under SaaS despite being
// productivity suites: what a scan of either actually inspects is the tenant's
// directory, its sign-in policy and its admin roles.
//
// None of the eight has a positional target, so the whole screen is flags, and
// the generic layer renders that as one undifferentiated list in declaration
// order. The work below is saying which two or three of those flags name the
// thing being scanned, which ones are the credential, and -- for the several
// that authenticate two different ways -- keeping the alternatives next to each
// other so it is visible that they are alternatives.
//
// Every flag named below was read out of the provider's own declaration rather
// than inferred from a flag description: applySpec drops a flag the connector
// does not declare, so an invented name produces a field that silently never
// appears. Four of the eight -- auth0, bitwarden, jumpcloud and keycloak --
// reach the catalog only through the compiled-in static list and so have no
// snapshot entry to check against; the package doc says what that costs and
// how it closes.

func init() {
	// activedirectory is the one host-shaped connector on this shelf: the
	// target is a domain controller and everything else describes how to bind
	// to it. The order here is the order the provider needs them in -- the DC
	// is the only flag ParseCLI refuses to proceed without.
	//
	// --backend is hidden. It declares two values and the connection rejects
	// one of them outright ("backend 'rsat' is not yet implemented; use
	// --backend=ldap (the default)"), so the flag can currently only do
	// nothing or fail.
	registerSpec("activedirectory", FormSpec{
		Target:     []string{"dc", "domain", "base-dn", "port"},
		Credential: []string{"user", "password", "kerberos", "keytab", "ccache", "krb5conf"},
		Hide:       []string{"backend"},
		Labels: map[string]string{
			"dc":         "domain controller",
			"base-dn":    "base DN",
			"user":       "bind user",
			"kerberos":   "use Kerberos (GSSAPI)",
			"keytab":     "Kerberos keytab",
			"ccache":     "Kerberos credential cache",
			"krb5conf":   "krb5.conf",
			"ldaps":      "LDAPS on port 636 (default)",
			"starttls":   "StartTLS on port 389",
			"plain-ldap": "plaintext LDAP on port 389",
			"insecure":   "skip TLS verification",
		},
	})

	// auth0 authenticates a machine-to-machine application against one tenant.
	// The tenant domain is the target; the client id and secret are one
	// credential in two fields and belong together.
	registerSpec("auth0", FormSpec{
		Target:     []string{"domain"},
		Credential: []string{"client-id", "client-secret"},
		Labels: map[string]string{
			"domain":        "tenant domain",
			"client-id":     "M2M client ID",
			"client-secret": "M2M client secret",
		},
	})

	// bitwarden reads organization governance through an organization API key.
	// The two URLs are what point the scan at a self-hosted deployment instead
	// of the hosted one, which makes them the target rather than options.
	registerSpec("bitwarden", FormSpec{
		Target:     []string{"api-url", "identity-url"},
		Credential: []string{"client-id", "client-secret"},
		Labels: map[string]string{
			"api-url":       "API base URL (self-hosted only)",
			"identity-url":  "identity URL (self-hosted only)",
			"client-id":     "organization client ID",
			"client-secret": "organization client secret",
		},
	})

	// google-workspace carries no secret at all, which is worth stating
	// because the screen looks like it should. --credentials-path names a
	// service account JSON file rather than holding one, so the classifier
	// reads it as a reference and it travels on the command line, which is
	// exactly what the provider expects: getGoogleCreds opens the path.
	//
	// All three flags are mandatory -- ParseCLI logs and fails for each one
	// missing -- so none of them is hidden and the two that identify the
	// tenant lead.
	registerSpec("google-workspace", FormSpec{
		Target:     []string{"customer-id", "impersonated-user-email"},
		Credential: []string{"credentials-path"},
		Labels: map[string]string{
			"customer-id":             "customer id",
			"impersonated-user-email": "impersonate user",
			"credentials-path":        "service account key",
		},
	})

	// jumpcloud rejects a missing key in ParseCLI rather than on connect, so
	// the environment route has to reach the child before it parses -- which it
	// does, and which is what the positive control confirmed. --org-id only
	// applies to multi-tenant keys, and the label says so rather than leaving
	// an unexplained field in TARGET.
	registerSpec("jumpcloud", FormSpec{
		Target:     []string{"org-id"},
		Credential: []string{"api-key"},
		Labels: map[string]string{
			"org-id":  "organization ID (multi-tenant keys only)",
			"api-key": "API key",
		},
	})

	// keycloak has two alternative authentication methods -- an admin user with
	// a password, or a service account on a confidential client -- and the
	// provider says so itself when neither is complete. Both secrets therefore
	// sit in CREDENTIAL, because either one alone is a valid form. Only one is
	// ever filled in practice, and keycloak is picky about which: it reads an
	// untagged password as the *client* secret and needs cred.User to mean the
	// admin one. Its own ParseCLI is what tags them.
	//
	// --url is the server and --realm narrows the scan, so those are TARGET.
	// --auth-realm sits with the credential because it says where the token is
	// requested from, not what gets scanned.
	registerSpec("keycloak", FormSpec{
		Target:     []string{"url", "realm"},
		Credential: []string{"username", "password", "client-id", "client-secret", "auth-realm"},
		Labels: map[string]string{
			"url":           "server URL",
			"realm":         "realm (leave empty for every realm)",
			"username":      "admin user",
			"client-id":     "service account client",
			"client-secret": "service account secret",
			"auth-realm":    "realm the token is requested from",
			"ca-cert":       "CA certificate",
		},
	})

	// ms365 identifies its target by tenant id -- the connection refuses
	// without one and builds its platform id out of it -- so tenant id leads
	// TARGET rather than sitting with the service principal it also belongs
	// to. The organization and the SharePoint site narrow what is scanned
	// inside that tenant.
	//
	// The three sign-in shapes are: a certificate, a client secret, or none of
	// the above and let the Azure identity chain find one, which --auth-method
	// names a single link of. The keyless shape is the one the connector's own
	// help recommends, and it is also the only one the launcher can run end to
	// end -- see the note below the spec.
	//
	// MaxArgs=5 is vestigial in the same way mondoo's is: ParseCLI reads
	// req.GetFlags() and never touches the arguments. Declaring no Positional
	// suppresses the box.
	registerSpec("ms365", FormSpec{
		Target: []string{"tenant-id", "organization", "sharepoint-url"},
		// The two secrets are named beside the halves they pair with, now that
		// they have somewhere to travel -- see the note below the spec.
		Credential: []string{
			"auth-method", "client-id", "client-secret",
			"certificate-path", "certificate-secret",
		},
		Choices: map[string][]string{
			"auth-method": {"cli", "env", "workload-identity", "managed-identity"},
		},
		Labels: map[string]string{
			"tenant-id":          "tenant id",
			"organization":       "organization (optional)",
			"sharepoint-url":     "SharePoint site (optional)",
			"auth-method":        "sign-in method",
			"client-id":          "application (client) id",
			"client-secret":      "application secret",
			"certificate-path":   "certificate (PKCS #12 or PEM)",
			"certificate-secret": "certificate passphrase",
		},
	})

	// okta authenticates two ways: an API token, which carries the full
	// privileges of the admin who minted it, or a service app with a private
	// key JWT, which holds only the scopes it was granted. The token is
	// ambient -- source_ambient.go owns its readout, its paste box and its
	// OKTA_API_TOKEN route, and composes --organization out of OKTA_ORG_NAME
	// and OKTA_BASE_URL around the provider's "." bug -- so all that is left
	// here is the ordering and the service app beneath it.
	//
	// The four service-app flags stay together and in the order the
	// connector's own example gives them, because three of them are useless
	// without the fourth and a form that scatters them across two sections
	// makes that a discovery rather than a layout.
	registerSpec("okta", FormSpec{
		Target: []string{"organization"},
		Credential: []string{
			"token",
			"client-id", "private-key", "private-key-id", "scopes",
		},
		// --private-key holds either the PEM itself or a path to it, and the
		// shared classifier reads it as a path: isSecretReference matches
		// "path to" in its description and returns before the private-key word
		// is ever consulted. That is right for github's --app-private-key,
		// which really is only a path, and wrong here, where pasting a PEM
		// would put a private key into `ps auxww`. Correcting it in the shared
		// word lists would re-read every other connector's flags; saying it
		// here corrects okta and nothing else.
		Secret: []string{"private-key"},
		// --organization keeps the flag's own name. It is the one field on this
		// form that the launcher fills in for the user, and the tests that
		// prove it does -- the ones guarding the "." bug in source_ambient.go
		// -- find it by label. Its description already reads "The domain of
		// the Okta organization to scan", so a nicer label would buy very
		// little and take those tests down with it.
		Labels: map[string]string{
			"token":          "API token",
			"client-id":      "service app client id",
			"private-key":    "service app private key",
			"private-key-id": "service app key id",
			"scopes":         "service app scopes",
		},
	})
}
