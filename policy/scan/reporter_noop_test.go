// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
)

func TestNoOpReporter_Reports(t *testing.T) {
	rated := func(v uint32) *policy.Score {
		return &policy.Score{Type: policy.ScoreType_Result, Value: v, ScoreCompletion: 100}
	}

	t.Run("empty scan is ok with no worst score", func(t *testing.T) {
		res := NewNoOpReporter().Reports()
		assert.True(t, res.Ok)
		assert.Nil(t, res.WorstScore)
		assert.NotNil(t, res.GetNone(), "report type none carries no collection")
	})

	t.Run("ok is false after a scan error", func(t *testing.T) {
		r := NewNoOpReporter()
		r.AddReport(&inventory.Asset{Mrn: "//assets/fine"}, &AssetReport{Report: &policy.Report{Score: rated(90)}})
		r.AddScanError(&inventory.Asset{Mrn: "//assets/broken"}, errors.New("connection refused"))

		res := r.Reports()
		assert.False(t, res.Ok)
		assert.Nil(t, res.GetFull(), "none must not start collecting reports")
		assert.NotNil(t, res.GetNone())
	})

	t.Run("worst score is the lowest rated score across reports", func(t *testing.T) {
		r := NewNoOpReporter()
		r.AddReport(&inventory.Asset{Mrn: "//assets/a"}, &AssetReport{Report: &policy.Report{Score: rated(80)}})
		r.AddReport(&inventory.Asset{Mrn: "//assets/b"}, &AssetReport{Report: &policy.Report{Score: rated(40)}})
		r.AddReport(&inventory.Asset{Mrn: "//assets/c"}, &AssetReport{Report: &policy.Report{Score: rated(60)}})

		res := r.Reports()
		require.NotNil(t, res.WorstScore)
		assert.Equal(t, uint32(40), res.WorstScore.Value)
		assert.True(t, res.Ok)
	})

	t.Run("unrated reports do not count as a score of zero", func(t *testing.T) {
		r := NewNoOpReporter()
		r.AddReport(&inventory.Asset{Mrn: "//assets/rated"}, &AssetReport{Report: &policy.Report{Score: rated(70)}})
		r.AddReport(&inventory.Asset{Mrn: "//assets/unscored"}, &AssetReport{Report: &policy.Report{Score: &policy.Score{Type: policy.ScoreType_Unscored, ScoreCompletion: 100}}})
		r.AddReport(&inventory.Asset{Mrn: "//assets/noscore"}, &AssetReport{Report: &policy.Report{}})

		res := r.Reports()
		require.NotNil(t, res.WorstScore)
		assert.Equal(t, uint32(70), res.WorstScore.Value)
	})
}

func TestErrorReporter_WorstScoreSkipsUnrated(t *testing.T) {
	r := NewErrorReporter()
	r.AddReport(&inventory.Asset{Mrn: "//assets/rated"}, &AssetReport{Report: &policy.Report{Score: &policy.Score{Type: policy.ScoreType_Result, Value: 55, ScoreCompletion: 100}}})
	r.AddReport(&inventory.Asset{Mrn: "//assets/unscored"}, &AssetReport{Report: &policy.Report{Score: &policy.Score{Type: policy.ScoreType_Unscored}}})

	res := r.Reports()
	require.NotNil(t, res.WorstScore)
	assert.Equal(t, uint32(55), res.WorstScore.Value)
	assert.True(t, res.Ok)
}
