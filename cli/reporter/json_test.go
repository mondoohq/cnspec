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
	"go.mondoo.com/mql/llx"
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

func TestJsonOutputErrorDetails(t *testing.T) {
	reportCollectionRaw, err := os.ReadFile("./testdata/simple-report.json")
	require.NoError(t, err)

	yr := &policy.ReportCollection{}
	require.NoError(t, json.Unmarshal(reportCollectionRaw, yr))

	// A passing check over data with one refused region: the score stands and
	// says what it is missing (mql ADR-46 §8).
	for _, report := range yr.Reports {
		for _, score := range report.Scores {
			score.ErrorDetails = []*llx.ErrorDetail{{
				Kind:        llx.ErrorKind_ERROR_KIND_FORBIDDEN,
				Scope:       llx.ErrorScope_ERROR_SCOPE_PARTITION,
				ScopeId:     "eu-west-1",
				Permissions: []string{"ec2:DescribeVpcs", "ec2:DescribeAddresses"},
			}}
		}
	}
	yr.ErrorDetails = map[string]*llx.ErrorDetail{
		"//assets/throttled": {Kind: llx.ErrorKind_ERROR_KIND_TOO_MANY_REQUESTS, RetryAfterMs: 2000},
	}

	buf := bytes.Buffer{}
	conf := defaultPrintConfig()
	conf.format = FormatJSONv1
	r := &Reporter{
		Conf:    conf,
		Printer: &printer.DefaultPrinter,
		Colors:  &colors.DefaultColorTheme,
		out:     &iox.IOWriter{Writer: &buf},
	}
	require.NoError(t, r.WriteReport(context.Background(), yr))
	require.True(t, json.Valid(buf.Bytes()))

	assert.Contains(t, buf.String(), "custom-query-passing-1\":{\"score\":100,\"riskScore\":0,\"status\":\"pass\","+
		"\"errorDetails\":[{\"kind\":\"forbidden\",\"scope\":\"partition\",\"scopeId\":\"eu-west-1\","+
		"\"permissions\":[\"ec2:DescribeAddresses\",\"ec2:DescribeVpcs\"]}]}")
	assert.Contains(t, buf.String(), "\"errors\":{},\"errorDetails\":{\"//assets/throttled\":{\"kind\":\"too_many_requests\",\"retryAfterMs\":2000}}}")
}
