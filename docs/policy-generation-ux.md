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
| **Intent** | `title` (+ optional `docs.desc`) | Captures it (prompts for it in `--interactive`) | Natural-language intent |
| **Target** | Confirm/adjust the provider | Guesses the provider from the title and proposes the asset `filter` | `provider` + `filters:` |
| **Grounding** | — | Ranks similar existing checks (BM25 over `./content`) and shows/feeds the top matches | Few-shot examples |
| **Generate** | — | Builds a prompt (intent + schema hints + examples + correctness rules) and calls your agent CLI | Candidate MQL |
| **Validate** | — | Compiles the MQL in-process; optionally executes it and requires a true/false verdict | Pass/fail + errors |
| **Review** | `accept` / `edit` / `regenerate with feedback` | Applies feedback, re-validates edits | Approved MQL |
| **Ship** | — | Writes the check into the bundle (atomic, formatting preserved); points you at `lint` | A committed check |

## Three ways to start

The stages are the same; only the driver differs.

### 1. Guided (the defined default) — `cnspec policy generate --interactive`
An interactive session, one check at a time. cnspec asks what to verify, guesses
the target, shows grounding, generates, and walks you through
accept / edit / regenerate before writing. Best for authoring from scratch.

### 2. Batch — `cnspec policy generate <bundle> --in-place`
You pre-write checks with `title` + `docs.desc` but no `mql`; cnspec fills them
all in. Best for backfilling a bundle or a directory of policies.

Each check reports as it goes (`… generating`, `↻ attempt 2/3, previous answer
rejected: …`, `✓ generated`), so a long run never sits silent. A check with no
asset filter of its own, in a group with none, gets its provider's default
filter, and the run says so (`added asset filter asset.family.contains("linux")`):
without one, the check would run against every asset a scan reaches. When a
check fails, its last candidate is printed, never written, so you can fix a
near-miss by hand. A failure several checks share (a provider that is not
installed, say) is spelled out once and referenced after that.

### 3. Agent-driven — you're already in Claude Code / Codex
You don't call `generate`; your agent does, using cnspec as the toolbox
(`cnspec providers resources`, `cnspec policy graph search --similar`,
`cnspec run --ast`) guided by the `mql` and `policy-graph` skills.

## Worked example

**Requirement:** "S3 buckets must be encrypted." (e.g. a CIS AWS control.)

```
$ cnspec policy generate --interactive
Bundle file to write checks to [policy.mql.yaml]: aws-s3.mql.yaml

── New check ──────────────────────────────
What should this check verify?: S3 buckets must be encrypted
More detail (optional): All S3 buckets need server-side encryption
Target provider (aws, gcp, azure, os, k8s, …) [aws]:   ← guessed from the title
Asset filter [asset.platform == "aws"]:     ← proposed from the provider

Similar existing checks (used as grounding):
  • mondoo-aws-security-s3-bucket-server-side-encryption-enabled-aws
      aws.s3.bucket.encrypted == true

Generating…
  … s3-buckets-must-be-encrypted: generating

Generated MQL:
    aws.s3.buckets.all(encryption != empty)
  [a]ccept, [e]dit, [r]egenerate with feedback, [s]kip [a]: a
Check UID [s3-buckets-must-be-encrypted]:
  ✓ added s3-buckets-must-be-encrypted to aws-s3.mql.yaml
Add another check? [Y/n]: n
```

A bundle file name without `.mql.yaml` gets the suffix, and an existing file
that cnspec cannot parse is reported at this first prompt, before any agent run.

**Output** — written into the bundle, ready to `lint`, `scan` and commit:

```yaml
policies:
  - uid: aws-s3
    name: Aws S3
    version: 1.0.0
    require:
      - provider: aws
    groups:
      - title: Generated checks
        checks:
          - uid: s3-buckets-must-be-encrypted
            title: S3 buckets must be encrypted
            filters: asset.platform == "aws"
            mql: aws.s3.buckets.all(encryption != empty)
            docs:
              desc: All S3 buckets need server-side encryption
```

The check goes into a policy group, not the top-level `queries:` block. A query
that belongs to no policy lints with a `query-unassigned` warning and then fails
`cnspec scan --policy-bundle` with *"a policy or framework mrn is required"* —
so a bundle that only lints is not enough. When the file already has a policy,
the check joins it (in a group of its own, so it does not inherit a group filter
that was written for other checks).

The policy also gets `require: [{provider: aws}]`, which is what lets
`cnspec scan --policy-bundle` install the provider the check needs, and what
lint's `policy-missing-require` rule asks for.

Then: `cnspec policy lint aws-s3.mql.yaml` → `cnspec scan aws --policy-bundle
aws-s3.mql.yaml` → commit.

## Validation depth (you choose how sure to be)

| Mode | Flag | What it proves | Runs on live infra? |
|------|------|----------------|---------------------|
| Compile (default) | — | Resources/fields exist, syntax valid | No |
| Execute-and-assert (recording) | `--test-recording <file>` | Query resolves to a concrete true/false verdict | No (replay) |
| Execute-and-assert (live) | `--test-target <provider>` | Same, against a real asset | **Yes** — warns first |

Compilation alone can't catch the semantic traps (`null && null == true`,
dotted-path husks); execute-and-assert can. Prefer `--test-recording` for
reproducible, credential-free validation.

## The target provider

Validation compiles against the provider that defines the check's resources, so
the target has to be known and installed before the agent runs. cnspec takes it
from, in order: the provider you confirmed in the guided flow, the check's
filters, its uid, and the words of its title and description (whole words only,
so "passwords" does not mean AWS RDS). A missing provider is installed the way
`cnspec scan` installs one, unless `--auto-update=false` pins the installed set;
if it cannot be installed, the check fails before any agent call, with the
install command. When nothing names a target, the provider of the generated
query's first resource is used for the asset filter.

## Guardrails

- **Human-in-the-loop.** Generated MQL is a reviewed, committed diff — never
  applied silently, never run at scan time. A prompt that cannot be answered
  ends the session: at EOF (Ctrl-D, a closed pipe, a script that ran out of
  answers) the wizard exits instead of taking its own default, so a
  non-interactive run can never "accept" on your behalf.
- **What you review is what is written.** Model output is rendered through the
  same normalization that writes it into the bundle, so an ANSI escape sequence
  in a generated query cannot repaint the review line and hide part of the query.
- **Trusted input only.** A check's description becomes the agent's prompt, and
  the agent can run tools; only generate on bundles you trust.
- **No lock-in.** cnspec ships no LLM SDK and stores no keys; it drives the
  coding-agent CLI you already have (`--agent claude|codex|kimi|deepseek`).

See `docs/adr/0013-description-first-policy-authoring.md` for the design
rationale.
