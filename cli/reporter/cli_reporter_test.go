// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v10/cli/printer"
	"go.mondoo.com/cnquery/v10/cli/theme/colors"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/upstream/mvd"
	"go.mondoo.com/cnquery/v10/shared"
	"go.mondoo.com/cnspec/v10/policy"
)

func TestCompactReporter(t *testing.T) {
	reportCollectionRaw, err := os.ReadFile("./testdata/report-ubuntu.json")
	require.NoError(t, err)

	yr := &policy.ReportCollection{}
	err = json.Unmarshal(reportCollectionRaw, yr)
	require.NoError(t, err)

	buf := bytes.Buffer{}
	writer := shared.IOWriter{Writer: &buf}

	r := &Reporter{
		Format:  Compact,
		Printer: &printer.DefaultPrinter,
		Colors:  &colors.DefaultColorTheme,
	}
	rr := &defaultReporter{
		Reporter:  r,
		isCompact: true,
		output:    &writer,
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

	r := NewReporter(Summary, false)
	r.out = &writer
	require.NoError(t, err)

	target := "index.docker.io/library/ubuntu@669e010b58ba"
	err = r.PrintVulns(report, target)
	require.NoError(t, err)

	r = NewReporter(Compact, false)
	r.out = &writer
	err = r.PrintVulns(report, target)
	require.NoError(t, err)

	assert.Contains(t, buf.String(), "5.5    libblkid1       2.34-0.1ubuntu9.1")
	assert.NotContains(t, buf.String(), "USN-5279-1")

	r = NewReporter(Full, false)
	r.out = &writer
	require.NoError(t, err)

	err = r.PrintVulns(report, target)
	require.NoError(t, err)

	assert.Contains(t, buf.String(), "5.5    libblkid1       2.34-0.1ubuntu9.1")
	assert.Contains(t, buf.String(), "USN-5279-1")

	r = NewReporter(YAML, false)
	r.out = &writer
	require.NoError(t, err)

	err = r.PrintVulns(report, target)
	require.NoError(t, err)

	assert.Contains(t, buf.String(), "score: 5.5")
	assert.Contains(t, buf.String(), "package: libblkid1")
	assert.Contains(t, buf.String(), "installed: 2.34-0.1ubuntu9.1")
	assert.Contains(t, buf.String(), "advisory: USN-5279-1")
}

func TestJsonOutput(t *testing.T) {
	reportCollectionRaw, err := os.ReadFile("./testdata/report-ubuntu.json")
	require.NoError(t, err)

	yr := &policy.ReportCollection{}
	err = json.Unmarshal(reportCollectionRaw, yr)
	require.NoError(t, err)

	buf := bytes.Buffer{}
	writer := shared.IOWriter{Writer: &buf}

	r := &Reporter{
		Format:  JSON,
		Printer: &printer.DefaultPrinter,
		Colors:  &colors.DefaultColorTheme,
		out:     &writer,
	}

	err = r.WriteReport(context.Background(), yr)
	require.NoError(t, err)
	valid := json.Valid(buf.Bytes())
	require.True(t, valid)

	assert.Contains(t, buf.String(), "//policy.api.mondoo.app/queries/mondoo-linux-security-permissions-on-etcgshadow-are-configured\":{\"score\":100,\"status\":\"pass\"}")
	assert.Contains(t, buf.String(), "\"errors\":{}")
}

func TestJsonOutputOnlyErrors(t *testing.T) {
	reportCollectionRaw, err := os.ReadFile("./testdata/report-k8s.json")
	require.NoError(t, err)

	yr := &policy.ReportCollection{}
	err = json.Unmarshal(reportCollectionRaw, yr)
	require.NoError(t, err)

	buf := bytes.Buffer{}
	writer := shared.IOWriter{Writer: &buf}

	r := &Reporter{
		Format:  JSON,
		Printer: &printer.DefaultPrinter,
		Colors:  &colors.DefaultColorTheme,
		out:     &writer,
	}

	err = r.WriteReport(context.Background(), yr)
	require.NoError(t, err)
	valid := json.Valid(buf.Bytes())
	require.True(t, valid)

	assert.NotContains(t, buf.String(), "{\"score\":100,\"status\":\"pass\"}")
	assert.NotContains(t, buf.String(), "\"errors\":{}\"")

	assert.Contains(t, buf.String(), "\"data\":{},\"scores\":{},\"errors\":{\"//policy")
}
