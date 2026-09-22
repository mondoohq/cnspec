// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build integration

package integration

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/cli/reporter"
	"google.golang.org/protobuf/encoding/protojson"
	"sigs.k8s.io/yaml"
)

// fsFixture is the filesystem fixture test/providers scans: a small fake Debian
// root. Reused here rather than a container because the format matrix cares
// about the reporter, not the target, and this is the fastest way to produce a
// real report with real findings.
const fsFixture = "../providers/testdata/fs"

// TestOutputFormats checks that every format cnspec advertises for a scan
// actually produces a parseable document.
//
// cli/reporter has unit tests for each format against fixture reports. What
// those cannot cover is the wiring: -o <name> -> format lookup -> output
// handler -> stdout. That path is one switch statement away from a format that
// errors at runtime for every user. `-o csv` did exactly that until it was
// implemented: it was in the format map, so `cnspec scan --help` listed it,
// but writing a scan report in it fell through to the default branch and the
// command exited 1 with "unknown reporter type".
func TestOutputFormats(t *testing.T) {
	formats := []struct {
		flag   string
		verify func(t *testing.T, out []byte)
	}{
		{"json", func(t *testing.T, out []byte) {
			rep := &reporter.Report{}
			opts := protojson.UnmarshalOptions{DiscardUnknown: true}
			require.NoError(t, opts.Unmarshal(out, rep))
			assert.NotEmpty(t, rep.GetAssets())
		}},
		{"json-v1", func(t *testing.T, out []byte) {
			// Hand-written, not a proto, and the only format that still carries
			// the absolute score: the v2 proto reserved that field and emits
			// riskScore instead.
			var doc struct {
				Assets map[string]json.RawMessage `json:"assets"`
				Scores map[string]json.RawMessage `json:"scores"`
			}
			require.NoError(t, json.Unmarshal(out, &doc))
			assert.NotEmpty(t, doc.Assets)
			assert.NotEmpty(t, doc.Scores)
		}},
		{"yaml-v2", func(t *testing.T, out []byte) {
			var doc map[string]any
			require.NoError(t, yaml.Unmarshal(out, &doc))
			assert.Contains(t, doc, "assets")
		}},
		{"junit", func(t *testing.T, out []byte) {
			var suites struct {
				XMLName xml.Name `xml:"testsuites"`
				Suites  []struct {
					Name  string `xml:"name,attr"`
					Tests int    `xml:"tests,attr"`
				} `xml:"testsuite"`
			}
			require.NoError(t, xml.Unmarshal(out, &suites))
			require.NotEmpty(t, suites.Suites, "no test suites in the JUnit report")
		}},
		{"sarif", func(t *testing.T, out []byte) {
			var doc struct {
				Version string            `json:"version"`
				Runs    []json.RawMessage `json:"runs"`
			}
			require.NoError(t, json.Unmarshal(out, &doc))
			assert.Equal(t, "2.1.0", doc.Version)
			assert.NotEmpty(t, doc.Runs)
		}},
		{"ocsf-json", func(t *testing.T, out []byte) {
			// Newline-delimited JSON, one OCSF finding per line -- not a single
			// document. Asserted explicitly because a change to a JSON array
			// would break every consumer downstream and is otherwise silent.
			requireNDJSON(t, out)
		}},
		{"csv", func(t *testing.T, out []byte) {
			// One row per asset x check. encoding/csv enforces a consistent
			// column count across records, so a converter that emitted a
			// ragged row fails here rather than in someone's spreadsheet.
			records, err := csv.NewReader(bytes.NewReader(out)).ReadAll()
			require.NoError(t, err)
			require.Greater(t, len(records), 1, "header only: no check was exported")
			assert.Equal(t, "Asset", records[0][0])
			assert.Equal(t, "Status", records[0][6])
		}},
		{"hdf", func(t *testing.T, out []byte) {
			var doc map[string]any
			require.NoError(t, json.Unmarshal(out, &doc))
			assert.Contains(t, doc, "platform")
		}},
	}

	for _, f := range formats {
		t.Run(f.flag, func(t *testing.T) {
			res := run(t, 5*time.Minute,
				"scan", "filesystem", "--path", fsFixture,
				"-f", linuxSecurity, "--detect-cicd=false", "-o", f.flag)
			if res.exitCode != 0 {
				res.saveArtifact(t, "format-"+f.flag)
				t.Fatalf("exit code %d, want 0\n%s", res.exitCode, res.dump())
			}
			require.NotEmpty(t, res.stdout, "format produced no output")
			f.verify(t, res.stdout)
			if t.Failed() {
				res.saveArtifact(t, "format-"+f.flag)
			}
		})
	}
}

// requireNDJSON asserts every non-empty line is its own JSON value.
func requireNDJSON(t *testing.T, out []byte) {
	t.Helper()
	var lines int
	for _, line := range splitLines(out) {
		if len(line) == 0 {
			continue
		}
		var v any
		if err := json.Unmarshal(line, &v); err != nil {
			t.Fatalf("line %d is not valid JSON: %v\n%.200s", lines+1, err, line)
		}
		lines++
	}
	require.NotZero(t, lines, "no records in the report")
}

func splitLines(b []byte) [][]byte {
	var out [][]byte
	start := 0
	for i, c := range b {
		if c == '\n' {
			out = append(out, trimCR(b[start:i]))
			start = i + 1
		}
	}
	if start < len(b) {
		out = append(out, trimCR(b[start:]))
	}
	return out
}

func trimCR(b []byte) []byte {
	if len(b) > 0 && b[len(b)-1] == '\r' {
		return b[:len(b)-1]
	}
	return b
}
