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

func walkQueryForDeprecatedSymbols(schema resources.ResourcesSchema, conf mqlc.CompilerConfig, filename string, q *Mquery) []*Entry {
	if q == nil || q.Mql == "" {
		return nil
	}

	usage, _, err := mqlc.AnalyzeQuery(q.Mql, mqlc.EmptyPropsHandler, conf)
	if err != nil || usage == nil {
		return nil
	}

	loc := []Location{{File: filename, Line: q.FileContext.Line, Column: q.FileContext.Column}}
	display := queryDisplayID(q)

	// Sort for stable output across runs.
	providerIDs := make([]string, 0, len(usage.Providers))
	for id := range usage.Providers {
		providerIDs = append(providerIDs, id)
	}
	sort.Strings(providerIDs)

	var entries []*Entry
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
				entries = append(entries, &Entry{
					RuleID:   QueryDeprecatedSymbolRuleID,
					Level:    LevelWarning,
					Message:  fmt.Sprintf("query '%s' uses deprecated resource '%s'", display, rname),
					Location: loc,
				})
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
				entries = append(entries, &Entry{
					RuleID:   QueryDeprecatedSymbolRuleID,
					Level:    LevelWarning,
					Message:  fmt.Sprintf("query '%s' uses deprecated field '%s.%s'", display, rname, fname),
					Location: loc,
				})
			}
		}
	}

	return entries
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
