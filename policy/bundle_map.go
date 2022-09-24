package policy

import (
	"sort"
	"strings"

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
func (p *PolicyBundleMap) ToList() *PolicyBundle {
	res := PolicyBundle{
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
