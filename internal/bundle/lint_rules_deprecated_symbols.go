// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package bundle

import (
	"fmt"

	"go.mondoo.com/mql/v13/llx"
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

	bundle, err := mqlc.Compile(q.Mql, mqlc.EmptyPropsHandler, conf)
	if err != nil || bundle == nil || bundle.CodeV2 == nil {
		return nil
	}

	seen := map[string]struct{}{}
	var entries []*Entry
	emit := func(symbol, message string) {
		if _, ok := seen[symbol]; ok {
			return
		}
		seen[symbol] = struct{}{}
		entries = append(entries, &Entry{
			RuleID:  QueryDeprecatedSymbolRuleID,
			Level:   LevelWarning,
			Message: message,
			Location: []Location{{
				File:   filename,
				Line:   q.FileContext.Line,
				Column: q.FileContext.Column,
			}},
		})
	}

	code := bundle.CodeV2
	for _, block := range code.Blocks {
		for _, chunk := range block.Chunks {
			if chunk == nil || chunk.Call != llx.Chunk_FUNCTION || chunk.Id == "" {
				continue
			}

			// A bare resource access (e.g. `processes`) is emitted as a FUNCTION
			// chunk with Function == nil. A resource initialized with arguments
			// (e.g. `file('/etc/passwd')`) is a FUNCTION chunk with Function set
			// and Binding == 0.
			if chunk.Function == nil || chunk.Function.Binding == 0 {
				resource := schema.Lookup(chunk.Id)
				if resource == nil {
					continue
				}
				if resource.GetMaturity() == resources.MaturityDeprecated {
					emit("resource:"+chunk.Id,
						fmt.Sprintf("query '%s' uses deprecated resource '%s'", queryDisplayID(q), chunk.Id))
				}
				continue
			}

			parentName := bindingResourceName(code, chunk.Function.Binding)
			if parentName == "" {
				continue
			}
			resource, field := schema.LookupField(parentName, chunk.Id)
			if resource == nil || field == nil {
				continue
			}
			if resources.EffectiveFieldMaturity(resource, field) == resources.MaturityDeprecated {
				emit("field:"+parentName+"."+chunk.Id,
					fmt.Sprintf("query '%s' uses deprecated field '%s.%s'", queryDisplayID(q), parentName, chunk.Id))
			}
		}
	}

	return entries
}

// bindingResourceName resolves the resource name returned by the chunk that a
// function call is bound to. Returns "" if the binding does not resolve to a
// resource type (e.g. a builtin operating on a primitive, or a block parameter).
func bindingResourceName(code *llx.CodeV2, ref uint64) string {
	if ref == 0 {
		return ""
	}
	blockIdx := int(ref>>32) - 1
	if blockIdx < 0 || blockIdx >= len(code.Blocks) {
		return ""
	}
	block := code.Blocks[blockIdx]
	chunkIdx := int(uint32(ref)) - 1
	if chunkIdx < 0 || chunkIdx >= len(block.Chunks) {
		return ""
	}
	chunk := block.Chunks[chunkIdx]
	if chunk == nil {
		return ""
	}
	typ := chunk.Type()
	for typ.IsArray() || typ.IsMap() {
		typ = typ.Child()
	}
	if !typ.IsResource() {
		return ""
	}
	return typ.ResourceName()
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
