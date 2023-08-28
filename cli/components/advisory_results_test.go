// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package components

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/providers-sdk/v1/upstream/mvd"
	"sigs.k8s.io/yaml"
)

func TestRenderAdvisoryResults(t *testing.T) {
	// load advisory report
	report, err := loadVulnReportFromFile("./testdata/advisory_report.yaml")
	require.NoError(t, err)
	renderer := NewAdvisoryResultTable()
	output, err := renderer.Render(report)
	require.NoError(t, err)
	fmt.Println(output)
	assert.True(t, len(output) > 0)
}

func TestRenderEmptyAdvisoryResults(t *testing.T) {
	// load advisory report
	report, err := loadVulnReportFromFile("./testdata/advisory_report_empty.yaml")
	require.NoError(t, err)
	renderer := NewAdvisoryResultTable()
	output, err := renderer.Render(report)
	require.NoError(t, err)
	fmt.Println(output)
	assert.True(t, len(output) == 0)
}

func loadVulnReportFromFile(filename string) (*mvd.VulnReport, error) {
	var report mvd.VulnReport
	data, err := os.ReadFile(filename)
	if err != nil {
		return &report, err
	}

	err = yaml.Unmarshal(data, &report)
	return &report, err
}
