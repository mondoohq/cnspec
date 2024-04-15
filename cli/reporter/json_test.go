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
	"go.mondoo.com/cnquery/v10/shared"
	"go.mondoo.com/cnspec/v10/policy"
)

func TestJsonOutput(t *testing.T) {
	reportCollectionRaw, err := os.ReadFile("./testdata/report-ubuntu.json")
	require.NoError(t, err)

	yr := &policy.ReportCollection{}
	err = json.Unmarshal(reportCollectionRaw, yr)
	require.NoError(t, err)

	buf := bytes.Buffer{}
	writer := shared.IOWriter{Writer: &buf}

	r := &Reporter{
		Format:  FormatJSONv1,
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
		Format:  FormatJSONv1,
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
