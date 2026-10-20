// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"sync"

	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/mql/llx"
	"go.mondoo.com/mql/providers-sdk/v1/resources"
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
// private copy once and then skips it, so another CompileExt call on the same
// bundle would mutate it a second time outside the cache.
type bundleCompileCache struct {
	mu          sync.Mutex
	compiling   bool
	compileDone chan struct{}
	bundle      *policy.Bundle
	key         string
	result      *policy.PolicyBundleMap
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

// copyBundleMap copies a compiled bundle map and attaches the given library.
//
// Policies, frameworks, queries, properties, and risk factors are cloned
// because SetBundleMap stores them in the data lake and may replace computed
// filters on them. Sharing those values across assets would race. Compiled
// code is immutable after compilation and remains shared so the cache still
// avoids duplicating the expensive result.
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
		if v != nil {
			dst.Policies[k] = v.CloneVT()
		}
	}
	for k, v := range src.Frameworks {
		if v != nil {
			dst.Frameworks[k] = v.CloneVT()
		}
	}
	for k, v := range src.Queries {
		if v != nil {
			dst.Queries[k] = v.CloneVT()
		}
	}
	for k, v := range src.Props {
		if v != nil {
			dst.Props[k] = v.CloneVT()
		}
	}
	for k, v := range src.Code {
		dst.Code[k] = v
	}
	for k, v := range src.RiskFactors {
		if v != nil {
			dst.RiskFactors[k] = v.CloneVT()
		}
	}

	return dst
}

// compile returns the compiled form of the bundle. It reuses the previous
// result when the bundle and the schema are unchanged, otherwise it compiles
// and stores the result.
//
// Each caller gets its own map values and its own Library, so concurrent
// assets do not write into shared bundle state.
func (c *bundleCompileCache) compile(ctx context.Context, bundle *policy.Bundle, conf policy.BundleCompileConf) (*policy.PolicyBundleMap, error) {
	key := schemaKey(conf.Schema)

	for {
		c.mu.Lock()
		if c.result != nil && c.bundle == bundle && c.key == key {
			res := c.result
			c.mu.Unlock()
			return copyBundleMap(res, conf.Library), nil
		}

		if c.compiling {
			done := c.compileDone
			c.mu.Unlock()
			select {
			case <-done:
				continue
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		c.compiling = true
		c.compileDone = make(chan struct{})
		done := c.compileDone
		c.mu.Unlock()

		compiledBundle := bundle.CloneVT()
		res, err := compiledBundle.CompileExt(ctx, conf)

		c.mu.Lock()
		if err == nil {
			c.bundle = bundle
			c.key = key
			c.result = res
		}
		c.compiling = false
		close(done)
		c.compileDone = nil
		c.mu.Unlock()

		if err != nil {
			return nil, err
		}
		return copyBundleMap(res, conf.Library), nil
	}
}
