// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/mattn/go-isatty"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/ksuid"
	"go.mondoo.com/cnspec/v13"
	"go.mondoo.com/cnspec/v13/cli/progress"
	"go.mondoo.com/cnspec/v13/internal/datalakes/inmemory"
	"go.mondoo.com/cnspec/v13/internal/datalakes/sqlite"
	"go.mondoo.com/cnspec/v13/policy"
	"go.mondoo.com/cnspec/v13/policy/executor"
	"go.mondoo.com/mql/v13"
	"go.mondoo.com/mql/v13/cli/config"
	"go.mondoo.com/mql/v13/cli/execruntime"
	"go.mondoo.com/mql/v13/discovery"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/logger"
	"go.mondoo.com/mql/v13/mqlc"
	"go.mondoo.com/mql/v13/mrn"
	"go.mondoo.com/mql/v13/providers"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/recording"
	"go.mondoo.com/mql/v13/providers-sdk/v1/upstream"
	"go.mondoo.com/mql/v13/providers-sdk/v1/upstream/gql"
	"go.mondoo.com/mql/v13/providers-sdk/v1/upstream/health"
	"go.mondoo.com/mql/v13/utils/multierr"
	ranger "go.mondoo.com/ranger-rpc"
	"go.mondoo.com/ranger-rpc/codes"
	"go.mondoo.com/ranger-rpc/status"
	"google.golang.org/protobuf/proto"
)

const (
	defaultRefreshInterval = 3600
)

type LocalScanner struct {
	queue           *diskQueueClient
	ctx             context.Context
	fetcher         *fetcher
	upstream        *upstream.UpstreamConfig
	_upstreamClient *upstream.UpstreamClient
	recording       llx.Recording
	runtime         llx.Runtime

	// allows setting the upstream credentials from a job
	allowJobCredentials bool
	disableProgressBar  bool
	reportType          ReportType
	autoUpdate          bool
	refreshInterval     int
}

type ScannerOption func(*LocalScanner)

func WithUpstream(conf *upstream.UpstreamConfig) ScannerOption {
	return func(s *LocalScanner) {
		s.upstream = conf
	}
}

func WithRecording(r llx.Recording) func(s *LocalScanner) {
	return func(s *LocalScanner) {
		s.recording = r
	}
}

func AllowJobCredentials() ScannerOption {
	return func(s *LocalScanner) {
		s.allowJobCredentials = true
	}
}

func DisableProgressBar() ScannerOption {
	return func(s *LocalScanner) {
		s.disableProgressBar = true
	}
}

func WithReportType(reportType ReportType) ScannerOption {
	return func(s *LocalScanner) {
		s.reportType = reportType
	}
}

func WithAutoUpdate(onoff bool) ScannerOption {
	return func(s *LocalScanner) {
		s.autoUpdate = onoff
	}
}

func WithRefreshInterval(refreshInterval int) ScannerOption {
	return func(s *LocalScanner) {
		s.refreshInterval = refreshInterval
	}
}

func WithRuntime(r *providers.Runtime) ScannerOption {
	return func(s *LocalScanner) {
		s.runtime = r
	}
}

func NewLocalScanner(opts ...ScannerOption) *LocalScanner {
	ls := &LocalScanner{
		fetcher: newFetcher(),
		ctx:     context.Background(),
		// By default, auto-update is enabled. It can be explicitly disabled
		// by passing WithAutoUpdate(false)
		autoUpdate: true,
	}

	for i := range opts {
		opts[i](ls)
	}

	if ls.recording == nil {
		ls.recording = recording.Null{}
	}

	if ls.runtime == nil {
		runtime := providers.DefaultRuntime()
		refreshInterval := defaultRefreshInterval
		if ls.refreshInterval > 0 {
			refreshInterval = ls.refreshInterval
		}

		runtime.AutoUpdate = providers.UpdateProvidersConfig{
			Enabled:         ls.autoUpdate,
			RefreshInterval: refreshInterval,
		}
		ls.runtime = runtime
	}

	return ls
}

func (s *LocalScanner) EnableQueue() error {
	var err error
	s.queue, err = newDqueClient(defaultDqueConfig, func(job *Job) {
		// this is the handler for jobs, when they are picked up
		ctx := mql.SetFeatures(s.ctx, mql.DefaultFeatures)
		_, err := s.Run(ctx, job)
		if err != nil {
			log.Error().Err(err).Msg("could not complete the scan")
		}
	})
	return err
}

func (s *LocalScanner) Schedule(ctx context.Context, job *Job) (*Empty, error) {
	if job == nil {
		return nil, status.Errorf(codes.InvalidArgument, "missing scan job")
	}

	if s.queue == nil {
		return nil, status.Errorf(codes.Unavailable, "job queue is not available")
	}

	s.queue.Channel() <- *job
	return &Empty{}, nil
}

func (s *LocalScanner) Run(ctx context.Context, job *Job) (*ScanResult, error) {
	if job == nil {
		return nil, status.Errorf(codes.InvalidArgument, "missing scan job")
	}

	if job.Inventory == nil {
		return nil, status.Errorf(codes.InvalidArgument, "missing inventory")
	}

	if ctx == nil {
		return nil, errors.New("no context provided to run job with local scanner")
	}

	upstream, err := s.getUpstreamConfig(false, job)
	if err != nil {
		return nil, err
	}

	// The job report type has precedence over the global report type. The default is FULL
	if job.ReportType == ReportType_FULL {
		job.ReportType = s.reportType
	}

	reports, err := s.distributeJob(job, ctx, upstream)
	if err != nil {
		return nil, err
	}

	return reports, nil
}

func (s *LocalScanner) RunIncognito(ctx context.Context, job *Job) (*ScanResult, error) {
	if job == nil {
		return nil, status.Errorf(codes.InvalidArgument, "missing scan job")
	}

	if job.Inventory == nil {
		return nil, status.Errorf(codes.InvalidArgument, "missing inventory")
	}

	if ctx == nil {
		return nil, errors.New("no context provided to run job with local scanner")
	}

	upstream, err := s.getUpstreamConfig(true, job)
	if err != nil {
		return nil, err
	}

	// The job report type has precedence over the global report type. The default is FULL
	if job.ReportType == ReportType_FULL {
		job.ReportType = s.reportType
	}

	reports, err := s.distributeJob(job, ctx, upstream)
	if err != nil {
		return nil, err
	}

	return reports, nil
}

// preprocessPolicyFilters expends short registry mrns into full mrns
func preprocessPolicyFilters(filters []string) []string {
	res := make([]string, len(filters))
	for i := range filters {
		f := filters[i]
		if strings.HasPrefix(f, "//") {
			res[i] = f
			continue
		}

		// expand short registry mrns
		m := strings.Split(f, "/")
		if len(m) == 2 {
			res[i] = policy.NewPolicyMrn(m[0], m[1])
		} else {
			res[i] = f
		}
	}
	return res
}

func createReporter(ctx context.Context, job *Job, upstream *upstream.UpstreamConfig) (Reporter, error) {
	var reporter Reporter
	switch job.ReportType {
	case ReportType_FULL:
		reporter = NewAggregateReporter()

		// case where users pass in a bundle directly via a file
		if job.Bundle != nil {
			reporter.AddBundle(job.Bundle)
			return reporter, nil
		}

		// - pass in bundle via file
		// - use Mondoo Platform upstream with/without incognito
		// - bundles fetched from public registry (not covered here, but in ensureBundle)
		//
		// if we use upstream with/without incognito, we want to fetch the bundle here to ensure we only fetch it once
		// for all assets in the same space
		if upstream != nil && upstream.Creds != nil {
			client, err := upstream.InitClient(ctx)
			if err != nil {
				return nil, err
			}

			services, err := policy.NewRemoteServices(client.ApiEndpoint, client.Plugins, client.HttpClient)
			if err != nil {
				return nil, err
			}

			// retrieve the bundle for the parent (which is the space). That bundle contains all policies, queries and checks
			bundle, err := services.GetBundle(ctx, &policy.Mrn{Mrn: upstream.Creds.ParentMrn}) //nolint:staticcheck // SA1019: ParentMrn is deprecated but still used for backward compatibility
			if err != nil {
				return nil, err
			}
			for i := range bundle.Policies {
				if bundle.Policies[i].Version == "n/a" {
					bundle.Policies[i].Version = "0.0.0" // space policy has no version but we need it to compile it
				}
			}
			job.Bundle = bundle // also update the job with the fetched bundle
			reporter.AddBundle(bundle)
		}
	case ReportType_ERROR:
		reporter = NewErrorReporter()
	case ReportType_NONE:
		reporter = NewNoOpReporter()
	default:
		return nil, errors.Errorf("unknown report type: %s", job.ReportType)
	}
	return reporter, nil
}

func (s *LocalScanner) distributeJob(job *Job, ctx context.Context, upstream *upstream.UpstreamConfig) (*ScanResult, error) {
	reporter, err := createReporter(ctx, job, upstream)
	if err != nil {
		return nil, err
	}

	// Ensure all required providers are installed before we try to run the scan.
	// We only check here for bundles fetched from the server. Local policy bundles
	// already have their requirements ensured before compilation in loadPolicies().
	if job.Bundle != nil && upstream != nil && upstream.Creds != nil && job.Bundle.HasRequirements() {
		if err := job.Bundle.EnsureRequirements(s.autoUpdate); err != nil {
			return nil, errors.Wrap(err, "failed to ensure policy requirements")
		}
	}

	// Enable staged discovery on root inventory assets so that providers
	// supporting it can split discovery into phases (e.g. cluster first,
	// workloads per namespace later), matching the batched scan approach.
	for _, asset := range job.Inventory.Spec.Assets {
		for _, conf := range asset.Connections {
			if conf.Options == nil {
				conf.Options = map[string]string{}
			}
			conf.Options[plugin.OptionStagedDiscovery] = ""
		}
	}

	log.Info().Msgf("discover related assets for %d asset(s)", len(job.Inventory.Spec.Assets))
	explorer, err := discovery.NewAssetExplorer(ctx, discovery.AssetExplorerConfig{
		Inventory: job.Inventory,
		Upstream:  upstream,
		Recording: s.recording,
	})
	if err != nil {
		return nil, err
	}
	defer explorer.Shutdown()

	// Report initial discovery errors
	for _, assetErr := range explorer.Errors() {
		reporter.AddScanError(assetErr.Asset, assetErr.Err)
	}

	if len(explorer.Connected()) == 0 {
		return reporter.Reports(), nil
	}

	multiprogress, err := createProgressBar(s.disableProgressBar)
	if err != nil {
		return nil, err
	}

	// start the progress bar
	scanGroups := sync.WaitGroup{}
	scanGroups.Add(1)
	go func() {
		defer scanGroups.Done()
		defer health.ReportPanic("cnspec", cnspec.Version, cnspec.Build)

		if err := multiprogress.Open(); err != nil {
			log.Error().Err(err).Msg("failed to open progress bar")
		}
	}()
	// Make sure the progress bar is closed when we exit early. Calling this multiple times
	// is safe
	defer multiprogress.Close()

	spaceMrn := ""
	var services *policy.Services
	if upstream != nil && upstream.ApiEndpoint != "" && !upstream.Incognito {
		client, err := upstream.InitClient(ctx)
		if err != nil {
			return nil, err
		}
		spaceMrn = client.SpaceMrn

		services, err = policy.NewRemoteServices(client.ApiEndpoint, client.Plugins, client.HttpClient)
		if err != nil {
			return nil, err
		}
	}

	scanCtx := &scanContext{
		scanner:       s,
		explorer:      explorer,
		job:           job,
		upstream:      upstream,
		reporter:      reporter,
		multiprogress: multiprogress,
		services:      services,
		spaceMrn:      spaceMrn,
	}

	// Process each root asset's subtree. The root is already connected by
	// NewAssetExplorer; scanSubtree will connect children one branch at a
	// time so only one parent's children data is in memory at once.
	for _, root := range explorer.Connected() {
		// Register the root and its already-discovered children in the
		// progress bar so the TODO list knows how much work is pending
		// and doesn't close prematurely.
		if len(root.Asset.PlatformIds) > 0 {
			multiprogress.AddTask(root.Asset.PlatformIds[0], root.Asset)
		}
		for _, child := range root.Children {
			if len(child.Asset.PlatformIds) > 0 {
				multiprogress.AddTask(child.Asset.PlatformIds[0], child.Asset)
			}
		}

		if err := scanCtx.scanSubtree(ctx, root); err != nil {
			return nil, err
		}
	}

	scanGroups.Wait() // wait for the progress bar to finish
	return reporter.Reports(), nil
}

const batchSize = 50

// scanContext holds the shared state needed while recursively walking the
// asset tree. It avoids threading many parameters through every call.
type scanContext struct {
	scanner       *LocalScanner
	explorer      *discovery.AssetExplorer
	job           *Job
	upstream      *upstream.UpstreamConfig
	reporter      Reporter
	multiprogress progress.MultiProgress
	services      *policy.Services
	spaceMrn      string
}

// scanSubtree processes a single connected node's subtree depth-first.
// It connects each child one at a time. If a child has its own children
// (i.e. it is a branch node like a namespace), the current leaf batch is
// flushed and we recurse into that child before touching any sibling.
// This guarantees only one branch node's children data is in memory at a time.
// After all children are handled, the node itself is scanned (if it has
// platform IDs) and then closed.
func (sc *scanContext) scanSubtree(ctx context.Context, node *discovery.TrackedAsset) error {
	var leafBatch []*discovery.TrackedAsset

	for _, child := range node.Children {
		// Check for cancellation before connecting the next child
		select {
		case <-ctx.Done():
			sc.multiprogress.Close()
			return ctx.Err()
		default:
		}

		// Connect the child — creates its runtime and discovers its children
		connected, err := sc.explorer.Connect(child)
		if err != nil {
			if !errors.Is(err, discovery.ErrDuplicateAsset) {
				sc.reporter.AddScanError(child.Asset, err)
			}
			continue
		}

		if len(connected.Asset.PlatformIds) == 0 {
			log.Warn().Str("name", connected.Asset.Name).Msg("asset has no platform IDs, skipping")
			if err := sc.explorer.CloseAsset(connected); err != nil {
				log.Error().Err(err).Str("asset", connected.Asset.Name).Msg("failed to close asset")
			}
			continue
		}

		if len(connected.Children) > 0 {
			// Branch node (e.g. a namespace with workloads under it).
			// Flush any pending leaf batch first, then recurse.
			if len(leafBatch) > 0 {
				if err := sc.syncAndScanBatch(ctx, leafBatch); err != nil {
					return err
				}
				leafBatch = nil
			}
			if err := sc.scanSubtree(ctx, connected); err != nil {
				return err
			}
			continue
		}

		// Leaf node — accumulate into the current batch
		leafBatch = append(leafBatch, connected)
		if len(leafBatch) >= batchSize {
			if err := sc.syncAndScanBatch(ctx, leafBatch); err != nil {
				return err
			}
			leafBatch = nil
		}
	}

	// Flush remaining leaves
	if len(leafBatch) > 0 {
		if err := sc.syncAndScanBatch(ctx, leafBatch); err != nil {
			return err
		}
	}

	// Scan the node itself (e.g. the namespace), then close to free resources.
	// Some nodes are pure gateways with no platform IDs (e.g. the k8s cluster
	// root with staged discovery) — just close them without scanning.
	if len(node.Asset.PlatformIds) > 0 {
		if err := sc.syncAndScanBatch(ctx, []*discovery.TrackedAsset{node}); err != nil {
			return err
		}
	} else {
		if err := sc.explorer.CloseAsset(node); err != nil {
			log.Error().Err(err).Str("asset", node.Asset.Name).Msg("failed to close asset")
		}
	}

	return nil
}

// syncAndScanBatch synchronizes, scans, and closes a batch of assets.
func (sc *scanContext) syncAndScanBatch(ctx context.Context, batch []*discovery.TrackedAsset) error {
	// Split the batch: assets with DelayDiscovery may not have platform IDs yet,
	// so we can't register them in the progress bar or sync them with upstream
	// until HandleDelayedDiscovery resolves them in the scan loop below.
	var readyToSync []*discovery.TrackedAsset
	for _, tracked := range batch {
		asset := tracked.Asset
		isDelayed := len(asset.Connections) > 0 && asset.Connections[0].DelayDiscovery
		if !isDelayed {
			if len(asset.PlatformIds) > 0 {
				sc.multiprogress.AddTask(asset.PlatformIds[0], asset)
			}
			readyToSync = append(readyToSync, tracked)
		}
	}

	// Synchronize only non-delayed assets with upstream or assign local MRNs.
	// Delayed assets are synced individually after HandleDelayedDiscovery.
	if len(readyToSync) > 0 {
		if err := syncBatchWithUpstream(ctx, readyToSync, sc.services, sc.spaceMrn, sc.scanner.recording); err != nil {
			return err
		}
	}

	// Scan each asset in a goroutine with panic reporting, then close it.
	// scanErr captures any error (e.g. context cancellation) from the goroutine
	// so it can be propagated to the caller.
	var scanErr error
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer health.ReportPanic("cnspec", cnspec.Version, cnspec.Build, func(product, version, build string, r any, stacktrace []byte) {
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
				"spaceMrn": sc.spaceMrn,
			}
			if len(batch) > 0 {
				tags["assetMrn"] = batch[0].Asset.Mrn
				tags["assetName"] = batch[0].Asset.Name
				tags["platformIDs"] = strings.Join(batch[0].Asset.PlatformIds, ",")
				tags["assetPlatform"] = batch[0].Asset.Platform.Name
				tags["assetPlatformVersion"] = batch[0].Asset.Platform.Version
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

		for i, tracked := range batch {
			asset := tracked.Asset
			runtime := tracked.Runtime

			if err := runtime.EnsureProvidersConnected(); err != nil {
				log.Error().Err(err).Msg("could not connect to providers")
			}

			log.Debug().Interface("platform", asset.Platform).Str("name", asset.Name).Msg("start scan")

			select {
			case <-ctx.Done():
				log.Warn().Msg("request context has been canceled")
				// Close all remaining unscanned assets in the batch
				for _, remaining := range batch[i:] {
					if err := sc.explorer.CloseAsset(remaining); err != nil {
						log.Error().Err(err).Str("asset", remaining.Asset.Name).Msg("failed to close asset")
					}
				}
				sc.multiprogress.Close()
				scanErr = ctx.Err()
				return
			default:
			}

			// Handle delayed discovery (e.g. container registry images).
			// This triggers the actual image download right before scanning,
			// ensuring only one container is on disk at a time.
			if len(asset.Connections) > 0 && asset.Connections[0].DelayDiscovery {
				updatedAsset, err := discovery.HandleDelayedDiscovery(ctx, asset, runtime)
				if err != nil {
					sc.reporter.AddScanError(asset, err)
					if err := sc.explorer.CloseAsset(tracked); err != nil {
						log.Error().Err(err).Str("asset", tracked.Asset.Name).Msg("failed to close asset")
					}
					continue
				}
				asset = updatedAsset
				tracked.Asset = asset

				// Now that the asset has real platform IDs, register it in the
				// progress bar and synchronize with upstream individually.
				if len(asset.PlatformIds) > 0 {
					sc.multiprogress.AddTask(asset.PlatformIds[0], asset)
				}
				if syncErr := syncBatchWithUpstream(ctx, []*discovery.TrackedAsset{tracked}, sc.services, sc.spaceMrn, sc.scanner.recording); syncErr != nil {
					sc.reporter.AddScanError(asset, syncErr)
					if len(asset.PlatformIds) > 0 {
						sc.multiprogress.Errored(asset.PlatformIds[0])
					}
					if err := sc.explorer.CloseAsset(tracked); err != nil {
						log.Error().Err(err).Str("asset", tracked.Asset.Name).Msg("failed to close asset")
					}
					continue
				}
			}

			if len(asset.PlatformIds) == 0 {
				log.Warn().Str("name", asset.Name).Msg("asset has no platform IDs after discovery, skipping")
				if err := sc.explorer.CloseAsset(tracked); err != nil {
					log.Error().Err(err).Str("asset", tracked.Asset.Name).Msg("failed to close asset")
				}
				continue
			}

			p := &progress.MultiProgressAdapter{Key: asset.PlatformIds[0], Multi: sc.multiprogress}
			sc.scanner.RunAssetJob(&AssetJob{
				DoRecord:         sc.job.DoRecord,
				UpstreamConfig:   sc.upstream,
				Asset:            asset,
				Bundle:           sc.job.Bundle,
				Props:            sc.job.Props,
				PolicyFilters:    preprocessPolicyFilters(sc.job.PolicyFilters),
				Ctx:              ctx,
				Reporter:         sc.reporter,
				ProgressReporter: p,
				runtime:          runtime,
			})

			// Close asset after scanning to free the gRPC connection
			if err := sc.explorer.CloseAsset(tracked); err != nil {
				log.Error().Err(err).Str("asset", tracked.Asset.Name).Msg("failed to close asset")
			}
		}
	}()
	wg.Wait()

	return scanErr
}

// syncBatchWithUpstream synchronizes a batch of connected assets with the
// upstream Mondoo Platform, or assigns local MRNs when running in incognito mode.
func syncBatchWithUpstream(
	ctx context.Context,
	batch []*discovery.TrackedAsset,
	services *policy.Services,
	spaceMrn string,
	rec llx.Recording,
) error {
	if services != nil {
		log.Info().Msg("synchronize assets")
		assetsToSync := make([]*inventory.Asset, 0, len(batch))
		for _, tracked := range batch {
			assetsToSync = append(assetsToSync, tracked.Asset)
		}
		log.Debug().Int("assets", len(assetsToSync)).Msg("synchronizing assets upstream")
		resp, err := services.SynchronizeAssets(ctx, &policy.SynchronizeAssetsReq{
			SpaceMrn: spaceMrn,
			List:     assetsToSync,
		})
		if err != nil {
			return err
		}
		log.Debug().Int("assets", len(resp.Details)).Msg("got assets details")
		platformAssetMapping := make(map[string]*policy.SynchronizeAssetsRespAssetDetail)
		for i := range resp.Details {
			log.Debug().Str("platform-mrn", resp.Details[i].PlatformMrn).Str("asset", resp.Details[i].AssetMrn).Msg("asset mapping")
			platformAssetMapping[resp.Details[i].PlatformMrn] = resp.Details[i]
		}

		for _, tracked := range batch {
			asset := tracked.Asset
			log.Debug().Str("asset", asset.Name).Strs("platform-ids", asset.PlatformIds).Msg("update asset")

			for _, platformMrn := range asset.PlatformIds {
				if details, ok := platformAssetMapping[platformMrn]; ok {
					asset.Mrn = details.AssetMrn
					asset.Url = details.Url
					asset.Labels["mondoo.com/project-id"] = details.ProjectId

					if asset.Annotations == nil {
						asset.Annotations = make(map[string]string)
					}
					for k, v := range details.Annotations {
						if _, ok := asset.Annotations[k]; !ok {
							asset.Annotations[k] = v
						}
					}

					err = tracked.Runtime.SetRecording(rec)
					if err != nil {
						log.Error().Err(err).Msg("could not set recording")
						break
					}
					tracked.Runtime.AssetUpdated(asset)
					break
				}
			}
		}
	} else {
		// Incognito mode: ensure we have non-empty asset MRNs
		for _, tracked := range batch {
			asset := tracked.Asset
			if asset.Mrn == "" {
				randID := "//" + policy.POLICY_SERVICE_NAME + "/" + policy.MRN_RESOURCE_ASSET + "/" + ksuid.New().String()
				x, err := mrn.NewMRN(randID)
				if err != nil {
					return multierr.Wrap(err, "failed to generate a random asset MRN")
				}
				asset.Mrn = x.String()
				tracked.Runtime.AssetUpdated(asset)
			}
		}
	}
	return nil
}

func (s *LocalScanner) upstreamServices(ctx context.Context, conf *upstream.UpstreamConfig) *policy.Services {
	if conf == nil ||
		conf.ApiEndpoint == "" ||
		conf.Incognito {
		return nil
	}

	client, err := s.upstreamClient(ctx, conf)
	if err != nil {
		log.Error().Err(err).Msg("could not init upstream client")
		return nil
	}

	res, err := policy.NewRemoteServices(client.ApiEndpoint, client.Plugins, client.HttpClient)
	if err != nil {
		log.Error().Err(err).Msg("could not connect to upstream")
	}

	return res
}

func (s *LocalScanner) RunAssetJob(job *AssetJob) {
	log.Debug().Msgf("connecting to asset %s", job.Asset.HumanName())

	results, err := s.runMotorizedAsset(job)
	if err != nil {
		log.Debug().Str("asset", job.Asset.Name).Msg("could not complete scan for asset")
		job.Reporter.AddScanError(job.Asset, err)
		job.ProgressReporter.Score(policy.ScoreRatingTextError)
		job.ProgressReporter.Errored()
		return
	}

	job.Reporter.AddReport(job.Asset, results)

	upstream := s.upstreamServices(job.Ctx, job.UpstreamConfig)
	// The vuln report is relevant only when we have an aggregate reporter
	if vulnReporter, isAggregateReporter := job.Reporter.(VulnReporter); upstream != nil && isAggregateReporter {
		// get new gql client
		mondooClient, err := gql.NewClient(job.UpstreamConfig, s._upstreamClient.HttpClient)
		if err != nil {
			return
		}

		gqlVulnReport, err := mondooClient.GetVulnCompactReport(job.Asset.Mrn)
		if err != nil {
			log.Error().Err(err).Msg("could not get vulnerability report")
			return
		}
		vulnReporter.AddVulnReport(job.Asset, gqlVulnReport)
	}

	// When the progress bar is disabled there's no feedback when an asset is done scanning. Adding this message
	// such that it is visible from the logs.
	if s.disableProgressBar {
		log.Info().Msgf("scan for asset %s completed", job.Asset.HumanName())
	}
}

func (s *LocalScanner) upstreamClient(ctx context.Context, conf *upstream.UpstreamConfig) (*upstream.UpstreamClient, error) {
	if s._upstreamClient != nil {
		return s._upstreamClient, nil
	}

	client, err := conf.InitClient(ctx)
	if err != nil {
		return nil, err
	}

	s._upstreamClient = client
	return client, nil
}

func (s *LocalScanner) runMotorizedAsset(job *AssetJob) (*AssetReport, error) {
	var res *AssetReport
	var policyErr error
	var client *upstream.UpstreamClient
	if job.UpstreamConfig.ApiEndpoint != "" && !job.UpstreamConfig.Incognito {
		var err error
		log.Debug().Msg("using API endpoint " + job.UpstreamConfig.ApiEndpoint)
		client, err = s.upstreamClient(job.Ctx, job.UpstreamConfig)
		if err != nil {
			return nil, err
		}
	}

	runtimeErr := WithServices(job.Ctx, s.runtime, job.Asset.Mrn, client, func(services *policy.LocalServices) error {
		scanner := &localAssetScanner{
			services:         services,
			job:              job,
			fetcher:          s.fetcher,
			Runtime:          job.runtime,
			ProgressReporter: job.ProgressReporter,
		}
		log.Debug().Str("asset", job.Asset.Name).Msg("run scan")
		res, policyErr = scanner.run()
		return policyErr
	})
	if runtimeErr != nil {
		return res, runtimeErr
	}

	return res, policyErr
}

func (s *LocalScanner) RunAdmissionReview(ctx context.Context, job *AdmissionReviewJob) (*ScanResult, error) {
	opts := job.Options
	if opts == nil {
		opts = make(map[string]string)
	}
	data, err := job.Data.MarshalJSON()
	if err != nil {
		return nil, err
	}
	opts["k8s-admission-review"] = base64.StdEncoding.EncodeToString(data)

	// construct the inventory to scan the admission review
	inv := &inventory.Inventory{
		Spec: &inventory.InventorySpec{
			Assets: []*inventory.Asset{{
				Connections: []*inventory.Config{{
					Type:     "k8s",
					Options:  opts,
					Discover: job.Discovery,
				}},
				Labels:   job.Labels,
				Category: inventory.AssetCategory_CATEGORY_CICD,
			}},
		},
	}

	runtimeEnv := execruntime.Detect()
	if runtimeEnv != nil {
		runtimeLabels := runtimeEnv.Labels()
		inv.ApplyLabels(runtimeLabels)
	}

	return s.Run(ctx, &Job{Inventory: inv, ReportType: job.ReportType})
}

func (s *LocalScanner) GarbageCollectAssets(ctx context.Context, garbageCollectOpts *GarbageCollectOptions) (*Empty, error) {
	if garbageCollectOpts == nil {
		return nil, status.Errorf(codes.InvalidArgument, "missing garbage collection options")
	}
	if s.upstream == nil {
		return nil, status.Errorf(codes.Internal, "missing upstream config in service")
	}

	client, err := s.upstreamClient(ctx, s.upstream)
	if err != nil {
		return nil, err
	}

	pClient, err := policy.NewRemoteServices(s.upstream.ApiEndpoint, client.Plugins, client.HttpClient)
	if err != nil {
		return nil, errors.Wrap(err, "could not initialize asset synchronization")
	}

	dar := &policy.PurgeAssetsRequest{
		SpaceMrn:        s.upstream.SpaceMrn,
		ManagedBy:       garbageCollectOpts.ManagedBy,
		PlatformRuntime: garbageCollectOpts.PlatformRuntime,
		Labels:          garbageCollectOpts.Labels,
	}

	if garbageCollectOpts.OlderThan != "" {
		timestamp, err := time.Parse(time.RFC3339, garbageCollectOpts.OlderThan)
		if err != nil {
			return nil, errors.Wrap(err, "failed converting timestamp from RFC3339 format")
		}

		dar.DateFilter = &policy.DateFilter{
			Timestamp: timestamp.Format(time.RFC3339),
			// LESS_THAN b/c we want assets with a lastUpdated timestamp older
			// (ie timewise considered less) than the timestamp provided
			Comparison: policy.Comparison_LESS_THAN,
			Field:      policy.DateFilterField_FILTER_LAST_UPDATED,
		}
	}

	_, err = pClient.PurgeAssets(ctx, dar)
	if err != nil {
		log.Error().Err(err).Msg("error while trying to garbage collect assets")
	}
	return &Empty{}, err
}

func (s *LocalScanner) HealthCheck(ctx context.Context, req *HealthCheckRequest) (*HealthCheckResponse, error) {
	// check the server overall health status.
	return &HealthCheckResponse{
		Status:     HealthCheckResponse_SERVING,
		Time:       time.Now().Format(time.RFC3339),
		ApiVersion: "v1",
		Build:      cnspec.GetBuild(),
		Version:    cnspec.GetVersion(),
	}, nil
}

func (s *LocalScanner) getUpstreamConfig(incognito bool, job *Job) (*upstream.UpstreamConfig, error) {
	var res *upstream.UpstreamConfig
	if s.upstream != nil {
		res = proto.Clone(s.upstream).(*upstream.UpstreamConfig)
	} else {
		res = &upstream.UpstreamConfig{}
	}
	res.Incognito = incognito

	jobCredentials := job.GetInventory().GetSpec().GetUpstreamCredentials()
	if s.allowJobCredentials && jobCredentials != nil {
		res.Creds = jobCredentials
		res.ApiEndpoint = jobCredentials.GetApiEndpoint()
		res.SpaceMrn = jobCredentials.GetParentMrn()
	}

	if !res.Incognito {
		if res.ApiEndpoint == "" {
			return nil, errors.New("missing API endpoint")
		}
		if res.SpaceMrn == "" {
			return nil, errors.New("missing space mrn")
		}
	}

	return res, nil
}

type localAssetScanner struct {
	services *policy.LocalServices
	job      *AssetJob
	fetcher  *fetcher

	Runtime          llx.Runtime
	ProgressReporter progress.Progress
}

// run() runs a bundle on a single asset. It returns the results of the scan and an error if the scan failed. Even in
// case of an error, the results may contain partial results. The error is only returned if the scan failed to run not
// when individual policies failed.
func (s *localAssetScanner) run() (*AssetReport, error) {
	if err := s.prepareAsset(); err != nil {
		return nil, err
	}

	resolvedPolicy, err := s.runPolicy()
	if err != nil {
		return nil, err
	}

	if mql.GetFeatures(s.job.Ctx).IsActive(mql.StoreResourcesData) && resolvedPolicy.HasFeature(policy.ServerFeature_STORE_RESOURCES_DATA) {
		log.Info().Str("mrn", s.job.Asset.Mrn).Msg("store resources for asset")
		recording := s.Runtime.Recording()
		data, ok := recording.GetAssetData(s.job.Asset.Mrn)
		if !ok {
			log.Debug().Msg("not storing resource data for this asset, nothing available")
		} else {
			_, err = s.services.StoreResults(context.Background(), &policy.StoreResultsReq{
				AssetMrn:  s.job.Asset.Mrn,
				Resources: data,
			})
			if err != nil {
				return nil, err
			}
		}
	}

	ar := &AssetReport{
		Mrn:            s.job.Asset.Mrn,
		ResolvedPolicy: resolvedPolicy,
	}

	report, err := s.getReport(resolvedPolicy)
	if err != nil {
		return ar, err
	}
	s.ProgressReporter.Score(report.Score.Rating().Text())
	if report.Score.Rating().Text() == policy.ScoreRatingTextUnrated {
		s.ProgressReporter.NotApplicable()
	} else {
		s.ProgressReporter.Completed()
	}

	log.Debug().Str("asset", s.job.Asset.Mrn).Msg("scan complete")
	ar.Report = report
	return ar, nil
}

func filterPolicyMrns(b *policy.Bundle, filters []string) []string {
	if len(filters) == 0 {
		res := make([]string, len(b.Policies))
		for i := range b.Policies {
			res[i] = b.Policies[i].Mrn
		}
		return res
	}

	var res []string
	for i := range b.Policies {
		cur := b.Policies[i]
		uid, _ := mrn.GetResource(cur.Mrn, policy.MRN_RESOURCE_POLICY)
		if slices.Contains(filters, uid) || slices.Contains(filters, cur.Mrn) {
			res = append(res, cur.Mrn)
		}
	}
	return res
}

func (s *localAssetScanner) prepareAsset() error {
	var hub policy.PolicyHub = s.services

	// if we are using upstream we get the bundle from there, no need to check for a bundle here
	if !s.job.UpstreamConfig.Incognito {
		return nil
	}

	// if we have a bundle we don't need to check for policies
	// e.g. we passed in a bundle directly via a file
	if s.job.Bundle == nil {
		// fetch bundles for public registry
		if err := s.fetchPublicRegistryBundle(); err != nil {
			return err
		}

		// add asset bundle to the reporter
		if s.job.Reporter != nil && s.job.Bundle != nil {
			s.job.Reporter.AddBundle(s.job.Bundle)
		}
	}

	if s.job.Bundle == nil {
		return errors.New("no bundle provided to run")
	}

	// set the bundle in local store
	_, err := hub.SetBundle(s.job.Ctx, s.job.Bundle)
	if err != nil {
		return err
	}

	policyMrns := filterPolicyMrns(s.job.Bundle, s.job.PolicyFilters)

	frameworkMrns := make([]string, len(s.job.Bundle.Frameworks))
	for i := range s.job.Bundle.Frameworks {
		frameworkMrns[i] = s.job.Bundle.Frameworks[i].Mrn
	}

	var resolver policy.PolicyResolver = s.services
	_, err = resolver.Assign(s.job.Ctx, &policy.PolicyAssignment{
		AssetMrn:      s.job.Asset.Mrn,
		PolicyMrns:    policyMrns,
		FrameworkMrns: frameworkMrns,
		Action:        policy.Action_ACTIVATE,
	})
	if err != nil {
		return err
	}

	if len(s.job.Props) != 0 {
		propsReq, err := s.mapPropOverrides()
		if err != nil {
			return fmt.Errorf("failed to map property overrides: %w", err)
		}

		_, err = resolver.SetProps(s.job.Ctx, propsReq)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *localAssetScanner) mapPropOverrides() (*policy.PropsReq, error) {
	exposedProps := make(map[string][]string, len(s.job.Props))
	for _, pol := range s.job.Bundle.Policies {
		for _, prop := range pol.Props {
			propUid, err := policy.GetPropName(prop.Mrn)
			if err != nil {
				return nil, fmt.Errorf("failed to get property name for %s: %w", prop.Mrn, err)
			}
			exposedProps[propUid] = append(exposedProps[propUid], prop.Mrn)
		}
	}

	propsReq := policy.PropsReq{
		EntityMrn: s.job.Asset.Mrn,
		Props:     make([]*policy.Property, 0, len(s.job.Props)),
	}
	for k, v := range s.job.Props {
		newProp := &policy.Property{
			Uid: k,
			Mql: v,
		}
		forProps := exposedProps[k]
		if len(forProps) == 0 {
			continue
		}
		for _, propMrn := range forProps {
			newProp.For = append(newProp.For, &policy.ObjectRef{
				Mrn: propMrn,
			})
		}
		if err := newProp.RefreshMRN(s.job.Asset.Mrn); err != nil {
			return nil, fmt.Errorf("failed to refresh MRN for property %s: %w", newProp.Mrn, err)
		}
		propsReq.Props = append(propsReq.Props, newProp)
	}

	return &propsReq, nil
}

var (
	assetDetectBundle     *llx.CodeBundle
	assetDetectBundleOnce sync.Once
)

func getAssetDetectBundle() *llx.CodeBundle {
	assetDetectBundleOnce.Do(func() {
		assetDetectBundle = mustCompile("asset { kind platform runtime version family }")
	})
	return assetDetectBundle
}

func (s *localAssetScanner) fetchPublicRegistryBundle() error {
	features := mql.GetFeatures(s.job.Ctx)
	_, res, err := executor.ExecuteQuery(s.Runtime, getAssetDetectBundle(), nil, features)
	if err != nil {
		return errors.Wrap(err, "failed to run asset detection query")
	}

	// FIXME: remove hardcoded lookup and use embedded datastructures instead
	data := res["IA0bVPKFxIh8Z735sqDh7bo/FNIYUQ/B4wLijN+YhiBZePu1x2sZCMcHoETmWM9jocdWbwGykKvNom/7QSm8ew=="].Data.Value.(map[string]any)
	kind := data["1oxYPIhW1eZ+14s234VsQ0Q7p9JSmUaT/RTWBtDRG1ZwKr8YjMcXz76x10J9iu13AcMmGZd43M1NNqPXZtTuKQ=="].(*llx.RawData).Value.(string)
	platform := data["W+8HW/v60Fx0nqrVz+yTIQjImy4ki4AiqxcedooTPP3jkbCESy77ptEhq9PlrKjgLafHFn8w4vrimU4bwCi6aQ=="].(*llx.RawData).Value.(string)
	runtime := data["a3RMPjrhk+jqkeXIISqDSi7EEP8QybcXCeefqNJYVUNcaDGcVDdONFvcTM2Wts8qTRXL3akVxpskitXWuI/gdA=="].(*llx.RawData).Value.(string)
	version := data["5d4FZxbPkZu02MQaHp3C356NJ9TeVsJBw8Enu+TDyBGdWlZM/AE+J5UT/TQ72AmDViKZe97Hxz1Jt3MjcEH/9Q=="].(*llx.RawData).Value.(string)
	fraw := data["l/aGjrixdNHvCxu5ib4NwkYb0Qrh3sKzcrGTkm7VxNWfWaaVbOxOEoGEMnjGJTo31jhYNeRm39/zpepZaSbUIw=="].(*llx.RawData).Value.([]any)
	family := make([]string, len(fraw))
	for i := range fraw {
		family[i] = fraw[i].(string)
	}

	var hub policy.PolicyHub = s.services
	urls, err := hub.DefaultPolicies(s.job.Ctx, &policy.DefaultPoliciesReq{
		Kind:     kind,
		Platform: platform,
		Runtime:  runtime,
		Version:  version,
		Family:   family,
	})
	if err != nil {
		return err
	}

	if len(urls.Urls) == 0 {
		return errors.New("cannot find any default policies for this asset (" + platform + ")")
	}

	conf := s.services.NewCompilerConfig()
	s.job.Bundle, err = s.fetcher.fetchBundles(s.job.Ctx, conf, urls.Urls...)
	return err
}

func (s *localAssetScanner) runPolicy() (*policy.ResolvedPolicy, error) {
	var hub policy.PolicyHub = s.services
	var resolver policy.PolicyResolver = s.services

	// If we run in debug mode, download the asset bundle and dump it to disk
	if val, ok := os.LookupEnv("DEBUG"); ok && (val == "1" || val == "true") {
		log.Debug().Str("asset", s.job.Asset.Mrn).Msg("client> request policies bundle for asset")
		assetBundle, err := hub.GetBundle(s.job.Ctx, &policy.Mrn{Mrn: s.job.Asset.Mrn})
		if err != nil {
			return nil, err
		}
		log.Debug().Msg("client> got policy bundle")
		logger.TraceJSON(assetBundle)
		logger.DebugDumpYAML("assetBundle", assetBundle)
	}

	rawFilters, err := hub.GetPolicyFilters(s.job.Ctx, &policy.Mrn{Mrn: s.job.Asset.Mrn})
	if err != nil {
		return nil, err
	}
	log.Debug().Str("asset", s.job.Asset.Mrn).Msg("client> got policy filters")
	logger.TraceJSON(rawFilters)
	logger.DebugDumpYAML("policyFilters", rawFilters)

	filters, err := s.UpdateFilters(&policy.Mqueries{Items: rawFilters.Items}, 5*time.Second)
	if err != nil {
		return nil, err
	}
	log.Debug().Str("asset", s.job.Asset.Mrn).Msg("client> shell update filters")
	logger.DebugJSON(filters)
	logger.DebugDumpYAML("assetFilters", filters)

	resolvedPolicy, err := resolver.ResolveAndUpdateJobs(s.job.Ctx, &policy.UpdateAssetJobsReq{
		AssetMrn:     s.job.Asset.Mrn,
		AssetFilters: filters,
	})
	if err != nil {
		return resolvedPolicy, err
	}
	log.Debug().Str("asset", s.job.Asset.Mrn).Msg("client> got resolved policy bundle for asset")
	logger.DebugDumpJSON("resolvedPolicy", resolvedPolicy)

	features := mql.GetFeatures(s.job.Ctx)
	err = executor.ExecuteResolvedPolicy(s.job.Ctx, s.Runtime, resolver, s.job.Asset.Mrn, resolvedPolicy, features, s.ProgressReporter)
	if err != nil {
		return nil, err
	}

	return resolvedPolicy, nil
}

func (s *localAssetScanner) getReport(resolvedPolicy *policy.ResolvedPolicy) (*policy.Report, error) {
	var resolver policy.PolicyResolver = s.services

	// TODO: we do not needs this anymore since we receive updates already
	log.Debug().Str("asset", s.job.Asset.Mrn).Msg("client> send all results")
	_, err := policy.WaitUntilDone(resolver, s.job.Asset.Mrn, s.job.Asset.Mrn, 1*time.Second)
	// handle error
	if err != nil {
		return &policy.Report{
			EntityMrn:  s.job.Asset.Mrn,
			ScoringMrn: s.job.Asset.Mrn,
		}, err
	}

	log.Debug().Str("asset", s.job.Asset.Mrn).Msg("generate report")
	report, err := resolver.GetReport(s.job.Ctx, &policy.EntityScoreReq{
		// NOTE: we assign policies to the asset before we execute the tests, therefore this resolves all policies assigned to the asset
		EntityMrn: s.job.Asset.Mrn,
		ScoreMrn:  s.job.Asset.Mrn,
	})
	if err != nil {
		return &policy.Report{
			EntityMrn:  s.job.Asset.Mrn,
			ScoringMrn: s.job.Asset.Mrn,
		}, err
	}

	return report, nil
}

// FilterQueries returns all queries whose result is truthy
func (s *localAssetScanner) FilterQueries(queries []*policy.Mquery, timeout time.Duration) ([]*policy.Mquery, []error) {
	return executor.ExecuteFilterQueries(s.Runtime, queries, timeout)
}

// UpdateFilters takes a list of test filters and runs them against the backend
// to return the matching ones
func (s *localAssetScanner) UpdateFilters(filters *policy.Mqueries, timeout time.Duration) ([]*policy.Mquery, error) {
	queries, errs := s.FilterQueries(filters.Items, timeout)

	var err error
	if len(errs) != 0 {
		w := strings.Builder{}
		for i := range errs {
			w.WriteString(errs[i].Error() + "\n")
		}
		err = errors.New("received multiple errors: " + w.String())
	}

	return queries, err
}

func sendErrorToMondooPlatform(serviceAccount *upstream.ServiceAccountCredentials, event *health.SendErrorReq) {
	// 3. send error to mondoo platform
	proxy, err := config.GetAPIProxy()
	if err != nil {
		log.Error().Err(err).Msg("failed to parse proxy setting")
		return
	}
	httpClient := ranger.NewHttpClient(ranger.WithProxy(proxy))

	plugins := []ranger.ClientPlugin{}
	certAuth, err := upstream.NewServiceAccountRangerPlugin(serviceAccount)
	if err != nil {
		return
	}
	plugins = append(plugins, certAuth)

	cl, err := health.NewErrorReportingClient(serviceAccount.ApiEndpoint, httpClient, plugins...)
	if err != nil {
		log.Error().Err(err).Msg("failed to create error reporting client")
		return
	}

	_, err = cl.SendError(context.Background(), event)
	if err != nil {
		log.Error().Err(err).Msg("failed to send error to Mondoo Platform")
		return
	}
}

func WithServices(ctx context.Context, runtime llx.Runtime, assetMrn string, upstreamClient *upstream.UpstreamClient, f func(*policy.LocalServices) error) error {
	var withServicesFunc func(context.Context, llx.Runtime, string, *upstream.UpstreamClient, func(*policy.LocalServices) error) error
	if mql.IsFeatureActive(ctx, mql.UploadResultsV2) {
		withServicesFunc = sqlite.WithServices
	} else {
		withServicesFunc = inmemory.WithServices
	}
	return withServicesFunc(ctx, runtime, assetMrn, upstreamClient, f)
}

func mustCompile(code string) *llx.CodeBundle {
	codeBundle, err := mqlc.Compile(code, nil,
		mqlc.NewConfig(providers.Coordinator.Schema(), mql.DefaultFeatures))
	if err != nil {
		panic(err)
	}
	return codeBundle
}

func createProgressBar(disableProgressBar bool) (progress.MultiProgress, error) {
	if isatty.IsTerminal(os.Stdout.Fd()) && !disableProgressBar && !strings.EqualFold(logger.GetLevel(), "debug") && !strings.EqualFold(logger.GetLevel(), "trace") {
		return progress.NewTodoList(progress.WithScore())
	}
	return progress.NoopMultiProgress{}, nil
}
