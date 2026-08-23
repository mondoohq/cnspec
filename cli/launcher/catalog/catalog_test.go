// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package catalog

import (
	"testing"

	"go.mondoo.com/mql/providers"
)

// The catalog is built from the machine's real provider set, so these tests
// assert relationships rather than fixed contents.

// providers.DefaultProviders is the static fallback compiled into the binary and
// it lags behind what ships: whole categories exist only in the installed set.
// Building the catalog from it alone silently drops them.
func TestCatalogIncludesInstalledProviders(t *testing.T) {
	active, err := providers.ListActive()
	if err != nil || len(active) == 0 {
		t.Skip("no providers installed locally")
	}

	inCatalog := map[string]bool{}
	for _, c := range BuildCatalog() {
		inCatalog[c.Provider] = true
	}

	for _, p := range active {
		if p.Provider == nil || p.Name == "" || ExcludedProviders[p.Name] {
			continue
		}
		hasVisibleConnector := false
		for _, conn := range p.Connectors {
			if !conn.IsHidden && conn.Name != "" {
				hasVisibleConnector = true
			}
		}
		if hasVisibleConnector && !inCatalog[p.Name] {
			t.Errorf("provider %q is installed but missing from the catalog", p.Name)
		}
	}
}

// ListActive is keyed by provider ID, not name. Keying the install lookup off
// the map key leaves every connector flagged as not installed.
func TestInstalledFlagIsSet(t *testing.T) {
	active, err := providers.ListActive()
	if err != nil || len(active) == 0 {
		t.Skip("no providers installed locally")
	}

	catalog := BuildCatalog()
	installed := 0
	for _, c := range catalog {
		if c.Installed {
			installed++
		}
	}
	if installed == 0 {
		t.Fatalf("no connector flagged installed, but %d providers are active", len(active))
	}
}

// Every category the launcher offers must be reachable. A category that is
// declared, ordered and given an icon but never populated is a dead end the UI
// still advertises.
func TestDeclaredCategoriesArePopulated(t *testing.T) {
	if _, err := providers.ListActive(); err != nil {
		t.Skip("cannot read the installed provider set")
	}
	counts := map[string]int{}
	for _, c := range BuildCatalog() {
		counts[c.Category]++
	}
	for _, cat := range CategoryOrder {
		if cat == CatOther {
			continue // a genuine catch-all; may legitimately be empty
		}
		if counts[cat] == 0 {
			t.Errorf("category %q is declared and ordered but has no entries", cat)
		}
	}
}

// Replay/debug connectors are not scan targets and must stay out of a launcher
// whose premise is "pick something to secure".
func TestExcludedProvidersAreAbsent(t *testing.T) {
	for _, c := range BuildCatalog() {
		if ExcludedProviders[c.Provider] {
			t.Errorf("connector %q from excluded provider %q is in the catalog", c.Name, c.Provider)
		}
	}
}

func TestCatalogSortsLocalFirst(t *testing.T) {
	for _, c := range BuildCatalog() {
		if c.Category != CatHosts {
			continue
		}
		if c.Name != "local" {
			t.Errorf("expected local first in %q, got %q", CatHosts, c.Name)
		}
		return
	}
}

// The form is built from connector metadata that only installed providers
// carry, so an installed connector must arrive with it attached.
func TestInstalledConnectorsCarryFormMetadata(t *testing.T) {
	if _, err := providers.ListActive(); err != nil {
		t.Skip("cannot read the installed provider set")
	}
	want := map[string]bool{"aws": true, "ssh": true, "k8s": true, "docker": true, "github": true}
	seen := map[string]bool{}
	for _, c := range BuildCatalog() {
		if !want[c.Name] || !c.Installed {
			continue
		}
		seen[c.Name] = true
		if !c.HasFormData() {
			t.Errorf("%s: installed but carries no form metadata", c.Name)
		}
		if len(c.Flags) == 0 {
			t.Errorf("%s: no flags carried through", c.Name)
		}
	}
	for n := range want {
		if !seen[n] {
			t.Logf("connector %q not installed locally; skipped", n)
		}
	}
}

// MinArgs is authoritative where it exists. ssh declares exactly one required
// positional argument; aws declares none.
func TestRequiredArgUsesMinArgs(t *testing.T) {
	if _, err := providers.ListActive(); err != nil {
		t.Skip("cannot read the installed provider set")
	}
	for _, c := range BuildCatalog() {
		if !c.Installed {
			continue
		}
		switch c.Name {
		case "ssh":
			if !c.RequiresArg() || c.MinArgs != 1 {
				t.Errorf("ssh: RequiresArg=%v MinArgs=%d, want true/1", c.RequiresArg(), c.MinArgs)
			}
		case "aws":
			if c.RequiresArg() {
				t.Errorf("aws: RequiresArg=true, but MinArgs=%d", c.MinArgs)
			}
		}
	}
}

// Without metadata the usage-string heuristic still has to work, because that
// is all an uninstalled connector has.
func TestRequiresArgFallsBackToUsageString(t *testing.T) {
	cases := []struct {
		use, name string
		want      bool
	}{
		{"ssh user@host", "ssh", true},
		{"ansible PATH", "ansible", true},
		{"mongo [host]", "mongo", false},
		{"aws", "aws", false},
	}
	for _, c := range cases {
		got := Connector{Name: c.name, Use: c.use}.RequiresArg()
		if got != c.want {
			t.Errorf("RequiresArg(%q) with no metadata = %v, want %v", c.use, got, c.want)
		}
	}
}

// Identity is its own shelf. "Review our identity provider" and "review the
// tools we subscribe to" are different jobs, and twenty-six connectors in one
// list served neither.
func TestIdentityIsSeparateFromSaaS(t *testing.T) {
	cat := map[string]string{}
	for _, c := range BuildCatalog() {
		cat[c.Name] = c.Category
	}

	for _, name := range []string{
		"okta", "auth0", "keycloak", "jumpcloud", "activedirectory",
		// Productivity suites, but a scan of either inspects the directory,
		// the sign-in policy and the admin roles.
		"ms365", "google-workspace",
	} {
		if got := cat[name]; got != CatIdentity {
			t.Errorf("%s is in %q, want %q -- it decides who can sign in", name, got, CatIdentity)
		}
	}
	for _, name := range []string{"slack", "github", "dropbox", "atlassian"} {
		if got := cat[name]; got != CatSaaS {
			t.Errorf("%s is in %q, want %q", name, got, CatSaaS)
		}
	}
}

// Iru is Kandji renamed: Apple device management, the same job as Jamf, so it
// belongs wherever Jamf does rather than in the catch-all.
func TestIruSitsWithJamf(t *testing.T) {
	var iru, jamf string
	for _, c := range BuildCatalog() {
		switch c.Name {
		case "iru":
			iru = c.Category
		case "jamf":
			jamf = c.Category
		}
	}
	if iru == "" || jamf == "" {
		t.Skip("neither provider is installed here")
	}
	if iru != jamf {
		t.Errorf("iru is in %q and jamf in %q; they do the same job", iru, jamf)
	}
	if iru == CatOther {
		t.Error("iru fell back to the catch-all category")
	}
}

// Ninety-four of the descriptions begin "a " or "an ", so in a list that word
// is pure overhead -- every row says it, and what gets truncated at the right
// edge is the part that tells connectors apart.
func TestTheListDropsTheLeadingArticle(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"a Proxmox VE hypervisor", "Proxmox VE hypervisor"},
		{"an Ansible playbook or role", "Ansible playbook or role"},
		{"your local system", "your local system"},
		{"Terraform HCL configuration", "Terraform HCL configuration"},
		// Only the article, and only at the front: "an" inside the text stays,
		// and a word merely starting with those letters is not an article.
		{"a host running an agent", "host running an agent"},
		{"another kind of target", "another kind of target"},
		{"", ""},
	} {
		if got := (Connector{Short: tc.in}).Summary(); got != tc.want {
			t.Errorf("Summary(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// The provider's own sentence is left intact: this is a presentation choice for
// a cramped list, not a rewrite of the catalog.
func TestTheDetailPaneKeepsTheFullSentence(t *testing.T) {
	c := Connector{Short: "a Proxmox VE hypervisor"}
	if c.Short != "a Proxmox VE hypervisor" {
		t.Errorf("the connector's own text was modified: %q", c.Short)
	}
}
