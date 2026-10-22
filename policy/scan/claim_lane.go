// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"context"
	"maps"
	"slices"
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
			list[i] = claimProjection(child)
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

	if len(candidates) > 0 {
		log.Info().Int("asked", len(candidates)).Int("claimed", len(claimed)).
			Str("space", spaceMrn).Msg("claim lane verdicts")
		for child := range claimed {
			ev := log.Debug().Str("name", child.Asset.Name)
			if len(child.Asset.PlatformIds) > 0 {
				ev = ev.Str("platformId", child.Asset.PlatformIds[0])
			}
			ev.Msg("asset claimed, not scanning")
		}
	}
	return claimed, nil
}

// claimProjection builds the asset sent to ClaimAssets: a minimal projection of
// a pre-Connect child. It carries only Id, Name, PlatformIds, Claims, Labels,
// Annotations, ManagedBy, TraceId and Category. It never carries Connections
// (they hold credentials and the claim endpoint has no use for them) or
// anything else the asset holds.
//
// A child that was never connected has not been through mql's prepareAsset,
// which would have stamped it with its root's bookkeeping. The server's GC of
// claimed assets needs that bookkeeping, so the projection inherits it from the
// subtree root, which is connected and prepared already: ManagedBy and TraceId
// when the child has none, the root's mondoo labels (AddMondooLabels, the same
// call prepareAsset makes) merged under the child's own labels, and the root's
// annotations for keys the child does not set. This mirrors prepareAsset
// (mondoohq/mql discovery/discovery.go) for exactly these fields, and so can
// drift from it: if prepareAsset starts propagating more, this must follow.
// It deliberately differs in one way: the child's own values win over the
// root's, where prepareAsset lets the root overwrite.
func claimProjection(child *discovery.TrackedAsset) *inventory.Asset {
	a := child.Asset
	p := &inventory.Asset{
		Id:          a.Id,
		Name:        a.Name,
		PlatformIds: slices.Clone(a.PlatformIds),
		Claims:      slices.Clone(a.Claims),
		Labels:      maps.Clone(a.Labels),
		Annotations: maps.Clone(a.Annotations),
		ManagedBy:   a.ManagedBy,
		TraceId:     a.TraceId,
		Category:    a.Category,
	}

	root := subtreeRoot(child)
	if root == nil || root.Asset == nil {
		return p
	}
	if p.ManagedBy == "" {
		p.ManagedBy = root.Asset.ManagedBy
	}
	if p.TraceId == "" {
		p.TraceId = root.Asset.TraceId
	}
	p.AddMondooLabels(root.Asset)
	for k, v := range root.Asset.Annotations {
		if _, ok := p.Annotations[k]; ok {
			continue
		}
		if p.Annotations == nil {
			p.Annotations = map[string]string{}
		}
		p.Annotations[k] = v
	}
	return p
}

// subtreeRoot returns the topmost tracked ancestor of child, or nil when the
// child has no parent.
func subtreeRoot(child *discovery.TrackedAsset) *discovery.TrackedAsset {
	var root *discovery.TrackedAsset
	for p := child.Parent; p != nil; p = p.Parent {
		root = p
	}
	return root
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
