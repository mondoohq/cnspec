package inmemory

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/explorer"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/types"
	"go.mondoo.com/cnspec/policy"
)

// return true if the framework was changed
func (db *Db) mutateFramework(ctx context.Context, mrn string, actions map[string]explorer.AssignmentDelta_Action, createIfMissing bool) (wrapFramework, bool, error) {
	frameworkw, err := db.ensureFramework(ctx, mrn, createIfMissing)
	if err != nil {
		return frameworkw, false, err
	}

	deps := make(map[string]*policy.FrameworkRef, len(frameworkw.Dependencies))
	for i := range frameworkw.Dependencies {
		dep := frameworkw.Dependencies[i]
		deps[dep.Mrn] = dep
	}

	changed := false

	for frameworkMrn, action := range actions {
		switch action {
		case explorer.AssignmentDelta_DELETE:
			x, ok := db.cache.Get(dbIDFramework + frameworkMrn)
			if !ok {
				return frameworkw, false, errors.New("cannot find child framework '" + frameworkMrn + "' when trying to unassign it")
			}
			childw := x.(wrapPolicy)

			delete(deps, frameworkMrn)
			delete(frameworkw.children, frameworkMrn)
			delete(childw.parents, mrn)
			if ok := db.cache.Set(dbIDFramework+frameworkMrn, childw, 2); !ok {
				return frameworkw, false, errors.New("failed to update child-parent relationship for framework '" + frameworkMrn + "'")
			}

			changed = true

		default:
			x, ok := db.cache.Get(dbIDFramework + frameworkMrn)
			if !ok {
				return frameworkw, false, errors.New("cannot find child framework '" + frameworkMrn + "' when trying to assign it")
			}
			childw := x.(wrapFramework)

			deps[frameworkMrn] = &policy.FrameworkRef{
				Mrn: frameworkMrn,
			}
			frameworkw.children[frameworkMrn] = struct{}{}
			childw.parents[mrn] = struct{}{}
			if ok := db.cache.Set(dbIDFramework+frameworkMrn, childw, 2); !ok {
				return frameworkw, false, errors.New("failed to update child-parent relationship for framework '" + frameworkMrn + "'")
			}

			changed = true
		}
	}

	if !changed {
		return frameworkw, false, nil
	}

	// since the map was only used for faster create/delete, we now have to translate it back
	frameworkw.Dependencies = make([]*policy.FrameworkRef, len(deps))
	i := 0
	for _, v := range deps {
		frameworkw.Dependencies[i] = v
		i++
	}

	ok := db.cache.Set(dbIDFramework+mrn, frameworkw, 2)
	if !ok {
		return frameworkw, true, errors.New("failed to store updated framework in cache")
	}

	frameworkw.Framework.ClearExecutionChecksums()
	err = frameworkw.Framework.UpdateChecksums(ctx,
		func(ctx context.Context, mrn string) (*policy.Framework, error) { return db.GetFramework(ctx, mrn) },
		nil,
	)
	if err != nil {
		return frameworkw, true, err
	}

	return frameworkw, true, nil
}

func (db *Db) mutatePolicy(ctx context.Context, mrn string, actions map[string]explorer.AssignmentDelta_Action, createIfMissing bool) (wrapPolicy, bool, error) {
	policyw, err := db.ensurePolicy(ctx, mrn, createIfMissing)
	if err != nil {
		return policyw, false, err
	}

	if len(policyw.Policy.Groups) == 0 {
		log.Error().Str("policy", mrn).Msg("resolver.db> failed to modify policy, it has no specs")
		return policyw, false, errors.New("cannot modify policy, it has no specs (invalid state)")
	}

	group := policyw.Policy.Groups[0]
	changed := false

	// prepare a map for easier processing
	policies := map[string]*policy.PolicyRef{}
	for i := range group.Policies {
		cur := group.Policies[i]
		policies[cur.Mrn] = cur
	}

	for policyMrn, action := range actions {
		switch action {
		case explorer.AssignmentDelta_DELETE:
			x, ok := db.cache.Get(dbIDPolicy + policyMrn)
			if !ok {
				return policyw, false, errors.New("cannot find child policy '" + policyMrn + "' when trying to unassign it")
			}
			childw := x.(wrapPolicy)

			delete(policies, policyMrn)
			delete(policyw.children, policyMrn)
			delete(childw.parents, mrn)
			if ok := db.cache.Set(dbIDPolicy+policyMrn, childw, 2); !ok {
				return policyw, false, errors.New("failed to update child-parent relationship for policy '" + policyMrn + "'")
			}

			changed = true

		default:
			if _, ok := policies[policyMrn]; ok {
				continue
			}

			// FIXME: upstream policies

			x, ok := db.cache.Get(dbIDPolicy + policyMrn)
			if !ok {
				return policyw, false, errors.New("cannot find child policy '" + policyMrn + "' when trying to assign it")
			}
			childw := x.(wrapPolicy)

			policies[policyMrn] = &policy.PolicyRef{
				Mrn: policyMrn,
			}
			policyw.children[policyMrn] = struct{}{}
			childw.parents[mrn] = struct{}{}
			if ok := db.cache.Set(dbIDPolicy+policyMrn, childw, 2); !ok {
				return policyw, false, errors.New("failed to update child-parent relationship for policy '" + policyMrn + "'")
			}

			changed = true
		}
	}

	if !changed {
		return policyw, false, nil
	}

	// since the map was only used for faster create/delete, we now have to translate it back
	group.Policies = make([]*policy.PolicyRef, len(policies))
	i := 0
	for _, v := range policies {
		group.Policies[i] = v
		i++
	}

	err = db.refreshAssetFilters(ctx, &policyw)
	if err != nil {
		return policyw, true, err
	}

	policyw.Policy.InvalidateExecutionChecksums()
	err = policyw.Policy.UpdateChecksums(ctx,
		func(ctx context.Context, mrn string) (*policy.Policy, error) { return db.GetValidatedPolicy(ctx, mrn) },
		func(ctx context.Context, mrn string) (*explorer.Mquery, error) { return db.GetQuery(ctx, mrn) },
		nil,
	)
	if err != nil {
		return policyw, true, err
	}

	ok := db.cache.Set(dbIDPolicy+mrn, policyw, 2)
	if !ok {
		return policyw, true, errors.New("failed to store updated policy in cache")
	}

	return policyw, true, nil
}

func (db *Db) MutateAssignments(ctx context.Context, mutation *policy.AssetMutation, createIfMissing bool) error {
	targetMRN := mutation.AssetMrn
	frameworkw, changedFramework, err := db.mutateFramework(ctx, targetMRN, mutation.FrameworkActions, createIfMissing)
	if err != nil {
		return err
	}

	policyw, changedPolicy, err := db.mutatePolicy(ctx, targetMRN, mutation.PolicyActions, createIfMissing)
	if err != nil {
		return err
	}

	if !changedFramework && !changedPolicy {
		return nil
	}

	err = db.checkAndInvalidatePolicyBundle(ctx, targetMRN, &policyw, &frameworkw)
	if err != nil {
		return err
	}

	err = db.refreshDependentAssetFilters(ctx, policyw)
	if err != nil {
		return err
	}

	return nil
}

// DeprecatedV8_MutatePolicy modifies a policy. If it does not find the policy, and if the
// caller chooses to, it will treat the MRN as an asset and create it + its policy
// Replaced by MutateAssignments
func (db *Db) DeprecatedV8_MutatePolicy(ctx context.Context, mutation *policy.PolicyMutationDelta, createIfMissing bool) (*policy.Policy, error) {
	targetMRN := mutation.PolicyMrn

	actions := make(map[string]explorer.AssignmentDelta_Action, len(mutation.PolicyDeltas))
	for k, v := range mutation.PolicyDeltas {
		if v.Action == policy.PolicyDelta_DELETE {
			actions[k] = explorer.AssignmentDelta_DELETE
		} else {
			actions[k] = explorer.AssignmentDelta_ADD
		}
	}

	policyw, changed, err := db.mutatePolicy(ctx, targetMRN, actions, createIfMissing)
	if err != nil {
		return nil, err
	}
	if !changed {
		return policyw.Policy, nil
	}

	err = db.checkAndInvalidatePolicyBundle(ctx, targetMRN, &policyw, nil)
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

	_, policyw, _, err := db.ensureAsset(ctx, mrn)
	return policyw, err
}

func (db *Db) ensureFramework(ctx context.Context, mrn string, createIfMissing bool) (wrapFramework, error) {
	x, ok := db.cache.Get(dbIDFramework + mrn)
	if ok {
		return x.(wrapFramework), nil
	}

	if !createIfMissing {
		return wrapFramework{}, errors.New("failed to modify framework '" + mrn + "', could not find it")
	}

	_, _, frameworkw, err := db.ensureAsset(ctx, mrn)
	return frameworkw, err
}

func (db *Db) refreshAssetFilters(ctx context.Context, policyw *wrapPolicy) error {
	policyObj := policyw.Policy
	filters, err := policyObj.ComputeAssetFilters(ctx,
		func(ctx context.Context, mrn string) (*policy.Policy, error) { return db.GetRawPolicy(ctx, mrn) },
		func(ctx context.Context, mrn string) (*explorer.Mquery, error) { return db.GetQuery(ctx, mrn) },
		false,
	)
	if err != nil {
		return errors.New("failed to compute asset filters: " + err.Error())
	}

	policyObj.ComputedFilters = &explorer.Filters{
		Items: map[string]*explorer.Mquery{},
	}
	for i := range filters {
		filter := filters[i]
		policyObj.ComputedFilters.Items[filter.CodeId] = filter
	}

	depMrns := policyObj.DependentPolicyMrns()
	for mrn := range depMrns {
		dep, err := db.GetRawPolicy(ctx, mrn)
		if err != nil {
			return errors.New("failed to get dependent policy '" + mrn + "': " + err.Error())
		}

		if dep.ComputedFilters == nil {
			continue
		}

		for k, v := range dep.ComputedFilters.Items {
			policyObj.ComputedFilters.Items[k] = v
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
				func(ctx context.Context, mrn string) (*explorer.Mquery, error) { return db.GetQuery(ctx, mrn) },
				nil,
			)
			if err != nil {
				return err
			}

			db.cache.Set(dbIDPolicy+policyw.Policy.Mrn, policyw, 2)
			err = db.checkAndInvalidatePolicyBundle(ctx, policyw.Policy.Mrn, &policyw, nil)
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
			Msg("resolver.db> could not fetch scores for asset")
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
			Msg("resolver.db> could not fetch data for asset")
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

// GetResolvedPolicy returns the resolved policy for a given asset
func (db *Db) GetResolvedPolicy(ctx context.Context, assetMrn string) (*policy.ResolvedPolicy, error) {
	x, ok := db.cache.Get(dbIDAsset + assetMrn)
	if !ok {
		return nil, errors.New("cannot find asset '" + assetMrn + "'")
	}

	assetw := x.(wrapAsset)

	if assetw.ResolvedPolicy == nil {
		return nil, errors.New("cannot find resolved policy for asset '" + assetMrn + "'")
	}

	return assetw.ResolvedPolicy, nil
}

// CachedResolvedPolicy returns the resolved policy if it exists
func (db *Db) CachedResolvedPolicy(ctx context.Context, entityMrn string, assetFilterChecksum string, version policy.ResolvedPolicyVersion) (*policy.ResolvedPolicy, error) {
	sum, err := db.entityGraphExecutionChecksum(ctx, entityMrn)
	if err != nil {
		return nil, errors.New("cannot determine execution checksum for: '" + entityMrn + "'")
	}

	res, ok := db.resolvedPolicyCache.Get(dbIDResolvedPolicy + sum + "\x00" + assetFilterChecksum)
	if !ok {
		return nil, nil
	}

	return res, nil
}

// ResolveQuery looks up a given query and caches it for later access (optional)
func (db *Db) ResolveQuery(ctx context.Context, mrn string) (*explorer.Mquery, error) {
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

// SetAssetResolvedPolicy sets and initialized all fields for an asset's resolved policy
func (db *Db) SetAssetResolvedPolicy(ctx context.Context, assetMrn string, resolvedPolicy *policy.ResolvedPolicy, version policy.ResolvedPolicyVersion) error {
	x, ok := db.cache.Get(dbIDAsset + assetMrn)
	if !ok {
		return errors.New("cannot find asset '" + assetMrn + "'")
	}

	assetw := x.(wrapAsset)

	if assetw.ResolvedPolicy != nil && assetw.ResolvedPolicy.GraphExecutionChecksum == resolvedPolicy.GraphExecutionChecksum && assetw.resolvedPolicyVersion == string(version) {
		log.Debug().
			Str("asset", assetMrn).
			Msg("resolverj.db> asset resolved policy is already cached (and unchanged)")
		return nil
	}

	assetw.ResolvedPolicy = resolvedPolicy
	assetw.resolvedPolicyVersion = string(version)

	var err error
	collectorJob := resolvedPolicy.CollectorJob
	for checksum, info := range collectorJob.Datapoints {
		err = db.initDataValue(ctx, assetMrn, checksum, types.Type(info.Type))
		if err != nil {
			log.Error().
				Err(err).
				Str("asset", assetMrn).
				Str("query checksum", checksum).
				Msg("resolver.db> failed to set asset resolved policy, failed to initialize data value")
			return errors.New("failed to create asset scoring job (failed to init data)")
		}
	}

	reportingJobs := collectorJob.ReportingJobs
	for _, job := range reportingJobs {
		qrid := job.QrId
		if qrid == "root" {
			qrid = assetMrn
		}

		err = db.initEmptyScore(ctx, assetMrn, qrid)
		if err != nil {
			log.Error().
				Err(err).
				Str("asset", assetMrn).
				Str("score qrID", qrid).
				Msg("resolver.db> failed to set asset resolved policy, failed to initialize score")
			return errors.New("failed to create asset scoring job (failed to init score)")
		}
	}

	ok = db.cache.Set(dbIDAsset+assetMrn, assetw, 1)
	if !ok {
		return errors.New("failed to save resolved policy for asset '" + assetMrn + "'")
	}

	return nil
}

func (db *Db) initDataValue(ctx context.Context, assetMrn string, checksum string, typ types.Type) error {
	id := dbIDData + assetMrn + "\x00" + checksum
	_, ok := db.cache.Get(id)
	if ok {
		return nil
	}

	ok = db.cache.Set(id, nil, 1)
	if !ok {
		return errors.New("failed to initialize data value for asset '" + assetMrn + "' with checksum '" + checksum + "'")
	}
	return nil
}

func (db *Db) initEmptyScore(ctx context.Context, assetMrn string, qrid string) error {
	id := dbIDScore + assetMrn + "\x00" + qrid

	ok := db.cache.Set(id, policy.Score{}, 1)
	if !ok {
		return errors.New("failed to initialize score for asset '" + assetMrn + "' with qrID '" + qrid + "'")
	}
	return nil
}

// GetCollectorJob returns the collector job for a given asset
func (db *Db) GetCollectorJob(ctx context.Context, assetMrn string) (*policy.CollectorJob, error) {
	x, ok := db.cache.Get(dbIDAsset + assetMrn)
	if !ok {
		return nil, errors.New("cannot find asset '" + assetMrn + "'")
	}

	assetw := x.(wrapAsset)

	if assetw.ResolvedPolicy == nil {
		return nil, errors.New("cannot find resolved policy for asset '" + assetMrn + "'")
	}
	if assetw.ResolvedPolicy.CollectorJob == nil {
		return nil, errors.New("cannot find collectorJob for asset '" + assetMrn + "'")
	}

	return assetw.ResolvedPolicy.CollectorJob, nil
}

var errTypesDontMatch = errors.New("types don't match")

// UpdateData sets the list of data value for a given asset and returns a list of updated IDs
func (db *Db) UpdateData(ctx context.Context, assetMrn string, data map[string]*llx.Result) (map[string]types.Type, error) {
	collectorJob, err := db.GetCollectorJob(ctx, assetMrn)
	if err != nil {
		return nil, errors.New("cannot find collectorJob to store data: " + err.Error())
	}

	res := make(map[string]types.Type, len(data))
	var errList error
	for dpChecksum, val := range data {
		info, ok := collectorJob.Datapoints[dpChecksum]
		if !ok {
			return nil, errors.New("cannot find this datapoint to store values: " + dpChecksum)
		}

		if val.Data != nil && !val.Data.IsNil() && val.Data.Type != "" &&
			val.Data.Type != info.Type && types.Type(info.Type) != types.Unset {
			log.Warn().
				Str("checksum", dpChecksum).
				Str("asset", assetMrn).
				Interface("data", val.Data).
				Str("expected", types.Type(info.Type).Label()).
				Str("received", types.Type(val.Data.Type).Label()).
				Msg("resolver.db> failed to store data, types don't match")

			errList = multierror.Append(errList, fmt.Errorf("failed to store data for %q, %w: expected %s, got %s",
				dpChecksum, errTypesDontMatch, types.Type(info.Type).Label(), types.Type(val.Data.Type).Label()))

			continue
		}

		err := db.setDatum(ctx, assetMrn, dpChecksum, val)
		if err != nil {
			errList = multierror.Append(errList, err)
			continue
		}

		// TODO: we don't know which data was updated and which wasn't yet, so
		// we currently always notify...
		res[dpChecksum] = types.Type(info.Type)
	}

	if errList != nil {
		return nil, errList
	}

	return res, nil
}

func (db *Db) setDatum(ctx context.Context, assetMrn string, checksum string, value *llx.Result) error {
	id := dbIDData + assetMrn + "\x00" + checksum
	ok := db.cache.Set(id, value, 1)
	if !ok {
		return errors.New("failed to save asset data for asset '" + assetMrn + "' and checksum '" + checksum + "'")
	}
	return nil
}

// UpdateScores sets the given scores and returns true if any were updated
func (db *Db) UpdateScores(ctx context.Context, assetMrn string, scores []*policy.Score) (map[string]struct{}, error) {
	updated := map[string]struct{}{}
	now := db.nowProvider().Unix()

	for i := range scores {
		score := scores[i]
		ok, err := db.updateScore(ctx, assetMrn, score, now)
		if err != nil {
			return nil, err
		}

		if ok {
			updated[score.QrId] = struct{}{}
		}
	}

	return updated, nil
}

// set one score and return true if it was updated
func (db *Db) updateScore(ctx context.Context, assetMrn string, score *policy.Score, now int64) (bool, error) {
	org, err := db.GetScore(ctx, assetMrn, score.QrId)
	if err == nil &&
		org.Value == score.Value &&
		org.Type == score.Type &&
		org.DataCompletion == score.DataCompletion &&
		org.DataTotal == score.DataTotal &&
		org.ScoreCompletion == score.ScoreCompletion &&
		org.Weight == score.Weight {
		return false, nil
	}

	// if this is the first time saving the score
	if err != nil || (org.ScoreCompletion == 0 && score.Type == policy.ScoreType_Result) {
		score.ValueModifiedTime = now
		if score.Value == 100 || score.ScoreCompletion < 100 {
			score.FailureTime = 0
		} else {
			score.FailureTime = now
		}
	} else if (org.Value != score.Value || org.ScoreCompletion == 0) && score.Type == policy.ScoreType_Result {
		score.ValueModifiedTime = now
		// we are failing from 100 => something else
		if org.Value == 100 {
			score.FailureTime = now
		} else {
			score.FailureTime = org.FailureTime
		}
	} else {
		score.ValueModifiedTime = org.ValueModifiedTime
		score.FailureTime = org.FailureTime
	}

	ok := db.cache.Set(dbIDScore+assetMrn+"\x00"+score.QrId, *score, 1)
	if !ok {
		return false, errors.New("failed to set score for asset '" + assetMrn + "' with ID '" + score.QrId + "'")
	}

	log.Debug().
		Str("asset", assetMrn).
		Str("query", score.QrId).
		Str("type", score.TypeLabel()).
		Int("value", int(score.Value)).
		Int("score-completion", int(score.ScoreCompletion)).
		Int("data-completion", int(score.DataCompletion)).
		Int("data-total", int(score.DataTotal)).
		Str("error_msg", score.Message).
		Msg("resolver.db> update score")
	return true, nil
}

// SetProps will override properties for a given entity (asset, space, org)
func (db *Db) SetProps(ctx context.Context, req *explorer.PropsReq) error {
	policyw, err := db.ensurePolicy(ctx, req.EntityMrn, false)
	if err != nil {
		return err
	}

	allProps := make(map[string]*explorer.Property, len(policyw.Props))
	for i := range policyw.Props {
		cur := policyw.Props[i]
		if cur.Mrn != "" {
			allProps[cur.Mrn] = cur
		}
		if cur.Uid != "" {
			allProps[cur.Uid] = cur
		}
	}

	for i := range req.Props {
		cur := req.Props[i]
		id := cur.Mrn
		if id == "" {
			id = cur.Uid
		}
		if id == "" {
			return errors.New("cannot set property without MRN: " + cur.Mql)
		}

		if cur.Mql == "" {
			delete(allProps, id)
		}
		allProps[id] = cur
	}

	policyw.Props = []*explorer.Property{}
	for k, v := range allProps {
		// since props can be in the list with both UIDs and MRNs, in the case
		// where a property sets both we want to ignore one entry to avoid duplicates
		if v.Mrn != "" && v.Uid != "" && k == v.Uid {
			continue
		}
		policyw.Props = append(policyw.Props, v)
	}

	policyw.Policy.InvalidateExecutionChecksums()
	err = policyw.Policy.UpdateChecksums(ctx,
		func(ctx context.Context, mrn string) (*policy.Policy, error) { return db.GetValidatedPolicy(ctx, mrn) },
		func(ctx context.Context, mrn string) (*explorer.Mquery, error) { return db.GetQuery(ctx, mrn) },
		nil,
	)
	if err != nil {
		return err
	}

	ok := db.cache.Set(dbIDPolicy+req.EntityMrn, policyw, 2)
	if !ok {
		return errors.New("")
	}

	err = db.checkAndInvalidatePolicyBundle(ctx, policyw.Policy.Mrn, &policyw, nil)
	if err != nil {
		return err
	}

	return nil
}
