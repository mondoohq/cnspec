package vuln_reporter

import (
	"fmt"

	"go.mondoo.com/cnquery/resources/packs/core/vadvisor"
	"go.mondoo.com/cnquery/resources/packs/core/versions/generic"
)

type Entry struct {
	Pkg           *vadvisor.Package
	Advisories    []*vadvisor.Advisory
	MaxVulnerable string
}

func pkgID(pkg *vadvisor.Package) string {
	return fmt.Sprintf("%s/%s/%s/%s", pkg.Namespace, pkg.Name, pkg.Version, pkg.Arch)
}

// AdvisoriesByPackage iterates over advisories and store all advisories per package
// The entry.Pkg.Fixed will state the latest version that fixes all advisories
func AdvisoriesByPackage(advisories []*vadvisor.Advisory) map[string]Entry {
	advisoriesByPackage := map[string]Entry{}
	for i := range advisories {
		advisory := advisories[i]
		// iterate over all affected packages per advisory
		for j := range advisory.Affected {

			pkg := advisory.Affected[j]
			fixed := findVulnerablePackageWithoutNamespace(advisory, pkg)

			maxVulnerable := ""
			if fixed != nil {
				maxVulnerable = fixed.Version
			}

			pID := pkgID(pkg)
			pkgEntry, ok := advisoriesByPackage[pID]
			if !ok {
				advisoriesByPackage[pID] = Entry{
					Pkg:           pkg,
					Advisories:    []*vadvisor.Advisory{advisory},
					MaxVulnerable: maxVulnerable,
				}
			} else {
				if fixed != nil {
					// determine max version of required fix for advisory
					cmp, err := generic.Compare(pkgEntry.Pkg.Format, pkgEntry.MaxVulnerable, maxVulnerable)
					if err == nil && cmp < 0 {
						pkgEntry.MaxVulnerable = maxVulnerable
					}
				}

				pkgEntry.Advisories = append(pkgEntry.Advisories, advisory)
				advisoriesByPackage[pID] = pkgEntry
			}
		}
	}
	return advisoriesByPackage
}

func findVulnerablePackageWithoutNamespace(advisory *vadvisor.Advisory, innstalledPkg *vadvisor.Package) *vadvisor.Package {
	var match *vadvisor.Package
	for i := range advisory.Fixed {
		if advisory.Fixed[i].Name == innstalledPkg.Name || advisory.Fixed[i].Name == innstalledPkg.Origin {
			match = advisory.Fixed[i]
			return match
		}
	}
	return nil
}
