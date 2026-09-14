// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/mql/providers-sdk/v1/upstream/fex"
)

func TestRenderVulnRowsSummary(t *testing.T) {
	t.Run("no vulnerabilities", func(t *testing.T) {
		out := RenderVulnRowsSummary(nil)
		assert.Contains(t, out, "No vulnerabilities found")
	})

	t.Run("counts by severity", func(t *testing.T) {
		out := RenderVulnRowsSummary(fex.VulnRows(sampleVEX()))
		assert.Contains(t, out, "Vulnerabilities: 3")
		assert.Contains(t, out, "Critical: 1")
		assert.Contains(t, out, "High: 1")
		assert.Contains(t, out, "Medium: 1")
	})
}

func TestRenderVulnRowsTable(t *testing.T) {
	rows := fex.VulnRows(sampleVEX())

	t.Run("empty", func(t *testing.T) {
		assert.Equal(t, "", RenderVulnRowsTable(nil, true))
	})

	t.Run("severity ordering, most severe first", func(t *testing.T) {
		out := RenderVulnRowsTable(rows, false)
		crit := strings.Index(out, "CVE-2022-0001")
		high := strings.Index(out, "CVE-2021-3999")
		med := strings.Index(out, "CVE-2021-3995")
		assert.Positive(t, crit)
		assert.Less(t, crit, high, "critical must sort before high")
		assert.Less(t, high, med, "high must sort before medium")
		assert.NotContains(t, out, "REMEDIATION")
	})

	t.Run("detailed adds remediation column", func(t *testing.T) {
		out := RenderVulnRowsTable(rows, true)
		assert.Contains(t, out, "REMEDIATION")
		assert.Contains(t, out, "Upgrade lodash to 4.17.21")
	})
}
