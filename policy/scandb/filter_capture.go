// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scandb

import (
	"context"

	"go.mondoo.com/cnspec/v13/policy"
)

// FilterCaptureFunc is invoked once per asset by the scanner with the asset
// filters it sent to ResolveAndUpdateJobs. Implementations typically persist
// the filters to a scan database so the loadtest tool can replay them
// against synthetic assets.
type FilterCaptureFunc func(filters *policy.Mqueries)

type filterCaptureKey struct{}

// WithFilterCapture returns a copy of ctx that, when passed to CaptureFilters,
// invokes f. Set by sqlite.WithServices when --output-scan-db is provided —
// otherwise the captured ctx has no value and CaptureFilters is a no-op.
func WithFilterCapture(ctx context.Context, f FilterCaptureFunc) context.Context {
	if f == nil {
		return ctx
	}
	return context.WithValue(ctx, filterCaptureKey{}, f)
}

// CaptureFilters invokes the FilterCaptureFunc previously installed via
// WithFilterCapture, if any. Safe to call on any ctx.
func CaptureFilters(ctx context.Context, filters *policy.Mqueries) {
	if filters == nil || len(filters.Items) == 0 {
		return
	}
	if cb, ok := ctx.Value(filterCaptureKey{}).(FilterCaptureFunc); ok && cb != nil {
		cb(filters)
	}
}
