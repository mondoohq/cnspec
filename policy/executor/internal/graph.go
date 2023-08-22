// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package internal

import (
	"container/heap"
	"fmt"
	"os"
	"time"

	vrs "github.com/hashicorp/go-version"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnspec/policy"
)

type (
	NodeType string
	NodeID   = string
)

type Node struct {
	id       NodeID
	nodeType NodeType
	data     nodeData
}

// envelope represents data that can be passed
// between nodes
type envelope struct {
	res   *llx.RawResult
	score *policy.Score
}

// nodeData must be implemented by each node type and will
// be attached to the Node struct. Pushing data through the
// graph involves 2 calls for each node. First, the graph
// will ask the node to consume any new data is has from
// the nodes dependants (in edges). Once all the data has
// been sent, it will ask the node to recalculate and return
// any data that it should send from this node to its out
// edges.
type nodeData interface {
	initialize()
	// consume sends data to this node from a dependant node.
	// consume should be defer as much work to recalculate as
	// possible, as recalculate will only be called after all
	// available dependant data has been sent
	consume(from NodeID, data *envelope)
	// recalculate is used to recalculate data for this node.
	// If nothing has changed and the out edges do not need
	// to be notified, this function should return nil
	recalculate() *envelope
}

type GraphExecutor struct {
	nodes         map[NodeID]*Node
	edges         map[NodeID][]NodeID
	priorityMap   map[NodeID]int
	queryTimeout  time.Duration
	mondooVersion *vrs.Version

	executionManager *executionManager
	resultChan       chan *llx.RawResult
	doneChan         chan struct{}

	featureFlagFailErrors bool
}

// Execute executes the graph
//
// The algorithm:
// Tell the nodes to initialize themselves. This invalidates
// them if needed before any messages are sent. For example,
// execution nodes become invalidated if all their property
// dependencies are specified or they have no property
// dependencies.
//
// The execution happens in rounds of asking nodes to consume,
// and then recalculate, starting with datapoint nodes. A round
// starts when a batch of datapoints has been received
//
// The execution of queries reports datapoints. The nodes that represent
// these datapoints are looked up. We first ask these nodes to consume
// the results that were received, and put each on in a priority queue.
//
// For each node in the priority, we ask it to recalculate itself. If
// recalculate returns non-nil, we call consume on each out edge and
// put those nodes in the priority queue.
//
// The round ends when the priority queue is empty. At the end of the round,
// the reporting graph will be fully up-to-date. Because the graph is acyclic
// and we assign a priority to each node, each node in the graph should only
// recalculate at most once in each round
func (ge *GraphExecutor) Execute() error {
	ge.executionManager.Start()

	// Trigger the execution nodes
	maxPriority := len(ge.nodes) + 1
	q := make(PriorityQueue, 0, len(ge.nodes))
	heap.Init(&q)
	for nodeID, n := range ge.nodes {
		n.data.initialize()
		heap.Push(&q, &Item{
			priority: maxPriority,
			receiver: nodeID,
			sender:   "__initialize__",
		})
	}

	done := false
	var err error
OUTER:
	for {
		// process queue
		for q.Len() > 0 {
			item := heap.Pop(&q).(*Item)

			n := ge.nodes[item.receiver]
			dataToSend := n.data.recalculate()
			log.Trace().
				Str("from", item.sender).
				Str("to", item.receiver).
				Msg("recalculate result")

			if dataToSend != nil {
				edges := ge.edges[item.receiver]
				for _, v := range edges {
					log.Trace().
						Str("from", item.receiver).
						Str("to", v).
						Bool("hasResult", dataToSend.res != nil).
						Interface("score", dataToSend.score).
						Msg("consume result")
					childNode := ge.nodes[v]
					childNode.data.consume(n.id, dataToSend)
					heap.Push(&q, &Item{
						priority: ge.priorityMap[v],
						receiver: v,
						sender:   item.receiver,
					})
				}
			}
		}

		if done {
			break OUTER
		}

		// Wait for message
		select {
		case res := <-ge.resultChan:
			nodeID := res.CodeID
			n := ge.nodes[nodeID]
			n.data.consume("", &envelope{res: res})
			heap.Push(&q, &Item{
				priority: maxPriority,
				receiver: nodeID,
				sender:   "",
			})
		case <-ge.doneChan:
			done = true
		case err = <-ge.executionManager.Err():
			break OUTER
		}
		// drain all available messages
	DRAIN:
		for {
			select {
			case res := <-ge.resultChan:
				nodeID := res.CodeID
				n := ge.nodes[nodeID]
				n.data.consume("", &envelope{res: res})
				heap.Push(&q, &Item{
					priority: maxPriority,
					receiver: nodeID,
					sender:   "",
				})
			default:
				break DRAIN
			}
		}
	}

	ge.executionManager.Stop()
	return err
}

func (ge *GraphExecutor) Debug() {
	if val, ok := os.LookupEnv("DEBUG"); ok && (val == "1" || val == "true") {
	} else {
		return
	}
	f, err := os.Create("mondoo-debug-resolved-policy.dot")
	if err != nil {
		log.Error().Err(err).Msg("failed to write debug graph")
		return
	}
	defer f.Close()

	f.WriteString("digraph \"resolvedpolicy\" {\n")
	for k, n := range ge.nodes {
		var shape string
		label := fmt.Sprintf("priority\n%d\ntype\n%s\n", ge.priorityMap[k], n.nodeType)
		switch n.nodeType {
		case ExecutionQueryNodeType:
			shape = "circle"
			nodeData := n.data.(*ExecutionQueryNodeData)
			label = fmt.Sprintf("%squery_id\n%s", label, nodeData.queryID)
		case DatapointNodeType:
			shape = "invtriangle"
			maxLen := 6
			if len(k) < 6 {
				maxLen = len(k)
			}
			label = fmt.Sprintf("%schecksum\n%s...", label, k[:maxLen])
		case ReportingQueryNodeType:
			shape = "ellipse"
			label = fmt.Sprintf("%squery_id\n%s", label, k)
		case ReportingJobNodeType:
			nodeData := n.data.(*ReportingJobNodeData)
			label = fmt.Sprintf("%srj_id\n%s\nquery_id\n%s\nscoring_system\n%s",
				label, k, nodeData.queryID, nodeData.scoringSystem.String())
			shape = "box"
		case DatapointCollectorNodeType:
			shape = "cds"
		case ScoreCollectorNodeType:
			shape = "cds"
		case CollectionFinisherNodeType:
			shape = "hexagon"
		}
		fmt.Fprintf(f, "\t%q [group=%s shape=%s label=%q]\n", k, n.nodeType, shape, label)
	}

	for from, tos := range ge.edges {
		for _, to := range tos {
			fmt.Fprintf(f, "\t%q -> %q\n", from, to)
		}
	}
	f.WriteString("}")

	if err := f.Close(); err != nil {
		log.Error().Err(err).Msg("failed to write debug graph")
		return
	}

	return
}
