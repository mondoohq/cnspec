// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/providers-sdk/v1/upstream/fex"
	"go.mondoo.com/mql/utils/iox"
)

func TestJsonConverter(t *testing.T) {
	rows := fex.VulnRows(sampleVEX())

	buf := bytes.Buffer{}
	writer := iox.IOWriter{Writer: &buf}
	require.NoError(t, VulnReportToJSON("index.docker.io/library/ubuntu:focal", rows, &writer))

	// It must be valid JSON and carry the target, severity stats, and the richer
	// per-vulnerability fields.
	var doc vulnJSONReport
	require.NoError(t, json.Unmarshal(buf.Bytes(), &doc))

	assert.Equal(t, "index.docker.io/library/ubuntu:focal", doc.Target)
	assert.Equal(t, 3, doc.Stats.Total)
	assert.Equal(t, 1, doc.Stats.Critical)
	assert.Equal(t, 1, doc.Stats.High)
	assert.Equal(t, 1, doc.Stats.Medium)
	require.Len(t, doc.Vulnerabilities, 3)

	// sorted most-severe first
	first := doc.Vulnerabilities[0]
	assert.Equal(t, "CVE-2022-0001", first.Advisory)
	assert.Equal(t, fex.SeverityCritical, first.Severity)
	assert.Equal(t, "lodash", first.Package)
	assert.Equal(t, "4.17.20", first.Installed)
	assert.Equal(t, "4.17.21", first.Fixed)
	assert.Equal(t, "pkg:npm/lodash@4.17.20", first.Purl)
	assert.Equal(t, "Upgrade lodash to 4.17.21", first.Remediation)
}
