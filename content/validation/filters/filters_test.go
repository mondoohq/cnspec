// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Package filters holds static checks on the shape of `filters:` in the shipped
// policies and query packs. Like ../compliance it only reads the bundle files,
// so it needs no providers and runs in the default `go test ./...`.
package filters

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
)

// contentGlobs cover the policies and the query packs, relative to this package.
var contentGlobs = []string{
	"../../*.mql.yaml",
	"../../querypacks/*.mql.yaml",
}

// TestNoMultiItemFilterLists rejects a `filters:` written as a list of two or
// more `- mql:` items.
//
// A filter list is an OR: the resolver applies a query when any one item
// matches (policy.Filters.Supports returns on the first hit). The list form
// reads like an AND, and that is how it gets used -- "this asset type, and this
// condition" -- so the broad item matches on its own and the narrowing one never
// takes effect. Every multi-item list in the repo was an AND written as a list
// when this test was added: twelve of them, each pairing an asset check with a narrowing
// predicate. The effect was workstation-only inventory running on container
// images, and an incident-response pack reporting every S3 bucket as public
// and every IAM user as an administrator.
//
// Write a multi-condition filter as one expression. For an AND, join the
// conditions with a trailing `&&`; for a genuine OR, write `||` so the intent is
// on the page.
func TestNoMultiItemFilterLists(t *testing.T) {
	var files []string
	for _, g := range contentGlobs {
		m, err := filepath.Glob(g)
		require.NoError(t, err)
		files = append(files, m...)
	}
	sort.Strings(files)
	// A glob that silently matched nothing would make this test pass while
	// checking nothing. There are well over a hundred bundles.
	require.Greater(t, len(files), 100, "content globs matched too few files; is the path still right?")

	var offenders []string
	for _, f := range files {
		raw, err := os.ReadFile(f)
		require.NoError(t, err)

		var doc any
		require.NoError(t, yaml.Unmarshal(raw, &doc), f)

		walk(doc, func(uid string, items []any) {
			offenders = append(offenders, fmt.Sprintf("%s: %s has %d filter items, which the resolver ORs",
				filepath.Base(f), uid, len(items)))
		})
	}

	if len(offenders) > 0 {
		t.Errorf("filters written as a list are an OR, not an AND; join the conditions "+
			"into one expression with `&&` (or `||` if an OR is really meant):\n  %s",
			strings.Join(offenders, "\n  "))
	}
}

// walk calls report for every map that carries a uid and a filter list of two
// or more items.
func walk(node any, report func(uid string, items []any)) {
	switch n := node.(type) {
	case map[string]any:
		if items, ok := n["filters"].([]any); ok && len(items) >= 2 {
			uid, _ := n["uid"].(string)
			report(uid, items)
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
