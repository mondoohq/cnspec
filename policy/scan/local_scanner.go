// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"context"
	"encoding/base64"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/ksuid"
	"go.mondoo.com/cnquery/v10"
	"go.mondoo.com/cnquery/v10/cli/execruntime"
	"go.mondoo.com/cnquery/v10/cli/progress"
	"go.mondoo.com/cnquery/v10/explorer"
	ee "go.mondoo.com/cnquery/v10/explorer/executor"
	"go.mondoo.com/cnquery/v10/explorer/scan"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/logger"
	"go.mondoo.com/cnquery/v10/mrn"
	"go.mondoo.com/cnquery/v10/providers"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/upstream"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/upstream/gql"
	"go.mondoo.com/cnquery/v10/utils/multierr"
	"go.mondoo.com/cnquery/v10/utils/slicesx"
	"go.mondoo.com/cnspec/v10"
	"go.mondoo.com/cnspec/v10/internal/datalakes/inmemory"
	"go.mondoo.com/cnspec/v10/policy"
	"go.mondoo.com/cnspec/v10/policy/executor"
	"go.mondoo.com/ranger-rpc/codes"
	"go.mondoo.com/ranger-rpc/status"
	"google.golang.org/protobuf/proto"
)

type LocalScanner struct {
	resolvedPolicyCache *inmemory.ResolvedPolicyCache
	queue               *diskQueueClient
	ctx                 context.Context
	fetcher             *fetcher
	upstream            *upstream.UpstreamConfig
	_upstreamClient     *upstream.UpstreamClient
	recording           llx.Recording
	runtime             llx.Runtime

	// allows setting the upstream credentials from a job
	allowJobCredentials bool
	disableProgressBar  bool
	reportType          ReportType
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

func NewLocalScanner(opts ...ScannerOption) *LocalScanner {
	runtime := providers.Coordinator.NewRuntime()

	ls := &LocalScanner{
		resolvedPolicyCache: inmemory.NewResolvedPolicyCache(ResolvedPolicyCacheSize),
		runtime:             runtime,
		fetcher:             newFetcher(),
		ctx:                 context.Background(),
	}

	for i := range opts {
		opts[i](ls)
	}

	if ls.recording == nil {
		ls.recording = providers.NullRecording{}
	}

	return ls
}

func (s *LocalScanner) EnableQueue() error {
	var err error
	s.queue, err = newDqueClient(defaultDqueConfig, func(job *Job) {
		// this is the handler for jobs, when they are picked up
		ctx := cnquery.SetFeatures(s.ctx, cnquery.DefaultFeatures)
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
		// - use Mondoo platform upstream with/without incognito
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
			bundle, err := services.GetBundle(ctx, &policy.Mrn{Mrn: upstream.Creds.ParentMrn})
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

	log.Info().Msgf("discover related assets for %d asset(s)", len(job.Inventory.Spec.Assets))
	discoveredAssets, err := scan.DiscoverAssets(ctx, job.Inventory, upstream, s.recording)
	if err != nil {
		return nil, err
	}

	// if we had asset errors we want to place them into the reporter
	for i := range discoveredAssets.Errors {
		reporter.AddScanError(discoveredAssets.Errors[i].Asset, discoveredAssets.Errors[i].Err)
	}

	// For each discovered asset, we initialize a new runtime and connect to it.
	// Within this process, we set up a catch-all deferred function, that shuts
	// down all runtimes, in case we exit early.
	defer func() {
		for _, asset := range discoveredAssets.Assets {
			// we can call close multiple times and it will only execute once
			if asset.Runtime != nil {
				asset.Runtime.Close()
			}
		}
	}()

	if len(discoveredAssets.Assets) == 0 {
		return reporter.Reports(), nil
	}

	multiprogress, err := scan.CreateProgressBar(discoveredAssets, s.disableProgressBar)
	if err != nil {
		return nil, err
	}

	// start the progress bar
	scanGroups := sync.WaitGroup{}
	scanGroups.Add(1)
	go func() {
		defer scanGroups.Done()
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

	assetBatches := slicesx.Batch(discoveredAssets.Assets, 100)
	for i := range assetBatches {
		batch := assetBatches[i]

		// sync assets
		if services != nil {
			log.Info().Msg("synchronize assets")
			assetsToSync := make([]*inventory.Asset, 0, len(batch))
			for i := range batch {
				// If discovery has been skipped, then we don't sync that asset just yet. We will do that during the scan
				if batch[i].Asset.Connections[0].DelayDiscovery {
					continue
				}
				assetsToSync = append(assetsToSync, batch[i].Asset)
			}

			resp, err := services.SynchronizeAssets(ctx, &policy.SynchronizeAssetsReq{
				SpaceMrn: spaceMrn,
				List:     assetsToSync,
			})
			if err != nil {
				return nil, err
			}
			log.Debug().Int("assets", len(resp.Details)).Msg("got assets details")
			platformAssetMapping := make(map[string]*policy.SynchronizeAssetsRespAssetDetail)
			for i := range resp.Details {
				log.Debug().Str("platform-mrn", resp.Details[i].PlatformMrn).Str("asset", resp.Details[i].AssetMrn).Msg("asset mapping")
				platformAssetMapping[resp.Details[i].PlatformMrn] = resp.Details[i]
			}

			// attach the asset details to the assets list
			for i := range batch {
				asset := batch[i].Asset
				log.Debug().Str("asset", asset.Name).Strs("platform-ids", asset.PlatformIds).Msg("update asset")
				platformMrn := asset.PlatformIds[0]
				if details, ok := platformAssetMapping[platformMrn]; ok {
					asset.Mrn = details.AssetMrn
					asset.Url = details.Url
					asset.Labels["mondoo.com/project-id"] = details.ProjectId
				}
			}
		} else {
			// ensure we have non-empty asset MRNs
			for i := range batch {
				asset := batch[i].Asset
				if asset.Mrn == "" {
					randID := "//" + policy.POLICY_SERVICE_NAME + "/" + policy.MRN_RESOURCE_ASSET + "/" + ksuid.New().String()
					x, err := mrn.NewMRN(randID)
					if err != nil {
						return nil, multierr.Wrap(err, "failed to generate a random asset MRN")
					}
					asset.Mrn = x.String()
				}
			}
		}

		// // if a bundle was provided check that it matches the filter, bundles can also be downloaded
		// // later therefore we do not want to stop execution here
		// if job.Bundle != nil && job.Bundle.FilterPolicies(job.PolicyFilters) {
		// 	return nil, false, errors.New("all available packs filtered out. nothing to do.")
		// }

		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range batch {
				asset := batch[i].Asset
				runtime := batch[i].Runtime

				log.Debug().Interface("asset", asset).Msg("start scan")

				// Make sure the context has not been canceled in the meantime. Note that this approach works only for single threaded execution. If we have more than 1 thread calling this function,
				// we need to solve this at a different level.
				select {
				case <-ctx.Done():
					log.Warn().Msg("request context has been canceled")
					// When we scan concurrently, we need to call Errored(asset.Mrn) status for this asset
					multiprogress.Close()
					return
				default:
				}

				if asset.Connections[0].DelayDiscovery {
					discoveredAsset, err := handleDelayedDiscovery(ctx, asset, runtime, services, spaceMrn)
					if err != nil {
						reporter.AddScanError(asset, err)
						multiprogress.Errored(asset.PlatformIds[0])
						continue
					}
					asset = discoveredAsset
				}

				p := &progress.MultiProgressAdapter{Key: asset.PlatformIds[0], Multi: multiprogress}
				s.RunAssetJob(&AssetJob{
					DoRecord:         job.DoRecord,
					UpstreamConfig:   upstream,
					Asset:            asset,
					Bundle:           job.Bundle,
					Props:            job.Props,
					PolicyFilters:    preprocessPolicyFilters(job.PolicyFilters),
					Ctx:              ctx,
					Reporter:         reporter,
					ProgressReporter: p,
					runtime:          runtime,
				})

				// shut down all ephemeral runtimes
				runtime.Close()
			}
		}()
		wg.Wait()
	}
	scanGroups.Wait() // wait for all scans to complete
	return reporter.Reports(), nil
}

func handleDelayedDiscovery(ctx context.Context, asset *inventory.Asset, runtime *providers.Runtime, services *policy.Services, spaceMrn string) (*inventory.Asset, error) {
	asset.Connections[0].DelayDiscovery = false
	if err := runtime.Connect(&plugin.ConnectReq{Asset: asset}); err != nil {
		return nil, err
	}
	if services != nil {
		resp, err := services.SynchronizeAssets(ctx, &policy.SynchronizeAssetsReq{
			SpaceMrn: spaceMrn,
			List:     []*inventory.Asset{asset},
		})
		if err != nil {
			return nil, err
		}

		asset = runtime.Provider.Connection.Asset
		slices.Sort(asset.PlatformIds)
		details := resp.Details[asset.PlatformIds[0]]
		asset.Mrn = details.AssetMrn
		asset.Url = details.Url
		asset.KindString = asset.GetPlatform().Kind
		asset.Labels["mondoo.com/project-id"] = details.ProjectId
	}
	return asset, nil
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
		job.ProgressReporter.Score("X")
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

	runtimeErr := inmemory.WithDb(s.runtime, s.resolvedPolicyCache, func(db *inmemory.Db, services *policy.LocalServices) error {
		if job.UpstreamConfig.ApiEndpoint != "" && !job.UpstreamConfig.Incognito {
			log.Debug().Msg("using API endpoint " + job.UpstreamConfig.ApiEndpoint)
			client, err := s.upstreamClient(job.Ctx, job.UpstreamConfig)
			if err != nil {
				return err
			}

			upstream, err := policy.NewRemoteServices(client.ApiEndpoint, client.Plugins, client.HttpClient)
			if err != nil {
				return err
			}
			services.Upstream = upstream
		}

		scanner := &localAssetScanner{
			db:               db,
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
	db       *inmemory.Db
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

	ar := &AssetReport{
		Mrn:            s.job.Asset.Mrn,
		ResolvedPolicy: resolvedPolicy,
	}

	report, err := s.getReport()
	if err != nil {
		return ar, err
	}
	s.ProgressReporter.Score(report.Score.Rating().Letter())
	if report.Score.Rating().Letter() == "U" {
		s.ProgressReporter.NotApplicable()
	} else {
		s.ProgressReporter.Completed()
	}

	log.Debug().Str("asset", s.job.Asset.Mrn).Msg("scan complete")
	ar.Report = report
	return ar, nil
}

func noPolicyErr(availablePolicies []string, filter []string) error {
	var sb strings.Builder
	sb.WriteString("bundle doesn't contain any policies\n")
	sb.WriteString("\n")

	if len(availablePolicies) > 0 {
		sb.WriteString("The following policies are available:\n")
		for i := range availablePolicies {
			policyMrn := availablePolicies[i]
			sb.WriteString("- " + policyMrn + "\n")
		}
		sb.WriteString("\n")
	} else {
		sb.WriteString("The policy bundle for the asset does not contain any policies\n\n")
	}

	if len(filter) > 0 {
		sb.WriteString("User selected policies that are allowed to run:\n")
		for i := range filter {
			policyMrn := filter[i]
			sb.WriteString("- " + policyMrn + "\n")
		}
		sb.WriteString("\n")
	}

	return errors.New(sb.String())
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
	})
	if err != nil {
		return err
	}

	if len(s.job.Props) != 0 {
		propsReq := explorer.PropsReq{
			EntityMrn: s.job.Asset.Mrn,
			Props:     make([]*explorer.Property, len(s.job.Props)),
		}
		i := 0
		for k, v := range s.job.Props {
			propsReq.Props[i] = &explorer.Property{
				Uid: k,
				Mql: v,
			}
			i++
		}

		_, err = resolver.SetProps(s.job.Ctx, &propsReq)
		if err != nil {
			return err
		}
	}
	return nil
}

var assetDetectBundle = ee.MustCompile("asset { kind platform runtime version family }")

func (s *localAssetScanner) fetchPublicRegistryBundle() error {
	features := cnquery.GetFeatures(s.job.Ctx)
	_, res, err := executor.ExecuteQuery(s.Runtime, assetDetectBundle, nil, features)
	if err != nil {
		return errors.Wrap(err, "failed to run asset detection query")
	}

	// FIXME: remove hardcoded lookup and use embedded datastructures instead
	data := res["IA0bVPKFxIh8Z735sqDh7bo/FNIYUQ/B4wLijN+YhiBZePu1x2sZCMcHoETmWM9jocdWbwGykKvNom/7QSm8ew=="].Data.Value.(map[string]interface{})
	kind := data["1oxYPIhW1eZ+14s234VsQ0Q7p9JSmUaT/RTWBtDRG1ZwKr8YjMcXz76x10J9iu13AcMmGZd43M1NNqPXZtTuKQ=="].(*llx.RawData).Value.(string)
	platform := data["W+8HW/v60Fx0nqrVz+yTIQjImy4ki4AiqxcedooTPP3jkbCESy77ptEhq9PlrKjgLafHFn8w4vrimU4bwCi6aQ=="].(*llx.RawData).Value.(string)
	runtime := data["a3RMPjrhk+jqkeXIISqDSi7EEP8QybcXCeefqNJYVUNcaDGcVDdONFvcTM2Wts8qTRXL3akVxpskitXWuI/gdA=="].(*llx.RawData).Value.(string)
	version := data["5d4FZxbPkZu02MQaHp3C356NJ9TeVsJBw8Enu+TDyBGdWlZM/AE+J5UT/TQ72AmDViKZe97Hxz1Jt3MjcEH/9Q=="].(*llx.RawData).Value.(string)
	fraw := data["l/aGjrixdNHvCxu5ib4NwkYb0Qrh3sKzcrGTkm7VxNWfWaaVbOxOEoGEMnjGJTo31jhYNeRm39/zpepZaSbUIw=="].(*llx.RawData).Value.([]interface{})
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

	filters, err := s.UpdateFilters(&explorer.Mqueries{Items: rawFilters.Items}, 5*time.Second)
	if err != nil {
		return nil, err
	}
	log.Debug().Str("asset", s.job.Asset.Mrn).Msg("client> shell update filters")
	logger.DebugJSON(filters)

	resolvedPolicy, err := resolver.ResolveAndUpdateJobs(s.job.Ctx, &policy.UpdateAssetJobsReq{
		AssetMrn:     s.job.Asset.Mrn,
		AssetFilters: filters,
	})
	if err != nil {
		return resolvedPolicy, err
	}
	log.Debug().Str("asset", s.job.Asset.Mrn).Msg("client> got resolved policy bundle for asset")
	logger.DebugDumpJSON("resolvedPolicy", resolvedPolicy)

	features := cnquery.GetFeatures(s.job.Ctx)
	err = executor.ExecuteResolvedPolicy(s.Runtime, resolver, s.job.Asset.Mrn, resolvedPolicy, features, s.ProgressReporter)
	if err != nil {
		return nil, err
	}

	return resolvedPolicy, nil
}

func (s *localAssetScanner) getReport() (*policy.Report, error) {
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
func (s *localAssetScanner) FilterQueries(queries []*explorer.Mquery, timeout time.Duration) ([]*explorer.Mquery, []error) {
	return executor.ExecuteFilterQueries(s.Runtime, queries, timeout)
}

// UpdateFilters takes a list of test filters and runs them against the backend
// to return the matching ones
func (s *localAssetScanner) UpdateFilters(filters *explorer.Mqueries, timeout time.Duration) ([]*explorer.Mquery, error) {
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
