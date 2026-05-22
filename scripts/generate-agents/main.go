// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// generate-agents scans skills/*/skills/*/SKILL.md for frontmatter,
// generates agents/AGENTS.md from a template, and validates that
// plugin marketplace manifests are in sync with discovered skills.
//
// Usage:
//
//	go run ./scripts/generate-agents            # generate
//	go run ./scripts/generate-agents --check    # verify up-to-date (CI)
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type skill struct {
	Name        string
	Description string
	Path        string // relative path to the skill dir (e.g. "skills/mondoo-mql/skills/mondoo-mql")
}

type marketplaceJSON struct {
	Plugins []marketplacePlugin `json:"plugins"`
}

type marketplacePlugin struct {
	Name        string `json:"name"`
	Source      string `json:"source"`
	Skills      string `json:"skills"`
	Description string `json:"description"`
}

var frontmatterRe = regexp.MustCompile(`(?s)^---\s*\n(.*?)\n---`)

func parseFrontmatter(data []byte) map[string]string {
	m := frontmatterRe.FindSubmatch(data)
	if m == nil {
		return nil
	}
	result := map[string]string{}
	for _, line := range strings.Split(string(m[1]), "\n") {
		key, val, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		result[strings.TrimSpace(key)] = strings.TrimSpace(val)
	}
	return result
}

func collectSkills(root string) ([]skill, error) {
	pattern := filepath.Join(root, "skills", "*", "SKILL.md")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	var skills []skill
	for _, path := range matches {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", path, err)
		}
		fm := parseFrontmatter(data)
		if fm["name"] == "" || fm["description"] == "" {
			fmt.Fprintf(os.Stderr, "warning: %s missing name or description in frontmatter, skipping\n", path)
			continue
		}
		rel, err := filepath.Rel(root, filepath.Dir(path))
		if err != nil {
			return nil, fmt.Errorf("computing relative path for %s: %w", path, err)
		}
		skills = append(skills, skill{
			Name:        fm["name"],
			Description: fm["description"],
			Path:        rel,
		})
	}

	sort.Slice(skills, func(i, j int) bool {
		return strings.ToLower(skills[i].Name) < strings.ToLower(skills[j].Name)
	})
	return skills, nil
}

func renderTemplate(tmpl string, skills []skill) string {
	blockRe := regexp.MustCompile(`(?s)\{\{#skills\}\}(.*?)\{\{/skills\}\}`)
	return blockRe.ReplaceAllStringFunc(tmpl, func(match string) string {
		inner := blockRe.FindStringSubmatch(match)
		if len(inner) < 2 {
			return match
		}
		block := strings.TrimLeft(inner[1], "\n")
		block = strings.TrimRight(block, "\n")

		var lines []string
		for _, s := range skills {
			line := block
			line = strings.ReplaceAll(line, "{{name}}", s.Name)
			line = strings.ReplaceAll(line, "{{description}}", s.Description)
			line = strings.ReplaceAll(line, "{{path}}", s.Path)
			lines = append(lines, line)
		}
		return strings.Join(lines, "\n")
	})
}

func validateMarketplace(path string, skills []skill) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return []string{fmt.Sprintf("reading %s: %v", path, err)}
	}

	var mp marketplaceJSON
	if err := json.Unmarshal(data, &mp); err != nil {
		return []string{fmt.Sprintf("parsing %s: %v", path, err)}
	}

	pluginByName := map[string]marketplacePlugin{}
	for _, p := range mp.Plugins {
		pluginByName[p.Name] = p
	}

	var errs []string
	for _, s := range skills {
		if _, ok := pluginByName[s.Name]; !ok {
			errs = append(errs, fmt.Sprintf("skill %q missing from %s", s.Name, path))
		}
	}
	for _, p := range mp.Plugins {
		found := false
		for _, s := range skills {
			if s.Name == p.Name {
				found = true
				break
			}
		}
		if !found {
			errs = append(errs, fmt.Sprintf("plugin %q in %s has no SKILL.md", p.Name, path))
		}
	}
	return errs
}

func main() {
	check := flag.Bool("check", false, "verify generated files are up-to-date")
	flag.Parse()

	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	skills, err := collectSkills(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error collecting skills: %v\n", err)
		os.Exit(1)
	}

	if len(skills) == 0 {
		fmt.Fprintln(os.Stderr, "warning: no skills found")
	}

	tmplPath := filepath.Join(root, "scripts", "AGENTS_TEMPLATE.md")
	tmplData, err := os.ReadFile(tmplPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading template: %v\n", err)
		os.Exit(1)
	}

	output := renderTemplate(string(tmplData), skills)
	outPath := filepath.Join(root, "agents", "AGENTS.md")

	if *check {
		existing, err := os.ReadFile(outPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading %s: %v\n", outPath, err)
			os.Exit(1)
		}
		if string(existing) != output {
			fmt.Fprintln(os.Stderr, "agents/AGENTS.md is outdated. Run: go run ./scripts/generate-agents")
			os.Exit(1)
		}
		fmt.Println("agents/AGENTS.md is up to date.")
	} else {
		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		if err := os.WriteFile(outPath, []byte(output), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "error writing %s: %v\n", outPath, err)
			os.Exit(1)
		}
		fmt.Printf("Wrote %s with %d skills.\n", outPath, len(skills))
	}

	var allErrs []string
	for _, mp := range []string{
		filepath.Join(root, ".claude-plugin", "marketplace.json"),
		filepath.Join(root, ".cursor-plugin", "marketplace.json"),
	} {
		allErrs = append(allErrs, validateMarketplace(mp, skills)...)
	}
	if len(allErrs) > 0 {
		fmt.Fprintln(os.Stderr, "\nMarketplace validation errors:")
		for _, e := range allErrs {
			fmt.Fprintf(os.Stderr, "  - %s\n", e)
		}
		os.Exit(1)
	}
	if !*check {
		fmt.Println("Marketplace validation passed.")
	}
}
