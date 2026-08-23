// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Package connectors is the read side of the connector metadata artifact.
//
// internal/connectorgen writes connectors.json by walking an mql checkout; this
// package embeds that file and hands it to production code. The two halves are
// deliberately separate packages: the generator imports go/ast and go/parser to
// do its job, and nothing that ships in the cnspec binary should depend on a
// source-extraction tool to read its own data.
//
// # Why an embedded JSON rather than generated Go source
//
// The repo has a *_gen.go convention and it would have been the other obvious
// choice. It is not used here, for four reasons:
//
//   - Size. The artifact is ~190KB describing 106 connectors. As a Go literal
//     that is several thousand lines, which is larger than the hand-written
//     metadata this exists to replace, and every provider change would produce
//     a diff of that size for a human to scroll past.
//   - One representation. The JSON is already the contract. The generator
//     writes it, `make connectors/generate` produces it, and the launcher's
//     existing tests parse it. Emitting Go source as well would create a second
//     shape of the same facts and a second thing to keep in step.
//   - The build guarantee is unchanged either way. cnspec must compile with no
//     mql checkout present, and it does because the artifact is checked in --
//     which is true of an embedded file exactly as it is of generated source.
//   - The one thing generated source buys is that a malformed artifact fails at
//     compile time rather than at run time. That is bought here instead by
//     TestArtifactParses, which runs in CI with no providers installed. The
//     file is generated and checked in, so the only way it can be malformed is
//     a hand edit, and a hand edit is what that test is looking for.
//
// The types below are a projection of internal/connectorgen's Artifact rather
// than a copy of it: they carry the fields a consumer reads and drop the
// provenance the generator records for its own report. TestProjectionIsFaithful
// in this package is what keeps the projection honest when the generator's
// schema grows.
package connectors

import (
	_ "embed"
	"encoding/json"
	"sync"

	"github.com/cockroachdb/errors"
)

//go:embed connectors.json
var raw []byte

// Artifact is the generated file as a consumer reads it.
type Artifact struct {
	// Schema is connectorgen.SchemaVersion at the time of generation.
	Schema int `json:"schema"`
	// Connectors is the extraction, sorted by name then provider.
	Connectors []Connector `json:"connectors"`
}

// Connector is one (provider, connector) pair as the artifact records it.
//
// The first block is also obtainable from an installed provider. The second is
// what only the provider's Go source carries, and is the whole reason this file
// exists: neither fact survives into the shipped provider metadata.
type Connector struct {
	Name      string   `json:"name"`
	Provider  string   `json:"provider"`
	Use       string   `json:"use,omitempty"`
	Short     string   `json:"short,omitempty"`
	Long      string   `json:"long,omitempty"`
	Aliases   []string `json:"aliases,omitempty"`
	MinArgs   uint     `json:"min_args,omitempty"`
	MaxArgs   uint     `json:"max_args,omitempty"`
	Flags     []Flag   `json:"flags,omitempty"`
	Discovery []string `json:"discovery,omitempty"`

	// Env names the environment variables the provider reads a flag's value
	// from, in the order its source consults them.
	//
	// This is detection data, not delivery data. Knowing that a provider reads
	// OPENAI_API_KEY lets a form say "already set in your environment" instead
	// of drawing an empty box; it is not a licence to put a secret into a
	// variable and hand it to a child process. Providers resolve the inventory
	// first and the environment last, so the environment is the weakest route
	// available, not the strongest.
	Env []FlagEnv `json:"env,omitempty"`
	// Positional is the sub-command vocabulary -- the literal words the
	// provider compares a positional argument against, which is how `github
	// org <ORG>` differs from `github repo <ORG>/<REPO>`.
	Positional []Positional `json:"positional,omitempty"`
	// CarriedForward marks a connector this extraction did not re-derive; its
	// metadata is as old as whenever it last was.
	CarriedForward bool `json:"carried_forward,omitempty"`
}

// Flag mirrors plugin.Flag. Type and Option are the numeric enum values, so a
// consumer compares them against plugin.FlagType_String and
// plugin.FlagOption_Password directly.
type Flag struct {
	Long    string `json:"long"`
	Short   string `json:"short,omitempty"`
	Default string `json:"default,omitempty"`
	Type    int32  `json:"type,omitempty"`
	Option  int32  `json:"option,omitempty"`
	Desc    string `json:"desc,omitempty"`
	Config  string `json:"config,omitempty"`
}

// FlagEnv is one flag and the environment variables that back it.
type FlagEnv struct {
	Flag string   `json:"flag"`
	Vars []string `json:"vars"`
	// Via is "parse-cli" when the variable is consulted while parsing the
	// command line and "connection" when the connection package reads it at
	// connect time. A parse-cli variable yields to the flag; a connection one
	// often overrides it.
	Via string `json:"via,omitempty"`
	// Declared reports whether this connector declares a flag by that name. A
	// false is a finding, not noise: the provider reads a value the CLI has no
	// way to pass.
	Declared bool `json:"declared"`
	// Composed marks variables that are joined rather than chosen between, so
	// setting only the first will not work.
	Composed bool `json:"composed,omitempty"`
}

// Positional is the vocabulary for one positional argument.
type Positional struct {
	// Index is the argument position as the provider counts it: req.Args[0] is
	// index 0.
	Index int `json:"index"`
	// Values are the literal words compared against, in source order.
	Values []string `json:"values"`
}

// load parses the embedded artifact once.
//
// A parse failure here is a corrupt checked-in file rather than anything a user
// did, so it is carried rather than panicked on: the launcher degrades to the
// metadata the installed providers declare, which is the same screen it drew
// before this package existed.
var load = sync.OnceValues(func() (*Artifact, error) {
	var art Artifact
	if err := json.Unmarshal(raw, &art); err != nil {
		return nil, errors.Wrap(err, "cannot parse the embedded connector artifact")
	}
	if len(art.Connectors) == 0 {
		return nil, errors.New("the embedded connector artifact describes no connectors")
	}
	return &art, nil
})

var index = sync.OnceValue(func() map[string]Connector {
	art, err := load()
	if err != nil {
		return map[string]Connector{}
	}
	out := make(map[string]Connector, len(art.Connectors))
	for _, c := range art.Connectors {
		// First wins, matching the artifact's own sort: a connector name is
		// unique across providers today, and if that ever stops being true the
		// duplicate is a generator bug rather than something to resolve here.
		if _, taken := out[c.Name]; !taken {
			out[c.Name] = c
		}
	}
	return out
})

// Load returns the parsed artifact.
func Load() (*Artifact, error) { return load() }

// ByName returns the recorded metadata for one connector.
func ByName(name string) (Connector, bool) {
	c, ok := index()[name]
	return c, ok
}

// All returns every recorded connector, in the artifact's order.
func All() []Connector {
	art, err := load()
	if err != nil {
		return nil
	}
	return art.Connectors
}

// Raw returns the embedded bytes, for a consumer that needs a field this
// projection drops.
func Raw() []byte { return raw }
