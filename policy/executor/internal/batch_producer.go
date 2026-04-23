// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package internal

import (
	"sync"

	"go.mondoo.com/cnspec/v13/policy"
)

// BatchScoreProducer emits a fixed batch of pre-computed scores into the
// graph and then closes its output. Used for rescoring: each input score
// is fanned out to its leaf reporting job's notify targets (parent
// ReportingJobNodes that already have childScores slots from
// rj.ChildJobs) and to the score collector — no node state mutation, no
// slot reservation.
type BatchScoreProducer struct {
	output   chan addressedEnvelope
	errChan  chan error
	batch    []addressedEnvelope
	stopOnce sync.Once
	stopChan chan struct{}
	wg       sync.WaitGroup
}

// NewBatchScoreProducer builds a BatchScoreProducer from the resolved
// policy's reporting jobs and a map of pre-computed scores keyed by
// QrId. Honors the "root" → assetMrn rename that addReportingJobNode
// performs so input keyed by the asset MRN finds its leaf.
func NewBatchScoreProducer(scores map[string]*policy.Score, rjs map[string]*policy.ReportingJob, assetMrn string) *BatchScoreProducer {
	leafByQrId := make(map[string]*policy.ReportingJob, len(rjs))
	for _, rj := range rjs {
		qid := rj.QrId
		if qid == "root" {
			qid = assetMrn
		}
		leafByQrId[qid] = rj
	}

	batch := make([]addressedEnvelope, 0, len(scores)*2)
	for qrId, score := range scores {
		leaf, ok := leafByQrId[qrId]
		if !ok {
			continue
		}
		s := score.CloneVT()
		s.DataCompletion = 100
		s.ScoreCompletion = 100
		for _, parentUuid := range leaf.Notify {
			batch = append(batch, addressedEnvelope{
				to:   parentUuid,
				from: leaf.Uuid,
				env:  envelope{score: s},
			})
		}
		batch = append(batch, addressedEnvelope{
			to:   ScoreCollectorID,
			from: leaf.Uuid,
			env:  envelope{score: s},
		})
	}

	return &BatchScoreProducer{
		output:   make(chan addressedEnvelope, len(batch)),
		errChan:  make(chan error, 1),
		batch:    batch,
		stopChan: make(chan struct{}),
	}
}

func (p *BatchScoreProducer) Output() <-chan addressedEnvelope { return p.output }
func (p *BatchScoreProducer) Err() <-chan error                { return p.errChan }

func (p *BatchScoreProducer) Start() {
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		for _, e := range p.batch {
			select {
			case p.output <- e:
			case <-p.stopChan:
				return
			}
		}
		close(p.output)
	}()
}

func (p *BatchScoreProducer) Stop() {
	p.stopOnce.Do(func() {
		close(p.stopChan)
		p.wg.Wait()
	})
}
