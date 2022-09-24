package scan

import (
	"context"

	"github.com/gogo/status"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery"
	"go.mondoo.com/cnquery/motor/inventory"
	v1 "go.mondoo.com/cnquery/motor/inventory/v1"
	"go.mondoo.com/cnquery/motor/providers/resolver"
	"go.mondoo.com/cnquery/motor/vault"
	"go.mondoo.com/cnspec/internal/datalakes/inmemory"
	"go.mondoo.com/cnspec/policy"
	"google.golang.org/grpc/codes"
)

// 50MB default size
const ResolvedPolicyCacheSize = 52428800

type Job struct {
	DoRecord  bool
	Inventory *v1.Inventory
	Bundle    *policy.PolicyBundleMap
	Ctx       context.Context
}

type AssetJob struct {
	DoRecord      bool
	Asset         *asset.Asset
	Bundle        *policy.PolicyBundleMap
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

func (s *LocalScanner) RunIncognito(job *Job) ([]*policy.Report, error) {
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

func (s *LocalScanner) distributeJob(job *Job, ctx context.Context) ([]*policy.Report, bool, error) {
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

	reporter := NewAggregateReporter()

	for i := range assetList {
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
			Asset:         assetList[i],
			Bundle:        job.Bundle,
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

		scanner := &localAssetScanner{
			db:       db,
			services: services,
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
}

func (l *localAssetScanner) run() (*AssetReport, error) {
	if err := l.prepareAsset(); err != nil {
		return nil, err
	}

	bundle, resolvedPolicy, err := l.runPolicy()
	if err != nil {
		return nil, err
	}

	report, err := l.getReport()
	if err != nil {
		return nil, err
	}

	log.Debug().Str("asset", l.job.Asset.Mrn).Msg("scan complete")
	return &AssetReport{
		Mrn:            l.job.Asset.Mrn,
		ResolvedPolicy: resolvedPolicy,
		Bundle:         bundle,
		Report:         report,
	}, nil
}

func (s *localAssetScanner) prepareAsset() error {
	panic("implement prepareAsset")
	return nil
}

func (s *localAssetScanner) runPolicy() (*policy.PolicyBundle, *policy.ResolvedPolicy, error) {
	s.services.Datalakes
	panic("implement runPolicy")
	return nil, nil, nil
}

func (s *localAssetScanner) getReport() (*policy.Report, error) {
	panic("implement getReport")
	return nil, nil
}
