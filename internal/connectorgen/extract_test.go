// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connectorgen

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.mondoo.com/mql/providers-sdk/v1/plugin"
)

// The extractor is checked against a stand-in provider under testdata rather
// than against a real mql checkout, because CI has no mql checkout and a test
// that skips itself there would be protecting nothing. The fixture holds one
// instance of each shape the real tree uses, and each assertion below names the
// shape it is about.

const fixtureRoot = "testdata/repo"

func extractFixture(t *testing.T) *Artifact {
	t.Helper()
	art, err := Extract([]Root{{Name: "demo", Path: fixtureRoot}})
	if err != nil {
		t.Fatalf("extracting the fixture: %v", err)
	}
	return art
}

func connectorNamed(t *testing.T, art *Artifact, name string) Connector {
	t.Helper()
	for _, c := range art.Connectors {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("the fixture yielded no connector named %q", name)
	return Connector{}
}

func envFor(c Connector, flag string) (FlagEnv, bool) {
	for _, fe := range c.Env {
		if fe.Flag == flag {
			return fe, true
		}
	}
	return FlagEnv{}, false
}

func gapKinds(gaps []Gap) map[string][]string {
	out := map[string][]string{}
	for _, g := range gaps {
		out[g.Kind] = append(out[g.Kind], g.Detail)
	}
	return out
}

// The config literal, including the parts that are not literals: a constant
// from a sibling package, an enum identifier, an OR of two options, and a Long
// assembled with fmt.Sprintf.
func TestExtractReadsTheConfigLiteral(t *testing.T) {
	c := connectorNamed(t, extractFixture(t), "demo")

	if c.Provider != "demo" || c.Use != "demo" || c.Short != "a demonstration target" {
		t.Errorf("connector header wrong: %+v", c)
	}
	if c.MinArgs != 2 || c.MaxArgs != 2 {
		t.Errorf("argument counts: got %d..%d, want 2..2", c.MinArgs, c.MaxArgs)
	}
	if len(c.Aliases) != 1 || c.Aliases[0] != "dem" {
		t.Errorf("aliases: got %v, want [dem]", c.Aliases)
	}
	// Discovery entries are constants in the connection package, so an empty
	// list here means the cross-package resolution stopped working.
	if strings.Join(c.Discovery, ",") != "all,units" {
		t.Errorf("discovery: got %v, want [all units]", c.Discovery)
	}
	// The Long is a fmt.Sprintf whose argument is the variable name. Resolving
	// the call is what keeps the sentence a user reads out of the blanks.
	if !strings.Contains(c.Long, "DEMO_TOKEN") {
		t.Errorf("Long did not resolve its Sprintf arguments: %q", c.Long)
	}

	byName := map[string]Flag{}
	for _, fl := range c.Flags {
		byName[fl.Long] = fl
	}
	if len(byName) != 6 {
		t.Fatalf("got %d flags, want 6: %+v", len(byName), c.Flags)
	}
	if got := byName["token"].Type; got != int32(plugin.FlagType_String) {
		t.Errorf("--token type: got %d, want %d", got, plugin.FlagType_String)
	}
	wantOption := int32(plugin.FlagOption_Password | plugin.FlagOption_Required)
	if got := byName["password"].Option; got != wantOption {
		t.Errorf("--password option: got %d, want %d (the OR of two identifiers)", got, wantOption)
	}
	if got := byName["region"].Config; got != "-" {
		t.Errorf("--region ConfigEntry: got %q, want %q", got, "-")
	}
	if got := byName["legacy"].Option; got != int32(plugin.FlagOption_Hidden) {
		t.Errorf("--legacy option: got %d, want Hidden", got)
	}

	// Sorted by name, because the checked-in artifact's diff has to be about
	// what changed rather than how the literal was ordered.
	for i := 1; i < len(c.Flags); i++ {
		if c.Flags[i-1].Long > c.Flags[i].Long {
			t.Fatalf("flags are not sorted: %q before %q", c.Flags[i-1].Long, c.Flags[i].Long)
		}
	}
}

// The four ways a provider ties a flag to a variable, and the one way it must
// not be tied.
func TestExtractReadsEnvironmentRoutes(t *testing.T) {
	c := connectorNamed(t, extractFixture(t), "demo")

	cases := []struct {
		flag     string
		vars     []string
		composed bool
		via      string
		why      string
	}{
		{
			flag: "token", vars: []string{"DEMO_TOKEN"}, via: "parse-cli",
			why: "the longhand chain: flags[\"token\"] into x, x into token, DEMO_TOKEN into token",
		},
		{
			flag: "region", vars: []string{"DEMO_REGION", "DEMO_REGION_ID"}, via: "parse-cli",
			why: "the variadic accessor, whose variables are a fallback list rather than a composition",
		},
		{
			flag: "password", vars: []string{"DEMO_PASSWORD"}, via: "parse-cli",
			why: "a flag-only accessor plus a variable returned by another function",
		},
		{
			flag: "organization", vars: []string{"DEMO_ORG_NAME", "DEMO_BASE_URL"}, composed: true, via: "parse-cli",
			why: "two variables joined by a \".\", so setting only the first will not work",
		},
		{
			flag: "endpoint", vars: []string{"DEMO_ENDPOINT"}, via: "connection",
			why: "read at connect time out of the options map, not while parsing",
		},
	}

	for _, tc := range cases {
		fe, ok := envFor(c, tc.flag)
		if !ok {
			t.Errorf("--%s has no environment route; %s", tc.flag, tc.why)
			continue
		}
		if strings.Join(fe.Vars, ",") != strings.Join(tc.vars, ",") {
			t.Errorf("--%s travels in %v, want %v (%s)", tc.flag, fe.Vars, tc.vars, tc.why)
		}
		if fe.Composed != tc.composed {
			t.Errorf("--%s composed=%v, want %v (%s)", tc.flag, fe.Composed, tc.composed, tc.why)
		}
		if fe.Via != tc.via {
			t.Errorf("--%s via=%q, want %q", tc.flag, fe.Via, tc.via)
		}
		if !fe.Declared {
			t.Errorf("--%s is declared by the connector but was not marked so", tc.flag)
		}
	}

	if len(c.Env) != len(cases) {
		t.Errorf("got %d routes, want %d: %+v", len(c.Env), len(cases), c.Env)
	}
}

// The two ways a wrong route gets invented, both of which the fixture baits.
func TestExtractInventsNoRoute(t *testing.T) {
	c := connectorNamed(t, extractFixture(t), "demo")

	// ParseCLI binds `x` twice, to --token and to --organization. A walk that
	// keyed variables by name would hand --token the organization's variables.
	if fe, ok := envFor(c, "token"); ok {
		for _, v := range fe.Vars {
			if strings.HasPrefix(v, "DEMO_ORG") || v == "DEMO_BASE_URL" {
				t.Errorf("--token was given %s, which belongs to --organization: "+
					"the two `x` bindings were not kept apart", v)
			}
		}
	}

	// newClient(region, password) combines two values. Neither may pick up the
	// other's variable through the client.
	if fe, ok := envFor(c, "region"); ok {
		for _, v := range fe.Vars {
			if v == "DEMO_PASSWORD" {
				t.Error("--region was given DEMO_PASSWORD: a value built from two others was treated as one of them")
			}
		}
	}

	// demolite declares only --endpoint, so nothing else may reach it.
	lite := connectorNamed(t, extractFixture(t), "demolite")
	for _, fe := range lite.Env {
		if fe.Flag != "endpoint" {
			t.Errorf("demolite was given a route for --%s, a flag it does not declare", fe.Flag)
		}
	}
}

// The sub-command vocabulary, which exists only as case labels.
func TestExtractReadsTheSubCommandVocabulary(t *testing.T) {
	art := extractFixture(t)
	c := connectorNamed(t, art, "demo")

	if len(c.Positional) != 1 {
		t.Fatalf("got %d positional entries, want 1: %+v", len(c.Positional), c.Positional)
	}
	p := c.Positional[0]
	if p.Index != 0 {
		t.Errorf("vocabulary is for argument %d, want 0", p.Index)
	}
	if strings.Join(p.Values, "|") != "unit|group" {
		t.Errorf("vocabulary: got %v, want [unit group]", p.Values)
	}
}

// What the walk could not settle. Each of these is a claim about the fixture,
// so a gap that stops being raised is a change in behaviour rather than an
// improvement in the source.
func TestExtractReportsItsGaps(t *testing.T) {
	art := extractFixture(t)
	kinds := gapKinds(art.Gaps)

	// The connection package reads DEMO_TAILNET for an option that is not a
	// flag, so no field could ever carry it.
	outside := strings.Join(kinds[GapEnvOutsideParseCLI], " ")
	if !strings.Contains(outside, "DEMO_TAILNET") {
		t.Errorf("DEMO_TAILNET is read for an option no connector declares and was not reported: %q", outside)
	}

	// providers/shipped holds a built provider and no source.
	absent := strings.Join(kinds[GapSourceAbsent], " ")
	if len(kinds[GapSourceAbsent]) == 0 {
		t.Error("a provider directory with a dist/ and no config was not reported as source-absent")
	} else if !strings.Contains(absent, "no config package") {
		t.Errorf("source-absent gap says something unexpected: %q", absent)
	}
	found := false
	for _, g := range art.Gaps {
		if g.Kind == GapSourceAbsent && g.Provider == "shipped" {
			found = true
		}
	}
	if !found {
		t.Error("the source-absent gap does not name the provider it is about")
	}
}

// A gap that belongs to a connector is repeated on it, so reading one connector
// shows what is missing from it.
func TestConnectorGapsAreRepeatedOnTheConnector(t *testing.T) {
	art := extractFixture(t)
	for _, c := range art.Connectors {
		for _, g := range c.Gaps {
			if g.Provider != "" || g.Connector != "" {
				t.Errorf("%s repeats a gap that still names its subject: %+v", c.Name, g)
			}
			if g.Kind == "" || g.Detail == "" {
				t.Errorf("%s has a gap with no kind or no detail: %+v", c.Name, g)
			}
		}
	}
}

// Nothing to read is an error, not an empty artifact: the artifact is checked
// in, and a run that produced an empty one would replace a complete file with
// nothing.
func TestExtractRefusesATreeWithNoProviders(t *testing.T) {
	if _, err := Extract(nil); err == nil {
		t.Error("extracting from no root at all succeeded")
	}
	if _, err := Extract([]Root{{Name: "empty", Path: t.TempDir()}}); err == nil {
		t.Error("extracting from a directory with no providers/ succeeded")
	}
}

// A regeneration never loses a connector: what the source no longer covers is
// kept, marked, and reported.
func TestCarryForwardKeepsWhatTheSourceDropped(t *testing.T) {
	art := extractFixture(t)

	previous := Artifact{
		Schema:     SchemaVersion,
		Connectors: append([]Connector{{Name: "gone", Provider: "gone", MaxArgs: 1}}, art.Connectors...),
	}
	data, err := json.MarshalIndent(previous, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "connectors.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}

	if err := CarryForward(art, path); err != nil {
		t.Fatalf("carrying forward: %v", err)
	}

	c := connectorNamed(t, art, "gone")
	if !c.CarriedForward {
		t.Error("a carried connector was not marked as one, so nothing in the diff says its metadata is stale")
	}
	kinds := gapKinds(art.Gaps)
	if len(kinds[GapCarriedForward]) != 1 {
		t.Errorf("got %d carried-forward gaps, want 1", len(kinds[GapCarriedForward]))
	}
	for i := 1; i < len(art.Connectors); i++ {
		if art.Connectors[i-1].Name > art.Connectors[i].Name {
			t.Fatalf("carrying forward broke the ordering: %q before %q",
				art.Connectors[i-1].Name, art.Connectors[i].Name)
		}
	}

	// A missing previous artifact is the first run, not a failure.
	if err := CarryForward(art, filepath.Join(t.TempDir(), "absent.json")); err != nil {
		t.Errorf("carrying forward from a file that does not exist: %v", err)
	}
}

// The bare array the launcher's snapshot used before this tool existed, which
// the first regeneration has to read.
func TestCarryForwardReadsTheOldFormat(t *testing.T) {
	art := extractFixture(t)
	data := []byte(`[{"name":"legacy","provider":"legacy","max_args":1}]`)
	path := filepath.Join(t.TempDir(), "connectors.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := CarryForward(art, path); err != nil {
		t.Fatalf("carrying forward from the old format: %v", err)
	}
	if c := connectorNamed(t, art, "legacy"); !c.CarriedForward {
		t.Error("an entry from the old format was not carried forward")
	}
}

// The report is the deliverable, so it has to name every gap it counted.
func TestReportNamesEveryGap(t *testing.T) {
	art := extractFixture(t)
	var buf bytes.Buffer
	WriteReport(&buf, art)
	out := buf.String()

	s := Summarise(art)
	if s.Connectors != len(art.Connectors) {
		t.Errorf("the summary counted %d connectors, the artifact has %d", s.Connectors, len(art.Connectors))
	}
	for kind := range s.GapsByKind {
		if !strings.Contains(out, kind) {
			t.Errorf("the report counts %q but never lists it", kind)
		}
	}
	for _, g := range art.Gaps {
		if !strings.Contains(out, g.Detail) {
			t.Errorf("the report omits a gap: %s", g.Detail)
		}
	}
}

// The artifact is checked in, so its encoding has to be stable and readable.
func TestWriteJSONIsStable(t *testing.T) {
	art := extractFixture(t)
	var first, second bytes.Buffer
	if err := WriteJSON(&first, art); err != nil {
		t.Fatal(err)
	}
	if err := WriteJSON(&second, art); err != nil {
		t.Fatal(err)
	}
	if first.String() != second.String() {
		t.Error("two encodings of one artifact differ, so a regeneration diff would be noise")
	}
	if !bytes.HasSuffix(first.Bytes(), []byte("\n")) {
		t.Error("the artifact does not end in a newline")
	}
}
