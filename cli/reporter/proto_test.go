// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/internal/reportfixture"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
)

func TestProtoConversion(t *testing.T) {
	t.Run("test policy report conversion", func(t *testing.T) {
		yr, err := reportfixture.UbuntuScan()
		require.NoError(t, err)

		report, err := ConvertToProto(yr)
		require.NoError(t, err)

		assert.Equal(t, 1, len(report.Assets))

		assetMrn := "//assets.api.mondoo.app/spaces/test-infallible-taussig-796596/assets/310qEuaqCbVCMU5tGa1HbLx8TZc"
		require.Contains(t, assetMrn, assetMrn)

		asset := report.Assets[assetMrn]
		assert.Equal(t, "ubuntu:24.04", asset.Name)
		assert.Equal(t, "ubuntu", asset.PlatformName)

		assert.Equal(t, 1, len(report.Scores))
		assert.Equal(t, 0, len(report.Errors))
		assert.Equal(t, 1, len(report.Data))

		assert.Equal(t, 53, len(report.Scores[assetMrn].Values))

		score := report.Scores[assetMrn].Values["//policy.api.mondoo.app/queries/mondoo-linux-security-permissions-on-etcgshadow-are-configured"]
		assert.Equal(t, 0, int(score.RiskScore))
		assert.Equal(t, "pass", score.Status)
	})

	t.Run("carries scan warnings (v2 JSON/YAML path)", func(t *testing.T) {
		yr := &policy.ReportCollection{
			Assets: map[string]*inventory.Asset{},
			Warnings: map[string]*policy.ScanWarnings{
				"//assets/1": {Messages: []string{"the 'os' provider crashed: connection refused"}},
				// An entry with no messages must not produce an empty
				// Warnings value in the proto report.
				"//assets/2": {},
			},
		}

		report, err := ConvertToProto(yr)
		require.NoError(t, err)

		require.Contains(t, report.Warnings, "//assets/1")
		assert.Equal(t, []string{"the 'os' provider crashed: connection refused"}, report.Warnings["//assets/1"].Messages)
		assert.NotContains(t, report.Warnings, "//assets/2")
	})
}
