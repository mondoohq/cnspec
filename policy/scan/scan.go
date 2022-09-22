package scan

import (
	"github.com/gogo/status"
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
}

type LocalService struct {
	resolvedPolicyCache *inmemory.ResolvedPolicyCache
}

func NewLocalService() *LocalService {
	return &LocalService{
		resolvedPolicyCache: inmemory.NewResolvedPolicyCache(ResolvedPolicyCacheSize),
	}
}

func (s LocalService) RunIncognito(job *Job) (*policy.Report, error) {
	if job == nil {
		return nil, status.Errorf(codes.InvalidArgument, "missing scan job")
	}

	if job.Inventory == nil {
		return nil, status.Errorf(codes.InvalidArgument, "missing inventory")
	}

	panic("NOT YET IMPLEMENTED")
	return nil, nil
}
