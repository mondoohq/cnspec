// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	"flag"
	"reflect"
	"sort"
	"strconv"
	"testing"
)

// updateSnapshot rewrites internal/connectors/connectors.json from the installed provider
// set. It lives here rather than in snapshot_test.go only because a package can
// register a flag once.
var updateSnapshot = flag.Bool("update", false, "rewrite internal/connectors/connectors.json from the installed providers")

// The gate over the curated overlays.
//
// applySpec drops any flag the connector does not declare, which is deliberate:
// an overlay that goes stale degrades to the generic screen rather than
// emitting a flag that no longer exists. The cost of that graceful degradation
// is that a typo is completely silent -- the field just never appears, with
// nothing anywhere saying why -- and inventing a plausible flag is the
// documented failure mode on this branch, caught last time only by a human
// reading it.
//
// This is checked against the recorded snapshot rather than the machine,
// because the machine has no providers on it in CI. See snapshot_test.go.
//
// A flag is checked against every connector that shares the spec naming it, not
// against each one separately, and that distinction is the whole difficulty of
// this test. databaseSpec is registered for fifteen connectors and is
// deliberately the union of the family's vocabulary -- postgresdb has no --sid,
// oracledb has no --auth-db -- so a per-connector rule would report a hundred
// failures for a spec that is doing exactly what its comment says. A flag that
// *no* connector sharing the spec declares is a different thing entirely: it
// cannot ever appear on a screen, which is what a typo looks like. For a spec
// registered for one connector -- every spec written from here on -- the two
// rules are the same rule.
func TestEverySpecNamesRealFlags(t *testing.T) {
	byName := snapshotByName(t)
	var missing []string
	checked, groups := 0, 0

	for _, group := range specGroups() {
		// The flags declared by any connector in the group, plus --discover,
		// which the CLI synthesizes for every connector with discovery targets
		// and which therefore never appears in Flags.
		declared := map[string]bool{}
		var known []string
		for _, name := range group.connectors {
			snap, ok := byName[name]
			if !ok {
				continue
			}
			known = append(known, name)
			for _, fl := range snap.Flags {
				declared[fl.Long] = true
			}
			if len(snap.Discovery) > 0 {
				declared["discover"] = true
			}
		}
		if len(known) == 0 {
			// No connector in the group is in the snapshot, so there is nothing
			// to check it against. Worth seeing, not worth failing: a provider
			// may simply not have been installed when the snapshot was taken.
			t.Logf("%s: no snapshot entry, so its flags were not checked",
				group.connectors[0])
			continue
		}
		groups++
		checked += len(known)

		label := group.connectors[0]
		if len(group.connectors) > 1 {
			label = group.connectors[0] + " (shared by " + strconv.Itoa(len(group.connectors)) + " connectors)"
		}
		report := func(section, flag string) {
			if flag == "" || declared[flag] {
				return
			}
			missing = append(missing, label+" "+section+" names --"+flag+
				", which no connector using this spec declares")
		}

		spec := group.spec
		for _, f := range spec.Target {
			report("Target", f)
		}
		for _, f := range spec.Credential {
			report("Credential", f)
		}
		for _, f := range spec.Secret {
			report("Secret", f)
		}
		for _, f := range spec.NotSecret {
			report("NotSecret", f)
		}
		for _, f := range spec.Hide {
			report("Hide", f)
		}
		for f := range spec.Labels {
			report("Labels", f)
		}
		for f := range spec.Sources {
			report("Sources", f)
		}
		for f := range spec.LiveSources {
			report("LiveSources", f)
		}
		for f := range spec.Choices {
			report("Choices", f)
		}
		for f := range spec.ShowFlagsIf {
			report("ShowFlagsIf", f)
		}
	}

	sort.Strings(missing)
	for _, m := range missing {
		t.Error(m)
	}
	if checked == 0 {
		t.Fatal("no spec was checked against the snapshot, so this test proves nothing")
	}
	t.Logf("checked %d of %d registered connectors, in %d spec groups",
		checked, len(formSpecs), groups)
}

// specGroup is one spec and every connector registered with it.
type specGroup struct {
	spec       FormSpec
	connectors []string
}

// specGroups collects the registered specs, folding connectors that share an
// identical spec into one group. Sharing is by value because that is all the
// registry keeps -- registerSpec takes a FormSpec, not a pointer to one.
func specGroups() []specGroup {
	names := make([]string, 0, len(formSpecs))
	for name := range formSpecs {
		names = append(names, name)
	}
	sort.Strings(names)

	var out []specGroup
	for _, name := range names {
		spec := formSpecs[name]
		placed := false
		for i := range out {
			if reflect.DeepEqual(out[i].spec, spec) {
				out[i].connectors = append(out[i].connectors, name)
				placed = true
				break
			}
		}
		if !placed {
			out = append(out, specGroup{spec: spec, connectors: []string{name}})
		}
	}
	return out
}

// A connector that requires positional arguments needs somewhere to put them.
// A spec that replaces the derived slots with fewer than the connector demands
// produces a form that cannot assemble a valid command, and says nothing about
// why.
func TestEverySpecHasRoomForItsArguments(t *testing.T) {
	byName := snapshotByName(t)
	for name, spec := range formSpecs {
		snap, ok := byName[name]
		if !ok || len(spec.Positional) == 0 {
			continue
		}
		if uint(len(spec.Positional)) < snap.MinArgs {
			t.Errorf("%s declares %d positional fields but the connector requires %d",
				name, len(spec.Positional), snap.MinArgs)
		}
	}
}

// Two registrations for one connector mean two people each believed they owned
// it, and init order decides which one the user gets. registerSpec keeps the
// first and records the collision here rather than letting the second win in
// silence.
func TestNoConnectorHasTwoSpecs(t *testing.T) {
	for _, name := range duplicateSpecs {
		t.Errorf("%s has more than one registered spec", name)
	}
}

// Every environment contributor has to name a connector, a field and something
// to do, or it silently contributes nothing and the target it was written for
// is scanned as though it had never been chosen.
func TestEveryEnvSpecIsComplete(t *testing.T) {
	for connector, specs := range envSpecs {
		for _, s := range specs {
			if s.Connector != connector {
				t.Errorf("%s: registered under %q but declares connector %q",
					connector, connector, s.Connector)
			}
			if s.Field == "" {
				t.Errorf("%s: an EnvSpec with no Field matches nothing", connector)
			}
			if s.Apply == nil {
				t.Errorf("%s: an EnvSpec with no Apply contributes nothing", connector)
			}
		}
	}
}

// A positional declares either a value table or a value picker, never both.
//
// The two are alternative answers to "what does this field's display mean on a
// command line", and field.emit holds one of them. A picker's answer arrives
// with the picker -- attachSource takes them together, because a source
// assigned without its mapping emits an annotated AWS profile verbatim -- so a
// selector that switched pickers would drop a table declared beside it. Nothing
// declares both today; this is what keeps that true, since the alternative is a
// spec whose Emit entries stop applying the moment the selector above it moves.
func TestNoPositionalDeclaresBothATableAndAPicker(t *testing.T) {
	for name, spec := range formSpecs {
		for _, p := range spec.Positional {
			if p.Emit == nil {
				continue
			}
			if p.Source != "" || p.SourceBy != nil {
				t.Errorf("%s: %q declares an Emit table and a value picker; "+
					"a picker brings its own mapping and would replace the table",
					name, p.Label)
			}
		}
	}
}

// Every registered spec has a snapshot entry, so TestEverySpecNamesRealFlags
// above skipped none of them.
//
// That gate logs and continues when a spec's connector is absent from the
// recorded artifact, which is the right behaviour -- a provider may simply not
// have been installed when the artifact was generated -- and is also the shape
// that turns a green run into a claim about nothing. A spec whose connector is
// missing has its Target, Credential, Secret, Hide and picker flags checked by
// nothing at all, and looks identical to one that passed.
//
// forms_misc_test.go used to say this for the twelve connectors one contributor
// had been assigned, and forms_saas_c_test.go worked around it for six others
// by re-checking their flags against a hand-written fixture. Both were
// per-file answers to a question about the whole registry. This is the
// question: how many of the registered specs did the gate actually see.
func TestEverySpecIsCoveredByTheSnapshotGate(t *testing.T) {
	byName := snapshotByName(t)

	var unchecked []string
	checked := 0
	for name := range formSpecs {
		if _, ok := byName[name]; !ok {
			unchecked = append(unchecked, name)
			continue
		}
		checked++
	}
	sort.Strings(unchecked)

	if checked == 0 {
		t.Fatal("no registered spec has a snapshot entry, so the flag gate above " +
			"checked nothing and passed")
	}
	for _, name := range unchecked {
		t.Errorf("%s has a spec but no entry in %s, so nothing checks the flags it "+
			"names. Refresh the artifact with `make connectors/generate`, or say here "+
			"why it is absent", name, connectorSnapshotPath)
	}
	t.Logf("%d of %d registered specs are covered by the snapshot gate",
		checked, len(formSpecs))
}
