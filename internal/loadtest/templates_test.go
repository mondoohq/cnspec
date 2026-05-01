// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package loadtest

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/cockroachdb/errors"
	_ "github.com/glebarez/go-sqlite"
	"github.com/stretchr/testify/require"

	"go.mondoo.com/cnspec/v13/policy"
	"go.mondoo.com/cnspec/v13/policy/scandb"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
)

func writeFixtureDB(t *testing.T, dir, name string, withAsset bool) string {
	t.Helper()
	path := filepath.Join(dir, name)
	asset := &inventory.Asset{
		Mrn:         "//assets.api.mondoo.com/spaces/x/assets/" + name,
		Name:        name,
		PlatformIds: []string{"//platformid.api.mondoo.app/runtime/test/" + name},
		Platform:    &inventory.Platform{Name: "ubuntu", Version: "22.04"},
	}
	store, err := scandb.NewSqliteScanDataStore(path, asset.Mrn)
	require.NoError(t, err)
	if withAsset {
		require.NoError(t, store.WriteAsset(context.Background(), asset))
		require.NoError(t, store.WriteAssetFilters(context.Background(), &policy.Mqueries{Items: []*policy.Mquery{
			{CodeId: "filter-code-1", Mql: "asset.platform == \"ubuntu\""},
			{CodeId: "filter-code-2", Mql: "true"},
		}}))
	}
	require.NoError(t, store.WriteScores(context.Background(), []*policy.Score{
		{QrId: "q1", Value: 100, Type: 1, Weight: 1},
		{QrId: "q2", Value: 0, Type: 1, Weight: 1},
	}))
	require.NoError(t, store.WriteData(context.Background(), []*llx.Result{
		{CodeId: "c1", Data: llx.BoolPrimitive(true)},
	}))
	_, err = store.Finalize()
	require.NoError(t, err)
	require.NoError(t, store.Close())
	return path
}

func TestLoadTemplatesRoundtrip(t *testing.T) {
	dir := t.TempDir()
	writeFixtureDB(t, dir, "a.db", true)
	writeFixtureDB(t, dir, "b.db", true)

	templates, err := LoadTemplates(context.Background(), dir)
	require.NoError(t, err)
	require.Len(t, templates, 2)

	for _, tpl := range templates {
		require.NotNil(t, tpl.Asset)
		require.Equal(t, "ubuntu", tpl.Asset.Platform.Name)
		require.Len(t, tpl.Scores, 2)
		require.Contains(t, tpl.Data, "c1")
		require.NotNil(t, tpl.Filters)
		require.Len(t, tpl.Filters.Items, 2)
		require.Equal(t, "asset.platform == \"ubuntu\"", tpl.Filters.Items[0].Mql, "MQL must round-trip — server compiles it")
	}
}

func TestLoadTemplatesRejectsMissingFilters(t *testing.T) {
	dir := t.TempDir()
	// Write a fixture with the asset but no filters by calling WriteAsset
	// directly without WriteAssetFilters. (The default writeFixtureDB writes
	// both, so this case needs a manual setup.)
	path := filepath.Join(dir, "noflt.db")
	asset := &inventory.Asset{
		Mrn:         "//assets.api.mondoo.com/spaces/x/assets/noflt",
		Name:        "noflt",
		PlatformIds: []string{"//platformid.api.mondoo.app/runtime/test/noflt"},
		Platform:    &inventory.Platform{Name: "ubuntu"},
	}
	store, err := scandb.NewSqliteScanDataStore(path, asset.Mrn)
	require.NoError(t, err)
	require.NoError(t, store.WriteAsset(context.Background(), asset))
	require.NoError(t, store.WriteScores(context.Background(), []*policy.Score{{QrId: "q1", Value: 100, Type: 1, Weight: 1}}))
	_, err = store.Finalize()
	require.NoError(t, err)
	require.NoError(t, store.Close())

	_, err = LoadTemplates(context.Background(), dir)
	require.Error(t, err, "loadtest must refuse scan dbs without captured filters")
}

func TestLoadTemplatesRejectsMissingAsset(t *testing.T) {
	dir := t.TempDir()
	writeFixtureDB(t, dir, "old.db", false)

	_, err := LoadTemplates(context.Background(), dir)
	require.Error(t, err, "loadtest must refuse scan dbs without an embedded asset")
}

func TestLoadTemplatesEmptyDir(t *testing.T) {
	dir := t.TempDir()
	_, err := LoadTemplates(context.Background(), dir)
	require.Error(t, err)
}

func TestGetAssetMissing(t *testing.T) {
	// Verifies the scandb GetAsset returns ErrAssetNotFound (not a raw SQL
	// error) when no asset has been written.
	dir := t.TempDir()
	path := writeFixtureDB(t, dir, "noasset.db", false)
	store, err := scandb.NewSqliteScanDataStoreReader(path)
	require.NoError(t, err)
	defer store.Close()

	_, err = store.GetAsset(context.Background())
	require.True(t, errors.Is(err, policy.ErrAssetNotFound), "expected ErrAssetNotFound, got: %v", err)
}
