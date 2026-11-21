// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package bundle

import (
	"strings"
	"testing"
)

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"S3 buckets must be encrypted": "s3-buckets-must-be-encrypted",
		"  Trim  --  dashes  ":         "trim-dashes",
		"CIS 1.2: root/login!":         "cis-1-2-root-login",
		"":                             "",
	}
	for in, want := range cases {
		if got := Slugify(in); got != want {
			t.Errorf("Slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCheckUIDForTitle(t *testing.T) {
	// Synthetic: each title is chosen to land on one edge of the query uid rule.
	long := strings.Repeat("abcdefghij ", 30) // 330 characters
	cases := map[string]string{
		"S3 buckets must be encrypted": "s3-buckets-must-be-encrypted",
		"SSH":                          "ssh-check",
		"":                             "generated-check",
		"日本語のタイトル":                     "generated-check",
	}
	for in, want := range cases {
		if got := CheckUIDForTitle(in); got != want {
			t.Errorf("CheckUIDForTitle(%q) = %q, want %q", in, got, want)
		}
	}
	for _, in := range []string{"SSH", "", "日本語のタイトル", long, "S3 buckets must be encrypted"} {
		uid := CheckUIDForTitle(in)
		if !queryUid.MatchString(uid) {
			t.Errorf("CheckUIDForTitle(%q) = %q does not pass the lint uid rule", in, uid)
		}
		// a collision suffix must still lint
		if suffixed := NextFreeUID(uid, map[string]bool{uid: true}); !queryUid.MatchString(suffixed) {
			t.Errorf("NextFreeUID(%q) = %q does not pass the lint uid rule", uid, suffixed)
		}
	}
}

// TestAddCheckRequiresProvider: the policy a check joins lists the check's
// provider once, so the bundle lints without policy-missing-require and a scan
// installs the provider. Synthetic.
func TestAddCheckRequiresProvider(t *testing.T) {
	b := &Bundle{}
	for _, c := range []struct{ uid, provider string }{
		{"s3-encrypted", "aws"},
		{"s3-versioned", "aws"},
		{"ssh-root", "os"},
		{"unknown-target", ""},
	} {
		if err := AddCheck(b, "demo", c.uid, c.uid, "", "", "true", c.provider); err != nil {
			t.Fatal(err)
		}
	}
	var got []string
	for _, r := range b.Policies[0].Require {
		got = append(got, r.Provider)
	}
	if strings.Join(got, ",") != "aws,os" {
		t.Fatalf("require = %v, want [aws os]", got)
	}
	out, err := FormatBundle(b, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "provider: aws") {
		t.Errorf("require was not written:\n%s", out)
	}
}
