// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/providers-sdk/v1/upstream/fex"
	"go.mondoo.com/mql/utils/iox"
)

func TestCsvConverter(t *testing.T) {
	rows := fex.VulnRows(sampleVEX())

	buf := bytes.Buffer{}
	writer := iox.IOWriter{Writer: &buf}
	require.NoError(t, VulnReportToCSV(rows, &writer))

	out := buf.String()
	// header
	assert.Contains(t, out, "Severity,Advisory,Package,Installed,Fixed,PURL,References,Remediation")
	// the critical row carries the richer VEX fields (severity, purl, fix, refs)
	assert.Contains(t, out, "CRITICAL,CVE-2022-0001,lodash,4.17.20,4.17.21,pkg:npm/lodash@4.17.20,https://nvd.nist.gov/vuln/detail/CVE-2022-0001,Upgrade lodash to 4.17.21")
}

func TestCsvConverterNeutralizesFormulas(t *testing.T) {
	// A scanned target controls package names; inject a spreadsheet formula
	// payload as an affected package name so it flows through to the CSV output.
	const payload = `=HYPERLINK("http://evil","click")`
	vex := []*fex.VulnerabilityExchange{{
		Id:      "CVE-2030-0001",
		Ratings: []*fex.Rating{{Severity: "high"}},
		Affects: []*fex.Affects{{
			Component: &fex.Component{Id: payload},
		}},
	}}
	rows := fex.VulnRows(vex)

	buf := bytes.Buffer{}
	writer := iox.IOWriter{Writer: &buf}
	require.NoError(t, VulnReportToCSV(rows, &writer))

	out := buf.String()
	// The payload must survive but be neutralized with a leading single quote,
	// and no CSV field may begin with the raw formula.
	assert.Contains(t, out, `'=HYPERLINK`)
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
