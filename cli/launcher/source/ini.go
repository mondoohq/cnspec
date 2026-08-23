// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package source

import (
	"bufio"
	"io/fs"
	"os"
	"strings"
)

// The one ini scanner, for the four credential files the launcher reads names
// out of: ~/.aws/config, ~/.aws/credentials, ~/.oci/config,
// ~/.alibabacloud/credentials and ~/.snowflake/connections.toml.
//
// There used to be two, hand-rolled a few months apart, and the second one
// carefully re-derived the first one's rule: **read the section headers, hold
// no values**. Every file this is pointed at keeps live access keys, private
// key passphrases or bearer tokens next to the names being read, and an ini or
// TOML library would pull all of them into this process to answer a question
// about headers. That is why this is a scanner rather than a parse, and why
// the keys it will read at all are an allowlist the caller passes in --
// sso_account_id, account_id, account -- each of which is a locator rather
// than a credential.
//
// It is tolerant enough for every dialect involved: the AWS `[profile name]`
// spelling, a TOML `[[table]]`, a quoted table name, and a trailing `; comment`
// after a header all resolve to the name. What it does not model is a TOML
// multi-line string whose body contains a line starting with `[`, which would
// be read as a section -- harmless here, because a section invented that way
// has no allowlisted key under it and so contributes nothing.
//
// Two of those tolerances are wider than the AWS reader used to be, and both
// only show on a file the AWS CLI would itself reject:
//
//   - A header with anything after its closing bracket used to be no header at
//     all, which left the section pointer on the *previous* profile and
//     attached that profile's account id to the wrong name. Now it resolves to
//     the name, the way ~/.oci/config's `[eu-audit]  ; read-only auditor`
//     always had to.
//   - A quoted value loses its quotes, which TOML requires and which an AWS
//     file has no reason to carry.
//
// A well-formed shared config parses identically either way: a header that
// ends in `]` has one closing bracket, and an account id is not quoted.

// iniFile is one file to walk, and what its section headers spell.
type iniFile struct {
	path string
	// strip comes off the front of a section name, and is a per-file property
	// rather than a per-scan one: ~/.aws/config spells its sections
	// `[profile prod]` while ~/.aws/credentials spells the same profile
	// `[prod]`, and it is the same profile.
	strip string
}

// iniScan says which files to walk and what may be read out of them.
type iniScan struct {
	// files are read in order, and a section already named by an earlier file
	// keeps its first position. Only the AWS reader passes more than one:
	// ~/.aws/config and ~/.aws/credentials are one namespace split across two
	// files, and reading either alone hides profiles the CLI can see.
	files []iniFile
	// want is the whole of what may be read from inside a section. An empty
	// allowlist means headers only, which is what three of the four readers
	// need.
	want map[string]bool
}

// iniPath is the common case: one file whose headers are spelled plainly.
func iniPath(path string) []iniFile { return []iniFile{{path: path}} }

// iniAssign is one allowlisted key=value, kept in the order the file made it.
//
// Order is why this is a slice and not a map. The AWS reader treats
// sso_account_id and account_id as two spellings of one fact and takes
// whichever the file stated last; a map keyed by name cannot answer that, and
// the reader that used to do this by hand got it right by accident of writing
// both keys into the same slot.
type iniAssign struct {
	key, value string
}

// iniValues are the allowlisted assignments under one section.
type iniValues []iniAssign

// last returns the value of the last assignment to any of keys, or "".
func (v iniValues) last(keys ...string) string {
	for i := len(v) - 1; i >= 0; i-- {
		for _, k := range keys {
			if v[i].key == k {
				return v[i].value
			}
		}
	}
	return ""
}

// iniSections walks the declared files and returns the section names in file
// order, plus the allowlisted values under each.
//
// A file that cannot be opened is reported rather than skipped, because "no
// such file" and "nothing configured" are different answers and a picker that
// shows an empty list for a file it never opened is the failure the Source
// contract exists to stop. With several paths, the first failure is what is
// returned and the rest are still read: ~/.aws/credentials alone is a working
// setup, and so is ~/.aws/config alone.
func iniSections(scan iniScan) ([]string, map[string]iniValues, error) {
	var names []string
	values := map[string]iniValues{}
	seen := map[string]bool{}
	var firstErr error

	for _, file := range scan.files {
		if file.path == "" {
			if firstErr == nil {
				firstErr = fs.ErrNotExist
			}
			continue
		}
		f, err := os.Open(file.path)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		section := ""
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
				continue
			}
			if strings.HasPrefix(line, "[") {
				name, ok := iniSectionName(line, file.strip)
				if !ok {
					continue
				}
				section = name
				if section != "" && !seen[section] {
					seen[section] = true
					names = append(names, section)
				}
				continue
			}
			if section == "" || len(scan.want) == 0 {
				continue
			}
			key, value, ok := strings.Cut(line, "=")
			if key = strings.TrimSpace(key); !ok || !scan.want[key] {
				continue
			}
			values[section] = append(values[section], iniAssign{
				key:   key,
				value: strings.Trim(strings.TrimSpace(value), `"'`),
			})
		}
		f.Close()
		// A read that died part way still knows about the sections it
		// reached, and a short list is more use than none.
		if err := sc.Err(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return names, values, firstErr
}

// iniSectionName reads a section header. The name runs to the first closing
// bracket, so a trailing comment does not become part of it; the extra Trim
// handles TOML's `[[array]]`, whose first `]` leaves one bracket behind.
func iniSectionName(line, strip string) (string, bool) {
	end := strings.IndexByte(line, ']')
	if end < 0 {
		return "", false
	}
	name := strings.Trim(strings.TrimSpace(strings.Trim(line[1:end], "[]")), `"'`)
	return strings.TrimPrefix(name, strip), true
}
