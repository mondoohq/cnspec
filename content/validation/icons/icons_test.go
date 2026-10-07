// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Package icons holds static checks on the `mondoo.com/icon` tag of the shipped
// policies and query packs. Like ../filters it only reads the bundle files, so
// it needs no providers and runs in the default `go test ./...`.
package icons

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
)

const iconTag = "mondoo.com/icon"

// contentGlobs cover the policies and the query packs, relative to this package.
var contentGlobs = []string{
	"../../*.mql.yaml",
	"../../querypacks/*.mql.yaml",
}

// iconName is the shape of a console icon name: a member of the Mondoo
// GraphQL API's ICON_IDS enum in lowercase kebab-case.
var iconName = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// TestEveryBundleNamesItsIcon requires every policy and query pack to name the
// mark the console shows for it (ADR 0042).
//
// The value is a member of the Mondoo GraphQL API's ICON_IDS enum in lowercase
// kebab-case, such as `aws`, `postgresql` or `palo-alto-networks`. Without the
// tag the console falls back to matching keywords in the title, and a title it
// does not recognise gets the generic icon. Name the product's mark where the
// enum has one, the vendor's otherwise, and `policy` for a policy that covers
// no single product.
//
// The enum lives in the API, so this test checks the shape of the value and
// not that the enum has it. A name outside the enum renders as the
// title-matched icon, not as an error.
func TestEveryBundleNamesItsIcon(t *testing.T) {
	var offenders []string
	for _, f := range contentFiles(t) {
		doc := readBundle(t, f)
		for _, kind := range []string{"policies", "packs"} {
			items, _ := doc[kind].([]any)
			for _, item := range items {
				m, ok := item.(map[string]any)
				if !ok {
					continue
				}
				uid, _ := m["uid"].(string)
				tags, _ := m["tags"].(map[string]any)
				value, _ := tags[iconTag].(string)
				switch {
				case value == "":
					offenders = append(offenders, fmt.Sprintf("%s: %s has no %s tag", filepath.Base(f), uid, iconTag))
				case !iconName.MatchString(value):
					offenders = append(offenders, fmt.Sprintf("%s: %s has %s %q, want a lowercase kebab-case icon name",
						filepath.Base(f), uid, iconTag, value))
				}
			}
		}
	}

	if len(offenders) > 0 {
		t.Errorf("every policy and query pack names its console icon with %s:\n  %s",
			iconTag, strings.Join(offenders, "\n  "))
	}
}

// TestIconTagOnlyOnBundles rejects `mondoo.com/icon` on anything below a policy
// or pack. A check variant names its mark with `mondoo.com/filter-icon`, next to
// `mondoo.com/filter-title`; written as `mondoo.com/icon` it reads as a
// policy-level tag and the variant shows no mark.
func TestIconTagOnlyOnBundles(t *testing.T) {
	var offenders []string
	for _, f := range contentFiles(t) {
		doc := readBundle(t, f)
		for key, v := range doc {
			if key == "policies" || key == "packs" {
				items, _ := v.([]any)
				for _, item := range items {
					m, _ := item.(map[string]any)
					for k, child := range m {
						if k != "tags" {
							walk(child, func(uid string) {
								offenders = append(offenders, fmt.Sprintf("%s: %s", filepath.Base(f), uid))
							})
						}
					}
				}
				continue
			}
			walk(v, func(uid string) {
				offenders = append(offenders, fmt.Sprintf("%s: %s", filepath.Base(f), uid))
			})
		}
	}

	if len(offenders) > 0 {
		sort.Strings(offenders)
		t.Errorf("%s belongs on a policy or query pack; a check or variant uses mondoo.com/filter-icon:\n  %s",
			iconTag, strings.Join(offenders, "\n  "))
	}
}

func contentFiles(t *testing.T) []string {
	t.Helper()
	var files []string
	for _, g := range contentGlobs {
		m, err := filepath.Glob(g)
		require.NoError(t, err)
		files = append(files, m...)
	}
	sort.Strings(files)
	// A glob that silently matched nothing would make these tests pass while
	// checking nothing. There are well over a hundred bundles.
	require.Greater(t, len(files), 100, "content globs matched too few files; is the path still right?")
	return files
}

func readBundle(t *testing.T, f string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(f)
	require.NoError(t, err)
	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(raw, &doc), f)
	return doc
}

// walk calls report for every map that carries a uid and a tags map with the
// icon tag.
func walk(node any, report func(uid string)) {
	switch n := node.(type) {
	case map[string]any:
		if tags, ok := n["tags"].(map[string]any); ok {
			if _, ok := tags[iconTag]; ok {
				uid, _ := n["uid"].(string)
				report(uid)
			}
		}
		for _, v := range n {
			walk(v, report)
		}
	case []any:
		for _, v := range n {
			walk(v, report)
		}
	}
}
