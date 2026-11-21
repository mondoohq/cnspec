// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package generate

import (
	"fmt"
	"strings"
)

// PromptData is everything the prompt template needs for one check.
type PromptData struct {
	Title       string
	Desc        string
	Provider    string
	Props       []Prop
	Examples    []Example
	Explain     bool
	Guidance    string   // optional extra author steering
	SkillPaths  []string // paths to any discovered skill files (mql, policy-graph)
	RetryError  string   // compiler/validation error from the previous attempt
	PreviousMQL string   // the MQL that failed, when retrying
}

// BuildPrompt renders the full instruction handed to a coding-agent backend. It
// is deliberately explicit about the MQL correctness traps that a compile check
// cannot catch, and it tells the agent to use cnspec's own commands to confirm
// the schema and validate the query before returning.
func BuildPrompt(d PromptData) string {
	var b strings.Builder

	b.WriteString("You are generating one MQL (Mondoo Query Language) query for a cnspec security policy check. ")
	b.WriteString("Return a single boolean MQL expression that PASSES when the asset is compliant and FAILS otherwise.\n\n")

	b.WriteString("## Intent\n")
	fmt.Fprintf(&b, "Title: %s\n", strings.TrimSpace(d.Title))
	if strings.TrimSpace(d.Desc) != "" {
		fmt.Fprintf(&b, "Description: %s\n", strings.TrimSpace(d.Desc))
	}
	if d.Provider != "" {
		fmt.Fprintf(&b, "Target provider: %s\n", d.Provider)
	}
	if strings.TrimSpace(d.Guidance) != "" {
		fmt.Fprintf(&b, "Additional guidance from the author: %s\n", strings.TrimSpace(d.Guidance))
	}
	if len(d.Props) > 0 {
		b.WriteString("Available properties (reference these as props.<name>, do not hardcode their values):\n")
		for _, p := range d.Props {
			fmt.Fprintf(&b, "  - props.%s", p.Name)
			if p.Type != "" {
				fmt.Fprintf(&b, " (%s)", p.Type)
			}
			if p.Desc != "" {
				fmt.Fprintf(&b, ": %s", strings.TrimSpace(p.Desc))
			}
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")

	b.WriteString("## Use cnspec to ground and validate your query\n")
	if d.Provider != "" {
		fmt.Fprintf(&b, "- Run `cnspec providers resources %s --json` to list resources, and `cnspec providers resources %s <resource> --json` to confirm exact field names and types.\n", d.Provider, d.Provider)
	} else {
		b.WriteString("- Run `cnspec providers resources <provider> --json` and `cnspec providers resources <provider> <resource> --json` to confirm exact resource and field names.\n")
	}
	b.WriteString("- Validate that your query compiles before returning it: `cnspec run <connection> -c \"<your mql>\" --ast` (exit 0 = compiles, exit 1 = unknown resource/field or syntax error). Do not return a query that fails this check.\n")
	if len(d.SkillPaths) > 0 {
		b.WriteString("- Read these skill files for MQL syntax, patterns, and navigating existing policies:\n")
		for _, p := range d.SkillPaths {
			fmt.Fprintf(&b, "  - %s\n", p)
		}
	} else {
		b.WriteString("- If your environment provides the cnspec `mql` or `policy-graph` skills, consult them for MQL syntax and for finding existing checks to mirror.\n")
	}
	b.WriteString("\n")

	if len(d.Examples) > 0 {
		b.WriteString("## Similar existing checks (validated MQL — mirror these patterns)\n")
		for _, ex := range d.Examples {
			title := ex.Title
			if title == "" {
				title = ex.UID
			}
			fmt.Fprintf(&b, "- %s\n  ```mql\n  %s\n  ```\n", strings.TrimSpace(title), indent(strings.TrimSpace(ex.Mql), "  "))
		}
		b.WriteString("\n")
	}

	b.WriteString(correctnessRules)
	b.WriteString("\n")

	if d.RetryError != "" {
		b.WriteString("## Your previous attempt failed validation — fix it\n")
		if d.PreviousMQL != "" {
			fmt.Fprintf(&b, "Previous MQL:\n```mql\n%s\n```\n", strings.TrimSpace(d.PreviousMQL))
		}
		fmt.Fprintf(&b, "Validation error: %s\n\n", strings.TrimSpace(d.RetryError))
	}

	b.WriteString("## Output format\n")
	b.WriteString("Return ONLY a single fenced json block, nothing else:\n")
	if d.Explain {
		b.WriteString("```json\n{\"mql\": \"<the query>\", \"explanation\": \"<one or two sentences on how it maps to the intent>\"}\n```\n")
	} else {
		b.WriteString("```json\n{\"mql\": \"<the query>\"}\n```\n")
	}
	b.WriteString("The mql value must be a single expression. Do not include the `filters:` selector — that is handled separately.\n")

	return b.String()
}

// correctnessRules are the semantic traps a compile check cannot catch, drawn
// from cnspec/CLAUDE.md. They are the difference between MQL that compiles and
// MQL that returns the right verdict.
const correctnessRules = `## MQL correctness rules (a query can compile and still be wrong — follow these)
- Assert presence before comparing. ` + "`null && null` evaluates to `true`" + `, so a check that only compares fields silently passes when the data never resolved. Write ` + "`field != empty && field == \"x\"`" + `, not ` + "`field == \"x\"`" + ` alone.
- Use ` + "`!= empty`" + ` for non-empty checks, never ` + "`!= \"\"`" + ` (` + "`null != \"\"`" + ` is true, so it is not null-safe).
- ` + "`.all()` / `.none()` on `null` errors" + `; on an empty list they pass vacuously. When a collection may be absent, guard it: ` + "`x == empty || x.all(...)`" + `.
- Do not reach a sub-object through a path that is itself a resource name — it compiles to an empty husk and every field reads null. Prefer an accessor path or a block bound to the parent (` + "`parent { child.field != \"none\" }`" + `).
- MQL has NO parenthesized grouping of boolean expressions. Rely on precedence (` + "`&&` binds tighter than `||`" + `) or split into separate ` + "`.any()` / `.all()`" + ` calls.
- Regex match is ` + "`string == /pattern/`" + `, not ` + "`=~`" + `.
- Multiple newline-separated lines in the query are implicitly AND-ed.
- Keep asset selection (` + "`asset.platform == ...`" + `) OUT of the query — it belongs in filters, not here.`

func indent(s, prefix string) string {
	lines := strings.Split(s, "\n")
	for i := range lines {
		if i == 0 {
			continue // first line already positioned by caller
		}
		lines[i] = prefix + lines[i]
	}
	return strings.Join(lines, "\n")
}
