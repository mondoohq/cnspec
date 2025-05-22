// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package bundle

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"go.mondoo.com/cnquery/v11/providers-sdk/v1/resources"
	"go.mondoo.com/cnspec/v11/policy"
	k8sYaml "sigs.k8s.io/yaml"
)

// Constants for Rule IDs that are checked at the bundle level, not per-item.
const (
	BundleCompileErrorRuleID = "bundle-compile-error"
	BundleInvalidRuleID      = "bundle-invalid"
	BundleUnknownFieldRuleID = "bundle-unknown-field"
	BundleInvalidUidRuleID   = "bundle-invalid-uid" // Shared by policy/query checks
)

const (
	LevelError   = "error"
	LevelWarning = "warning"
)

// Lint validates a policy bundle for consistency
func Lint(schema resources.ResourcesSchema, files ...string) (*Results, error) {
	aggregatedResults := &Results{
		BundleLocations: []string{},
		Entries:         []Entry{},
	}

	policyChecks := GetPolicyLintChecks()
	queryChecks := GetQueryLintChecks()

	var absFiles []string
	parsedBundles := make(map[string]*Bundle)

	// First pass: Parse all files and collect initial metadata for context
	for _, file := range files {
		absPath, err := filepath.Abs(file)
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute path for %s: %w", file, err)
		}
		absFiles = append(absFiles, absPath)
		aggregatedResults.BundleLocations = append(aggregatedResults.BundleLocations, absPath)

		data, err := os.ReadFile(absPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", absPath, err)
		}

		policyBundle, err := ParseYaml(data)
		if err != nil {
			aggregatedResults.Entries = append(aggregatedResults.Entries, Entry{
				RuleID:  BundleInvalidRuleID,
				Message: fmt.Sprintf("Cannot parse YAML file %s: %s", filepath.Base(absPath), err.Error()),
				Level:   LevelError,
				Location: []Location{{
					File:   absPath,
					Line:   1,
					Column: 1,
				}},
			})
			continue
		}
		parsedBundles[absPath] = policyBundle

		// Check for unknown fields (UnmarshalStrict)
		strictCheckBundle := &policy.Bundle{}
		if err := k8sYaml.UnmarshalStrict(data, strictCheckBundle); err != nil {
			aggregatedResults.Entries = append(aggregatedResults.Entries, Entry{
				RuleID:  BundleUnknownFieldRuleID,
				Message: fmt.Sprintf("Bundle file %s contains unknown fields: %s", filepath.Base(absPath), err.Error()),
				Level:   LevelError,
				Location: []Location{{
					File:   absPath,
					Line:   1,
					Column: 1,
				}},
			})
		}
	}

	// Initialize LintContext with data from ALL parsed bundles for cross-file context
	// This context will be shared for checks that need to know about the whole bundle.
	// However, some checks are file-specific (like UID uniqueness within a file).
	// For simplicity, we'll build a context per file, but some parts of the context
	// (like GlobalQueriesByUid) could be built from all files if needed for more complex checks.
	// The original code's `globalQueriesUids` was effectively bundle-wide for uniqueness.

	// For now, let's process file by file, building context as we go.
	// More complex global context can be added if specific checks require it.
	// The current structure of checks (e.g. UID uniqueness) is per-file.

	for _, absPath := range absFiles {
		policyBundle := parsedBundles[absPath]

		lintCtx := &LintContext{
			FilePath:              absPath,
			PolicyBundle:          policyBundle,
			GlobalQueriesUids:     make(map[string]int),
			GlobalQueriesByUid:    make(map[string]*Mquery),
			PolicyUidsInFile:      make(map[string]struct{}),
			GlobalQueryUidsInFile: make(map[string]struct{}),
			AssignedQueryUIDs:     make(map[string]struct{}),
			QueryUsageAsCheck:     make(map[string]struct{}),
			QueryUsageAsData:      make(map[string]struct{}),
			VariantMapping:        make(map[string]string),
		}

		// Populate context: global queries and variants from the current file
		for _, q := range policyBundle.Queries {
			lintCtx.GlobalQueriesUids[q.Uid]++
			lintCtx.GlobalQueriesByUid[q.Uid] = q
			for _, variantRef := range q.Variants {
				lintCtx.VariantMapping[variantRef.Uid] = q.Uid
			}
		}
		// Populate context: variants defined in embedded queries within policies
		for _, p := range policyBundle.Policies {
			for _, pg := range p.Groups {
				for _, checkQuery := range pg.Checks {
					for _, variantRef := range checkQuery.Variants {
						lintCtx.VariantMapping[variantRef.Uid] = checkQuery.Uid
					}
				}
				for _, dataQuery := range pg.Queries {
					for _, variantRef := range dataQuery.Variants {
						lintCtx.VariantMapping[variantRef.Uid] = dataQuery.Uid
					}
				}
			}
		}

		// Populate context: AssignedQueryUIDs, QueryUsageAsCheck, QueryUsageAsData
		// This needs to iterate through policies.
		for _, p := range policyBundle.Policies {
			for _, group := range p.Groups {
				for _, checkRef := range group.Checks {
					lintCtx.AssignedQueryUIDs[checkRef.Uid] = struct{}{}
					lintCtx.QueryUsageAsCheck[checkRef.Uid] = struct{}{}
					for _, v := range checkRef.Variants { // Also mark variants as assigned
						lintCtx.AssignedQueryUIDs[v.Uid] = struct{}{}
						// Variants inherit usage type from parent
						lintCtx.QueryUsageAsCheck[v.Uid] = struct{}{}
					}
				}
				for _, queryRef := range group.Queries {
					lintCtx.AssignedQueryUIDs[queryRef.Uid] = struct{}{}
					lintCtx.QueryUsageAsData[queryRef.Uid] = struct{}{}
					for _, v := range queryRef.Variants {
						lintCtx.AssignedQueryUIDs[v.Uid] = struct{}{}
						lintCtx.QueryUsageAsData[v.Uid] = struct{}{}
					}
				}
			}
		}

		// Run Policy Checks
		for _, p := range policyBundle.Policies {
			for _, check := range policyChecks {
				entries := check.Run(lintCtx, p)
				aggregatedResults.Entries = append(aggregatedResults.Entries, entries...)
			}
		}

		// Run Query Checks for Global Queries
		for _, q := range policyBundle.Queries {
			for _, check := range queryChecks {
				entries := check.Run(lintCtx, QueryLintInput{Query: q, IsGlobal: true})
				aggregatedResults.Entries = append(aggregatedResults.Entries, entries...)
			}
		}

		// Run Query Checks for Embedded Queries in Policies
		for _, p := range policyBundle.Policies {
			for _, group := range p.Groups {
				for _, checkQuery := range group.Checks {
					if isQueryDefinitionComplete(checkQuery) {
						for _, check := range queryChecks {
							entries := check.Run(lintCtx, QueryLintInput{Query: checkQuery, IsGlobal: false})
							aggregatedResults.Entries = append(aggregatedResults.Entries, entries...)
						}
					}
				}
				for _, dataQuery := range group.Queries {
					if isQueryDefinitionComplete(dataQuery) {
						for _, check := range queryChecks {
							entries := check.Run(lintCtx, QueryLintInput{Query: dataQuery, IsGlobal: false})
							aggregatedResults.Entries = append(aggregatedResults.Entries, entries...)
						}
					}
				}
			}
		}
	}

	if len(files) > 0 {
		bundleLoader := policy.DefaultBundleLoader()
		policyBundleForCompilation, err := bundleLoader.BundleFromPaths(files...)
		if err == nil {
			_, compileErr := policyBundleForCompilation.Compile(context.Background(), schema, nil)
			if compileErr != nil {
				var locs []Location
				for _, absPath := range absFiles {
					locs = append(locs, Location{
						File:   absPath,
						Line:   1,
						Column: 1,
					})
				}
				aggregatedResults.Entries = append(aggregatedResults.Entries, Entry{
					RuleID:   BundleCompileErrorRuleID,
					Message:  "Could not compile policy bundle: " + compileErr.Error(),
					Level:    LevelError,
					Location: locs,
				})
			}
		} else {
			var locs []Location
			for _, absPath := range absFiles {
				locs = append(locs, Location{File: absPath, Line: 1, Column: 1})
			}
			aggregatedResults.Entries = append(aggregatedResults.Entries, Entry{
				RuleID:   BundleInvalidRuleID,
				Message:  "Could not load policy bundle for compilation: " + err.Error(),
				Level:    LevelError,
				Location: locs,
			})
		}
	}

	return aggregatedResults, nil
}
