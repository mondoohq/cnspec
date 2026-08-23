// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	"encoding/json"
	"sort"
	"strings"
	"testing"

	"go.mondoo.com/cnspec/internal/connectors"
	"go.mondoo.com/mql/providers-sdk/v1/plugin"
)

// The recorded connector metadata, and why the specs are checked against it
// rather than against the machine.
//
// Connector.Flags only exists for providers installed locally: the static
// DefaultProviders list compiled into the binary strips it. CI creates an empty
// .providers directory and points PROVIDERS_PATH at it, so ListActive() returns
// nothing, BuildCatalog() degenerates to that flagless list, and every check
// that needs a flag skips itself. Verified by running the suite that way: the
// existing metadata tests log "not installed locally; skipped" and pass green.
//
// That matters because applySpec drops a flag the connector does not declare,
// on purpose -- it is what makes a stale overlay degrade to the generic screen
// instead of emitting a flag that no longer exists. The same mechanism makes a
// typo invisible: the field simply never appears, and nothing says why. The one
// gate that would catch it therefore has to run somewhere other than the
// author's laptop, which is what the connector artifact is for.
//
// The file is generated from mql provider source by internal/connectorgen:
//
//	make connectors/generate MQL=../mql
//
// It is no longer written from the installed provider set. The generator reads
// two things the installed metadata does not carry -- which environment
// variable backs which flag, and the sub-command words a connector accepts --
// and rewriting the file from a machine would drop both without saying so.
// TestConnectorSnapshotMatchesLive still reports when the recorded flags have
// drifted from what is installed here.
//
// It used to live in this package's testdata/. It now lives in
// internal/connectors, embedded, because production reads it too: testdata is a
// test-only location by convention and providerFormSpec is not a test. These
// checks read the embedded bytes rather than a path, so a test and the launcher
// can no longer be looking at two different files.

const connectorSnapshotPath = "internal/connectors/connectors.json"

// connectorSnapshot is the subset of a connector's declaration the spec checks
// need. It carries no values, only what the connector says about itself: names,
// argument counts, flags and discovery targets.
type connectorSnapshot struct {
	Name      string        `json:"name"`
	Provider  string        `json:"provider"`
	MinArgs   uint          `json:"min_args,omitempty"`
	MaxArgs   uint          `json:"max_args,omitempty"`
	Flags     []flagSnippet `json:"flags,omitempty"`
	Discovery []string      `json:"discovery,omitempty"`

	// Env is what the provider's own source does with a flag: the environment
	// variables it reads that flag's value from, in the order it consults them.
	// It is the fact the interim registry in delivery.go holds by hand.
	Env []flagEnvSnippet `json:"env,omitempty"`
	// Positional is the sub-command vocabulary -- github's org, user and repo --
	// which exists nowhere else in machine-readable form.
	Positional []positionalSnippet `json:"positional,omitempty"`

	// Category is not recorded. categorize() owns it, and a second copy in a
	// generated file would be a second thing to keep in step.
}

// connectorArtifact is the generated file: the connectors, plus what the
// generator could not determine about them.
type connectorArtifact struct {
	Connectors []connectorSnapshot `json:"connectors"`
	Gaps       []connectorGap      `json:"gaps"`
}

// connectorGap is one thing internal/connectorgen could not read out of the
// provider source. It is recorded beside the metadata rather than in a separate
// report so that a test can ask why a connector is missing something.
type connectorGap struct {
	Provider  string `json:"provider,omitempty"`
	Connector string `json:"connector,omitempty"`
	Kind      string `json:"kind"`
	Detail    string `json:"detail"`
}

type flagEnvSnippet struct {
	Flag string   `json:"flag"`
	Vars []string `json:"vars"`
	// Via is "parse-cli" when the variable is consulted while parsing the
	// command line, and "connection" when the connection package reads it at
	// connect time. The second sometimes takes precedence over the flag rather
	// than yielding to it, which is a difference a launcher has to respect.
	Via string `json:"via,omitempty"`
	// Composed marks variables that are joined rather than chosen between, so
	// setting only the first of them does not work.
	Composed bool `json:"composed,omitempty"`
}

type positionalSnippet struct {
	Index  int      `json:"index"`
	Values []string `json:"values"`
}

type flagSnippet struct {
	Long   string `json:"long"`
	Type   int32  `json:"type,omitempty"`
	Option int32  `json:"option,omitempty"`
	Desc   string `json:"desc,omitempty"`
	// Config is the flag's ConfigEntry: "-" means the flag is read from cobra
	// alone and no configuration key or environment variable is consulted for
	// it, and a non-empty entry sends it through viper on a key of the
	// provider's choosing.
	//
	// It used to decide whether a credential could be delivered at all, because
	// mql reads MONDOO_<FLAG> only for a flag that declares no config mapping
	// and the launcher relied on that. It no longer decides anything the
	// launcher does -- a credential goes into a ParseCLI request, which cobra's
	// resolution never enters -- and it is still worth recording, because it is
	// what a *user* running the same command by hand would be subject to.
	Config string `json:"config,omitempty"`
}

// connector rebuilds a Connector from the snapshot, so a test can build the
// real form for it.
func (s connectorSnapshot) connector() Connector {
	c := Connector{
		Provider: s.Provider,
		Name:     s.Name,
		Use:      s.Name,
		Category: categorize(s.Provider, s.Name),
		MinArgs:  s.MinArgs,
		MaxArgs:  s.MaxArgs,
		// A snapshot only exists for a connector that was installed when it was
		// taken, so it is one whose metadata is real.
		Installed: true,
		Discovery: s.Discovery,
	}
	for _, fl := range s.Flags {
		c.Flags = append(c.Flags, plugin.Flag{
			Long:        fl.Long,
			Desc:        fl.Desc,
			Type:        plugin.FlagType(fl.Type),
			Option:      plugin.FlagOption(fl.Option),
			ConfigEntry: fl.Config,
		})
	}
	return c
}

// loadConnectorSnapshot reads the recorded connector metadata. It is a hard
// failure rather than a skip: the whole point is that these checks run where no
// provider is installed.
//
// It returns the connectors the launcher could offer, which is fewer than the
// file holds. The file is a record of what the provider source declares, so it
// includes the replay providers and the connectors that declare nothing at all;
// the two filters below are the ones BuildCatalog already applies, and they
// used to be applied when the file was written. Applying them here instead is
// what keeps this file a faithful record of the source while every caller sees
// the same set it always did.
func loadConnectorSnapshot(t *testing.T) []connectorSnapshot {
	t.Helper()
	var art connectorArtifact
	if err := json.Unmarshal(connectors.Raw(), &art); err != nil {
		t.Fatalf("cannot parse %s: %v", connectorSnapshotPath, err)
	}
	if len(art.Connectors) == 0 {
		t.Fatalf("%s is empty", connectorSnapshotPath)
	}
	out := make([]connectorSnapshot, 0, len(art.Connectors))
	for _, s := range art.Connectors {
		// mock and recording replay a connection rather than making one, and
		// the launcher does not offer them.
		if excludedProviders[s.Provider] {
			continue
		}
		// A connector with no flag, no argument and no discovery target says
		// nothing a form could be built from, and there is nothing here to
		// check a spec against.
		if !s.connector().DeclaresMetadata() {
			continue
		}
		out = append(out, s)
	}
	if len(out) == 0 {
		t.Fatalf("%s holds no connector the launcher could offer", connectorSnapshotPath)
	}
	return out
}

// snapshotByName indexes the snapshot for lookups by connector name.
func snapshotByName(t *testing.T) map[string]connectorSnapshot {
	t.Helper()
	out := map[string]connectorSnapshot{}
	for _, s := range loadConnectorSnapshot(t) {
		out[s.Name] = s
	}
	return out
}

// snapshotFromCatalog builds the snapshot from what is installed right now.
func snapshotFromCatalog() []connectorSnapshot {
	var out []connectorSnapshot
	for _, c := range BuildCatalog() {
		// DeclaresMetadata, not HasFormData: the snapshot records what a
		// connector says about itself, including the connectors whose every
		// flag is hidden. Filtering on renderability would drop `device` from
		// the snapshot, and the reason device gets no spec is precisely a claim
		// about its flags that a test has to be able to read.
		if !c.Installed || !c.DeclaresMetadata() {
			continue
		}
		s := connectorSnapshot{
			Name:      c.Name,
			Provider:  c.Provider,
			MinArgs:   c.MinArgs,
			MaxArgs:   c.MaxArgs,
			Discovery: c.Discovery,
		}
		for _, fl := range c.Flags {
			s.Flags = append(s.Flags, flagSnippet{
				Long:   fl.Long,
				Type:   int32(fl.Type),
				Option: int32(fl.Option),
				Desc:   fl.Desc,
				Config: fl.ConfigEntry,
			})
		}
		sort.Slice(s.Flags, func(i, j int) bool { return s.Flags[i].Long < s.Flags[j].Long })
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// TestConnectorSnapshot checks the recorded metadata parses and describes
// something.
//
// -update used to rewrite the file from the installed provider set, and it
// refuses now instead of doing it. The file gained two facts an installed
// provider does not carry -- the environment variable behind each flag, and the
// sub-command words -- so writing it from a machine would silently delete both,
// leaving a file that still parses and still passes every test here while the
// delivery routes it describes have vanished. Refusing points at the generator
// that can produce all of it.
func TestConnectorSnapshot(t *testing.T) {
	if *updateSnapshot {
		t.Fatalf("%s is generated from mql provider source and cannot be written from the installed providers: "+
			"the installed metadata carries no environment variables and no sub-command words, so this would drop them. "+
			"Run `make connectors/generate MQL=<path to an mql checkout>` instead.", connectorSnapshotPath)
	}

	snap := loadConnectorSnapshot(t)
	for _, s := range snap {
		if s.Name == "" || s.Provider == "" {
			t.Errorf("snapshot entry with no name or provider: %+v", s)
		}
		if !s.connector().DeclaresMetadata() {
			t.Errorf("%s: snapshotted but carries no declared metadata", s.Name)
		}
	}
}

// The snapshot is a copy, so it goes stale. This says so where it can be seen,
// and skips where it cannot -- on CI, where no provider is installed, there is
// nothing to compare against and the snapshot is all there is.
func TestConnectorSnapshotMatchesLive(t *testing.T) {
	// The condition is what the catalog actually yielded, not what ListActive
	// returned: with PROVIDERS_PATH pointed at an empty directory it still
	// returns entries, they just carry no Provider, so BuildCatalog degenerates
	// to the flagless static list. Skipping on the empty result is what keeps
	// this from passing vacuously and reading as "no drift".
	live := snapshotFromCatalog()
	if len(live) == 0 {
		t.Skip("no connector metadata available here; the snapshot cannot be checked for drift")
	}
	recorded := snapshotByName(t)

	checked, drifted := 0, 0
	for _, l := range live {
		r, ok := recorded[l.Name]
		if !ok {
			drifted++
			t.Logf("drift: %s is installed but not recorded; re-run make connectors/generate", l.Name)
			continue
		}
		checked++
		have := map[string]bool{}
		for _, fl := range r.Flags {
			have[fl.Long] = true
		}
		for _, fl := range l.Flags {
			if !have[fl.Long] {
				drifted++
				t.Logf("drift: %s --%s is declared locally but not recorded; re-run make connectors/generate",
					l.Name, fl.Long)
			}
		}
	}
	// Counted, because a drift report that says nothing is indistinguishable
	// from one that had nothing to compare against.
	t.Logf("%d installed connectors compared against the recorded metadata, %d drifts", checked, drifted)
}

// The recorded environment routes used to be cross-checked against a registry
// in delivery.go: the extractor read them out of provider source, the registry
// was established by running the provider binaries, and where both spoke about
// one flag they had to agree.
//
// The registry is gone -- the launcher asks the provider rather than listing
// what it reads -- so there is nothing left to disagree with. The extracted
// routes stay: they are what the ambient credential readouts describe, and
// "which variable would this provider have used if you had supplied nothing" is
// still a useful thing for a form to say. What they no longer decide is where a
// credential the user typed goes.

// A route the extraction found for a flag no connector declares would be a
// field the launcher could never offer, so the artifact must not carry one.
func TestRecordedEnvOnlyNamesDeclaredFlags(t *testing.T) {
	for _, s := range loadConnectorSnapshot(t) {
		declared := map[string]bool{}
		for _, fl := range s.Flags {
			declared[fl.Long] = true
		}
		for _, route := range s.Env {
			if !declared[route.Flag] {
				t.Errorf("%s records a route for --%s, which it does not declare", s.Name, route.Flag)
			}
			if len(route.Vars) == 0 {
				t.Errorf("%s --%s records a route with no variable in it", s.Name, route.Flag)
			}
		}
	}
}

// The sub-command words only mean something for a connector that takes an
// argument to put them in.
func TestRecordedPositionalFitsTheArgumentCount(t *testing.T) {
	vocabularies := 0
	for _, s := range loadConnectorSnapshot(t) {
		for _, p := range s.Positional {
			vocabularies++
			if s.MaxArgs == 0 {
				t.Errorf("%s records a vocabulary for argument %d but declares it takes no arguments",
					s.Name, p.Index)
			}
			if p.Index >= int(s.MaxArgs) {
				t.Errorf("%s records a vocabulary for argument %d but takes at most %d",
					s.Name, p.Index, s.MaxArgs)
			}
			if len(p.Values) == 0 {
				t.Errorf("%s records an empty vocabulary for argument %d", s.Name, p.Index)
			}
		}
	}
	if vocabularies == 0 {
		t.Error("no connector records a sub-command vocabulary, so this checked nothing")
	}
}

// github is the connector the whole extraction was justified by: its org, user
// and repo live only in its help prose and in its own ParseCLI, and getting
// them wrong is the trap delivery.go already names.
func TestGithubVocabularyIsRecorded(t *testing.T) {
	snap, ok := snapshotByName(t)["github"]
	if !ok {
		t.Fatal("github is not in the recorded metadata")
	}
	if len(snap.Positional) != 1 {
		t.Fatalf("github records %d vocabularies, want 1: %+v", len(snap.Positional), snap.Positional)
	}
	got := strings.Join(snap.Positional[0].Values, "|")
	if got != "org|user|repo" {
		t.Errorf("github's sub-command words are %q, want %q", got, "org|user|repo")
	}
}
