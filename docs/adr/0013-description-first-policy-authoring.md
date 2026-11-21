# ADR-0004: Description-First Policy Authoring

**Date:** 2026-08-08
**Status:** Accepted (implemented)

## Context

Writing MQL requires specialized knowledge: cnspec resource schemas and field
names (`aws.s3.buckets.encryption.rules`), MQL syntax (`.all()`, `.none()`,
`.where()`), null/empty edge-case handling, and platform-specific details across
AWS, GCP, Azure, Kubernetes, Linux, and Windows. Security teams generally know
*what* they want to check but not *how* to express it in MQL, which creates a
bottleneck: policy authors depend on MQL experts to implement their intent.

Today a query carries both the human intent and the implementation, authored
separately and by hand:

```yaml
- uid: s3-encryption
  title: S3 buckets must be encrypted
  docs:
    desc: All S3 buckets must have server-side encryption enabled using AES-256 or KMS.
  mql: |
    aws.s3.buckets.all(encryption.rules.all(
      applyServerSideEncryptionByDefault.sseAlgorithm == "AES256" ||
      applyServerSideEncryptionByDefault.sseAlgorithm == "aws:kms"))
```

The `desc` and the `mql` drift apart, the MQL can silently contradict the
description, and new team members cannot easily verify correctness.

A prior RFC proposed embedding an LLM client (Anthropic SDK, API keys) inside
cnspec and routing schema lookup and validation through a **cnspec MCP server**.
Investigation of the codebase found that the MCP server does not exist, that
everything it would expose already exists as deterministic CLI commands
(`cnspec providers resources … --json`, `cnspec run … --ast`), and that cnspec
already ships an `mql` agent skill (`skills/mql/SKILL.md`). The right move is not
to reimplement an agent loop with a raw SDK, but to build the tools well and let
capable coding agents (which users already run) do the generation.

This ADR makes the two intended use cases **explicit**, because they have
different drivers and must both be first-class:

- **UC1 — Coding agents generate policies using cnspec's tools.** The agent
  (Claude Code, Codex, Kimi 3, DeepSeek, or any future one) is the driver. cnspec
  provides everything the agent needs to be effective: schema discovery, MQL
  validation, similar-query search, and a maintained skill.
- **UC2 — cnspec drives the user through the policy-writing journey.** cnspec is
  the driver. `cnspec policy generate` walks the user through choosing a
  **target**, stating **what will be tested**, **searching similar existing
  queries** to ground generation, and producing validated MQL — using an agent
  CLI as its backend under the hood.

The key design property: **both use cases consume one shared tool layer.** UC1 is
that layer exposed to an external agent; UC2 is cnspec orchestrating the same
layer for a human. We build the tools once.

## Decision

### 1. Support both use cases explicitly, over one shared tool layer

```
                    ┌───────────────────────────────────────┐
   UC1: agent  ───► │  Shared tool layer                    │
   drives          │   • schema discovery                   │
                    │   • MQL validation (compile + assert)  │ ◄─── UC2: cnspec
   UC2: cnspec ───► │   • similar-query search               │      drives user
   drives          │   • maintained mql skill               │      + agent backend
                    └───────────────────────────────────────┘
```

Neither use case gets a private mechanism. Anything UC2's guided flow needs is a
command/tool that UC1's agents can also call, and vice versa. This keeps the two
consistent and avoids a second, divergent code path.

Common to both: cnspec ships **no** LLM SDK, **no** API-key handling, and **no**
MCP server for this. When a model call is needed it is delegated to an agent CLI
the user already configured. Scan-time execution is untouched and fully
deterministic — generation is a dev-time step producing committed MQL.

### 2. The shared tool layer

| Capability | Command | Status |
|------------|---------|--------|
| List resources for a provider | `cnspec providers resources <provider> --json` | ships today |
| Inspect a resource's fields/types | `cnspec providers resources <provider> <resource> --json` | ships today |
| Validate MQL compiles against schema | `cnspec run <conn> -c "<mql>" --ast` (exit 1 on bad resource/field) | ships today |
| Execute-and-assert (semantics, §5) | `cnspec run <conn> -c "<mql>"` against a fixture | new gate (§5) |
| Search similar existing queries (§4) | `cnspec policy graph search "<intent>" <path> --similar [--provider <p>]` | new (§4) |
| Lint / format a bundle | `cnspec policy lint … -o sarif` / `cnspec policy format …` | ships today |
| MQL know-how + correctness rules | `skills/mql/SKILL.md` | ships; needs updates (§6) |

The two genuinely new build items are the **execute-and-assert** gate (§5) and
**similar-query search** (§4). Everything else already ships.

### 3. UC1 — cnspec as tool provider for coding agents

The agent's own harness is the driver; cnspec is the toolbox. The **contract is
the `mql` skill**: it documents every tool above and the MQL correctness rules,
and is surfaced to each agent through its native entry file — same content,
different front door:

| Agent | Entry point |
|-------|-------------|
| Claude Code | `CLAUDE.md` + `skills/mql/SKILL.md` (plugin, auto-loads on MQL intent) |
| Codex | `agents/AGENTS.md` → `skills/mql/SKILL.md` |
| Kimi 3 | `AGENTS.md` (agentic-coding convention) |
| DeepSeek | `AGENTS.md` / explicit skill path |

Because the contract is "read a skill, run a CLI," UC1 works for any repo-aware
agent; the four above are the supported/tested set. No cnspec orchestration
process is involved in UC1 — the agent calls the commands directly.

### 4. UC2 — cnspec drives the policy-writing journey

`cnspec policy generate <bundle.mql.yaml>` (interactive) walks the user through
the journey and orchestrates a coding-agent CLI as its generation **backend**:

1. **Target.** Pick the provider/platform the check applies to. cnspec resolves
   this to the check's `filters:` (`asset.platform == "..."`) — the authoritative
   provider signal — rather than guessing from the UID.
2. **Intent.** Capture *what will be tested* from `title` + `docs.desc` (or
   prompt the user for a missing `desc`).
3. **Search similar.** Query the existing corpus (`content/*.mql.yaml`, 68
   policies today) for checks with similar intent/target via
   `cnspec policy graph search --similar`, and feed the top matches to the backend as grounding
   examples. This is the highest-leverage accuracy lever — real, validated MQL
   for the same provider beats generating from schema alone. It builds on the
   existing `policy-graph search` (`apps/cnspec/cmd/policy_graph.go:212`);
   whether structural search suffices or we add semantic/embedding search is an
   open question (§ below; no embedding libs present today).
4. **Generate + validate.** cnspec hands the backend the intent, the resolved
   target's schema, the similar-query examples, and the skill; the backend runs
   its own validate/fix loop calling the shared commands (`--ast`,
   execute-and-assert); cnspec **owns the write-back** (sets `q.Mql`, runs
   `policy format`).

**Pluggable backends.** cnspec talks to agent CLIs through a small adapter
interface; each adapter knows one CLI's headless invocation.

```go
// generate/backend.go
type AgentBackend interface {
    Name() string
    Available() bool                              // CLI installed + authed?
    Generate(ctx context.Context, t GenTask) (GenResult, error)
}
```

| Backend | Headless invocation | Selected by |
|---------|---------------------|-------------|
| Claude Code | `claude -p …` | `--agent claude` (default if present) |
| Codex | `codex exec …` | `--agent codex` |
| Kimi 3 | Kimi CLI, non-interactive | `--agent kimi` |
| DeepSeek | DeepSeek CLI, non-interactive | `--agent deepseek` |

```
cnspec policy generate <bundle.mql.yaml> [flags]
  --in-place        modify the input file (default: stdout)
  --dry-run         preview without writing
  --force           regenerate queries that already have mql
  --agent string    backend: claude | codex | kimi | deepseek (default: auto)
  --model string    model passed through to the agent CLI
  --explain         include the agent's reasoning per query
  --non-interactive skip the guided prompts (batch fill empty mql)
```

Per query: empty `mql` → generate; else skip (unless `--force`). Existing MQL is
never touched without `--force`. No schema change — `Mquery` already has
`Title`, `Docs.Desc`, `Mql` (`policy/cnspec_policy.pb.go:1616-1628`).

### 5. Validation must go past "it compiles"

Compilation (`--ast`) is necessary but **far** from sufficient. `cnspec/CLAUDE.md`
is largely a catalog of MQL that compiles cleanly and returns the *wrong*
verdict: `null && null` is **`true`** (a check passes when both operands never
resolved); a dotted path that is also a resource name compiles to a fieldless
husk where every field reads `null` and `null != "x"` is `true`; `.all()` on
`null` errors; `!= ""` is not null-safe (`!= empty` is).

An LLM backend will hit all of these and a compile check catches none. So the
shared layer includes an **execute-and-assert** gate: run the query against a
recorded/fixture asset and confirm it *distinguishes* compliant from
non-compliant. A query returning `true` for both is rejected regardless of
compilation. This is the main correctness gate for both use cases and is the
highest-leverage item alongside similar-query search.

Framing: this feature is **agent-assisted drafting with mandatory expert
review**, not a replacement for MQL expertise.

### 6. Skill updates (serves both use cases)

The skill is the UC1 contract and the UC2 backend prompt, so it is shipped
product kept in sync with `CLAUDE.md`:

- Demote MCP to an optional footnote; make the deterministic CLI the single
  documented path (`skills/mql/SKILL.md`, `skills/mql/README.md`,
  `skills/README.md`, `skills/mql/.claude-plugin/plugin.json`, `agents/AGENTS.md`).
- Add the generation workflow (§4), the execute-and-assert gate, and
  similar-query search usage.
- Expand "Anti-Patterns" into a real **correctness-traps** reference from
  `cnspec/CLAUDE.md` (null three-valued logic, dotted-path husks, null-safe
  emptiness, `.all()`-on-null, newline-as-AND, `filters:` is selection not logic).
- Add provider-selection-from-`filters:` and variants guidance
  (`content/CLAUDE.md`: variant siblings differ in strictness; don't unify by
  copying one body).

## Implementation

Shipped in this change (all in the cnspec repo; no mql or schema changes):

- `internal/generate/` — the shared, backend-agnostic core:
  - `backend.go` — `AgentBackend` interface and CLI adapters for claude / codex /
    kimi / deepseek, with PATH-based availability detection, per-backend
    `CNSPEC_AGENT_*_BIN` overrides, and auto-selection.
  - `response.go` — defensive parsing of agent output (fenced ```json, bare
    balanced objects, ```mql fallback).
  - `corpus.go` + `text.go` — the example corpus and BM25 similarity search
    with provider bias (backs both `policy graph search --similar` and generation grounding).
  - `target.go` — provider resolution from `filters:` (with hyphenated-platform
    and UID fallbacks).
  - `validate.go` + `props.go` — in-process `mqlc.Compile` gate mirroring the
    linter, plus the execute-and-assert gate (`QueryRunner` + `executeValidator`).
    A validator is handed a `ValidationRequest` (query, props, target provider),
    not a bare string: `props.<name>` only compiles when the props travel with
    the query, and the target provider is what lets the gate say "that provider
    is not installed" instead of blaming the query for an unknown resource. The
    gate also asserts the query answers a verdict (bool, a scoring `switch`, or a
    block that asserts) rather than returning data.
  - `execrunner.go` — `RuntimeRunner` (executes a query against a connected
    provider runtime via `exec.ExecuteCode`) and `ConnectTarget` for the
    `--test-target` flag.
  - `prompt.go` — prompt assembly, including the correctness-trap rules.
  - `generator.go` — the per-check skip/generate/validate/retry loop.
- `internal/bundle/generate_io.go` — `AllQueries` / `QueryDesc` /
  `QueryFilterStrings` / `VariantParents`, the comment-preserving read/write seam
  reusing `ParseYaml` + `FormatBundle`.
- `internal/textrank/` — the shared BM25 ranker (tokenizer + stemmer + index)
  used by both the generation grounding search and `policy graph search
  --similar`. There is no separate `policy search` command: relevance ranking is
  a `--similar` mode on the existing `policy graph search`, so structural
  navigation and semantic ranking live under one verb.
- `internal/generate/harness.go` + `testdata/harness_cases.yaml` — the
  intent→MQL generation-quality harness: golden cases (intent + expectation) run
  by `RunCases`, deterministic offline with a scripted backend (regression guard)
  and reusable against a real agent to measure a model.
- `apps/cnspec/cmd/policy_generate.go` — the `cnspec policy generate` command (UC2
  driver): batch fill-in over bundle files.
- `apps/cnspec/cmd/policy_generate_interactive.go` — `--interactive` (`-i`): the
  guided authoring flow (describe → guess target/filter → show grounding →
  generate → accept/edit/regenerate → write one check at a time). This is the
  defined end-to-end experience for authoring from scratch.
- `apps/cnspec/cmd/policy_graph.go` — the shared grounding tool: `--similar`
  relevance ranking added to `cnspec policy graph search`.
- `skills/mql/` and `agents/AGENTS.md` — MCP demoted to the CLI path; generation
  workflow and correctness-traps reference added (the UC1 contract).
- Tests: `internal/generate/generate_test.go`,
  `internal/bundle/generate_io_test.go`.

Execute-and-assert ships opt-in via `--test-target <provider>` (live) or
`--test-recording <file>` (reproducible, offline — the recording provider is
built in, so no live credentials or provider binary are needed): the generated
query must compile AND run, resolving to a concrete true/false verdict (a null
result — the null-unsafe / dotted-path-husk class — is rejected). Default remains
compile-only.

Variant leaves inherit their intent (title/description) from the parent that
declares them, so per-platform variants generate correctly; parents are skipped.
Parameterized `props` are passed to the agent with name/type/description so it
references `props.<name>` instead of hardcoding literals. Similar-query ranking
is BM25 with field weighting (title > description > MQL) and light stemming;
embedding-based semantic search is intentionally not added, to keep cnspec free
of a model/embedding dependency (the same principle that keeps the LLM out of the
binary).

## Security considerations

- **Untrusted policy text is prompt-injection input.** A check's `title` and
  `docs.desc` become the prompt sent to the coding-agent CLI, and that agent can
  run tools and commands. Generating from an untrusted third-party bundle is
  therefore equivalent to feeding attacker-controlled text to your agent. The
  command documents this and it is a trust boundary users must respect; we do not
  sandbox the agent (it is the user's own tool, invoked with their environment).
- **The trust boundary is the working directory, not just the bundle.** Two of
  the agent's inputs are resolved relative to the current directory rather than
  to the install: the grounding corpus defaults to `./content` when present, and
  `findSkills()` (`apps/cnspec/cmd/policy_generate.go`) probes `./skills` and
  `../skills` *before* the executable's own directory, so a `skills/mql/SKILL.md`
  planted in the working tree is handed to the agent as instructions in
  preference to the shipped one. That is an injection route with no bundle
  involved, so "only generate from bundles you trust" understates it: the
  accurate guidance is to run `policy generate` only in a directory you trust.
  Pinning skills to the install directory and requiring `--corpus` to be explicit
  would close it; that is deferred, and the help text states the boundary as it
  actually stands.
- **cnspec adds no credentials, and forwards only an allowlisted environment.**
  cnspec holds no model credentials and stores none; the agent uses its own
  configured auth. It does not follow that the agent sees nothing sensitive: a
  subprocess inherits its parent's environment unless told otherwise, so the
  agent was being handed every cloud credential in the operator's shell —
  `AWS_SECRET_ACCESS_KEY`, `MONDOO_API_TOKEN`, `SSH_AUTH_SOCK` and the rest — to
  a program that runs tools on text an untrusted bundle supplied. The backend
  now builds the child's environment from an allowlist (`internal/generate/env.go`):
  what the process needs to start (PATH, HOME, TMPDIR, terminal and locale,
  proxy and CA settings) and the agent's own auth (`ANTHROPIC_*`, `OPENAI_*`,
  `CODEX_*`, `KIMI_*`, `MOONSHOT_*`, `DEEPSEEK_*`), and nothing else.
  `CNSPEC_AGENT_ENV=NAME,OTHER` forwards more by name, and `CNSPEC_AGENT_ENV=*`
  restores the old inherit-everything behavior for whoever needs it. The agent
  still runs in the operator's working directory, because inspecting the bundle
  it is authoring is the point. The prompt is passed as a subprocess argument (no
  shell), so there is no shell-injection surface from prompt content.
- **The agent's output is model-authored text on a terminal.** The
  accept/edit/regenerate loop is the control this design leans on, and a terminal
  renders control characters rather than showing them: a returned query carrying
  `ESC[2K ESC[1G` compiled fine and *displayed* as a harmless one-liner while
  containing something else. Everything the agent returns — MQL, explanation, and
  its own stdout/stderr quoted into errors — goes through
  `generate.SanitizeModelText`, which escapes C0/C1 controls, DEL, bidi overrides
  and invalid UTF-8 into visible `\xNN` / `\uNNNN` form (newline and tab survive,
  since MQL uses them and they cannot repaint a line).
- **The agent subprocess is bounded.** It runs in its own process group with a
  `WaitDelay`, so a timeout kills the tree rather than blocking on a forked
  helper that still holds the stdout pipe, and its output is capped (8 MiB
  stdout, 256 KiB stderr) with the overflow reported rather than parsed.
- **`--test-target` executes agent-generated MQL against a live system before a
  human reviews it.** Most MQL reads state, but the `os` provider exposes
  `command()`/`file()` which run on the target — so a prompt-injected query could
  execute there. It is opt-in, off by default, prints a warning, and should only
  be used with trusted bundles. `--test-recording` replays a recording instead
  and executes nothing on a live system — the safe default for validation.
  `--no-validate` is rejected together with either flag so validation can't be
  silently dropped.
- **Write-back reformats the whole file** through the same formatter as
  `cnspec policy format` (comment-preserving, canonical layout); it carries the
  same round-trip contract and is intended to run under version control.

## Consequences

**Positive**
- Two use cases, one tool layer — no divergent second code path; UC1 and UC2 stay
  consistent by construction.
- Scan-time stays deterministic and model-free.
- cnspec takes on no LLM SDK, no API keys, no MCP server; the model call is
  delegated to a CLI the user already authed.
- Reuses shipping commands; UC1 is largely functional today once the skill is
  updated.
- Similar-query search grounds generation in 68 policies of real, validated MQL —
  the biggest accuracy win available.
- No schema change.

**Negative / risks**
- UC2 adds a runtime dependency: a supported agent CLI installed and authed.
  Fail fast with a clear message when none is `Available()`. UC1 has no such
  dependency (the agent brings its own model).
- Output quality varies by backend; keep an intent→expected-MQL regression corpus
  per backend and per skill change.
- Correctness rests on skill quality + execute-and-assert, not a compiler
  guarantee; a stale skill silently degrades both use cases.
- Non-determinism between runs; mitigated by skip-unless-`--force`.
- No automatic staleness detection when `desc` changes but `mql` doesn't; if
  wanted later, a `tags: cnspec.io/generated: <hash>` — additive, out of scope.

## Alternatives considered

- **Embed an LLM client + build a cnspec MCP server** (prior RFC). Rejected: MCP
  server doesn't exist and would wrap capabilities the binary already calls
  directly; forces a cloud LLM dependency and key management into a security CLI.
- **One use case only.** Supporting only UC1 abandons non-agent users; only UC2
  ignores the reality that many users already author inside a coding agent.
  Both are real; the shared tool layer makes supporting both cheap.
- **Agent edits YAML in place** (UC2) instead of cnspec write-back. Gives up
  canonical formatting and deterministic file handling; revisit if structured
  per-uid return proves awkward.
- **Compile-only validation.** Insufficient — the hardest MQL bugs compile clean.
- **New `intent:` field / separate source files / `_compiled` metadata.**
  Schema/layout churn for marginal benefit over `title` + `docs.desc` (+ `tags`).

## Resolved during implementation

1. **Similar-query search depth.** BM25 over a field-weighted term bag (title >
   description > MQL) with light stemming, in a shared `internal/textrank`
   package. Exposed as `policy graph search --similar` (relevance ranking beside
   the existing identifier navigation — no duplicate `policy search` command) and
   reused for generation grounding. Embedding/semantic search is deliberately
   declined to avoid a model dependency.
4. **Regression harness.** `internal/generate/harness.go` runs an intent→MQL
   golden corpus; deterministic offline with a scripted backend, reusable against
   a real agent to measure a model.
2. **Execute-and-assert fixtures.** Both a live target (`--test-target`) and the
   built-in recording provider (`--test-recording`) are supported; the recording
   path is reproducible and needs no live credentials.
3. **Variants & props.** Variant leaves inherit parent intent and generate;
   parents are skipped. Props are surfaced to the agent by name/type/description.

## Open questions

1. **Backend invocation contract.** Exact headless flags and prompt shape per CLI
   (`claude -p`, `codex exec`, Kimi, DeepSeek); how each is pointed at the skill;
   how structured per-uid MQL is returned and parsed. (Adapters localize this and
   are overridable via `CNSPEC_AGENT_*_BIN`.)
2. **UC2 UX when no backend is available.** Detection, messaging, recommended
   default agent.
3. **Staleness detection.** Optional `tags: cnspec.io/generated: <hash>` to flag
   when `desc` changed but `mql` didn't.
