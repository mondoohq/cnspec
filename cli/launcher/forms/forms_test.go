// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package forms

import (
	"os"
	"sort"
	"strings"
	"testing"

	"go.mondoo.com/cnspec/cli/launcher/catalog"
	"go.mondoo.com/cnspec/internal/connectors"
)

// categoryFile is where a connector's spec belongs, by the category the
// launcher files that connector under.
//
// This is the whole convention, written down once. Every entry in
// catalog.CategoryOrder has a file and no file is shared, so "where is the jamf
// form" has the same answer as "where does the launcher show jamf".
var categoryFile = map[string]string{
	catalog.CatHosts:     "forms_hosts.go",
	catalog.CatContainer: "forms_container.go",
	catalog.CatCloud:     "forms_cloud.go",
	catalog.CatIaC:       "forms_iac.go",
	catalog.CatIdentity:  "forms_identity.go",
	catalog.CatSaaS:      "forms_saas.go",
	catalog.CatNetwork:   "forms_network.go",
	catalog.CatDatabase:  "forms_database.go",
	catalog.CatAI:        "forms_ai.go",
	catalog.CatDev:       "forms_dev.go",
	catalog.CatOther:     "forms_other.go",
}

// A category the catalog groups connectors under, with nowhere here to put its
// specs, is how the taxonomy drifts back apart: the first spec for it lands in
// whichever file its author had open.
func TestEveryCatalogCategoryHasAFile(t *testing.T) {
	for _, cat := range catalog.CategoryOrder {
		if categoryFile[cat] == "" {
			t.Errorf("the catalog groups connectors under %q and no file is named for it; "+
				"add one to categoryFile and create it", cat)
		}
	}
	if len(categoryFile) != len(catalog.CategoryOrder) {
		t.Errorf("categoryFile has %d entries, the catalog has %d categories",
			len(categoryFile), len(catalog.CategoryOrder))
	}

	seen := map[string]string{}
	for cat, file := range categoryFile {
		if other, taken := seen[file]; taken {
			t.Errorf("%s is named for both %q and %q; one file per category",
				file, other, cat)
		}
		seen[file] = cat
	}
}

// The rule that keeps the file names structural rather than historical.
//
// The files used to be forms_saas_a.go, forms_saas_b.go, forms_saas_c.go and so
// on, where the letter recorded which contributor wrote which half. Nothing
// objected, because nothing could: a spec is registered by its own init and the
// registry never asked where the call came from. registerSpec records that now,
// and this is what reads it back.
//
// A spec in the wrong file fails here with the name of the file it belongs in.
// It does not fail at run time and it does not change a single screen, which is
// exactly why it needs a test: the launcher works perfectly well with every
// spec in one file, and so does a launcher with a hundred of them scattered at
// random.
func TestEverySpecIsFiledUnderItsCategory(t *testing.T) {
	art, err := connectors.Load()
	if err != nil {
		t.Fatalf("the connector artifact does not parse: %v", err)
	}
	provider := make(map[string]string, len(art.Connectors))
	for _, c := range art.Connectors {
		provider[c.Name] = c.Provider
	}

	names := make([]string, 0, len(specFiles))
	for name := range specFiles {
		names = append(names, name)
	}
	sort.Strings(names)

	checked := 0
	for _, name := range names {
		// The provider is what decides the category, and it is not always the
		// connector's own name: ciscocatalyst and nd-ssh are served by
		// networkdevices, host by network, mcp by ai.
		prov, ok := provider[name]
		if !ok {
			t.Errorf("%s has a spec but is not in internal/connectors/connectors.json, "+
				"so its category cannot be resolved and its file cannot be checked. "+
				"Refresh the artifact with `make connectors/generate`, or say here why it "+
				"is absent", name)
			continue
		}
		want := categoryFile[catalog.Categorize(prov, name)]
		if want == "" {
			t.Errorf("%s is categorized as %q, which no file is named for",
				name, catalog.Categorize(prov, name))
			continue
		}
		checked++
		if got := specFiles[name]; got != want {
			t.Errorf("%s is registered in %s but the launcher files it under %q, "+
				"so its spec belongs in %s", name, got, catalog.Categorize(prov, name), want)
		}
	}

	if checked == 0 {
		t.Fatal("no spec was checked against its category, so this test proves nothing")
	}
	t.Logf("checked %d of %d registrations", checked, len(specFiles))
}

// The other half of the rule: a file that is not named for a category is a
// place a spec can hide from the test above.
//
// forms_saas_d.go is the failure this exists for. It would compile, its inits
// would run, every screen would be identical -- and the split by category would
// be back to a split by author with the first spec somebody appended to it.
func TestNoSpecFileIsNamedForSomethingElse(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	allowed := map[string]bool{}
	for _, f := range categoryFile {
		allowed[f] = true
	}

	found := 0
	for _, e := range entries {
		n := e.Name()
		if !strings.HasPrefix(n, "forms_") || !strings.HasSuffix(n, ".go") ||
			strings.HasSuffix(n, "_test.go") {
			continue
		}
		found++
		if !allowed[n] {
			t.Errorf("%s is not named for a catalog category. The specs are split by "+
				"what the launcher shows, not by who wrote them; put its contents in the "+
				"file for each connector's category", n)
		}
	}
	if found == 0 {
		t.Fatal("no spec file was found, so this test proves nothing")
	}
	t.Logf("checked %d spec files", found)
}

// Registering a connector twice means two people each believed they owned it,
// and init order decides which one the user gets. The launcher reports this
// too; it is asserted here as well because this is the package where the second
// registration would be written.
func TestNoConnectorIsRegisteredTwice(t *testing.T) {
	for _, name := range Duplicates() {
		t.Errorf("%s has more than one registered spec", name)
	}
}

// Specs() and Uncurated() hand out copies, so that the only way into the
// registry is registerSpec's first-wins rule. A caller that could reach the
// live map could add a connector's form from anywhere.
func TestTheRegistryIsNotWritableThroughItsAccessors(t *testing.T) {
	before := len(formSpecs)
	Specs()["not-a-connector"] = FormSpec{}
	Uncurated()["not-a-connector"] = "nor is this"
	dup := Duplicates()
	Duplicates()
	_ = dup

	if len(formSpecs) != before {
		t.Errorf("the spec registry grew from %d to %d through a returned map",
			before, len(formSpecs))
	}
	if _, ok := SpecFor("not-a-connector"); ok {
		t.Error("a connector written into a returned map reached the registry")
	}
	if _, ok := UncuratedReason("not-a-connector"); ok {
		t.Error("a reason written into a returned map reached the registry")
	}
}
