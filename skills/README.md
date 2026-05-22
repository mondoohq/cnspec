# cnspec Skills

cnspec skills are definitions for MQL development and policy navigation tasks. They are interoperable with all major coding agent tools like Anthropic's Claude Code, OpenAI Codex, Google DeepMind's Gemini CLI, and Cursor.

The skills in this repository follow the standardized [Agent Skills](https://agentskills.io/home) format.

> [!TIP]
> If your agent doesn't support skills, you can use [`agents/AGENTS.md`](../agents/AGENTS.md) directly as a fallback.

## How do skills work?

Skills are self-contained folders that package instructions and resources together for an AI agent to use on a specific use case. Each folder includes a `SKILL.md` file with YAML frontmatter (name and description) followed by the guidance your coding agent follows while the skill is active.

## Installation

cnspec skills are compatible with Claude Code, Codex, Gemini CLI, and Cursor.

### Claude Code

1. Register the repository as a plugin marketplace:

```
/plugin marketplace add mondoohq/cnspec
```

2. Install a skill:

```
/plugin install mql@mondoohq/cnspec
/plugin install policy-graph@mondoohq/cnspec
```

### Codex

1. Copy or symlink skill directories from `skills/` into one of Codex's standard `.agents/skills` locations (e.g., `$REPO_ROOT/.agents/skills` or `$HOME/.agents/skills`) as described in the [Codex Skills guide](https://developers.openai.com/codex/skills/).

2. Once available, Codex will discover the skill and load the `SKILL.md` instructions automatically.

3. If your Codex setup still relies on `AGENTS.md`, use the generated [`agents/AGENTS.md`](../agents/AGENTS.md) file as a fallback.

### Gemini CLI

Install locally:

```
gemini extensions install . --consent
```

Or from GitHub:

```
gemini extensions install https://github.com/mondoohq/cnspec.git --consent
```

See [Gemini CLI extensions docs](https://geminicli.com/docs/extensions/#installing-an-extension) for more help.

### Cursor

This repository includes Cursor plugin manifests:

- `.cursor-plugin/plugin.json`
- `.cursor-plugin/marketplace.json`

Install from repository URL or local checkout via the Cursor plugin flow.

## Available skills

| Name | Description | Documentation |
|------|-------------|---------------|
| `mql` | MQL query development with syntax guidance, platform-specific patterns, and schema discovery | [SKILL.md](mql/SKILL.md) |
| `policy-graph` | Navigate cnspec policy bundles using graph commands — search, trace compliance mappings, explore structure | [SKILL.md](policy-graph/SKILL.md) |

### mql

The MQL skill provides comprehensive guidance for writing MQL queries and security policies:

- **MQL Reference** — Complete syntax documentation, best practices, and anti-patterns
- **Platform Samples** — Ready-to-use patterns for AWS, Azure, Linux, Windows, and Microsoft 365
- **Schema Discovery** — Real-time schema lookup via `cnspec` CLI or Mondoo MCP tools
- **Query Validation** — Compile-time syntax and semantic checking
- **Policy Management** — Linting, formatting, and scaffolding policy bundles

### policy-graph

The policy graph skill provides structured navigation of `.mql.yaml` policy bundles:

- **Search** — Find nodes by name, title, or UID
- **Callers/Callees** — Explore inbound and outbound relationships
- **Context** — Get LLM-friendly markdown with YAML snippets for any node
- **Paths** — Trace compliance mappings from frameworks to checks
- **Reachable** — Find all nodes transitively reachable from a starting point

See [docs/graph.md](../docs/graph.md) for full command documentation.

## Adding a new skill

1. Create a directory under `skills/<name>/` with a `SKILL.md` containing frontmatter:

   ```markdown
   ---
   name: <skill-name>
   description: <one-line description of when to use this skill>
   ---

   # Skill content here
   ```

2. Add the skill to the marketplace manifests in `.claude-plugin/marketplace.json` and `.cursor-plugin/marketplace.json`.

3. Regenerate `agents/AGENTS.md`:

   ```bash
   make skills/generate
   ```

4. Verify everything is in sync:

   ```bash
   make skills/generate/check
   ```

## Requirements

- `cnspec` CLI installed and on PATH
- Works with any `.mql.yaml` policy or query pack files
