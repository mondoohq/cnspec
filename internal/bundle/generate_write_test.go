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
