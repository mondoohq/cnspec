# ADR-0002: AI Shadow IT Detection — MQL Resource Coverage

**Date:** 2026-04-19
**Status:** Proposed

## Context

Organizations lack visibility into which AI tools employees use on corporate
endpoints. Shadow AI creates concrete risks:

- **Data egress** — source code, PII, secrets pasted into consumer-tier AI services.
- **Supply-chain risk via MCP servers** — AI agents granted tool access to
  internal systems (Gmail, Drive, Jira, databases) through Model Context
  Protocol servers the security team has never reviewed.
- **Unapproved IDE coding assistants** — tools like Cursor, Codeium, Continue,
  Tabnine, etc. ingesting proprietary source.
- **Malicious browser extensions** — extensions posing as AI helpers that
  exfiltrate tokens or inject content.
- **Unreviewed skills and plugins** — third-party skills and plugins that
  extend agent capabilities with unaudited code and tool access.

The `os` provider in MQL now ships native resources for detecting AI coding
agents, their skills, MCP servers, plugins, IDE extensions, and browser
extensions. This ADR documents the detection capabilities these resources
enable and the design decisions behind them.

## Decision

Use native MQL resources from the **`os` provider** for all AI shadow IT
detection. Do not fall back to raw filesystem scraping (`files.find`,
`parse.json` on config paths) when a native resource exists — native resources
are maintained by Mondoo, version-stable, and cross-platform.

Detection capabilities are organized into six areas:

### 1. AI Coding Agent Detection

27 agent resources detect the presence and configuration of AI coding agents
by scanning per-user configuration directories (e.g., `~/.claude`, `~/.cursor`,
`~/.config/goose`). All are marked `@maturity("preview")` and accept an
optional `configPath` override via `init(configPath? string)`.

**Rich agents** (8) expose authentication, settings, and account information
in addition to skills:

| Resource | URL | Notable fields |
|----------|-----|----------------|
| `claude.code` | claude.ai/code | email, organization, role, subscription, userId, settings, enabledPlugins |
| `openai.codex` | openai.com | authMode, accountId, version, lastRefresh |
| `cursor` | cursor.com | (mcpServers, rules, skills — no auth fields) |
| `github.copilot` | github.com/features/copilot | accounts (user, githubAppId) |
| `windsurf` | windsurf.com | (rules, mcpServers, skills) |
| `gemini` | github.com/google-gemini/gemini-cli | authType, settings |
| `goose` | block.github.io/goose | provider, model, telemetryEnabled |
| `zed` | zed.dev | settings (dict), extensions (names only) |

**Lightweight agents** (18) follow a uniform pattern of `configPath` +
`skills()` only:

`roo`, `cline`, `kiro`, `continuedev`, `trae`, `opencode`, `pi`,
`mistral.vibe`, `antigravity`, `ibm.bob`, `openclaw`, `snowflake.cortex`,
`junie`, `augment`, `warp`, `kilocode`, `openhands`, `qwen.code`

Each lightweight agent has a single child resource (`<agent>.skill`) following
the same schema described in section 2.

### 2. AI Agent Skill Detection

All agent resources expose a `skills()` field returning `[]<agent>.skill`
sub-resources. Every skill resource shares the same schema:

| Field | Type | Purpose |
|-------|------|---------|
| `name` | string | Skill identifier |
| `description` | string | What the skill does |
| `allowedTools` | []string | Tools the skill can invoke |
| `argumentHint` | string | Expected argument format |
| `source` | string | Filesystem path of the skill definition |
| `content` | string | Full skill content |
| `sha256()` | string | Content hash for change detection |

Additionally, `cursor.rule` and `windsurf.rule` resources capture agent
instruction and memory files (`name`, `content`, `source`).

**Security relevance:** Skills define what tools an agent can access and what
instructions it follows. A malicious or overly-permissive skill is a
supply-chain and data-exfiltration vector. The `sha256()` field enables drift
detection between scans.

### 3. MCP Server Detection

Six agents expose `*.mcpServer` sub-resources for discovering configured
Model Context Protocol servers:

| Agent | mcpServer fields |
|-------|------------------|
| `claude.code.mcpServer` | name, needsAuth, lastChecked |
| `openai.codex.mcpServer` | name, type, url, note, plugin |
| `cursor.mcpServer` | name, command, url, args, hasEnv |
| `github.copilot.mcpServer` | name, type, command, args |
| `windsurf.mcpServer` | name, command, args, hasEnv |
| `gemini.mcpServer` | name, command, args, hasEnv |

Fields vary because agents configure MCP servers differently — some use
`command` + `args` (stdio transport), others use `url` (HTTP/SSE transport).
The `hasEnv` field flags whether environment variables (potentially containing
secrets) are passed to the server.

**Security relevance:** MCP servers grant AI agents tool access to internal
systems. An unreviewed MCP server can read email, access source control, query
databases, or post to Slack on behalf of the user. This is the highest-risk
shadow IT vector.

### 4. AI Agent Plugin Detection

Three agents expose plugin or extension sub-resources:

**`claude.code.plugin`** — name, version, scope (user/project), installPath,
installedAt, lastUpdated, gitCommitSha, enabled

**`openai.codex.plugin`** — name, version, description, author, category,
capabilities[], skillNames[], hasMcp, hasHooks

**`openai.codex.connector`** — name, id, plugin (OAuth app connectors that
grant agents access to external services)

**`goose.extension`** — name, enabled, type (platform/builtin), description,
bundled, timeout

**Security relevance:** Plugins extend agent capabilities with third-party
code. The `hasMcp` and `hasHooks` fields on Codex plugins flag particularly
sensitive extensions. OAuth connectors (`openai.codex.connector`) represent
delegated access grants.

### 5. IDE Extension Detection

The `vscode` resource and its child `vscode.extension` scan extensions
installed across **eight editor variants** from a single resource:

- VS Code (`~/.vscode/extensions`)
- VS Code Insiders (`~/.vscode-insiders/extensions`)
- VSCodium (`~/.vscode-oss/extensions`)
- Cursor (`~/.cursor/extensions`)
- Windsurf (`~/.windsurf/extensions`)
- Positron (`~/.positron/extensions`)
- Kiro (`~/.kiro/extensions`)
- Antigravity (`~/.antigravity/extensions`)

Each `vscode.extension` exposes: `identifier` (publisher.name format), `name`,
`displayName`, `version`, `description`, `publisher`, `editor`, `path`,
`vscodeVersion`, `categories[]`.

The `editor` field is key — it identifies which editor variant the extension
belongs to, making it possible to detect AI-native editors (Cursor, Windsurf,
Kiro) alongside traditional VS Code installations.

### 6. Browser Extension Detection

The preexisting `chrome`, `firefox`, and `safari` resources enumerate
installed browser extensions. While not new additions, they are essential for
AI shadow IT detection — browser extensions posing as AI helpers can
exfiltrate OAuth tokens, session cookies, or inject content into pages.

### 7. Running AI Processes

The `processes` resource detects actively running AI tools regardless of
whether they have a configuration directory on disk. This catches:

- AI-native editors and agents running as desktop apps (Cursor, Windsurf, Claude)
- Local LLM runtimes (Ollama, LM Studio, llama.cpp, LocalAI, GPT4All)
- Background AI agent processes that may not leave persistent config files

Use `processes.where(name == /(?i)ollama|llama|lmstudio|cursor|windsurf|claude/)` to
filter for known AI process names. Combine with `ports.listening` to detect
local LLM inference servers on known ports (11434 for Ollama, 1234 for LM
Studio, 1337 for LocalAI).

### 8. Installed AI Packages

The `packages` resource (and `homebrew.packages` on macOS) detects AI tools
installed through system package managers, even if they are not currently
running or configured. This catches:

- AI CLI tools installed via Homebrew, apt, dnf, or Windows package managers
- Local LLM runtimes installed as system packages (Ollama, llama.cpp)
- AI coding agents installed as standalone packages

On Windows, `packages` includes entries from the Uninstall registry
(HKLM + HKCU), covering tools installed via MSI, exe installers, or winget.

## Coverage

- **Cross-platform:** All agent resources scan per-user config directories on
  macOS and Linux. Windows support depends on where each tool stores its
  configuration (most use `%USERPROFILE%` equivalents).
- **Declarative:** Detection uses first-class MQL resources instead of
  filesystem path scraping, surviving cnspec upgrades and tool storage format
  changes.
- **Uniform skill schema:** All 27 agents expose the same `*.skill` structure,
  enabling a single query pattern for skill enumeration regardless of agent.
- **Drift detection:** The `sha256()` field on skills enables change alerting
  between scans without storing full content.
- **MCP server coverage:** Six agents expose MCP server configuration —
  covering the highest-risk shadow IT vector (delegated tool access to
  internal systems).
- **Multi-editor IDE scanning:** A single `vscode` resource covers eight
  editor variants, including AI-native editors that would otherwise require
  individual detection logic.
- **Browser extensions:** Pre-existing `chrome`, `firefox`, and `safari`
  resources can be applied to AI shadow IT detection without new resource
  development.
- **Process detection:** `processes` catches running AI tools even without
  persistent config directories — essential for local LLM runtimes and
  portable AI agents.
- **Package detection:** `packages` and `homebrew.packages` detect installed
  AI tools regardless of whether they are running or configured, providing
  a baseline inventory of what is available on the endpoint.

## Future Outlook

- **Preview maturity graduation:** All agent resources are currently
  `@maturity("preview")`. Field names may evolve before stabilization. Policy
  authors should pin to cnspec versions in CI until resources reach stable
  maturity.
- **Lightweight agent enrichment:** The 18 lightweight agents currently expose
  only `configPath` + `skills()`. Future versions should add MCP server and
  plugin detection for agents that support them (e.g., Cline, Roo Code,
  Continue all support MCP servers in practice).
- **Scan-time limitations:** Detection is limited to filesystem state at scan
  time. Portable or ephemeral agent usage between scans will not appear in
  inventory. Complementary network-layer detection (SASE/SWG) addresses this
  gap.
- **Tool rebrand sensitivity:** Agent name-based detection is sensitive to
  rebrands, merges, and forks. The resource catalog requires periodic review
  as the AI tooling landscape evolves.
- **Model weight file scanning:** Future work may add `files.find`-based
  detection for local model weight files (`.gguf`, `.ggml`, `.safetensors`)
  to identify downloaded LLM models on corporate hardware.
