// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"bytes"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnspec"
	"go.mondoo.com/cnspec/cli/reporter/ocsf"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/mql/cli/printer"
	cr "go.mondoo.com/mql/cli/reporter"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/providers-sdk/v1/upstream/mvd"
	"go.mondoo.com/mql/utils/iox"
)

// The OCSF report carries the same content as the SARIF report, re-cut along the
// event classes a security data lake indexes on:
//
//	Compliance Finding (2003)     one per check per asset, plus one per asset
//	                              that failed to scan at all
//	Vulnerability Finding (2002)  one per advisory per asset, plus one per
//	                              affected package no advisory accounts for
//	Device Inventory Info (5001)  one per asset, carrying the platform details
//	                              and (with the "data" option) the results of
//	                              data-only queries
//
// Every event is self-describing: `class_uid` says what it is, `metadata.version`
// says which OCSF version it follows. See cli/reporter/ocsf for the schema.

// ocsfProductName and friends identify cnspec as the producer of the events.
const (
	ocsfProductName   = "cnspec"
	ocsfProductVendor = "Mondoo, Inc."
	ocsfProductURL    = "https://cnspec.io"
)

// ConvertToOCSF converts a report collection into OCSF events.
func ConvertToOCSF(r *policy.ReportCollection, conf *PrintConfig) (*ocsf.Events, error) {
	return convertToOCSF(r, conf.ocsfConfig(), time.Now())
}

// ConvertToOCSFJSON writes a report collection as newline-delimited OCSF events.
func ConvertToOCSFJSON(r *policy.ReportCollection, conf *PrintConfig, out io.Writer) error {
	events, err := ConvertToOCSF(r, conf)
	if err != nil {
		return err
	}
	return events.WriteJSON(out)
}

// VulnReportToOCSFJSON writes a standalone vulnerability report (cnspec vuln) as
// newline-delimited OCSF Vulnerability Findings.
func VulnReportToOCSFJSON(target string, data *mvd.VulnReport, version ocsf.Version, out io.Writer) error {
	c := &ocsfConverter{version: version, now: time.Now().UnixMilli()}
	asset := &inventory.Asset{Name: target}
	events := &ocsf.Events{}
	c.addVulnerabilityFindings(events, data, &ocsfAssetContext{
		asset:    asset,
		device:   buildOcsfDevice(asset, version),
		resource: buildOcsfResource(asset),
	})
	events.Sort()
	return events.WriteJSON(out)
}

func convertToOCSF(r *policy.ReportCollection, conf ocsfConfig, now time.Time) (*ocsf.Events, error) {
	c := &ocsfConverter{
		version:     conf.version,
		findings:    conf.findings,
		includeData: conf.includeData,
		now:         now.UnixMilli(),
	}
	return c.convert(r)
}

// ocsfConfig is what the output format options say about the events to produce.
type ocsfConfig struct {
	version     ocsf.Version
	findings    OcsfFindingClasses
	includeData bool
}

// OcsfFindingClasses selects which OCSF class a check result is reported as.
//
// Every check is reported exactly once, in one class. That is what other OCSF
// producers do: Security Lake routes each Security Hub finding to the one class
// that fits it, and Prowler reports every check as a Detection Finding. Emitting
// a check as two events would double it in anything that counts findings across
// classes.
//
// A cnspec check is a compliance check, so Compliance Finding (2003) is the
// default: it has a compliance object for the framework mappings and the control.
// Detection Finding (2004) is what Splunk Enterprise Security and similar tools
// model findings on; it has no compliance object, so the mappings travel in
// unmapped, and it has the risk and impact attributes 2003 lacks.
type OcsfFindingClasses byte

const (
	OcsfFindingsCompliance OcsfFindingClasses = iota + 1
	OcsfFindingsDetection
)

func (f OcsfFindingClasses) compliance() bool { return f == OcsfFindingsCompliance }

func (f OcsfFindingClasses) detection() bool { return f == OcsfFindingsDetection }

// ocsfConverter holds everything that is the same for every event of one run.
type ocsfConverter struct {
	version     ocsf.Version
	findings    OcsfFindingClasses
	includeData bool
	// now is the scan time in milliseconds since the epoch. It is a field rather
	// than a call to time.Now so that a conversion is reproducible.
	now int64
}

// ocsfAssetContext carries the per-asset values every event of that asset needs.
type ocsfAssetContext struct {
	assetMrn     string
	asset        *inventory.Asset
	platformKeys map[string]bool
	policyTitles map[string][]string
	device       *ocsf.Device
	cloud        *ocsf.Cloud
	resource     ocsf.ResourceDetails
}

func (c *ocsfConverter) convert(r *policy.ReportCollection) (*ocsf.Events, error) {
	events := &ocsf.Events{}
	if r == nil {
		return events, nil
	}
	if r.Bundle == nil {
		return nil, errors.New("no policy bundle found")
	}

	bundle := r.Bundle.ToMap()
	queries := reporterQueryMap(bundle)
	policyTitles := policyTitlesByQuery(bundle)

	for _, assetMrn := range sortedKeys(r.Assets) {
		asset := r.Assets[assetMrn]
		if asset == nil {
			continue
		}
		ctx := &ocsfAssetContext{
			assetMrn:     assetMrn,
			asset:        asset,
			platformKeys: platformRemediationKeys(asset.Platform),
			policyTitles: policyTitles,
			device:       buildOcsfDevice(asset, c.version),
			cloud:        buildOcsfCloud(asset),
			resource:     buildOcsfResource(asset),
		}

		events.InventoryInfos = append(events.InventoryInfos, c.inventoryInfo(r, ctx))
		c.addComplianceFindings(events, r, ctx, queries)
		c.addAssetError(events, r, ctx)
		c.addVulnerabilityFindings(events, r.VulnReports[assetMrn], ctx)
	}

	events.Sort()
	return events, nil
}

// metadata is the same for every event of a run, except for the profiles.
//
// Profiles are not decoration: an OCSF attribute that belongs to a profile is
// only allowed on an event that declares the profile. `device` on a finding
// comes from the host profile and `cloud` from the cloud profile, so an event
// carrying either without saying so is rejected by a schema-aware validator.
func (c *ocsfConverter) metadata(profiles ...string) ocsf.Metadata {
	res := ocsf.Metadata{
		Version: string(c.version),
		Product: ocsf.Product{
			Name:       ocsfProductName,
			VendorName: ocsfProductVendor,
			Version:    cnspec.GetVersion(),
			URLString:  ocsfProductURL,
		},
		LoggedTime: c.now,
	}
	if len(profiles) > 0 {
		res.Profiles = profiles
	}
	return res
}

// findingProfiles lists the OCSF profiles a finding of this asset uses. Findings
// carry the asset as a device (host profile) and, for cloud assets, the cloud
// environment (cloud profile).
func (ctx *ocsfAssetContext) findingProfiles() []string {
	res := []string{}
	if ctx.device != nil {
		res = append(res, ocsf.ProfileHost)
	}
	if ctx.cloud != nil {
		res = append(res, ocsf.ProfileCloud)
	}
	return res
}

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

func (c *ocsfConverter) findingInfo(query *policy.Mquery, score *policy.Score, title string) ocsf.FindingInfo {
	res := ocsf.FindingInfo{
		UID:           queryRuleID(query),
		Title:         title,
		Desc:          queryDescription(query),
		CreatedTime:   c.now,
		FirstSeenTime: ocsfMillis(score.GetFailureTime()),
		ModifiedTime:  ocsfMillis(score.GetValueModifiedTime()),
		Types:         []string{"Compliance Check"},
		DataSources:   []string{ocsfProductName},
	}
	if refs := queryRefs(query); len(refs) > 0 {
		res.SrcURL = refs[0].Url
	}
	return res
}

// checkUnmapped keeps the cnspec-specific values that OCSF has no attribute for.
func (c *ocsfConverter) checkUnmapped(query *policy.Mquery, score *policy.Score, ctx *ocsfAssetContext) map[string]string {
	res := map[string]string{}
	if ctx.assetMrn != "" {
		res["asset_mrn"] = ctx.assetMrn
	}
	if query.Mrn != "" {
		res["query_mrn"] = query.Mrn
	}
	if mql := strings.TrimSpace(queryMql(query)); mql != "" {
		res["mql"] = mql
	}
	if audit := queryAudit(query); audit != "" {
		res["audit"] = audit
	}
	if impact, ok := queryImpact(query); ok {
		res["impact"] = strconv.Itoa(int(impact))
	}
	if titles := ctx.policyTitles[query.Mrn]; len(titles) > 0 {
		res["policies"] = strings.Join(titles, ", ")
	}
	if score != nil {
		res["score_type"] = score.TypeLabel()
		res["completion"] = strconv.Itoa(int(score.Completion()))
		if score.Type == policy.ScoreType_Result {
			res["score"] = strconv.Itoa(int(score.Value))
			res["risk"] = strconv.Itoa(int(scoreRisk(score)))
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

// inventoryInfo reports the asset itself, so the lake knows what was scanned even
// when the scan produced no findings.
func (c *ocsfConverter) inventoryInfo(r *policy.ReportCollection, ctx *ocsfAssetContext) ocsf.InventoryInfo {
	res := ocsf.NewInventoryInfo(ocsf.InventoryInfoActivityCollect)
	res.Cloud = ctx.cloud
	if ctx.device != nil {
		res.Device = *ctx.device
	}
	res.Time = c.now
	res.SeverityID = ocsf.SeverityInformational
	res.Severity = ocsf.SeverityName(res.SeverityID)
	// device is a class attribute of inventory_info, not a host-profile one, so
	// only the cloud profile has to be declared here.
	var profiles []string
	if ctx.cloud != nil {
		profiles = append(profiles, ocsf.ProfileCloud)
	}
	res.Metadata = c.metadata(profiles...)
	res.Unmapped = c.assetUnmapped(r, ctx)
	return res
}

// assetUnmapped carries the asset details and, when data output is enabled, the
// results of the data-only queries of the scan.
func (c *ocsfConverter) assetUnmapped(r *policy.ReportCollection, ctx *ocsfAssetContext) map[string]string {
	res := map[string]string{}
	if ctx.assetMrn != "" {
		res["asset_mrn"] = ctx.assetMrn
	}
	asset := ctx.asset
	if len(asset.PlatformIds) > 0 {
		res["platform_ids"] = strings.Join(asset.PlatformIds, ",")
	}
	for k, v := range asset.Labels {
		res["label."+k] = v
	}
	for k, v := range asset.Annotations {
		res["annotation."+k] = v
	}
	if asset.Platform != nil {
		if asset.Platform.Kind != "" {
			res["platform_kind"] = asset.Platform.Kind
		}
		if asset.Platform.Runtime != "" {
			res["platform_runtime"] = asset.Platform.Runtime
		}
	}

	if !c.includeData {
		return res
	}
	for mrn, value := range assetDataResults(r, ctx.assetMrn) {
		res["data."+mrn] = value
	}
	return res
}

// assetDataResults renders the results of the asset's data-only queries as JSON,
// keyed by query MRN.
func assetDataResults(r *policy.ReportCollection, assetMrn string) map[string]string {
	report, ok := r.Reports[assetMrn]
	if !ok {
		return nil
	}
	resolved, ok := r.ResolvedPolicies[assetMrn]
	if !ok || resolved.ExecutionJob == nil || resolved.CollectorJob == nil {
		return nil
	}

	qid2mrn := map[string]string{}
	if r.Bundle != nil {
		for _, query := range r.Bundle.Queries {
			if query.CodeId != "" {
				qid2mrn[query.CodeId] = query.Mrn
			}
		}
	}

	reportingJobs := map[string]*policy.ReportingJob{}
	for _, job := range resolved.CollectorJob.ReportingJobs {
		reportingJobs[job.QrId] = job
	}

	results := report.RawResults()
	res := map[string]string{}
	for qid, query := range resolved.ExecutionJob.Queries {
		mrn := qid2mrn[qid]
		if mrn == "" {
			continue
		}
		if job, ok := reportingJobs[mrn]; ok &&
			job.Type != policy.ReportingJob_DATA_QUERY && job.Type != policy.ReportingJob_CHECK_AND_DATA_QUERY {
			continue
		}

		buf := &bytes.Buffer{}
		w := iox.IOWriter{Writer: buf}
		if err := cr.CodeBundleToJSON(query.Code, results, &w); err != nil {
			log.Warn().Err(err).Str("query", mrn).Msg("could not render a data query result for the OCSF report")
			continue
		}
		res[mrn] = strings.TrimSpace(buf.String())
	}
	return res
}

// addVulnerabilityFindings emits one finding per advisory that affects the asset,
// plus one for every affected package no advisory in the report accounts for.
func (c *ocsfConverter) addVulnerabilityFindings(events *ocsf.Events, vulnReport *mvd.VulnReport, ctx *ocsfAssetContext) {
	if vulnReport == nil {
		return
	}

	affected := map[string]*mvd.Package{}
	byName := map[string]*mvd.Package{}
	for _, pkg := range vulnReport.Packages {
		if pkg == nil || !pkg.Affected {
			continue
		}
		affected[vulnPackageKey(pkg)] = pkg
		byName[pkg.Name] = pkg
	}
	if len(affected) == 0 {
		return
	}

	advisories := make([]*mvd.Advisory, 0, len(vulnReport.Advisories))
	for _, advisory := range vulnReport.Advisories {
		if advisory != nil && advisory.ID != "" {
			advisories = append(advisories, advisory)
		}
	}
	sort.Slice(advisories, func(i, j int) bool { return advisories[i].ID < advisories[j].ID })

	covered := map[string]bool{}
	for _, advisory := range advisories {
		pkgs := advisoryPackages(advisory, affected, byName)
		if len(pkgs) == 0 {
			continue
		}
		for _, pkg := range pkgs {
			covered[vulnPackageKey(pkg)] = true
		}
		events.VulnerabilityFindings = append(events.VulnerabilityFindings, c.vulnerabilityFinding(advisory, pkgs, ctx))
	}

	var remaining []string
	for key := range affected {
		if !covered[key] {
			remaining = append(remaining, key)
		}
	}
	if len(remaining) == 0 {
		return
	}

	// A vulnerability with neither a CVE nor an advisory has nothing to identify
	// it, and OCSF rejects such an object (1.3: at least one of cve/cwe, 1.9:
	// exactly one of advisory/cve/cwe). Rather than invent an identifier, report
	// the gap: the packages still show up in every other output format.
	sort.Strings(remaining)
	log.Warn().
		Strs("packages", remaining).
		Msg("no advisory covers these affected packages, skipping them in the OCSF vulnerability findings")
}

// vulnerabilityFinding builds one finding for an advisory and every package of
// it the asset is affected by.
func (c *ocsfConverter) vulnerabilityFinding(advisory *mvd.Advisory, pkgs []*mvd.Package, ctx *ocsfAssetContext) ocsf.VulnerabilityFinding {
	score := int32(0)
	uid := "vulnerable-package"
	title := "Vulnerable package"
	desc := "An installed package is affected by known vulnerabilities."
	if len(pkgs) > 0 {
		score = pkgs[0].Score
		uid = "vulnerable-package/" + vulnPackageKey(pkgs[0])
		title = pkgs[0].Name + " " + pkgs[0].Version + " has known vulnerabilities"
	}
	if advisory != nil {
		if advisory.Score > 0 {
			score = advisory.Score
		}
		uid = advisory.ID
		title = advisory.Title
		if title == "" {
			title = advisory.ID
		}
		if advisory.Description != "" {
			desc = advisory.Description
		}
	}

	packages := make([]ocsf.AffectedPackage, 0, len(pkgs))
	fixAvailable := false
	for _, pkg := range pkgs {
		packages = append(packages, ocsf.AffectedPackage{
			Name:           pkg.Name,
			Version:        pkg.Version,
			Architecture:   pkg.Arch,
			FixedInVersion: pkg.Available,
			PackageManager: pkg.Format,
		})
		if pkg.Available != "" {
			fixAvailable = true
		}
	}

	published := ocsfParseTime(advisoryPublished(advisory))
	vuln := ocsf.Vulnerability{
		Title:            title,
		Desc:             desc,
		Severity:         ocsf.SeverityName(ocsfSeverityFromRisk(score)),
		AffectedPackages: packages,
		IsFixAvailable:   fixAvailable,
		FirstSeenTime:    published,
		References:       advisoryRefs(advisory),
	}

	vulns := []ocsf.Vulnerability{}
	if advisory != nil {
		for _, cve := range advisoryCves(advisory) {
			cur := vuln
			cur.CVE = &ocsf.CVE{
				UID:          cve.ID,
				Desc:         cve.Summary,
				CreatedTime:  ocsfParseTime(cve.Published),
				ModifiedTime: ocsfParseTime(cve.Modified),
				CVSS:         ocsfCvss(cve),
			}
			if cve.Url != "" {
				cur.References = append(append([]string{}, cur.References...), cve.Url)
			}
			vulns = append(vulns, cur)
		}
		if len(vulns) == 0 {
			// An advisory with no CVEs still has to identify itself. 1.9 has the
			// advisory object for exactly this; 1.3 has no such attribute, so the
			// advisory id goes in cve.uid, which is where every OCSF producer puts
			// a non-CVE vulnerability identifier.
			cur := vuln
			if c.version.AtLeast(ocsf.Version190) {
				cur.Advisory = &ocsf.Advisory{
					UID:        advisory.ID,
					Title:      advisory.Title,
					Desc:       advisory.Description,
					References: advisoryRefs(advisory),
				}
			} else {
				cur.CVE = &ocsf.CVE{UID: advisory.ID, Title: advisory.Title, Desc: advisory.Description}
			}
			vulns = append(vulns, cur)
		}
	}

	finding := ocsf.NewVulnerabilityFinding(ocsf.VulnerabilityFindingActivityCreate)
	finding.FindingInfo = ocsf.FindingInfo{
		UID:           uid,
		Title:         title,
		Desc:          desc,
		CreatedTime:   c.now,
		FirstSeenTime: published,
		Types:         []string{"Vulnerability"},
		DataSources:   []string{ocsfProductName},
	}
	finding.Vulnerabilities = vulns
	finding.Device = ctx.device
	finding.Cloud = ctx.cloud
	if ctx.resource.UID != "" || ctx.resource.Name != "" {
		finding.Resources = []ocsf.ResourceDetails{ctx.resource}
	}
	finding.Time = c.now
	finding.SeverityID = ocsfSeverityFromRisk(score)
	finding.Severity = ocsf.SeverityName(finding.SeverityID)
	finding.StatusID = ocsf.StatusNew
	finding.Status = ocsf.StatusName(finding.StatusID)
	finding.StatusCode = "FAIL"
	finding.Message = title
	finding.Metadata = c.metadata(ctx.findingProfiles()...)

	unmapped := map[string]string{"cvss_score": strconv.Itoa(int(score))}
	if ctx.assetMrn != "" {
		unmapped["asset_mrn"] = ctx.assetMrn
	}
	if advisory != nil {
		unmapped["advisory"] = advisory.ID
	}
	finding.Unmapped = unmapped

	return finding
}

func advisoryPublished(advisory *mvd.Advisory) string {
	if advisory == nil {
		return ""
	}
	return advisory.Published
}

func advisoryRefs(advisory *mvd.Advisory) []string {
	if advisory == nil {
		return nil
	}
	var res []string
	for _, ref := range advisory.Refs {
		if ref != nil && ref.Url != "" {
			res = append(res, ref.Url)
		}
	}
	sort.Strings(res)
	return res
}

func ocsfCvss(cve *mvd.CVE) []ocsf.CVSS {
	var res []ocsf.CVSS
	for _, score := range cve.Cvss {
		if score == nil || score.Vector == "" {
			continue
		}
		res = append(res, ocsf.CVSS{
			Version:      score.Version(),
			BaseScore:    float64(score.Score),
			VectorString: score.Vector,
			Severity:     score.Severity().String(),
		})
	}
	return res
}

// maxAssessmentBytes caps the "expected vs actual" detail of a check. A check
// that fails across thousands of resources can render megabytes of assessment,
// which bloats every row of the Parquet file and can exceed what an HTTP
// collector accepts in one event. The head of it is what makes a finding
// actionable; the rest is repetition.
const maxAssessmentBytes = 64 * 1024

// checkAssessment renders the "expected vs actual" detail of a check, which is
// what makes a failing finding actionable.
func checkAssessment(resolved *policy.ResolvedPolicy, report *policy.Report, query *policy.Mquery) string {
	if resolved == nil || resolved.ExecutionJob == nil {
		return ""
	}
	codeBundle := resolved.GetCodeBundle(query)
	if codeBundle == nil {
		return ""
	}
	assessment := policy.Query2Assessment(codeBundle, report)
	if assessment == nil {
		return ""
	}
	return truncateAssessment(strings.TrimSpace(printer.PlainNoColorPrinter.Assessment(codeBundle, assessment)))
}

func truncateAssessment(detail string) string {
	if len(detail) <= maxAssessmentBytes {
		return detail
	}
	// Cut on a rune boundary so the result stays valid UTF-8 and valid JSON.
	cut := maxAssessmentBytes
	for cut > 0 && !utf8.RuneStart(detail[cut]) {
		cut--
	}
	return detail[:cut] + "\n… assessment truncated"
}

func refURLs(query *policy.Mquery) []string {
	refs := queryRefs(query)
	res := make([]string, 0, len(refs))
	for _, ref := range refs {
		res = append(res, ref.Url)
	}
	return res
}

// ocsfCheckSeverity maps a check outcome to an OCSF severity. A check that passed
// or did not run is informational; a failing one carries the severity of its risk.
func ocsfCheckSeverity(score *policy.Score) int {
	if score == nil {
		return ocsf.SeverityInformational
	}
	switch score.Type {
	case policy.ScoreType_Error:
		// The check itself could not be evaluated. That is a gap in coverage, not
		// a security finding of its own.
		return ocsf.SeverityMedium
	case policy.ScoreType_Result:
		if score.Value == 100 {
			return ocsf.SeverityInformational
		}
		return ocsfSeverityFromRisk(scoreRisk(score))
	default:
		return ocsf.SeverityInformational
	}
}

// ocsfSeverityFromRisk maps a cnspec risk value (0-100) to an OCSF severity_id.
// The bands are the ones riskSeverityLabel uses, so the OCSF severity and the
// severity cnspec prints stay in sync.
func ocsfSeverityFromRisk(risk int32) int {
	switch {
	case risk >= 90:
		return ocsf.SeverityCritical
	case risk >= 70:
		return ocsf.SeverityHigh
	case risk >= 40:
		return ocsf.SeverityMedium
	case risk >= 1:
		return ocsf.SeverityLow
	default:
		return ocsf.SeverityInformational
	}
}

// ocsfFindingStatus maps a check outcome to the finding status and the label
// that goes with it. Findings cnspec produces are always newly observed; a
// skipped check is reported as suppressed, which is what "we deliberately did
// not evaluate this" means in OCSF, and a check that errored has no outcome.
//
// The label of an "Other" status is the cnspec outcome, not the word "Other":
// OCSF expects the string sibling of an Other enum to carry the producer's own
// value, and a validator flags one that just repeats the caption.
func ocsfFindingStatus(score *policy.Score) (int, string) {
	if score.GetType() == policy.ScoreType_Error {
		return ocsf.StatusOther, "Error"
	}
	switch scoreToSarifKind(score) {
	case "pass", "fail":
		return ocsf.StatusNew, ocsf.StatusName(ocsf.StatusNew)
	case "notApplicable":
		return ocsf.StatusSuppressed, ocsf.StatusName(ocsf.StatusSuppressed)
	case "informational":
		return ocsf.StatusOther, "Unscored"
	default:
		return ocsf.StatusUnknown, ocsf.StatusName(ocsf.StatusUnknown)
	}
}

// ocsfComplianceStatus maps a check outcome to the compliance verdict and its
// label. An errored check is deliberately not a Fail: nothing was evaluated, so
// calling it non-compliant would report an outage as a violation.
func ocsfComplianceStatus(score *policy.Score) (int, string) {
	if score.GetType() == policy.ScoreType_Error {
		return ocsf.ComplianceStatusUnknown, ocsf.ComplianceStatusName(ocsf.ComplianceStatusUnknown)
	}
	switch scoreToSarifKind(score) {
	case "pass":
		return ocsf.ComplianceStatusPass, ocsf.ComplianceStatusName(ocsf.ComplianceStatusPass)
	case "fail":
		return ocsf.ComplianceStatusFail, ocsf.ComplianceStatusName(ocsf.ComplianceStatusFail)
	case "notApplicable":
		return ocsf.ComplianceStatusOther, "Skipped"
	default:
		return ocsf.ComplianceStatusUnknown, ocsf.ComplianceStatusName(ocsf.ComplianceStatusUnknown)
	}
}

// ocsfMillis normalizes a cnspec timestamp to milliseconds. cnspec stores some of
// them in seconds and OCSF wants milliseconds throughout; anything below the
// year 5138 in seconds is far below a plausible millisecond timestamp, so the
// magnitude tells the two apart.
func ocsfMillis(ts int64) int64 {
	if ts <= 0 {
		return 0
	}
	if ts < 1e11 {
		return ts * 1000
	}
	return ts
}

// ocsfParseTime turns one of the date strings the vulnerability API returns into
// milliseconds since the epoch. Unparsable values are dropped rather than
// reported as the zero time, which would show up as the year 1 in the lake.
func ocsfParseTime(raw string) int64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05", "2006-01-02 15:04:05", "2006-01-02"} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.UnixMilli()
		}
	}
	return 0
}

// buildOcsfResource describes the scanned asset as the resource a finding is about.
func buildOcsfResource(asset *inventory.Asset) ocsf.ResourceDetails {
	res := ocsf.ResourceDetails{
		UID:  asset.Mrn,
		Name: asset.Name,
	}
	if res.UID == "" && len(asset.PlatformIds) > 0 {
		res.UID = asset.PlatformIds[0]
	}
	if asset.Platform != nil {
		res.Type = asset.Platform.Title
		if res.Type == "" {
			res.Type = asset.Platform.Name
		}
		res.Version = asset.Platform.Version
	}
	for _, key := range sortedKeys(asset.Labels) {
		res.Labels = append(res.Labels, key+"="+asset.Labels[key])
	}
	if arn, ok := awsARN(asset); ok {
		res.CloudPartition = arn.partition
		res.Region = arn.region
	}
	return res
}

// buildOcsfDevice describes the scanned asset as an endpoint.
func buildOcsfDevice(asset *inventory.Asset, version ocsf.Version) *ocsf.Device {
	res := &ocsf.Device{
		TypeID:   ocsf.DeviceTypeOther,
		UID:      asset.Mrn,
		Name:     asset.Name,
		Hostname: asset.Fqdn,
	}
	if res.UID == "" && len(asset.PlatformIds) > 0 {
		res.UID = asset.PlatformIds[0]
	}

	platform := asset.Platform
	if platform == nil {
		return res
	}

	switch platform.Kind {
	case inventory.AssetKindBaremetal:
		res.TypeID = ocsf.DeviceTypeServer
	case inventory.AssetKindCloudVM, "virtualmachine-image":
		res.TypeID = ocsf.DeviceTypeVirtual
	}
	res.Type = ocsfDeviceTypeName(res.TypeID, platform)

	if osType := ocsfOsType(platform); osType != ocsf.OSTypeUnknown {
		res.OS = &ocsf.OS{
			Name:    platform.Name,
			TypeID:  osType,
			Type:    ocsf.OSTypeName(osType),
			Version: platform.Version,
			Build:   platform.Build,
		}
	}
	if platform.Arch != "" {
		// 1.9 deprecates cpu_type in favor of a per-processor cpu_info_list.
		if version.AtLeast(ocsf.Version190) {
			res.HwInfo = &ocsf.DeviceHwInfo{
				CPUInfoList: []ocsf.CPUInfo{{CPUArchitecture: platform.Arch}},
			}
		} else {
			res.HwInfo = &ocsf.DeviceHwInfo{CPUType: platform.Arch}
		}
	}
	if arn, ok := awsARN(asset); ok {
		res.Region = arn.region
	}
	return res
}

// ocsfDeviceTypeName is the string sibling of device.type_id. For a device that
// is none of OCSF's known types the sibling carries what cnspec calls the
// platform, which is more useful than the word "Other" and is what OCSF asks for.
func ocsfDeviceTypeName(typeID int, platform *inventory.Platform) string {
	if typeID != ocsf.DeviceTypeOther {
		return ocsf.DeviceTypeName(typeID)
	}
	if platform == nil {
		return ""
	}
	if platform.Title != "" {
		return platform.Title
	}
	return platform.Name
}

// ocsfOsType detects the operating system family of a platform. Assets that are
// not operating systems (cloud APIs, SaaS, Kubernetes objects) have none.
func ocsfOsType(platform *inventory.Platform) int {
	families := append([]string{platform.Name}, platform.Family...)
	for _, family := range families {
		switch strings.ToLower(family) {
		case "windows":
			return ocsf.OSTypeWindows
		case "linux":
			return ocsf.OSTypeLinux
		case "darwin", "macos":
			return ocsf.OSTypeMacOS
		case "android":
			return ocsf.OSTypeAndroid
		case "solaris":
			return ocsf.OSTypeSolaris
		case "aix":
			return ocsf.OSTypeAIX
		case "hpux":
			return ocsf.OSTypeHPUX
		case "unix", "bsd":
			return ocsf.OSTypeOther
		}
	}
	return ocsf.OSTypeUnknown
}

// buildOcsfCloud fills in the cloud environment of an asset. It returns nil for
// assets that are not cloud resources, which keeps the cloud profile off those
// events.
func buildOcsfCloud(asset *inventory.Asset) *ocsf.Cloud {
	provider := ocsfCloudProvider(asset)
	if provider == "" {
		return nil
	}

	res := &ocsf.Cloud{Provider: provider}
	if arn, ok := awsARN(asset); ok {
		res.Region = arn.region
		if arn.account != "" {
			res.Account = &ocsf.Account{UID: arn.account, Type: "AWS Account"}
		}
	}
	for _, id := range asset.PlatformIds {
		if project, ok := platformIDSegment(id, "/runtime/gcp/projects/"); ok {
			res.ProjectUID = project
		}
		if sub, ok := platformIDSegment(id, "/runtime/azure/subscriptions/"); ok {
			res.Account = &ocsf.Account{UID: sub, Type: "Azure Subscription"}
		}
		if account, ok := platformIDSegment(id, "/runtime/aws/accounts/"); ok {
			res.Account = &ocsf.Account{UID: account, Type: "AWS Account"}
		}
	}
	return res
}

// ocsfCloudProvider reports the cloud an asset belongs to, using the same
// vocabulary OCSF consumers expect ("AWS", "Azure", "GCP").
func ocsfCloudProvider(asset *inventory.Asset) string {
	candidates := []string{}
	if asset.Platform != nil {
		candidates = append(candidates, asset.Platform.Runtime, asset.Platform.Name)
	}
	candidates = append(candidates, asset.PlatformIds...)

	for _, candidate := range candidates {
		switch {
		case candidate == "":
			continue
		case strings.HasPrefix(candidate, "arn:aws"), strings.HasPrefix(candidate, "aws"),
			strings.Contains(candidate, "/runtime/aws/"):
			return "AWS"
		case strings.HasPrefix(candidate, "azure"), strings.Contains(candidate, "/runtime/azure/"):
			return "Azure"
		case strings.HasPrefix(candidate, "gcp"), strings.HasPrefix(candidate, "google"),
			strings.Contains(candidate, "/runtime/gcp/"):
			return "GCP"
		}
	}
	return ""
}

type parsedARN struct {
	partition string
	region    string
	account   string
}

// awsARN pulls the partition, region and account out of the first ARN among an
// asset's platform ids. An ARN is
// arn:<partition>:<service>:<region>:<account>:<resource>, and region and account
// are empty for global and account-less resources.
func awsARN(asset *inventory.Asset) (parsedARN, bool) {
	for _, id := range asset.PlatformIds {
		if !strings.HasPrefix(id, "arn:") {
			continue
		}
		parts := strings.SplitN(id, ":", 6)
		if len(parts) < 5 {
			continue
		}
		return parsedARN{partition: parts[1], region: parts[3], account: parts[4]}, true
	}
	return parsedARN{}, false
}

// platformIDSegment returns the segment that follows a marker in a platform id,
// e.g. the project of //platformid.api.mondoo.app/runtime/gcp/projects/my-project.
func platformIDSegment(id, marker string) (string, bool) {
	idx := strings.Index(id, marker)
	if idx < 0 {
		return "", false
	}
	rest := id[idx+len(marker):]
	if end := strings.Index(rest, "/"); end >= 0 {
		rest = rest[:end]
	}
	if rest == "" {
		return "", false
	}
	return rest, true
}
