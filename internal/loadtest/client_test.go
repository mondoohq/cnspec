// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package loadtest

import (
	"context"
	"os"
	"testing"

	_ "github.com/glebarez/go-sqlite"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/v13/policy"
	"go.mondoo.com/cnspec/v13/policy/scandb"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
)

// TestWriteScanDBRoundtrip verifies the temp scan db produced by the upload
// path is a valid SQLite scan database that the existing scandb reader can
// open and stream. If the server can't read what we write, the load test
// produces no useful load.
func TestWriteScanDBRoundtrip(t *testing.T) {
	dir := t.TempDir()
	assetMrn := "//assets.api.mondoo.com/spaces/x/assets/lt-1"

	payload := &ScanPayload{
		Asset: &inventory.Asset{
			Mrn:         assetMrn,
			Name:        "loadtest-1",
			PlatformIds: []string{"//platformid.api.mondoo.app/loadtest/abc"},
			Platform:    &inventory.Platform{Name: "ubuntu", Version: "22.04"},
		},
		Scores: []*policy.Score{
			{QrId: "q1", Value: 100, Type: 1, Weight: 1},
			{QrId: "q2", Value: 0, Type: 1, Weight: 1},
		},
		Data: map[string]*llx.Result{
			"c1": {CodeId: "c1", Data: llx.BoolPrimitive(true)},
		},
		Risks: []*policy.ScoredRiskFactor{
			{Mrn: "risk-a", Risk: 0.5, IsDetected: true},
		},
	}

	path, err := writeScanDB(context.Background(), dir, assetMrn, payload)
	require.NoError(t, err)
	defer os.Remove(path)

	store, err := scandb.NewSqliteScanDataStoreReader(path)
	require.NoError(t, err)
	defer store.Close()

	gotAsset, err := store.GetAsset(context.Background())
	require.NoError(t, err)
	require.Equal(t, payload.Asset.Mrn, gotAsset.Mrn)
	require.Equal(t, "ubuntu", gotAsset.Platform.Name)

	var scores []*policy.Score
	require.NoError(t, store.StreamScores(context.Background(), func(s *policy.Score) error {
		scores = append(scores, s)
		return nil
	}))
	require.Len(t, scores, 2)

	gotData, err := store.GetData(context.Background(), "c1")
	require.NoError(t, err)
	require.NotNil(t, gotData)

	gotRisk, err := store.GetRisk(context.Background(), "risk-a")
	require.NoError(t, err)
	require.True(t, gotRisk.IsDetected)
}
