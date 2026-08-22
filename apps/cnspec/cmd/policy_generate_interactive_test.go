// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.mondoo.com/cnspec/v13/internal/bundle"
)

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"S3 buckets must be encrypted": "s3-buckets-must-be-encrypted",
		"  Trim  --  dashes  ":         "trim-dashes",
		"CIS 1.2: root/login!":         "cis-1-2-root-login",
		"":                             "",
	}
	for in, want := range cases {
		if got := slugify(in); got != want {
			t.Errorf("slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestGuessProviderAndFilter(t *testing.T) {
	if p := guessProvider("S3 buckets must be encrypted"); p != "aws" {
		t.Errorf("aws guess = %q", p)
	}
	if p := guessProvider("SSH root login disabled"); p != "os" {
		t.Errorf("os guess = %q", p)
	}
	if p := guessProvider("Something generic"); p != "" {
		t.Errorf("expected no guess, got %q", p)
	}
	if f := defaultFilter("aws"); f != `asset.platform == "aws"` {
		t.Errorf("aws filter = %q", f)
	}
	if f := defaultFilter("os"); f != `asset.family.contains("linux")` {
		t.Errorf("os filter = %q", f)
	}
	if f := defaultFilter(""); f != "" {
		t.Errorf("empty provider filter = %q", f)
	}
}

func TestAppendCheck(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "policy.mql.yaml")

	// create + first append
	if err := appendCheck(file, "c1", "First check", "does a thing", `asset.platform == "aws"`, "aws.s3.buckets.length > 0"); err != nil {
		t.Fatalf("appendCheck 1: %v", err)
	}
	// append a second into the existing file
	if err := appendCheck(file, "c2", "Second check", "", "", "users.all(uid >= 0)"); err != nil {
		t.Fatalf("appendCheck 2: %v", err)
	}

	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)

	// scalar filter must round-trip (not the map form)
	if !strings.Contains(s, `filters: asset.platform == "aws"`) {
		t.Errorf("filter not scalar-serialized:\n%s", s)
	}

	// both checks present and parseable, with the right fields
	b, err := bundle.ParseYaml(data)
	if err != nil {
		t.Fatalf("re-parse: %v", err)
	}
	byUID := map[string]*bundle.Mquery{}
	for _, q := range bundle.AllQueries(b) {
		byUID[q.Uid] = q
	}
	if byUID["c1"] == nil || byUID["c2"] == nil {
		t.Fatalf("expected c1 and c2, got %v", func() []string {
			var ks []string
			for k := range byUID {
				ks = append(ks, k)
			}
			return ks
		}())
	}
	if got := byUID["c1"].Mql; got != "aws.s3.buckets.length > 0" {
		t.Errorf("c1 mql = %q", got)
	}
	if byUID["c1"].Docs == nil || byUID["c1"].Docs.Desc != "does a thing" {
		t.Errorf("c1 desc missing")
	}
	if byUID["c2"].Filters != nil {
		t.Errorf("c2 should have no filter")
	}
}
