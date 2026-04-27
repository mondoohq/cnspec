// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scandb

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/v13/policy"
	"go.mondoo.com/mql/v13/llx"
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

	// Test resource methods
	testResource := &llx.ResourceRecording{
		Resource: "wrapper-resource",
		Id:       "res-1",
		Fields: map[string]*llx.Result{
			"field1": {CodeId: "field1", Data: llx.StringPrimitive("wrapper resource data")},
		},
	}

	t.Run("WriteResource with correct asset MRN", func(t *testing.T) {
		err := wrapper.WriteResource(ctx, assetMrn, testResource)
		require.NoError(t, err)
	})

	t.Run("WriteResource with incorrect asset MRN", func(t *testing.T) {
		wrongAssetMrn := "//assets.api.mondoo.com/spaces/wrong/assets/wrong"
		err := wrapper.WriteResource(ctx, wrongAssetMrn, testResource)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "asset MRN mismatch")
	})

	t.Run("GetResource with correct asset MRN", func(t *testing.T) {
		resource, err := wrapper.GetResource(ctx, assetMrn, "wrapper-resource", "res-1")
		require.NoError(t, err)
		assert.Equal(t, "wrapper-resource", resource.Resource)
		assert.Equal(t, "res-1", resource.Id)
		assert.Equal(t, llx.StringPrimitive("wrapper resource data"), resource.Fields["field1"].Data)
	})

	t.Run("GetResource with incorrect asset MRN", func(t *testing.T) {
		wrongAssetMrn := "//assets.api.mondoo.com/spaces/wrong/assets/wrong"
		_, err := wrapper.GetResource(ctx, wrongAssetMrn, "wrapper-resource", "res-1")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "asset MRN mismatch")
	})

	t.Run("GetResource for non-existent resource", func(t *testing.T) {
		_, err := wrapper.GetResource(ctx, assetMrn, "nonexistent", "nope")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "resource not found")
	})

	t.Run("StreamResources with correct asset MRN", func(t *testing.T) {
		var resources []*llx.ResourceRecording
		err := wrapper.StreamResources(ctx, assetMrn, func(r *llx.ResourceRecording) error {
			resources = append(resources, r)
			return nil
		})
		require.NoError(t, err)
		require.Len(t, resources, 1)
		assert.Equal(t, "wrapper-resource", resources[0].Resource)
	})

	t.Run("StreamResources with incorrect asset MRN", func(t *testing.T) {
		wrongAssetMrn := "//assets.api.mondoo.com/spaces/wrong/assets/wrong"
		err := wrapper.StreamResources(ctx, wrongAssetMrn, func(r *llx.ResourceRecording) error {
			return nil
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "asset MRN mismatch")
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
