// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package checksum

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	// The same driver the cnspec binary registers (apps/cnspec/cnspec.go):
	// modernc.org/sqlite and glebarez both claim the driver name "sqlite",
	// so linking both into one binary panics at init — never import modernc
	// here or in anything the cnspec binary may grow to include.
	_ "github.com/glebarez/go-sqlite"
)

// TestManifestSchema pins that the declared schema is executable and shaped
// as every reader assumes: four checksum tables plus metadata, all
// key-addressable, checksums surviving the int64 bit-pattern round trip.
func TestManifestSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "m.manifest.db")
	db, err := sql.Open("sqlite", path)
	require.NoError(t, err)
	defer db.Close() //nolint:errcheck

	_, err = db.Exec(ManifestSchema)
	require.NoError(t, err)

	// High bit set: the uint64 → int64 storage convention must round-trip.
	cs := uint64(0xffffffffffffffff)
	_, err = db.Exec(`INSERT INTO data (code_id, checksum) VALUES (?, ?)`, "c1", int64(cs))
	require.NoError(t, err)
	var got int64
	require.NoError(t, db.QueryRow(`SELECT checksum FROM data WHERE code_id = ?`, "c1").Scan(&got))
	assert.Equal(t, cs, uint64(got))

	// Resources are keyed by the (name, id) pair.
	_, err = db.Exec(`INSERT INTO resources (name, id, checksum) VALUES (?, ?, ?)`, "user", "root", int64(1))
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO resources (name, id, checksum) VALUES (?, ?, ?)`, "user", "root", int64(2))
	assert.Error(t, err, "the composite key is the identity")

	for _, kv := range [][2]string{
		{ManifestMetaAlgoVersion, "1"},
		{ManifestMetaAssetMrn, "//assets/a1"},
	} {
		_, err = db.Exec(`INSERT INTO metadata (key, value) VALUES (?, ?)`, kv[0], kv[1])
		require.NoError(t, err)
	}
	var algo string
	require.NoError(t, db.QueryRow(`SELECT value FROM metadata WHERE key = ?`, ManifestMetaAlgoVersion).Scan(&algo))
	assert.Equal(t, "1", algo)
}
