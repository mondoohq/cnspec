// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package generate

import (
	"strings"
	"testing"
)

func TestSanitizeModelText(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"plain text is untouched", "aws.s3.buckets.all(encryption != empty)", "aws.s3.buckets.all(encryption != empty)"},
		{"newline and tab survive", "a == 1\n\tb == 2", "a == 1\n\tb == 2"},
		{
			// the spoof: the escape sequences erase the line and return the
			// cursor, so a terminal shows only what follows them
			"ansi erase sequence is escaped",
			"asset.name != 'safe-looking'\x1b[2K\x1b[1GBACKDOOR",
			`asset.name != 'safe-looking'\x1b[2K\x1b[1GBACKDOOR`,
		},
		// a lone CR would return the cursor and let "malicious" overwrite
		// "safe"; as a line break both stay on screen
		{"lone carriage return becomes a line break", "safe\rmalicious", "safe\nmalicious"},
		{"crlf becomes lf", "a == 1\r\nb == 2\r\n", "a == 1\nb == 2\n"},
		{"bell and delete are escaped", "a\x07b\x7fc", `a\x07b\x7fc`},
		{"c1 controls are escaped", "a\u0085b", `a\u0085b`},
		{"bidi override is escaped", "a\u202eb", `a\u202eb`},
		{"zero width space is escaped", "a\u200bb", `a\u200bb`},
		{"non-ascii text survives", "héllo — wörld ✓", "héllo — wörld ✓"},
		{"invalid utf8 byte is escaped", "a\xffb", `a\xffb`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := SanitizeModelText(tc.in)
			if got != tc.want {
				t.Fatalf("SanitizeModelText(%q) = %q, want %q", tc.in, got, tc.want)
			}
			if strings.ContainsRune(got, 0x1b) {
				t.Fatalf("result still contains ESC: %q", got)
			}
		})
	}
}

// TestParseResponseSanitizesModelText pins the reason the sanitizer exists: the
// query the reviewer reads must be the query that gets stored. A model answer
// carrying terminal escapes displayed as one thing and contained another, and it
// passed the compile gate because the escapes sat inside a string literal.
func TestParseResponseSanitizesModelText(t *testing.T) {
	raw := "```json\n{\"mql\": \"asset.name != 'safe\\u001b[2K\\u001b[1G BACKDOOR'\", \"explanation\": \"looks\\u001b[2K fine\"}\n```"

	res := parseResponse(raw)
	if strings.ContainsRune(res.MQL, 0x1b) {
		t.Fatalf("parsed MQL still carries an escape sequence: %q", res.MQL)
	}
	if !strings.Contains(res.MQL, "BACKDOOR") {
		t.Fatalf("the hidden part of the query must stay visible, got %q", res.MQL)
	}
	if strings.ContainsRune(res.Explanation, 0x1b) {
		t.Fatalf("parsed explanation still carries an escape sequence: %q", res.Explanation)
	}
}

// TestParseResponseNormalizesCRLF covers an agent that answers with Windows line
// endings. Synthetic: no agent run was captured; the bytes are what a CRLF
// writer produces for a fenced answer.
func TestParseResponseNormalizesCRLF(t *testing.T) {
	// two lines, so the CR sits inside the query where TrimSpace cannot reach it
	line1 := `sshd.config.params["PermitRootLogin"] == "no"`
	line2 := `sshd.config.params["PasswordAuthentication"] == "no"`
	want := line1 + "\n" + line2
	esc := func(s string) string { return strings.ReplaceAll(s, `"`, `\"`) }
	for name, raw := range map[string]string{
		"fenced": "```mql\r\n" + line1 + "\r\n" + line2 + "\r\n```\r\n",
		"json":   "```json\r\n{\"mql\": \"" + esc(line1) + "\\r\\n" + esc(line2) + "\", \"explanation\": \"x\"}\r\n```",
	} {
		t.Run(name, func(t *testing.T) {
			res := parseResponse(raw)
			if strings.TrimSpace(res.MQL) != want {
				t.Fatalf("parsed MQL = %q, want %q", res.MQL, want)
			}
			if strings.Contains(res.MQL, `\x0d`) || strings.ContainsRune(res.MQL, '\r') {
				t.Fatalf("parsed MQL still carries a carriage return: %q", res.MQL)
			}
		})
	}
}
