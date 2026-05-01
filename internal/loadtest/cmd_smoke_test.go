// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package loadtest

import (
	"context"
	"testing"

	_ "github.com/glebarez/go-sqlite"
	"github.com/stretchr/testify/require"
)

// TestEndToEndDryRun exercises the same code path the CLI uses: load templates
// from disk, run with the dry-run client, verify expected call counts. Acts as
// a tripwire if any of the wiring (templates → synth → mutator → runner)
// regresses.
func TestEndToEndDryRun(t *testing.T) {
	dir := t.TempDir()
	writeFixtureDB(t, dir, "fixture-1.db", true)
	writeFixtureDB(t, dir, "fixture-2.db", true)

	templates, err := LoadTemplates(context.Background(), dir)
	require.NoError(t, err)

	stats, err := Run(context.Background(), Config{
		SpaceMrn:      "//captain.api.mondoo.app/spaces/test",
		Templates:     templates,
		Assets:        5,
		ScansPerAsset: 3,
		ChangePct:     25,
		Seed:          42,
		TotalShards:   1,
		Workers:       2,
		Client:        NewDryRunClient(),
	})
	require.NoError(t, err)
	require.EqualValues(t, 15, stats.SyncCalls, "default flow: 5 assets * 3 scans")
	require.EqualValues(t, 15, stats.ResolveCalls, "default flow: 5 assets * 3 scans")
	require.EqualValues(t, 15, stats.UploadCalls, "5 assets * 3 scans")
	require.EqualValues(t, 0, stats.ErrorsSync)
	require.EqualValues(t, 0, stats.ErrorsResolve)
	require.EqualValues(t, 0, stats.ErrorsUpload)
}
