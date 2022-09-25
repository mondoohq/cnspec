package inmemory

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnspec/policy"
)

// this section lists internal datastrcutures that map additional metadata
// with their proto counterparts

type wrapQuery struct {
	*policy.Mquery
	isScored bool
}

type wrapPolicy struct {
	*policy.Policy
	invalidated bool
	parents     map[string]struct{}
	children    map[string]struct{}
}

type wrapBundle struct {
	*policy.PolicyBundle
	graphContentChecksum string
	invalidated          bool
}

// GetQuery retrieves a given query
func (db *Db) GetQuery(ctx context.Context, mrn string) (*policy.Mquery, error) {
	q, ok := db.cache.Get(dbIDQuery + mrn)
	if !ok {
		return nil, errors.New("query '" + mrn + "' not found")
	}
	return (q.(wrapQuery)).Mquery, nil
}

// SetQuery stores a given query
// Note: the query must be defined, it cannot be nil
func (db *Db) SetQuery(ctx context.Context, mrn string, mquery *policy.Mquery, isScored bool) error {
	v := wrapQuery{mquery, isScored}
	ok := db.cache.Set(dbIDQuery+mrn, v, 1)
	if !ok {
		return errors.New("failed to save query '" + mrn + "' to cache")
	}
	return nil
}

// GetRawPolicy retrieves the policy without fixing any invalidations (fast)
func (db *Db) GetRawPolicy(ctx context.Context, mrn string) (*policy.Policy, error) {
	q, ok := db.cache.Get(dbIDPolicy + mrn)
	if !ok {
		return nil, errors.New("policy '" + mrn + "' not found")
	}
	return (q.(wrapPolicy)).Policy, nil
}

// GetPolicyFilters retrieves the list of asset filters for a policy (fast)
func (db *Db) GetPolicyFilters(ctx context.Context, mrn string) ([]*policy.Mquery, error) {
	r, err := db.GetRawPolicy(ctx, mrn)
	if err != nil {
		return nil, err
	}

	res := make([]*policy.Mquery, len(r.AssetFilters))
	var i int
	for _, v := range r.AssetFilters {
		res[i] = v
		i++
	}

	return res, nil
}

// SetPolicy stores a given policy in the data lake
func (db *Db) SetPolicy(ctx context.Context, policyObj *policy.Policy, filters []*policy.Mquery) error {
	_, err := db.setPolicy(ctx, policyObj, filters)
	return err
}

func (db *Db) setPolicy(ctx context.Context, policyObj *policy.Policy, filters []*policy.Mquery) (wrapPolicy, error) {
	var err error

	// we may use the cached parents if this policy already exists i.e. if it's
	// alrady referenced by others
	parents := map[string]struct{}{}

	x, exists := db.cache.Get(dbIDPolicy + policyObj.Mrn)
	if exists {
		existing := x.(wrapPolicy)

		parents = existing.parents

		if existing.LocalContentChecksum == policyObj.LocalContentChecksum &&
			existing.LocalExecutionChecksum == policyObj.LocalExecutionChecksum {
			if existing.GraphContentChecksum != policyObj.GraphContentChecksum ||
				existing.GraphExecutionChecksum != policyObj.GraphExecutionChecksum {
				return wrapPolicy{}, db.checkAndInvalidatePolicyBundle(ctx, &existing)
			}
			return wrapPolicy{}, nil
		}

		// fall through, re-create the policy
	}

	policyObj.AssetFilters = map[string]*policy.Mquery{}
	for i := range filters {
		filter := filters[i]
		policyObj.AssetFilters[filter.Mrn] = filter
		if err = db.SetQuery(ctx, filter.Mrn, filter, false); err != nil {
			return wrapPolicy{}, err
		}
	}

	children := policyObj.DependentPolicyMrns()
	for childMrn := range children {
		y, ok := db.cache.Get(dbIDPolicy + childMrn)
		if !ok {
			return wrapPolicy{}, errors.New("failed to get child policy '" + childMrn + "'")
		}
		child := y.(wrapPolicy)

		child.parents[policyObj.Mrn] = struct{}{}
		ok = db.cache.Set(dbIDPolicy+childMrn, child, 2)
		if !ok {
			return wrapPolicy{}, errors.New("failed to save child policy '" + childMrn + "' to cache")
		}
	}

	obj := wrapPolicy{
		Policy:      policyObj,
		invalidated: false,
		parents:     parents,
		children:    children,
	}

	ok := db.cache.Set(dbIDPolicy+policyObj.Mrn, obj, 2)
	if !ok {
		return wrapPolicy{}, errors.New("failed to save policy '" + policyObj.Mrn + "' to cache")
	}

	list, err := db.listPolicies()
	if err != nil {
		return wrapPolicy{}, err
	}

	list[policyObj.Mrn] = struct{}{}
	ok = db.cache.Set(dbIDListPolicies, list, 0)
	if !ok {
		return wrapPolicy{}, errors.New("failed to update policies list cache")
	}

	return obj, db.checkAndInvalidatePolicyBundle(ctx, &obj)
}

func (db *Db) checkAndInvalidatePolicyBundle(ctx context.Context, wrap *wrapPolicy) error {
	x, ok := db.cache.Get(dbIDBundle + wrap.Policy.Mrn)
	if !ok {
		return db.invalidatePolicyAndBundleAncestors(ctx, wrap)
	}

	bundleObj := x.(wrapBundle)
	if bundleObj.graphContentChecksum == wrap.Policy.GraphContentChecksum {
		log.Trace().Str("policy", wrap.Policy.Mrn).Msg("marketplace> policy cache is up-to-date")
		return nil
	}

	return db.invalidatePolicyAndBundleAncestors(ctx, wrap)
}

func (db *Db) invalidatePolicyAndBundleAncestors(ctx context.Context, wrap *wrapPolicy) error {
	mrn := wrap.Policy.Mrn
	log.Debug().Str("policy", mrn).Msg("invalidate policy cache")

	// invalidate the policy if its not invalided
	if wrap.invalidated == false {
		wrap.invalidated = true
		db.cache.Set(dbIDPolicy+mrn, *wrap, 2)
	}

	x, ok := db.cache.Get(dbIDBundle + mrn)
	if ok {
		wrapB := x.(wrapBundle)

		// invalidate the policy bundle if its not invalided
		if wrapB.invalidated == false {
			wrapB.invalidated = true
			db.cache.Set(dbIDBundle+mrn, wrapB, 3)
		}
	}

	// update all dependencies
	for parentMrn := range wrap.parents {
		x, ok := db.cache.Get(dbIDPolicy + parentMrn)
		if !ok {
			return errors.New("policy '" + mrn + "' not found")
		}
		parent := x.(wrapPolicy)

		if err := db.invalidatePolicyAndBundleAncestors(ctx, &parent); err != nil {
			return err
		}
	}

	return nil
}

// DeletePolicy removes a given policy
// Note: the MRN has to be valid
func (db *Db) DeletePolicy(ctx context.Context, mrn string) error {
	x, ok := db.cache.Get(dbIDPolicy + mrn)
	if !ok {
		return nil
	}
	wpolicy := x.(wrapPolicy)
	if len(wpolicy.parents) != 0 {
		return errors.New("cannot remove policy '" + mrn + "' it has " + strconv.Itoa(len(wpolicy.parents)) + " other policies attached")
	}

	errors := strings.Builder{}

	// list update
	list, err := db.listPolicies()
	if err != nil {
		return err
	}

	delete(list, mrn)
	ok = db.cache.Set(dbIDListPolicies, list, 0)
	if !ok {
		errors.WriteString("failed to update policies list cache")
	}

	// relationship updates
	for childMrn := range wpolicy.children {
		y, ok := db.cache.Get(dbIDPolicy + childMrn)
		if !ok {
			errors.WriteString("cannot find child policy '" + childMrn + "' while deleting '" + mrn + "'")
			continue
		}

		child := y.(wrapPolicy)
		delete(child.parents, mrn)
		db.cache.Set(dbIDPolicy+childMrn, child, 2)
	}

	for parentMrn := range wpolicy.parents {
		y, ok := db.cache.Get(dbIDPolicy + parentMrn)
		if !ok {
			errors.WriteString("cannot find child policy '" + parentMrn + "' while deleting '" + mrn + "'")
			continue
		}

		parent := y.(wrapPolicy)
		delete(parent.children, mrn)
		db.cache.Set(dbIDPolicy+parentMrn, parent, 2)
	}

	db.cache.Del(dbIDPolicy + mrn)

	return nil
}

func (db *Db) listPolicies() (map[string]struct{}, error) {
	x, ok := db.cache.Get(dbIDListPolicies)
	if ok {
		return x.(map[string]struct{}), nil
	}

	nu := map[string]struct{}{}
	ok = db.cache.Set(dbIDListPolicies, nu, 0)
	if !ok {
		return nil, errors.New("failed to initialize policies list cache")
	}
	return nu, nil
}

// GetValidatedBundle retrieves and if necessary updates the policy bundle
// Note: the checksum and graphchecksum of the policy must be computed to the right number
func (db *Db) GetValidatedBundle(ctx context.Context, mrn string) (*policy.PolicyBundle, error) {
	policyv, err := db.GetValidatedPolicy(ctx, mrn)
	if err != nil {
		return nil, err
	}

	y, ok := db.cache.Get(dbIDBundle + mrn)
	var wbundle wrapBundle
	if ok {
		wbundle = y.(wrapBundle)

		if !wbundle.invalidated && wbundle.graphContentChecksum == policyv.GraphContentChecksum {
			return wbundle.PolicyBundle, nil
		}
	}

	// these fields may be outdated in the data bundle:
	wbundle.graphContentChecksum = policyv.GraphContentChecksum

	bundle, err := db.services.ComputeBundle(ctx, policyv)
	if err != nil {
		return nil, errors.New("failed to compute policy bundle: " + err.Error())
	}

	wbundle.PolicyBundle = bundle
	wbundle.invalidated = false
	wbundle.graphContentChecksum = policyv.GraphContentChecksum

	if ok = db.cache.Set(dbIDBundle+mrn, wbundle, 3); !ok {
		return nil, errors.New("failed to save policy bundle '" + policyv.Mrn + "' to cache")
	}

	return bundle, nil
}

// GetValidatedPolicy retrieves and if necessary updates the policy
func (db *Db) GetValidatedPolicy(ctx context.Context, mrn string) (*policy.Policy, error) {
	q, ok := db.cache.Get(dbIDPolicy + mrn)
	if !ok {
		return nil, errors.New("policy '" + mrn + "' not found")
	}

	p := q.(wrapPolicy)
	if p.invalidated {
		err := db.fixInvalidatedPolicy(ctx, &p)
		if err != nil {
			return nil, err
		}
	}

	return p.Policy, nil
}

func (db *Db) fixInvalidatedPolicy(ctx context.Context, wrap *wrapPolicy) error {
	wrap.Policy.InvalidateGraphChecksums()
	wrap.Policy.UpdateChecksums(ctx,
		func(ctx context.Context, mrn string) (*policy.Policy, error) { return db.GetValidatedPolicy(ctx, mrn) },
		func(ctx context.Context, mrn string) (*policy.Mquery, error) { return db.GetQuery(ctx, mrn) },
		nil)

	ok := db.cache.Set(dbIDPolicy+wrap.Policy.Mrn, *wrap, 2)
	if !ok {
		return errors.New("failed to save policy '" + wrap.Policy.Mrn + "' to cache")
	}
	return nil
}
