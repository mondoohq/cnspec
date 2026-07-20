// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scandb

import (
	"context"
	"path/filepath"
	"testing"

	_ "github.com/glebarez/go-sqlite"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.mondoo.com/cnspec/v13/policy"
	"go.mondoo.com/mql/v13/llx"
)

func TestErroredScoreQrIds(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "scan_counts.db")
	ctx := context.Background()

	store, err := NewSqliteScanDataStore(dbPath, "//assets/1")
	require.NoError(t, err)
	defer store.Close()

	scores := []*policy.Score{
		// Result-typed scores — should NOT appear in ErroredScoreQrIds
		{QrId: "result-1", RiskScore: 80, Type: uint32(policy.ScoreType_Result), Value: 90, Weight: 10, Message: "ok"},
		{QrId: "result-2", RiskScore: 70, Type: uint32(policy.ScoreType_Result), Value: 80, Weight: 10, Message: "ok"},
		{QrId: "result-3", RiskScore: 60, Type: uint32(policy.ScoreType_Result), Value: 70, Weight: 10, Message: "ok"},
		// Error-typed scores — should appear in ErroredScoreQrIds
		{QrId: "error-1", RiskScore: 0, Type: uint32(policy.ScoreType_Error), Value: 0, Weight: 10, Message: "boom"},
		{QrId: "error-2", RiskScore: 0, Type: uint32(policy.ScoreType_Error), Value: 0, Weight: 10, Message: "crash"},
	}

	err = store.WriteScores(ctx, scores)
	require.NoError(t, err)

	erroredQrIds, err := store.ErroredScoreQrIds(ctx)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"error-1", "error-2"}, erroredQrIds)
}

func TestErroredDataCount(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "scan_counts_data.db")
	ctx := context.Background()

	store, err := NewSqliteScanDataStore(dbPath, "//assets/1")
	require.NoError(t, err)
	defer store.Close()

	data := []*llx.Result{
		// Results without errors
		{CodeId: "data-ok-1", Data: llx.BoolPrimitive(true)},
		{CodeId: "data-ok-2", Data: llx.BoolPrimitive(false)},
		{CodeId: "data-ok-3", Data: llx.StringPrimitive("hello")},
		// Results with errors
		{CodeId: "data-err-1", Error: "boom"},
		{CodeId: "data-err-2", Error: "something went wrong"},
	}

	err = store.WriteData(ctx, data)
	require.NoError(t, err)

	count, err := store.ErroredDataCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

func TestErroredCounts_EmptyStore(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "scan_counts_empty.db")
	ctx := context.Background()

	store, err := NewSqliteScanDataStore(dbPath, "//assets/1")
	require.NoError(t, err)
	defer store.Close()

	erroredQrIds, err := store.ErroredScoreQrIds(ctx)
	require.NoError(t, err)
	assert.Empty(t, erroredQrIds)

	count, err := store.ErroredDataCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}
