// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v11/llx"
)

func TestSqliteScanDataStore(t *testing.T) {
	// Create a temporary file for testing
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_upload.db")

	defer func() {
		os.RemoveAll(tempDir) // Clean up temp directory
	}()

	assetMrn := "//assets.api.mondoo.com/spaces/test-space/assets/test-asset"
	uploadSessionId := "session-123"

	// Test data
	testScores := []*Score{
		{
			QrId:      "score-1",
			RiskScore: 85,
			Type:      1,
			Value:     100,
			Weight:    50,
			Message:   "Test score 1",
			RiskFactors: &ScoredRiskFactors{
				Items: []*ScoredRiskFactor{
					{Mrn: "risk1", Risk: 0.5, IsDetected: true},
				},
			},
			Sources: &Sources{
				Items: []*Source{
					{Name: "test-source", Version: "1.0"},
				},
			},
		},
		{
			QrId:      "score-2",
			RiskScore: 75,
			Type:      2,
			Value:     80,
			Weight:    30,
			Message:   "Test score 2",
		},
	}

	testData := []*llx.Result{
		{
			CodeId: "data-1",
			Data:   llx.BoolPrimitive(true),
		},
		{
			CodeId: "data-2",
			Data:   llx.BoolPrimitive(false),
		},
	}

	// Write data to SQLite file
	t.Run("Write", func(t *testing.T) {
		store, err := NewSqliteScanDataStore(dbPath, assetMrn, uploadSessionId)
		require.NoError(t, err)
		defer store.Close()

		// Write scores
		err = store.WriteScores(context.Background(), testScores)
		require.NoError(t, err)

		// Write data
		err = store.WriteData(context.Background(), testData)
		require.NoError(t, err)
	})

	// Verify the file exists
	_, err := os.Stat(dbPath)
	require.NoError(t, err)

	// Read and verify data
	t.Run("Read", func(t *testing.T) {
		store, err := NewSqliteScanDataStoreReader(dbPath)
		require.NoError(t, err)
		defer store.Close()

		// Validate file structure
		err = store.ValidateFile()
		require.NoError(t, err)

		// Check metadata
		metadata, err := store.GetMetadata()
		require.NoError(t, err)
		assert.Equal(t, SchemaVersion, metadata.SchemaVersion)
		assert.Equal(t, assetMrn, metadata.AssetMrn)
		assert.Equal(t, uploadSessionId, metadata.UploadSessionId)

		// Read scores
		var scores []*Score
		err = store.StreamScores(context.Background(), func(score *Score) error {
			scores = append(scores, score)
			return nil
		})
		require.NoError(t, err)
		require.Len(t, scores, 2)

		// Verify first score
		score1 := scores[0]
		assert.Equal(t, "score-1", score1.QrId)
		assert.Equal(t, uint32(85), score1.RiskScore)
		assert.Equal(t, uint32(1), score1.Type)
		assert.Equal(t, uint32(100), score1.Value)
		assert.Equal(t, uint32(50), score1.Weight)
		assert.Equal(t, "Test score 1", score1.Message)
		require.NotNil(t, score1.RiskFactors)
		require.Len(t, score1.RiskFactors.Items, 1)
		assert.Equal(t, "risk1", score1.RiskFactors.Items[0].Mrn)
		require.NotNil(t, score1.Sources)
		require.Len(t, score1.Sources.Items, 1)
		assert.Equal(t, "test-source", score1.Sources.Items[0].Name)

		// Verify second score (minimal data)
		score2 := scores[1]
		assert.Equal(t, "score-2", score2.QrId)
		assert.Equal(t, uint32(75), score2.RiskScore)
		assert.Equal(t, "Test score 2", score2.Message)
		assert.Nil(t, score2.RiskFactors)
		assert.Nil(t, score2.Sources)

		// Read data
		var data map[string]*llx.Result
		err = store.StreamData(context.Background(), func(codeId string, result *llx.Result) error {
			if data == nil {
				data = make(map[string]*llx.Result)
			}
			data[codeId] = result
			return nil
		})
		require.NoError(t, err)
		require.Len(t, data, 2)

		// Verify data
		result1, exists := data["data-1"]
		require.True(t, exists)
		assert.Equal(t, "data-1", result1.CodeId)
		assert.Equal(t, llx.BoolPrimitive(true), result1.Data)

		result2, exists := data["data-2"]
		require.True(t, exists)
		assert.Equal(t, "data-2", result2.CodeId)
		assert.Equal(t, llx.BoolPrimitive(false), result2.Data)

		// Test specific get methods
		specificScore, err := store.GetScore(context.Background(), "score-1")
		require.NoError(t, err)
		assert.Equal(t, "score-1", specificScore.QrId)
		assert.Equal(t, uint32(85), specificScore.RiskScore)

		specificData, err := store.GetData(context.Background(), "data-1")
		require.NoError(t, err)
		assert.Equal(t, "data-1", specificData.CodeId)
		assert.Equal(t, llx.BoolPrimitive(true), specificData.Data)

		// Test not found cases
		_, err = store.GetScore(context.Background(), "nonexistent-score")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "score not found")

		_, err = store.GetData(context.Background(), "nonexistent-data")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "data not found")
	})
}

func TestStreamingReads(t *testing.T) {
	// Create a temporary file for testing
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_streaming.db")

	defer func() {
		os.RemoveAll(tempDir) // Clean up temp directory
	}()

	assetMrn := "//assets.api.mondoo.com/spaces/test-space/assets/streaming-test"
	uploadSessionId := "streaming-session"

	// Create test data with more entries
	testScores := make([]*Score, 10)
	for i := 0; i < 10; i++ {
		testScores[i] = &Score{
			QrId:      "score-" + string(rune('0'+i)),
			RiskScore: uint32(50 + i*5),
			Type:      uint32(i % 3),
			Value:     uint32(80 + i*2),
			Weight:    uint32(10 + i),
			Message:   "Test score " + string(rune('0'+i)),
		}
	}

	testData := make([]*llx.Result, 5)
	for i := 0; i < len(testData); i++ {
		codeId := "data-" + string(rune('0'+i))
		testData[i] = &llx.Result{
			CodeId: codeId,
			Data:   llx.StringPrimitive("streaming test data " + string(rune('0'+i))),
		}
	}

	// Write data
	store, err := NewSqliteScanDataStore(dbPath, assetMrn, uploadSessionId)
	require.NoError(t, err)

	err = store.WriteScores(context.Background(), testScores)
	require.NoError(t, err)

	err = store.WriteData(context.Background(), testData)
	require.NoError(t, err)

	err = store.Close()
	require.NoError(t, err)

	// Test streaming reads
	store, err = NewSqliteScanDataStoreReader(dbPath)
	require.NoError(t, err)
	defer store.Close()

	// Test streaming scores
	t.Run("StreamScores", func(t *testing.T) {
		var streamedScores []*Score

		err := store.StreamScores(context.Background(), func(score *Score) error {
			streamedScores = append(streamedScores, score)
			return nil
		})

		require.NoError(t, err)
		assert.Len(t, streamedScores, 10)

		// Verify first and last scores
		assert.Equal(t, "score-0", streamedScores[0].QrId)
		assert.Equal(t, "score-9", streamedScores[9].QrId)
	})

	// Test streaming data
	t.Run("StreamData", func(t *testing.T) {
		streamedData := make(map[string]*llx.Result)

		err := store.StreamData(context.Background(), func(codeId string, result *llx.Result) error {
			streamedData[codeId] = result
			return nil
		})

		require.NoError(t, err)
		assert.Len(t, streamedData, 5)

		// Verify data integrity
		for i := 0; i < 5; i++ {
			key := "data-" + string(rune('0'+i))
			result, exists := streamedData[key]
			require.True(t, exists)
			assert.Equal(t, key, result.CodeId)
			expected := "streaming test data " + string(rune('0'+i))
			assert.Equal(t, llx.StringPrimitive(expected), result.Data)
		}
	})
}

func TestSqliteScanDataStore_Finalize(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_finalize.db")

	defer func() {
		os.RemoveAll(tempDir)
	}()

	assetMrn := "//assets.api.mondoo.com/spaces/test-space/assets/finalize-test"
	uploadSessionId := "finalize-session"

	// Create the store in write mode
	store, err := NewSqliteScanDataStore(dbPath, assetMrn, uploadSessionId)
	require.NoError(t, err)

	ctx := context.Background()

	// Write some test data
	testScore := &Score{
		QrId:      "finalize-score-1",
		RiskScore: 80,
		Type:      1,
		Value:     90,
		Weight:    45,
		Message:   "Finalize test score",
	}

	testData := &llx.Result{
		CodeId: "finalize-data-1",
		Data:   llx.StringPrimitive("finalize test data"),
	}

	err = store.WriteScores(ctx, []*Score{testScore})
	require.NoError(t, err)

	err = store.WriteData(ctx, []*llx.Result{testData})
	require.NoError(t, err)

	// Test finalize
	path, err := store.Finalize()
	require.NoError(t, err)
	assert.Equal(t, dbPath, path)

	// Verify store is now read-only
	assert.True(t, store.readOnly)

	// Should still be able to read data
	score, err := store.GetScore(ctx, "finalize-score-1")
	require.NoError(t, err)
	assert.Equal(t, "finalize-score-1", score.QrId)
	assert.Equal(t, uint32(80), score.RiskScore)

	result, err := store.GetData(ctx, "finalize-data-1")
	require.NoError(t, err)
	assert.Equal(t, "finalize-data-1", result.CodeId)
	assert.Equal(t, llx.StringPrimitive("finalize test data"), result.Data)

	// Writing should now fail
	newScore := &Score{
		QrId:      "finalize-score-2",
		RiskScore: 85,
		Type:      1,
		Value:     95,
		Weight:    50,
		Message:   "Post-finalize score",
	}

	err = store.WriteScores(ctx, []*Score{newScore})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read-only mode")

	// Test finalize again (should be no-op)
	path2, err := store.Finalize()
	require.NoError(t, err)
	assert.Equal(t, dbPath, path2)

	store.Close()
}

func TestSqliteScanDataStoreReader_Finalize(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_reader_finalize.db")

	defer func() {
		os.RemoveAll(tempDir)
	}()

	assetMrn := "//assets.api.mondoo.com/spaces/test-space/assets/reader-finalize-test"
	uploadSessionId := "reader-finalize-session"

	// First create a database with data
	store, err := NewSqliteScanDataStore(dbPath, assetMrn, uploadSessionId)
	require.NoError(t, err)

	ctx := context.Background()
	testScore := &Score{
		QrId:      "reader-finalize-score-1",
		RiskScore: 75,
		Type:      1,
		Value:     85,
		Weight:    40,
		Message:   "Reader finalize test score",
	}

	err = store.WriteScores(ctx, []*Score{testScore})
	require.NoError(t, err)
	store.Close()

	// Open as reader
	reader, err := NewSqliteScanDataStoreReader(dbPath)
	require.NoError(t, err)
	defer reader.Close()

	// Test finalize on read-only store (should be no-op)
	path, err := reader.Finalize()
	require.NoError(t, err)
	assert.Equal(t, dbPath, path)

	// Should still be able to read
	score, err := reader.GetScore(ctx, "reader-finalize-score-1")
	require.NoError(t, err)
	assert.Equal(t, "reader-finalize-score-1", score.QrId)
}
