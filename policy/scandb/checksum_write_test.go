// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scandb

import (
	"context"
	"database/sql"
	"math"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/v13/policy"
	"go.mondoo.com/cnspec/v13/policy/checksum"
	"go.mondoo.com/mql/v13/llx"
	llxchecksum "go.mondoo.com/mql/v13/llx/checksum"
	"go.mondoo.com/mql/v13/types"
)

// parityCorpus is the divergence-prone write sequence shared by the parity
// tests: empty containers, an upsert, unicode+NUL, error rows,
// rounding-hostile floats, empty resource ids, nil map values. finalScores
// holds the score rows as they exist after the upsert.
type parityCorpus struct {
	finalScores []*policy.Score
	data        []*llx.Result
	risks       []*policy.ScoredRiskFactor
	resources   []*llx.ResourceRecording
}

// writeParityCorpus performs the identical write sequence against any store.
func writeParityCorpus(t *testing.T, ctx context.Context, w *SqliteScanDataStore) parityCorpus {
	t.Helper()

	scores := []*policy.Score{
		{QrId: "bare", RiskScore: 25, Type: 2, Value: 80, Weight: 1},
		{QrId: "empty-containers", Value: 100, RiskFactors: &policy.ScoredRiskFactors{}, Sources: &policy.Sources{}},
		{QrId: "unicode", Value: 1, Message: "žluťoučký kůň 🐎 \x00 with NUL"},
		{QrId: "loaded", RiskScore: 90, Type: 2, Weight: 3, Message: "m",
			RiskFactors: &policy.ScoredRiskFactors{Items: []*policy.ScoredRiskFactor{
				{Mrn: "//r/a", Risk: 0.7, IsToxic: true, Data: map[string]*llx.Result{
					"q1": {CodeId: "q1", Data: llx.StringPrimitive("v")},
				}},
			}},
			Sources: &policy.Sources{Items: []*policy.Source{{
				Name: "s", Url: "https://e.com", Version: "1", Vendor: policy.Source_MICROSOFT,
				FirstDetectedAt: "2026-01-01T00:00:00Z", FixedAt: "2026-02-01T00:00:00Z",
			}}}},
	}
	require.NoError(t, w.WriteScores(ctx, scores))
	// An upsert's replacement row carries its own checksum — final state wins.
	upserted := &policy.Score{QrId: "bare", Value: 99, Message: "final"}
	require.NoError(t, w.WriteScores(ctx, []*policy.Score{upserted}))

	data := []*llx.Result{
		{CodeId: "d-str", Data: llx.StringPrimitive("hello")},
		{CodeId: "d-nil"},
		{CodeId: "d-err", Error: "query failed: boom"},
	}
	require.NoError(t, w.WriteData(ctx, data))

	risks := []*policy.ScoredRiskFactor{
		{Mrn: "//r/1", Risk: 0.7, IsToxic: true, IsDetected: true},
		{Mrn: "//r/2", Risk: float32(math.Copysign(0, -1))},
	}
	for _, r := range risks {
		require.NoError(t, w.WriteRisk(ctx, r))
	}

	resources := []*llx.ResourceRecording{
		{Resource: "singleton", Id: "", Fields: map[string]*llx.Result{"f": {CodeId: "f", Data: llx.StringPrimitive("v")}}},
		// nil map value: folds like the zero Result (mql llx/checksum), so
		// the write-time hash matches the recompute from the stored row.
		{Resource: "nilresult", Id: "y", Fields: map[string]*llx.Result{"gone": nil}},
	}
	for _, rec := range resources {
		require.NoError(t, w.WriteResource(ctx, rec))
	}

	return parityCorpus{
		finalScores: append([]*policy.Score{upserted}, scores[1:]...),
		data:        data,
		risks:       risks,
		resources:   resources,
	}
}

// collectChecksumColumns reads every checksum column of every checksummed
// table, keyed by the row's natural key.
func collectChecksumColumns(t *testing.T, path string) map[string]uint64 {
	t.Helper()
	dst := map[string]uint64{}
	db, err := sql.Open("sqlite", path)
	require.NoError(t, err)
	defer db.Close() //nolint:errcheck
	for _, q := range []string{
		`SELECT 'data:' || code_id, checksum FROM data`,
		`SELECT 'score:' || qr_id, checksum FROM scores`,
		`SELECT 'risk:' || mrn, checksum FROM scored_risk_factors`,
		`SELECT 'res:' || name || '/' || id, checksum FROM resources`,
	} {
		rows, err := db.Query(q)
		require.NoError(t, err)
		for rows.Next() {
			var k string
			var v int64
			require.NoError(t, rows.Scan(&k, &v))
			dst[k] = uint64(v)
		}
		require.NoError(t, rows.Err())
		require.NoError(t, rows.Close())
	}
	return dst
}

// readAlgoStamp returns the checksum_algo_version metadata value, or "" when
// the file is unstamped.
func readAlgoStamp(t *testing.T, path string) string {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	require.NoError(t, err)
	defer db.Close() //nolint:errcheck
	var v string
	err = db.QueryRow(`SELECT value FROM metadata WHERE key = ?`, MetaChecksumAlgoVersion).Scan(&v)
	if err == sql.ErrNoRows {
		return ""
	}
	require.NoError(t, err)
	return v
}

// TestWriteTimeChecksumParity is the write-time contract: rows hashed as
// they are written land bit-identical to what the post-pass (and the
// server's recompute) derives from the stored rows — across the shapes
// that historically diverged and hostile inputs.
func TestWriteTimeChecksumParity(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "scan.db")

	w, err := NewSqliteScanDataStore(path, "//assets/a1", WithWriteTimeChecksums())
	require.NoError(t, err)
	corpus := writeParityCorpus(t, ctx, w)

	cs := w.WriteChecksumStats()
	assert.Equal(t, ChecksumCounts{Data: 3, Scores: 5, Risks: 2, Resources: 2}, cs.Counts,
		"counts are hash operations: the upsert hashes twice for one final row")
	assert.Zero(t, cs.Errors)
	assert.Positive(t, cs.Duration)

	// The scan owner stamps once, when writes are complete — the same call
	// the sqlite datalake makes.
	w.StampChecksums()
	_, err = w.Finalize()
	require.NoError(t, err)
	require.NoError(t, w.Close())

	// Every stored column is bit-equal to the canonical packages' output
	// for the in-memory row — for "bare" that is the upserted replacement:
	// a row's checksum is of the final write.
	for _, s := range corpus.finalScores {
		want, err := checksum.HashScoreRow(s)
		require.NoError(t, err)
		assert.Equal(t, want, readChecksum(t, path, `SELECT checksum FROM scores WHERE qr_id = ?`, s.QrId), s.QrId)
	}
	for _, d := range corpus.data {
		want, err := llxchecksum.HashDataRow(d.CodeId, d)
		require.NoError(t, err)
		assert.Equal(t, want, readChecksum(t, path, `SELECT checksum FROM data WHERE code_id = ?`, d.CodeId), d.CodeId)
	}
	for _, r := range corpus.risks {
		want, err := checksum.HashRiskRow(r)
		require.NoError(t, err)
		assert.Equal(t, want, readChecksum(t, path, `SELECT checksum FROM scored_risk_factors WHERE mrn = ?`, r.Mrn), r.Mrn)
	}
	for _, rec := range corpus.resources {
		want, err := llxchecksum.HashResourceRow(rec)
		require.NoError(t, err)
		assert.Equal(t, want,
			readChecksum(t, path, `SELECT checksum FROM resources WHERE name = ? AND id = ?`, rec.Resource, rec.Id), rec.Resource)
	}

	// The file is stamped: every hash succeeded.
	assert.Equal(t, llxchecksum.AlgoVersion, readAlgoStamp(t, path))

	// Re-running the post-pass over this file rewrites every column to the
	// exact same bits the write path stored — and does not depend on the
	// pre-existing column state.
	before := collectChecksumColumns(t, path)
	cw, err := OpenForChecksums(path)
	require.NoError(t, err)
	_, err = cw.ComputeChecksums(ctx)
	require.NoError(t, err)
	require.NoError(t, cw.Close())
	assert.Equal(t, before, collectChecksumColumns(t, path),
		"write-time and post-pass checksums must be bit-identical for every row")
}

// TestComputeChecksumsMatchesWriteTimePath is the cross-file equivalence:
// the same content written twice — once through the normal write-time path,
// once bare and then stamped by ComputeChecksums — carries bit-identical
// checksum columns and the same algorithm stamp. This is the exact
// guarantee the server relies on: its recompute over an uploaded file must
// agree with what the scanning client wrote, without depending on any
// pre-existing column state (the bare file starts at DEFAULT 0 everywhere).
func TestComputeChecksumsMatchesWriteTimePath(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	writePath := filepath.Join(dir, "writetime.db")
	postPath := filepath.Join(dir, "postpass.db")

	// One file checksummed the normal way: at write time, stamped once by
	// the scan owner.
	ww, err := NewSqliteScanDataStore(writePath, "//assets/a1", WithWriteTimeChecksums())
	require.NoError(t, err)
	writeParityCorpus(t, ctx, ww)
	ww.StampChecksums()
	_, err = ww.Finalize()
	require.NoError(t, err)
	require.NoError(t, ww.Close())

	// The identical content written bare, then stamped by the post-pass.
	wp, err := NewSqliteScanDataStore(postPath, "//assets/a1")
	require.NoError(t, err)
	writeParityCorpus(t, ctx, wp)
	_, err = wp.Finalize()
	require.NoError(t, err)
	require.NoError(t, wp.Close())

	cw, err := OpenForChecksums(postPath)
	require.NoError(t, err)
	counts, err := cw.ComputeChecksums(ctx)
	require.NoError(t, err)
	require.NoError(t, cw.Close())
	assert.Equal(t, ChecksumCounts{Data: 3, Scores: 4, Risks: 2, Resources: 2}, counts,
		"the post-pass hashes final rows: four scores remain after the upsert")

	assert.Equal(t,
		collectChecksumColumns(t, writePath),
		collectChecksumColumns(t, postPath),
		"the write-time path and ComputeChecksums must produce bit-identical columns for identical content")

	// Both paths announce the same algorithm.
	assert.Equal(t, llxchecksum.AlgoVersion, readAlgoStamp(t, writePath))
	assert.Equal(t, llxchecksum.AlgoVersion, readAlgoStamp(t, postPath))
}

// TestWriteTimeChecksumsDisabledByDefault: without the option, writes cost
// zero checksum work, columns keep the schema default, and Finalize never
// stamps.
func TestWriteTimeChecksumsDisabledByDefault(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "scan.db")

	w, err := NewSqliteScanDataStore(path, "//assets/a1")
	require.NoError(t, err)
	require.NoError(t, w.WriteScores(ctx, []*policy.Score{{QrId: "q1", Value: 100}}))
	require.NoError(t, w.WriteData(ctx, []*llx.Result{{CodeId: "d1", Data: llx.StringPrimitive("v")}}))
	cs := w.WriteChecksumStats()
	assert.Zero(t, cs.Counts)
	assert.Zero(t, cs.Duration)
	w.StampChecksums() // no-op on a store without the option
	_, err = w.Finalize()
	require.NoError(t, err)
	require.NoError(t, w.Close())

	assert.Zero(t, readChecksum(t, path, `SELECT checksum FROM scores WHERE qr_id = ?`, "q1"))
	assert.Zero(t, readChecksum(t, path, `SELECT checksum FROM data WHERE code_id = ?`, "d1"))
	assert.Empty(t, readAlgoStamp(t, path), "a store without checksums must never stamp the algo version")
}

// TestWriteTimeChecksumFailureNeverFailsScan pins the failure contract: a
// row whose hash errors is still written (the scan continues), the failure
// is counted, and Finalize leaves the file unstamped so it is consumed
// exactly like a pre-checksum file — fail-open, never a wrong skip.
func TestWriteTimeChecksumFailureNeverFailsScan(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "scan.db")

	w, err := NewSqliteScanDataStore(path, "//assets/a1", WithWriteTimeChecksums())
	require.NoError(t, err)

	// Nesting deeper than llx/checksum's canon bound errors the hash while
	// remaining perfectly writable as a blob.
	deep := llx.StringPrimitive("leaf")
	for i := 0; i < 70; i++ {
		inner, err := deep.MarshalVT()
		require.NoError(t, err)
		deep = &llx.Primitive{Type: string(types.Dict), Value: inner}
	}
	poisoned := &llx.Result{CodeId: "poisoned", Data: deep}
	fine := &llx.Result{CodeId: "fine", Data: llx.StringPrimitive("v")}

	require.NoError(t, w.WriteData(ctx, []*llx.Result{poisoned, fine}),
		"a hash failure must never fail the write")

	cs := w.WriteChecksumStats()
	assert.Equal(t, int64(1), cs.Errors)
	assert.Equal(t, 1, cs.Counts.Data, "the healthy row still got its checksum")

	w.StampChecksums() // refuses: a hash failed, the file must stay unstamped
	_, err = w.Finalize()
	require.NoError(t, err, "a hash failure must never fail Finalize")
	require.NoError(t, w.Close())

	// The poisoned row is stored (scan data intact), without a checksum.
	assert.Zero(t, readChecksum(t, path, `SELECT checksum FROM data WHERE code_id = ?`, "poisoned"))
	assert.NotZero(t, readChecksum(t, path, `SELECT checksum FROM data WHERE code_id = ?`, "fine"))

	// And the file is NOT stamped: partially checksummed never masquerades
	// as checksummed.
	assert.Empty(t, readAlgoStamp(t, path))
}

// TestStampChecksumsWithoutFinalize pins the no-upload path: a kept scan
// database whose scan never reaches Finalize (incognito / --output-scan-db
// without an upstream) must still be stamped by an explicit StampChecksums
// call — the regression that shipped 10 fully-checksummed but unstamped
// files during verification.
func TestStampChecksumsWithoutFinalize(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "scan.db")

	w, err := NewSqliteScanDataStore(path, "//assets/a1", WithWriteTimeChecksums())
	require.NoError(t, err)
	require.NoError(t, w.WriteScores(ctx, []*policy.Score{{QrId: "q1", Value: 100}}))

	w.StampChecksums()
	require.NoError(t, w.Close()) // no Finalize on this path

	assert.Equal(t, llxchecksum.AlgoVersion, readAlgoStamp(t, path),
		"kept files must be stamped even when Finalize never runs")
	assert.NotZero(t, readChecksum(t, path, `SELECT checksum FROM scores WHERE qr_id = ?`, "q1"))

	// Idempotence with the Finalize path: stamping again through a fresh
	// writer-style open must not error or change the value. (Also: a
	// disabled store's StampChecksums is a no-op — covered by
	// TestWriteTimeChecksumsDisabledByDefault via Finalize.)
	w2, err := OpenForChecksums(path)
	require.NoError(t, err)
	_, err = w2.ComputeChecksums(ctx)
	require.NoError(t, err)
	require.NoError(t, w2.Close())
	assert.Equal(t, llxchecksum.AlgoVersion, readAlgoStamp(t, path))
}
