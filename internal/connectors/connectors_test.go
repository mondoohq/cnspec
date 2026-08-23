// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connectors

import (
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"testing"

	"go.mondoo.com/cnspec/internal/connectorgen"
)

// TestArtifactParses is what an embedded JSON buys instead of generated Go
// source: the compile-time guarantee that the data is well formed, moved to a
// test that runs in CI where no provider is installed.
func TestArtifactParses(t *testing.T) {
	art, err := Load()
	if err != nil {
		t.Fatalf("the embedded artifact does not parse: %v", err)
	}
	if art.Schema != connectorgen.SchemaVersion {
		t.Errorf("artifact schema %d, generator writes %d -- regenerate it with `make connectors/generate MQL=<path>`",
			art.Schema, connectorgen.SchemaVersion)
	}
	if len(art.Connectors) < 50 {
		t.Fatalf("only %d connectors recorded; the artifact looks truncated", len(art.Connectors))
	}
	for _, c := range art.Connectors {
		if c.Name == "" || c.Provider == "" {
			t.Errorf("connector with no name or provider: %+v", c)
		}
	}
	t.Logf("%d connectors, %d with a positional vocabulary, %d with env routes",
		len(art.Connectors), countWith(art.Connectors, func(c Connector) bool { return len(c.Positional) > 0 }),
		countWith(art.Connectors, func(c Connector) bool { return len(c.Env) > 0 }))
}

func countWith(cs []Connector, pred func(Connector) bool) int {
	n := 0
	for _, c := range cs {
		if pred(c) {
			n++
		}
	}
	return n
}

// TestProjectionIsFaithful is the contract between this package and the
// generator that writes the file it embeds.
//
// The types here are a projection: they carry what a consumer reads and drop
// the provenance the generator records for its own report. A projection is only
// safe while everyone knows which fields it drops, so this asserts exactly
// that -- it fails when the generator grows a connector field this package does
// not carry and has not been told to ignore.
//
// The failure is deliberately not "add the field". It is "decide": a new
// generator field is either something production should read, in which case it
// belongs on Connector above, or something only the report wants, in which case
// it belongs in the ignore list beside a reason.
func TestProjectionIsFaithful(t *testing.T) {
	// Fields the generator records that a consumer deliberately does not read.
	ignored := map[string]string{
		"IsHidden": "the catalog decides what to offer from the installed provider set, not from a recording",
		"Maturity": "not consumed by any form; the launcher shows what a provider declares at runtime",
		"Gaps":     "a report of what the extraction could not determine, for a human, not an input to a form",
	}

	gen := reflect.TypeOf(connectorgen.Connector{})
	proj := reflect.TypeOf(Connector{})
	have := map[string]reflect.StructField{}
	for i := 0; i < proj.NumField(); i++ {
		have[proj.Field(i).Name] = proj.Field(i)
	}

	var missing []string
	for i := 0; i < gen.NumField(); i++ {
		f := gen.Field(i)
		if _, skip := ignored[f.Name]; skip {
			continue
		}
		got, ok := have[f.Name]
		if !ok {
			missing = append(missing, f.Name)
			continue
		}
		if tagName(got.Tag.Get("json")) != tagName(f.Tag.Get("json")) {
			t.Errorf("%s: json tag %q here, %q in the generator -- the projection would silently read nothing",
				f.Name, got.Tag.Get("json"), f.Tag.Get("json"))
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("connectorgen.Connector has fields this projection neither carries nor ignores: %v.\n"+
			"Either add them to connectors.Connector, or list them in `ignored` above with the reason a form does not need them.",
			missing)
	}
}

func tagName(tag string) string {
	name, _, _ := strings.Cut(tag, ",")
	return name
}

// TestRawIsTheParsedFile guards the one way Raw() and the typed accessors can
// disagree: a caller reading the bytes and a caller reading the projection have
// to be looking at the same connectors.
func TestRawIsTheParsedFile(t *testing.T) {
	var direct struct {
		Connectors []struct {
			Name string `json:"name"`
		} `json:"connectors"`
	}
	if err := json.Unmarshal(Raw(), &direct); err != nil {
		t.Fatal(err)
	}
	if len(direct.Connectors) != len(All()) {
		t.Fatalf("Raw() holds %d connectors, All() returns %d", len(direct.Connectors), len(All()))
	}
	for i, c := range direct.Connectors {
		if All()[i].Name != c.Name {
			t.Fatalf("connector %d: Raw() says %q, All() says %q", i, c.Name, All()[i].Name)
		}
	}
}
