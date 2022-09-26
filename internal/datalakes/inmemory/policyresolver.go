package inmemory

import (
	"context"
	"errors"

	"github.com/gogo/status"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/types"
	"go.mondoo.com/cnspec/policy"
	"google.golang.org/grpc/codes"
)

// MutatePolicy modifies a policy. If it does not find the policy, and if the
// caller chooses to, it will treat the MRN as an asset and create it + its policy
func (db *Db) MutatePolicy(ctx context.Context, mutation *policy.PolicyMutationDelta, createIfMissing bool) (*policy.Policy, error) {
	mrn := mutation.PolicyMrn

	policyw, err := db.ensurePolicy(ctx, mrn, createIfMissing)
	if err != nil {
		return nil, err
	}

	if len(policyw.Policy.Specs) == 0 {
		log.Error().Str("policy", mrn).Msg("distributor> failed to modify policy, it has no specs")
		return nil, errors.New("cannot modify policy, it has no specs (invalid state)")
	}

	spec := policyw.Policy.Specs[0]
	changed := false

	for policyMrn, delta := range mutation.PolicyDeltas {
		switch delta.Action {
		case policy.PolicyDelta_ADD:
			if _, ok := spec.Policies[policyMrn]; ok {
				continue
			}

			// FIXME: upstream policies

			x, ok := db.cache.Get(dbIDPolicy + policyMrn)
			if !ok {
				return nil, errors.New("cannot find child policy '" + policyMrn + "' when trying to assign it")
			}
			childw := x.(wrapPolicy)

			spec.Policies[policyMrn] = nil
			policyw.children[policyMrn] = struct{}{}
			childw.parents[mrn] = struct{}{}
			if ok := db.cache.Set(dbIDPolicy+policyMrn, childw, 2); !ok {
				return nil, errors.New("failed to update child-parent relationship for policy '" + policyMrn + "'")
			}

			changed = true

		case policy.PolicyDelta_DELETE:
			x, ok := db.cache.Get(dbIDPolicy + policyMrn)
			if !ok {
				return nil, errors.New("cannot find child policy '" + policyMrn + "' when trying to assign it")
			}
			childw := x.(wrapPolicy)

			delete(spec.Policies, policyMrn)
			delete(policyw.children, policyMrn)
			delete(childw.parents, mrn)
			if ok := db.cache.Set(dbIDPolicy+policyMrn, childw, 2); !ok {
				return nil, errors.New("failed to update child-parent relationship for policy '" + policyMrn + "'")
			}

			changed = true

		default:
			return nil, status.Error(codes.InvalidArgument, "unsupported change  is required")
		}
	}

	if !changed {
		return policyw.Policy, nil
	}

	err = db.refreshAssetFilters(ctx, &policyw)
	if err != nil {
		return nil, err
	}

	policyw.Policy.InvalidateExecutionChecksums()
	err = policyw.Policy.UpdateChecksums(ctx,
		func(ctx context.Context, mrn string) (*policy.Policy, error) { return db.GetValidatedPolicy(ctx, mrn) },
		func(ctx context.Context, mrn string) (*policy.Mquery, error) { return db.GetQuery(ctx, mrn) },
		nil,
	)
	if err != nil {
		return nil, err
	}

	ok := db.cache.Set(dbIDPolicy+mrn, policyw, 2)
	if !ok {
		return nil, errors.New("")
	}

	err = db.checkAndInvalidatePolicyBundle(ctx, &policyw)
	if err != nil {
		return nil, err
	}

	err = db.refreshDependentAssetFilters(ctx, policyw)
	if err != nil {
		return nil, err
	}

	return policyw.Policy, nil
}

func (db *Db) ensurePolicy(ctx context.Context, mrn string, createIfMissing bool) (wrapPolicy, error) {
	x, ok := db.cache.Get(dbIDPolicy + mrn)
	if ok {
		return x.(wrapPolicy), nil
	}

	if !createIfMissing {
		return wrapPolicy{}, errors.New("failed to modify policy '" + mrn + "', could not find it")
	}

	_, policyw, err := db.ensureAsset(ctx, mrn)
	return policyw, err
}

func (db *Db) refreshAssetFilters(ctx context.Context, policyw *wrapPolicy) error {
	policyObj := policyw.Policy
	filters, err := policyObj.ComputeAssetFilters(ctx,
		func(ctx context.Context, mrn string) (*policy.Policy, error) { return db.GetRawPolicy(ctx, mrn) },
		false,
	)
	if err != nil {
		return errors.New("failed to compute asset filters: " + err.Error())
	}

	policyObj.AssetFilters = map[string]*policy.Mquery{}
	for i := range filters {
		filter := filters[i]
		policyObj.AssetFilters[filter.CodeId] = filter
	}

	depMrns := policyObj.DependentPolicyMrns()
	for mrn := range depMrns {
		dep, err := db.GetRawPolicy(ctx, mrn)
		if err != nil {
			return errors.New("failed to get dependent policy '" + mrn + "': " + err.Error())
		}

		for k, v := range dep.AssetFilters {
			policyObj.AssetFilters[k] = v
		}
	}

	ok := db.cache.Set(dbIDPolicy+policyObj.Mrn, *policyw, 2)
	if !ok {
		return errors.New("failed to update policy asset filters for '" + policyObj.Mrn + "'")
	}

	return nil
}

func (db *Db) refreshDependentAssetFilters(ctx context.Context, startPolicy wrapPolicy) error {
	needsUpdate := map[string]wrapPolicy{}

	for k := range startPolicy.parents {
		x, ok := db.cache.Get(dbIDPolicy + k)
		if !ok {
			return errors.New("failed to get parent policy '" + k + "'")
		}
		needsUpdate[k] = x.(wrapPolicy)
	}

	for len(needsUpdate) > 0 {
		for k, policyw := range needsUpdate {
			err := db.refreshAssetFilters(ctx, &policyw)
			if err != nil {
				return err
			}

			policyw.Policy.InvalidateGraphChecksums()
			err = policyw.Policy.UpdateChecksums(ctx,
				func(ctx context.Context, mrn string) (*policy.Policy, error) { return db.GetValidatedPolicy(ctx, mrn) },
				func(ctx context.Context, mrn string) (*policy.Mquery, error) { return db.GetQuery(ctx, mrn) },
				nil,
			)
			if err != nil {
				return err
			}

			db.cache.Set(dbIDPolicy+policyw.Policy.Mrn, policyw, 2)
			err = db.checkAndInvalidatePolicyBundle(ctx, &policyw)
			if err != nil {
				return err
			}

			for k := range policyw.parents {
				x, ok := db.cache.Get(dbIDPolicy + k)
				if !ok {
					return errors.New("failed to get parent policy '" + k + "'")
				}
				needsUpdate[k] = x.(wrapPolicy)
			}

			delete(needsUpdate, k)
		}
	}

	return nil
}

// GetReport retrieves all scores and data for a given asset
func (db *Db) GetReport(ctx context.Context, assetMrn string, qrID string) (*policy.Report, error) {
	emptyReport := &policy.Report{
		EntityMrn:  assetMrn,
		ScoringMrn: qrID,
	}

	score, err := db.GetScore(ctx, assetMrn, qrID)
	if err != nil {
		return emptyReport, nil
	}

	x, ok := db.cache.Get(dbIDAsset + assetMrn)
	if !ok {
		return nil, errors.New("cannot find asset '" + assetMrn + "'")
	}

	assetw := x.(wrapAsset)
	resolvedPolicy := assetw.ResolvedPolicy
	resolvedPolicyVersion := assetw.resolvedPolicyVersion

	includedScores := map[string]struct{}{}
	for _, job := range resolvedPolicy.CollectorJob.ReportingJobs {
		qrid := job.QrId
		if qrid == "root" {
			qrid = assetMrn
		}

		includedScores[qrid] = struct{}{}
	}
	scoreQrIDs := make([]string, len(includedScores))
	i := 0
	for k := range includedScores {
		scoreQrIDs[i] = k
		i++
	}

	scores, err := db.GetScores(ctx, assetMrn, scoreQrIDs)
	if err != nil {
		log.Error().
			Err(err).
			Str("entity", assetMrn).
			Msg("reportsstore> could not fetch scores for asset")
		return nil, err
	}

	datapoints := resolvedPolicy.CollectorJob.Datapoints
	fields := make(map[string]types.Type, len(datapoints))
	for field, info := range datapoints {
		fields[field] = types.Type(info.Type)
	}

	data, err := db.GetData(ctx, assetMrn, fields)
	if err != nil {
		log.Error().
			Err(err).
			Str("entity", assetMrn).
			Msg("reportsstore> could not fetch data for asset")
		return nil, err
	}

	res := policy.Report{
		EntityMrn:             assetMrn,
		ScoringMrn:            qrID,
		Score:                 &score,
		Scores:                scores,
		Data:                  data,
		ResolvedPolicyVersion: resolvedPolicyVersion,
	}

	return &res, nil
}

// GetScore retrieves one score for an asset
func (db *Db) GetScore(ctx context.Context, assetMrn, scoreID string) (policy.Score, error) {
	x, ok := db.cache.Get(dbIDScore + assetMrn + "\x00" + scoreID)
	if !ok {
		return policy.Score{}, errors.New("cannot find score")
	}
	return x.(policy.Score), nil
}

// GetScores retrieves a map of score for an asset
func (db *Db) GetScores(ctx context.Context, assetMrn string, qrIDs []string) (map[string]*policy.Score, error) {
	res := make(map[string]*policy.Score, len(qrIDs))

	for i := range qrIDs {
		qrID := qrIDs[i]

		x, ok := db.cache.Get(dbIDScore + assetMrn + "\x00" + qrID)
		if !ok {
			return nil, errors.New("score for asset '" + assetMrn + "' with ID '" + qrID + "' not found")
		}

		score := x.(policy.Score)
		res[qrID] = &score
	}

	return res, nil
}

// GetData retrieves a map of requested data fields for an asset
func (db *Db) GetData(ctx context.Context, assetMrn string, fields map[string]types.Type) (map[string]*llx.Result, error) {
	res := make(map[string]*llx.Result, len(fields))

	for checksum := range fields {
		x, ok := db.cache.Get(dbIDData + assetMrn + "\x00" + checksum)
		if !ok {
			return nil, errors.New("failed to get data for asset '" + assetMrn + "' and checksum '" + checksum + "'")
		}

		if x == nil {
			res[checksum] = nil
		} else {
			res[checksum] = x.(*llx.Result)
		}
	}

	return res, nil
}

// CachedResolvedPolicy returns the resolved policy if it exists
func (db *Db) CachedResolvedPolicy(ctx context.Context, policyMrn string, assetFilterChecksum string, version policy.ResolvedPolicyVersion) (*policy.ResolvedPolicy, error) {
	policyObj, err := db.GetValidatedPolicy(ctx, policyMrn)
	if err != nil {
		return nil, errors.New("cannot find policy for resolver: '" + policyMrn + "'")
	}

	res, ok := db.resolvedPolicyCache.Get(dbIDResolvedPolicy + policyObj.GraphExecutionChecksum + "\x00" + assetFilterChecksum)
	if !ok {
		return nil, nil
	}

	return res, nil
}

// ResolveQuery looks up a given query and caches it for later access (optional)
func (db *Db) ResolveQuery(ctx context.Context, mrn string, cache map[string]interface{}) (*policy.Mquery, error) {
	x, ok := db.cache.Get(dbIDQuery + mrn)
	if !ok {
		return nil, errors.New("failed to get query '" + mrn + "'")
	}

	res := x.(wrapQuery)
	return res.Mquery, nil
}

// SetResolvedPolicy to the data store; cached indicates if it was cached from
// upstream, thus preventing any attempts of resolving it in the client
func (db *Db) SetResolvedPolicy(ctx context.Context, mrn string, resolvedPolicy *policy.ResolvedPolicy, version policy.ResolvedPolicyVersion, cached bool) error {
	ok := db.resolvedPolicyCache.Set(dbIDResolvedPolicy+resolvedPolicy.GraphExecutionChecksum+"\x00"+resolvedPolicy.FiltersChecksum, resolvedPolicy)
	if !ok {
		return errors.New("failed to save resolved policy '" + mrn + "'")
	}

	if cached {
		x, ok := db.cache.Get(dbIDPolicy + mrn)
		if !ok {
			return errors.New("failed to save resolved policy as cached entry in this client, cannot find its parent policy locally: '" + mrn + "'")
		}

		policyw := x.(wrapPolicy)
		policyw.Policy.GraphExecutionChecksum = resolvedPolicy.GraphExecutionChecksum
		policyw.invalidated = false

		ok = db.cache.Set(dbIDPolicy+mrn, policyw, 1)
		if !ok {
			return errors.New("failed to save resolved policy as cached entryin this client, failed to update parent policy locally: '" + mrn + "'")
		}
	}

	return nil
}
