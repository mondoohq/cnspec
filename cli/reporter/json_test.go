// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/mql/cli/printer"
	"go.mondoo.com/mql/cli/theme/colors"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/utils/iox"
)

func TestJsonOutput(t *testing.T) {
	// You can reproduce the report by running
	// DEBUG=1 cnspec scan local -f bundle.mql.yaml
	// where
	// bundle.mql.yaml contains
	// policies:
	// - uid: custom-test-policy-1
	//   name: Custom Test Policy 1
	//   groups:
	//   - filters: |
	// 	  return true
	// 	checks:
	// 	- uid: custom-query-passing-1
	// 	  title: Failing Query
	// 	  mql: |
	// 		true == true

	reportCollectionRaw, err := os.ReadFile("./testdata/simple-report.json")
	require.NoError(t, err)

	yr := &policy.ReportCollection{}
	err = json.Unmarshal(reportCollectionRaw, yr)
	require.NoError(t, err)

	buf := bytes.Buffer{}
	writer := iox.IOWriter{Writer: &buf}

	conf := defaultPrintConfig()
	conf.format = FormatJSONv1
	r := &Reporter{
		Conf:    conf,
		Printer: &printer.DefaultPrinter,
		Colors:  &colors.DefaultColorTheme,
		out:     &writer,
	}

	err = r.WriteReport(context.Background(), yr)
	require.NoError(t, err)
	valid := json.Valid(buf.Bytes())
	fmt.Println(buf.String())
	require.True(t, valid)

	assert.Contains(t, buf.String(), "//local.cnspec.io/run/local-execution/queries/custom-query-passing-1\":{\"score\":100,\"riskScore\":0,\"status\":\"pass\"}")
	assert.Contains(t, buf.String(), "\"errors\":{}")
}

func TestJsonOutputOnlyErrors(t *testing.T) {
	reportCollectionRaw, err := os.ReadFile("./testdata/report-k8s.json")
	require.NoError(t, err)

	yr := &policy.ReportCollection{}
	err = json.Unmarshal(reportCollectionRaw, yr)
	require.NoError(t, err)

	buf := bytes.Buffer{}
	writer := iox.IOWriter{Writer: &buf}

	conf := defaultPrintConfig()
	conf.format = FormatJSONv1
	r := &Reporter{
		Conf:    conf,
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

// TestJsonOutput_RendersScanWarnings covers ConvertToJSON's rendering of
// ReportCollection.Warnings (the -o json / -o yaml path). Unlike errors,
// warnings must be visible without implying the scan failed -- there is no
// exit-code assertion here because ConvertToJSON never touches it; that
// guarantee lives in apps/cnspec/cmd/scan.go, which only ever reads
// report.Errors.
func TestJsonOutput_RendersScanWarnings(t *testing.T) {
	yr := &policy.ReportCollection{
		Assets: map[string]*inventory.Asset{},
		Warnings: map[string]*policy.ScanWarnings{
			"//assets/1": {Messages: []string{"the 'os' provider crashed: connection refused"}},
		},
	}

	buf := bytes.Buffer{}
	writer := iox.IOWriter{Writer: &buf}

	err := ConvertToJSON(yr, &writer)
	require.NoError(t, err)
	require.True(t, json.Valid(buf.Bytes()))

	var parsed struct {
		Warnings map[string][]string `json:"warnings"`
	}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &parsed))
	assert.Equal(t, []string{"the 'os' provider crashed: connection refused"}, parsed.Warnings["//assets/1"])
}

// TestJsonOutput_EmptyWarningsIsAnEmptyObject keeps the shape stable (an
// object, never a missing key or null) when nothing crashed -- the common
// case.
func TestJsonOutput_EmptyWarningsIsAnEmptyObject(t *testing.T) {
	yr := &policy.ReportCollection{Assets: map[string]*inventory.Asset{}}

	buf := bytes.Buffer{}
	writer := iox.IOWriter{Writer: &buf}

	err := ConvertToJSON(yr, &writer)
	require.NoError(t, err)
	require.True(t, json.Valid(buf.Bytes()))
	assert.Contains(t, buf.String(), "\"warnings\":{}")
}
