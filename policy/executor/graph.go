// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package executor

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnspec/cli/progress"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/cnspec/policy/executor/internal"
	"go.mondoo.com/cnspec/policy/scanstats"
	"go.mondoo.com/mql"
	"go.mondoo.com/mql/llx"
	"go.mondoo.com/mql/mqlc"
)

type GraphExecutor interface {
	Execute()
}

// ScoreCollector receives scores emitted by the graph executor. Re-exported
// for callers of RescoreResolvedPolicy.
type ScoreCollector = internal.ScoreCollector

// RescoreResolvedPolicy rolls up pre-computed leaf scores through the graph
// executor without running any queries. Reporting jobs whose QrId is a key
// in scores are built as static nodes holding the supplied score; aggregate
// reporting jobs (policies, controls, frameworks) roll up their children
// using their configured ScoringSystem. Rolled-up scores are delivered to
// scoreCollector. The "root" reporting job's QrId is rewritten to assetMrn
// when matching against scores, consistent with the normal execution path.
func RescoreResolvedPolicy(
	assetMrn string,
	resolvedPolicy *policy.ResolvedPolicy,
	scores map[string]*policy.Score,
	scoreCollector ScoreCollector,
) error {
	builder := builderFromResolvedPolicy(resolvedPolicy)
	builder.AddScoreCollector(scoreCollector)
	builder.WithRescore(scores)

	ge, err := builder.Build(nil, assetMrn)
	if err != nil {
		return err
	}
	return ge.Execute()
}

func ExecuteResolvedPolicy(ctx context.Context, runtime llx.Runtime, collectorSvc policy.PolicyResolver, assetMrn string,
	resolvedPolicy *policy.ResolvedPolicy, features mql.Features, progressReporter progress.Progress,
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

	builder := builderFromResolvedPolicy(resolvedPolicy)
	builder.AddDatapointCollector(collector)
	builder.AddScoreCollector(collector)

	stats := scanstats.CollectorFromContext(ctx)
	var counter *countingCollector
	if stats != nil {
		counter = newCountingCollector(resolvedPolicy)
		builder.AddScoreCollector(counter)
		builder.AddDatapointCollector(counter)
	}

	if progressReporter != nil {
		builder.WithProgressReporter(progressReporter)
	}

	if features.IsActive(mql.ErrorsAsFailures) {
		builder.WithFeatureFlagFailErrors()
	}

	ge, err := builder.Build(runtime, assetMrn)
	if err != nil {
		return err
	}

	ge.Debug(ctx, "resolved-policy")

	err = ge.Execute()
	if counter != nil {
		counter.recordTo(stats)
	}
	return err
}

func ExecuteFilterQueries(ctx context.Context, runtime llx.Runtime, queries []*policy.Mquery, timeout time.Duration) ([]*policy.Mquery, []error) {
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

	tracker := newFilterScoreTracker()
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
				tracker.record(s)
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
	log.Debug().Msg("finished executing filter queries")

	ge.Debug(ctx, "filter-queries")

	// Decided only now that execution has finished, so every provisional score
	// has had the chance to be corrected. See filterScoreTracker.
	passingFilterQueries := tracker.passing()

	filteredQueries := []*policy.Mquery{}
	for id, query := range queryMap {
		if _, ok := passingFilterQueries[id]; ok {
			filteredQueries = append(filteredQueries, query)
		}
	}

	return filteredQueries, errors
}

// filterScoreTracker records the scores the graph emits for filter queries and
// decides which filters matched once execution has finished.
//
// The graph emits a score for a reporting query on EVERY round in which all of
// that query's entrypoints are resolved, and those intermediate scores are
// provisional. Datapoint checksums are content-addressed and shared across
// queries, so another query can resolve one of this query's entrypoints before
// this query executes it itself — with a short-circuit nil for a branch it
// never evaluated, or with a broadcast placeholder for a query that could not
// run at all. ReportingQueryNodeData.score() skips nil-valued entrypoints
// rather than failing them, so a multi-statement filter can transiently score
// 100 on just the subset that happened to resolve.
//
// Concretely: the Debian 8 and Debian 9 policy filters are identical apart from
// their version line —
//
//	asset.platform == "debian"
//	asset.version == /^8\./
//	asset.kind != "container-image"
//
// On a Debian 11 host, statements 1 and 3 are resolved TRUE by the Debian
// 9/10/11 filters, which compile to the same checksums. If the version
// statement is nil for a single round, the Debian 8 filter scores 100 and the
// host is told to run the whole Debian 8 policy. ScoreCompletion is 100 on
// every round, so a provisional score is indistinguishable from a final one at
// this layer.
//
// resultUpgrades() repairs the datapoint and the node recalculates to the real
// score, so the LAST score emitted for a query is the authoritative one.
// Latching the first passing score threw that correction away and made the
// filter set depend on the order results happened to arrive in.
type filterScoreTracker struct {
	last map[string]*policy.Score
}

func newFilterScoreTracker() *filterScoreTracker {
	return &filterScoreTracker{last: map[string]*policy.Score{}}
}

// record keeps the most recent score seen for a query, replacing any earlier
// provisional one.
func (t *filterScoreTracker) record(s *policy.Score) {
	if s == nil {
		return
	}
	t.last[s.QrId] = s
}

// passing returns the queries whose FINAL score means the filter matched.
func (t *filterScoreTracker) passing() map[string]struct{} {
	res := make(map[string]struct{}, len(t.last))
	for id, s := range t.last {
		if s.ScoreCompletion == 100 && s.Value == 100 {
			res[id] = struct{}{}
		}
	}
	return res
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
