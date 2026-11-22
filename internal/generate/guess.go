// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package generate

import (
	"fmt"
	"strings"
	"unicode"

	"go.mondoo.com/mql/providers"
)

// Guessing a check's target from its wording is the front end's first act, and
// both front ends -- the line wizard and the launcher's authoring pane -- have
// to guess the same way, so it lives here beside ResolveProvider rather than
// beside either of them.

// Terms are matched as whole words (a trailing plural "s" allowed), never as
// substrings: "rds" must not fire on "passwords", "iam" on "William", "eks" on
// "weeks", "aks" on "leaks" or "pam" on "company". A wrong guess is worse than
// none, because the pane fills in that provider's asset filter on its own.
func GuessProvider(title string) string {
	t := newWordSet(title)
	switch {
	case t.hasAny("aws", "s3", "ec2", "iam", "cloudtrail", "rds", "eks", "lambda"):
		return "aws"
	case t.hasAny("gcp", "gke", "bigquery", "cloud sql", "compute engine"):
		return "gcp"
	case t.hasAny("azure", "aks", "blob"):
		return "azure"
	case t.hasAny("ssh", "sshd", "kernel", "linux", "package", "systemd", "sudo", "pam"):
		return "os"
	case t.hasAny("kubernetes", "k8s", "pod", "container"):
		return "k8s"
	}
	return ""
}

// curatedFilters maps a provider to the asset filter the wizard proposes.
//
// These are *platform* names, not provider names, and the two differ for most
// providers. A filter naming a platform that does not exist is dead: it lints
// clean, it scans clean, and it never matches an asset — which is what
// `asset.platform == "gcp"` and `asset.platform == "k8s"` were, since neither
// name exists (gcp's platforms are gcp-project, gcp-gke-cluster, …; "k8s" is a
// platform *family*, and its platforms are k8s-cluster, k8s-pod, …).
//
// Every name here is taken from installed provider metadata
// (~/.config/mondoo/providers/<n>/<n>.json, Platforms[].name and .family) and is
// in use by real checks in content/ — see TestDefaultFilterUsesRealPlatformNames,
// which re-derives both. The choice among a provider's platforms is the scope
// where that provider's account-wide resources resolve.
var curatedFilters = map[string]string{
	// the account-level asset; content/ uses it for ~200 checks
	"aws": `asset.platform == "aws"`,
	// the subscription-level asset
	"azure": `asset.platform == "azure"`,
	// there is no platform named "gcp"; the project is the account-level asset
	"gcp": `asset.platform == "gcp-project"`,
	// cluster and manifest are the two scopes where the cluster-wide k8s.*
	// resources resolve; workload platforms (k8s-pod, k8s-deployment, …) are
	// per-object and too narrow to guess at
	"k8s": `asset.platform == "k8s-cluster" || asset.platform == "k8s-manifest"`,
	// os platforms are per-distribution (ubuntu, redhat, …); the family is the
	// portable way to say "any Linux"
	"os": `asset.family.contains("linux")`,
}

// defaultFilter proposes an asset filter for a provider: a curated one where the
// provider has many platforms and only some are plausible, otherwise one derived
// from the installed provider metadata. Returns "" when neither knows the
// provider — an empty prompt the user fills in beats a filter that silently
// matches nothing.
func DefaultFilter(provider string) string {
	provider = strings.TrimSpace(provider)
	if provider == "" {
		return ""
	}
	if f, ok := curatedFilters[provider]; ok {
		return f
	}
	name, family := lookupPlatform(provider)
	switch {
	case name != "":
		return fmt.Sprintf(`asset.platform == %q`, name)
	case family != "":
		return fmt.Sprintf(`asset.family.contains(%q)`, family)
	}
	return ""
}

// installedPlatforms reports the platform names and families a provider
// declares. It is a variable so tests can drive DefaultFilter without depending
// on which providers happen to be installed.
var installedPlatforms = func(provider string) (names []string, families []string) {
	// ListAll only reads the installed provider metadata: no network, no
	// installs, and resource schemas stay unparsed.
	all, err := providers.ListAll()
	if err != nil {
		return nil, nil
	}
	seen := map[string]bool{}
	for _, p := range all {
		if p == nil || p.Provider == nil || p.Name != provider {
			continue
		}
		for _, pl := range p.Platforms {
			if pl == nil {
				continue
			}
			names = append(names, pl.Name)
			for _, f := range pl.Family {
				if !seen[f] {
					seen[f] = true
					families = append(families, f)
				}
			}
		}
	}
	return names, families
}

// lookupPlatform resolves a provider name against installed metadata: an exact
// platform of that name (aws, digitalocean, oci, …), else a family of that name
// (github, terraform, …). Anything more ambiguous is left to the user.
func lookupPlatform(provider string) (name, family string) {
	names, families := installedPlatforms(provider)
	for _, n := range names {
		if n == provider {
			return n, ""
		}
	}
	for _, f := range families {
		if f == provider {
			return "", f
		}
	}
	return "", ""
}

// wordSet is a title split into lower-case alphanumeric words.
type wordSet struct {
	words  map[string]bool
	joined string // " w1 w2 … wn ", for multi-word terms
}

func newWordSet(s string) wordSet {
	fields := strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	ws := wordSet{words: make(map[string]bool, len(fields)), joined: " " + strings.Join(fields, " ") + " "}
	for _, f := range fields {
		ws.words[f] = true
	}
	return ws
}

// hasAny reports whether any term occurs as a whole word, or for a multi-word
// term as a run of whole words. A single trailing "s" is accepted as a plural
// ("pods", "packages", "containers").
func (ws wordSet) hasAny(terms ...string) bool {
	for _, term := range terms {
		if strings.Contains(term, " ") {
			if strings.Contains(ws.joined, " "+term+" ") || strings.Contains(ws.joined, " "+term+"s ") {
				return true
			}
			continue
		}
		if ws.words[term] || ws.words[term+"s"] {
			return true
		}
	}
	return false
}
