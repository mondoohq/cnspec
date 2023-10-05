// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package inmemory

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v9/checksums"
	"go.mondoo.com/cnquery/v9/explorer"
	"go.mondoo.com/cnspec/v9/policy"
	"google.golang.org/protobuf/proto"
)

// this section lists internal data structures that map additional metadata
// with their proto counterparts

type wrapQuery struct {
	*explorer.Mquery
}

type wrapPolicy struct {
	*policy.Policy
	invalidated bool
	parents     map[string]struct{}
	children    map[string]struct{}
}

type wrapFramework struct {
	*policy.Framework
	invalidated bool
	parents     map[string]struct{}
	children    map[string]struct{}
}

type wrapBundle struct {
	*policy.Bundle
	graphContentChecksum string
	policyChecksum       string
	frameworkChecksum    string
	invalidated          bool
}

// QueryExists checks if the given MRN exists
func (db *Db) QueryExists(ctx context.Context, mrn string) (bool, error) {
	_, ok := db.cache.Get(dbIDQuery + mrn)
	return ok, nil
}

// PolicyExists checks if the given MRN exists
func (db *Db) PolicyExists(ctx context.Context, mrn string) (bool, error) {
	_, ok := db.cache.Get(dbIDPolicy + mrn)
	return ok, nil
}

// GetQuery retrieves a given query
func (db *Db) GetQuery(ctx context.Context, mrn string) (*explorer.Mquery, error) {
	q, ok := db.cache.Get(dbIDQuery + mrn)
	if !ok {
		return nil, errors.New("query '" + mrn + "' not found")
	}
	return (q.(wrapQuery)).Mquery, nil
}

// SetQuery stores a given query
// Note: the query must be defined, it cannot be nil
func (db *Db) SetQuery(ctx context.Context, mrn string, mquery *explorer.Mquery) error {
	v := wrapQuery{mquery}
	ok := db.cache.Set(dbIDQuery+mrn, v, 1)
	if !ok {
		return errors.New("failed to save query '" + mrn + "' to cache")
	}
	return nil
}

// GetProperty retrieves a given property
func (db *Db) GetProperty(ctx context.Context, mrn string) (*explorer.Property, error) {
	q, ok := db.cache.Get(dbIDProp + mrn)
	if !ok {
		return nil, errors.New("query '" + mrn + "' not found")
	}
	return proto.Clone(q.(*explorer.Property)).(*explorer.Property), nil
}

// SetProperty stores a given query
// Note: the query must be defined, it cannot be nil
func (db *Db) SetProperty(ctx context.Context, mrn string, prop *explorer.Property) error {
	ok := db.cache.Set(dbIDProp+mrn, prop, 1)
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
	return proto.Clone((q.(wrapPolicy)).Policy).(*policy.Policy), nil
}

// GetPolicyFilters retrieves the list of asset filters for a policy (fast)
func (db *Db) GetPolicyFilters(ctx context.Context, mrn string) ([]*explorer.Mquery, error) {
	r, err := db.GetRawPolicy(ctx, mrn)
	if err != nil {
		return nil, err
	}

	if r.ComputedFilters == nil || len(r.ComputedFilters.Items) == 0 {
		return nil, nil
	}

	res := make([]*explorer.Mquery, len(r.ComputedFilters.Items))
	var i int
	for _, v := range r.ComputedFilters.Items {
		res[i] = proto.Clone(v).(*explorer.Mquery)
		i++
	}

	return res, nil
}

// SetPolicy stores a given policy in the data lake
func (db *Db) SetPolicy(ctx context.Context, policyObj *policy.Policy, filters []*explorer.Mquery) error {
	_, err := db.setPolicy(ctx, policyObj, filters)
	return err
}

func (db *Db) setPolicy(ctx context.Context, policyObj *policy.Policy, filters []*explorer.Mquery) (wrapPolicy, error) {
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
				return wrapPolicy{}, db.checkAndInvalidatePolicyBundle(ctx, existing.Policy.Mrn, &existing, nil)
			}
			return wrapPolicy{}, nil
		}

		// fall through, re-create the policy
	}

	policyObj.ComputedFilters = &explorer.Filters{
		Items: make(map[string]*explorer.Mquery, len(filters)),
	}
	for i := range filters {
		filter := filters[i]
		policyObj.ComputedFilters.Items[filter.CodeId] = filter
		if err = db.SetQuery(ctx, filter.Mrn, filter); err != nil {
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

	return obj, db.checkAndInvalidatePolicyBundle(ctx, obj.Mrn, &obj, nil)
}

func (db *Db) checkAndInvalidatePolicyBundle(ctx context.Context, mrn string, policyw *wrapPolicy, frameworkw *wrapFramework) error {
	invalidatePolicy := policyw != nil
	invalidateFramework := frameworkw != nil

	x, ok := db.cache.Get(dbIDBundle + mrn)
	if ok {
		bundleObj := x.(wrapBundle)
		// We don't want to unnecessarily update things that are up to date
		if policyw != nil && bundleObj.policyChecksum == policyw.Policy.GraphContentChecksum {
			log.Trace().Str("policy", mrn).Msg("marketplace> policy cache is up-to-date")
			invalidatePolicy = false
		}
		if frameworkw != nil && bundleObj.frameworkChecksum == frameworkw.Framework.GraphContentChecksum {
			log.Trace().Str("framework", mrn).Msg("marketplace> framework cache is up-to-date")
			invalidateFramework = false
		}
	}

	if invalidatePolicy {
		if err := db.invalidatePolicyAndBundleAncestors(ctx, policyw); err != nil {
			return err
		}
	}

	if invalidateFramework {
		if err := db.invalidateFrameworkAndBundleAncestors(ctx, frameworkw); err != nil {
			return err
		}
	}

	return nil
}

func (db *Db) invalidatePolicyAndBundleAncestors(ctx context.Context, wrap *wrapPolicy) error {
	mrn := wrap.Policy.Mrn
	log.Debug().Str("policy", mrn).Msg("invalidate policy cache")

	// invalidate the policy if it isn't invalided
	if !wrap.invalidated {
		wrap.invalidated = true
		db.cache.Set(dbIDPolicy+mrn, *wrap, 2)
	}

	x, ok := db.cache.Get(dbIDBundle + mrn)
	if ok {
		wrapB := x.(wrapBundle)

		// invalidate the bundle if it isn't invalided
		if !wrapB.invalidated {
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

func (db *Db) invalidateFrameworkAndBundleAncestors(ctx context.Context, wrap *wrapFramework) error {
	mrn := wrap.Framework.Mrn
	log.Debug().Str("framework", mrn).Msg("invalidate framework cache")

	// invalidate the framework if it isn't invalided
	if !wrap.invalidated {
		wrap.invalidated = true
		db.cache.Set(dbIDFramework+mrn, *wrap, 2)
	}

	x, ok := db.cache.Get(dbIDBundle + mrn)
	if ok {
		wrapB := x.(wrapBundle)

		// invalidate the bundle if it isn't invalided
		if !wrapB.invalidated {
			wrapB.invalidated = true
			db.cache.Set(dbIDBundle+mrn, wrapB, 3)
		}
	}

	// update all dependencies
	for parentMrn := range wrap.parents {
		x, ok := db.cache.Get(dbIDFramework + parentMrn)
		if !ok {
			return errors.New("framework '" + mrn + "' not found")
		}
		parent := x.(wrapFramework)

		if err := db.invalidateFrameworkAndBundleAncestors(ctx, &parent); err != nil {
			return err
		}
	}

	return nil
}

// ListPolicies all policies for a given owner
// Note: Owner MRN is required
func (db *Db) ListPolicies(ctx context.Context, ownerMrn string, name string) ([]*policy.Policy, error) {
	mrns, err := db.listPolicies()
	if err != nil {
		return nil, err
	}

	res := []*policy.Policy{}
	for k := range mrns {
		policyObj, err := db.GetRawPolicy(ctx, k)
		if err != nil {
			return nil, err
		}

		if policyObj.OwnerMrn != ownerMrn {
			continue
		}

		res = append(res, policyObj)
	}

	return res, nil
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

// GetValidatedBundle retrieves and if necessary updates the policy bundle
// Note: the checksum and graphchecksum of the policy must be computed to the right number
func (db *Db) GetValidatedBundle(ctx context.Context, mrn string) (*policy.Bundle, error) {
	sum, err := db.EntityGraphContentChecksum(ctx, mrn)
	if err != nil {
		return nil, err
	}

	y, ok := db.cache.Get(dbIDBundle + mrn)
	var wbundle wrapBundle
	if ok {
		wbundle = y.(wrapBundle)

		if !wbundle.invalidated && wbundle.graphContentChecksum == sum {
			return wbundle.Bundle, nil
		}
	}

	policyv, err1 := db.GetValidatedPolicy(ctx, mrn)
	frameworkv, err2 := db.GetFramework(ctx, mrn)
	if err1 != nil && err2 != nil {
		return nil, errors.New("failed to retrieve validated contents for bundle: " + mrn)
	}

	bundle, err := db.services.ComputeBundle(ctx, policyv, frameworkv)
	if err != nil {
		return nil, errors.New("failed to compute policy bundle: " + err.Error())
	}

	wbundle.Bundle = bundle
	wbundle.invalidated = false
	wbundle.graphContentChecksum = sum
	if policyv != nil {
		wbundle.policyChecksum = policyv.GraphContentChecksum
	} else {
		wbundle.policyChecksum = ""
	}
	if frameworkv != nil {
		wbundle.frameworkChecksum = frameworkv.GraphContentChecksum
	} else {
		wbundle.frameworkChecksum = ""
	}

	if ok = db.cache.Set(dbIDBundle+mrn, wbundle, 3); !ok {
		return nil, errors.New("failed to save bundle for '" + mrn + "' to cache")
	}

	return proto.Clone(bundle).(*policy.Bundle), nil
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

	return proto.Clone(p.Policy).(*policy.Policy), nil
}

// entityGraphExecutionChecksum retrieves the execution checksum for a given entity.
// This is most useful when dealing with assets, which may have policies and
// frameworks assigned to them
func (db *Db) entityGraphExecutionChecksum(ctx context.Context, mrn string) (string, error) {
	var policyObj *policy.Policy
	var framework *policy.Framework
	var err error

	exist, err := db.PolicyExists(ctx, mrn)
	if err != nil {
		return "", err
	}
	if exist {
		policyObj, err = db.GetValidatedPolicy(ctx, mrn)
		if err != nil {
			return "", err
		}
	}

	exist, err = db.FrameworkExists(ctx, mrn)
	if err != nil {
		return "", err
	}
	if exist {
		framework, err = db.GetFramework(ctx, mrn)
		if err != nil {
			return "", err
		}
	}

	return policy.BundleExecutionChecksum(policyObj, framework), nil
}

// EntityGraphContentChecksum retrieves the content checksum for a given entity.
// This is most useful when dealing with assets, which may have policies and
// frameworks assigned to them
func (db *Db) EntityGraphContentChecksum(ctx context.Context, mrn string) (string, error) {
	res := checksums.New

	if policy, err := db.GetValidatedPolicy(ctx, mrn); err == nil {
		res = res.Add(policy.GraphContentChecksum)
	}

	if framework, err := db.GetFramework(ctx, mrn); err == nil {
		res = res.Add(framework.GraphContentChecksum)
	}

	if res == checksums.New {
		return "", errors.New("could not find: " + mrn)
	}
	return res.String(), nil
}

func (db *Db) fixInvalidatedPolicy(ctx context.Context, wrap *wrapPolicy) error {
	wrap.Policy.InvalidateGraphChecksums()
	wrap.Policy.UpdateChecksums(ctx,
		func(ctx context.Context, mrn string) (*policy.Policy, error) { return db.GetValidatedPolicy(ctx, mrn) },
		func(ctx context.Context, mrn string) (*explorer.Mquery, error) { return db.GetQuery(ctx, mrn) },
		nil,
		db.services.Schema(),
	)

	ok := db.cache.Set(dbIDPolicy+wrap.Policy.Mrn, *wrap, 2)
	if !ok {
		return errors.New("failed to save policy '" + wrap.Policy.Mrn + "' to cache")
	}
	return nil
}

// SetFramework stores a given framework in the data lake. Note: it does not
// store any framework maps, there is a separate call for them.
func (db *Db) SetFramework(ctx context.Context, framework *policy.Framework) error {
	_, err := db.setFramework(ctx, framework)
	return err
}

func (db *Db) setFramework(ctx context.Context, framework *policy.Framework) (wrapFramework, error) {
	// TODO: add updates to frameworks with existing parents and children;
	// see SetPolicy for how it's done there
	x := wrapFramework{
		Framework:   framework,
		invalidated: false,
		parents:     map[string]struct{}{},
		children:    map[string]struct{}{},
	}

	ok := db.cache.Set(dbIDFramework+framework.Mrn, x, 1)
	if !ok {
		return x, errors.New("failed to store framework in cache DB")
	}
	return x, nil
}

// SetFrameworkMaps stores a list of framework maps connecting frameworks
// to policies.
func (db *Db) SetFrameworkMaps(ctx context.Context, ownerFramework string, maps []*policy.FrameworkMap) error {
	raw, ok := db.cache.Get(dbIDFrameworkMap + ownerFramework)
	var storedMaps map[string]*policy.FrameworkMap
	if !ok {
		storedMaps = make(map[string]*policy.FrameworkMap)
	} else {
		storedMaps = raw.(map[string]*policy.FrameworkMap)
	}

	for i := range maps {
		cur := maps[i]
		storedMaps[cur.Mrn] = cur
	}

	ok = db.cache.Set(dbIDFrameworkMap+ownerFramework, storedMaps, 1)
	if !ok {
		return errors.New("failed to store framework map in cache DB")
	}
	return nil
}

func (db *Db) FrameworkExists(ctx context.Context, mrn string) (bool, error) {
	_, ok := db.cache.Get(dbIDFramework + mrn)
	return ok, nil
}

// GetFramework retrieves a framework from storage. This does not include
// framework maps!
func (db *Db) GetFramework(ctx context.Context, mrn string) (*policy.Framework, error) {
	raw, ok := db.cache.Get(dbIDFramework + mrn)
	if !ok {
		return nil, errors.New("framework '" + mrn + "' not found")
	}

	return proto.Clone(raw.(wrapFramework).Framework).(*policy.Framework), nil
}

// GetFrameworkMaps retrieves a set of framework maps for a given framework
// from the data lake. This doesn't include controls metadata. If there
// are no framework maps for this MRN, it returns nil (no error).
func (db *Db) GetFrameworkMaps(ctx context.Context, frameworkMrn string) ([]*policy.FrameworkMap, error) {
	raw, ok := db.cache.Get(dbIDFrameworkMap + frameworkMrn)
	if !ok {
		return nil, nil
	}

	storedMaps := raw.(map[string]*policy.FrameworkMap)
	res := make([]*policy.FrameworkMap, len(storedMaps))
	var i int
	for _, v := range storedMaps {
		res[i] = v
		i++
	}

	return res, nil
}
