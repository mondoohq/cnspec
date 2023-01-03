package bundle

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/Masterminds/semver"
	"go.mondoo.com/cnspec/policy"
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
	policyMissingAssignedQuery = "policy-missing-assigned-query"
	policyMissingChecks        = "policy-missing-checks"
	policyMissingVersion       = "policy-missing-version"
	policyWrongVersion         = "policy-wrong-version"
	queryUid                   = "query-uid"
	queryTitle                 = "query-name"
	queryUidUnique             = "query-uid-unique"
	queryUnassigned            = "query-unassigned"
)

type Rule struct {
	ID          string
	Name        string
	Description string
}

var rules = []Rule{
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
		Description: "Every policy uid must not be used twice in the same bundle and namespace",
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
func Lint(files ...string) (*Results, error) {
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
	policyBundle, err := policy.BundleFromPaths(files...)
	if err == nil {
		_, err = policyBundle.Compile(context.Background(), nil)
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

	// index queries
	queryUids := map[string]int{}
	for i := range policyBundle.Queries {
		query := policyBundle.Queries[i]
		queryUids[query.Uid]++ // count the times the query uid is used
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

		if len(policy.Specs) == 0 {
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
		for j := range policy.Specs {
			spec := policy.Specs[j]

			// issue warning if no check is assigned
			if len(spec.ScoringQueries) == 0 {
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
			for uid := range spec.ScoringQueries {
				assignedQueries[uid] = struct{}{}

				_, ok := queryUids[uid]
				if !ok {
					res.Entries = append(res.Entries, Entry{
						RuleID:  policyMissingAssignedQuery,
						Message: fmt.Sprintf("policy %s assigned missing check query %s", policy.Uid, uid),
						Level:   levelError,
						Location: []Location{{
							File:   file,
							Line:   spec.FileContext.Line,
							Column: spec.FileContext.Column,
						}},
					})
				}
			}

			// check that all assigned data queries exist
			for uid := range spec.DataQueries {
				assignedQueries[uid] = struct{}{}

				_, ok := queryUids[uid]
				if !ok {
					res.Entries = append(res.Entries, Entry{
						RuleID:  policyMissingAssignedQuery,
						Message: fmt.Sprintf("policy %s assigned missing data query %s", policy.Uid, uid),
						Level:   levelError,
						Location: []Location{{
							File:   file,
							Line:   spec.FileContext.Line,
							Column: spec.FileContext.Column,
						}},
					})
				}
			}
		}
	}

	// validate the queries
	for i := range policyBundle.Queries {
		query := policyBundle.Queries[i]
		queryId := strconv.Itoa(i)
		uid := query.Uid

		if uid == "" {
			res.Entries = append(res.Entries, Entry{
				RuleID:  queryUid,
				Message: fmt.Sprintf("query %s does not define a UID", queryId),
				Level:   levelError,
				Location: []Location{{
					File:   file,
					Line:   query.FileContext.Line,
					Column: query.FileContext.Column,
				}},
			})
		} else {
			queryId = uid

			// check that the uid is valid
			if !reResourceID.MatchString(uid) {
				res.Entries = append(res.Entries, Entry{
					RuleID:  bundleInvalidUid,
					Message: fmt.Sprintf("query %s UID does not meet the requirements", queryId),
					Level:   levelError,
					Location: []Location{{
						File:   file,
						Line:   query.FileContext.Line,
						Column: query.FileContext.Column,
					}},
				})
			}
		}

		if query.Title == "" {
			res.Entries = append(res.Entries, Entry{
				RuleID:  queryTitle,
				Message: fmt.Sprintf("query %s does not define a title", queryId),
				Level:   levelError,
				Location: []Location{{
					File:   file,
					Line:   query.FileContext.Line,
					Column: query.FileContext.Column,
				}},
			})
		}

		// check if policy id was used already
		if queryUids[uid] > 1 {
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
	}
	return res, nil
}
