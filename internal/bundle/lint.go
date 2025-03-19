// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package bundle

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/Masterminds/semver"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/resources"
	"go.mondoo.com/cnspec/v11/policy"
)

const (
	levelError                 = "error"
	levelWarning               = "warning"
	bundleCompileError         = "bundle-compile-error"
	bundleInvalid              = "bundle-invalid"
	bundleInvalidUid           = "bundle-invalid-uid"
	policyUid                  = "policy-uid"
	policyName                 = "policy-name"
	policyUidUnique            = "policy-uid-unique"
	policyMissingAssetFilter   = "policy-missing-asset-filter"
	policyMissingAssignedQuery = "policy-missing-assigned-query"
	policyMissingChecks        = "policy-missing-checks"
	policyMissingVersion       = "policy-missing-version"
	policyWrongVersion         = "policy-wrong-version"
	queryUid                   = "query-uid"
	queryTitle                 = "query-name"
	queryUidUnique             = "query-uid-unique"
	queryUnassigned            = "query-unassigned"
	queryUsedAsDifferentTypes  = "query-used-as-different-types"
)

type Rule struct {
	ID          string
	Name        string
	Description string
}

var LinterRules = []Rule{
	{
		ID:          bundleCompileError,
		Name:        "MQL compile error",
		Description: "Could not compile the MQL bundle",
	},
	{
		ID:          bundleInvalid,
		Name:        "Invalid bundle",
		Description: "The bundle is not properly YAML formatted",
	},
	{
		ID:          bundleInvalidUid,
		Name:        "UID is not valid",
		Description: "Every UID need to meet the following requirement: lowercase letters, digits, dots or hyphens, fewer than 200 chars, more than 5 chars",
	},
	{
		ID:          policyUid,
		Name:        "Missing policy UID",
		Description: "Every policy needs to have a `uid: identifier` field",
	},
	{
		ID:          policyName,
		Name:        "Missing policy name",
		Description: "Every policy needs to have a `name: My Policy Name` field",
	},
	{
		ID:          policyUidUnique,
		Name:        "No unique policy UID",
		Description: "Every policy UID must not be used twice in the same bundle and namespace",
	},
	{
		ID:          policyMissingAssetFilter,
		Name:        "Policy Spec is missing an asset filter",
		Description: "Policy Spec doesn't define an asset filter.",
	},
	{
		ID:          policyMissingChecks,
		Name:        "Policy is missing checks",
		Description: "Policy has no checks assigned",
	},
	{
		ID:          policyMissingAssignedQuery,
		Name:        "Assigned query missing",
		Description: "The assigned query does not exist",
	},
	{
		ID:          policyMissingVersion,
		Name:        "Policy version is missing",
		Description: "A policy version need to be set",
	},
	{
		ID:          policyWrongVersion,
		Name:        "Policy version is not valid",
		Description: "Policy versions must follow the semver pattern",
	},
	{
		ID:          queryUid,
		Name:        "Missing query UID",
		Description: "Every query needs to have a `uid: identifier` field",
	},
	{
		ID:          queryTitle,
		Name:        "Missing query title",
		Description: "Every query needs to have a `title: My Query Title` field",
	},
	{
		ID:          queryUidUnique,
		Name:        "No unique query UID",
		Description: "Every query uid must not be used twice in the same bundle and namespace",
	},
	{
		ID:          queryUnassigned,
		Name:        "Unassigned query",
		Description: "The query is not assigned to any policy",
	},
	{
		ID:          queryUsedAsDifferentTypes,
		Name:        "Query used as a check and data query",
		Description: "The query is used both as a check and a data query",
	},
}

type Results struct {
	BundleLocations []string
	Entries         []Entry
}

func (r *Results) HasError() bool {
	for i := range r.Entries {
		if r.Entries[i].Level == levelError {
			return true
		}
	}
	return false
}

type Entry struct {
	RuleID   string
	Level    string
	Message  string
	Location []Location
}

type Location struct {
	File   string
	Line   int
	Column int
}

// Lint validates a policy bundle for consistency
func Lint(schema resources.ResourcesSchema, files ...string) (*Results, error) {
	aggregatedResults := &Results{
		BundleLocations: []string{},
	}

	absFiles := []string{}
	for i := range files {
		file := files[i]
		absPath, err := filepath.Abs(file)
		if err != nil {
			return nil, err
		}
		absFiles = append(absFiles, absPath)
		aggregatedResults.BundleLocations = append(aggregatedResults.BundleLocations, absPath)
		res, err := lintFile(absPath)
		if err != nil {
			return nil, err
		}
		aggregatedResults.Entries = append(aggregatedResults.Entries, res.Entries...)
	}

	// Note: we only run compile on the aggregated level to ensure the bundle in combination is valid
	// Invalid yaml files are already caught by the individual linting, therefore we do not need extra error handling here
	bundleLoader := policy.DefaultBundleLoader()
	policyBundle, err := bundleLoader.BundleFromPaths(files...)
	if err == nil {
		_, err = policyBundle.Compile(context.Background(), schema, nil)
		if err != nil {
			locs := []Location{}

			for i := range absFiles {
				locs = append(locs, Location{
					File:   absFiles[i],
					Line:   1,
					Column: 1,
				})
			}

			aggregatedResults.Entries = append(aggregatedResults.Entries, Entry{
				RuleID:   bundleCompileError,
				Message:  "could not compile policy bundle:" + err.Error(),
				Level:    levelError,
				Location: locs,
			})
		}
	}

	return aggregatedResults, nil
}

// reResourceID: lowercase letters, digits, dots or hyphens, fewer than 200 chars, more than 5 chars
var reResourceID = regexp.MustCompile(`^([\d-_\.]|[a-zA-Z]){5,200}$`)

func lintFile(file string) (*Results, error) {
	res := &Results{}

	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	policyBundle, err := ParseYaml(data)
	if err != nil {
		// if we cannot compile the bundle, we cannot do any further checks
		// we do not return it as an error, but as a linting error
		res.Entries = append(res.Entries, Entry{
			RuleID:  bundleInvalid,
			Message: fmt.Sprintf("cannot parse the yaml file %s: %s", filepath.Base(file), err.Error()),
			Level:   levelError,
			Location: []Location{{
				File:   file,
				Line:   1,
				Column: 1,
			}},
		})
		return res, nil
	}

	// index global queries that are not embedded
	globalQueriesUids := map[string]int{}
	globalQueriesByUid := map[string]*Mquery{}
	checks := map[string]struct{}{}
	dataQueries := map[string]struct{}{}
	// child to parent mapping
	variantMapping := map[string]string{}
	for i := range policyBundle.Queries {
		query := policyBundle.Queries[i]
		globalQueriesUids[query.Uid]++ // count the times the query uid is used
		globalQueriesByUid[query.Uid] = query
		if len(query.Variants) > 0 {
			for _, variant := range query.Variants {
				variantMapping[variant.Uid] = query.Uid
			}
		}
	}
	for _, p := range policyBundle.Policies {
		for _, pg := range p.Groups {
			for _, query := range pg.Checks {
				if len(query.Variants) > 0 {
					for _, variant := range query.Variants {
						variantMapping[variant.Uid] = query.Uid
					}
				}
			}
			for _, query := range pg.Queries {
				if len(query.Variants) > 0 {
					for _, variant := range query.Variants {
						variantMapping[variant.Uid] = query.Uid
					}
				}
			}
		}
	}

	// validate policies
	policyUids := map[string]struct{}{}
	assignedQueries := map[string]struct{}{}
	for i := range policyBundle.Policies {
		policy := policyBundle.Policies[i]
		policyId := strconv.Itoa(i) // use index as default and change it to uid when the policy has a uid

		if policy.Uid == "" {
			res.Entries = append(res.Entries, Entry{
				RuleID:  policyUid,
				Message: fmt.Sprintf("policy %s does not define a UID", policyId),
				Level:   levelError,
				Location: []Location{{
					File:   file,
					Line:   policy.FileContext.Line,
					Column: policy.FileContext.Column,
				}},
			})
		} else {
			policyId = policy.Uid
			// check that the uid is valid
			if !reResourceID.MatchString(policy.Uid) {
				res.Entries = append(res.Entries, Entry{
					RuleID:  bundleInvalidUid,
					Message: fmt.Sprintf("policy %s UID does not meet the requirements", policyId),
					Level:   levelError,
					Location: []Location{{
						File:   file,
						Line:   policy.FileContext.Line,
						Column: policy.FileContext.Column,
					}},
				})
			}
		}

		if policy.Name == "" {
			res.Entries = append(res.Entries, Entry{
				RuleID:  policyName,
				Message: fmt.Sprintf("policy %s does not define a name", policyId),
				Level:   levelError,
				Location: []Location{{
					File:   file,
					Line:   policy.FileContext.Line,
					Column: policy.FileContext.Column,
				}},
			})
		}

		// check if policy id was used already
		_, ok := policyUids[policy.Uid]
		if ok {
			res.Entries = append(res.Entries, Entry{
				RuleID:  policyUidUnique,
				Message: fmt.Sprintf("policy uid %s is used multiple times", policy.Uid),
				Level:   levelError,
				Location: []Location{{
					File:   file,
					Line:   policy.FileContext.Line,
					Column: policy.FileContext.Column,
				}},
			})
		}
		policyUids[policy.Uid] = struct{}{}

		// check for missing version
		if policy.Version == "" {
			res.Entries = append(res.Entries, Entry{
				RuleID:  policyMissingVersion,
				Message: "Policy " + policy.Uid + " is missing version",
				Level:   levelError,
				Location: []Location{{
					File:   file,
					Line:   policy.FileContext.Line,
					Column: policy.FileContext.Column,
				}},
			})
		}

		// check that the version is valid semver
		_, err := semver.NewVersion(policy.Version)
		if err != nil {
			res.Entries = append(res.Entries, Entry{
				RuleID:  policyWrongVersion,
				Message: "Policy " + policy.Uid + " has invalid version: " + policy.Version,
				Level:   levelError,
				Location: []Location{{
					File:   file,
					Line:   policy.FileContext.Line,
					Column: policy.FileContext.Column,
				}},
			})
		}

		if len(policy.Groups) == 0 {
			res.Entries = append(res.Entries, Entry{
				RuleID:  policyMissingChecks,
				Message: "Policy " + policy.Uid + " is missing checks",
				Level:   levelError,
				Location: []Location{{
					File:   file,
					Line:   policy.FileContext.Line,
					Column: policy.FileContext.Column,
				}},
			})
		}

		// check that all assigned checks actually exist as full queries
		for j := range policy.Groups {
			group := policy.Groups[j]

			/* OLD
			// issue warning if no filters are assigned, but do not show the warning if the policy has variants
			if (group.Filters == nil || len(group.Filters.Items) == 0) && len(group.Policies) == 0 && !hasVariants(group, globalQueriesByUid) {
				location := Location{
					File:   file,
					Line:   group.FileContext.Line,
					Column: group.FileContext.Column,
				}

				if group.Filters != nil {
					location = Location{
						File:   file,
						Line:   group.Filters.FileContext.Line,
						Column: group.Filters.FileContext.Column,
					}
				}

				res.Entries = append(res.Entries, Entry{
					RuleID:   policyMissingAssetFilter,
					Message:  "Policy " + policy.Uid + " doesn't define an asset filter.",
					Level:    levelWarning,
					Location: []Location{location},
				})
			}
			*/

			// issue warning if no filters are assigned, but do not show the warning if the
			// OLD:
			// if (group.Filters == nil || len(group.Filters.Items) == 0) && len(group.Policies) == 0 && !hasVariants(group, globalQueriesByUid) {
			if (group.Filters == nil || len(group.Filters.Items) == 0) && len(group.Policies) == 0 && !checksHaveFiltersOrVariants(group, globalQueriesByUid) {
				location := Location{
					File:   file,
					Line:   group.FileContext.Line,
					Column: group.FileContext.Column,
				}

				if group.Filters != nil {
					location = Location{
						File:   file,
						Line:   group.Filters.FileContext.Line,
						Column: group.Filters.FileContext.Column,
					}
				}

				res.Entries = append(res.Entries, Entry{
					RuleID:   policyMissingAssetFilter,
					Message:  "Policy " + policy.Uid + " doesn't define an asset filter.",
					Level:    levelWarning,
					Location: []Location{location},
				})
			}

			// issue warning if no checks or data queries are assigned
			if len(group.Checks) == 0 && len(group.Queries) == 0 && len(group.Policies) == 0 {
				res.Entries = append(res.Entries, Entry{
					RuleID:  policyMissingChecks,
					Message: "Policy " + policy.Uid + " is missing checks",
					Level:   levelError,
					Location: []Location{{
						File:   file,
						Line:   policy.FileContext.Line,
						Column: policy.FileContext.Column,
					}},
				})
			}

			// check that all assigned queries actually exist as queries
			for ic := range group.Checks {
				check := group.Checks[ic]
				uid := check.Uid
				updateAssignedQueries(check, assignedQueries, globalQueriesByUid)

				checks[check.Uid] = struct{}{}
				if _, ok := dataQueries[check.Uid]; ok {
					res.Entries = append(res.Entries, Entry{
						RuleID:  queryUsedAsDifferentTypes,
						Message: fmt.Sprintf("query %s is used as a check and data query", uid),
						Level:   levelError,
						Location: []Location{{
							File:   file,
							Line:   group.FileContext.Line,
							Column: group.FileContext.Column,
						}},
					})
				}

				// check if the query is embedded
				if isEmbeddedQuery(check) {
					// NOTE: embedded queries do not need a uid
					lintQuery(check, file, globalQueriesUids, assignedQueries, variantMapping, false)
				} else {
					// if the query is not embedded, then it needs to be available globally
					_, ok := globalQueriesUids[uid]
					if !ok {
						res.Entries = append(res.Entries, Entry{
							RuleID:  policyMissingAssignedQuery,
							Message: fmt.Sprintf("policy %s assigned missing check query %s", policy.Uid, uid),
							Level:   levelError,
							Location: []Location{{
								File:   file,
								Line:   group.FileContext.Line,
								Column: group.FileContext.Column,
							}},
						})
					}
				}
			}

			// check that all assigned data queries exist
			for iq := range group.Queries {
				query := group.Queries[iq]
				uid := query.Uid
				updateAssignedQueries(query, assignedQueries, globalQueriesByUid)

				dataQueries[query.Uid] = struct{}{}
				if _, ok := checks[query.Uid]; ok {
					res.Entries = append(res.Entries, Entry{
						RuleID:  queryUsedAsDifferentTypes,
						Message: fmt.Sprintf("query %s is used as a check and data query", uid),
						Level:   levelError,
						Location: []Location{{
							File:   file,
							Line:   group.FileContext.Line,
							Column: group.FileContext.Column,
						}},
					})
				}

				// check if the query is embedded
				if isEmbeddedQuery(query) {
					// NOTE: embedded queries do not need a uid
					lintQuery(query, file, globalQueriesUids, assignedQueries, variantMapping, false)
				} else {
					// if the query is not embedded, then it needs to be available globally
					_, ok := globalQueriesUids[uid]
					if !ok {
						res.Entries = append(res.Entries, Entry{
							RuleID:  policyMissingAssignedQuery,
							Message: fmt.Sprintf("policy %s assigned missing data query %s", policy.Uid, uid),
							Level:   levelError,
							Location: []Location{{
								File:   file,
								Line:   group.FileContext.Line,
								Column: group.FileContext.Column,
							}},
						})
					}
				}
			}
		}
	}

	// validate the global queries
	for i := range policyBundle.Queries {
		query := policyBundle.Queries[i]
		queryResults := lintQuery(query, file, globalQueriesUids, assignedQueries, variantMapping, true)
		res.Entries = append(res.Entries, queryResults.Entries...)
	}
	return res, nil
}

func isEmbeddedQuery(query *Mquery) bool {
	if query.Title != "" || query.Mql != "" || len(query.Variants) > 0 {
		return true
	}
	return false
}

func hasVariants(group *PolicyGroup, queryMap map[string]*Mquery) bool {
	for _, check := range group.Checks {
		// check embedded query
		if check.Variants != nil {
			return true
		}

		// check referenced query
		q, ok := queryMap[check.Uid]
		if ok && q.Variants != nil {
			return true
		}
	}
	return false
}

func checksHaveFiltersOrVariants(group *PolicyGroup, queryMap map[string]*Mquery) bool {
	for _, check := range group.Checks {

		if (check.Filters != nil && len(check.Filters.Items) > 0) && check.Variants != nil {
			return true
		}

		// check referenced query
		q, ok := queryMap[check.Uid]

		if ok && (q.Filters != nil && len(q.Filters.Items) > 0) {
			return true
		}

	}
	return false
}

func lintQuery(query *Mquery, file string, globalQueriesUids map[string]int, assignedQueries map[string]struct{}, variantMapping map[string]string, requiresUID bool) *Results {
	res := &Results{}
	uid := query.Uid

	if requiresUID {
		if uid == "" {
			res.Entries = append(res.Entries, Entry{
				RuleID:  queryUid,
				Message: fmt.Sprintf("query does not define a UID"),
				Level:   levelError,
				Location: []Location{{
					File:   file,
					Line:   query.FileContext.Line,
					Column: query.FileContext.Column,
				}},
			})
		} else {
			// check that the uid is valid
			if !reResourceID.MatchString(uid) {
				res.Entries = append(res.Entries, Entry{
					RuleID:  bundleInvalidUid,
					Message: fmt.Sprintf("query %s UID does not meet the requirements", uid),
					Level:   levelError,
					Location: []Location{{
						File:   file,
						Line:   query.FileContext.Line,
						Column: query.FileContext.Column,
					}},
				})
			}
		}
	}

	if query.Title == "" {
		_, hasParent := variantMapping[query.Uid]
		if !hasParent {
			res.Entries = append(res.Entries, Entry{
				RuleID:  queryTitle,
				Message: fmt.Sprintf("query %s does not define a title", uid),
				Level:   levelError,
				Location: []Location{{
					File:   file,
					Line:   query.FileContext.Line,
					Column: query.FileContext.Column,
				}},
			})
		}
	}

	// check if query id was used already
	if globalQueriesUids[uid] > 1 {
		res.Entries = append(res.Entries, Entry{
			RuleID:  queryUidUnique,
			Message: fmt.Sprintf("query uid %s is used multiple times", uid),
			Level:   levelError,
			Location: []Location{{
				File:   file,
				Line:   query.FileContext.Line,
				Column: query.FileContext.Column,
			}},
		})
	}

	// check if the query is assigned to a policy
	_, ok := assignedQueries[uid]
	if !ok {
		res.Entries = append(res.Entries, Entry{
			RuleID:  queryUnassigned,
			Message: fmt.Sprintf("query uid %s is not assigned to a policy", uid),
			Level:   levelWarning,
			Location: []Location{{
				File:   file,
				Line:   query.FileContext.Line,
				Column: query.FileContext.Column,
			}},
		})
	}
	return res
}

var emptyQueryTracker = map[string]*Mquery{}

func updateAssignedQueries(query *Mquery, assignedTracker map[string]struct{}, queryTracker map[string]*Mquery) {
	assignedTracker[query.Uid] = struct{}{}

	for i := range query.Variants {
		variant := query.Variants[i]
		assignedTracker[variant.Uid] = struct{}{}
	}

	if base, ok := queryTracker[query.Uid]; ok {
		updateAssignedQueries(base, assignedTracker, emptyQueryTracker)
	}
}
