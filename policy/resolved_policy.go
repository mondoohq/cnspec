// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"github.com/pkg/errors"
	"go.mondoo.com/cnquery/explorer"
	"go.mondoo.com/cnquery/llx"
)

// WithDataQueries cycles through all data queries of the resolved policy and calls the given function
func (x *ResolvedPolicy) WithDataQueries(f func(id string, query *ExecutionQuery)) {
	for id, query := range x.ExecutionJob.Queries {
		// we don't care about scoring queries
		if _, ok := x.CollectorJob.ReportingQueries[id]; ok {
			continue
		}

		f(id, query)
	}
}

// TODO: attach to ResolvedPolicy
func (r *ResolvedPolicy) GetCodeBundle(query *explorer.Mquery) *llx.CodeBundle {
	executionQuery := r.ExecutionJob.Queries[query.CodeId]
	return executionQuery.GetCode()
}

// WithDataQueries cycles through all data queries of the resolved policy and calls the given function
func (x *ResolvedPolicy) NumDataQueries() int {
	numDataQueries := 0
	for id := range x.ExecutionJob.Queries {
		// we don't care about scoring queries
		if _, ok := x.CollectorJob.ReportingQueries[id]; ok {
			continue
		}
		numDataQueries++
	}
	return numDataQueries
}

// RootBundlePolicies retrieves the top policies from the resolved policy based on a given
// bundle. Typically, this would be the asset, if it is in the bundle. Otherwise,
// it may be whatever first policies show up in the bundle.
func (x *ResolvedPolicy) RootBundlePolicies(bundle *PolicyBundleMap, assetMrn string) ([]*Policy, error) {
	if p := bundle.Policies[assetMrn]; p != nil {
		return []*Policy{p}, nil
	}

	// if we can't find the asset, we look for the first working policy instead
	startJob := x.ReportingJobUuid
	if startJob == "" {
		return nil, errors.New("cannot find the primary policy")
	}

	jobs := []string{startJob}
	res := []*Policy{}
	for i := 0; i < len(jobs); i++ {
		job := x.CollectorJob.ReportingJobs[jobs[i]]
		if job == nil {
			return nil, errors.New("cannot find jobs in resolved policy to get policies")
		}

		p, ok := bundle.Policies[job.QrId]
		if ok {
			res = append(res, p)
		} else {
			for j := range job.ChildJobs {
				jobs = append(jobs, j)
			}
		}
	}

	if len(res) == 0 {
		return nil, errors.New("cannot find any primary policies")
	}

	return res, nil
}
