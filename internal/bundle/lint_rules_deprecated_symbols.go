// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package bundle

import (
	"fmt"
	"sort"

	"go.mondoo.com/mql/v13/mqlc"
	"go.mondoo.com/mql/v13/providers-sdk/v1/resources"
)

const (
	QueryDeprecatedSymbolRuleID = "query-deprecated-symbol"

	// FilterDeprecatedSymbolRuleID covers every filters block, whoever owns it —
	// queries, policy groups, query packs and pack groups. The rule is split by
	// site rather than by owner: QueryDeprecatedSymbolRuleID is what a query's
	// mql uses, and a filter is a filter regardless of what it hangs off.
	FilterDeprecatedSymbolRuleID = "filter-deprecated-symbol"
)

// lintDeprecatedSymbols compiles every query and filter in the bundle and
// reports resources or fields whose effective maturity is "deprecated". Compile
// errors are intentionally ignored here — bundle-compile-error already surfaces
// them.
//
// Reporting is split by site, not by owner. A query's mql reports under
// QueryDeprecatedSymbolRuleID; every filters block reports under
// FilterDeprecatedSymbolRuleID, whether it hangs off a query, a policy group, a
// query pack or a pack group. Filters carry their own file context, so each
// warning points at the filters block that has to change.
//
// Container filters raise the stakes over a query's: when the symbol is removed
// the filter stops matching and the whole group drops out of scoring rather than
// one check.
//
// ComputedFilters is deliberately skipped. It is derived at load time from the
// group and query filters already walked here, so reporting it would duplicate
// warnings at a location the author never wrote.
func lintDeprecatedSymbols(schema resources.ResourcesSchema, conf mqlc.CompilerConfig, filename string, b *Bundle) []*Entry {
	var entries []*Entry

	visitFilters := func(subject string, filters *Filters) {
		entries = append(entries, walkFiltersForDeprecatedSymbols(conf, filename, subject, filters)...)
	}
	visit := func(q *Mquery) {
		if q == nil {
			return
		}
		entries = append(entries, walkQueryForDeprecatedSymbols(schema, conf, filename, q)...)
		visitFilters(fmt.Sprintf("query '%s'", queryDisplayID(q)), q.Filters)
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

// walkFiltersForDeprecatedSymbols reports deprecated symbols used by a filters
// block, whether it belongs to a query, a policy group, a query pack or a pack
// group. A Filters records its own file context, so warnings point straight at
// the block that has to change rather than at whatever owns it.
func walkFiltersForDeprecatedSymbols(conf mqlc.CompilerConfig, filename string, subject string, filters *Filters) []*Entry {
	symbols := deprecatedSymbolsInFilters(filters, conf)
	if len(symbols) == 0 {
		return nil
	}

	loc := []Location{{File: filename, Line: filters.FileContext.Line, Column: filters.FileContext.Column}}

	entries := make([]*Entry, 0, len(symbols))
	for _, symbol := range symbols {
		entries = append(entries, &Entry{
			RuleID:   FilterDeprecatedSymbolRuleID,
			Level:    LevelWarning,
			Message:  fmt.Sprintf("%s uses %s in its filters", subject, symbol),
			Location: loc,
		})
	}

	return entries
}

// deprecatedSymbolsInFilters returns the deprecated symbols used across every
// item of a filters block, reporting each one once no matter how many items
// mention it. Filters.Items is a map, so items are walked in sorted key order
// for stable output across runs.
func deprecatedSymbolsInFilters(filters *Filters, conf mqlc.CompilerConfig) []string {
	if filters == nil || len(filters.Items) == 0 {
		return nil
	}

	keys := make([]string, 0, len(filters.Items))
	for key := range filters.Items {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	seen := map[string]struct{}{}
	var symbols []string
	for _, key := range keys {
		filter := filters.Items[key]
		if filter == nil {
			continue
		}
		for _, symbol := range deprecatedSymbolsIn(filter.Mql, conf) {
			if _, ok := seen[symbol]; ok {
				continue
			}
			seen[symbol] = struct{}{}
			symbols = append(symbols, symbol)
		}
	}

	return symbols
}

// groupDisplay names a policy or query-pack group. Groups rarely carry a uid or
// a title, so it falls back to the group's position, which is still enough to
// find it alongside the reported line.
func groupDisplay(uid, title string, index int) string {
	if uid != "" {
		return fmt.Sprintf("group '%s'", uid)
	}
	if title != "" {
		return fmt.Sprintf("group '%s'", title)
	}
	return fmt.Sprintf("group %d", index+1)
}

func policyDisplayID(p *Policy) string {
	if p.Uid != "" {
		return p.Uid
	}
	return p.Mrn
}

func queryPackDisplayID(pack *QueryPack) string {
	if pack.Uid != "" {
		return pack.Uid
	}
	return pack.Mrn
}

// walkQueryForDeprecatedSymbols reports deprecated symbols used by a query's
// mql. Its filters are reported separately by walkFiltersForDeprecatedSymbols,
// which points at the filters block rather than at the query.
func walkQueryForDeprecatedSymbols(schema resources.ResourcesSchema, conf mqlc.CompilerConfig, filename string, q *Mquery) []*Entry {
	if q == nil {
		return nil
	}

	symbols := deprecatedSymbolsIn(q.Mql, conf)
	if len(symbols) == 0 {
		return nil
	}

	loc := []Location{{File: filename, Line: q.FileContext.Line, Column: q.FileContext.Column}}
	display := queryDisplayID(q)

	entries := make([]*Entry, 0, len(symbols))
	for _, symbol := range symbols {
		entries = append(entries, &Entry{
			RuleID:   QueryDeprecatedSymbolRuleID,
			Level:    LevelWarning,
			Message:  fmt.Sprintf("query '%s' uses %s", display, symbol),
			Location: loc,
		})
	}

	return entries
}

// deprecatedSymbolsIn compiles a single MQL snippet and returns the deprecated
// symbols it uses, as stable-sorted descriptions such as "deprecated resource
// 'processes'" or "deprecated field 'file.basename'". Compile errors are
// intentionally ignored — bundle-compile-error already surfaces them.
func deprecatedSymbolsIn(query string, conf mqlc.CompilerConfig) []string {
	if query == "" {
		return nil
	}

	usage, _, err := mqlc.AnalyzeQuery(query, mqlc.EmptyPropsHandler, conf)
	if err != nil || usage == nil {
		return nil
	}

	// Sort for stable output across runs.
	providerIDs := make([]string, 0, len(usage.Providers))
	for id := range usage.Providers {
		providerIDs = append(providerIDs, id)
	}
	sort.Strings(providerIDs)

	var symbols []string
	for _, pid := range providerIDs {
		pu := usage.Providers[pid]
		resourceNames := make([]string, 0, len(pu.Resources))
		for name := range pu.Resources {
			resourceNames = append(resourceNames, name)
		}
		sort.Strings(resourceNames)

		for _, rname := range resourceNames {
			ru := pu.Resources[rname]
			if ru.Maturity == resources.MaturityDeprecated {
				symbols = append(symbols, fmt.Sprintf("deprecated resource '%s'", rname))
				// Skip field warnings on a deprecated resource — every field
				// inherits the deprecated effective maturity, which would
				// drown the resource-level warning in noise.
				continue
			}

			fieldNames := make([]string, 0, len(ru.Fields))
			for name := range ru.Fields {
				fieldNames = append(fieldNames, name)
			}
			sort.Strings(fieldNames)

			for _, fname := range fieldNames {
				if ru.Fields[fname].EffectiveMaturity != resources.MaturityDeprecated {
					continue
				}
				symbols = append(symbols, fmt.Sprintf("deprecated field '%s.%s'", rname, fname))
			}
		}
	}

	return symbols
}

func queryDisplayID(q *Mquery) string {
	if q.Uid != "" {
		return q.Uid
	}
	if q.Mrn != "" {
		return q.Mrn
	}
	return fmt.Sprintf("at line %d", q.FileContext.Line)
}
