// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/reports/ocsf"
)

func TestParseConfig(t *testing.T) {
	tests := []struct {
		conf string
		f    func(t *testing.T, conf *PrintConfig)
	}{
		{"", func(t *testing.T, conf *PrintConfig) {
			assert.Equal(t, defaultPrintConfig(), conf)
		}},
		{"compact", func(t *testing.T, conf *PrintConfig) {
			assert.Equal(t, defaultPrintConfig(), conf)
		}},
		{"summary", func(t *testing.T, conf *PrintConfig) {
			expect := defaultPrintConfig()
			expect.format = FormatSummary
			expect.printData = false
			expect.printVulnerabilities = false
			expect.printControls = false
			expect.printChecks = false
			expect.printRisks = false
			assert.Equal(t, expect, conf)
		}},
		{"summary,checks,DATA,vulns", func(t *testing.T, conf *PrintConfig) {
			expect := defaultPrintConfig()
			expect.format = FormatSummary
			expect.printControls = false
			expect.printRisks = false
			expect.printData = true
			assert.Equal(t, expect, conf)
		}},
		{"full", func(t *testing.T, conf *PrintConfig) {
			expect := defaultPrintConfig()
			expect.format = FormatFull
			expect.isCompact = false
			expect.printControls = true
			expect.printData = true
			assert.Equal(t, expect, conf)
		}},
		{"ocsf", func(t *testing.T, conf *PrintConfig) {
			expect := defaultPrintConfig()
			expect.format = FormatOcsfJson
			assert.Equal(t, expect, conf)
		}},
		{"ocsf-parquet,ocsf-version=1.9.0", func(t *testing.T, conf *PrintConfig) {
			expect := defaultPrintConfig()
			expect.format = FormatOcsfParquet
			expect.ocsfVersion = ocsf.Version190
			assert.Equal(t, expect, conf)
		}},
		{"nodata,noVuln,noRiSks", func(t *testing.T, conf *PrintConfig) {
			expect := defaultPrintConfig()
			expect.printData = false
			expect.printVulnerabilities = false
			expect.printRisks = false
			assert.Equal(t, expect, conf)
		}},
	}

	for i := range tests {
		cur := tests[i]
		t.Run(cur.conf, func(t *testing.T) {
			res, err := ParseConfig(cur.conf)
			require.NoError(t, err)
			cur.f(t, res)
		})
	}

	t.Run("unsupported ocsf version", func(t *testing.T) {
		_, err := ParseConfig("ocsf-json,ocsf-version=1.2.0")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported OCSF version")
	})

	t.Run("unknown key=value option", func(t *testing.T) {
		_, err := ParseConfig("ocsf-json,nope=1")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown terms entered")
	})

	t.Run("file keeps the case of its path", func(t *testing.T) {
		conf, err := ParseConfig("SARIF,File=Out/Results.sarif")
		require.NoError(t, err)
		assert.Equal(t, FormatSarif, conf.format)
		assert.Equal(t, "Out/Results.sarif", conf.Target())
	})

	t.Run("file needs a path", func(t *testing.T) {
		_, err := ParseConfig("sarif,file=")
		require.Error(t, err)
	})

	t.Run("target keeps its value and is not forced to a file", func(t *testing.T) {
		conf, err := ParseConfig("json,Target=https://sqs.us-east-1.amazonaws.com/123456789012/Q")
		require.NoError(t, err)
		assert.Equal(t, "https://sqs.us-east-1.amazonaws.com/123456789012/Q", conf.Target())
		assert.False(t, conf.targetIsFile)
	})

	t.Run("one destination per output", func(t *testing.T) {
		_, err := ParseConfig("json,file=a.json,target=b.json")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "one file= or target=")
	})

	t.Run("unknown options", func(t *testing.T) {
		_, err := ParseConfig("notknown")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown terms entered: notknown")
		assert.Contains(t, err.Error(), "Available output formats: compact, csv, ")
		assert.Contains(t, err.Error(), "Available options: [no]checks, [no]controls, ")
	})
}
