// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers-sdk/v1/upstream/mvd"
	"go.mondoo.com/mql/v13/utils/iox"
)

func TestCsvConverter(t *testing.T) {
	reportRaw, err := os.ReadFile("./testdata/mondoo-debug-vulnReport.json")
	require.NoError(t, err)

	report := &mvd.VulnReport{}
	err = json.Unmarshal(reportRaw, report)
	require.NoError(t, err)

	buf := bytes.Buffer{}
	writer := iox.IOWriter{Writer: &buf}
	err = VulnReportToCSV(report, &writer)
	require.NoError(t, err)

	assert.Contains(t, buf.String(), "libblkid1,5.5,2.34-0.1ubuntu9.1,2.34-0.1ubuntu9.3,2.34-0.1ubuntu9.3,USN-5279-1,CVE-2021-3995 CVE-2021-3996")
}

func TestCsvConverterNeutralizesFormulas(t *testing.T) {
	reportRaw, err := os.ReadFile("./testdata/mondoo-debug-vulnReport.json")
	require.NoError(t, err)

	report := &mvd.VulnReport{}
	err = json.Unmarshal(reportRaw, report)
	require.NoError(t, err)

	// A scanned target controls package names; inject a spreadsheet formula
	// payload into an affected package so it flows through to the CSV output.
	const payload = `=HYPERLINK("http://evil","click")`
	found := false
	for _, pkg := range report.Packages {
		if pkg.Name == "libblkid1" {
			pkg.Name = payload
			found = true
		}
	}
	require.True(t, found, "expected to find a package to mutate")

	buf := bytes.Buffer{}
	writer := iox.IOWriter{Writer: &buf}
	err = VulnReportToCSV(report, &writer)
	require.NoError(t, err)

	out := buf.String()
	// The payload must survive but be neutralized with a leading single quote,
	// and no CSV field may begin with the raw formula.
	assert.Contains(t, out, `'=HYPERLINK`)
	assert.NotContains(t, out, "\n="+`HYPERLINK`)
	for _, line := range strings.Split(out, "\n") {
		assert.False(t, strings.HasPrefix(line, "="), "line begins with a formula: %q", line)
	}
}

func TestEscapeCSVCell(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "empty", input: "", expected: ""},
		{name: "plain", input: "libblkid1", expected: "libblkid1"},
		{name: "equals formula", input: "=HYPERLINK(\"http://evil\")", expected: "'=HYPERLINK(\"http://evil\")"},
		{name: "plus", input: "+1+1", expected: "'+1+1"},
		{name: "minus", input: "-2+3", expected: "'-2+3"},
		{name: "at", input: "@SUM(A1)", expected: "'@SUM(A1)"},
		{name: "tab", input: "\t=1", expected: "'\t=1"},
		{name: "carriage return", input: "\r=1", expected: "'\r=1"},
		{name: "formula char not leading", input: "a=b", expected: "a=b"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, escapeCSVCell(tt.input))
		})
	}
}
