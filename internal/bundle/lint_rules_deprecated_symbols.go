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
)

// lintDeprecatedSymbols compiles every query with non-empty MQL and reports
// resources or fields whose effective maturity is "deprecated". Compile errors
// are intentionally ignored here — bundle-compile-error already surfaces them.
func lintDeprecatedSymbols(schema resources.ResourcesSchema, conf mqlc.CompilerConfig, filename string, b *Bundle) []*Entry {
	var entries []*Entry

	visit := func(q *Mquery) {
		entries = append(entries, walkQueryForDeprecatedSymbols(schema, conf, filename, q)...)
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

// walkQueryForDeprecatedSymbols reports deprecated symbols used by a query,
// looking at both its mql and its filters. Filters matter as much as mql here:
// when a deprecated symbol is finally removed, a filter that references it stops
// matching and the check silently stops scoring assets instead of failing loudly.
//
// A symbol used in both places is reported once. The mql site wins, because that
// is where a reader will find it; the "in its filters" hint is only added for
// symbols that appear nowhere else, since every warning points at the query's own
// line and filter Mqueries carry no file context of their own.
func walkQueryForDeprecatedSymbols(schema resources.ResourcesSchema, conf mqlc.CompilerConfig, filename string, q *Mquery) []*Entry {
	if q == nil {
		return nil
	}

	fromMql := deprecatedSymbolsIn(q.Mql, conf)

	seen := make(map[string]struct{}, len(fromMql))
	for _, symbol := range fromMql {
		seen[symbol] = struct{}{}
	}

	var fromFilters []string
	if q.Filters != nil {
		// Sort for stable output across runs.
		filterKeys := make([]string, 0, len(q.Filters.Items))
		for key := range q.Filters.Items {
			filterKeys = append(filterKeys, key)
		}
		sort.Strings(filterKeys)

		for _, key := range filterKeys {
			filter := q.Filters.Items[key]
			if filter == nil {
				continue
			}
			for _, symbol := range deprecatedSymbolsIn(filter.Mql, conf) {
				if _, ok := seen[symbol]; ok {
					continue
				}
				seen[symbol] = struct{}{}
				fromFilters = append(fromFilters, symbol)
			}
		}
	}

	if len(fromMql) == 0 && len(fromFilters) == 0 {
		return nil
	}

	loc := []Location{{File: filename, Line: q.FileContext.Line, Column: q.FileContext.Column}}
	display := queryDisplayID(q)

	entries := make([]*Entry, 0, len(fromMql)+len(fromFilters))
	newEntry := func(symbol, where string) *Entry {
		return &Entry{
			RuleID:   QueryDeprecatedSymbolRuleID,
			Level:    LevelWarning,
			Message:  fmt.Sprintf("query '%s' uses %s%s", display, symbol, where),
			Location: loc,
		}
	}
	for _, symbol := range fromMql {
		entries = append(entries, newEntry(symbol, ""))
	}
	for _, symbol := range fromFilters {
		entries = append(entries, newEntry(symbol, " in its filters"))
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
