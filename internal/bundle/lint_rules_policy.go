// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package bundle

import (
	"fmt"
	"regexp"

	"github.com/Masterminds/semver"
)

// reResourceID: lowercase letters, digits, dots or hyphens, fewer than 200 chars, more than 5 chars
var (
	policyUid = regexp.MustCompile(`^([\d-_]|[a-z]){4,100}$`)
	queryUid  = regexp.MustCompile(`^([\d-_\.]|[a-zA-Z]){5,200}$`)
)

// LintContext provides shared information and state to lint check functions.
type LintContext struct {
	FilePath              string
	PolicyBundle          *Bundle
	GlobalQueriesUids     map[string]int      // Counts occurrences of global query UIDs
	GlobalQueriesByUid    map[string]*Mquery  // Maps global query UID to Mquery object
	PolicyUidsInFile      map[string]struct{} // Tracks policy UIDs defined in the current file for uniqueness
	GlobalQueryUidsInFile map[string]struct{} // Tracks global query UIDs defined in the current file for uniqueness
	AssignedQueryUIDs     map[string]struct{} // UIDs of queries (and their variants) assigned in policies
	QueryUsageAsCheck     map[string]struct{} // UIDs of queries used as checks
	QueryUsageAsData      map[string]struct{} // UIDs of queries used as data queries
	VariantMapping        map[string]string   // Maps variant child UID to parent query UID
}

// LintRule defines the structure for a single linting rule.
// The item any will be cast to *Policy or QueryLintInput.
type LintRule struct {
	ID          string
	Name        string
	Description string
	Severity    string
	Run         func(ctx *LintContext, item any) []*Entry
}

// QueryLintInput is used to pass a query and its context (global or embedded) to check functions.
type QueryLintInput struct {
	Query    *Mquery
	IsGlobal bool // True if the query is from the top-level `queries:` block
	// IsEmbedded bool // True if the query is defined inline within a policy group (can be inferred if !IsGlobal and has MQL/Title/Variants)
}

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
	PolicyMissingRequireRuleID       = "policy-missing-require"
)

// GetPolicyLintRules returns a list of lint checks for policies.
func GetPolicyLintRules() []LintRule {
	return []LintRule{
		{
			ID:          PolicyUidRuleID,
			Name:        "Policy UID Presence and Format",
			Description: "Checks if a policy has a UID and if it conforms to naming standards. Also checks for uniqueness within the file.",
			Severity:    LevelError,
			Run:         runRulePolicyUid,
		},
		{
			ID:          PolicyNameRuleID,
			Name:        "Policy Name Presence",
			Description: "Ensures every policy has a `name` field.",
			Severity:    LevelError,
			Run:         runRulePolicyName,
		},
		{
			ID:          PolicyRequiredTagsMissingRuleID,
			Name:        "Policy Required Tags",
			Description: "Ensures policies have required tags like 'mondoo.com/category' and 'mondoo.com/platform'.",
			Severity:    LevelWarning,
			Run:         runRulePolicyRequiredTags,
		},
		{
			ID:          PolicyMissingVersionRuleID,
			Name:        "Policy Version Presence",
			Description: "Ensures every policy has a `version` field.",
			Severity:    LevelError,
			Run:         runRulePolicyMissingVersion,
		},
		{
			ID:          PolicyWrongVersionRuleID,
			Name:        "Policy Version Format",
			Description: "Ensures policy versions follow semantic versioning (semver).",
			Severity:    LevelError,
			Run:         runRulePolicyWrongVersion,
		},
		{
			ID:          PolicyMissingChecksRuleID, // Covers empty groups and empty checks/queries in groups
			Name:        "Policy Missing Checks or Groups",
			Description: "Ensures policies have defined groups, and groups have checks or queries.",
			Severity:    LevelError,
			Run:         runRulePolicyGroupsAndChecks,
		},
		{
			ID:          PolicyMissingAssetFilterRuleID,
			Name:        "Policy Group Missing Asset Filter",
			Description: "Warns if a policy group or its checks lack asset filters or variants.",
			Severity:    LevelWarning, // Original was warning
			Run:         runRulePolicyGroupAssetFilter,
		},
		{
			ID:          PolicyMissingAssignedQueryRuleID,
			Name:        "Policy Assigned Query Existence",
			Description: "Ensures that queries assigned in policy groups exist globally or are valid embedded queries.",
			Severity:    LevelError,
			Run:         runRulePolicyAssignedQueriesExist,
		},
		{
			ID:          PolicyMissingRequireRuleID,
			Name:        "Policy Require Providers",
			Description: "Ensures that policies define required providers.",
			Severity:    LevelWarning,
			Run:         runRulePolicyRequireExist,
		},
	}
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

func runRulePolicyUid(ctx *LintContext, item any) []*Entry {
	p, ok := item.(*Policy)
	if !ok {
		return nil
	}
	var entries []*Entry

	if p.Uid == "" {
		entries = append(entries, &Entry{
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
		if !policyUid.MatchString(p.Uid) {
			entries = append(entries, &Entry{
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
			entries = append(entries, &Entry{
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

func runRulePolicyName(ctx *LintContext, item any) []*Entry {
	p, ok := item.(*Policy)
	if !ok {
		return nil
	}
	if p.Name == "" {
		return []*Entry{{
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

func runRulePolicyRequiredTags(ctx *LintContext, item any) []*Entry {
	p, ok := item.(*Policy)
	if !ok {
		return nil
	}
	var entries []*Entry
	requiredTags := []string{"mondoo.com/category", "mondoo.com/platform"}
	for _, tagKey := range requiredTags {
		if _, exists := p.Tags[tagKey]; !exists {
			entries = append(entries, &Entry{
				RuleID:  PolicyRequiredTagsMissingRuleID,
				Message: fmt.Sprintf("%s does not contain the required tag `%s`", policyIdentifier(p), tagKey),
				Level:   LevelWarning,
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

func runRulePolicyMissingVersion(ctx *LintContext, item any) []*Entry {
	p, ok := item.(*Policy)
	if !ok {
		return nil
	}
	if p.Version == "" {
		return []*Entry{{
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

func runRulePolicyWrongVersion(ctx *LintContext, item any) []*Entry {
	p, ok := item.(*Policy)
	if !ok {
		return nil
	}
	if p.Version != "" { // Only check if version is present
		_, err := semver.NewVersion(p.Version)
		if err != nil {
			return []*Entry{{
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

func runRulePolicyGroupsAndChecks(ctx *LintContext, item any) []*Entry {
	p, ok := item.(*Policy)
	if !ok {
		return nil
	}
	var entries []*Entry
	if len(p.Groups) == 0 {
		entries = append(entries, &Entry{
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
			entries = append(entries, &Entry{
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

func runRulePolicyGroupAssetFilter(ctx *LintContext, item any) []*Entry {
	p, ok := item.(*Policy)
	if !ok {
		return nil
	}
	var entries []*Entry
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

			entries = append(entries, &Entry{
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

func runRulePolicyAssignedQueriesExist(ctx *LintContext, item any) []*Entry {
	p, ok := item.(*Policy)
	if !ok {
		return nil
	}
	var entries []*Entry

	for _, group := range p.Groups {
		for _, checkRef := range group.Checks { // checkRef is an Mquery struct (either a ref or embedded)
			if !isQueryDefinitionComplete(checkRef) && checkRef.Uid != "" { // It's a reference
				if _, exists := ctx.GlobalQueriesByUid[checkRef.Uid]; !exists {
					entries = append(entries, &Entry{
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
					entries = append(entries, &Entry{
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

func runRulePolicyRequireExist(ctx *LintContext, item any) []*Entry {
	p, ok := item.(*Policy)
	if !ok {
		return nil
	}
	var entries []*Entry
	if len(p.Require) == 0 {
		entries = append(entries, &Entry{
			RuleID:  PolicyMissingRequireRuleID,
			Message: fmt.Sprintf("%s does not define any required providers", policyIdentifier(p)),
			Level:   LevelWarning,
			Location: []Location{{
				File:   ctx.FilePath,
				Line:   p.FileContext.Line,
				Column: p.FileContext.Column,
			}},
		})
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
