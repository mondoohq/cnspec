// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reporter

import "go.mondoo.com/mql/providers-sdk/v1/upstream/fex"

// sampleVEX returns a small, fixed set of VEX documents covering the fields the
// reporters render: severity (word and score derived), affected package coords
// via a sub-component PURL, a structured fixed_version, references, and a
// remediation hint. The rows are intentionally spread across severities so the
// severity ordering and summary counts are exercised.
func sampleVEX() []*fex.VulnerabilityExchange {
	return []*fex.VulnerabilityExchange{
		{
			Id:      "CVE-2022-0001",
			Summary: "Prototype pollution in lodash",
			Ratings: []*fex.Rating{{Severity: "critical", Score: 9.8}},
			Affects: []*fex.Affects{{
				Component: &fex.Component{Id: "npm"},
				SubComponents: []*fex.Component{{
					Id: "lodash",
					Identifiers: map[string]string{
						"purl":          "pkg:npm/lodash@4.17.20",
						"version":       "4.17.20",
						"fixed_version": "4.17.21",
					},
				}},
			}},
			References: []*fex.Reference{{Url: "https://nvd.nist.gov/vuln/detail/CVE-2022-0001", Type: "ADVISORY"}},
			Remediations: []*fex.Remediation{
				{Summary: "Upgrade lodash to 4.17.21", FixType: "package"},
			},
		},
		{
			Id:      "CVE-2021-3999",
			Summary: "Off-by-one in glibc getcwd",
			Ratings: []*fex.Rating{{Severity: "high"}},
			Affects: []*fex.Affects{{
				SubComponents: []*fex.Component{{
					Id: "libc6",
					Identifiers: map[string]string{
						"purl":          "pkg:deb/ubuntu/libc6@2.31",
						"version":       "2.31-0ubuntu9.2",
						"fixed_version": "2.31-0ubuntu9.7",
					},
				}},
			}},
			References: []*fex.Reference{{Url: "https://ubuntu.com/security/CVE-2021-3999"}},
		},
		{
			Id:      "CVE-2021-3995",
			Summary: "util-linux libmount improper handling",
			Ratings: []*fex.Rating{{Score: 5.5}}, // score-derived MEDIUM
			Affects: []*fex.Affects{{
				SubComponents: []*fex.Component{{
					Id: "libblkid1",
					Identifiers: map[string]string{
						"purl":    "pkg:deb/ubuntu/libblkid1@2.34-0.1ubuntu9.1",
						"version": "2.34-0.1ubuntu9.1",
					},
				}},
			}},
			Remediations: []*fex.Remediation{
				{Summary: "Upgrade to 2.34-0.1ubuntu9.3", FixType: "package"},
			},
		},
	}
}
