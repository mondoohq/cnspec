// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	"fmt"
	"strings"
)

// Special home-tile keys that are not categories.
const (
	tileSkills = "__skills__"
	tileSearch = "__search__"
)

// homeTile is one entry on the use-case landing screen.
type homeTile struct {
	key       string // a category name, or tileSkills / tileSearch
	icon      string
	title     string
	blurb     string
	highlight bool // draw with the AI accent (AI Services, Skills)
	count     int  // number of connectors, for category tiles
}

// categoryIcon gives each use-case a glyph for the launcher.
var categoryIcon = map[string]string{
	catCloud:     "☁",
	catSaaS:      "⬡",
	catAI:        "✦",
	catContainer: "◈",
	catHosts:     "▚",
	catDatabase:  "⛁",
	catNetwork:   "⟟",
	catIaC:       "❏",
	catDev:       "⚙",
	catOther:     "•",
}

// homeCategoryOrder curates the landing order so the marquee use-cases
// (cloud, SaaS, AI services) come first.
var homeCategoryOrder = []string{
	catCloud,
	catSaaS,
	catAI,
	catContainer,
	catHosts,
	catDatabase,
	catNetwork,
	catIaC,
	catDev,
	catOther,
}

// homeTitle overrides the category name on the landing screen where a friendlier
// use-case label reads better.
func homeTitle(cat string) string {
	switch cat {
	case catAI:
		return "AI Services"
	case catContainer:
		return "Kubernetes & Containers"
	default:
		return cat
	}
}

// prettyName maps connector names to display-friendly labels for tile blurbs.
var prettyName = map[string]string{
	"aws": "AWS", "gcp": "GCP", "oci": "OCI", "k8s": "Kubernetes",
	"ms365": "Microsoft 365", "mcp": "MCP", "openai": "OpenAI",
	"cloudformation": "CloudFormation", "activedirectory": "Active Directory",
	"depsdev": "deps.dev", "alicloud": "Alibaba Cloud",
	"vllm": "vLLM", "huggingface": "Hugging Face", "mongodbatlas": "MongoDB Atlas",
	"digitalocean": "DigitalOcean", "hcp": "HCP", "vcd": "VMware Cloud Director",
	"ipmi": "IPMI", "bigip": "BIG-IP", "nextdns": "NextDNS", "opcua": "OPC UA",
	"mssql": "SQL Server", "postgresdb": "PostgreSQL", "mysqldb": "MySQL",
	"redisdb": "Redis", "google-workspace": "Google Workspace",
}

func titleize(name string) string {
	if p, ok := prettyName[name]; ok {
		return p
	}
	if name == "" {
		return name
	}
	return strings.ToUpper(name[:1]) + name[1:]
}

func blurbFor(examples []string, count int) string {
	pretty := make([]string, 0, len(examples))
	for _, e := range examples {
		pretty = append(pretty, titleize(e))
	}
	s := strings.Join(pretty, ", ")
	if count > len(examples) {
		s += fmt.Sprintf(" +%d more", count-len(examples))
	}
	return s
}

// buildHomeTiles derives the landing screen from the catalog, skipping empty
// categories and appending the Skills and Search entries.
func buildHomeTiles(catalog []Connector) []homeTile {
	counts := map[string]int{}
	examples := map[string][]string{}
	for _, c := range catalog {
		counts[c.Category]++
		if len(examples[c.Category]) < 3 {
			examples[c.Category] = append(examples[c.Category], c.Name)
		}
	}

	var tiles []homeTile
	for _, cat := range homeCategoryOrder {
		if counts[cat] == 0 {
			continue
		}
		tiles = append(tiles, homeTile{
			key:       cat,
			icon:      categoryIcon[cat],
			title:     homeTitle(cat),
			blurb:     blurbFor(examples[cat], counts[cat]),
			highlight: cat == catAI,
			count:     counts[cat],
		})
	}

	tiles = append(tiles, homeTile{
		key:       tileSkills,
		icon:      "✳",
		title:     "Skills",
		blurb:     "AI-agent skills for MQL & policy work — Claude Code, Codex, Cursor",
		highlight: true,
	})
	tiles = append(tiles, homeTile{
		key:   tileSearch,
		icon:  "⌕",
		title: "Search all connectors",
		blurb: fmt.Sprintf("Fuzzy-find across all %d providers and targets", len(catalog)),
	})
	return tiles
}
