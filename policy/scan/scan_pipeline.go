// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"context"
	"fmt"
	"os"
	goruntime "runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnspec/v13"
	"go.mondoo.com/cnspec/v13/cli/progress"
	"go.mondoo.com/cnspec/v13/internal/syslimits"
	"go.mondoo.com/cnspec/v13/policy"
	"go.mondoo.com/mql/v13/discovery"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/upstream"
	"go.mondoo.com/mql/v13/providers-sdk/v1/upstream/health"
)

const (
	defaultMaxConnections = 50
	syncBatchSize         = 5 // how many assets to batch for upstream sync calls

	// estimatedMemoryPerConnectionBytes is a conservative estimate of the memory
	// a single concurrent provider connection can consume — the provider
	// subprocess plus cnspec-side per-asset data. Provider memory lives in
	// separate processes, outside the Go heap that GOMEMLIMIT governs, so in a
	// memory-limited cgroup we use this to bound how many providers run at once
	// and keep the whole process group under the limit. It is deliberately
	// rough; operators can always override the result via
	// MONDOO_MAX_PROVIDER_CONNECTIONS.
	estimatedMemoryPerConnectionBytes = 200 * 1024 * 1024
)

// getMaxConnections returns the cap on simultaneously connected assets (and
// therefore concurrent provider subprocesses). An explicit
// MONDOO_MAX_PROVIDER_CONNECTIONS always wins. Otherwise it starts from
// defaultMaxConnections and, when running under a cgroup memory limit, reduces
// the cap so the aggregate memory of the provider subprocesses is unlikely to
// exceed the limit — GOMEMLIMIT only bounds cnspec's own heap, not the
// providers.
func getMaxConnections() int {
	if v := os.Getenv("MONDOO_MAX_PROVIDER_CONNECTIONS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}

	maxConn := defaultMaxConnections
	if limit := syslimits.Detect().MemoryLimitBytes; limit > 0 {
		if budget := connectionsForMemory(limit); budget < maxConn {
			log.Info().
				Uint64("memory_limit_bytes", limit).
				Int("max_connections", budget).
				Int("default", defaultMaxConnections).
				Msg("reducing max provider connections to fit cgroup memory limit")
			maxConn = budget
		}
	}
	return maxConn
}

// connectionsForMemory converts a cgroup memory limit into the number of
// concurrent provider connections that should fit within it, using a
// conservative per-connection estimate. It never returns less than 1 and never
// more than defaultMaxConnections. A limit of 0 (unlimited) yields the default.
func connectionsForMemory(limitBytes uint64) int {
	if limitBytes == 0 {
		return defaultMaxConnections
	}
	budget := int(limitBytes / estimatedMemoryPerConnectionBytes)
	if budget < 1 {
		budget = 1
	}
	if budget > defaultMaxConnections {
		budget = defaultMaxConnections
	}
	return budget
}

// ---------------------------------------------------------------------------
// syncBatcher: accumulates connected assets and syncs them with upstream
// in batches.
// ---------------------------------------------------------------------------

// syncBatcher collects connected assets and calls syncBatchWithUpstream once
// the batch reaches syncBatchSize. The caller can force a flush at any time
// (e.g. before recursing into a branch or draining at end-of-subtree).
// After syncing, assets are forwarded to a scanDispatcher for execution.
type syncBatcher struct {
	dispatcher    *scanDispatcher
	services      *policy.Services
	spaceMrn      string
	recording     llx.Recording
	multiprogress progress.MultiProgress

	buf []*discovery.TrackedAsset
}

func newSyncBatcher(dispatcher *scanDispatcher, services *policy.Services, spaceMrn string, rec llx.Recording, mp progress.MultiProgress) *syncBatcher {
	return &syncBatcher{
		dispatcher:    dispatcher,
		services:      services,
		spaceMrn:      spaceMrn,
		recording:     rec,
		multiprogress: mp,
	}
}

// Add appends an asset to the batch. If the batch is full, it is
// automatically flushed (synced and dispatched).
func (sb *syncBatcher) Add(ctx context.Context, tracked *discovery.TrackedAsset) error {
	sb.buf = append(sb.buf, tracked)
	if len(sb.buf) >= syncBatchSize {
		return sb.Flush(ctx)
	}
	return nil
}

// Flush syncs all buffered assets with upstream and dispatches them for
// scanning. It is a no-op if the buffer is empty.
func (sb *syncBatcher) Flush(ctx context.Context) error {
	if len(sb.buf) == 0 {
		return nil
	}
	batch := sb.buf
	sb.buf = nil

	// Split delayed-discovery assets — they can't be synced until the scan
	// goroutine resolves them via HandleDelayedDiscovery.
	var readyToSync []*discovery.TrackedAsset
	for _, tracked := range batch {
		asset := tracked.Asset
		isDelayed := len(asset.Connections) > 0 && asset.Connections[0].DelayDiscovery
		if !isDelayed {
			if len(asset.PlatformIds) > 0 {
				sb.multiprogress.AddTask(asset.PlatformIds[0], asset)
			}
			readyToSync = append(readyToSync, tracked)
		}
	}

	if len(readyToSync) > 0 {
		if err := syncBatchWithUpstream(ctx, readyToSync, sb.services, sb.spaceMrn, sb.recording); err != nil {
			for _, tracked := range batch {
				assetName := ""
				if tracked.Asset != nil {
					assetName = tracked.Asset.Name
				}
				if closeErr := sb.dispatcher.explorer.CloseAsset(tracked); closeErr != nil {
					log.Error().Err(closeErr).Str("asset", assetName).Msg("failed to close asset")
				}
				tracked.Asset = nil
				<-sb.dispatcher.connSem
			}
			return err
		}
	}

	// Hand each synced asset to the dispatcher for scanning.
	for _, tracked := range batch {
		sb.dispatcher.Submit(ctx, tracked)
	}

	return nil
}

// ---------------------------------------------------------------------------
// scanDispatcher: manages a pool of scan workers.
// ---------------------------------------------------------------------------

// scanDispatcher owns a bounded pool of scan workers. Assets are submitted
// via Submit and executed concurrently up to the configured parallelism.
// The caller uses Wait to block until all submitted scans have completed.
type scanDispatcher struct {
	scanSem chan struct{}
	connSem chan struct{}
	wg      sync.WaitGroup

	// Dependencies for scanning a single asset.
	scanner       *LocalScanner
	explorer      *discovery.AssetExplorer
	job           *Job
	upstream      *upstream.UpstreamConfig
	reporter      Reporter
	multiprogress progress.MultiProgress
	services      *policy.Services
	spaceMrn      string
	scannedAssets *atomic.Int64
	failures      *failureReporter
}

func newScanDispatcher(
	parallelism int,
	connSem chan struct{},
	scanner *LocalScanner,
	explorer *discovery.AssetExplorer,
	job *Job,
	up *upstream.UpstreamConfig,
	reporter Reporter,
	mp progress.MultiProgress,
	services *policy.Services,
	spaceMrn string,
	scannedAssets *atomic.Int64,
	failures *failureReporter,
) *scanDispatcher {
	return &scanDispatcher{
		scanSem:       make(chan struct{}, parallelism),
		connSem:       connSem,
		scanner:       scanner,
		explorer:      explorer,
		job:           job,
		upstream:      up,
		reporter:      reporter,
		multiprogress: mp,
		services:      services,
		spaceMrn:      spaceMrn,
		scannedAssets: scannedAssets,
		failures:      failures,
	}
}

// Submit queues an asset for scanning. It returns immediately — the
// goroutine waits for a worker slot (scanSem) internally. This lets
// the batcher and tree walker continue without blocking on worker
// availability. The asset's connSem slot is released after the scan
// completes and the asset is closed.
func (d *scanDispatcher) Submit(ctx context.Context, tracked *discovery.TrackedAsset) {
	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		defer func() { <-d.connSem }()
		// Recover panics from this asset's scan directly here (recover only
		// works in a function deferred by the panicking goroutine). Reporting
		// and re-raising elsewhere would let the panic escape this bare
		// goroutine and crash the whole process, taking down every other
		// in-flight asset scan — exactly what must not happen in a critical
		// environment. Instead we contain it, report it, and keep scanning.
		defer func() {
			if r := recover(); r != nil {
				d.recoverAssetPanic(tracked, r, debug.Stack())
			}
		}()

		// Acquire worker slot, respecting context cancellation.
		select {
		case d.scanSem <- struct{}{}:
			defer func() { <-d.scanSem }()
		case <-ctx.Done():
			assetName := ""
			if tracked.Asset != nil {
				assetName = tracked.Asset.Name
			}
			if err := d.explorer.CloseAsset(tracked); err != nil {
				log.Error().Err(err).Str("asset", assetName).Msg("failed to close asset")
			}
			tracked.Asset = nil
			return
		}

		// Mark asset as in-progress as soon as we pick it up, not when
		// the first query result comes back.
		if tracked.Asset != nil && len(tracked.Asset.PlatformIds) > 0 {
			d.multiprogress.OnProgress(tracked.Asset.PlatformIds[0], 0)
		}

		d.scanSingleAsset(ctx, tracked)
	}()
}

// Wait blocks until all submitted scans have completed.
func (d *scanDispatcher) Wait() {
	d.wg.Wait()
}

// scanSingleAsset handles the full lifecycle of scanning one asset: delayed
// discovery, validation, scanning, error reporting, and closing.
func (d *scanDispatcher) scanSingleAsset(ctx context.Context, tracked *discovery.TrackedAsset) {
	asset := tracked.Asset
	runtime := tracked.Runtime

	if err := runtime.EnsureProvidersConnected(); err != nil {
		log.Error().Err(err).Msg("could not connect to providers")
	}

	log.Debug().Interface("platform", asset.Platform).Str("name", asset.Name).Msg("start scan")

	// Handle delayed discovery (e.g. container registry images).
	if len(asset.Connections) > 0 && asset.Connections[0].DelayDiscovery {
		updatedAsset, err := discovery.HandleDelayedDiscovery(ctx, asset, runtime)
		if err != nil {
			d.reporter.AddScanError(asset, err)
			d.failures.report(asset, err)
			if err := d.explorer.CloseAsset(tracked); err != nil {
				log.Error().Err(err).Str("asset", asset.Name).Msg("failed to close asset")
			}
			tracked.Asset = nil
			return
		}
		asset = updatedAsset
		tracked.Asset = asset

		if len(asset.PlatformIds) > 0 {
			d.multiprogress.AddTask(asset.PlatformIds[0], asset)
		}
		if syncErr := syncBatchWithUpstream(ctx, []*discovery.TrackedAsset{tracked}, d.services, d.spaceMrn, d.scanner.recording); syncErr != nil {
			d.reporter.AddScanError(asset, syncErr)
			d.failures.report(asset, syncErr)
			if len(asset.PlatformIds) > 0 {
				d.multiprogress.Errored(asset.PlatformIds[0])
			}
			if err := d.explorer.CloseAsset(tracked); err != nil {
				log.Error().Err(err).Str("asset", asset.Name).Msg("failed to close asset")
			}
			tracked.Asset = nil
			return
		}
	}

	if len(asset.PlatformIds) == 0 {
		log.Warn().Str("name", asset.Name).Msg("asset has no platform IDs after discovery, skipping")
		if err := d.explorer.CloseAsset(tracked); err != nil {
			log.Error().Err(err).Str("asset", asset.Name).Msg("failed to close asset")
		}
		tracked.Asset = nil
		return
	}

	d.scannedAssets.Add(1)
	p := &progress.MultiProgressAdapter{Key: asset.PlatformIds[0], Multi: d.multiprogress}
	d.scanner.RunAssetJob(&AssetJob{
		DoRecord:         d.job.DoRecord,
		UpstreamConfig:   d.upstream,
		Asset:            asset,
		Bundle:           d.job.Bundle,
		Props:            d.job.Props,
		PolicyFilters:    preprocessPolicyFilters(d.job.PolicyFilters),
		Ctx:              ctx,
		Reporter:         d.reporter,
		ProgressReporter: p,
		runtime:          runtime,
		failures:         d.failures,
	})

	// Report any recovered provider panics to the Mondoo Platform.
	for _, critErr := range runtime.CriticalErrors() {
		tags := assetErrorTags(d.spaceMrn, asset)
		health.ReportError("cnspec", cnspec.Version, cnspec.Build, critErr.Error(), health.WithTags(tags))
	}

	// Close asset after scanning to free the gRPC connection.
	if err := d.explorer.CloseAsset(tracked); err != nil {
		log.Error().Err(err).Str("asset", asset.Name).Msg("failed to close asset")
	}

	// Break the allAssets → Asset reference so the proto data (connections,
	// platform, labels, etc.) becomes GC-eligible immediately. The local
	// `asset` variable still holds a reference for logMemoryStats below;
	// reporters that need the data (AggregateReporter) already captured
	// their own reference via AddReport.
	tracked.Asset = nil

	debug.FreeOSMemory()

	if os.Getenv("DEBUG_PROVIDER_MEMORY") != "" {
		d.logMemoryStats(asset)
	}
}

// logMemoryStats emits memory diagnostics after an asset scan completes.
// Only called when DEBUG_PROVIDER_MEMORY is set.
func (d *scanDispatcher) logMemoryStats(asset *inventory.Asset) {
	var m goruntime.MemStats
	goruntime.ReadMemStats(&m)

	ev := log.Info().
		Str("asset", asset.Name).
		Int64("scanned_assets", d.scannedAssets.Load()).
		Int("goroutines", goruntime.NumGoroutine()).
		Uint64("heap_alloc_mb", m.Alloc/1024/1024).
		Uint64("heap_sys_mb", m.HeapSys/1024/1024).
		Uint64("total_alloc_mb", m.TotalAlloc/1024/1024)

	if cgroup, err := os.ReadFile("/sys/fs/cgroup/memory.current"); err == nil {
		ev = ev.Str("cgroup_bytes", strings.TrimSpace(string(cgroup)))
	}
	if cgroupMax, err := os.ReadFile("/sys/fs/cgroup/memory.max"); err == nil {
		ev = ev.Str("cgroup_max", strings.TrimSpace(string(cgroupMax)))
	}

	ev.Msg("memory after asset scan")

	if ar, ok := d.reporter.(*AggregateReporter); ok {
		rptBytes, rpBytes, vrBytes, aBytes := ar.AccumulatedBytes()
		log.Info().
			Int("reports", rptBytes).
			Int("resolved_policies", rpBytes).
			Int("vuln_reports", vrBytes).
			Int("assets", aBytes).
			Int("total", rptBytes+rpBytes+vrBytes+aBytes).
			Msg("reporter accumulated bytes")
	}
}

// recoverAssetPanic handles a panic recovered from an asset's scan goroutine.
// It reports the panic upstream WITHOUT re-raising it, records the asset as a
// scan failure, marks it errored in the progress bar, and releases the asset's
// provider connection — so a single asset's panic degrades to one failed asset
// instead of crashing the entire scan.
func (d *scanDispatcher) recoverAssetPanic(tracked *discovery.TrackedAsset, r any, stacktrace []byte) {
	var asset *inventory.Asset
	if tracked != nil {
		asset = tracked.Asset
	}
	assetName := ""
	if asset != nil {
		assetName = asset.Name
	}

	log.Error().
		Interface("panic", r).
		Str("asset", assetName).
		Bytes("stacktrace", stacktrace).
		Msg("recovered from panic during asset scan; continuing with remaining assets")

	// Report the panic upstream with asset/space tags. ReportRecoveredPanic
	// (unlike ReportPanic) does not re-panic, which is what lets us keep going.
	tags := assetErrorTags(d.spaceMrn, asset)
	health.ReportRecoveredPanic("cnspec", cnspec.Version, cnspec.Build, r, stacktrace, tags)

	// Surface the panic as a scan error so it appears in the report and drives
	// a non-zero exit code, and mark it failed in the progress bar.
	if asset != nil {
		d.reporter.AddScanError(asset, fmt.Errorf("panic during scan: %v", r))
		if len(asset.PlatformIds) > 0 {
			d.multiprogress.Errored(asset.PlatformIds[0])
		}
	}

	// Best-effort close so the provider connection and its goroutines are
	// released; the connSem slot itself is released by Submit's defer. Guard the
	// explorer: this is a crash handler, so it must not itself nil-panic.
	if tracked != nil && tracked.Asset != nil {
		if d.explorer != nil {
			if err := d.explorer.CloseAsset(tracked); err != nil {
				log.Error().Err(err).Str("asset", assetName).Msg("failed to close asset after panic")
			}
		}
		tracked.Asset = nil
	}
}
