// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scandb

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"

	_ "github.com/glebarez/go-sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWriteScanWarnings_PresentWithExpectedJSON is the happy path: a scan
// that recorded provider crashes writes them under MetaScanWarnings as a
// JSON array, readable back exactly.
func TestWriteScanWarnings_PresentWithExpectedJSON(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "warnings.db")
	store, err := NewSqliteScanDataStore(dbPath, "//assets/1")
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()
	warnings := []string{
		"the 'os' provider crashed: connection refused",
		"the 'aws' provider crashed: EOF",
	}
	require.NoError(t, store.WriteScanWarnings(ctx, warnings))

	raw, err := store.queries.GetMetadataByKey(ctx, MetaScanWarnings)
	require.NoError(t, err)

	var got []string
	require.NoError(t, json.Unmarshal([]byte(raw), &got))
	assert.Equal(t, warnings, got)
}

// TestWriteScanWarnings_AbsentWhenNeverCalled is the common case: a clean
// scan never calls WriteScanWarnings, so the key must simply not exist --
// not an empty array, not an empty string.
func TestWriteScanWarnings_AbsentWhenNeverCalled(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "no-warnings.db")
	store, err := NewSqliteScanDataStore(dbPath, "//assets/1")
	require.NoError(t, err)
	defer store.Close()

	_, err = store.queries.GetMetadataByKey(context.Background(), MetaScanWarnings)
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

// TestWriteScanWarnings_EmptyIsNoOp mirrors AddScanWarning's contract: an
// empty slice never touches the metadata table at all.
func TestWriteScanWarnings_EmptyIsNoOp(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "empty-warnings.db")
	store, err := NewSqliteScanDataStore(dbPath, "//assets/1")
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()
	require.NoError(t, store.WriteScanWarnings(ctx, nil))
	require.NoError(t, store.WriteScanWarnings(ctx, []string{}))

	_, err = store.queries.GetMetadataByKey(ctx, MetaScanWarnings)
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

// TestWriteScanWarnings_UpsertOnSecondCall covers the coordinator's exact
// concern: WithServices could in principle write scan warnings more than
// once for the same store (a re-finalize). InsertMetadata would fail the
// metadata table's PRIMARY KEY on the second write to the same key;
// WriteScanWarnings must not, and the second value must win.
func TestWriteScanWarnings_UpsertOnSecondCall(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "upsert-warnings.db")
	store, err := NewSqliteScanDataStore(dbPath, "//assets/1")
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()
	require.NoError(t, store.WriteScanWarnings(ctx, []string{"first crash"}))
	require.NoError(t, store.WriteScanWarnings(ctx, []string{"first crash", "second crash"}))

	raw, err := store.queries.GetMetadataByKey(ctx, MetaScanWarnings)
	require.NoError(t, err)
	var got []string
	require.NoError(t, json.Unmarshal([]byte(raw), &got))
	assert.Equal(t, []string{"first crash", "second crash"}, got)

	// Exactly one row for the key -- an upsert, not a second row.
	var count int
	require.NoError(t, store.sqlDB.QueryRowContext(ctx,
		`SELECT count(*) FROM metadata WHERE key = ?`, MetaScanWarnings).Scan(&count))
	assert.Equal(t, 1, count)
}

// TestWriteScanWarnings_ReadOnlyModeFails matches every other Write* method:
// a Finalize'd (or reader-opened) store refuses writes instead of silently
// dropping them.
func TestWriteScanWarnings_ReadOnlyModeFails(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "readonly-warnings.db")
	store, err := NewSqliteScanDataStore(dbPath, "//assets/1")
	require.NoError(t, err)

	ctx := context.Background()
	_, err = store.Finalize()
	require.NoError(t, err)
	defer store.Close()

	err = store.WriteScanWarnings(ctx, []string{"too late"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read-only mode")
}
