package policy

import (
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
