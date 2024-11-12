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

type edgeImpact struct {
	edge   string
	impact *explorer.Impact
}

type resolvedPolicyBuilder struct {
	bundleMrn            string
	bundleMap            *PolicyBundleMap
	assetFilters         map[string]struct{}
	nodes                map[string]rpBuilderNode
	reportsToEdges       map[string][]string
	reportsFromEdges     map[string][]edgeImpact
	policyScoringSystems map[string]explorer.ScoringSystem
	actionOverrides      map[string]explorer.Action
	impactOverrides      map[string]*explorer.Impact
	riskMagnitudes       map[string]*RiskMagnitude
	queryTypes           map[string]queryType
	propsCache           explorer.PropsCache
	now                  time.Time
}

type rpBuilderNodeType int

const (
	// rpBuilderNodeTypePolicy is a policy node. Checks and data queries report to this
	rpBuilderNodeTypePolicy = iota
	// rpBuilderNodeTypeFramework is a framework node. Controls report to this
	rpBuilderNodeTypeFramework
	// rpBuilderNodeTypeControl is a control node. Checks, data queries, and policies report to this
	rpBuilderNodeTypeControl
	// rpBuilderTypeRiskFactor is a risk factor node. This is the leaf node
	rpBuilderNodeTypeRiskFactor
	// rpBuilderNodeTypeQuery is a check and/or a data query. Execution queries report to this
	rpBuilderNodeTypeQuery
	// rpBuilderNodeTypeExecutionQuery is an execution query. This is the leaf node
	rpBuilderNodeTypeExecutionQuery
)

// rpBuilderData is the data that is used to build the resolved policy
type rpBuilderData struct {
	baseChecksum    string
	impactOverrides map[string]*explorer.Impact
	propsCache      explorer.PropsCache
	compilerConf    mqlc.CompilerConfig
}

func (d *rpBuilderData) relativeChecksum(s string) string {
	return checksumStrings(d.baseChecksum, s)
}

type rpBuilderNode interface {
	getType() rpBuilderNodeType
	getId() string
	isPrunable() bool
	build(*ResolvedPolicy, *rpBuilderData) error
}

type rpBuilderPolicyNode struct {
	policy        *Policy
	scoringSystem explorer.ScoringSystem
	isRoot        bool
}

func (n *rpBuilderPolicyNode) getType() rpBuilderNodeType {
	return rpBuilderNodeTypePolicy
}

func (n *rpBuilderPolicyNode) getId() string {
	return n.policy.Mrn
}

func (n *rpBuilderPolicyNode) isPrunable() bool {
	return n.isRoot
}

func (n *rpBuilderPolicyNode) build(rp *ResolvedPolicy, data *rpBuilderData) error {
	if n.isRoot {
		addReportingJob(n.policy.Mrn, true, data.relativeChecksum(n.policy.GraphExecutionChecksum), ReportingJob_POLICY, rp)
	} else {
		// TODO: the uuid used to be a checksum of the policy mrn, impact, and action
		// I don't think this can be correct in all cases as you could at some point
		// have a policy report to multiple other policies with different impacts
		// (we don't have that case right now)
		// These checksum changes should be accounted for in the root
		rj := addReportingJob(n.policy.Mrn, true, data.relativeChecksum(n.policy.Mrn), ReportingJob_POLICY, rp)
		rj.ScoringSystem = n.scoringSystem
	}

	return nil
}

type rpBuilderControlNode struct {
	controlMrn string
}

func (n *rpBuilderControlNode) getType() rpBuilderNodeType {
	return rpBuilderNodeTypeControl
}

func (n *rpBuilderControlNode) getId() string {
	return n.controlMrn
}

func (n *rpBuilderControlNode) isPrunable() bool {
	return false
}

func (n *rpBuilderControlNode) build(rp *ResolvedPolicy, data *rpBuilderData) error {
	addReportingJob(n.controlMrn, true, data.relativeChecksum(n.controlMrn), ReportingJob_CONTROL, rp)
	return nil
}

type rpBuilderFrameworkNode struct {
	frameworkMrn string
}

func (n *rpBuilderFrameworkNode) getType() rpBuilderNodeType {
	return rpBuilderNodeTypeFramework
}

func (n *rpBuilderFrameworkNode) getId() string {
	return n.frameworkMrn
}

func (n *rpBuilderFrameworkNode) isPrunable() bool {
	return false
}

func (n *rpBuilderFrameworkNode) build(rp *ResolvedPolicy, data *rpBuilderData) error {
	addReportingJob(n.frameworkMrn, true, data.relativeChecksum(n.frameworkMrn), ReportingJob_FRAMEWORK, rp)
	return nil
}

type rpBuilderRiskFactorNode struct {
	riskFactor      *RiskFactor
	magnitude       *RiskMagnitude
	selectedCodeIds []string
}

func (n *rpBuilderRiskFactorNode) getType() rpBuilderNodeType {
	return rpBuilderNodeTypeRiskFactor
}

func (n *rpBuilderRiskFactorNode) getId() string {
	return n.riskFactor.Mrn
}

func (n *rpBuilderRiskFactorNode) isPrunable() bool {
	return false
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

type rpBuilderExecutionQueryNode struct {
	query *explorer.Mquery
}

func (n *rpBuilderExecutionQueryNode) getType() rpBuilderNodeType {
	return rpBuilderNodeTypeExecutionQuery
}

func (n *rpBuilderExecutionQueryNode) getId() string {
	return n.query.CodeId
}

func (n *rpBuilderExecutionQueryNode) isPrunable() bool {
	return true
}

func (n *rpBuilderExecutionQueryNode) build(rp *ResolvedPolicy, data *rpBuilderData) error {
	propTypes, propToChecksums, err := compileProps(n.query, rp, data)
	if err != nil {
		return err
	}
	if rp.ExecutionJob.Queries[n.query.CodeId] == nil {
		eq, _, err := mquery2executionQuery(n.query, propTypes, propToChecksums, rp.CollectorJob, false, data.compilerConf)
		if err != nil {
			return err
		}
		rp.ExecutionJob.Queries[n.query.CodeId] = eq
	}

	executionQuery := rp.ExecutionJob.Queries[n.query.CodeId]

	codeIdReportingJobUUID := data.relativeChecksum(n.query.CodeId)

	if _, ok := rp.CollectorJob.ReportingJobs[codeIdReportingJobUUID]; !ok {
		codeIdReportingJob := addReportingJob(n.query.CodeId, false, codeIdReportingJobUUID, ReportingJob_UNSPECIFIED, rp)
		connectDatapointsToReportingJob(executionQuery, codeIdReportingJob, rp.CollectorJob.Datapoints)
	}

	return nil
}

type rpBuilderGenericQueryNode struct {
	query           *explorer.Mquery
	selectedVariant *explorer.Mquery
	queryType       queryType
	selectedCodeId  string
}

func (n *rpBuilderGenericQueryNode) getType() rpBuilderNodeType {
	return rpBuilderNodeTypeQuery
}

func (n *rpBuilderGenericQueryNode) getId() string {
	return n.query.Mrn
}

func (n *rpBuilderGenericQueryNode) isPrunable() bool {
	return false
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

func (n *rpBuilderGenericQueryNode) build(rp *ResolvedPolicy, data *rpBuilderData) error {

	reportingJobUUID := data.relativeChecksum(n.query.Mrn)

	if _, ok := rp.CollectorJob.ReportingJobs[reportingJobUUID]; !ok {
		addReportingJob(n.query.Mrn, true, reportingJobUUID, ReportingJob_UNSPECIFIED, rp)
	}

	if n.queryType == queryTypeScoring || n.queryType == queryTypeBoth {
		if _, ok := rp.CollectorJob.ReportingQueries[n.query.CodeId]; !ok {
			rp.CollectorJob.ReportingQueries[n.query.CodeId] = &StringArray{}
		}
		rp.CollectorJob.ReportingQueries[n.query.CodeId].Items = append(rp.CollectorJob.ReportingQueries[n.query.CodeId].Items, reportingJobUUID)
	}

	return nil
}

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

func (b *resolvedPolicyBuilder) collectQueryTypes(policyMrn string, acc map[string]queryType) {
	policy := b.bundleMap.Policies[policyMrn]
	if policy == nil {
		return
	}

	var accumulate func(queryMrn string, t queryType)
	accumulate = func(queryMrn string, t queryType) {
		if existing, ok := acc[queryMrn]; !ok {
			acc[queryMrn] = t
		} else {
			if existing != t {
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
			continue
		}

		for _, c := range g.Checks {
			accumulate(c.Mrn, queryTypeScoring)
		}

		for _, q := range g.Queries {
			accumulate(q.Mrn, queryTypeData)
		}

		for _, pRef := range g.Policies {
			b.collectQueryTypes(pRef.Mrn, acc)
		}
	}

	for _, r := range policy.RiskFactors {
		for _, c := range r.Checks {
			accumulate(c.Mrn, queryTypeScoring)
		}
	}
}

func (b *resolvedPolicyBuilder) gatherOverridesFromPolicy(policy *Policy) (map[string]explorer.Action, map[string]*explorer.Impact, map[string]explorer.ScoringSystem, map[string]*RiskMagnitude) {
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

			a, i, s, r := b.gatherOverridesFromPolicy(p)
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
			actions[pRef.Mrn] = action
			impacts[pRef.Mrn] = pRef.Impact
			scoringSystem := pRef.ScoringSystem

			if scoringSystem != explorer.ScoringSystem_SCORING_UNSPECIFIED {
				scoringSystems[pRef.Mrn] = pRef.ScoringSystem
			} else {
				if p, ok := b.bundleMap.Policies[pRef.Mrn]; ok {
					scoringSystems[pRef.Mrn] = p.ScoringSystem
				}
			}
		}

		getWorstImpact := func(impact1 *explorer.Impact, impact2 *explorer.Impact) *explorer.Impact {
			if impact1 == nil {
				return impact2
			}
			if impact2 == nil {
				return impact1
			}

			if impact1.Scoring == explorer.ScoringSystem_IGNORE_SCORE {
				return impact1
			}

			if impact2.Scoring == explorer.ScoringSystem_IGNORE_SCORE {
				return impact2
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
				impact = &explorer.Impact{
					Scoring: explorer.ScoringSystem_IGNORE_SCORE,
				}
			}
			if qBundle, ok := b.bundleMap.Queries[c.Mrn]; ok {
				impact = getWorstImpact(impact, qBundle.Impact)
			}
			if action != explorer.Action_UNSPECIFIED {
				actions[c.Mrn] = action
			}
			impact = getWorstImpact(impact, impacts[c.Mrn])
			if impact != nil {
				impacts[c.Mrn] = impact
			}
		}

		for _, q := range g.Queries {
			if q.Action != explorer.Action_UNSPECIFIED {
				a := normalizeAction(g.Type, q.Action, q.Impact)
				switch a {
				case explorer.Action_IGNORE, explorer.Action_OUT_OF_SCOPE, explorer.Action_DEACTIVATE:
					actions[q.Mrn] = a
				default:
					log.Warn().Str("mrn", q.Mrn).Msg("Invalid action for data query")
				}
			}
		}
	}

	for _, r := range policy.RiskFactors {
		if r.Magnitude != nil {
			riskMagnitudes[r.Mrn] = r.Magnitude
		}

		if r.Action != explorer.Action_UNSPECIFIED {
			actions[r.Mrn] = r.Action
		}
	}

	return actions, impacts, scoringSystems, riskMagnitudes
}

func canRun(action explorer.Action) bool {
	return !(action == explorer.Action_DEACTIVATE || action == explorer.Action_OUT_OF_SCOPE)
}

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

func isOverride(action explorer.Action) bool {
	return action != explorer.Action_UNSPECIFIED
}

func (b *resolvedPolicyBuilder) addPolicy(policy *Policy) bool {
	action := b.actionOverrides[policy.Mrn]

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

			if _, ok := b.addCheck(c); ok {
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

			if _, ok := b.addDataQuery(q); ok {
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
		if selectedCodeId, ok := b.addCheck(c); ok {
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

func (b *resolvedPolicyBuilder) addCheck(query *explorer.Mquery) (string, bool) {
	return b.addQuery(query)

}
func (b *resolvedPolicyBuilder) addDataQuery(query *explorer.Mquery) (string, bool) {
	return b.addQuery(query)
}

func (b *resolvedPolicyBuilder) addQuery(query *explorer.Mquery) (string, bool) {
	action := b.actionOverrides[query.Mrn]
	impact := b.impactOverrides[query.Mrn]
	queryType, queryTypeSet := b.queryTypes[query.Mrn]
	if !queryTypeSet {
		return "", false
	}

	if !canRun(action) {
		return "", false
	}

	if len(query.Variants) != 0 {
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
		b.addNode(&rpBuilderGenericQueryNode{query: query, selectedVariant: matchingVariant, selectedCodeId: selectedCodeId, queryType: queryType})

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
		b.addNode(&rpBuilderGenericQueryNode{query: query, selectedCodeId: query.CodeId, queryType: queryType})

		// Add edge from execution query to query
		b.addEdge(query.CodeId, query.Mrn, impact)

		return query.CodeId, true
	}
}

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

	actions, impacts, scoringSystems, riskMagnitudes := builder.gatherOverridesFromPolicy(policyObj)
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
		baseChecksum:    checksumStrings(resolvedPolicyExecutionChecksum, assetFiltersChecksum, "v2"),
		impactOverrides: impacts,
		propsCache:      builder.propsCache,
		compilerConf:    compilerConf,
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

	// We will build from the leaf nodes out. This means that if something is not connected
	// to a leaf node, it will not be included in the resolved policy
	leafNodes := make([]rpBuilderNode, 0, len(builder.nodes))

	for _, n := range builder.nodes {
		if n.isPrunable() {
			leafNodes = append(leafNodes, n)
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

	for _, n := range leafNodes {
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
