// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package aibom

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/charmbracelet/lipgloss"
)

var (
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))
	dimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	statStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
)

type TextListFormatter struct{}

func (f *TextListFormatter) Render(w io.Writer, bom *AiBom) error {
	fmt.Fprintf(w, "\n%s\n", titleStyle.Render("AIBOM for "+bom.Asset.GetName()))

	// Collect totals across all agents
	type skillEntry struct {
		Agent string
		Skill *AgentSkill
	}
	type mcpEntry struct {
		Agent string
		Mcp   *McpServer
	}
	type pluginEntry struct {
		Agent  string
		Plugin *AgentPlugin
	}
	type extEntry struct {
		Agent string
		Ext   *AgentExtension
	}

	var allSkills []skillEntry
	var allMcp []mcpEntry
	var allPlugins []pluginEntry
	var allExtensions []extEntry

	for _, a := range bom.Agents {
		for _, s := range a.Skills {
			allSkills = append(allSkills, skillEntry{Agent: a.Name, Skill: s})
		}
		for _, m := range a.McpServers {
			allMcp = append(allMcp, mcpEntry{Agent: a.Name, Mcp: m})
		}
		for _, p := range a.Plugins {
			allPlugins = append(allPlugins, pluginEntry{Agent: a.Name, Plugin: p})
		}
		for _, e := range a.Extensions {
			allExtensions = append(allExtensions, extEntry{Agent: a.Name, Ext: e})
		}
	}

	// Summary stats
	var stats []string
	if len(bom.Models) > 0 {
		stats = append(stats, statStyle.Render(fmt.Sprintf("%d models", len(bom.Models))))
	}
	if len(bom.Agents) > 0 {
		stats = append(stats, statStyle.Render(fmt.Sprintf("%d agents", len(bom.Agents))))
	}
	if len(allSkills) > 0 {
		stats = append(stats, statStyle.Render(fmt.Sprintf("%d skills", len(allSkills))))
	}
	if len(allMcp) > 0 {
		stats = append(stats, statStyle.Render(fmt.Sprintf("%d MCP servers", len(allMcp))))
	}
	if len(allPlugins) > 0 {
		stats = append(stats, statStyle.Render(fmt.Sprintf("%d plugins", len(allPlugins))))
	}
	if len(allExtensions) > 0 {
		stats = append(stats, statStyle.Render(fmt.Sprintf("%d extensions", len(allExtensions))))
	}
	if len(bom.Guardrails) > 0 {
		stats = append(stats, statStyle.Render(fmt.Sprintf("%d guardrails", len(bom.Guardrails))))
	}
	if len(bom.KnowledgeBases) > 0 {
		stats = append(stats, statStyle.Render(fmt.Sprintf("%d knowledge bases", len(bom.KnowledgeBases))))
	}
	if len(bom.ComputeAccess) > 0 {
		stats = append(stats, statStyle.Render(fmt.Sprintf("%d compute services", len(bom.ComputeAccess))))
	}
	if len(bom.AIDependencies) > 0 {
		stats = append(stats, statStyle.Render(fmt.Sprintf("%d AI dependencies", len(bom.AIDependencies))))
	}
	if len(stats) > 0 {
		fmt.Fprintf(w, "%s\n", strings.Join(stats, dimStyle.Render("  ·  ")))
	}

	// Models
	if len(bom.Models) > 0 {
		renderSection(w, "Models", "AI/ML models from local caches, registries, and cloud providers")
		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "  NAME\tPROVIDER\tVERSION\tLICENSE\tFAMILY\tPARAMS")
		fmt.Fprintln(tw, "  ----\t--------\t-------\t-------\t------\t------")
		for _, m := range bom.Models {
			license := m.License
			if len(license) > 25 {
				license = license[:25] + "..."
			}
			fmt.Fprintf(tw, "  %s\t%s\t%s\t%s\t%s\t%s\n",
				m.Name, m.Provider, m.Version, license, m.ArchitectureFamily, m.ParameterSize)
		}
		tw.Flush()
	}

	// Agents
	if len(bom.Agents) > 0 {
		renderSection(w, "Agents", "AI coding agents and cloud orchestration services")
		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "  AGENT\tPROVIDER\tVERSION\tMODEL")
		fmt.Fprintln(tw, "  -----\t--------\t-------\t-----")
		for _, a := range bom.Agents {
			fmt.Fprintf(tw, "  %s\t%s\t%s\t%s\n",
				a.Name, a.Provider, a.Version, a.Model)
		}
		tw.Flush()
	}

	// Skills
	if len(allSkills) > 0 {
		renderSection(w, "Skills", "Registered capabilities that extend agent behavior")
		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "  SKILL\tAGENT")
		fmt.Fprintln(tw, "  -----\t-----")
		for _, s := range allSkills {
			fmt.Fprintf(tw, "  %s\t%s\n", s.Skill.Name, s.Agent)
		}
		tw.Flush()
	}

	// MCP Servers
	if len(allMcp) > 0 {
		renderSection(w, "MCP Servers", "Model Context Protocol servers providing tools to agents")
		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "  NAME\tAGENT\tTYPE\tENDPOINT")
		fmt.Fprintln(tw, "  ----\t-----\t----\t--------")
		for _, m := range allMcp {
			endpoint := m.Mcp.Url
			if endpoint == "" && m.Mcp.Command != "" {
				endpoint = m.Mcp.Command
				if len(m.Mcp.Args) > 0 {
					endpoint += " " + strings.Join(m.Mcp.Args, " ")
				}
				if len(endpoint) > 60 {
					endpoint = endpoint[:60] + "..."
				}
			}
			fmt.Fprintf(tw, "  %s\t%s\t%s\t%s\n",
				m.Mcp.Name, m.Agent, m.Mcp.Type, endpoint)
		}
		tw.Flush()
	}

	// Plugins
	if len(allPlugins) > 0 {
		renderSection(w, "Plugins", "Agent plugins and integrations")
		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "  PLUGIN\tAGENT\tVERSION\tENABLED")
		fmt.Fprintln(tw, "  ------\t-----\t-------\t-------")
		for _, p := range allPlugins {
			enabled := "yes"
			if !p.Plugin.Enabled {
				enabled = "no"
			}
			fmt.Fprintf(tw, "  %s\t%s\t%s\t%s\n",
				p.Plugin.Name, p.Agent, p.Plugin.Version, enabled)
		}
		tw.Flush()
	}

	// Extensions
	if len(allExtensions) > 0 {
		renderSection(w, "Extensions", "Agent extensions and add-ons")
		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "  EXTENSION\tAGENT\tTYPE\tENABLED")
		fmt.Fprintln(tw, "  ---------\t-----\t----\t-------")
		for _, e := range allExtensions {
			enabled := "yes"
			if !e.Ext.Enabled {
				enabled = "no"
			}
			fmt.Fprintf(tw, "  %s\t%s\t%s\t%s\n",
				e.Ext.Name, e.Agent, e.Ext.Type, enabled)
		}
		tw.Flush()
	}

	// Guardrails
	if len(bom.Guardrails) > 0 {
		renderSection(w, "Guardrails", "AI safety and content filtering policies")
		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "  GUARDRAIL\tPROVIDER\tSTATUS\tVERSION\tPOLICIES")
		fmt.Fprintln(tw, "  ---------\t--------\t------\t-------\t--------")
		for _, g := range bom.Guardrails {
			fmt.Fprintf(tw, "  %s\t%s\t%s\t%s\t%d\n",
				g.Name, g.Provider, g.Status, g.Version, len(g.Policies))
		}
		tw.Flush()
	}

	// Knowledge Bases
	if len(bom.KnowledgeBases) > 0 {
		renderSection(w, "Knowledge Bases", "Vector stores and retrieval-augmented generation data")
		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "  NAME\tPROVIDER\tSTATUS\tSTORAGE\tDATA SOURCES")
		fmt.Fprintln(tw, "  ----\t--------\t------\t-------\t------------")
		for _, kb := range bom.KnowledgeBases {
			fmt.Fprintf(tw, "  %s\t%s\t%s\t%s\t%d\n",
				kb.Name, kb.Provider, kb.Status, kb.StorageType, len(kb.DataSources))
		}
		tw.Flush()
	}

	// Compute with AI Access
	if len(bom.ComputeAccess) > 0 {
		renderSection(w, "Compute with AI Access", "Serverless functions and services with AI service permissions")
		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "  SERVICE\tTYPE\tPROVIDER\tRUNTIME\tAI SERVICES\tENV HINTS")
		fmt.Fprintln(tw, "  -------\t----\t--------\t-------\t-----------\t---------")
		for _, ca := range bom.ComputeAccess {
			services := strings.Join(ca.AIServices, ",")
			if len(services) > 30 {
				services = services[:30] + "..."
			}
			fmt.Fprintf(tw, "  %s\t%s\t%s\t%s\t%s\t%d\n",
				ca.Name, ca.Type, ca.Provider, ca.Runtime, services, len(ca.EnvHints))
		}
		tw.Flush()
	}

	// AI Dependencies
	if len(bom.AIDependencies) > 0 {
		renderSection(w, "AI Dependencies", "AI/ML libraries detected in package manifests")
		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "  PACKAGE\tVERSION\tCATEGORY\tLANGUAGE")
		fmt.Fprintln(tw, "  -------\t-------\t--------\t--------")
		for _, d := range bom.AIDependencies {
			fmt.Fprintf(tw, "  %s\t%s\t%s\t%s\n",
				d.Name, d.Version, d.Category, d.Language)
		}
		tw.Flush()
	}

	fmt.Fprintln(w)
	return nil
}

func renderSection(w io.Writer, title, description string) {
	fmt.Fprintf(w, "\n%s %s\n", headerStyle.Render("▌ "+title), dimStyle.Render("— "+description))
}
