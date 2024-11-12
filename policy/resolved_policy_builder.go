// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/explorer"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/mqlc"
	"go.mondoo.com/cnquery/v11/mrn"
)

// buildResolvedPolicy builds a resolved policy from a bundle
func buildResolvedPolicy(ctx context.Context, bundleMrn string, bundle *Bundle, assetFilters []*explorer.Mquery, now time.Time, compilerConf mqlc.CompilerConfig) (*ResolvedPolicy, error) {
	bundleMap := bundle.ToMap()
	assetFilterMap := make(map[string]struct{}, len(assetFilters))
	for _, f := range assetFilters {
		assetFilterMap[f.CodeId] = struct{}{}
	}

	policyObj := bundleMap.Policies[bundleMrn]
	frameworkObj := bundleMap.Frameworks[bundleMrn]

	builder := &resolvedPolicyBuilder{
		bundleMrn:            bundleMrn,
		bundleMap:            bundleMap,
		assetFilters:         assetFilterMap,
		nodes:                map[string]rpBuilderNode{},
		reportsToEdges:       map[string][]string{},
		reportsFromEdges:     map[string][]edgeImpact{},
		policyScoringSystems: map[string]explorer.ScoringSystem{},
		actionOverrides:      map[string]explorer.Action{},
		impactOverrides:      map[string]*explorer.Impact{},
		riskMagnitudes:       map[string]*RiskMagnitude{},
		propsCache:           explorer.NewPropsCache(),
		now:                  now,
	}

	actions, impacts, scoringSystems, riskMagnitudes := builder.gatherGlobalInfoFromPolicy(policyObj)
	builder.queryTypes = make(map[string]queryType)
	builder.collectQueryTypes(bundleMrn, builder.queryTypes)
	builder.actionOverrides = actions
	builder.impactOverrides = impacts
	builder.policyScoringSystems = scoringSystems
	builder.riskMagnitudes = riskMagnitudes

	builder.addPolicy(policyObj)

	if frameworkObj != nil {
		builder.addFramework(frameworkObj)
	}

	resolvedPolicyExecutionChecksum := BundleExecutionChecksum(ctx, policyObj, frameworkObj)
	assetFiltersChecksum, err := ChecksumAssetFilters(assetFilters, compilerConf)
	if err != nil {
		return nil, err
	}

	builderData := &rpBuilderData{
		baseChecksum: checksumStrings(resolvedPolicyExecutionChecksum, assetFiltersChecksum, "v2"),
		propsCache:   builder.propsCache,
		compilerConf: compilerConf,
	}

	resolvedPolicy := &ResolvedPolicy{
		ExecutionJob: &ExecutionJob{
			Checksum: "",
			Queries:  map[string]*ExecutionQuery{},
		},
		CollectorJob: &CollectorJob{
			Checksum:         "",
			ReportingJobs:    map[string]*ReportingJob{},
			ReportingQueries: map[string]*StringArray{},
			Datapoints:       map[string]*DataQueryInfo{},
			RiskMrns:         map[string]*StringArray{},
			RiskFactors:      map[string]*RiskFactor{},
		},
		Filters:                assetFilters,
		GraphExecutionChecksum: resolvedPolicyExecutionChecksum,
		FiltersChecksum:        assetFiltersChecksum,
	}

	// We will walk the graph from the non prunable nodes out. This means that if something is not connected
	// to a non prunable node, it will not be included in the resolved policy
	nonPrunables := make([]rpBuilderNode, 0, len(builder.nodes))

	for _, n := range builder.nodes {
		if !n.isPrunable() {
			nonPrunables = append(nonPrunables, n)
		}
	}

	visited := make(map[string]struct{}, len(builder.nodes))
	var walk func(node rpBuilderNode) error
	walk = func(node rpBuilderNode) error {
		// Check if we've already visited this node
		if _, ok := visited[node.getId()]; ok {
			return nil
		}
		visited[node.getId()] = struct{}{}

		// Build the necessary parts of the resolved policy for each node
		if err := node.build(resolvedPolicy, builderData); err != nil {
			log.Error().Err(err).Str("node", node.getId()).Msg("error building node")
			return err
		}
		// Walk to each parent node and recurse
		for _, edge := range builder.reportsToEdges[node.getId()] {
			if edgeNode, ok := builder.nodes[edge]; ok {
				if err := walk(edgeNode); err != nil {
					return err
				}

			} else {
				log.Debug().Str("from", node.getId()).Str("to", edge).Msg("edge not found")
			}
		}
		return nil
	}

	for _, n := range nonPrunables {
		if err := walk(n); err != nil {
			return nil, err
		}
	}

	// We need to connect the reporting jobs. We've stored them by uuid in the collector job. However,
	// our graph uses the qr id to connect them.
	reportingJobsByQrId := make(map[string]*ReportingJob, len(resolvedPolicy.CollectorJob.ReportingJobs))
	for _, rj := range resolvedPolicy.CollectorJob.ReportingJobs {
		if _, ok := reportingJobsByQrId[rj.QrId]; ok {
			// We should never have multiple reporting jobs with the same qr id. Scores are stored
			// by qr id, not by uuid. This would cause issues where scores would flop around
			log.Error().Str("qr_id", rj.QrId).Msg("multipe reporting jobs with the same qr id")
			return nil, errors.New("multiple reporting jobs with the same qr id")
		}
		reportingJobsByQrId[rj.QrId] = rj
	}

	// For each parent qr id, we need to connect the child reporting jobs with the impact.
	// connectReportingJobNotifies will add the link from the child to the parent, and
	// the parent to the child with the impact
	for parentQrId, edges := range builder.reportsFromEdges {
		for _, edge := range edges {
			parent := reportingJobsByQrId[parentQrId]
			if parent == nil {
				// It's possible that the parent reporting job was not included in the resolved policy
				// because it was not connected to a leaf node (e.g. a policy that was not connected to
				// any check or data query). In this case, we can just skip it
				log.Debug().Str("parent", parentQrId).Msg("reporting job not found")
				continue
			}

			if child, ok := reportingJobsByQrId[edge.edge]; ok {
				// Also possible a child was not included in the resolved policy
				connectReportingJobNotifies(child, parent, edge.impact)
			}
		}
	}

	rootReportingJob := reportingJobsByQrId[bundleMrn]
	if rootReportingJob == nil {
		return nil, explorer.NewAssetMatchError(bundleMrn, "policies", "no-matching-policy", assetFilters, policyObj.ComputedFilters)
	}
	rootReportingJob.QrId = "root"

	resolvedPolicy.ReportingJobUuid = rootReportingJob.Uuid

	refreshChecksums(resolvedPolicy.ExecutionJob, resolvedPolicy.CollectorJob)
	for _, rj := range resolvedPolicy.CollectorJob.ReportingJobs {
		rj.RefreshChecksum()
	}

	return resolvedPolicy, nil
}

// resolvedPolicyBuilder contains data that helps build the resolved policy. It maintains a graph of nodes.
// These nodes are the policies, controls, frameworks, checks, data queries, and execution queries. They
// get a chance to add themselves to the resolved policy in the way that they need to be added. They all
// add reporting jobs. Some nodes do other things like add the compiled query to the resolved policy. These nodes
// are connected by edges. These edges are the edges used to connect the reporting jobs in the resolved policy.
// Edges are added using the addEdge method. This will take care of maintaining the notifies edge and the childJobs
// edge from the reporting jobs simultaneously so that they are in sync.
type resolvedPolicyBuilder struct {
	// bundleMrn is the mrn of the bundle that is being resolved. It will be replaced by "root" in the
	// resolved policy's reporting jobs so that it can be reused by other bundles that are identical in
	// everything except the mrn of the root.
	bundleMrn string
	// bundleMap is the bundle that is being resolved converted into a PolicyBundleMap
	bundleMap *PolicyBundleMap

	// nodes is a map of all the nodes that are in the graph. These nodes will build the resolved
	// policy. nodes is walked from the non prunable nodes out. This means that if something is not
	// connected to a non prunable node, it will not be included in the resolved policy
	nodes map[string]rpBuilderNode
	// reportsToEdges maintains the notifies edges from the reporting jobs.
	reportsToEdges map[string][]string
	// reportsFromEdges maintains the childJobs edges from the reporting jobs. This is where the impact
	// is stored.
	reportsFromEdges map[string][]edgeImpact

	// assetFilters is the asset filters that are used to select the policies and queries that are
	// run
	assetFilters map[string]struct{}
	// policyScoringSystems is a map of the scoring systems for each policy
	policyScoringSystems map[string]explorer.ScoringSystem
	// actionOverrides is a map of the actions that are overridden by the policies
	actionOverrides map[string]explorer.Action
	// impactOverrides is a map of the impacts that are overridden by the policies. The worst impact
	// is used
	impactOverrides map[string]*explorer.Impact
	// riskMagnitudes is a map of the risk magnitudes that are set for risk factors
	riskMagnitudes map[string]*RiskMagnitude
	// queryTypes is a map of the query types for each query. A query can be a scoring query, a data query,
	// or both. We analyze all matching policies to determine the query type. If a query shows up in checks,
	// it is a scoring query. If it shows up in data queries, it is a data query. If it shows up in both, it is
	// set to both.
	queryTypes map[string]queryType
	// propsCache is a cache of the properties that are used in the queries
	propsCache explorer.PropsCache
	// now is the time that the resolved policy is being built
	now time.Time
}

type edgeImpact struct {
	edge   string
	impact *explorer.Impact
}

// rpBuilderNode is a node in the graph. It represents a policy, control, framework, check, data query, or execution query.
// Each node implementation decides how it needs to be added to the resolved policy. It is currently assumed that
// each node will add a reporting job to the resolved policy, as the edges are used to automatically connect the reporting jobs.
type rpBuilderNode interface {
	// getId returns the id of the node. This is used to identify the node in the graph, a mrn or code id
	getId() string
	// isPrunable returns whether the node can be pruned from the graph. It will be pruned if it a non-prunable node
	// doesn't have a path TO it. In context of building the resolved policy, this means that the node is not connected
	// to an executable query, or is the root node.
	isPrunable() bool
	// build is responsible for updating the resolved policy. It will add things like reporting jobs, connect datapoints,
	// adding the compiled query, etc.
	build(*ResolvedPolicy, *rpBuilderData) error
}

// rpBuilderData is the data that is used to build the resolved policy
type rpBuilderData struct {
	baseChecksum string
	propsCache   explorer.PropsCache
	compilerConf mqlc.CompilerConfig
}

func (d *rpBuilderData) relativeChecksum(s string) string {
	return checksumStrings(d.baseChecksum, s)
}

// rpBuilderPolicyNode is a node that represents a policy in the graph. It will add a reporting job to the resolved policy
// for the policy
type rpBuilderPolicyNode struct {
	policy        *Policy
	scoringSystem explorer.ScoringSystem
	isRoot        bool
}

func (n *rpBuilderPolicyNode) getId() string {
	return n.policy.Mrn
}

func (n *rpBuilderPolicyNode) isPrunable() bool {
	// We do not allow pruning the root node. This covers cases where the policy matches the asset filters,
	// but we have no active checks or queries. This will end up reporting a U for the score
	return !n.isRoot
}

func (n *rpBuilderPolicyNode) build(rp *ResolvedPolicy, data *rpBuilderData) error {
	if n.isRoot {
		// If the policy is the root, we need a different checksum for the reporting job because we want it
		// to be reusable by other bundles that are identical in everything except the root mrn
		addReportingJob(n.policy.Mrn, true, data.relativeChecksum(n.policy.GraphExecutionChecksum), ReportingJob_POLICY, rp)
	} else {
		// the uuid used to be a checksum of the policy mrn, impact, and action
		// I don't think this can be correct in all cases as you could at some point
		// have a policy report to multiple other policies with different impacts
		// (we don't have that case right now)
		// These checksum changes should be accounted for in the root
		rj := addReportingJob(n.policy.Mrn, true, data.relativeChecksum(n.policy.Mrn), ReportingJob_POLICY, rp)
		rj.ScoringSystem = n.scoringSystem
	}

	return nil
}

// rpBuilderGenericQueryNode is a node that represents a query by mrn in the graph. It will add a reporting job,
// and fill out the reporting queries in the resolved policy
type rpBuilderGenericQueryNode struct {
	// queryMrn is the mrn of the query
	queryMrn string
	// queryType is the type of query. It can be a scoring query, a data query, or both
	queryType queryType
	// selectedCodeId is the code id that actually gets executed. It is the code id of the specific query
	// that is run, traversed down the variants if necessary. We keep track of this because we need to connect
	// controls to the specific query that is run so they are not influenced by impacts
	selectedCodeId string
}

func (n *rpBuilderGenericQueryNode) getId() string {
	return n.queryMrn
}

func (n *rpBuilderGenericQueryNode) isPrunable() bool {
	return true
}

func (n *rpBuilderGenericQueryNode) build(rp *ResolvedPolicy, data *rpBuilderData) error {
	reportingJobUUID := data.relativeChecksum(n.queryMrn)

	// Because a query can be both a scoring query and a data query, UNSPECIFIED is used
	// for the reporting job type. We need to get rid of the specific types for check and
	// data query and have something that can be both
	addReportingJob(n.queryMrn, true, reportingJobUUID, ReportingJob_UNSPECIFIED, rp)

	// Add scoring queries to the reporting queries section
	if n.queryType == queryTypeScoring || n.queryType == queryTypeBoth {
		if _, ok := rp.CollectorJob.ReportingQueries[n.selectedCodeId]; !ok {
			rp.CollectorJob.ReportingQueries[n.selectedCodeId] = &StringArray{}
		}
		rp.CollectorJob.ReportingQueries[n.selectedCodeId].Items = append(rp.CollectorJob.ReportingQueries[n.selectedCodeId].Items, reportingJobUUID)
	}

	return nil
}

// rpBuilderExecutionQueryNode is a node that represents a executable query in the graph. It will add a reporting job to the resolved policy,
// and add the compiled query to the execution job, and connect the datapoints to the reporting job.
// This node is a leaf. Anything connected to an executable query will not be pruned.
// This node is represented by a code id in the reporting jobs. We do not apply impact at this point so
// any scores will be either 0 or 100
type rpBuilderExecutionQueryNode struct {
	query *explorer.Mquery
}

func (n *rpBuilderExecutionQueryNode) getId() string {
	return n.query.CodeId
}

func (n *rpBuilderExecutionQueryNode) isPrunable() bool {
	// Executable queries are leaf nodes in the graph. They cannot be pruned
	// If something is connected to an executable query, we want to keep it around
	return false
}

func (n *rpBuilderExecutionQueryNode) build(rp *ResolvedPolicy, data *rpBuilderData) error {
	// Compile the properties
	propTypes, propToChecksums, err := compileProps(n.query, rp, data)
	if err != nil {
		return err
	}
	// Add the compiled query to the execution job. This also collects the datapoints into the collector job
	executionQuery, _, err := mquery2executionQuery(n.query, propTypes, propToChecksums, rp.CollectorJob, false, data.compilerConf)
	if err != nil {
		return err
	}
	rp.ExecutionJob.Queries[n.query.CodeId] = executionQuery

	codeIdReportingJobUUID := data.relativeChecksum(n.query.CodeId)

	// Create a reporting job for the code id
	codeIdReportingJob := addReportingJob(n.query.CodeId, false, codeIdReportingJobUUID, ReportingJob_UNSPECIFIED, rp)
	// Connect the datapoints to the reporting job
	err = connectDatapointsToReportingJob(executionQuery, codeIdReportingJob, rp.CollectorJob.Datapoints)
	if err != nil {
		return err
	}

	return nil
}

// rpBuilderFrameworkNode is a node that represents a framework in the graph. It will add a reporting job to the resolved policy
type rpBuilderFrameworkNode struct {
	frameworkMrn string
}

func (n *rpBuilderFrameworkNode) getId() string {
	return n.frameworkMrn
}

func (n *rpBuilderFrameworkNode) isPrunable() bool {
	return true
}

func (n *rpBuilderFrameworkNode) build(rp *ResolvedPolicy, data *rpBuilderData) error {
	addReportingJob(n.frameworkMrn, true, data.relativeChecksum(n.frameworkMrn), ReportingJob_FRAMEWORK, rp)
	return nil
}

// rpBuilderControlNode is a node that represents a control in the graph. It will add a reporting job to the resolved policy
type rpBuilderControlNode struct {
	controlMrn string
}

func (n *rpBuilderControlNode) getId() string {
	return n.controlMrn
}

func (n *rpBuilderControlNode) isPrunable() bool {
	return true
}

func (n *rpBuilderControlNode) build(rp *ResolvedPolicy, data *rpBuilderData) error {
	addReportingJob(n.controlMrn, true, data.relativeChecksum(n.controlMrn), ReportingJob_CONTROL, rp)
	return nil
}

// rpBuilderRiskFactorNode is a node that represents a risk factor in the graph. It will add a reporting job to the resolved policy,
// and fill out the RiskFactors and RiskMrns sections in the collector job
type rpBuilderRiskFactorNode struct {
	riskFactor      *RiskFactor
	magnitude       *RiskMagnitude
	selectedCodeIds []string
}

func (n *rpBuilderRiskFactorNode) getId() string {
	return n.riskFactor.Mrn
}

func (n *rpBuilderRiskFactorNode) isPrunable() bool {
	return true
}

func (n *rpBuilderRiskFactorNode) build(rp *ResolvedPolicy, data *rpBuilderData) error {
	risk := n.riskFactor
	if n.magnitude != nil {
		risk.Magnitude = n.magnitude
	}
	rp.CollectorJob.RiskFactors[risk.Mrn] = &RiskFactor{
		Scope:                   risk.Scope,
		Magnitude:               risk.Magnitude,
		Resources:               risk.Resources,
		DeprecatedV11Magnitude:  risk.Magnitude.GetValue(),
		DeprecatedV11IsAbsolute: risk.Magnitude.GetIsToxic(),
	}
	addReportingJob(risk.Mrn, true, data.relativeChecksum(risk.Mrn), ReportingJob_RISK_FACTOR, rp)

	for _, codeId := range n.selectedCodeIds {
		if _, ok := rp.CollectorJob.RiskMrns[codeId]; !ok {
			rp.CollectorJob.RiskMrns[codeId] = &StringArray{
				Items: []string{},
			}
		}
		rp.CollectorJob.RiskMrns[codeId].Items = append(rp.CollectorJob.RiskMrns[codeId].Items, risk.Mrn)
	}
	return nil
}

func (b *resolvedPolicyBuilder) addEdge(from, to string, impact *explorer.Impact) {
	if _, ok := b.reportsToEdges[from]; !ok {
		b.reportsToEdges[from] = make([]string, 0, 1)
	}
	for _, e := range b.reportsToEdges[from] {
		// If the edge already exists, don't add it
		if e == to {
			return
		}
	}
	b.reportsToEdges[from] = append(b.reportsToEdges[from], to)

	if _, ok := b.reportsFromEdges[to]; !ok {
		b.reportsFromEdges[to] = make([]edgeImpact, 0, 1)
	}

	b.reportsFromEdges[to] = append(b.reportsFromEdges[to], edgeImpact{edge: from, impact: impact})
}

func (b *resolvedPolicyBuilder) addNode(node rpBuilderNode) {
	b.nodes[node.getId()] = node
}

type queryType int

const (
	queryTypeScoring queryType = iota
	queryTypeData
	queryTypeBoth
)

// collectQueryTypes collects the query types for each query in the policy. A query can be a scoring query, a data query,
// or both. We analyze all matching policies to determine the query type. If a query shows up in checks, it is a scoring query.
// If it shows up in data queries, it is a data query. If it shows up in both, it is set to both.
func (b *resolvedPolicyBuilder) collectQueryTypes(policyMrn string, acc map[string]queryType) {
	policy := b.bundleMap.Policies[policyMrn]
	if policy == nil {
		return
	}

	var accumulate func(queryMrn string, t queryType)
	accumulate = func(queryMrn string, t queryType) {
		if existing, ok := acc[queryMrn]; !ok {
			// If it doesn't exist, add it
			acc[queryMrn] = t
		} else {
			if existing != t && existing != queryTypeBoth {
				// If it exists, but is different, set it to both
				acc[queryMrn] = queryTypeBoth
			}
		}
		q := b.bundleMap.Queries[queryMrn]
		if q == nil {
			return
		}

		for _, v := range q.Variants {
			accumulate(v.Mrn, t)
		}
	}

	for _, g := range policy.Groups {
		if !b.isPolicyGroupMatching(g) {
			// skip groups that don't match
			continue
		}

		for _, c := range g.Checks {
			accumulate(c.Mrn, queryTypeScoring)
		}

		for _, q := range g.Queries {
			accumulate(q.Mrn, queryTypeData)
		}

		for _, pRef := range g.Policies {
			// recursively collect query types from referenced policies
			b.collectQueryTypes(pRef.Mrn, acc)
		}
	}

	// queries in risk factors are checks
	for _, r := range policy.RiskFactors {
		for _, c := range r.Checks {
			accumulate(c.Mrn, queryTypeScoring)
		}
	}
}

// gatherGlobalInfoFromPolicy gathers the action, impact, scoring system, and risk magnitude overrides from the policy. We
// apply this information in a second pass when building the nodes
func (b *resolvedPolicyBuilder) gatherGlobalInfoFromPolicy(policy *Policy) (map[string]explorer.Action, map[string]*explorer.Impact, map[string]explorer.ScoringSystem, map[string]*RiskMagnitude) {
	actions := make(map[string]explorer.Action)
	impacts := make(map[string]*explorer.Impact)
	scoringSystems := make(map[string]explorer.ScoringSystem)
	riskMagnitudes := make(map[string]*RiskMagnitude)

	for _, g := range policy.Groups {
		if !b.isPolicyGroupMatching(g) {
			continue
		}
		for _, pRef := range g.Policies {
			p := b.bundleMap.Policies[pRef.Mrn]

			a, i, s, r := b.gatherGlobalInfoFromPolicy(p)
			for k, v := range a {
				actions[k] = v
			}

			for k, v := range i {
				impacts[k] = v
			}

			for k, v := range s {
				scoringSystems[k] = v
			}

			for k, v := range r {
				riskMagnitudes[k] = v
			}

			action := normalizeAction(g.Type, pRef.Action, pRef.Impact)
			if action != explorer.Action_UNSPECIFIED && action != explorer.Action_MODIFY {
				actions[pRef.Mrn] = action
			}

			if pRef.Impact != nil {
				impacts[pRef.Mrn] = pRef.Impact
			}
			scoringSystem := pRef.ScoringSystem

			if scoringSystem != explorer.ScoringSystem_SCORING_UNSPECIFIED {
				scoringSystems[pRef.Mrn] = pRef.ScoringSystem
			} else {
				if p, ok := b.bundleMap.Policies[pRef.Mrn]; ok {
					scoringSystems[pRef.Mrn] = p.ScoringSystem
				}
			}
		}

		// We always want to select the worst impact that we find
		getWorstImpact := func(impact1 *explorer.Impact, impact2 *explorer.Impact) *explorer.Impact {
			if impact1 == nil {
				return impact2
			}
			if impact2 == nil {
				return impact1
			}

			if impact1.Value.GetValue() > impact2.Value.GetValue() {
				return impact1
			}
			return impact2
		}

		for _, c := range g.Checks {
			impact := c.Impact
			action := normalizeAction(g.Type, c.Action, impact)
			if action == explorer.Action_IGNORE {
				// Since we traversed the children first, we overwrite the impact with ignore
				// if the action is ignore. We do not merge in this case
				impact = &explorer.Impact{
					Scoring: explorer.ScoringSystem_IGNORE_SCORE,
				}
			} else {
				if qBundle, ok := b.bundleMap.Queries[c.Mrn]; ok {
					// Check the impact defined on the query
					impact = getWorstImpact(impact, qBundle.Impact)
				}

				impact = getWorstImpact(impact, impacts[c.Mrn])
			}

			if action != explorer.Action_UNSPECIFIED && action != explorer.Action_MODIFY {
				actions[c.Mrn] = action
			}

			if impact != nil {
				impacts[c.Mrn] = impact
			}
		}

		for _, q := range g.Queries {
			if q.Action != explorer.Action_UNSPECIFIED {
				action := normalizeAction(g.Type, q.Action, q.Impact)
				if action != explorer.Action_UNSPECIFIED && action != explorer.Action_MODIFY {
					actions[q.Mrn] = action
				}
			}
		}
	}

	for _, r := range policy.RiskFactors {
		if r.Magnitude != nil {
			riskMagnitudes[r.Mrn] = r.Magnitude
		}

		if r.Action != explorer.Action_UNSPECIFIED && r.Action != explorer.Action_MODIFY {
			actions[r.Mrn] = r.Action
		}
	}

	return actions, impacts, scoringSystems, riskMagnitudes
}

func canRun(action explorer.Action) bool {
	return !(action == explorer.Action_DEACTIVATE || action == explorer.Action_OUT_OF_SCOPE)
}

// isPolicyGroupMatching checks if the policy group is matching. A policy group is matching if it is not rejected,
// and it is not expired. If it has filters, it must have at least one filter that matches the asset filters
func (b *resolvedPolicyBuilder) isPolicyGroupMatching(group *PolicyGroup) bool {
	if group.ReviewStatus == ReviewStatus_REJECTED {
		return false
	}

	if group.EndDate != 0 {
		// TODO: we also need to check if the group is accepted or rejected
		endDate := time.Unix(group.EndDate, 0)
		if endDate.Before(b.now) {
			return false
		}
	}

	if group.Filters == nil || len(group.Filters.Items) == 0 {
		return true
	}

	for _, filter := range group.Filters.Items {
		if _, ok := b.assetFilters[filter.CodeId]; ok {
			return true
		}
	}

	return false
}

// addPolicy recurses a policy and adds all the nodes and edges to the graph. It will add the policy, its dependent policies, checks, and queries
func (b *resolvedPolicyBuilder) addPolicy(policy *Policy) bool {
	action := b.actionOverrides[policy.Mrn]

	// Check if we can run this policy. If not, then we do not add it to the graph
	if !canRun(action) {
		return false
	}

	if !b.anyFilterMatches(policy.ComputedFilters) {
		return false
	}

	b.propsCache.Add(policy.Props...)

	// Add node for policy
	scoringSystem := b.policyScoringSystems[policy.Mrn]
	b.addNode(&rpBuilderPolicyNode{policy: policy, scoringSystem: scoringSystem, isRoot: b.bundleMrn == policy.Mrn})
	hasMatchingGroup := false
	for _, g := range policy.Groups {
		if !b.isPolicyGroupMatching(g) {
			continue
		}
		hasMatchingGroup = true
		for _, pRef := range g.Policies {
			p := b.bundleMap.Policies[pRef.Mrn]
			if b.addPolicy(p) {
				var impact *explorer.Impact
				if pRefAction, ok := b.actionOverrides[pRef.Mrn]; ok && pRefAction == explorer.Action_IGNORE {
					impact = &explorer.Impact{
						Scoring: explorer.ScoringSystem_IGNORE_SCORE,
					}
				} else if i, ok := b.impactOverrides[pRef.Mrn]; ok {
					impact = i
				}
				b.addEdge(pRef.Mrn, policy.Mrn, impact)
			}
		}

		for _, c := range g.Checks {
			// Check the action. If its an override, we don't need to add the check
			// because it will get included in a policy that wants it run.
			// This will prevent the check from being connected to the policy that
			// overrides its action
			if isOverride(c.Action) {
				b.propsCache.Add(c.Props...)
				continue
			}

			c, ok := b.bundleMap.Queries[c.Mrn]
			if !ok {
				log.Warn().Str("mrn", c.Mrn).Msg("check not found in bundle")
				continue
			}

			if _, ok := b.addQuery(c); ok {
				b.addEdge(c.Mrn, policy.Mrn, nil)
			}
		}

		for _, q := range g.Queries {
			// Check the action. If its an override, we don't need to add the query
			// because it will get included in a policy that wants it run.
			// This will prevent the query from being connected to the policy that
			// overrides its action
			if isOverride(q.Action) {
				b.propsCache.Add(q.Props...)
				continue
			}

			q, ok := b.bundleMap.Queries[q.Mrn]
			if !ok {
				log.Warn().Str("mrn", q.Mrn).Msg("query not found in bundle")
				continue
			}

			if _, ok := b.addQuery(q); ok {
				b.addEdge(q.Mrn, policy.Mrn, &explorer.Impact{
					Scoring: explorer.ScoringSystem_IGNORE_SCORE,
				})
			}
		}
	}

	hasMatchingRiskFactor := false
	for _, r := range policy.RiskFactors {
		if len(r.Checks) == 0 || isOverride(r.Action) {
			continue
		}

		if b.addRiskFactor(r) {
			b.addEdge(r.Mrn, policy.Mrn, &explorer.Impact{Scoring: explorer.ScoringSystem_IGNORE_SCORE})
			hasMatchingRiskFactor = true
		}
	}

	return hasMatchingGroup || hasMatchingRiskFactor
}

// addQuery adds a query to the graph. It will add the query, its variants, and connect the query to the variants
func (b *resolvedPolicyBuilder) addQuery(query *explorer.Mquery) (string, bool) {
	action := b.actionOverrides[query.Mrn]
	impact := b.impactOverrides[query.Mrn]
	queryType := b.queryTypes[query.Mrn]

	if !canRun(action) {
		return "", false
	}

	if len(query.Variants) != 0 {
		// If we have variants, we need to find the first matching variant.
		// We will also recursively find the code id of the query that will
		// be run
		var matchingVariant *explorer.Mquery
		var selectedCodeId string
		for _, v := range query.Variants {
			q, ok := b.bundleMap.Queries[v.Mrn]
			if !ok {
				log.Warn().Str("mrn", v.Mrn).Msg("variant not found in bundle")
				continue
			}
			if codeId, added := b.addQuery(q); added {
				// The first matching variant is selected
				matchingVariant = q
				selectedCodeId = codeId
				break
			}
		}

		if matchingVariant == nil {
			return "", false
		}

		b.propsCache.Add(query.Props...)
		b.propsCache.Add(matchingVariant.Props...)

		// Add node for query
		b.addNode(&rpBuilderGenericQueryNode{queryMrn: query.Mrn, selectedCodeId: selectedCodeId, queryType: queryType})

		// Add edge from variant to query
		b.addEdge(matchingVariant.Mrn, query.Mrn, impact)

		return selectedCodeId, true
	} else {
		if !b.anyFilterMatches(query.Filters) {
			return "", false
		}

		b.propsCache.Add(query.Props...)

		// Add node for execution query
		b.addNode(&rpBuilderExecutionQueryNode{query: query})
		// Add node for query
		b.addNode(&rpBuilderGenericQueryNode{queryMrn: query.Mrn, selectedCodeId: query.CodeId, queryType: queryType})

		// Add edge from execution query to query
		b.addEdge(query.CodeId, query.Mrn, impact)

		return query.CodeId, true
	}
}

// addRiskFactor adds a risk factor to the graph. It will add the risk factor, its checks, and connect the checks to the risk factor
func (b *resolvedPolicyBuilder) addRiskFactor(riskFactor *RiskFactor) bool {
	action := b.actionOverrides[riskFactor.Mrn]
	if !canRun(action) {
		return false
	}

	if !b.anyFilterMatches(riskFactor.Filters) {
		return false
	}

	selectedCodeIds := make([]string, 0, len(riskFactor.Checks))
	for _, c := range riskFactor.Checks {
		if selectedCodeId, ok := b.addQuery(c); ok {
			selectedCodeIds = append(selectedCodeIds, selectedCodeId)
			b.addEdge(c.Mrn, riskFactor.Mrn, &explorer.Impact{Scoring: explorer.ScoringSystem_IGNORE_SCORE})
		}
	}

	if len(selectedCodeIds) == 0 {
		return false
	}

	b.addNode(&rpBuilderRiskFactorNode{riskFactor: riskFactor, magnitude: b.riskMagnitudes[riskFactor.Mrn], selectedCodeIds: selectedCodeIds})

	return true
}

func (b *resolvedPolicyBuilder) anyFilterMatches(f *explorer.Filters) bool {
	return f.Supports(b.assetFilters)
}

// addFramework adds a framework to the graph. It will add the framework, its dependent frameworks, its controls, and connect
// the controls to the framework
func (b *resolvedPolicyBuilder) addFramework(framework *Framework) bool {
	action := b.actionOverrides[framework.Mrn]
	if !canRun(action) {
		return false
	}

	// Create a node for the framework, but only if its a valid framework mrn
	// Otherwise, we have the asset / space policies which we will connect
	// to. We need to do this because we cannot have a space frame and space
	// policy reporting job because they would have the same qr id.
	// TODO: we should create a new reporting job type for asset and space
	// reporting jobs so its cleare that we can connect both frameworks and
	// policies to them
	// If the node already exists, its represented by the asset or space policy
	// and is not a valid framework mrn
	var impact *explorer.Impact
	if _, ok := b.nodes[framework.Mrn]; !ok {
		b.addNode(&rpBuilderFrameworkNode{frameworkMrn: framework.Mrn})
	} else {
		impact = &explorer.Impact{Scoring: explorer.ScoringSystem_IGNORE_SCORE}
	}

	for _, fmap := range framework.FrameworkMaps {
		for _, control := range fmap.Controls {
			if b.addControl(control) {
				b.addEdge(control.Mrn, fmap.FrameworkOwner.Mrn, nil)
			}
		}
	}

	for _, fdep := range framework.Dependencies {
		f, ok := b.bundleMap.Frameworks[fdep.Mrn]
		if !ok {
			log.Warn().Str("mrn", fdep.Mrn).Msg("framework not found in bundle")
			continue
		}
		if b.addFramework(f) {
			b.addEdge(fdep.Mrn, framework.Mrn, impact)
		}
	}

	return true
}

// addControl adds a control to the graph and connect policies, controls, checks, and queries to the control
func (b *resolvedPolicyBuilder) addControl(control *ControlMap) bool {
	action := b.actionOverrides[control.Mrn]
	if !canRun(action) {
		return false
	}

	hasChild := false

	for _, q := range control.Checks {
		if _, ok := b.nodes[q.Mrn]; ok {
			n := b.nodes[q.Mrn]
			if n == nil {
				continue
			}
			qNode, ok := n.(*rpBuilderGenericQueryNode)
			if ok {
				b.addEdge(qNode.selectedCodeId, control.Mrn, nil)
				hasChild = true
			}
		}
	}

	for _, q := range control.Queries {
		if _, ok := b.nodes[q.Mrn]; ok {
			n := b.nodes[q.Mrn]
			if n == nil {
				continue
			}
			qNode, ok := n.(*rpBuilderGenericQueryNode)
			if ok {
				b.addEdge(qNode.selectedCodeId, control.Mrn, nil)
				hasChild = true
			}
		}
	}

	for _, p := range control.Policies {
		if _, ok := b.nodes[p.Mrn]; ok {
			// Add the edge from the control to the policy
			b.addEdge(p.Mrn, control.Mrn, nil)
			hasChild = true
		}
	}

	for _, c := range control.Controls {
		// We will just assume that the control is in the graph
		// If its not, it will get filtered out later when we build
		// the resolved policy
		// Doing this so we don't need to topologically sort the dependency
		// tree for the controls
		b.addEdge(c.Mrn, control.Mrn, nil)
		hasChild = true
	}

	if hasChild {
		// Add node for control
		b.addNode(&rpBuilderControlNode{controlMrn: control.Mrn})
	}

	return true
}

func addReportingJob(qrId string, qrIdIsMrn bool, uuid string, typ ReportingJob_Type, rp *ResolvedPolicy) *ReportingJob {
	if _, ok := rp.CollectorJob.ReportingJobs[uuid]; !ok {
		rp.CollectorJob.ReportingJobs[uuid] = &ReportingJob{
			QrId:       qrId,
			Uuid:       uuid,
			ChildJobs:  map[string]*explorer.Impact{},
			Datapoints: map[string]bool{},
			Type:       typ,
		}
		if qrIdIsMrn {
			rp.CollectorJob.ReportingJobs[uuid].Mrns = []string{qrId}
		}
	}
	return rp.CollectorJob.ReportingJobs[uuid]
}

func compileProps(query *explorer.Mquery, rp *ResolvedPolicy, data *rpBuilderData) (map[string]*llx.Primitive, map[string]string, error) {
	var propTypes map[string]*llx.Primitive
	var propToChecksums map[string]string
	if len(query.Props) != 0 {
		propTypes = make(map[string]*llx.Primitive, len(query.Props))
		propToChecksums = make(map[string]string, len(query.Props))
		for j := range query.Props {
			prop := query.Props[j]

			// we only get this if there is an override higher up in the policy
			override, name, _ := data.propsCache.Get(prop.Mrn)
			if override != nil {
				prop = override
			}
			if name == "" {
				var err error
				name, err = mrn.GetResource(prop.Mrn, MRN_RESOURCE_QUERY)
				if err != nil {
					return nil, nil, errors.New("failed to get property name")
				}
			}

			executionQuery, dataChecksum, err := mquery2executionQuery(prop, nil, map[string]string{}, rp.CollectorJob, false, data.compilerConf)
			if err != nil {
				return nil, nil, errors.New("resolver> failed to compile query for MRN " + prop.Mrn + ": " + err.Error())
			}
			if dataChecksum == "" {
				return nil, nil, errors.New("property returns too many value, cannot determine entrypoint checksum: '" + prop.Mql + "'")
			}
			rp.ExecutionJob.Queries[prop.CodeId] = executionQuery

			propTypes[name] = &llx.Primitive{Type: prop.Type}
			propToChecksums[name] = dataChecksum
		}
	}
	return propTypes, propToChecksums, nil
}

// connectReportingJobNotifies adds the notifies and child jobs links in the reporting jobs
func connectReportingJobNotifies(child *ReportingJob, parent *ReportingJob, impact *explorer.Impact) {
	for _, n := range child.Notify {
		if n == parent.Uuid {
			fmt.Println("already connected")
		}
	}
	child.Notify = append(child.Notify, parent.Uuid)
	parent.ChildJobs[child.Uuid] = impact
}

// normalizeAction normalizes the action based on the group type and impact. We need to do this because
// we've had different ways of representing actions in the past and we need to normalize them to the current
func normalizeAction(groupType GroupType, action explorer.Action, impact *explorer.Impact) explorer.Action {
	switch groupType {
	case GroupType_DISABLE:
		return explorer.Action_DEACTIVATE
	case GroupType_OUT_OF_SCOPE:
		return explorer.Action_OUT_OF_SCOPE
	case GroupType_IGNORED:
		return explorer.Action_IGNORE
	default:
		if impact != nil && impact.Scoring == explorer.ScoringSystem_IGNORE_SCORE {
			return explorer.Action_IGNORE
		}
		return action
	}
}

func isOverride(action explorer.Action) bool {
	return action != explorer.Action_UNSPECIFIED
}
