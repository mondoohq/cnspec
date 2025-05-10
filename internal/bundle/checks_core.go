// internal/bundle/checks_core.go
// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package bundle

import (
	"regexp"
)

const (
	LevelError   = "error"
	LevelWarning = "warning"
)

// reResourceID: lowercase letters, digits, dots or hyphens, fewer than 200 chars, more than 5 chars
var reResourceID = regexp.MustCompile(`^([\d-_\.]|[a-zA-Z]){5,200}$`)

// Entry represents a single linting issue found.
type Entry struct {
	RuleID   string
	Level    string
	Message  string
	Location []Location
}

// Location specifies the file, line, and column of a linting issue.
type Location struct {
	File   string
	Line   int
	Column int
}

// Results holds all linting entries for a bundle.
type Results struct {
	BundleLocations []string
	Entries         []Entry
}

// HasError checks if there are any error-level entries.
func (r *Results) HasError() bool {
	for i := range r.Entries {
		if r.Entries[i].Level == LevelError {
			return true
		}
	}
	return false
}

// HasWarning checks if there are any warning-level entries.
func (r *Results) HasWarning() bool {
	for i := range r.Entries {
		if r.Entries[i].Level == LevelWarning {
			return true
		}
	}
	return false
}

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

// LintCheck defines the structure for a single linting rule.
// The item interface{} will be cast to *Policy or QueryLintInput.
type LintCheck struct {
	ID          string
	Name        string
	Description string
	Severity    string
	Run         func(ctx *LintContext, item interface{}) []Entry
}

// QueryLintInput is used to pass a query and its context (global or embedded) to check functions.
type QueryLintInput struct {
	Query    *Mquery
	IsGlobal bool // True if the query is from the top-level `queries:` block
	// IsEmbedded bool // True if the query is defined inline within a policy group (can be inferred if !IsGlobal and has MQL/Title/Variants)
}

var (
	policyChecks []LintCheck
	queryChecks  []LintCheck
)

// RegisterPolicyCheck adds a policy check to the registry.
func RegisterPolicyCheck(check LintCheck) {
	policyChecks = append(policyChecks, check)
}

// RegisterQueryCheck adds a query check to the registry.
func RegisterQueryCheck(check LintCheck) {
	queryChecks = append(queryChecks, check)
}

// Rule is used for SARIF output and listing available rules.
// This structure matches the original LinterRules.
type Rule struct {
	ID          string
	Name        string
	Description string
}

// AllLinterRules dynamically generates the list of all linter rules
// from registered checks and adds bundle-specific static rules.
func AllLinterRules() []Rule {
	var rules []Rule

	// Add rules from registered policy checks
	for _, check := range policyChecks {
		rules = append(rules, Rule{
			ID:          check.ID,
			Name:        check.Name,
			Description: check.Description,
		})
	}

	// Add rules from registered query checks
	for _, check := range queryChecks {
		rules = append(rules, Rule{
			ID:          check.ID,
			Name:        check.Name,
			Description: check.Description,
		})
	}

	// Add bundle-level static rules (those not tied to a specific policy/query item)
	// These were previously in LinterRules and are typically checked outside the item-loop.
	rules = append(rules, Rule{
		ID:          BundleCompileErrorRuleID, // Define these constants
		Name:        "MQL compile error",
		Description: "Could not compile the MQL bundle",
	})
	rules = append(rules, Rule{
		ID:          BundleInvalidRuleID,
		Name:        "Invalid bundle",
		Description: "The bundle is not properly YAML formatted",
	})
	rules = append(rules, Rule{
		ID:          BundleUnknownFieldRuleID,
		Name:        "Bundle unknown field",
		Description: "The bundle YAML contains fields not defined in the schema",
	})
	// Note: bundleInvalidUidRuleID is covered by specific policy/query UID checks.

	return rules
}

// Constants for Rule IDs that are checked at the bundle level, not per-item.
const (
	BundleCompileErrorRuleID = "bundle-compile-error"
	BundleInvalidRuleID      = "bundle-invalid"
	BundleUnknownFieldRuleID = "bundle-unknown-field"
	BundleInvalidUidRuleID   = "bundle-invalid-uid" // Shared by policy/query checks
)
