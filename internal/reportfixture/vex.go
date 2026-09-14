// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reportfixture

import "go.mondoo.com/mql/providers-sdk/v1/upstream/fex"

// VexTarget is the asset name the VEX fixture reports against. `cnspec vuln`
// has no scan behind it, so a name is all a target is.
const VexTarget = "alpine:3.19@6cf065f724d5"

// VexRows is the fixed set of VEX rows the vulnerability reporters are tested
// against. It is the output shape of fex.VulnRows -- one row per (vulnerability,
// affected package), already ordered by severity and then id -- and it covers
// the cases the reporters have to decide something about:
//
//   - a real CVE id, which belongs in OCSF's cve.uid;
//   - a distro advisory id, which does not, and which affects two packages, so
//     the OCSF grouping back into one finding per advisory is exercised;
//   - a vulnerability with no fixed version;
//   - a row whose component never resolved, which VulnRows emits rather than
//     dropping the vulnerability, and which therefore has to render;
//   - an unrecognised severity, which has to land in the same band as none.
//
// Callers must not mutate the result.
func VexRows() []fex.VulnRow {
	return []fex.VulnRow{
		{
			ID:              "USN-1234-1",
			Severity:        "CRITICAL",
			Summary:         "openssl vulnerabilities",
			AffectedName:    "libssl3",
			AffectedVersion: "3.0.2-0ubuntu1.10",
			AffectedPurl:    "pkg:deb/ubuntu/libssl3@3.0.2-0ubuntu1.10?arch=amd64&distro=ubuntu-22.04",
			FixedVersion:    "3.0.2-0ubuntu1.12",
			References:      []string{"https://ubuntu.com/security/notices/USN-1234-1"},
			RemediationHint: "Update libssl3 to 3.0.2-0ubuntu1.12.",
		},
		{
			ID:              "USN-1234-1",
			Severity:        "CRITICAL",
			Summary:         "openssl vulnerabilities",
			AffectedName:    "openssl",
			AffectedVersion: "3.0.2-0ubuntu1.10",
			AffectedPurl:    "pkg:deb/ubuntu/openssl@3.0.2-0ubuntu1.10?arch=amd64&distro=ubuntu-22.04",
			FixedVersion:    "3.0.2-0ubuntu1.12",
			References:      []string{"https://ubuntu.com/security/notices/USN-1234-1"},
			RemediationHint: "Update libssl3 to 3.0.2-0ubuntu1.12.",
		},
		{
			ID:              "CVE-2023-0286",
			Severity:        "HIGH",
			Summary:         "X.400 address type confusion in X.509 GeneralName.",
			AffectedName:    "libssl3",
			AffectedVersion: "3.0.2-0ubuntu1.10",
			AffectedPurl:    "pkg:deb/ubuntu/libssl3@3.0.2-0ubuntu1.10?arch=amd64&distro=ubuntu-22.04",
			FixedVersion:    "3.0.2-0ubuntu1.12",
			References: []string{
				"https://nvd.nist.gov/vuln/detail/CVE-2023-0286",
				"https://www.openssl.org/news/secadv/20230207.txt",
			},
		},
		{
			ID:              "CVE-2024-9999",
			Severity:        "MEDIUM",
			Summary:         "No fix has been published for this issue yet.",
			AffectedName:    "zlib1g",
			AffectedVersion: "1:1.2.11.dfsg-2ubuntu9.2",
			AffectedPurl:    "pkg:deb/ubuntu/zlib1g@1%3A1.2.11.dfsg-2ubuntu9.2?arch=amd64&distro=ubuntu-22.04",
		},
		{
			ID:       "CVE-2024-7777",
			Severity: "LOW",
			Summary:  "The affected component could not be resolved.",
		},
		{
			ID:              "GHSA-xxxx-yyyy-zzzz",
			Severity:        "SOMETHING-ELSE",
			Summary:         "An advisory whose severity label is not one cnspec knows.",
			AffectedName:    "left-pad",
			AffectedVersion: "1.3.0",
			AffectedPurl:    "pkg:npm/left-pad@1.3.0",
		},
	}
}
