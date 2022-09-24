package inmemory

import (
	"context"
	"errors"

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
