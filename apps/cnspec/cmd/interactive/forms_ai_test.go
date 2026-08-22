// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	"sort"
	"strings"
	"testing"
)

// The AI family is nine connectors whose whole credential is an API key, so
// every test below injects the environment rather than reading it. Those are
// exactly the variables a developer running this suite is most likely to have
// exported, and a test that consulted the real environment would be reading
// that developer's actual keys to decide whether it passes.
//
// withAmbientEnv, in source_ambient_test.go, is the seam.

// aiConnectors are the nine this file curates.
var aiConnectors = filedHere(
	"anthropic", "claude", "huggingface", "mcp", "mistral",
	"ollama", "openai", "together", "vllm",
)

// aiKey is a stand-in credential. Nothing that renders, and nothing that
// reaches a command line, may ever contain it.
const aiKey = "must-never-be-displayed"

// Every one of the nine is claimed by this file. Without this, dropping a
// registerSpec call leaves that connector on the generic flag-derived screen,
// which looks like a form and says nothing about being uncurated.
func TestEveryAIConnectorIsCurated(t *testing.T) {
	for _, name := range aiConnectors {
		if _, ok := formSpecs[name]; !ok {
			t.Errorf("%s has no registered spec", name)
		}
	}
}

// aiCredentialCase is one connector's credential: the flag that carries it, the
// variable a value typed into that flag must travel in, and the variables the
// readout watches, in the order the provider reads them.
type aiCredentialCase struct {
	connector string
	flag      string
	env       string
	ambient   []string
	// fill supplies whatever else the form needs before it can be launched.
	fill func(t *testing.T, f *form)
}

func aiCredentialCases() []aiCredentialCase {
	return []aiCredentialCase{
		{connector: "anthropic", flag: "token", env: "ANTHROPIC_API_KEY",
			ambient: []string{"ANTHROPIC_API_KEY"}},
		{connector: "anthropic", flag: "admin-token", env: "ANTHROPIC_ADMIN_API_KEY"},
		{connector: "claude", flag: "token", env: "ANTHROPIC_API_KEY",
			ambient: []string{"ANTHROPIC_API_KEY"}},
		{connector: "claude", flag: "admin-token", env: "ANTHROPIC_ADMIN_API_KEY"},
		{connector: "huggingface", flag: "token", env: "HF_TOKEN",
			ambient: []string{"HF_TOKEN"}},
		{connector: "mcp", flag: "token", env: "MCP_TOKEN", fill: fillMCP},
		{connector: "mistral", flag: "token", env: "MISTRAL_API_KEY",
			ambient: []string{"MISTRAL_API_KEY", "MISTRAL_KEY"}},
		{connector: "ollama", flag: "token", env: "OLLAMA_API_TOKEN",
			ambient: []string{"OLLAMA_API_TOKEN"}},
		{connector: "openai", flag: "token", env: "OPENAI_API_KEY",
			ambient: []string{"OPENAI_API_KEY"}},
		{connector: "together", flag: "token", env: "TOGETHER_API_KEY",
			ambient: []string{"TOGETHER_API_KEY"}},
		{connector: "vllm", flag: "api-key", env: "VLLM_API_KEY",
			ambient: []string{"VLLM_API_KEY"}, fill: fillVLLM},
	}
}

func fillMCP(t *testing.T, f *form) {
	t.Helper()
	fieldByLabel(t, *f, "transport").SetValue("https")
	fieldByLabel(t, *f, "server").SetValue("https://mcp.example.com/mcp")
}

func fillVLLM(t *testing.T, f *form) {
	t.Helper()
	fieldByLabel(t, *f, "endpoint").SetValue("http://localhost:8000")
}

// The whole point of the delivery machinery: a key the user types reaches the
// provider, and never becomes a word on a command line, where `ps auxww` would
// hand it to every other user on the machine.
//
// This used to also assert which environment variable each key travelled in,
// because the launcher had to name one and the name was a fact about the
// provider that a person had checked. The names are still recorded on the cases
// below, but as what the *readout* watches -- what the provider reads when the
// launcher supplies nothing -- which is a different claim and the only one they
// were ever safe to make.
func TestEveryAIKeyReachesTheProviderAndNotArgv(t *testing.T) {
	for _, tc := range aiCredentialCases() {
		t.Run(tc.connector+"/"+tc.flag, func(t *testing.T) {
			withAmbientEnv(t, nil)
			c, f := formFor(t, tc.connector)
			if tc.fill != nil {
				tc.fill(t, &f)
			}

			i := f.IndexOfFlag(tc.flag)
			if i < 0 {
				t.Fatalf("no --%s field on the form: %v", tc.flag, fieldLabels(f))
			}
			if !f.Fields()[i].Secret {
				t.Fatalf("--%s is not marked secret, so its value could reach argv", tc.flag)
			}
			if f.Fields()[i].Section != sectionCredential {
				t.Errorf("--%s sits in %q, want CREDENTIAL", tc.flag, f.Fields()[i].Section)
			}
			f.Fields()[i].SetValue(aiKey)

			assertCredentialReachesTheProvider(t, c, f, tc.flag, aiKey)

			// The command bar is what a user checks this against, so it has to
			// agree with what launch really does.
			if preview := (launchRequest{form: f}).preview(c, scanAction()); strings.Contains(preview, aiKey) {
				t.Fatalf("the command preview showed the key: %q", preview)
			}
		})
	}
}

// The readout's one job is to say where a credential came from. Its one
// prohibition is saying what it is.
func TestAIReadoutsNameTheVariableNeverTheKey(t *testing.T) {
	for _, tc := range aiCredentialCases() {
		if len(tc.ambient) == 0 {
			continue
		}
		t.Run(tc.connector, func(t *testing.T) {
			withAmbientEnv(t, map[string]string{tc.ambient[0]: aiKey})
			_, f := formFor(t, tc.connector)

			fd := fieldByLabel(t, f, "credential")
			if fd.Kind != fieldCredentialState {
				t.Fatalf("the readout is a %v, want a credential-state field", fd.Kind)
			}
			if fd.Value() != tc.ambient[0] {
				t.Errorf("readout = %q, want %q", fd.Value(), tc.ambient[0])
			}
			if strings.Contains(fd.Display(), aiKey) {
				t.Fatalf("the readout displayed the key: %q", fd.Display())
			}
			for _, g := range f.Fields() {
				if strings.Contains(g.Value(), aiKey) {
					t.Fatalf("%q holds the key itself", g.Label)
				}
			}
			// And it is launcher-owned, so it has no flag to be emitted under
			// and must not be mistaken for a positional argument either.
			for _, a := range f.Args() {
				if strings.Contains(a, tc.ambient[0]) || strings.Contains(a, aiKey) {
					t.Fatalf("the credential reached the command line: %v", f.Args())
				}
			}
		})
	}
}

// An empty row and a row with nothing to fill in look identical, which is the
// failure the readout exists to end: every one of them has to name the variable
// to export when it finds nothing.
func TestEveryAIReadoutSaysWhatToExportWhenNothingIsSet(t *testing.T) {
	for _, tc := range aiCredentialCases() {
		if len(tc.ambient) == 0 {
			continue
		}
		t.Run(tc.connector, func(t *testing.T) {
			withAmbientEnv(t, nil)
			_, f := formFor(t, tc.connector)

			fd := fieldByLabel(t, f, "credential")
			if fd.IsSet() {
				t.Fatalf("nothing is exported but the readout says %q", fd.Value())
			}
			if hint := credentialHint(*fd); !strings.Contains(hint, tc.ambient[0]) {
				t.Errorf("the hint should name %s, got %q", tc.ambient[0], hint)
			}
		})
	}
}

// mistral reads MISTRAL_API_KEY and falls back to MISTRAL_KEY, so the readout
// has to name whichever the provider will actually reach for. Both were
// confirmed against the shipped provider: with neither set it refuses and names
// MISTRAL_API_KEY, and MISTRAL_KEY alone gets past that check.
func TestMistralReadoutFollowsTheProvidersPrecedence(t *testing.T) {
	for _, tc := range []struct {
		name string
		env  map[string]string
		want string
	}{
		{"both", map[string]string{"MISTRAL_API_KEY": aiKey, "MISTRAL_KEY": aiKey}, "MISTRAL_API_KEY"},
		{"only the fallback", map[string]string{"MISTRAL_KEY": aiKey}, "MISTRAL_KEY"},
		{"only the first", map[string]string{"MISTRAL_API_KEY": aiKey}, "MISTRAL_API_KEY"},
		{"neither", nil, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			withAmbientEnv(t, tc.env)
			_, f := formFor(t, "mistral")
			if got := fieldByLabel(t, f, "credential").Value(); got != tc.want {
				t.Errorf("readout = %q, want %q", got, tc.want)
			}
		})
	}
}

// A pasted key wins over the environment, because that is the precedence every
// one of these providers implements. A row naming a variable while a typed key
// was about to be used would be confidently wrong about the one fact it states.
func TestAITypedKeyBeatsTheEnvironment(t *testing.T) {
	withAmbientEnv(t, map[string]string{"OPENAI_API_KEY": aiKey})
	_, f := formFor(t, "openai")

	if got := fieldByLabel(t, f, "credential").Value(); got != "OPENAI_API_KEY" {
		t.Fatalf("readout = %q, want the variable", got)
	}
	paste := fieldByLabel(t, f, "API key")
	if paste.Kind != fieldPaste {
		t.Fatalf("the token flag is a %v, want a paste box", paste.Kind)
	}
	paste.SetValue("typed-" + aiKey)

	resolveSources(&f)
	readout := fieldByLabel(t, f, "credential")
	if readout.Value() != "pasted" {
		t.Errorf("readout = %q, want it to follow the typed key", readout.Value())
	}
	if strings.Contains(readout.Display(), aiKey) {
		t.Fatal("the readout displayed the typed key")
	}
}

// ollama is normally a server on this machine with no authentication at all, so
// the generic hint -- "export OLLAMA_API_TOKEN, or paste one below" -- would
// send a local user looking for a credential that does not exist.
func TestOllamaSaysALocalServerNeedsNoToken(t *testing.T) {
	withAmbientEnv(t, nil)
	_, f := formFor(t, "ollama")

	hint := credentialHint(*fieldByLabel(t, f, "credential"))
	if !strings.Contains(hint, "local server does not need one") {
		t.Errorf("the hint should say a local server needs no token, got %q", hint)
	}
	if !strings.Contains(hint, "OLLAMA_API_TOKEN") {
		t.Errorf("the hint should still name the variable, got %q", hint)
	}
}

// The two locally-hosted connectors are addressed by URL, not by credential, so
// the address is what their TARGET has to ask for.
func TestLocallyHostedAIConnectorsAskForAnAddress(t *testing.T) {
	withAmbientEnv(t, nil)

	_, ollama := formFor(t, "ollama")
	host := fieldByLabel(t, ollama, "server")
	if host.Section != sectionTarget {
		t.Errorf("ollama's server sits in %q, want TARGET", host.Section)
	}
	if !containsString(host.Options, "http://localhost:11434") {
		t.Errorf("the default address is not offered: %v", host.Options)
	}
	if host.Strict {
		// A suggestion, not a validation list: a server on another host or
		// port has to stay typeable.
		t.Error("ollama's server field refuses anything but the suggestion")
	}

	_, vllm := formFor(t, "vllm")
	endpoint := fieldByLabel(t, vllm, "endpoint")
	if endpoint.Flag != "" {
		t.Errorf("vllm's endpoint is --%s, want the positional argument", endpoint.Flag)
	}
	if !endpoint.Required {
		t.Error("vllm cannot be scanned without an endpoint, so it must be required")
	}
	fieldByLabel(t, vllm, "endpoint").SetValue("http://localhost:8000")
	if !containsString(vllm.Args(), "http://localhost:8000") {
		t.Errorf("the endpoint did not reach the command: %v", vllm.Args())
	}
}

// vllm's credential flag is --api-key rather than --token, so the readout
// cannot redraw it as a paste box; the flag keeps its own field. The readout
// still has to appear, and has to say that the usual unauthenticated server
// needs nothing at all.
func TestVLLMReportsItsKeyWithoutClaimingOne(t *testing.T) {
	withAmbientEnv(t, nil)
	_, f := formFor(t, "vllm")

	readout := fieldByLabel(t, f, "credential")
	if readout.Kind != fieldCredentialState {
		t.Fatalf("vllm has no credential readout, only %v", fieldLabels(f))
	}
	hint := credentialHint(*readout)
	if !strings.Contains(hint, "needs neither") {
		t.Errorf("the hint should say an open server needs no key, got %q", hint)
	}
	// A readout with no flag to sit above is appended, so it renders after the
	// field it describes and must not point downward at it.
	if strings.Contains(hint, "below") {
		t.Errorf("the hint points below at a field that is above it: %q", hint)
	}
	if key := fieldByLabel(t, f, "API key"); key.Kind == fieldPaste {
		t.Error("--api-key was redrawn as a paste box, which only --token can be")
	}
}

// ollama and openai read values from the environment that are not credentials,
// and the provider will use them whether or not the user knows. Mirroring them
// into an editable field is how the user gets to notice.
func TestAIFieldsPrefilledFromTheEnvironment(t *testing.T) {
	for _, tc := range []struct {
		connector string
		env       map[string]string
		label     string
		want      string
	}{
		{"ollama", map[string]string{"OLLAMA_HOST": "http://ollama.internal:11434"},
			"server", "http://ollama.internal:11434"},
		{"openai", map[string]string{"OPENAI_ORG_ID": "org-123"}, "organization", "org-123"},
		{"openai", map[string]string{"OPENAI_PROJECT_ID": "proj-123"}, "project", "proj-123"},
		{"openai", map[string]string{"OPENAI_BASE_URL": "https://gw.example.com/v1"},
			"API base URL", "https://gw.example.com/v1"},
	} {
		t.Run(tc.connector+"/"+tc.label, func(t *testing.T) {
			withAmbientEnv(t, tc.env)
			_, f := formFor(t, tc.connector)

			fd := fieldByLabel(t, f, tc.label)
			if fd.Value() != tc.want {
				t.Fatalf("%s = %q, want %q", tc.label, fd.Value(), tc.want)
			}
			if fd.Prefilled() == "" {
				t.Error("a value the user did not type must say where it came from")
			}
			if fd.Secret {
				t.Fatal("a secret must never be prefilled from anywhere")
			}
			// It is an ordinary flag, so it does travel on the command line --
			// which is correct here, and is why nothing secret is prefilled.
			if !containsString(f.Args(), "--"+fd.Flag) {
				t.Errorf("the prefilled value did not reach the command: %v", f.Args())
			}
		})
	}
}

// A value the user typed is never replaced by one from the environment.
func TestAIPrefillDoesNotOverwriteTheUser(t *testing.T) {
	withAmbientEnv(t, map[string]string{"OLLAMA_HOST": "http://ollama.internal:11434"})
	_, f := formFor(t, "ollama")
	fieldByLabel(t, f, "server").SetValue("http://chosen:11434")
	resolveSources(&f)
	if got := fieldByLabel(t, f, "server").Value(); got != "http://chosen:11434" {
		t.Fatalf("server = %q, want what the user typed", got)
	}
}

// claude is the only connector in the family with anything to enumerate, and
// the picker for it is a remote call: it needs a credential and a round trip,
// so it must be attached as a live source that runs when the field is opened
// rather than one that runs when the form is built.
func TestClaudeWorkspacesAreAPickerThatWaitsToBeAsked(t *testing.T) {
	withAmbientEnv(t, nil)
	_, f := formFor(t, "claude")

	fd := fieldByLabel(t, f, "workspace")
	if fd.LiveSource != srcDiscoverClaudeWorkspaces {
		t.Fatalf("workspace liveSource = %q, want %q", fd.LiveSource, srcDiscoverClaudeWorkspaces)
	}
	s, ok := sourceByID(srcDiscoverClaudeWorkspaces)
	if !ok {
		t.Fatal("the workspace source is not registered")
	}
	if s.Cost != CostRemote {
		t.Errorf("workspace discovery costs a round trip, Cost = %v", s.Cost)
	}
	if fd.Kind != fieldChoice {
		t.Errorf("workspace is a %v, want a picker", fd.Kind)
	}
	// The discovery targets the connector declares must still be offered.
	discover := fieldByLabel(t, f, "discover")
	for _, want := range []string{"organization", "workspaces"} {
		if !containsString(discover.Options, want) {
			t.Errorf("--discover does not offer %q: %v", want, discover.Options)
		}
	}
}

// mcp is the one connector here whose target is not a credential: it takes a
// transport and then either a URL or the command that starts a server, and
// which of its flags mean anything depends on the first answer.
func TestMCPAsksForATransportThenWhatItNeeds(t *testing.T) {
	withAmbientEnv(t, nil)
	_, f := formFor(t, "mcp")

	transport := fieldByLabel(t, f, "transport")
	if !transport.Strict {
		t.Error("the transport is a closed set, so the field must not accept anything else")
	}
	for _, want := range []string{"http", "https", "stdio"} {
		if !containsString(transport.Options, want) {
			t.Errorf("transport does not offer %q: %v", want, transport.Options)
		}
	}

	visibleLabels := func() []string {
		var out []string
		for _, i := range f.VisibleIndices() {
			out = append(out, f.Fields()[i].Label)
		}
		sort.Strings(out)
		return out
	}

	fieldByLabel(t, f, "transport").SetValue("stdio")
	fieldByLabel(t, f, "server").SetValue("npx -y @modelcontextprotocol/server-github")
	if got := visibleLabels(); containsString(got, "bearer token") {
		t.Errorf("a stdio server was asked for a bearer token: %v", got)
	}
	if got := visibleLabels(); !containsString(got, "working directory") {
		t.Errorf("a stdio server was not offered a working directory: %v", got)
	}
	if got := f.Args(); !containsString(got, "stdio") {
		t.Errorf("the transport did not reach the command: %v", got)
	}

	fieldByLabel(t, f, "transport").SetValue("https")
	fieldByLabel(t, f, "server").SetValue("https://mcp.example.com/mcp")
	if got := visibleLabels(); !containsString(got, "bearer token") {
		t.Errorf("an HTTPS server was not offered a bearer token: %v", got)
	}
	if got := visibleLabels(); containsString(got, "working directory") {
		t.Errorf("an HTTPS server was asked for a working directory: %v", got)
	}
}

// huggingface's --namespace-type is rejected by the provider unless it is
// "user" or "org", so those two are the whole set rather than a suggestion.
func TestHuggingFaceOffersTheNamespaceKindsTheProviderAccepts(t *testing.T) {
	withAmbientEnv(t, nil)
	_, f := formFor(t, "huggingface")

	fd := fieldByLabel(t, f, "namespace kind")
	sorted := append([]string(nil), fd.Options...)
	sort.Strings(sorted)
	if strings.Join(sorted, ",") != "org,user" {
		t.Errorf("namespace kind offers %v, want user and org", fd.Options)
	}
}

// The failure this whole redesign started from: a picker with nothing in it and
// no explanation. A field drawn as a choice has to have somewhere for its
// values to come from -- a declared list, a source, or a live source -- or it is
// an empty box the user cannot fill and cannot escape.
func TestNoAIFormOffersAPickerWithNothingBehindIt(t *testing.T) {
	withAmbientEnv(t, nil)
	for _, name := range aiConnectors {
		t.Run(name, func(t *testing.T) {
			_, f := formFor(t, name)
			for _, fd := range f.Fields() {
				if fd.Kind != fieldChoice && fd.Kind != fieldMultiChoice {
					continue
				}
				if len(fd.Options) > 0 || fd.Source() != "" || fd.LiveSource != "" ||
					fd.SourceBy != nil || fd.LiveSourceBy != nil {
					continue
				}
				t.Errorf("%q is a picker with no values and no source", fd.Label)
			}
		})
	}
}

// Both Anthropic keys typed into one form travel together.
//
// This was a refusal once, and then it was two environment variables. Both were
// artifacts of the launcher having to name a route: a variable carries one
// flag, so two credentials needed two names, and neither name was the
// launcher's to know. One ParseCLI call carries the whole form, so "how many
// secrets does this route carry" stopped being a question.
func TestAnthropicCarriesBothKeysAtOnce(t *testing.T) {
	for _, name := range []string{"anthropic", "claude"} {
		t.Run(name, func(t *testing.T) {
			withAmbientEnv(t, nil)
			c, f := formFor(t, name)
			fieldByLabel(t, f, "API key").SetValue(aiKey)
			fieldByLabel(t, f, "admin API key").SetValue(aiKey + "-admin")

			p := withParser(t, &fakeParser{secretFlag: "token"})
			plan, err := launchRequest{form: f}.plan(c, scanAction())
			if plan.cleanup != nil {
				defer plan.cleanup()
			}
			if err != nil {
				t.Fatalf("two credentials on one form were refused: %v", err)
			}

			for flag, want := range map[string]string{
				"token":       aiKey,
				"admin-token": aiKey + "-admin",
			} {
				got, ok := p.sentValue(flag)
				if !ok {
					t.Errorf("--%s never reached the provider; it was sent %v", flag, sortedKeys(p.flags))
					continue
				}
				if got != want {
					t.Errorf("--%s reached the provider as %q, want %q", flag, got, want)
				}
			}

			preview := (launchRequest{form: f}).preview(c, scanAction())
			if strings.Contains(preview, aiKey) {
				t.Fatalf("the preview showed a key: %q", preview)
			}
			if strings.Contains(strings.Join(plan.args, " "), aiKey) {
				t.Fatalf("a key reached argv: %v", plan.args)
			}
			if strings.Contains(strings.Join(plan.env, " "), aiKey) {
				t.Fatalf("a key reached the child's environment: %v", plan.env)
			}
		})
	}
}
