// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"sync"

	"go.mondoo.com/cnspec/v13/policy"
	"go.mondoo.com/mql/v13/llx"
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
//
// Invariant: for a given bundle the cache must be the only caller of
// CompileExt. CompileExt mutates the bundle in place, it assigns MRNs,
// recomputes checksums and removes failing queries. The cache compiles a
// bundle once and then skips it, so another CompileExt call on the same
// bundle would mutate it a second time outside the cache.
type bundleCompileCache struct {
	mu     sync.Mutex
	bundle *policy.Bundle
	key    string
	result *policy.PolicyBundleMap
}

func newBundleCompileCache() *bundleCompileCache {
	return &bundleCompileCache{}
}

// schemaKey identifies the resource schema behind a compiler config. Loading a
// provider changes the set of providers and the number of resources, and both
// change the queries that compile.
//
// Provider versions are not part of the key because resources.ProviderInfo
// carries only an id and a name. That is sufficient here: the cache lives for
// one scan inside one process, and a provider cannot change version while the
// process runs. The resource count catches a schema that gains resources
// without gaining a provider, for example a resource extension.
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

	return strconv.Itoa(len(schema.AllResources())) + ":" + strings.Join(ids, ",")
}

// copyBundleMap copies the map headers of a compiled bundle map and attaches
// the given library.
//
// The values stay shared. They point into the bundle, which every asset
// already shares. The headers must not be shared, because SetBundleMap and the
// policy hub insert into Queries, Policies, Props and Frameworks while a scan
// runs. Sharing a header across concurrent assets would be a concurrent map
// write.
func copyBundleMap(src *policy.PolicyBundleMap, library policy.Library) *policy.PolicyBundleMap {
	dst := &policy.PolicyBundleMap{
		OwnerMrn:    src.OwnerMrn,
		Library:     library,
		Policies:    make(map[string]*policy.Policy, len(src.Policies)),
		Frameworks:  make(map[string]*policy.Framework, len(src.Frameworks)),
		Queries:     make(map[string]*policy.Mquery, len(src.Queries)),
		Props:       make(map[string]*policy.Property, len(src.Props)),
		Code:        make(map[string]*llx.CodeBundle, len(src.Code)),
		RiskFactors: make(map[string]*policy.RiskFactor, len(src.RiskFactors)),
	}

	for k, v := range src.Policies {
		dst.Policies[k] = v
	}
	for k, v := range src.Frameworks {
		dst.Frameworks[k] = v
	}
	for k, v := range src.Queries {
		dst.Queries[k] = v
	}
	for k, v := range src.Props {
		dst.Props[k] = v
	}
	for k, v := range src.Code {
		dst.Code[k] = v
	}
	for k, v := range src.RiskFactors {
		dst.RiskFactors[k] = v
	}

	return dst
}

// compile returns the compiled form of the bundle. It reuses the previous
// result when the bundle and the schema are unchanged, otherwise it compiles
// and stores the result.
//
// Each caller gets its own map headers and its own Library, so concurrent
// assets do not write into shared maps.
func (c *bundleCompileCache) compile(ctx context.Context, bundle *policy.Bundle, conf policy.BundleCompileConf) (*policy.PolicyBundleMap, error) {
	key := schemaKey(conf.Schema)

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.result != nil && c.bundle == bundle && c.key == key {
		return copyBundleMap(c.result, conf.Library), nil
	}

	res, err := bundle.CompileExt(ctx, conf)
	if err != nil {
		return nil, err
	}

	c.bundle = bundle
	c.key = key
	c.result = res

	return copyBundleMap(res, conf.Library), nil
}
