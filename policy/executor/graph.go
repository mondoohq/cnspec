package executor

import (
	"time"

	"go.mondoo.com/cnquery"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/mqlc"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnspec/cli/progress"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/cnspec/policy/executor/internal"
)

type GraphExecutor interface {
	Execute()
}

func ExecuteResolvedPolicy(schema *resources.Schema, runtime *resources.Runtime, collectorSvc policy.PolicyResolver, assetMrn string,
	resolvedPolicy *policy.ResolvedPolicy, features cnquery.Features, progressFn progress.Progress,
) error {
	useV2Code := features.IsActive(cnquery.PiperCode)

	collector := internal.NewBufferedCollector(internal.NewPolicyServiceCollector(assetMrn, collectorSvc, useV2Code))
	defer collector.FlushAndStop()

	builder := builderFromResolvedPolicy(resolvedPolicy)
	builder.WithUseV2Code(useV2Code)
	builder.AddDatapointCollector(collector)
	builder.AddScoreCollector(collector)
	builder.WithFeatureBoolAssertions(features.IsActive(cnquery.BoolAssertions))
	if progressFn != nil {
		builder.WithProgressReporter(progressFn)
	}

	ge, err := builder.Build(schema, runtime, assetMrn)
	if err != nil {
		return err
	}

	ge.Debug()

	return ge.Execute()
}

func ExecuteFilterQueries(schema *resources.Schema, runtime *resources.Runtime, queries []*policy.Mquery, timeout time.Duration) ([]*policy.Mquery, []error) {
	var errs []error
	queryMap := map[string]*policy.Mquery{}

	builder := internal.NewBuilder()
	for _, m := range queries {
		codeBundle, err := mqlc.Compile(m.Query, schema, cnquery.Features{}, nil)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		builder.AddQuery(codeBundle, nil, nil)

		builder.CollectScore(codeBundle.DeprecatedV5Code.Id)
		queryMap[codeBundle.DeprecatedV5Code.Id] = m
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

	filteredQueries := []*policy.Mquery{}
	for id, query := range queryMap {
		if _, ok := passingFilterQueries[id]; ok {
			filteredQueries = append(filteredQueries, query)
		}
	}

	return filteredQueries, errs
}

func ExecuteQuery(schema *resources.Schema, runtime *resources.Runtime, codeBundle *llx.CodeBundle, props map[string]*llx.Primitive, features cnquery.Features) (*policy.Score, map[string]*llx.RawResult, error) {
	useV2Code := features.IsActive(cnquery.PiperCode)

	builder := internal.NewBuilder()
	builder.WithUseV2Code(useV2Code)
	builder.WithFeatureBoolAssertions(features.IsActive(cnquery.BoolAssertions))

	builder.AddQuery(codeBundle, nil, props)
	for _, checksum := range internal.CodepointChecksums(codeBundle, useV2Code) {
		builder.CollectDatapoint(checksum)
	}
	qrID := ""
	if useV2Code {
		qrID = codeBundle.CodeV2.Id
	} else {
		qrID = codeBundle.DeprecatedV5Code.Id
	}
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
