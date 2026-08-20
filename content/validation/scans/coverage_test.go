// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build iac_variants

package scans

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/policy"
)

// TestTerraformVariantCoverage requires every IaC variant in every registered
// policy to carry both a pass and a fail fixture. Coverage reached 100% across
// the corpus, so this is a flat assertion rather than a per-policy debt budget:
// a variant added without fixtures fails here instead of merging untested.
//
// A variant whose filter already asserts what its query asserts has no possible
// failing input; those carry a fail/IMPOSSIBLE.md marker (see failIsImpossible)
// and count as fail-covered. That marker is the only sanctioned way to ship a
// variant without a real fail fixture, and it has to state why.
func TestTerraformVariantCoverage(t *testing.T) {
	for _, pol := range tfVariantPolicies {
		policyDir := strings.TrimSuffix(pol.slugPrefix, "-")
		bundle, err := policy.DefaultBundleLoader().BundleFromPaths(bundlePath(pol.bundleFile))
		require.NoError(t, err)

		// per-suffix tally: total and covered
		total := map[string]int{}
		covered := map[string]int{}
		missing := map[string][]string{}
		for _, q := range bundle.Queries {
			suffix, ok := iacSuffix(q.Uid)
			if !ok {
				continue
			}
			total[suffix]++
			var hasPass, hasFail bool
			for _, sc := range scenariosFor(policyDir, q.Uid) {
				if sc.want == outcomePassed {
					hasPass = true
				} else {
					hasFail = true
				}
			}
			// Some checks assert exactly what their filter requires, so no failing
			// input exists. Such variants carry a fail/IMPOSSIBLE.md marker and are
			// considered fail-covered.
			if !hasFail && failIsImpossible(policyDir, q.Uid) {
				hasFail = true
			}
			if hasPass && hasFail {
				covered[suffix]++
			} else {
				missing[suffix] = append(missing[suffix], q.Uid)
			}
		}
		for _, suffix := range iacVariantSuffixes {
			if total[suffix] == 0 {
				continue
			}
			pct := float64(covered[suffix]) / float64(total[suffix]) * 100
			t.Logf("%s %s: %d/%d covered (%.1f%%)", policyDir, suffix, covered[suffix], total[suffix], pct)
			uncovered := missing[suffix]
			sort.Strings(uncovered)
			if len(uncovered) > 0 {
				t.Errorf("%s %s: %d of %d variants lack pass+fail fixtures. Every IaC variant "+
					"must have both.\nAdd fixtures under %s/%s/<uid>/{pass,fail}/<scenario>/ for "+
					"the variants below. If a variant's filter already asserts what its query "+
					"asserts, no failing input exists: add %s/%s/<uid>/fail/IMPOSSIBLE.md "+
					"explaining why instead.\nuncovered (%d):\n%s",
					policyDir, suffix, len(uncovered), total[suffix],
					tfVariantsRoot, policyDir, tfVariantsRoot, policyDir,
					len(uncovered), strings.Join(indent(uncovered), "\n"))
			}
		}
	}
}

// failIsImpossible reports whether a variant is marked as having no possible
// failing input (a fail/IMPOSSIBLE.md marker exists).
func failIsImpossible(policyDir, uid string) bool {
	_, err := os.Stat(filepath.Join(tfVariantsRoot, policyDir, uid, "fail", "IMPOSSIBLE.md"))
	return err == nil
}
