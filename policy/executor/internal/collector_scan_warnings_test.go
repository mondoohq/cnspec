// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package internal

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/policy"
)

// fakeStoreResultsResolver implements policy.PolicyResolver, recording every
// StoreResultsReq it receives. Every other method is left to the embedded
// nil PolicyResolver (same pattern as policy.NoStoreResults) -- the tests
// here only exercise the Sink -> StoreResults path, so calling anything else
// would be a test bug, not a case to support.
type fakeStoreResultsResolver struct {
	policy.PolicyResolver

	mu   sync.Mutex
	reqs []*policy.StoreResultsReq
}

func (f *fakeStoreResultsResolver) StoreResults(_ context.Context, req *policy.StoreResultsReq) (*policy.Empty, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reqs = append(f.reqs, req)
	return &policy.Empty{}, nil
}

func (f *fakeStoreResultsResolver) allReqs() []*policy.StoreResultsReq {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]*policy.StoreResultsReq(nil), f.reqs...)
}

// TestBufferedCollector_FlushAndStop_AttachesScanWarningsToFinalBatch covers
// the normal case: an asset that produced scores also had a provider crash
// recorded against it. The warnings must land on the SAME StoreResultsReq
// that carries IsLastBatch=true, not a separate message.
func TestBufferedCollector_FlushAndStop_AttachesScanWarningsToFinalBatch(t *testing.T) {
	resolver := &fakeStoreResultsResolver{}
	psc := NewPolicyServiceCollector("//test/assets/1", resolver)
	bc := NewBufferedCollector(context.Background(), psc)

	bc.SinkScore([]*policy.Score{{QrId: "q1", Value: 100, ScoreCompletion: 100}})
	bc.FlushAndStop([]string{"the 'os' provider crashed: connection refused"})

	reqs := resolver.allReqs()
	require.NotEmpty(t, reqs, "expected at least one StoreResultsReq")

	last := reqs[len(reqs)-1]
	assert.True(t, last.IsLastBatch, "the batch carrying ScanWarnings must be the last batch")
	assert.Equal(t, []string{"the 'os' provider crashed: connection refused"}, last.ScanWarnings)

	// Only the final batch carries warnings; earlier batches (none here,
	// since everything was buffered into one flush) must never see them.
	for _, req := range reqs[:len(reqs)-1] {
		assert.Empty(t, req.ScanWarnings, "only the last batch should carry ScanWarnings")
	}
}

// TestBufferedCollector_FlushAndStop_SendsWarningsEvenWithAnEmptyFinalBatch
// covers the edge case Sink's early-return guard used to hide: a crash with
// no trailing scores/risks. Before this change, Sink would receive
// isDone=true with empty results/scores/risks and send nothing at all,
// silently dropping the warning.
func TestBufferedCollector_FlushAndStop_SendsWarningsEvenWithAnEmptyFinalBatch(t *testing.T) {
	resolver := &fakeStoreResultsResolver{}
	psc := NewPolicyServiceCollector("//test/assets/1", resolver)
	bc := NewBufferedCollector(context.Background(), psc)

	// No SinkData/SinkScore calls at all.
	bc.FlushAndStop([]string{"the 'aws' provider crashed: EOF"})

	reqs := resolver.allReqs()
	require.Len(t, reqs, 1, "an otherwise-empty final batch must still be sent when there are scan warnings")
	assert.True(t, reqs[0].IsLastBatch)
	assert.Equal(t, []string{"the 'aws' provider crashed: EOF"}, reqs[0].ScanWarnings)
	assert.Empty(t, reqs[0].Scores)
	assert.Empty(t, reqs[0].Risks)
}

// TestBufferedCollector_FlushAndStop_NoWarningsNoEmptyBatch is the
// regression guard for the change above: an asset with nothing to report at
// all (no scores, no risks, no warnings) must still send nothing, exactly
// as before.
func TestBufferedCollector_FlushAndStop_NoWarningsNoEmptyBatch(t *testing.T) {
	resolver := &fakeStoreResultsResolver{}
	psc := NewPolicyServiceCollector("//test/assets/1", resolver)
	bc := NewBufferedCollector(context.Background(), psc)

	bc.FlushAndStop(nil)

	assert.Empty(t, resolver.allReqs(), "nothing to report must still send nothing")
}
