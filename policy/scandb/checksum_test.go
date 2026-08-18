// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scandb

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/v13/policy"
	"go.mondoo.com/cnspec/v13/policy/checksum"
	"go.mondoo.com/mql/v13/llx"
	llxchecksum "go.mondoo.com/mql/v13/llx/checksum"
)

// readChecksum reads one row's checksum column back as the uint64 the
// packages produce (SQLite stores the int64 bit pattern).
func readChecksum(t *testing.T, path, query string, args ...any) uint64 {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	require.NoError(t, err)
	defer db.Close() //nolint:errcheck
	var v int64
	require.NoError(t, db.QueryRow(query, args...).Scan(&v))
	return uint64(v)
}

// TestComputeChecksums pins the writer half of the wire contract: the
// checksum columns land in every table bit-equal to what the canonical
// packages produce for the same rows, the metadata announces the algorithm,
// the pass is deterministic, and it folds final table state (an upsert
// before the pass is what gets checksummed).
func TestComputeChecksums(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "scan.db")

	score := &policy.Score{
		QrId: "check1", RiskScore: 25, Type: 2, Value: 100, Weight: 1,
		RiskFactors: &policy.ScoredRiskFactors{Items: []*policy.ScoredRiskFactor{{Mrn: "//r/1", Risk: 0.5}}},
		Sources:     &policy.Sources{Items: []*policy.Source{{Name: "scanner", Version: "1.0"}}},
	}
	data := &llx.Result{CodeId: "data1", Data: llx.StringPrimitive("hello")}
	risk := &policy.ScoredRiskFactor{Mrn: "//r/1", Risk: 0.5, IsToxic: true}
	res := &llx.ResourceRecording{Resource: "user", Id: "root",
		Fields: map[string]*llx.Result{"c1": {CodeId: "c1", Data: llx.StringPrimitive("v1")}}}

	w, err := NewSqliteScanDataStore(path, "//assets/a1")
	require.NoError(t, err)
	require.NoError(t, w.WriteScores(ctx, []*policy.Score{score}))
	require.NoError(t, w.WriteData(ctx, []*llx.Result{data}))
	require.NoError(t, w.WriteRisk(ctx, risk))
	require.NoError(t, w.WriteResource(ctx, res))
	_, err = w.Finalize()
	require.NoError(t, err)

	cw, err := OpenForChecksums(path)
	require.NoError(t, err)
	counts, err := cw.ComputeChecksums(ctx)
	require.NoError(t, err)
	assert.Equal(t, ChecksumCounts{Data: 1, Scores: 1, Risks: 1, Resources: 1}, counts)

	// Deterministic: a second pass over unchanged state rewrites the same
	// values.
	again, err := cw.ComputeChecksums(ctx)
	require.NoError(t, err)
	assert.Equal(t, counts, again)
	require.NoError(t, cw.Close())

	// Every column is bit-equal to the canonical packages' output for the
	// same row — the exact values the server's extract-vs-recompute parity
	// gate compares.
	wantData, err := llxchecksum.HashDataRow("data1", data)
	require.NoError(t, err)
	assert.Equal(t, wantData, readChecksum(t, path, `SELECT checksum FROM data WHERE code_id = ?`, "data1"))

	wantScore, err := checksum.HashScoreRow(score)
	require.NoError(t, err)
	assert.Equal(t, wantScore, readChecksum(t, path, `SELECT checksum FROM scores WHERE qr_id = ?`, "check1"))

	wantRisk, err := checksum.HashRiskRow(risk)
	require.NoError(t, err)
	assert.Equal(t, wantRisk,
		readChecksum(t, path, `SELECT checksum FROM scored_risk_factors WHERE mrn = ?`, "//r/1"))

	wantRes, err := llxchecksum.HashResourceRow(res)
	require.NoError(t, err)
	assert.Equal(t, wantRes, readChecksum(t, path, `SELECT checksum FROM resources WHERE name = ? AND id = ?`, "user", "root"))

	// The metadata key announces the algorithm the columns were written with.
	db, err := sql.Open("sqlite", path)
	require.NoError(t, err)
	var algo string
	require.NoError(t, db.QueryRow(`SELECT value FROM metadata WHERE key = ?`, MetaChecksumAlgoVersion).Scan(&algo))
	require.NoError(t, db.Close())
	assert.Equal(t, llxchecksum.AlgoVersion, algo)
}

// TestComputeChecksumsFoldsFinalState: rows are upserted during a scan; the
// checksum pass must only ever see the final value.
func TestComputeChecksumsFoldsFinalState(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "scan.db")

	w, err := NewSqliteScanDataStore(path, "//assets/a1")
	require.NoError(t, err)
	require.NoError(t, w.WriteScores(ctx, []*policy.Score{{QrId: "check1", Value: 40}}))
	final := &policy.Score{QrId: "check1", Value: 100}
	require.NoError(t, w.WriteScores(ctx, []*policy.Score{final})) // upsert
	_, err = w.Finalize()
	require.NoError(t, err)

	cw, err := OpenForChecksums(path)
	require.NoError(t, err)
	counts, err := cw.ComputeChecksums(ctx)
	require.NoError(t, err)
	require.NoError(t, cw.Close())
	assert.Equal(t, 1, counts.Scores, "one row per qr_id after the upsert")

	want, err := checksum.HashScoreRow(final)
	require.NoError(t, err)
	assert.Equal(t, want, readChecksum(t, path, `SELECT checksum FROM scores WHERE qr_id = ?`, "check1"),
		"the checksum is of the final upserted row, never a stale intermediate")
}

// TestComputeChecksumsReadOnly: a reader must refuse the pass — checksums
// are written by the scan owner, never by a consumer.
func TestComputeChecksumsReadOnly(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "scan.db")
	w, err := NewSqliteScanDataStore(path, "//assets/a1")
	require.NoError(t, err)
	_, err = w.Finalize()
	require.NoError(t, err)

	r, err := NewSqliteScanDataStoreReader(path)
	require.NoError(t, err)
	defer r.Close() //nolint:errcheck
	_, err = r.ComputeChecksums(ctx)
	assert.Error(t, err)
}

// TestComputeChecksumsPreChecksumFile: a scan database written by a client
// that predates the checksum columns must be refused up front — before any
// row is streamed or hashed — with ErrNoChecksumColumns, and must never be
// stamped with checksum_algo_version. The stamped schema_version is "1.0"
// either way, so the column check is the only guard. This covers both an
// old file with rows and the sharper trap: an old file whose tables are
// empty, where the row updates would all no-op and (without the guard) the
// metadata would be committed anyway.
func TestComputeChecksumsPreChecksumFile(t *testing.T) {
	ctx := context.Background()

	// The pre-checksum schema exactly as origin/main declared it.
	oldSchema := `
CREATE TABLE metadata (key TEXT PRIMARY KEY NOT NULL, value TEXT NOT NULL);
CREATE TABLE scores (qr_id TEXT PRIMARY KEY, risk_score INTEGER NOT NULL, type INTEGER NOT NULL,
  value INTEGER NOT NULL, weight INTEGER NOT NULL, message TEXT, risk_factors BLOB, sources BLOB);
CREATE TABLE data (code_id TEXT PRIMARY KEY, data BLOB NOT NULL);
CREATE TABLE scored_risk_factors (mrn TEXT PRIMARY KEY, risk REAL NOT NULL, is_toxic BOOLEAN NOT NULL, is_detected BOOLEAN NOT NULL);
CREATE TABLE resources (name TEXT NOT NULL, id TEXT NOT NULL, data BLOB NOT NULL, PRIMARY KEY (name, id));
CREATE TABLE asset (id INTEGER PRIMARY KEY CHECK (id = 0), data BLOB NOT NULL);
CREATE TABLE asset_filters (code_id TEXT NOT NULL PRIMARY KEY);
`

	for _, tc := range []struct {
		name    string
		withRow bool
	}{
		{name: "with rows", withRow: true},
		{name: "empty tables", withRow: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "old.db")
			db, err := sql.Open("sqlite", path)
			require.NoError(t, err)
			_, err = db.Exec(oldSchema)
			require.NoError(t, err)
			_, err = db.Exec(`INSERT INTO metadata (key, value) VALUES ('schema_version', '1.0')`)
			require.NoError(t, err)
			if tc.withRow {
				_, err = db.Exec(`INSERT INTO scores (qr_id, risk_score, type, value, weight) VALUES ('q1', 10, 2, 100, 1)`)
				require.NoError(t, err)
			}
			require.NoError(t, db.Close())

			cw, err := OpenForChecksums(path)
			require.NoError(t, err)
			defer cw.Close() //nolint:errcheck

			_, err = cw.ComputeChecksums(ctx)
			require.Error(t, err)
			assert.True(t, errors.Is(err, ErrNoChecksumColumns),
				"pre-checksum files must fail with the typed sentinel, got: %v", err)

			// The refusal must leave no trace: no algo-version stamp.
			db, err = sql.Open("sqlite", path)
			require.NoError(t, err)
			defer db.Close() //nolint:errcheck
			var v string
			scanErr := db.QueryRow(`SELECT value FROM metadata WHERE key = ?`, MetaChecksumAlgoVersion).Scan(&v)
			assert.ErrorIs(t, scanErr, sql.ErrNoRows,
				"a refused file must never be stamped with checksum_algo_version")
		})
	}
}

// TestOpenForChecksumsMissingPath: the entry point's contract is an existing
// file — a wrong path must fail at open with a clear error and must never
// fabricate an empty database at that path.
func TestOpenForChecksumsMissingPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exist.db")
	_, err := OpenForChecksums(path)
	require.Error(t, err)
	_, statErr := os.Stat(path)
	assert.True(t, os.IsNotExist(statErr), "a failed open must not leave a stray file behind")
}

// TestReaderIsReadOnlyAtTheDatabase: the checksum ownership contract
// (checksums are written by the scan owner, never a consumer) must be
// enforced by SQLite itself, not only by the Go-level readOnly bool — raw
// SQL through a reader handle must fail, and a reader must never fabricate
// a missing file.
func TestReaderIsReadOnlyAtTheDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "scan.db")
	w, err := NewSqliteScanDataStore(path, "//assets/a1")
	require.NoError(t, err)
	_, err = w.Finalize()
	require.NoError(t, err)
	require.NoError(t, w.Close())

	r, err := NewSqliteScanDataStoreReader(path)
	require.NoError(t, err)
	defer r.Close() //nolint:errcheck
	_, err = r.sqlDB.Exec(`INSERT INTO metadata (key, value) VALUES ('probe', 'x')`)
	assert.Error(t, err, "writes through a reader handle must be rejected by the database")

	// A reader on a missing path must error, not create the file.
	missing := filepath.Join(t.TempDir(), "missing.db")
	r2, err := NewSqliteScanDataStoreReader(missing)
	if err == nil {
		_, err = r2.GetMetadata()
		require.NoError(t, r2.Close())
	}
	assert.Error(t, err, "a reader must not open a nonexistent database")
	_, statErr := os.Stat(missing)
	assert.True(t, os.IsNotExist(statErr), "a reader must never fabricate the file")
}
