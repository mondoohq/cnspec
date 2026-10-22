// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/cli/progress"
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

// filteredCounter is a MultiProgress that only counts Filtered calls.
type filteredCounter struct {
	progress.NoopMultiProgress
	filtered int
}

func (f *filteredCounter) Filtered(n int) { f.filtered += n }

func TestScanParametersClaimAssetsEnabled(t *testing.T) {
	var nilParams *policy.ScanParameters
	assert.False(t, nilParams.GetClaimAssetsEnabled(), "a nil response (failed call, incognito) must read as off")
	assert.False(t, (&policy.ScanParameters{}).GetClaimAssetsEnabled(), "a server that says nothing means off")
	assert.True(t, (&policy.ScanParameters{ClaimAssetsEnabled: true}).GetClaimAssetsEnabled())

	r := &claimResolver{}
	services := claimServices(r)
	assert.False(t, newClaimLaneState(nil, services, "").active(), "no parameters: off")
	assert.False(t, newClaimLaneState(&policy.ScanParameters{}, services, "").active(), "bit unset: off")
	assert.True(t, newClaimLaneState(&policy.ScanParameters{ClaimAssetsEnabled: true}, services, "").active(), "bit set: on")
}

func TestClaimLaneEnvOverride(t *testing.T) {
	services := claimServices(&claimResolver{})
	on := &policy.ScanParameters{ClaimAssetsEnabled: true}
	off := &policy.ScanParameters{}

	tests := []struct {
		name   string
		env    string
		params *policy.ScanParameters
		want   bool
	}{
		{"force-on with the bit off", "force", off, true},
		{"1 with the bit off", "1", off, true},
		{"force-on with no parameters", "1", nil, true},
		{"force-off with the bit on", "0", on, false},
		{"unset follows the bit: on", "", on, true},
		{"unset follows the bit: off", "", off, false},
		{"unrecognized value follows the bit", "yes", on, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, newClaimLaneState(tc.params, services, tc.env).active())
		})
	}

	t.Run("distributeJob reads the variable", func(t *testing.T) {
		t.Setenv(claimAssetsEnvVar, "force")
		assert.True(t, newClaimLaneState(off, services, os.Getenv(claimAssetsEnvVar)).active())
		t.Setenv(claimAssetsEnvVar, "0")
		assert.False(t, newClaimLaneState(on, services, os.Getenv(claimAssetsEnvVar)).active())
	})
}

func TestClaimLaneIncognitoOff(t *testing.T) {
	// Incognito runs have no services. Even forced on, the lane never
	// becomes active, so the pre-pass can never reach a nil services.
	for _, env := range []string{"", "1", "force"} {
		lane := newClaimLaneState(&policy.ScanParameters{ClaimAssetsEnabled: true}, nil, env)
		assert.False(t, lane.active(), "env %q", env)

		sc := &scanContext{claimLane: lane, multiprogress: &filteredCounter{}}
		child := claimChild("a")
		require.NotPanics(t, func() {
			sc.claimSubtreeChildren(context.Background(), &discovery.TrackedAsset{
				Children: []*discovery.TrackedAsset{child},
			})
		})
		assert.NotNil(t, child.Asset, "nothing skipped")
	}
}

func TestSkipUnconnectedAsset(t *testing.T) {
	mp := &filteredCounter{}
	// explorer is nil on purpose: skipUnconnectedAsset calling CloseAsset
	// would panic here (and CloseAsset errors on an unconnected asset).
	sc := &scanContext{multiprogress: mp, connSem: make(chan struct{}, 1)}

	// A slot held by another asset must not be released by the skip. This
	// runs first so a stray receive fails the assert instead of blocking.
	sc.connSem <- struct{}{}
	child := claimChild("claimed")
	require.NotPanics(t, func() { sc.skipUnconnectedAsset(child) })
	assert.Equal(t, 1, mp.filtered)
	assert.Nil(t, child.Asset)
	require.Len(t, sc.connSem, 1, "skipUnconnectedAsset must not release connSem")
	<-sc.connSem

	// Nor did it take one.
	sc.skipUnconnectedAsset(claimChild("another"))
	assert.Equal(t, 2, mp.filtered)
	select {
	case sc.connSem <- struct{}{}:
	default:
		t.Fatal("connSem slot was not free after skipUnconnectedAsset")
	}
}

func TestClaimPrePassSkipsClaimedAndDisablesOnError(t *testing.T) {
	newNode := func() (*discovery.TrackedAsset, []*discovery.TrackedAsset) {
		kids := []*discovery.TrackedAsset{claimChild("a"), claimChild("b"), claimlessChild("c")}
		return &discovery.TrackedAsset{Children: kids}, kids
	}

	t.Run("claimed children are skipped", func(t *testing.T) {
		r := &claimResolver{claimed: func(_, i int) bool { return i == 0 }}
		mp := &filteredCounter{}
		sc := &scanContext{
			claimLane:     newClaimLaneState(&policy.ScanParameters{ClaimAssetsEnabled: true}, claimServices(r), ""),
			services:      claimServices(r),
			spaceMrn:      "//space",
			multiprogress: mp,
			connSem:       make(chan struct{}, 1),
		}
		node, kids := newNode()

		sc.claimSubtreeChildren(context.Background(), node)

		assert.Nil(t, kids[0].Asset, "claimed child skipped")
		assert.NotNil(t, kids[1].Asset, "scan verdict child kept")
		assert.NotNil(t, kids[2].Asset, "claimless child kept")
		assert.Equal(t, 1, mp.filtered)
		require.Len(t, r.reqs, 1)
		assert.Equal(t, "//space", r.reqs[0].SpaceMrn)
		assert.True(t, sc.claimLane.active())
	})

	t.Run("an error skips nothing and disables the lane", func(t *testing.T) {
		r := &claimResolver{err: errors.New("boom")}
		mp := &filteredCounter{}
		sc := &scanContext{
			claimLane:     newClaimLaneState(&policy.ScanParameters{ClaimAssetsEnabled: true}, claimServices(r), ""),
			services:      claimServices(r),
			spaceMrn:      "//space",
			multiprogress: mp,
			connSem:       make(chan struct{}, 1),
		}
		node, kids := newNode()

		sc.claimSubtreeChildren(context.Background(), node)
		for _, k := range kids {
			assert.NotNil(t, k.Asset, "nothing skipped on error")
		}
		assert.Zero(t, mp.filtered)
		assert.False(t, sc.claimLane.active())
		require.Len(t, r.reqs, 1)

		node2, _ := newNode()
		sc.claimSubtreeChildren(context.Background(), node2)
		assert.Len(t, r.reqs, 1, "a disabled lane makes no further RPC")
	})

	t.Run("a nil lane is inert", func(t *testing.T) {
		sc := &scanContext{multiprogress: &filteredCounter{}}
		node, kids := newNode()
		require.NotPanics(t, func() { sc.claimSubtreeChildren(context.Background(), node) })
		assert.NotNil(t, kids[0].Asset)
	})
}
