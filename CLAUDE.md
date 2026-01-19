# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

cnspec is an open-source, cloud-native security and policy project that assesses infrastructure security and compliance. It finds vulnerabilities and misconfigurations across cloud environments, Kubernetes, containers, servers, SaaS products, and more.

## Essential Commands

### Build & Install

```bash
# Build cnspec binary
make cnspec/build

# Install to $GOBIN
make cnspec/install

# Build for specific platforms
make cnspec/build/linux
make cnspec/build/linux/arm
make cnspec/build/windows
```

### Code Generation

When modifying protobuf files, auto-generated files, or policy structures:

```bash
# Install required tools (first time only)
make prep

# Clone/verify cnquery dependency (required for proto compilation)
make prep/repos

# Update cnquery dependency when needed
make prep/repos/update

# Regenerate all generated code (proto, policy, reporter)
make cnspec/generate
```

**Important**: Always run `make cnspec/generate` after modifying:
- `.proto` files
- Policy bundle structures
- Reporter configurations

### Testing

```bash
# Run all tests
make test

# Run Go tests only
make test/go

# Run tests with coverage
make test/go/plain

# Run CI-friendly tests with JUnit output
make test/go/plain-ci

# Run linter
make test/lint

# Lint policy files
make test/lint/policies

# Run benchmarks
make benchmark/go
```

### Running Individual Tests

```bash
# Run a specific test
go test -v ./policy -run TestSpecificTest

# Run tests in a package
go test -v ./policy/...

# Run with race detection
go test -race ./...
```

### Policy Linting

```bash
# Lint all policies in content directory
cnspec policies lint ./content

# Lint a specific policy file
cnspec policies lint ./content/mondoo-linux-security.mql.yaml
```

## High-Level Architecture

### cnspec vs cnquery

**cnspec is built on top of cnquery** - it's not just a dependency relationship:

- **cnquery provides**: MQL query engine, provider system, resource framework, data gathering
- **cnspec adds**: Policy evaluation, scoring, compliance frameworks, security assessments, risk factors
- **Shared runtime**: Both use the same provider system (`providers.Runtime`) for connecting to target systems
- **Import path**: cnspec imports `go.mondoo.com/cnquery/v12` extensively

When working on cnspec, you may need to understand or modify cnquery components.

### Policy Engine Flow

The policy execution pipeline has four main stages:

#### 1. Bundle Loading & Compilation
- Policy files (`.mql.yaml`) loaded via `BundleLoader` in [policy/bundle.go](policy/bundle.go)
- Bundles contain: Policies, QueryPacks, Frameworks, Queries, Properties, Migrations
- MQL code compiled to executable `llx.CodeBundle` (protobuf format) via `mqlc.CompilerConfig`
- Code bundles are cached and reusable across multiple asset scans

#### 2. Policy Resolution
- Core logic in [policy/resolver.go](policy/resolver.go) and [policy/resolved_policy_builder.go](policy/resolved_policy_builder.go)
- Asset filters evaluated to determine which policies apply to each asset
- Builds dependency graph of: Policies → Frameworks → Controls → Checks/Queries
- Graph walking from non-prunable leaf nodes ensures only applicable checks run
- Generates two critical structures:
  - **ExecutionJob**: All queries to execute with checksums and code bundles
  - **CollectorJob**: Score aggregation rules with scoring systems and hierarchy

#### 3. Execution
- Orchestrated by `GraphExecutor` in [policy/executor/graph.go](policy/executor/graph.go)
- Queries executed via cnquery's `llx.MQLExecutorV2`
- Results collected through `BufferedCollector` pattern
- Handles: data points, scores, risk factors, upstream transmission

#### 4. Result Collection & Scoring
- Scores calculated via `ScoreCalculator` in [policy/score_calculator.go](policy/score_calculator.go)
- Reporting jobs aggregate child scores based on impact ratings
- Final `Report` structures contain scores, statistics, CVSS data

### Scanning Architecture

Main orchestrator: `LocalScanner` in [policy/scan/local_scanner.go](policy/scan/local_scanner.go)

**Scan Flow**:
```
Job (Inventory + Bundle + Options)
  ↓
distributeJob() - Batches assets, connects providers
  ↓
Multiple provider.Runtime instances (one per asset)
  ↓
RunAssetJob() - Per-asset execution
  ↓
localAssetScanner.run() - Policy execution for single asset
  ↓
Results → Reporters (console, SARIF, JUnit, etc.)
```

**Key Components**:
- **Inventory**: List of assets to scan (from CLI flags or inventory files)
- **Asset**: Describes target system (platform, connections, credentials)
- **Provider Runtime**: Manages provider plugins via gRPC, handles connections
- **DataLake**: Storage layer with two implementations:
  - InMemory (default): Fast, ephemeral
  - SQLite (feature flag): Persistent storage
- **Reporters**: Various output formats (human-readable, JSON, SARIF, etc.)

### Provider System

The provider system (inherited from cnquery) enables scanning diverse targets:

**Connection Flow**:
```
Inventory Asset → Runtime.Connect() → Provider Plugin (gRPC) → Target System
```

**Providers are separate processes**:
- Communicate via gRPC using `providers-sdk`
- Can be written in any language
- Support auto-update with versioned protocols
- Examples: os, aws, k8s, azure, gcp, github, etc.

**Provider Discovery**:
- Can delay discovery (`DelayDiscovery` flag)
- Detect platform details and update asset info on connect
- Platform IDs synchronized with Mondoo Platform for asset tracking

### Protocol Buffers & gRPC

Protobuf is central to cnspec's architecture:

**Data Structures** ([policy/cnspec_policy.proto](policy/cnspec_policy.proto)):
- All major types defined in proto: `Policy`, `Bundle`, `Framework`, `ResolvedPolicy`, `ExecutionJob`, `Report`, `Score`
- Enables versioning, backwards compatibility, fast serialization
- vtproto used for optimized marshaling (`cnspec_policy_vtproto.pb.go`)

**RPC Services**:
- `PolicyHub`: CRUD operations for policies and frameworks
- `PolicyResolver`: Policy resolution, job updates, result storage
- ranger-rpc (Mondoo's gRPC wrapper) for communication

**Provider Plugins**:
- Each provider implements gRPC service defined in `providers-sdk`
- Allows distributed provider ecosystem
- Versioned protocol ensures compatibility

### Key Architectural Patterns

1. **Graph-Based Policy Resolution**: Dependencies form a directed graph, traversed from leaves (checks) to roots (policies), with pruning for efficiency

2. **Two-Phase Execution**: Compilation (MQL → Code Bundles, cached) + Execution (Code Bundles → Results, per asset)

3. **Upstream/Local Hybrid**: Works standalone (incognito mode) or delegates to Mondoo Platform for policy storage, asset tracking, vulnerability data

4. **Service Locator**: `LocalServices` bundles PolicyHub and PolicyResolver, can wrap upstream services for seamless online/offline operation

5. **Collector Pattern**: Multiple observers can attach to execution; examples: BufferedCollector, FuncCollector, PolicyServiceCollector

6. **Migration System**: Bundles include versioned migrations (CREATE/MODIFY/DELETE) for policy evolution

## Important Development Notes

### Dependency Management

- **Forbidden packages**: Do not use `github.com/pkg/errors` (use `github.com/cockroachdb/errors`) or `github.com/mitchellh/mapstructure` (use `github.com/go-viper/mapstructure/v2`)
- **cnquery dependency**: When proto files reference cnquery types, ensure cnquery repo is present via `make prep/repos`

### Error Handling

Use `github.com/cockroachdb/errors` for error handling:
```go
import "github.com/cockroachdb/errors"

// Wrap errors with context
return errors.Wrap(err, "failed to load policy")

// Create new errors
return errors.New("invalid policy structure")
```

### Generated Code

Never edit these files manually:
- `*.pb.go` - Generated from proto files
- `*.ranger.go` - Generated ranger-rpc code
- `*.vtproto.pb.go` - Optimized vtproto marshaling
- `*_gen.go` - Generated via `go generate`

Regenerate with `make cnspec/generate` after source changes.

### Policy Bundle Development

Policy files (`.mql.yaml`) structure:
```yaml
policies:
  - uid: example-policy
    name: Example Policy
    version: 1.0.0
    groups:
      - title: Security Checks
        filters: asset.platform == "linux"
        checks:
          - uid: example-check
            title: Example Check
            impact: 80
            mql: |
              users.where(name == "root").list {
                shell != "/bin/bash"
              }
```

**Key concepts**:
- **uid**: Unique identifier for policies, checks, queries
- **filters**: MQL expressions that determine applicability
- **impact**: Risk score 0-100 for prioritization
- **checks**: Scoring queries (pass/fail)
- **queries**: Data collection queries (no scoring)

Test policy files:
```bash
# Lint before committing
cnspec policies lint ./content/your-policy.mql.yaml

# Test locally
cnspec scan local -f ./content/your-policy.mql.yaml
```

### MQL Development

MQL (Mondoo Query Language) syntax examples:
```coffee
# Resource access
users.where(name == "root")

# Filtering and assertions
sshd.config.params["PermitRootLogin"] == "no"

# List operations
processes.list { name pid }

# Relationships
files("/etc").where(name == /\.conf$/)
```

For MQL resources available per provider, see [MQL resources documentation](https://mondoo.com/docs/mql/resources/).

### Testing Policies

When adding/modifying policies:

1. **Lint the policy**: `cnspec policies lint <file>`
2. **Test against target**: `cnspec scan <target> -f <policy-file>`
3. **Check for regressions**: Run existing policy tests in `content/` directory
4. **Verify scoring**: Ensure impact ratings and scoring systems work correctly

### Working with Providers

To test against specific providers:
```bash
# Local system
cnspec scan local

# Docker
cnspec scan docker image ubuntu:22.04

# AWS (uses local AWS CLI config)
cnspec scan aws

# Kubernetes
cnspec scan k8s

# SSH
cnspec scan ssh user@host
```

Provider development happens in separate repos but can be tested with cnspec.

## Directory Structure

- **`apps/cnspec/`**: Main CLI application entry point
  - `cmd/`: All CLI commands (scan, shell, bundle, etc.)
- **`policy/`**: Policy engine core
  - `scan/`: Scanning orchestration
  - `executor/`: Policy execution engine
  - `scandb/`: Database for scan results
- **`cli/`**: CLI components
  - `components/`: Reusable UI components
  - `reporter/`: Output formatters (SARIF, JUnit, JSON, etc.)
- **`content/`**: Default security policies (AWS, Linux, K8s, etc.)
- **`internal/`**: Internal packages
  - `bundle/`: Bundle loading and validation
  - `datalakes/`: Data storage implementations
  - `lsp/`: Language Server Protocol support
- **`examples/`**: Example policy files
- **`test/`**: Integration tests
- **`docs/`**: Documentation

## Resources

- [cnspec Documentation](https://mondoo.com/docs/cnspec/home/)
- [MQL Documentation](https://mondoo.com/docs/mql/home/)
- [Policy Authoring Guide](https://mondoo.com/docs/cnspec/cnspec-policies/write/)
- [cnquery Repository](https://github.com/mondoohq/cnquery)
