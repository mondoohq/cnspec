// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package bundle

import (
	"fmt"
	"sort"
	"strings"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/mqlc"
	"go.mondoo.com/mql/v13/providers-sdk/v1/resources"
)

const (
	QueryShadowedAccessorRuleID = "query-shadowed-accessor"
)

// lintShadowedAccessors reports queries that spell out a dotted path which is
// also a resource name, when the resource one level up declares an accessor of
// that name.
//
// The compiler resolves the longest matching resource name first
// (mqlc.compileResource) without checking whether the resource it already has
// declares a field by that name. So
//
//	azure.subscription.aksService.cluster.autoUpgradeProfile.upgradeChannel
//
// does not compile to a field read on a cluster: the whole prefix is itself a
// resource name, so it compiles to a bare resource creation with no binding
// and no arguments. The runtime builds that resource with no id and no fields
// set, the cluster's autoUpgradeProfile() accessor never runs, and every field
// read on it crosses the plugin boundary as an empty DataRes -- logging
//
//	provider returned no data and no error for a field ... field=upgradeChannel id=
//	llx: encountered a primitive with no type information, coercing to null
//
// once per field, per asset, per scan.
//
// The query does not fail. It evaluates against nulls, and because MQL's
// `null != "x"` is true while `null == "x"` is false, the check reports a
// confident wrong answer in one direction or the other rather than an error.
// That is what makes this worth linting: it is invisible in the results.
func lintShadowedAccessors(schema resources.ResourcesSchema, conf mqlc.CompilerConfig, filename string, b *Bundle) []*Entry {
	var entries []*Entry

	visit := func(q *Mquery) {
		entries = append(entries, walkQueryForShadowedAccessors(schema, conf, filename, q)...)
	}

	for _, q := range b.Queries {
		visit(q)
	}
	for _, p := range b.Policies {
		for _, group := range p.Groups {
			for _, q := range group.Checks {
				visit(q)
			}
			for _, q := range group.Queries {
				visit(q)
			}
			// filters are compiled and run like any other query
			if group.Filters != nil {
				for _, q := range group.Filters.Items {
					visit(q)
				}
			}
		}
	}
	for _, pack := range b.Packs {
		for _, q := range pack.Queries {
			visit(q)
		}
		for _, group := range pack.Groups {
			for _, q := range group.Queries {
				visit(q)
			}
		}
	}

	return entries
}

func walkQueryForShadowedAccessors(schema resources.ResourcesSchema, conf mqlc.CompilerConfig, filename string, q *Mquery) []*Entry {
	if q == nil || q.Mql == "" {
		return nil
	}

	// Compile errors are already reported by bundle-compile-error; a query that
	// does not compile has nothing to say about accessor shadowing.
	_, bundle, err := mqlc.AnalyzeQuery(q.Mql, mqlc.EmptyPropsHandler, conf)
	if err != nil || bundle == nil || bundle.CodeV2 == nil {
		return nil
	}

	loc := []Location{{File: filename, Line: q.FileContext.Line, Column: q.FileContext.Column}}
	display := queryDisplayID(q)

	// A query can reach the same resource more than once; report it once.
	seen := map[string]struct{}{}
	var shadowed []string
	for _, block := range bundle.CodeV2.Blocks {
		for _, chunk := range block.Chunks {
			name, ok := bareResourceName(chunk)
			if !ok {
				continue
			}
			if _, dup := seen[name]; dup {
				continue
			}
			if _, _, ok := shadowedAccessor(schema, name); !ok {
				continue
			}
			seen[name] = struct{}{}
			shadowed = append(shadowed, name)
		}
	}
	sort.Strings(shadowed)

	var entries []*Entry
	for _, name := range shadowed {
		parent, field, _ := shadowedAccessor(schema, name)
		entries = append(entries, &Entry{
			RuleID: QueryShadowedAccessorRuleID,
			Level:  LevelWarning,
			Message: fmt.Sprintf(
				"query '%s' reaches '%s' as a resource, which shadows the '%s' accessor on '%s'. "+
					"The resource is created with no id, so every field reads null and the check silently "+
					"asserts against nothing. Reach it through the accessor instead, e.g. `%s { %s.<field> }`",
				display, name, field, parent, parent, field),
			Location: loc,
		})
	}

	return entries
}

// bareResourceName returns the resource a chunk creates from scratch -- no
// binding, no parent -- and whether the chunk is that kind of chunk at all.
//
// mqlc.addResource emits `Chunk{Call: FUNCTION, Id: <resource name>}` with a
// nil Function for a bare resource, or a Function carrying only args (never a
// Binding) when the resource is called with arguments. A field read always
// carries a Binding, and the implicit-resource path emits Id "createResource",
// so neither is mistaken for a bare creation here.
func bareResourceName(chunk *llx.Chunk) (string, bool) {
	if chunk == nil || chunk.Call != llx.Chunk_FUNCTION {
		return "", false
	}
	if chunk.Function != nil && chunk.Function.Binding != 0 {
		return "", false
	}
	if !strings.Contains(chunk.Id, ".") {
		return "", false
	}
	return chunk.Id, true
}

// shadowedAccessor reports whether reaching `name` as a bare resource bypasses
// an accessor that its parent declares, and returns that parent and leaf.
//
// Three conditions have to hold together. Each one on its own produces false
// positives, and all three are readable from the schema:
//
//  1. The parent declares a field of the leaf name that is NOT
//     `IsImplicitResource`. The schema builder synthesizes a field on the
//     parent for every dotted resource name, so `aws.ec2` gets an `instance`
//     field purely because `aws.ec2.instance` exists; the dotted path is the
//     only way to reach those and is correct. A non-implicit field was written
//     by hand in the .lr as `field() some.resource`, so a real accessor exists.
//
//  2. The resource has no `init`. A resource declaring `init(...)` knows how to
//     resolve itself from arguments or from the scanned asset -- list
//     resources like `microsoft.users` and asset-scoped ones are meant to be
//     reached bare.
//
//  3. At least one field is static (`IsMandatory`, which lrcore sets from
//     BasicField.isStatic -- a field declared without accessor parens). Static
//     fields have no source other than the creator, so a bare resource leaves
//     them unset. A resource whose fields are all computed accessors, like
//     `os.date` with `time()`/`timezone()`, fetches its own data and is
//     perfectly fine to create bare.
func shadowedAccessor(schema resources.ResourcesSchema, name string) (parent string, field string, ok bool) {
	idx := strings.LastIndex(name, ".")
	if idx < 0 {
		return "", "", false
	}
	parent, field = name[:idx], name[idx+1:]

	pinfo := schema.Lookup(parent)
	if pinfo == nil {
		return "", "", false
	}
	f := pinfo.Fields[field]
	if f == nil || f.IsImplicitResource {
		return "", "", false
	}

	rinfo := schema.Lookup(name)
	if rinfo == nil || rinfo.Init != nil {
		return "", "", false
	}
	if !hasStaticField(rinfo) {
		return "", "", false
	}
	return parent, field, true
}

// hasStaticField reports whether the resource declares at least one field that
// only its creator can populate.
func hasStaticField(r *resources.ResourceInfo) bool {
	for _, f := range r.Fields {
		if f.IsMandatory {
			return true
		}
	}
	return false
}
