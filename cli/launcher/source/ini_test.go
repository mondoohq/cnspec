// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package source

import (
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/cockroachdb/errors"
)

// The scanner was two scanners, written a few months apart, and merging them is
// only safe if the merged one still parses everything either of them did. Each
// test below is a property one of the originals had; together they are the
// whole of what was merged.

func write(t *testing.T, name, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// The AWS reader's property: the two files are one namespace, and the section
// spelling differs between them.
//
// `strip` is per file rather than per scan for exactly this reason. Stripping
// "profile " from the credentials file as well would mangle a profile whose
// name begins with those eight characters, which is a thing an ini file may
// legally contain.
func TestTheProfilePrefixIsStrippedFromOnlyTheFileThatUsesIt(t *testing.T) {
	config := write(t, "config", "[profile prod]\n[default]\n")
	creds := write(t, "credentials", "[profile weirdly-named]\n[staging]\n")

	names, _ := awsProfileSections(config, creds)
	want := []string{"default", "prod", "profile weirdly-named", "staging"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("profiles = %v, want %v", names, want)
	}
}

// The AWS reader's other property: sso_account_id and account_id are two
// spellings of one fact, and the file's last word on it wins.
//
// This is the case that decided the shape of iniValues. The original wrote both
// keys into one map slot, so "last wins" was a property of the loop; a map
// keyed by name cannot express it, and reading sso_account_id in preference
// would flip the answer for the file below.
func TestTheLastAccountKeyWins(t *testing.T) {
	config := write(t, "config", strings.Join([]string{
		"[profile both]",
		"sso_account_id = 111111111111",
		"account_id = 222222222222",
		"[profile sso-first]",
		"account_id = 333333333333",
		"sso_account_id = 444444444444",
	}, "\n"))

	_, accounts := awsProfileSections(config, "")
	if got := accounts["both"].last(awsAccountKeys...); got != "222222222222" {
		t.Errorf("account = %q, want the one the file stated last", got)
	}
	if got := accounts["sso-first"].last(awsAccountKeys...); got != "444444444444" {
		t.Errorf("account = %q, want the one the file stated last", got)
	}
}

// And across the two files: the credentials file is read second, so it has the
// last word.
func TestTheCredentialsFileHasTheLastWordOnAnAccount(t *testing.T) {
	config := write(t, "config", "[profile prod]\naccount_id = 111111111111\n")
	creds := write(t, "credentials", "[prod]\naccount_id = 222222222222\n")

	_, accounts := awsProfileSections(config, creds)
	if got := accounts["prod"].last(awsAccountKeys...); got != "222222222222" {
		t.Errorf("account = %q, want the credentials file's", got)
	}
}

// The other original's property: the dialects. A TOML `[[table]]`, a quoted
// name and a trailing comment all resolve to the name.
func TestEveryDialectResolvesToTheName(t *testing.T) {
	for _, tc := range []struct{ line, want string }{
		{"[default]", "default"},
		{"[profile prod]", "profile prod"},
		{"[[connections.audit]]", "connections.audit"},
		{`["quoted name"]`, "quoted name"},
		{"[eu-audit]  ; read-only auditor", "eu-audit"},
		{"[  spaced  ]", "spaced"},
	} {
		got, ok := iniSectionName(tc.line, "")
		if !ok || got != tc.want {
			t.Errorf("iniSectionName(%q) = %q/%v, want %q", tc.line, got, ok, tc.want)
		}
	}
	// A header with no closing bracket is not a header, and must not become
	// one: a TOML multi-line string can start a line with `[`.
	if _, ok := iniSectionName("[unterminated", ""); ok {
		t.Error("an unterminated header was accepted")
	}
}

// The allowlist is the whole point, and it is what makes reading these files
// allowable at all: every one of them keeps live key material beside the names.
func TestNothingOutsideTheAllowlistIsHeld(t *testing.T) {
	path := write(t, "credentials", strings.Join([]string{
		"[prod]",
		"aws_secret_access_key = wJalrXUtnFEMI-must-never-be-held",
		"account_id = 123456789012",
		"aws_session_token = also-must-never-be-held",
	}, "\n"))

	_, values, err := iniSections(iniScan{
		files: iniPath(path),
		want:  map[string]bool{"account_id": true},
	})
	if err != nil {
		t.Fatal(err)
	}
	for section, kvs := range values {
		for _, kv := range kvs {
			if kv.key != "account_id" {
				t.Errorf("[%s] kept %q, which is not on the allowlist", section, kv.key)
			}
			if strings.Contains(kv.value, "must-never-be-held") {
				t.Fatalf("[%s] holds key material: %q", section, kv.value)
			}
		}
	}

	// With no allowlist at all, nothing inside a section is read.
	_, values, err = iniSections(iniScan{files: iniPath(path)})
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 0 {
		t.Errorf("a headers-only scan kept %d sections' values", len(values))
	}
}

// Comments never become sections or values, in either dialect's comment
// character.
func TestCommentsAreNotContent(t *testing.T) {
	path := write(t, "config", strings.Join([]string{
		"# [profile commented-out]",
		"; [profile also-commented]",
		"[profile real]",
		"# account_id = 999999999999",
		"account_id = 123456789012",
	}, "\n"))

	names, values, err := iniSections(iniScan{
		files: []iniFile{{path: path, strip: "profile "}},
		want:  map[string]bool{"account_id": true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(names, []string{"real"}) {
		t.Errorf("sections = %v, want just the uncommented one", names)
	}
	if got := values["real"].last("account_id"); got != "123456789012" {
		t.Errorf("account = %q", got)
	}
}

// A file that could not be opened is reported, because "no such file" and
// "nothing configured" are different answers and only one of them is something
// the user can act on.
func TestAnUnopenableFileIsReportedRatherThanEmpty(t *testing.T) {
	_, _, err := iniSections(iniScan{files: iniPath(filepath.Join(t.TempDir(), "absent"))})
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("a missing file gave %v", err)
	}
	// An empty path is the same answer, and is what a reader gets when there
	// is no home directory to resolve against.
	if _, _, err := iniSections(iniScan{files: iniPath("")}); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("an empty path gave %v", err)
	}
}

// One readable file among several is a working setup: ~/.aws/credentials alone
// is one, and so is ~/.aws/config alone.
func TestOneReadableFileIsEnough(t *testing.T) {
	creds := write(t, "credentials", "[staging]\n")
	names, _, err := iniSections(iniScan{files: []iniFile{
		{path: filepath.Join(t.TempDir(), "absent"), strip: "profile "},
		{path: creds},
	}})
	if err == nil {
		t.Error("the unreadable file was not reported")
	}
	if !reflect.DeepEqual(names, []string{"staging"}) {
		t.Errorf("sections = %v, want the readable file's", names)
	}
}

// The two profile readers were byte-identical and are now one, so the one has
// to answer for both dialects: the OCI file is ini-like and capitalises its
// conventional section, the Alibaba Cloud file is a true ini and does not.
func TestOneProfileReaderAnswersForBothFiles(t *testing.T) {
	oci, err := profileNamesFrom(ociFixture)
	if err != nil {
		t.Fatal(err)
	}
	ali, err := profileNamesFrom(alicloudFixture)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(oci, []string{"DEFAULT", "SANDBOX", "eu-audit"}) {
		t.Errorf("oci profiles = %v", oci)
	}
	if !reflect.DeepEqual(ali, []string{"default", "ecs", "ram-role", "staging"}) {
		t.Errorf("alicloud profiles = %v", ali)
	}
	// Neither reader ever held a value, which is why one function can serve
	// both without a decision about what to keep.
	for _, path := range []string{ociFixture, alicloudFixture} {
		_, values, err := iniSections(iniScan{files: iniPath(path)})
		if err != nil {
			t.Fatal(err)
		}
		if len(values) != 0 {
			t.Errorf("%s: the reader kept %d sections' values", path, len(values))
		}
	}
}

// The one place the merged scanner is wider than the AWS reader it replaced,
// pinned so the widening is a decision rather than a surprise.
//
// A header with anything after its closing bracket used to be no header at
// all, which left the section pointer on the previous profile -- so the keys
// under the malformed header were filed against the wrong name. The OCI
// dialect requires the other reading, and both readers are now one.
func TestAHeaderWithATrailingCommentNamesItsSection(t *testing.T) {
	config := write(t, "config", strings.Join([]string{
		"[profile prod]",
		"account_id = 111111111111",
		"[profile audit]  ; read-only",
		"account_id = 222222222222",
	}, "\n"))

	names, accounts := awsProfileSections(config, "")
	if !reflect.DeepEqual(names, []string{"audit", "prod"}) {
		t.Fatalf("profiles = %v, want both named", names)
	}
	if got := accounts["prod"].last(awsAccountKeys...); got != "111111111111" {
		t.Errorf("prod account = %q; the second profile's id leaked into it", got)
	}
	if got := accounts["audit"].last(awsAccountKeys...); got != "222222222222" {
		t.Errorf("audit account = %q", got)
	}
}

// A quoted value loses its quotes, which TOML requires. An AWS account id is
// never quoted, so nothing well-formed changes.
func TestQuotedValuesLoseTheirQuotes(t *testing.T) {
	path := write(t, "connections.toml", "[default]\naccount = \"myorg-prod\"\n")
	_, values, err := iniSections(iniScan{files: iniPath(path), want: snowflakeAccountKey})
	if err != nil {
		t.Fatal(err)
	}
	if got := values["default"].last("account"); got != "myorg-prod" {
		t.Errorf("account = %q, want it unquoted", got)
	}
}
