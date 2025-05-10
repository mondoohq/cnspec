// internal/bundle/policy_checks.go
// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package bundle

import (
	"fmt"

	"github.com/Masterminds/semver"
)

// Policy Rule ID Constants
const (
	PolicyUidRuleID                  = "policy-uid"
	PolicyNameRuleID                 = "policy-name"
	PolicyUidUniqueRuleID            = "policy-uid-unique"
	PolicyMissingAssetFilterRuleID   = "policy-missing-asset-filter"
	PolicyMissingAssignedQueryRuleID = "policy-missing-assigned-query"
	PolicyMissingChecksRuleID        = "policy-missing-checks"
	PolicyMissingVersionRuleID       = "policy-missing-version"
	PolicyWrongVersionRuleID         = "policy-wrong-version"
	PolicyRequiredTagsMissingRuleID  = "policy-required-tags-missing"
)

func init() {
	RegisterPolicyCheck(LintCheck{
		ID:          PolicyUidRuleID,
		Name:        "Policy UID Presence and Format",
		Description: "Checks if a policy has a UID and if it conforms to naming standards. Also checks for uniqueness within the file.",
		Severity:    LevelError,
		Run:         runCheckPolicyUid,
	})
	RegisterPolicyCheck(LintCheck{
		ID:          PolicyNameRuleID,
		Name:        "Policy Name Presence",
		Description: "Ensures every policy has a `name` field.",
		Severity:    LevelError,
		Run:         runCheckPolicyName,
	})
	RegisterPolicyCheck(LintCheck{
		ID:          PolicyRequiredTagsMissingRuleID,
		Name:        "Policy Required Tags",
		Description: "Ensures policies have required tags like 'mondoo.com/category' and 'mondoo.com/platform'.",
		Severity:    LevelError,
		Run:         runCheckPolicyRequiredTags,
	})
	RegisterPolicyCheck(LintCheck{
		ID:          PolicyMissingVersionRuleID,
		Name:        "Policy Version Presence",
		Description: "Ensures every policy has a `version` field.",
		Severity:    LevelError,
		Run:         runCheckPolicyMissingVersion,
	})
	RegisterPolicyCheck(LintCheck{
		ID:          PolicyWrongVersionRuleID,
		Name:        "Policy Version Format",
		Description: "Ensures policy versions follow semantic versioning (semver).",
		Severity:    LevelError,
		Run:         runCheckPolicyWrongVersion,
	})
	RegisterPolicyCheck(LintCheck{
		ID:          PolicyMissingChecksRuleID, // Covers empty groups and empty checks/queries in groups
		Name:        "Policy Missing Checks or Groups",
		Description: "Ensures policies have defined groups, and groups have checks or queries.",
		Severity:    LevelError,
		Run:         runCheckPolicyGroupsAndChecks,
	})
	RegisterPolicyCheck(LintCheck{
		ID:          PolicyMissingAssetFilterRuleID,
		Name:        "Policy Group Missing Asset Filter",
		Description: "Warns if a policy group or its checks lack asset filters or variants.",
		Severity:    LevelWarning, // Original was warning
		Run:         runCheckPolicyGroupAssetFilter,
	})
	RegisterPolicyCheck(LintCheck{
		ID:          PolicyMissingAssignedQueryRuleID,
		Name:        "Policy Assigned Query Existence",
		Description: "Ensures that queries assigned in policy groups exist globally or are valid embedded queries.",
		Severity:    LevelError,
		Run:         runCheckPolicyAssignedQueriesExist,
	})
}

func policyIdentifier(p *Policy) string {
	if p.Uid != "" {
		return fmt.Sprintf("policy '%s'", p.Uid)
	}
	if p.Name != "" {
		return fmt.Sprintf("policy '%s' (at line %d)", p.Name, p.FileContext.Line)
	}
	return fmt.Sprintf("policy at line %d", p.FileContext.Line)
}

func runCheckPolicyUid(ctx *LintContext, item interface{}) []Entry {
	p, ok := item.(*Policy)
	if !ok {
		return nil
	}
	var entries []Entry

	if p.Uid == "" {
		entries = append(entries, Entry{
			RuleID:  PolicyUidRuleID,
			Message: fmt.Sprintf("%s does not define a UID", policyIdentifier(p)),
			Level:   LevelError,
			Location: []Location{{
				File:   ctx.FilePath,
				Line:   p.FileContext.Line,
				Column: p.FileContext.Column,
			}},
		})
	} else {
		if !reResourceID.MatchString(p.Uid) {
			entries = append(entries, Entry{
				RuleID:  BundleInvalidUidRuleID,
				Message: fmt.Sprintf("%s UID does not meet the requirements", policyIdentifier(p)),
				Level:   LevelError,
				Location: []Location{{
					File:   ctx.FilePath,
					Line:   p.FileContext.Line,
					Column: p.FileContext.Column,
				}},
			})
		}
		if _, exists := ctx.PolicyUidsInFile[p.Uid]; exists {
			entries = append(entries, Entry{
				RuleID:  PolicyUidUniqueRuleID,
				Message: fmt.Sprintf("Policy UID '%s' is used multiple times in the same file", p.Uid),
				Level:   LevelError,
				Location: []Location{{
					File:   ctx.FilePath,
					Line:   p.FileContext.Line,
					Column: p.FileContext.Column,
				}},
			})
		} else {
			ctx.PolicyUidsInFile[p.Uid] = struct{}{}
		}
	}
	return entries
}

func runCheckPolicyName(ctx *LintContext, item interface{}) []Entry {
	p, ok := item.(*Policy)
	if !ok {
		return nil
	}
	if p.Name == "" {
		return []Entry{{
			RuleID:  PolicyNameRuleID,
			Message: fmt.Sprintf("%s does not define a name", policyIdentifier(p)),
			Level:   LevelError,
			Location: []Location{{
				File:   ctx.FilePath,
				Line:   p.FileContext.Line,
				Column: p.FileContext.Column,
			}},
		}}
	}
	return nil
}

func runCheckPolicyRequiredTags(ctx *LintContext, item interface{}) []Entry {
	p, ok := item.(*Policy)
	if !ok {
		return nil
	}
	var entries []Entry
	requiredTags := []string{"mondoo.com/category", "mondoo.com/platform"}
	for _, tagKey := range requiredTags {
		if _, exists := p.Tags[tagKey]; !exists {
			entries = append(entries, Entry{
				RuleID:  PolicyRequiredTagsMissingRuleID,
				Message: fmt.Sprintf("%s does not contain the required tag `%s`", policyIdentifier(p), tagKey),
				Level:   LevelError,
				Location: []Location{{
					File:   ctx.FilePath,
					Line:   p.FileContext.Line,
					Column: p.FileContext.Column,
				}},
			})
		}
	}
	return entries
}

func runCheckPolicyMissingVersion(ctx *LintContext, item interface{}) []Entry {
	p, ok := item.(*Policy)
	if !ok {
		return nil
	}
	if p.Version == "" {
		return []Entry{{
			RuleID:  PolicyMissingVersionRuleID,
			Message: fmt.Sprintf("%s is missing version", policyIdentifier(p)),
			Level:   LevelError,
			Location: []Location{{
				File:   ctx.FilePath,
				Line:   p.FileContext.Line,
				Column: p.FileContext.Column,
			}},
		}}
	}
	return nil
}

func runCheckPolicyWrongVersion(ctx *LintContext, item interface{}) []Entry {
	p, ok := item.(*Policy)
	if !ok {
		return nil
	}
	if p.Version != "" { // Only check if version is present
		_, err := semver.NewVersion(p.Version)
		if err != nil {
			return []Entry{{
				RuleID:  PolicyWrongVersionRuleID,
				Message: fmt.Sprintf("%s has invalid version '%s': %s", policyIdentifier(p), p.Version, err.Error()),
				Level:   LevelError,
				Location: []Location{{
					File:   ctx.FilePath,
					Line:   p.FileContext.Line, // Or version specific line
					Column: p.FileContext.Column,
				}},
			}}
		}
	}
	return nil
}

func runCheckPolicyGroupsAndChecks(ctx *LintContext, item interface{}) []Entry {
	p, ok := item.(*Policy)
	if !ok {
		return nil
	}
	var entries []Entry
	if len(p.Groups) == 0 {
		entries = append(entries, Entry{
			RuleID:  PolicyMissingChecksRuleID, // Using this ID for missing groups too
			Message: fmt.Sprintf("%s has no groups defined", policyIdentifier(p)),
			Level:   LevelError,
			Location: []Location{{
				File:   ctx.FilePath,
				Line:   p.FileContext.Line,
				Column: p.FileContext.Column,
			}},
		})
		return entries // No further group checks if no groups
	}

	for _, group := range p.Groups {
		if len(group.Checks) == 0 && len(group.Queries) == 0 && len(group.Policies) == 0 {
			entries = append(entries, Entry{
				RuleID: PolicyMissingChecksRuleID,
				Message: fmt.Sprintf("%s, group '%s' (line %d) has no checks, data queries, or sub-policies defined",
					policyIdentifier(p), group.Title, group.FileContext.Line),
				Level: LevelError,
				Location: []Location{{
					File:   ctx.FilePath,
					Line:   group.FileContext.Line,
					Column: group.FileContext.Column,
				}},
			})
		}
	}
	return entries
}

func runCheckPolicyGroupAssetFilter(ctx *LintContext, item interface{}) []Entry {
	p, ok := item.(*Policy)
	if !ok {
		return nil
	}
	var entries []Entry
	for _, group := range p.Groups {
		groupHasFilter := group.Filters != nil && len(group.Filters.Items) > 0
		if groupHasFilter {
			continue
		}

		// If group has no filter, all its checks must have filters or variants
		for _, checkRef := range group.Checks {
			// Check 1: Embedded query definition
			if isQueryDefinitionComplete(checkRef) { // checkRef is an Mquery object
				if hasVariantsOrFilters(checkRef) {
					continue
				}
			} else {
				// Check 2: Referenced global query
				globalQuery, exists := ctx.GlobalQueriesByUid[checkRef.Uid]
				if exists && hasVariantsOrFilters(globalQuery) {
					continue
				}
			}

			// If neither the embedded def nor the global ref has filters/variants
			location := Location{
				File:   ctx.FilePath,
				Line:   group.FileContext.Line, // Default to group location
				Column: group.FileContext.Column,
			}
			// Try to get more specific location if possible (e.g., checkRef.FileContext)
			// For now, group location is a reasonable approximation.

			entries = append(entries, Entry{
				RuleID: PolicyMissingAssetFilterRuleID,
				Message: fmt.Sprintf("%s, group '%s' (line %d): Check '%s' lacks an asset filter or variants, and the group also has no filter.",
					policyIdentifier(p), group.Title, group.FileContext.Line, queryRefIdentifier(checkRef)),
				Level:    LevelWarning,
				Location: []Location{location},
			})
		}
	}
	return entries
}

func runCheckPolicyAssignedQueriesExist(ctx *LintContext, item interface{}) []Entry {
	p, ok := item.(*Policy)
	if !ok {
		return nil
	}
	var entries []Entry

	for _, group := range p.Groups {
		for _, checkRef := range group.Checks { // checkRef is an Mquery struct (either a ref or embedded)
			if !isQueryDefinitionComplete(checkRef) && checkRef.Uid != "" { // It's a reference
				if _, exists := ctx.GlobalQueriesByUid[checkRef.Uid]; !exists {
					entries = append(entries, Entry{
						RuleID: PolicyMissingAssignedQueryRuleID,
						Message: fmt.Sprintf("%s, group '%s': Assigned check query UID '%s' does not exist as a global query.",
							policyIdentifier(p), group.Title, checkRef.Uid),
						Level: LevelError,
						Location: []Location{{ // Location of the check reference within the group
							File:   ctx.FilePath,
							Line:   checkRef.FileContext.Line, // checkRef is the Mquery object in the group.checks list
							Column: checkRef.FileContext.Column,
						}},
					})
				}
			}
		}
		for _, queryRef := range group.Queries { // data queries
			if !isQueryDefinitionComplete(queryRef) && queryRef.Uid != "" { // It's a reference
				if _, exists := ctx.GlobalQueriesByUid[queryRef.Uid]; !exists {
					entries = append(entries, Entry{
						RuleID: PolicyMissingAssignedQueryRuleID,
						Message: fmt.Sprintf("%s, group '%s': Assigned data query UID '%s' does not exist as a global query.",
							policyIdentifier(p), group.Title, queryRef.Uid),
						Level: LevelError,
						Location: []Location{{
							File:   ctx.FilePath,
							Line:   queryRef.FileContext.Line,
							Column: queryRef.FileContext.Column,
						}},
					})
				}
			}
		}
	}
	return entries
}

// Helper to identify a query reference (UID or line number)
func queryRefIdentifier(q *Mquery) string {
	if q.Uid != "" {
		return q.Uid
	}
	return fmt.Sprintf("query at line %d", q.FileContext.Line)
}

// Helper to determine if an Mquery struct in a policy group is a full definition or just a reference.
// A full definition would have MQL, Title, or Variants.
func isQueryDefinitionComplete(q *Mquery) bool {
	return q.Mql != "" || q.Title != "" || len(q.Variants) > 0 || (q.Docs != nil && q.Docs.Desc != "")
}

// Helper: (similar to original)
func hasVariantsOrFilters(q *Mquery) bool {
	if q == nil {
		return false
	}
	if len(q.Variants) > 0 {
		return true
	}
	if q.Filters != nil && len(q.Filters.Items) > 0 {
		return true
	}
	return false
}
