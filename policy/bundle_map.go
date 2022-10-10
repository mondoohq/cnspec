package policy

import (
	"context"
	"sort"
	"strings"

	"github.com/hashicorp/go-version"
	"github.com/pkg/errors"
	"go.mondoo.com/cnquery/llx"
)

// PolicyBundleMap is a PolicyBundle with easier access to policies and queries
type PolicyBundleMap struct {
	OwnerMrn string                     `json:"owner_mrn,omitempty"`
	Policies map[string]*Policy         `json:"policies,omitempty"`
	Queries  map[string]*Mquery         `json:"queries,omitempty"`
	Props    map[string]*Mquery         `json:"props,omitempty"`
	Code     map[string]*llx.CodeBundle `json:"code,omitempty"`
	Library  Library                    `json:"library,omitempty"`
}

// NewPolicyBundleMap creates a new empty initialized map
// dataLake (optional) connects an additional data layer which may provide queries/policies
func NewPolicyBundleMap(ownerMrn string) *PolicyBundleMap {
	return &PolicyBundleMap{
		OwnerMrn: ownerMrn,
		Policies: make(map[string]*Policy),
		Queries:  make(map[string]*Mquery),
		Props:    make(map[string]*Mquery),
		Code:     make(map[string]*llx.CodeBundle),
	}
}

// SelectPolicies selects the policies by name from the list given.
// If a given name does not exist in the map, an error will be thrown.
// The final map will only have the given policies selected. This call does not
// remove queries (at this time).
func (b *PolicyBundleMap) SelectPolicies(names []string) error {
	if len(names) == 0 {
		return nil
	}

	filters := map[string]struct{}{}
	var missing []string

	for i := range names {
		name := names[i]
		if _, ok := b.Policies[name]; !ok {
			missing = append(missing, name)
			continue
		}
		filters[name] = struct{}{}
	}

	if len(missing) != 0 {
		return errors.New("failed to find the following policies: " + strings.Join(missing, ", "))
	}

	for name := range b.Policies {
		if _, ok := filters[name]; !ok {
			delete(b.Policies, name)
		}
	}

	return nil
}

// ToList converts the map to a regular bundle
func (p *PolicyBundleMap) ToList() *Bundle {
	res := Bundle{
		OwnerMrn: p.OwnerMrn,
	}
	var i int
	var ids []string

	// policies
	ids = make([]string, len(p.Policies))
	i = 0
	for k := range p.Policies {
		ids[i] = k
		i++
	}
	sort.Strings(ids)

	res.Policies = make([]*Policy, len(p.Policies))
	for i := range ids {
		res.Policies[i] = p.Policies[ids[i]]
	}

	// queries
	ids = make([]string, len(p.Queries))
	i = 0
	for k := range p.Queries {
		ids[i] = k
		i++
	}
	sort.Strings(ids)

	res.Queries = make([]*Mquery, len(p.Queries))
	for i := range ids {
		res.Queries[i] = p.Queries[ids[i]]
	}

	// props
	ids = make([]string, len(p.Props))
	i = 0
	for k := range p.Props {
		ids[i] = k
		i++
	}
	sort.Strings(ids)

	res.Props = make([]*Mquery, len(p.Props))
	for i := range ids {
		res.Props[i] = p.Props[ids[i]]
	}

	return &res
}

// PoliciesSortedByDependency sorts policies by their dependencies
// note: the MRN field must be set and dependencies in specs must be specified by MRN
func (p *PolicyBundleMap) PoliciesSortedByDependency() ([]*Policy, error) {
	indexer := map[string]struct{}{}
	var res []*Policy

	for i := range p.Policies {
		policy := p.Policies[i]

		if _, ok := indexer[policy.Mrn]; ok {
			continue
		}

		depRes, err := sortPolicies(policy, p, indexer)
		if err != nil {
			return nil, err
		}

		res = append(res, depRes...)
	}

	return res, nil
}

func sortPolicies(p *Policy, bundle *PolicyBundleMap, indexer map[string]struct{}) ([]*Policy, error) {
	var res []*Policy
	indexer[p.Mrn] = struct{}{}

	for i := range p.Specs {
		spec := p.Specs[i]
		for mrn := range spec.Policies {
			// we only do very cursory sanity checking
			if mrn == "" {
				return nil, errors.New("failed to sort policies: dependency MRN is empty")
			}

			if _, ok := indexer[mrn]; ok {
				continue
			}

			dep, ok := bundle.Policies[mrn]
			if !ok {
				// ignore, since we are only looking to sort the policies of the map
				continue
			}

			depRes, err := sortPolicies(dep, bundle, indexer)
			if err != nil {
				return nil, err
			}

			res = append(res, depRes...)
		}
	}

	res = append(res, p)
	return res, nil
}

// ValidatePolicy against the given bundle
func (p *PolicyBundleMap) ValidatePolicy(ctx context.Context, policy *Policy) error {
	if err := IsPolicyMrn(policy.Mrn); err != nil {
		return err
	}

	for i := range policy.Specs {
		if err := p.validateSpec(ctx, policy.Specs[i]); err != nil {
			return err
		}
	}

	// semver checks are a bit optional
	if policy.Version != "" {
		_, err := version.NewSemver(policy.Version)
		if err != nil {
			return errors.New("policy '" + policy.Mrn + "' version '" + policy.Version + "' is not a valid semver version")
		}
	}

	return nil
}

func (p *PolicyBundleMap) validateSpec(ctx context.Context, spec *PolicySpec) error {
	if spec == nil {
		return errors.New("spec cannot be nil")
	}

	var err error

	if spec.AssetFilter != nil {
		// since asset filters are run beforehand and don't make it into the report
		// we don't store their code bundles separately
		if _, err := spec.AssetFilter.RefreshAsAssetFilter(""); err != nil {
			return err
		}
	}

	for mrn, spec := range spec.ScoringQueries {
		if err = p.queryExists(ctx, mrn); err != nil {
			return err
		}

		if spec != nil && spec.Action == QueryAction_UNSPECIFIED {
			return errors.New("received a query spec without an action: " + mrn)
		}
	}

	for mrn, action := range spec.DataQueries {
		if err = p.queryExists(ctx, mrn); err != nil {
			return err
		}

		if action == QueryAction_UNSPECIFIED {
			// in this case users don't have to specify an action and we will
			// interpret it as their intention to add the query
			spec.DataQueries[mrn] = QueryAction_ACTIVATE
		}
	}

	for mrn := range spec.Policies {
		if _, ok := p.Policies[mrn]; ok {
			continue
		}

		if p.Library != nil {
			x, err := p.Library.PolicyExists(ctx, mrn)
			if err != nil {
				return err
			}
			if !x {
				return errors.New("cannot find policy '" + mrn + "'")
			}

			p.Policies[mrn] = nil
			continue
		}

		return errors.New("cannot find policy '" + mrn + "'")
	}

	return nil
}

func (p *PolicyBundleMap) queryExists(ctx context.Context, mrn string) error {
	if _, ok := p.Queries[mrn]; ok {
		return nil
	}

	if p.Library != nil {
		x, err := p.Library.QueryExists(ctx, mrn)
		if err != nil {
			return err
		}

		if !x {
			return errors.New("cannot find query '" + mrn + "'")
		}

		p.Queries[mrn] = nil
		return nil
	}

	return errors.New("cannot find query '" + mrn + "'")
}

// QueryMap extracts all the queries from the policy bundle map
func (bundle *PolicyBundleMap) QueryMap() map[string]*Mquery {
	res := make(map[string]*Mquery, len(bundle.Queries))
	for _, v := range bundle.Queries {
		res[v.CodeId] = v
	}
	return res
}
