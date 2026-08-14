// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"context"
	"sort"
	"strings"
	"sync"

	"go.mondoo.com/cnspec/v13/policy"
	"go.mondoo.com/mql/v13/providers-sdk/v1/resources"
)

// bundleCompileCache reuses the compiled form of a policy bundle across the
// assets of one scan.
//
// Bundle.CompileExt compiles every query in the bundle. The scanner calls it
// once per asset, so a scan of N assets compiles the same bundle N times. The
// result only depends on the bundle and on the resource schema, so a scan of
// assets that share a schema can compile once and reuse the result.
//
// The cache holds a single entry. Scans are normally homogeneous, and a scan
// that alternates between schemas still stays correct because a miss falls
// back to a full compile.
type bundleCompileCache struct {
	mu     sync.Mutex
	bundle *policy.Bundle
	key    string
	result *policy.PolicyBundleMap
}

func newBundleCompileCache() *bundleCompileCache {
	return &bundleCompileCache{}
}

// schemaKey identifies the set of providers that back a resource schema.
// Loading a provider changes the set, which changes the queries that compile.
func schemaKey(schema resources.ResourcesSchema) string {
	if schema == nil {
		return ""
	}
	deps := schema.AllDependencies()
	ids := make([]string, 0, len(deps))
	for id := range deps {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return strings.Join(ids, ",")
}

// compile returns the compiled form of the bundle. It reuses the previous
// result when the bundle and the schema are unchanged, otherwise it compiles
// and stores the result.
//
// The returned map is a shallow copy that carries the caller's Library. The
// Library is not used during compilation, it is only attached to the result,
// so callers that share a compiled bundle still get their own data lake.
func (c *bundleCompileCache) compile(ctx context.Context, bundle *policy.Bundle, conf policy.BundleCompileConf) (*policy.PolicyBundleMap, error) {
	key := schemaKey(conf.Schema)

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.result != nil && c.bundle == bundle && c.key == key {
		res := *c.result
		res.Library = conf.Library
		return &res, nil
	}

	res, err := bundle.CompileExt(ctx, conf)
	if err != nil {
		return nil, err
	}

	c.bundle = bundle
	c.key = key
	c.result = res

	cp := *res
	cp.Library = conf.Library
	return &cp, nil
}
