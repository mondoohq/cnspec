// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	"os"
	"strings"
	"testing"

	tuiform "go.mondoo.com/cnspec/cli/tui/form"

	"go.mondoo.com/mql/providers-sdk/v1/plugin"
)

// The seams the launcher is extended through, tested as contracts rather than
// through whichever connector happens to use them today. Every one of them
// exists so that adding a source or a form is an append in a file its author
// owns, and each was a shared table before.

// A source's Needs names a field by identity, so it can depend on a positional.
//
// It used to compare against fd.flag, which a positional leaves empty. That
// made every dependency worth having inexpressible: the gcp id, the container
// reference and the github name are all positional, so a source keyed off one
// quietly answered for the whole connector instead of for the chosen target --
// the confidently wrong answer the parameterised cache key exists to prevent.
func TestNeedsCanNameAPositionalField(t *testing.T) {
	const src = "test.needs.positional"
	register(Source{
		ID: src, Class: ClassPostConnection, Cost: CostRemote,
		Activity: "asking test for things", Tool: "test",
		Needs: []string{"p:1"},
		Fetch: func([]string) ([]string, error) { return nil, nil },
	})
	t.Cleanup(func() { delete(registry, src) })

	f := tuiform.New("test", []field{
		valued(tuiform.Decl{Label: "kind", Pos: 0, Kind: fieldChoice}, "repo"),
		valued(tuiform.Decl{Label: "name", Pos: 1, Kind: fieldText}, "mondoohq/cnspec"),
	})

	params := sourceParamsFor(f, src)
	if len(params) != 1 || params[0] != "p:1=mondoohq/cnspec" {
		t.Fatalf("sourceParams = %v, want the positional's value", params)
	}
	// And the key it produces must scope to it, or two targets share a list.
	if sourceKeyFor(f, src) == sourceKey(src, nil) {
		t.Error("the key does not vary with the positional it depends on")
	}
}

// The bare flag name is how Needs has always been spelled, and breaking it
// would silently unscope the one source already relying on it.
func TestNeedsStillAcceptsABareFlagName(t *testing.T) {
	f := tuiform.New("k8s", []field{
		valued(tuiform.Decl{Label: "cluster", Flag: "context", Kind: fieldChoice}, "prod"),
	})
	params := sourceParamsFor(f, srcK8sNamespace)
	if len(params) != 1 || params[0] != "context=prod" {
		t.Fatalf("sourceParams = %v, want context=prod", params)
	}
	// The explicit identity spelling of the same thing must resolve to it too.
	if !tuiform.MatchesIdentity(f.Fields()[0], "f:context") {
		t.Error(`"f:context" did not match the --context field`)
	}
	// A bare name is a flag shorthand only. Reading it as a positional index
	// would make "0" mean two different fields.
	if tuiform.MatchesIdentity(tuiform.NewField(tuiform.Decl{Pos: 0}), "0") {
		t.Error(`"0" matched a positional; the shorthand is flags only`)
	}
}

// A spec corrects the shared classifier for its own connector, rather than
// widening a word list that every other connector also reads.
func TestSpecCanOverrideTheSecretClassifier(t *testing.T) {
	c := Connector{
		Provider: "test", Name: "test-secret-override", Use: "test", Installed: true,
		Flags: []plugin.Flag{
			// Classified as a secret by the shared word list.
			{Long: "session-token", Type: plugin.FlagType_String},
			// Classified as harmless by it.
			{Long: "handle", Type: plugin.FlagType_String},
		},
	}
	f := tuiform.New(c.Name, genericFields(c))
	applySpec(&f, c, FormSpec{
		Secret:    []string{"handle"},
		NotSecret: []string{"session-token"},
	})

	byFlag := map[string]field{}
	for _, fd := range f.Fields() {
		byFlag[fd.Flag] = fd
	}
	if !byFlag["handle"].Secret {
		t.Error("Secret did not mark --handle")
	}
	if byFlag["handle"].Section != sectionCredential {
		t.Errorf("--handle is a secret but sits in %q", byFlag["handle"].Section)
	}
	if byFlag["session-token"].Secret {
		t.Error("NotSecret did not clear --session-token")
	}
	if byFlag["session-token"].Section == sectionCredential {
		t.Error("--session-token is no longer a secret but stayed in CREDENTIAL")
	}
}

// A value with no flag to carry it reaches the child through its environment.
// alicloud declares no --profile, snowflake takes no connection name, and
// docker and container take no --context, so this is four connectors' only
// route rather than a special case.
func TestSpecCanDeclareAnEnvironmentVariable(t *testing.T) {
	c := Connector{
		Provider: "test", Name: "test-env", Use: "test", Installed: true,
		MinArgs: 1, MaxArgs: 1,
		Flags: []plugin.Flag{{Long: "region", Type: plugin.FlagType_String}},
	}
	f := tuiform.New(c.Name, genericFields(c))
	applySpec(&f, c, FormSpec{
		Positional: []PositionalSpec{{Label: "profile", Required: true}},
		Target:     []string{"region"},
		Env:        map[string]string{"p:0": "ALIBABA_CLOUD_PROFILE"},
	})
	for i := range f.Fields() {
		if f.Fields()[i].Label == "profile" {
			f.Fields()[i].SetValue("staging")
		}
	}

	r := launchRequest{form: f}
	env, cleanup, err := r.environment()
	if cleanup != nil {
		defer cleanup()
	}
	if err != nil {
		t.Fatal(err)
	}
	if len(env) != 1 || env[0] != "ALIBABA_CLOUD_PROFILE=staging" {
		t.Fatalf("environment = %v, want ALIBABA_CLOUD_PROFILE=staging", env)
	}

	// An unset field contributes nothing: an empty variable is not the same as
	// an absent one, and several SDKs treat it as an explicit empty profile.
	r.form.Fields()[0].SetValue("")
	if env, _, _ := r.environment(); len(env) != 0 {
		t.Errorf("an unset field contributed %v", env)
	}
}

// A source can carry the variable instead, for a value that means the same
// thing wherever it is offered -- a docker context is DOCKER_CONTEXT on every
// form that lets you pick one.
func TestSourceCanDeclareAnEnvironmentVariable(t *testing.T) {
	const src = "test.Env.Source()"
	register(Source{
		ID: src, Class: ClassEnumerated, Cost: CostInstant,
		Activity: "reading ~/.test", Tool: "~/.test", Env: "TEST_CONTEXT",
		Fetch: func([]string) ([]string, error) { return nil, nil },
	})
	t.Cleanup(func() { delete(registry, src) })

	fd := sourced(tuiform.Decl{Label: "context", Kind: fieldChoice}, src)
	fd.SetValue("desktop")
	r := launchRequest{form: tuiform.New("test", []field{fd})}
	env, _, err := r.environment()
	if err != nil {
		t.Fatal(err)
	}
	if len(env) != 1 || env[0] != "TEST_CONTEXT=desktop" {
		t.Fatalf("environment = %v, want TEST_CONTEXT=desktop", env)
	}
}

// A source can also declare what its displayed values mean on a command line,
// for a picker that annotates what it found. The engine must not know which
// picker does that: emitted() used to open with a test for the AWS profile
// source by name, and the second annotated picker would have added a second.
func TestSourceCanDeclareWhatItsValuesEmit(t *testing.T) {
	const src = "test.emit.source"
	register(Source{
		ID: src, Class: ClassEnumerated, Cost: CostInstant,
		Activity: "reading ~/.test", Tool: "~/.test",
		Emit:  func(display string) string { return strings.TrimSuffix(display, "  (annotated)") },
		Fetch: func([]string) ([]string, error) { return nil, nil },
	})
	t.Cleanup(func() { delete(registry, src) })

	f := tuiform.New("test", []field{
		valued(tuiform.Decl{Label: "kind", Pos: 0, Kind: fieldChoice}, "annotated"),
		valued(tuiform.Decl{Label: "thing", Flag: "thing", Kind: fieldChoice,
			SourceBy: map[string]string{"annotated": src}}, "real-name  (annotated)"),
	})
	resolveSources(&f)
	if got := strings.Join(f.Args(), " "); got != "annotated --thing real-name" {
		t.Fatalf("args = %q, want the annotation stripped by the source's own mapping", got)
	}
}

// The k8s case is the one that needed a function rather than a variable name:
// the chosen cluster only means anything once a kubeconfig has been written
// whose current-context is that cluster. It must keep working exactly as it
// did when it was an `if connector == "k8s"` in launchArgs.
func TestKubeContextStillTravelsAsAKubeconfig(t *testing.T) {
	specs := envSpecsFor("k8s")
	if len(specs) != 1 {
		t.Fatalf("k8s declares %d environment contributors, want 1", len(specs))
	}
	if specs[0].Field != "context" {
		t.Errorf("the k8s contributor reads %q, want the --context field", specs[0].Field)
	}

	env, cleanup, err := specs[0].Apply("")
	if cleanup != nil {
		cleanup()
	}
	if err != nil || len(env) != 0 {
		t.Errorf("no chosen cluster gave env=%v err=%v, want nothing", env, err)
	}

	// And no other connector picks up the k8s treatment.
	if len(envSpecsFor("docker")) != 0 {
		t.Error("docker inherited an environment contributor it did not declare")
	}
}

// The same case, all the way through launchArgs: a chosen cluster still leaves
// as KUBECONFIG pointing at a copy whose current-context is that cluster.
//
// This is the assertion that says the generalisation did not change what k8s
// does. It was not covered before -- kubeconfig_test.go tests the copy, not the
// launch -- so the one connector the contract was extracted from had nothing
// checking it arrived.
func TestKubeTargetingSurvivesTheLaunch(t *testing.T) {
	t.Setenv("KUBECONFIG", "testdata/kubeconfig")

	c := Connector{
		Provider: "k8s", Name: "k8s", Use: "k8s", Category: catContainer, Installed: true,
		MaxArgs: 1,
		Flags: []plugin.Flag{
			{Long: "context", Type: plugin.FlagType_String},
			{Long: "namespaces", Type: plugin.FlagType_String},
		},
	}
	f := newForm(c)
	for i := range f.Fields() {
		if f.Fields()[i].Flag == "context" {
			f.Fields()[i].SetValue("aks-trial")
		}
	}

	r := launchRequest{form: f}
	plan, err := r.plan(c, scanAction())
	if plan.cleanup != nil {
		defer plan.cleanup()
	}
	if err != nil {
		t.Fatal(err)
	}

	var kubeconfig string
	for _, e := range plan.env {
		if path, ok := strings.CutPrefix(e, "KUBECONFIG="); ok {
			kubeconfig = path
		}
	}
	if kubeconfig == "" {
		t.Fatalf("the chosen cluster did not reach the child: env = %v", plan.env)
	}
	data, err := os.ReadFile(kubeconfig)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "current-context: aks-trial") {
		t.Errorf("the kubeconfig handed to the child does not select the chosen cluster:\n%s", data)
	}
	if plan.cleanup == nil {
		t.Error("nothing removes the kubeconfig copy after the run")
	}
}

// The three seams that used to sit here are gone with the registries they
// declared into, and what they were guarding is worth recording once.
//
// registerSecretEnv named the variable a provider reads for itself, and it won
// over the one mql derives from the flag name. registerInventoryVerified said
// that somebody had checked a connector's inventory shape against the provider
// that consumes it. And a reserved-name check refused any flag whose derived
// variable was one cnspec reads for its own service account -- MONDOO_TOKEN and
// MONDOO_PRIVATE_KEY above all, the second confirmed end to end by exporting it
// and watching `cnspec status` fail with "invalid credentials: cannot load
// retrieved key".
//
// All three existed because the launcher had to decide, for each connector,
// where a credential could be put so that the provider would find it. It does
// not decide any more: it hands the flags to that provider's own ParseCLI and
// writes back the asset that comes out. There is no derived variable to collide
// with a reserved name, no registry to keep in step with the providers, and no
// shape to have checked by hand.
//
// What replaced them is one test that cannot be satisfied by anyone's
// diligence: TestEveryCredentialFieldRoundTrips looks for the value it sent in
// the asset that came back, over whatever providers are installed.

// The launcher-owned field kinds are stubs, but a stub still has to be inert
// rather than broken: nothing may open a picker over one, and nothing may put
// a credential readout's contents on the command line.
func TestLauncherOwnedKindsAreInert(t *testing.T) {
	m := Model{detail: detailState{form: tuiform.New("test", []field{
		valued(tuiform.Decl{Label: "token found", Kind: fieldCredentialState,
			Special: "credential-state"}, "GITHUB_TOKEN"),
		valued(tuiform.Decl{Label: "paste a token", Kind: fieldPaste,
			Special: "paste", Secret: true}, "<PLACEHOLDER-not-a-real-secret>"),
	})}}

	for i, fd := range m.detail.form.Fields() {
		m.detail.form.SetCursor(i)
		if _, _ = m.openModal(); m.picker.modal.open {
			t.Errorf("%s opened a picker with nothing to pick", fd.Label)
			m.picker.modal = modalState{}
		}
		if !fd.IsSet() {
			t.Errorf("%s holds a value but reports itself unset", fd.Label)
		}
	}

	if got := m.detail.form.Fields()[0].Display(); got != "GITHUB_TOKEN" {
		t.Errorf("the credential readout displayed %q, want where it came from", got)
	}
	if got := m.detail.form.Fields()[1].Display(); strings.Contains(got, "<PLACEHOLDER-not-a-real-secret>") {
		t.Errorf("the paste field displayed its value: %q", got)
	}
	if args := m.detail.form.Args(); len(args) != 0 {
		t.Errorf("launcher-owned fields reached the command line: %v", args)
	}
}
