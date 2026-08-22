// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	"sort"
	"strings"
	"testing"
)

// How far the launcher reaches, asserted so that the numbers cannot quietly
// stop being true.
//
// Every other test in this package checks one connector, or one file's worth of
// them, against something its author knew. This checks the opposite direction:
// that no connector fell through the gaps between those files. A connector
// nobody was assigned looks exactly like a connector somebody decided not to
// curate, and the only difference is whether a reason was written down.
//
// # The denominator moves, so nothing here counts
//
// The catalog is the union of the static DefaultProviders list compiled into
// the binary and whatever providers are installed locally: 88 providers and 94
// connectors on the machine this was written on, 103 names in the union, and a
// different number on a colleague's laptop. Worse, CI points PROVIDERS_PATH at
// an empty directory, so BuildCatalog() degenerates to the flagless static
// list.
//
// A test asserting a total would therefore fail on one machine and pass
// vacuously on the other -- and the vacuous pass is the dangerous half, because
// green is exactly what a reach test is supposed to mean. So every assertion
// below is a property of one connector, evaluated over whatever the catalog
// happens to hold, and anything that could not be checked is counted and named
// in the log. A green run that skipped ninety connectors says so.
//
// The snapshot is what keeps that number small. internal/connectors/connectors.json is
// checked in, so the form-shape checks run against real metadata even where no
// provider is installed; only a connector that is in neither the snapshot nor
// the local catalog is genuinely uncheckable.

// reachEntry is one connector to check, from whichever source knew about it.
type reachEntry struct {
	name string
	// c carries flags when the connector was installed here or is in the
	// snapshot; otherwise it is the flagless static entry.
	c Connector
	// detailed is true when c came from real metadata rather than from the
	// static list, and therefore whether the form checks can say anything.
	detailed bool
}

// reachCatalog is every connector the launcher could offer, from the union of
// the built catalog and the recorded snapshot, preferring whichever has real
// metadata.
//
// The union rather than either alone: the catalog is the only place the
// static-list connectors appear (db2 and artifactory are there and nowhere
// else), and the snapshot is the only place any flags appear when no provider
// is installed.
func reachCatalog(t *testing.T) []reachEntry {
	t.Helper()
	byName := map[string]reachEntry{}

	for _, s := range loadConnectorSnapshot(t) {
		byName[s.Name] = reachEntry{name: s.Name, c: s.connector(), detailed: true}
	}
	for _, c := range BuildCatalog() {
		if prev, ok := byName[c.Name]; ok && prev.detailed && !c.DeclaresMetadata() {
			// The snapshot knows more about it than this machine does.
			continue
		}
		byName[c.Name] = reachEntry{name: c.Name, c: c, detailed: c.DeclaresMetadata()}
	}

	out := make([]reachEntry, 0, len(byName))
	for _, e := range byName {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out
}

// Every connector is either curated or recorded as deliberately not, and the
// launcher has something to show for it either way.
//
// forms_misc_test.go used to run this same four-way check over the twelve
// connectors one contributor had been assigned, which is the shape a per-file
// copy of a catalog-wide property always takes: it proves the property for the
// names its author happened to write down and says nothing about the rest. The
// four cases below are that check, over every connector the launcher can offer.
func TestEveryConnectorIsCuratedOrExcused(t *testing.T) {
	entries := reachCatalog(t)
	if len(entries) == 0 {
		t.Fatal("the catalog and the snapshot are both empty; nothing was checked")
	}

	var curated, excused, uncheckable []string
	for _, e := range entries {
		_, hasSpec := formSpecs[e.name]
		reason, excluded := uncuratedReason(e.name)

		switch {
		case hasSpec && excluded:
			t.Errorf("%s has a spec and is also recorded as deliberately uncurated; "+
				"one of the two is wrong", e.name)
		case hasSpec:
			curated = append(curated, e.name)
		case excluded:
			if strings.TrimSpace(reason) == "" {
				t.Errorf("%s is excluded with an empty reason, which says nothing "+
					"a future reader can act on", e.name)
			}
			excused = append(excused, e.name)
		default:
			t.Errorf("%s has no FormSpec and no recorded reason for not having one. "+
				"Register a spec in the cli/launcher/forms file for its catalog category, or call "+
				"registerUncurated with what stops one from being written.", e.name)
		}

		if !e.detailed {
			uncheckable = append(uncheckable, e.name)
		}
	}

	t.Logf("%d connectors: %d curated, %d deliberately not, %d with no metadata "+
		"here to check the form against (%s)",
		len(entries), len(curated), len(excused), len(uncheckable),
		strings.Join(uncheckable, ", "))

	// The exclusions are a decision, so removing one has to be a decision too.
	// Naming them here rather than counting them is the point: a fourth
	// exclusion appearing silently is exactly what this is for.
	wantExcused := map[string]bool{"device": true, "ipinfo": true, "artifactory": true}
	got := map[string]bool{}
	for _, name := range excused {
		got[name] = true
	}
	for name := range wantExcused {
		if !got[name] {
			t.Errorf("%s is no longer recorded as deliberately uncurated. If it grew "+
				"a form, drop it from this list; if its registerUncurated call was lost, "+
				"that is the bug.", name)
		}
	}
	for name := range got {
		if !wantExcused[name] {
			t.Errorf("%s was excluded from curation without being named here. "+
				"An exclusion is a decision -- add it with the reason, or write the spec.",
				name)
		}
	}
}

// Opening any connector puts something on the screen: fields to fill in, or a
// sentence saying why there are none. Never a blank pane.
//
// Two connectors reached this the hard way and from opposite directions.
// `device` declares nine flags and every one of them is Hidden or Deprecated,
// so genericFields builds nothing while HasFormData() -- which counted
// len(c.Flags) -- said there was a form. `mondoo` declares no flags at all and
// MaxArgs=4 that its ParseCLI discards, so its spec suppresses the argument box
// and leaves the same emptiness behind. Both rendered a pane with a scan button
// and nothing above it.
//
// The assertion is on the rendered plan rather than on the predicate, because
// the predicate is not the thing that was wrong -- what was wrong is what the
// user saw, and there is more than one way to arrive at it.
func TestEveryConnectorPaneSaysSomething(t *testing.T) {
	checked, skipped := 0, 0
	for _, e := range reachCatalog(t) {
		if !e.detailed {
			skipped++
			continue
		}
		checked++

		m := newTestModel()
		m.list.filtered = []Connector{e.c}
		m.list.cursor = 0
		m.detail.form = newForm(e.c)

		var fields, text int
		for _, item := range m.detailPlan() {
			switch item.kind {
			case diField, diMore:
				fields++
			case diText:
				text++
			}
		}
		if fields == 0 && text < 2 {
			// One diText is always there: the connector's own one-line
			// description. A pane with nothing to fill in owes the reader a
			// second line saying so.
			t.Errorf("%s opens on a pane with no fields and no explanation for it, "+
				"which is a scan button under a heading", e.name)
		}
	}
	t.Logf("checked %d connectors, skipped %d with no metadata available here", checked, skipped)
	if checked == 0 {
		t.Fatal("no connector carried metadata, so this proved nothing; " +
			"internal/connectors/connectors.json should have supplied it")
	}
}

// Every field on every form is reachable and answerable.
//
// The two failures this catches are both ones the launcher cannot show you: a
// required field parked in OPTIONS sits behind the "more" fold, where the
// cursor focusFirstMissing leaves cannot be moved to; and a picker with no
// options and no source is a row no keystroke fills, because openModal declines
// to open an empty box and storeCursorField will not write into a multi-choice.
//
// Both were found per-file by the agents who curated those files. Running them
// over the whole catalog is what covers the connectors nobody curated, whose
// forms come from genericFields and were never looked at by anyone.
func TestEveryFieldOnEveryFormCanBeAnswered(t *testing.T) {
	checked, skipped := 0, 0
	for _, e := range reachCatalog(t) {
		if !e.detailed {
			skipped++
			continue
		}
		checked++
		f := newForm(e.c)
		for i := range f.Fields() {
			fd := f.Fields()[i]
			// Required-in-OPTIONS is checked over every field rather than only
			// the visible ones, which is the scope forms_misc_test.go used for
			// its own copy of this check and the stricter of the two: a field
			// that is required and folded away is a form that cannot be
			// completed the moment whatever hides it stops hiding it.
			if fd.Required && fd.Section == sectionOptions {
				t.Errorf("%s: %q is required but sits behind the OPTIONS fold, "+
					"where the cursor cannot reach it", e.name, fd.Label)
			}
			if !f.Visible(i) {
				// An empty picker nobody can see is not a row a keystroke has
				// to reach, so the check below is the visible fields only.
				continue
			}
			if fd.Kind != fieldChoice && fd.Kind != fieldMultiChoice {
				continue
			}
			if len(fd.Options) == 0 && fd.Source() == "" && fd.LiveSource == "" &&
				fd.SourceBy == nil && fd.LiveSourceBy == nil {
				t.Errorf("%s: %q is a picker with nothing to pick and no source, "+
					"so no keystroke can fill it", e.name, fd.Label)
			}
		}
	}
	t.Logf("checked %d connectors, skipped %d with no metadata available here", checked, skipped)
	if checked == 0 {
		t.Fatal("no connector carried metadata, so this proved nothing")
	}
}

// A credential the form lets you type has to reach the provider, and must not
// reach a command line.
//
// This used to have a list attached: seven connector/flag pairs the launcher
// refused, each with the reason it could not carry them. Three declared
// ConfigEntry "-" so no derived variable reached them, two declared a prompt
// flag the CLI does not act on, and two derived a name cnspec reads for its own
// service account. Every one of those was a fact about how a *flag value* gets
// from this process into a child, and none of them is true any more: the value
// goes into a ParseCLI request over the plugin's gRPC connection, which is the
// same place a flag would have ended up.
//
// The list is gone rather than emptied. An empty exception list invites the
// next refusal to be added to it; there is no mechanism left that could produce
// one from the form alone.
//
// # Why this also drives the launch
//
// It used to stop at "routes as an inventory", and forms_saas_c_test.go carried
// a second test that went the rest of the way -- planning the launch and asking
// a stand-in provider what it was handed -- over the ten connectors one
// contributor had curated. Those are the same claim at two depths, and the
// deeper one was the one scoped to a tenth of the catalog.
//
// So the deeper one is here and covers all of it. The route is not the promise;
// the promise is that the value arrives, which is what assertCredentialReaches-
// TheProvider checks and what the delivery check alone cannot: an inventory
// that routes perfectly and carries the credential under a key the provider
// never reads satisfies deliveryFor and fails the user.
func TestEveryTypeableCredentialReachesTheProvider(t *testing.T) {
	const sentinel = "must-never-reach-argv"
	checked, skipped := 0, 0

	for _, e := range reachCatalog(t) {
		if !e.detailed {
			skipped++
			continue
		}
		f := newForm(e.c)
		for i := range f.Fields() {
			fd := &f.Fields()[i]
			// Visible only, and that is a real limit rather than an oversight:
			// a secret behind a selector -- atlassian's four product tokens,
			// databricks' workspace token -- is not typeable until the selector
			// is answered, and answering it is a decision this loop has no way
			// to make. Those are covered by name, with the selector value
			// beside them, in the category file for their connector.
			if !fd.Secret || fd.Flag == "" || !f.Visible(i) {
				continue
			}
			checked++

			// One secret at a time: several providers pick a different
			// authentication path when they are shown a second one, and what
			// matters here is that each has a way through at all.
			probe := newForm(e.c)
			probe.Fields()[i].SetValue(sentinel)

			if got := deliveryFor(probe); got != deliverInventory {
				t.Errorf("%s --%s is typeable but routes as %v, so the credential "+
					"would travel on the command line", e.name, fd.Flag, got)
				continue
			}
			if args := strings.Join(probe.Args(), " "); strings.Contains(args, sentinel) {
				t.Errorf("%s --%s reached argv: %s", e.name, fd.Flag, args)
			}
			t.Run(e.name+"/"+fd.Flag, func(t *testing.T) {
				assertCredentialReachesTheProvider(t, e.c, probe, fd.Flag, sentinel)
			})
		}
	}
	t.Logf("checked %d typeable credentials over the catalog; skipped %d connectors "+
		"with no metadata available here", checked, skipped)
	if checked == 0 {
		t.Fatal("no credential field was built, so this proved nothing")
	}
}
