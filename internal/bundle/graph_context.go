// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package bundle

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cockroachdb/errors"
)

// WriteContext writes a markdown-formatted context document for an LLM.
func (g *PolicyGraph) WriteContext(w io.Writer, nodeID string, depth int, rootDir string) error {
	g.ensureBuilt()

	center := g.nodeIdx[nodeID]
	if center == nil {
		return errors.New("node " + nodeID + " not found")
	}

	nodes := g.Neighborhood(nodeID, depth)

	fmt.Fprintf(w, "# Policy context for %s\n\n", center.QualName)
	fmt.Fprintf(w, "**Focus**: `%s` (%s) at %s:%d\n", center.QualName, center.Kind, center.File, center.Line)
	if center.Title != "" {
		fmt.Fprintf(w, "**Title**: %s\n", center.Title)
	}
	if center.Impact > 0 {
		fmt.Fprintf(w, "**Impact**: %d\n", center.Impact)
	}
	if len(center.Tags) > 0 {
		var tags []string
		for k, v := range center.Tags {
			tags = append(tags, k+"="+v)
		}
		sort.Strings(tags)
		fmt.Fprintf(w, "**Tags**: %s\n", strings.Join(tags, ", "))
	}
	fmt.Fprintf(w, "**Neighborhood**: %d nodes within %d hops\n\n", len(nodes), depth)

	fileGroups := make(map[string][]*GraphNode)
	for _, n := range nodes {
		fileGroups[n.File] = append(fileGroups[n.File], n)
	}

	var files []string
	for f := range fileGroups {
		files = append(files, f)
	}
	sort.Strings(files)

	fileCache := make(map[string][]string)

	for _, file := range files {
		fmt.Fprintf(w, "## %s\n\n", file)

		for _, n := range fileGroups[file] {
			role := ""
			if n.ID == nodeID {
				role = " ← FOCUS"
			}
			fmt.Fprintf(w, "### %s (%s, L%d)%s\n", n.QualName, n.Kind, n.Line, role)
			if n.Title != "" {
				fmt.Fprintf(w, "**Title**: %s\n", n.Title)
			}
			if n.Impact > 0 {
				fmt.Fprintf(w, "**Impact**: %d\n", n.Impact)
			}

			inNames := g.inEdgeNames(n.ID)
			outNames := g.outEdgeNames(n.ID)
			if len(inNames) > 0 {
				fmt.Fprintf(w, "Referenced by: %s\n", strings.Join(inNames, ", "))
			}
			if len(outNames) > 0 {
				fmt.Fprintf(w, "Contains: %s\n", strings.Join(outNames, ", "))
			}
			fmt.Fprintln(w)

			lines, ok := fileCache[file]
			if !ok {
				lines = readSourceLines(rootDir, file)
				fileCache[file] = lines
			}
			if lines != nil && n.Line > 0 {
				endLine := estimateEndLine(nodes, n, len(lines))
				snippet := extractSnippet(lines, n.Line, endLine)
				if snippet != "" {
					fmt.Fprintf(w, "```yaml\n%s\n```\n\n", snippet)
				}
			}
		}
	}

	return nil
}

func (g *PolicyGraph) inEdgeNames(nodeID string) []string {
	var names []string
	seen := map[string]bool{}
	for _, e := range g.inEdges[nodeID] {
		name := e.Source
		if n := g.nodeIdx[e.Source]; n != nil {
			name = n.QualName
		}
		label := fmt.Sprintf("%s [%s]", name, e.Kind)
		if !seen[label] {
			seen[label] = true
			names = append(names, label)
		}
	}
	return names
}

func (g *PolicyGraph) outEdgeNames(nodeID string) []string {
	var names []string
	seen := map[string]bool{}
	for _, e := range g.outEdges[nodeID] {
		name := e.Target
		if n := g.nodeIdx[e.Target]; n != nil {
			name = n.QualName
		}
		label := fmt.Sprintf("%s [%s]", name, e.Kind)
		if !seen[label] {
			seen[label] = true
			names = append(names, label)
		}
	}
	return names
}

func readSourceLines(rootDir, relPath string) []string {
	path := relPath
	if rootDir != "" {
		path = filepath.Join(rootDir, relPath)
	}
	path = filepath.Clean(path)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return strings.Split(string(data), "\n")
}

func estimateEndLine(siblings []*GraphNode, current *GraphNode, totalLines int) int {
	nextLine := totalLines
	for _, n := range siblings {
		if n.File == current.File && n.Line > current.Line && n.Line < nextLine {
			nextLine = n.Line - 1
		}
	}
	maxLines := 40
	if current.Line+maxLines < nextLine {
		nextLine = current.Line + maxLines
	}
	return nextLine
}

func extractSnippet(lines []string, startLine, endLine int) string {
	if startLine < 1 {
		startLine = 1
	}
	if endLine > len(lines) {
		endLine = len(lines)
	}
	if startLine > len(lines) {
		return ""
	}
	return strings.Join(lines[startLine-1:endLine], "\n")
}
