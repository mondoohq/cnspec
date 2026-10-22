// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"context"
	"sync/atomic"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/mql/discovery"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
)

// claimBatchSize is how many claim-bearing children one ClaimAssets call asks
// about.
const claimBatchSize = 100

// claimAssetsBatch asks the server which claim-bearing children it can
// materialize from claim records. Children without claims are never
// sent. Chunked at claimBatchSize per call; verdicts map back by index within
// each chunk. A missing verdict means SCAN. Any error returns (nil, err) —
// the caller fails open and disables the lane for the run.
//
// ClaimAssets is called directly, without upstreamRetryFrom(ctx).do. The lane
// is an optimization that fails open: a transient error costs one run of
// ordinary scans, while retrying would delay the scan (up to the retry's
// 2-minute wait) for something it does not need, and would let a claim failure
// flip the shared "platform is down" state that the calls a scan depends on
// read. An old server answers the unknown method with NotFound, which is
// permanent in the classifier too, but never reaches it here.
func claimAssetsBatch(
	ctx context.Context,
	services *policy.Services,
	spaceMrn string,
	children []*discovery.TrackedAsset,
) (map[*discovery.TrackedAsset]struct{}, error) {
	candidates := make([]*discovery.TrackedAsset, 0, len(children))
	for _, child := range children {
		if child != nil && child.Asset != nil && len(child.Asset.Claims) > 0 {
			candidates = append(candidates, child)
		}
	}

	claimed := map[*discovery.TrackedAsset]struct{}{}
	for start := 0; start < len(candidates); start += claimBatchSize {
		chunk := candidates[start:min(start+claimBatchSize, len(candidates))]
		list := make([]*inventory.Asset, len(chunk))
		for i, child := range chunk {
			list[i] = child.Asset
		}

		resp, err := services.ClaimAssets(ctx, &policy.ClaimAssetsReq{SpaceMrn: spaceMrn, List: list})
		if err != nil {
			return nil, err
		}
		for _, v := range resp.GetVerdicts() {
			if v.GetAction() != policy.ClaimAssetVerdict_CLAIMED {
				continue
			}
			// Indices are relative to this chunk's request.
			if v.Index < 0 || int(v.Index) >= len(chunk) {
				continue
			}
			claimed[chunk[v.Index]] = struct{}{}
		}
	}
	return claimed, nil
}

// claimLaneState is the per-run state of the claim lane. enabled is set once
// at wiring time; disable is one-way for the rest of the run.
type claimLaneState struct {
	enabled  bool
	disabled atomic.Bool
}

// active reports whether the lane is on and has not been disabled.
func (c *claimLaneState) active() bool {
	return c != nil && c.enabled && !c.disabled.Load()
}

// disable turns the lane off for the rest of the run and logs why, once.
func (c *claimLaneState) disable(reason string, err error) {
	if !c.disabled.CompareAndSwap(false, true) {
		return
	}
	log.Warn().Err(err).Str("reason", reason).
		Msg("claim lane disabled for this run, scanning claim-bearing assets as usual")
}
