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
	"go.mondoo.com/cnspec/cli/progress"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/mql/discovery"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/providers-sdk/v1/vault"
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

// filteredCounter is a MultiProgress that records Filtered and AddTask calls.
type filteredCounter struct {
	progress.NoopMultiProgress
	filtered int
	added    []string
}

func (f *filteredCounter) AddTask(index string, _ *inventory.Asset) { f.added = append(f.added, index) }

func (f *filteredCounter) Filtered(n int) { f.filtered += n }

func TestClaimAssetsEnabledGetterAndLaneState(t *testing.T) {
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

	t.Run("env variable name", func(t *testing.T) {
		assert.Equal(t, "MONDOO_CLAIM_ASSETS", claimAssetsEnvVar)
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

func TestClaimPrePassIsIdempotentForRoots(t *testing.T) {
	r := &claimResolver{claimed: func(_, i int) bool { return i == 0 }}
	sc := &scanContext{
		claimLane:     newClaimLaneState(&policy.ScanParameters{ClaimAssetsEnabled: true}, claimServices(r), ""),
		services:      claimServices(r),
		spaceMrn:      "//space",
		multiprogress: &filteredCounter{},
	}
	root := &discovery.TrackedAsset{Asset: &inventory.Asset{}, Children: []*discovery.TrackedAsset{claimChild("a"), claimChild("b")}}

	// distributeJob's path: registerRootTasks, then scanSubtree's own call.
	sc.registerRootTasks(context.Background(), root)
	sc.claimSubtreeChildren(context.Background(), root)
	assert.Len(t, r.reqs, 1, "scanSubtree's pre-pass on a registered root makes no RPC")

	// A branch node reached by recursion is processed but not retained.
	branch := &discovery.TrackedAsset{Asset: &inventory.Asset{}, Children: []*discovery.TrackedAsset{claimChild("c")}}
	sc.claimSubtreeChildren(context.Background(), branch)
	assert.Len(t, r.reqs, 2)
	assert.Len(t, sc.claimedSubtrees, 1, "only the root is in the visited set")
	assert.Contains(t, sc.claimedSubtrees, root)
}

// A claimed root child must not be registered as a pending progress task:
// distributeJob runs the pre-pass before its AddTask loop and the loop skips
// nil-asset children. This drives the same sequence the loop runs.
func TestClaimedRootChildIsNotRegisteredAsTask(t *testing.T) {
	r := &claimResolver{claimed: func(_, i int) bool { return i == 0 }}
	mp := &filteredCounter{}
	sc := &scanContext{
		claimLane:     newClaimLaneState(&policy.ScanParameters{ClaimAssetsEnabled: true}, claimServices(r), ""),
		services:      claimServices(r),
		spaceMrn:      "//space",
		multiprogress: mp,
	}
	claimedKid, keptKid := claimChild("claimed"), claimChild("kept")
	claimedKid.Asset.PlatformIds = []string{"pid-claimed"}
	keptKid.Asset.PlatformIds = []string{"pid-kept"}
	root := &discovery.TrackedAsset{Asset: &inventory.Asset{PlatformIds: []string{"pid-root"}}, Children: []*discovery.TrackedAsset{claimedKid, keptKid}}

	sc.registerRootTasks(context.Background(), root)

	assert.Equal(t, []string{"pid-root", "pid-kept"}, mp.added, "claimed child is never a pending task")
	assert.Equal(t, 1, mp.filtered)
}

func TestClaimPrePassCancelledContextKeepsLane(t *testing.T) {
	var buf bytes.Buffer
	prev := log.Logger
	log.Logger = zerolog.New(&buf)
	t.Cleanup(func() { log.Logger = prev })

	r := &claimResolver{claimed: func(int, int) bool { return true }}
	mp := &filteredCounter{}
	sc := &scanContext{
		claimLane:     newClaimLaneState(&policy.ScanParameters{ClaimAssetsEnabled: true}, claimServices(r), ""),
		services:      claimServices(r),
		spaceMrn:      "//space",
		multiprogress: mp,
	}
	node := &discovery.TrackedAsset{Children: []*discovery.TrackedAsset{claimChild("a")}}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	// The fake ignores ctx, so make it fail the way a cancelled call does.
	r.err = ctx.Err()
	sc.claimSubtreeChildren(ctx, node)

	assert.True(t, sc.claimLane.active(), "cancellation must not disable the lane")
	assert.Empty(t, buf.String(), "nothing logged")
	assert.NotNil(t, node.Children[0].Asset, "nothing skipped")
	assert.Zero(t, mp.filtered)
}

func TestClaimProjectionIsMinimalAndInheritsFromRoot(t *testing.T) {
	root := &discovery.TrackedAsset{Asset: &inventory.Asset{
		Name:      "registry",
		ManagedBy: "mondoo-test-root",
		TraceId:   "trace-root",
		Labels: map[string]string{
			"mondoo.com/exec-environment": "ci",
			"team":                        "not-a-mondoo-label",
		},
		Annotations: map[string]string{"owner": "root", "shared": "root"},
	}}
	child := &discovery.TrackedAsset{Parent: root, Asset: &inventory.Asset{
		Id:          "id-1",
		Name:        "image",
		PlatformIds: []string{"pid-1", "pid-2"},
		Claims:      []*inventory.AssetClaim{{}},
		Labels:      map[string]string{"own": "label"},
		Annotations: map[string]string{"shared": "child"},
		Category:    inventory.AssetCategory_CATEGORY_CICD,
		Connections: []*inventory.Config{{Type: "container-registry", Credentials: []*vault.Credential{{User: "u"}}}},
		Options:     map[string]string{"opt": "x"},
		Url:         "https://example.com",
		Platform:    &inventory.Platform{Name: "ubuntu"},
	}}
	// A grandchild still reaches the root by walking Parent.
	grandchild := &discovery.TrackedAsset{Parent: child, Asset: &inventory.Asset{Name: "layer", Claims: []*inventory.AssetClaim{{}}}}

	r := &claimResolver{}
	_, err := claimAssetsBatch(context.Background(), claimServices(r), "//space", []*discovery.TrackedAsset{child, grandchild})
	require.NoError(t, err)
	require.Len(t, r.reqs, 1)
	got := r.reqs[0].List[0]

	assert.Nil(t, got.Connections, "connections (and their credentials) are never sent")
	assert.Nil(t, got.Options)
	assert.Nil(t, got.Platform)
	assert.Empty(t, got.Url)
	assert.NotNil(t, child.Asset.Connections, "the original asset is untouched")

	assert.Equal(t, "id-1", got.Id)
	assert.Equal(t, "image", got.Name)
	assert.Equal(t, []string{"pid-1", "pid-2"}, got.PlatformIds)
	assert.Len(t, got.Claims, 1)
	assert.Equal(t, inventory.AssetCategory_CATEGORY_CICD, got.Category)

	assert.Equal(t, "mondoo-test-root", got.ManagedBy, "inherited from the root")
	assert.Equal(t, "trace-root", got.TraceId)
	assert.Equal(t, map[string]string{"own": "label", "mondoo.com/exec-environment": "ci"}, got.Labels,
		"own labels plus the root's mondoo labels only")
	assert.Equal(t, map[string]string{"owner": "root", "shared": "child"}, got.Annotations,
		"root annotations fill in, the child's own win")
	assert.Equal(t, map[string]string{"own": "label"}, child.Asset.Labels, "the original labels are not mutated")

	g := r.reqs[0].List[1]
	assert.Equal(t, "mondoo-test-root", g.ManagedBy, "walks Parent up to the root")

	t.Run("own values win and a parentless child inherits nothing", func(t *testing.T) {
		own := &discovery.TrackedAsset{Parent: root, Asset: &inventory.Asset{Name: "x", ManagedBy: "mine", TraceId: "t", Claims: []*inventory.AssetClaim{{}}}}
		orphan := claimChild("orphan")
		r := &claimResolver{}
		_, err := claimAssetsBatch(context.Background(), claimServices(r), "//space", []*discovery.TrackedAsset{own, orphan})
		require.NoError(t, err)
		assert.Equal(t, "mine", r.reqs[0].List[0].ManagedBy)
		assert.Equal(t, "t", r.reqs[0].List[0].TraceId)
		assert.Empty(t, r.reqs[0].List[1].ManagedBy)
	})
}

func TestClaimAssetsBatchLogsVerdicts(t *testing.T) {
	var buf bytes.Buffer
	prev := log.Logger
	log.Logger = zerolog.New(&buf)
	t.Cleanup(func() { log.Logger = prev })

	kid := claimChild("claimed-one")
	kid.Asset.PlatformIds = []string{"pid-x"}
	r := &claimResolver{claimed: func(_, i int) bool { return i == 0 }}
	_, err := claimAssetsBatch(context.Background(), claimServices(r), "//space/s1",
		[]*discovery.TrackedAsset{kid, claimChild("kept"), claimlessChild("no-claims")})
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, `"level":"info"`)
	assert.Contains(t, out, `"asked":2`)
	assert.Contains(t, out, `"claimed":1`)
	assert.Contains(t, out, `"space":"//space/s1"`)
	assert.Contains(t, out, `"level":"debug"`)
	assert.Contains(t, out, `"name":"claimed-one"`)
	assert.Contains(t, out, `"platformId":"pid-x"`)

	buf.Reset()
	_, err = claimAssetsBatch(context.Background(), claimServices(r), "//space/s1", []*discovery.TrackedAsset{claimlessChild("a")})
	require.NoError(t, err)
	assert.Empty(t, buf.String(), "nothing asked, nothing logged")
}

// An all-claimed subtree: every child is claimed, so the loop never touches
// the explorer (a zero explorer would reject Connect), and the claim count
// is what keeps the run from failing as "nothing to connect to".
func TestScanSubtreeAllClaimed(t *testing.T) {
	r := &claimResolver{claimed: func(int, int) bool { return true }}
	mp := &filteredCounter{}
	connSem := make(chan struct{}, 2)
	dispatcher := &scanDispatcher{}
	sc := &scanContext{
		explorer:      &discovery.AssetExplorer{},
		reporter:      NewAggregateReporter(),
		multiprogress: mp,
		connSem:       connSem,
		batcher:       newSyncBatcher(dispatcher, nil, "", nil, mp, 0),
		dispatcher:    dispatcher,
		claimLane:     newClaimLaneState(&policy.ScanParameters{ClaimAssetsEnabled: true}, claimServices(r), ""),
		services:      claimServices(r),
		spaceMrn:      "//space",
	}
	// No platform IDs on the node: it is filtered, not dispatched.
	root := &discovery.TrackedAsset{Asset: &inventory.Asset{Name: "registry"}}
	root.Children = []*discovery.TrackedAsset{claimChild("a"), claimChild("b"), claimChild("c")}

	require.NoError(t, sc.scanSubtree(context.Background(), root))

	assert.Equal(t, 3, sc.claimedCount)
	assert.Empty(t, connSem, "no connection slot was taken")
	assert.Nil(t, root.Children, "subtree cleanup still runs")
	assert.NoError(t, sc.errNothingScanned(), "an all-claimed run is a success")
}

func TestErrNothingScanned(t *testing.T) {
	assert.Error(t, (&scanContext{}).errNothingScanned(), "nothing scanned, nothing claimed")
	assert.NoError(t, (&scanContext{claimedCount: 1}).errNothingScanned())
	assert.NoError(t, (&scanContext{platformIdsExclude: map[string]struct{}{"x": {}}}).errNothingScanned())
}
