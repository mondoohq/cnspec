// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package bundle

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cockroachdb/errors"
	"go.mondoo.com/cnspec/v13/policy"
)

type NodeKind string

const (
	KindPolicy       NodeKind = "policy"
	KindGroup        NodeKind = "group"
	KindCheck        NodeKind = "check"
	KindQuery        NodeKind = "query"
	KindFramework    NodeKind = "framework"
	KindControl      NodeKind = "control"
	KindFrameworkMap NodeKind = "framework_map"
)

type EdgeKind string

const (
	EdgeContains  EdgeKind = "contains"
	EdgeMapsTo    EdgeKind = "maps_to"
	EdgeDependsOn EdgeKind = "depends_on"
	EdgeVariantOf EdgeKind = "variant_of"
)

type GraphNode struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	QualName string            `json:"qual_name"`
	Kind     NodeKind          `json:"kind"`
	File     string            `json:"file"`
	Line     int               `json:"line"`
	Column   int               `json:"column"`
	Title    string            `json:"title,omitempty"`
	MQL      string            `json:"mql,omitempty"`
	Impact   int               `json:"impact,omitempty"`
	Tags     map[string]string `json:"tags,omitempty"`
	ParentID string            `json:"parent_id,omitempty"`
}

type GraphEdge struct {
	Source string   `json:"source"`
	Target string   `json:"target"`
	Kind   EdgeKind `json:"kind"`
}

type PolicyGraph struct {
	Nodes []*GraphNode `json:"nodes"`
	Edges []*GraphEdge `json:"edges"`

	nodeIdx  map[string]*GraphNode
	outEdges map[string][]*GraphEdge
	inEdges  map[string][]*GraphEdge
	built    bool
}

func NewPolicyGraph() *PolicyGraph {
	return &PolicyGraph{
		nodeIdx: make(map[string]*GraphNode),
	}
}

func (g *PolicyGraph) addNode(n *GraphNode) {
	if _, exists := g.nodeIdx[n.ID]; exists {
		return
	}
	g.nodeIdx[n.ID] = n
	g.Nodes = append(g.Nodes, n)
	g.built = false
}

func (g *PolicyGraph) addEdge(e *GraphEdge) {
	g.Edges = append(g.Edges, e)
	g.built = false
}

func (g *PolicyGraph) Build() {
	g.outEdges = make(map[string][]*GraphEdge, len(g.Nodes))
	g.inEdges = make(map[string][]*GraphEdge, len(g.Nodes))
	for _, e := range g.Edges {
		g.outEdges[e.Source] = append(g.outEdges[e.Source], e)
		g.inEdges[e.Target] = append(g.inEdges[e.Target], e)
	}
	g.built = true
}

func (g *PolicyGraph) ensureBuilt() {
	if !g.built {
		g.Build()
	}
}

func (g *PolicyGraph) Node(id string) *GraphNode {
	return g.nodeIdx[id]
}

func (g *PolicyGraph) FindNode(query string) []*GraphNode {
	query = strings.ToLower(query)
	var result []*GraphNode
	for _, n := range g.Nodes {
		if strings.Contains(strings.ToLower(n.Name), query) ||
			strings.Contains(strings.ToLower(n.QualName), query) ||
			strings.Contains(strings.ToLower(n.ID), query) {
			result = append(result, n)
		}
	}
	return result
}

func (g *PolicyGraph) InEdges(nodeID string, kinds ...EdgeKind) []*GraphEdge {
	g.ensureBuilt()
	if len(kinds) == 0 {
		return g.inEdges[nodeID]
	}
	kindSet := make(map[EdgeKind]bool, len(kinds))
	for _, k := range kinds {
		kindSet[k] = true
	}
	var result []*GraphEdge
	for _, e := range g.inEdges[nodeID] {
		if kindSet[e.Kind] {
			result = append(result, e)
		}
	}
	return result
}

func (g *PolicyGraph) OutEdges(nodeID string, kinds ...EdgeKind) []*GraphEdge {
	g.ensureBuilt()
	if len(kinds) == 0 {
		return g.outEdges[nodeID]
	}
	kindSet := make(map[EdgeKind]bool, len(kinds))
	for _, k := range kinds {
		kindSet[k] = true
	}
	var result []*GraphEdge
	for _, e := range g.outEdges[nodeID] {
		if kindSet[e.Kind] {
			result = append(result, e)
		}
	}
	return result
}

func (g *PolicyGraph) Reachable(nodeID string) []string {
	g.ensureBuilt()
	visited := map[string]bool{nodeID: true}
	queue := []string{nodeID}
	var result []string
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, e := range g.outEdges[current] {
			if !visited[e.Target] {
				visited[e.Target] = true
				result = append(result, e.Target)
				queue = append(queue, e.Target)
			}
		}
	}
	return result
}

func (g *PolicyGraph) FindPaths(srcID, dstID string, maxDepth int) [][]string {
	g.ensureBuilt()
	if maxDepth <= 0 {
		maxDepth = 20
	}
	var results [][]string
	g.findPathsDFS(srcID, dstID, []string{srcID}, map[string]bool{srcID: true}, maxDepth, &results)
	return results
}

func (g *PolicyGraph) findPathsDFS(current, dst string, path []string, visited map[string]bool, maxDepth int, results *[][]string) {
	if len(path) > maxDepth {
		return
	}
	if current == dst {
		cp := make([]string, len(path))
		copy(cp, path)
		*results = append(*results, cp)
		return
	}
	for _, e := range g.outEdges[current] {
		if !visited[e.Target] {
			visited[e.Target] = true
			g.findPathsDFS(e.Target, dst, append(path, e.Target), visited, maxDepth, results)
			delete(visited, e.Target)
		}
	}
}

func (g *PolicyGraph) Neighborhood(nodeID string, depth int) []*GraphNode {
	g.ensureBuilt()
	if depth <= 0 {
		depth = 2
	}
	visited := map[string]bool{nodeID: true}
	type entry struct {
		id    string
		level int
	}
	queue := []entry{{nodeID, 0}}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if current.level >= depth {
			continue
		}
		for _, e := range g.outEdges[current.id] {
			if !visited[e.Target] && g.nodeIdx[e.Target] != nil {
				visited[e.Target] = true
				queue = append(queue, entry{e.Target, current.level + 1})
			}
		}
		for _, e := range g.inEdges[current.id] {
			if !visited[e.Source] && g.nodeIdx[e.Source] != nil {
				visited[e.Source] = true
				queue = append(queue, entry{e.Source, current.level + 1})
			}
		}
	}
	var result []*GraphNode
	for id := range visited {
		if n := g.nodeIdx[id]; n != nil {
			result = append(result, n)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].File != result[j].File {
			return result[i].File < result[j].File
		}
		return result[i].Line < result[j].Line
	})
	return result
}

func (g *PolicyGraph) resolveEdges() {
	nameIdx := make(map[string][]string)
	qualIdx := make(map[string][]string)
	for _, n := range g.Nodes {
		nameIdx[n.Name] = append(nameIdx[n.Name], n.ID)
		qualIdx[n.QualName] = append(qualIdx[n.QualName], n.ID)
	}
	resolve := func(ref string) string {
		if !strings.HasPrefix(ref, "?:") {
			return ref
		}
		unresolved := strings.TrimPrefix(ref, "?:")
		if ids, ok := qualIdx[unresolved]; ok && len(ids) == 1 {
			return ids[0]
		}
		if ids, ok := nameIdx[unresolved]; ok && len(ids) == 1 {
			return ids[0]
		}
		return ref
	}
	for _, e := range g.Edges {
		e.Source = resolve(e.Source)
		e.Target = resolve(e.Target)
	}
	g.built = false
}

func nodeID(file string, kind NodeKind, uid string) string {
	return file + "::" + string(kind) + ":" + uid
}

func qualName(kind NodeKind, uid string) string {
	return string(kind) + ":" + uid
}

// BuildGraphFromPaths walks a path for .mql.yaml files, parses them, and builds a graph.
func BuildGraphFromPaths(paths ...string) (*PolicyGraph, error) {
	files, err := policy.WalkPolicyBundleFiles(paths...)
	if err != nil {
		return nil, err
	}

	bundles := make(map[string]*Bundle, len(files))
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			return nil, errors.Wrap(err, "failed to read "+f)
		}
		b, err := ParseYaml(data)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse "+f)
		}
		rel, err := filepath.Rel(".", f)
		if err != nil {
			rel = f
		}
		bundles[rel] = b
	}
	return BuildGraph(bundles), nil
}

// BuildGraph builds a PolicyGraph from pre-parsed bundles keyed by filename.
func BuildGraph(bundles map[string]*Bundle) *PolicyGraph {
	g := NewPolicyGraph()
	files := make([]string, 0, len(bundles))
	for f := range bundles {
		files = append(files, f)
	}
	sort.Strings(files)
	for _, file := range files {
		extractBundle(g, file, bundles[file])
	}
	g.resolveEdges()
	g.Build()
	return g
}

func extractBundle(g *PolicyGraph, file string, b *Bundle) {
	for _, q := range b.Queries {
		extractTopLevelQuery(g, file, q)
	}
	for _, p := range b.Policies {
		extractPolicy(g, file, p)
	}
	for _, fw := range b.Frameworks {
		extractFramework(g, file, fw)
	}
	for _, fm := range b.FrameworkMaps {
		extractFrameworkMap(g, file, fm)
	}
}

func extractTopLevelQuery(g *PolicyGraph, file string, q *Mquery) {
	if q.Uid == "" {
		return
	}
	kind := KindCheck
	if q.Mql == "" && len(q.Variants) == 0 {
		kind = KindQuery
	}
	impact := 0
	if q.Impact != nil && q.Impact.Value != nil {
		impact = int(q.Impact.Value.Value)
	}
	id := nodeID(file, kind, q.Uid)
	title := q.Title
	if title == "" && q.Docs != nil {
		title = q.Docs.Desc
	}
	g.addNode(&GraphNode{
		ID:       id,
		Name:     q.Uid,
		QualName: qualName(kind, q.Uid),
		Kind:     kind,
		File:     file,
		Line:     q.FileContext.Line,
		Column:   q.FileContext.Column,
		Title:    title,
		MQL:      q.Mql,
		Impact:   impact,
		Tags:     q.Tags,
	})
	for _, v := range q.Variants {
		if v.Uid == "" {
			continue
		}
		g.addEdge(&GraphEdge{
			Source: id,
			Target: "?:" + qualName(KindCheck, v.Uid),
			Kind:   EdgeVariantOf,
		})
	}
}

func extractPolicy(g *PolicyGraph, file string, p *Policy) {
	if p.Uid == "" {
		return
	}
	pID := nodeID(file, KindPolicy, p.Uid)
	g.addNode(&GraphNode{
		ID:       pID,
		Name:     p.Uid,
		QualName: qualName(KindPolicy, p.Uid),
		Kind:     KindPolicy,
		File:     file,
		Line:     p.FileContext.Line,
		Column:   p.FileContext.Column,
		Title:    p.Name,
	})
	for i, grp := range p.Groups {
		extractPolicyGroup(g, file, grp, pID, p.Uid, i)
	}
}

func extractPolicyGroup(g *PolicyGraph, file string, grp *PolicyGroup, policyID string, policyUID string, idx int) {
	groupUID := grp.Uid
	if groupUID == "" {
		groupUID = fmt.Sprintf("%s-group-%d", policyUID, idx)
	}
	gID := nodeID(file, KindGroup, groupUID)
	title := grp.Title
	g.addNode(&GraphNode{
		ID:       gID,
		Name:     groupUID,
		QualName: qualName(KindGroup, groupUID),
		Kind:     KindGroup,
		File:     file,
		Line:     grp.FileContext.Line,
		Column:   grp.FileContext.Column,
		Title:    title,
		ParentID: policyID,
	})
	g.addEdge(&GraphEdge{Source: policyID, Target: gID, Kind: EdgeContains})

	for _, c := range grp.Checks {
		if c.Uid == "" {
			continue
		}
		if c.Mql != "" || c.Title != "" {
			extractInlineQuery(g, file, c, KindCheck, gID)
		}
		g.addEdge(&GraphEdge{
			Source: gID,
			Target: "?:" + qualName(KindCheck, c.Uid),
			Kind:   EdgeContains,
		})
	}
	for _, q := range grp.Queries {
		if q.Uid == "" {
			continue
		}
		if q.Mql != "" || q.Title != "" {
			extractInlineQuery(g, file, q, KindQuery, gID)
		}
		g.addEdge(&GraphEdge{
			Source: gID,
			Target: "?:" + qualName(KindQuery, q.Uid),
			Kind:   EdgeContains,
		})
	}
	for _, pr := range grp.Policies {
		if pr.Uid == "" {
			continue
		}
		g.addEdge(&GraphEdge{
			Source: gID,
			Target: "?:" + qualName(KindPolicy, pr.Uid),
			Kind:   EdgeDependsOn,
		})
	}
}

func extractInlineQuery(g *PolicyGraph, file string, q *Mquery, kind NodeKind, parentID string) {
	if q.Uid == "" {
		return
	}
	impact := 0
	if q.Impact != nil && q.Impact.Value != nil {
		impact = int(q.Impact.Value.Value)
	}
	title := q.Title
	if title == "" && q.Docs != nil {
		title = q.Docs.Desc
	}
	id := nodeID(file, kind, q.Uid)
	g.addNode(&GraphNode{
		ID:       id,
		Name:     q.Uid,
		QualName: qualName(kind, q.Uid),
		Kind:     kind,
		File:     file,
		Line:     q.FileContext.Line,
		Column:   q.FileContext.Column,
		Title:    title,
		MQL:      q.Mql,
		Impact:   impact,
		Tags:     q.Tags,
		ParentID: parentID,
	})
	for _, v := range q.Variants {
		if v.Uid == "" {
			continue
		}
		g.addEdge(&GraphEdge{
			Source: id,
			Target: "?:" + qualName(KindCheck, v.Uid),
			Kind:   EdgeVariantOf,
		})
	}
}

func extractFramework(g *PolicyGraph, file string, fw *Framework) {
	if fw.Uid == "" {
		return
	}
	fwID := nodeID(file, KindFramework, fw.Uid)
	g.addNode(&GraphNode{
		ID:       fwID,
		Name:     fw.Uid,
		QualName: qualName(KindFramework, fw.Uid),
		Kind:     KindFramework,
		File:     file,
		Line:     fw.FileContext.Line,
		Column:   fw.FileContext.Column,
		Title:    fw.Name,
	})
	for _, dep := range fw.Dependencies {
		if dep.Uid == "" {
			continue
		}
		g.addEdge(&GraphEdge{
			Source: fwID,
			Target: "?:" + qualName(KindFramework, dep.Uid),
			Kind:   EdgeDependsOn,
		})
	}
	for _, grp := range fw.Groups {
		for _, ctrl := range grp.Controls {
			extractControl(g, file, ctrl, fwID)
		}
	}
	for _, fm := range fw.FrameworkMaps {
		extractFrameworkMap(g, file, fm)
	}
}

func extractControl(g *PolicyGraph, file string, ctrl *Control, frameworkID string) {
	if ctrl.Uid == "" {
		return
	}
	cID := nodeID(file, KindControl, ctrl.Uid)
	title := ctrl.Title
	if title == "" && ctrl.Docs != nil {
		title = ctrl.Docs.Desc
	}
	g.addNode(&GraphNode{
		ID:       cID,
		Name:     ctrl.Uid,
		QualName: qualName(KindControl, ctrl.Uid),
		Kind:     KindControl,
		File:     file,
		Line:     ctrl.FileContext.Line,
		Column:   ctrl.FileContext.Column,
		Title:    title,
		ParentID: frameworkID,
		Tags:     ctrl.Tags,
	})
	g.addEdge(&GraphEdge{Source: frameworkID, Target: cID, Kind: EdgeContains})
}

func extractFrameworkMap(g *PolicyGraph, file string, fm *FrameworkMap) {
	if fm.Uid == "" && fm.FrameworkOwner == nil {
		return
	}
	fmUID := fm.Uid
	if fmUID == "" && fm.FrameworkOwner != nil {
		fmUID = "fmap-" + fm.FrameworkOwner.Uid
	}
	fmID := nodeID(file, KindFrameworkMap, fmUID)
	g.addNode(&GraphNode{
		ID:       fmID,
		Name:     fmUID,
		QualName: qualName(KindFrameworkMap, fmUID),
		Kind:     KindFrameworkMap,
		File:     file,
		Line:     fm.FileContext.Line,
		Column:   fm.FileContext.Column,
	})
	if fm.FrameworkOwner != nil && fm.FrameworkOwner.Uid != "" {
		g.addEdge(&GraphEdge{
			Source: fmID,
			Target: "?:" + qualName(KindFramework, fm.FrameworkOwner.Uid),
			Kind:   EdgeDependsOn,
		})
	}
	for _, pd := range fm.PolicyDependencies {
		if pd.Uid == "" {
			continue
		}
		g.addEdge(&GraphEdge{
			Source: fmID,
			Target: "?:" + qualName(KindPolicy, pd.Uid),
			Kind:   EdgeDependsOn,
		})
	}
	for _, cm := range fm.Controls {
		extractControlMapping(g, file, cm, fmID)
	}
}

func extractControlMapping(g *PolicyGraph, _ string, cm *ControlMap, fmapID string) {
	if cm.Uid == "" {
		return
	}
	controlTarget := "?:" + qualName(KindControl, cm.Uid)
	g.addEdge(&GraphEdge{Source: fmapID, Target: controlTarget, Kind: EdgeContains})
	for _, ref := range cm.Checks {
		if ref.Uid == "" {
			continue
		}
		g.addEdge(&GraphEdge{
			Source: controlTarget,
			Target: "?:" + qualName(KindCheck, ref.Uid),
			Kind:   EdgeMapsTo,
		})
	}
	for _, ref := range cm.Queries {
		if ref.Uid == "" {
			continue
		}
		g.addEdge(&GraphEdge{
			Source: controlTarget,
			Target: "?:" + qualName(KindQuery, ref.Uid),
			Kind:   EdgeMapsTo,
		})
	}
	for _, ref := range cm.Policies {
		if ref.Uid == "" {
			continue
		}
		g.addEdge(&GraphEdge{
			Source: controlTarget,
			Target: "?:" + qualName(KindPolicy, ref.Uid),
			Kind:   EdgeMapsTo,
		})
	}
	for _, ref := range cm.Controls {
		if ref.Uid == "" {
			continue
		}
		g.addEdge(&GraphEdge{
			Source: controlTarget,
			Target: "?:" + qualName(KindControl, ref.Uid),
			Kind:   EdgeMapsTo,
		})
	}
}
