// Copyright Mondoo, Inc. 2026
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnspec/v13"
	"go.mondoo.com/cnspec/v13/cli/progress"
	"go.mondoo.com/cnspec/v13/policy"
	"go.mondoo.com/mql/v13/cli/config"
	"go.mondoo.com/mql/v13/discovery"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/upstream"
	"go.mondoo.com/mql/v13/providers-sdk/v1/upstream/health"
)

const (
	maxConnections = 50 // max assets connected (with runtimes open) at a time
	syncBatchSize  = 10 // how many assets to batch for upstream sync calls
)

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
	}
}

// Submit dispatches a single asset for scanning. It blocks if all worker
// slots are currently occupied. The asset's connSem slot is released after
// the scan completes and the asset is closed.
func (d *scanDispatcher) Submit(ctx context.Context, tracked *discovery.TrackedAsset) {
	d.scanSem <- struct{}{} // acquire scan slot
	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		defer func() { <-d.scanSem }()
		defer func() { <-d.connSem }()
		defer d.reportPanic(tracked)

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
			if err := d.explorer.CloseAsset(tracked); err != nil {
				log.Error().Err(err).Str("asset", tracked.Asset.Name).Msg("failed to close asset")
			}
			return
		}
		asset = updatedAsset
		tracked.Asset = asset

		if len(asset.PlatformIds) > 0 {
			d.multiprogress.AddTask(asset.PlatformIds[0], asset)
		}
		if syncErr := syncBatchWithUpstream(ctx, []*discovery.TrackedAsset{tracked}, d.services, d.spaceMrn, d.scanner.recording); syncErr != nil {
			d.reporter.AddScanError(asset, syncErr)
			if len(asset.PlatformIds) > 0 {
				d.multiprogress.Errored(asset.PlatformIds[0])
			}
			if err := d.explorer.CloseAsset(tracked); err != nil {
				log.Error().Err(err).Str("asset", tracked.Asset.Name).Msg("failed to close asset")
			}
			return
		}
	}

	if len(asset.PlatformIds) == 0 {
		log.Warn().Str("name", asset.Name).Msg("asset has no platform IDs after discovery, skipping")
		if err := d.explorer.CloseAsset(tracked); err != nil {
			log.Error().Err(err).Str("asset", tracked.Asset.Name).Msg("failed to close asset")
		}
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
	})

	// Report any recovered provider panics to the Mondoo Platform.
	for _, critErr := range runtime.CriticalErrors() {
		tags := map[string]string{
			"assetMrn":  asset.Mrn,
			"assetName": asset.Name,
		}
		if asset.Platform != nil {
			tags["platformIDs"] = strings.Join(asset.PlatformIds, ",")
			tags["assetPlatform"] = asset.Platform.Name
			tags["assetPlatformVersion"] = asset.Platform.Version
		}
		health.ReportError("cnspec", cnspec.Version, cnspec.Build, critErr.Error(), health.WithTags(tags))
	}

	// Close asset after scanning to free the gRPC connection.
	if err := d.explorer.CloseAsset(tracked); err != nil {
		log.Error().Err(err).Str("asset", tracked.Asset.Name).Msg("failed to close asset")
	}
}

// reportPanic captures panics from scan goroutines and reports them.
func (d *scanDispatcher) reportPanic(tracked *discovery.TrackedAsset) {
	health.ReportPanic("cnspec", cnspec.Version, cnspec.Build, func(product, version, build string, r any, stacktrace []byte) {
		opts, err := config.Read()
		if err != nil {
			log.Error().Err(err).Msg("failed to read config")
			return
		}

		serviceAccount := opts.GetServiceCredential()
		if serviceAccount == nil {
			log.Error().Msg("no service account configured")
			return
		}

		tags := map[string]string{
			"spaceMrn": d.spaceMrn,
		}
		if tracked != nil {
			tags["assetMrn"] = tracked.Asset.Mrn
			tags["assetName"] = tracked.Asset.Name
			tags["platformIDs"] = strings.Join(tracked.Asset.PlatformIds, ",")
			if tracked.Asset.Platform != nil {
				tags["assetPlatform"] = tracked.Asset.Platform.Name
				tags["assetPlatformVersion"] = tracked.Asset.Platform.Version
			}
		}

		event := &health.SendErrorReq{
			ServiceAccountMrn: opts.ServiceAccountMrn,
			AgentMrn:          opts.AgentMrn,
			Product: &health.ProductInfo{
				Name:    product,
				Version: version,
				Build:   build,
			},
			Error: &health.ErrorInfo{
				Message:    "panic: " + fmt.Sprintf("%v -- %v", r, tags),
				Stacktrace: string(stacktrace),
			},
		}

		sendErrorToMondooPlatform(serviceAccount, event)
		log.Info().Msg("reported panic to Mondoo Platform")
	})
}
