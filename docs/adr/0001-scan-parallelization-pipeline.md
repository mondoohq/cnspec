# ADR-0001: Scan Parallelization Pipeline

**Date:** 2026-04-03
**Status:** Accepted

## Context

cnspec scans can discover hundreds or thousands of assets (e.g. container images in a registry, pods across Kubernetes namespaces). The scanner walks an asset tree depth-first: root nodes (like a K8s cluster) contain branch nodes (namespaces) which contain leaf nodes (individual workloads).

The original implementation used a batch-and-wait model: accumulate up to 50 assets, synchronize them with upstream, scan the entire batch (with bounded parallelism), wait for all scans to finish, then start the next batch. This created idle periods where scan workers had nothing to do while the last few slow assets in a batch completed.

## Decision

Replace the batch-and-wait model with a continuous pipeline that separates three concerns — connection limiting, upstream synchronization, and scan execution — each with its own bound.

### Three independent controls

| Concern | Mechanism | Default |
|---|---|---|
| **Max open connections** | `connSem` — a buffered channel of size `maxConnections` | 50 |
| **Upstream sync batching** | Small buffer (`syncBatchSize`) flushed to `syncBatchWithUpstream` | 10 |
| **Concurrent scans** | `scanSem` — a buffered channel of size `parallelism` | Configured per job |

### Flow

```
for each child in node.Children:
    connSem <- acquire            # block if 50 connections open
    connected = Connect(child)

    if branch node:
        release connSem           # branches don't hold connections
        flush syncBatch           # sync + dispatch any pending leaves
        recurse into subtree
    else (leaf):
        append to syncBatch
        if len(syncBatch) >= 10:
            syncBatchWithUpstream(syncBatch)
            for each asset in syncBatch:
                scanSem <- acquire   # block if all scan workers busy
                go scanSingleAsset() # releases scanSem + connSem on completion

scanWg.Wait()                     # drain all in-flight scans before closing parent
```

### Key properties

1. **No idle workers.** A scan worker picks up the next ready asset as soon as it finishes the current one. There is no batch boundary that forces workers to wait.

2. **Bounded resource usage.** The connection semaphore (`maxConnections = 50`) caps the number of simultaneously open provider runtimes/gRPC connections. This is independent of scan parallelism — you can have 50 connected assets with only 8 actively scanning.

3. **Efficient upstream sync.** Assets are still batched (in groups of 10) for the `syncBatchWithUpstream` network call, avoiding per-asset round-trips. But after syncing, each asset is immediately dispatched — we don't wait for the batch to finish scanning before syncing the next one.

4. **Depth-first tree walk preserved.** Branch nodes flush the current sync batch and recurse, so only one branch's children are being connected at a time. This keeps memory usage predictable for deep trees (e.g. cluster > namespace > pod).

5. **Connection slot lifecycle.** A connection slot is acquired before `Connect()` and released after `scanSingleAsset()` closes the asset. For branch nodes (which recurse rather than scan), the slot is released immediately since their children acquire their own slots. For skipped assets (no platform IDs), the slot is released after closing.

## Consequences

- Scan throughput improves significantly for large asset sets because workers stay busy continuously.
- The `parallelism` job setting controls scan concurrency as before; `maxConnections` is a separate, higher ceiling for connected-but-not-yet-scanning assets.
- The upstream sync batch size (10) is small enough to keep dispatch latency low while still avoiding per-asset API calls.
- `scanWg.Wait()` at the end of each subtree ensures child scans complete before the parent node is scanned or closed, maintaining correct lifecycle ordering.
