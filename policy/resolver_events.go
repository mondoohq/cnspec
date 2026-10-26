// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package policy

import "context"

// ResolverEvents receives notable outcomes of policy resolution, so a host
// (e.g. the Mondoo server) can record them in its own telemetry without the
// resolver depending on it. Implementations must be safe for concurrent use
// and should not block: they are called on the scan resolve path.
type ResolverEvents interface {
	// ResolvedPolicyCacheMiss is called when tryResolve looked up the
	// resolved-policy cache and did not find a row, so it fell through to a
	// full bundle load and resolve.
	ResolvedPolicyCacheMiss(ctx context.Context, ev *ResolvedPolicyCacheMiss)
}

// ResolvedPolicyCacheMiss describes one resolved-policy cache miss.
//
// The cache is looked up and stored by the checksum of all asset filters the
// caller sent, while the policy is resolved with only the filters that match
// it. DroppedFilters names the filters the caller sent that the policy does
// not match.
type ResolvedPolicyCacheMiss struct {
	// PolicyMrn is the policy that was resolved; the asset MRN on the scan path.
	PolicyMrn string
	// LookupFiltersChecksum is the cache key that missed: the checksum of all
	// asset filters the caller sent.
	LookupFiltersChecksum string
	// StoredFiltersChecksum is the key the resolved policy is stored under. It
	// equals LookupFiltersChecksum; a difference means a stored row can never
	// be found again. Empty when no filter matched and nothing was stored.
	StoredFiltersChecksum string
	// AssetFilters is the number of filters the caller sent.
	AssetFilters int
	// MatchingFilters is the number of those filters the policy supports.
	MatchingFilters int
	// DroppedFilters are the filters the caller sent that the policy does not
	// support.
	DroppedFilters []DroppedAssetFilter
}

// DroppedAssetFilter is an asset filter that did not match the policy's
// computed filters.
type DroppedAssetFilter struct {
	Mql string
	// SentCodeId is the CodeId as the caller sent it, before the resolver
	// recompiled the filter. Empty if the caller sent none.
	SentCodeId string
	// CodeId is the CodeId after the resolver recompiled the filter, which is
	// what matching compares.
	CodeId string
	// PolicyCodeId is the CodeId the policy carries for a filter with the same
	// MQL. Empty when the policy has no such filter. When it is set and differs
	// from CodeId, the filter was dropped because the same MQL compiled to a
	// different CodeId, not because the policy lacks it.
	PolicyCodeId string
}

type resolverEventsCtxKey struct{}

// WithResolverEvents returns a ctx carrying ev, which the resolver reports its
// events to.
func WithResolverEvents(ctx context.Context, ev ResolverEvents) context.Context {
	return context.WithValue(ctx, resolverEventsCtxKey{}, ev)
}

// ResolverEventsFromContext returns the ResolverEvents carried by ctx, or nil.
func ResolverEventsFromContext(ctx context.Context) ResolverEvents {
	ev, _ := ctx.Value(resolverEventsCtxKey{}).(ResolverEvents)
	return ev
}

// droppedAssetFilters returns the asset filters that are not in matching.
// sentCodeIds holds each filter's CodeId from before it was recompiled.
func droppedAssetFilters(assetFilters []*Mquery, matching []*Mquery, sentCodeIds map[*Mquery]string, policyFilters *Filters) []DroppedAssetFilter {
	matched := make(map[string]struct{}, len(matching))
	for i := range matching {
		matched[matching[i].CodeId] = struct{}{}
	}

	policyCodeIds := map[string]string{}
	if policyFilters != nil {
		for _, f := range policyFilters.Items {
			policyCodeIds[f.Mql] = f.CodeId
		}
	}

	var res []DroppedAssetFilter
	for i := range assetFilters {
		f := assetFilters[i]
		if _, ok := matched[f.CodeId]; ok {
			continue
		}
		res = append(res, DroppedAssetFilter{
			Mql:          f.Mql,
			SentCodeId:   sentCodeIds[f],
			CodeId:       f.CodeId,
			PolicyCodeId: policyCodeIds[f.Mql],
		})
	}
	return res
}
