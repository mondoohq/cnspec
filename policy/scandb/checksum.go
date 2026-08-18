// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scandb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"go.mondoo.com/cnspec/v13/policy"
	"go.mondoo.com/cnspec/v13/policy/checksum"
	"go.mondoo.com/cnspec/v13/policy/scandb/sqlc"
	"go.mondoo.com/mql/v13/llx"
	llxchecksum "go.mondoo.com/mql/v13/llx/checksum"
	"google.golang.org/protobuf/proto"
)

// Scan-content checksums: every checksummed row gets an 8-byte checksum
// stored in its table's `checksum` column, and the metadata table announces
// the algorithm via checksum_algo_version. There is no kind- or asset-level
// fold anywhere: row equality (the manifest diff — server-side today, the
// client's manifest pull in client_compare later) is the only comparison
// currency, and checksums from different algorithm versions never compare.
// The (key, resource_id) pair is derived, never stored: key is the row's
// own primary key, resource_id is resources' id column ('' for every other
// kind). The asset and asset_filters tables are exempt (replay-only; ingest
// never consumes them).
//
// The CLIENT hashes at write time (see WithWriteTimeChecksums in
// scan_data_store.go) — no second pass over the database during a scan.
// This file is the post-pass over ALREADY-WRITTEN files: server-side
// stamping and verification, backfill, tooling, benchmarks. Both paths are
// bit-identical by construction and pinned so by TestWriteTimeChecksumParity.
//
// The row checksums come from mql's llx/checksum (data, resources) and
// cnspec's policy/checksum (scores, risks) — never from local code.

// MetaChecksumAlgoVersion is the metadata key announcing which checksum
// algorithm wrote the checksum columns. Deliberately the manifest's own
// key (the manifest is a trimmed scandb), referenced rather than repeated
// so the two can never diverge.
const MetaChecksumAlgoVersion = checksum.ManifestMetaAlgoVersion

// ErrNoChecksumColumns reports a scan database whose tables lack the
// `checksum` columns — one written by a client that predates them. The
// stamped schema_version is "1.0" either way (see SchemaVersion), so column
// presence is the only way to tell; callers doing bulk stamping should treat
// this as "old file, skip or migrate", not corruption.
var ErrNoChecksumColumns = errors.New("scan database predates checksum columns")

// rowEntry is one checksummed row: its rowid and checksum. The rowid is the
// one key shape every table shares (all four are ordinary rowid tables —
// none is WITHOUT ROWID — so even resources' composite (name, id) primary
// key collapses to a single integer). Rowids are stable for exactly the
// window this pass uses them: hash phase and update phase run back to back
// on the same open handle, with no deletes and no VACUUM in between
// (Finalize's VACUUM comes after).
type rowEntry struct {
	rid      int64
	checksum uint64
}

// ChecksumCounts reports how many rows were checksummed per table.
type ChecksumCounts struct {
	Data      int
	Scores    int
	Risks     int
	Resources int
}

func (d ChecksumCounts) String() string {
	return fmt.Sprintf("data=%d scores=%d risks=%d resources=%d",
		d.Data, d.Scores, d.Risks, d.Resources)
}

// OpenForChecksums opens an existing scan database read-write without
// re-initializing the schema — for re-running ComputeChecksums over an
// already-written file (server-side stamping, tooling, benchmarks). Files
// from pre-checksum clients open fine but fail ComputeChecksums with
// ErrNoChecksumColumns. A missing path is an error — this entry point's
// contract is an EXISTING file, so it must never fabricate one.
func OpenForChecksums(filePath string) (*SqliteScanDataStore, error) {
	// Stat first for a clear error (a bare driver error for a missing file
	// reads like corruption), then open via a file: URI with mode=rw —
	// read-write without create — so even a race cannot fabricate the file
	// the way a plain-path DSN (READWRITE|CREATE) silently would.
	if _, err := os.Stat(filePath); err != nil {
		return nil, fmt.Errorf("cannot open scan database: %w", err)
	}
	sqlDB, err := sql.Open("sqlite", "file:"+filepath.ToSlash(filePath)+"?mode=rw")
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite file: %w", err)
	}
	queries, err := sqlc.Prepare(context.Background(), sqlDB)
	if err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("failed to prepare queries: %w", err)
	}
	return &SqliteScanDataStore{
		sqlDB:    sqlDB,
		queries:  queries,
		filePath: filePath,
		readOnly: false,
	}, nil
}

// checksumTargets names every checksummed table and its stage-apply
// statement, in the order the pass processes them. The statements are
// compile-time literals — no SQL is ever assembled from data (or from
// anything else: even identifier interpolation via Sprintf trips SQL
// scanners for no benefit when the full set of tables is known here).
var checksumTargets = []struct {
	table     string
	updateSQL string
}{
	{"data", `UPDATE data SET checksum = s.checksum FROM _checksum_stage AS s WHERE data.rowid = s.rid`},
	{"scores", `UPDATE scores SET checksum = s.checksum FROM _checksum_stage AS s WHERE scores.rowid = s.rid`},
	{"scored_risk_factors", `UPDATE scored_risk_factors SET checksum = s.checksum FROM _checksum_stage AS s WHERE scored_risk_factors.rowid = s.rid`},
	{"resources", `UPDATE resources SET checksum = s.checksum FROM _checksum_stage AS s WHERE resources.rowid = s.rid`},
}

// ensureChecksumColumns fails fast — before any row is read or hashed —
// when the file predates the checksum columns, so old files can never be
// half-processed or falsely stamped.
func (s *SqliteScanDataStore) ensureChecksumColumns(ctx context.Context) error {
	for _, target := range checksumTargets {
		var n int
		if err := s.sqlDB.QueryRowContext(ctx,
			`SELECT count(*) FROM pragma_table_info(?) WHERE name = 'checksum'`,
			target.table).Scan(&n); err != nil {
			return fmt.Errorf("failed to inspect %s schema: %w", target.table, err)
		}
		if n == 0 {
			return fmt.Errorf("table %s has no checksum column: %w", target.table, ErrNoChecksumColumns)
		}
	}
	return nil
}

// ComputeChecksums computes the per-row checksums for every checksummed
// table, stores them in each table's `checksum` column, and — only after
// every row's update has been verified against the streamed row count —
// announces the algorithm version in the metadata table. Databases written
// by pre-checksum clients are refused up front with ErrNoChecksumColumns
// rather than half-processed or falsely stamped.
//
// The scanning client does NOT run this: it hashes at write time
// (WithWriteTimeChecksums), which produces bit-identical columns. This
// pass exists for already-written files — server-side stamping and
// verification recompute, backfill, tooling — and reads with row-at-a-time
// cursors (never whole-table loads), hashing each stored row exactly as a
// reader decodes it.
func (s *SqliteScanDataStore) ComputeChecksums(ctx context.Context) (ChecksumCounts, error) {
	if s.readOnly {
		return ChecksumCounts{}, fmt.Errorf("cannot compute checksums in read-only mode")
	}
	if err := s.ensureChecksumColumns(ctx); err != nil {
		return ChecksumCounts{}, err
	}

	dataRows, err := s.hashDataRows(ctx)
	if err != nil {
		return ChecksumCounts{}, err
	}
	scoreRows, err := s.hashScoreRows(ctx)
	if err != nil {
		return ChecksumCounts{}, err
	}
	riskRows, err := s.hashRiskRows(ctx)
	if err != nil {
		return ChecksumCounts{}, err
	}
	resourceRows, err := s.hashResourceRows(ctx)
	if err != nil {
		return ChecksumCounts{}, err
	}

	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return ChecksumCounts{}, fmt.Errorf("failed to begin checksum tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	entriesFor := map[string][]rowEntry{
		"data":                dataRows,
		"scores":              scoreRows,
		"scored_risk_factors": riskRows,
		"resources":           resourceRows,
	}
	for _, target := range checksumTargets {
		if err := applyChecksums(ctx, tx, target.updateSQL, entriesFor[target.table]); err != nil {
			return ChecksumCounts{}, fmt.Errorf("failed to write %s checksums: %w", target.table, err)
		}
	}

	if _, err := tx.ExecContext(ctx,
		`INSERT OR REPLACE INTO metadata (key, value) VALUES (?, ?)`,
		MetaChecksumAlgoVersion, llxchecksum.AlgoVersion); err != nil {
		return ChecksumCounts{}, fmt.Errorf("failed to write metadata %s: %w", MetaChecksumAlgoVersion, err)
	}

	if err := tx.Commit(); err != nil {
		return ChecksumCounts{}, fmt.Errorf("failed to commit checksums: %w", err)
	}

	return ChecksumCounts{
		Data:      len(dataRows),
		Scores:    len(scoreRows),
		Risks:     len(riskRows),
		Resources: len(resourceRows),
	}, nil
}

// hashDataRows cursors over the data table, holding one row at a time.
func (s *SqliteScanDataStore) hashDataRows(ctx context.Context) ([]rowEntry, error) {
	rows, err := s.sqlDB.QueryContext(ctx, `SELECT rowid, code_id, data FROM data`)
	if err != nil {
		return nil, fmt.Errorf("failed to query data: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	var entries []rowEntry
	for rows.Next() {
		var rid int64
		var codeID string
		var blob []byte
		if err := rows.Scan(&rid, &codeID, &blob); err != nil {
			return nil, fmt.Errorf("failed to scan data row: %w", err)
		}
		result := &llx.Result{}
		if err := proto.Unmarshal(blob, result); err != nil {
			return nil, fmt.Errorf("failed to unmarshal result %s: %w", codeID, err)
		}
		d, err := llxchecksum.HashDataRow(codeID, result)
		if err != nil {
			return nil, fmt.Errorf("data row %s: %w", codeID, err)
		}
		entries = append(entries, rowEntry{rid: rid, checksum: d})
	}
	return entries, rows.Err()
}

// hashScoreRows cursors over the scores table, decoding each row through
// convertScore so the hashed value is exactly what every reader sees.
func (s *SqliteScanDataStore) hashScoreRows(ctx context.Context) ([]rowEntry, error) {
	rows, err := s.sqlDB.QueryContext(ctx,
		`SELECT rowid, qr_id, risk_score, type, value, weight, message, risk_factors, sources FROM scores`)
	if err != nil {
		return nil, fmt.Errorf("failed to query scores: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	var entries []rowEntry
	for rows.Next() {
		var rid int64
		var row sqlc.StreamScoresRow
		if err := rows.Scan(&rid, &row.QrID, &row.RiskScore, &row.Type, &row.Value,
			&row.Weight, &row.Message, &row.RiskFactors, &row.Sources); err != nil {
			return nil, fmt.Errorf("failed to scan score row: %w", err)
		}
		score, err := s.convertScore(&row)
		if err != nil {
			return nil, fmt.Errorf("failed to convert score %s: %w", row.QrID, err)
		}
		d, err := checksum.HashScoreRow(score)
		if err != nil {
			return nil, fmt.Errorf("score row %s: %w", score.QrId, err)
		}
		entries = append(entries, rowEntry{rid: rid, checksum: d})
	}
	return entries, rows.Err()
}

// hashRiskRows cursors over the scored_risk_factors table, mirroring
// StreamRisks' column decoding (REAL narrowed to the proto's float32).
func (s *SqliteScanDataStore) hashRiskRows(ctx context.Context) ([]rowEntry, error) {
	rows, err := s.sqlDB.QueryContext(ctx,
		`SELECT rowid, mrn, risk, is_toxic, is_detected FROM scored_risk_factors`)
	if err != nil {
		return nil, fmt.Errorf("failed to query risk factors: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	var entries []rowEntry
	for rows.Next() {
		var rid int64
		var mrn string
		var riskValue float64
		var isToxic, isDetected bool
		if err := rows.Scan(&rid, &mrn, &riskValue, &isToxic, &isDetected); err != nil {
			return nil, fmt.Errorf("failed to scan risk row: %w", err)
		}
		d, err := checksum.HashRiskRow(&policy.ScoredRiskFactor{
			Mrn:        mrn,
			Risk:       float32(riskValue),
			IsToxic:    isToxic,
			IsDetected: isDetected,
		})
		if err != nil {
			return nil, fmt.Errorf("risk row %s: %w", mrn, err)
		}
		entries = append(entries, rowEntry{rid: rid, checksum: d})
	}
	return entries, rows.Err()
}

// hashResourceRows cursors over the resources table. The update key is the
// physical rowid — never the blob-internal identity — so a row whose
// payload disagrees with its key columns can never be skipped silently (the
// row-count verification in applyChecksums would catch it regardless).
func (s *SqliteScanDataStore) hashResourceRows(ctx context.Context) ([]rowEntry, error) {
	rows, err := s.sqlDB.QueryContext(ctx, `SELECT rowid, name, id, data FROM resources`)
	if err != nil {
		return nil, fmt.Errorf("failed to query resources: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	var entries []rowEntry
	for rows.Next() {
		var rid int64
		var name, id string
		var blob []byte
		if err := rows.Scan(&rid, &name, &id, &blob); err != nil {
			return nil, fmt.Errorf("failed to scan resource row: %w", err)
		}
		rec := &llx.ResourceRecording{}
		if err := proto.Unmarshal(blob, rec); err != nil {
			return nil, fmt.Errorf("failed to unmarshal resource %s/%s: %w", name, id, err)
		}
		d, err := llxchecksum.HashResourceRow(rec)
		if err != nil {
			return nil, fmt.Errorf("resource row %s/%s: %w", name, id, err)
		}
		entries = append(entries, rowEntry{rid: rid, checksum: d})
	}
	return entries, rows.Err()
}

// applyStageChunk is the number of staged rows per INSERT statement. At 2
// bound parameters per row it stays well under SQLite's most conservative
// host-parameter limit (999).
const applyStageChunk = 300

// applyChecksums stages the entries in a temp table and applies them with a
// single UPDATE ... FROM per table, inside the caller's transaction. The
// update statement is one of the compile-time literals in checksumTargets.
// The update's affected-row count must equal the number of streamed entries
// — a mismatch (a key that matched nothing) aborts the pass so a row can
// never be reported as checksummed while keeping its DEFAULT 0 sentinel.
func applyChecksums(ctx context.Context, tx *sql.Tx, updateSQL string, entries []rowEntry) error {
	if len(entries) == 0 {
		return nil
	}

	// Self-healing stage lifecycle: reset any leftover stage before creating
	// (so a previous call's failed drop can never resurface here as a
	// misleading "table already exists"), and drop error-checked on the
	// success path. Error paths return without dropping — the caller's
	// rollback undoes the CREATE TEMP TABLE along with everything else, and
	// the reset here covers even the case where it somehow does not.
	if _, err := tx.ExecContext(ctx, `DROP TABLE IF EXISTS _checksum_stage`); err != nil {
		return fmt.Errorf("failed to reset checksum stage: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		`CREATE TEMP TABLE _checksum_stage (rid INTEGER NOT NULL, checksum INTEGER NOT NULL)`); err != nil {
		return fmt.Errorf("failed to create checksum stage: %w", err)
	}

	for start := 0; start < len(entries); start += applyStageChunk {
		end := min(start+applyStageChunk, len(entries))
		chunk := entries[start:end]
		query := `INSERT INTO _checksum_stage (rid, checksum) VALUES `
		args := make([]any, 0, len(chunk)*2)
		for i, e := range chunk {
			if i > 0 {
				query += ","
			}
			query += "(?, ?)"
			// SQLite INTEGER is int64; store the uint64 bit pattern.
			args = append(args, e.rid, int64(e.checksum))
		}
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return fmt.Errorf("failed to stage checksums: %w", err)
		}
	}

	res, err := tx.ExecContext(ctx, updateSQL)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to read affected rows: %w", err)
	}
	if affected != int64(len(entries)) {
		return fmt.Errorf("checksum update matched %d of %d rows", affected, len(entries))
	}

	if _, err := tx.ExecContext(ctx, `DROP TABLE IF EXISTS _checksum_stage`); err != nil {
		return fmt.Errorf("failed to drop checksum stage: %w", err)
	}
	return nil
}
