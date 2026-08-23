// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package forms

import "strings"

// The AI & LLM family: nine connectors that mostly reduce to one API key from
// one environment variable.
//
// That shape is the one the launcher got wrong first. A provider with a single
// token has nothing to enumerate, so a value picker over it renders an empty
// box with no explanation -- which is a category error, not a missing feature.
// The credential-state readout in source_ambient.go is the widget for this
// class: it names the *variable* that supplied the token, never the token, and
// it is paired with a paste box for the shell the user has already left.
//
// But the family is not uniform, and applying one template to all nine would be
// its own version of the same mistake:
//
//   - ambient API key: anthropic, claude, huggingface, mistral, openai,
//     together. One token, one variable, a readout and a paste box.
//   - URL-addressed, credential optional: ollama and vllm. Both are usually
//     something you run yourself, addressed by host or endpoint, with no key at
//     all. Each provider *does* declare a token for its hosted form, so the
//     readout says so rather than implying the local case is missing something.
//   - neither: mcp. Its target is a transport plus either a URL or a command to
//     run, and its bearer token applies only to the HTTP(S) form. A readout that
//     could not be hidden for the stdio form would be wrong half the time, so
//     mcp gets an ordinary credential field with a verified route instead.
//
// Every environment variable named below was read out of the provider that
// receives it, and each was then confirmed a second way -- the string is present
// in the shipped binary, and for most of them the running provider was observed
// using it. The flag *descriptions* were not treated as evidence: openai's
// --token description does not mention OPENAI_API_KEY even though
// connection.go reads it, and claude's --organization-id description names
// ANTHROPIC_ORGANIZATION_ID which the connector never reads (only the SDK's
// env-federation path does, which is why nothing below prefills from it).
//
//	anthropic   ANTHROPIC_API_KEY        --token
//	            ANTHROPIC_ADMIN_API_KEY  --admin-token
//	claude      ANTHROPIC_API_KEY        --token
//	            ANTHROPIC_ADMIN_API_KEY  --admin-token
//	huggingface HF_TOKEN                 --token
//	mcp         MCP_TOKEN                --token
//	mistral     MISTRAL_API_KEY          --token, then MISTRAL_KEY
//	ollama      OLLAMA_API_TOKEN         --token
//	            OLLAMA_HOST              --host
//	openai      OPENAI_API_KEY           --token
//	            OPENAI_ORG_ID            --organization
//	            OPENAI_PROJECT_ID        --project
//	            OPENAI_BASE_URL          --base-url
//	together    TOGETHER_API_KEY         --token
//	vllm        VLLM_API_KEY             --api-key
//
// None of these declare a ClassAmbient source id: the readout is filled
// synchronously by refreshAmbient, and a source only adds an Activity for
// something to wait on. `Source: ""` is the shape for a credential the launcher
// reports on without one, and it keeps nine ids out of source_ids.go that
// nothing would ever name.

func init() {
	registerAmbient(
		// anthropic and claude are two connectors over the same platform and
		// read the same two variables. Verified twice over: claude's
		// connection.go reads ANTHROPIC_API_KEY and ANTHROPIC_ADMIN_API_KEY
		// directly, and the shipped anthropic provider was observed doing the
		// same -- with nothing set it refuses with "admin API key required: set
		// --admin-token or ANTHROPIC_ADMIN_API_KEY", and with the variable set
		// to a junk value it reaches the API and is turned away with a 401.
		//
		// Only the plain key gets a readout, though both connectors read a
		// second variable for --admin-token. A second readout needs a `special`
		// marker of its own and that marker has to be declared in
		// launcherOwnedFields, which lives in a file this one does not own; the
		// admin key therefore keeps its ordinary --admin-token field and the
		// verified ANTHROPIC_ADMIN_API_KEY route registered below, and simply
		// does not get a row saying whether the environment already holds one.
		// Adding it later is one line there and a Name:"admin" credential here.
		ambientCredential{
			Connector: "anthropic", Flag: "token",
			Env: []string{"ANTHROPIC_API_KEY"},
		},
		ambientCredential{
			Connector: "claude", Flag: "token",
			Env: []string{"ANTHROPIC_API_KEY"},
		},

		ambientCredential{
			Connector: "huggingface", Flag: "token",
			Env: []string{"HF_TOKEN"},
		},

		// mistral reads two, in this order, and Env order is what the readout
		// reports, so it has to match: tokenFromConfig takes MISTRAL_API_KEY
		// and falls back to MISTRAL_KEY. Both were observed on the shipped
		// provider -- with neither set it refuses with "mistral: API key is
		// required (use --token or set MISTRAL_API_KEY)", and MISTRAL_KEY alone
		// gets past that check.
		ambientCredential{
			Connector: "mistral", Flag: "token",
			Env: []string{"MISTRAL_API_KEY", "MISTRAL_KEY"},
		},

		// ollama is normally a server on this machine with no authentication at
		// all, so the default hint -- "export OLLAMA_API_TOKEN, or paste one
		// below" -- reads as an instruction and would send a local user looking
		// for a credential that does not exist. This still names the variable,
		// because that is what a hint is for, and adds the part the generic
		// wording cannot know: that the token is for Ollama Cloud.
		ambientCredential{
			Connector: "ollama", Flag: "token",
			Env: []string{"OLLAMA_API_TOKEN"},
			Hint: "not set — for Ollama Cloud export OLLAMA_API_TOKEN or paste " +
				"one below; a local server does not need one",
			PasteHint: "paste an Ollama Cloud token — it never reaches the command line",
			Compose:   composeOllamaHost,
		},

		ambientCredential{
			Connector: "openai", Flag: "token",
			Env:     []string{"OPENAI_API_KEY"},
			Compose: composeOpenAIAccount,
		},

		ambientCredential{
			Connector: "together", Flag: "token",
			Env: []string{"TOGETHER_API_KEY"},
		},

		// vllm's credential flag is --api-key rather than --token, and an
		// ambient credential can only redraw a flag it names into a paste box.
		// The flag is already classified as a secret and already sits in
		// CREDENTIAL with a verified route, so this is a readout on its own:
		// it reports whether VLLM_API_KEY is set and says that an
		// unauthenticated server -- the usual case -- needs neither.
		//
		// The hint points at the field by name rather than by direction. A
		// readout with no flag to sit above is appended to the form, so it
		// lands after the field it is talking about, and "below" would be
		// pointing at the wrong half of the screen.
		ambientCredential{
			Connector: "vllm",
			Env:       []string{"VLLM_API_KEY"},
			Hint: "not set — export VLLM_API_KEY, or fill in the API key " +
				"field; an unauthenticated server needs neither",
		},
	)

	// There used to be a table of delivery routes here, one variable per flag,
	// because a typed credential with no registered variable could not be
	// carried at all. The connector's own ParseCLI answers that now, so the
	// variables above are only what the *readouts* name -- what the provider
	// reads when the launcher supplies nothing -- and no longer double as the
	// launcher's only way to hand a value over.
	//
	// This family is where that distinction bites hardest. openai, ollama,
	// huggingface and claude never read conn.Credentials at all, so a token
	// typed into any of their forms lands in conn.Options and the OS keychain
	// cannot hold it; the launch says so by name rather than pretending
	// otherwise. See launchRequest.inventoryPlan.

	// anthropic asks for nothing but a credential: no target, no options, and
	// MinArgs of zero, so the empty Positional is what suppresses the derived
	// argument slot the generic layer would otherwise offer.
	registerSpec("anthropic", FormSpec{
		Credential: []string{"token", "admin-token"},
		Labels: map[string]string{
			"token":       "API key",
			"admin-token": "admin API key",
		},
	})

	// claude is the same platform with a workload-identity path and discovery
	// on top. The workspace picker is the one live source in this family:
	// srcDiscoverClaudeWorkspaces asks cnspec's own discovery, which costs a
	// round trip and a credential, so it is a LiveSource that runs when the
	// field is opened rather than when the form is built.
	//
	// The WIF fields sit in CREDENTIAL because that is what they configure, not
	// because they are secret -- an id and a path are neither, and they travel
	// on the command line as the provider expects. Only the two keys are
	// secrets, and both are classified as such by their names.
	registerSpec("claude", FormSpec{
		Target:      []string{"organization-id", "workspace-id"},
		LiveSources: map[string]string{"workspace-id": srcDiscoverClaudeWorkspaces},
		Credential: []string{
			"token", "admin-token",
			"identity-token-file", "federation-rule-id", "service-account-id",
		},
		Labels: map[string]string{
			"organization-id":     "organization",
			"workspace-id":        "workspace",
			"token":               "API key",
			"admin-token":         "admin API key",
			"identity-token-file": "OIDC token file",
			"federation-rule-id":  "federation rule",
			"service-account-id":  "service account",
		},
	})

	// huggingface scopes a scan to one user or organization. The two values
	// --namespace-type accepts are the provider's own: it rejects anything
	// other than "user" or "org" in ParseCLI, so offering them is offering the
	// whole set.
	registerSpec("huggingface", FormSpec{
		Target:  []string{"namespace", "namespace-type"},
		Choices: map[string][]string{"namespace-type": {"user", "org"}},
		Labels: map[string]string{
			"namespace":      "user or organization",
			"namespace-type": "namespace kind",
			"token":          "API token",
		},
	})

	// mcp takes two arguments -- a transport and then either a URL or the
	// command that starts a server -- and which of its flags mean anything
	// depends on the first. A bearer token is for the HTTP(S) form; a working
	// directory is for the stdio one.
	//
	// --env stays hidden, and now for one reason rather than two. It was both
	// unfillable -- a list flag with no values to list, drawn as a multi-select
	// with nothing in it -- and unsafe, and typeEmptyLists fixed the first: an
	// optionless list is a typed, comma-separated field now, so a row here
	// would work.
	//
	// The second reason is the one that decides it. What goes in --env for a
	// stdio MCP server is that server's own credentials, `--env
	// GITHUB_TOKEN=…` being the documented shape, and the flag is a list of
	// KEY=VALUE pairs rather than a credential the classifier can recognise --
	// so the launcher would put it on a command line `ps auxww` can read, with
	// nothing to warn the person typing. It is not dropped from cnspec, only
	// from this screen; `--env KEY=VALUE` still works on the command line,
	// which is where the person typing it can see where it goes.
	registerSpec("mcp", FormSpec{
		Positional: []PositionalSpec{
			{
				Label: "transport", Desc: "how to reach the server",
				Required: true,
				Options:  []string{"http", "https", "stdio"},
			},
			{
				Label: "server", Desc: "the server URL, or the command that starts one",
				Required: true,
			},
		},
		Target:     []string{"cwd"},
		Credential: []string{"token"},
		Labels: map[string]string{
			"cwd":   "working directory",
			"token": "bearer token",
		},
		ShowFlagsIf: map[string][]string{
			"token": {"http", "https"},
			"cwd":   {"stdio"},
		},
		Hide: []string{"env"},
	})

	registerSpec("mistral", FormSpec{
		Target: []string{"workspace", "base-url"},
		Labels: map[string]string{
			"workspace": "workspace",
			"base-url":  "API base URL",
			"token":     "API key",
		},
	})

	// ollama is addressed by URL, and the default is the one every local
	// install listens on. It is offered as a suggestion rather than a fixed
	// value: the field stays free text, so a server on another host or port
	// still works.
	registerSpec("ollama", FormSpec{
		Target:  []string{"host"},
		Choices: map[string][]string{"host": {"http://localhost:11434"}},
		Labels: map[string]string{
			"host":  "server",
			"token": "cloud API token",
		},
	})

	registerSpec("openai", FormSpec{
		Target: []string{"organization", "project", "base-url"},
		Labels: map[string]string{
			"organization": "organization",
			"project":      "project",
			"base-url":     "API base URL",
			"token":        "API key",
		},
	})

	registerSpec("together", FormSpec{
		Target: []string{"project", "base-url"},
		Labels: map[string]string{
			"project":  "project",
			"base-url": "API base URL",
			"token":    "API key",
		},
	})

	// vllm is the other locally-hosted one, and the only connector in the
	// family that takes its target as an argument: `vllm <endpoint>`, declared
	// MinArgs and MaxArgs of one.
	registerSpec("vllm", FormSpec{
		Positional: []PositionalSpec{{
			Label:    "endpoint",
			Desc:     "the vLLM server URL, e.g. http://localhost:8000",
			Required: true,
		}},
		Credential: []string{"api-key"},
		Labels: map[string]string{
			"api-key":  "API key",
			"insecure": "skip TLS verification",
		},
	})
}

// composeOllamaHost shows the server the scan will actually talk to.
//
// The provider takes --host, then OLLAMA_HOST, then localhost:11434. Mirroring
// the middle step into an editable field is how a user gets to notice that a
// variable in their shell is pointing the scan somewhere other than where they
// assumed.
func composeOllamaHost(f *form, env envLookup) {
	prefillFromEnv(f, env, "host", "OLLAMA_HOST")
}

// composeOpenAIAccount fills in the three non-secret values openai reads from
// the environment alongside its key.
//
// Only these three: OPENAI_ADMIN_KEY and the webhook and custom-header
// variables also appear in the shipped binary, but they come from the vendored
// SDK rather than from the connector, and the connector's own reads are the
// only ones a launcher can honestly report.
func composeOpenAIAccount(f *form, env envLookup) {
	prefillFromEnv(f, env, "organization", "OPENAI_ORG_ID")
	prefillFromEnv(f, env, "project", "OPENAI_PROJECT_ID")
	prefillFromEnv(f, env, "base-url", "OPENAI_BASE_URL")
}

// prefillFromEnv fills one non-secret field from the variable the provider
// reads for it, and says so.
//
// A value the user typed is never overwritten, and neither is one another
// source already found: the environment is what the provider would have fallen
// back to, not an override. Nothing here is ever called for a secret -- a
// credential's value is exactly what the launcher does not hold.
func prefillFromEnv(f *form, env envLookup, flag, name string) {
	value := strings.TrimSpace(envValue(env, name))
	if value == "" {
		return
	}
	i := f.IndexOfFlag(flag)
	if i < 0 || f.Fields()[i].Secret || f.Fields()[i].IsSet() {
		return
	}
	f.Fields()[i].SetValue(value)
	f.Fields()[i].SetPrefilled(ambientWhyEnv)
}
