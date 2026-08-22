// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	"strings"
	"testing"

	tuiform "go.mondoo.com/cnspec/cli/tui/form"

	"go.mondoo.com/mql/providers-sdk/v1/plugin"
)

// The curated Containers & Kubernetes forms.
//
// These three arrived in forms_misc_test.go, a file whose list held twelve
// connectors from five different categories -- helm beside ansible beside
// device beside sbom -- because "miscellaneous" was a statement about the
// contributor's queue rather than about the launcher. They are filed by what
// the launcher shows now; see forms_filing_test.go.
//
// docker, container and k8s are curated in cli/launcher/forms/forms_container.go
// and are exercised elsewhere in this package: the docker context is a
// launcher-owned field with its own tests in source_ambient_test.go, and k8s
// has kubeconfig_test.go.

// containerConnectors are the Containers & Kubernetes connectors this file
// covers. filedHere is what proves the category claim rather than restating it.
var containerConnectors = filedHere("helm", "kustomize", "portainer")

// Every connector this file claims has a spec, and exactly one.
func TestContainerSpecsAreRegisteredExactlyOnce(t *testing.T) {
	for _, name := range containerConnectors {
		if _, ok := formSpecs[name]; !ok {
			t.Errorf("%s has no registered spec", name)
		}
		if containsString(duplicateSpecs, name) {
			t.Errorf("%s was registered twice, so two files claim it", name)
		}
	}
}

// kustomize's whole input is a path, and a curated form owes it an argument
// slot that says what the argument is. See assertPathShapedConnector.
func TestKustomizeNamesItsOverlayPath(t *testing.T) {
	assertPathShapedConnector(t, "kustomize")
}

// helm's argument is two different things, and the questions worth asking
// differ between them. A chart on disk has no repository, no version to pull
// and no registry credential; a chart fetched by name has nothing else.
func TestHelmAsksOnlyWhatTheChosenShapeNeeds(t *testing.T) {
	_, f := formFor(t, "helm")

	if f.Fields()[0].Label != "chart source" {
		t.Fatalf("the selector is not leading the form: %v", fieldLabels(f))
	}
	if f.Fields()[0].Kind != fieldChoice {
		t.Errorf("the selector is not a picker")
	}

	remoteOnly := []string{"repo", "version", "username", "password"}
	for _, flag := range remoteOnly {
		if !hasFlagField(f, flag) {
			t.Fatalf("--%s is not on the form at all: %v", flag, fieldLabels(f))
		}
	}

	visibleFlags := func() map[string]bool {
		out := map[string]bool{}
		for _, i := range f.VisibleIndices() {
			if fd := f.Fields()[i]; fd.Flag != "" {
				out[fd.Flag] = true
			}
		}
		return out
	}

	// A local chart: the repository half is not asked.
	f.Fields()[0].SetValue("local chart")
	shown := visibleFlags()
	for _, flag := range remoteOnly {
		if shown[flag] {
			t.Errorf("a local chart was asked for --%s", flag)
		}
	}
	f.Fields()[1].SetValue("./charts/api")
	if got := f.Args(); len(got) != 1 || got[0] != "./charts/api" {
		t.Errorf("a local chart produced %v, want just the path", got)
	}

	// A remote chart: the repository half is asked, and the selector still
	// contributes nothing to the command line.
	f.Fields()[0].SetValue("chart repository")
	shown = visibleFlags()
	for _, flag := range remoteOnly {
		if !shown[flag] {
			t.Errorf("a remote chart was not asked for --%s", flag)
		}
	}
	f.Fields()[1].SetValue("ingress-nginx")
	fieldByLabel(t, f, "repository URL").SetValue("https://kubernetes.github.io/ingress-nginx")
	got := strings.Join(f.Args(), " ")
	if got != "ingress-nginx --repo https://kubernetes.github.io/ingress-nginx" {
		t.Errorf("a remote chart produced %q", got)
	}
	if strings.Contains(got, "chart repository") {
		t.Error("the selector reached the command line")
	}
}

// helm's list flags, and which of them the form shows now that a list with
// nothing to pick from can be typed into.
//
// The predecessor of this test asserted that all six were hidden, and said in
// its own comment that if the field engine ever grew a way to type into a list
// the right response was to unhide them rather than to loosen the test. It did,
// so --values is shown. The other five are hidden for reasons of their own --
// helm's per-key escaping for --set, cluster-specific render fidelity for
// --api-versions -- which is why this enumerates them rather than deriving the
// answer from the flag type.
func TestHelmShowsTheValuesFilesAndHidesThePerKeyOverrides(t *testing.T) {
	c, f := formFor(t, "helm")

	shown := map[string]bool{"values": true}
	for _, fl := range c.Flags {
		if fl.Type != plugin.FlagType_List {
			continue
		}
		if got := hasFlagField(f, fl.Long); got != shown[fl.Long] {
			t.Errorf("--%s on the form = %v, want %v", fl.Long, got, shown[fl.Long])
		}
	}

	// And the one that is shown can actually be filled in and emitted, which is
	// the whole difference: a multi-choice with no options is a row no
	// keystroke reaches, because openModal declines to open an empty picker and
	// storeCursorField refuses to write the input back into one.
	values := fieldByFlag(t, f, "values")
	if values.Kind != fieldText {
		t.Fatalf("--values is a %v, want a typed field", values.Kind)
	}
	f.Fields()[0].SetValue("local chart")
	f.Fields()[1].SetValue("./chart")
	values.SetValue("prod.yaml,secrets.yaml")
	if got := strings.Join(f.Args(), " "); got != "./chart --values prod.yaml,secrets.yaml" {
		t.Errorf("args = %q, want the comma-separated list pflag's StringSlice splits", got)
	}

	// The premise for the five that stay hidden, asserted rather than assumed.
	empty := tuiform.NewField(tuiform.Decl{Kind: fieldMultiChoice})
	m := Model{detail: detailState{form: tuiform.New("helm", []field{empty})}}
	if _, _ = m.openModal(); m.picker.modal.open {
		t.Error("an empty multi-choice opened a picker after all")
	}
}

// helm's repository password reaches the provider like any other credential.
//
// Its history is worth keeping. It was first refused outright, because no HELM_*
// variable exists to register and the launcher would not put a password on a
// command line `ps auxww` reads. Then it travelled in MONDOO_PASSWORD, a name
// derived from the flag rather than known to the provider, which worked only
// because mql resolves any flag with no config mapping from one. Neither was a
// fact about helm; both were the launcher working around not being able to ask.
func TestHelmRepositoryPasswordReachesTheProvider(t *testing.T) {
	c, f := formFor(t, "helm")
	f.Fields()[0].SetValue("chart repository")
	f.Fields()[1].SetValue("ingress-nginx")
	fieldByLabel(t, f, "repository password").SetValue(sentinel)

	assertCredentialReachesTheProvider(t, c, f, "password", sentinel)
}

// portainer's address is a positional and --address at once, and its token
// must reach the provider without becoming a word on the command line.
func TestPortainerAsksTheAddressOnceAndKeepsTheTokenOffArgv(t *testing.T) {
	c, f := formFor(t, "portainer")

	pos := positionalFields(&f)
	if len(pos) != 1 || pos[0].Label != "instance address" {
		t.Fatalf("portainer does not ask for its address: %v", fieldLabels(f))
	}
	if !pos[0].Required {
		t.Error("the address is optional, so a scan can launch with nothing to connect to")
	}
	if hasFlagField(f, "address") {
		t.Error("--address is offered as well as the positional; the address is asked twice")
	}

	token := fieldByLabel(t, f, "access token")
	if !token.Secret {
		t.Fatal("the access token is not classified as a secret")
	}
	if token.Section != sectionCredential {
		t.Errorf("the access token sits in %q", token.Section)
	}

	f.Fields()[0].SetValue("https://portainer.example.com")
	token.SetValue(sentinel)

	p := assertCredentialReachesTheProvider(t, c, f, "access-token", sentinel)
	if len(p.args) != 1 || p.args[0] != "https://portainer.example.com" {
		t.Errorf("the address did not reach the provider as its argument: %v", p.args)
	}
}
