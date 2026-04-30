# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

cnspec is an open-source, cloud-native security and policy project that assesses infrastructure security and compliance. It finds vulnerabilities and misconfigurations across cloud environments, Kubernetes, containers, servers, SaaS products, and more.

**cnspec is built on top of cnquery** (`go.mondoo.com/cnquery/v12`). cnquery provides the MQL query engine, provider system, and resource framework; cnspec adds policy evaluation, scoring, compliance frameworks, and security assessments.

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
make prep/repos          # Clone/verify cnquery dependency (required for proto compilation)
make prep/repos/update   # Update cnquery dependency
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
- When proto files reference cnquery types, ensure the cnquery repo is present via `make prep/repos`.

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

## Resources

- [cnspec Documentation](https://mondoo.com/docs/cnspec/)
- [MQL Documentation](https://mondoo.com/docs/mql/) · [Built-in Functions](https://mondoo.com/docs/mql/functions) · [Resources by Provider](https://mondoo.com/docs/mql/resources/)
- [Policy Authoring Guide](https://mondoo.com/docs/cnspec/write-policies/write-intro/)
- [cnquery Repository](https://github.com/mondoohq/cnquery)
- [Full Mondoo Docs (LLM-friendly text)](https://mondoo.com/docs/llms-full.txt)
