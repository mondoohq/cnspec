# policy-graph

An agent skill for navigating and understanding cnspec policy bundles using graph commands.

## What it does

Provides structured navigation of `.mql.yaml` policy bundles that replaces manual file reading with single commands:

- **Callers**: Find what policies, groups, or compliance controls reference a check
- **Callees**: Find what checks, queries, or sub-policies a policy contains
- **Context**: Get LLM-friendly markdown with YAML snippets for any node
- **Paths**: Trace compliance mappings from frameworks to checks
- **Reachable**: Find all nodes transitively reachable from a starting point
- **Export**: Export the full graph as JSON or DOT

## Usage

```
/policy-graph what checks does mondoo-linux-security contain?
/policy-graph which compliance controls map to the SSH root login check?
/policy-graph show me the context for mondoo-aws-security-iam-root-user-mfa
/policy-graph trace the path from CIS benchmark to SSH ciphers check
```

## Installation

See [skills/README.md](../README.md) for installation instructions across Claude Code, Codex, Gemini CLI, and Cursor.
