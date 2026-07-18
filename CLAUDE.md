# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

cnspec is an open-source, cloud-native security and policy project that assesses infrastructure security and compliance. It finds vulnerabilities and misconfigurations across cloud environments, Kubernetes, containers, servers, SaaS products, and more.

**cnspec is built on top of mql** (`go.mondoo.com/mql/v13`). mql provides the MQL query engine, provider system, and resource framework; cnspec adds policy evaluation, scoring, compliance frameworks, and security assessments.

## Where things live

- **`apps/cnspec/cmd/`** — CLI entry point and commands (scan, shell, bundle, etc.).
- **`policy/`** — policy engine core (resolution, execution, scoring). See `policy/CLAUDE.md` for engine internals, scanning flow, and protobuf/gRPC patterns.
- **`content/`** — default security policies (`*.mql.yaml`). See `content/CLAUDE.md` for policy authoring rules (variants, compliance tags, MQL, validation).
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
make test/lint/content   # Lint policy and querypack files
make benchmark/go        # Benchmarks

# Single test
go test -v ./policy -run TestSpecificTest
go test -v ./policy/...
go test -race ./...
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

### MQL gotchas that cause false positives

- **No parenthesized grouping.** MQL does **not** support `()` to group boolean expressions — `( a == 1 ) || ( b > 0 && b <= 5 )` fails to compile (`expected operand, got token "("`). Do **not** suggest adding parens "for clarity." Authors rely on `&&` > `||` precedence, or split into separate `.any()`/`.all()` calls.
- **`.all()` / `.none()` on `null` fails; on `[]` it passes vacuously.** An absent HCL/map key (e.g. `values['x']` when `x` is missing) is `null`, not an empty list, so `values['x'].all(...)` *errors* when the block is absent. Don't recommend rewriting `blocks.where(type=='x').all(y)` (vacuously true when nothing matches) into `values['x'].all(y)` — it silently flips absent-case behavior. The null-safe form is `values['x'] == empty || values['x'].all(y)`.
- **`!= ""` is not null-safe.** `"" == empty` is true, but `null != ""` is also true — use `!= empty` for null-safe non-empty assertions.
- **`filters:` is asset selection, not logic.** `filters:` only selects which assets a check applies to (`asset.platform == "..."`). Predicate logic (`field != empty`, `flag == true`, …) belongs in `mql:`. Don't flag a query for "duplicating" a condition that legitimately lives in `mql:` rather than `filters:`, and don't suggest lifting predicates into `filters:` — that silently drops assets from scoring. Multi-line `filters:` join with explicit `&&`; multi-line `mql:` uses newline-as-AND.
- **Newline-as-AND in `mql:`.** Multiple lines in an `mql:` block are implicitly AND-ed. A query that "looks like it ignores" an earlier line is usually relying on this — verify before flagging.
- **`null && null` is `true`.** MQL boolean logic is three-valued — two null operands `&&`-ed yield `true`, not null or false. A check like `field_a == "x" && field_b == "y"` silently *passes* when both fields are absent (each comparison is null, and `null && null` is true), scoring the asset compliant on data that never resolved. Assert presence first (`field_a != empty`) before comparing, or the check reports a false pass.

### Policy/content specifics

`content/` holds the security policies (`*.mql.yaml`). See `content/CLAUDE.md` for authoring rules (variants, compliance tags, MQL idioms, validation) and `policy/CLAUDE.md` for engine internals. When reviewing content:

- **Compliance tags** — never assume a `compliance/*` tag is correct because a neighboring check has it. Verify the control against the framework text before flagging or endorsing.
- **Variant siblings are not interchangeable** — the Terraform/HCL version of a check is often stricter than the plan/state version; don't recommend unifying them by copying one body into another.

## Resources

- [cnspec Documentation](https://mondoo.com/docs/cnspec)
- [MQL Documentation](https://mondoo.com/docs/mql) · [Built-in Functions](https://mondoo.com/docs/mql/functions) · [Resources by Provider](https://mondoo.com/docs/mql/resources)
- [MQL operator precedence](https://github.com/mondoohq/mql/blob/main/mqlc/parser/operators.go#L11) — reference for operator precedence during policy reviews
- [Policy Authoring Guide](https://mondoo.com/docs/cnspec/write-policies/write-intro)
- [mql Repository](https://github.com/mondoohq/mql)
- [Full Mondoo Docs (LLM-friendly text)](https://mondoo.com/docs/llms-full.txt)
