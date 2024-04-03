// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/v10/policy"
)

func TestProtoConversion(t *testing.T) {
	t.Run("test policy report conversion", func(t *testing.T) {
		reportCollectionRaw, err := os.ReadFile("./testdata/report-ubuntu.json")
		require.NoError(t, err)

		yr := &policy.ReportCollection{}
		err = json.Unmarshal(reportCollectionRaw, yr)
		require.NoError(t, err)

		report, err := ConvertToProto(yr)
		require.NoError(t, err)

		assert.Equal(t, 1, len(report.Assets))

		assetMrn := "//assets.api.mondoo.app/spaces/dazzling-golick-767384/assets/2DRZ1cCWFyTYCArycAXHwvn1oU2"
		asset := report.Assets[assetMrn]
		assert.Equal(t, "X1", asset.Name)
		assert.Equal(t, "ubuntu", asset.PlatformName)

		assert.Equal(t, 1, len(report.Scores))
		assert.Equal(t, 0, len(report.Errors))
		assert.Equal(t, 1, len(report.Data))

		assert.Equal(t, 108, len(report.Scores[assetMrn].Values))

		score := report.Scores[assetMrn].Values["//policy.api.mondoo.app/queries/mondoo-linux-security-permissions-on-etcgshadow-are-configured"]
		assert.Equal(t, 100, int(score.Score))
		assert.Equal(t, "pass", score.Status)
	})

}
