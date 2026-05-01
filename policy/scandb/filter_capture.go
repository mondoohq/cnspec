// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scandb

import (
	"context"
)

// FilterCaptureFunc is invoked once per asset by the scanner with the
// code_ids of the filters it sent to ResolveAndUpdateJobs. Implementations
// typically persist the code_ids to a scan database so the loadtest tool
// can replay the same filters against synthetic assets.
type FilterCaptureFunc func(codeIDs []string)

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
func CaptureFilters(ctx context.Context, codeIDs []string) {
	if len(codeIDs) == 0 {
		return
	}
	if cb, ok := ctx.Value(filterCaptureKey{}).(FilterCaptureFunc); ok && cb != nil {
		cb(codeIDs)
	}
}
