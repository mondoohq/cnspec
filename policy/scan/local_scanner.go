package scan

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/ksuid"
	"go.mondoo.com/cnquery"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/logger"
	"go.mondoo.com/cnquery/motor"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery"
	"go.mondoo.com/cnquery/motor/inventory"
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
	apiEndpoint string
	spaceMrn    string
	plugins     []ranger.ClientPlugin
}

type ScannerOption func(*LocalScanner)

func WithUpstream(apiEndpoint string, spaceMrn string, plugins []ranger.ClientPlugin) func(s *LocalScanner) {
	return func(s *LocalScanner) {
		s.apiEndpoint = apiEndpoint
		s.plugins = plugins
		s.spaceMrn = spaceMrn
	}
}

func NewLocalScanner(opts ...ScannerOption) *LocalScanner {
	ls := &LocalScanner{
		resolvedPolicyCache: inmemory.NewResolvedPolicyCache(ResolvedPolicyCacheSize),
		fetcher:             newFetcher(),
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

func (s *LocalScanner) Run(ctx context.Context, job *Job) (*policy.ReportCollection, error) {
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

func (s *LocalScanner) RunIncognito(ctx context.Context, job *Job) (*policy.ReportCollection, error) {
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

func (s *LocalScanner) distributeJob(job *Job, ctx context.Context, upstreamConfig resources.UpstreamConfig) (*policy.ReportCollection, bool, error) {
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

	// sync assets
	if upstreamConfig.ApiEndpoint != "" && !upstreamConfig.Incognito {
		log.Info().Msg("syncing assets")
		upstream, err := policy.NewRemoteServices(s.apiEndpoint, s.plugins)
		if err != nil {
			return nil, false, err
		}
		resp, err := upstream.SynchronizeAssets(ctx, &policy.SynchronizeAssetsReq{
			SpaceMrn: s.spaceMrn,
			List:     assetList,
		})
		if err != nil {
			return nil, false, err
		}
		log.Debug().Int("assets", len(resp.Details)).Msg("got assets details")
		platformAssetMapping := make(map[string]*policy.SynchronizeAssetsRespAssetDetail)
		for i := range resp.Details {
			log.Debug().Str("platform-mrn", resp.Details[i].PlatformMrn).Str("asset", resp.Details[i].AssetMrn).Msg("asset mapping")
			platformAssetMapping[resp.Details[i].PlatformMrn] = resp.Details[i]
		}

		// attach the asset details to the assets list
		for i := range assetList {
			log.Debug().Str("asset", assetList[i].Name).Strs("platform-ids", assetList[i].PlatformIds).Msg("update asset")
			platformMrn, err := s.getPlatformMrnFromAsset(assetList[i])
			if err != nil {
				return nil, false, errors.Wrap(err, "failed to generate a platform MRN")
			}
			assetList[i].Mrn = platformAssetMapping[platformMrn].AssetMrn
			assetList[i].Url = platformAssetMapping[platformMrn].Url
		}
	} else {
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
	reporter := NewAggregateReporter(assetList)
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

	// run over all connections
	connections, err := resolver.OpenAssetConnections(job.Ctx, job.Asset, job.GetCredential, job.DoRecord)
	if err != nil {
		job.Reporter.AddScanError(job.Asset, err)
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
					// resyncAssets = append(resyncAssets, assetEntry)
				}
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
}

func (s *LocalScanner) runMotorizedAsset(job *AssetJob) (*AssetReport, error) {
	var res *AssetReport
	var policyErr error

	runtimeErr := inmemory.WithDb(s.resolvedPolicyCache, func(db *inmemory.Db, services *policy.LocalServices) error {
		if job.UpstreamConfig.ApiEndpoint != "" && !job.UpstreamConfig.Incognito {
			log.Debug().Msg("using API endpoint " + s.apiEndpoint)
			upstream, err := policy.NewRemoteServices(s.apiEndpoint, s.plugins)
			if err != nil {
				return err
			}
			services.Upstream = upstream
		}

		registry := all.Registry
		schema := registry.Schema()
		runtime := resources.NewRuntime(registry, job.connection)
		runtime.UpstreamConfig = &job.UpstreamConfig

		scanner := &localAssetScanner{
			db:       db,
			services: services,
			job:      job,
			fetcher:  s.fetcher,
			Registry: registry,
			Schema:   schema,
			Runtime:  runtime,
			Progress: progress.New(job.Asset.Mrn, job.Asset.Name),
		}
		res, policyErr = scanner.run()
		return policyErr
	})
	if runtimeErr != nil {
		return res, runtimeErr
	}

	return res, policyErr
}

type detectedCicdProject struct {
	Name      string
	ProjectID string
	Type      string
}

func returnTheEmptyOnes(labels map[string]string, required []string) []string {
	empty := []string{}
	for _, l := range required {
		if labels[l] == "" {
			empty = append(empty, l)
		}
	}
	return empty
}

const CICDPlatformIdPrefix = "//platformid.api.mondoo.app/runtime/cicd/"

// getPlatformMrnFromAsset read the asset labels and determines the project
// To achieve this it builds up a project identifier that can be used
// to re-recognize it over many runs.
func (s *LocalScanner) getPlatformMrnFromAsset(in *asset.Asset) (string, error) {
	if in.Category != asset.AssetCategory_CATEGORY_CICD {
		return in.PlatformIds[0], nil
	}

	cicdDetected := detectedCicdProject{}
	projectId := ""
	platformId := ""

	labels := in.Labels
	switch labels["mondoo.com/exec-environment"] {
	case "actions.github.com":
		cicdDetected.Type = labels["mondoo.com/exec-environment"]
		cicdDetected.Name = labels["actions.github.com/repository"]
		safeRef := mrn.SafeComponentString(labels["actions.github.com/ref"])
		runID := mrn.SafeComponentString(labels["actions.github.com/run-id"])
		job := mrn.SafeComponentString(labels["actions.github.com/job"])
		action := mrn.SafeComponentString(labels["actions.github.com/action"])

		if cicdDetected.Name != "" && safeRef != "" && runID != "" && job != "" && action != "" {
			projectId = CICDPlatformIdPrefix + "actions.github.com/" + mrn.SafeComponentString(cicdDetected.Name)
			platformId = projectId + "/ref/" + safeRef + "/run/" + runID + "/job/" + job + "/action/" + action
		} else {
			return "", fmt.Errorf("missing required env var for cicd asset: %v", returnTheEmptyOnes(labels, []string{"actions.github.com/repository", "actions.github.com/ref", "actions.github.com/run-id", "actions.github.com/job", "actions.github.com/action"}))
		}

	case "gitlab.com":
		cicdDetected.Type = labels["mondoo.com/exec-environment"]
		cicdDetected.Name = labels["gitlab.com/project-path"]
		// TODO(jaym): The docs dont mention CI_COMMIT_REF_NAME works with
		// pull requests
		safeRef := mrn.SafeComponentString(labels["gitlab.com/commit-ref-name"])
		jobID := mrn.SafeComponentString(labels["gitlab.com/job-id"])

		if cicdDetected.Name != "" && safeRef != "" && jobID != "" {
			projectId = CICDPlatformIdPrefix + "gitlab.com/" + mrn.SafeComponentString(cicdDetected.Name)
			platformId = projectId + "/ref/" + safeRef + "/run/" + jobID
		} else {
			return "", fmt.Errorf("missing required env var for cicd asset: %v", returnTheEmptyOnes(labels, []string{"gitlab.com/project-path", "gitlab.com/commit-ref-name", "gitlab.com/job-id"}))
		}

	case "k8s.mondoo.com":
		cicdDetected.Type = labels["mondoo.com/exec-environment"]
		// TODO: allow users to define a cluster name with the integration
		cicdDetected.Name = "K8S Cluster " + labels["k8s.mondoo.com/cluster-id"]
		clusterID := mrn.SafeComponentString(labels["k8s.mondoo.com/cluster-id"])
		resourceUID := mrn.SafeComponentString(labels["k8s.mondoo.com/uid"])
		resourceVersion := mrn.SafeComponentString(labels["k8s.mondoo.com/resource-version"])

		// resource version is important but not always there, the CREATE event has no resourceVersion yet
		if clusterID != "" {
			projectId = CICDPlatformIdPrefix + "k8s.mondoo.com/" + clusterID
			if resourceVersion != "" {
				platformId = projectId + "/" + resourceUID + "/" + resourceVersion
			} else {
				platformId = projectId + "/" + resourceUID
			}
		} else if clusterID == "" || resourceUID == "" {
			return "", fmt.Errorf("missing required env var for cicd asset: %v", returnTheEmptyOnes(labels, []string{"k8s.mondoo.com/cluster-id", "k8s.mondoo.com/uid"}))
		}

	case "circleci.com":
		cicdDetected.Type = labels["mondoo.com/exec-environment"]
		cicdDetected.Name = labels["circleci.com/project-reponame"]
		safeRef := mrn.SafeComponentString(labels["circleci.com/sha1"])
		jobID := mrn.SafeComponentString(labels["circleci.com/build-num"])

		if cicdDetected.Name != "" && safeRef != "" && jobID != "" {
			projectId = CICDPlatformIdPrefix + "circleci.com/" + mrn.SafeComponentString(cicdDetected.Name)
			platformId = projectId + "/ref/" + safeRef + "/run/" + jobID
		} else {
			return "", fmt.Errorf("missing required env var for cicd asset: %v", returnTheEmptyOnes(labels, []string{"circleci.com/project-reponame", "circleci.com/sha1", "circleci.com/build-num"}))
		}

	case "devops.azure.com":
		cicdDetected.Type = labels["mondoo.com/exec-environment"]
		cicdDetected.Name = labels["devops.azure.com/repository-name"]
		safeRef := mrn.SafeComponentString(labels["devops.azure.com/sourceversion"])
		jobID := mrn.SafeComponentString(labels["devops.azure.com/buildid"])

		if cicdDetected.Name != "" && safeRef != "" && jobID != "" {
			projectId = CICDPlatformIdPrefix + "devops.azure.com/" + mrn.SafeComponentString(cicdDetected.Name)
			platformId = projectId + "/ref/" + safeRef + "/run/" + jobID
		} else {
			return "", fmt.Errorf("missing required env var for cicd asset: %v", returnTheEmptyOnes(labels, []string{"devops.azure.com/repository-name", "devops.azure.com/sourceversion", "devops.azure.com/buildid"}))
		}

	case "jenkins.io":
		cicdDetected.Type = labels["mondoo.com/exec-environment"]
		cicdDetected.Name = labels["jenkins.io/jobname"]
		safeRef := mrn.SafeComponentString(labels["jenkins.io/gitcommit"])
		if safeRef == "" {
			log.Warn().Msg("no git commit value found in env for jenkins job, using job name")
			safeRef = labels["jenkins.io/jobname"]
		}
		jobID := mrn.SafeComponentString(labels["jenkins.io/buildid"])

		if cicdDetected.Name != "" && safeRef != "" && jobID != "" {
			projectId = CICDPlatformIdPrefix + "jenkins.io/" + mrn.SafeComponentString(cicdDetected.Name)
			platformId = projectId + "/ref/" + safeRef + "/run/" + jobID
		} else {
			return "", fmt.Errorf("missing required env var for cicd asset: %v", returnTheEmptyOnes(labels, []string{"jenkins.io/jobname", "jenkins.io/gitcommit", "jenkins.io/buildid", "jenkins.io/jobname"}))
		}

	default:
		return "", errors.New("unexpected mondoo.com/exec-environment for cicd asset: " + labels["mondoo.com/exec-environment"])
	}

	if projectId == "" || platformId == "" {
		return "", errors.New("could not determine projectId or platformId for cicd asset")
	}

	if strings.HasPrefix(in.PlatformIds[0], CICDPlatformIdPrefix) {
		platformId = in.PlatformIds[0]
	} else {
		// Since we can have >1 asset for a single CI/CD scan, we hash the fleet platformId for the asset and append it
		// to the CI/CD platformId. In this way we make sure each asset from a single CI/CD scan gets a unique platformId.
		h := sha256.New()
		h.Write([]byte(in.PlatformIds[0]))
		hash := hex.EncodeToString(h.Sum(nil))
		platformId = platformId + "/hash/" + hash
	}

	cicdDetected.ProjectID = projectId

	return platformId, nil
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
