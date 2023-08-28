// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package internal

import (
	"errors"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/llx"
)

type executionManager struct {
	runtime llx.Runtime
	// runQueue is the channel the execution manager will read
	// items that need to be run from
	runQueue chan runQueueItem
	// resultChan is the channel the execution manager will write
	// results to
	resultChan chan *llx.RawResult
	// errChan is used to signal an unrecoverable error. The execution
	// manager writes to this channel
	errChan chan error
	// timeout is the amount of time the executor will wait for a query
	// to return all the results after
	timeout time.Duration
	// stopChan is a channel that is closed when a stop is requested
	stopChan chan struct{}
	wg       sync.WaitGroup
}

type runQueueItem struct {
	codeBundle *llx.CodeBundle
	props      map[string]*llx.Result
}

func newExecutionManager(runtime llx.Runtime, runQueue chan runQueueItem,
	resultChan chan *llx.RawResult, timeout time.Duration,
) *executionManager {
	return &executionManager{
		runQueue:   runQueue,
		runtime:    runtime,
		resultChan: resultChan,
		errChan:    make(chan error, 1),
		stopChan:   make(chan struct{}),
		timeout:    timeout,
	}
}

func (em *executionManager) Start() {
	em.wg.Add(1)
	go func() {
		defer em.wg.Done()
		for {
			// Prioritize stopChan
			select {
			case <-em.stopChan:
				return
			default:
			}

			select {
			case item, ok := <-em.runQueue:
				if !ok {
					return
				}
				props := make(map[string]*llx.Primitive)
				errMsg := ""
				for k, r := range item.props {
					if r.Error != "" {
						// This case is tricky to handle. If we cannot run the query at
						// all, its unclear what to report for the datapoint. If we
						// report them in, then another query cant report them, at least
						// with the way things are right now. If we don't report them,
						// things will wait around for datapoint results that will never
						// arrive.
						errMsg = "property " + k + " errored: " + r.Error
						break
					}
					props[k] = r.Data
				}

				if err := em.executeCodeBundle(item.codeBundle, props, errMsg); err != nil {
					// an error is returned if we cannot execute a query. This happens
					// if the lumi runtime doesn't report back expected data, there is
					// a problem with the lumi runtime, or the query is somehow invalid.
					// We need to give up here because the underlying runtime is in a bad
					// state and/or we will not be able to report certain datapoints and
					// we cannot be confident about which ones
					select {
					case em.errChan <- err:
					default:
					}
					return
				}
			case <-em.stopChan:
				return
			}
		}
	}()
}

func (em *executionManager) Err() chan error {
	return em.errChan
}

func (em *executionManager) Stop() {
	close(em.stopChan)
	em.wg.Wait()
}

func (em *executionManager) executeCodeBundle(codeBundle *llx.CodeBundle, props map[string]*llx.Primitive, errMsg string) error {
	wg := NewWaitGroup()

	sendResult := func(rr *llx.RawResult) {
		log.Trace().Str("codeID", rr.CodeID).Msg("received result from executor")
		wg.Done(rr.CodeID)
		select {
		case em.resultChan <- rr:
		case <-em.stopChan:
		}
	}

	checksums := map[string]struct{}{}
	// Find the list of things we must wait for before execution of this codebundle is considered done
	for _, checksum := range CodepointChecksums(codeBundle) {
		if _, ok := checksums[checksum]; !ok {
			checksums[checksum] = struct{}{}
			// We must use a synchronization primitive because the llx.Run callback
			// is not guaranteed to happen in a single thread
			wg.Add(checksum)
			if errMsg != "" {
				// TODO: this is not entirely correct when looking at things as a whole.
				// Its possible that another query executing will produce a non error.
				// However, datapoint nodes take the first data that was reported. This
				// issue exists in general for any query that errors
				sendResult(&llx.RawResult{
					CodeID: checksum,
					Data: &llx.RawData{
						Error: errors.New(errMsg),
					},
				})
			}
		}
	}

	if errMsg != "" {
		return nil
	}

	var err error

	codeID := codeBundle.CodeV2.GetId()
	log.Debug().Str("qrid", codeID).Msg("starting query execution")
	defer func() {
		log.Debug().Str("qrid", codeID).Msg("finished query execution")
	}()
	// TODO(jaym): sendResult may not be correct. We may need to fill in the
	// checksum
	x, err := llx.NewExecutorV2(codeBundle.CodeV2, em.runtime, props, sendResult)
	if err == nil {
		x.Run()
	}

	if err != nil {
		return err
	}

	execDoneChan := make(chan struct{})
	go func() {
		wg.Wait()
		close(execDoneChan)
	}()

	var errOut error

	timer := time.NewTimer(em.timeout)
	defer timer.Stop()
	select {
	case <-timer.C:
		log.Error().Dur("timeout", em.timeout).Str("qrid", codeID).Msg("execution timed out")
		errOut = errQueryTimeout
	case <-execDoneChan:
	}

	unreported := wg.Decommission()
	if len(unreported) > 0 {
		log.Warn().Strs("missing", unreported).Str("qrid", codeID).Msg("unreported datapoints")
	}

	return errOut
}

var errQueryTimeout = errors.New("query execution timed out")
