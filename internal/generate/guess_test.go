// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package generate

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func TestGuessProvider(t *testing.T) {
	if p := GuessProvider("S3 buckets must be encrypted"); p != "aws" {
		t.Errorf("aws guess = %q", p)
	}
	if p := GuessProvider("SSH root login disabled"); p != "os" {
		t.Errorf("os guess = %q", p)
	}
	if p := GuessProvider("Something generic"); p != "" {
		t.Errorf("expected no guess, got %q", p)
	}
}

// --- default asset filters ---------------------------------------------------

var (
	platformFilterRe = regexp.MustCompile(`asset\.platform\s*==\s*"([a-z0-9._-]+)"`)
	familyFilterRe   = regexp.MustCompile(`asset\.family\.contains\("([a-z0-9._-]+)"\)`)
)

// TestDefaultFilterUsesRealPlatformNames validates every filter the wizard
// proposes against authorities outside this file: the platform and family names
// that real checks in content/ filter on, and — when providers are installed —
// the Platforms[] metadata of the provider itself.
//
// It replaces a test that restated defaultFilter's implementation back to
// itself, which is how `asset.platform == "gcp"` and `asset.platform == "k8s"`
// shipped green: neither name exists (gcp's platforms are gcp-project,
// gcp-gke-cluster, …; k8s is a family, its platforms are k8s-cluster, k8s-pod,
// …), so both filters matched no asset, ever, while lint stayed silent.
func TestDefaultFilterUsesRealPlatformNames(t *testing.T) {
	contentPlatforms, contentFamilies := contentFilterNames(t)
	if len(contentPlatforms) == 0 {
		t.Fatal("found no asset.platform filters in content/; the corpus this test derives from is missing")
	}

	var checkedMeta, skippedMeta int
	for provider := range curatedFilters {
		filter := DefaultFilter(provider)
		platforms := allSubmatches(platformFilterRe, filter)
		families := allSubmatches(familyFilterRe, filter)
		if len(platforms)+len(families) == 0 {
			t.Errorf("DefaultFilter(%q) = %q names neither a platform nor a family", provider, filter)
			continue
		}

		for _, name := range platforms {
			if !contentPlatforms[name] {
				t.Errorf("DefaultFilter(%q) = %q filters on platform %q, which no check in content/ uses; a platform name that does not exist matches no asset", provider, filter, name)
			}
		}
		for _, name := range families {
			if !contentFamilies[name] {
				t.Errorf("DefaultFilter(%q) = %q filters on family %q, which no check in content/ uses", provider, filter, name)
			}
		}

		// second authority: the provider's own metadata, when it is installed
		metaPlatforms, metaFamilies := installedPlatforms(provider)
		if len(metaPlatforms) == 0 {
			skippedMeta++
			continue
		}
		checkedMeta++
		known := map[string]bool{}
		for _, n := range metaPlatforms {
			known[n] = true
		}
		knownFamily := map[string]bool{}
		for _, f := range metaFamilies {
			knownFamily[f] = true
		}
		for _, name := range platforms {
			if !known[name] {
				t.Errorf("DefaultFilter(%q) = %q filters on platform %q, which the installed %s provider does not declare (Platforms[].name)", provider, filter, name, provider)
			}
		}
		for _, name := range families {
			if !knownFamily[name] {
				t.Errorf("DefaultFilter(%q) = %q filters on family %q, which the installed %s provider does not declare (Platforms[].family)", provider, filter, name, provider)
			}
		}
	}

	// CI runs with no providers installed, so say out loud how much of this test
	// actually ran instead of reporting a vacuous pass.
	t.Logf("checked %d/%d provider default(s) against installed provider metadata (%d skipped: provider not installed); all %d checked against content/",
		checkedMeta, len(curatedFilters), skippedMeta, len(curatedFilters))
}

// TestDefaultFilterDerivesUnknownProviders covers the providers with no curated
// default: the filter has to come from installed metadata, and when metadata
// cannot answer, the wizard must offer no default at all rather than the old
// `asset.platform == <provider name>` guess, which is dead for every provider
// whose platforms are not named after it (github, terraform, gitlab, …).
func TestDefaultFilterDerivesUnknownProviders(t *testing.T) {
	orig := installedPlatforms
	t.Cleanup(func() { installedPlatforms = orig })
	installedPlatforms = func(provider string) ([]string, []string) {
		switch provider {
		case "github":
			return []string{"github-org", "github-user", "github-repo"}, []string{"github"}
		case "digitalocean":
			return []string{"digitalocean", "digitalocean-database"}, []string{"digitalocean"}
		}
		return nil, nil
	}

	cases := map[string]string{
		// a platform is named after the provider: use it
		"digitalocean": `asset.platform == "digitalocean"`,
		// none is, but the family is: use the family
		"github": `asset.family.contains("github")`,
		// nothing knows this provider: no default beats a dead one
		"nosuchprovider": "",
		"":               "",
	}
	for provider, want := range cases {
		if got := DefaultFilter(provider); got != want {
			t.Errorf("DefaultFilter(%q) = %q, want %q", provider, got, want)
		}
	}
}

// contentFilterNames returns the platform and family names that checks in
// content/ actually filter on.
func contentFilterNames(t *testing.T) (platforms, families map[string]bool) {
	t.Helper()
	platforms, families = map[string]bool{}, map[string]bool{}

	files, err := filepath.Glob(filepath.Join("..", "..", "content", "*.mql.yaml"))
	if err != nil {
		t.Fatalf("glob content: %v", err)
	}
	packs, err := filepath.Glob(filepath.Join("..", "..", "content", "querypacks", "*.mql.yaml"))
	if err != nil {
		t.Fatalf("glob querypacks: %v", err)
	}
	for _, f := range append(files, packs...) {
		fh, err := os.Open(f)
		if err != nil {
			t.Fatalf("open %s: %v", f, err)
		}
		sc := bufio.NewScanner(fh)
		sc.Buffer(make([]byte, 1024*1024), 1024*1024)
		for sc.Scan() {
			line := sc.Text()
			for _, m := range allSubmatches(platformFilterRe, line) {
				platforms[m] = true
			}
			for _, m := range allSubmatches(familyFilterRe, line) {
				families[m] = true
			}
		}
		fh.Close()
	}
	return platforms, families
}

func allSubmatches(re *regexp.Regexp, s string) []string {
	var out []string
	for _, m := range re.FindAllStringSubmatch(s, -1) {
		out = append(out, m[1])
	}
	return out
}

// --- wizard plumbing for tests ----------------------------------------------
