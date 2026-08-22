# Policy Generation: From Requirement to Check

This document defines the user experience for turning a security **requirement**
(what you want to check) into a validated MQL **check** (what cnspec runs),
without writing MQL by hand.

## The promise

> Describe a check in plain language. cnspec generates the MQL, grounds it in
> your existing policies, validates it, and lets you review before it's written.
> The result is ordinary, committed MQL — scans never call a model.

You bring the *intent*; cnspec produces the *implementation* and you approve it.

## The journey

```
  REQUIREMENT            INTENT              TARGET            GROUNDING
  "S3 buckets    ──►  title + desc    ──►  provider +   ──►  similar
   must be              (plain language)    filter            existing checks
   encrypted"                               (auto-guessed)    (BM25 over content)
                                                                    │
                                                                    ▼
     SHIP        ◄──   REVIEW        ◄──   VALIDATE      ◄──   GENERATE
  lint + commit        accept /            compile             agent CLI writes
  + scan               edit /              (+ optional          candidate MQL
                       regenerate          execute-assert)
```

Every stage has a defined input, a defined cnspec action, and a defined output.

| Stage | You provide | cnspec does | Output |
|-------|-------------|-------------|--------|
| **Requirement** | The thing to check, in your head or a compliance control | — | A one-line intent |
| **Intent** | `title` (+ optional `docs.desc`) | Captures it (prompts for it in `-i`) | Natural-language intent |
| **Target** | Confirm/adjust the provider | Guesses the provider from the title and proposes the asset `filter` | `provider` + `filters:` |
| **Grounding** | — | Ranks similar existing checks (BM25 over `./content`) and shows/feeds the top matches | Few-shot examples |
| **Generate** | — | Builds a prompt (intent + schema hints + examples + correctness rules) and calls your agent CLI | Candidate MQL |
| **Validate** | — | Compiles the MQL in-process; optionally executes it and requires a true/false verdict | Pass/fail + errors |
| **Review** | `accept` / `edit` / `regenerate with feedback` | Applies feedback, re-validates edits | Approved MQL |
| **Ship** | — | Writes the check into the bundle (atomic, formatting preserved); points you at `lint` | A committed check |

## Three ways to start

The stages are the same; only the driver differs.

### 1. Guided (the defined default) — `cnspec policy generate -i`
An interactive session, one check at a time. cnspec asks what to verify, guesses
the target, shows grounding, generates, and walks you through
accept / edit / regenerate before writing. Best for authoring from scratch.

### 2. Batch — `cnspec policy generate <bundle> --in-place`
You pre-write checks with `title` + `docs.desc` but no `mql`; cnspec fills them
all in. Best for backfilling a bundle or a directory of policies.

### 3. Agent-driven — you're already in Claude Code / Codex
You don't call `generate`; your agent does, using cnspec as the toolbox
(`cnspec providers resources`, `cnspec policy graph search --similar`,
`cnspec run --ast`) guided by the `mql` and `policy-graph` skills.

## Worked example

**Requirement:** "S3 buckets must be encrypted." (e.g. a CIS AWS control.)

```
$ cnspec policy generate -i
What should this check verify?: S3 buckets must be encrypted
More detail (optional): All S3 buckets need server-side encryption
Target provider [aws]:                      ← guessed from the title
Asset filter [asset.platform == "aws"]:     ← proposed from the provider

Similar existing checks (used as grounding):
  • mondoo-aws-security-s3-bucket-server-side-encryption-enabled-aws
      aws.s3.bucket.encrypted == true

Generated MQL:
    aws.s3.buckets.all(encryption != empty)
  [a]ccept, [e]dit, [r]egenerate with feedback, [s]kip [a]: a
  ✓ added s3-buckets-must-be-encrypted to aws-s3.mql.yaml
```

**Output** — written into the bundle, ready to `lint` and commit:

```yaml
queries:
  - uid: s3-buckets-must-be-encrypted
    title: S3 buckets must be encrypted
    filters: asset.platform == "aws"
    mql: aws.s3.buckets.all(encryption != empty)
    docs:
      desc: All S3 buckets need server-side encryption
```

Then: `cnspec policy lint aws-s3.mql.yaml` → commit → `cnspec scan aws`.

## Validation depth (you choose how sure to be)

| Mode | Flag | What it proves | Runs on live infra? |
|------|------|----------------|---------------------|
| Compile (default) | — | Resources/fields exist, syntax valid | No |
| Execute-and-assert (recording) | `--test-recording <file>` | Query resolves to a concrete true/false verdict | No (replay) |
| Execute-and-assert (live) | `--test-target <provider>` | Same, against a real asset | **Yes** — warns first |

Compilation alone can't catch the semantic traps (`null && null == true`,
dotted-path husks); execute-and-assert can. Prefer `--test-recording` for
reproducible, credential-free validation.

## Guardrails

- **Human-in-the-loop.** Generated MQL is a reviewed, committed diff — never
  applied silently, never run at scan time.
- **Trusted input only.** A check's description becomes the agent's prompt, and
  the agent can run tools; only generate on bundles you trust.
- **No lock-in.** cnspec ships no LLM SDK and stores no keys; it drives the
  coding-agent CLI you already have (`--agent claude|codex|kimi|deepseek`).

See `docs/adr/0004-description-first-policy-authoring.md` for the design
rationale.
