// Copyright (c) Mondoo, Inc.
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
	"go.mondoo.com/cnquery/v12/cli/printer"
	"go.mondoo.com/cnquery/v12/cli/theme/colors"
	"go.mondoo.com/cnquery/v12/utils/iox"
	"go.mondoo.com/cnspec/v12/policy"
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
