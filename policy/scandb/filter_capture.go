// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scandb

import (
	"context"
)

// FilterCaptureFunc is invoked once per asset by the scanner with the
// code_ids of the filters it sent to ResolveAndUpdateJobs. Implementations
// typically persist the code_ids to a scan database so downstream replay
// (loadtest tool, debugging) can call ResolveAndUpdateJobs without having
// to re-derive the filter set from policy bundles.
type FilterCaptureFunc func(codeIDs []string)

type filterCaptureKey struct{}

// WithFilterCapture returns a copy of ctx that, when passed to
// CaptureFilters, invokes f. The SQLite datalake installs this hook on
// every scan; ctx values without a hook make CaptureFilters a no-op.
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
