// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package bundle

import (
	"sort"
	"strings"
)

// NodeIndex provides fast node lookups by name, qualified name, kind, and file.
// It uses a search cascade: exact name → exact qualName → prefix → substring.
type NodeIndex struct {
	byName     map[string][]*GraphNode
	byQualName map[string][]*GraphNode
	byTitle    map[string][]*GraphNode
	byKind     map[NodeKind][]*GraphNode
	byFile     map[string][]*GraphNode
	sorted     []string // sorted lowercase names for prefix binary search
}

// BuildNodeIndex creates a NodeIndex from a PolicyGraph's nodes.
func BuildNodeIndex(g *PolicyGraph) *NodeIndex {
	idx := &NodeIndex{
		byName:     make(map[string][]*GraphNode),
		byQualName: make(map[string][]*GraphNode),
		byTitle:    make(map[string][]*GraphNode),
		byKind:     make(map[NodeKind][]*GraphNode),
		byFile:     make(map[string][]*GraphNode),
	}

	nameSet := make(map[string]bool)
	for _, n := range g.Nodes {
		lower := strings.ToLower(n.Name)
		idx.byName[lower] = append(idx.byName[lower], n)
		nameSet[lower] = true

		lowerQual := strings.ToLower(n.QualName)
		idx.byQualName[lowerQual] = append(idx.byQualName[lowerQual], n)

		if n.Title != "" {
			lowerTitle := strings.ToLower(n.Title)
			idx.byTitle[lowerTitle] = append(idx.byTitle[lowerTitle], n)
		}

		idx.byKind[n.Kind] = append(idx.byKind[n.Kind], n)
		idx.byFile[n.File] = append(idx.byFile[n.File], n)
	}

	idx.sorted = make([]string, 0, len(nameSet))
	for name := range nameSet {
		idx.sorted = append(idx.sorted, name)
	}
	sort.Strings(idx.sorted)
	return idx
}

// ExactName returns nodes whose Name exactly matches (case-insensitive).
func (idx *NodeIndex) ExactName(name string) []*GraphNode {
	return idx.byName[strings.ToLower(name)]
}

// ExactQualName returns nodes whose QualName exactly matches (case-insensitive).
func (idx *NodeIndex) ExactQualName(qualName string) []*GraphNode {
	return idx.byQualName[strings.ToLower(qualName)]
}

// Prefix returns nodes whose Name starts with the given prefix (case-insensitive).
// Uses binary search on a sorted name slice for O(log n) lookup.
func (idx *NodeIndex) Prefix(prefix string, limit int) []*GraphNode {
	lower := strings.ToLower(prefix)
	i := sort.SearchStrings(idx.sorted, lower)

	var result []*GraphNode
	for ; i < len(idx.sorted) && strings.HasPrefix(idx.sorted[i], lower); i++ {
		nodes := idx.byName[idx.sorted[i]]
		result = append(result, nodes...)
		if limit > 0 && len(result) >= limit {
			return result[:limit]
		}
	}
	return result
}

// Substring returns nodes whose Name, QualName, or Title contains the query (case-insensitive).
func (idx *NodeIndex) Substring(query string, limit int) []*GraphNode {
	lower := strings.ToLower(query)
	seen := make(map[string]bool)
	var result []*GraphNode

	for name, nodes := range idx.byName {
		if strings.Contains(name, lower) {
			for _, n := range nodes {
				if !seen[n.ID] {
					seen[n.ID] = true
					result = append(result, n)
					if limit > 0 && len(result) >= limit {
						return result
					}
				}
			}
		}
	}

	for qualName, nodes := range idx.byQualName {
		if strings.Contains(qualName, lower) {
			for _, n := range nodes {
				if !seen[n.ID] {
					seen[n.ID] = true
					result = append(result, n)
					if limit > 0 && len(result) >= limit {
						return result
					}
				}
			}
		}
	}

	for title, nodes := range idx.byTitle {
		if strings.Contains(title, lower) {
			for _, n := range nodes {
				if !seen[n.ID] {
					seen[n.ID] = true
					result = append(result, n)
					if limit > 0 && len(result) >= limit {
						return result
					}
				}
			}
		}
	}

	return result
}

// ByKind returns all nodes of a specific kind.
func (idx *NodeIndex) ByKind(kind NodeKind) []*GraphNode {
	return idx.byKind[kind]
}

// SearchOpts controls filtering for the Search method.
type SearchOpts struct {
	Kind      NodeKind
	TagKey    string
	MinImpact int
	Limit     int
}

// Search finds nodes matching a query using a cascade: exact name → exact qualName → prefix → substring.
// Results are filtered by SearchOpts and capped at Limit (default 50).
func (idx *NodeIndex) Search(query string, opts SearchOpts) []*GraphNode {
	if opts.Limit <= 0 {
		opts.Limit = 50
	}

	var candidates []*GraphNode

	if query == "" {
		// Empty query with kind filter: return all nodes of that kind.
		if opts.Kind != "" {
			candidates = idx.ByKind(opts.Kind)
		}
	} else {
		candidates = idx.ExactName(query)
		if len(candidates) == 0 {
			candidates = idx.ExactQualName(query)
		}
		if len(candidates) == 0 {
			candidates = idx.Prefix(query, 0)
		}
		if len(candidates) == 0 {
			candidates = idx.Substring(query, 0)
		}
	}

	if opts.Kind != "" || opts.TagKey != "" || opts.MinImpact > 0 {
		filtered := make([]*GraphNode, 0, len(candidates))
		for _, n := range candidates {
			if opts.Kind != "" && n.Kind != opts.Kind {
				continue
			}
			if opts.TagKey != "" {
				if _, ok := n.Tags[opts.TagKey]; !ok {
					continue
				}
			}
			if opts.MinImpact > 0 && n.Impact < opts.MinImpact {
				continue
			}
			filtered = append(filtered, n)
		}
		candidates = filtered
	}

	if len(candidates) > opts.Limit {
		candidates = candidates[:opts.Limit]
	}
	return candidates
}
