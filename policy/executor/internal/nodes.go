// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package internal

import (
	"strings"

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
}

func (p *executionQueryProperty) Resolve(value *llx.Result) {
	p.value = value
	p.resolved = true
}

func (p *executionQueryProperty) IsResolved() bool {
	return p.resolved
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

	invalidated        bool
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

func (nodeData *ExecutionQueryNodeData) preseed(_ *envelope) {}

// consume saves any received data that matches any the required properties
func (nodeData *ExecutionQueryNodeData) consume(from NodeID, data *envelope) {
	if nodeData.runState == executedQueryRunState {
		// Nothing can change once the query has been marked as executed
		return
	}

	if len(nodeData.requiredProperties) == 0 {
		nodeData.invalidated = true
	}

	if data.res != nil {
		for _, p := range nodeData.requiredProperties {
			// Find the property with the matching checksum
			if p.checksum == data.res.CodeID {
				// Save the value of the property
				p.Resolve(data.res.Result())
				// invalidate the node for recalculation
				nodeData.invalidated = true
			}
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
	var props map[string]*llx.Result

	if len(nodeData.requiredProperties) > 0 {
		props = make(map[string]*llx.Result)
		for _, p := range nodeData.requiredProperties {
			props[p.name] = p.value
		}
	}

	nodeData.runState = executedQueryRunState

	nodeData.runQueue <- runQueueItem{
		codeBundle: nodeData.codeBundle,
		props:      props,
	}
}

// updateRunState sets the query to runnable if all the
// required properties needed have been received
func (d *ExecutionQueryNodeData) updateRunState() {
	if d.runState == readyQueryRunState {
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

func (nodeData *DatapointNodeData) preseed(_ *envelope) {}

// consume saves the result of the datapoint.
func (nodeData *DatapointNodeData) consume(from NodeID, data *envelope) {
	if data == nil || data.res == nil {
		// This can be triggered with no data by the execution query nodes. These
		// messages are not the ones we care about
		return
	}
	if nodeData.isReported {
		// Allow a real result to override a nil one. This handles the case
		// where short-circuit evaluation (e.g. &&) reports nil for an
		// unevaluated branch, and a later query reports the actual value.
		if isNilResult(nodeData.res) && hasRealData(data.res) {
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

func (nodeData *ReportingQueryNodeData) preseed(_ *envelope) {}

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
		// Allow a real result to override a nil one (short-circuit case)
		if isNilResult(dr.value) && hasRealData(data.res) {
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

	var err multierr.Errors
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

			err.Add(cur.Data.Error)
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
	rjType        policy.ReportingJob_Type

	childScores map[NodeID]*reportingJobResult
	datapoints  map[NodeID]*reportingJobDatapoint
	completed   bool
	invalidated bool

	featureFlagFailErrors bool
	scoreRisk             bool
}

func (nodeData *ReportingJobNodeData) preseed(data *envelope) {
	if data == nil || data.score == nil {
		return
	}
	if !policy.IsForwardScoreType(nodeData.rjType) {
		return
	}
	s := data.score.CloneVT()
	s.DataCompletion = 100
	s.ScoreCompletion = 100
	// Replace childScores with the preseeded score. This means if
	// consume() is called on this node (e.g. from a ReportingQueryNode
	// edge), it will panic because the original keys are gone. This is
	// intentional: preseed is only valid with a noop ExecutionManager
	// where no query results flow through the graph. If we ever need
	// to support consuming llx results alongside preseeded scores,
	// this will need to preserve original childScores keys.
	nodeData.childScores = map[NodeID]*reportingJobResult{
		"__preseed__": {score: s},
	}
	nodeData.invalidated = true
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
		// Score is stored as-is. CONTROL-type remapping (errors->0, skip->100)
		// happens at scoring time in CalculateReportingJobScore.
		rjRes.score = data.score
		nodeData.invalidated = true
	}

	if data.res != nil {
		dp, ok := nodeData.datapoints[from]
		if !ok {
			panic("invalid datapoint report")
		}
		// If the previously-reported result was nil (from short-circuit) and
		// we're now getting real data, reset completed so the score can be
		// recalculated with the actual data.
		if nodeData.completed && isNilResult(dp.res) && hasRealData(data.res) {
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

	children := make([]policy.ChildScore, 0, len(nodeData.childScores))
	for _, rjRes := range nodeData.childScores {
		children = append(children, policy.ChildScore{
			Score:  rjRes.score,
			Impact: rjRes.impact,
		})
	}

	totalDP := len(nodeData.datapoints)
	s, err := policy.CalculateReportingJobScore(
		nodeData.queryID,
		nodeData.rjType,
		nodeData.scoringSystem,
		children,
		totalDP,
		finishedDatapoints,
		nodeData.featureFlagFailErrors,
	)
	if err != nil || s == nil {
		return s, err
	}

	if !nodeData.scoreRisk {
		return s, nil
	}

	// Roll up RiskScore using the same scoring logic but with
	// Value=RiskScore on each child.
	riskChildren := make([]policy.ChildScore, 0, len(nodeData.childScores))
	for _, rjRes := range nodeData.childScores {
		if rjRes.score == nil {
			continue
		}

		// make a copy, as the CalculateReportingJobScore works with Value only,
		// we replace withRiskScore here.
		rs := rjRes.score.CloneVT()
		rs.Value = rs.RiskScore
		riskChildren = append(riskChildren, policy.ChildScore{
			Score:  rs,
			Impact: rjRes.impact,
		})
	}

	rs, err := policy.CalculateReportingJobScore(
		nodeData.queryID,
		nodeData.rjType,
		nodeData.scoringSystem,
		riskChildren,
		totalDP,
		finishedDatapoints,
		nodeData.featureFlagFailErrors,
	)
	if err != nil {
		return s, err
	}
	if rs != nil {
		s.RiskScore = rs.Value
	}

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

func (nodeData *CollectionFinisherNodeData) preseed(_ *envelope) {}

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

func (nodeData *DatapointCollectorNodeData) preseed(_ *envelope) {}

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

func (nodeData *ScoreCollectorNodeData) preseed(_ *envelope) {}

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
