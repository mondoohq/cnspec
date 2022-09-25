package policy

import (
	"context"

	"github.com/gogo/status"
	"github.com/pkg/errors"
	"go.mondoo.com/cnquery/logger"
	"go.opentelemetry.io/otel"
	"google.golang.org/grpc/codes"
)

var tracer = otel.Tracer("go.mondoo.com/cnspec/policy")

func (s *LocalServices) setPolicyFromBundle(ctx context.Context, policyObj *Policy, bundleMap *PolicyBundleMap) error {
	logCtx := logger.FromContext(ctx)
	policyObj, filters, err := s.PreparePolicy(ctx, policyObj, bundleMap)
	if err != nil {
		return err
	}

	err = s.DataLake.SetPolicy(ctx, policyObj, filters)
	if err != nil {
		return err
	}

	// necessary to refresh the bundle
	_, err = s.DataLake.GetValidatedBundle(ctx, policyObj.Mrn)
	if err != nil {
		logCtx.Error().
			Str("name", policyObj.Name).
			Str("mrn", policyObj.Mrn).
			Err(err).
			Msg("marketplace> ensure policyBundle error")
		return err
	}

	return nil
}

func (s *LocalServices) setPolicyBundleFromMap(ctx context.Context, bundleMap *PolicyBundleMap) error {
	logCtx := logger.FromContext(ctx)

	// sort policies, so that we store child policies before their parents
	policies, err := bundleMap.PoliciesSortedByDependency()
	if err != nil {
		return err
	}

	for i := range policies {
		policyObj := policies[i]
		logCtx.Debug().Str("owner", policyObj.OwnerMrn).Str("uid", policyObj.Uid).Str("mrn", policyObj.Mrn).Msg("store policy")
		policyObj.OwnerMrn = bundleMap.OwnerMrn

		// If this is a user generated policy, it must be non-public
		if bundleMap.OwnerMrn != "//policy.api.mondoo.app" {
			policyObj.IsPublic = false
		}

		if err = s.setPolicyFromBundle(ctx, policyObj, bundleMap); err != nil {
			return err
		}
	}

	return nil
}

// GetPolicy without cascading dependencies
func (s *LocalServices) GetPolicy(ctx context.Context, in *Mrn) (*Policy, error) {
	logCtx := logger.FromContext(ctx)

	if in == nil || len(in.Mrn) == 0 {
		return nil, status.Error(codes.InvalidArgument, "policy mrn is required")
	}

	b, err := s.DataLake.GetValidatedPolicy(ctx, in.Mrn)
	if err == nil {
		logCtx.Debug().Str("policy", in.Mrn).Err(err).Msg("marketplace> get policy bundle from db")
		return b, nil
	}
	if s.Upstream == nil {
		return nil, err
	}

	// try upstream; once it's cached, try again
	_, err = s.cacheUpstreamPolicy(ctx, in.Mrn)
	if err != nil {
		return nil, err
	}
	return s.DataLake.GetValidatedPolicy(ctx, in.Mrn)
}

// GetPolicyBundle retrieves the given policy and all its dependencies (policies/queries)
func (s *LocalServices) GetPolicyBundle(ctx context.Context, in *Mrn) (*PolicyBundle, error) {
	if in == nil || len(in.Mrn) == 0 {
		return nil, status.Error(codes.InvalidArgument, "policy mrn is required")
	}

	b, err := s.DataLake.GetValidatedBundle(ctx, in.Mrn)
	if err == nil {
		return b, nil
	}
	if s.Upstream == nil {
		return nil, err
	}

	// try upstream
	return s.cacheUpstreamPolicy(ctx, in.Mrn)
}

// cacheUpstreamPolicy by storing a copy of the upstream policy bundle in this db
// Note: upstream marketplace has to be defined
func (s *LocalServices) cacheUpstreamPolicy(ctx context.Context, mrn string) (*PolicyBundle, error) {
	logCtx := logger.FromContext(ctx)
	if s.Upstream == nil {
		return nil, errors.New("failed to retrieve upstream policy " + mrn + " since upstream is not defined")
	}

	logCtx.Debug().Str("policy", mrn).Msg("marketplace> fetch policy bundle from upstream")
	bundle, err := s.Upstream.GetPolicyBundle(ctx, &Mrn{Mrn: mrn})
	if err != nil {
		logCtx.Error().Err(err).Str("policy", mrn).Msg("marketplace> failed to retrieve policy bundle from upstream")
		return nil, errors.New("failed to retrieve upstream policy " + mrn + ": " + err.Error())
	}

	// fixme - this is a hack, more deets at method definition
	FixZeroValuesInPolicyBundle(bundle)

	bundleMap := bundle.ToMap()

	err = s.setPolicyBundleFromMap(ctx, bundleMap)
	if err != nil {
		logCtx.Error().Err(err).Str("policy", mrn).Msg("marketplace> failed to set policy bundle retrieved from upstream")
		return nil, errors.New("failed to cache upstream policy " + mrn + ": " + err.Error())
	}

	logCtx.Debug().Str("policy", mrn).Msg("marketplace> fetched policy bundle from upstream")
	return bundle, nil
}

func (s *LocalServices) DeletePolicy(ctx context.Context, in *Mrn) (*Empty, error) {
	if in == nil || len(in.Mrn) == 0 {
		return nil, status.Error(codes.InvalidArgument, "policy MRN is required")
	}

	return globalEmpty, s.DataLake.DeletePolicy(ctx, in.Mrn)
}

// fixme - this is a hack to deal with the fact that zero valued ScoringSpecs are getting deserialized
// instead of nil pointers for ScoringSpecs.
// This is a quick fix for https://gitlab.com/mondoolabs/mondoo/-/issues/455
// so that we can get a fix out while figuring out wtf is up with our null pointer serialization
// open issue for deserialization: https://gitlab.com/mondoolabs/mondoo/-/issues/508
func FixZeroValuesInPolicyBundle(bundle *PolicyBundle) {
	for _, policy := range bundle.Policies {
		for _, spec := range policy.Specs {
			if spec.Policies != nil {
				for k, v := range spec.Policies {
					// v.Action is only 0 for zero value structs
					if v != nil && v.Action == 0 {
						spec.Policies[k] = nil
					}
				}
			}
			if spec.ScoringQueries != nil {
				for k, v := range spec.ScoringQueries {
					// v.Action is only 0 for zero value structs
					if v != nil && v.Action == 0 {
						spec.ScoringQueries[k] = nil
					}
				}
			}
		}
	}
}
