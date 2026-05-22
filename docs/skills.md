# cnspec agent skills

cnspec ships with agent skills that provide coding agents with MQL expertise and policy navigation capabilities. Skills follow the [Agent Skills](https://agentskills.io/home) format and work across Claude Code, Cursor, Gemini CLI, and Codex.

## Available skills

| Skill | Description |
|-------|-------------|
| **mql** | MQL query development with syntax guidance, platform-specific patterns, and schema discovery |
| **policy-graph** | Navigate policy bundles using graph commands — search, trace compliance mappings, explore structure |

## Installation

### Claude Code

Register the cnspec repository as a plugin marketplace:

```shell
/plugin marketplace add mondoohq/cnspec
```

Install a skill:

```shell
/plugin install mql@mondoohq/cnspec
/plugin install policy-graph@mondoohq/cnspec
```

Or install all skills locally from a checkout:

```bash
make install/skills
```

### Codex

Copy or symlink skill directories from `skills/` into one of Codex's standard `.agents/skills` locations (e.g., `$REPO_ROOT/.agents/skills` or `$HOME/.agents/skills`).

If your Codex setup relies on `AGENTS.md`, use the generated [`agents/AGENTS.md`](../agents/AGENTS.md) file.

### Gemini CLI

Install from a local checkout:

```shell
gemini extensions install . --consent
```

Or from GitHub:

```shell
gemini extensions install https://github.com/mondoohq/cnspec.git --consent
```

The `gemini-extension.json` at the repo root points Gemini CLI to `agents/AGENTS.md`.

### Cursor

The `.cursor-plugin/` directory at the repo root provides Cursor plugin manifests. Install from the repository URL or a local checkout via the Cursor plugin flow.

## MQL skill

The MQL skill (`skills/mql/`) provides comprehensive guidance for writing MQL queries and security policies.

### What it includes

| File | Purpose |
|------|---------|
| `SKILL.md` | Skill definition with schema discovery, CLI commands, and quick reference |
| `mql-reference.md` | Complete MQL language reference — syntax, patterns, best practices, anti-patterns |
| `samples/general.md` | General MQL patterns |
| `samples/aws.md` | AWS resource patterns (IAM, EC2, S3, CloudTrail, KMS) |
| `samples/azure.md` | Azure resource patterns (VMs, storage, SQL, App Service) |
| `samples/linux.md` | Linux system patterns (files, services, packages, kernel, users) |
| `samples/windows.md` | Windows patterns (registry, secpol, auditpol, PowerShell) |
| `samples/ms365.md` | Microsoft 365 patterns |

### Schema discovery

The skill teaches agents to use `cnspec` CLI commands for real-time schema lookup:

```bash
# List all providers
cnspec providers list --json

# List resources in a provider
cnspec providers resources aws --json

# Get field details for a resource
cnspec providers resources aws aws.ec2.instance --json

# Validate a query (semantic check — catches invalid resources/fields)
cnspec run local -c "asset.name" --ast

# Lint a policy bundle
cnspec policy lint policy.mql.yaml -o sarif

# Format a policy bundle
cnspec policy format policy.mql.yaml
```

### When it activates

The skill triggers when an agent is:

- Writing MQL queries or policies
- Validating MQL syntax
- Exploring available MQL resources and fields
- Developing platform-specific security checks

## Policy graph skill

The policy graph skill (`skills/policy-graph/`) provides structured navigation of `.mql.yaml` policy bundles. See [docs/graph.md](graph.md) for full documentation of the graph commands.

### When it activates

The skill triggers when an agent is:

- Exploring policy bundle structure
- Finding compliance mappings between frameworks and checks
- Tracing relationships across policies, groups, and controls
- Investigating large bundles (10K+ line `.mql.yaml` files)

## Generating AGENTS.md

The `agents/AGENTS.md` file is auto-generated from skill frontmatter by a Go program. It serves as a fallback for agents that don't support the plugin/skill system natively.

```bash
# Regenerate after adding or modifying skills
make skills/generate

# Verify the generated file is up-to-date (for CI)
make skills/generate/check
```

The generator (`scripts/generate-agents/main.go`) scans `skills/*/SKILL.md` for frontmatter, renders `agents/AGENTS.md` from the template at `scripts/AGENTS_TEMPLATE.md`, and validates that the `.claude-plugin/marketplace.json` and `.cursor-plugin/marketplace.json` manifests are in sync with discovered skills.

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
