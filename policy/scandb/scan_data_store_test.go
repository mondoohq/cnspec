// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package scandb

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/glebarez/go-sqlite"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnspec/v12/policy"
)

func TestSqliteScanDataStore(t *testing.T) {
	// Create a temporary file for testing
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_upload.db")

	defer func() {
		os.RemoveAll(tempDir) // Clean up temp directory
	}()

	assetMrn := "//assets.api.mondoo.com/spaces/test-space/assets/test-asset"

	// Test data
	testScores := []*policy.Score{
		{
			QrId:      "score-1",
			RiskScore: 85,
			Type:      1,
			Value:     100,
			Weight:    50,
			Message:   "Test score 1",
			RiskFactors: &policy.ScoredRiskFactors{
				Items: []*policy.ScoredRiskFactor{
					{Mrn: "risk1", Risk: 0.5, IsDetected: true},
				},
			},
			Sources: &policy.Sources{
				Items: []*policy.Source{
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

	testRisks := []*policy.ScoredRiskFactor{
		{
			Mrn:        "risk-1",
			Risk:       0.75,
			IsToxic:    true,
			IsDetected: true,
		},
		{
			Mrn:        "risk-2",
			Risk:       0.25,
			IsToxic:    false,
			IsDetected: false,
		},
	}

	// Write data to SQLite file
	t.Run("Write", func(t *testing.T) {
		store, err := NewSqliteScanDataStore(dbPath, assetMrn)
		require.NoError(t, err)
		defer store.Close()

		// Write scores
		err = store.WriteScores(context.Background(), testScores)
		require.NoError(t, err)

		// Write data
		err = store.WriteData(context.Background(), testData)
		require.NoError(t, err)

		// Write risks
		for _, risk := range testRisks {
			err = store.WriteRisk(context.Background(), risk)
			require.NoError(t, err)
		}
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

		// Read scores
		var scores []*policy.Score
		err = store.StreamScores(context.Background(), func(score *policy.Score) error {
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

		// Read risks
		var risks []*policy.ScoredRiskFactor
		err = store.StreamRisks(context.Background(), func(risk *policy.ScoredRiskFactor) error {
			risks = append(risks, risk)
			return nil
		})
		require.NoError(t, err)
		require.Len(t, risks, 2)

		// Verify first risk
		risk1 := risks[0]
		assert.Equal(t, "risk-1", risk1.Mrn)
		assert.Equal(t, float32(0.75), risk1.Risk)
		assert.Equal(t, true, risk1.IsToxic)
		assert.Equal(t, true, risk1.IsDetected)

		// Verify second risk
		risk2 := risks[1]
		assert.Equal(t, "risk-2", risk2.Mrn)
		assert.Equal(t, float32(0.25), risk2.Risk)
		assert.Equal(t, false, risk2.IsToxic)
		assert.Equal(t, false, risk2.IsDetected)

		// Test specific get risk method
		specificRisk, err := store.GetRisk(context.Background(), "risk-1")
		require.NoError(t, err)
		assert.Equal(t, "risk-1", specificRisk.Mrn)
		assert.Equal(t, float32(0.75), specificRisk.Risk)

		// Test not found case for risk
		_, err = store.GetRisk(context.Background(), "nonexistent-risk")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "risk not found")
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

	// Create test data with more entries
	testScores := make([]*policy.Score, 10)
	for i := 0; i < 10; i++ {
		testScores[i] = &policy.Score{
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
	store, err := NewSqliteScanDataStore(dbPath, assetMrn)
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
		var streamedScores []*policy.Score

		err := store.StreamScores(context.Background(), func(score *policy.Score) error {
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

	// Create the store in write mode
	store, err := NewSqliteScanDataStore(dbPath, assetMrn)
	require.NoError(t, err)

	ctx := context.Background()

	// Write some test data
	testScore := &policy.Score{
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

	err = store.WriteScores(ctx, []*policy.Score{testScore})
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
	newScore := &policy.Score{
		QrId:      "finalize-score-2",
		RiskScore: 85,
		Type:      1,
		Value:     95,
		Weight:    50,
		Message:   "Post-finalize score",
	}

	err = store.WriteScores(ctx, []*policy.Score{newScore})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read-only mode")

	// Writing risks should also fail
	newRisk := &policy.ScoredRiskFactor{
		Mrn:        "finalize-risk-2",
		Risk:       0.5,
		IsToxic:    false,
		IsDetected: true,
	}

	err = store.WriteRisk(ctx, newRisk)
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

	// First create a database with data
	store, err := NewSqliteScanDataStore(dbPath, assetMrn)
	require.NoError(t, err)

	ctx := context.Background()
	testScore := &policy.Score{
		QrId:      "reader-finalize-score-1",
		RiskScore: 75,
		Type:      1,
		Value:     85,
		Weight:    40,
		Message:   "Reader finalize test score",
	}

	err = store.WriteScores(ctx, []*policy.Score{testScore})
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

func TestSqliteScanDataStore_UpsertScores(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_upsert_scores.db")

	defer func() {
		os.RemoveAll(tempDir)
	}()

	assetMrn := "//assets.api.mondoo.com/spaces/test-space/assets/upsert-test"

	// Create the store
	store, err := NewSqliteScanDataStore(dbPath, assetMrn)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Initial score
	initialScore := &policy.Score{
		QrId:      "upsert-score-1",
		RiskScore: 70,
		Type:      1,
		Value:     80,
		Weight:    35,
		Message:   "Initial score",
	}

	// Write initial score
	err = store.WriteScores(ctx, []*policy.Score{initialScore})
	require.NoError(t, err)

	// Verify initial score
	score, err := store.GetScore(ctx, "upsert-score-1")
	require.NoError(t, err)
	assert.Equal(t, uint32(70), score.RiskScore)
	assert.Equal(t, "Initial score", score.Message)

	// Updated score with same qr_id
	updatedScore := &policy.Score{
		QrId:      "upsert-score-1", // Same QR ID
		RiskScore: 85,               // Different values
		Type:      2,
		Value:     95,
		Weight:    50,
		Message:   "Updated score",
	}

	// Write updated score (should upsert)
	err = store.WriteScores(ctx, []*policy.Score{updatedScore})
	require.NoError(t, err)

	// Verify the score was updated, not duplicated
	score, err = store.GetScore(ctx, "upsert-score-1")
	require.NoError(t, err)
	assert.Equal(t, uint32(85), score.RiskScore)
	assert.Equal(t, "Updated score", score.Message)
	assert.Equal(t, uint32(2), score.Type)
	assert.Equal(t, uint32(95), score.Value)
	assert.Equal(t, uint32(50), score.Weight)

	// Verify there's only one score with this QR ID
	count := 0
	err = store.StreamScores(ctx, func(s *policy.Score) error {
		if s.QrId == "upsert-score-1" {
			count++
		}
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, 1, count, "Should have exactly one score with the given QR ID")
}

func TestSqliteScanDataStore_UpsertData(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_upsert_data.db")

	defer func() {
		os.RemoveAll(tempDir)
	}()

	assetMrn := "//assets.api.mondoo.com/spaces/test-space/assets/upsert-data-test"

	// Create the store
	store, err := NewSqliteScanDataStore(dbPath, assetMrn)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Initial data
	initialData := &llx.Result{
		CodeId: "upsert-data-1",
		Data:   llx.StringPrimitive("initial data"),
	}

	// Write initial data
	err = store.WriteData(ctx, []*llx.Result{initialData})
	require.NoError(t, err)

	// Verify initial data
	result, err := store.GetData(ctx, "upsert-data-1")
	require.NoError(t, err)
	assert.Equal(t, llx.StringPrimitive("initial data"), result.Data)

	// Updated data with same code_id
	updatedData := &llx.Result{
		CodeId: "upsert-data-1",                     // Same Code ID
		Data:   llx.StringPrimitive("updated data"), // Different value
	}

	// Write updated data (should upsert)
	err = store.WriteData(ctx, []*llx.Result{updatedData})
	require.NoError(t, err)

	// Verify the data was updated, not duplicated
	result, err = store.GetData(ctx, "upsert-data-1")
	require.NoError(t, err)
	assert.Equal(t, llx.StringPrimitive("updated data"), result.Data)

	// Verify there's only one data entry with this Code ID
	count := 0
	err = store.StreamData(ctx, func(codeId string, r *llx.Result) error {
		if codeId == "upsert-data-1" {
			count++
		}
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, 1, count, "Should have exactly one data entry with the given Code ID")
}

func TestSqliteScanDataStore_MixedUpsert(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_mixed_upsert.db")

	defer func() {
		os.RemoveAll(tempDir)
	}()

	assetMrn := "//assets.api.mondoo.com/spaces/test-space/assets/mixed-upsert-test"

	// Create the store
	store, err := NewSqliteScanDataStore(dbPath, assetMrn)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Write batch with some new and some duplicate qr_ids
	scores := []*policy.Score{
		{QrId: "score-1", RiskScore: 70, Type: 1, Value: 80, Weight: 35, Message: "Score 1 v1"},
		{QrId: "score-2", RiskScore: 75, Type: 1, Value: 85, Weight: 40, Message: "Score 2 v1"},
		{QrId: "score-3", RiskScore: 80, Type: 1, Value: 90, Weight: 45, Message: "Score 3 v1"},
	}

	err = store.WriteScores(ctx, scores)
	require.NoError(t, err)

	// Write another batch with updates and new scores
	updatedScores := []*policy.Score{
		{QrId: "score-1", RiskScore: 85, Type: 2, Value: 95, Weight: 50, Message: "Score 1 v2"},  // Update
		{QrId: "score-2", RiskScore: 90, Type: 2, Value: 100, Weight: 55, Message: "Score 2 v2"}, // Update
		{QrId: "score-4", RiskScore: 60, Type: 1, Value: 70, Weight: 30, Message: "Score 4 v1"},  // New
	}

	err = store.WriteScores(ctx, updatedScores)
	require.NoError(t, err)

	// Verify all scores
	score1, err := store.GetScore(ctx, "score-1")
	require.NoError(t, err)
	assert.Equal(t, uint32(85), score1.RiskScore)
	assert.Equal(t, "Score 1 v2", score1.Message)

	score2, err := store.GetScore(ctx, "score-2")
	require.NoError(t, err)
	assert.Equal(t, uint32(90), score2.RiskScore)
	assert.Equal(t, "Score 2 v2", score2.Message)

	score3, err := store.GetScore(ctx, "score-3")
	require.NoError(t, err)
	assert.Equal(t, uint32(80), score3.RiskScore)
	assert.Equal(t, "Score 3 v1", score3.Message) // Unchanged

	score4, err := store.GetScore(ctx, "score-4")
	require.NoError(t, err)
	assert.Equal(t, uint32(60), score4.RiskScore)
	assert.Equal(t, "Score 4 v1", score4.Message) // New

	// Count total scores
	totalCount := 0
	err = store.StreamScores(ctx, func(s *policy.Score) error {
		totalCount++
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, 4, totalCount, "Should have exactly 4 scores total")
}

func TestSqliteScanDataStore_RiskFactors(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_risk_factors.db")

	defer func() {
		os.RemoveAll(tempDir)
	}()

	assetMrn := "//assets.api.mondoo.com/spaces/test-space/assets/risk-test"

	// Create the store
	store, err := NewSqliteScanDataStore(dbPath, assetMrn)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Test risk factors
	risks := []*policy.ScoredRiskFactor{
		{
			Mrn:        "risk-factor-1",
			Risk:       0.85,
			IsToxic:    true,
			IsDetected: true,
		},
		{
			Mrn:        "risk-factor-2",
			Risk:       0.25,
			IsToxic:    false,
			IsDetected: true,
		},
		{
			Mrn:        "risk-factor-3",
			Risk:       0.65,
			IsToxic:    true,
			IsDetected: false,
		},
	}

	// Write risks one by one
	for _, risk := range risks {
		err = store.WriteRisk(ctx, risk)
		require.NoError(t, err)
	}

	// Test GetRisk
	retrievedRisk, err := store.GetRisk(ctx, "risk-factor-1")
	require.NoError(t, err)
	assert.Equal(t, "risk-factor-1", retrievedRisk.Mrn)
	assert.Equal(t, float32(0.85), retrievedRisk.Risk)
	assert.Equal(t, true, retrievedRisk.IsToxic)
	assert.Equal(t, true, retrievedRisk.IsDetected)

	// Test GetRisk for non-existent risk
	_, err = store.GetRisk(ctx, "non-existent-risk")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "risk not found")

	// Test StreamRisk
	var streamedRisks []*policy.ScoredRiskFactor
	err = store.StreamRisks(ctx, func(risk *policy.ScoredRiskFactor) error {
		streamedRisks = append(streamedRisks, risk)
		return nil
	})
	require.NoError(t, err)
	require.Len(t, streamedRisks, 3)

	// Verify all risks were streamed correctly (ordered by mrn)
	expectedMrns := []string{"risk-factor-1", "risk-factor-2", "risk-factor-3"}
	expectedRisks := []float32{0.85, 0.25, 0.65}
	expectedToxic := []bool{true, false, true}
	expectedDetected := []bool{true, true, false}

	for i, risk := range streamedRisks {
		assert.Equal(t, expectedMrns[i], risk.Mrn)
		assert.Equal(t, expectedRisks[i], risk.Risk)
		assert.Equal(t, expectedToxic[i], risk.IsToxic)
		assert.Equal(t, expectedDetected[i], risk.IsDetected)
	}
}

func TestSqliteScanDataStore_UpsertRisk(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_upsert_risk.db")

	defer func() {
		os.RemoveAll(tempDir)
	}()

	assetMrn := "//assets.api.mondoo.com/spaces/test-space/assets/upsert-risk-test"

	// Create the store
	store, err := NewSqliteScanDataStore(dbPath, assetMrn)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Initial risk
	initialRisk := &policy.ScoredRiskFactor{
		Mrn:        "upsert-risk-1",
		Risk:       0.3,
		IsToxic:    false,
		IsDetected: false,
	}

	// Write initial risk
	err = store.WriteRisk(ctx, initialRisk)
	require.NoError(t, err)

	// Verify initial risk
	risk, err := store.GetRisk(ctx, "upsert-risk-1")
	require.NoError(t, err)
	assert.Equal(t, float32(0.3), risk.Risk)
	assert.Equal(t, false, risk.IsToxic)
	assert.Equal(t, false, risk.IsDetected)

	// Updated risk with same mrn
	updatedRisk := &policy.ScoredRiskFactor{
		Mrn:        "upsert-risk-1", // Same MRN
		Risk:       0.9,             // Different values
		IsToxic:    true,
		IsDetected: true,
	}

	// Write updated risk (should upsert)
	err = store.WriteRisk(ctx, updatedRisk)
	require.NoError(t, err)

	// Verify the risk was updated, not duplicated
	risk, err = store.GetRisk(ctx, "upsert-risk-1")
	require.NoError(t, err)
	assert.Equal(t, float32(0.9), risk.Risk)
	assert.Equal(t, true, risk.IsToxic)
	assert.Equal(t, true, risk.IsDetected)

	// Verify there's only one risk with this MRN
	count := 0
	err = store.StreamRisks(ctx, func(r *policy.ScoredRiskFactor) error {
		if r.Mrn == "upsert-risk-1" {
			count++
		}
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, 1, count, "Should have exactly one risk with the given MRN")
}

func TestSqliteScanDataStore_InsertResource(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_upsert_resource.db")

	defer func() {
		os.RemoveAll(tempDir)
	}()

	assetMrn := "//assets.api.mondoo.com/spaces/test-space/assets/upsert-resource-test"
	// Create the store
	store, err := NewSqliteScanDataStore(dbPath, assetMrn)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()
	// Initial resource
	initialResource := &llx.ResourceRecording{
		Resource: "upsert-resource-1",
		Id:       "res-1",
		Fields: map[string]*llx.Result{
			"field1": {CodeId: "field1", Data: llx.StringPrimitive("initial value")},
		},
	}

	// Write initial resource
	err = store.WriteResource(ctx, initialResource)
	require.NoError(t, err)
}
