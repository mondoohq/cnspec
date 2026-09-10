# CLAUDE.md

Notes for Claude Code working in this repo. Terse on purpose.

## Overview

cnspec is built on mql (`go.mondoo.com/mql`). mql gives us the MQL query engine, the provider system, and the resource framework. cnspec adds policy evaluation, scoring, compliance frameworks, and security assessments.

## Where things live

- `apps/cnspec/cmd/` — CLI entry point and commands (scan, shell, bundle, ...).
- `policy/` — policy engine: resolution, execution, scoring. Engine internals, scanning flow, and protobuf/gRPC patterns are in `policy/CLAUDE.md`.
- `content/` — the shipped security policies (`*.mql.yaml`). `querypacks/` holds data-collection bundles that don't score. Authoring rules: `content/CLAUDE.md`. User-facing catalog: `content/README.md`.
- `content/validation/` — every test and validator that runs against those policies, plus fixtures. Reference: `content/validation/README.md`.
- `cli/` — terminal components, plus the output formats still coupled to the CLI (compact, SARIF, JUnit, JSON, CSV). `cli/reporter` also owns `PrintConfig`, the format registry, and every format's output handler.
- `reports/` — report standards, none terminal-facing. `reports/ocsf` is the OCSF schema and imports no cnspec package on purpose, so it stays extractable as `go.mondoo.com/ocsf` (`docs/adr/0005-ocsf-type-generation.md`); the cnspec mapping onto it is in `reports/ocsf/convert`. `reports/hdf` is OHDF. `reports/reportdoc` is where every format reads a check's docs and outcome.
- `internal/bundle/`, `internal/datalakes/`, `internal/lsp/` — bundle loading, storage, LSP.
- `examples/`, `test/`, `docs/`.

## Commands

### Build

```bash
make cnspec/build              # build the binary
make cnspec/install            # install to $GOBIN
make cnspec/build/linux        # cross-compile (also /linux/arm, /windows)
```

### Codegen

Run after changing `.proto` files, policy bundle structures, or reporter configs.

```bash
make prep                # install tools (first time only)
make prep/repos          # clone/verify mql (needed for proto compilation)
make prep/repos/update   # update mql
make cnspec/generate     # regenerate proto, policy, reporter code
```

### Tests

```bash
make test                # everything
make test/go             # Go only
make test/go/plain       # with coverage
make test/lint           # linter
make benchmark/go        # benchmarks
```

### Content validation

Lives in `content/validation/`. What each check proves, when CI runs it, how to run it manually: [`content/validation/README.md`](content/validation/README.md).

```bash
make test/content        # lint + bundle scans + compliance mappings
make test/content/lint   # cnspec policy lint over content/ and content/querypacks
make test/content/iac    # IaC fixture suites (slow; run when you touch a variant)
```

Most validators are allowlist-driven: a new policy is covered only once it's registered with them. Add a `*.mql.yaml` without wiring it into the variant suites and the remediation validators and the whole bundle ships unexamined, every gate green. See [Adding a policy: what to register](content/validation/README.md#adding-a-policy-what-to-register).

### Scanning and linting

```bash
cnspec scan local                      # local system
cnspec scan docker image ubuntu:22.04  # docker
cnspec scan aws                        # AWS (local AWS CLI config)
cnspec scan k8s                        # Kubernetes
cnspec scan ssh user@host              # SSH

cnspec policy lint ./content                                    # all policies
cnspec policy lint ./content/mondoo-linux-security.mql.yaml     # one policy
```

## Working in this repo

### Commits and PRs

Titles are `<emoji> <scope>: <lowercase description>`. The emoji is part of the convention, not decoration. Counts over the last 200 commits on `main`: ✨ new capability or coverage (57), 🧹 cleanup/refactor/maintenance (47), 🐛 bug fix (40), 👷 CI and automation (8), 📝 docs (3). Scope is the area, not the file: `validation`, `content`, `ci`, or a provider name like `aws` or `alibaba`.

### Stacked PRs

Squash-merging a base branch does not retarget the PRs stacked on it. The squash lands a new SHA on `main` and leaves the original branch commit orphaned but alive, so GitHub keeps the stacked PR pointed at a dead branch and will merge into it. The PR then reports `MERGED` with none of its work on `main`.

After a base branch merges, check every PR stacked on it:

```bash
git merge-base --is-ancestor <pr-merge-commit> origin/main && echo on-main || echo ORPHANED
```

Recover it: branch from `origin/main`, `git cherry-pick <pr-merge-commit>` (a squash commit has one parent, so it applies cleanly), confirm `git diff <pr-merge-commit> HEAD` is empty, open the replacement. Also diff the recovered tree against the merged one, since a base branch amended after its own squash merge strands those fixes too.

### Worktrees

Feature work happens in worktrees, and many branches are already checked out in one, which makes `git checkout <branch>` fail. Find the worktree with `git worktree list` and work on the branch in place with `git -C <worktree>`.

### Local mql development

A check often needs a provider field that doesn't exist yet. `make prep/repos` clones mql into `./mql`; `go.mod` has a commented `replace go.mondoo.com/mql => ../mql` for building against a sibling checkout. After changing a provider's `.lr` schema: regenerate, rebuild that provider, copy it into `~/.config/mondoo/providers/<name>/`. That installed copy, not the source, is what `cnspec policy lint` resolves against.

## Development rules

### Dependencies

- Banned: `github.com/pkg/errors` (use `github.com/cockroachdb/errors` and `errors.Wrap`) and `github.com/mitchellh/mapstructure` (use `github.com/go-viper/mapstructure/v2`).
- Proto files referencing mql types need the mql repo present: `make prep/repos`.

### Generated code

Never hand-edit. Regenerate with `make cnspec/generate`:

- `*.pb.go` — from proto files
- `*.ranger.go` — ranger-rpc
- `*.vtproto.pb.go` — vtproto marshaling
- `*_gen.go` — `go generate`

## Reviewing PRs (bots and automated reviewers)

For automated reviewers (mondoo-code-review, Claude) commenting on PRs here. Most false positives come from guessing how MQL behaves instead of verifying it. Before claiming a query is wrong, a field is missing, or precedence is off, check the references below. Can't verify it? Ask a question ("Does `x` exist on this resource?") instead of filing a defect.

### Verify first

Resource and field existence — don't assume something is missing, check what the provider exposes:

- [Resources by Provider](https://mondoo.com/docs/mql/resources) — resources and fields, grouped by provider (aws-pack, azure-pack, gcp-pack, core-pack, ...).
- [Built-in Functions](https://mondoo.com/docs/mql/functions) — `parse.json`, `parse.date`, `regex`, list ops (`all`, `any`, `where`, `contains`, `none`, `map`).
- [llms-full.txt](https://mondoo.com/docs/llms-full.txt) — raw dump of all docs; grep it to confirm a field or function fast.
- Locally, the *installed* schema is what lint resolves against: `~/.config/mondoo/providers/<name>/<name>.resources.json`. Source of truth in code: `providers/<name>/resources/<name>.lr` in the [mql repo](https://github.com/mondoohq/mql).
- End to end: `cnquery run <provider> -c '<mql>'` (no TTY needed) or `cnspec policy lint ./content/<file>.mql.yaml`. Run the query before claiming it returns the wrong thing.

Operator precedence is fixed; see [`mqlc/parser/operators.go`](https://github.com/mondoohq/mql/blob/main/mqlc/parser/operators.go#L11). `&&` binds tighter than `||`, so `a || b && c` already parses as `a || (b && c)`. That's usually intentional.

### Don't flag these, they're correct MQL

All verified against the compiler and explained in full in [`content/CLAUDE.md`](content/CLAUDE.md). Repeated here so a reviewer that never opens that file still doesn't file the false positive.

| Pattern | Why it's not a bug |
|---|---|
| `a == 1 \|\| b > 0 && b <= 5`, unparenthesized | MQL has no parenthesized grouping at all; `(` is rejected as an operand. `&&` binds tighter than `\|\|`, so the grouping is already what the author meant. Never suggest parens "for clarity". |
| `guard \|\| guard \|\| D && E` called "skipped", "short-circuited past", or "silently passes" | This is the guard chain, the dominant shape in `content/`. Short-circuiting decides what is *evaluated*, never the *verdict*: if `D` is false, `D && E` is false, the disjunction is false, the check fails. Build the truth table and name the row where the current form passes and a parenthesized form fails. There isn't one. Most-filed false positive on this repo. |
| A literal flagged on character count from the diff (ARN colons, missing path segment) | Rendered diffs distort spacing; don't count characters in one. Resolve the literal against its oracle and quote the output (`aws iam get-policy`, `cfn-lint`, the provider schema). AWS-managed policy ARNs have an empty account field, so `arn:aws:iam::aws:policy/...` with two colons is canonical. |
| `blocks.where(type == 'x').all(y)` where `values['x'].all(y)` looks simpler | Not equivalent. `.all()` passes vacuously on an empty list and fails on `null`, and an absent key is `null`. The rewrite flips the absent-block verdict. |
| `field != empty` rather than `field != ""` | `null != ""` is true, so `!= ""` isn't a non-empty test. `!= empty` is the null-safe form. |
| A predicate in `mql:` that "could" live in `filters:` | `filters:` is asset selection. Moving a predicate there drops assets from scoring instead of failing them. |
| Several lines in one `mql:` block | Newline is an implicit AND. A later line isn't ignoring an earlier one. |
| A `-terraform-hcl` variant stricter than its `-plan`/`-state` sibling | Usually deliberate: HCL sees author intent, plan/state see resolved values. Don't unify them by copying one body into another. |
| A `compliance/*` tag unlike a neighbouring check's | Neighbours map different control objectives. Check the framework text before flagging *or* endorsing. |

### Two real bugs that are easy to miss

`null && null` is `true`, and it's the only null combination that is:

```
m["absent"] && m["also_absent"]   -> [ok] true
m["absent"] && true               -> [failed]
m["absent"] && false              -> [failed]
m["absent"] || false              -> [failed]
```

Two bare boolean fields joined with `&&` therefore pass when neither resolved. Doesn't extend to comparisons: `null == "x"` is `false`, not null, so `field_a == "x" && field_b == "y"` fails when both are absent. Flag the bare-field form, leave the comparison form alone. The comparison form has its own asymmetry (`field != "insecure"` passes when the field is absent), covered in `content/CLAUDE.md`.

A dotted path that is also a resource name is not a field read. The compiler extends the resource path greedily, so `azure.subscription.aksService.cluster.autoUpgradeProfile.upgradeChannel` builds a bare `...cluster.autoUpgradeProfile` resource whose accessor never runs; every field reads `null` and the check answers confidently wrong. Suspect it when the value is a sub-object and the full path appears as a resource in `cnspec providers resources <provider> --json`. Confirm by running the query and looking for `provider returned no data and no error for a field ... id=` with an empty `id=`. Not Azure-specific: Cloudflare, GCP, AWS, vSphere, and Arista all have resources shaped this way. Full treatment and fix in `content/CLAUDE.md`.

## Links

- [cnspec docs](https://mondoo.com/docs/cnspec)
- [Policy authoring guide](https://mondoo.com/docs/cnspec/write-policies/write-intro)

MQL references (resource lists, functions, precedence, `llms-full.txt`) are linked in "Verify first" above.
