// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Vulnerability Finding (OCSF class 2002) built from VEX rather than from the
// legacy mvd vulnerability report. Same class and the same field choices as
// vulnerability.go -- see the notes there on cve.uid and cvss_score, which apply
// identically here.

package convert

import (
	"regexp"
	"sort"
	"strconv"
	"strings"

	"go.mondoo.com/cnspec/reports/ocsf"
	"go.mondoo.com/cnspec/reports/reportdoc"
	"go.mondoo.com/mql/providers-sdk/v1/upstream/fex"
)

// cveIDPattern matches a CVE identifier. VEX advisory ids are a mix of real CVEs
// and distro advisory ids (ALPINE-CVE-..., USN-..., RHSA-...), and only the
// former belongs in cve.uid; see nonCVEUID in vulnerability.go.
var cveIDPattern = regexp.MustCompile(`^CVE-[0-9]{4}-[0-9]+$`)

// addVexFindings emits one finding per advisory affecting the asset, with every
// package that advisory affects listed on it. Rows arrive one per
// (vulnerability, affected package), which is the grouping this undoes -- the
// mvd path builds the same shape from advisory.Packages.
func (c *converter) addVexFindings(events *ocsf.Events, rows []fex.VulnRow, ctx *assetContext) {
	if len(rows) == 0 {
		return
	}

	// Group by advisory id, preserving each group's first-seen order so the
	// severity ordering VulnRows already applied survives grouping.
	order := []string{}
	groups := map[string][]fex.VulnRow{}
	for _, row := range rows {
		id := row.ID
		if _, ok := groups[id]; !ok {
			order = append(order, id)
		}
		groups[id] = append(groups[id], row)
	}

	for _, id := range order {
		events.VulnerabilityFindings = append(
			events.VulnerabilityFindings, c.vexFinding(id, groups[id], ctx))
	}
}

// vexFinding builds one finding: one advisory, every package it affects here.
func (c *converter) vexFinding(id string, rows []fex.VulnRow, ctx *assetContext) ocsf.VulnerabilityFinding {
	first := rows[0]
	risk := reportdoc.RiskFromSeverityLabel(first.Severity)

	uid := id
	title := id
	desc := strings.TrimSpace(first.Summary)
	if id == "" {
		// A row with no advisory id still reports; the mvd path calls this case
		// vulnerable-package and so does this one.
		uid = "vulnerable-package"
		title = "Vulnerable package"
		if name := strings.TrimSpace(first.AffectedName); name != "" {
			uid = "vulnerable-package/" + reportdoc.VexPackageKey(first)
			title = name + " " + first.AffectedVersion + " has known vulnerabilities"
		}
	}
	if desc == "" {
		desc = "An installed package is affected by known vulnerabilities."
	}

	packages := make([]ocsf.AffectedPackage, 0, len(rows))
	fixAvailable := false
	seen := map[string]bool{}
	for _, row := range rows {
		key := reportdoc.VexPackageKey(row)
		if row.AffectedName == "" || seen[key] {
			continue
		}
		seen[key] = true
		packages = append(packages, ocsf.AffectedPackage{
			Name:           row.AffectedName,
			Version:        row.AffectedVersion,
			FixedInVersion: row.FixedVersion,
			PackageManager: fex.EcosystemOf(row.AffectedPurl),
			PURL:           row.AffectedPurl,
		})
		if row.FixedVersion != "" {
			fixAvailable = true
		}
	}

	vuln := ocsf.Vulnerability{
		Title:            title,
		Desc:             desc,
		Severity:         ocsf.SeverityName(severityFromRisk(risk)),
		AffectedPackages: packages,
		IsFixAvailable:   fixAvailable,
		References:       vexReferences(rows),
	}

	// See nonCVEUID in vulnerability.go: an advisory id in cve.uid joins against
	// nothing in NVD, so it is marked rather than passed off as a CVE.
	var nonCVEUID string
	switch {
	case id == "":
		// Nothing to identify it by; leave both unset.
	case cveIDPattern.MatchString(id):
		vuln.CVE = &ocsf.CVE{UID: id, Desc: desc}
	case c.version.AtLeast(ocsf.Version190):
		vuln.Advisory = &ocsf.Advisory{
			UID:        id,
			Title:      title,
			Desc:       desc,
			References: vuln.References,
		}
	default:
		vuln.CVE = &ocsf.CVE{UID: id, Title: title, Desc: desc}
		nonCVEUID = id
	}

	finding := ocsf.NewVulnerabilityFinding(ocsf.VulnerabilityFindingActivityCreate)
	finding.FindingInfo = ocsf.FindingInfo{
		UID:         findingUID(uid, ctx),
		Title:       title,
		Desc:        desc,
		CreatedTime: c.now,
		Types:       []string{"Vulnerability"},
		DataSources: []string{productName},
	}
	finding.Vulnerabilities = []ocsf.Vulnerability{vuln}
	finding.Device = ctx.device
	finding.Cloud = ctx.cloud
	if ctx.resource.UID != "" || ctx.resource.Name != "" {
		finding.Resources = []ocsf.ResourceDetails{ctx.resource}
	}
	finding.Time = c.now
	finding.SeverityID = severityFromRisk(risk)
	finding.Severity = ocsf.SeverityName(finding.SeverityID)
	finding.StatusID = ocsf.StatusNew
	finding.Status = ocsf.StatusName(finding.StatusID)
	finding.StatusCode = "FAIL"
	finding.Message = title
	finding.Metadata = c.metadata(ctx.findingProfiles()...)

	unmapped := map[string]string{
		"cvss_score": strconv.FormatFloat(float64(risk)/10, 'f', 1, 64),
	}
	if ctx.assetMrn != "" {
		unmapped["asset_mrn"] = ctx.assetMrn
	}
	if id != "" {
		unmapped["advisory"] = id
	}
	if nonCVEUID != "" {
		unmapped["non_cve_uid"] = nonCVEUID
	}
	if hint := strings.TrimSpace(first.RemediationHint); hint != "" {
		unmapped["remediation"] = hint
	}
	finding.Unmapped = unmapped

	return finding
}

// vexReferences collects the reference URLs across an advisory's rows, sorted
// and deduplicated so the finding does not repeat what every row carries.
func vexReferences(rows []fex.VulnRow) []string {
	seen := map[string]bool{}
	var res []string
	for _, row := range rows {
		for _, ref := range row.References {
			if ref == "" || seen[ref] {
				continue
			}
			seen[ref] = true
			res = append(res, ref)
		}
	}
	sort.Strings(res)
	return res
}
