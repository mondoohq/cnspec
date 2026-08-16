// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build iac_variants

package scans

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/v13/policy"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
)

// This suite asks the one question the linters cannot: does the fix we ship
// actually make our own check pass?
//
// cfn-lint, tflint and `bicep build` prove a snippet is well-formed. They cannot
// prove it is *right*. A snippet can name every property correctly and still
// demonstrate the exact misconfiguration the check forbids, target a resource
// type the check does not match, or set a neighbouring property instead of the
// one asserted — and every one of those has been found in this corpus.
//
// The material to answer it was already here and simply never connected: each
// IaC variant has a check, and its parent has a remediation snippet in the same
// language. This writes that snippet into a scenario directory and scans it with
// the same machinery the pass/fail fixtures use, requiring the check to pass.
//
// The snippet is generated per run rather than checked in. It is not a fixture
// in its own right — it is a copy of content that already lives in the policy,
// and a checked-in copy would drift from it silently.

// remediationMethod maps an IaC variant suffix to the remediation `- id:` that
// documents the fix for it, and to the fence language and filename that method
// uses.
var remediationMethod = map[string]struct {
	id       string
	fences   []string
	filename string
}{
	"-terraform-hcl":  {"terraform", []string{"hcl", "terraform", "tf"}, "main.tf"},
	"-cloudformation": {"cloudformation", []string{"yaml", "yml"}, "template.yaml"},
	"-bicep":          {"bicep", []string{"bicep"}, "main.bicep"},
}

// fencePattern matches a fenced block and captures its language and body.
var fencePattern = regexp.MustCompile("(?s)```([A-Za-z0-9_+-]*)[ \t]*\n(.*?)```")

// snippetFor returns the remediation code a variant's parent documents for that
// variant's language, already dedented out of the YAML block scalar.
func snippetFor(parent *policy.Mquery, suffix string) (string, bool) {
	method, ok := remediationMethod[suffix]
	if !ok || parent == nil || parent.Docs == nil || parent.Docs.Remediation == nil {
		return "", false
	}
	for _, item := range parent.Docs.Remediation.Items {
		if item.Id != method.id {
			continue
		}
		var bodies []string
		for _, m := range fencePattern.FindAllStringSubmatch(item.Desc, -1) {
			lang := strings.ToLower(m[1])
			for _, want := range method.fences {
				if lang == want {
					bodies = append(bodies, dedent(m[2]))
					break
				}
			}
		}
		if len(bodies) == 0 {
			return "", false
		}
		// Several ```hcl fences in one entry form a single configuration, the
		// way ../remediation/code/terraform.py treats them. A second
		// CloudFormation or Bicep fence is an alternative example, so only the
		// first is used there.
		if suffix == "-terraform-hcl" {
			return strings.Join(bodies, "\n\n"), true
		}
		return bodies[0], true
	}
	return "", false
}

// dedent strips the common leading indentation the fence inherits from the
// surrounding `desc: |` block scalar. Bicep is indentation-sensitive once a
// nested object is involved, and YAML always is.
func dedent(block string) string {
	lines := strings.Split(block, "\n")
	common := -1
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			continue
		}
		n := len(l) - len(strings.TrimLeft(l, " "))
		if common == -1 || n < common {
			common = n
		}
	}
	if common <= 0 {
		return block
	}
	for i, l := range lines {
		if len(l) >= common {
			lines[i] = l[common:]
		} else {
			lines[i] = strings.TrimLeft(l, " ")
		}
	}
	return strings.Join(lines, "\n")
}

var hclResourcePrefix = regexp.MustCompile(`(?m)^\s*(?:resource|data)\s+"([a-z][a-z0-9]*)_`)

// A documentation snippet elides the parts that are not the point, and marks a
// placeholder the reader must replace. Neither is valid in the language, so both
// are removed before scanning — the same substitutions
// ../remediation/code/terraform.py makes before handing a snippet to tflint.
// Without this, `microsoft_defender { … }` inside a resource that also contains
// a bare `...` line reads as a parse failure and looks like a content defect.
var (
	ellipsisLine      = regexp.MustCompile(`(?m)^\s*\.\.\.\s*$`)
	quotedPlaceholder = regexp.MustCompile(`"<[^"<>\n]{1,60}>"`)
	barePlaceholder   = regexp.MustCompile(`<[^<>\n]{1,60}>`)
)

func sanitizeSnippet(code string) string {
	code = ellipsisLine.ReplaceAllString(code, "")
	code = quotedPlaceholder.ReplaceAllString(code, `"placeholder"`)
	return barePlaceholder.ReplaceAllString(code, `"placeholder"`)
}

// materialize writes a snippet into dir as the file its provider expects,
// adding only what the snippet needs to be a scannable document of its kind.
func materialize(dir, suffix, snippet string) (string, error) {
	method := remediationMethod[suffix]
	body := snippet
	if suffix == "-terraform-hcl" {
		// YAML and Bicep snippets are not sanitized: an ellipsis or an
		// angle-bracket placeholder is a *string* there, so it parses, and
		// rewriting it could change what the check reads.
		body = sanitizeSnippet(body)
		snippet = body
	}

	switch suffix {
	case "-terraform-hcl":
		// A snippet declares resources but no provider block. Most checks do not
		// care, but a filter that reads terraform.providers would see an empty
		// list and skip the check, which would look like a content failure.
		seen := map[string]bool{}
		var providers []string
		for _, m := range hclResourcePrefix.FindAllStringSubmatch(snippet, -1) {
			if !seen[m[1]] {
				seen[m[1]] = true
				providers = append(providers, m[1])
			}
		}
		sort.Strings(providers)
		var b strings.Builder
		for _, p := range providers {
			b.WriteString("provider \"" + p + "\" {}\n")
		}
		if b.Len() > 0 {
			body = b.String() + "\n" + snippet
		}
	case "-cloudformation":
		// A fragment starts at `Resources:`; the template needs its preamble.
		if !strings.Contains(snippet, "AWSTemplateFormatVersion") {
			body = "AWSTemplateFormatVersion: '2010-09-09'\n" + snippet
		}
	}

	path := filepath.Join(dir, method.filename)
	return path, os.WriteFile(path, []byte(body+"\n"), 0o644)
}

// scanSnippet runs the check against a materialised snippet, retrying a failed
// scan before believing it.
//
// Each scan spawns a provider subprocess, and under the suite's concurrency one
// occasionally dies mid-request: `rpc error: code = Unavailable desc = error
// reading from server: EOF`. That is contention, not a property of the snippet —
// it hit four checks in one run and two different ones in the next. Recorded as
// a finding it would put a flaky entry in the budget, and then fail the *next*
// run for "now satisfies it, remove the entry". A deterministic error, such as
// `asset doesn't support any policies` for a snippet that yields no scannable
// asset, reproduces on every attempt and is reported after the last one.
func scanSnippet(bundleFile, policyMrn string, asset *inventory.Asset) ([]*policy.Report, error) {
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		var reports []*policy.Report
		reports, err = runBundleReports(bundleFile, policyMrn, asset)
		if err == nil {
			return reports, nil
		}
	}
	return nil, err
}

// TestRemediationSatisfiesCheck scans each IaC variant's own remediation snippet
// and requires the check that recommends it to pass.
//
// Every variant in the corpus now satisfies its own remediation, so this is a
// flat assertion rather than a shrink-only debt budget: a snippet that stops
// closing its check fails here instead of being excused by an entry on a list.
func TestRemediationSatisfiesCheck(t *testing.T) {
	shardIndex, shardTotal := iacShard(t)

	for _, pol := range tfVariantPolicies {
		bundle, err := policy.DefaultBundleLoader().BundleFromPaths(bundlePath(pol.bundleFile))
		require.NoError(t, err)

		byUid := make(map[string]*policy.Mquery, len(bundle.Queries))
		for _, q := range bundle.Queries {
			byUid[q.Uid] = q
		}

		uids := make([]string, 0, len(bundle.Queries))
		for _, q := range bundle.Queries {
			uids = append(uids, q.Uid)
		}
		sort.Strings(uids)

		for _, uid := range uids {
			suffix, ok := iacSuffix(uid)
			if !ok {
				continue
			}
			parent := byUid[strings.TrimSuffix(uid, suffix)]
			snippet, ok := snippetFor(parent, suffix)
			if !ok {
				// No same-language remediation to test. Whether one is required
				// is TestTerraformVariantCoverage's and the content rules'
				// business, not this suite's.
				continue
			}
			if !inShard(uid, shardIndex, shardTotal) {
				continue
			}

			t.Run(uid, func(t *testing.T) {
				t.Parallel()
				dir := t.TempDir()
				_, err := materialize(dir, suffix, snippet)
				require.NoError(t, err)

				reports, err := scanSnippet(bundlePath(pol.bundleFile), pol.policyMrn, tfAssetForVariant(uid, dir))
				outcome := outcomeSkipped
				if err == nil {
					outcome, _ = checkResultAcross(reports, queryMrnPrefix+uid)
				}

				if err != nil {
					t.Fatalf("scanning the remediation snippet failed: %v\n"+
						"the snippet is not a scannable %s document on its own\ncheck: %s",
						err, strings.TrimPrefix(suffix, "-"), uid)
				}
				if outcome == outcomeSkipped {
					t.Fatalf("the check did not run against its own remediation snippet\n"+
						"the snippet does not declare a resource this check's filter matches, so "+
						"applying it as shown leaves the check unevaluated\ncheck: %s", uid)
				}
				if outcome != outcomePassed {
					t.Fatalf("the remediation for this check does not satisfy it\n"+
						"a reader who applies the documented fix still fails the check\ncheck: %s", uid)
				}
			})
		}
	}
}
