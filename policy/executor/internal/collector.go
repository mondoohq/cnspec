// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package internal

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnspec/v10/policy"
	"google.golang.org/protobuf/proto"
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
	results   map[string]*llx.RawResult
	scores    map[string]*policy.Score
	lock      sync.Mutex
	collector *PolicyServiceCollector
	duration  time.Duration
	stopChan  chan struct{}
	wg        sync.WaitGroup
}

type BufferedCollectorOpt func(*BufferedCollector)

func NewBufferedCollector(collector *PolicyServiceCollector, opts ...BufferedCollectorOpt) *BufferedCollector {
	c := &BufferedCollector{
		results:   map[string]*llx.RawResult{},
		scores:    map[string]*policy.Score{},
		duration:  5 * time.Second,
		collector: collector,
		stopChan:  make(chan struct{}),
	}
	c.run()
	return c
}

func (c *BufferedCollector) run() {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()

		done := false
		results := []*llx.RawResult{}
		scores := []*policy.Score{}
		for {

			c.lock.Lock()
			for _, rr := range c.results {
				results = append(results, rr)
			}
			for k := range c.results {
				delete(c.results, k)
			}

			for _, s := range c.scores {
				scores = append(scores, s)
			}
			for k := range c.scores {
				delete(c.scores, k)
			}
			c.lock.Unlock()

			// If we have something to send or this is the last batch, we do a Sink
			if len(scores) > 0 || len(results) > 0 || done {
				c.collector.Sink(results, scores, done)
			}

			results = results[:0]
			scores = scores[:0]

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
		c.scores[s.QrId] = proto.Clone(s).(*policy.Score)
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

func (c *PolicyServiceCollector) toResult(rr *llx.RawResult) *llx.Result {
	v := rr.Result()
	if v.Data.Size() > MAX_DATAPOINT {
		log.Warn().
			Str("asset", c.assetMrn).
			Str("id", rr.CodeID).
			Msg("executor.scoresheet> not storing datafield because it is too large")

		v = &llx.Result{
			Error:  "datafield was removed because it is too large",
			CodeId: v.CodeId,
		}
	}
	return v
}

func (c *PolicyServiceCollector) Sink(results []*llx.RawResult, scores []*policy.Score, isDone bool) {
	// If we have nothing to send and also this is not the last batch, we just skip
	if len(results) == 0 && len(scores) == 0 && !isDone {
		return
	}
	resultsToSend := make(map[string]*llx.Result, len(results))
	for _, rr := range results {
		resultsToSend[rr.CodeID] = c.toResult(rr)
	}
	log.Debug().Msg("Sending datapoints and scores")
	_, err := c.resolver.StoreResults(context.Background(), &policy.StoreResultsReq{
		AssetMrn:       c.assetMrn,
		Data:           resultsToSend,
		Scores:         scores,
		IsPreprocessed: true,
		IsLastBatch:    isDone,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to send datapoints and scores")
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
