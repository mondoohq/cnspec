// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package bundle

import (
	"fmt"
	"sort"

	"go.mondoo.com/mql/llx"
	"go.mondoo.com/mql/mqlc"
	"go.mondoo.com/mql/types"
)

const (
	// QueryStandaloneWhereRuleID reports a query whose mql filters a resource
	// list with a condition that never reads the list element.
	QueryStandaloneWhereRuleID = "query-standalone-where"

	// FilterStandaloneWhereRuleID is the same finding in a filters block, split
	// by site like the deprecated-symbol rules.
	FilterStandaloneWhereRuleID = "filter-standalone-where"
)

// lintStandaloneWhere compiles every query and filter in the bundle and
// reports each resource list that is filtered by a condition which never
// references the list element, such as `users.all(processes.length > 0)`.
// Such a condition has the same value for every element, so the filter keeps
// or drops the whole list, and a check written with it passes or fails
// independently of the elements it claims to test. `where`, `all`, `any`,
// `none`, `one` and `contains` on a resource list all compile to such a
// filter.
//
// The compiler only logs this case without saying which query it came from;
// this rule names the query and points at its line. Compile errors are
// ignored here, bundle-compile-error already surfaces them.
func lintStandaloneWhere(conf mqlc.CompilerConfig, filename string, b *Bundle) []*Entry {
	var entries []*Entry

	visitFilters := func(subject string, filters *Filters) {
		resources := standaloneWhereInFilters(filters, conf)
		if len(resources) == 0 {
			return
		}
		loc := []Location{{File: filename, Line: filters.FileContext.Line, Column: filters.FileContext.Column}}
		for _, name := range resources {
			entries = append(entries, &Entry{
				RuleID:   FilterStandaloneWhereRuleID,
				Level:    LevelWarning,
				Message:  fmt.Sprintf("%s filters '%s' in its filters with a condition that never references the list element, so it keeps or drops the whole list", subject, name),
				Location: loc,
			})
		}
	}
	visit := func(q *Mquery) {
		if q == nil {
			return
		}
		display := queryDisplayID(q)
		loc := []Location{{File: filename, Line: q.FileContext.Line, Column: q.FileContext.Column}}
		for _, name := range standaloneWhereIn(q.Mql, conf) {
			entries = append(entries, &Entry{
				RuleID:   QueryStandaloneWhereRuleID,
				Level:    LevelWarning,
				Message:  fmt.Sprintf("query '%s' filters '%s' with a condition that never references the list element, so it keeps or drops the whole list", display, name),
				Location: loc,
			})
		}
		visitFilters(fmt.Sprintf("query '%s'", display), q.Filters)
	}

	for _, q := range b.Queries {
		visit(q)
	}
	for _, p := range b.Policies {
		policy := fmt.Sprintf("policy '%s'", policyDisplayID(p))
		for i, group := range p.Groups {
			visitFilters(fmt.Sprintf("%s %s", policy, groupDisplay(group.Uid, group.Title, i)), group.Filters)
			for _, q := range group.Checks {
				visit(q)
			}
			for _, q := range group.Queries {
				visit(q)
			}
		}
	}
	for _, pack := range b.Packs {
		queryPack := fmt.Sprintf("query pack '%s'", queryPackDisplayID(pack))
		visitFilters(queryPack, pack.Filters)
		for _, q := range pack.Queries {
			visit(q)
		}
		for i, group := range pack.Groups {
			visitFilters(fmt.Sprintf("%s %s", queryPack, groupDisplay(group.Uid, group.Title, i)), group.Filters)
			for _, q := range group.Queries {
				visit(q)
			}
		}
	}

	return entries
}

// standaloneWhereInFilters returns the resource lists filtered by a standalone
// condition across every item of a filters block, each reported once. Items
// are walked in sorted key order for stable output.
func standaloneWhereInFilters(filters *Filters, conf mqlc.CompilerConfig) []string {
	if filters == nil || len(filters.Items) == 0 {
		return nil
	}

	keys := make([]string, 0, len(filters.Items))
	for key := range filters.Items {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	seen := map[string]struct{}{}
	var names []string
	for _, key := range keys {
		filter := filters.Items[key]
		if filter == nil {
			continue
		}
		for _, name := range standaloneWhereIn(filter.Mql, conf) {
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
			names = append(names, name)
		}
	}
	return names
}

// standaloneWhereIn compiles one MQL snippet and returns, in order of
// appearance, the resource lists it filters with a condition that never
// references the list element. Compile errors yield nothing.
func standaloneWhereIn(query string, conf mqlc.CompilerConfig) []string {
	if query == "" {
		return nil
	}
	bundle, err := mqlc.Compile(query, mqlc.EmptyPropsHandler, conf)
	if err != nil || bundle == nil || bundle.CodeV2 == nil {
		return nil
	}
	return standaloneResourceFilters(bundle.CodeV2)
}

// standaloneResourceFilters walks compiled code for resource-list filters
// (`where`, and `$whereNot` which `all` compiles to) whose function block
// never reads its bound element. A resource filter's chunk carries the
// resource type, while the same call on an array carries an array type; on
// an array a standalone value is a legitimate comparison (`list.contains(1)`)
// and is not reported.
func standaloneResourceFilters(code *llx.CodeV2) []string {
	var names []string
	for _, block := range code.Blocks {
		for _, chunk := range block.Chunks {
			if chunk.Call != llx.Chunk_FUNCTION || chunk.Function == nil {
				continue
			}
			if chunk.Id != "where" && chunk.Id != "$whereNot" {
				continue
			}
			typ := types.Type(chunk.Function.Type)
			if !typ.IsResource() || len(chunk.Function.Args) < 2 {
				continue
			}
			fn := chunk.Function.Args[1]
			if types.Type(fn.Type).Underlying() != types.FunctionLike {
				continue
			}
			blockRef, ok := fn.RefV2()
			if !ok || blockRef == 0 || int(blockRef>>32) > len(code.Blocks) {
				continue
			}
			if !blockReadsRef(code.Block(blockRef), blockRef|1) {
				names = append(names, typ.ResourceName())
			}
		}
	}
	return names
}

// blockReadsRef reports whether any chunk of the block reads ref, as the
// binding of a call or as an argument. A nested block that reads it passes it
// in as a dependency argument of the call that owns the nested block, so the
// chunks of this block are enough.
func blockReadsRef(block *llx.Block, ref uint64) bool {
	for _, ep := range block.Entrypoints {
		if ep == ref {
			return true
		}
	}
	for _, chunk := range block.Chunks {
		if chunk.Function == nil {
			if primitiveReadsRef(chunk.Primitive, ref) {
				return true
			}
			continue
		}
		if chunk.Function.Binding == ref {
			return true
		}
		for _, arg := range chunk.Function.Args {
			if primitiveReadsRef(arg, ref) {
				return true
			}
		}
	}
	return false
}

func primitiveReadsRef(p *llx.Primitive, ref uint64) bool {
	if p == nil {
		return false
	}
	if types.Type(p.Type) == types.Ref {
		if r, ok := p.RefV2(); ok && r == ref {
			return true
		}
	}
	for _, item := range p.Array {
		if primitiveReadsRef(item, ref) {
			return true
		}
	}
	for _, v := range p.Map {
		if primitiveReadsRef(v, ref) {
			return true
		}
	}
	return false
}
