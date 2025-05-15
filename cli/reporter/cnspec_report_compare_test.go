// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReportComparison(t *testing.T) {
	passReport, err := FromSingleFile("testdata/cnspec_report_pass.json")
	require.NoError(t, err)

	failReport, err := FromSingleFile("testdata/cnspec_report_mixed.json")
	require.NoError(t, err)

	t.Run("pass vs pass", func(t *testing.T) {
		equal := CompareReports(passReport, passReport)
		assert.True(t, equal)

	})

	t.Run("pass vs fail", func(t *testing.T) {
		equal := CompareReports(passReport, failReport)
		require.False(t, equal)
	})
}

func TestReportComparisonWithExplicitMrn(t *testing.T) {
	passAssetMrn := "//policy.api.mondoo.com/assets/2x5c0nwiaianioOv2cda2OuAAzk"
	passReport, err := FromSingleFile("testdata/cnspec_report_pass.json")
	require.NoError(t, err)

	failAssetMrn := "//policy.api.mondoo.com/assets/2x5c0nwiaianioOv2cda2OuQKwk"
	failReport, err := FromSingleFile("testdata/cnspec_report_mixed.json")
	require.NoError(t, err)

	t.Run("pass vs pass", func(t *testing.T) {
		equal := CompareAsset(passReport, passAssetMrn, passReport, passAssetMrn)
		assert.True(t, equal)
	})

	t.Run("pass vs fail", func(t *testing.T) {
		equal := CompareAsset(passReport, passAssetMrn, failReport, failAssetMrn)
		require.False(t, equal)
	})
}
