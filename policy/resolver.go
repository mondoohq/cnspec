// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"math/rand"
	"sort"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/fasthash/fnv1a"
	"go.mondoo.com/cnquery/v11/checksums"
	"go.mondoo.com/cnquery/v11/explorer"
	resources "go.mondoo.com/cnquery/v11/explorer/resources"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/logger"
	"go.mondoo.com/cnquery/v11/mqlc"
	"go.mondoo.com/cnquery/v11/mrn"
	"go.mondoo.com/cnquery/v11/utils/sortx"
	"go.mondoo.com/ranger-rpc/codes"
	"go.mondoo.com/ranger-rpc/status"
)

const (
	POLICY_SERVICE_NAME = "policy.api.mondoo.com"
	// This is used to change the checksum of the resolved policy when we want it to be recalculated
	// This can be updated, e.g., when we change how the report jobs are generated
	// A change of this string will force an update of all the stored resolved policies
	RESOLVER_VERSION = "v2024-12-02"
)

type AssetMutation struct {
	AssetMrn            string
	PolicyMrns          []string
	FrameworkMrns       []string
	Action              explorer.Action
	PolicyScoringSystem explorer.ScoringSystem
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

	err := s.DataLake.MutateAssignments(ctx, &AssetMutation{
		AssetMrn:            assignment.AssetMrn,
		PolicyMrns:          assignment.PolicyMrns,
		FrameworkMrns:       assignment.FrameworkMrns,
		Action:              assignment.Action,
		PolicyScoringSystem: assignment.ScoringSystem,
	}, true)
	return globalEmpty, err
}

// Unassign a policy to an asset
func (s *LocalServices) Unassign(ctx context.Context, assignment *PolicyAssignment) (*Empty, error) {
	if len(assignment.PolicyMrns)+len(assignment.FrameworkMrns) == 0 {
		return nil, status.Error(codes.InvalidArgument, "a policy or framework mrn is required")
	}

	// all remote, call upstream
	if s.Upstream != nil && !s.Incognito {
		return s.Upstream.PolicyResolver.Unassign(ctx, assignment)
	}

	err := s.DataLake.MutateAssignments(ctx, &AssetMutation{
		AssetMrn:      assignment.AssetMrn,
		PolicyMrns:    assignment.PolicyMrns,
		FrameworkMrns: assignment.FrameworkMrns,
		Action:        explorer.Action_DEACTIVATE,
	}, true)
	return globalEmpty, err
}

func (s *LocalServices) SetProps(ctx context.Context, req *explorer.PropsReq) (*explorer.Empty, error) {
	// validate that the queries compile and fill in checksums
	conf := s.NewCompilerConfig()
	for i := range req.Props {
		prop := req.Props[i]
		// set props is used for both setting and unsetting props
		if prop.Mql == "" {
			continue
		}
		code, err := prop.RefreshChecksumAndType(conf)
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

	_, err = s.DataLake.UpdateRisks(ctx, req.AssetMrn, req.Risks)
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

func (s *LocalServices) GetResourcesData(ctx context.Context, req *resources.EntityResourcesReq) (*resources.EntityResourcesRes, error) {
	res, err := s.DataLake.GetResources(ctx, req.EntityMrn, req.Resources)
	return &resources.EntityResourcesRes{
		Resources: res,
		EntityMrn: req.EntityMrn,
	}, err
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
	codeIdToMrn          map[string][]string
	riskFactors          map[string]*RiskFactor
	// assigned queries, listed by their UUID (i.e. policy context)
	executionQueries map[string]*ExecutionQuery
	dataQueries      map[string]struct{}
	queriesByMsum    map[string]*explorer.Mquery // Msum == Mquery.Checksum
	riskMrns         map[string]*explorer.Mquery
	riskInfos        map[string]*RiskFactor
	propsCache       explorer.PropsCache

	reportingJobsByUUID   map[string]*ReportingJob
	reportingJobsByMsum   map[string][]*ReportingJob // Msum == Mquery.Checksum, i.e. only reporting jobs for mqueries
	reportingJobsByCodeId map[string][]*ReportingJob // CodeId == Mquery.CodeId
	reportingJobsActive   map[string]bool
	errors                []*policyResolutionError
	bundleMap             *PolicyBundleMap

	compilerConfig mqlc.CompilerConfig
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
	now := time.Now()
	conf := s.NewCompilerConfig()

	// phase 1: resolve asset filters and see if we can find a cached policy
	// trying first with all asset filters
	allFiltersChecksum, err := ChecksumAssetFilters(assetFilters, conf)
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

	policyObj := bundleMap.Policies[bundleMrn]

	matchingFilters, err := MatchingAssetFilters(bundleMrn, assetFilters, policyObj)
	if err != nil {
		return nil, err
	}
	if len(matchingFilters) == 0 {
		return nil, explorer.NewAssetMatchError(bundleMrn, "policies", "no-matching-policy", assetFilters, policyObj.ComputedFilters)
	}

	resolvedPolicy, err := buildResolvedPolicy(ctx, bundleMrn, bundle, matchingFilters, now, conf)
	if err != nil {
		return nil, err
	}

	err = s.DataLake.SetResolvedPolicy(ctx, bundleMrn, resolvedPolicy, V2Code, false)
	if err != nil {
		return nil, err
	}

	return resolvedPolicy, nil

}

func refreshChecksums(executionJob *ExecutionJob, collectorJob *CollectorJob) {
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

func connectDatapointsToReportingJob(query *ExecutionQuery, job *ReportingJob, datapoints map[string]*DataQueryInfo) error {
	for _, dpId := range query.Datapoints {
		datapointInfo, ok := datapoints[dpId]
		if !ok {
			return errors.New("failed to identity datapoint in collectorjob")
		}

		datapointInfo.Notify = append(datapointInfo.Notify, job.Uuid)
		if job.Datapoints == nil {
			job.Datapoints = map[string]bool{}
		}
		job.Datapoints[dpId] = true
	}
	return nil
}

type queryLike interface {
	Compile(props map[string]*llx.Primitive, conf mqlc.CompilerConfig) (*llx.CodeBundle, error)
	GetChecksum() string
	GetMql() string
}

func mquery2executionQuery(query queryLike, props map[string]*llx.Primitive, propsToChecksums map[string]string, collectorJob *CollectorJob, isScoring bool, conf mqlc.CompilerConfig) (*ExecutionQuery, string, error) {
	bundle, err := query.Compile(props, conf)
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

// CheckValidUntil returns whether the given time is laying in the future or not.
// Special case is a unix ts of 0, which is treated as forever.
func CheckValidUntil(validUntil int64, mrn string) bool {
	stillIgnore := false
	// empty validUntil means ignore forever
	if validUntil == 0 {
		stillIgnore = true
		log.Debug().Str("mrn", mrn).Msg("control is ignored forever")
	} else {
		validTime := time.Unix(validUntil, 0)
		if validTime.After(time.Now()) {
			stillIgnore = true
			log.Debug().Str("mrn", mrn).Msg("is ignored for now because of validUntil timestamp")
		}
	}
	return stillIgnore
}
