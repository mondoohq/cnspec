---
name: policy-graph
description: Navigates cnspec policy/framework bundles using graph commands. Use when exploring policies, finding checks, tracing compliance mappings, or understanding policy structure.
allowed-tools: Bash Read Glob Grep
---

# cnspec Policy Graph Navigation

Navigate and understand cnspec policy bundles (`.mql.yaml` files) using structured graph commands.

## When to Use

- Exploring policy bundle structure ("what checks does this policy have?")
- Finding compliance mappings ("which framework controls map to this check?")
- Tracing relationships ("how does this framework relate to that check?")
- Understanding large bundles (10K+ line `.mql.yaml` files)
- Getting context for a specific check with its MQL code, impact, and docs

## When NOT to Use

- Scanning assets with cnspec (`cnspec scan`)
- Writing MQL queries
- Linting bundles (`cnspec policy lint`)
- Editing policy YAML directly (use the Read/Edit tools for that)

## Commands Reference

| Command | Purpose |
|---------|---------|
| `cnspec policy graph callers <uid> <path>` | What references this node (inbound edges) |
| `cnspec policy graph callees <uid> <path>` | What this node contains/references (outbound edges) |
| `cnspec policy graph context <uid> <path> [--depth N]` | LLM-friendly context with YAML snippets |
| `cnspec policy graph paths <from> <to> <path>` | Find paths between two nodes |
| `cnspec policy graph reachable <uid> <path>` | All nodes transitively reachable |
| `cnspec policy graph export <path> [--format json\|dot]` | Export full graph |

All commands support `--json` for structured output.

## Graph Concepts

### Node Types

- **policy**: A security policy (e.g., `mondoo-linux-security`)
- **group**: A group of checks within a policy (e.g., "SSH Configuration")
- **check**: A scoring query with MQL code and impact rating
- **query**: A data-only query (no scoring)
- **framework**: A compliance framework (e.g., CIS Benchmark, ISO 27001)
- **control**: A control within a framework (e.g., CIS 5.2.1)
- **framework_map**: Maps framework controls to policy checks

### Edge Types

- **contains**: Structural parent-child (policy → group → check, framework → control)
- **maps_to**: Compliance mapping (control → check/policy via framework_map)
- **depends_on**: Policy/framework dependencies
- **variant_of**: Check variant relationship (parent → platform-specific variant)
