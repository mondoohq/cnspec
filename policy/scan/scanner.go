package scan

import (
	"context"
	"encoding/base64"

	"go.mondoo.com/cnquery/cli/execruntime"
	"go.mondoo.com/cnquery/motor/asset"
	v1 "go.mondoo.com/cnquery/motor/inventory/v1"
	providers "go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/ranger-rpc/codes"
	"go.mondoo.com/ranger-rpc/status"
)

type Scanner struct {
	localScanner *LocalScanner
}

func NewScanner(upstreamConfig resources.UpstreamConfig) *Scanner {
	return &Scanner{localScanner: NewLocalScanner(WithUpstream(upstreamConfig.ApiEndpoint, upstreamConfig.SpaceMrn, upstreamConfig.Plugins))}
}

func (s *Scanner) Run(ctx context.Context, job *Job) (*ScanResult, error) {
	return s.localScanner.Run(ctx, job)
}

func (s *Scanner) RunIncognito(ctx context.Context, job *Job) (*ScanResult, error) {
	return s.localScanner.RunIncognito(ctx, job)
}

func (s *Scanner) Schedule(ctx context.Context, job *Job) (*Empty, error) {
	if job == nil {
		return nil, status.Errorf(codes.InvalidArgument, "missing scan job")
	}

	if s.localScanner.queue == nil {
		return nil, status.Errorf(codes.Unavailable, "job queue is not available")
	}

	s.localScanner.queue.Channel() <- *job
	return &Empty{}, nil
}

func (s *Scanner) RunAdmissionReview(ctx context.Context, job *AdmissionReviewJob) (*ScanResult, error) {
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

func (s *Scanner) GarbageCollectAssets(ctx context.Context, garbageCollectOpts *GarbageCollectOptions) (*Empty, error) {
	// if garbageCollectOpts == nil {
	// 	return nil, status.Errorf(codes.InvalidArgument, "missing garbage collection options")
	// }

	// pClient, err := mp.NewRemoteServices(s.opts.ApiEndpoint, s.opts.Plugins)
	// if err != nil {
	// 	return nil, errors.Wrap(err, "could not initialize asset synchronization")
	// }

	// dar := &mp.DeleteAssetsRequest{
	// 	SpaceMrn:        s.opts.SpaceMrn,
	// 	ManagedBy:       garbageCollectOpts.MangagedBy,
	// 	PlatformRuntime: garbageCollectOpts.PlatformRuntime,
	// }

	// if garbageCollectOpts.OlderThan != "" {
	// 	timestamp, err := time.Parse(time.RFC3339, garbageCollectOpts.OlderThan)
	// 	if err != nil {
	// 		return nil, errors.Wrap(err, "failed converting timestamp from RFC3339 format")
	// 	}

	// 	dar.DateFilter = &mp.DateFilter{
	// 		Timestamp: timestamp.Format(time.RFC3339),
	// 		// LESS_THAN b/c we want assets with a lastUpdated timestamp older
	// 		// (ie timewise considered less) than the timestamp provided
	// 		Comparison: mp.Comparison_LESS_THAN,
	// 		Field:      mp.DateFilterField_FILTER_LAST_UPDATED,
	// 	}
	// }

	// _, err = pClient.DeleteAssets(ctx, dar)
	// if err != nil {
	// 	log.Error().Err(err).Msg("error while trying to garbage collect assets")
	// }
	// return nil, err
	return &Empty{}, nil
}
