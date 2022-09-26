package inmemory

import (
	"context"
	"errors"

	"github.com/gogo/status"
	"github.com/rs/zerolog/log"
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
