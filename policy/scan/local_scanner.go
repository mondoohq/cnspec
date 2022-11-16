package scan

import (
	"context"
	"encoding/base64"
	"os"
	"strings"
	"time"

	"github.com/mattn/go-isatty"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/ksuid"
	"go.mondoo.com/cnquery"
	"go.mondoo.com/cnquery/cli/execruntime"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/logger"
	"go.mondoo.com/cnquery/motor"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery"
	"go.mondoo.com/cnquery/motor/inventory"
	v1 "go.mondoo.com/cnquery/motor/inventory/v1"
	providers "go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/providers/resolver"
	"go.mondoo.com/cnquery/mrn"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/all"
	"go.mondoo.com/cnspec/cli/progress"
	"go.mondoo.com/cnspec/internal/datalakes/inmemory"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/cnspec/policy/executor"
	"go.mondoo.com/ranger-rpc"
	"go.mondoo.com/ranger-rpc/codes"
	"go.mondoo.com/ranger-rpc/status"
)

type LocalScanner struct {
	resolvedPolicyCache *inmemory.ResolvedPolicyCache
	queue               *diskQueueClient
	ctx                 context.Context
	fetcher             *fetcher

	// for remote connectivity
	apiEndpoint        string
	spaceMrn           string
	plugins            []ranger.ClientPlugin
	disableProgressBar bool
}

type ScannerOption func(*LocalScanner)

func WithUpstream(apiEndpoint string, spaceMrn string, plugins []ranger.ClientPlugin) ScannerOption {
	return func(s *LocalScanner) {
		s.apiEndpoint = apiEndpoint
		s.plugins = plugins
		s.spaceMrn = spaceMrn
	}
}

func DisableProgressBar() ScannerOption {
	return func(s *LocalScanner) {
		s.disableProgressBar = true
	}
}

func NewLocalScanner(opts ...ScannerOption) *LocalScanner {
	ls := &LocalScanner{
		resolvedPolicyCache: inmemory.NewResolvedPolicyCache(ResolvedPolicyCacheSize),
		fetcher:             newFetcher(),
		ctx:                 context.Background(),
	}

	for i := range opts {
		opts[i](ls)
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

	dctx := discovery.InitCtx(ctx)

	upstreamConfig := resources.UpstreamConfig{
		SpaceMrn:    s.spaceMrn,
		ApiEndpoint: s.apiEndpoint,
		Incognito:   false,
		Plugins:     s.plugins,
	}

	reports, _, err := s.distributeJob(job, dctx, upstreamConfig)
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

	dctx := discovery.InitCtx(ctx)

	upstreamConfig := resources.UpstreamConfig{
		Incognito: true,
	}

	reports, _, err := s.distributeJob(job, dctx, upstreamConfig)
	if err != nil {
		return nil, err
	}

	return reports, nil
}

func (s *LocalScanner) distributeJob(job *Job, ctx context.Context, upstreamConfig resources.UpstreamConfig) (*ScanResult, bool, error) {
	log.Info().Msgf("discover related assets for %d asset(s)", len(job.Inventory.Spec.Assets))
	im, err := inventory.New(inventory.WithInventory(job.Inventory))
	if err != nil {
		return nil, false, errors.Wrap(err, "could not load asset information")
	}

	assetErrors := im.Resolve(ctx)
	if len(assetErrors) > 0 {
		for a := range assetErrors {
			log.Error().Err(assetErrors[a]).Str("asset", a.Name).Msg("could not resolve asset")
		}
		return nil, false, errors.New("failed to resolve multiple assets")
	}

	assetList := im.GetAssets()
	if len(assetList) == 0 {
		return nil, false, errors.New("could not find an asset that we can connect to")
	}

	if upstreamConfig.ApiEndpoint == "" || upstreamConfig.Incognito {
		// ensure we have non-empty asset MRNs
		for i := range assetList {
			cur := assetList[i]
			if cur.Mrn == "" && cur.Id == "" {
				randID := "//" + policy.POLICY_SERVICE_NAME + "/" + policy.MRN_RESOURCE_ASSET + "/" + ksuid.New().String()
				x, err := mrn.NewMRN(randID)
				if err != nil {
					return nil, false, errors.Wrap(err, "failed to generate a random asset MRN")
				}
				cur.Mrn = x.String()
			}
		}
	}

	// plan scan jobs
	var reporter Reporter
	switch job.ReportType {
	case ReportType_FULL:
		reporter = NewAggregateReporter(assetList)
	case ReportType_ERROR:
		reporter = NewErrorReporter(assetList)
	case ReportType_NONE:
		reporter = NewNoOpReporter(assetList)
	default:
		return nil, false, errors.Errorf("unknown report type: %s", job.ReportType)
	}
	job.Bundle.FilterPolicies(job.PolicyFilters)

	for i := range assetList {
		asset := assetList[i]

		// Make sure the context has not been canceled in the meantime. Note that this approach works only for single threaded execution. If we have more than 1 thread calling this function,
		// we need to solve this at a different level.
		select {
		case <-ctx.Done():
			log.Warn().Msg("request context has been canceled")
			return reporter.Reports(), false, nil
		default:
		}

		s.RunAssetJob(&AssetJob{
			DoRecord:       job.DoRecord,
			UpstreamConfig: upstreamConfig,
			Asset:          asset,
			Bundle:         job.Bundle,
			PolicyFilters:  job.PolicyFilters,
			Ctx:            ctx,
			GetCredential:  im.GetCredential,
			Reporter:       reporter,
		})
	}

	return reporter.Reports(), true, nil
}

func (s *LocalScanner) RunAssetJob(job *AssetJob) {
	log.Info().Msgf("connecting to asset %s", job.Asset.HumanName())

	var upstream *policy.Services
	var err error
	if job.UpstreamConfig.ApiEndpoint != "" && !job.UpstreamConfig.Incognito {
		log.Debug().Msg("using API endpoint " + job.UpstreamConfig.ApiEndpoint)
		upstream, err = policy.NewRemoteServices(job.UpstreamConfig.ApiEndpoint, job.UpstreamConfig.Plugins)
		if err != nil {
			log.Error().Err(err).Msg("could not connect to upstream")
		}
	}

	// run over all connections
	connections, err := resolver.OpenAssetConnections(job.Ctx, job.Asset, job.GetCredential, job.DoRecord)
	if err != nil {
		job.Reporter.AddScanError(job.Asset, err)
		if upstream != nil {
			_, err := upstream.SynchronizeAssets(job.Ctx, &policy.SynchronizeAssetsReq{
				SpaceMrn: job.UpstreamConfig.SpaceMrn,
				List:     []*asset.Asset{job.Asset},
			})
			if err != nil {
				log.Error().Err(err).Msgf("failed to synchronize asset %s", job.Asset.Mrn)
			}
		}
		return
	}

	for c := range connections {
		// We use a function since we want to close the motor once the current iteration finishes. If we directly
		// use defer in the loop m.Close() for each connection will only be executed once the entire loop is
		// finished.
		func(m *motor.Motor) {
			// ensures temporary files get deleted
			defer m.Close()

			log.Debug().Msg("established connection")
			// It's possible that the platform information was not collected at all or only partially during the
			// discovery phase.
			// For example, the ebs discovery does not detect the platform because it requires mounting
			// the filesystem. Another example is the docker container discovery, where it collects a lot of metadata
			// but does not have platform name and arch available.
			// TODO: It feels like this will only happen for performance optimizations. I think a better approach
			// would be to make it so that the motor used in the discovery phase gets reused here, instead
			// of being recreated.
			if job.Asset.Platform == nil || job.Asset.Platform.Name == "" {
				p, err := m.Platform()
				if err != nil {
					log.Warn().Err(err).Msg("failed to query platform information")
				} else {
					job.Asset.Platform = p
				}
			}

			if upstream != nil {
				resp, err := upstream.SynchronizeAssets(job.Ctx, &policy.SynchronizeAssetsReq{
					SpaceMrn: job.UpstreamConfig.SpaceMrn,
					List:     []*asset.Asset{job.Asset},
				})
				if err != nil {
					log.Error().Err(err).Msgf("failed to synchronize asset %s", job.Asset.Mrn)
				}

				log.Debug().Str("asset", job.Asset.Name).Strs("platform-ids", job.Asset.PlatformIds).Msg("update asset")
				platformId := job.Asset.PlatformIds[0]
				job.Asset.Mrn = resp.Details[platformId].AssetMrn
				job.Asset.Url = resp.Details[platformId].Url
				// only needed for CI/CD, can be removed when fleet and CI/CD have the same platform MRN
				job.Asset.PlatformIds = []string{resp.Details[platformId].PlatformMrn}
			}

			job.connection = m
			results, err := s.runMotorizedAsset(job)
			if err != nil {
				log.Warn().Err(err).Str("asset", job.Asset.Name).Msg("could not scan asset")
				job.Reporter.AddScanError(job.Asset, err)
				return
			}

			job.Reporter.AddReport(job.Asset, results)
		}(connections[c])
	}

	// When the progress bar is disabled there's no feedback when an asset is done scanning. Adding this message
	// such that it is visible from the logs.
	if s.disableProgressBar {
		log.Info().Msgf("scan for asset %s completed", job.Asset.HumanName())
	}
}

func (s *LocalScanner) runMotorizedAsset(job *AssetJob) (*AssetReport, error) {
	var res *AssetReport
	var policyErr error

	runtimeErr := inmemory.WithDb(s.resolvedPolicyCache, func(db *inmemory.Db, services *policy.LocalServices) error {
		if job.UpstreamConfig.ApiEndpoint != "" && !job.UpstreamConfig.Incognito {
			log.Debug().Msg("using API endpoint " + job.UpstreamConfig.ApiEndpoint)
			upstream, err := policy.NewRemoteServices(job.UpstreamConfig.ApiEndpoint, job.UpstreamConfig.Plugins)
			if err != nil {
				return err
			}
			services.Upstream = upstream
		}

		registry := all.Registry
		schema := registry.Schema()
		runtime := resources.NewRuntime(registry, job.connection)
		runtime.UpstreamConfig = &job.UpstreamConfig

		var progressListener progress.Progress
		if isatty.IsTerminal(os.Stdout.Fd()) && !s.disableProgressBar {
			progressListener = progress.New(job.Asset.Mrn, job.Asset.Name)
		} else {
			progressListener = progress.Noop{}
		}

		scanner := &localAssetScanner{
			db:       db,
			services: services,
			job:      job,
			fetcher:  s.fetcher,
			Registry: registry,
			Schema:   schema,
			Runtime:  runtime,
			Progress: progressListener,
		}
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
	inv := &v1.Inventory{
		Spec: &v1.InventorySpec{
			Assets: []*asset.Asset{
				{
					Connections: []*providers.Config{
						{
							Backend:  providers.ProviderType_K8S,
							Options:  opts,
							Discover: job.Discovery,
						},
					},
					Labels:   job.Labels,
					Category: asset.AssetCategory_CATEGORY_CICD,
				},
			},
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

	pClient, err := policy.NewRemoteServices(s.apiEndpoint, s.plugins)
	if err != nil {
		return nil, errors.Wrap(err, "could not initialize asset synchronization")
	}

	dar := &policy.DeleteAssetsRequest{
		SpaceMrn:        s.spaceMrn,
		ManagedBy:       garbageCollectOpts.MangagedBy,
		PlatformRuntime: garbageCollectOpts.PlatformRuntime,
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

	_, err = pClient.DeleteAssets(ctx, dar)
	if err != nil {
		log.Error().Err(err).Msg("error while trying to garbage collect assets")
	}
	return &Empty{}, err
}

type localAssetScanner struct {
	db       *inmemory.Db
	services *policy.LocalServices
	job      *AssetJob
	fetcher  *fetcher

	Registry *resources.Registry
	Schema   *resources.Schema
	Runtime  *resources.Runtime
	Progress progress.Progress
}

// run() runs a bundle on a single asset. It returns the results of the scan and an error if the scan failed. Even in
// case of an error, the results may contain partial results. The error is only returned if the scan failed to run not
// when individual policies failed.
func (s *localAssetScanner) run() (*AssetReport, error) {
	s.Progress.Open()

	// fallback to always close the progressbar if we error before getting the report
	defer s.Progress.Close()

	if err := s.prepareAsset(); err != nil {
		return nil, err
	}

	bundle, resolvedPolicy, err := s.runPolicy()
	if err != nil {
		return nil, err
	}

	ar := &AssetReport{
		Mrn:            s.job.Asset.Mrn,
		ResolvedPolicy: resolvedPolicy,
		Bundle:         bundle,
	}

	report, err := s.getReport()
	if err != nil {
		return ar, err
	}

	log.Debug().Str("asset", s.job.Asset.Mrn).Msg("scan complete")
	ar.Report = report
	return ar, nil
}

func (s *localAssetScanner) prepareAsset() error {
	var hub policy.PolicyHub = s.services

	// if we are using upstream we get the bundle from there
	if !s.job.UpstreamConfig.Incognito {
		return nil
	}

	if err := s.ensureBundle(); err != nil {
		return err
	}

	if s.job.Bundle == nil {
		return errors.New("no bundle provided to run")
	}

	if len(s.job.Bundle.Policies) == 0 {
		return errors.New("bundle doesn't contain any policies")
	}

	// FIXME: we do not currently respect policy filters!
	_, err := hub.SetBundle(s.job.Ctx, s.job.Bundle)
	if err != nil {
		return err
	}

	policyMrns := make([]string, len(s.job.Bundle.Policies))
	for i := range s.job.Bundle.Policies {
		policyMrns[i] = s.job.Bundle.Policies[i].Mrn
	}

	var resolver policy.PolicyResolver = s.services
	_, err = resolver.Assign(s.job.Ctx, &policy.PolicyAssignment{
		AssetMrn:   s.job.Asset.Mrn,
		PolicyMrns: policyMrns,
	})
	return err
}

var assetDetectBundle = executor.MustCompile("asset { kind platform runtime version family }")

func (s *localAssetScanner) ensureBundle() error {
	if s.job.Bundle != nil {
		return nil
	}

	features := cnquery.GetFeatures(s.job.Ctx)
	_, res, err := executor.ExecuteQuery(s.Schema, s.Runtime, assetDetectBundle, nil, features)
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

	s.job.Bundle, err = s.fetcher.fetchBundles(s.job.Ctx, urls.Urls...)
	return err
}

func (s *localAssetScanner) runPolicy() (*policy.Bundle, *policy.ResolvedPolicy, error) {
	var hub policy.PolicyHub = s.services
	var resolver policy.PolicyResolver = s.services

	log.Debug().Str("asset", s.job.Asset.Mrn).Msg("client> request policies bundle for asset")
	assetBundle, err := hub.GetBundle(s.job.Ctx, &policy.Mrn{Mrn: s.job.Asset.Mrn})
	if err != nil {
		return nil, nil, err
	}
	log.Debug().Msg("client> got policy bundle")
	logger.TraceJSON(assetBundle)
	logger.DebugDumpJSON("assetBundle", assetBundle)

	rawFilters, err := hub.GetPolicyFilters(s.job.Ctx, &policy.Mrn{Mrn: s.job.Asset.Mrn})
	if err != nil {
		return nil, nil, err
	}
	log.Debug().Str("asset", s.job.Asset.Mrn).Msg("client> got policy filters")
	logger.TraceJSON(rawFilters)

	filters, err := s.UpdateFilters(rawFilters, 5*time.Second)
	if err != nil {
		return s.job.Bundle, nil, err
	}
	log.Debug().Str("asset", s.job.Asset.Mrn).Msg("client> shell update filters")
	logger.DebugJSON(filters)

	resolvedPolicy, err := resolver.ResolveAndUpdateJobs(s.job.Ctx, &policy.UpdateAssetJobsReq{
		AssetMrn:     s.job.Asset.Mrn,
		AssetFilters: filters,
	})
	if err != nil {
		return s.job.Bundle, resolvedPolicy, err
	}
	log.Debug().Str("asset", s.job.Asset.Mrn).Msg("client> got resolved policy bundle for asset")
	logger.DebugDumpJSON("resolvedPolicy", resolvedPolicy)

	features := cnquery.GetFeatures(s.job.Ctx)
	err = executor.ExecuteResolvedPolicy(s.Schema, s.Runtime, resolver, s.job.Asset.Mrn, resolvedPolicy, features, s.Progress)
	if err != nil {
		return nil, nil, err
	}

	return assetBundle, resolvedPolicy, nil
}

func (s *localAssetScanner) getReport() (*policy.Report, error) {
	var resolver policy.PolicyResolver = s.services

	// TODO: we do not needs this anymore since we recieve updates already
	log.Debug().Str("asset", s.job.Asset.Mrn).Msg("client> send all results")
	_, err := policy.WaitUntilDone(resolver, s.job.Asset.Mrn, s.job.Asset.Mrn, 1*time.Second)
	s.Progress.Close()
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
	return executor.ExecuteFilterQueries(s.Schema, s.Runtime, queries, timeout)
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
