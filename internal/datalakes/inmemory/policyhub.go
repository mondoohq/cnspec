package inmemory

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"go.mondoo.com/cnspec/policy"
)

// this section lists internal datastrcutures that map additional metadata
// with their proto counterparts

type wrapPolicy struct {
	*policy.Policy
	invalidated bool
	parents     map[string]struct{}
	children    map[string]struct{}
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
