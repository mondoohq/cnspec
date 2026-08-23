// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Detection Finding (OCSF class 2004): the class Splunk Enterprise Security and
// similar tools model findings on. It has no compliance object, so the framework
// mappings travel in unmapped, and it has the risk and impact attributes 2003
// lacks.

package convert

import (
	"maps"
	"slices"
	"strings"

	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/cnspec/reports/ocsf"
	"go.mondoo.com/cnspec/reports/reportdoc"
)

// detectionFinding reports a failing check as OCSF class 2004, which is the
// class Splunk Enterprise Security and similar tools model findings on.
//
// It carries the same identity and remediation as the compliance finding, but
// swaps the compliance object, which 2004 does not have, for the risk and impact
// attributes it does: the check becomes an analytic under finding_info, and the
// framework mappings travel in unmapped.
func (c *converter) detectionFinding(resolved *policy.ResolvedPolicy, report *policy.Report, query *policy.Mquery, score *policy.Score, ctx *assetContext) ocsf.DetectionFinding {
	title := query.Title
	if title == "" {
		title = reportdoc.QueryRuleID(query)
	}

	outcome := reportdoc.OutcomeOf(score).Label()
	message := title + ": " + outcome
	if msg := score.MessageLine(); msg != "" {
		message += " · " + msg
	}

	finding := ocsf.NewDetectionFinding(ocsf.DetectionFindingActivityCreate)
	finding.Time = c.now
	finding.SeverityID = checkSeverity(query, score)
	finding.Severity = ocsf.SeverityName(finding.SeverityID)
	finding.StatusID, finding.Status = findingStatus(score)
	finding.StatusCode = outcome
	finding.StatusDetail = checkAssessment(resolved, report, query)
	finding.Message = message
	finding.Metadata = c.metadata(ctx.findingProfiles()...)
	finding.Unmapped = c.detectionUnmapped(query, score, ctx)

	finding.FindingInfo = c.findingInfo(query, score, title, ctx)
	finding.FindingInfo.Analytic = &ocsf.Analytic{
		TypeID:   ocsf.AnalyticTypeRule,
		Type:     ocsf.AnalyticTypeName(ocsf.AnalyticTypeRule),
		Name:     title,
		UID:      reportdoc.QueryRuleID(query),
		Desc:     strings.TrimSpace(reportdoc.QueryMql(query)),
		Category: firstOrEmpty(ctx.policyTitles[query.Mrn]),
	}

	// Risk is derived only from a real result. ScoreRisk is 100 - score.Value, and
	// Value is 0 for every outcome that carries no verdict -- errored, skipped,
	// unscored -- so an unguarded conversion reports a check that never ran as
	// risk_score 100, Critical. risk_score is the field Splunk ES Risk-Based
	// Alerting sums per asset, so a Linux host with a Kubernetes policy attached
	// would accrue 100 risk points for each check that could not apply to it.
	// checkSeverity and checkUnmapped already guard on Type for the same
	// reason; this is the third place that has to. No verdict means no score and
	// the Info band, which is what risk 0 maps to.
	var risk int32
	if score.GetType() == policy.ScoreType_Result {
		risk = reportdoc.ScoreRisk(score)
		finding.RiskScore = int(risk)
	}
	finding.RiskLevelID = riskLevel(risk)
	finding.RiskLevel = ocsf.RiskLevelName(finding.RiskLevelID)
	if impact, ok := reportdoc.QueryImpact(query); ok {
		finding.ImpactScore = int(impact)
		finding.ImpactID = impactLevel(impact)
		finding.Impact = ocsf.ImpactName(finding.ImpactID)
	}

	if rem := reportdoc.QueryRemediation(query, ctx.platformKeys); rem != "" {
		finding.Remediation = &ocsf.Remediation{Desc: rem, References: refURLs(query)}
	}
	finding.Resources = []ocsf.ResourceDetails{ctx.resource}
	finding.Device = ctx.device
	finding.Cloud = ctx.cloud
	return finding
}

// detectionUnmapped carries what a detection finding has no attribute for. That
// includes the compliance mappings, since class 2004 has no compliance object.
func (c *converter) detectionUnmapped(query *policy.Mquery, score *policy.Score, ctx *assetContext) map[string]string {
	res := c.checkUnmapped(query, score, ctx)
	tags := reportdoc.QueryComplianceTags(query)
	if frameworks := slices.Sorted(maps.Keys(tags)); len(frameworks) > 0 {
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

// detectionAssetError reports an asset that could not be scanned as class 2004.
func (c *converter) detectionAssetError(errMsg string, ctx *assetContext) ocsf.DetectionFinding {
	finding := ocsf.NewDetectionFinding(ocsf.DetectionFindingActivityCreate)
	finding.Time = c.now
	finding.SeverityID = assetErrorSeverity
	finding.Severity = ocsf.SeverityName(assetErrorSeverity)
	finding.StatusID = ocsf.StatusOther
	finding.Status = "Error"
	finding.StatusCode = reportdoc.OutcomeError.Label()
	finding.StatusDetail = errMsg
	finding.Message = "Asset scan error: " + errMsg
	finding.Metadata = c.metadata(ctx.findingProfiles()...)
	finding.Unmapped = assetErrorUnmapped(ctx)
	finding.FindingInfo = c.assetErrorInfo(ctx)
	finding.Resources = []ocsf.ResourceDetails{ctx.resource}
	finding.Device = ctx.device
	finding.Cloud = ctx.cloud
	return finding
}
