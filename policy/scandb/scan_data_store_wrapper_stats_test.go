// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scandb

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/v13/policy"
	"go.mondoo.com/mql/v13/llx"
)

func newWrapperForTest(t *testing.T) *ScanDataStoreWrapper {
	t.Helper()
	store, err := NewSqliteScanDataStore(t.TempDir()+"/scan.db", "//assets/1")
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })
	return NewScanDataStoreWrapper(store, "//assets/1")
}

func TestWrapper_CountsExecutedAndErrored(t *testing.T) {
	ctx := context.Background()
	w := newWrapperForTest(t)

	require.NoError(t, w.WriteScore(ctx, "//assets/1", &policy.Score{QrId: "q1", Type: policy.ScoreType_Result}))
	require.NoError(t, w.WriteScore(ctx, "//assets/1", &policy.Score{QrId: "q2", Type: policy.ScoreType_Error}))
	require.NoError(t, w.WriteData(ctx, "//assets/1", &llx.Result{CodeId: "d1"}))
	require.NoError(t, w.WriteData(ctx, "//assets/1", &llx.Result{CodeId: "d2", Error: "boom"}))

	require.Equal(t, int64(4), w.ExecutedCount()) // 2 scores + 2 data
	require.Equal(t, int64(2), w.ErroredCount())  // 1 error score + 1 error data
}
