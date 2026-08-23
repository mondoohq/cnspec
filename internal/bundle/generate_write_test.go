// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package bundle

import "testing"

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
