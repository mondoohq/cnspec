// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scanstats

import "context"

type collectorCtxKey struct{}

// ContextWithCollector returns a ctx carrying c so downstream code (e.g. the
// graph executor) can record scan metrics into it without threading it through
// every signature.
func ContextWithCollector(ctx context.Context, c *Collector) context.Context {
	return context.WithValue(ctx, collectorCtxKey{}, c)
}

// CollectorFromContext returns the Collector carried by ctx, or nil.
func CollectorFromContext(ctx context.Context) *Collector {
	c, _ := ctx.Value(collectorCtxKey{}).(*Collector)
	return c
}
