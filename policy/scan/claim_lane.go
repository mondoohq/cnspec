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

// claimAssetsEnvVar overrides the server's claim_assets_enabled for one run:
// "1" or "force" turns the lane on regardless of the server, "0" turns it off
// regardless. Anything else defers to the server. It is read once per run, in
// distributeJob.
const claimAssetsEnvVar = "MONDOO_CLAIM_ASSETS"

// newClaimLaneState decides whether the lane is on for a run: the server's
// ScanParameters bit, overridden by the env value. The lane needs services to
// call ClaimAssets, so a run without them (incognito, no credentials) is never
// enabled, even when forced. Computing that here is what lets claimAssetsBatch
// rely on a non-nil services.
func newClaimLaneState(params *policy.ScanParameters, services *policy.Services, env string) *claimLaneState {
	enabled := params.GetClaimAssetsEnabled()
	switch env {
	case "1", "force":
		enabled = true
	case "0":
		enabled = false
	case "":
	default:
		log.Debug().Str("env", claimAssetsEnvVar).Str("value", env).
			Msg("unrecognized value, expected 1, force or 0; using the server setting")
	}
	return &claimLaneState{enabled: enabled && services != nil}
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
