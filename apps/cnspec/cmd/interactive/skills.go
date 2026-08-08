// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

// Skill describes a cnspec AI-agent skill: a packaged set of instructions and
// resources that coding agents (Claude Code, Codex, Gemini CLI, Cursor) load to
// help with a specific cnspec task. These ship in the cnspec repo under skills/.
type Skill struct {
	Name    string
	Summary string
	// Detail is a short multi-line description shown when the skill is focused.
	Detail []string
}

// Skills is the catalog of skills cnspec publishes. Kept as a small static list
// because the running binary does not carry the repo's skills/ directory.
var Skills = []Skill{
	{
		Name:    "mql",
		Summary: "Write & validate MQL queries and security policies with your agent",
		Detail: []string{
			"Guidance for authoring MQL (Mondoo Query Language) queries and policies,",
			"with live schema lookup and query validation through the cnspec CLI or",
			"Mondoo's MCP tools. Use it while writing checks, exploring resources, or",
			"debugging a policy.",
		},
	},
	{
		Name:    "policy-graph",
		Summary: "Navigate and reason about cnspec policy graphs",
		Detail: []string{
			"Helps an agent traverse policies, controls, and their query dependencies,",
			"so it can explain scoring, find the checks behind a control, and map",
			"compliance frameworks to the queries that satisfy them.",
		},
	},
}

// skillInstall are the steps to add cnspec skills to a coding agent. Shown on
// the Skills screen so users can act on the highlight.
var skillInstall = []string{
	"Claude Code:  /plugin marketplace add mondoohq/cnspec",
	"              /plugin install mql@cnspec-skills",
	"Codex/Cursor: copy skills/<name>/ into your agent's skills directory",
	"Fallback:     use agents/AGENTS.md directly",
}
