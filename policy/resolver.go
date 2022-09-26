package policy

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/gogo/status"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/fasthash/fnv1a"
	"go.mondoo.com/cnquery"
	"go.mondoo.com/cnquery/checksums"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/logger"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
)

const (
	POLICY_SERVICE_NAME = "policy.api.mondoo.com"
)

// Assign a policy to an asset
//
// We need to handle multiple cases:
// 1. all local, polices and assets are available locally
// 2. asset is local (via incognito mode) but policy is upstream
// 3. asset and policy are upstream
func (s *LocalServices) Assign(ctx context.Context, assignment *PolicyAssignment) (*Empty, error) {

	if len(assignment.PolicyMrns) == 0 {
		return nil, status.Error(codes.InvalidArgument, "a policy mrn is required")
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

	// assign policy locally
	deltas := map[string]*PolicyDelta{}
	for i := range assignment.PolicyMrns {
		policyMrn := assignment.PolicyMrns[i]
		deltas[policyMrn] = &PolicyDelta{
			PolicyMrn: policyMrn,
			Action:    PolicyDelta_ADD,
		}
	}

	_, err := s.DataLake.MutatePolicy(ctx, &PolicyMutationDelta{
		PolicyMrn:    assignment.AssetMrn,
		PolicyDeltas: deltas,
	}, true)
	return globalEmpty, err
}

// Resolve a given policy for a set of asset filters
func (s *LocalServices) Resolve(ctx context.Context, req *ResolveReq) (*ResolvedPolicy, error) {
	if s.Upstream != nil && !s.Incognito {
		return s.Upstream.Resolve(ctx, req)
	}

	return s.resolve(ctx, req.PolicyMrn, req.AssetFilters)
}

// GetReport retreives a report for a given asset and policy
func (s *LocalServices) GetReport(ctx context.Context, req *EntityScoreRequest) (*Report, error) {
	return s.DataLake.GetReport(ctx, req.EntityMrn, req.ScoreMrn)
}

// GetScore retrieves one score for an asset
func (s *LocalServices) GetScore(ctx context.Context, req *EntityScoreRequest) (*Report, error) {
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

// HELPER METHODS
// =================

// CreatePolicyObject creates a policy object without saving it and returns it
func (s *LocalServices) CreatePolicyObject(policyMrn string, ownerMrn string) *Policy {
	policyScoringSpec := map[string]*ScoringSpec{}

	// TODO: this should be handled better and I'm not sure yet how...
	// we need to ensure a good owner MRN exists for all objects, including orgs and spaces
	// this is the case when we are in incognito mode
	if ownerMrn == "" {
		log.Debug().Str("policyMrn", policyMrn).Msg("ownerMrn is missing")
		ownerMrn = "//policy.api.mondoo.app"
	}

	policyObj := Policy{
		Mrn:  policyMrn,
		Name: policyMrn, // just as a placeholder, replace with something better
		// should we set a semver version here as well?, right now, the policy validation makes an
		// exception for space and asset policies
		Version: "n/a",
		Specs: []*PolicySpec{{
			Policies:       policyScoringSpec,
			ScoringQueries: map[string]*ScoringSpec{},
			DataQueries:    map[string]QueryAction{},
		}},
		OwnerMrn: ownerMrn,
		IsPublic: false,
	}

	return &policyObj
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
	graphExecutionChecksum string
	assetFiltersChecksum   string
	assetFilters           map[string]struct{}

	// assigned queries, listed by their UUID (i.e. policy context)
	executionQueries map[string]*ExecutionQuery
	dataQueries      map[string]struct{}
	propQueries      map[string]struct{}
	queries          map[string]interface{}

	reportingJobsByQrID map[string]*ReportingJob
	reportingJobsByUUID map[string]*ReportingJob
	reportingJobsActive map[string]bool
	errors              []*policyResolutionError
	useV2Code           bool
	bundleMap           *PolicyBundleMap
}

type policyResolverCache struct {
	removedPolicies map[string]struct{} // tracks policies that will not be added
	removedQueries  map[string]struct{} // tracks queries that will not be added
	parentPolicies  map[string]struct{} // tracks policies in the ancestry, to prevent loops
	childPolicies   map[string]struct{} // tracks policies that were added below (at any level)
	childQueries    map[string]struct{} // tracks queries that were added below (at any level)
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
	if r.useV2Code {
		return checksumStrings(r.graphExecutionChecksum, r.assetFiltersChecksum, "v2", s)
	} else {
		return checksumStrings(r.graphExecutionChecksum, r.assetFiltersChecksum, s)
	}
}

func (p *policyResolverCache) clone() *policyResolverCache {
	res := &policyResolverCache{
		removedPolicies: map[string]struct{}{},
		removedQueries:  map[string]struct{}{},
		parentPolicies:  map[string]struct{}{},
		childPolicies:   map[string]struct{}{},
		childQueries:    map[string]struct{}{},
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
	for k, v := range other.childPolicies {
		p.childPolicies[k] = v
	}
	for k, v := range other.childQueries {
		p.childQueries[k] = v
	}
}

func (s *LocalServices) resolve(ctx context.Context, policyMrn string, assetFilters []*Mquery) (*ResolvedPolicy, error) {
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
				logCtx.Error().Int("try", i+1).Dur("sleepTime", sleepTime).Msg("Retrying policy resolution")
				time.Sleep(sleepTime)
			}
		} else {
			return resolvedPolicy, nil
		}
	}
	return nil, errors.New("concurrent policy resolve")
}

func (s *LocalServices) tryResolve(ctx context.Context, policyMrn string, assetFilters []*Mquery) (*ResolvedPolicy, error) {
	logCtx := logger.FromContext(ctx)
	features := cnquery.GetFeatures(ctx)
	useV2Code := features.IsActive(cnquery.PiperCode)

	// phase 1: resolve asset filters and see if we can find a cached policy
	// trying first with all asset filters
	allFiltersChecksum, err := ChecksumAssetFilters(assetFilters)
	if err != nil {
		return nil, err
	}

	var rp *ResolvedPolicy
	if useV2Code {
		rp, err = s.DataLake.CachedResolvedPolicy(ctx, policyMrn, allFiltersChecksum, V2Code)
	} else {
		rp, err = s.DataLake.CachedResolvedPolicy(ctx, policyMrn, allFiltersChecksum, MassResolved)
	}
	if err != nil {
		return nil, err
	}
	if rp != nil {
		return rp, nil
	}

	// next we will try to only use the matching asset filters for the given policy...
	bundle, err := s.DataLake.GetValidatedBundle(ctx, policyMrn)
	if err != nil {
		return nil, err
	}
	bundleMap := bundle.ToMap()

	policyObj := bundleMap.Policies[policyMrn]
	matchingFilters, err := MatchingAssetFilters(policyMrn, assetFilters, policyObj)
	if err != nil {
		return nil, err
	}
	if len(matchingFilters) == 0 {
		return nil, newPolicyAssetMatchError(assetFilters, policyObj)
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
		if useV2Code {
			rp, err = s.DataLake.CachedResolvedPolicy(ctx, policyMrn, assetFiltersChecksum, V2Code)
		} else {
			rp, err = s.DataLake.CachedResolvedPolicy(ctx, policyMrn, assetFiltersChecksum, MassResolved)
		}
		if err != nil {
			return nil, err
		}
		if rp != nil {
			return rp, nil
		}
	}

	// intermission: prep for the other phases
	logCtx.Debug().
		Str("policy mrn", policyMrn).
		Interface("asset filters", matchingFilters).
		Msg("resolver> phase 1: no cached result, resolve the policy now")

	cache := &resolverCache{
		graphExecutionChecksum: policyObj.GraphExecutionChecksum,
		assetFiltersChecksum:   assetFiltersChecksum,
		assetFilters:           assetFiltersMap,
		executionQueries:       map[string]*ExecutionQuery{},
		dataQueries:            map[string]struct{}{},
		propQueries:            map[string]struct{}{},
		queries:                map[string]interface{}{},
		reportingJobsByQrID:    map[string]*ReportingJob{},
		reportingJobsByUUID:    map[string]*ReportingJob{},
		reportingJobsActive:    map[string]bool{},
		useV2Code:              useV2Code,
		bundleMap:              bundleMap,
	}

	rjUUID := cache.relativeChecksum(policyObj.GraphExecutionChecksum)

	reportingJob := &ReportingJob{
		Uuid:       rjUUID,
		QrId:       "root",
		Spec:       map[string]*ScoringSpec{},
		Datapoints: map[string]bool{},
	}

	cache.reportingJobsByUUID[reportingJob.Uuid] = reportingJob
	cache.reportingJobsByQrID[reportingJob.QrId] = reportingJob

	// phase 2: optimizations for assets
	// assets are always connected to a space, so figure out if a space policy exists
	// everything else in an asset can be aggregated into a shared policy

	// TODO: IMPLEMENT

	// phase 3: build the policy and scoring tree
	policyToJobsCache := &policyResolverCache{
		removedPolicies: map[string]struct{}{},
		removedQueries:  map[string]struct{}{},
		parentPolicies:  map[string]struct{}{},
		childPolicies:   map[string]struct{}{},
		childQueries:    map[string]struct{}{},
		global:          cache,
	}
	err = s.policyToJobs(ctx, policyMrn, reportingJob, policyToJobsCache)
	if err != nil {
		logCtx.Error().
			Err(err).
			Str("policy", policyMrn).
			Msg("resolver> phase 3: internal error, trying to turn policy mrn into jobs")
		return nil, err
	}
	logCtx.Debug().
		Str("policy", policyMrn).
		Msg("resolver> phase 3: turn policy into jobs [ok]")

	// phase 4: get all queries + assign them reporting jobs + update scoring jobs
	executionJob, collectorJob, err := s.jobsToQueries(ctx, useV2Code, policyMrn, cache)
	if err != nil {
		logCtx.Error().
			Err(err).
			Str("policy", policyMrn).
			Msg("resolver> phase 4: internal error, trying to turn policy jobs into queries")
		return nil, err
	}
	logCtx.Debug().
		Str("policy", policyMrn).
		Msg("resolver> phase 4: aggregate queries and jobs [ok]")

	// phase 5: refresh all checksums
	s.refreshChecksums(executionJob, collectorJob, useV2Code)

	// the final phases are done in the DataLake
	for _, rj := range collectorJob.ReportingJobs {
		rj.RefreshChecksum(useV2Code)
	}

	resolvedPolicy := ResolvedPolicy{
		GraphExecutionChecksum: policyObj.GraphExecutionChecksum,
		Filters:                matchingFilters,
		FiltersChecksum:        assetFiltersChecksum,
		ExecutionJob:           executionJob,
		CollectorJob:           collectorJob,
		ReportingJobUuid:       reportingJob.Uuid,
	}

	if useV2Code {
		err = s.DataLake.SetResolvedPolicy(ctx, policyMrn, &resolvedPolicy, V2Code, false)
	} else {
		err = s.DataLake.SetResolvedPolicy(ctx, policyMrn, &resolvedPolicy, MassResolved, false)
	}
	if err != nil {
		return nil, err
	}

	return &resolvedPolicy, nil
}

func newPolicyAssetMatchError(assetFilters []*Mquery, p *Policy) error {
	if len(assetFilters) == 0 {
		// send a proto error with details, so that the agent can render it properly
		msg := "asset does not match any of the activated policies"
		st := status.New(codes.InvalidArgument, msg)

		std, err := st.WithDetails(&errdetails.ErrorInfo{
			Domain: POLICY_SERVICE_NAME,
			Reason: "no-matching-policy", // TODO: make those error codes global for policy service
			Metadata: map[string]string{
				"policy": p.Mrn,
			},
		})
		if err != nil {
			log.Error().Err(err).Msg("could not send status with additional information")
			return st.Err()
		}
		return std.Err()
	}

	policyFilter := []string{}
	for k := range p.AssetFilters {
		policyFilter = append(policyFilter, strings.TrimSpace(k))
	}

	filters := make([]string, len(assetFilters))
	for i := range assetFilters {
		filters[i] = strings.TrimSpace(assetFilters[i].Query)
	}

	msg := "asset does not support any policy\nfilter supported by policies:\n" + strings.Join(policyFilter, ",\n") + "\n\nasset supports the following filters:\n" + strings.Join(filters, ",\n")
	return status.Error(codes.InvalidArgument, msg)
}

func (s *LocalServices) refreshChecksums(executionJob *ExecutionJob, collectorJob *CollectorJob, useV2Code bool) {
	// execution job
	{
		queryKeys := make([]string, len(executionJob.Queries))
		i := 0
		for k := range executionJob.Queries {
			queryKeys[i] = k
			i++
		}
		sort.Strings(queryKeys)

		checksum := checksums.New
		if useV2Code {
			checksum = checksum.Add("v2")
		}
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
			reportingJobKeys := make([]string, len(collectorJob.ReportingJobs))
			i := 0
			for k, rj := range collectorJob.ReportingJobs {
				rj.RefreshChecksum(useV2Code)
				reportingJobKeys[i] = k
				i++
			}
			sort.Strings(reportingJobKeys)

			for i := range reportingJobKeys {
				key := reportingJobKeys[i]
				checksum = checksum.Add(key)
				checksum = checksum.Add(collectorJob.ReportingJobs[key].Checksum)
			}
		}
		{
			datapointsKeys := make([]string, len(collectorJob.Datapoints))

			i := 0
			for k := range collectorJob.Datapoints {
				datapointsKeys[i] = k
				i++
			}
			sort.Strings(datapointsKeys)

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

func (s *LocalServices) policyToJobs(ctx context.Context, policyMrn string, ownerJob *ReportingJob, parentCache *policyResolverCache) error {
	ctx, span := tracer.Start(ctx, "resolver/policyToJobs")
	defer span.End()

	policyObj, ok := parentCache.global.bundleMap.Policies[policyMrn]
	if !ok || policyObj == nil {
		return errors.New("cannot find policy '" + policyMrn + "' while resolving")
	}

	if len(policyObj.Specs) == 0 {
		return nil
	}

	cache := parentCache.clone()
	cache.parentPolicies[policyMrn] = struct{}{}

	// properties to execution queries cache
	for k, v := range policyObj.Props {
		if v != "" {
			return errors.New("Cannot support property overwrites in resolver yet")
		}

		// we set it to nil here, as we don't know the mquery yet, it will be added in a later step
		cache.global.executionQueries[k] = nil
		cache.global.propQueries[k] = struct{}{}
	}

	// get a list of matching specs
	matchingSpecs := []*PolicySpec{}
	for i := range policyObj.Specs {
		spec := policyObj.Specs[i]

		if spec.AssetFilter == nil {
			matchingSpecs = append(matchingSpecs, spec)
			continue
		}

		checksum := spec.AssetFilter.CodeId
		if _, ok := cache.global.assetFilters[checksum]; ok {
			matchingSpecs = append(matchingSpecs, spec)
		}
	}

	// aggregate all removed policies and queries
	for i := range matchingSpecs {
		spec := matchingSpecs[i]
		for mrn, scoring := range spec.Policies {
			if scoring != nil && scoring.Action == QueryAction_DEACTIVATE {
				cache.removedPolicies[mrn] = struct{}{}
			}
		}
		for mrn, scoring := range spec.ScoringQueries {
			if scoring != nil && scoring.Action == QueryAction_DEACTIVATE {
				cache.removedQueries[mrn] = struct{}{}
			}
		}
		for mrn, action := range spec.DataQueries {
			if action == QueryAction_DEACTIVATE {
				cache.removedQueries[mrn] = struct{}{}
			}
		}
	}

	// resolve the rest
	var err error
	for i := range matchingSpecs {
		spec := matchingSpecs[i]
		if err = s.policyspecToJobs(ctx, policyMrn, spec, ownerJob, cache); err != nil {
			log.Error().Err(err).Msg("resolver> policyToJobs error")
			return err
		}
	}

	// finalize
	parentCache.addChildren(cache)

	return nil
}

func (s *LocalServices) policyspecToJobs(ctx context.Context, policyMrn string, spec *PolicySpec, ownerJob *ReportingJob, cache *policyResolverCache) error {
	ctx, span := tracer.Start(ctx, "resolver/policyspecToJobs")
	defer span.End()

	// include referenced policies
	for mrn, scoring := range spec.Policies {

		// ADD
		if scoring == nil || scoring.Action == QueryAction_ACTIVATE {
			if _, ok := cache.parentPolicies[mrn]; ok {
				return errors.New("trying to resolve policy spec twice, it is cyclical for MRN: " + mrn)
			}

			if _, ok := cache.removedPolicies[mrn]; ok {
				continue
			}

			// before adding any reporting job, make sure this policy actually works for
			// this set of asset filters
			policyObj, ok := cache.global.bundleMap.Policies[mrn]
			if !ok || policyObj == nil {
				return errors.New("cannot find policy '" + policyMrn + "' while resolving")
			}

			var found bool
			for checksum := range policyObj.AssetFilters {
				if _, ok := cache.global.assetFilters[checksum]; ok {
					found = true
					break
				}
			}
			if !found {
				continue
			}

			// the job itself is global to the resolution
			policyJob := cache.global.reportingJobsByQrID[mrn]
			if policyJob == nil {
				policyJob = &ReportingJob{
					QrId:          mrn,
					Uuid:          cache.global.relativeChecksum(mrn),
					Spec:          map[string]*ScoringSpec{},
					Datapoints:    map[string]bool{},
					ScoringSystem: policyObj.ScoringSystem,
				}
				cache.global.reportingJobsByQrID[mrn] = policyJob
				cache.global.reportingJobsByUUID[policyJob.Uuid] = policyJob
			}

			// local aspects for the resolved policy
			policyJob.Notify = append(policyJob.Notify, ownerJob.Uuid)
			ownerJob.Spec[policyJob.Uuid] = scoring
			cache.childPolicies[mrn] = struct{}{}

			if err := s.policyToJobs(ctx, mrn, policyJob, cache); err != nil {
				return err
			}

			continue
		}

		// MODIFY
		if scoring.Action == QueryAction_MODIFY {
			_, ok := cache.childPolicies[mrn]
			if !ok {
				cache.global.errors = append(cache.global.errors, &policyResolutionError{
					ID:       mrn,
					IsPolicy: true,
					Error:    "cannot modify policy, it doesn't exist",
				})
				continue
			}

			policyJob := cache.global.reportingJobsByQrID[mrn]
			for _, id := range policyJob.Notify {
				parentJob := cache.global.reportingJobsByUUID[id]
				if parentJob != nil {
					parentJob.Spec[policyJob.Uuid] = scoring
				}
			}
		}
	}

	// handle scoring queries
	for mrn, scoring := range spec.ScoringQueries {

		// ADD
		if scoring == nil || scoring.Action == QueryAction_ACTIVATE {
			if _, ok := cache.removedQueries[mrn]; ok {
				continue
			}

			// the job itself is global to the resolution
			queryJob := cache.global.reportingJobsByQrID[mrn]
			if queryJob == nil {
				queryJob = &ReportingJob{
					Uuid:       cache.global.relativeChecksum(mrn),
					QrId:       mrn,
					Spec:       map[string]*ScoringSpec{},
					Datapoints: map[string]bool{},
				}
				cache.global.reportingJobsByQrID[mrn] = queryJob
				cache.global.reportingJobsByUUID[queryJob.Uuid] = queryJob
			}

			// local aspects for the resolved policy
			queryJob.Notify = append(queryJob.Notify, ownerJob.Uuid)

			ownerJob.Spec[queryJob.Uuid] = scoring
			cache.childQueries[mrn] = struct{}{}

			// we set it to nil here, as we don't know the mquery yet, it will be added in a later step
			cache.global.executionQueries[mrn] = nil

			continue
		}

		// MODIFY
		if scoring.Action == QueryAction_MODIFY {
			_, ok := cache.childQueries[mrn]
			if !ok {
				cache.global.errors = append(cache.global.errors, &policyResolutionError{
					ID:       mrn,
					IsPolicy: true,
					Error:    "cannot modify query, it doesn't exist",
				})
				continue
			}

			queryJob := cache.global.reportingJobsByQrID[mrn]
			for _, id := range queryJob.Notify {
				parentJob := cache.global.reportingJobsByUUID[id]
				if parentJob != nil {
					parentJob.Spec[queryJob.Uuid] = scoring
				}
			}
		}
	}

	// handle data queries
	for mrn, action := range spec.DataQueries {
		// ADD
		if action == QueryAction_ACTIVATE {
			if _, ok := cache.removedQueries[mrn]; ok {
				continue
			}

			// the job itself is global to the resolution
			// note: the ReportingJob is only a placeholder and is replaced by individual query LLX checksum ReportingJobs
			queryJob := cache.global.reportingJobsByQrID[mrn]
			if queryJob == nil {
				queryJob = &ReportingJob{
					Uuid:       cache.global.relativeChecksum(mrn),
					QrId:       mrn,
					Spec:       map[string]*ScoringSpec{},
					Datapoints: map[string]bool{},
					IsData:     true,
				}
				cache.global.reportingJobsByQrID[mrn] = queryJob
				cache.global.reportingJobsByUUID[queryJob.Uuid] = queryJob
			}

			// local aspects for the resolved policy
			queryJob.Notify = append(queryJob.Notify, ownerJob.Uuid)

			ownerJob.Datapoints[queryJob.Uuid] = true
			cache.childQueries[mrn] = struct{}{}

			// we set it to nil here, as we don't know the mquery yet, it will be added in a later step
			cache.global.executionQueries[mrn] = nil
			cache.global.dataQueries[mrn] = struct{}{}

			continue
		}
	}

	return nil
}

func (s *LocalServices) jobsToQueries(ctx context.Context, useV2Code bool, policyMrn string, cache *resolverCache) (*ExecutionJob, *CollectorJob, error) {
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
	props := map[string]*llx.Primitive{}
	propsToChecksums := map[string]string{}

	// sort execution queries by property dependencies
	// FIXME: sort by internal dependencies of props as well
	executionMrns := make([]string, len(cache.executionQueries))
	var i int
	var j int = len(cache.executionQueries) - 1
	for mrn := range cache.executionQueries {
		if _, ok := cache.propQueries[mrn]; ok {
			executionMrns[i] = mrn
			i++
		} else {
			executionMrns[j] = mrn
			j--
		}
	}

	// fill in all reporting jobs. we will remove the data query jobs and replace
	// them with direct collections into their parent job later
	for _, rj := range cache.reportingJobsByQrID {
		collectorJob.ReportingJobs[rj.Uuid] = rj
	}

	// second, we want to inject all the real query checksums and connect them to
	// the uuids of data queries and reporting jobs
	// note: the queries are NOT defined yet, we only have MRNs at this stage
	for i := 0; i < len(executionMrns); i++ {
		mrn := executionMrns[i]

		var mquery *Mquery
		var err error

		if m, ok := cache.bundleMap.Queries[mrn]; ok {
			mquery = m
		} else {
			mquery, err = s.DataLake.ResolveQuery(ctx, mrn, cache.queries)
			if err != nil {
				return nil, nil, err
			}
		}

		codeID := mquery.CodeId

		if existing, ok := executionJob.Queries[codeID]; ok {
			logCtx.Debug().
				Str("codeID", mquery.CodeId).
				Str("existing", existing.Query).
				Str("new", mquery.Query).
				Msg("resolver> found duplicate query")
		}

		_, isDataQuery := cache.dataQueries[mrn]
		_, isPropQuery := cache.propQueries[mrn]

		executionQuery, dataChecksum, err := s.mquery2executionQuery(mquery, useV2Code, props, propsToChecksums, collectorJob, !(isDataQuery || isPropQuery))
		if err != nil {
			return nil, nil, errors.New("resolver> failed to compile query for ID " + mrn + ": " + err.Error())
		}

		if executionQuery == nil {
			// This case happens when we were able to compile with the
			// v2 compiler but not the v1 compiler. In such case, we
			// will expunge the query and reporting chain from the
			// resolved policy
			if rj, ok := cache.reportingJobsByQrID[mrn]; ok {
				delete(cache.reportingJobsByQrID, mrn)
				delete(cache.reportingJobsByUUID, rj.Uuid)
				delete(collectorJob.ReportingJobs, rj.Uuid)
				for _, parentID := range rj.Notify {
					if parentJob, ok := collectorJob.ReportingJobs[parentID]; ok {
						delete(parentJob.Spec, rj.Uuid)
					}
				}
			}

			continue
		}

		cache.executionQueries[mrn] = executionQuery
		executionJob.Queries[codeID] = executionQuery

		// (1) Property Queries handling
		// properties will be executed but not reported (for now)
		if isPropQuery {
			propName, err := Mrn2PropertyName(mrn)
			if err != nil {
				return nil, nil, errors.New("could not read property name from query mrn: " + mrn)
			}
			props[propName] = &llx.Primitive{Type: mquery.Type} // placeholder

			if dataChecksum == "" {
				return nil, nil, errors.New("property returns too many value, cannot determine entrypoint checksum: '" + mquery.Query + "'")
			}
			propsToChecksums[propName] = dataChecksum

			continue
		}

		// (2+3) Scoring+Data Queries handling
		rj, ok := cache.reportingJobsByQrID[mrn]
		if !ok {
			logCtx.Debug().
				Interface("reportingJobs", cache.reportingJobsByQrID).
				Str("query", mrn).
				Str("policy", policyMrn).
				Msg("resolver> phase 2: cannot find reporting job")
			return nil, nil, errors.New("cannot find reporting job for query " + mrn + " in policy " + policyMrn)
		}

		// (2) Scoring Queries handling
		if !isDataQuery {
			rj.QrId = mquery.CodeId

			if mquery.Severity != nil {
				for _, parentID := range rj.Notify {
					parentJob, ok := collectorJob.ReportingJobs[parentID]
					if !ok {
						return nil, nil, errors.New("failed to connect datapoint to reporting job")
					}
					spec := parentJob.Spec[rj.Uuid]
					if spec == nil {
						spec = &ScoringSpec{}
						parentJob.Spec[rj.Uuid] = spec
					}
					if spec.Severity == nil {
						spec.Severity = &SeverityValue{Value: mquery.Severity.Value}
					}
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
					return nil, nil, errors.New("failed to identiy datapoint in collectorjob")
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
					return nil, nil, errors.New("failed to identiy datapoint in collectorjob")
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

	return executionJob, collectorJob, nil
}
