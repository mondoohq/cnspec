// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package scandb

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnspec/v12/policy"
)

func TestScanDataStoreWrapper(t *testing.T) {
	// Create a temporary file for testing
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_wrapper.db")

	defer func() {
		os.RemoveAll(tempDir)
	}()

	assetMrn := "//assets.api.mondoo.com/spaces/test-space/assets/wrapper-test"

	// Create the underlying store
	store, err := NewSqliteScanDataStore(dbPath, assetMrn)
	require.NoError(t, err)
	defer store.Close()

	// Create wrapper
	wrapper := NewScanDataStoreWrapper(store, assetMrn)

	ctx := context.Background()

	// Test data
	testScore := &policy.Score{
		QrId:      "wrapper-score-1",
		RiskScore: 75,
		Type:      1,
		Value:     85,
		Weight:    40,
		Message:   "Wrapper test score",
	}

	testData := &llx.Result{
		CodeId: "wrapper-data-1",
		Data:   llx.StringPrimitive("wrapper test data"),
	}

	t.Run("WriteScore with correct asset MRN", func(t *testing.T) {
		err := wrapper.WriteScore(ctx, assetMrn, testScore)
		require.NoError(t, err)
	})

	t.Run("WriteScore with incorrect asset MRN", func(t *testing.T) {
		wrongAssetMrn := "//assets.api.mondoo.com/spaces/wrong/assets/wrong"
		err := wrapper.WriteScore(ctx, wrongAssetMrn, testScore)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "asset MRN mismatch")
	})

	t.Run("WriteData with correct asset MRN", func(t *testing.T) {
		err := wrapper.WriteData(ctx, assetMrn, testData)
		require.NoError(t, err)
	})

	t.Run("WriteData with incorrect asset MRN", func(t *testing.T) {
		wrongAssetMrn := "//assets.api.mondoo.com/spaces/wrong/assets/wrong"
		err := wrapper.WriteData(ctx, wrongAssetMrn, testData)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "asset MRN mismatch")
	})

	t.Run("GetScore with correct asset MRN", func(t *testing.T) {
		score, err := wrapper.GetScore(ctx, assetMrn, "wrapper-score-1")
		require.NoError(t, err)
		assert.Equal(t, "wrapper-score-1", score.QrId)
		assert.Equal(t, uint32(75), score.RiskScore)
		assert.Equal(t, "Wrapper test score", score.Message)
	})

	t.Run("GetScore with incorrect asset MRN", func(t *testing.T) {
		wrongAssetMrn := "//assets.api.mondoo.com/spaces/wrong/assets/wrong"
		_, err := wrapper.GetScore(ctx, wrongAssetMrn, "wrapper-score-1")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "asset MRN mismatch")
	})

	t.Run("GetData with correct asset MRN", func(t *testing.T) {
		result, err := wrapper.GetData(ctx, assetMrn, "wrapper-data-1")
		require.NoError(t, err)
		assert.Equal(t, "wrapper-data-1", result.CodeId)
		assert.Equal(t, llx.StringPrimitive("wrapper test data"), result.Data)
	})

	t.Run("GetData with incorrect asset MRN", func(t *testing.T) {
		wrongAssetMrn := "//assets.api.mondoo.com/spaces/wrong/assets/wrong"
		_, err := wrapper.GetData(ctx, wrongAssetMrn, "wrapper-data-1")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "asset MRN mismatch")
	})

	t.Run("GetScore for non-existent score", func(t *testing.T) {
		_, err := wrapper.GetScore(ctx, assetMrn, "non-existent-score")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "score not found")
	})

	t.Run("GetData for non-existent data", func(t *testing.T) {
		_, err := wrapper.GetData(ctx, assetMrn, "non-existent-data")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "data not found")
	})

	t.Run("Finalize database", func(t *testing.T) {
		path, err := wrapper.Finalize()
		require.NoError(t, err)
		assert.Equal(t, dbPath, path)

		// After finalize, should still be able to read but not write
		score, err := wrapper.GetScore(ctx, assetMrn, "wrapper-score-1")
		require.NoError(t, err)
		assert.Equal(t, "wrapper-score-1", score.QrId)

		// Writing should fail after finalize
		newScore := &policy.Score{
			QrId:      "wrapper-score-2",
			RiskScore: 85,
			Type:      1,
			Value:     95,
			Weight:    50,
			Message:   "Post-finalize score",
		}
		err = wrapper.WriteScore(ctx, assetMrn, newScore)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "read-only mode")
	})
}
