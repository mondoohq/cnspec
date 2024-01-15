// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package internal

import (
	"fmt"
	"math"
	"sort"
	"time"

	vrs "github.com/hashicorp/go-version"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v10/cli/progress"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnspec/v10"
	"go.mondoo.com/cnspec/v10/policy"
)

type query struct {
	codeBundle         *llx.CodeBundle
	requiredProps      map[string]string
	resolvedProperties map[string]*llx.Primitive
}

type GraphBuilder struct {
	// queries is a map of QrID to query
	queries []query
	// datapointCollectors contains the collectors which will receive
	// datapoints
	datapointCollectors []DatapointCollector
	// scoreCollectors contains the collectors which will receive
	// scores
	scoreCollectors []ScoreCollector
	// reportingJobs is a list of reporting jobs, sourced from the
	// resolved policy
	reportingJobs []*policy.ReportingJob
	// collectDatapointChecksums specifies additional datapoints outside
	// the reporting job to collect
	collectDatapointChecksums []string
	// collectScoreQrIDs specifies additional scores outside
	// the reporting job to collect
	collectScoreQrIDs []string
	// datapointType is a map of checksum to type for datapoint type
	// conversion. This is sourced from the resolved policy
	datapointType map[string]string
	// progressReporter is a configured interface to receive progress
	// updates
	progressReporter progress.Progress
	// mondooVersion is the version of mondoo. This is generally sourced
	// from the binary, but is configurable to make testing easier
	mondooVersion string
	// queryTimeout is the amount of time to wait for the underlying lumi
	// runtime to send all the expected datapoints.
	queryTimeout time.Duration

	// featureFlagFailErrors is a feature flag to count errors as failures
	// See https://www.notion.so/mondoo/Errors-and-Scoring-5dc554348aad4118a1dbf35123368329
	featureFlagFailErrors bool
}

func NewBuilder() *GraphBuilder {
	return &GraphBuilder{
		queries:                   []query{},
		datapointCollectors:       []DatapointCollector{},
		scoreCollectors:           []ScoreCollector{},
		reportingJobs:             []*policy.ReportingJob{},
		collectDatapointChecksums: []string{},
		collectScoreQrIDs:         []string{},
		datapointType:             map[string]string{},
		progressReporter:          progress.Noop{},
		mondooVersion:             cnspec.GetCoreVersion(),
		queryTimeout:              5 * time.Minute,
	}
}

// AddQuery adds the provided code to be executed to the graph
func (b *GraphBuilder) AddQuery(c *llx.CodeBundle, propertyChecksums map[string]string, resolvedProperties map[string]*llx.Primitive) {
	b.queries = append(b.queries, query{
		codeBundle:         c,
		requiredProps:      propertyChecksums,
		resolvedProperties: resolvedProperties,
	})
}

func (b *GraphBuilder) AddDatapointType(datapointChecksum string, typ string) {
	b.datapointType[datapointChecksum] = typ
}

// CollectDatapoint requests the provided checksum be collected and sent to
// the configured datapoint collectors
func (b *GraphBuilder) CollectDatapoint(datapointChecksum string) {
	b.collectDatapointChecksums = append(b.collectDatapointChecksums, datapointChecksum)
}

// CollectScore requests the score of the provided query id be collected and
// sent to the configured score collectors. This will add a ReportingQueryNode
// node to the graph
func (b *GraphBuilder) CollectScore(queryID string) {
	b.collectScoreQrIDs = append(b.collectScoreQrIDs, queryID)
}

// AddReportingJob adds a reporting job to the graph. This adds a few edges:
//   - The reporting job sends its score to the reporting jobs listed in Notify
//   - If the reporting job represents a query, the ReportingQuery node is configured to send its score
//     to this reporting job
//   - scores and datapoints mentioned in this reporting job are automatically sent to
//     to collectors
func (b *GraphBuilder) AddReportingJob(rj *policy.ReportingJob) {
	b.reportingJobs = append(b.reportingJobs, rj)
}

// AddScoreCollector adds a score collector. Collected scores will be sent to
// all the provided score collectors
func (b *GraphBuilder) AddScoreCollector(c ScoreCollector) {
	b.scoreCollectors = append(b.scoreCollectors, c)
}

// AddDatapointCollector adds a datapoint collector. Collected datapoints
// will be sent to all the provided datapoint collectors
func (b *GraphBuilder) AddDatapointCollector(c DatapointCollector) {
	b.datapointCollectors = append(b.datapointCollectors, c)
}

// WithProgressReporter sets the interface which will receive progress updates
func (b *GraphBuilder) WithProgressReporter(r progress.Progress) {
	b.progressReporter = r
}

// WithMondooVersion sets the version of mondoo
func (b *GraphBuilder) WithMondooVersion(mondooVersion string) {
	b.mondooVersion = mondooVersion
}

// WithMondooVersion sets the version of mondoo
func (b *GraphBuilder) WithQueryTimeout(timeout time.Duration) {
	b.queryTimeout = timeout
}

// WithFeatureFlagFailErrors sets the feature flag to count errors as failures
func (b *GraphBuilder) WithFeatureFlagFailErrors() {
	b.featureFlagFailErrors = true
}

func (b *GraphBuilder) Build(runtime llx.Runtime, assetMrn string) (*GraphExecutor, error) {
	resultChan := make(chan *llx.RawResult, 128)

	queries := make(map[string]query, len(b.queries))
	for _, q := range b.queries {
		queries[q.codeBundle.GetCodeV2().GetId()] = q
	}

	ge := &GraphExecutor{
		nodes:        map[NodeID]*Node{},
		edges:        map[NodeID][]NodeID{},
		priorityMap:  map[NodeID]int{},
		queryTimeout: b.queryTimeout,
		executionManager: newExecutionManager(runtime, make(chan runQueueItem, len(queries)),
			resultChan, b.queryTimeout),
		resultChan: resultChan,
		doneChan:   make(chan struct{}),

		featureFlagFailErrors: b.featureFlagFailErrors,
	}

	ge.nodes[DatapointCollectorID] = &Node{
		id:       DatapointCollectorID,
		nodeType: DatapointCollectorNodeType,
		data: &DatapointCollectorNodeData{
			unreported: map[string]*llx.RawResult{},
			collectors: b.datapointCollectors,
		},
	}

	ge.nodes[ScoreCollectorID] = &Node{
		id:       ScoreCollectorID,
		nodeType: ScoreCollectorNodeType,
		data: &ScoreCollectorNodeData{
			unreported: map[string]*policy.Score{},
			collectors: b.scoreCollectors,
		},
	}

	unrunnableQueries := []query{}

	var mondooVersion *vrs.Version
	if b.mondooVersion != "" && b.mondooVersion != "unstable" {
		var err error
		mondooVersion, err = vrs.NewVersion(b.mondooVersion)
		if err != nil {
			log.Warn().Err(err).Str("version", b.mondooVersion).Msg("unable to parse mondoo version")
		}
	}

	for queryID, q := range queries {
		canRun := checkVersion(q.codeBundle, mondooVersion)
		if canRun {
			ge.addExecutionQueryNode(queryID, q, q.resolvedProperties, b.datapointType)
		} else {
			unrunnableQueries = append(unrunnableQueries, q)
		}
		ge.addReportingQueryNode(queryID, q)
	}

	scoresToCollect := make([]string, len(b.collectScoreQrIDs))
	copy(scoresToCollect, b.collectScoreQrIDs)
	datapointsToCollect := make([]string, len(b.collectDatapointChecksums))
	copy(datapointsToCollect, b.collectDatapointChecksums)

	for _, rj := range b.reportingJobs {
		_, isQuery := queries[rj.QrId]
		scoresToCollect = append(scoresToCollect, rj.Uuid)
		for datapointChecksum := range rj.Datapoints {
			datapointsToCollect = append(datapointsToCollect, datapointChecksum)
		}
		ge.addReportingJobNode(assetMrn, rj.Uuid, rj, isQuery)
	}

	for _, queryID := range scoresToCollect {
		ge.addEdge(NodeID(queryID), ScoreCollectorID)
	}

	for _, datapointChecksum := range datapointsToCollect {
		ge.addEdge(NodeID(datapointChecksum), DatapointCollectorID)
	}

	ge.handleUnrunnableQueries(unrunnableQueries)

	ge.createFinisherNode(b.progressReporter)

	for nodeID := range ge.nodes {
		prioritizeNode(ge.nodes, ge.edges, ge.priorityMap, nodeID)
	}

	// The finisher is the lowest priority node. This makes it so that
	// when a recalculation is triggered through a datapoint being reported,
	// the finisher only gets notified after all other intermediate nodes are
	// notified
	ge.priorityMap[CollectionFinisherID] = math.MinInt

	return ge, nil
}

// handleUnrunnableQueries takes the queries for which the running version does
// to meet the minimum version requirement and marks the datapoints as error.
// This is only done for datapoints which will not be reported by a runnable query
func (ge *GraphExecutor) handleUnrunnableQueries(unrunnableQueries []query) {
	for _, q := range unrunnableQueries {
		for _, checksum := range CodepointChecksums(q.codeBundle) {
			if _, ok := ge.nodes[NodeID(checksum)]; ok {
				// If the datapoint will be reported by another query, skip
				// handling it
				continue
			}

			ge.addDatapointNode(
				checksum,
				nil,
				&llx.RawResult{
					CodeID: checksum,
					Data: &llx.RawData{
						Error: fmt.Errorf("Unable to run query, cnspec version %s required", q.codeBundle.MinMondooVersion),
					},
				})
		}
	}
}

func (ge *GraphExecutor) addEdge(from NodeID, to NodeID) {
	ge.edges[from] = insertSorted(ge.edges[from], to)
}

func (ge *GraphExecutor) createFinisherNode(r progress.Progress) {
	nodeID := CollectionFinisherID
	nodeData := &CollectionFinisherNodeData{
		remainingDatapoints: make(map[string]struct{}, len(ge.nodes)),
		doneChan:            ge.doneChan,
		progressReporter:    r,
	}

	for datapointNodeID, n := range ge.nodes {
		if n.nodeType == DatapointNodeType {
			ge.addEdge(datapointNodeID, nodeID)
			nodeData.remainingDatapoints[datapointNodeID] = struct{}{}
		}
	}
	totalDatapoints := len(nodeData.remainingDatapoints)
	nodeData.totalDatapoints = totalDatapoints

	ge.nodes[nodeID] = &Node{
		id:       nodeID,
		nodeType: CollectionFinisherNodeType,
		data:     nodeData,
	}
}

func (ge *GraphExecutor) addExecutionQueryNode(queryID string, q query, resolvedProperties map[string]*llx.Primitive, datapointTypeMap map[string]string) {
	n, ok := ge.nodes[NodeID(queryID)]
	if ok {
		return
	}

	codeBundle := q.codeBundle

	nodeData := &ExecutionQueryNodeData{
		queryID:            queryID,
		codeBundle:         codeBundle,
		requiredProperties: map[string]*executionQueryProperty{},
		runState:           notReadyQueryNotReady,
		runQueue:           ge.executionManager.runQueue,
	}

	n = &Node{
		id:       NodeID(string(ExecutionQueryNodeType) + "/" + queryID),
		nodeType: ExecutionQueryNodeType,
		data:     nodeData,
	}

	// These don't report anything, but they make the graph connected
	for _, checksum := range CodepointChecksums(codeBundle) {
		var expectedType *string
		if t, ok := datapointTypeMap[checksum]; ok {
			expectedType = &t
		}
		ge.addDatapointNode(checksum, expectedType, nil)
		ge.addEdge(n.id, NodeID(checksum))
	}

	for name, checksum := range q.requiredProps {
		nodeData.requiredProperties[name] = &executionQueryProperty{
			name:     name,
			checksum: checksum,
			resolved: false,
			value:    nil,
		}
		ge.addEdge(NodeID(checksum), n.id)
	}

	for name, val := range resolvedProperties {
		if rp, ok := nodeData.requiredProperties[name]; !ok {
			nodeData.requiredProperties[name] = &executionQueryProperty{
				name:     name,
				checksum: "",
				resolved: true,
				value: &llx.Result{
					Data: val,
				},
			}
		} else {
			rp.value = &llx.Result{
				Data: val,
			}
			rp.resolved = true
		}
	}

	ge.nodes[n.id] = n
}

func (ge *GraphExecutor) addReportingQueryNode(queryID string, q query) {
	n, ok := ge.nodes[NodeID(queryID)]
	if ok {
		return
	}

	nodeData := &ReportingQueryNodeData{
		results: map[string]*DataResult{},
		queryID: queryID,
	}

	n = &Node{
		id:       NodeID(queryID),
		nodeType: ReportingQueryNodeType,
		data:     nodeData,
	}

	// We don't add edges from the code.Datapoints to the query
	// These get send to the reporting job and are not important
	// for the query completion
	for _, checksum := range EntrypointChecksums(q.codeBundle) {
		nodeData.results[checksum] = &DataResult{
			checksum: checksum,
			resolved: false,
			value:    nil,
		}
		ge.addEdge(NodeID(checksum), n.id)
	}

	ge.nodes[n.id] = n
}

func (ge *GraphExecutor) addReportingJobNode(assetMrn string, reportingJobID string, rj *policy.ReportingJob, isQuery bool) {
	n, ok := ge.nodes[NodeID(reportingJobID)]
	if ok {
		return
	}

	queryID := rj.QrId
	// TODO: This needs to be handled by the server so as not to
	// break existing clients. The function that was doing the
	// translation was RecalcScore. That function will no longer
	// be called
	if queryID == "root" {
		queryID = assetMrn
	}

	nodeData := &ReportingJobNodeData{
		queryID:               queryID,
		isQuery:               isQuery,
		rjType:                rj.Type,
		childScores:           map[string]*reportingJobResult{},
		datapoints:            map[string]*reportingJobDatapoint{},
		featureFlagFailErrors: ge.featureFlagFailErrors,
	}
	n = &Node{
		id:       NodeID(reportingJobID),
		nodeType: ReportingJobNodeType,
		data:     nodeData,
	}

	for checksum := range rj.Datapoints {
		nodeData.datapoints[checksum] = &reportingJobDatapoint{}
		ge.addEdge(NodeID(checksum), n.id)
	}

	for _, e := range rj.Notify {
		ge.addEdge(n.id, NodeID(e))
	}

	if isQuery {
		// The specs of the reporting job doesn't contain the query
		// Not all rj.QrIds are represented in the graph, only those
		// that correspond to actual queries. For example, a QrId that
		// is a policy is not represented as a node directly in the graph.
		// So, this is special handling to make sure the reporting job
		// knows that a reporting query is going to send it information
		// and that it needs to use that information to calculate its score
		nodeData.childScores[rj.QrId] = &reportingJobResult{}
		ge.addEdge(NodeID(rj.QrId), n.id)
	}

	for childReportingJobID, ss := range rj.ChildJobs {
		nodeData.childScores[childReportingJobID] = &reportingJobResult{
			impact: ss,
		}
	}

	nodeData.scoringSystem = rj.ScoringSystem

	ge.nodes[n.id] = n
}

func (ge *GraphExecutor) addDatapointNode(datapointChecksum string, expectedType *string, res *llx.RawResult) {
	n, ok := ge.nodes[NodeID(datapointChecksum)]
	if ok {
		return
	}

	nodeData := &DatapointNodeData{
		expectedType: expectedType,
		isReported:   res != nil,
		res:          res,
	}
	n = &Node{
		id:       NodeID(datapointChecksum),
		nodeType: DatapointNodeType,
		data:     nodeData,
	}

	ge.nodes[NodeID(datapointChecksum)] = n
}

// prioritizeNode assigns each node in the graph a priority. The priority makes graph traversal
// act like a breadth-first search, minimizing the number of recalculations needed for each node.
// For example, the reporting job with a query id of the asset will have a lower priority than
// reporting jobs which have a query id of a policy mrn. In a similar way, the reporting jobs
// that have a query id of policy mrns have a lower priority than reporting jobs for queries.
// This means that if a batch of data arrives, all query reporting jobs will be recalculated first.
// The policy reporting jobs will be calculated after that, and then the asset reporting job.
func prioritizeNode(nodes map[NodeID]*Node, edges map[NodeID][]NodeID, priorityMap map[NodeID]int, n NodeID) int {
	if d, ok := priorityMap[n]; ok {
		return d
	}
	childrenMaxDepth := 0
	for _, v := range edges[n] {
		childDepth := prioritizeNode(nodes, edges, priorityMap, v)
		if childDepth > childrenMaxDepth {
			childrenMaxDepth = childDepth
		}
	}
	myDepth := childrenMaxDepth + 1
	priorityMap[n] = myDepth
	return myDepth
}

func checkVersion(codeBundle *llx.CodeBundle, curMin *vrs.Version) bool {
	if curMin != nil && codeBundle.MinMondooVersion != "" {
		requiredVer := codeBundle.MinMondooVersion
		reqMin, err := vrs.NewVersion(requiredVer)
		if err == nil && curMin.LessThan(reqMin) {
			return false
		}
	}
	return true
}

func insertSorted(ss []string, s string) []string {
	i := sort.SearchStrings(ss, s)
	if i < len(ss) && ss[i] == s {
		return ss
	}
	ss = append(ss, "")
	copy(ss[i+1:], ss[i:])
	ss[i] = s
	return ss
}

func CodepointChecksums(codeBundle *llx.CodeBundle) []string {
	return append(EntrypointChecksums(codeBundle),
		DatapointChecksums(codeBundle)...)
}

func EntrypointChecksums(codeBundle *llx.CodeBundle) []string {
	var checksums []string
	checksums = make([]string, len(codeBundle.CodeV2.Blocks[0].Entrypoints))
	for i, ref := range codeBundle.CodeV2.Blocks[0].Entrypoints {
		checksums[i] = codeBundle.CodeV2.Checksums[ref]
	}
	return checksums
}

func DatapointChecksums(codeBundle *llx.CodeBundle) []string {
	var checksums []string
	checksums = make([]string, len(codeBundle.CodeV2.Blocks[0].Datapoints))
	for i, ref := range codeBundle.CodeV2.Blocks[0].Datapoints {
		checksums[i] = codeBundle.CodeV2.Checksums[ref]
	}
	return checksums
}
