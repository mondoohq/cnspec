// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package bundle

import "sort"

// Sorts the queries, policies and queries' variants in the bundle.
func (p *Bundle) SortContents() {
	sort.SliceStable(p.Queries, func(i, j int) bool {
		if p.Queries[i].Mrn == "" || p.Queries[j].Mrn == "" {
			return p.Queries[i].Uid < p.Queries[j].Uid
		}
		return p.Queries[i].Mrn < p.Queries[j].Mrn
	})

	sort.SliceStable(p.Policies, func(i, j int) bool {
		if p.Policies[i].Mrn == "" || p.Policies[j].Mrn == "" {
			return p.Policies[i].Uid < p.Policies[j].Uid
		}
		return p.Policies[i].Mrn < p.Policies[j].Mrn
	})

	for _, q := range p.Queries {
		sort.SliceStable(q.Variants, func(i, j int) bool {
			if q.Variants[i].Mrn == "" || q.Variants[j].Mrn == "" {
				return q.Variants[i].Uid < q.Variants[j].Uid
			}
			return q.Variants[i].Mrn < q.Variants[j].Mrn
		})
	}
	for _, pl := range p.Policies {
		for _, g := range pl.Groups {
			for _, q := range g.Queries {
				sort.SliceStable(q.Variants, func(i, j int) bool {
					if q.Variants[i].Mrn == "" || q.Variants[j].Mrn == "" {
						return q.Variants[i].Uid < q.Variants[j].Uid
					}
					return q.Variants[i].Mrn < q.Variants[j].Mrn
				})
			}
			for _, c := range g.Checks {
				sort.SliceStable(c.Variants, func(i, j int) bool {
					if c.Variants[i].Mrn == "" || c.Variants[j].Mrn == "" {
						return c.Variants[i].Uid < c.Variants[j].Uid
					}
					return c.Variants[i].Mrn < c.Variants[j].Mrn
				})
			}
		}
	}
}
