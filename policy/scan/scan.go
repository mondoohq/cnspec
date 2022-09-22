package scan

import (
	"context"

	"github.com/gogo/status"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery"
	"go.mondoo.com/cnquery/motor/inventory"
	v1 "go.mondoo.com/cnquery/motor/inventory/v1"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/cnspec/policy/datalakes/inmemory"
	"google.golang.org/grpc/codes"
)

// 50MB default size
const ResolvedPolicyCacheSize = 52428800

type Job struct {
	DoRecord  bool
	Inventory *v1.Inventory
	Bundle    *policy.PolicyBundleMap
	Context   context.Context
}

type AssetJob struct {
	DoRecord bool
	Asset    *asset.Asset
	Bundle   *policy.PolicyBundleMap
	Context  context.Context
}

type LocalService struct {
	resolvedPolicyCache *inmemory.ResolvedPolicyCache
}

func NewLocalService() *LocalService {
	return &LocalService{
		resolvedPolicyCache: inmemory.NewResolvedPolicyCache(ResolvedPolicyCacheSize),
	}
}

func (s *LocalService) RunIncognito(job *Job) ([]*policy.Report, error) {
	if job == nil {
		return nil, status.Errorf(codes.InvalidArgument, "missing scan job")
	}

	if job.Inventory == nil {
		return nil, status.Errorf(codes.InvalidArgument, "missing inventory")
	}

	if job.Context == nil {
		return nil, errors.New("no context provided to run job with local scanner")
	}

	// Inventory
	ctx := discovery.InitCtx(job.Context)

	log.Info().Msgf("discover related assets for %d asset(s)", len(job.Inventory.Spec.Assets))
	im, err := inventory.New(inventory.WithInventory(job.Inventory))
	if err != nil {
		return nil, errors.Wrap(err, "could not load asset information")
	}
	assetErrors := im.Resolve(ctx)
	if len(assetErrors) > 0 {
		for a := range assetErrors {
			log.Error().Err(assetErrors[a]).Str("asset", a.Name).Msg("could not resolve asset")
		}
		return nil, errors.New("failed to resolve multiple assets")
	}

	assetList := im.GetAssets()
	if len(assetList) == 0 {
		return nil, errors.New("could not find an asset that we can connect to")
	}

	reports := []*policy.Report{}
	for i := range assetList {
		report, err := s.RunAssetIncognito(&AssetJob{
			DoRecord: job.DoRecord,
			Asset:    assetList[i],
			Bundle:   job.Bundle,
			Context:  ctx,
		})
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan asset "+assetList[i].Id)
		}
		reports = append(reports, report)
	}

	return reports, nil
}

func (s *LocalService) RunAssetIncognito(job *AssetJob) (*policy.Report, error) {
	panic("run asset incognito")
}
