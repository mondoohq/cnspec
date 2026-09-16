// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scandb

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/v13/policy"
	"go.mondoo.com/cnspec/v13/policy/checksum"
	"go.mondoo.com/mql/v13/llx"
	llxchecksum "go.mondoo.com/mql/v13/llx/checksum"
)

// buildManifest extracts a manifest from a scan database — the projection
// the server performs after processing an upload: each kind's key and
// checksum, nothing else, plus the algo-version and asset metadata.
func buildManifest(t *testing.T, scanPath, manifestPath, assetMrn string) {
	t.Helper()
	db, err := sql.Open("sqlite", manifestPath)
	require.NoError(t, err)
	defer db.Close() //nolint:errcheck

	_, err = db.Exec(checksum.ManifestSchema)
	require.NoError(t, err)
	_, err = db.Exec(`ATTACH DATABASE ? AS scan`, scanPath)
	require.NoError(t, err)
	for _, q := range []string{
		`INSERT INTO data SELECT code_id, checksum FROM scan.data`,
		`INSERT INTO scores SELECT qr_id, checksum FROM scan.scores`,
		`INSERT INTO scored_risk_factors SELECT mrn, checksum FROM scan.scored_risk_factors`,
		`INSERT INTO resources SELECT name, id, checksum FROM scan.resources`,
	} {
		_, err = db.Exec(q)
		require.NoError(t, err)
	}
	_, err = db.Exec(`INSERT INTO metadata SELECT key, value FROM scan.metadata WHERE key = ?`,
		checksum.ManifestMetaAlgoVersion)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO metadata (key, value) VALUES (?, ?)`,
		checksum.ManifestMetaAssetMrn, assetMrn)
	require.NoError(t, err)
	_, err = db.Exec(`DETACH DATABASE scan`)
	require.NoError(t, err)
}

// manifestDiff is the minimal assertion: key-addressed SQL over a scan
// database and a manifest, per kind — changed (both sides, different
// checksum), added (scan only), removed (manifest only). Pure SQL over the
// two files, so either side can run it.
type manifestDiff struct {
	changed, added, removed []string
}

func (d manifestDiff) unchanged() bool {
	return len(d.changed) == 0 && len(d.added) == 0 && len(d.removed) == 0
}

func diffAgainstManifest(t *testing.T, scanPath, manifestPath string) manifestDiff {
	t.Helper()
	db, err := sql.Open("sqlite", scanPath)
	require.NoError(t, err)
	defer db.Close() //nolint:errcheck
	_, err = db.Exec(`ATTACH DATABASE ? AS m`, manifestPath)
	require.NoError(t, err)

	collect := func(dst *[]string, query string) {
		rows, err := db.Query(query)
		require.NoError(t, err)
		defer rows.Close() //nolint:errcheck
		for rows.Next() {
			var k string
			require.NoError(t, rows.Scan(&k))
			*dst = append(*dst, k)
		}
		require.NoError(t, rows.Err())
	}

	var d manifestDiff
	for _, kind := range []struct{ prefix, table, key string }{
		{"data:", "data", "code_id"},
		{"score:", "scores", "qr_id"},
		{"risk:", "scored_risk_factors", "mrn"},
	} {
		collect(&d.changed, `SELECT '`+kind.prefix+`' || s.`+kind.key+` FROM `+kind.table+` s JOIN m.`+kind.table+` mm USING(`+kind.key+`) WHERE s.checksum != mm.checksum`)
		collect(&d.added, `SELECT '`+kind.prefix+`' || `+kind.key+` FROM (SELECT `+kind.key+` FROM `+kind.table+` EXCEPT SELECT `+kind.key+` FROM m.`+kind.table+`)`)
		collect(&d.removed, `SELECT '`+kind.prefix+`' || `+kind.key+` FROM (SELECT `+kind.key+` FROM m.`+kind.table+` EXCEPT SELECT `+kind.key+` FROM `+kind.table+`)`)
	}
	collect(&d.changed, `SELECT 'res:' || s.name || '/' || s.id FROM resources s JOIN m.resources mm USING(name, id) WHERE s.checksum != mm.checksum`)
	collect(&d.added, `SELECT 'res:' || name || '/' || id FROM (SELECT name, id FROM resources EXCEPT SELECT name, id FROM m.resources)`)
	collect(&d.removed, `SELECT 'res:' || name || '/' || id FROM (SELECT name, id FROM m.resources EXCEPT SELECT name, id FROM resources)`)
	return d
}

// TestManifestMinimalDiffAssertion proves the manifest schema carries
// everything the unchanged-scan short-circuit needs, from either side:
// a manifest extracted from one scan compares against another scan's rows
// with nothing but key-addressed SQL — equal content reads as unchanged
// (skip the upload), and a real change is attributed to exactly its rows.
func TestManifestMinimalDiffAssertion(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	// Scan A: the baseline upload the server processed and staged a
	// manifest for.
	pathA := filepath.Join(dir, "a.db")
	wa, err := NewSqliteScanDataStore(pathA, "//assets/a1", WithWriteTimeChecksums())
	require.NoError(t, err)
	writeParityCorpus(t, ctx, wa)
	wa.StampChecksums()
	_, err = wa.Finalize()
	require.NoError(t, err)
	require.NoError(t, wa.Close())

	manifestPath := filepath.Join(dir, "a.manifest.db")
	buildManifest(t, pathA, manifestPath, "//assets/a1")

	// The reader's gate: the manifest announces the same algorithm the
	// client runs; a version mismatch would mean "treat as absent".
	assert.Equal(t, llxchecksum.AlgoVersion, readAlgoStamp(t, manifestPath))

	// Scan B: identical content. The minimal assertion must read it as
	// unchanged — the upload short-circuit case.
	pathB := filepath.Join(dir, "b.db")
	wb, err := NewSqliteScanDataStore(pathB, "//assets/a1", WithWriteTimeChecksums())
	require.NoError(t, err)
	writeParityCorpus(t, ctx, wb)
	wb.StampChecksums()
	_, err = wb.Finalize()
	require.NoError(t, err)
	require.NoError(t, wb.Close())

	diff := diffAgainstManifest(t, pathB, manifestPath)
	assert.True(t, diff.unchanged(),
		"identical content must diff clean against the manifest, got changed=%v added=%v removed=%v",
		diff.changed, diff.added, diff.removed)

	// Scan C: one score's value flips, one data row appears, one risk row
	// disappears. The diff must attribute exactly those keys — change
	// attribution, not just a boolean.
	pathC := filepath.Join(dir, "c.db")
	wc, err := NewSqliteScanDataStore(pathC, "//assets/a1", WithWriteTimeChecksums())
	require.NoError(t, err)
	writeParityCorpus(t, ctx, wc)
	require.NoError(t, wc.WriteScores(ctx, []*policy.Score{{QrId: "unicode", Value: 2, Message: "žluťoučký kůň 🐎 \x00 with NUL"}}))
	require.NoError(t, wc.WriteData(ctx, []*llx.Result{{CodeId: "d-new", Data: llx.StringPrimitive("appeared")}}))
	wc.StampChecksums()
	_, err = wc.Finalize()
	require.NoError(t, err)
	require.NoError(t, wc.Close())
	// The store has no delete API (scans only upsert); drop a row directly
	// to exercise the removed leg of the diff.
	db, err := sql.Open("sqlite", pathC)
	require.NoError(t, err)
	_, err = db.Exec(`DELETE FROM scored_risk_factors WHERE mrn = '//r/2'`)
	require.NoError(t, err)
	require.NoError(t, db.Close())

	diff = diffAgainstManifest(t, pathC, manifestPath)
	assert.ElementsMatch(t, []string{"score:unicode"}, diff.changed)
	assert.ElementsMatch(t, []string{"data:d-new"}, diff.added)
	assert.ElementsMatch(t, []string{"risk://r/2"}, diff.removed)
}
