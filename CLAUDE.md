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

# Lint policy and querypack files
make test/lint/content

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
cnspec policy lint ./content

# Lint a specific policy file
cnspec policy lint ./content/mondoo-linux-security.mql.yaml
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
- **Multi-statement check MQL**: A check's `mql:` block can contain multiple top-level statements. Each statement is scored as a separate datapoint and the check passes only if every datapoint passes — it is *not* "last expression wins". Use this pattern when you want each assertion to surface independently in scan output; collapse to a single `&&`-joined expression only if you want one combined datapoint.

**Formatting requirements**:
- All `desc` (description) and `remediation` fields in policy files must be valid Markdown. These fields are rendered as Markdown in the UI, so use proper Markdown syntax for headings, lists, code blocks, links, etc.
- Check `title` fields must be 75 characters or fewer.
- When writing CLI commands in remediation steps, verify that the subcommands and flags you use are valid. For Azure, consult `content/validation/cmd_data/azure_commands.json`. For aws/oci/gcp, run `python3 content/validation/validate_remediation_commands.py <cloud>` — the validator introspects the locally-installed CLI and flags unknown commands or flags.

#### Compliance tags (`compliance/<framework>: <control-uid>`)

**Never copy compliance tags from a neighboring check.** The nearby check was mapped for a different control objective; reusing its tags propagates a wrong mapping and misleads auditors. Two checks that both "relate to identity" can map to different controls.

When adding or changing compliance tags, follow this process for **each** framework the policy already tags:

1. **Read the authoritative control text.** Open the framework definition in `cnspec-enterprise-policies/frameworks/<framework>.mql.yaml` (e.g., `iso-27001-2022.mql.yaml`, `soc2-2017.mql.yaml`, `nist-sp-800-53-rev5.mql.yaml`). Each control has a `uid`, `title`, and usually `docs.desc`. Ask the user where their clone lives if you don't already know; if the files aren't available, stop and tell the user — do not guess.
2. **State in one sentence what the check actually enforces.** If the check is about identity proofing, say so; if it's about encryption-at-rest, say so. Do not let the check's *title* mislead you — read the MQL.
3. **Find the single best-matching control** by scanning control titles and descriptions for language that covers the enforced behavior. Strict fit only: MFA, password policy, and session-timeout controls are *not* acceptable stand-ins for identity-proofing, encryption, network-isolation, etc.
4. **If no control fits, tag it `false`.** This is an established pattern in this repo (grep for `compliance/.*: false`). A missing mapping is strictly better than a wrong one — wrong mappings get caught in compliance audits and create trust debt.
5. **Cite the control you chose.** When you present tags to the user, include the control title and a short quote from the control description so the user can verify.

Known high-value anchors (verify before using):
- Identity proofing / email verification: `iso-27001-2022-a-5-16` (Identity management), `nist-csf-2-pr-aa-02` ("Identities are proofed and bound to credentials"), `nist-sp-800-53-rev5-ia-12` (Identity Proofing). No direct equivalent in NIST CSF 1.x, NIST 800-171 rev2, NIS2 Article 21(2), or SOC 2 2017.
- Authenticator / MFA strength: `iso-27001-2022-a-8-5`, `nist-csf-2-pr-aa-03`, `nist-sp-800-53-rev5-ia-2`, `soc2-control-cc6-1-4`. Do **not** reuse these for identity-proofing checks.

Test policy files:

```bash
# Lint before committing
cnspec policy lint ./content/your-policy.mql.yaml

# Test locally
cnspec scan local -f ./content/your-policy.mql.yaml
```

#### Terraform variants for cloud policies

When you add or modify a check in a cloud policy (`mondoo-aws-security`, `mondoo-azure-security`, `mondoo-gcp-security`, `mondoo-oci-security`, `mondoo-kubernetes-security`, etc.), the check should run against both the live cloud runtime **and** any Terraform asset that configures the same resource. Convert single-platform checks to a `variants:` block with up to four children:

- `<uid>-<cloud>` — runtime check (`asset.platform == 'gcp'`, `'aws'`, …)
- `<uid>-terraform-hcl` — `terraform.resources(...)` against HCL source
- `<uid>-terraform-plan` — `terraform.plan.resourceChanges` against `terraform plan` JSON
- `<uid>-terraform-state` — `terraform.state.resources` against `terraform.tfstate`

Reference patterns in this repo:

- GCP: `mondoo-gcp-security-memorystore-iam-auth-enabled` in `content/mondoo-gcp-security.mql.yaml`
- HCL nested-block fanout: `mondoo-gcp-security-cloud-sql-mysql-skip-show-database-enabled-terraform-*` (database_flags)
- Plan/state list-of-objects shape: `mondoo-gcp-security-cloud-storage-bucket-retention-policy-locked-terraform-*`

**When a Terraform variant is not possible, leave a YAML comment above the parent check explaining why** so future passes don't re-investigate. Common reasons:

- The runtime check evaluates operational telemetry (job state, latest execution status, observed traffic) that has no configuration analog.
- The cloud resource is managed only via SDK / CLI / console and has no Terraform resource (e.g., short-lived imperative API calls like Vertex AI custom jobs).
- The runtime check depends on cross-resource correlation (e.g., "every cluster has a backup plan that points at it") that the runtime check itself does not yet implement correctly — in which case fix the runtime first.
- The runtime check inspects a field whose Terraform analog is a different feature (don't paper over the mismatch with a vacuous variant).

Comment format (inserted on the line before `- uid:`):

```yaml
# No Terraform variants: <one-sentence reason>. <Optional: when this could be revisited>.
- uid: mondoo-<cloud>-security-...
```

The comment must explain the technical limitation, not just say "skip". Keep it to a few lines.

After every batch of variant additions, both must pass:

```bash
cnspec policy lint content/mondoo-<cloud>-security.mql.yaml
python3 content/validation/validate_remediation_commands.py <cloud>
```

**MQL gotcha for Terraform variants:** the parser rejects `.all((expr) || ...)` — a parenthesized clause as the first token inside `.all(`. Rely on `&&` binding tighter than `||` instead of writing leading parentheses.

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

1. **Lint the policy before committing**: `cnspec policy lint <file>` (e.g., `cnspec policy lint content/mondoo-aws-security.mql.yaml`). This must pass before committing any policy changes.
2. **Validate remediation commands**: `python3 content/validation/validate_remediation_commands.py` (see below).
3. **Check for regressions**: Run existing policy tests in `content/` directory
4. **Verify scoring**: Ensure impact ratings and scoring systems work correctly

### Validating Remediation CLI Commands

The `content/validation/` directory contains tooling to verify that CLI commands in remediation sections use valid subcommands and flags.

**Validate commands:**

```bash
# Validate all policies (currently AWS, Azure, and OCI)
python3 content/validation/validate_remediation_commands.py

# Validate a specific cloud
python3 content/validation/validate_remediation_commands.py aws
python3 content/validation/validate_remediation_commands.py azure
python3 content/validation/validate_remediation_commands.py oci
```

The validator checks each `aws`/`az`/`oci` command in ```` ```bash ```` code blocks within `id: cli` remediation sections against a known-good database of commands and flags. Output shows `[PASS]` or `[FAIL]` with the check UID and the offending command.

**How the validator sources command data:**

The validator builds its commands database for `aws`, `oci`, and `gcp` **in-memory** at validation time by introspecting each cloud's locally-installed CLI. The relevant CLI must be on PATH:

- **aws**: introspects botocore service models bundled with the AWS CLI v2
- **oci**: walks the Click command tree from the `oci_cli` Python package
- **gcp**: reads the Google Cloud SDK's static completion tree

If a required CLI is missing, the validator prints actionable install hints and exits non-zero.

**azure** is the exception: it still uses a checked-in `content/validation/cmd_data/azure_commands.json` because refreshing Azure CLI metadata is slow enough that doing it on every run would significantly extend CI.

**Regenerate Azure command data** (when the Azure CLI version changes):

```bash
python3 content/validation/dump_azure_commands.py
```

**Never hand-edit** `azure_commands.json`.

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
  - `validation/`: CLI command validation scripts and reference data
- **`internal/`**: Internal packages
  - `bundle/`: Bundle loading and validation
  - `datalakes/`: Data storage implementations
  - `lsp/`: Language Server Protocol support
- **`examples/`**: Example policy files
- **`test/`**: Integration tests
- **`docs/`**: Documentation

## Resources

- [cnspec Documentation](https://mondoo.com/docs/cnspec/)
- [MQL Documentation](https://mondoo.com/docs/mql/)
- [MQL Built-in Functions](https://mondoo.com/docs/mql/functions) (parse.json, parse.date, regex, etc.)
- [MQL Resources by Provider](https://mondoo.com/docs/mql/resources/) ([AWS](https://mondoo.com/docs/mql/resources/aws-pack/), [Azure](https://mondoo.com/docs/mql/resources/azure-pack/), [GCP](https://mondoo.com/docs/mql/resources/gcp-pack/), [Core](https://mondoo.com/docs/mql/resources/core-pack/))
- [Policy Authoring Guide](https://mondoo.com/docs/cnspec/write-policies/write-intro/)
- [cnquery Repository](https://github.com/mondoohq/cnquery)
- [Full Mondoo Docs (LLM-friendly text)](https://mondoo.com/docs/llms-full.txt)
