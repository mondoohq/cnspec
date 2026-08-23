// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reportdoc

import (
	"sort"

	"go.mondoo.com/mql/providers-sdk/v1/upstream/mvd"
)

func VulnPackageKey(pkg *mvd.Package) string {
	return pkg.Name + "@" + pkg.Version
}

// AdvisoryPackages returns the packages of an advisory that the asset is actually
// affected by, in deterministic order.
func AdvisoryPackages(advisory *mvd.Advisory, affected, byName map[string]*mvd.Package) []*mvd.Package {
	var res []*mvd.Package
	seen := map[string]bool{}
	for _, pkg := range advisory.Affected {
		if pkg == nil {
			continue
		}
		match, ok := affected[VulnPackageKey(pkg)]
		if !ok {
			// the advisory may carry a different version than the installed one
			match, ok = byName[pkg.Name]
			if !ok {
				continue
			}
		}
		key := VulnPackageKey(match)
		if seen[key] {
			continue
		}
		seen[key] = true
		res = append(res, match)
	}
	sort.Slice(res, func(i, j int) bool { return VulnPackageKey(res[i]) < VulnPackageKey(res[j]) })
	return res
}

// AdvisoryCves returns the CVEs of an advisory in deterministic order.
func AdvisoryCves(advisory *mvd.Advisory) []*mvd.CVE {
	res := make([]*mvd.CVE, 0, len(advisory.Cves))
	for _, cve := range advisory.Cves {
		if cve != nil && cve.ID != "" {
			res = append(res, cve)
		}
	}
	sort.Slice(res, func(i, j int) bool { return res[i].ID < res[j].ID })
	return res
}
