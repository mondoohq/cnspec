package executor

import (
	"time"

	"go.mondoo.com/cnquery"
	"go.mondoo.com/cnquery/cli/progress"
	"go.mondoo.com/cnquery/explorer"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/mqlc"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/cnspec/policy/executor/internal"
)

type GraphExecutor interface {
	Execute()
}

func ExecuteResolvedPolicy(schema *resources.Schema, runtime *resources.Runtime, collectorSvc policy.PolicyResolver, assetMrn string,
	resolvedPolicy *policy.ResolvedPolicy, features cnquery.Features, progressReporter progress.Progress,
) error {
	collector := internal.NewBufferedCollector(internal.NewPolicyServiceCollector(assetMrn, collectorSvc))
	defer collector.FlushAndStop()

	builder := builderFromResolvedPolicy(resolvedPolicy)
	builder.AddDatapointCollector(collector)
	builder.AddScoreCollector(collector)
	if progressReporter != nil {
		builder.WithProgressReporter(progressReporter)
	}

	ge, err := builder.Build(schema, runtime, assetMrn)
	if err != nil {
		return err
	}

	ge.Debug()

	return ge.Execute()
}

func ExecuteFilterQueries(schema *resources.Schema, runtime *resources.Runtime, queries []*explorer.Mquery, timeout time.Duration) ([]*explorer.Mquery, []error) {
	var errs []error
	queryMap := map[string]*explorer.Mquery{}

	builder := internal.NewBuilder()
	for _, m := range queries {
		codeBundle, err := mqlc.Compile(m.Mql, nil, mqlc.NewConfig(schema, cnquery.DefaultFeatures))
		if err != nil {
			errs = append(errs, err)
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

	ge, err := builder.Build(schema, runtime, "")
	if err != nil {
		errs = append(errs, err)
		return nil, errs
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

	return filteredQueries, errs
}

func ExecuteQuery(schema *resources.Schema, runtime *resources.Runtime, codeBundle *llx.CodeBundle, props map[string]*llx.Primitive, features cnquery.Features) (*policy.Score, map[string]*llx.RawResult, error) {
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

	ge, err := builder.Build(schema, runtime, "")
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
