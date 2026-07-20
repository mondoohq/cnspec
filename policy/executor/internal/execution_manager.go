// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package internal

import (
	"fmt"
	"os"
	goruntime "runtime"
	"runtime/debug"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnspec/v13"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/upstream/health"
)

const MEM_DEBUG_ENV = "MEM_DEBUG"

var memDebug = false

func init() {
	memDebug = os.Getenv(MEM_DEBUG_ENV) == "1"
}

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

	dumpDatapoints bool

	// durationCollectors receive the wall-clock execution time of each
	// query after the executor returns
	durationCollectors []DurationCollector
}

type runQueueItem struct {
	codeBundle *llx.CodeBundle
	// props returns the query's property values. It is invoked when the item
	// is dequeued (not when it is queued), so a transient nil property that
	// was upgraded to its real value while waiting in the queue is picked up.
	// It may be nil for queries without properties.
	props func() map[string]*llx.Result
}

func newExecutionManager(runtime llx.Runtime, runQueue chan runQueueItem,
	resultChan chan *llx.RawResult, timeout time.Duration, dumpDatapoints bool,
	durationCollectors []DurationCollector,
) *executionManager {
	return &executionManager{
		runQueue:           runQueue,
		runtime:            runtime,
		resultChan:         resultChan,
		errChan:            make(chan error, 1),
		stopChan:           make(chan struct{}),
		timeout:            timeout,
		dumpDatapoints:     dumpDatapoints,
		durationCollectors: durationCollectors,
	}
}

func (em *executionManager) Start() {
	em.wg.Add(1)
	go func() {
		defer em.wg.Done()
		// current is the code bundle being executed; the deferred panic
		// handler snapshots it at crash time so the report carries WHICH
		// query was running, not just where the engine died. The recover
		// stays at the goroutine top so the stacktrace points at the
		// panic site. Instead of crashing the process, the panic is
		// reported upstream and surfaced as an unrecoverable execution
		// error, mirroring the executeCodeBundle error path below.
		// Like that path, the goroutine deliberately exits and takes the
		// whole pipeline for this scan with it: after a panic the runtime
		// state can't be trusted, so we don't keep executing the remaining
		// queries.
		var current *llx.CodeBundle
		defer func() {
			r := recover()
			if r == nil {
				return
			}
			var tags map[string]string
			if current != nil {
				tags = health.QueryPanicTags(current.CodeV2.GetId(), current.Source)
			}
			stack := debug.Stack()
			health.ReportRecoveredPanic("cnspec", cnspec.Version, cnspec.Build, r, stack, tags)
			log.Error().
				Str("stacktrace", string(stack)).
				Msgf("recovered from panic during query execution: %v", r)
			select {
			case em.errChan <- fmt.Errorf("panic during query execution: %v", r):
			default:
			}
		}()
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
				var itemProps map[string]*llx.Result
				if item.props != nil {
					itemProps = item.props()
				}
				for k, r := range itemProps {
					if r.Error != "" {
						// This case is tricky to handle. If we cannot run the query at
						// all, it's unclear what to report for the datapoint. If we
						// report them in, then another query can't report them, at least
						// with the way things are right now. If we don't report them,
						// things will wait around for datapoint results that will never
						// arrive.
						errMsg = "property " + k + " errored: " + r.Error
						break
					}
					props[k] = r.Data
				}

				current = item.codeBundle
				err := em.executeCodeBundle(item.codeBundle, props, errMsg)
				current = nil
				if err != nil {
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
		tr := log.Trace().Str("codeID", rr.CodeID)
		if em.dumpDatapoints {
			tr.Interface("data", rr.Data)
		}
		tr.Msg("received result from executor")
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
				// The query cannot run; broadcast a typed placeholder for
				// every codepoint so downstream consumers don't wait forever.
				// Datapoint checksums are content-addressed and shared across
				// queries, so a healthy query may still report a real result
				// for the same checksum. Consumers let executed results
				// override these placeholders and never let a placeholder
				// override an executed result (see queryRunError).
				sendResult(&llx.RawResult{
					CodeID: checksum,
					Data: &llx.RawData{
						Error: &queryRunError{
							originCodeID: codeBundle.CodeV2.GetId(),
							err:          errors.New(errMsg),
						},
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
	startTime := time.Now()
	log.Debug().Str("qrid", codeID).Msg("starting query execution")
	defer func() {
		duration := time.Since(startTime)
		log.Debug().
			Str("qrid", codeID).
			Dur("duration", duration).
			Msg("finished query execution")
		for _, c := range em.durationCollectors {
			c.SinkDuration(codeID, duration)
		}
	}()
	// TODO(jaym): sendResult may not be correct. We may need to fill in the
	// checksum
	x, err := llx.NewExecutorV2(codeBundle.CodeV2, em.runtime, props, sendResult)
	if err == nil {
		_ = x.Run()
	}

	if memDebug {
		var m goruntime.MemStats
		goruntime.ReadMemStats(&m)

		log.Warn().Uint64("allocated", bToMb(m.Alloc)).Str("qrid", codeID).Msg("memory allocated after query")
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

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}

var errQueryTimeout = errors.New("query execution timed out")
