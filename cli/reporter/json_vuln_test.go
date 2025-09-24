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
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/upstream/mvd"
	"go.mondoo.com/cnquery/v12/utils/iox"
)

func TestJsonConverter(t *testing.T) {
	reportRaw, err := os.ReadFile("./testdata/mondoo-debug-vulnReport.json")
	require.NoError(t, err)

	report := &mvd.VulnReport{}
	err = json.Unmarshal(reportRaw, report)
	require.NoError(t, err)

	buf := bytes.Buffer{}
	writer := iox.IOWriter{Writer: &buf}
	err = VulnReportToJSON("index.docker.io/ubuntu:focal-20220113", report, &writer)
	require.NoError(t, err)

	assert.Contains(t, buf.String(), "\"cves\":[\"CVE-2021-43618\"]")
}
