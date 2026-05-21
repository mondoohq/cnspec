// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"go.mondoo.com/cnspec/v13/internal/bundle"
)

func init() {
	policyCmd.AddCommand(policyGraphCmd)

	policyGraphCallersCmd.Flags().Bool("json", false, "Output as JSON")
	policyGraphCmd.AddCommand(policyGraphCallersCmd)

	policyGraphCalleesCmd.Flags().Bool("json", false, "Output as JSON")
	policyGraphCmd.AddCommand(policyGraphCalleesCmd)

	policyGraphContextCmd.Flags().Int("depth", 2, "Neighborhood depth (hops)")
	policyGraphCmd.AddCommand(policyGraphContextCmd)

	policyGraphPathsCmd.Flags().Bool("json", false, "Output as JSON")
	policyGraphCmd.AddCommand(policyGraphPathsCmd)

	policyGraphReachableCmd.Flags().Bool("json", false, "Output as JSON")
	policyGraphCmd.AddCommand(policyGraphReachableCmd)

	policyGraphExportCmd.Flags().String("format", "json", "Output format: json, dot")
	policyGraphCmd.AddCommand(policyGraphExportCmd)

	policyGraphSearchCmd.Flags().String("kind", "", "Filter by node kind (policy, check, group, query, framework, control)")
	policyGraphSearchCmd.Flags().String("tag", "", "Filter by tag key")
	policyGraphSearchCmd.Flags().Int("impact", 0, "Minimum impact score")
	policyGraphSearchCmd.Flags().Int("limit", 50, "Maximum results")
	policyGraphSearchCmd.Flags().Bool("json", false, "Output as JSON")
	policyGraphCmd.AddCommand(policyGraphSearchCmd)
}

var policyGraphCmd = &cobra.Command{
	Use:   "graph",
	Short: "Navigate policy bundle structure via graph commands",
	Long:  "Build and query a graph of policies, checks, frameworks, and controls from .mql.yaml bundle files.",
}

var policyGraphCallersCmd = &cobra.Command{
	Use:   "callers <uid> <path>",
	Short: "Show what references a node (inbound edges)",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		uid, paths := args[0], args[1:]
		g := mustBuildGraph(paths)
		node := mustFindNode(g, uid)

		edges := g.InEdges(node.ID)
		if jsonOut, _ := cmd.Flags().GetBool("json"); jsonOut {
			printJSON(edgesWithNodes(g, edges, true))
			return
		}
		fmt.Printf("%s is referenced by:\n", node.QualName)
		for _, e := range edges {
			name := e.Source
			loc := ""
			if n := g.Node(e.Source); n != nil {
				name = n.QualName
				loc = fmt.Sprintf(" (%s:%d)", n.File, n.Line)
			}
			fmt.Printf("  [%s] %s%s\n", e.Kind, name, loc)
		}
	},
}

var policyGraphCalleesCmd = &cobra.Command{
	Use:   "callees <uid> <path>",
	Short: "Show what a node contains or references (outbound edges)",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		uid, paths := args[0], args[1:]
		g := mustBuildGraph(paths)
		node := mustFindNode(g, uid)

		edges := g.OutEdges(node.ID)
		if jsonOut, _ := cmd.Flags().GetBool("json"); jsonOut {
			printJSON(edgesWithNodes(g, edges, false))
			return
		}
		fmt.Printf("%s contains/references:\n", node.QualName)
		for _, e := range edges {
			name := e.Target
			loc := ""
			if n := g.Node(e.Target); n != nil {
				name = n.QualName
				loc = fmt.Sprintf(" (%s:%d)", n.File, n.Line)
			}
			fmt.Printf("  [%s] %s%s\n", e.Kind, name, loc)
		}
	},
}

var policyGraphContextCmd = &cobra.Command{
	Use:   "context <uid> <path>",
	Short: "Show LLM-friendly context with YAML snippets",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		uid, paths := args[0], args[1:]
		depth, _ := cmd.Flags().GetInt("depth")
		g := mustBuildGraph(paths)
		node := mustFindNode(g, uid)

		if err := g.WriteContext(os.Stdout, node.ID, depth, ""); err != nil {
			log.Fatal().Err(err).Msg("failed to write context")
		}
	},
}

var policyGraphPathsCmd = &cobra.Command{
	Use:   "paths <from-uid> <to-uid> <path>",
	Short: "Find paths between two nodes",
	Args:  cobra.MinimumNArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		fromUID, toUID, paths := args[0], args[1], args[2:]
		g := mustBuildGraph(paths)
		fromNode := mustFindNode(g, fromUID)
		toNode := mustFindNode(g, toUID)

		results := g.FindPaths(fromNode.ID, toNode.ID, 20)
		if jsonOut, _ := cmd.Flags().GetBool("json"); jsonOut {
			printJSON(results)
			return
		}
		if len(results) == 0 {
			fmt.Printf("No paths found from %s to %s\n", fromNode.QualName, toNode.QualName)
			return
		}
		for i, path := range results {
			fmt.Printf("Path %d:\n", i+1)
			for j, nodeID := range path {
				name := nodeID
				if n := g.Node(nodeID); n != nil {
					name = n.QualName
				}
				if j > 0 {
					fmt.Print("  → ")
				} else {
					fmt.Print("  ")
				}
				fmt.Println(name)
			}
		}
	},
}

var policyGraphReachableCmd = &cobra.Command{
	Use:   "reachable <uid> <path>",
	Short: "Show all nodes reachable from a node",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		uid, paths := args[0], args[1:]
		g := mustBuildGraph(paths)
		node := mustFindNode(g, uid)

		reachable := g.Reachable(node.ID)
		if jsonOut, _ := cmd.Flags().GetBool("json"); jsonOut {
			var nodes []*bundle.GraphNode
			for _, id := range reachable {
				if n := g.Node(id); n != nil {
					nodes = append(nodes, n)
				}
			}
			printJSON(nodes)
			return
		}
		fmt.Printf("%d nodes reachable from %s:\n", len(reachable), node.QualName)
		for _, id := range reachable {
			name := id
			loc := ""
			if n := g.Node(id); n != nil {
				name = n.QualName
				loc = fmt.Sprintf(" (%s:%d)", n.File, n.Line)
			}
			fmt.Printf("  %s%s\n", name, loc)
		}
	},
}

var policyGraphExportCmd = &cobra.Command{
	Use:   "export <path>",
	Short: "Export the full policy graph",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		format, _ := cmd.Flags().GetString("format")
		g := mustBuildGraph(args)

		switch format {
		case "json":
			printJSON(g)
		case "dot":
			writeDot(g)
		default:
			log.Fatal().Str("format", format).Msg("unknown format (use json or dot)")
		}
	},
}

var policyGraphSearchCmd = &cobra.Command{
	Use:   "search <query> <path>",
	Short: "Search for nodes by name, title, or UID",
	Long:  "Find policy graph nodes using multi-strategy search: exact name, prefix, or substring match across names, qualified names, and titles.",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		query, paths := args[0], args[1:]
		g := mustBuildGraph(paths)
		idx := g.BuildNodeIndex()

		kind, _ := cmd.Flags().GetString("kind")
		tag, _ := cmd.Flags().GetString("tag")
		impact, _ := cmd.Flags().GetInt("impact")
		limit, _ := cmd.Flags().GetInt("limit")

		results := idx.Search(query, bundle.SearchOpts{
			Kind:      bundle.NodeKind(kind),
			TagKey:    tag,
			MinImpact: impact,
			Limit:     limit,
		})

		if jsonOut, _ := cmd.Flags().GetBool("json"); jsonOut {
			printJSON(results)
			return
		}

		if len(results) == 0 {
			fmt.Printf("No nodes found matching %q\n", query)
			return
		}

		for _, n := range results {
			title := truncateRunes(n.Title, 40)
			qualName := truncateRunes(n.QualName, 50)
			fmt.Printf("%-12s %-50s %-40s (%s:%d)\n", n.Kind, qualName, title, n.File, n.Line)
		}
	},
}

func mustBuildGraph(paths []string) *bundle.PolicyGraph {
	g, err := bundle.BuildGraphFromPaths(paths...)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to build graph")
	}
	return g
}

func mustFindNode(g *bundle.PolicyGraph, uid string) *bundle.GraphNode {
	nodes := g.FindNode(uid)
	if len(nodes) == 0 {
		log.Fatal().Str("uid", uid).Msg("no node found matching query")
	}
	if len(nodes) == 1 {
		return nodes[0]
	}
	// Prefer exact name match
	for _, n := range nodes {
		if n.Name == uid {
			return n
		}
	}
	// Prefer exact qual_name match
	for _, n := range nodes {
		if n.QualName == uid {
			return n
		}
	}
	// Ambiguous — show options
	fmt.Fprintf(os.Stderr, "Ambiguous query %q matched %d nodes:\n", uid, len(nodes))
	for _, n := range nodes {
		fmt.Fprintf(os.Stderr, "  %s (%s) at %s:%d\n", n.QualName, n.Kind, n.File, n.Line)
	}
	os.Exit(1)
	return nil
}

func printJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		log.Fatal().Err(err).Msg("failed to encode JSON")
	}
}

type edgeResult struct {
	Edge *bundle.GraphEdge `json:"edge"`
	Node *bundle.GraphNode `json:"node,omitempty"`
}

func edgesWithNodes(g *bundle.PolicyGraph, edges []*bundle.GraphEdge, useSource bool) []edgeResult {
	var results []edgeResult
	for _, e := range edges {
		id := e.Target
		if useSource {
			id = e.Source
		}
		results = append(results, edgeResult{
			Edge: e,
			Node: g.Node(id),
		})
	}
	return results
}

func truncateRunes(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-3]) + "..."
}

func writeDot(g *bundle.PolicyGraph) {
	fmt.Println("digraph policy_graph {")
	fmt.Println("  rankdir=LR;")
	fmt.Println("  node [shape=box, fontname=monospace];")
	for _, n := range g.Nodes {
		color := "white"
		switch n.Kind {
		case bundle.KindPolicy:
			color = "lightblue"
		case bundle.KindCheck:
			color = "lightyellow"
		case bundle.KindFramework:
			color = "lightgreen"
		case bundle.KindControl:
			color = "palegreen"
		case bundle.KindGroup:
			color = "lavender"
		}
		label := strings.ReplaceAll(n.QualName, `"`, `\"`)
		fmt.Printf("  %q [label=%q, style=filled, fillcolor=%q];\n", n.ID, label, color)
	}
	for _, e := range g.Edges {
		if strings.HasPrefix(e.Source, "?:") || strings.HasPrefix(e.Target, "?:") {
			continue
		}
		style := "solid"
		if e.Kind == bundle.EdgeMapsTo {
			style = "dashed"
		}
		fmt.Printf("  %q -> %q [label=%q, style=%s];\n", e.Source, e.Target, e.Kind, style)
	}
	fmt.Println("}")
}
