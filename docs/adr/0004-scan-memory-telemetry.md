# ADR-0004: Scan Memory Telemetry

**Date:** 2026-08-14
**Status:** Proposed

## Context

cnspec runs in memory-constrained environments — operator pods, CI runners, containers with a hard cgroup limit. When a scan of a large asset set exhausts that limit, the process is killed by the kernel and the operator is left with no evidence beyond a truncated log and an exit code. There is currently no way to answer, after the fact, how much memory a scan was actually using, how close it came to its limit, or how that varies across asset types and releases.

The pieces needed to answer those questions already exist in the tree, but they are not connected:

- **Memory is measured, but only locally.** `policy/scan/scan_pipeline.go` samples heap statistics, goroutine count, cgroup `memory.current` / `memory.max`, and accumulated reporter bytes after each asset scan. `policy/executor/internal/execution_manager.go` samples allocations per query. All of it is gated behind the `DEBUG_PROVIDER_MEMORY` environment variable and emitted as log lines, so it is available only to someone who has already reproduced the problem on a machine they control. That is precisely what cannot be done for a scan that has already failed somewhere else.

- **A metrics channel exists, but carries no memory data.** `policy.ScanStatistics` (a repeated `policy.Metric` of namespaced name/unit/value) is packed into `ReportUploadCompletedReq.details` and sent upstream (#3103, extended by #3148 and #3234). The `policy/scanstats` package collects these through a context-propagated `Collector`. Today it reports scan duration, upload size, upload duration and throughput, and per-kind counts of checks, data queries, policies, controls and frameworks. `policy.Metric` is deliberately namespace-based so that new measurements require no proto change.

Two properties of the existing scan pipeline constrain any design here:

1. **The metrics channel cannot report its own process's death.** `ReportUploadCompleted` is called at the end of a successful asset upload. If cnspec is killed by the kernel, no call is made and nothing is sent. Telemetry can therefore describe the approach to a memory ceiling, but not the moment of crossing it.

2. **Collectors are per-asset; memory is per-process.** `WithServices` is invoked once per asset (`policy/scan/local_scanner.go`), so each `Collector` has a single asset's lifetime. But assets scan concurrently up to `parallelism` (`scanSem` in `policy/scan/scan_pipeline.go`), so any runtime memory reading taken during one asset's scan includes every other asset in flight. There is no honest way to derive a per-asset memory cost from a process-wide number under concurrency.

## Decision

Record process-wide memory high-water marks during a scan and emit them as `scanstats` metrics on the existing `ScanStatistics` channel. Make no per-asset attribution claim; instead publish the concurrency level alongside the peak so consumers can normalize.

### 1. Scope: the `UploadResultsV2` path

`ScanStatistics` only reaches upstream through `ReportUploadCompleted`, which is only called on the scan-database upload path selected when the `UploadResultsV2` feature is active (`policy/scan/local_scanner.go`). Scans on the in-memory path have no metrics transport at all.

This ADR accepts that boundary rather than building a second transport. Adding memory metrics to the legacy path would mean either a new upstream RPC or overloading the error-reporting endpoint with routine telemetry, both of which cost more than they return for a path that is being superseded.

### 2. Sampling: `runtime/metrics`, not `MemStats.Alloc`

A new `scanstats.MemTracker` owns a process-wide high-water mark, updated by a single sampler goroutine started in `RunLocalScan` and stopped when it returns.

The sampled quantity is the Go runtime footprint:

```
/memory/classes/total:bytes  -  /memory/classes/heap/released:bytes
```

read via `runtime/metrics`. This is deliberately not `runtime.MemStats.Alloc`, which the existing debug logging uses. `Alloc` counts live heap objects and excludes stacks, the allocator's own structures, and memory the runtime has retained but not returned to the OS — so it systematically undercounts the quantity the kernel's OOM killer actually acts on. The expression above is the footprint the Go runtime accounts against `GOMEMLIMIT`, which makes it both the right predictor of an OOM kill and directly comparable to a configured limit.

`runtime/metrics.Read` also does not stop the world, unlike `runtime.ReadMemStats`. A one-second sampling interval is therefore free, where the same interval built on `ReadMemStats` would perturb the scan it is measuring.

### 3. Attribution: peak plus concurrency denominator

Each per-asset `Collector` snapshots the tracker immediately before `ToProto()`, so every asset's upload carries the process state as of that asset's completion. Ordering the resulting records by the existing `cnspec.scan.duration` reconstructs the memory ramp across a run.

A peak is uninterpretable without knowing how loaded the pipeline was when it occurred: 3 GB with 48 assets in flight and 3 GB with 2 in flight describe entirely different problems. The in-flight count is therefore captured **at the moment the peak is set**, not at asset finish. The scan dispatcher registers an in-flight accessor (`len(scanSem)`) on the tracker, and the sampler reads it on any tick that raises the maximum.

Because the peak is process-wide but a record is emitted per asset, a single run scanning N assets emits N records carrying one process's monotonically rising high-water mark. These are repeated observations of one process, not N independent measurements, so they must be aggregated with `max` per run rather than averaged — an average is weighted by asset count and drifts below the true peak.

That aggregation requires knowing which records came from the same process, and nothing in the scan path currently identifies a run: the collector is per-asset, the upload session is per-asset, and there is no job or run identifier. A `cnspec.scan.run_id` string metric is therefore generated once per `RunLocalScan` invocation and recorded into every asset's collector. `policy.Metric` already carries a `string_value`, so this needs no proto change; `scanstats.Collector` gains an `AddString` alongside its existing typed setters.

Per-asset heap deltas were considered and rejected — see Alternatives.

### 4. Metrics

All names extend the existing `cnspec.scan.*` namespace. No proto change is required.

| Metric | Unit | Meaning |
|---|---|---|
| `cnspec.scan.run_id` | — | Identifies records from one scan process |
| `cnspec.scan.mem.runtime_peak_bytes` | bytes | Go runtime footprint high-water mark |
| `cnspec.scan.mem.runtime_at_finish_bytes` | bytes | Footprint when this asset completed |
| `cnspec.scan.mem.goroutines_peak` | count | Goroutine high-water mark; leak signal |
| `cnspec.scan.mem.cgroup_current_bytes` | bytes | Cgroup usage, includes provider subprocesses |
| `cnspec.scan.mem.cgroup_peak_bytes` | bytes | Cgroup high-water mark |
| `cnspec.scan.mem.cgroup_max_bytes` | bytes | Cgroup limit; proximity is the OOM predictor |
| `cnspec.scan.concurrency.in_flight_at_peak` | count | Assets scanning when the peak was set |
| `cnspec.scan.concurrency.parallelism` | count | Configured scan concurrency |
| `cnspec.scan.concurrency.max_connections` | count | Configured connection ceiling |

### 5. Provider subprocesses are covered by the cgroup, not by direct measurement

Providers run as plugin subprocesses, so the Go runtime footprint of the cnspec process excludes them — and for image, registry and Kubernetes scans they are frequently where the memory actually goes.

Rather than measure them directly, this design relies on the cgroup metrics: provider subprocesses are forked into the same cgroup as their parent, so `memory.current` and `memory.peak` already account for the full tree. In containers and Kubernetes — the environments where memory limits are enforced and OOM kills occur — the cgroup reading is the true total footprint at no additional cost.

The accepted gap is that on hosts without cgroups (bare metal, macOS and Windows workstations) only the Go footprint is reported, which undercounts providers; and that no per-provider attribution is available anywhere. See Alternatives.

### 6. Platform coverage

Cgroup metrics are read best-effort from cgroup v2. `memory.peak` requires Linux 5.19 or later and is omitted on older kernels. A limit of `max` is reported as absent rather than as a sentinel number.

Where a cgroup value cannot be read, the metric is **omitted entirely** rather than reported as zero. A zero would be indistinguishable from a genuine measurement and would corrupt any aggregate computed across a mixed fleet. `Collector.ToProto()` already returns nil for an empty metric set, so absence propagates cleanly.

### 7. Always-on within scope

These metrics are collected unconditionally on the `UploadResultsV2` path. Making them opt-in would defeat the purpose: the data is valuable precisely because it is present for the scan that failed, which is never the scan someone thought to enable diagnostics for. The cost is one goroutine performing a sub-microsecond read once per second.

`DEBUG_PROVIDER_MEMORY` retains its current meaning and continues to control verbose local logging only.

### 8. File layout

- `policy/scanstats/mem.go` — `MemTracker`: sampling loop, high-water tracking, in-flight correlation, cgroup reader, and the snapshot that records into a `Collector`.
- `policy/scanstats/collector.go` — metric name constants and `AddString`, for the run identifier.
- `policy/scanstats/context.go` — tracker propagation, mirroring the existing `ContextWithCollector`.
- `policy/scan/local_scanner.go` — tracker construction, sampler lifecycle, recording the static concurrency settings.
- `policy/scan/scan_pipeline.go` — registers the in-flight accessor; existing `logMemoryStats` is repointed at the tracker so there is a single source of truth for memory readings.
- `internal/datalakes/sqlite/sqlite.go` — snapshots the tracker into the per-asset collector before upload.

The tracker takes an injected sample function and an injected cgroup root, so high-water logic, in-flight correlation, and cgroup parsing (v2 present, absent, `max` literal, malformed) are all testable without a live runtime or a real `/sys`.

## Alternatives considered

**Per-asset heap deltas.** Record the footprint at an asset's start and end and report the difference. Rejected: under concurrency the window contains every other in-flight asset's allocations and frees, so the value can be arbitrarily inflated or negative. It has the shape of a per-asset cost without the meaning of one, which is worse than not reporting it.

**Crash breadcrumb file.** Persist the running peak to disk and report it on the next startup, so a killed process leaves evidence behind. Rejected: it adds a writable-path requirement, a staleness policy and a cleanup path, and in an ephemeral container the breadcrumb is destroyed along with the process it was meant to outlive — the exact case motivating it.

**Cgroup OOM-kill counter.** Read `memory.events` at scan start and end to detect that the kernel killed something in the cgroup during the scan. Rejected for now as scope beyond the ramp signal; it also resets when a killed container is replaced, which is the case it would most be wanted for.

**Per-provider RSS.** Measure each provider subprocess directly. This requires exposing the plugin process ID on the running-provider handle upstream in mql, plus either per-OS process memory reads or a new dependency for cross-platform support. Rejected as disproportionate given that the cgroup already captures the total in the environments that matter. Recorded here as the known extension if per-provider attribution is later needed.

## Consequences

- A scan that approaches its memory ceiling produces an observable ramp: rising peak values with a known concurrency denominator and a known limit. The kill itself remains unreported, and is inferred from the gap that follows the last record.
- Memory data becomes available for scans nobody instrumented in advance, which is the only class of scan where it is genuinely needed.
- Comparing memory cost across asset types or across releases becomes possible in aggregate, over many scans, rather than per individual scan.
- Scans on the in-memory path emit no memory telemetry. Coverage is tied to `UploadResultsV2` rollout.
- Hosts without cgroups report Go footprint only, which understates total usage by the provider subprocess memory. Aggregates that mix containerized and non-containerized scans must key on the presence of the cgroup metrics to remain comparable.
- Peak values must be aggregated with `max` grouped by `cnspec.scan.run_id`. Averaging them across assets yields a number that looks like a fleet average but is weighted by run size and understates every peak.
- Metric names become a wire contract. As with `upload.FailureKind`, names may be added freely but must not be renamed or repurposed.
- One additional goroutine per scan process, sampling once per second.

## Out of scope

- How the platform consumes, stores or presents `ScanStatistics`. This ADR covers the cnspec client and stops at the wire boundary.
- Per-provider memory attribution (see Alternatives).
- Memory telemetry on the in-memory scan path.
- Detection or reporting of the OOM kill event itself.

## References

- #3103 — per-scan statistics on `ReportUploadCompleted`
- #3148 — upload failure kind reporting
- #3234 — scan-data upload retry
- #2932 — configurable provider connection ceiling and `DEBUG_PROVIDER_MEMORY` diagnostics
- ADR-0001 — Scan Parallelization Pipeline (`parallelism`, `scanSem`, `connSem`)
