// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package internal

import (
	"strings"
	"sync"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnspec/v13/cli/progress"
	"go.mondoo.com/cnspec/v13/policy"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/types"
	"go.mondoo.com/mql/v13/utils/multierr"
)

// isNilResult returns true if the result has nil value and no error,
// which indicates it came from short-circuit evaluation.
func isNilResult(res *llx.RawResult) bool {
	return res != nil && res.Data != nil &&
		res.Data.Value == nil && res.Data.Error == nil
}

// hasRealData returns true if the result contains a non-nil value or an error.
func hasRealData(res *llx.RawResult) bool {
	return res != nil && res.Data != nil &&
		(res.Data.Value != nil || res.Data.Error != nil)
}

// queryRunError marks a result that was broadcast for a query that could not
// be executed at all (e.g. one of its properties errored). Such results are
// placeholders: they exist so that datapoint consumers do not wait forever
// for results that will never arrive. They are NOT produced by evaluating the
// statement they are reported for. Datapoint checksums are content-addressed
// and shared across queries, so a placeholder may land on a checksum that a
// different, healthy query will still report a real result for. Consumers
// therefore let any executed result (value, nil, or error) override a
// placeholder, and never let a placeholder override anything.
type queryRunError struct {
	// originCodeID is the CodeV2.Id of the query that failed to run
	originCodeID string
	err          error
}

func (e *queryRunError) Error() string { return e.err.Error() }
func (e *queryRunError) Unwrap() error { return e.err }

// isPlaceholderResult returns true if the result is a broadcast placeholder
// for a query that never ran (see queryRunError).
func isPlaceholderResult(res *llx.RawResult) bool {
	if res == nil || res.Data == nil || res.Data.Error == nil {
		return false
	}
	var qre *queryRunError
	return errors.As(res.Data.Error, &qre)
}

// placeholderOrigin returns the CodeID of the query whose failure produced
// this placeholder result, or "" if the result is not a placeholder.
func placeholderOrigin(res *llx.RawResult) string {
	if res == nil || res.Data == nil || res.Data.Error == nil {
		return ""
	}
	var qre *queryRunError
	if errors.As(res.Data.Error, &qre) {
		return qre.originCodeID
	}
	return ""
}

// resultUpgrades returns true if next should replace prev:
//   - an executed result (real data, error, or evaluated nil) replaces a
//     placeholder
//   - real data replaces an evaluated nil (short-circuit case)
//
// A placeholder never replaces anything.
func resultUpgrades(prev, next *llx.RawResult) bool {
	if isPlaceholderResult(next) {
		return false
	}
	if isPlaceholderResult(prev) {
		return true
	}
	return isNilResult(prev) && hasRealData(next)
}

const (
	// ExecutionQueryNodeType represents a node that will execute
	// a query. It can be notified by datapoint nodes, representing
	// its dependent properties
	ExecutionQueryNodeType NodeType = "execution_query"
	// DatapointNodeType represents a node that is a datapoint/entrypoint.
	// These nodes are implicitly notified when results are received from
	// the executor threads. They also have edges from execution query nodes,
	// however these just connect the execution and reporting nodes in the graph.
	// When triggered by an execution query, the result will be a noop. These nodes
	// typically notify execution query nodes with properties, reporting query
	// nodes to calculate a query score, and reporting job nodes the calculate
	// data collection completion.
	DatapointNodeType NodeType = "datapoint"
	// ReportingQueryNodeType represents a query to be scored. Each execution
	// query has one of these. It is notified by datapoint nodes. These datapoints
	// are entrypoints which cause the node to calculate a score for the query.
	// Nodes of this type report to reporting job nodes, passing along the score
	// calculated to the reporting job.
	ReportingQueryNodeType NodeType = "reporting_query"
	// ReportingJobNodeType represent scores that needed to be collected. This
	// information is sourced from the resolved policy. Nodes of this type are
	// notified by datapoints to indicate collection of data, reporting query
	// nodes to be notified of query scores, and other reporting job nodes to
	// be notified of scores of dependent reporting jobs
	ReportingJobNodeType NodeType = "reporting_job"
	// DatapointCollectorNodeType represents a sink for datapoints in the graph.
	// There is only one of these nodes in the graph, and it can only be notified
	// by datapoint nodes
	DatapointCollectorNodeType NodeType = "datapoint_collector"
	// ScoreCollectorNodeType represents a sink for scores in the graph. There is
	// only one of these nodes in the graph, and can be notified by reporting query
	// nodes and reporting job nodes (nodes that pass a score)
	ScoreCollectorNodeType NodeType = "score_collector"
	// CollectionFinisherNodeType represents a node that collects datapoints. It is
	// used to notify of completion when all the expected datapoints have been received.
	// It is different from the datapoint collector node in that it always has the lowest
	// priority, so all other work is guaranteed to complete before it says things are done
	CollectionFinisherNodeType NodeType = "collection_finisher"

	DatapointCollectorID NodeID = "__datapoint_collector__"
	ScoreCollectorID     NodeID = "__score_collector__"
	CollectionFinisherID NodeID = "__collection_finisher__"
)

type executionQueryProperty struct {
	name     string
	checksum string
	value    *llx.Result
	resolved bool
	// resolvedNil is true when the property resolved to a nil value with no
	// error. Short-circuit evaluation (e.g. &&, ||) reports nil for
	// unevaluated branches; a query sharing the property's checksum can
	// therefore deliver a transient nil BEFORE the real value arrives. Such a
	// nil may still be upgraded to real data later (see isNilResult /
	// hasRealData on the datapoint consumers).
	resolvedNil bool
	// resolvedPlaceholder is true when the property resolved from a broadcast
	// placeholder (queryRunError) of a query that never ran. An executed
	// result may upgrade it.
	resolvedPlaceholder bool
}

func (p *executionQueryProperty) IsResolved() bool {
	return p.resolved
}

// isNilResultProto returns true if the proto result carries a nil value and
// no error. This is the proto counterpart of isNilResult.
//
// It relies on RawData.Result() normalizing nil values to types.Nil in the
// proto representation: raw2primitive returns NilPrimitive for any nil value
// regardless of the declared type (llx/data_conversions.go), so a typed nil
// (e.g. RawData{Type: types.String, Value: nil}) arrives here with
// Data.Type == types.Nil. Do NOT additionally test Data.Value == nil:
// falsy real values encode as empty/zero byte slices (e.g. "" -> []byte{}),
// which proto round-trips can deliver as nil slices — such a check would
// misclassify real values as nil.
func isNilResultProto(res *llx.Result) bool {
	if res == nil || res.Error != "" {
		return false
	}
	data := res.Data
	if data == nil {
		return true
	}
	return types.Type(data.Type) == types.Nil
}

type DataResult struct {
	checksum string
	resolved bool
	value    *llx.RawResult
}

type queryRunState int

const (
	notReadyQueryNotReady queryRunState = iota
	readyQueryRunState
	executedQueryRunState
)

// ExecutionQueryNodeData represents a node of type ExecutionQueryNodeType
type ExecutionQueryNodeData struct {
	queryID    string
	codeBundle *llx.CodeBundle

	invalidated bool
	// propsLock guards requiredProperties value/resolved state. Values are
	// written by the graph loop and read by the execution manager goroutine
	// when a queued query materializes its properties.
	propsLock          sync.Mutex
	requiredProperties map[string]*executionQueryProperty
	runState           queryRunState
	runQueue           chan<- runQueueItem
}

func (nodeData *ExecutionQueryNodeData) initialize() {
	nodeData.updateRunState()
	if nodeData.runState == readyQueryRunState {
		nodeData.invalidated = true
	}
}

// consume saves any received data that matches any the required properties
func (nodeData *ExecutionQueryNodeData) consume(from NodeID, data *envelope) {
	if len(nodeData.requiredProperties) == 0 {
		if nodeData.runState != executedQueryRunState {
			nodeData.invalidated = true
		}
		return
	}

	if data.res == nil {
		return
	}

	nodeData.propsLock.Lock()
	defer nodeData.propsLock.Unlock()

	for _, p := range nodeData.requiredProperties {
		// Find the property with the matching checksum
		if p.checksum != data.res.CodeID {
			continue
		}

		if !p.resolved {
			// First result for this property. A nil (no value, no error) may
			// come from short-circuit evaluation and may be followed by the
			// real value. Accept it so execution is not blocked (a property
			// can legitimately be null), but remember that it may be upgraded.
			p.value = data.res.Result()
			p.resolved = true
			p.resolvedNil = isNilResultProto(p.value)
			p.resolvedPlaceholder = isPlaceholderResult(data.res)
			if nodeData.runState != executedQueryRunState {
				nodeData.invalidated = true
			}
			continue
		}

		// Already resolved: allow an executed result to replace a broadcast
		// placeholder, and a real result to replace a transient nil. This
		// mirrors the upgrade semantics of the datapoint consumers.
		if p.resolvedPlaceholder && !isPlaceholderResult(data.res) {
			p.value = data.res.Result()
			p.resolvedPlaceholder = false
			p.resolvedNil = isNilResultProto(p.value)
			continue
		}
		if p.resolvedNil && hasRealData(data.res) && !isPlaceholderResult(data.res) {
			p.value = data.res.Result()
			p.resolvedNil = false
		}
	}
}

// recalculate checks if all required properties are satisfied. Once
// all have been received, the query is queued for execution
func (nodeData *ExecutionQueryNodeData) recalculate() *envelope {
	if !nodeData.invalidated {
		// Nothing can change once the query has been marked as executed
		return nil
	}

	// Update the run state so we know if the state changed to
	// runnable
	nodeData.updateRunState()
	nodeData.invalidated = false

	if nodeData.runState == readyQueryRunState {
		nodeData.run()
	}

	// An empty envelope notifies the parent. These nodes always point at
	// Datapoint nodes. The datapoint nodes don't need this message, and
	// it actually makes more work for the datapoint node. The reason to
	// send it is to uphold the contract of if something changes, we push
	// a message through the graph. And in this case, something did
	// technically change
	return &envelope{}
}

// run sends this query to be run to the executor queue
// this should only be called when the query is runnable (
// all properties needed are available)
func (nodeData *ExecutionQueryNodeData) run() {
	nodeData.runState = executedQueryRunState

	nodeData.runQueue <- runQueueItem{
		codeBundle: nodeData.codeBundle,
		props:      nodeData.materializeProps,
	}
}

// materializeProps snapshots the current property values. It is called by the
// execution manager when the query is dequeued, NOT when it is queued: if a
// transient nil property was upgraded to its real value while the query was
// waiting in the run queue, the query executes with the real value.
func (nodeData *ExecutionQueryNodeData) materializeProps() map[string]*llx.Result {
	nodeData.propsLock.Lock()
	defer nodeData.propsLock.Unlock()

	if len(nodeData.requiredProperties) == 0 {
		return nil
	}
	props := make(map[string]*llx.Result, len(nodeData.requiredProperties))
	for _, p := range nodeData.requiredProperties {
		props[p.name] = p.value
	}
	return props
}

// updateRunState sets the query to runnable if all the
// required properties needed have been received
func (d *ExecutionQueryNodeData) updateRunState() {
	if d.runState == readyQueryRunState || d.runState == executedQueryRunState {
		return
	}

	runnable := true

	for _, p := range d.requiredProperties {
		runnable = runnable && p.IsResolved()
	}

	if runnable {
		d.runState = readyQueryRunState
	} else {
		d.runState = notReadyQueryNotReady
	}
}

// DatapointNodeData is the data for queries of type DatapointNodeType.
type DatapointNodeData struct {
	expectedType   *string
	isReported     bool
	invalidated    bool
	dumpDatapoints bool
	res            *llx.RawResult
}

func (nodeData *DatapointNodeData) initialize() {
	if nodeData.res != nil {
		nodeData.set(nodeData.res)
	}
}

// consume saves the result of the datapoint.
func (nodeData *DatapointNodeData) consume(from NodeID, data *envelope) {
	if data == nil || data.res == nil {
		// This can be triggered with no data by the execution query nodes. These
		// messages are not the ones we care about
		return
	}
	if nodeData.isReported {
		// Allow upgrades only:
		// - an executed result overrides a broadcast placeholder from a query
		//   that never ran (see queryRunError)
		// - a real result overrides a nil one from short-circuit evaluation
		if resultUpgrades(nodeData.res, data.res) {
			nodeData.set(data.res)
		}
		return
	}

	nodeData.set(data.res)
}

func (nodeData *DatapointNodeData) set(res *llx.RawResult) {
	nodeData.invalidated = true
	nodeData.isReported = true

	if nodeData.expectedType == nil || types.Type(*nodeData.expectedType) == types.Unset ||
		res.Data.Type == types.Nil || res.Data.Type == types.Type(*nodeData.expectedType) ||
		res.Data.Error != nil {
		nodeData.res = res
	} else {
		nodeData.res = res.CastResult(types.Type(*nodeData.expectedType)).RawResultV2()
	}
}

// recalculate passes on the datapoint's result if it's available
func (nodeData *DatapointNodeData) recalculate() *envelope {
	if !nodeData.invalidated {
		return nil
	}

	nodeData.invalidated = false

	if nodeData.dumpDatapoints {
		log.Trace().Str("codeId", nodeData.res.CodeID).Interface("data", nodeData.res.Data).Msg("datapoint collected")
	}
	return &envelope{
		res: nodeData.res,
	}
}

// ReportingQueryNodeData is the data for queries of type ReportingQueryNodeType.
type ReportingQueryNodeData struct {
	queryID string

	results     map[string]*DataResult
	invalidated bool
}

func (nodeData *ReportingQueryNodeData) initialize() {
	invalidated := len(nodeData.results) == 0
	for _, dr := range nodeData.results {
		invalidated = invalidated || dr.resolved
	}
	nodeData.invalidated = invalidated
}

// consume stores datapoint results sent to it. These represent entrypoints which
// are needed to calculate the score
func (nodeData *ReportingQueryNodeData) consume(from NodeID, data *envelope) {
	dr, ok := nodeData.results[from]
	if !ok {
		return
	}
	if dr.resolved {
		// Allow upgrades only: executed results override broadcast
		// placeholders, real results override short-circuit nils.
		if resultUpgrades(dr.value, data.res) {
			dr.value = data.res
			nodeData.invalidated = true
		}
		return
	}

	dr.value = data.res
	dr.resolved = true
	nodeData.invalidated = true
}

// recalculate recalculates the score based on the saved data
func (nodeData *ReportingQueryNodeData) recalculate() *envelope {
	if !nodeData.invalidated {
		return nil
	}

	nodeData.invalidated = false

	s := nodeData.score()
	if s == nil {
		return nil
	}

	return &envelope{
		score: s,
	}
}

func (nodeData *ReportingQueryNodeData) score() *policy.Score {
	allFound := true
	allSkipped := true
	allTrue := true
	foundError := false
	assetVanishedDuringScan := false
	var scoreFound *llx.RawData
	var scoreValue int

	// Errors are aggregated in two buckets. Placeholder errors broadcast by
	// OTHER queries that could not run (see queryRunError) land on shared
	// datapoint checksums; their messages describe a different query's
	// failure and must not leak into this query's message if this query has
	// errors of its own.
	var err multierr.Errors
	var foreignErr multierr.Errors
	for _, dr := range nodeData.results {
		cur := dr.value
		if cur == nil {
			allFound = false
			break
		}

		if cur.Data.Error != nil {
			msg := cur.Data.Error.Error()
			if strings.HasPrefix(msg, "could not find resource") {
				assetVanishedDuringScan = true
			} else {
				allSkipped = false
				foundError = true
			}

			if origin := placeholderOrigin(cur); origin != "" && origin != nodeData.queryID {
				foreignErr.Add(cur.Data.Error)
			} else {
				err.Add(cur.Data.Error)
			}
			continue
		}

		if cur.Data.Value == nil {
			continue
		}

		if v, ok := cur.Data.Score(); ok {
			scoreFound = cur.Data
			scoreValue = v
			allSkipped = false
			continue
		}

		success, valid := cur.Data.IsSuccess()
		if !success && valid {
			allTrue = false
		}
		if valid {
			allSkipped = false
		}
	}

	// Prefer this query's own errors for the message; fall back to foreign
	// placeholder errors only when they are all we have (e.g. this query's
	// shared statement was poisoned and nothing else errored).
	if len(err.Errors) == 0 {
		err = foreignErr
	}

	if allFound {
		if assetVanishedDuringScan {
			return &policy.Score{
				QrId:            nodeData.queryID,
				Type:            policy.ScoreType_Unscored,
				Value:           0,
				ScoreCompletion: 100,
				Weight:          1,
				Message:         err.Deduplicate().Error(),
			}
		} else if foundError {
			return &policy.Score{
				QrId:            nodeData.queryID,
				Type:            policy.ScoreType_Error,
				Value:           0,
				ScoreCompletion: 100,
				Weight:          1,
				Message:         err.Deduplicate().Error(),
			}
		} else if allSkipped {
			return &policy.Score{
				QrId:            nodeData.queryID,
				Type:            policy.ScoreType_Skip,
				Value:           0,
				ScoreCompletion: 100,
				Weight:          1,
				Message:         "",
			}
		} else {
			if scoreFound == nil {
				if allTrue {
					scoreValue = 100
				} else {
					scoreValue = 0
				}
			}
			return &policy.Score{
				QrId:            nodeData.queryID,
				Type:            policy.ScoreType_Result,
				Value:           uint32(scoreValue),
				ScoreCompletion: 100,
				Weight:          1,
				Message:         "",
			}
		}
	}
	return nil
}

// StaticReportingJobNodeData holds a pre-computed score and emits it once on
// its first recalculate. Used in rescore mode to replace leaf reporting-job
// nodes whose score is supplied up front. It has no inbound edges; consume
// is a no-op.
type StaticReportingJobNodeData struct {
	score *policy.Score
	sent  bool
}

func (nodeData *StaticReportingJobNodeData) initialize() {}

func (nodeData *StaticReportingJobNodeData) consume(from NodeID, data *envelope) {
	// Static nodes have no inbound edges by construction. A consume call
	// here means the builder wired one in by mistake; surface it instead
	// of silently dropping the data.
	log.Debug().Str("from", string(from)).Msg("static reporting job received unexpected consume; ignoring")
}

func (nodeData *StaticReportingJobNodeData) recalculate() *envelope {
	if nodeData.sent {
		return nil
	}
	nodeData.sent = true
	return &envelope{
		score: nodeData.score,
	}
}

type reportingJobDatapoint struct {
	res *llx.RawResult
}

type reportingJobResult struct {
	impact *policy.Impact
	score  *policy.Score
}

// ReportingJobNodeData is the data for nodes of type ReportingJobNodeType
type ReportingJobNodeData struct {
	queryID       string
	scoringSystem policy.ScoringSystem
	forwardScore  bool
	rjType        policy.ReportingJob_Type

	childScores map[NodeID]*reportingJobResult
	datapoints  map[NodeID]*reportingJobDatapoint
	completed   bool
	invalidated bool

	featureFlagFailErrors bool
}

func (nodeData *ReportingJobNodeData) initialize() {
	nodeData.invalidated = true
}

// consume saves scores from dependent reporting queries and reporting jobs, and
// results from dependent datapoints
func (nodeData *ReportingJobNodeData) consume(from NodeID, data *envelope) {
	if data.score != nil {
		rjRes, ok := nodeData.childScores[from]
		if !ok {
			panic("invalid score report")
		}
		score := data.score
		switch nodeData.rjType {
		case policy.ReportingJob_CONTROL:
			if score.Type != policy.ScoreType_Result {
				// We map errors to failed results.
				// Skip and unknown are mapped to passing results
				score = score.CloneVT()
				switch score.Type {
				case policy.ScoreType_Error:
					score.Type = policy.ScoreType_Result
					score.Value = 0
				case policy.ScoreType_Skip, policy.ScoreType_Unscored:
					score.Type = policy.ScoreType_Result
					score.Value = 100
				}
			}
		}
		rjRes.score = score
		nodeData.invalidated = true
	}

	if data.res != nil {
		dp, ok := nodeData.datapoints[from]
		if !ok {
			panic("invalid datapoint report")
		}
		// A broadcast placeholder (queryRunError) never replaces an executed
		// result, and any non-upgrade write (duplicate, downgrade) is
		// dropped — mirroring DatapointNodeData/ReportingQueryNodeData.
		// Upgrades (executed result over placeholder, real data over
		// short-circuit nil) reopen a completed job so the score is
		// recalculated with the better data.
		if dp.res != nil && isPlaceholderResult(data.res) && !isPlaceholderResult(dp.res) {
			return
		}
		if dp.res != nil && !resultUpgrades(dp.res, data.res) {
			return
		}
		if nodeData.completed {
			nodeData.completed = false
		}
		dp.res = data.res
		nodeData.invalidated = true
	}
}

// recalculate recalculates the score based on the dependent datapoint results
// and scores
func (nodeData *ReportingJobNodeData) recalculate() *envelope {
	if nodeData.queryID == "" {
		panic("invalid query id")
	}
	if !nodeData.invalidated || nodeData.completed {
		return nil
	}

	nodeData.invalidated = false

	s, err := nodeData.score()
	if err != nil {
		nodeData.completed = true
		return &envelope{
			score: &policy.Score{
				QrId:            nodeData.queryID,
				Type:            policy.ScoreType_Error,
				ScoreCompletion: 100,
				Weight:          1,
				Message:         err.Error(),
			},
		}
	}
	if s == nil {
		// nil is returned for the score if the child reporting jobs (non-query
		// have not reported at least once. We need to know how many datapoints
		// there will be from the children before this reporting job
		// score can be calculated. Without knowing that information,
		// we risk flapping the data completion value
		return nil
	}
	if s.Completion() == 100 {
		nodeData.completed = true
	} else {
		return nil
	}
	return &envelope{
		score: s,
	}
}

func (nodeData *ReportingJobNodeData) score() (*policy.Score, error) {
	finishedDatapoints := 0
	for _, datapointRes := range nodeData.datapoints {
		if datapointRes.res != nil {
			finishedDatapoints++
		}
	}

	if nodeData.forwardScore {
		var s *policy.Score

		if len(nodeData.childScores) == 0 {
			s = &policy.Score{
				QrId:            nodeData.queryID,
				Type:            policy.ScoreType_Unscored,
				ScoreCompletion: 100,
			}
		} else {
			if len(nodeData.childScores) != 1 {
				panic("invalid reporting job")
			}
			var child string
			for k := range nodeData.childScores {
				child = k
			}
			if c := nodeData.childScores[child]; c.score == nil {
				s = &policy.Score{
					QrId: nodeData.queryID,
					Type: policy.ScoreType_Result,
				}
			} else {
				s = c.score.CloneVT()
				s.QrId = nodeData.queryID

				if c.impact.GetScoring() == policy.ScoringSystem_DISABLED {
					s.Type = policy.ScoreType_Disabled
				} else if s.Type == policy.ScoreType_Result {
					// We can't just forward the score if impact is set and we have a result.
					// We still need to apply impact to the score
					if c.impact != nil {
						if c.impact.Value != nil {
							floor := 100 - uint32(c.impact.Value.Value)
							if floor > s.Value {
								s.Value = floor
							}
						}
					}
				}
			}
		}

		// TODO: It's unclear if we should do this if the score is skipped or errored
		// If the executor doesn't return something for every datapoint, then this will
		// be broken in other ways. For example, the completion relies on getting every
		// datapoint reported
		if totalDatapoints := len(nodeData.datapoints); totalDatapoints > 0 {
			s.DataTotal = uint32(totalDatapoints)
			s.DataCompletion = uint32((100 * finishedDatapoints) / totalDatapoints)
		}

		return s, nil
	}

	scoreCalculatorOptions := []policy.ScoreCalculatorOption{}
	if nodeData.featureFlagFailErrors {
		scoreCalculatorOptions = append(scoreCalculatorOptions, policy.WithScoreCalculatorFeatureFlagFailErrors())
	}
	calculator, err := policy.NewScoreCalculator(nodeData.scoringSystem, scoreCalculatorOptions...)
	if err != nil {
		return nil, err
	}

	for _, rjRes := range nodeData.childScores {
		s := rjRes.score
		if s == nil {
			return nil, nil
		}
		policy.AddSpecdScore(calculator, s, rjRes.score != nil, rjRes.impact)
	}

	policy.AddDataScore(calculator, len(nodeData.datapoints), finishedDatapoints)

	s := calculator.Calculate()
	s.QrId = nodeData.queryID
	return s, nil
}

// CollectionFinisherNodeData represents the node of type CollectionFinisherNodeType
// It keeps track of the datapoints that have yet to report back
type CollectionFinisherNodeData struct {
	progressReporter progress.Progress
	totalDatapoints  int

	remainingDatapoints map[NodeID]struct{}
	doneChan            chan struct{}
	invalidated         bool
	assetPlatformId     string
}

func (nodeData *CollectionFinisherNodeData) initialize() {
	if len(nodeData.remainingDatapoints) == 0 {
		nodeData.invalidated = true
	}
}

// consume marks the received datapoints as finished
func (nodeData *CollectionFinisherNodeData) consume(from NodeID, data *envelope) {
	if len(nodeData.remainingDatapoints) == 0 {
		return
	}
	log.Debug().Msgf("%s finished", from)
	delete(nodeData.remainingDatapoints, from)
	nodeData.invalidated = true
}

// recalculate closes the completion channel if all the data has been received
func (nodeData *CollectionFinisherNodeData) recalculate() *envelope {
	if !nodeData.invalidated {
		return nil
	}
	nodeData.progressReporter.OnProgress(nodeData.totalDatapoints-len(nodeData.remainingDatapoints), nodeData.totalDatapoints)
	nodeData.invalidated = false
	if len(nodeData.remainingDatapoints) == 0 {
		log.Debug().Msg("graph has received all datapoints")
		close(nodeData.doneChan)
	}
	return nil
}

// DatapointCollectorNodeData is the data for nodes of type DatapointCollectorNodeType
type DatapointCollectorNodeData struct {
	collectors  []DatapointCollector
	unreported  map[string]*llx.RawResult
	invalidated bool
}

func (nodeData *DatapointCollectorNodeData) initialize() {
	if len(nodeData.unreported) > 0 {
		nodeData.invalidated = true
	}
}

// consume collects datapoints
func (nodeData *DatapointCollectorNodeData) consume(from NodeID, data *envelope) {
	if data.res != nil {
		nodeData.unreported[data.res.CodeID] = data.res
		nodeData.invalidated = true
	}
}

// recalculate passes the newly collected datapoints to the configured collectors
func (nodeData *DatapointCollectorNodeData) recalculate() *envelope {
	if !nodeData.invalidated {
		return nil
	}
	nodeData.invalidated = false
	arr := make([]*llx.RawResult, len(nodeData.unreported))
	i := 0
	for _, rr := range nodeData.unreported {
		arr[i] = rr
		i++
	}
	for _, dc := range nodeData.collectors {
		dc.SinkData(arr)
	}
	for k := range nodeData.unreported {
		delete(nodeData.unreported, k)
	}
	return nil
}

// ScoreCollectorNodeData represents nodes of type ScoreCollectorNodeType
type ScoreCollectorNodeData struct {
	collectors  []ScoreCollector
	unreported  map[string]*policy.Score
	invalidated bool
}

func (nodeData *ScoreCollectorNodeData) initialize() {
	if len(nodeData.unreported) > 0 {
		nodeData.invalidated = true
	}
}

// consume collects scores
func (nodeData *ScoreCollectorNodeData) consume(from NodeID, data *envelope) {
	if data.score != nil {
		if data.score.QrId == "" {
			panic("no qrid")
		}
		nodeData.unreported[from] = data.score
		nodeData.invalidated = true
	}
}

// recalculate passes newly collected scores to the configured collectors
func (nodeData *ScoreCollectorNodeData) recalculate() *envelope {
	if !nodeData.invalidated {
		return nil
	}
	nodeData.invalidated = false
	arr := make([]*policy.Score, len(nodeData.unreported))
	i := 0
	for _, s := range nodeData.unreported {
		arr[i] = s
		i++
	}
	for _, sc := range nodeData.collectors {
		sc.SinkScore(arr)
	}
	for k := range nodeData.unreported {
		delete(nodeData.unreported, k)
	}
	return nil
}
