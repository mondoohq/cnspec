// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Compliance Finding (OCSF class 2003): the default class for a cnspec check,
// because it has a compliance object for the framework mappings and the control.

package convert

import (
	"maps"
	"slices"
	"strings"

	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/cnspec/reports/ocsf"
	"go.mondoo.com/cnspec/reports/reportdoc"
)

func (c *converter) complianceFinding(resolved *policy.ResolvedPolicy, report *policy.Report, query *policy.Mquery, score *policy.Score, ctx *assetContext) ocsf.ComplianceFinding {
	title := query.Title
	if title == "" {
		title = reportdoc.QueryRuleID(query)
	}

	outcome := reportdoc.OutcomeOf(score).Label()
	message := title + ": " + outcome
	if msg := score.MessageLine(); msg != "" {
		message += " · " + msg
	}

	finding := ocsf.NewComplianceFinding(ocsf.ComplianceFindingActivityCreate)
	finding.Compliance = c.compliance(resolved, report, query, score, ctx)
	finding.FindingInfo = c.findingInfo(query, score, title, ctx)
	finding.Resources = []ocsf.ResourceDetails{ctx.resource}
	finding.Device = ctx.device
	finding.Cloud = ctx.cloud
	finding.Time = c.now
	finding.SeverityID = checkSeverity(query, score)
	finding.Severity = ocsf.SeverityName(finding.SeverityID)
	finding.StatusID, finding.Status = findingStatus(score)
	finding.StatusCode = outcome
	finding.Message = message
	finding.Metadata = c.metadata(ctx.findingProfiles()...)
	finding.Unmapped = c.checkUnmapped(query, score, ctx)

	if rem := reportdoc.QueryRemediation(query, ctx.platformKeys); rem != "" {
		finding.Remediation = &ocsf.Remediation{
			Desc:       rem,
			References: refURLs(query),
		}
	}
	return finding
}

// compliance builds the compliance context of a check: which frameworks it maps
// to, which control it covers, and how the check came out.
func (c *converter) compliance(resolved *policy.ResolvedPolicy, report *policy.Report, query *policy.Mquery, score *policy.Score, ctx *assetContext) ocsf.Compliance {
	tags := reportdoc.QueryComplianceTags(query)
	frameworks := slices.Sorted(maps.Keys(tags))

	res := ocsf.Compliance{}
	res.StatusID, res.Status = complianceStatus(score)

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
		res.Control = reportdoc.QueryRuleID(query)
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
		res.Desc = reportdoc.QueryDescription(query)
		if len(policies) > 0 {
			res.Category = policies[0]
		}
	}
	return res
}

// complianceAssetError reports an asset that could not be scanned as class 2003.
func (c *converter) complianceAssetError(errMsg string, ctx *assetContext) ocsf.ComplianceFinding {
	finding := ocsf.NewComplianceFinding(ocsf.ComplianceFindingActivityCreate)
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
	finding.Compliance = c.errorCompliance(errMsg)
	finding.FindingInfo = c.assetErrorInfo(ctx)
	finding.Resources = []ocsf.ResourceDetails{ctx.resource}
	finding.Device = ctx.device
	finding.Cloud = ctx.cloud
	return finding
}

// errorCompliance is the compliance context of an asset that could not be
// scanned: no verdict, and the error as the detail.
func (c *converter) errorCompliance(errMsg string) ocsf.Compliance {
	res := ocsf.Compliance{
		Standards: []string{"Mondoo Policy"},
		Control:   assetErrorUID,
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
