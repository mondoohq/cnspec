// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Package branches holds a static check on the git branch names written into
// the shipped `audit:` and `remediation:` snippets. Like ../icons it only reads
// the bundle files, so it needs no providers and runs in the default
// `go test ./...`.
package branches

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

var contentGlobs = []string{
	"../../*.mql.yaml",
	"../../querypacks/*.mql.yaml",
}

var fence = regexp.MustCompile("(?s)```[^\n]*\n.*?```")

// branchLiterals match a git branch named `main` or `master` inside a code
// block. Each pattern is tied to a branch-shaped position (a branch argument,
// a ref, a protection pattern) so that `main` as a Terraform resource label or
// a PostgreSQL cluster directory does not match.
var branchLiterals = []*regexp.Regexp{
	regexp.MustCompile(`/branches/(main|master)\b`),
	regexp.MustCompile(`refs/heads/(main|master)\b`),
	regexp.MustCompile(`--branch[ =](main|master)\b`),
	regexp.MustCompile(`-f (name|branch)=(main|master)\b`),
	regexp.MustCompile(`\b(branch|pattern|production_branch|target_branch|source_branch)\s*[:=]\s*["']?(main|master)["']?(\s|$)`),
}

// callout is the sentence that tells the reader to substitute their own
// branch, such as "Replace `main` with the name of your default branch" or
// "Set `pattern` to the name of your default branch".
var callout = regexp.MustCompile(`(?i)\b(replace|set)\b[^.\n]*\b(default|production) branch\b`)

// TestBranchNamesAreCalledOut requires every audit or remediation block whose
// code names a `main` or `master` branch to tell the reader to substitute
// their own branch name.
//
// Repositories do not all use `main`. Pasted as-is, a snippet that names it
// protects, audits or deploys a branch that is not the repository's default,
// or one that does not exist, and the finding stays open. A Terraform
// branch-protection resource applied with the wrong `pattern` creates a rule
// on the wrong branch without an error, and a `production_branch` value
// changes which branch the site deploys from.
func TestBranchNamesAreCalledOut(t *testing.T) {
	var offenders []string
	for _, f := range contentFiles(t) {
		doc := readBundle(t, f)
		queries, _ := doc["queries"].([]any)
		for _, q := range queries {
			qm, _ := q.(map[string]any)
			uid, _ := qm["uid"].(string)
			docs, _ := qm["docs"].(map[string]any)
			if docs == nil {
				continue
			}
			check := func(where, text string) {
				if lit := branchLiteral(text); lit != "" && !callout.MatchString(fence.ReplaceAllString(text, "")) {
					offenders = append(offenders, fmt.Sprintf("%s: %s %s names %q", filepath.Base(f), uid, where, lit))
				}
			}
			if audit, ok := docs["audit"].(string); ok {
				check("audit", audit)
			}
			rems, _ := docs["remediation"].([]any)
			for _, r := range rems {
				rm, _ := r.(map[string]any)
				id, _ := rm["id"].(string)
				desc, _ := rm["desc"].(string)
				check("remediation "+id, desc)
			}
		}
	}

	if len(offenders) > 0 {
		sort.Strings(offenders)
		t.Errorf("a snippet that names a git branch needs a sentence such as "+
			"\"Replace `main` with the name of your default branch\" outside the code block:\n  %s",
			strings.Join(offenders, "\n  "))
	}
}

// branchLiteral returns the first branch literal found inside a code block of
// text, or "" when there is none.
func branchLiteral(text string) string {
	for _, block := range fence.FindAllString(text, -1) {
		for _, re := range branchLiterals {
			if m := re.FindString(block); m != "" {
				return strings.TrimSpace(m)
			}
		}
	}
	return ""
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
	// A glob that silently matched nothing would make this test pass while
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
