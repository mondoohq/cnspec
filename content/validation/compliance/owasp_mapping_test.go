// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Package compliance holds static validation tests for the compliance-tag
// mappings carried by the security policies in content/*.mql.yaml. These tests
// only read the bundle files, so they intentionally live in their own package:
// the ../scans package's TestMain provisions cloud providers for scan tests,
// which a pure mapping check neither needs nor should depend on.
//
// See ../README.md for what these cover and how to run them.
package compliance

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// OWASP Top 10:2025 mapping invariants.
//
// Security checks are mapped to compliance frameworks through per-check
// `tags:` entries (e.g. `compliance/owasp-top-10-2025: owasp-top-10-2025-a02`).
// A check participates in framework mapping when its tags block carries a
// `compliance/pci-dss-4:` entry (the marker every fully-mapped check shares).
//
// These tests guard the OWASP Top 10:2025 mapping so it cannot silently drift:
//   - every mapped value is well-formed,
//   - a policy is either fully OWASP-mapped or not mapped at all (no partial
//     coverage that would under-report a policy),
//   - the application-only categories are never used, and
//   - each in-scope category is actually exercised somewhere in the content.
var (
	reOwaspBaseUID = regexp.MustCompile(`^  - uid:\s*(\S+)\s*$`)
	reOwaspPci     = regexp.MustCompile(`^\s*compliance/pci-dss-4:`)
	reOwaspTag     = regexp.MustCompile(`^\s*compliance/owasp-top-10-2025:\s*(\S+)\s*$`)
	reOwaspValue   = regexp.MustCompile(`^(owasp-top-10-2025-a(0[1-9]|10)|false)$`)
	reOwaspCat     = regexp.MustCompile(`^owasp-top-10-2025-(a\d\d)$`)

	reLlmTag   = regexp.MustCompile(`^\s*compliance/owasp-llm-top-10-2025:\s*(\S+)\s*$`)
	reLlmValue = regexp.MustCompile(`^owasp-llm-top-10-2025-llm(0[1-9]|10)$`)
	reLlmCat   = regexp.MustCompile(`^owasp-llm-top-10-2025-(llm\d\d)$`)
)

// contentGlob points at the policy bundles two directories up
// (content/validation/compliance -> content). Go runs a test with its working
// directory set to the test file's package directory.
const contentGlob = "../../mondoo-*.mql.yaml"

// inScopeOwaspCategories are the OWASP Top 10:2025 categories that
// infrastructure and posture scanning can meaningfully assert.
var inScopeOwaspCategories = []string{"a01", "a02", "a03", "a04", "a07", "a08", "a09"}

// outOfScopeOwaspCategories are application-source or design concerns that a
// posture scanner cannot assert (A05 Injection, A06 Insecure Design, A10
// Mishandling of Exceptional Conditions). They must never appear as a mapping
// value; a wrong compliance claim is worse than an honest gap.
var outOfScopeOwaspCategories = map[string]bool{"a05": true, "a06": true, "a10": true}

type owaspCheck struct {
	hasPci   bool
	owaspVal string // "" when no owasp tag present
}

// scanOwaspFile returns, per base-check uid, whether it is mapped (pci) and its
// owasp tag value (if any), plus whether the file participates in OWASP mapping.
func scanOwaspFile(t *testing.T, path string) (map[string]*owaspCheck, bool) {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	checks := map[string]*owaspCheck{}
	var cur string
	participates := false
	for _, line := range strings.Split(string(data), "\n") {
		if m := reOwaspBaseUID.FindStringSubmatch(line); m != nil {
			cur = m[1]
			if _, ok := checks[cur]; !ok {
				checks[cur] = &owaspCheck{}
			}
			continue
		}
		if cur == "" {
			continue
		}
		if reOwaspPci.MatchString(line) {
			checks[cur].hasPci = true
			continue
		}
		if m := reOwaspTag.FindStringSubmatch(line); m != nil {
			checks[cur].owaspVal = m[1]
			participates = true
		}
	}
	return checks, participates
}

func TestOwaspTop10Mapping(t *testing.T) {
	files, err := filepath.Glob(contentGlob)
	require.NoError(t, err)
	require.NotEmpty(t, files)

	globalCatCount := map[string]int{}

	for _, f := range files {
		f := f
		t.Run(filepath.Base(f), func(t *testing.T) {
			checks, participates := scanOwaspFile(t, f)

			for uid, c := range checks {
				if c.owaspVal != "" {
					// values must be well-formed
					assert.Regexp(t, reOwaspValue, c.owaspVal,
						"%s: check %q has malformed owasp-top-10-2025 value %q", f, uid, c.owaspVal)
					// an owasp tag only belongs on a framework-mapped check
					assert.True(t, c.hasPci,
						"%s: check %q carries an owasp-top-10-2025 tag but is not framework-mapped (no compliance/pci-dss-4)", f, uid)
					// out-of-scope categories are forbidden
					if m := reOwaspCat.FindStringSubmatch(c.owaspVal); m != nil {
						assert.False(t, outOfScopeOwaspCategories[m[1]],
							"%s: check %q maps to out-of-scope OWASP category %q; A05/A06/A10 are not posture-assertable", f, uid, m[1])
						globalCatCount[m[1]]++
					}
				}
			}

			if !participates {
				return
			}
			// parity: a policy that is OWASP-mapped must map every mapped check
			for uid, c := range checks {
				if c.hasPci {
					assert.NotEmpty(t, c.owaspVal,
						"%s: check %q is framework-mapped but missing a compliance/owasp-top-10-2025 tag (policy must be fully mapped)", f, uid)
				}
			}
		})
	}

	// coverage: every in-scope OWASP category must be exercised somewhere
	t.Run("in-scope-coverage", func(t *testing.T) {
		var missing []string
		for _, cat := range inScopeOwaspCategories {
			if globalCatCount[cat] == 0 {
				missing = append(missing, cat)
			}
		}
		sort.Strings(missing)
		assert.Empty(t, missing,
			"in-scope OWASP Top 10:2025 categories with zero mapped checks: %v", missing)
	})
}

// llmAnchoredPolicies must each carry at least one OWASP Top 10 for LLM
// Applications tag: they are the AI/LLM-focused policies, so a regression that
// dropped their LLM mapping should fail loudly.
var llmAnchoredPolicies = []string{
	"mondoo-ai-security.mql.yaml",
	"mondoo-vllm-security.mql.yaml",
	"mondoo-mcp-security.mql.yaml",
}

// TestOwaspLlmTop10Mapping guards the OWASP Top 10 for LLM Applications:2025
// mapping. Unlike the application Top 10, this framework is targeted: it is
// applied only to AI/LLM/GenAI checks, so there is no per-policy parity rule.
// The tags still must be well-formed, sit on framework-mapped checks, cover a
// meaningful spread of categories, and never go missing from the AI policies.
func TestOwaspLlmTop10Mapping(t *testing.T) {
	files, err := filepath.Glob(contentGlob)
	require.NoError(t, err)
	require.NotEmpty(t, files)

	catCount := map[string]int{}
	perFileLlm := map[string]int{}
	total := 0

	for _, f := range files {
		data, err := os.ReadFile(f)
		require.NoError(t, err)

		var cur string
		hasApp := map[string]bool{}
		llmVal := map[string]string{}
		for _, line := range strings.Split(string(data), "\n") {
			if m := reOwaspBaseUID.FindStringSubmatch(line); m != nil {
				cur = m[1]
				continue
			}
			if cur == "" {
				continue
			}
			if reOwaspTag.MatchString(line) {
				hasApp[cur] = true
			}
			if m := reLlmTag.FindStringSubmatch(line); m != nil {
				llmVal[cur] = m[1]
			}
		}

		for uid, v := range llmVal {
			total++
			perFileLlm[filepath.Base(f)]++
			assert.Regexp(t, reLlmValue, v,
				"%s: check %q has malformed owasp-llm-top-10-2025 value %q", f, uid, v)
			// the LLM tag is inserted just above the application tag; a check
			// carrying it must also be an OWASP-mapped (framework) check
			assert.True(t, hasApp[uid],
				"%s: check %q has an owasp-llm-top-10-2025 tag but no owasp-top-10-2025 anchor", f, uid)
			if m := reLlmCat.FindStringSubmatch(v); m != nil {
				catCount[m[1]]++
			}
		}
	}

	assert.NotZero(t, total, "expected some OWASP Top 10 for LLM Applications mappings")
	// a meaningful spread of the ten LLM categories should be exercised
	assert.GreaterOrEqual(t, len(catCount), 5,
		"expected at least 5 distinct OWASP LLM categories, got %d: %v", len(catCount), catCount)

	t.Run("ai-policies-mapped", func(t *testing.T) {
		for _, p := range llmAnchoredPolicies {
			assert.NotZero(t, perFileLlm[p],
				"%s carries no owasp-llm-top-10-2025 tags", p)
		}
	})
}
