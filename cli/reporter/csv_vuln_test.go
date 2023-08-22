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
	"go.mondoo.com/cnquery/shared"
	"go.mondoo.com/cnquery/upstream/mvd"
)

func TestCsvConverter(t *testing.T) {
	reportRaw, err := os.ReadFile("./testdata/mondoo-debug-vulnReport.json")
	require.NoError(t, err)

	report := &mvd.VulnReport{}
	err = json.Unmarshal(reportRaw, report)
	require.NoError(t, err)

	buf := bytes.Buffer{}
	writer := shared.IOWriter{Writer: &buf}
	err = VulnReportToCSV(report, &writer)
	require.NoError(t, err)

	assert.Contains(t, buf.String(), "libblkid1,5.5,2.34-0.1ubuntu9.1,2.34-0.1ubuntu9.3,2.34-0.1ubuntu9.3,USN-5279-1,CVE-2021-3995 CVE-2021-3996")
}
