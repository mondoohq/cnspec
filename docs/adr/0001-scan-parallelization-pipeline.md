# ADR-0001: Scan Parallelization Pipeline

**Date:** 2026-04-03
**Status:** Accepted

## Context

cnspec scans can discover hundreds or thousands of assets (e.g. container images in a registry, pods across Kubernetes namespaces). The scanner walks an asset tree depth-first: root nodes (like a K8s cluster) contain branch nodes (namespaces) which contain leaf nodes (individual workloads).

The original implementation used a batch-and-wait model: accumulate up to 50 assets, synchronize them with upstream, scan the entire batch (with bounded parallelism), wait for all scans to finish, then start the next batch. This created idle periods where scan workers had nothing to do while the last few slow assets in a batch completed. It also mixed connection management, upstream synchronization, and scan execution into a single function, making the flow difficult to follow.

## Decision

Replace the batch-and-wait model with a three-stage pipeline where each stage has a single responsibility and a clear interface. The stages communicate through direct method calls, and concurrency is managed via channel-based semaphores.

### Architecture

```
 ┌──────────────┐       ┌──────────────┐       ┌────────────────┐
 │  Tree Walker  │──Add──▶  syncBatcher  │──Submit──▶ scanDispatcher │
 │ (scanSubtree) │       │              │       │                │
 │              │──Flush─▶              │       │   worker pool  │
 │              │       └──────────────┘       │  (goroutines)  │
 │              │──Wait────────────────────────▶│                │
 └──────────────┘                               └────────────────┘
```

### Stage 1: Tree Walker (`scanContext.scanSubtree`)

Walks the asset tree depth-first. For each child:

- **Acquires a connection slot** (`connSem`) before calling `Connect()`.
- **Branch nodes**: releases the connection slot, calls `batcher.Flush()` and `dispatcher.Wait()` to drain in-flight work (to avoid holding multiple subtrees in memory), then recurses.
- **Leaf nodes**: feeds the connected asset to `batcher.Add()`.
- **Skipped assets** (no platform IDs): closes the asset and releases the connection slot.

After all children are processed, flushes remaining leaves, then dispatches the node itself (e.g. the namespace) for scanning. The node is dispatched after all children are connected and dispatched — this ensures the node's runtime isn't closed before children are connected, while still allowing the node to scan concurrently with its children. Finally, `dispatcher.Wait()` drains all in-flight scans before returning.

### Stage 2: Sync Batcher (`syncBatcher`)

Accumulates connected assets and calls `syncBatchWithUpstream` when the buffer reaches `syncBatchSize` (5). After syncing, forwards each asset to the scan dispatcher.

**Interface:**
- `Add(ctx, asset)` — buffers the asset; auto-flushes when full.
- `Flush(ctx)` — syncs and dispatches all buffered assets. No-op if empty.

Assets with `DelayDiscovery` are forwarded without syncing — the scan goroutine handles their sync individually after resolving the actual platform.

### Stage 3: Scan Dispatcher (`scanDispatcher`)

Manages a bounded pool of scan workers.

**Interface:**
- `Submit(ctx, asset)` — returns immediately. Spawns a goroutine that waits for a worker slot internally, so the batcher and tree walker are never blocked on worker availability.
- `Wait()` — blocks until all submitted scans have completed.

Each submitted goroutine:
1. Waits for a scan slot (`scanSem`).
2. Marks the asset as in-progress in the progress bar (via `OnProgress` with 0%).
3. Runs the full scan lifecycle (delayed discovery, policy evaluation, result collection, vuln report fetch).
4. Marks the asset as completed in the progress bar — only after all post-scan work finishes, so the visual state accurately reflects when the worker slot is freed.
5. Closes the asset (frees the gRPC connection).
6. Releases the scan slot and the connection slot.

### Three independent controls

| Concern | Mechanism | Default |
|---|---|---|
| **Max open connections** | `connSem` — buffered channel of size `maxConnections` | 50 |
| **Upstream sync batching** | `syncBatcher` buffer flushed at `syncBatchSize` | 5 |
| **Concurrent scans** | `scanSem` — buffered channel of size `parallelism` | Configured per job |

### Flow

```
for each child in node.Children:
    connSem <- acquire               # block if 50 connections open
    connected = Connect(child)

    if branch node:
        release connSem              # branches don't hold connections
        batcher.Flush()              # sync any pending leaves
        dispatcher.Wait()            # drain scans (memory constraint)
        recurse into subtree
    else (leaf):
        batcher.Add(connected)       # buffers; auto-flushes at 5
                                     #   flush calls syncBatchWithUpstream
                                     #   then dispatcher.Submit for each asset
                                     #     Submit returns immediately,
                                     #     goroutine waits for worker slot

batcher.Flush()                      # flush remaining leaves
dispatch node itself                 # runs concurrently with children
dispatcher.Wait()                    # drain all before returning
```

### Non-blocking submission

`Submit` does not block on worker availability. It spawns a goroutine that waits for `scanSem` internally. This means `batcher.Flush()` returns as soon as all assets are queued, letting the tree walker continue connecting more children while submitted scans wait for worker slots. This eliminates the pattern where `Flush` would block in a dispatch loop, preventing the tree walker from making progress.

### Progress tracking

- **In-progress**: marked as soon as a worker picks up the asset (acquires `scanSem`), not when the first query result comes back. This gives accurate visual feedback about which assets are actively being scanned.
- **Completed**: marked only after all post-scan work finishes (report collection, vulnerability report fetch from upstream). This ensures the progress bar accurately reflects when the worker slot is actually freed, avoiding the misleading pattern where assets appear done but workers are still busy with post-scan network calls.

### Memory constraint: one subtree at a time

Before recursing into a branch node, the tree walker calls `dispatcher.Wait()` to drain all in-flight scans. This ensures only one branch's children data (e.g. one namespace's pods) is in memory at a time. Within a single subtree, scans run concurrently up to `parallelism`, but subtrees are processed sequentially.

### Key properties

1. **No idle workers.** Workers pick up assets as soon as they're submitted. Non-blocking `Submit` means the tree walker, batcher, and dispatcher pipeline flows continuously without blocking on worker availability.

2. **Bounded resource usage.** The connection semaphore (`maxConnections = 50`) caps the number of simultaneously open provider runtimes/gRPC connections. This is independent of scan parallelism — you can have 50 connected assets with only a few actively scanning.

3. **Efficient upstream sync.** Assets are batched (in groups of 5) for the `syncBatchWithUpstream` network call, avoiding per-asset round-trips. After syncing, each asset is immediately submitted for scanning.

4. **Depth-first tree walk preserved.** Branch nodes flush, drain, and recurse, so only one branch's children are in memory at a time. This keeps memory usage predictable for deep trees (e.g. cluster > namespace > pod).

5. **Connection slot lifecycle.** A connection slot is acquired by the tree walker before `Connect()` and released by the scan dispatcher goroutine after the asset is closed. For branch nodes (which recurse rather than scan), the slot is released immediately since their children acquire their own slots. For skipped assets (no platform IDs), the slot is released after closing.

6. **Separation of concerns.** Each stage has a single responsibility and a minimal interface (`Add`/`Flush`, `Submit`/`Wait`). The tree walker knows nothing about upstream sync or worker pools. The batcher knows nothing about connections or scan execution. The dispatcher knows nothing about tree structure or batching.

7. **Accurate progress reporting.** The progress bar reflects the true state of each asset: in-progress when a worker picks it up, completed when all work (including post-scan network calls) is done.

### File layout

- `local_scanner.go` — tree walker (`scanContext`, `scanSubtree`), job orchestration (`distributeJob`), `RunAssetJob`, `syncBatchWithUpstream`.
- `scan_pipeline.go` — `syncBatcher` and `scanDispatcher` implementations, `scanSingleAsset`, panic reporting.

## Consequences

- Scan throughput improves for large asset sets because the pipeline flows continuously — connecting, syncing, and scanning overlap instead of happening in rigid batch phases.
- The `parallelism` job setting controls scan concurrency; `maxConnections` is a separate, higher ceiling for connected-but-not-yet-scanning assets.
- The upstream sync batch size (5) is small enough to keep dispatch latency low while still avoiding per-asset API calls.
- `dispatcher.Wait()` at subtree boundaries ensures child scans complete before moving to the next subtree, maintaining the one-namespace-at-a-time memory constraint.
- The modular design makes each stage independently testable and easier to reason about.
- Progress bar accuracy is improved — in-progress and completed states reflect actual worker activity, not internal pipeline events.
