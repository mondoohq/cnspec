// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/mql/discovery"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
	"go.mondoo.com/ranger-rpc/codes"
	"go.mondoo.com/ranger-rpc/status"
)

// claimResolver records ClaimAssets calls and answers with claimed(call, i):
// whether the i-th asset of the call-th request (both 0-based) is CLAIMED.
type claimResolver struct {
	policy.PolicyResolver
	err     error
	claimed func(call, i int) bool
	reqs    []*policy.ClaimAssetsReq
}

func (r *claimResolver) ClaimAssets(_ context.Context, req *policy.ClaimAssetsReq) (*policy.ClaimAssetsResp, error) {
	call := len(r.reqs)
	r.reqs = append(r.reqs, req)
	if r.err != nil {
		return nil, r.err
	}
	resp := &policy.ClaimAssetsResp{}
	for i := range req.List {
		if r.claimed != nil && r.claimed(call, i) {
			resp.Verdicts = append(resp.Verdicts, &policy.ClaimAssetVerdict{
				Index:  int32(i),
				Action: policy.ClaimAssetVerdict_CLAIMED,
			})
		}
	}
	return resp, nil
}

func claimChild(name string) *discovery.TrackedAsset {
	return &discovery.TrackedAsset{Asset: &inventory.Asset{
		Name:   name,
		Claims: []*inventory.AssetClaim{{}},
	}}
}

func claimlessChild(name string) *discovery.TrackedAsset {
	return &discovery.TrackedAsset{Asset: &inventory.Asset{Name: name}}
}

func claimServices(r *claimResolver) *policy.Services {
	return &policy.Services{PolicyResolver: r}
}

func TestClaimAssetsBatchSkipsClaimless(t *testing.T) {
	r := &claimResolver{claimed: func(int, int) bool { return true }}
	withClaims := claimChild("with")
	children := []*discovery.TrackedAsset{claimlessChild("a"), withClaims, claimlessChild("b")}

	claimed, err := claimAssetsBatch(context.Background(), claimServices(r), "//space", children)
	require.NoError(t, err)
	require.Len(t, r.reqs, 1)
	require.Len(t, r.reqs[0].List, 1, "claimless children are never sent")
	assert.Equal(t, "with", r.reqs[0].List[0].Name)
	assert.Equal(t, "//space", r.reqs[0].SpaceMrn)
	assert.Equal(t, map[*discovery.TrackedAsset]struct{}{withClaims: {}}, claimed)

	r = &claimResolver{}
	claimed, err = claimAssetsBatch(context.Background(), claimServices(r), "//space",
		[]*discovery.TrackedAsset{claimlessChild("a"), claimlessChild("b")})
	require.NoError(t, err)
	assert.Empty(t, claimed)
	assert.Empty(t, r.reqs, "no RPC when no child carries a claim")
}

func TestClaimAssetsBatchMapsVerdictsByIndex(t *testing.T) {
	r := &claimResolver{claimed: func(_, i int) bool { return i == 0 || i == 2 }}
	children := []*discovery.TrackedAsset{claimChild("a"), claimChild("b"), claimChild("c")}

	claimed, err := claimAssetsBatch(context.Background(), claimServices(r), "//space", children)
	require.NoError(t, err)
	assert.Equal(t, map[*discovery.TrackedAsset]struct{}{children[0]: {}, children[2]: {}}, claimed)
}

func TestClaimAssetsBatchMissingVerdictMeansScan(t *testing.T) {
	r := &claimResolver{} // answers with no verdicts at all
	children := []*discovery.TrackedAsset{claimChild("a"), claimChild("b")}

	claimed, err := claimAssetsBatch(context.Background(), claimServices(r), "//space", children)
	require.NoError(t, err)
	assert.Empty(t, claimed)
}

func TestClaimAssetsBatchIgnoresOutOfRangeVerdicts(t *testing.T) {
	r := &claimResolver{}
	children := []*discovery.TrackedAsset{claimChild("a")}
	rr := &outOfRangeResolver{claimResolver: r}

	claimed, err := claimAssetsBatch(context.Background(), &policy.Services{PolicyResolver: rr}, "//space", children)
	require.NoError(t, err)
	assert.Empty(t, claimed)
}

type outOfRangeResolver struct{ *claimResolver }

func (r *outOfRangeResolver) ClaimAssets(context.Context, *policy.ClaimAssetsReq) (*policy.ClaimAssetsResp, error) {
	return &policy.ClaimAssetsResp{Verdicts: []*policy.ClaimAssetVerdict{
		{Index: -1, Action: policy.ClaimAssetVerdict_CLAIMED},
		{Index: 5, Action: policy.ClaimAssetVerdict_CLAIMED},
	}}, nil
}

func TestClaimAssetsBatchNotFoundReturnsError(t *testing.T) {
	// An old server answers an unknown method with NotFound.
	r := &claimResolver{err: status.Error(codes.NotFound, "method not found")}

	claimed, err := claimAssetsBatch(context.Background(), claimServices(r), "//space",
		[]*discovery.TrackedAsset{claimChild("a")})
	require.Error(t, err)
	assert.Nil(t, claimed)
	assert.Len(t, r.reqs, 1, "not retried")
}

func TestClaimAssetsBatchErrorInLaterChunkDiscardsEarlierVerdicts(t *testing.T) {
	r := &claimResolver{claimed: func(int, int) bool { return true }}
	children := make([]*discovery.TrackedAsset, claimBatchSize+1)
	for i := range children {
		children[i] = claimChild(fmt.Sprintf("c%d", i))
	}
	fail := &failSecondResolver{claimResolver: r}

	claimed, err := claimAssetsBatch(context.Background(), &policy.Services{PolicyResolver: fail}, "//space", children)
	require.Error(t, err)
	assert.Nil(t, claimed, "any error returns no partial result")
}

type failSecondResolver struct{ *claimResolver }

func (r *failSecondResolver) ClaimAssets(ctx context.Context, req *policy.ClaimAssetsReq) (*policy.ClaimAssetsResp, error) {
	if len(r.reqs) == 1 {
		r.reqs = append(r.reqs, req)
		return nil, errors.New("boom")
	}
	return r.claimResolver.ClaimAssets(ctx, req)
}

func TestClaimAssetsBatchChunks(t *testing.T) {
	n := claimBatchSize + 1
	children := make([]*discovery.TrackedAsset, n)
	for i := range children {
		children[i] = claimChild(fmt.Sprintf("c%d", i))
	}
	// The verdict sits at index 0 of the second chunk, which is child 100
	// overall: the index restarts per call.
	r := &claimResolver{claimed: func(call, i int) bool { return call == 1 && i == 0 }}

	claimed, err := claimAssetsBatch(context.Background(), claimServices(r), "//space", children)
	require.NoError(t, err)
	require.Len(t, r.reqs, 2)
	assert.Len(t, r.reqs[0].List, claimBatchSize)
	assert.Len(t, r.reqs[1].List, 1)
	assert.Equal(t, map[*discovery.TrackedAsset]struct{}{children[claimBatchSize]: {}}, claimed)
}

func TestClaimLaneStateDisablesOnce(t *testing.T) {
	var buf bytes.Buffer
	prev := log.Logger
	log.Logger = zerolog.New(&buf)
	t.Cleanup(func() { log.Logger = prev })

	c := &claimLaneState{enabled: true}
	assert.True(t, c.active())

	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.disable("test", errors.New("boom"))
		}()
	}
	wg.Wait()
	c.disable("again", errors.New("boom"))

	assert.False(t, c.active())
	assert.Equal(t, 1, strings.Count(buf.String(), "\n"), "logged exactly once: %q", buf.String())

	off := &claimLaneState{}
	assert.False(t, off.active(), "never enabled")
}
