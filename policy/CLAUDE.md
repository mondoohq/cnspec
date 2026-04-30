# policy/CLAUDE.md

Policy engine internals. Loaded automatically when working under `policy/`.

For policy *content* authoring (writing `.mql.yaml` files), see `content/CLAUDE.md`.

## cnspec vs cnquery

**cnspec is built on top of cnquery** — it's not just a dependency relationship:

- **cnquery provides**: MQL query engine, provider system, resource framework, data gathering.
- **cnspec adds**: Policy evaluation, scoring, compliance frameworks, security assessments, risk factors.
- **Shared runtime**: Both use the same provider system (`providers.Runtime`) for connecting to target systems.
- **Import path**: cnspec imports `go.mondoo.com/cnquery/v12` extensively.

When working on cnspec, you may need to understand or modify cnquery components.

## Policy engine flow

The policy execution pipeline has four main stages.

### 1. Bundle loading & compilation

- Policy files (`.mql.yaml`) loaded via `BundleLoader` in [bundle.go](bundle.go).
- Bundles contain: Policies, QueryPacks, Frameworks, Queries, Properties, Migrations.
- MQL code compiled to executable `llx.CodeBundle` (protobuf format) via `mqlc.CompilerConfig`.
- Code bundles are cached and reusable across multiple asset scans.

### 2. Policy resolution

- Core logic in [resolver.go](resolver.go) and [resolved_policy_builder.go](resolved_policy_builder.go).
- Asset filters evaluated to determine which policies apply to each asset.
- Builds dependency graph of: Policies → Frameworks → Controls → Checks/Queries.
- Graph walking from non-prunable leaf nodes ensures only applicable checks run.
- Generates two critical structures:
  - **ExecutionJob**: All queries to execute with checksums and code bundles.
  - **CollectorJob**: Score aggregation rules with scoring systems and hierarchy.

### 3. Execution

- Orchestrated by `GraphExecutor` in [executor/graph.go](executor/graph.go).
- Queries executed via cnquery's `llx.MQLExecutorV2`.
- Results collected through `BufferedCollector` pattern.
- Handles: data points, scores, risk factors, upstream transmission.

### 4. Result collection & scoring

- Scores calculated via `ScoreCalculator` in [score_calculator.go](score_calculator.go).
- Reporting jobs aggregate child scores based on impact ratings.
- Final `Report` structures contain scores, statistics, CVSS data.

## Scanning architecture

Main orchestrator: `LocalScanner` in [scan/local_scanner.go](scan/local_scanner.go).

**Scan flow**:

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

**Key components**:

- **Inventory**: List of assets to scan (from CLI flags or inventory files).
- **Asset**: Describes target system (platform, connections, credentials).
- **Provider Runtime**: Manages provider plugins via gRPC, handles connections.
- **DataLake**: Storage layer with two implementations:
  - InMemory (default): Fast, ephemeral.
  - SQLite (feature flag): Persistent storage.
- **Reporters**: Various output formats (human-readable, JSON, SARIF, etc.).

## Provider system

The provider system (inherited from cnquery) enables scanning diverse targets.

**Connection flow**:

```
Inventory Asset → Runtime.Connect() → Provider Plugin (gRPC) → Target System
```

**Providers are separate processes**:

- Communicate via gRPC using `providers-sdk`.
- Can be written in any language.
- Support auto-update with versioned protocols.
- Examples: os, aws, k8s, azure, gcp, github, etc.

**Provider discovery**:

- Can delay discovery (`DelayDiscovery` flag).
- Detect platform details and update asset info on connect.
- Platform IDs synchronized with Mondoo Platform for asset tracking.

## Protocol buffers & gRPC

Protobuf is central to cnspec's architecture.

**Data structures** ([cnspec_policy.proto](cnspec_policy.proto)):

- All major types defined in proto: `Policy`, `Bundle`, `Framework`, `ResolvedPolicy`, `ExecutionJob`, `Report`, `Score`.
- Enables versioning, backwards compatibility, fast serialization.
- vtproto used for optimized marshaling (`cnspec_policy_vtproto.pb.go`).

**RPC services**:

- `PolicyHub`: CRUD operations for policies and frameworks.
- `PolicyResolver`: Policy resolution, job updates, result storage.
- ranger-rpc (Mondoo's gRPC wrapper) for communication.

**Provider plugins**:

- Each provider implements gRPC service defined in `providers-sdk`.
- Allows distributed provider ecosystem.
- Versioned protocol ensures compatibility.

## Key architectural patterns

1. **Graph-based policy resolution**: Dependencies form a directed graph, traversed from leaves (checks) to roots (policies), with pruning for efficiency.
2. **Two-phase execution**: Compilation (MQL → Code Bundles, cached) + Execution (Code Bundles → Results, per asset).
3. **Upstream/local hybrid**: Works standalone (incognito mode) or delegates to Mondoo Platform for policy storage, asset tracking, vulnerability data.
4. **Service locator**: `LocalServices` bundles PolicyHub and PolicyResolver, can wrap upstream services for seamless online/offline operation.
5. **Collector pattern**: Multiple observers can attach to execution; examples: BufferedCollector, FuncCollector, PolicyServiceCollector.
6. **Migration system**: Bundles include versioned migrations (CREATE/MODIFY/DELETE) for policy evolution.

## Generated code

Never edit these files manually:

- `*.pb.go` — Generated from proto files.
- `*.ranger.go` — Generated ranger-rpc code.
- `*.vtproto.pb.go` — Optimized vtproto marshaling.
- `*_gen.go` — Generated via `go generate`.

Regenerate with `make cnspec/generate` after source changes. When proto files reference cnquery types, ensure the cnquery repo is present via `make prep/repos`.
