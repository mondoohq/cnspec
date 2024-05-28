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
	"go.mondoo.com/cnquery/v11/cli/printer"
	"go.mondoo.com/cnquery/v11/cli/theme/colors"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/upstream/mvd"
	"go.mondoo.com/cnquery/v11/utils/iox"
	"go.mondoo.com/cnspec/v11/policy"
)

func TestCompactReporter(t *testing.T) {
	reportCollectionRaw, err := os.ReadFile("./testdata/report-ubuntu.json")
	require.NoError(t, err)

	yr := &policy.ReportCollection{}
	err = json.Unmarshal(reportCollectionRaw, yr)
	require.NoError(t, err)

	buf := bytes.Buffer{}
	writer := iox.IOWriter{Writer: &buf}

	r := &Reporter{
		Conf:    defaultPrintConfig(),
		Printer: &printer.DefaultPrinter,
		Colors:  &colors.DefaultColorTheme,
	}
	rr := &defaultReporter{
		Reporter: r,
		output:   &writer,
		data:     yr,
	}
	rr.print()

	assert.Contains(t, buf.String(), "✕ Fail:       Ensure")
	assert.Contains(t, buf.String(), ". Skipped:    Set")
	assert.Contains(t, buf.String(), "! Error:      Set")
	assert.Contains(t, buf.String(), "✓ Pass:  100  Ensure")
	assert.Contains(t, buf.String(), "✕ Fail:    0  Ensure")
}

func TestVulnReporter(t *testing.T) {
	reportRaw, err := os.ReadFile("./testdata/mondoo-debug-vulnReport.json")
	require.NoError(t, err)

	report := &mvd.VulnReport{}
	err = json.Unmarshal(reportRaw, report)
	require.NoError(t, err)

	buf := bytes.Buffer{}
	writer := iox.IOWriter{Writer: &buf}
	target := "index.docker.io/library/ubuntu@669e010b58ba"

	t.Run("format=summary", func(t *testing.T) {
		conf := defaultPrintConfig().setFormat(FormatSummary)
		r := NewReporter(conf, false)
		r.out = &writer
		require.NoError(t, err)
		err = r.PrintVulns(report, target)
		require.NoError(t, err)
	})

	t.Run("format=compact", func(t *testing.T) {
		conf := defaultPrintConfig().setFormat(FormatCompact)
		r := NewReporter(conf, false)
		r.out = &writer
		err = r.PrintVulns(report, target)
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "5.5    libblkid1       2.34-0.1ubuntu9.1")
		assert.NotContains(t, buf.String(), "USN-5279-1")
	})

	t.Run("format=full", func(t *testing.T) {
		conf := defaultPrintConfig().setFormat(FormatFull)
		r := NewReporter(conf, false)
		r.out = &writer
		require.NoError(t, err)
		err = r.PrintVulns(report, target)
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "5.5    libblkid1       2.34-0.1ubuntu9.1")
		assert.Contains(t, buf.String(), "USN-5279-1")
	})

	t.Run("format=yaml", func(t *testing.T) {
		conf := defaultPrintConfig().setFormat(FormatYAMLv1)
		r := NewReporter(conf, false)
		r.out = &writer
		require.NoError(t, err)
		err = r.PrintVulns(report, target)
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "score: 5.5")
		assert.Contains(t, buf.String(), "package: libblkid1")
		assert.Contains(t, buf.String(), "installed: 2.34-0.1ubuntu9.1")
		assert.Contains(t, buf.String(), "advisory: USN-5279-1")
	})
}
