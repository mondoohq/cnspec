// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Compliance Finding (OCSF class 2003): one per check per asset, plus one per
// asset that could not be scanned at all.

package reporter

import (
	"strings"

	"go.mondoo.com/cnspec/cli/reporter/ocsf"
	"go.mondoo.com/cnspec/policy"
)

// addComplianceFindings emits one finding per reporting check of an asset.
func (c *ocsfConverter) addComplianceFindings(events *ocsf.Events, r *policy.ReportCollection, ctx *ocsfAssetContext, queries map[string]*policy.Mquery) {
	report, ok := r.Reports[ctx.assetMrn]
	if !ok {
		return
	}
	resolved, ok := r.ResolvedPolicies[ctx.assetMrn]
	if !ok || resolved.CollectorJob == nil {
		return
	}

	for _, id := range sortedKeys(report.Scores) {
		score := report.Scores[id]
		if _, ok := resolved.CollectorJob.ReportingQueries[id]; !ok {
			continue
		}
		query, ok := queries[id]
		if !ok {
			continue
		}

		// Both classes report every check, passing ones included, with the outcome
		// in status_code. A detection stream that dropped the passes would not be
		// a complete record of the scan on its own.
		if c.findings.detection() {
			events.DetectionFindings = append(events.DetectionFindings,
				c.detectionFinding(resolved, report, query, score, ctx))
		} else {
			events.ComplianceFindings = append(events.ComplianceFindings,
				c.complianceFinding(resolved, report, query, score, ctx))
		}
	}
}

func (c *ocsfConverter) complianceFinding(resolved *policy.ResolvedPolicy, report *policy.Report, query *policy.Mquery, score *policy.Score, ctx *ocsfAssetContext) ocsf.ComplianceFinding {
	title := query.Title
	if title == "" {
		title = queryRuleID(query)
	}

	message := title + ": " + scoreStatusLabel(score)
	if msg := score.MessageLine(); msg != "" {
		message += " · " + msg
	}

	finding := ocsf.NewComplianceFinding(ocsf.ComplianceFindingActivityCreate)
	finding.Compliance = c.compliance(resolved, report, query, score, ctx)
	finding.FindingInfo = c.findingInfo(query, score, title)
	finding.Resources = []ocsf.ResourceDetails{ctx.resource}
	finding.Device = ctx.device
	finding.Cloud = ctx.cloud
	finding.Time = c.now
	finding.SeverityID = ocsfCheckSeverity(score)
	finding.Severity = ocsf.SeverityName(finding.SeverityID)
	finding.StatusID, finding.Status = ocsfFindingStatus(score)
	finding.StatusCode = scoreStatusLabel(score)
	finding.Message = message
	finding.Metadata = c.metadata(ctx.findingProfiles()...)
	finding.Unmapped = c.checkUnmapped(query, score, ctx)

	if rem := queryRemediation(query, ctx.platformKeys); rem != "" {
		finding.Remediation = &ocsf.Remediation{
			Desc:       rem,
			References: refURLs(query),
		}
	}
	return finding
}

// compliance builds the compliance context of a check: which frameworks it maps
// to, which control it covers, and how the check came out.
func (c *ocsfConverter) compliance(resolved *policy.ResolvedPolicy, report *policy.Report, query *policy.Mquery, score *policy.Score, ctx *ocsfAssetContext) ocsf.Compliance {
	tags := queryComplianceTags(query)
	frameworks := sortedKeys(tags)

	res := ocsf.Compliance{}
	res.StatusID, res.Status = ocsfComplianceStatus(score)

	for _, framework := range frameworks {
		res.Standards = append(res.Standards, strings.TrimPrefix(framework, "compliance/"))
		res.Requirements = append(res.Requirements, tags[framework])
	}
	policies := ctx.policyTitles[query.Mrn]
	if len(res.Standards) == 0 {
		// Without a compliance mapping the policy is the standard being applied.
		res.Standards = policies
	}
	if len(res.Standards) == 0 {
		// standards is a required attribute, so it always carries something.
		res.Standards = []string{"Mondoo Policy"}
	}

	if len(res.Requirements) > 0 {
		res.Control = res.Requirements[0]
	} else {
		res.Control = queryRuleID(query)
	}

	if detail := checkAssessment(resolved, report, query); detail != "" {
		// 1.9 deprecates the singular status_detail in favor of status_details.
		if c.version.AtLeast(ocsf.Version190) {
			res.StatusDetails = []string{detail}
		} else {
			res.StatusDetail = detail
		}
	}

	// compliance.category and compliance.desc were added in OCSF 1.9.
	if c.version.AtLeast(ocsf.Version190) {
		res.Desc = queryDescription(query)
		if len(policies) > 0 {
			res.Category = policies[0]
		}
	}
	return res
}

// addAssetError reports an asset that could not be scanned at all. Without it the
// asset would silently look clean: it has no checks and therefore no findings.
func (c *ocsfConverter) addAssetError(events *ocsf.Events, r *policy.ReportCollection, ctx *ocsfAssetContext) {
	errMsg, ok := r.Errors[ctx.assetMrn]
	if !ok || errMsg == "" {
		return
	}

	info := ocsf.FindingInfo{
		UID:         ocsfAssetErrorUID,
		Title:       "Asset scan error",
		Desc:        "cnspec could not complete the scan of this asset. No policy results are available for it.",
		CreatedTime: c.now,
		Types:       []string{"Scan Error"},
		DataSources: []string{ocsfProductName},
	}
	unmapped := map[string]string{}
	if ctx.assetMrn != "" {
		unmapped["asset_mrn"] = ctx.assetMrn
	}

	// A scan that did not run leaves every check unanswered, which is worse than
	// a single check erroring out.
	const severity = ocsf.SeverityHigh
	message := "Asset scan error: " + errMsg

	if c.findings.detection() {
		finding := ocsf.NewDetectionFinding(ocsf.DetectionFindingActivityCreate)
		finding.Time = c.now
		finding.SeverityID = severity
		finding.Severity = ocsf.SeverityName(severity)
		finding.StatusID = ocsf.StatusOther
		finding.Status = "Error"
		finding.StatusCode = "ERROR"
		finding.StatusDetail = errMsg
		finding.Message = message
		finding.Metadata = c.metadata(ctx.findingProfiles()...)
		finding.Unmapped = unmapped
		finding.FindingInfo = info
		finding.Resources = []ocsf.ResourceDetails{ctx.resource}
		finding.Device = ctx.device
		finding.Cloud = ctx.cloud
		events.DetectionFindings = append(events.DetectionFindings, finding)
		return
	}

	finding := ocsf.NewComplianceFinding(ocsf.ComplianceFindingActivityCreate)
	finding.Time = c.now
	finding.SeverityID = severity
	finding.Severity = ocsf.SeverityName(severity)
	finding.StatusID = ocsf.StatusOther
	finding.Status = "Error"
	finding.StatusCode = "ERROR"
	finding.StatusDetail = errMsg
	finding.Message = message
	finding.Metadata = c.metadata(ctx.findingProfiles()...)
	finding.Unmapped = unmapped
	finding.Compliance = c.errorCompliance(errMsg)
	finding.FindingInfo = info
	finding.Resources = []ocsf.ResourceDetails{ctx.resource}
	finding.Device = ctx.device
	finding.Cloud = ctx.cloud
	events.ComplianceFindings = append(events.ComplianceFindings, finding)
}

const ocsfAssetErrorUID = "asset-error"

// errorCompliance is the compliance context of an asset that could not be
// scanned: no verdict, and the error as the detail.
func (c *ocsfConverter) errorCompliance(errMsg string) ocsf.Compliance {
	res := ocsf.Compliance{
		Standards: []string{"Mondoo Policy"},
		Control:   ocsfAssetErrorUID,
		StatusID:  ocsf.ComplianceStatusUnknown,
		Status:    ocsf.ComplianceStatusName(ocsf.ComplianceStatusUnknown),
	}
	if c.version.AtLeast(ocsf.Version190) {
		res.StatusDetails = []string{errMsg}
	} else {
		res.StatusDetail = errMsg
	}
	return res
}
