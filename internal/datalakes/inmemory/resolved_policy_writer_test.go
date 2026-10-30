// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package inmemory

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/mql/llx"
)

// recordingWriter is a DataStore that only remembers SetResolvedPolicy calls;
// everything else is a no-op, enough for an empty collector job.
type recordingWriter struct {
	calls []recordedResolvedPolicy
	err   error
}

type recordedResolvedPolicy struct {
	assetMrn string
	rp       *policy.ResolvedPolicy
}

func (r *recordingWriter) WriteScore(context.Context, string, *policy.Score) error { return nil }
func (r *recordingWriter) GetScore(context.Context, string, string) (*policy.Score, error) {
	return nil, nil
}
func (r *recordingWriter) WriteData(context.Context, string, *llx.Result) error { return nil }
func (r *recordingWriter) GetData(context.Context, string, string) (*llx.Result, error) {
	return nil, nil
}
func (r *recordingWriter) WriteRisk(context.Context, string, *policy.ScoredRiskFactor) error {
	return nil
}
func (r *recordingWriter) GetRisk(context.Context, string, string) (*policy.ScoredRiskFactor, error) {
	return nil, nil
}
func (r *recordingWriter) StreamRisks(context.Context, string, func(*policy.ScoredRiskFactor) error) error {
	return nil
}
func (r *recordingWriter) WriteResource(context.Context, string, *llx.ResourceRecording) error {
	return nil
}
func (r *recordingWriter) SetAssetFilters(context.Context, string, *policy.Mqueries) error {
	return nil
}
func (r *recordingWriter) SetResolvedPolicy(_ context.Context, assetMrn string, rp *policy.ResolvedPolicy) error {
	r.calls = append(r.calls, recordedResolvedPolicy{assetMrn: assetMrn, rp: rp})
	return r.err
}

func newDbWithWriter(t *testing.T, writer DataStore, assetMrn string) *Db {
	t.Helper()
	db, _, err := NewServices(nil, WithDataWriter(writer))
	require.NoError(t, err)
	// The asset the resolved policy is set on; seeded directly so the test
	// does not depend on policy preparation.
	require.True(t, db.cache.Set(dbIDAsset+assetMrn, wrapAsset{mrn: assetMrn}, 1))
	return db
}

func TestDb_SetAssetResolvedPolicy_ForwardsToWriter(t *testing.T) {
	ctx := context.Background()
	assetMrn := "//assets.api.mondoo.com/spaces/test-space/assets/forwarded"

	t.Run("the writer sees the resolved policy once per distinct checksum", func(t *testing.T) {
		writer := &recordingWriter{}
		db := newDbWithWriter(t, writer, assetMrn)

		rp := &policy.ResolvedPolicy{
			GraphExecutionChecksum: "graph-1",
			CollectorJob:           &policy.CollectorJob{},
		}
		require.NoError(t, db.SetAssetResolvedPolicy(ctx, assetMrn, rp, policy.V2Code))
		require.Len(t, writer.calls, 1)
		assert.Equal(t, assetMrn, writer.calls[0].assetMrn)
		assert.Same(t, rp, writer.calls[0].rp)

		// Same checksum again: the cached copy is kept and nothing is re-sent.
		require.NoError(t, db.SetAssetResolvedPolicy(ctx, assetMrn, rp, policy.V2Code))
		assert.Len(t, writer.calls, 1)

		// A different resolved policy is forwarded again.
		rp2 := &policy.ResolvedPolicy{
			GraphExecutionChecksum: "graph-2",
			CollectorJob:           &policy.CollectorJob{},
		}
		require.NoError(t, db.SetAssetResolvedPolicy(ctx, assetMrn, rp2, policy.V2Code))
		require.Len(t, writer.calls, 2)
		assert.Same(t, rp2, writer.calls[1].rp)
	})

	t.Run("a failing writer does not fail the resolve", func(t *testing.T) {
		writer := &recordingWriter{err: errors.New("disk full")}
		db := newDbWithWriter(t, writer, assetMrn)

		rp := &policy.ResolvedPolicy{
			GraphExecutionChecksum: "graph-1",
			CollectorJob:           &policy.CollectorJob{},
		}
		require.NoError(t, db.SetAssetResolvedPolicy(ctx, assetMrn, rp, policy.V2Code))
		assert.Len(t, writer.calls, 1)
	})
}
