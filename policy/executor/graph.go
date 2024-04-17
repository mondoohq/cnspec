// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package executor

import (
	"time"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11"
	"go.mondoo.com/cnquery/v11/cli/progress"
	"go.mondoo.com/cnquery/v11/explorer"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/mqlc"
	"go.mondoo.com/cnspec/v11/policy"
	"go.mondoo.com/cnspec/v11/policy/executor/internal"
)

type GraphExecutor interface {
	Execute()
}

func ExecuteResolvedPolicy(runtime llx.Runtime, collectorSvc policy.PolicyResolver, assetMrn string,
	resolvedPolicy *policy.ResolvedPolicy, features cnquery.Features, progressReporter progress.Progress,
) error {
	var opts []internal.BufferedCollectorOpt

	riskOpt, err := internal.WithResolvedPolicy(resolvedPolicy)
	if err != nil {
		log.Warn().Err(err).Msg("failed to execute advanced features in resolved policy")
	} else {
		opts = append(opts, riskOpt)
	}

	collector := internal.NewBufferedCollector(
		internal.NewPolicyServiceCollector(assetMrn, collectorSvc),
		opts...,
	)
	defer collector.FlushAndStop()

	builder := builderFromResolvedPolicy(resolvedPolicy)
	builder.AddDatapointCollector(collector)
	builder.AddScoreCollector(collector)
	if progressReporter != nil {
		builder.WithProgressReporter(progressReporter)
	}

	if features.IsActive(cnquery.ErrorsAsFailures) {
		builder.WithFeatureFlagFailErrors()
	}

	ge, err := builder.Build(runtime, assetMrn)
	if err != nil {
		return err
	}

	ge.Debug()

	return ge.Execute()
}

func ExecuteFilterQueries(runtime llx.Runtime, queries []*explorer.Mquery, timeout time.Duration) ([]*explorer.Mquery, []error) {
	queryMap := map[string]*explorer.Mquery{}

	builder := internal.NewBuilder()
	for _, m := range queries {
		codeBundle, err := mqlc.Compile(m.Mql, nil, mqlc.NewConfig(runtime.Schema(), cnquery.DefaultFeatures))
		// Errors for filter queries are common when they reference resources for
		// providers that are not found on the system.
		if err != nil {
			log.Debug().Err(err).Str("mql", m.Mql).Msg("skipping filter query, not supported")
			continue
		}
		builder.AddQuery(codeBundle, nil, nil)

		builder.CollectScore(codeBundle.CodeV2.Id)
		queryMap[codeBundle.CodeV2.Id] = m
	}

	passingFilterQueries := map[string]struct{}{}
	collector := &internal.FuncCollector{
		SinkScoreFunc: func(scores []*policy.Score) {
			for _, s := range scores {
				// TODO: s.Completion() is 50 and s.ScoreCompletion is 100
				// since data collection is part of the reporting job, queries
				// need to indicate there is no data so the completion is 100
				if s.ScoreCompletion == 100 && s.Value == 100 {
					passingFilterQueries[s.QrId] = struct{}{}
				}
			}
		},
	}
	builder.AddScoreCollector(collector)
	builder.WithQueryTimeout(timeout)

	var errors []error
	ge, err := builder.Build(runtime, "")
	if err != nil {
		errors = append(errors, err)
		return nil, errors
	}

	if err := ge.Execute(); err != nil {
		return nil, []error{err}
	}

	filteredQueries := []*explorer.Mquery{}
	for id, query := range queryMap {
		if _, ok := passingFilterQueries[id]; ok {
			filteredQueries = append(filteredQueries, query)
		}
	}

	return filteredQueries, errors
}

func ExecuteQuery(runtime llx.Runtime, codeBundle *llx.CodeBundle, props map[string]*llx.Primitive, features cnquery.Features) (*policy.Score, map[string]*llx.RawResult, error) {
	builder := internal.NewBuilder()

	builder.AddQuery(codeBundle, nil, props)
	for _, checksum := range internal.CodepointChecksums(codeBundle) {
		builder.CollectDatapoint(checksum)
	}
	qrID := codeBundle.CodeV2.Id
	builder.CollectScore(qrID)

	resultMap := map[string]*llx.RawResult{}
	score := &policy.Score{
		QrId: qrID,
	}
	collector := &internal.FuncCollector{
		SinkDataFunc: func(results []*llx.RawResult) {
			for _, d := range results {
				resultMap[d.CodeID] = d
			}
		},
		SinkScoreFunc: func(scores []*policy.Score) {
			for _, s := range scores {
				if s.QrId == qrID {
					score = s
				}
			}
		},
	}
	builder.AddDatapointCollector(collector)
	builder.AddScoreCollector(collector)

	ge, err := builder.Build(runtime, "")
	if err != nil {
		return nil, nil, err
	}

	if err := ge.Execute(); err != nil {
		return nil, nil, err
	}

	return score, resultMap, nil
}

func builderFromResolvedPolicy(resolvedPolicy *policy.ResolvedPolicy) *internal.GraphBuilder {
	b := internal.NewBuilder()

	for _, eq := range resolvedPolicy.ExecutionJob.Queries {
		b.AddQuery(eq.Code, eq.Properties, nil)
	}

	for _, rj := range resolvedPolicy.CollectorJob.ReportingJobs {
		b.AddReportingJob(rj)
	}

	for datapointChecksum, dqi := range resolvedPolicy.CollectorJob.Datapoints {
		b.AddDatapointType(datapointChecksum, dqi.Type)
	}

	return b
}
