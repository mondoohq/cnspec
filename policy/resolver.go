package policy

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"math/rand"
	"sort"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/fasthash/fnv1a"
	"go.mondoo.com/cnquery/checksums"
	"go.mondoo.com/cnquery/explorer"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/logger"
	"go.mondoo.com/cnquery/mrn"
	"go.mondoo.com/cnquery/sortx"
	"go.mondoo.com/ranger-rpc/codes"
	"go.mondoo.com/ranger-rpc/status"
)

const (
	POLICY_SERVICE_NAME = "policy.api.mondoo.com"
)

type AssetMutation struct {
	AssetMrn         string
	PolicyActions    map[string]explorer.Action
	FrameworkActions map[string]explorer.Action
}

// Assign a policy to an asset
//
// We need to handle multiple cases:
// 1. all local, policies and assets are available locally
// 2. asset is local (via incognito mode) but policy is upstream
// 3. asset and policy are upstream
func (s *LocalServices) Assign(ctx context.Context, assignment *PolicyAssignment) (*Empty, error) {
	if len(assignment.PolicyMrns)+len(assignment.FrameworkMrns) == 0 {
		return nil, status.Error(codes.InvalidArgument, "a policy or framework mrn is required")
	}

	// all remote, call upstream
	if s.Upstream != nil && !s.Incognito {
		return s.Upstream.PolicyResolver.Assign(ctx, assignment)
	}

	// policies may be stored in upstream, cache them first
	if s.Upstream != nil && s.Incognito {
		// NOTE: by calling GetPolicy it is automatically cached
		for i := range assignment.PolicyMrns {
			mrn := assignment.PolicyMrns[i]
			_, err := s.GetPolicy(ctx, &Mrn{
				Mrn: mrn,
			})
			if err != nil {
				return nil, err
			}
		}
	}

	if err := s.DataLake.EnsureAsset(ctx, assignment.AssetMrn); err != nil {
		return nil, err
	}

	policyActions := map[string]explorer.Action{}
	for i := range assignment.PolicyMrns {
		policyMrn := assignment.PolicyMrns[i]
		policyActions[policyMrn] = assignment.Action
	}

	frameworkActions := map[string]explorer.Action{}
	for i := range assignment.FrameworkMrns {
		frameworkMrn := assignment.FrameworkMrns[i]
		frameworkActions[frameworkMrn] = assignment.Action
	}

	err := s.DataLake.MutateAssignments(ctx, &AssetMutation{
		AssetMrn:         assignment.AssetMrn,
		PolicyActions:    policyActions,
		FrameworkActions: frameworkActions,
	}, true)
	return globalEmpty, err
}

// Unassign a policy to an asset
func (s *LocalServices) Unassign(ctx context.Context, assignment *PolicyAssignment) (*Empty, error) {
	if len(assignment.PolicyMrns) == 0 {
		return nil, status.Error(codes.InvalidArgument, "a policy mrn is required")
	}

	// all remote, call upstream
	if s.Upstream != nil && !s.Incognito {
		return s.Upstream.PolicyResolver.Unassign(ctx, assignment)
	}

	policyActions := map[string]explorer.Action{}
	for i := range assignment.PolicyMrns {
		policyMrn := assignment.PolicyMrns[i]
		policyActions[policyMrn] = explorer.Action_DEACTIVATE
	}

	frameworkActions := map[string]explorer.Action{}
	for i := range assignment.FrameworkMrns {
		frameworkMrn := assignment.FrameworkMrns[i]
		frameworkActions[frameworkMrn] = explorer.Action_DEACTIVATE
	}

	err := s.DataLake.MutateAssignments(ctx, &AssetMutation{
		AssetMrn:         assignment.AssetMrn,
		PolicyActions:    policyActions,
		FrameworkActions: frameworkActions,
	}, true)
	return globalEmpty, err
}

func (s *LocalServices) SetProps(ctx context.Context, req *explorer.PropsReq) (*explorer.Empty, error) {
	// validate that the queries compile and fill in checksums
	for i := range req.Props {
		prop := req.Props[i]
		code, err := prop.RefreshChecksumAndType()
		if err != nil {
			return nil, err
		}
		prop.CodeId = code.CodeV2.Id
	}

	return &explorer.Empty{}, s.DataLake.SetProps(ctx, req)
}

// Resolve a given policy for a set of asset filters
func (s *LocalServices) Resolve(ctx context.Context, req *ResolveReq) (*ResolvedPolicy, error) {
	if s.Upstream != nil && !s.Incognito {
		return s.Upstream.Resolve(ctx, req)
	}

	return s.resolve(ctx, req.PolicyMrn, req.AssetFilters)
}

// ResolveAndUpdateJobs will resolve an asset's policy and update its jobs
func (s *LocalServices) ResolveAndUpdateJobs(ctx context.Context, req *UpdateAssetJobsReq) (*ResolvedPolicy, error) {
	if s.Upstream == nil || s.Incognito {
		res, err := s.resolve(ctx, req.AssetMrn, req.AssetFilters)
		if err != nil {
			return nil, err
		}

		if res.CollectorJob != nil {
			err := res.CollectorJob.Validate()
			if err != nil {
				logger.FromContext(ctx).Error().
					Err(err).
					Msg("resolver> resolved policy is invalid")
			}
		}

		err = s.DataLake.SetAssetResolvedPolicy(ctx, req.AssetMrn, res, V2Code)
		if err != nil {
			return nil, err
		}

		return res, nil
	}

	res, err := s.Upstream.PolicyResolver.ResolveAndUpdateJobs(ctx, req)
	if err != nil {
		return nil, err
	}

	err = s.cacheUpstreamJobs(ctx, req.AssetMrn, res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// UpdateAssetJobs by recalculating them
func (s *LocalServices) UpdateAssetJobs(ctx context.Context, req *UpdateAssetJobsReq) (*Empty, error) {
	if s.Upstream == nil || s.Incognito {
		return globalEmpty, s.updateAssetJobs(ctx, req.AssetMrn, req.AssetFilters)
	}

	if _, err := s.Upstream.PolicyResolver.UpdateAssetJobs(ctx, req); err != nil {
		return nil, err
	}

	resolvedPolicy, err := s.Upstream.PolicyResolver.Resolve(ctx, &ResolveReq{
		PolicyMrn:    req.AssetMrn,
		AssetFilters: req.AssetFilters,
	})
	if err != nil {
		return nil, errors.New("resolver> failed to resolve upstream jobs for caching: " + err.Error())
	}

	return globalEmpty, s.cacheUpstreamJobs(ctx, req.AssetMrn, resolvedPolicy)
}

// GetResolvedPolicy for a given asset
func (s *LocalServices) GetResolvedPolicy(ctx context.Context, mrn *Mrn) (*ResolvedPolicy, error) {
	if s.Upstream != nil && !s.Incognito {
		return s.Upstream.GetResolvedPolicy(ctx, mrn)
	}

	res, err := s.DataLake.GetResolvedPolicy(ctx, mrn.Mrn)
	return res, err
}

// StoreResults saves the given scores and date for an asset
func (s *LocalServices) StoreResults(ctx context.Context, req *StoreResultsReq) (*Empty, error) {
	logger.AddTag(ctx, "asset", req.AssetMrn)

	_, err := s.DataLake.UpdateScores(ctx, req.AssetMrn, req.Scores)
	if err != nil {
		return globalEmpty, err
	}

	_, err = s.DataLake.UpdateData(ctx, req.AssetMrn, req.Data)
	if err != nil {
		return globalEmpty, err
	}

	if s.Upstream != nil && !s.Incognito {
		_, err := s.Upstream.PolicyResolver.StoreResults(ctx, req)
		if err != nil {
			return globalEmpty, err
		}
	}

	return globalEmpty, nil
}

// GetReport retrieves a report for a given asset and policy
func (s *LocalServices) GetReport(ctx context.Context, req *EntityScoreReq) (*Report, error) {
	return s.DataLake.GetReport(ctx, req.EntityMrn, req.ScoreMrn)
}

// GetFrameworkReport retrieves a report for a given asset and framework
func (s *LocalServices) GetFrameworkReport(ctx context.Context, req *EntityScoreReq) (*FrameworkReport, error) {
	panic("NOT YET IMPLEMENTED")
}

// GetScore retrieves one score for an asset
func (s *LocalServices) GetScore(ctx context.Context, req *EntityScoreReq) (*Report, error) {
	score, err := s.DataLake.GetScore(ctx, req.EntityMrn, req.ScoreMrn)
	if err != nil {
		return nil, err
	}

	return &Report{
		EntityMrn:  req.EntityMrn,
		ScoringMrn: req.ScoreMrn,
		Score:      &score,
	}, nil
}

// SynchronizeAssets is not require for local services
func (s *LocalServices) SynchronizeAssets(ctx context.Context, req *SynchronizeAssetsReq) (*SynchronizeAssetsResp, error) {
	return nil, nil
}

// DeleteAssets is not require for local services
func (s *LocalServices) PurgeAssets(context.Context, *PurgeAssetsRequest) (*PurgeAssetsConfirmation, error) {
	return nil, nil
}

// HELPER METHODS
// =================

// CreatePolicyObject creates a policy object without saving it and returns it
func (s *LocalServices) CreatePolicyObject(policyMrn string, ownerMrn string) *Policy {
	// TODO: this should be handled better and I'm not sure yet how...
	// we need to ensure a good owner MRN exists for all objects, including orgs and spaces
	// this is the case when we are in incognito mode
	if ownerMrn == "" {
		log.Debug().Str("policyMrn", policyMrn).Msg("resolver> ownerMrn is missing")
		ownerMrn = "//policy.api.mondoo.app"
	}

	name, _ := mrn.GetResource(policyMrn, MRN_RESOURCE_ASSET)
	if name == "" {
		name = policyMrn
	}

	return &Policy{
		Mrn:     policyMrn,
		Name:    name, // placeholder
		Version: "",   // no version, semver otherwise
		Groups: []*PolicyGroup{{
			Policies: []*PolicyRef{},
			Checks:   []*explorer.Mquery{},
			Queries:  []*explorer.Mquery{},
		}},
		ComputedFilters: &explorer.Filters{},
		OwnerMrn:        ownerMrn,
	}
}

// CreateFrameworkObject creates a framework object without saving it and returns it
func (s *LocalServices) CreateFrameworkObject(frameworkMrn string, ownerMrn string) *Framework {
	// TODO: this should be handled better, similar to CreatePolicyObject.
	// we need to ensure a good owner MRN exists for all objects, including orgs and spaces
	// this is the case when we are in incognito mode
	if ownerMrn == "" {
		log.Debug().Str("frameworkMrn", frameworkMrn).Msg("resolver> ownerMrn is missing")
		ownerMrn = "//policy.api.mondoo.app"
	}

	name, _ := mrn.GetResource(frameworkMrn, MRN_RESOURCE_ASSET)
	if name == "" {
		name = frameworkMrn
	}

	return &Framework{
		Mrn:     frameworkMrn,
		Name:    name, // placeholder
		Version: "",   // no version, semver otherwise
		// no Groups; this call usually creates frameworks for assets, where we
		// don't need groups since dependencies are handled in the Dependencies field
	}
}

// POLICY RESOLUTION
// =====================

const (
	maxResolveRetry              = 3
	maxResolveRetryBackoff       = 25 * time.Millisecond
	maxResolveRetryBackoffjitter = 25 * time.Millisecond
)

var ErrRetryResolution = errors.New("retry policy resolution")

type policyResolutionError struct {
	ID       string
	IsPolicy bool
	Error    string
}

type resolverCache struct {
	baseChecksum         string
	assetFiltersChecksum string
	assetFilters         map[string]struct{}

	// assigned queries, listed by their UUID (i.e. policy context)
	executionQueries map[string]*ExecutionQuery
	dataQueries      map[string]struct{}
	queriesByMsum    map[string]*explorer.Mquery // Msum == Mquery.Checksum
	propsCache       explorer.PropsCache

	reportingJobsByUUID map[string]*ReportingJob
	reportingJobsByMsum map[string][]*ReportingJob // Msum == Mquery.Checksum, i.e. only reporting jobs for mqueries
	reportingJobsActive map[string]bool
	errors              []*policyResolutionError
	bundleMap           *PolicyBundleMap
}

type policyResolverCache struct {
	removedPolicies map[string]struct{}        // tracks policies that will not be added
	removedQueries  map[string]struct{}        // tracks queries that will not be added
	parentPolicies  map[string]struct{}        // tracks policies in the ancestry, to prevent loops
	childJobsByMrn  map[string][]*ReportingJob // tracks policies+queries+checks that were added below (at any level)
	global          *resolverCache
}

func checksum2string(checksum uint64) string {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, checksum)
	return base64.StdEncoding.EncodeToString(b)
}

func checksumStrings(strings ...string) string {
	checksum := fnv1a.Init64
	for i := range strings {
		checksum = fnv1a.AddString64(checksum, strings[i])
	}
	return checksum2string(checksum)
}

func (r *resolverCache) relativeChecksum(s string) string {
	return checksumStrings(r.baseChecksum, r.assetFiltersChecksum, "v2", s)
}

func (p *policyResolverCache) clone() *policyResolverCache {
	res := &policyResolverCache{
		removedPolicies: map[string]struct{}{},
		removedQueries:  map[string]struct{}{},
		parentPolicies:  map[string]struct{}{},
		childJobsByMrn:  map[string][]*ReportingJob{},
		global:          p.global,
	}

	for k, v := range p.removedPolicies {
		res.removedPolicies[k] = v
	}
	for k, v := range p.removedQueries {
		res.removedQueries[k] = v
	}
	for k, v := range p.parentPolicies {
		res.parentPolicies[k] = v
	}

	return res
}

func (p *policyResolverCache) addChildren(other *policyResolverCache) {
	// we copy these back into the parent, but don't keep them around in the global
	// cache. The reason for that is that policy siblings could accidentally access
	// each others jobs when they shouldn't be able to.
	// In this sense, reporting jobs by MRN only bubble up, never down or sideways.
	for k, v := range other.childJobsByMrn {
		p.childJobsByMrn[k] = append(p.childJobsByMrn[k], v...)
	}
}

func (s *LocalServices) resolve(ctx context.Context, policyMrn string, assetFilters []*explorer.Mquery) (*ResolvedPolicy, error) {
	logCtx := logger.FromContext(ctx)
	for i := 0; i < maxResolveRetry; i++ {
		resolvedPolicy, err := s.tryResolve(ctx, policyMrn, assetFilters)
		if err != nil {
			if !errors.Is(err, ErrRetryResolution) {
				return nil, err
			}
			if i+1 < maxResolveRetry {
				jitter := time.Duration(rand.Int63n(int64(maxResolveRetryBackoffjitter)))
				sleepTime := maxResolveRetryBackoff + jitter
				logCtx.Error().Int("try", i+1).Dur("sleepTime", sleepTime).Msg("retrying policy resolution")
				time.Sleep(sleepTime)
			}
		} else {
			return resolvedPolicy, nil
		}
	}
	return nil, errors.New("concurrent policy resolve")
}

func (s *LocalServices) tryResolve(ctx context.Context, bundleMrn string, assetFilters []*explorer.Mquery) (*ResolvedPolicy, error) {
	logCtx := logger.FromContext(ctx)
	now := time.Now()

	// phase 1: resolve asset filters and see if we can find a cached policy
	// trying first with all asset filters
	allFiltersChecksum, err := ChecksumAssetFilters(assetFilters)
	if err != nil {
		return nil, err
	}

	var rp *ResolvedPolicy
	rp, err = s.DataLake.CachedResolvedPolicy(ctx, bundleMrn, allFiltersChecksum, V2Code)
	if err != nil {
		return nil, err
	}
	if rp != nil {
		return rp, nil
	}

	// next we will try to only use the matching asset filters for the given policy...
	bundle, err := s.DataLake.GetValidatedBundle(ctx, bundleMrn)
	if err != nil {
		return nil, err
	}
	bundleMap := bundle.ToMap()

	frameworkObj := bundleMap.Frameworks[bundleMrn]
	policyObj := bundleMap.Policies[bundleMrn]

	matchingFilters, err := MatchingAssetFilters(bundleMrn, assetFilters, policyObj)
	if err != nil {
		return nil, err
	}
	if len(matchingFilters) == 0 {
		return nil, explorer.NewAssetMatchError(bundleMrn, "policies", "no-matching-policy", assetFilters, policyObj.ComputedFilters)
	}

	assetFiltersMap := make(map[string]struct{}, len(matchingFilters))
	for i := range matchingFilters {
		assetFiltersMap[matchingFilters[i].CodeId] = struct{}{}
	}

	assetFiltersChecksum, err := ChecksumAssetFilters(matchingFilters)
	if err != nil {
		return nil, err
	}

	// ... and if the filters changed, try to look up the resolved policy again
	if assetFiltersChecksum != allFiltersChecksum {
		rp, err = s.DataLake.CachedResolvedPolicy(ctx, bundleMrn, assetFiltersChecksum, V2Code)
		if err != nil {
			return nil, err
		}
		if rp != nil {
			return rp, nil
		}
	}

	// intermission: prep for the other phases
	logCtx.Debug().
		Str("bundle mrn", bundleMrn).
		Interface("asset filters", matchingFilters).
		Msg("resolver> phase 1: no cached result, resolve the bundle now")

	cache := &resolverCache{
		baseChecksum:         BundleExecutionChecksum(policyObj, frameworkObj),
		assetFiltersChecksum: assetFiltersChecksum,
		assetFilters:         assetFiltersMap,
		executionQueries:     map[string]*ExecutionQuery{},
		dataQueries:          map[string]struct{}{},
		propsCache:           explorer.NewPropsCache(),
		queriesByMsum:        map[string]*explorer.Mquery{},
		reportingJobsByUUID:  map[string]*ReportingJob{},
		reportingJobsByMsum:  map[string][]*ReportingJob{},
		reportingJobsActive:  map[string]bool{},
		bundleMap:            bundleMap,
	}

	rjUUID := cache.relativeChecksum(policyObj.GraphExecutionChecksum)

	reportingJob := &ReportingJob{
		Uuid:       rjUUID,
		QrId:       "root",
		ChildJobs:  map[string]*explorer.Impact{},
		Datapoints: map[string]bool{},
		Type:       ReportingJob_POLICY,
		// FIXME: DEPRECATED, remove in v9.0 vv
		DeprecatedV7Spec: map[string]*DeprecatedV7_ScoringSpec{},
		// ^^
	}

	cache.reportingJobsByUUID[reportingJob.Uuid] = reportingJob

	// phase 2: optimizations for assets
	// assets are always connected to a space, so figure out if a space policy exists
	// everything else in an asset can be aggregated into a shared policy

	// TODO: IMPLEMENT

	// phase 3: build the policy and scoring tree
	policyToJobsCache := &policyResolverCache{
		removedPolicies: map[string]struct{}{},
		removedQueries:  map[string]struct{}{},
		parentPolicies:  map[string]struct{}{},
		childJobsByMrn:  map[string][]*ReportingJob{},
		global:          cache,
	}
	err = s.policyToJobs(ctx, bundleMrn, reportingJob, policyToJobsCache, now)
	if err != nil {
		logCtx.Error().
			Err(err).
			Str("policy", bundleMrn).
			Msg("resolver> phase 3: internal error, trying to turn policy mrn into jobs")
		return nil, err
	}
	logCtx.Debug().
		Str("policy", bundleMrn).
		Msg("resolver> phase 3: turn policy into jobs [ok]")

	// phase 4: get all queries + assign them reporting jobs + update scoring jobs
	executionJob, collectorJob, err := s.jobsToQueries(ctx, bundleMrn, cache)
	if err != nil {
		logCtx.Error().
			Err(err).
			Str("policy", bundleMrn).
			Msg("resolver> phase 4: internal error, trying to turn policy jobs into queries")
		return nil, err
	}
	logCtx.Debug().
		Str("policy", bundleMrn).
		Msg("resolver> phase 4: aggregate queries and jobs [ok]")

	// phase 5: add frameworks and controls
	resolvedFramework := ResolveFramework(bundleMrn, bundleMap.Frameworks)
	cacheFrameworkJobs := &frameworkResolverCache{
		resolverCache:      cache,
		frameworkJobsByMrn: make(map[string]*ReportingJob),
	}
	if err := s.jobsToFrameworks(cacheFrameworkJobs, resolvedFramework, collectorJob, bundleMrn, reportingJob); err != nil {
		logCtx.Error().Err(err).
			Str("bundle", bundleMrn).
			Msg("resolver> phase 5: internal error, trying to attach framework to resolved policy")
		return nil, err
	}

	queries := bundleMap.QueryMap()
	if err := s.jobsToControls(cacheFrameworkJobs, resolvedFramework, collectorJob, queries); err != nil {
		logCtx.Error().
			Err(err).
			Str("bundle", bundleMrn).
			Msg("resolver> phase 5: internal error, trying to attach controls to resolved policy [ok]")
	}

	logCtx.Debug().
		Str("bundle", bundleMrn).
		Msg("resolver> phase 5: resolve controls [ok]")

	// phase 6: refresh all checksums
	s.refreshChecksums(executionJob, collectorJob)

	// the final phases are done in the DataLake
	for _, rj := range collectorJob.ReportingJobs {
		rj.RefreshChecksum()
	}

	resolvedPolicy := ResolvedPolicy{
		GraphExecutionChecksum: policyObj.GraphExecutionChecksum,
		Filters:                matchingFilters,
		FiltersChecksum:        assetFiltersChecksum,
		ExecutionJob:           executionJob,
		CollectorJob:           collectorJob,
		ReportingJobUuid:       reportingJob.Uuid,
	}

	err = s.DataLake.SetResolvedPolicy(ctx, bundleMrn, &resolvedPolicy, V2Code, false)
	if err != nil {
		return nil, err
	}

	return &resolvedPolicy, nil
}

func (s *LocalServices) refreshChecksums(executionJob *ExecutionJob, collectorJob *CollectorJob) {
	// execution job
	{
		queryKeys := sortx.Keys(executionJob.Queries)
		checksum := checksums.New
		checksum = checksum.Add("v2")
		for i := range queryKeys {
			key := queryKeys[i]
			checksum = checksum.Add(executionJob.Queries[key].Checksum)
		}
		executionJob.Checksum = checksum.String()
	}

	// collector job
	{
		checksum := checksums.New
		{
			reportingJobKeys := sortx.Keys(collectorJob.ReportingJobs)
			for i := range reportingJobKeys {
				key := reportingJobKeys[i]
				checksum = checksum.Add(key)
				checksum = checksum.Add(collectorJob.ReportingJobs[key].Checksum)
			}
		}
		{
			datapointsKeys := sortx.Keys(collectorJob.Datapoints)
			for i := range datapointsKeys {
				key := datapointsKeys[i]
				info := collectorJob.Datapoints[key]
				checksum = checksum.Add(key)
				checksum = checksum.Add(info.Type)

				notify := make([]string, len(info.Notify))
				copy(notify, info.Notify)
				sort.Strings(notify)
				for j := range notify {
					checksum = checksum.Add(notify[j])
				}
			}
		}
		scoringChecksumStr := checksum.String()
		collectorJob.Checksum = scoringChecksumStr
	}
}

func (s *LocalServices) policyToJobs(ctx context.Context, policyMrn string, ownerJob *ReportingJob,
	parentCache *policyResolverCache, now time.Time,
) error {
	ctx, span := tracer.Start(ctx, "resolver/policyToJobs")
	defer span.End()

	policyObj, ok := parentCache.global.bundleMap.Policies[policyMrn]
	if !ok || policyObj == nil {
		return errors.New("cannot find policy '" + policyMrn + "' while resolving")
	}

	if len(policyObj.Groups) == 0 {
		return nil
	}

	cache := parentCache.clone()
	cache.parentPolicies[policyMrn] = struct{}{}

	// properties to execution queries cache
	parentCache.global.propsCache.Add(policyObj.Props...)

	// get a list of matching specs
	matchingGroups := []*PolicyGroup{}
	for i := range policyObj.Groups {
		group := policyObj.Groups[i]

		// Filter out groups that are not active
		if group.EndDate != 0 {
			endDate := time.Unix(group.EndDate, 0)
			if endDate.Before(now) {
				continue
			}
		}

		if group.Filters == nil || len(group.Filters.Items) == 0 {
			matchingGroups = append(matchingGroups, group)
			continue
		}

		for j := range group.Filters.Items {
			filter := group.Filters.Items[j]
			if _, ok := cache.global.assetFilters[filter.CodeId]; ok {
				matchingGroups = append(matchingGroups, group)
				break
			}
		}
	}

	// aggregate all removed policies and queries
	for i := range matchingGroups {
		group := matchingGroups[i]
		for i := range group.Policies {
			policy := group.Policies[i]
			if policy.Action == explorer.Action_DEACTIVATE {
				cache.removedPolicies[policy.Mrn] = struct{}{}
			}
		}
		for i := range group.Checks {
			check := group.Checks[i]
			if check.Action == explorer.Action_DEACTIVATE {
				cache.removedQueries[check.Mrn] = struct{}{}
			}
		}
		for i := range group.Queries {
			query := group.Queries[i]
			if query.Action == explorer.Action_DEACTIVATE {
				cache.removedQueries[query.Mrn] = struct{}{}
			}
		}
	}

	// resolve the rest
	var err error
	for i := range matchingGroups {
		group := matchingGroups[i]
		if err = s.policyGroupToJobs(ctx, group, ownerJob, cache, now); err != nil {
			log.Error().Err(err).Msg("resolver> policyToJobs error")
			return err
		}
	}

	// finalize
	parentCache.addChildren(cache)

	return nil
}

func (s *LocalServices) policyGroupToJobs(ctx context.Context, group *PolicyGroup, ownerJob *ReportingJob, cache *policyResolverCache, now time.Time) error {
	ctx, span := tracer.Start(ctx, "resolver/policyGroupToJobs")
	defer span.End()

	// include referenced policies
	for i := range group.Policies {
		policy := group.Policies[i]

		impact := policy.Impact
		if policy.Action == explorer.Action_IGNORE {
			impact = &explorer.Impact{
				Scoring: explorer.ScoringSystem_IGNORE_SCORE,
			}
		}

		// ADD
		if policy.Action == explorer.Action_UNSPECIFIED || policy.Action == explorer.Action_ACTIVATE || policy.Action == explorer.Action_IGNORE {
			if _, ok := cache.parentPolicies[policy.Mrn]; ok {
				return errors.New("trying to resolve policy spec twice, it is cyclical for MRN: " + policy.Mrn)
			}

			if _, ok := cache.removedPolicies[policy.Mrn]; ok {
				continue
			}

			// before adding any reporting job, make sure this policy actually works for
			// this set of asset filters
			policyObj, ok := cache.global.bundleMap.Policies[policy.Mrn]
			if !ok || policyObj == nil {
				return errors.New("cannot find policy '" + policy.Mrn + "' while resolving")
			}

			// make sure this policy supports the selected filters, otherwise we
			// don't need to include it
			var found bool
			for checksum := range policyObj.ComputedFilters.Items {
				if _, ok := cache.global.assetFilters[checksum]; ok {
					found = true
					break
				}
			}
			if !found {
				continue
			}

			// TODO: We currently enforce the policy object to only be created
			// once per resolution. It can be attached to multiple other jobs,
			// i.e. it can be called by multiple different owners. But it only
			// translates into one reportingjob. This will need to be expanded
			// in case we allow for modifications on the policy object.
			// vv-------------- singular policyref ----------------
			var policyJob *ReportingJob
			policyJobs := cache.childJobsByMrn[policy.Mrn]
			if len(policyJobs) == 0 {
				// FIXME: we are receiving policies here that may not have their checksum calculated
				// This should not be the case, policy bundles that are downloaded
				// should have their checksums updated.
				policy.RefreshChecksum()

				policyJob = &ReportingJob{
					QrId:          policy.Mrn,
					Uuid:          cache.global.relativeChecksum(policy.Checksum),
					ChildJobs:     map[string]*explorer.Impact{},
					Datapoints:    map[string]bool{},
					ScoringSystem: policyObj.ScoringSystem,
					Type:          ReportingJob_POLICY,
					// FIXME: DEPRECATED, remove in v9.0 vv
					DeprecatedV7Spec: map[string]*DeprecatedV7_ScoringSpec{},
					// ^^
				}
				cache.global.reportingJobsByUUID[policyJob.Uuid] = policyJob
				cache.childJobsByMrn[policy.Mrn] = []*ReportingJob{policyJob}
			} else {
				if len(policyJobs) != 1 {
					log.Warn().Msg("found more than one policy job for " + policy.Mrn)
				}
				policyJob = policyJobs[0]
			}
			// ^^-------------- singular policyref ----------------

			// local aspects for the resolved policy
			policyJob.Notify = append(policyJob.Notify, ownerJob.Uuid)
			ownerJob.ChildJobs[policyJob.Uuid] = impact
			// FIXME: DEPRECATED, remove in v9.0 vv
			ownerJob.DeprecatedV7Spec[policyJob.Uuid] = Impact2ScoringSpec(impact)
			// ^^

			if err := s.policyToJobs(ctx, policy.Mrn, policyJob, cache, now); err != nil {
				return err
			}

			continue
		}

		// MODIFY
		if policy.Action == explorer.Action_MODIFY {
			policyJobs, ok := cache.childJobsByMrn[policy.Mrn]
			if !ok {
				cache.global.errors = append(cache.global.errors, &policyResolutionError{
					ID:       policy.Mrn,
					IsPolicy: true,
					Error:    "cannot modify policy, it doesn't exist",
				})
				continue
			}

			for j := range policyJobs {
				policyJob := policyJobs[j]
				for _, id := range policyJob.Notify {
					parentJob := cache.global.reportingJobsByUUID[id]
					if parentJob != nil {
						parentJob.ChildJobs[policyJob.Uuid] = impact
						// FIXME: DEPRECATED, remove in v9.0 vv
						parentJob.DeprecatedV7Spec[policyJob.Uuid] = Impact2ScoringSpec(impact)
						// ^^
					}
				}
			}
		}
	}

	// handle scoring queries
	for i := range group.Checks {
		check := group.Checks[i]

		if _, ok := cache.removedQueries[check.Mrn]; ok {
			continue
		}

		if base, ok := cache.global.bundleMap.Queries[check.Mrn]; ok {
			check = check.Merge(base)
			if err := check.RefreshChecksum(ctx, explorer.QueryMap(cache.global.bundleMap.Queries).GetQuery); err != nil {
				return err
			}
		}
		if check.Checksum == "" {
			return errors.New("invalid check encountered, missing checksum for: " + check.Mrn)
		}

		if !check.Filters.Supports(cache.global.assetFilters) {
			continue
		}

		validUntil := time.Unix(group.EndDate, 0).Format(time.RFC3339)
		if check.Action == explorer.Action_IGNORE {
			stillValid := CheckValidUntil(validUntil, check.Mrn)
			if !stillValid {
				// the exception is no longer valid => score the check
				check.Action = explorer.Action_ACTIVATE
			}
		}
		// If we ignore this check, we have to transfer this info to the impact var,
		// which is used to inform how to aggregate the scores of all child jobs.
		impact := check.Impact
		if impact == nil {
			impact = &explorer.Impact{}
		}
		if check.Action == explorer.Action_IGNORE {
			impact.Scoring = explorer.ScoringSystem_IGNORE_SCORE
			impact.Action = check.Action
		}

		cache.global.propsCache.Add(check.Props...)

		if check.Action == explorer.Action_UNSPECIFIED || check.Action == explorer.Action_ACTIVATE {
			cache.addCheckJob(ctx, check, impact, ownerJob)
			continue
		}

		if check.Action == explorer.Action_MODIFY {
			cache.modifyCheckJob(check, impact)
		}
	}

	// handle data queries
	for i := range group.Queries {
		query := group.Queries[i]

		if _, ok := cache.removedQueries[query.Mrn]; ok {
			continue
		}

		if base, ok := cache.global.bundleMap.Queries[query.Mrn]; ok {
			query = query.Merge(base)
			if err := query.RefreshChecksum(ctx, explorer.QueryMap(cache.global.bundleMap.Queries).GetQuery); err != nil {
				return err
			}
		}
		if query.Checksum == "" {
			return errors.New("invalid query encountered, missing checksum for: " + query.Mrn)
		}

		if !query.Filters.Supports(cache.global.assetFilters) {
			continue
		}

		// Dom: Note: we do not carry over the impact from data queries yet

		cache.global.propsCache.Add(query.Props...)

		// ADD
		if query.Action == explorer.Action_UNSPECIFIED || query.Action == explorer.Action_ACTIVATE {
			cache.addDataQueryJob(ctx, query, ownerJob)
		}
	}

	return nil
}

func (cache *policyResolverCache) addCheckJob(ctx context.Context, check *explorer.Mquery, impact *explorer.Impact, ownerJob *ReportingJob) {
	uuid := cache.global.relativeChecksum(check.Checksum)
	queryJob := cache.global.reportingJobsByUUID[uuid]

	if queryJob == nil {
		queryJob = &ReportingJob{
			Uuid:       uuid,
			QrId:       check.Mrn,
			ChildJobs:  map[string]*explorer.Impact{},
			Datapoints: map[string]bool{},
			Type:       ReportingJob_CHECK,
			// FIXME: DEPRECATED, remove in v9.0 vv
			DeprecatedV7Spec: map[string]*DeprecatedV7_ScoringSpec{},
			// ^^
		}
		cache.global.reportingJobsByUUID[uuid] = queryJob
		cache.global.reportingJobsByMsum[check.Checksum] = append(cache.global.reportingJobsByMsum[check.Checksum], queryJob)
		cache.childJobsByMrn[check.Mrn] = append(cache.childJobsByMrn[check.Mrn], queryJob)
	}

	// local aspects for the resolved policy
	queryJob.Notify = append(queryJob.Notify, ownerJob.Uuid)

	ownerJob.ChildJobs[queryJob.Uuid] = impact
	// FIXME: DEPRECATED, remove in v9.0 vv
	ownerJob.DeprecatedV7Spec[queryJob.Uuid] = Impact2ScoringSpec(impact)
	// ^^

	if len(check.Variants) != 0 {
		err := cache.addCheckJobVariants(ctx, check, queryJob)
		if err != nil {
			log.Error().Err(err).Str("checkMrn", check.Mrn).Msg("failed to add data query variants")
		}
	} else {
		// we set a placeholder for the execution query, just to indicate it will be added
		cache.global.executionQueries[check.Checksum] = nil
		cache.global.queriesByMsum[check.Checksum] = check
	}
}

func (cache *policyResolverCache) addCheckJobVariants(ctx context.Context, query *explorer.Mquery, ownerJob *ReportingJob) error {
	for i := range query.Variants {
		mrn := query.Variants[i].Mrn

		if _, ok := cache.removedQueries[mrn]; ok {
			continue
		}

		v, ok := cache.global.bundleMap.Queries[mrn]
		if !ok {
			return errors.New("cannot find variant " + mrn)
		}
		if v.Checksum == "" {
			return errors.New("invalid check encountered, missing checksum for: " + mrn)
		}

		if !v.Filters.Supports(cache.global.assetFilters) {
			continue
		}

		// Dom: Note: we do not carry over the impact from data queries yet

		cache.global.propsCache.Add(v.Props...)

		// ADD
		if v.Action == explorer.Action_UNSPECIFIED || v.Action == explorer.Action_ACTIVATE {
			cache.addCheckJob(ctx, v, v.Impact, ownerJob)
		}
	}

	return nil
}

func (cache *policyResolverCache) addDataQueryJob(ctx context.Context, query *explorer.Mquery, ownerJob *ReportingJob) {
	if len(query.Variants) != 0 {
		err := cache.addDataQueryVariants(ctx, query, ownerJob)
		if err != nil {
			log.Error().Err(err).Str("queryMrn", query.Mrn).Msg("failed to add data query variants")
		}
		return
	}

	uuid := cache.global.relativeChecksum(query.Checksum)
	queryJob := cache.global.reportingJobsByUUID[uuid]

	// note: the ReportingJob is only a placeholder and is replaced by individual query LLX checksum ReportingJobs
	if queryJob == nil {
		queryJob = &ReportingJob{
			Uuid:       cache.global.relativeChecksum(query.Checksum),
			QrId:       query.Mrn,
			ChildJobs:  map[string]*explorer.Impact{},
			Datapoints: map[string]bool{},
			Type:       ReportingJob_DATA_QUERY,
			// FIXME: DEPRECATED, remove in v10.0 vv
			DeprecatedV8IsData: true,
			// ^^
			// FIXME: DEPRECATED, remove in v9.0 vv
			DeprecatedV7Spec: map[string]*DeprecatedV7_ScoringSpec{},
			// ^^
		}
		cache.global.reportingJobsByUUID[queryJob.Uuid] = queryJob
		cache.global.reportingJobsByMsum[query.Checksum] = append(cache.global.reportingJobsByMsum[query.Checksum], queryJob)
		cache.childJobsByMrn[query.Mrn] = append(cache.childJobsByMrn[query.Mrn], queryJob)
	}

	// local aspects for the resolved policy
	queryJob.Notify = append(queryJob.Notify, ownerJob.Uuid)

	ownerJob.Datapoints[queryJob.Uuid] = true
	// we set a placeholder for the execution query, just to indicate it will be added
	cache.global.executionQueries[query.Checksum] = nil
	cache.global.dataQueries[query.Checksum] = struct{}{}
	cache.global.queriesByMsum[query.Checksum] = query
}

func (cache *policyResolverCache) addDataQueryVariants(ctx context.Context, query *explorer.Mquery, ownerJob *ReportingJob) error {
	for i := range query.Variants {
		mrn := query.Variants[i].Mrn

		if _, ok := cache.removedQueries[mrn]; ok {
			continue
		}

		v, ok := cache.global.bundleMap.Queries[mrn]
		if !ok {
			return errors.New("cannot find variant " + mrn)
		}
		if v.Checksum == "" {
			return errors.New("invalid query encountered, missing checksum for: " + mrn)
		}

		if !v.Filters.Supports(cache.global.assetFilters) {
			continue
		}

		// Dom: Note: we do not carry over the impact from data queries yet

		cache.global.propsCache.Add(v.Props...)

		// ADD
		if v.Action == explorer.Action_UNSPECIFIED || v.Action == explorer.Action_ACTIVATE {
			cache.addDataQueryJob(ctx, v, ownerJob)
		}
	}

	return nil
}

func (cache *policyResolverCache) modifyCheckJob(check *explorer.Mquery, impact *explorer.Impact) {
	queryJobs, ok := cache.childJobsByMrn[check.Mrn]
	if !ok {
		cache.global.errors = append(cache.global.errors, &policyResolutionError{
			ID:       check.Mrn,
			IsPolicy: true,
			Error:    "cannot modify query, it doesn't exist",
		})
		return
	}

	for i := range queryJobs {
		queryJob := queryJobs[i]
		for _, id := range queryJob.Notify {
			parentJob := cache.global.reportingJobsByUUID[id]
			if parentJob != nil {
				parentJob.ChildJobs[queryJob.Uuid] = impact
				// FIXME: DEPRECATED, remove in v9.0 vv
				parentJob.DeprecatedV7Spec[queryJob.Uuid] = Impact2ScoringSpec(impact)
				// ^^
			}
		}
	}
}

// type propInfo struct {
// 	prop         *explorer.Property
// 	typ          *llx.Primitive
// 	dataChecksum string
// 	name         string
// }

func (s *LocalServices) jobsToQueries(ctx context.Context, policyMrn string, cache *resolverCache) (*ExecutionJob, *CollectorJob, error) {
	ctx, span := tracer.Start(ctx, "resolver/jobsToQueries")
	defer span.End()

	logCtx := logger.FromContext(ctx)
	collectorJob := &CollectorJob{
		ReportingJobs:    map[string]*ReportingJob{},
		ReportingQueries: map[string]*StringArray{},
		Datapoints:       map[string]*DataQueryInfo{},
	}
	executionJob := &ExecutionJob{
		Queries: map[string]*ExecutionQuery{},
	}

	// fill in all reporting jobs. we will remove the data query jobs and replace
	// them with direct collections into their parent job later
	for _, rj := range cache.reportingJobsByUUID {
		collectorJob.ReportingJobs[rj.Uuid] = rj
	}

	// FIXME: sort by internal dependencies of props as well

	// next we can continue with queries, after properties are all done
	for checksum, query := range cache.queriesByMsum {
		codeID := query.CodeId

		if existing, ok := executionJob.Queries[codeID]; ok {
			logCtx.Debug().
				Str("codeID", codeID).
				Str("existing", existing.Query).
				Str("new", query.Mql).
				Msg("resolver> found duplicate query")
		}

		_, isDataQuery := cache.dataQueries[query.Checksum]

		var propTypes map[string]*llx.Primitive
		var propToChecksums map[string]string
		if len(query.Props) != 0 {
			propTypes = make(map[string]*llx.Primitive, len(query.Props))
			propToChecksums = make(map[string]string, len(query.Props))
			for j := range query.Props {
				prop := query.Props[j]

				// we only get this if there is an override higher up in the policy
				override, name, _ := cache.propsCache.Get(ctx, prop.Mrn)
				if override != nil {
					prop = override
				}
				if name == "" {
					var err error
					name, err = mrn.GetResource(prop.Mrn, MRN_RESOURCE_QUERY)
					if err != nil {
						return nil, nil, errors.New("failed to get property name")
					}
				}

				executionQuery, dataChecksum, err := mquery2executionQuery(prop, nil, map[string]string{}, collectorJob, false)
				if err != nil {
					return nil, nil, errors.New("resolver> failed to compile query for MRN " + prop.Mrn + ": " + err.Error())
				}
				if dataChecksum == "" {
					return nil, nil, errors.New("property returns too many value, cannot determine entrypoint checksum: '" + prop.Mql + "'")
				}
				cache.executionQueries[checksum] = executionQuery
				executionJob.Queries[prop.CodeId] = executionQuery

				propTypes[name] = &llx.Primitive{Type: prop.Type}
				propToChecksums[name] = dataChecksum
			}
		}

		executionQuery, _, err := mquery2executionQuery(query, propTypes, propToChecksums, collectorJob, !isDataQuery)
		if err != nil {
			return nil, nil, errors.New("resolver> failed to compile query for MRN " + query.Mrn + ": " + err.Error())
		}

		if executionQuery == nil {
			// This case happens when we were able to compile with the
			// v2 compiler but not the v1 compiler. In such case, we
			// will expunge the query and reporting chain from the
			// resolved policy
			if reportingjobs, ok := cache.reportingJobsByMsum[query.Checksum]; ok {
				for i := range reportingjobs {
					rj := reportingjobs[i]
					delete(cache.reportingJobsByUUID, rj.Uuid)
					delete(collectorJob.ReportingJobs, rj.Uuid)
					for _, parentID := range rj.Notify {
						if parentJob, ok := collectorJob.ReportingJobs[parentID]; ok {
							delete(parentJob.ChildJobs, rj.Uuid)
						}
					}
				}
				delete(cache.reportingJobsByMsum, query.Checksum)
			}

			continue
		}

		cache.executionQueries[checksum] = executionQuery
		executionJob.Queries[codeID] = executionQuery

		// Scoring+Data Queries handling
		reportingjobs, ok := cache.reportingJobsByMsum[query.Checksum]
		if !ok {
			logCtx.Debug().
				Interface("reportingJobs", cache.reportingJobsByMsum).
				Str("query", query.Mrn).
				Str("checksum", query.Checksum).
				Str("policy", policyMrn).
				Msg("resolver> phase 4: cannot find reporting job")
			return nil, nil, errors.New("cannot find reporting job for query " + query.Mrn + " in policy " + policyMrn)
		}

		for i := range reportingjobs {
			rj := reportingjobs[i]
			// (2) Scoring Queries handling
			if !isDataQuery {
				rj.QrId = codeID

				if query.Impact != nil {
					for _, parentID := range rj.Notify {
						parentJob, ok := collectorJob.ReportingJobs[parentID]
						if !ok {
							return nil, nil, errors.New("failed to connect datapoint to reporting job")
						}
						base := parentJob.ChildJobs[rj.Uuid]
						query.Impact.AddBase(base)
					}
				}

				arr, ok := collectorJob.ReportingQueries[codeID]
				if !ok {
					arr = &StringArray{}
					collectorJob.ReportingQueries[codeID] = arr
				}
				arr.Items = append(arr.Items, rj.Uuid)

				// process all the datapoints of this job
				for dp := range executionQuery.Datapoints {
					datapointID := executionQuery.Datapoints[dp]
					datapointInfo, ok := collectorJob.Datapoints[datapointID]
					if !ok {
						return nil, nil, errors.New("failed to identity datapoint in collectorjob")
					}

					datapointInfo.Notify = append(datapointInfo.Notify, rj.Uuid)
					rj.Datapoints[datapointID] = true
				}

				continue
			}

			// (3) Data Queries handling
			for _, parentID := range rj.Notify {
				parentJob, ok := collectorJob.ReportingJobs[parentID]
				if !ok {
					return nil, nil, errors.New("failed to connect datapoint to reporting job")
				}

				for dp := range executionQuery.Datapoints {
					datapointID := executionQuery.Datapoints[dp]
					datapointInfo, ok := collectorJob.Datapoints[datapointID]
					if !ok {
						return nil, nil, errors.New("failed to identity datapoint in collectorjob")
					}

					datapointInfo.Notify = append(datapointInfo.Notify, parentJob.Uuid)
					parentJob.Datapoints[datapointID] = true
				}

				// we don't need this any longer, since every datapoint now reports into
				// the parent job (i.e. reports into the policy instead of the data query)
				delete(collectorJob.ReportingJobs, rj.Uuid)
				delete(parentJob.Datapoints, rj.Uuid)
			}
		}
	}

	return executionJob, collectorJob, nil
}

type queryLike interface {
	Compile(props map[string]*llx.Primitive) (*llx.CodeBundle, error)
	GetChecksum() string
	GetMql() string
}

func mquery2executionQuery(query queryLike, props map[string]*llx.Primitive, propsToChecksums map[string]string, collectorJob *CollectorJob, isScoring bool) (*ExecutionQuery, string, error) {
	bundle, err := query.Compile(props)
	if err != nil {
		return nil, "", err
	}

	code := bundle.CodeV2

	dataChecksum := ""
	codeEntrypoints := code.Entrypoints()
	if len(codeEntrypoints) == 1 {
		ref := codeEntrypoints[0]
		dataChecksum = code.Checksums[ref]
	}

	codeDatapoints := code.Datapoints()

	refs := make([]uint64, len(codeDatapoints))
	copy(refs, codeDatapoints)

	// We collect the entrypoints as they contain information we
	// need to correctly build the assessments
	refs = append(refs, codeEntrypoints...)

	datapoints := make([]string, len(refs))
	for i := range refs {
		ref := refs[i]
		checksum := code.Checksums[ref]
		datapoints[i] = checksum

		// TODO: correct transplation from upper and lower parts of ref

		typ := code.Chunk(ref).DereferencedTypeV2(code)
		collectorJob.Datapoints[checksum] = &DataQueryInfo{
			Type: string(typ),
		}
	}

	// translate properties: we get bundle props < name => Type >
	// we need to get execution query props: < name => checksum >
	eqProps := map[string]string{}
	for name := range bundle.Props {
		checksum := propsToChecksums[name]
		if checksum == "" {
			return nil, "", errors.New("cannot find checksum for property " + name + " in query '" + query.GetMql() + "'")
		}
		eqProps[name] = checksum
	}

	res := ExecutionQuery{
		Query:      query.GetMql(),
		Checksum:   query.GetChecksum(),
		Properties: eqProps,
		Datapoints: datapoints,
		Code:       bundle,
	}

	return &res, dataChecksum, nil
}

func ensureControlJob(cache *frameworkResolverCache, jobs map[string]*ReportingJob, controlMrn string, framework *ResolvedFramework, frameworkGroupByControlMrn map[string]*FrameworkGroup) *ReportingJob {
	uuid := cache.relativeChecksum(controlMrn)

	if found, ok := jobs[uuid]; ok {
		return found
	}

	// If we ignore this control, we have to transfer this info to the impact var,
	// which is used to inform how to aggregate the results of all child jobs.
	impact := &explorer.Impact{}
	validUntil := ""
	if frameworkGroup, ok := frameworkGroupByControlMrn[controlMrn]; ok {
		if frameworkGroup.Type == GroupType_IGNORED {
			if frameworkGroup.EndDate != 0 {
				validUntil = time.Unix(frameworkGroup.EndDate, 0).Format(time.RFC3339)
			}
			stillIgnore := CheckValidUntil(validUntil, controlMrn)
			if stillIgnore {
				impact.Scoring = explorer.ScoringSystem_IGNORE_SCORE
				impact.Action = explorer.Action_IGNORE
			}
		}
	}

	controlJob := &ReportingJob{
		Uuid:          uuid,
		QrId:          controlMrn,
		ChildJobs:     map[string]*explorer.Impact{},
		ScoringSystem: explorer.ScoringSystem_WORST,
		Type:          ReportingJob_CONTROL,
	}
	jobs[uuid] = controlJob

	parents := framework.ReportTargets[controlMrn]
	for _, parentMrn := range parents {
		parentUuid := cache.relativeChecksum(parentMrn)

		frameworkJob, ok := cache.frameworkJobsByMrn[parentMrn]
		if !ok {
			continue
		}

		frameworkJob.ChildJobs[uuid] = impact
		controlJob.Notify = append(controlJob.Notify, parentUuid)
	}

	return controlJob
}

type frameworkResolverCache struct {
	*resolverCache
	frameworkJobsByMrn map[string]*ReportingJob
}

func (s *LocalServices) jobsToFrameworks(cache *frameworkResolverCache, resolvedFramework *ResolvedFramework, job *CollectorJob, frameworkMrn string, parent *ReportingJob) error {
	for k, rj := range job.ReportingJobs {
		if frameworkJob := cache.bundleMap.Frameworks[rj.QrId]; frameworkJob != nil {
			cache.frameworkJobsByMrn[rj.QrId] = job.ReportingJobs[k]
		}
	}
	return s.jobsToFrameworksInner(cache, resolvedFramework, job, frameworkMrn, parent)
}

func (s *LocalServices) jobsToFrameworksInner(cache *frameworkResolverCache, resolvedFramework *ResolvedFramework, job *CollectorJob, frameworkMrn string, parent *ReportingJob) error {
	for _, source := range resolvedFramework.ReportSources[frameworkMrn] {
		if childFramework, ok := cache.bundleMap.Frameworks[source]; ok {
			var reportingJob *ReportingJob
			if found, ok := cache.frameworkJobsByMrn[childFramework.Mrn]; ok {
				// Look for an existing job. This will happen for asset and space frameworks.
				// Creating a new job for these would likely cause confusion, as then we'd
				// end up with multiple jobs with the same QrId
				reportingJob = found
			} else {
				uuid := cache.relativeChecksum(childFramework.Mrn)
				reportingJob = &ReportingJob{
					Uuid:          uuid,
					QrId:          childFramework.Mrn,
					ChildJobs:     map[string]*explorer.Impact{},
					ScoringSystem: explorer.ScoringSystem_WORST,
					Type:          ReportingJob_FRAMEWORK,
				}
			}

			if _, exist := parent.ChildJobs[reportingJob.Uuid]; !exist {
				// If we already have a child job, we don't need to do anything
				// In the case that its a space or asset, we defiently don't want to
				// overwrite it as that would change the scoring system
				impact := &explorer.Impact{}
				if parent.Type == ReportingJob_FRAMEWORK {
					impact.Scoring = explorer.ScoringSystem_WORST
				} else {
					impact.Scoring = explorer.ScoringSystem_IGNORE_SCORE
				}
				parent.ChildJobs[reportingJob.Uuid] = impact
			}
			reportingJob.Notify = append(reportingJob.Notify, parent.Uuid)
			job.ReportingJobs[reportingJob.Uuid] = reportingJob
			cache.frameworkJobsByMrn[childFramework.Mrn] = reportingJob
			if err := s.jobsToFrameworksInner(cache, resolvedFramework, job, source, reportingJob); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *LocalServices) jobsToControls(cache *frameworkResolverCache, framework *ResolvedFramework, job *CollectorJob, querymap map[string]*explorer.Mquery) error {
	nuJobs := map[string]*ReportingJob{}

	// try to find all framework groups of type IGNORE or DISABLE for this and depending frameworks
	// these groups are needed to determine if a control is ignored/snoozed or disabled
	frameworkGroupByControlMrn := map[string]*FrameworkGroup{}
	assetFramework := cache.bundleMap.Frameworks[framework.Mrn]
	if assetFramework != nil {
		for i := range assetFramework.Dependencies {
			depFramework := cache.bundleMap.Frameworks[assetFramework.Dependencies[i].Mrn]
			if depFramework != nil {
				for j := range depFramework.Groups {
					group := depFramework.Groups[j]
					if group.Type != GroupType_IGNORED && group.Type != GroupType_DISABLE {
						continue
					}
					for k := range group.Controls {
						frameworkGroupByControlMrn[group.Controls[k].Mrn] = group
					}
				}
			}
		}
	}

	for _, rj := range job.ReportingJobs {
		query, ok := querymap[rj.QrId]
		if !ok {
			log.Warn().Str("mrn", framework.Mrn)
			continue
		}

		targets, ok := framework.ReportTargets[query.Mrn]
		if !ok {
			continue
		}

		for i := range targets {
			controlMrn := targets[i]
			// skip controls which are part of a FrameworkGroup with type DISABLE
			if group, ok := frameworkGroupByControlMrn[controlMrn]; ok {
				if group.Type == GroupType_DISABLE {
					continue
				}
			}
			controlJob := ensureControlJob(cache, nuJobs, controlMrn, framework, frameworkGroupByControlMrn)

			controlJob.ChildJobs[rj.Uuid] = nil
			rj.Notify = append(rj.Notify, controlJob.Uuid)
		}
	}

	for k, v := range nuJobs {
		job.ReportingJobs[k] = v
	}

	return nil
}

func (s *LocalServices) cacheUpstreamJobs(ctx context.Context, assetMrn string, resolvedPolicy *ResolvedPolicy) error {
	var err error

	if err = s.DataLake.EnsureAsset(ctx, assetMrn); err != nil {
		return errors.New("resolver> failed to cache upstream jobs: " + err.Error())
	}

	err = s.DataLake.SetResolvedPolicy(ctx, assetMrn, resolvedPolicy, V2Code, true)
	if err != nil {
		return errors.New("resolver> failed to cache resolved upstream policy: " + err.Error())
	}

	err = s.DataLake.SetAssetResolvedPolicy(ctx, assetMrn, resolvedPolicy, V2Code)
	if err != nil {
		return errors.New("resolver> failed to cache resolved upstream policy into asset: " + err.Error())
	}

	return nil
}

func (s *LocalServices) updateAssetJobs(ctx context.Context, assetMrn string, assetFilters []*explorer.Mquery) error {
	resolvedPolicy, err := s.resolve(ctx, assetMrn, assetFilters)
	if err != nil {
		return err
	}

	return s.DataLake.SetAssetResolvedPolicy(ctx, assetMrn, resolvedPolicy, V2Code)
}

// CheckValidUntil returns whether the given time is lying in the future or not.
// Specialcase is an empty string, which is treated as forever.
func CheckValidUntil(validUntil string, mrn string) bool {
	stillIgnore := false
	// empty validUntil means ignore forever
	if validUntil == "" {
		stillIgnore = true
		log.Debug().Str("mrn", mrn).Msg("control is ignored forever")
	} else {
		validTime, err := time.Parse(time.RFC3339, validUntil)
		if err != nil {
			// user wanted an exception, but something went wrong with the date
			// should we bubble up the error?
			log.Error().Err(err).Str("mrn", mrn).Msg("failed to parse validUntil")
		} else {
			if validTime.After(time.Now()) {
				stillIgnore = true
				log.Debug().Str("mrn", mrn).Msg("is ignored for now because of validUntil timestamp")
			}
		}
	}
	return stillIgnore
}
