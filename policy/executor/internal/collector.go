// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package internal

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v12"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/upstream/health"
	"go.mondoo.com/cnquery/v12/utils/iox"
	"go.mondoo.com/cnspec/v12"
	"go.mondoo.com/cnspec/v12/policy"
)

const (
	// MAX_DATAPOINT is the limit in bytes of any data field. The limit
	// is used to prevent sending data upstream that is too large for the
	// server to store. The limit is specified in bytes.
	// TODO: needed to increase the size for vulnerability reports
	// we need to size down the vulnerability reports with just current cves and advisories
	MAX_DATAPOINT = 2 * (1 << 20)
)

type DatapointCollector interface {
	SinkData([]*llx.RawResult)
}

type ScoreCollector interface {
	SinkScore([]*policy.Score)
}

type Collector interface {
	DatapointCollector
	ScoreCollector
}

type BufferedCollector struct {
	ctx            context.Context
	results        map[string]*llx.RawResult
	scores         map[string]*policy.Score
	lock           sync.Mutex
	collector      *PolicyServiceCollector
	resolvedPolicy *policy.ResolvedPolicy
	riskMRNs       map[string][]string
	keepQrIds      map[string]bool
	duration       time.Duration
	stopChan       chan struct{}
	wg             sync.WaitGroup
}

type BufferedCollectorOpt func(*BufferedCollector)

func WithResolvedPolicy(resolved *policy.ResolvedPolicy) (BufferedCollectorOpt, error) {
	// TODO: need a more native way to integrate this part. We don't want to
	// introduce a score type.
	riskMRNs := map[string][]string{}
	keepQrIds := map[string]bool{}
	for _, rj := range resolved.CollectorJob.ReportingJobs {
		if rj.Type == policy.ReportingJob_RISK_FACTOR {
			for k := range rj.ChildJobs {
				cjob := resolved.CollectorJob.ReportingJobs[k]
				if resolved.CollectorJob.RiskMrns == nil {
					return nil, errors.New("missing query MRNs in resolved policy")
				}

				mrns := resolved.CollectorJob.RiskMrns[cjob.Uuid]
				if mrns == nil {
					return nil, errors.New("missing query MRNs for job uuid=" + cjob.Uuid + " checksum=" + cjob.Checksum)
				}

				riskMRNs[cjob.QrId] = append(riskMRNs[cjob.QrId], mrns.Items...)
			}
		} else {
			for k := range rj.ChildJobs {
				cjob := resolved.CollectorJob.ReportingJobs[k]
				keepQrIds[cjob.QrId] = true
			}
		}
	}

	return func(b *BufferedCollector) {
		b.resolvedPolicy = resolved
		b.riskMRNs = riskMRNs
		b.keepQrIds = keepQrIds
	}, nil
}

func NewBufferedCollector(ctx context.Context, collector *PolicyServiceCollector, opts ...BufferedCollectorOpt) *BufferedCollector {
	c := &BufferedCollector{
		ctx:       ctx,
		results:   map[string]*llx.RawResult{},
		scores:    map[string]*policy.Score{},
		duration:  5 * time.Second,
		collector: collector,
		stopChan:  make(chan struct{}),
	}

	for i := range opts {
		opts[i](c)
	}

	c.run()
	return c
}

func (c *BufferedCollector) consumeRisk(score *policy.Score, risks map[string]bool) bool {
	riskMRNs, ok := c.riskMRNs[score.QrId]
	if !ok {
		return false
	}

	for _, riskMRN := range riskMRNs {
		isDetected := score.Value == 100
		risks[riskMRN] = risks[riskMRN] || isDetected
	}
	return true
}

func (c *BufferedCollector) run() {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		defer health.ReportPanic("cnspec", cnspec.Version, cnspec.Build)

		done := false
		results := []*llx.RawResult{}
		scores := []*policy.Score{}
		risksIdx := map[string]bool{}
		for {

			c.lock.Lock()

			for _, rr := range c.results {
				results = append(results, rr)
			}
			for k := range c.results {
				delete(c.results, k)
			}

			for _, s := range c.scores {
				consumedRisk := c.consumeRisk(s, risksIdx)
				shouldKeepIfConsumed := c.keepQrIds[s.QrId]
				if !consumedRisk || shouldKeepIfConsumed {
					scores = append(scores, s)
				}
			}
			for k := range c.scores {
				delete(c.scores, k)
			}

			c.lock.Unlock()

			if len(results) > 0 {
				c.collector.Sink(c.ctx, results, nil, nil, false)
				results = results[:0]
			}

			if done {
				risks := listScoredRisks(risksIdx)
				c.collector.updateRiskScores(c.resolvedPolicy, scores, risks)
				c.collector.Sink(c.ctx, nil, scores, risks, done)
				scores = scores[:0]
				risksIdx = map[string]bool{}
			}

			if done {
				return
			}

			// TODO: we should only use one timer
			timer := time.NewTimer(c.duration)
			select {
			case <-c.stopChan:
				done = true
			case <-timer.C:
			}
			timer.Stop()
		}
	}()
}

func listScoredRisks(risksIdx map[string]bool) []*policy.ScoredRiskFactor {
	risks := make([]*policy.ScoredRiskFactor, len(risksIdx))
	ri := 0
	for mrn, isDetected := range risksIdx {
		risks[ri] = &policy.ScoredRiskFactor{
			Mrn:        mrn,
			IsDetected: isDetected,
		}
		ri++
	}
	return risks
}

func (c *BufferedCollector) FlushAndStop() {
	close(c.stopChan)
	c.wg.Wait()
}

func (c *BufferedCollector) SinkData(results []*llx.RawResult) {
	c.lock.Lock()
	defer c.lock.Unlock()
	for _, rr := range results {
		c.results[rr.CodeID] = rr
	}
}

func (c *BufferedCollector) SinkScore(scores []*policy.Score) {
	c.lock.Lock()
	defer c.lock.Unlock()
	for _, s := range scores {
		// We are making a clone of s. This safe-guards us is a
		// consumer of s decides to mutate it
		c.scores[s.QrId] = s.CloneVT()
	}
}

type PolicyServiceCollector struct {
	assetMrn string
	resolver policy.PolicyResolver
}

func NewPolicyServiceCollector(assetMrn string, resolver policy.PolicyResolver) *PolicyServiceCollector {
	return &PolicyServiceCollector{
		assetMrn: assetMrn,
		resolver: resolver,
	}
}

func toResult(assetMrn string, rr *llx.RawResult) *llx.Result {
	v := rr.Result()
	if v.Data.Size() > MAX_DATAPOINT {
		log.Warn().
			Str("asset", assetMrn).
			Str("id", rr.CodeID).
			Msg("executor.scoresheet> not storing datafield because it is too large")

		v = &llx.Result{
			Error:  "datafield was removed because it is too large",
			CodeId: v.CodeId,
		}
	}
	return v
}

func (c *PolicyServiceCollector) updateRiskScores(resolvedPolicy *policy.ResolvedPolicy, scores []*policy.Score, scoredRisks []*policy.ScoredRiskFactor) {
	assetRisks := []*policy.ScoredRiskInfo{}
	resourceRisks := map[string][]*policy.ScoredRiskInfo{}

	for i := range scoredRisks {
		scoredRisk := scoredRisks[i]
		risk, ok := resolvedPolicy.CollectorJob.RiskFactors[scoredRisk.Mrn]
		if !ok {
			log.Debug().Str("riskMrn", scoredRisk.Mrn).Msg("failed to find risk factor in collector")
			continue
		}
		risk.Mrn = scoredRisk.Mrn

		if risk.Scope == policy.ScopeType_ASSET {
			assetRisks = append(assetRisks, &policy.ScoredRiskInfo{
				RiskFactor:       risk,
				ScoredRiskFactor: scoredRisk,
			})
		} else if risk.Scope == policy.ScopeType_RESOURCE || risk.Scope == policy.ScopeType_SOFTWARE_AND_RESOURCE {
			for ri := range risk.Resources {
				name := risk.Resources[ri].Name
				if name == "" {
					log.Warn().Str("mrn", scoredRisk.Mrn).Msg("ignoring resource-level risk factor with empty resource name")
					continue
				}
				resourceRisks[name] = append(resourceRisks[name], &policy.ScoredRiskInfo{
					RiskFactor:       risk,
					ScoredRiskFactor: scoredRisk,
				})
			}
		}
	}

	policy.SortScoredRiskInfo(assetRisks)

	names := resolvedPolicy.EnumerateQueryResources()
	csumsIdx := map[string][]*policy.ScoredRiskInfo{}
	for name, risks := range resourceRisks {
		csums := names[name]
		policy.SortScoredRiskInfo(risks)
		for _, csum := range csums {
			csumsIdx[csum] = risks
		}
	}

	for i := range scores {
		score := scores[i]
		risks := csumsIdx[score.QrId]
		policy.AdjustRiskScore(score, assetRisks, risks)
	}
}

func (c *PolicyServiceCollector) Sink(ctx context.Context, results []*llx.RawResult, scores []*policy.Score, risks []*policy.ScoredRiskFactor, isDone bool) {
	// If we have nothing to send and also this is not the last batch, we just skip
	if len(results) == 0 && len(scores) == 0 && len(risks) == 0 && !isDone {
		return
	}

	onTooLargeFn := func(item *llx.Result, msgSize int) {
		log.Warn().Msgf("Data %s %d exceeds maximum message size", item.CodeId, msgSize)
	}
	sendFn := func(chunk []*llx.Result) error {
		log.Debug().Msg("Sending datapoints")
		resultsToSend := make(map[string]*llx.Result, len(chunk))
		for _, rr := range chunk {
			resultsToSend[rr.CodeId] = rr
		}
		_, err := c.resolver.StoreResults(ctx, &policy.StoreResultsReq{
			AssetMrn:       c.assetMrn,
			Data:           resultsToSend,
			IsPreprocessed: true,
			IsLastBatch:    isDone,
		})
		if err != nil {
			log.Error().Err(err).Msg("failed to send datapoints")
		}
		return nil
	}
	if len(results) > 0 {
		llxResults := make([]*llx.Result, len(results))
		for i, rr := range results {
			llxResults[i] = toResult(c.assetMrn, rr)
		}

		err := iox.ChunkMessages(sendFn, cnquery.GetDisableMaxLimit(), onTooLargeFn, llxResults...)
		if err != nil {
			log.Error().Err(err).Msg("failed to send datapoints")
		}
	}

	if len(scores) > 0 || len(risks) > 0 {
		log.Debug().Msg("Sending scores")
		_, err := c.resolver.StoreResults(ctx, &policy.StoreResultsReq{
			AssetMrn:       c.assetMrn,
			Scores:         scores,
			Risks:          risks,
			IsPreprocessed: true,
			IsLastBatch:    isDone,
		})
		if err != nil {
			log.Error().Err(err).Msg("failed to send datapoints and scores")
			health.ReportError("cnspec", cnspec.Version, cnspec.Build, err.Error())
		}
	}
}

type FuncCollector struct {
	SinkDataFunc  func(results []*llx.RawResult)
	SinkScoreFunc func(scores []*policy.Score)
}

func (c *FuncCollector) SinkData(results []*llx.RawResult) {
	if len(results) == 0 || c.SinkDataFunc == nil {
		return
	}
	c.SinkDataFunc(results)
}

func (c *FuncCollector) SinkScore(scores []*policy.Score) {
	if len(scores) == 0 || c.SinkScoreFunc == nil {
		return
	}
	c.SinkScoreFunc(scores)
}
