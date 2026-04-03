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
- **Branch nodes**: releases the connection slot, calls `batcher.Flush()` and `dispatcher.Wait()` to drain in-flight work, then recurses.
- **Leaf nodes**: feeds the connected asset to `batcher.Add()`.
- **Skipped assets** (no platform IDs): closes the asset and releases the connection slot.

After all children are processed, flushes + drains, then processes the parent node itself.

### Stage 2: Sync Batcher (`syncBatcher`)

Accumulates connected assets and calls `syncBatchWithUpstream` when the buffer reaches `syncBatchSize`. After syncing, forwards each asset to the scan dispatcher.

**Interface:**
- `Add(ctx, asset)` — buffers the asset; auto-flushes when full.
- `Flush(ctx)` — syncs and dispatches all buffered assets. No-op if empty.

Assets with `DelayDiscovery` are forwarded without syncing — the scan goroutine handles their sync individually after resolving the actual platform.

### Stage 3: Scan Dispatcher (`scanDispatcher`)

Manages a bounded pool of scan workers. Each submitted asset is scanned in a goroutine that:

1. Acquires a scan slot (`scanSem`).
2. Runs the full scan lifecycle (delayed discovery, policy evaluation, result collection).
3. Closes the asset (frees the gRPC connection).
4. Releases the scan slot and the connection slot.

**Interface:**
- `Submit(ctx, asset)` — blocks if all scan slots are occupied.
- `Wait()` — blocks until all submitted scans have completed.

### Three independent controls

| Concern | Mechanism | Default |
|---|---|---|
| **Max open connections** | `connSem` — buffered channel of size `maxConnections` | 50 |
| **Upstream sync batching** | `syncBatcher` buffer flushed at `syncBatchSize` | 10 |
| **Concurrent scans** | `scanSem` — buffered channel of size `parallelism` | Configured per job |

### Flow

```
for each child in node.Children:
    connSem <- acquire               # block if 50 connections open
    connected = Connect(child)

    if branch node:
        release connSem              # branches don't hold connections
        batcher.Flush()              # sync any pending leaves
        dispatcher.Wait()            # drain running scans
        recurse into subtree
    else (leaf):
        batcher.Add(connected)       # buffers; auto-flushes at 10
                                     #   flush calls syncBatchWithUpstream
                                     #   then dispatcher.Submit for each asset
                                     #     Submit blocks if parallelism workers busy

batcher.Flush()                      # flush remaining leaves
dispatcher.Wait()                    # drain before scanning parent node
```

### Key properties

1. **No idle workers.** A scan worker picks up the next ready asset as soon as it finishes the current one. There is no batch boundary that forces workers to wait.

2. **Bounded resource usage.** The connection semaphore (`maxConnections = 50`) caps the number of simultaneously open provider runtimes/gRPC connections. This is independent of scan parallelism — you can have 50 connected assets with only 8 actively scanning.

3. **Efficient upstream sync.** Assets are still batched (in groups of 10) for the `syncBatchWithUpstream` network call, avoiding per-asset round-trips. But after syncing, each asset is immediately dispatched — we don't wait for the batch to finish scanning before syncing the next one.

4. **Depth-first tree walk preserved.** Branch nodes flush and drain before recursing, so only one branch's children are being connected at a time. This keeps memory usage predictable for deep trees (e.g. cluster > namespace > pod).

5. **Connection slot lifecycle.** A connection slot is acquired by the tree walker before `Connect()` and released by the scan dispatcher goroutine after the asset is closed. For branch nodes (which recurse rather than scan), the slot is released immediately since their children acquire their own slots. For skipped assets (no platform IDs), the slot is released after closing.

6. **Separation of concerns.** Each stage has a single responsibility and a minimal interface (`Add`/`Flush`, `Submit`/`Wait`). The tree walker knows nothing about upstream sync or worker pools. The batcher knows nothing about connections or scan execution. The dispatcher knows nothing about tree structure or batching.

### File layout

- `local_scanner.go` — tree walker (`scanContext`, `scanSubtree`), job orchestration (`distributeJob`).
- `scan_pipeline.go` — `syncBatcher` and `scanDispatcher` implementations, `scanSingleAsset`, panic reporting.

## Consequences

- Scan throughput improves significantly for large asset sets because workers stay busy continuously.
- The `parallelism` job setting controls scan concurrency; `maxConnections` is a separate, higher ceiling for connected-but-not-yet-scanning assets.
- The upstream sync batch size (10) is small enough to keep dispatch latency low while still avoiding per-asset API calls.
- `dispatcher.Wait()` at subtree boundaries ensures child scans complete before the parent node is scanned or closed, maintaining correct lifecycle ordering.
- The modular design makes each stage independently testable and easier to reason about.
