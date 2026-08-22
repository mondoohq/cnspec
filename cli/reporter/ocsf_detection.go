// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Detection Finding (OCSF class 2004): the class Splunk Enterprise Security and
// similar tools model findings on. It has no compliance object, so the framework
// mappings travel in unmapped, and it has the risk and impact attributes 2003
// lacks.

package reporter

import (
	"strings"

	"go.mondoo.com/cnspec/cli/reporter/ocsf"
	"go.mondoo.com/cnspec/policy"
)

// detectionFinding reports a failing check as OCSF class 2004, which is the
// class Splunk Enterprise Security and similar tools model findings on.
//
// It carries the same identity and remediation as the compliance finding, but
// swaps the compliance object, which 2004 does not have, for the risk and impact
// attributes it does: the check becomes an analytic under finding_info, and the
// framework mappings travel in unmapped.
func (c *ocsfConverter) detectionFinding(resolved *policy.ResolvedPolicy, report *policy.Report, query *policy.Mquery, score *policy.Score, ctx *ocsfAssetContext) ocsf.DetectionFinding {
	title := query.Title
	if title == "" {
		title = queryRuleID(query)
	}

	message := title + ": " + scoreStatusLabel(score)
	if msg := score.MessageLine(); msg != "" {
		message += " · " + msg
	}

	finding := ocsf.NewDetectionFinding(ocsf.DetectionFindingActivityCreate)
	finding.Time = c.now
	finding.SeverityID = ocsfCheckSeverity(score)
	finding.Severity = ocsf.SeverityName(finding.SeverityID)
	finding.StatusID, finding.Status = ocsfFindingStatus(score)
	finding.StatusCode = scoreStatusLabel(score)
	finding.StatusDetail = checkAssessment(resolved, report, query)
	finding.Message = message
	finding.Metadata = c.metadata(ctx.findingProfiles()...)
	finding.Unmapped = c.detectionUnmapped(query, score, ctx)

	finding.FindingInfo = c.findingInfo(query, score, title)
	finding.FindingInfo.Analytic = &ocsf.Analytic{
		TypeID:   ocsf.AnalyticTypeRule,
		Type:     ocsf.AnalyticTypeName(ocsf.AnalyticTypeRule),
		Name:     title,
		UID:      queryRuleID(query),
		Desc:     strings.TrimSpace(queryMql(query)),
		Category: firstOrEmpty(ctx.policyTitles[query.Mrn]),
	}

	risk := scoreRisk(score)
	finding.RiskScore = int(risk)
	finding.RiskLevelID = ocsfRiskLevel(risk)
	finding.RiskLevel = ocsf.RiskLevelName(finding.RiskLevelID)
	if impact, ok := queryImpact(query); ok {
		finding.ImpactScore = int(impact)
		finding.ImpactID = ocsfImpactLevel(impact)
		finding.Impact = ocsf.ImpactName(finding.ImpactID)
	}

	if rem := queryRemediation(query, ctx.platformKeys); rem != "" {
		finding.Remediation = &ocsf.Remediation{Desc: rem, References: refURLs(query)}
	}
	finding.Resources = []ocsf.ResourceDetails{ctx.resource}
	finding.Device = ctx.device
	finding.Cloud = ctx.cloud
	return finding
}

// detectionUnmapped carries what a detection finding has no attribute for. That
// includes the compliance mappings, since class 2004 has no compliance object.
func (c *ocsfConverter) detectionUnmapped(query *policy.Mquery, score *policy.Score, ctx *ocsfAssetContext) map[string]string {
	res := c.checkUnmapped(query, score, ctx)
	tags := queryComplianceTags(query)
	if frameworks := sortedKeys(tags); len(frameworks) > 0 {
		standards := make([]string, 0, len(frameworks))
		controls := make([]string, 0, len(frameworks))
		for _, framework := range frameworks {
			standards = append(standards, strings.TrimPrefix(framework, "compliance/"))
			controls = append(controls, tags[framework])
		}
		res["compliance_standards"] = strings.Join(standards, ", ")
		res["compliance_controls"] = strings.Join(controls, ", ")
	}
	return res
}

// ocsfRiskLevel maps a cnspec risk value to an OCSF risk_level_id, whose bands
// are Info/Low/Medium/High/Critical rather than the severity ones.
func ocsfRiskLevel(risk int32) int {
	switch {
	case risk >= 90:
		return ocsf.RiskLevelCritical
	case risk >= 70:
		return ocsf.RiskLevelHigh
	case risk >= 40:
		return ocsf.RiskLevelMedium
	case risk >= 1:
		return ocsf.RiskLevelLow
	default:
		return ocsf.RiskLevelInfo
	}
}

// ocsfImpactLevel maps a check's configured impact to an OCSF impact_id.
func ocsfImpactLevel(impact int32) int {
	switch {
	case impact >= 90:
		return ocsf.ImpactCritical
	case impact >= 70:
		return ocsf.ImpactHigh
	case impact >= 40:
		return ocsf.ImpactMedium
	case impact >= 1:
		return ocsf.ImpactLow
	default:
		return ocsf.ImpactUnknown
	}
}

func firstOrEmpty(items []string) string {
	if len(items) == 0 {
		return ""
	}
	return items[0]
}
