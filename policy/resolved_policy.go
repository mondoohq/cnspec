package policy

import "go.mondoo.com/cnquery/llx"

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
func (r *ResolvedPolicy) GetCodeBundle(query *Mquery) *llx.CodeBundle {
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
