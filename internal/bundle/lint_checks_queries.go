// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package bundle

import (
	"fmt"
)

// Query Rule ID Constants
const (
	QueryUidRuleID                         = "query-uid"
	QueryTitleRuleID                       = "query-name" // Original ID was query-name
	QueryUidUniqueRuleID                   = "query-uid-unique"
	QueryUnassignedRuleID                  = "query-unassigned"
	QueryUsedAsDifferentTypesRuleID        = "query-used-as-different-types"
	QueryMissingMQLRuleID                  = "query-missing-mql"
	QueryVariantUsesNonDefaultFieldsRuleID = "query-variant-uses-non-default-fields"
)

// GetQueryLintChecks is the input type for lint checks on queries.
func GetQueryLintChecks() []LintCheck {
	return []LintCheck{
		{
			ID:          QueryUidRuleID, // This check now covers UID presence, format, and uniqueness for global queries
			Name:        "Query UID Validation",
			Description: "Ensures global queries have a UID, it's correctly formatted, and unique within the file.",
			Severity:    LevelError,
			Run:         runCheckQueryUid,
		}, {
			ID:          QueryTitleRuleID,
			Name:        "Query Title Presence",
			Description: "Ensures non-variant queries have a `title` field.",
			Severity:    LevelError,
			Run:         runCheckQueryTitle,
		}, {
			ID:          QueryVariantUsesNonDefaultFieldsRuleID,
			Name:        "Query Variant Field Restrictions",
			Description: "Ensures variant queries do not define fields like impact, title, tags, or nested variants.",
			Severity:    LevelError,
			Run:         runCheckQueryVariantFields,
		}, {
			ID:          QueryMissingMQLRuleID,
			Name:        "Query MQL Presence (for Variants and Non-Variant Parents)",
			Description: "Ensures variant queries have MQL. Ensures parent queries without variants have MQL.",
			Severity:    LevelError,
			Run:         runCheckQueryMQLPresence,
		}, {
			ID:          QueryUnassignedRuleID,
			Name:        "Unassigned Query",
			Description: "Warns if a global query is defined but not assigned to any policy.",
			Severity:    LevelWarning,
			Run:         runCheckQueryUnassigned,
		}, {
			ID:          QueryUsedAsDifferentTypesRuleID,
			Name:        "Query Usage Consistency",
			Description: "Ensures a query is not used as both a check and a data query within policies.",
			Severity:    LevelError,
			Run:         runCheckQueryUsageConsistency,
		},
	}
}

func queryIdentifier(q *Mquery, isGlobal bool) string {
	prefix := "Embedded query"
	if isGlobal {
		prefix = "Global query"
	}
	if q.Uid != "" {
		return fmt.Sprintf("%s '%s'", prefix, q.Uid)
	}
	return fmt.Sprintf("%s at line %d", prefix, q.FileContext.Line)
}

func runCheckQueryUid(ctx *LintContext, item interface{}) []Entry {
	input, ok := item.(QueryLintInput)
	if !ok {
		return nil
	}
	q := input.Query
	isGlobal := input.IsGlobal
	var entries []Entry

	if isGlobal { // UID is mandatory and must be valid/unique for global queries
		if q.Uid == "" {
			entries = append(entries, Entry{
				RuleID:  QueryUidRuleID,
				Message: fmt.Sprintf("%s does not define a UID", queryIdentifier(q, isGlobal)),
				Level:   LevelError,
				Location: []Location{{
					File:   ctx.FilePath,
					Line:   q.FileContext.Line,
					Column: q.FileContext.Column,
				}},
			})
		} else {
			if !reResourceID.MatchString(q.Uid) {
				entries = append(entries, Entry{
					RuleID:  BundleInvalidUidRuleID, // Shared Rule ID
					Message: fmt.Sprintf("%s UID does not meet the requirements", queryIdentifier(q, isGlobal)),
					Level:   LevelError,
					Location: []Location{{
						File:   ctx.FilePath,
						Line:   q.FileContext.Line,
						Column: q.FileContext.Column,
					}},
				})
			}
			// Check for uniqueness among global queries defined in this file
			if _, exists := ctx.GlobalQueryUidsInFile[q.Uid]; exists {
				entries = append(entries, Entry{
					RuleID:  QueryUidUniqueRuleID,
					Message: fmt.Sprintf("Global query UID '%s' is used multiple times in the same file", q.Uid),
					Level:   LevelError,
					Location: []Location{{
						File:   ctx.FilePath,
						Line:   q.FileContext.Line,
						Column: q.FileContext.Column,
					}},
				})
			} else {
				ctx.GlobalQueryUidsInFile[q.Uid] = struct{}{}
			}
			// The original check `globalQueriesUids[uid] > 1` was for the entire bundle (potentially multiple files).
			// This is now implicitly handled by the bundle compilation check at the end of `Lint`.
			// If a more specific error message for this case is needed before compilation,
			// `ctx.GlobalQueriesUids` (populated from all files before individual checks) could be used.
			// For now, file-level uniqueness is checked here.
		}
	}
	// For embedded queries, UID is not strictly required unless it's only a reference.
	// If it's an embedded definition (has MQL/Title/Variants), UID is optional.
	return entries
}

func runCheckQueryTitle(ctx *LintContext, item interface{}) []Entry {
	input, ok := item.(QueryLintInput)
	if !ok {
		return nil
	}
	q := input.Query
	_, isVariant := ctx.VariantMapping[q.Uid]

	if q.Title == "" && !isVariant { // Title required for non-variants
		// Also, if it's an embedded query that's just a reference (no MQL, no Variants), it doesn't need a title.
		// This check is primarily for query definitions.
		if input.IsGlobal || isQueryDefinitionComplete(q) {
			return []Entry{{
				RuleID:  QueryTitleRuleID,
				Message: fmt.Sprintf("%s does not define a title", queryIdentifier(q, input.IsGlobal)),
				Level:   LevelError,
				Location: []Location{{
					File:   ctx.FilePath,
					Line:   q.FileContext.Line,
					Column: q.FileContext.Column,
				}},
			}}
		}
	}
	return nil
}

func runCheckQueryVariantFields(ctx *LintContext, item interface{}) []Entry {
	input, ok := item.(QueryLintInput)
	if !ok {
		return nil
	}
	q := input.Query
	var entries []Entry

	_, isVariant := ctx.VariantMapping[q.Uid]
	if !isVariant {
		return nil // This check only applies to variants
	}

	// Variant checks
	if q.Impact != nil {
		entries = append(entries, Entry{
			RuleID:   QueryVariantUsesNonDefaultFieldsRuleID,
			Message:  fmt.Sprintf("Query variant '%s' must not define 'impact'", q.Uid),
			Level:    LevelError,
			Location: []Location{{File: ctx.FilePath, Line: q.FileContext.Line, Column: q.FileContext.Column}},
		})
	}
	if q.Title != "" {
		entries = append(entries, Entry{
			RuleID:   QueryVariantUsesNonDefaultFieldsRuleID,
			Message:  fmt.Sprintf("Query variant '%s' must not define 'title'", q.Uid),
			Level:    LevelError,
			Location: []Location{{File: ctx.FilePath, Line: q.FileContext.Line, Column: q.FileContext.Column}},
		})
	}
	if len(q.Tags) > 0 {
		entries = append(entries, Entry{
			RuleID:   QueryVariantUsesNonDefaultFieldsRuleID,
			Message:  fmt.Sprintf("Query variant '%s' must not define 'tags'", q.Uid),
			Level:    LevelError,
			Location: []Location{{File: ctx.FilePath, Line: q.FileContext.Line, Column: q.FileContext.Column}},
		})
	}
	if len(q.Variants) > 0 {
		entries = append(entries, Entry{
			RuleID:   QueryVariantUsesNonDefaultFieldsRuleID,
			Message:  fmt.Sprintf("Query variant '%s' must not define nested 'variants'", q.Uid),
			Level:    LevelError,
			Location: []Location{{File: ctx.FilePath, Line: q.FileContext.Line, Column: q.FileContext.Column}},
		})
	}
	return entries
}

func runCheckQueryMQLPresence(ctx *LintContext, item interface{}) []Entry {
	input, ok := item.(QueryLintInput)
	if !ok {
		return nil
	}
	q := input.Query
	_, isVariant := ctx.VariantMapping[q.Uid]

	if isVariant { // Variants must have MQL
		if q.Mql == "" {
			return []Entry{{
				RuleID:   QueryMissingMQLRuleID,
				Message:  fmt.Sprintf("Query variant '%s' must define MQL", q.Uid),
				Level:    LevelError,
				Location: []Location{{File: ctx.FilePath, Line: q.FileContext.Line, Column: q.FileContext.Column}},
			}}
		}
	} else { // Non-variants (parents)
		if len(q.Variants) == 0 && q.Mql == "" {
			// If it's a parent query that itself has no variants, it must have MQL.
			// This applies to global queries and fully defined embedded queries.
			if input.IsGlobal || isQueryDefinitionComplete(q) {
				return []Entry{{
					RuleID:   QueryMissingMQLRuleID,
					Message:  fmt.Sprintf("%s has no variants and must define MQL", queryIdentifier(q, input.IsGlobal)),
					Level:    LevelError,
					Location: []Location{{File: ctx.FilePath, Line: q.FileContext.Line, Column: q.FileContext.Column}},
				}}
			}
		}
	}
	return nil
}

func runCheckQueryUnassigned(ctx *LintContext, item interface{}) []Entry {
	input, ok := item.(QueryLintInput)
	if !ok {
		return nil
	}
	q := input.Query

	// This check only applies to globally defined queries that are not variants themselves.
	_, isVariant := ctx.VariantMapping[q.Uid]
	if !input.IsGlobal || q.Uid == "" || isVariant {
		return nil
	}

	if _, isAssigned := ctx.AssignedQueryUIDs[q.Uid]; !isAssigned {
		return []Entry{{
			RuleID:  QueryUnassignedRuleID,
			Message: fmt.Sprintf("Global query UID '%s' is defined but not assigned to any policy", q.Uid),
			Level:   LevelWarning,
			Location: []Location{{
				File:   ctx.FilePath,
				Line:   q.FileContext.Line,
				Column: q.FileContext.Column,
			}},
		}}
	}
	return nil
}

func runCheckQueryUsageConsistency(ctx *LintContext, item interface{}) []Entry {
	input, ok := item.(QueryLintInput)
	if !ok {
		return nil
	}
	q := input.Query

	// This check applies to any query UID that might appear in policies.
	// It doesn't matter if it's global or embedded, as long as it has a UID.
	if q.Uid == "" {
		return nil
	}

	_, usedAsCheck := ctx.QueryUsageAsCheck[q.Uid]
	_, usedAsData := ctx.QueryUsageAsData[q.Uid]

	if usedAsCheck && usedAsData {
		// Find a relevant line number. This is tricky as the usage is in policies.
		// The query's own definition line is a fallback.
		return []Entry{{
			RuleID:  QueryUsedAsDifferentTypesRuleID,
			Message: fmt.Sprintf("Query UID '%s' is used as both a check and a data query in policies", q.Uid),
			Level:   LevelError,
			Location: []Location{{ // Location of the query definition
				File: ctx.FilePath, // This might be misleading if the query is defined in one file and used in another.
				// The context would need to track usage locations. For now, query def location.
				Line:   q.FileContext.Line,
				Column: q.FileContext.Column,
			}},
		}}
	}
	return nil
}
