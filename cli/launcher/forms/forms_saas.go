// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package forms

// SaaS: the accounts an organisation holds with someone else -- atlassian,
// cloudflare, databricks, datadog, dropbox, github, gitlab, grafana, iru, jamf,
// mondoo, mongodbatlas, netlify, nextdns, slack, snowflake, tailscale, vercel
// and zoom.
//
// The systems that decide who can sign in are next door in forms_identity.go.
// What is left here is a wide shelf with two recurring difficulties, and the
// specs below divide along them rather than along who wrote them.
//
// For some the difficulty is the *target*: what is being scanned, and which
// questions that answer makes relevant. atlassian is a sub-command shape the
// usage string only hints at, github and gitlab are an org or a repo or a user,
// databricks is an account plane or a workspace plane, and each of those
// choices changes which credential flags are even applicable. Showing all of
// them at once was the old screen's worst offence: most are always wrong.
//
// For the rest the difficulty is the *credential*. Most have no positional
// target at all, so the whole screen is flags, and the generic layer renders
// that as one undifferentiated list in declaration order. The work is saying
// which two or three of those flags name the thing being scanned, which ones
// are the credential, and -- for the ones that authenticate two different ways
// -- keeping the alternatives next to each other so it is visible that they are
// alternatives.
//
// Every flag named below was read out of the provider's own declaration --
// internal/connectors/connectors.json, the recorded copy of what the installed
// providers declare -- rather than inferred from a flag description: applySpec
// drops a flag the connector does not declare, so an invented name produces a
// field that silently never appears. dropbox and zoom reach the catalog only
// through the compiled-in static list and so have no snapshot entry to check
// against; the package doc says what that costs and how it closes.

// datadogSites is a display list, not a validation list: the site flag stays
// free text so a Datadog region newer than this binary still works.
var datadogSites = []string{
	"datadoghq.com",
	"us3.datadoghq.com",
	"us5.datadoghq.com",
	"datadoghq.eu",
	"ap1.datadoghq.com",
	"ddog-gov.com",
}

func init() {
	// atlassian is a sub-command shape the usage string only hints at. Its
	// ParseCLI refuses outright without a first argument -- "use `atlassian
	// jira`, `atlassian admin`, `atlassian confluence`, or `atlassian scim
	// {directoryID}`" -- and refuses again if `scim` arrives without a
	// directory id, while the metadata declares MinArgs=0 and MaxArgs=2 and
	// says nothing about any of it. So the product is a required picker whose
	// four options are the four words the connector accepts, emitted verbatim,
	// and the directory id only appears for the one that takes it.
	//
	// The credential differs per product as well, and each is read by a
	// different connection constructor: jira and confluence take a site, an
	// account email and a user API token; admin takes an admin token and
	// nothing else; scim takes a SCIM token and the directory id. Showing all
	// five credential flags at once was the old screen's worst offence here --
	// four of them are always wrong.
	registerSpec("atlassian", FormSpec{
		Positional: []PositionalSpec{
			{
				Label: "product", Desc: "which Atlassian product to scan",
				Required: true,
				Options:  []string{"jira", "confluence", "admin", "scim"},
			},
			{
				Label: "directory id", Desc: "the SCIM directory to scan",
				Required: true, ShowIf: []string{"scim"},
			},
		},
		Target:     []string{"host"},
		Credential: []string{"user", "user-token", "admin-token", "scim-token"},
		Labels: map[string]string{
			"host":        "site URL",
			"user":        "account email",
			"user-token":  "API token",
			"admin-token": "admin API token",
			"scim-token":  "SCIM API token",
		},
		// Read off the four connection constructors, not guessed from the flag
		// descriptions: jira/connection.go and confluence/connection.go read
		// host, user and user-token; admin/connection.go reads admin-token
		// alone; scim/connection.go reads scim-token and the directory id.
		ShowFlagsIf: map[string][]string{
			"host":        {"jira", "confluence"},
			"user":        {"jira", "confluence"},
			"user-token":  {"jira", "confluence"},
			"admin-token": {"admin"},
			"scim-token":  {"scim"},
		},
	})

	// cloudflare's entire form is its credential: one --token flag and the
	// discovery list. The ambient widgets and the CLOUDFLARE_TOKEN route are
	// already declared in source_ambient.go, so all that is left is to say
	// where the flag belongs and what to call it.
	registerSpec("cloudflare", FormSpec{
		Credential: []string{"token"},
		Labels:     map[string]string{"token": "API token"},
	})

	// databricks has two connection planes and one flag vocabulary spanning
	// both, which is why the flat screen was unreadable: --account-id is
	// meaningless for a workspace and --token is meaningless for the account
	// console, and nothing on the screen said so.
	//
	// The plane is a UI distinction rather than a sub-command -- the connector
	// declares MaxArgs=0 and infers the plane from whether an account id
	// arrived -- so the selector emits nothing at all, the way the container
	// kind selector does. It only steers which questions are asked.
	registerSpec("databricks", FormSpec{
		Positional: []PositionalSpec{{
			Label: "connect to", Desc: "the account console, or a single workspace",
			Required: true,
			Options:  []string{"account console", "workspace"},
			// Neither word is one cnspec would understand: the connector takes
			// no positional argument at all.
			Emit: map[string]string{"account console": "", "workspace": ""},
		}},
		Target:     []string{"host", "account-id"},
		Credential: []string{"token", "client-id", "client-secret"},
		Labels: map[string]string{
			"host":          "host",
			"account-id":    "account id",
			"token":         "personal access token",
			"client-id":     "OAuth client id",
			"client-secret": "OAuth client secret",
		},
		// An account id routes to the account console and a personal access
		// token is workspace-only ("Personal access token for a direct
		// workspace connect"); OAuth M2M works on both planes, so it is not
		// gated. The host is asked for either way -- it is optional for the
		// account console, which defaults to accounts.cloud.databricks.com,
		// and required for a workspace.
		ShowFlagsIf: map[string][]string{
			"account-id": {"account console"},
			"token":      {"workspace"},
		},
	})

	// datadog needs two credentials at once, and the shared classifier only
	// catches one of them: --api-key ends in a strong secret word and
	// --app-key does not, so the application key would have gone on the
	// command line where `ps auxww` reads it. Saying so here rather than
	// widening the word list is the point of FormSpec.Secret -- "app-key"
	// added to the shared list would re-classify every other connector's
	// key-shaped flag.
	//
	// Filling both boxes used to be refused: the launcher could name one
	// environment variable for one secret, and a form holding two fell through
	// to an inventory shape datadog does not read. Both travel now, because
	// datadog's own ParseCLI is what decides where they land -- and if it
	// turns out to keep only one of them, the launch says which one it dropped
	// rather than half-delivering.
	registerSpec("datadog", FormSpec{
		Target:     []string{"site"},
		Credential: []string{"api-key", "app-key"},
		Secret:     []string{"app-key"},
		Choices:    map[string][]string{"site": datadogSites},
		Labels: map[string]string{
			"api-key": "API key",
			"app-key": "application key",
			"site":    "site",
		},
	})

	// dropbox is a single team token and nothing else. The classifier already
	// puts it in CREDENTIAL, so the spec only names it properly -- the flag is
	// --token but the thing being asked for is a team-scoped token, and a
	// per-user one fails in a way the label can prevent.
	registerSpec("dropbox", FormSpec{
		Credential: []string{"token"},
		Labels:     map[string]string{"token": "team access token"},
	})

	registerSpec("github", FormSpec{
		Positional: []PositionalSpec{
			{
				Label: "kind", Desc: "what to scan", Required: true,
				Options: []string{"org", "repo", "user"},
			},
			{Label: "name", Desc: "ORG, OWNER/REPO, or USER", Required: true},
		},
		Target:     []string{"enterprise-url"},
		Credential: []string{"token", "app-id", "app-installation-id", "app-private-key", "app-private-key-content"},
		Labels:     map[string]string{"token": "personal access token"},
	})

	// gitlab's group and project are the whole target, and both can be
	// enumerated -- but only through cnspec's own discovery, which needs the
	// token first. So they are live pickers: nothing runs until the field is
	// opened, and by then the credential above it is filled in.
	//
	// The token itself is left alone. gitlab is an ambient connector, so the
	// readout and the paste box come from source_ambient.go and its
	// GITLAB_TOKEN route is registered there; curating it again here would put
	// two claims about one credential in two files.
	registerSpec("gitlab", FormSpec{
		Target: []string{"group", "project", "url"},
		LiveSources: map[string]string{
			"group":   srcDiscoverGitLabGroups,
			"project": srcDiscoverGitLabProjects,
		},
		// Only --url is relabelled. The token row belongs to the ambient
		// widgets: source_ambient_test.go reaches for gitlab's field by the
		// label "token", so renaming it here would break a claim made about a
		// credential this file does not own.
		Labels: map[string]string{"url": "self-managed URL"},
	})

	// grafana needs exactly two things and refuses without either: an instance
	// URL and a service account token.
	registerSpec("grafana", FormSpec{
		Target:     []string{"url"},
		Credential: []string{"token"},
		Labels: map[string]string{
			"url":   "instance URL",
			"token": "service account token",
		},
	})

	// iru takes no argument at all: the tenant is --subdomain and the
	// credential is --token, which the provider marks FlagOption_Password, so
	// the classifier needs no correction here. The empty Positional is what
	// suppresses the derived slot the way local's does.
	registerSpec("iru", FormSpec{
		Target:     []string{"subdomain"},
		Credential: []string{"token"},
		Labels: map[string]string{
			"subdomain": "tenant subdomain",
			"token":     "API token",
		},
	})

	// jamf authenticates one way: a Jamf Pro API client id and secret against
	// an instance domain. The domain is the target -- it names the tenant --
	// and the pair below it is the credential.
	//
	// --client-secret is the one flag in this file that the provider itself
	// marks FlagOption_Password, so the classifier needs no help with it.
	registerSpec("jamf", FormSpec{
		Target:     []string{"instance-domain"},
		Credential: []string{"client-id", "client-secret"},
		Labels: map[string]string{
			"instance-domain": "Jamf Pro URL",
			"client-id":       "API client id",
			"client-secret":   "API client secret",
		},
	})

	// mondoo takes nothing at all. Its ParseCLI builds an inventory config
	// from req.Connector and returns; it reads no flag and no argument, and
	// the credential comes from the workstation's Mondoo registration through
	// req.Upstream rather than from anything a form could ask for.
	//
	// The connector nevertheless declares MaxArgs=4, which the generic layer
	// faithfully renders as an "argument 1" box whose contents ParseCLI
	// discards without a word. The empty spec is what suppresses it -- the
	// same reason `local` has one.
	registerSpec("mondoo", FormSpec{})

	// mongodbatlas is the two-credentials connector. A programmatic API key is
	// a public/private pair; a service account is a client id and secret. Both
	// pairs are declared at once and only one is filled, so they are listed in
	// pair order, public half before secret half, to make the pairing visible.
	//
	// --public-key and --client-id are not secrets and stay on the command
	// line, which is where the provider expects them. They are in CREDENTIAL
	// anyway because a public key sitting under OPTIONS, three fields away
	// from the private half it is useless without, reads as an unrelated
	// setting.
	//
	// The org id and the project id are the two targets: with a project id the
	// connection scopes to that project, without one it connects to the
	// organization and discovers its projects. The project picker is that same
	// discovery, so it narrows to the organization already chosen.
	registerSpec("mongodbatlas", FormSpec{
		Target:  []string{"org-id", "project-id"},
		Sources: map[string]string{"project-id": srcDiscoverAtlasProjects},
		Credential: []string{
			"public-key", "private-key",
			"client-id", "client-secret",
		},
		Labels: map[string]string{
			"org-id":        "organization id",
			"project-id":    "project id (optional)",
			"public-key":    "API key — public",
			"private-key":   "API key — private",
			"client-id":     "service account client id",
			"client-secret": "service account client secret",
		},
	})

	// ms365 reads no environment variable of its own: its ParseCLI takes its
	// flags out of req.GetFlags() and calls os.Getenv for nothing whatsoever.
	// That is now the whole story, because req.GetFlags() is exactly what the
	// launcher fills -- over gRPC, straight into the provider, with no command
	// line and no variable in between.
	//
	// It is also the connector that shows what a hand-written inventory could
	// not do. ms365's connection reads conf.Credentials[0] and wants the
	// certificate as a pkcs12 credential with the passphrase in Password; the
	// launcher's own builder had no user here to attach a typed credential to
	// and would have written conn.Options["client-secret"], which ms365 reads
	// its options from a fixed list that has no such key. Its own ParseCLI
	// builds the right thing, and that is now what gets written.
	//
	// `--auth-method env` remains a different thing from either flag: it makes
	// the Azure identity chain read its own AZURE_* triple.

	// netlify is one token and an optional account to narrow to. The account
	// picker is cnspec's own discovery, which is why it costs a round trip and
	// only runs when the picker is opened.
	registerSpec("netlify", FormSpec{
		Target:     []string{"account"},
		Sources:    map[string]string{"account": discoverSourceID("netlify", "accounts")},
		Credential: []string{"token"},
		Labels: map[string]string{
			"account": "account (optional)",
			"token":   "personal access token",
		},
	})

	// nextdns is one API key and nothing else: no target flag, no positional,
	// and its two discovery targets (accounts, profiles) have no flag to
	// consume them, so there is nothing to attach a picker to. The whole form
	// is the credential, which is exactly what the screen should say.
	registerSpec("nextdns", FormSpec{
		Credential: []string{"api-key"},
		Labels:     map[string]string{"api-key": "API key"},
	})

	// slack's credential is ambient and already has its readout, its paste box
	// and its SLACK_TOKEN route; see source_ambient.go. All that is left is the
	// rest of the screen: the team is what is being scanned, so --team-id leads
	// TARGET.
	//
	// --token is deliberately left with its own name. Relabelling it costs
	// nothing on screen -- applyAmbient replaces the description with "paste a
	// token — it never reaches the command line" and the readout above it
	// already says what the field is -- and slack is the connector
	// TestPastedTokenBeatsTheEnvironment reaches for by label, precisely
	// because it is the ambient one whose token flag is still called "token".
	registerSpec("slack", FormSpec{
		Target:     []string{"team-id"},
		Credential: []string{"token"},
		Labels:     map[string]string{"team-id": "team ID"},
	})

	// snowflake identifies the target with --account, and the account
	// identifier is the one thing in ~/.snowflake/connections.toml the
	// connector can actually use -- no flag takes a connection name, which is
	// the finding srcSnowflakeConnection records. --user leads CREDENTIAL
	// rather than TARGET because ParseCLI reads it only to name the password or
	// private-key credential it builds; it is who you sign in as, not what is
	// being scanned.
	//
	// --password has --ask-pass, so it delivers by prompt, which is the best
	// route there is: the secret never exists outside the process that uses it.
	//
	// --token, the programmatic access token, was the launcher's last standing
	// refusal and is not one any more. The provider reads no environment
	// variable for it -- verified by running it: with SNOWFLAKE_TOKEN,
	// SNOWFLAKE_PASSWORD and SNOWFLAKE_PAT all set, the connector still failed
	// with "missing credentials for snowflake connection" -- and it declares
	// both --token and --password with ConfigEntry "-", so mql derived no name
	// for them either. Every one of those facts is about how a flag *value*
	// gets into the process, and the value now goes straight into req.Flags,
	// which is where cobra would have put it. See TestSnowflakeTokenTravels.
	registerSpec("snowflake", FormSpec{
		Target:     []string{"account", "region", "role"},
		Credential: []string{"user", "ask-pass", "password", "identity-file", "token"},
		Sources:    map[string]string{"account": srcSnowflakeConnection},
		Labels: map[string]string{
			"account":       "account identifier",
			"user":          "user name",
			"ask-pass":      "prompt for password",
			"identity-file": "private key file",
			"token":         "programmatic access token (PAT)",
		},
	})

	// tailscale takes the tailnet as its single positional argument, and the
	// usage string does not name it: Use is the bare word "tailscale", so
	// positionalLabels falls through to "argument 1". Naming it is most of what
	// this spec is for. It reaches Options["tailnet"] straight from req.Args --
	// confirmed by running it, which logs "connecting to custom tailnet
	// tailnet=example.com" -- so it needs no environment route of its own.
	//
	// CREDENTIAL follows the provider's own precedence: AuthenticationMethod
	// checks for an OAuth client first and only then for a token, so the OAuth
	// pair leads and the token is the alternative below it.
	//
	// --base-url is left in OPTIONS. It points at a different coordination
	// server rather than selecting what to scan, and promoting an override into
	// TARGET would suggest it is a question every user has to answer.
	registerSpec("tailscale", FormSpec{
		Positional: []PositionalSpec{{
			Label: "tailnet",
			Desc:  "the tailnet to scan, e.g. example.com — leave empty for the default one",
		}},
		Credential: []string{"client-id", "client-secret", "token"},
		Labels: map[string]string{
			"client-id":     "OAuth client ID",
			"client-secret": "OAuth client secret",
			"token":         "API access token",
			"base-url":      "API base URL",
		},
	})

	// vercel scopes a scan with --team, and cnspec's own discovery is the only
	// thing that can enumerate them, so the picker is the registered
	// discover.vercel.teams source. It is CostRemote, which means it is skipped
	// when the form opens, shows "press enter to look it up" and runs only when
	// the field is opened -- the designed path for a source that needs a
	// credential and a round trip.
	//
	// It is attached through Sources rather than LiveSources on purpose:
	// applySources only fills a field whose `source` is set, so a field with
	// nothing but a live source would spin, answer, and never show what it
	// found. LiveSources is the second half of a pair, not a source on its own.
	//
	// discoverSourceID is called rather than srcDiscoverVercelProjects because
	// the latter is declared but not registered -- vercel's registered target
	// is teams, which is also the one --team consumes.
	registerSpec("vercel", FormSpec{
		Target:     []string{"team"},
		Credential: []string{"token"},
		Sources:    map[string]string{"team": discoverSourceID("vercel", "teams")},
		Labels: map[string]string{
			"team":  "team (slug or ID)",
			"token": "API token",
		},
	})

	// zoom uses Server-to-Server OAuth: an account id plus a client id and
	// secret. The account is what is being scanned; the other two are the
	// credential.
	registerSpec("zoom", FormSpec{
		Target:     []string{"account-id"},
		Credential: []string{"client-id", "client-secret"},
		Labels: map[string]string{
			"account-id":    "account ID",
			"client-id":     "S2S OAuth client ID",
			"client-secret": "S2S OAuth client secret",
		},
	})
}
