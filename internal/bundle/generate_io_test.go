// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package bundle

import (
	"strings"
	"testing"
)

const genIOFixture = `# managed policy bundle
policies:
  - uid: example-policy
    name: Example
    groups:
      - filters: asset.family.contains("linux")
        checks:
          - uid: inline-check
            title: Inline check
queries:
  - uid: needs-mql
    title: S3 buckets must be encrypted
    filters: asset.platform == "aws"
    docs:
      desc: All S3 buckets must have server-side encryption enabled.
  - uid: has-mql
    title: Already done
    mql: |
      aws.s3.buckets.all(encryption != empty)
`

func TestAllQueriesAndWriteBack(t *testing.T) {
	b, err := ParseYaml([]byte(genIOFixture))
	if err != nil {
		t.Fatalf("ParseYaml: %v", err)
	}

	queries := AllQueries(b)
	// top-level: needs-mql, has-mql ; group check: inline-check
	byUID := map[string]*Mquery{}
	for _, q := range queries {
		byUID[q.Uid] = q
	}
	for _, want := range []string{"needs-mql", "has-mql", "inline-check"} {
		if byUID[want] == nil {
			t.Fatalf("AllQueries missing %q (got %d queries)", want, len(queries))
		}
	}

	// intent extraction
	if got := QueryDesc(byUID["needs-mql"]); !strings.Contains(got, "server-side encryption") {
		t.Fatalf("QueryDesc = %q", got)
	}
	if got := QueryFilterStrings(byUID["needs-mql"]); len(got) != 1 || !strings.Contains(got[0], "aws") {
		t.Fatalf("QueryFilterStrings = %v", got)
	}

	// write MQL back and re-format
	byUID["needs-mql"].Mql = "aws.s3.buckets.all(encryption != empty)"
	out, err := FormatBundle(b, false)
	if err != nil {
		t.Fatalf("FormatBundle: %v", err)
	}
	s := string(out)
	if !strings.Contains(s, "aws.s3.buckets.all(encryption != empty)") {
		t.Fatalf("formatted output missing generated mql:\n%s", s)
	}
	// document-level comment preserved (same guarantee as `cnspec policy format`)
	if !strings.Contains(s, "# managed policy bundle") {
		t.Fatalf("document comment not preserved:\n%s", s)
	}
	if strings.Count(s, "aws.s3.buckets.all(encryption != empty)") != 2 {
		t.Fatalf("expected both queries to carry the mql, got:\n%s", s)
	}
}
