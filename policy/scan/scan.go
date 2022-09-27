package scan

import (
	"context"
	"math/rand"
	"strings"
	"time"

	"github.com/gogo/status"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/ksuid"
	"go.mondoo.com/cnquery"
	"go.mondoo.com/cnquery/logger"
	"go.mondoo.com/cnquery/motor"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery"
	"go.mondoo.com/cnquery/motor/inventory"
	v1 "go.mondoo.com/cnquery/motor/inventory/v1"
	"go.mondoo.com/cnquery/motor/providers/resolver"
	"go.mondoo.com/cnquery/motor/vault"
	"go.mondoo.com/cnquery/mrn"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/all"
	"go.mondoo.com/cnspec/cli/progress"
	"go.mondoo.com/cnspec/internal/datalakes/inmemory"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/cnspec/policy/executor"
	"google.golang.org/grpc/codes"
)

// 50MB default size
const ResolvedPolicyCacheSize = 52428800

func init() {
	rand.Seed(time.Now().UnixNano())
}

type Job struct {
	DoRecord      bool
	Inventory     *v1.Inventory
	Bundle        *policy.PolicyBundle
	PolicyFilters []string
	Ctx           context.Context
}

type AssetJob struct {
	DoRecord      bool
	Asset         *asset.Asset
	Bundle        *policy.PolicyBundle
	PolicyFilters []string
	Ctx           context.Context
	GetCredential func(cred *vault.Credential) (*vault.Credential, error)
	Reporter      Reporter
	connection    *motor.Motor
}

type AssetReport struct {
	Mrn            string
	ResolvedPolicy *policy.ResolvedPolicy
	Bundle         *policy.PolicyBundle
	Report         *policy.Report
}

type LocalScanner struct {
	resolvedPolicyCache *inmemory.ResolvedPolicyCache
}

func NewLocalScanner() *LocalScanner {
	return &LocalScanner{
		resolvedPolicyCache: inmemory.NewResolvedPolicyCache(ResolvedPolicyCacheSize),
	}
}

func (s *LocalScanner) RunIncognito(job *Job) (*policy.ReportCollection, error) {
	if job == nil {
		return nil, status.Errorf(codes.InvalidArgument, "missing scan job")
	}

	if job.Inventory == nil {
		return nil, status.Errorf(codes.InvalidArgument, "missing inventory")
	}

	if job.Ctx == nil {
		return nil, errors.New("no context provided to run job with local scanner")
	}

	ctx := discovery.InitCtx(job.Ctx)

	reports, _, err := s.distributeJob(job, ctx)
	if err != nil {
		return nil, err
	}

	return reports, nil
}

func (s *LocalScanner) distributeJob(job *Job, ctx context.Context) (*policy.ReportCollection, bool, error) {
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

	reporter := NewAggregateReporter(job.Bundle, assetList)

	for i := range assetList {
		asset := assetList[i]

		// Make sure the context has not been canceled in the meantime. Note that this approach works only for single threaded execution. If we have more than 1 thread calling this function,
		// we need to solve this at a different level.
		select {
		case <-ctx.Done():
			log.Warn().Msg("request context has been canceled")
			return reporter.Reports(), false, reporter.Error()
		default:
		}

		s.RunAssetJob(&AssetJob{
			DoRecord:      job.DoRecord,
			Asset:         asset,
			Bundle:        job.Bundle,
			PolicyFilters: job.PolicyFilters,
			Ctx:           ctx,
			GetCredential: im.GetCredential,
			Reporter:      reporter,
		})
	}

	return reporter.Reports(), true, reporter.Error()
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
			policyResults, err := s.runMotorizedAsset(job)

			if err != nil {
				job.Reporter.AddScanError(job.Asset, err)
				return
			}

			job.Reporter.AddReport(job.Asset, policyResults)

		}(connections[c])
	}
}

func (s *LocalScanner) runMotorizedAsset(job *AssetJob) (*AssetReport, error) {
	var res *AssetReport
	var policyErr error

	runtimeErr := inmemory.WithDb(s.resolvedPolicyCache, func(db *inmemory.Db, services *policy.LocalServices) error {
		if services.Upstream != nil {
			panic("cannot work with upstream yet")
		}

		registry := all.Registry
		schema := registry.Schema()
		runtime := resources.NewRuntime(registry, job.connection)

		scanner := &localAssetScanner{
			db:       db,
			services: services,
			job:      job,
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

type localAssetScanner struct {
	db       *inmemory.Db
	services *policy.LocalServices
	job      *AssetJob

	Registry *resources.Registry
	Schema   *resources.Schema
	Runtime  *resources.Runtime
	Progress progress.Progress
}

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

	report, err := s.getReport()
	if err != nil {
		return nil, err
	}

	log.Debug().Str("asset", s.job.Asset.Mrn).Msg("scan complete")
	return &AssetReport{
		Mrn:            s.job.Asset.Mrn,
		ResolvedPolicy: resolvedPolicy,
		Bundle:         bundle,
		Report:         report,
	}, nil
}

func (s *localAssetScanner) prepareAsset() error {
	var hub policy.PolicyHub = s.services
	// FIXME: we do not currently respect policy filters!
	_, err := hub.SetPolicyBundle(s.job.Ctx, s.job.Bundle)
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

func (s *localAssetScanner) runPolicy() (*policy.PolicyBundle, *policy.ResolvedPolicy, error) {
	var hub policy.PolicyHub = s.services
	var resolver policy.PolicyResolver = s.services

	log.Debug().Str("asset", s.job.Asset.Mrn).Msg("client> request policies bundle for asset")
	assetBundle, err := hub.GetPolicyBundle(s.job.Ctx, &policy.Mrn{Mrn: s.job.Asset.Mrn})
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

	return s.job.Bundle, resolvedPolicy, nil
}

func (s *localAssetScanner) getReport() (*policy.Report, error) {
	var resolver policy.PolicyResolver = s.services

	// TODO: we do not needs this anymore since we recieve updates already
	log.Info().Str("asset", s.job.Asset.Mrn).Msg("client> send all results")
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
	report, err := resolver.GetReport(s.job.Ctx, &policy.EntityScoreRequest{
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
