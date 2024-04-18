// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			assert.Equal(t, expect, conf)
		}},
		{"full", func(t *testing.T, conf *PrintConfig) {
			expect := defaultPrintConfig()
			expect.format = FormatFull
			expect.isCompact = false
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

	t.Run("unknown options", func(t *testing.T) {
		_, err := ParseConfig("notknown")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown terms entered: notknown")
		assert.Contains(t, err.Error(), "Available output formats: compact, csv, ")
		assert.Contains(t, err.Error(), "Available options: [no]checks, [no]controls, ")
	})
}
