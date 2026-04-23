// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package executor

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnspec/v13/cli/progress"
	"go.mondoo.com/cnspec/v13/policy"
	"go.mondoo.com/cnspec/v13/policy/executor/internal"
	"go.mondoo.com/mql/v13"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/mqlc"
)

type GraphExecutor interface {
	Execute()
}

// ScoreCollector receives scores produced by the graph executor.
type ScoreCollector = internal.ScoreCollector

// defaultQueryTimeout is the per-query execution timeout used by
// ExecuteResolvedPolicy. Matches the previous local_scanner default.
const defaultQueryTimeout = 5 * time.Minute

// ExecuteResolvedPolicy builds a graph from the resolved policy,
// executes queries against the provided runtime, and sends results to
// the PolicyResolver via a BufferedCollector.
func ExecuteResolvedPolicy(
	ctx context.Context,
	runtime llx.Runtime,
	collectorSvc policy.PolicyResolver,
	assetMrn string,
	resolvedPolicy *policy.ResolvedPolicy,
	features mql.Features,
	progressReporter progress.Progress,
) error {
	var opts []internal.BufferedCollectorOpt

	riskOpt, err := internal.WithResolvedPolicy(resolvedPolicy)
	if err != nil {
		log.Warn().Err(err).Msg("failed to execute advanced features in resolved policy")
	} else {
		opts = append(opts, riskOpt)
	}

	collector := internal.NewBufferedCollector(
		ctx,
		internal.NewPolicyServiceCollector(assetMrn, collectorSvc),
		opts...,
	)
	defer collector.FlushAndStop()

	numQueries := len(resolvedPolicy.GetExecutionJob().GetQueries())
	producer := internal.NewDefaultProducer(runtime, numQueries, defaultQueryTimeout, false)

	builder := builderFromResolvedPolicy(resolvedPolicy)
	builder.AddDatapointCollector(collector)
	builder.AddScoreCollector(collector)
	builder.WithProducer(producer)
	builder.WithRunQueue(producer.RunQueue())
	if progressReporter != nil {
		builder.WithProgressReporter(progressReporter)
	}
	if features.IsActive(mql.ErrorsAsFailures) {
		builder.WithFeatureFlagFailErrors()
	}

	ge, err := builder.Build(assetMrn)
	if err != nil {
		return err
	}

	ge.Debug("resolved-policy")

	return ge.Execute()
}

// RescoreResolvedPolicy builds a graph from the resolved policy, feeds
// the provided pre-computed scores in as a single batch via an internal
// BatchScoreProducer, and sends the rolled-up results to the given
// ScoreCollector. No runtime is needed -- no queries are executed.
func RescoreResolvedPolicy(
	assetMrn string,
	resolvedPolicy *policy.ResolvedPolicy,
	scores map[string]*policy.Score,
	scoreCollector ScoreCollector,
) error {
	rjs := resolvedPolicy.GetCollectorJob().GetReportingJobs()
	producer := internal.NewBatchScoreProducer(scores, rjs, assetMrn)

	builder := builderFromResolvedPolicy(resolvedPolicy)
	builder.AddScoreCollector(scoreCollector)
	builder.WithProducer(producer)
	builder.EnableScoreRisk()

	ge, err := builder.Build(assetMrn)
	if err != nil {
		return err
	}

	ge.Debug("resolved-policy")

	return ge.Execute()
}

func ExecuteFilterQueries(runtime llx.Runtime, queries []*policy.Mquery, timeout time.Duration) ([]*policy.Mquery, []error) {
	log.Debug().Msg("executing filter queries")
	queryMap := map[string]*policy.Mquery{}

	builder := internal.NewBuilder()
	builder.WithDumpDatapoints()
	for _, m := range queries {
		codeBundle, err := mqlc.Compile(m.Mql, nil, mqlc.NewConfig(runtime.Schema(), mql.DefaultFeatures))
		// Errors for filter queries are common when they reference resources for
		// providers that are not found on the system.
		if err != nil {
			log.Debug().Err(err).Str("mql", m.Mql).Msg("skipping filter query, not supported")
			continue
		}
		builder.AddQuery(codeBundle, nil, nil, nil)

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
				log.Debug().Str("qrId", s.QrId).
					Int("scoreCompletion", int(s.ScoreCompletion)).
					Int("dataCompletion", int(s.DataCompletion)).
					Int("value", int(s.Value)).
					Msg("filter query score received")
				if s.ScoreCompletion == 100 && s.Value == 100 {
					passingFilterQueries[s.QrId] = struct{}{}
				}
			}
		},
	}
	builder.AddScoreCollector(collector)
	builder.WithQueryTimeout(timeout)
	p := internal.NewDefaultProducer(runtime, len(queryMap), timeout, false)
	builder.WithProducer(p)
	builder.WithRunQueue(p.RunQueue())

	var errors []error
	ge, err := builder.Build("")
	if err != nil {
		errors = append(errors, err)
		return nil, errors
	}

	if err := ge.Execute(); err != nil {
		return nil, []error{err}
	}
	log.Debug().Msg("finished executing filter queries")

	ge.Debug("filter-queries")

	filteredQueries := []*policy.Mquery{}
	for id, query := range queryMap {
		if _, ok := passingFilterQueries[id]; ok {
			filteredQueries = append(filteredQueries, query)
		}
	}

	return filteredQueries, errors
}

func ExecuteQuery(runtime llx.Runtime, codeBundle *llx.CodeBundle, props map[string]*llx.Primitive, features mql.Features) (*policy.Score, map[string]*llx.RawResult, error) {
	builder := internal.NewBuilder()

	builder.AddQuery(codeBundle, nil, props, nil)
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
	p := internal.NewDefaultProducer(runtime, 1, 5*time.Minute, false)
	builder.WithProducer(p)
	builder.WithRunQueue(p.RunQueue())

	ge, err := builder.Build("")
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
	rqs := resolvedPolicy.GetCollectorJob().GetReportingQueries()
	if rqs == nil {
		rqs = map[string]*policy.StringArray{}
	}
	for _, eq := range resolvedPolicy.GetExecutionJob().GetQueries() {
		var notifies []string
		if sa := rqs[eq.GetCode().GetCodeV2().GetId()]; sa != nil {
			if len(sa.Items) > 0 {
				notifies = sa.Items
			}
		}
		b.AddQuery(eq.GetCode(), eq.GetProperties(), nil, notifies)
	}

	for _, rj := range resolvedPolicy.GetCollectorJob().GetReportingJobs() {
		b.AddReportingJob(rj)
	}

	for datapointChecksum, dqi := range resolvedPolicy.GetCollectorJob().GetDatapoints() {
		b.AddDatapointType(datapointChecksum, dqi.Type)
	}

	return b
}
