// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/cli/printer"
	"go.mondoo.com/cnquery/cli/theme/colors"
	"go.mondoo.com/cnquery/providers-sdk/v1/upstream/mvd"
	"go.mondoo.com/cnquery/shared"
	"go.mondoo.com/cnspec/policy"
)

func TestJunitConverter(t *testing.T) {
	reportCollectionRaw, err := os.ReadFile("./testdata/report-ubuntu.json")
	require.NoError(t, err)

	yr := &policy.ReportCollection{}
	err = json.Unmarshal(reportCollectionRaw, yr)
	require.NoError(t, err)

	buf := bytes.Buffer{}
	writer := shared.IOWriter{Writer: &buf}

	r := &Reporter{
		Format:  Formats["compact"],
		Printer: &printer.DefaultPrinter,
		Colors:  &colors.DefaultColorTheme,
	}

	rr := &defaultReporter{
		Reporter:  r,
		isCompact: true,
		out:       &writer,
		data:      yr,
	}
	rr.print()

	assert.Contains(t, buf.String(), "✕ Fail:         Ensure")
	assert.Contains(t, buf.String(), ". Skipped:      Set")
	assert.Contains(t, buf.String(), "! Error:        Set")
	assert.Contains(t, buf.String(), "✓ Pass:  A 100  Ensure")
	assert.Contains(t, buf.String(), "✕ Fail:  F   0  Ensure")
}

func TestVulnReporter(t *testing.T) {
	reportRaw, err := os.ReadFile("./testdata/mondoo-debug-vulnReport.json")
	require.NoError(t, err)

	report := &mvd.VulnReport{}
	err = json.Unmarshal(reportRaw, report)
	require.NoError(t, err)

	buf := bytes.Buffer{}
	writer := shared.IOWriter{Writer: &buf}

	r, err := New("summary")
	require.NoError(t, err)

	target := "index.docker.io/library/ubuntu@669e010b58ba"
	err = r.PrintVulns(report, &writer, target)
	require.NoError(t, err)

	r, err = New("compact")
	require.NoError(t, err)

	err = r.PrintVulns(report, &writer, target)
	require.NoError(t, err)

	assert.Contains(t, buf.String(), "5.5    libblkid1       2.34-0.1ubuntu9.1")
	assert.NotContains(t, buf.String(), "USN-5279-1")

	r, err = New("full")
	require.NoError(t, err)

	err = r.PrintVulns(report, &writer, target)
	require.NoError(t, err)

	assert.Contains(t, buf.String(), "5.5    libblkid1       2.34-0.1ubuntu9.1")
	assert.Contains(t, buf.String(), "USN-5279-1")

	r, err = New("yaml")
	require.NoError(t, err)

	err = r.PrintVulns(report, &writer, target)
	require.NoError(t, err)

	assert.Contains(t, buf.String(), "score: 5.5")
	assert.Contains(t, buf.String(), "package: libblkid1")
	assert.Contains(t, buf.String(), "installed: 2.34-0.1ubuntu9.1")
	assert.Contains(t, buf.String(), "advisory: USN-5279-1")
}
