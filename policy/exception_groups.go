// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"go.mondoo.com/mql/checksums"
)

// withExceptionGroups returns a copy of bundle whose asset policy carries the
// given exception groups (ADR-0006). The groups go on the asset policy, the
// root of the resolution, because that is where the resolver gathers overrides
// that apply to every check below it; an upstream applies the exceptions it
// holds the same way. The resolver then does the rest: a DISABLE group swaps
// the check for the disabled placeholder so it does not run, an IGNORED group
// keeps it from scoring, and a group past its valid.until is skipped.
//
// The asset policy's execution checksum takes the groups in, so a resolved
// policy that applies exceptions never carries the checksum of one that does
// not.
func withExceptionGroups(bundle *Bundle, assetMrn string, groups []*PolicyGroup) *Bundle {
	res := bundle.CloneVT()
	for _, p := range res.Policies {
		if p.Mrn != assetMrn {
			continue
		}

		sum := checksums.New.Add(p.GraphExecutionChecksum).Add("exceptions")
		for _, g := range groups {
			// Deterministic: an exception group carries no maps.
			raw, err := g.MarshalVT()
			if err == nil {
				sum = sum.Add(string(raw))
			}
			p.Groups = append(p.Groups, g.CloneVT())
		}
		p.GraphExecutionChecksum = sum.String()
		return res
	}
	return res
}
