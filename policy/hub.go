// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"context"
	"os"
	"path"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v9/explorer"
	"go.mondoo.com/cnquery/v9/logger"
	"go.mondoo.com/cnquery/v9/mrn"
	"go.mondoo.com/ranger-rpc"
	"go.mondoo.com/ranger-rpc/codes"
	"go.mondoo.com/ranger-rpc/status"
	"go.opentelemetry.io/otel"
)

const (
	defaultRegistryUrl    = "https://registry.api.mondoo.com"
	RegistryServiceName   = "registry.mondoo.com"
	CollectionIDNamespace = "namespace"
	CollectionIDPolicies  = "policies"
)

var tracer = otel.Tracer("go.mondoo.com/cnspec/v9/policy")

func NewPolicyMrn(namespace string, uid string) string {
	m := &mrn.MRN{
		ServiceName:          RegistryServiceName,
		RelativeResourceName: path.Join(CollectionIDNamespace, namespace, CollectionIDPolicies, uid),
	}
	return m.String()
}

// ValidateBundle and check queries, relationships, MRNs, and versions
func (s *LocalServices) ValidateBundle(ctx context.Context, bundle *Bundle) (*Empty, error) {
	_, err := bundle.Compile(ctx, s.Runtime.Schema(), s.DataLake)
	return globalEmpty, err
}

// SetBundle stores a bundle of policies and queries in this marketplace
func (s *LocalServices) SetBundle(ctx context.Context, bundle *Bundle) (*Empty, error) {
	// See https://gitlab.com/mondoolabs/mondoo/-/issues/595

	bundleMap, err := bundle.Compile(ctx, s.Runtime.Schema(), s.DataLake)
	if err != nil {
		return globalEmpty, err
	}

	if err := s.SetBundleMap(ctx, bundleMap); err != nil {
		return nil, err
	}

	return globalEmpty, nil
}

// PreparePolicy takes a policy and an optional bundle and gets it
// ready to be saved in the DB, including asset filters.
//
// Note1: The bundle must have been pre-compiled and validated!
// Note2: The bundle may be nil, in which case we will try to find what is needed for the policy
func (s *LocalServices) PreparePolicy(ctx context.Context, policyObj *Policy, bundle *PolicyBundleMap) (*Policy, []*explorer.Mquery, error) {
	logCtx := logger.FromContext(ctx)
	var err error

	if policyObj == nil || len(policyObj.Mrn) == 0 {
		return nil, nil, status.Error(codes.InvalidArgument, "policy mrn is required")
	}

	var queriesLookup map[string]*explorer.Mquery
	if bundle != nil {
		queriesLookup = bundle.Queries
	}

	// TODO: we need to decide if it is up to the caller to ensure that the checksum is up-to-date
	// e.g. ApplyScoringMutation changes the group. Right now we assume the caller invalidates the checksum
	//
	// the only reason we make this conditional is because in a bundle we may have
	// already done the work for a policy that is a dependency of another
	// in that case we don't want to recalculate the graph and use it instead
	// Note 1: It relies on the fact that the compile step clears out the checksums
	// to make sure users don't override them
	// Note 2: We don't need to compute the checksum since the GraphChecksum depends
	// on it and will force it in case it is missing (no graph checksum => no checksum)

	// NOTE: its important to update the checksum AFTER the queries have been changed,
	// otherwise we generate the old GraphChecksum
	if policyObj.GraphExecutionChecksum == "" || policyObj.GraphContentChecksum == "" {
		logCtx.Trace().Str("policy", policyObj.Mrn).Msg("marketplace> update graphchecksum")
		err = policyObj.UpdateChecksums(
			ctx,
			s.DataLake.GetValidatedPolicy,
			s.DataLake.GetQuery,
			bundle,
			s.Runtime.Schema(),
		)
		if err != nil {
			return nil, nil, err
		}
	}

	filters, err := policyObj.ComputeAssetFilters(
		ctx,
		s.DataLake.GetRawPolicy,
		func(ctx context.Context, mrn string) (*explorer.Mquery, error) {
			if q, ok := queriesLookup[mrn]; ok {
				return q, nil
			}
			return s.DataLake.GetQuery(ctx, mrn)
		},
		false,
	)
	if err != nil {
		return nil, nil, err
	}

	return policyObj, filters, nil
}

// PrepareFramework takes a framework and an optional bundle and gets it
// ready to be saved in the DB.
//
// Note1: The bundle must have been pre-compiled and validated!
// Note2: The bundle may be nil, in which case we will try to find what is needed
func (s *LocalServices) PrepareFramework(ctx context.Context, frameworkObj *Framework, bundle *PolicyBundleMap) (*Framework, error) {
	logCtx := logger.FromContext(ctx)
	var err error

	if frameworkObj == nil || len(frameworkObj.Mrn) == 0 {
		return nil, status.Error(codes.InvalidArgument, "framework mrn is required")
	}

	// TODO: we need to decide if it is up to the caller to ensure that the checksum is up-to-date
	// e.g. ApplyScoringMutation changes the group. Right now we assume the caller invalidates the checksum
	//
	// the only reason we make this conditional is because in a bundle we may have
	// already done the work for a policy that is a dependency of another
	// in that case we don't want to recalculate the graph and use it instead
	// Note 1: It relies on the fact that the compile step clears out the checksums
	// to make sure users don't override them
	// Note 2: We don't need to compute the checksum since the GraphChecksum depends
	// on it and will force it in case it is missing (no graph checksum => no checksum)

	// NOTE: its important to update the checksum AFTER the queries have been changed,
	// otherwise we generate the old GraphChecksum
	if frameworkObj.GraphExecutionChecksum == "" || frameworkObj.GraphContentChecksum == "" {
		logCtx.Trace().Str("framework", frameworkObj.Mrn).Msg("update graphchecksum")
		err = frameworkObj.UpdateChecksums(ctx,
			s.DataLake.GetFramework,
			s.DataLake.GetFrameworkMaps,
			bundle)
		if err != nil {
			return nil, err
		}
	}

	return frameworkObj, nil
}

// SetPolicyFromBundle takes a policy and stores it in the datalake. The
// bundle is used as an optional local reference.
func (s *LocalServices) SetPolicyFromBundle(ctx context.Context, policyObj *Policy, bundleMap *PolicyBundleMap) error {
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
	_, err = s.DataLake.GetValidatedBundle(ctx, &BundleReq{Mrn: policyObj.Mrn})
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

// SetBundleMap takes a bundle map (converted from a policy bundle) and
// creates all queries and policies in it.
func (s *LocalServices) SetBundleMap(ctx context.Context, bundleMap *PolicyBundleMap) error {
	logCtx := logger.FromContext(ctx)

	for mrn, query := range bundleMap.Queries {
		if err := s.setQuery(ctx, mrn, query); err != nil {
			return err
		}
	}

	// sort policies, so that we store child policies before their parents
	policies, err := bundleMap.PoliciesSortedByDependency()
	if err != nil {
		return err
	}

	for i := range policies {
		policyObj := policies[i]
		logCtx.Debug().Str("owner", policyObj.OwnerMrn).Str("uid", policyObj.Uid).Str("mrn", policyObj.Mrn).Msg("store policy")
		policyObj.OwnerMrn = bundleMap.OwnerMrn

		if err = s.SetPolicyFromBundle(ctx, policyObj, bundleMap); err != nil {
			return err
		}
	}

	for i := range bundleMap.Frameworks {
		framework := bundleMap.Frameworks[i]
		framework.OwnerMrn = bundleMap.OwnerMrn

		framework, err = s.PrepareFramework(ctx, framework, bundleMap)
		if err != nil {
			return err
		}

		if err := s.DataLake.SetFramework(ctx, framework); err != nil {
			return err
		}
		if err := s.DataLake.SetFrameworkMaps(ctx, framework.Mrn, framework.FrameworkMaps); err != nil {
			return err
		}
	}

	return nil
}

func (s *LocalServices) setQuery(ctx context.Context, mrn string, query *explorer.Mquery) error {
	if query == nil {
		return errors.New("cannot set query '" + mrn + "' as it is not defined")
	}

	if query.Title == "" {
		query.Title = query.Mql
	}

	return s.DataLake.SetQuery(ctx, mrn, query)
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
	_, err = s.cacheUpstreamPolicyOld(ctx, in.Mrn)
	if err != nil {
		return nil, err
	}
	return s.DataLake.GetValidatedPolicy(ctx, in.Mrn)
}

// GetBundle retrieves the given policy and all its dependencies (policies/queries)
func (s *LocalServices) GetBundle(ctx context.Context, in *Mrn) (*Bundle, error) {
	// TODO: once the backend implements GetBundleV2, we can remove this implementation
	// and directly call GetBundleV2
	if in == nil || len(in.Mrn) == 0 {
		return nil, status.Error(codes.InvalidArgument, "policy mrn is required")
	}

	b, err := s.DataLake.GetValidatedBundle(ctx, &BundleReq{Mrn: in.Mrn})
	if err == nil {
		return b, nil
	}
	if s.Upstream == nil {
		return nil, err
	}

	// try upstream
	return s.cacheUpstreamPolicyOld(ctx, in.Mrn)
}

func (s *LocalServices) GetBundleV2(ctx context.Context, in *BundleReq) (*Bundle, error) {
	if in == nil || len(in.Mrn) == 0 {
		return nil, status.Error(codes.InvalidArgument, "policy mrn is required")
	}

	b, err := s.DataLake.GetValidatedBundle(ctx, in)
	if err == nil {
		return b, nil
	}
	if s.Upstream == nil {
		return nil, err
	}

	// try upstream
	return s.cacheUpstreamPolicy(ctx, in)
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
func (s *LocalServices) List(ctx context.Context, filter *ListReq) (*Policies, error) {
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

// DefaultPolicies retrieves a list of default policies for a given asset
func (s *LocalServices) DefaultPolicies(ctx context.Context, req *DefaultPoliciesReq) (*URLs, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "no filters provided")
	}

	if s.Upstream != nil {
		return s.Upstream.DefaultPolicies(ctx, req)
	}

	registryEndpoint := os.Getenv("REGISTRY_URL")
	if registryEndpoint == "" {
		registryEndpoint = defaultRegistryUrl
	}

	// Note, this does not use the proxy config override from the mondoo.yml since we only get here when
	// it is used without upstream config
	client, err := NewPolicyHubClient(registryEndpoint, ranger.DefaultHttpClient())
	if err != nil {
		return nil, err
	}
	return client.DefaultPolicies(ctx, req)
}

func (s *LocalServices) GetFramework(ctx context.Context, req *Mrn) (*Framework, error) {
	panic("NOT YET IMPLEMENTED")
}

func (s *LocalServices) DeleteFramework(ctx context.Context, req *Mrn) (*Empty, error) {
	panic("NOT YET IMPLEMENTED")
	return globalEmpty, nil
}

func (s *LocalServices) ListFrameworks(ctx context.Context, req *ListReq) (*Frameworks, error) {
	panic("NOT YET IMPLEMENTED")
}

// HELPER METHODS
// =================

// ComputeBundle creates a policy bundle (with queries and dependencies) for a given policy
func (s *LocalServices) ComputeBundle(ctx context.Context, mpolicyObj *Policy, mframeworkObj *Framework) (*Bundle, error) {
	bundleMap := PolicyBundleMap{
		OwnerMrn:   mpolicyObj.OwnerMrn,
		Policies:   map[string]*Policy{},
		Frameworks: map[string]*Framework{},
		Queries:    map[string]*explorer.Mquery{},
		Props:      map[string]*explorer.Property{},
	}

	if err := s.computePolicyBundle(ctx, &bundleMap, mpolicyObj); err != nil {
		return nil, err
	}
	if err := s.computeFrameworkBundle(ctx, &bundleMap, mframeworkObj); err != nil {
		return nil, err
	}

	list := bundleMap.ToList().Clean()
	return list, nil
}

func (s *LocalServices) computeFrameworkBundle(ctx context.Context, bundleMap *PolicyBundleMap, frameworkObj *Framework) error {
	if frameworkObj == nil {
		return nil
	}

	maps, err := s.DataLake.GetFrameworkMaps(ctx, frameworkObj.Mrn)
	if err != nil {
		return errors.New("failed to get framework maps for: " + frameworkObj.Mrn)
	}
	frameworkObj.FrameworkMaps = maps

	bundleMap.Frameworks[frameworkObj.Mrn] = frameworkObj

	for i := range frameworkObj.Dependencies {
		dep := frameworkObj.Dependencies[i]

		depObj, err := s.DataLake.GetFramework(ctx, dep.Mrn)
		if err != nil {
			return err
		}
		bundleMap.Frameworks[dep.Mrn] = depObj

		if err := s.computeFrameworkBundle(ctx, bundleMap, depObj); err != nil {
			return err
		}
	}

	return nil
}

func (s *LocalServices) computePolicyBundle(ctx context.Context, bundleMap *PolicyBundleMap, mpolicyObj *Policy) error {
	if mpolicyObj == nil {
		return nil
	}

	bundleMap.Policies[mpolicyObj.Mrn] = mpolicyObj

	// we need to re-compute the asset filters
	localFilters, err := gatherLocalAssetFilters(ctx, mpolicyObj.Groups, s.DataLake.GetQuery)
	if err != nil {
		return err
	}

	mpolicyObj.ComputedFilters = localFilters

	for i := range mpolicyObj.Props {
		prop := mpolicyObj.Props[i]
		bundleMap.Props[prop.Mrn] = prop
	}

	for i := range mpolicyObj.Groups {
		group := mpolicyObj.Groups[i]

		// For all queries and checks we are looking to get the shared objects only.
		// This is because the embedded queries and checks are already part of the
		// policy and what the bundle represents in its toplevel Queries field is
		// the collection of shared content (not its overrides). So the section
		// below is all about adding the shared content only.

		for i := range group.Queries {
			query := group.Queries[i]
			if base, _ := s.DataLake.GetQuery(ctx, query.Mrn); base != nil {
				query = base
			}
			bundleMap.Queries[query.Mrn] = query

			for j := range query.Variants {
				if v, _ := s.DataLake.GetQuery(ctx, query.Variants[j].Mrn); v != nil {
					bundleMap.Queries[v.Mrn] = v
				}
			}
		}

		for i := range group.Checks {
			check := group.Checks[i]
			if base, _ := s.DataLake.GetQuery(ctx, check.Mrn); base != nil {
				check = base
			}
			bundleMap.Queries[check.Mrn] = check

			for j := range check.Variants {
				if v, _ := s.DataLake.GetQuery(ctx, check.Variants[j].Mrn); v != nil {
					bundleMap.Queries[v.Mrn] = v
				}
			}
		}

		for i := range group.Policies {
			policy := group.Policies[i]

			nuBundle, err := s.DataLake.GetValidatedBundle(ctx, &BundleReq{Mrn: policy.Mrn})
			if err != nil {
				return err
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
				prop := nuBundle.Props[i]
				bundleMap.Props[prop.Mrn] = prop
			}

			nuPolicy := bundleMap.Policies[policy.Mrn]
			if nuPolicy == nil {
				return errors.New("pulled policy bundle for " + policy.Mrn + " but couldn't find the policy in the bundle")
			}

			if nuPolicy.ComputedFilters == nil {
				// This should not happen, because `GetValidatedBundle` above should
				// only retrieve a fully validated bundle from the backend, i.e.
				// that bundle has all its computed asset filters included.
				// However, we do this to avoid breaking the execution, while still
				// logging the error.
				log.Error().Str("new-policy-mrn", policy.Mrn).Str("caller", mpolicyObj.Mrn).Msg("received a policy with nil ComputedFilters; trying to refresh it")
				filters, err := nuPolicy.ComputeAssetFilters(ctx, s.DataLake.GetValidatedPolicy, s.DataLake.GetQuery, true)
				if err != nil {
					return err
				}

				nuPolicy.ComputedFilters = &explorer.Filters{
					Items: make(map[string]*explorer.Mquery, len(filters)),
				}
				for _, f := range filters {
					nuPolicy.ComputedFilters.Items[f.CodeId] = f
				}
			}

			for k, v := range nuPolicy.ComputedFilters.Items {
				mpolicyObj.ComputedFilters.Items[k] = v
			}
		}
	}

	return nil
}

// cacheUpstreamPolicy by storing a copy of the upstream policy bundle in this db
// Note: upstream marketplace has to be defined
func (s *LocalServices) cacheUpstreamPolicy(ctx context.Context, in *BundleReq) (*Bundle, error) {
	logCtx := logger.FromContext(ctx)
	mrn := in.Mrn
	if s.Upstream == nil {
		return nil, errors.New("failed to retrieve upstream policy " + mrn + " since upstream is not defined")
	}

	logCtx.Debug().Str("policy", mrn).Msg("marketplace> fetch policy bundle from upstream")
	bundle, err := s.Upstream.GetBundleV2(ctx, in)
	if err != nil {
		logCtx.Error().Err(err).Str("policy", mrn).Msg("marketplace> failed to retrieve policy bundle from upstream")
		return nil, errors.New("failed to retrieve upstream policy " + mrn + ": " + err.Error())
	}

	bundleMap := bundle.ToMap()

	err = s.SetBundleMap(ctx, bundleMap)
	if err != nil {
		logCtx.Error().Err(err).Str("policy", mrn).Msg("marketplace> failed to set policy bundle retrieved from upstream")
		return nil, errors.New("failed to cache upstream policy " + mrn + ": " + err.Error())
	}

	logCtx.Debug().Str("policy", mrn).Msg("marketplace> fetched policy bundle from upstream")
	return bundle, nil
}

// TODO: this can be removed once the backend implements GetBundleV2
func (s *LocalServices) cacheUpstreamPolicyOld(ctx context.Context, mrn string) (*Bundle, error) {
	logCtx := logger.FromContext(ctx)
	if s.Upstream == nil {
		return nil, errors.New("failed to retrieve upstream policy " + mrn + " since upstream is not defined")
	}

	logCtx.Debug().Str("policy", mrn).Msg("marketplace> fetch policy bundle from upstream")
	var bundle *Bundle
	var err error

	bundle, err = s.Upstream.GetBundle(ctx, &Mrn{Mrn: mrn})
	if err != nil {
		logCtx.Error().Err(err).Str("policy", mrn).Msg("marketplace> failed to retrieve policy bundle from upstream")
		return nil, errors.New("failed to retrieve upstream policy " + mrn + ": " + err.Error())
	}

	bundleMap := bundle.ToMap()

	err = s.SetBundleMap(ctx, bundleMap)
	if err != nil {
		logCtx.Error().Err(err).Str("policy", mrn).Msg("marketplace> failed to set policy bundle retrieved from upstream")
		return nil, errors.New("failed to cache upstream policy " + mrn + ": " + err.Error())
	}

	logCtx.Debug().Str("policy", mrn).Msg("marketplace> fetched policy bundle from upstream")
	return bundle, nil
}
