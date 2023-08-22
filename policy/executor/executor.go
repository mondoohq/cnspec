// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package executor

import (
	"errors"
	"fmt"
	"sync"
	"time"

	vrs "github.com/hashicorp/go-version"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/logger"
	"go.mondoo.com/cnquery/mqlc"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/types"
	"go.mondoo.com/cnspec"
	"go.mondoo.com/cnspec/policy/executor/internal"
)

// Executor helps you run multiple pieces of mondoo code and process results
type Executor struct {
	schema        *resources.Schema
	runtime       *resources.Runtime
	RunningCode   *RunningCode
	ScoreResults  *ScoreResults
	Results       *RawResults
	CodeID2Bundle *CodeID2Bundle
	bundleErrors  *llx.DictGroupTracker
	executors     map[string]*llx.MQLExecutorV2
	watchers      *watcherMap
	waitGroup     *internal.WaitGroup
}

// Results is a thread-safe map of raw results
type RawResults struct{ sync.Map }

// Store sets the value for a key.
func (c *RawResults) Store(k string, v *llx.RawResult) {
	c.Map.Store(k, v)
}

// Load returns the value stored in the map for a key, or nil if no value is present. The ok result indicates whether value was found in the map.
func (c *RawResults) Load(k string) (*llx.RawResult, bool) {
	res, ok := c.Map.Load(k)
	if !ok {
		return nil, ok
	}
	return res.(*llx.RawResult), ok
}

// Range calls f sequentially for each key and value present in the map. If f returns false, range stops the iteration.
func (c *RawResults) Range(f func(k string, v *llx.RawResult) bool) {
	c.Map.Range(func(k, v interface{}) bool {
		return f(k.(string), v.(*llx.RawResult))
	})
}

// ScoreResults is a thread-safe map of true/false results for queries
type ScoreResults struct{ sync.Map }

// Store sets the value for a key.
func (c *ScoreResults) Store(k string, v int) {
	c.Map.Store(k, v)
}

// Load returns the value stored in the map for a key, or nil if no value is present. The ok result indicates whether value was found in the map.
func (c *ScoreResults) Load(k string) (int, bool) {
	res, ok := c.Map.Load(k)
	if !ok {
		return 0, ok
	}
	return res.(int), ok
}

// CodeID2Bundle is a thread-safe map that lists all bundles for a given code ID / checksum
type CodeID2Bundle struct{ sync.Map }

// Store sets the value for a key.
func (c *CodeID2Bundle) Store(k string, v map[string]struct{}) {
	c.Map.Store(k, v)
}

// Load returns the value stored in the map for a key, or nil if no value is present. The ok result indicates whether value was found in the map.
func (c *CodeID2Bundle) Load(k string) (map[string]struct{}, bool) {
	res, ok := c.Map.Load(k)
	if !ok {
		return nil, ok
	}
	return res.(map[string]struct{}), ok
}

// LoadOrStore returns the existing value for the key if present. Otherwise, it stores and returns the given value.
// The loaded result is true if the value was loaded, false if stored.
func (c *CodeID2Bundle) LoadOrStore(k string, v map[string]struct{}) (map[string]struct{}, bool) {
	res, loaded := c.Map.LoadOrStore(k, v)

	return res.(map[string]struct{}), loaded
}

// RunningCode is a thread-safe map that lists code bundles for given IDs
type RunningCode struct{ sync.Map }

// Store sets the value for a key.
func (c *RunningCode) Store(k string, v *llx.CodeBundle) {
	c.Map.Store(k, v)
}

// Load returns the value stored in the map for a key, or nil if no value is present. The ok result indicates whether value was found in the map.
func (c *RunningCode) Load(k string) (*llx.CodeBundle, bool) {
	res, ok := c.Map.Load(k)
	if !ok {
		return nil, ok
	}
	return res.(*llx.CodeBundle), ok
}

// Range calls f sequentially for each key and value present in the map. If f returns false, range stops the iteration.
func (c *RunningCode) Range(f func(k string, v *llx.CodeBundle) bool) {
	c.Map.Range(func(k, v interface{}) bool {
		return f(k.(string), v.(*llx.CodeBundle))
	})
}

type (
	watcherMap  struct{ sync.Map }
	watcherFunc func(*llx.RawResult)
)

func (w *watcherMap) Load(k string) (watcherFunc, bool) {
	v, ok := w.Map.Load(k)
	if !ok {
		return nil, ok
	}

	return v.(watcherFunc), ok
}

func (w *watcherMap) Store(k string, v watcherFunc) {
	w.Map.Store(k, v)
}

func (w *watcherMap) Delete(k string) {
	w.Map.Delete(k)
}

func (w *watcherMap) Range(f func(k string, v watcherFunc) bool) {
	w.Map.Range(func(k, v interface{}) bool {
		return f(k.(string), v.(watcherFunc))
	})
}

// New creates a new Executor
// schema == nil will use the default schema
// runtime must be defined
func New(schema *resources.Schema, runtime *resources.Runtime) *Executor {
	if runtime == nil {
		panic("cannot have executor initialized with resources.Runtime == nil")
	}
	if schema == nil {
		panic("cannot have executor initialize with resources.Schema == nil")
	}

	res := &Executor{
		schema:        schema,
		runtime:       runtime,
		ScoreResults:  &ScoreResults{},
		Results:       &RawResults{},
		CodeID2Bundle: &CodeID2Bundle{},
		bundleErrors:  &llx.DictGroupTracker{},
		waitGroup:     internal.NewWaitGroup(),
	}
	res.DecomissionAndReset()

	return res
}

// AddWatcher to the executor whenever we have a new result
func (e *Executor) AddWatcher(watcherID string, f func(res *llx.RawResult)) {
	e.watchers.Store(watcherID, f)
}

// RemoveWatcher from the executor
func (e *Executor) RemoveWatcher(watcherID string) {
	e.watchers.Delete(watcherID)
}

// AreAllResultsCollected returns true if all registered results have been collected
// it looks at all instances of running code to determine this
func (e *Executor) AreAllResultsCollected() bool {
	allCollected := true

	e.RunningCode.Range(func(k string, _ *llx.CodeBundle) bool {
		if _, ok := e.ScoreResults.Load(k); !ok {
			allCollected = false
			return false
		}
		return true
	})

	return allCollected
}

// Compile a given code with the default schema
func (e *Executor) Compile(code string, props map[string]*llx.Primitive) (*llx.CodeBundle, error) {
	return mqlc.Compile(code, props, mqlc.NewConfig(e.schema, cnquery.DefaultFeatures))
}

func (e *Executor) AddCode(code string, props map[string]*llx.Primitive) (*llx.CodeBundle, error) {
	codeBundle, err := e.Compile(code, props)
	if err != nil {
		return nil, err
	}
	if err := e.AddCodeBundle(codeBundle, props); err != nil {
		return nil, err
	}
	return codeBundle, nil
}

// AddCode to the executor
func (e *Executor) AddCodeBundle(codeBundle *llx.CodeBundle, props map[string]*llx.Primitive) error {
	codeID := codeBundle.CodeV2.Id

	org, ok := e.RunningCode.Load(codeID)
	if ok && org.CodeV2.Id == codeBundle.CodeV2.Id {
		return nil
	}

	if ok {
		log.Warn().Str("codeID", codeID).Msg("executor> content for codeID doesn't match code content")
		e.RemoveCode(codeID, codeBundle.Source)
	}

	e.RunningCode.Store(codeID, codeBundle)
	e.bundleErrors.ClearGroup(codeID)
	if len(codeBundle.CodeV2.Entrypoints())+len(codeBundle.CodeV2.Datapoints()) == 0 {
		e.updateBundle(codeID)
	} else {
		runSafe, runerr := isRunSafe(codeBundle)
		e.incrementWaitGroup(codeBundle)

		// we extract a closure-based copy of the waitgroup. if we run into timeouts
		// the llx callback may happen after the waitgroup has been decommissioned
		// in that case due to the closure it will reference the old waitgroup
		waitGroup := e.waitGroup
		results := e.Results
		scoreResults := e.ScoreResults

		executor, err := llx.NewExecutorV2(codeBundle.CodeV2, e.runtime, props, func(res *llx.RawResult) {
			e.onResult(res, waitGroup, results, scoreResults)
		})
		if err != nil {
			return err
		}

		if !runSafe {
			executor.NoRun(runerr)
		} else {
			err = executor.Run()
		}

		if err != nil {
			return err
		}
		e.executors[codeID] = executor
	}
	return nil
}

func isRunSafe(codeBundle *llx.CodeBundle) (bool, error) {
	if codeBundle.MinMondooVersion != "" {
		requiredVer := codeBundle.MinMondooVersion
		currentVer := cnspec.GetCoreVersion()
		if currentVer == "unstable" {
			// Probably running locally since ldflags didn't config version
			// Entering yolo mode (its probably safe!)
			return true, nil
		}
		reqMin, err := vrs.NewVersion(requiredVer)
		curMin, err1 := vrs.NewVersion(currentVer)
		if err == nil && err1 == nil && curMin.LessThan(reqMin) {
			return false, fmt.Errorf("Unable to run query, cnspec version %s required", requiredVer)
		}
	}
	return true, nil
}

func (e *Executor) onResult(res *llx.RawResult, waitGroup *internal.WaitGroup, results *RawResults, scoreResults *ScoreResults) {
	log.Trace().Msg("executor> got result")
	logger.TraceJSON(res)

	if res.CodeID == "" {
		log.Error().Msg("executor> received a result without a CodeID")

		// note: we do not reduce the waitgroup here. this state is definitely
		// not expected and should not happen, but if we reduce the waitgroup
		// we may run into a negative waitgroup panic. if we don't reduce it
		// the loop won't exit, but it will time out (which we are monitoring)
		// and hannah is awesome.
		return
	}

	score, _ := res.Data.Score()
	oldScore, existing := scoreResults.Load(res.CodeID)
	if !existing {
		scoreResults.Store(res.CodeID, score)
		results.Store(res.CodeID, res)
	} else if oldScore != score {
		log.Trace().Str("codeID", res.CodeID).Msg("executor> result score changed")
		scoreResults.Store(res.CodeID, score)
		results.Store(res.CodeID, res)
	}

	// deduplicate error reporting for bundles
	if !e.isKnownError(res.CodeID, res.Data.Error) {
		e.watchers.Range(func(w string, f watcherFunc) bool {
			log.Trace().Str("watcher", w).Msg("executor> send result to watcher")
			f(res)
			return true
		})
	}

	// has to be run after the watchers are triggered for this result,
	// as it may trigger another round of watchers when all results are received.
	// don't run it last as it may cause the code to exit prematurely if the
	// waitgroup is closed beforehand
	if !waitGroup.IsDecommissioned() {
		// TODO(jaym): What if the waitGroup is decommissioned after we check
		e.tryUpdateBundles(res.CodeID)
	}

	// The reason we close the wait group last is for test and execution safety
	// while the results are being sent to all the watcher they could mistakenly think
	// they have all the data (in a RunOnce scenario) without having actually
	// received all of it. (Happens very rarely, but it does happen)
	// To prevent this, we close the waitGroup last.
	if !existing {
		waitGroup.Done(res.CodeID)
		stats := waitGroup.Stats()
		log.Trace().
			Int("total", stats.NumAdded).
			Int("completed", stats.NumDone).
			Str("code", res.CodeID).
			Msg("executor> first time result received")

	}
}

func (e *Executor) isKnownError(codeID string, err error) bool {
	if err == nil {
		return false
	}

	errorStr := err.Error()
	var isUnknown bool

	bundles, _ := e.CodeID2Bundle.Load(codeID)
	for bundleID := range bundles {
		exists := e.bundleErrors.CheckOrSet(bundleID, errorStr)
		if exists {
			continue
		}

		isUnknown = true
	}

	return !isUnknown
}

func (e *Executor) tryUpdateBundles(codeID string) {
	bundles, ok := e.CodeID2Bundle.Load(codeID)
	if !ok {
		log.Error().Msg("executor> cannot find bundle for this code result")
		return
	}

	for bundleID := range bundles {
		e.updateBundle(bundleID)
	}
}

func (e *Executor) updateBundle(bundleID string) {
	bundle, _ := e.RunningCode.Load(bundleID)
	if bundle == nil {
		log.Error().Str("bundle", bundleID).Msg("executor> bundle is already decommissioned")
		return
	}

	allFound := true
	allSkipped := true
	allTrue := true
	foundError := false
	var scoreFound *llx.RawData
	var score int

	var errorsMsg string
	entrypoints := bundle.CodeV2.Entrypoints()
	for i := range entrypoints {
		ref := entrypoints[i]
		checksum := bundle.CodeV2.Checksums[ref]
		cur, ok := e.Results.Load(checksum)
		if !ok {
			allFound = false
			break
		}

		if cur.Data.Error != nil {
			allSkipped = false
			foundError = true
			// append ; if we accumulate errors
			if errorsMsg != "" {
				errorsMsg += "; "
			}
			errorsMsg += cur.Data.Error.Error()
		} else if cur.Data.Value != nil {
			allSkipped = false

			if v, ok := cur.Data.Score(); ok {
				scoreFound = cur.Data
				score = v
			} else if truthy, _ := cur.Data.IsTruthy(); !truthy {
				allTrue = false
			}
		}
	}

	if allFound {
		log.Debug().
			Str("bundle", bundleID).
			Bool("succeeded", allTrue).
			Bool("skipped", allSkipped).
			Bool("hasError", foundError).
			Msg("executor> computed bundle result")
		res := &llx.RawResult{
			CodeID: bundleID,
		}

		if foundError {
			e.ScoreResults.Store(bundleID, 0)
		} else if scoreFound != nil {
			e.ScoreResults.Store(bundleID, score)
		} else if allTrue {
			e.ScoreResults.Store(bundleID, 100)
		} else {
			e.ScoreResults.Store(bundleID, 0)
		}

		if foundError {
			res.Data = &llx.RawData{
				Type:  types.Nil,
				Value: nil,
				Error: errors.New(errorsMsg),
			}
		} else if allSkipped {
			res.Data = llx.NilData
		} else if scoreFound != nil {
			res.Data = scoreFound
		} else if allTrue {
			res.Data = llx.BoolTrue
		} else {
			res.Data = llx.BoolFalse
		}

		e.watchers.Range(func(w string, f watcherFunc) bool {
			log.Trace().Str("watcher", w).Msg("executor> send result to watcher")
			f(res)
			return true
		})
	}
}

// RemoveCode code from executor
func (e *Executor) RemoveCode(codeID string, query string) {
	codeBundle, ok := e.RunningCode.Load(codeID)
	if !ok {
		log.Warn().
			Str("codeID", codeID).
			Str("query", query).
			Msg("executor> cannot find running code while trying to remove codeID")
	}
	e.RunningCode.Delete(codeID)

	executor := e.executors[codeID]
	if executor != nil {
		err := executor.Unregister()
		if err != nil {
			log.Error().Err(err).Msg("executor> failed to unregister executor")
		}
	}

	delete(e.executors, codeID)
	if codeBundle != nil {
		e.decrementWaitGroup(codeBundle)
	}
}

// DecomissionAndReset this executor. Typically used when you are done but
// can also be used to reset the executor on timeouts
func (e *Executor) DecomissionAndReset() {
	e.watchers = &watcherMap{}
	e.ScoreResults = &ScoreResults{}
	e.Results = &RawResults{}

	// close up the existing waitgroup
	if e.waitGroup != nil {
		e.waitGroup.Decommission()
	}

	if e.RunningCode == nil || e.executors == nil {
		e.RunningCode = &RunningCode{}
		e.executors = map[string]*llx.MQLExecutorV2{}
	} else {
		e.RunningCode.Range(func(codeID string, code *llx.CodeBundle) bool {
			e.RemoveCode(codeID, "< decommission >")
			return true
		})
	}

	// create a new wait group after we have removed all remaining code
	// (to make sure any waitgroup reductions are finished before creating it)
	e.waitGroup = internal.NewWaitGroup()

	e.CodeID2Bundle = &CodeID2Bundle{}
	e.bundleErrors.Clear()
}

func (e *Executor) incrementWaitGroup(codeBundle *llx.CodeBundle) {
	// map all codeIDs to the code bundle they belong to

	refs := append(codeBundle.CodeV2.Entrypoints(), codeBundle.CodeV2.Datapoints()...)
	for i := range refs {
		ep := refs[i]
		checksum := codeBundle.CodeV2.Checksums[ep]

		policyGroup, loaded := e.CodeID2Bundle.LoadOrStore(checksum, map[string]struct{}{})
		if !loaded {
			e.waitGroup.Add(checksum)
		}

		// TODO(jaym): Is it possible for multiple writers to policyGroup?
		policyGroup[codeBundle.CodeV2.Id] = struct{}{}
	}

	stats := e.waitGroup.Stats()
	log.Trace().
		Int("total", stats.NumAdded).
		Int("completed", stats.NumDone).
		Msg("executor> code added")
}

func (e *Executor) decrementWaitGroup(codeBundle *llx.CodeBundle) {
	// map all codeIDs to the code bundle they belong to
	refs := append(codeBundle.CodeV2.Entrypoints(), codeBundle.CodeV2.Datapoints()...)
	for i := range refs {
		ep := refs[i]
		checksum := codeBundle.CodeV2.Checksums[ep]

		policyGroup, ok := e.CodeID2Bundle.Load(checksum)
		if !ok {
			log.Warn().
				Str("checksum", checksum).
				Str("codeID", codeBundle.CodeV2.Id).
				Msg("executor> cannot find code entrypoint to decrement waitgroup")
			continue
		}

		delete(policyGroup, codeBundle.CodeV2.Id)
		if len(policyGroup) == 0 {
			e.CodeID2Bundle.Delete(checksum)

			if _, ok := e.ScoreResults.Load(checksum); ok {
				e.waitGroup.Done(checksum)
				e.ScoreResults.Delete(checksum)
			}
		}
	}

	e.ScoreResults.Delete(codeBundle.CodeV2.Id)

	// TODO: we probably need to clean up the wait group more... hard to tell

	stats := e.waitGroup.Stats()
	log.Trace().
		Int("total", stats.NumAdded).
		Int("completed", stats.NumDone).
		Msg("executor> code removed")
}

// WaitForResults and
// returns true if all results were received and
// returns false if we ran into a timeout
func (e *Executor) WaitForResults(timeout time.Duration) bool {
	done := make(chan struct{})

	go func() {
		defer close(done)
		e.waitGroup.Wait()
	}()

	select {
	case <-done:
		return true

	case <-time.After(timeout):
		return false
	}
}

// MissingQuery provides information about the missing query and missing entrypoints
type MissingQuery struct {
	Bundle      *llx.CodeBundle
	Entrypoints []string
}

// MissingQueries lists all queries that have not yet received results
func (e *Executor) MissingQueries() []*MissingQuery {
	queries := []*MissingQuery{}

	e.RunningCode.Range(func(id string, bundle *llx.CodeBundle) bool {
		var found bool
		missing := MissingQuery{
			Bundle:      bundle,
			Entrypoints: []string{},
		}

		_, ok := e.ScoreResults.Load(id)
		if !ok {
			found = true
			missing.Entrypoints = append(missing.Entrypoints, id)
		}

		entrypoints := bundle.CodeV2.Entrypoints()
		for i := range entrypoints {
			ep := entrypoints[i]
			checksum := bundle.CodeV2.Checksums[ep]
			_, ok := e.ScoreResults.Load(checksum)
			if !ok {
				found = true
				missing.Entrypoints = append(missing.Entrypoints, checksum)
			}
		}

		if found {
			queries = append(queries, &missing)
		}

		return true
	})

	return queries
}

// Schema is used for testing. Check carefully if you have other intentions
func (e *Executor) Schema() *resources.Schema {
	return e.schema
}
