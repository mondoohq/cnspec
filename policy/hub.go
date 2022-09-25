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

// SetPolicyBundle stores a bundle of policies and queries in this marketplace
func (s *LocalServices) SetPolicyBundle(ctx context.Context, bundle *PolicyBundle) (*Empty, error) {
	if len(bundle.OwnerMrn) == 0 {
		return globalEmpty, status.Error(codes.InvalidArgument, "owner MRN is required")
	}

	// See https://gitlab.com/mondoolabs/mondoo/-/issues/595
	FixZeroValuesInPolicyBundle(bundle)

	bundleMap, err := bundle.Compile(ctx, s.DataLake)
	if err != nil {
		return globalEmpty, err
	}

	if err := s.setPolicyBundleFromMap(ctx, bundleMap); err != nil {
		return nil, err
	}

	return globalEmpty, nil
}

// PreparePolicy takes a policy and an optional bundle and gets it
// ready to be saved in the DB, including asset filters.
// Note1: The bundle must have been pre-compiled and validated!
// Note2: The bundle may be nil, in which case we will try to find what is needed for the policy
// Note3: We create the ent.PolicyBundle in this function, not in the `SetPolicyBundle`
//
//	Reason: SetPolicyBundle may be setting 1 outer and 3 embedded policies.
//	But we need to create ent.PolicyBundles for all 4 of those.
func (s *LocalServices) PreparePolicy(ctx context.Context, policyObj *Policy, bundle *PolicyBundleMap) (*Policy, []*Mquery, error) {
	logCtx := logger.FromContext(ctx)
	var err error

	if policyObj == nil || len(policyObj.Mrn) == 0 {
		return nil, nil, status.Error(codes.InvalidArgument, "policy mrn is required")
	}

	if len(policyObj.OwnerMrn) == 0 {
		return nil, nil, status.Error(codes.InvalidArgument, "owner mrn is required")
	}

	policyObj.RefreshLocalAssetFilters()

	// store all queries
	// NOTE: if we modify the spec only, we may not have the queries available e.g. used by ApplyScoringMutation
	// FIXME: we need to verify that the policy has access to all referenced queries
	if bundle != nil {
		dataQueries := map[string]*Mquery{}
		scoredQueries := map[string]*Mquery{}
		for i := range policyObj.Specs {
			spec := policyObj.Specs[i]
			for k := range spec.DataQueries {
				q, ok := bundle.Queries[k]
				if !ok {
					return nil, nil, status.Error(codes.InvalidArgument, "policy "+policyObj.Mrn+" is referencing unknown query "+k)
				}
				dataQueries[k] = q
			}
			for k := range spec.ScoringQueries {
				q, ok := bundle.Queries[k]
				if !ok {
					return nil, nil, status.Error(codes.InvalidArgument, "policy "+policyObj.Mrn+" is referencing unknown query "+k)
				}
				scoredQueries[k] = q
			}
		}

		propsQueries := map[string]*Mquery{}
		for k := range policyObj.Props {
			q, ok := bundle.Props[k]
			if !ok {
				return nil, nil, status.Error(codes.InvalidArgument, "policy "+policyObj.Mrn+" is referencing unknown property "+k)
			}
			propsQueries[k] = q
		}

		// TODO: this may need to happen in a bulk call
		for k, v := range dataQueries {
			if err := s.setQuery(ctx, k, v, false); err != nil {
				return nil, nil, err
			}
		}
		for k, v := range scoredQueries {
			if err := s.setQuery(ctx, k, v, true); err != nil {
				return nil, nil, err
			}
		}
		for k, v := range propsQueries {
			if err := s.setQuery(ctx, k, v, true); err != nil {
				return nil, nil, err
			}
		}
	}

	// TODO: we need to decide if it is up to the caller to ensure that the checksum is up-to-date
	// e.g. ApplyScoringMutation changes the spec. Right now we assume the caller invalidates the checksum
	//
	// the only reason we make this conditional is because in a bundle we may have
	// already done the work for a policy that is a dependency of another
	// in that case we don't want to recalculate the graph and use it instead
	// Note 1: It relies on the fact that the compile step clears out the checksums
	// to make sure users don't override them
	// Note 2: We don't need to cmpute the checksum since the GraphChecksum depends
	// on it and will force it in case it is missing (no graph checksum => no checksum)

	// NOTE: its important to update the checksum AFTER the queries have been changed,
	// otherwise we generate the old GraphChecksum
	if policyObj.GraphExecutionChecksum == "" || policyObj.GraphContentChecksum == "" {
		logCtx.Trace().Str("policy", policyObj.Mrn).Msg("marketplace> update graphchecksum")
		policyObj.UpdateChecksums(ctx,
			s.DataLake.GetValidatedPolicy,
			s.DataLake.GetQuery,
			bundle)
	}

	filters, err := policyObj.ComputeAssetFilters(
		ctx,
		s.DataLake.GetRawPolicy,
		false,
	)
	if err != nil {
		return nil, nil, err
	}

	return policyObj, filters, nil
}

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

func (s *LocalServices) setQuery(ctx context.Context, mrn string, query *Mquery, isScored bool) error {
	if query == nil {
		return errors.New("cannot set query '" + mrn + "' as it is not defined")
	}

	if query.Title == "" {
		query.Title = query.Query
	}

	return s.DataLake.SetQuery(ctx, mrn, query, isScored)
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

// GetPolicyFilters retrieves the asset filter queries for a given policy
func (s *LocalServices) GetPolicyFilters(ctx context.Context, mrn *Mrn) (*Mqueries, error) {
	if mrn == nil || len(mrn.Mrn) == 0 {
		return nil, status.Error(codes.InvalidArgument, "policy mrn is required")
	}

	filters, err := s.DataLake.GetPolicyFilters(ctx, mrn.Mrn)
	if err != nil {
		return nil, errors.New("failed to get policy filters: " + err.Error())
	}

	return &Mqueries{Items: filters}, nil
}

// List all policies for a given owner
func (s *LocalServices) List(ctx context.Context, filter *PolicySearchFilter) (*Policies, error) {
	if filter == nil {
		return nil, status.Error(codes.InvalidArgument, "need to provide a filter object for list")
	}

	if len(filter.OwnerMrn) == 0 {
		return nil, status.Error(codes.InvalidArgument, "a MRN for the policy owner is required")
	}

	res, err := s.DataLake.ListPolicies(ctx, filter.OwnerMrn, filter.Name)
	if err != nil {
		return nil, err
	}
	if res == nil {
		res = []*Policy{}
	}

	return &Policies{
		Items: res,
	}, nil
}

// DeletePolicy removes a policy via its given MRN
func (s *LocalServices) DeletePolicy(ctx context.Context, in *Mrn) (*Empty, error) {
	if in == nil || len(in.Mrn) == 0 {
		return nil, status.Error(codes.InvalidArgument, "policy MRN is required")
	}

	return globalEmpty, s.DataLake.DeletePolicy(ctx, in.Mrn)
}

// HELPER METHODS
// =================

// ComputeBundle creates a policy bundle (with queries and dependencies) for a given policy
func (s *LocalServices) ComputeBundle(ctx context.Context, mpolicyObj *Policy) (*PolicyBundle, error) {
	bundleMap := PolicyBundleMap{
		OwnerMrn: mpolicyObj.OwnerMrn,
		Policies: map[string]*Policy{},
		Queries:  map[string]*Mquery{},
		Props:    map[string]*Mquery{},
	}

	// we need to re-compute the asset filters
	mpolicyObj.AssetFilters = map[string]*Mquery{}
	bundleMap.Policies[mpolicyObj.Mrn] = mpolicyObj

	for mrn, v := range mpolicyObj.Props {
		if v != "" {
			return nil, errors.New("cannot support properties which overwrite other properties")
		}

		query, err := s.DataLake.GetQuery(ctx, mrn)
		if err != nil {
			return nil, err
		}
		bundleMap.Props[mrn] = query
	}

	for i := range mpolicyObj.Specs {
		spec := mpolicyObj.Specs[i]

		if spec.AssetFilter != nil {
			filter := spec.AssetFilter
			mpolicyObj.AssetFilters[filter.CodeId] = filter
		}

		for mrn := range spec.DataQueries {
			query, err := s.DataLake.GetQuery(ctx, mrn)
			if err != nil {
				return nil, err
			}
			bundleMap.Queries[mrn] = query
		}

		for mrn := range spec.ScoringQueries {
			query, err := s.DataLake.GetQuery(ctx, mrn)
			if err != nil {
				return nil, err
			}
			bundleMap.Queries[mrn] = query
		}

		for mrn := range spec.Policies {
			nuBundle, err := s.DataLake.GetValidatedBundle(ctx, mrn)
			if err != nil {
				return nil, err
			}

			for i := range nuBundle.Policies {
				policy := nuBundle.Policies[i]
				bundleMap.Policies[policy.Mrn] = policy
			}
			for i := range nuBundle.Queries {
				query := nuBundle.Queries[i]
				bundleMap.Queries[query.Mrn] = query
			}
			for i := range nuBundle.Props {
				query := nuBundle.Props[i]
				bundleMap.Props[query.Mrn] = query
			}

			nuPolicy := bundleMap.Policies[mrn]
			if nuPolicy == nil {
				return nil, errors.New("pulled policy bundle for " + mrn + " but couldn't find the policy in the bundle")
			}
			for k, v := range nuPolicy.AssetFilters {
				mpolicyObj.AssetFilters[k] = v
			}
		}
	}

	// phew, done collecting. let's save and return

	list := bundleMap.ToList().Clean()
	return list, nil
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
