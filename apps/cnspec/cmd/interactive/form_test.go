// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	"strings"
	"testing"

	tuiform "go.mondoo.com/cnspec/cli/tui/form"

	"go.mondoo.com/mql/providers-sdk/v1/plugin"
)

// sshConnector mirrors what the installed os provider declares for ssh,
// including the one flag in the whole tree that actually sets
// FlagOption_Password.
func sshConnector() Connector {
	return Connector{
		Provider: "os", Name: "ssh", Use: "ssh user@host", Category: catHosts,
		Installed: true, MinArgs: 1, MaxArgs: 1,
		Flags: []plugin.Flag{
			{Long: "sudo", Type: plugin.FlagType_Bool, Default: "false", Desc: "Elevate privileges with sudo"},
			{Long: "insecure", Type: plugin.FlagType_Bool, Default: "false"},
			{Long: "ask-pass", Type: plugin.FlagType_Bool, Default: "false", ConfigEntry: "-"},
			{Long: "password", Short: "p", Type: plugin.FlagType_String, Option: plugin.FlagOption_Password},
			{Long: "identity-file", Short: "i", Type: plugin.FlagType_String},
			{Long: "id-detector", Type: plugin.FlagType_String, Option: plugin.FlagOption_Hidden},
		},
	}
}

func awsConnector() Connector {
	return Connector{
		Provider: "aws", Name: "aws", Use: "aws", Category: catCloud,
		Installed: true, MaxArgs: 4,
		Flags: []plugin.Flag{
			{Long: "profile", Type: plugin.FlagType_String, Desc: "Profile to use"},
			{Long: "region", Type: plugin.FlagType_String},
			{Long: "role", Type: plugin.FlagType_String},
			{Long: "filters", Type: plugin.FlagType_KeyValue},
		},
		Discovery: []string{"instances", "s3-buckets", "accounts"},
	}
}

func githubConnector() Connector {
	return Connector{
		Provider: "github", Name: "github", Use: "github", Category: catSaaS,
		Installed: true, MinArgs: 2, MaxArgs: 2,
		Flags: []plugin.Flag{
			{Long: "token", Type: plugin.FlagType_String, Desc: "GitHub personal access token"},
			{Long: "repos", Type: plugin.FlagType_String},
			{Long: "app-private-key-content", Type: plugin.FlagType_String},
		},
		Discovery: []string{"repos", "organization"},
	}
}

func fieldByLabel(t *testing.T, f form, label string) *field {
	t.Helper()
	for i := range f.Fields() {
		if f.Fields()[i].Label == label {
			return &f.Fields()[i]
		}
	}
	t.Fatalf("no field labelled %q in %v", label, fieldLabels(f))
	return nil
}

func fieldLabels(f form) []string {
	out := make([]string, len(f.Fields()))
	for i, fd := range f.Fields() {
		out[i] = fd.Label
	}
	return out
}

// Hidden and deprecated flags are noise in a launcher and must not appear.
func TestGenericFormSkipsHiddenFlags(t *testing.T) {
	f := newForm(sshConnector())
	for _, fd := range f.Fields() {
		if fd.Label == "id-detector" {
			t.Fatal("hidden flag id-detector should not be a form field")
		}
	}
}

// The positional field comes from MinArgs/MaxArgs, and the usage string names
// it when the token count lines up.
func TestPositionalFieldFromMetadata(t *testing.T) {
	f := newForm(sshConnector())
	fd := fieldByLabel(t, f, "user@host")
	if fd.Flag != "" || fd.Pos != 0 || !fd.Required {
		t.Fatalf("ssh positional = %+v, want positional 0 and required", fd)
	}
}

// --discover is synthesized by the CLI rather than declared, so the form has to
// add it, with all/auto always valid on top of the declared targets.
func TestDiscoverIsAMultiChoice(t *testing.T) {
	f := newForm(awsConnector())
	fd := fieldByLabel(t, f, "discover")
	if fd.Kind != fieldMultiChoice {
		t.Fatalf("discover kind = %v, want multi-choice", fd.Kind)
	}
	for _, want := range []string{"auto", "all", "instances", "s3-buckets"} {
		if !containsString(fd.Options, want) {
			t.Errorf("discover options missing %q: %v", want, fd.Options)
		}
	}
}

func TestFormEmitsArgs(t *testing.T) {
	f := newForm(sshConnector())
	fieldByLabel(t, f, "user@host").SetValue("chris@10.0.0.4")
	fieldByLabel(t, f, "sudo").SetOn(true)

	got := strings.Join(f.Args(), " ")
	if got != "chris@10.0.0.4 --sudo" {
		t.Fatalf("args = %q, want %q", got, "chris@10.0.0.4 --sudo")
	}
}

func TestMultiChoiceEmitsCommaSeparated(t *testing.T) {
	f := newForm(awsConnector())
	fieldByLabel(t, f, "profile").SetValue("prod")
	d := fieldByLabel(t, f, "discover")
	d.SetPicks(map[string]bool{"instances": true, "s3-buckets": true})

	got := strings.Join(f.Args(), " ")
	// Options order decides the rendering, so this is stable.
	if got != "--profile prod --discover instances,s3-buckets" {
		t.Fatalf("args = %q", got)
	}
}

// A connector that leads with an optional selector must collapse to just the
// later argument when the selector is left blank -- `terraform PATH` rather
// than `terraform "" PATH`.
func TestOptionalLeadingPositionalIsSkipped(t *testing.T) {
	c := Connector{
		Provider: "terraform", Name: "terraform", Use: "terraform PATH",
		Installed: true, MinArgs: 1, MaxArgs: 2,
	}
	f := newForm(c)
	fieldByLabel(t, f, "path").SetValue("./infra")

	if got := strings.Join(f.Args(), " "); got != "./infra" {
		t.Fatalf("args = %q, want %q", got, "./infra")
	}

	fieldByLabel(t, f, "kind").SetValue("plan")
	if got := strings.Join(f.Args(), " "); got != "plan ./infra" {
		t.Fatalf("args = %q, want %q", got, "plan ./infra")
	}
}

func TestValidateRequiresPositional(t *testing.T) {
	f := newForm(sshConnector())
	if err := f.Validate(); err == nil {
		t.Fatal("expected an error for an empty required positional")
	}
	fieldByLabel(t, f, "user@host").SetValue("chris@host")
	if err := f.Validate(); err != nil {
		t.Fatalf("unexpected error once set: %v", err)
	}
}

// The overlay must not invent flags. github declares no --enterprise-url in
// this fixture, so the curated entry for it has to vanish rather than render.
func TestOverlayDropsFlagsTheProviderDoesNotDeclare(t *testing.T) {
	f := newForm(githubConnector())
	for _, fd := range f.Fields() {
		if fd.Flag == "enterprise-url" {
			t.Fatal("overlay emitted a flag the provider does not declare")
		}
	}
}

// github takes two positional args that the usage string does not describe;
// only the overlay knows they are a kind and a name.
func TestGitHubOverlayShapesPositionals(t *testing.T) {
	f := newForm(githubConnector())
	kind := fieldByLabel(t, f, "kind")
	if kind.Kind != fieldChoice || !containsString(kind.Options, "repo") {
		t.Fatalf("github kind = %+v, want a choice including repo", kind)
	}
	kind.SetValue("repo")
	fieldByLabel(t, f, "name").SetValue("mondoohq/cnspec")

	if got := strings.Join(f.Args(), " "); got != "repo mondoohq/cnspec" {
		t.Fatalf("args = %q", got)
	}
}

// Sections order TARGET, then CREDENTIAL, then OPTIONS, and the curated fields
// lead their section.
func TestOverlayOrdersSections(t *testing.T) {
	f := newForm(awsConnector())
	sections, bySection := f.Ordered()
	if len(sections) == 0 || sections[0] != sectionTarget {
		t.Fatalf("sections = %v, want TARGET first", sections)
	}
	target := bySection[sectionTarget]
	if len(target) < 3 {
		t.Fatalf("expected profile/region/role promoted to TARGET, got %d fields", len(target))
	}
	if f.Fields()[target[0]].Label != "profile" {
		t.Errorf("first TARGET field = %q, want profile", f.Fields()[target[0]].Label)
	}
}

func TestOverlayAttachesPickers(t *testing.T) {
	if got := fieldByLabel(t, newForm(awsConnector()), "profile").Source(); got != srcAWSProfile {
		t.Errorf("aws profile source = %q, want %q", got, srcAWSProfile)
	}
	if got := fieldByLabel(t, newForm(sshConnector()), "user@host").Source(); got != srcSSHHost {
		t.Errorf("ssh host source = %q, want %q", got, srcSSHHost)
	}
}

func containerConnector() Connector {
	return Connector{
		Provider: "os", Name: "docker", Use: "docker", Category: catContainer,
		Installed: true, MinArgs: 1, MaxArgs: 2,
		Flags:     []plugin.Flag{{Long: "sudo", Type: plugin.FlagType_Bool, Default: "false"}},
		Discovery: []string{"container", "container-images"},
	}
}

// cnspec infers what a container reference points at from the reference
// itself, and the only sub-command word the connector accepts is `file`. The
// kind selector therefore steers the UI without adding a word to the command.
func TestContainerKindEmitsNoExtraWord(t *testing.T) {
	f := newForm(containerConnector())
	kind := fieldByLabel(t, f, "kind")
	ref := fieldByLabel(t, f, "reference")

	for _, k := range []string{"running container", "local image", "registry image"} {
		kind.SetValue(k)
		ref.SetValue("nginx:1.27")
		if got := strings.Join(f.Args(), " "); got != "nginx:1.27" {
			t.Errorf("kind %q emitted %q, want just the reference", k, got)
		}
	}

	// A Dockerfile is the one case where cnspec really does take a word.
	kind.SetValue("dockerfile")
	ref.SetValue("./Dockerfile")
	if got := strings.Join(f.Args(), " "); got != "file ./Dockerfile" {
		t.Errorf("dockerfile emitted %q, want %q", got, "file ./Dockerfile")
	}
}

// typeEmptyLists must not touch a list that does have a picker: --discover
// always has options, and turning it into a text box would lose the one screen
// in the launcher that shows what a connector can find.
func TestAListWithOptionsKeepsItsPicker(t *testing.T) {
	f := newForm(containerConnector())
	fd := fieldByFlag(t, f, "discover")
	if fd.Kind != fieldMultiChoice {
		t.Errorf("--discover is a %v, want the multi-select", fd.Kind)
	}
	if len(fd.Options) == 0 {
		t.Error("--discover has no options, so it would have been demoted to a text box")
	}
	// The selection has to be writable. It used to be a nil map on a field
	// nobody had picked from yet, which is a panic rather than a no-op.
	fd.TogglePick(fd.Options[0])
	if !fd.Picked(fd.Options[0]) {
		t.Error("--discover cannot record a selection")
	}
}

// The docker context is a target with no flag to travel in: docker and
// container declare none, so it reaches the child as DOCKER_CONTEXT and puts
// nothing on the command line.
//
// It also has to be between `kind` and `reference` on screen, because it
// decides what the reference can be -- which is the case that made the
// positional chain worth fixing rather than working around. `reference` takes
// its picker from `kind`, and a launcher-owned field declared between them must
// not become the thing it reads.
func TestDockerContextTravelsInTheEnvironmentOnly(t *testing.T) {
	f := newForm(containerConnector())
	ctx := fieldByLabel(t, f, "docker context")
	if ctx.Special != specialDockerContext {
		t.Fatalf("the context field is %q, want the launcher-owned marker %q",
			ctx.Special, specialDockerContext)
	}
	if ctx.Source() != srcDockerContext {
		t.Errorf("the context field has source %q, want %q", ctx.Source(), srcDockerContext)
	}

	fieldByLabel(t, f, "kind").SetValue("running container")
	resolveSources(&f)
	ctx.SetValue("staging")
	fieldByLabel(t, f, "reference").SetValue("web")

	// The chain behind it is intact: the reference still follows the kind.
	if got := fieldByLabel(t, f, "reference").Source(); got != srcDockerContainer {
		t.Errorf("reference source = %q, want %q; the context field broke the chain",
			got, srcDockerContainer)
	}
	if got := strings.Join(f.Args(), " "); got != "web" {
		t.Errorf("args = %q, want just the reference -- the context is not an argument", got)
	}

	r := launchRequest{form: f}
	env, cleanup, err := r.environment()
	if cleanup != nil {
		defer cleanup()
	}
	if err != nil {
		t.Fatal(err)
	}
	if !containsString(env, dockerContextEnv+"=staging") {
		t.Errorf("the chosen context did not reach the child: %v", env)
	}
	// DOCKER_HOST outranks DOCKER_CONTEXT in the docker CLI and in mql's own
	// dockerclient, so an inherited one has to be cleared or the choice does
	// nothing at all.
	if !containsString(env, dockerHostEnv+"=") {
		t.Errorf("an inherited DOCKER_HOST would still win: %v", env)
	}

	// And the pickers are scoped to it, or they would list the default
	// daemon's containers for a scan pointed somewhere else.
	for _, id := range []string{srcDockerContainer, srcDockerImage} {
		s, ok := sourceByID(id)
		if !ok {
			t.Fatalf("%s is not registered", id)
		}
		if !containsString(s.Needs, "s:"+specialDockerContext) {
			t.Errorf("%s does not depend on the chosen context: Needs = %v", id, s.Needs)
		}
	}
	params := sourceParamsFor(f, srcDockerContainer)
	if !containsString(params, "s:"+specialDockerContext+"=staging") {
		t.Fatalf("the picker is not keyed on the chosen context: %v", params)
	}
	if got := dockerContextEnvFrom(params); !containsString(got, dockerContextEnv+"=staging") {
		t.Errorf("the picker would run against the wrong daemon: %v", got)
	}
	// With no context chosen the picker inherits whatever the user's own shell
	// already says, which is what it did before there was a field at all.
	if got := dockerContextEnvFrom(nil); got != nil {
		t.Errorf("an unset context still forced an environment: %v", got)
	}
}

// Choosing a kind switches which picker the reference field uses, and drops
// values belonging to the previous one.
func TestContainerKindSwitchesPicker(t *testing.T) {
	f := newForm(containerConnector())
	kind := fieldByLabel(t, f, "kind")

	kind.SetValue("running container")
	resolveSources(&f)
	if got := fieldByLabel(t, f, "reference").Source(); got != srcDockerContainer {
		t.Errorf("running container source = %q, want %q", got, srcDockerContainer)
	}

	// Values found for containers must not be offered once images are asked for.
	fieldByLabel(t, f, "reference").Options = []string{"web", "db"}
	kind.SetValue("local image")
	resolveSources(&f)
	ref := fieldByLabel(t, f, "reference")
	if ref.Source() != srcDockerImage {
		t.Errorf("local image source = %q, want %q", ref.Source(), srcDockerImage)
	}
	if len(ref.Options) != 0 {
		t.Errorf("stale container values survived the switch: %v", ref.Options)
	}

	// A registry reference is not enumerated: listing a registry needs
	// credentials and a round trip, and --discover already does it properly.
	kind.SetValue("registry image")
	resolveSources(&f)
	if got := fieldByLabel(t, f, "reference").Source(); got != "" {
		t.Errorf("registry image should have no local picker, got %q", got)
	}
}

// A registry reference is enumerated by cnspec's own discovery, so choosing it
// asks for that enumeration instead of trying to list the registry locally.
func TestRegistryKindRequestsDiscovery(t *testing.T) {
	f := newForm(containerConnector())
	kind := fieldByLabel(t, f, "kind")

	kind.SetValue("registry image")
	resolveSources(&f)
	d := fieldByLabel(t, f, "discover")
	if !d.Picked("container-images") {
		t.Fatalf("expected container-images discovery to be requested, picks=%v", d.Selected())
	}
	if got := strings.Join(f.Args(), " "); !strings.Contains(got, "--discover container-images") {
		t.Fatalf("args = %q", got)
	}
}

// Re-resolving must not stamp over a choice the user made themselves.
func TestResolveDoesNotClobberUserChoices(t *testing.T) {
	f := newForm(containerConnector())
	fieldByLabel(t, f, "kind").SetValue("registry image")
	resolveSources(&f)

	d := fieldByLabel(t, f, "discover")
	d.SetPicks(map[string]bool{"container": true})
	resolveSources(&f) // same kind, so nothing should change
	if !d.Picked("container") || d.Picked("container-images") {
		t.Fatalf("re-resolving overwrote the user's discovery choice: %v", d.Selected())
	}
}

// The tests below assemble a form by hand rather than from a connector, which
// the launcher itself never does: a field's answer is the engine's, so a
// declaration and the answer to it arrive separately. These say the two in one
// line so a table of cases stays a table.

// valued is a field carrying an answer.
func valued(d tuiform.Decl, value string) field {
	fd := tuiform.NewField(d)
	fd.SetValue(value)
	return fd
}

// sourced is a field with a value picker attached, and whatever that picker
// says its values mean on a command line.
func sourced(d tuiform.Decl, source string) field {
	fd := tuiform.NewField(d)
	attachSource(&fd, source)
	return fd
}

// prefilled is a field carrying an answer it derived for the user, and why.
func prefilled(d tuiform.Decl, value, why string) field {
	fd := tuiform.NewField(d)
	fd.Prefill(value, why)
	return fd
}
