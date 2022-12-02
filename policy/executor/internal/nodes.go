package internal

import (
	"errors"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/types"
	"go.mondoo.com/cnspec/cli/progress"
	"go.mondoo.com/cnspec/policy"
	"google.golang.org/protobuf/proto"
)

const (
	// ExecutionQueryNodeType represents a node that will execute
	// a query. It can be notified by datapoint nodes, representing
	// its dependant properties
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
	// be notified of scores of dependant reporting jobs
	ReportingJobNodeType NodeType = "reporting_job"
	// DatapointCollectorNodeType represents a sink for datapoints in the graph.
	// There is only one of these nodes in the graph, and it can only be notified
	// by datapoint nodes
	DatapointCollectorNodeType NodeType = "datapoint_collector"
	// ScoreCollectorNodeType represents a sink for scores in the graph. There is
	// only of of these nodes in the graph, and can be notified by reporting query
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

	// An empty evelope notifies the parent. These nodes always point at
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
	expectedType *string
	isReported   bool
	invalidated  bool
	res          *llx.RawResult
}

func (nodeData *DatapointNodeData) initialize() {
	if nodeData.res != nil {
		nodeData.set(nodeData.res)
	}
}

// consume saves the result of the datapoint.
func (nodeData *DatapointNodeData) consume(from NodeID, data *envelope) {
	if nodeData.isReported {
		// No change detection happens. If a datapoint is reported once, that is the value
		// we will use.
		return
	}
	if data == nil || data.res == nil {
		// This can be triggered with no data by the execution query nodes. These
		// messages are not the ones we care about
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

// recalculate passes on the datapoint's result if its available
func (nodeData *DatapointNodeData) recalculate() *envelope {
	if !nodeData.invalidated {
		return nil
	}

	nodeData.invalidated = false

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

	var errorsMsg string
	for _, dr := range nodeData.results {
		cur := dr.value
		if cur == nil {
			allFound = false
			break
		}

		if cur.Data.Error != nil {
			var resourceNotFoundErr *resources.ResourceNotFound
			if errors.As(cur.Data.Error, &resourceNotFoundErr) {
				assetVanishedDuringScan = true
			} else {
				allSkipped = false
				foundError = true
			}
			// append ; if we accumulate errors
			if errorsMsg != "" {
				errorsMsg += "; "
			}
			errorsMsg += cur.Data.Error.Error()
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
				Message:         errorsMsg,
			}
		} else if foundError {
			return &policy.Score{
				QrId:            nodeData.queryID,
				Type:            policy.ScoreType_Error,
				Value:           0,
				ScoreCompletion: 100,
				Weight:          1,
				Message:         errorsMsg,
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
	scoringSpec *policy.ScoringSpec
	score       *policy.Score
}

// ReportingJobNodeData is the data for nodes of type ReportingJobNodeType
type ReportingJobNodeData struct {
	queryID       string
	scoringSystem policy.ScoringSystem
	isQuery       bool

	childScores map[NodeID]*reportingJobResult
	datapoints  map[NodeID]*reportingJobDatapoint
	completed   bool
	invalidated bool
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
		rjRes.score = data.score
		nodeData.invalidated = true
	}

	if data.res != nil {
		dp, ok := nodeData.datapoints[from]
		if !ok {
			panic("invalid datapoint report")
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

	if nodeData.isQuery {
		// if this reporting job represents a reporting query, we want to add that
		// datapoints calculation, but return the score as is

		if nodeData.childScores[nodeData.queryID] == nil {
			panic("invalid reporting job")
		}
		var s *policy.Score
		if v := nodeData.childScores[nodeData.queryID].score; v == nil {
			s = &policy.Score{
				QrId: nodeData.queryID,
				Type: policy.ScoreType_Result,
			}
		} else {
			s = proto.Clone(v).(*policy.Score)
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

	calculator, err := policy.NewScoreCalculator(policy.ScoringSystem(nodeData.scoringSystem))
	if err != nil {
		return nil, err
	}

	for _, rjRes := range nodeData.childScores {
		s := rjRes.score
		if s == nil {
			return nil, nil
		}
		policy.AddSpecdScore(calculator, s, rjRes.score != nil, rjRes.scoringSpec)
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
