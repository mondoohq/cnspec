// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// Where a connector's tests belong, and the rule that keeps it true.
//
// cli/launcher/forms files each spec under the category the launcher shows its
// connector in, and TestEverySpecIsFiledUnderItsCategory over there refuses a
// spec written anywhere else. The tests were the half that had not caught up:
// forms_saas_a_test.go, forms_saas_b_test.go, forms_saas_c_test.go and
// forms_misc_test.go recorded which contributor wrote them, and each iterated a
// package-level list spanning several categories -- saasAConnectors held six
// SaaS connectors and two Identity ones -- so no rule could have been stated
// about where any of them belonged.
//
// The lists are one category each now, and this is what reads that back.
//
// # What this can and cannot enforce
//
// It enforces placement for every connector named in a list built by filedHere,
// which is every list a category file iterates. It cannot enforce placement for
// a connector named only inside a test body -- TestDatadogSiteIsASuggestion
// hard-codes "datadog" and nothing here sees it -- because there is no call for
// runtime.Caller to attribute. That is a real gap and it is worth saying so
// rather than papering over: what the rule guarantees is that a file's *list*
// cannot drift from its name, which is the failure that actually happened.

// categoryTestFile is where a category's tests belong. It is the mirror of
// categoryFile in cli/launcher/forms/forms_test.go: the same eleven categories,
// the same one-file-per-category rule, with _test on the end.
//
// Two of these name files that do not exist yet. That is the same arrangement
// the spec side has for forms_other.go, and it is deliberate: naming the file
// before it exists is what stops the first Databases test from being appended
// to whichever file its author had open.
var categoryTestFile = map[string]string{
	catHosts:     "forms_hosts_test.go",
	catContainer: "forms_container_test.go",
	catCloud:     "forms_cloud_test.go",
	catIaC:       "forms_iac_test.go",
	catIdentity:  "forms_identity_test.go",
	catSaaS:      "forms_saas_test.go",
	catNetwork:   "forms_network_test.go",
	catDatabase:  "forms_database_test.go",
	catAI:        "forms_ai_test.go",
	catDev:       "forms_dev_test.go",
	catOther:     "forms_other_test.go",
}

// packageWideTestFiles are the forms_*_test.go files that are not named for a
// category because they are not about one. Each holds a gate that runs over the
// whole registry, so filing it under a category would be filing it under the
// wrong one ten times.
var packageWideTestFiles = map[string]string{
	"forms_test.go":           "the gates over every registered spec",
	"forms_generated_test.go": "the provider-declared vocabulary, over every connector",
	"forms_filing_test.go":    "this rule",
}

// connectorTestFile records which test file claimed each connector, taken from
// the call stack rather than declared.
//
// Declared would defeat the purpose: the failure this exists to catch is a
// block of tests pasted into the wrong file, and a pasted block brings its
// declared file name with it. The stack cannot be pasted.
var connectorTestFile = map[string]string{}

// filedHere records that the calling file owns these connectors, and returns
// them so it reads as the list it is:
//
//	var saasConnectors = filedHere("atlassian", "cloudflare", ...)
//
// It is called from a package-level var, so the frame above it is the
// declaration, which is exactly the file the connector's tests live in.
func filedHere(names ...string) []string {
	file := ""
	if _, f, _, ok := runtime.Caller(1); ok {
		file = filepath.Base(f)
	}
	for _, name := range names {
		connectorTestFile[name] = file
	}
	return names
}

// A category the catalog groups connectors under, with nowhere here to put its
// tests, is how the taxonomy drifts back apart: the first test for it lands in
// whichever file its author had open.
func TestEveryCatalogCategoryHasATestFile(t *testing.T) {
	for _, cat := range categoryOrder {
		if categoryTestFile[cat] == "" {
			t.Errorf("the catalog groups connectors under %q and no test file is "+
				"named for it; add one to categoryTestFile", cat)
		}
	}
	if len(categoryTestFile) != len(categoryOrder) {
		t.Errorf("categoryTestFile has %d entries, the catalog has %d categories",
			len(categoryTestFile), len(categoryOrder))
	}

	seen := map[string]string{}
	for cat, file := range categoryTestFile {
		if other, taken := seen[file]; taken {
			t.Errorf("%s is named for both %q and %q; one file per category",
				file, other, cat)
		}
		seen[file] = cat
	}
}

// The rule itself: a connector's tests live in the file named for the category
// the launcher shows it under.
//
// The category is resolved through categorize(), which is the launcher's own
// answer and the same one cli/launcher/forms checks its specs against. Writing
// a second connector-to-category table here would put the rule and the thing it
// checks in the same hand, and the table is what went stale last time:
// forms_misc_test.go filed iru under a comment reading "Other" while the
// launcher had been showing it under SaaS since jamf and iru were grouped.
func TestEveryTestedConnectorIsFiledUnderItsCategory(t *testing.T) {
	byName := snapshotByName(t)

	names := make([]string, 0, len(connectorTestFile))
	for name := range connectorTestFile {
		names = append(names, name)
	}
	sort.Strings(names)

	checked := 0
	for _, name := range names {
		// The provider decides the category and is not always the connector's
		// own name: ciscocatalyst and nd-ssh are served by networkdevices, host
		// by network, mcp by ai.
		snap, ok := byName[name]
		if !ok {
			t.Errorf("%s is claimed by %s but is not in %s, so its category cannot "+
				"be resolved and its filing cannot be checked",
				name, connectorTestFile[name], connectorSnapshotPath)
			continue
		}
		cat := categorize(snap.Provider, name)
		want := categoryTestFile[cat]
		if want == "" {
			t.Errorf("%s is categorized as %q, which no test file is named for", name, cat)
			continue
		}
		checked++
		if got := connectorTestFile[name]; got != want {
			t.Errorf("%s is tested in %s but the launcher files it under %q, "+
				"so its tests belong in %s", name, got, cat, want)
		}
	}

	if checked == 0 {
		t.Fatal("no connector was checked against its category, so this test proves nothing")
	}
	t.Logf("checked %d connector list entries against the catalog", checked)
}

// The other half of the rule: a forms_*_test.go that is not named for a
// category is a place a connector's tests can hide from the check above.
//
// forms_saas_d_test.go is the failure this exists for. It would compile, it
// would run, every assertion in it would pass -- and the split by category
// would be back to a split by author with the first test somebody appended to
// it.
func TestNoFormsTestFileIsNamedForSomethingElse(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	allowed := map[string]bool{}
	for _, f := range categoryTestFile {
		allowed[f] = true
	}
	for f := range packageWideTestFiles {
		allowed[f] = true
	}

	found := 0
	for _, e := range entries {
		n := e.Name()
		if !strings.HasPrefix(n, "forms_") || !strings.HasSuffix(n, "_test.go") {
			continue
		}
		found++
		if !allowed[n] {
			t.Errorf("%s is not named for a catalog category and is not one of the "+
				"package-wide gates. The form tests are split by what the launcher "+
				"shows, not by who wrote them; move each test to the file for its "+
				"connector's category", n)
		}
	}
	if found == 0 {
		t.Fatal("no form test file was found, so this test proves nothing")
	}
	t.Logf("checked %d form test files", found)
}

// No connector is claimed by two files. filedHere would silently let the second
// win, and the two lists would then disagree about who runs the sweep over it.
func TestNoConnectorIsClaimedByTwoTestFiles(t *testing.T) {
	// Rebuilding the claims from the lists is the only way to see a collision:
	// connectorTestFile keeps the last writer, so the map alone cannot show it.
	counts := map[string]int{}
	for _, list := range [][]string{
		aiConnectors, cloudAConnectors, cloudBConnectors, containerConnectors,
		devConnectors, hostsConnectors, iacConnectors, identityConnectors,
		networkAConnectors, networkBConnectors, saasConnectors,
	} {
		for _, name := range list {
			counts[name]++
		}
	}
	for name, n := range counts {
		if n > 1 {
			t.Errorf("%s appears in %d connector lists; one file owns a connector", name, n)
		}
	}
	if len(counts) != len(connectorTestFile) {
		t.Errorf("%d connectors across the lists, %d recorded by filedHere; "+
			"a list was declared without it", len(counts), len(connectorTestFile))
	}
}
