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

	"go.mondoo.com/cnspec/policy"
)

const resolvedPolicyTestAssetMrn = "//assets.api.mondoo.com/spaces/test-space/assets/resolved-policy-asset"

func fullResolvedPolicy(suffix string) *policy.ResolvedPolicy {
	return &policy.ResolvedPolicy{
		GraphExecutionChecksum: "graph-" + suffix,
		FiltersChecksum:        "filters-" + suffix,
		ExecutionJob:           &policy.ExecutionJob{Checksum: "execution-" + suffix},
		CollectorJob:           &policy.CollectorJob{Checksum: "collector-" + suffix},
	}
}

// metadataRows returns the raw metadata table, so a test can tell a key that
// was never written from one written empty.
func metadataRows(t *testing.T, store *SqliteScanDataStore) map[string]string {
	t.Helper()
	rows, err := store.queries.GetMetadata(context.Background())
	require.NoError(t, err)
	out := make(map[string]string, len(rows))
	for _, r := range rows {
		out[r.Key] = r.Value
	}
	return out
}

// finalizeAndReopen seals the store the way a scan does and reopens the file
// as a consumer would.
func finalizeAndReopen(t *testing.T, store *SqliteScanDataStore, dbPath string) *SqliteScanDataStore {
	t.Helper()
	_, err := store.Finalize()
	require.NoError(t, err)
	require.NoError(t, store.Close())
	reader, err := NewSqliteScanDataStoreReader(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = reader.Close() })
	return reader
}

func TestSqliteScanDataStore_ResolvedPolicyChecksums(t *testing.T) {
	ctx := context.Background()

	t.Run("written values survive finalize and read back through GetMetadata", func(t *testing.T) {
		dbPath := filepath.Join(t.TempDir(), "rp.db")
		store, err := NewSqliteScanDataStore(dbPath, resolvedPolicyTestAssetMrn)
		require.NoError(t, err)

		require.NoError(t, store.WriteResolvedPolicyChecksums(ctx, fullResolvedPolicy("1")))

		reader := finalizeAndReopen(t, store, dbPath)
		meta, err := reader.GetMetadata()
		require.NoError(t, err)
		assert.Equal(t, "graph-1", meta.ResolvedPolicyGraphExecutionChecksum)
		assert.Equal(t, "filters-1", meta.ResolvedPolicyFiltersChecksum)
		assert.Equal(t, "execution-1", meta.ResolvedPolicyExecutionJobChecksum)
		assert.Equal(t, "collector-1", meta.ResolvedPolicyCollectorJobChecksum)

		// The keys that were always there are untouched.
		assert.Equal(t, SchemaVersion, meta.SchemaVersion)
		assert.Equal(t, resolvedPolicyTestAssetMrn, meta.AssetMrn)
		assert.NotEmpty(t, meta.ClientVersion)
	})

	t.Run("a store that never reported a resolved policy has no such keys", func(t *testing.T) {
		dbPath := filepath.Join(t.TempDir(), "rp.db")
		store, err := NewSqliteScanDataStore(dbPath, resolvedPolicyTestAssetMrn)
		require.NoError(t, err)

		reader := finalizeAndReopen(t, store, dbPath)
		meta, err := reader.GetMetadata()
		require.NoError(t, err)
		assert.Empty(t, meta.ResolvedPolicyGraphExecutionChecksum)
		assert.Empty(t, meta.ResolvedPolicyFiltersChecksum)
		assert.Empty(t, meta.ResolvedPolicyExecutionJobChecksum)
		assert.Empty(t, meta.ResolvedPolicyCollectorJobChecksum)

		rows := metadataRows(t, reader)
		for _, key := range []string{
			MetaResolvedPolicyGraphExecutionChecksum,
			MetaResolvedPolicyFiltersChecksum,
			MetaResolvedPolicyExecutionJobChecksum,
			MetaResolvedPolicyCollectorJobChecksum,
		} {
			_, present := rows[key]
			assert.False(t, present, "key %s must be absent, not empty", key)
		}
	})

	t.Run("a second write replaces the values", func(t *testing.T) {
		dbPath := filepath.Join(t.TempDir(), "rp.db")
		store, err := NewSqliteScanDataStore(dbPath, resolvedPolicyTestAssetMrn)
		require.NoError(t, err)

		require.NoError(t, store.WriteResolvedPolicyChecksums(ctx, fullResolvedPolicy("1")))
		require.NoError(t, store.WriteResolvedPolicyChecksums(ctx, fullResolvedPolicy("2")))

		reader := finalizeAndReopen(t, store, dbPath)
		meta, err := reader.GetMetadata()
		require.NoError(t, err)
		assert.Equal(t, "graph-2", meta.ResolvedPolicyGraphExecutionChecksum)
		assert.Equal(t, "filters-2", meta.ResolvedPolicyFiltersChecksum)
		assert.Equal(t, "execution-2", meta.ResolvedPolicyExecutionJobChecksum)
		assert.Equal(t, "collector-2", meta.ResolvedPolicyCollectorJobChecksum)
	})

	t.Run("a second write removes keys whose value became empty", func(t *testing.T) {
		dbPath := filepath.Join(t.TempDir(), "rp.db")
		store, err := NewSqliteScanDataStore(dbPath, resolvedPolicyTestAssetMrn)
		require.NoError(t, err)

		require.NoError(t, store.WriteResolvedPolicyChecksums(ctx, fullResolvedPolicy("1")))
		// The re-resolve carries no execution or collector job: the keys
		// the first write left must not survive as the stale "1" values.
		require.NoError(t, store.WriteResolvedPolicyChecksums(ctx, &policy.ResolvedPolicy{
			GraphExecutionChecksum: "graph-2",
			FiltersChecksum:        "filters-2",
		}))

		reader := finalizeAndReopen(t, store, dbPath)
		meta, err := reader.GetMetadata()
		require.NoError(t, err)
		assert.Equal(t, "graph-2", meta.ResolvedPolicyGraphExecutionChecksum)
		assert.Equal(t, "filters-2", meta.ResolvedPolicyFiltersChecksum)
		assert.Empty(t, meta.ResolvedPolicyExecutionJobChecksum)
		assert.Empty(t, meta.ResolvedPolicyCollectorJobChecksum)

		rows := metadataRows(t, reader)
		_, present := rows[MetaResolvedPolicyExecutionJobChecksum]
		assert.False(t, present, "the first write's execution job checksum is removed, not kept")
		_, present = rows[MetaResolvedPolicyCollectorJobChecksum]
		assert.False(t, present, "the first write's collector job checksum is removed, not kept")
		assert.Equal(t, SchemaVersion, rows["schema_version"], "the other metadata keys are untouched")
	})

	t.Run("only non-empty values are written", func(t *testing.T) {
		dbPath := filepath.Join(t.TempDir(), "rp.db")
		store, err := NewSqliteScanDataStore(dbPath, resolvedPolicyTestAssetMrn)
		require.NoError(t, err)

		// No execution or collector job, as a resolve with nothing to run.
		require.NoError(t, store.WriteResolvedPolicyChecksums(ctx, &policy.ResolvedPolicy{
			GraphExecutionChecksum: "graph-only",
			FiltersChecksum:        "filters-only",
		}))

		reader := finalizeAndReopen(t, store, dbPath)
		meta, err := reader.GetMetadata()
		require.NoError(t, err)
		assert.Equal(t, "graph-only", meta.ResolvedPolicyGraphExecutionChecksum)
		assert.Equal(t, "filters-only", meta.ResolvedPolicyFiltersChecksum)
		assert.Empty(t, meta.ResolvedPolicyExecutionJobChecksum)
		assert.Empty(t, meta.ResolvedPolicyCollectorJobChecksum)

		rows := metadataRows(t, reader)
		_, present := rows[MetaResolvedPolicyExecutionJobChecksum]
		assert.False(t, present, "an empty execution job checksum is not written")
		_, present = rows[MetaResolvedPolicyCollectorJobChecksum]
		assert.False(t, present, "an empty collector job checksum is not written")
	})

	t.Run("nil resolved policy is a no-op", func(t *testing.T) {
		dbPath := filepath.Join(t.TempDir(), "rp.db")
		store, err := NewSqliteScanDataStore(dbPath, resolvedPolicyTestAssetMrn)
		require.NoError(t, err)

		require.NoError(t, store.WriteResolvedPolicyChecksums(ctx, nil))

		reader := finalizeAndReopen(t, store, dbPath)
		rows := metadataRows(t, reader)
		_, present := rows[MetaResolvedPolicyGraphExecutionChecksum]
		assert.False(t, present)
	})

	t.Run("read-only stores refuse the write", func(t *testing.T) {
		dbPath := filepath.Join(t.TempDir(), "rp.db")
		store, err := NewSqliteScanDataStore(dbPath, resolvedPolicyTestAssetMrn)
		require.NoError(t, err)
		reader := finalizeAndReopen(t, store, dbPath)

		assert.Error(t, reader.WriteResolvedPolicyChecksums(ctx, fullResolvedPolicy("1")))
	})
}

func TestScanDataStoreWrapper_SetResolvedPolicy(t *testing.T) {
	ctx := context.Background()

	t.Run("a different asset mrn is rejected and nothing is written", func(t *testing.T) {
		dbPath := filepath.Join(t.TempDir(), "rp.db")
		store, err := NewSqliteScanDataStore(dbPath, resolvedPolicyTestAssetMrn)
		require.NoError(t, err)
		wrapper := NewScanDataStoreWrapper(store, resolvedPolicyTestAssetMrn)

		err = wrapper.SetResolvedPolicy(ctx, "//assets.api.mondoo.com/spaces/test-space/assets/other", fullResolvedPolicy("1"))
		assert.Error(t, err)

		reader := finalizeAndReopen(t, store, dbPath)
		rows := metadataRows(t, reader)
		_, present := rows[MetaResolvedPolicyGraphExecutionChecksum]
		assert.False(t, present)
	})

	t.Run("nil resolved policy is a no-op", func(t *testing.T) {
		dbPath := filepath.Join(t.TempDir(), "rp.db")
		store, err := NewSqliteScanDataStore(dbPath, resolvedPolicyTestAssetMrn)
		require.NoError(t, err)
		wrapper := NewScanDataStoreWrapper(store, resolvedPolicyTestAssetMrn)

		require.NoError(t, wrapper.SetResolvedPolicy(ctx, resolvedPolicyTestAssetMrn, nil))

		reader := finalizeAndReopen(t, store, dbPath)
		rows := metadataRows(t, reader)
		_, present := rows[MetaResolvedPolicyGraphExecutionChecksum]
		assert.False(t, present)
	})

	t.Run("the matching asset mrn writes the checksums", func(t *testing.T) {
		dbPath := filepath.Join(t.TempDir(), "rp.db")
		store, err := NewSqliteScanDataStore(dbPath, resolvedPolicyTestAssetMrn)
		require.NoError(t, err)
		wrapper := NewScanDataStoreWrapper(store, resolvedPolicyTestAssetMrn)

		require.NoError(t, wrapper.SetResolvedPolicy(ctx, resolvedPolicyTestAssetMrn, fullResolvedPolicy("w")))

		reader := finalizeAndReopen(t, store, dbPath)
		meta, err := reader.GetMetadata()
		require.NoError(t, err)
		assert.Equal(t, "graph-w", meta.ResolvedPolicyGraphExecutionChecksum)
		assert.Equal(t, "filters-w", meta.ResolvedPolicyFiltersChecksum)
		assert.Equal(t, "execution-w", meta.ResolvedPolicyExecutionJobChecksum)
		assert.Equal(t, "collector-w", meta.ResolvedPolicyCollectorJobChecksum)
	})
}
