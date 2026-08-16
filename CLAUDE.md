# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

cnspec is an open-source, cloud-native security and policy project that assesses infrastructure security and compliance. It finds vulnerabilities and misconfigurations across cloud environments, Kubernetes, containers, servers, SaaS products, and more.

**cnspec is built on top of mql** (`go.mondoo.com/mql/v13`). mql provides the MQL query engine, provider system, and resource framework; cnspec adds policy evaluation, scoring, compliance frameworks, and security assessments.

## Where things live

- **`apps/cnspec/cmd/`** — CLI entry point and commands (scan, shell, bundle, etc.).
- **`policy/`** — policy engine core (resolution, execution, scoring). See `policy/CLAUDE.md` for engine internals, scanning flow, and protobuf/gRPC patterns.
- **`content/`** — default security policies (`*.mql.yaml`) and, under `querypacks/`, data-collection bundles that do not score. `content/CLAUDE.md` holds the authoring rules (variants, compliance tags, MQL semantics); `content/README.md` is the user-facing catalog.
- **`content/validation/`** — every test and validator that runs against those policies, plus its fixtures. `content/validation/README.md` is the definitive reference for content validation.
- **`cli/`** — reusable CLI components and reporters (SARIF, JUnit, JSON, …).
- **`internal/bundle/`, `internal/datalakes/`, `internal/lsp/`** — bundle loading, storage, LSP support.
- **`examples/`, `test/`, `docs/`** — examples, integration tests, docs.

## Essential commands

### Build & install

```bash
make cnspec/build              # Build the cnspec binary
make cnspec/install            # Install to $GOBIN
make cnspec/build/linux        # Cross-compile (also: /linux/arm, /windows)
```

### Code generation

Run after modifying `.proto` files, policy bundle structures, or reporter configurations.

```bash
make prep                # Install required tools (first time only)
make prep/repos          # Clone/verify mql dependency (required for proto compilation)
make prep/repos/update   # Update mql dependency
make cnspec/generate     # Regenerate all generated code (proto, policy, reporter)
```

### Testing

```bash
make test                # Run all tests
make test/go             # Go tests only
make test/go/plain       # With coverage
make test/lint           # Linter
make benchmark/go        # Benchmarks

# Single test
go test -v ./policy -run TestSpecificTest
go test -v ./policy/...
go test -race ./...
```

### Content validation

Everything that checks the policies in `content/` lives in `content/validation/`, and
**[`content/validation/README.md`](content/validation/README.md) is the definitive
reference** for it: what each check proves, when CI runs it, and how to run it manually.

```bash
make test/content        # lint + bundle scans + compliance mappings
make test/content/lint   # cnspec policy lint over content/ and content/querypacks
make test/content/iac    # the IaC fixture suites (slow; run when you touch a variant)
```

### Scanning & policy linting

```bash
cnspec scan local                      # Local system
cnspec scan docker image ubuntu:22.04  # Docker
cnspec scan aws                        # AWS (uses local AWS CLI config)
cnspec scan k8s                        # Kubernetes
cnspec scan ssh user@host              # SSH

cnspec policy lint ./content                                    # Lint all policies
cnspec policy lint ./content/mondoo-linux-security.mql.yaml     # Lint one policy
```

## Working in this repo

### Commits and pull requests

Commit titles are `<emoji> <scope>: <lowercase description>`, and the emoji is part of the convention rather than decoration. Across the last 200 commits on `main`: **✨** new capability or coverage (57), **🧹** cleanup, refactor, or maintenance (47), **🐛** bug fix (40), **👷** CI and automation (8), **📝** documentation (3). The scope is the area, not the file — `validation`, `content`, `ci`, or a provider name such as `aws` or `alibaba`.

### Stacked pull requests

Squash-merging a base branch does **not** retarget the PRs stacked on it. The squash lands a new SHA on `main` and leaves the original branch commit orphaned but alive, so GitHub keeps the stacked PR pointed at a dead branch and will happily merge into it. The PR then reports `MERGED` while none of its work is on `main`.

After a base branch merges, check every PR stacked on it:

```bash
git merge-base --is-ancestor <pr-merge-commit> origin/main && echo on-main || echo ORPHANED
```

To recover, branch from `origin/main` and `git cherry-pick <pr-merge-commit>` — a squash commit has a single parent, so it applies cleanly — then confirm `git diff <pr-merge-commit> HEAD` is empty before opening the replacement. Also diff the recovered tree against the merged one: a base branch amended after its own squash merge strands those fixes too.

### Worktrees

Feature work happens in worktrees, and many branches are already checked out in one. `git checkout <branch>` fails when it is, so operate on the branch in place with `git -C <worktree>` (find it with `git worktree list`) rather than trying to check it out again.

### Local mql development

A check often needs a provider field that does not exist yet. `make prep/repos` clones mql into `./mql`, and `go.mod` carries a commented `replace go.mondoo.com/mql/v13 => ../mql` for building against a sibling checkout. After changing a provider's `.lr` schema, regenerate and rebuild that provider, then copy it into `~/.config/mondoo/providers/<name>/` — that installed copy, not the source, is what `cnspec policy lint` resolves against.

## Development rules

### Dependency management

- **Forbidden packages**: do not use `github.com/pkg/errors` (use `github.com/cockroachdb/errors`) or `github.com/mitchellh/mapstructure` (use `github.com/go-viper/mapstructure/v2`).
- When proto files reference mql types, ensure the mql repo is present via `make prep/repos`.

### Error handling

Use `github.com/cockroachdb/errors`:

```go
import "github.com/cockroachdb/errors"

return errors.Wrap(err, "failed to load policy")
return errors.New("invalid policy structure")
```

### Generated code

Never edit these files manually. Regenerate with `make cnspec/generate`:

- `*.pb.go` — Generated from proto files.
- `*.ranger.go` — Generated ranger-rpc code.
- `*.vtproto.pb.go` — Optimized vtproto marshaling.
- `*_gen.go` — Generated via `go generate`.

## Reviewing pull requests (for bots & automated reviewers)

This section is for any automated reviewer (mondoo-code-review, Claude, etc.) commenting on PRs in this repo. **Most false positives come from guessing how MQL behaves instead of verifying it.** Before asserting that a query is wrong, that a field doesn't exist, or that precedence/grouping is off, confirm it against the references below. If you cannot verify a claim, frame it as a question ("Does `x` exist on this resource?"), not a defect.

### Verify before you claim

- **Resource & field existence** — Do not assume a resource or field is missing. Check what the provider actually exposes:
  - [Resources by Provider](https://mondoo.com/docs/mql/resources) — canonical list of resources and their fields, grouped by provider (aws-pack, azure-pack, gcp-pack, core-pack, …).
  - [Built-in Functions](https://mondoo.com/docs/mql/functions) — `parse.json`, `parse.date`, `regex`, list ops (`all`, `any`, `where`, `contains`, `none`, `map`), etc.
  - [Full Mondoo Docs (LLM-friendly text)](https://mondoo.com/docs/llms-full.txt) — single raw-text dump of all docs; grep it when you need to confirm a field or function quickly.
  - Locally, the *installed* provider schema is authoritative for what lint resolves against: `~/.config/mondoo/providers/<name>/<name>.resources.json`. The source of truth in code is `providers/<name>/resources/<name>.lr` in the [mql repo](https://github.com/mondoohq/mql).
  - To check a real query end to end: `cnquery run <provider> -c '<mql>'` (no TTY needed) or `cnspec policy lint ./content/<file>.mql.yaml`. **Run the query before claiming it returns the wrong thing.**
- **Operator precedence** — MQL precedence is fixed; consult [`mqlc/parser/operators.go`](https://github.com/mondoohq/mql/blob/main/mqlc/parser/operators.go#L11) before flagging precedence. Notably `&&` binds tighter than `||`, so `a || b && c` already parses as `a || (b && c)` — that is usually intentional, not a bug.

### Do not flag these — they are correct MQL

Each is verified against the compiler and explained in full in [`content/CLAUDE.md`](content/CLAUDE.md). The one-liners here exist so a reviewer that never opens that file still does not raise the false positive.

| Pattern | Why it is not a bug |
|---|---|
| `a == 1 \|\| b > 0 && b <= 5`, unparenthesized | MQL has no parenthesized grouping anywhere; `(` is rejected as an operand. `&&` binds tighter than `\|\|`, so the grouping already is what the author meant. Never suggest adding parens "for clarity". |
| `guard \|\| guard \|\| D && E` described as "skipped", "short-circuited past", or "silently passes" | This is the **guard chain**, the dominant shape in `content/`. Short-circuiting decides what is *evaluated*, never the *verdict*: if `D` is false, `D && E` is false, the disjunction is false, and the check **fails**. Before filing, build the truth table and name the row where the current form passes and a parenthesized form fails. There is no such row — this is the most-filed false positive on this repo. |
| A literal flagged on character count from the diff (ARN colons, a missing path segment) | Rendered diffs distort spacing; do not count characters in one. Resolve the literal against its oracle and quote the output (`aws iam get-policy`, `cfn-lint`, the provider schema). AWS-managed policy ARNs have an empty account field, so `arn:aws:iam::aws:policy/…` with two colons is canonical. |
| `blocks.where(type == 'x').all(y)` where `values['x'].all(y)` looks simpler | Not equivalent. `.all()` passes vacuously on an empty list and fails outright on `null`, and an absent key is `null`. The rewrite flips the absent-block verdict. |
| `field != empty` rather than `field != ""` | `null != ""` is true, so `!= ""` is not a non-empty test. `!= empty` is the null-safe form. |
| A predicate in `mql:` that "could" live in `filters:` | `filters:` is asset selection. Moving a predicate there drops assets from scoring rather than failing them. |
| Several lines in one `mql:` block | Newline is an implicit AND. A later line is not ignoring an earlier one. |
| A `-terraform-hcl` variant stricter than its `-plan`/`-state` sibling | Usually deliberate: HCL sees author intent, plan/state see resolved values. Do not recommend unifying them by copying one body into another. |
| A `compliance/*` tag unlike a neighbouring check's | Neighbours map different control objectives. Verify against the framework text before flagging *or* endorsing. |

### Two that are real bugs, and are easy to miss

**`null && null` is `true`** — and it is the only null combination that is. Verified:

```
m["absent"] && m["also_absent"]   → [ok] true
m["absent"] && true               → [failed]
m["absent"] && false              → [failed]
m["absent"] || false              → [failed]
```

So two **bare boolean fields** joined with `&&` pass when neither resolved. This does **not** extend to comparisons: `null == "x"` is `false`, not null, so `field_a == "x" && field_b == "y"` fails when both are absent. Flag the bare-field form; leave the comparison form alone. The comparison form has its own asymmetry — `field != "insecure"` passes when the field is absent — which `content/CLAUDE.md` covers.

**A dotted path that is also a resource name is not a field read.** The compiler extends the resource path greedily, so `azure.subscription.aksService.cluster.autoUpgradeProfile.upgradeChannel` builds a bare `…cluster.autoUpgradeProfile` resource whose accessor never runs; every field reads `null` and the check answers confidently wrong. Suspect it when the value is a sub-object and the full path appears as a resource in `cnspec providers resources <provider> --json`; confirm by running the query and looking for `provider returned no data and no error for a field … id=` with an **empty** `id=`. Not Azure-specific — Cloudflare, GCP, AWS, vSphere and Arista all have resources shaped this way. `content/CLAUDE.md` has the full treatment and the fix.

## Resources

- [cnspec Documentation](https://mondoo.com/docs/cnspec)
- [MQL Documentation](https://mondoo.com/docs/mql) · [Built-in Functions](https://mondoo.com/docs/mql/functions) · [Resources by Provider](https://mondoo.com/docs/mql/resources)
- [MQL operator precedence](https://github.com/mondoohq/mql/blob/main/mqlc/parser/operators.go#L11) — reference for operator precedence during policy reviews
- [Policy Authoring Guide](https://mondoo.com/docs/cnspec/write-policies/write-intro)
- [mql Repository](https://github.com/mondoohq/mql)
- [Full Mondoo Docs (LLM-friendly text)](https://mondoo.com/docs/llms-full.txt)
