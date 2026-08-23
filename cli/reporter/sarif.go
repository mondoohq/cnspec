// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/owenrumney/go-sarif/v2/sarif"
	"go.mondoo.com/cnspec"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/cnspec/reports/reportdoc"
	"go.mondoo.com/mql/cli/printer"
	"go.mondoo.com/mql/llx"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/providers-sdk/v1/upstream/mvd"
	"go.mondoo.com/mql/utils/iox"
)

const (
	sarifAssetErrorRuleID  = "asset-error"
	sarifVulnPackageRuleID = "vulnerable-package"
	sarifInformationURI    = "https://cnspec.io"
)

// sarifFingerprintKey is the key under which cnspec stores its stable
// per-finding fingerprint in SARIF partialFingerprints. It uses a URI-style
// namespace we own so it can't collide with well-known keys used by other
// tooling (e.g. GitHub code scanning's primaryLocationLineHash). The trailing
// version lets us evolve the fingerprint algorithm without remapping old runs.
const sarifFingerprintKey = "https://cnspec.io/fingerprint/v1"

// The SARIF report carries the same content as the detailed JUnit report, split
// along the lines the SARIF spec (and its consumers) expect:
//
//	rule.shortDescription   check title
//	rule.fullDescription    check description
//	rule.help               description, MQL, audit steps, remediation, references,
//	                        compliance mappings — as text and as markdown
//	rule.helpUri            first reference of the check
//	rule.properties         severity, impact, security-severity, tags, policies,
//	                        compliance mappings, MQL
//	result.message          title, status, score, assessment (expected vs actual)
//	result.locations        source location of each failing resource, plus the
//	                        asset as a logical location
//	result.properties       score, risk, severity, status, asset
//
// Vulnerabilities from the scan's vulnerability report are emitted alongside the
// policy findings, one result per affected package (per advisory when the report
// carries advisories), which mirrors the JUnit vulnerability test suite.

// sarifRunContext carries everything a single asset's run needs while its rules
// and results are built.
type sarifRunContext struct {
	assetMrn     string
	asset        *inventory.Asset
	logicalLoc   *sarif.LogicalLocation
	platformKeys map[string]bool
	policyTitles map[string][]string
}

// ConvertToSarif converts a ReportCollection into a SARIF 2.1.0 report.
// Each scanned asset is represented as a separate SARIF run.
func ConvertToSarif(r *policy.ReportCollection, out iox.OutputHelper) error {
	report, err := sarif.New(sarif.Version210)
	if err != nil {
		return err
	}

	if r == nil {
		return writeSarif(report, out)
	}

	if r.Bundle == nil {
		return fmt.Errorf("no policy bundle found")
	}

	bundle := r.Bundle.ToMap()
	queries := reportdoc.QueryMap(bundle)
	policyTitles := reportdoc.PolicyTitlesByQuery(bundle)

	// Create one run per asset (deterministic order via sorted keys)
	assetMrns := sortedKeys(r.Assets)
	for _, assetMrn := range assetMrns {
		assetObj := r.Assets[assetMrn]
		ctx := &sarifRunContext{
			assetMrn:     assetMrn,
			asset:        assetObj,
			logicalLoc:   assetLogicalLocation(assetObj),
			platformKeys: reportdoc.PlatformRemediationKeys(assetObj.Platform),
			policyTitles: policyTitles,
		}
		run := newAssetRun(r, ctx)

		// Register the asset-error rule if this asset has an error
		if _, hasErr := r.Errors[assetMrn]; hasErr {
			run.AddRule(sarifAssetErrorRuleID).
				WithName("Asset scan error").
				WithDescription("The asset could not be scanned successfully").
				WithFullDescription(sarif.NewMultiformatMessageString(
					"cnspec could not complete the scan of this asset. No policy results are available for it.")).
				WithDefaultConfiguration(sarif.NewReportingConfiguration().WithLevel("error")).
				WithProperties(sarif.Properties{
					"tags": []string{"cnspec", "scan"},
				})
		}

		// Register reporting queries applicable to this asset as SARIF rules
		registerAssetRules(run, r, ctx, queries)

		// Emit results for this asset
		addAssetErrors(run, r, ctx)
		addAssetResults(run, r, ctx, queries)
		addAssetVulnerabilities(run, r.VulnReports[assetMrn], ctx)

		report.AddRun(run)
	}

	return writeSarif(report, out)
}

// newAssetRun creates a new SARIF run for a given asset
func newAssetRun(r *policy.ReportCollection, ctx *sarifRunContext) *sarif.Run {
	run := sarif.NewRunWithInformationURI("cnspec", sarifInformationURI)
	run.Tool.Driver.
		WithVersion(cnspec.GetVersion()).
		WithFullName("cnspec " + cnspec.GetVersion()).
		WithShortDescription(sarif.NewMultiformatMessageString(
			"cnspec is an open source, cloud-native security and policy scanner"))
	run.Tool.Driver.Organization = strPtr("Mondoo")
	// cnspec reports 1-based line and column numbers on source locations.
	run.ColumnKind = "utf16CodeUnits"

	// Tag the run with asset metadata so consumers can identify which asset it
	// covers, and correlate re-scans of the same asset across uploads.
	run.AutomationDetails = sarif.NewRunAutomationDetails().
		WithID("cnspec/" + ctx.asset.Name).
		WithDescriptionText("cnspec scan of " + ctx.asset.Name)

	props := sarif.Properties{"asset": ctx.asset.Name}
	if ctx.assetMrn != "" {
		props["assetMrn"] = ctx.assetMrn
	}
	if ctx.asset.Platform != nil {
		if platformName := reportdoc.PlatformName(ctx.asset); platformName != "" {
			props["platform"] = platformName
		}
		if ctx.asset.Platform.Version != "" {
			props["platformVersion"] = ctx.asset.Platform.Version
		}
		if ctx.asset.Platform.Arch != "" {
			props["platformArch"] = ctx.asset.Platform.Arch
		}
	}
	addRunScoreProperties(props, r, ctx.assetMrn)
	addRunVulnProperties(props, r.VulnReports[ctx.assetMrn])
	run.Properties = props

	return run
}

// addRunScoreProperties summarizes the asset's policy results (overall score and
// per-status counts) into the run's property bag.
func addRunScoreProperties(props sarif.Properties, r *policy.ReportCollection, assetMrn string) {
	report, ok := r.Reports[assetMrn]
	if !ok || report == nil {
		return
	}

	if report.Score != nil {
		props["score"] = report.Score.Value
		props["grade"] = report.Score.Rating().Letter()
	}

	resolved, ok := r.ResolvedPolicies[assetMrn]
	if !ok || resolved == nil || resolved.CollectorJob == nil {
		return
	}

	var total, passed, failed, errored, skipped int
	for id, score := range report.Scores {
		if _, ok := resolved.CollectorJob.ReportingQueries[id]; !ok {
			continue
		}
		total++
		// The counts switch on the outcome, not on the SARIF kind. SARIF has no
		// error kind, so scoreToSarifKind folds Error into "fail"; counting off it
		// meant reaching back into policy.ScoreType to undo that fold right after
		// the helper had discarded it. reportdoc.Outcome is the shared collapse
		// that keeps the error case distinct, which is the whole reason it exists.
		switch reportdoc.OutcomeOf(score) {
		case reportdoc.OutcomePass:
			passed++
		case reportdoc.OutcomeFail:
			failed++
		case reportdoc.OutcomeError:
			errored++
		case reportdoc.OutcomeSkipped:
			skipped++
		}
	}

	props["checksTotal"] = total
	props["checksPassed"] = passed
	props["checksFailed"] = failed
	props["checksErrored"] = errored
	props["checksSkipped"] = skipped
}

// addRunVulnProperties summarizes the asset's vulnerability report into the run's
// property bag, mirroring the properties of the JUnit vulnerability test suite.
func addRunVulnProperties(props sarif.Properties, vulnReport *mvd.VulnReport) {
	if vulnReport == nil || vulnReport.Stats == nil {
		return
	}

	// Platforms without a package inventory (Terraform, cloud APIs, ...) report
	// empty stats; don't clutter their runs with zeros.
	if pkgs := vulnReport.Stats.Packages; pkgs != nil && pkgs.Total > 0 {
		props["packagesTotal"] = pkgs.Total
		props["packagesAffected"] = pkgs.Affected
		props["packagesCritical"] = pkgs.Critical
		props["packagesHigh"] = pkgs.High
		props["packagesMedium"] = pkgs.Medium
		props["packagesLow"] = pkgs.Low
		props["packagesNone"] = pkgs.None
	}
	if advisories := vulnReport.Stats.Advisories; advisories != nil && advisories.Total > 0 {
		props["advisoriesTotal"] = advisories.Total
	}
	if cves := vulnReport.Stats.Cves; cves != nil && cves.Total > 0 {
		props["cvesTotal"] = cves.Total
	}
}

// registerAssetRules registers the reporting queries for a single asset as SARIF rules on the run
func registerAssetRules(run *sarif.Run, r *policy.ReportCollection, ctx *sarifRunContext, queries map[string]*policy.Mquery) {
	resolved, ok := r.ResolvedPolicies[ctx.assetMrn]
	if !ok || resolved.CollectorJob == nil {
		return
	}
	queryIDs := sortedKeys(resolved.CollectorJob.ReportingQueries)
	for _, id := range queryIDs {
		query, ok := queries[id]
		if !ok {
			continue
		}
		registerCheckRule(run, query, ctx)
	}
}

// registerCheckRule turns a check into a SARIF rule that carries everything a
// consumer needs to render and act on the finding: title, description, help
// (query, audit steps, remediation, references, compliance mappings), severity
// and the policies it belongs to.
func registerCheckRule(run *sarif.Run, query *policy.Mquery, ctx *sarifRunContext) {
	ruleID := reportdoc.QueryRuleID(query)
	rb := run.AddRule(ruleID)

	title := query.Title
	if title == "" {
		title = ruleID
	}
	rb.WithName(title).WithDescription(title)

	if desc := strings.TrimSpace(reportdoc.QueryDescription(query)); desc != "" {
		rb.WithFullDescription(sarif.NewMultiformatMessageString(desc).WithMarkdown(desc))
	}

	if text, markdown := checkHelp(query, ctx); text != "" {
		rb.WithHelp(sarif.NewMultiformatMessageString(text).WithMarkdown(markdown))
	}

	if refs := reportdoc.QueryRefs(query); len(refs) > 0 {
		rb.WithHelpURI(refs[0].Url)
	}

	props := sarif.Properties{"tags": checkRuleTags(query, ctx)}
	if impact, ok := reportdoc.QueryImpact(query); ok {
		props["impact"] = impact
		props["severity"] = reportdoc.RiskSeverityLabel(impact)
		// GitHub code scanning reads the alert severity from this property.
		props["security-severity"] = securitySeverity(impact)
		rb.WithDefaultConfiguration(sarif.NewReportingConfiguration().WithLevel(riskSarifLevel(impact)))
	}
	if query.Mrn != "" {
		props["queryMrn"] = query.Mrn
	}
	if mql := strings.TrimSpace(reportdoc.QueryMql(query)); mql != "" {
		props["mql"] = mql
	}
	if titles := ctx.policyTitles[query.Mrn]; len(titles) > 0 {
		props["policies"] = titles
	}
	if compliance := reportdoc.QueryComplianceTags(query); len(compliance) > 0 {
		props["compliance"] = compliance
	}
	if rem := reportdoc.QueryRemediation(query, ctx.platformKeys); rem != "" {
		props["remediation"] = rem
	}
	rb.WithProperties(props)
}

// checkRuleTags builds the rule tags. Consumers use them to group and filter
// findings; "security" in particular is what makes GitHub code scanning treat the
// rule as a security alert.
func checkRuleTags(query *policy.Mquery, ctx *sarifRunContext) []string {
	tags := []string{"security", "cnspec"}
	for _, title := range ctx.policyTitles[query.Mrn] {
		tags = append(tags, "policy/"+title)
	}
	// the compliance tag keys are already namespaced, e.g. "compliance/iso-27001-2022"
	return append(tags, sortedKeys(reportdoc.QueryComplianceTags(query))...)
}

// checkHelp renders the static documentation of a check as plain text and as
// markdown. SARIF consumers show this next to every finding of the rule, so it
// carries the same sections as the detailed JUnit failure body.
func checkHelp(query *policy.Mquery, ctx *sarifRunContext) (string, string) {
	var text, md strings.Builder

	desc := strings.TrimSpace(reportdoc.QueryDescription(query))
	if desc != "" {
		text.WriteString(desc + "\n")
		md.WriteString(desc + "\n")
	}

	if impact, ok := reportdoc.QueryImpact(query); ok {
		severity := reportdoc.RiskSeverityLabel(impact) + " (impact " + strconv.Itoa(int(impact)) + ")"
		reportdoc.WriteDetailSection(&text, "Severity", severity)
		writeMarkdownSection(&md, "Severity", severity)
	}

	if titles := ctx.policyTitles[query.Mrn]; len(titles) > 0 {
		reportdoc.WriteDetailSection(&text, "Policies", strings.Join(titles, "\n"))
		writeMarkdownSection(&md, "Policies", markdownList(titles))
	}

	if mql := strings.TrimSpace(reportdoc.QueryMql(query)); mql != "" {
		reportdoc.WriteDetailSection(&text, "Query", mql)
		writeMarkdownSection(&md, "Query", markdownCode(mql))
	}

	if audit := reportdoc.QueryAudit(query); audit != "" {
		reportdoc.WriteDetailSection(&text, "Audit", audit)
		writeMarkdownSection(&md, "Audit", audit)
	}

	reportdoc.WriteDetailSection(&text, "Remediation", reportdoc.QueryRemediation(query, ctx.platformKeys))
	writeMarkdownSection(&md, "Remediation", markdownRemediation(query, ctx.platformKeys))

	reportdoc.WriteDetailSection(&text, "References", reportdoc.QueryReferences(query))
	writeMarkdownSection(&md, "References", markdownReferences(query))

	if compliance := reportdoc.QueryComplianceTags(query); len(compliance) > 0 {
		var lines []string
		for _, framework := range sortedKeys(compliance) {
			lines = append(lines, strings.TrimPrefix(framework, "compliance/")+": "+compliance[framework])
		}
		reportdoc.WriteDetailSection(&text, "Compliance", strings.Join(lines, "\n"))
		writeMarkdownSection(&md, "Compliance", markdownList(lines))
	}

	return strings.TrimSpace(text.String()), strings.TrimSpace(md.String())
}

func addAssetErrors(run *sarif.Run, r *policy.ReportCollection, ctx *sarifRunContext) {
	errMsg, ok := r.Errors[ctx.assetMrn]
	if !ok {
		return
	}
	text := fmt.Sprintf("Asset %s: %s", ctx.asset.Name, errMsg)
	markdown := "**" + ctx.asset.Name + "** could not be scanned\n\n" + markdownCode(errMsg)
	result := sarif.NewRuleResult(sarifAssetErrorRuleID).
		WithLevel("error").
		WithKind("fail").
		WithMessage(sarif.NewTextMessage(text).WithMarkdown(markdown)).
		WithLocations([]*sarif.Location{
			sarif.NewLocation().WithLogicalLocations([]*sarif.LogicalLocation{ctx.logicalLoc}),
		}).
		WithPartialFingerPrints(map[string]interface{}{
			sarifFingerprintKey: sarifFingerprint(sarifAssetErrorRuleID, ctx.assetMrn),
		})
	run.AddResult(result)
}

func addAssetResults(run *sarif.Run, r *policy.ReportCollection, ctx *sarifRunContext, queries map[string]*policy.Mquery) {
	report, ok := r.Reports[ctx.assetMrn]
	if !ok {
		return
	}

	resolved, ok := r.ResolvedPolicies[ctx.assetMrn]
	if !ok || resolved.CollectorJob == nil {
		return
	}

	// Sort score IDs for deterministic output
	scoreIDs := sortedKeys(report.Scores)
	for _, id := range scoreIDs {
		score := report.Scores[id]

		if _, ok := resolved.CollectorJob.ReportingQueries[id]; !ok {
			continue
		}

		query, ok := queries[id]
		if !ok {
			continue
		}

		ruleID := reportdoc.QueryRuleID(query)
		level := scoreToSarifLevel(score)
		kind := scoreToSarifKind(score)

		// Build the assessment once and reuse it for both the human-readable
		// detail and the structured source locations.
		var assessment *llx.Assessment
		var codeBundle *llx.CodeBundle
		if resolved.ExecutionJob != nil {
			codeBundle = resolved.GetCodeBundle(query)
			if codeBundle != nil {
				assessment = policy.Query2Assessment(codeBundle, report)
			}
		}

		var detail string
		if assessment != nil && codeBundle != nil {
			detail = strings.TrimSpace(printer.PlainNoColorPrinter.Assessment(codeBundle, assessment))
		}
		props := checkResultProperties(query, score, ctx)

		// Source locations of the failing resources. This covers terraform and
		// any resource that carries @context data; it is empty for scalar checks
		// or resources without context.
		var locations []llx.SourceContext
		if assessment != nil && codeBundle != nil {
			for _, sc := range codeBundle.FailingResourceContexts(assessment) {
				if sc.Path != "" {
					locations = append(locations, sc)
				}
			}
		}

		// The assessment covers all failing resources at once, so repeating it on
		// every location of a check that fails on many resources would grow the
		// report quadratically. Those results carry the offending source snippet in
		// their own region instead.
		if len(locations) > 1 {
			detail = ""
		}
		text, markdown := checkResultMessage(query, score, detail)

		if len(locations) == 0 {
			// No source context: anchor a single result to the asset. The full
			// assessment detail (expected vs actual) travels in the message.
			result := sarif.NewRuleResult(ruleID).
				WithLevel(level).
				WithKind(kind).
				WithMessage(sarif.NewTextMessage(text).WithMarkdown(markdown)).
				WithLocations([]*sarif.Location{
					sarif.NewLocation().WithLogicalLocations([]*sarif.LogicalLocation{ctx.logicalLoc}),
				}).
				WithPartialFingerPrints(map[string]interface{}{
					sarifFingerprintKey: sarifFingerprint(ruleID, ctx.assetMrn),
				})
			result.Properties = props
			withRiskRank(result, score)
			run.AddResult(result)
			continue
		}

		// One result per failing resource, each pointing at its exact source.
		// The code snippet travels in the region.
		for i := range locations {
			loc := sarif.NewLocationWithPhysicalLocation(physicalLocationFromContext(locations[i])).
				WithLogicalLocations([]*sarif.LogicalLocation{ctx.logicalLoc})
			result := sarif.NewRuleResult(ruleID).
				WithLevel(level).
				WithKind(kind).
				WithMessage(sarif.NewTextMessage(text).WithMarkdown(markdown)).
				WithLocations([]*sarif.Location{loc}).
				WithPartialFingerPrints(map[string]interface{}{
					sarifFingerprintKey: sarifLocationFingerprint(ruleID, locations[i]),
				})
			result.Properties = props
			withRiskRank(result, score)
			run.AddResult(result)
		}
	}
}

// checkResultMessage renders the dynamic part of a finding — what the check is,
// whether it passed, and how the actual state differed from the expected one — as
// plain text and as markdown.
func checkResultMessage(query *policy.Mquery, score *policy.Score, detail string) (string, string) {
	title := query.Title
	if title == "" {
		title = reportdoc.QueryRuleID(query)
	}

	// "FAIL · CRITICAL · score 0/100 · <what the check reported>"
	status := scoreStatusLabel(score)
	var details string
	if score != nil && score.Type == policy.ScoreType_Result {
		details += " · " + reportdoc.RiskSeverityLabel(reportdoc.ScoreRisk(score)) + " · score " + strconv.Itoa(int(score.Value)) + "/100"
	}
	if msg := score.MessageLine(); msg != "" {
		details += " · " + msg
	}

	text := title + ": " + status + details
	markdown := "**" + title + "**\n\n" + scoreStatusIcon(score) + " **" + status + "**" + details

	if detail != "" {
		text += "\n\n" + detail
		markdown += "\n\n" + markdownCode(detail)
	}

	return text, markdown
}

// checkResultProperties captures the scoring outcome of a single check so
// consumers can filter and sort findings without re-deriving it from the level.
func checkResultProperties(query *policy.Mquery, score *policy.Score, ctx *sarifRunContext) sarif.Properties {
	props := sarif.Properties{
		"asset":  ctx.asset.Name,
		"status": strings.ToLower(scoreStatusLabel(score)),
	}
	if ctx.assetMrn != "" {
		props["assetMrn"] = ctx.assetMrn
	}
	if score != nil {
		props["scoreType"] = score.TypeLabel()
		props["completion"] = score.Completion()
		if score.Type == policy.ScoreType_Result {
			props["score"] = score.Value
			props["risk"] = reportdoc.ScoreRisk(score)
			props["severity"] = reportdoc.RiskSeverityLabel(reportdoc.ScoreRisk(score))
		}
	}
	if titles := ctx.policyTitles[query.Mrn]; len(titles) > 0 {
		props["policies"] = titles
	}
	return props
}

// withRiskRank sets the SARIF rank (0-100, higher is more important) of failing
// results to the check's risk, so viewers that sort by rank show the most
// dangerous findings first.
func withRiskRank(result *sarif.Result, score *policy.Score) {
	if score == nil || score.Type != policy.ScoreType_Result || score.Value == 100 {
		return
	}
	result.WithRank(float32(reportdoc.ScoreRisk(score)))
}

// addAssetVulnerabilities emits the asset's vulnerability findings: one result per
// affected package and advisory, falling back to a single rule for affected
// packages that no advisory in the report accounts for.
func addAssetVulnerabilities(run *sarif.Run, vulnReport *mvd.VulnReport, ctx *sarifRunContext) {
	if vulnReport == nil {
		return
	}

	// Index the packages the asset is actually affected by. Advisories carry
	// their own package lists, which we intersect with this set.
	affected := map[string]*mvd.Package{}
	byName := map[string]*mvd.Package{}
	for _, pkg := range vulnReport.Packages {
		if pkg == nil || !pkg.Affected {
			continue
		}
		affected[reportdoc.VulnPackageKey(pkg)] = pkg
		byName[pkg.Name] = pkg
	}
	if len(affected) == 0 {
		return
	}

	covered := map[string]bool{}

	advisories := make([]*mvd.Advisory, 0, len(vulnReport.Advisories))
	for _, advisory := range vulnReport.Advisories {
		if advisory != nil && advisory.ID != "" {
			advisories = append(advisories, advisory)
		}
	}
	sort.Slice(advisories, func(i, j int) bool { return advisories[i].ID < advisories[j].ID })

	for _, advisory := range advisories {
		pkgs := reportdoc.AdvisoryPackages(advisory, affected, byName)
		if len(pkgs) == 0 {
			continue
		}

		registerAdvisoryRule(run, advisory)
		for _, pkg := range pkgs {
			covered[reportdoc.VulnPackageKey(pkg)] = true
			run.AddResult(vulnResult(advisory.ID, advisory, pkg, ctx))
		}
	}

	// Affected packages that no advisory in this report accounts for.
	var remaining []string
	for key := range affected {
		if !covered[key] {
			remaining = append(remaining, key)
		}
	}
	if len(remaining) == 0 {
		return
	}
	sort.Strings(remaining)

	run.AddRule(sarifVulnPackageRuleID).
		WithName("Vulnerable package").
		WithDescription("An installed package has known vulnerabilities").
		WithFullDescription(sarif.NewMultiformatMessageString(
			"An installed package is affected by known vulnerabilities. Update it to a fixed version.")).
		WithProperties(sarif.Properties{"tags": []string{"security", "cnspec", "vulnerability"}})

	for _, key := range remaining {
		run.AddResult(vulnResult(sarifVulnPackageRuleID, nil, affected[key], ctx))
	}
}

// registerAdvisoryRule registers an advisory (e.g. USN-5825-1) as a SARIF rule,
// carrying its description, CVEs and references.
func registerAdvisoryRule(run *sarif.Run, advisory *mvd.Advisory) {
	title := advisory.Title
	if title == "" {
		title = advisory.ID
	}

	rb := run.AddRule(advisory.ID).WithName(title).WithDescription(title)
	if desc := strings.TrimSpace(advisory.Description); desc != "" {
		rb.WithFullDescription(sarif.NewMultiformatMessageString(desc).WithMarkdown(desc))
	}

	var text, md strings.Builder
	if desc := strings.TrimSpace(advisory.Description); desc != "" {
		text.WriteString(desc + "\n")
		md.WriteString(desc + "\n")
	}

	severity := reportdoc.RiskSeverityLabel(advisory.Score) + " (score " + securitySeverity(advisory.Score) + ")"
	reportdoc.WriteDetailSection(&text, "Severity", severity)
	writeMarkdownSection(&md, "Severity", severity)

	cves := reportdoc.AdvisoryCves(advisory)
	if len(cves) > 0 {
		var textLines, mdLines []string
		for _, cve := range cves {
			line := cve.ID
			if cve.Score > 0 {
				line += " (CVSS " + strconv.FormatFloat(float64(cve.Score), 'f', 1, 32) + ")"
			}
			textLines = append(textLines, strings.TrimSpace(line+" "+cve.Url))
			if cve.Url != "" {
				line = "[" + line + "](" + cve.Url + ")"
			}
			mdLines = append(mdLines, line)
		}
		reportdoc.WriteDetailSection(&text, "CVEs", strings.Join(textLines, "\n"))
		writeMarkdownSection(&md, "CVEs", markdownList(mdLines))
	}

	var refLines, refMdLines []string
	for _, ref := range advisory.Refs {
		if ref == nil || ref.Url == "" {
			continue
		}
		refTitle := ref.Title
		if refTitle == "" {
			refTitle = ref.Url
		}
		refLines = append(refLines, refTitle+": "+ref.Url)
		refMdLines = append(refMdLines, "["+refTitle+"]("+ref.Url+")")
	}
	reportdoc.WriteDetailSection(&text, "References", strings.Join(refLines, "\n"))
	writeMarkdownSection(&md, "References", markdownList(refMdLines))

	if help := strings.TrimSpace(text.String()); help != "" {
		rb.WithHelp(sarif.NewMultiformatMessageString(help).WithMarkdown(strings.TrimSpace(md.String())))
	}
	if len(advisory.Refs) > 0 && advisory.Refs[0] != nil && advisory.Refs[0].Url != "" {
		rb.WithHelpURI(advisory.Refs[0].Url)
	} else if len(cves) > 0 && cves[0].Url != "" {
		rb.WithHelpURI(cves[0].Url)
	}

	props := sarif.Properties{
		"tags":              []string{"security", "cnspec", "vulnerability", "advisory"},
		"security-severity": securitySeverity(advisory.Score),
		"severity":          reportdoc.RiskSeverityLabel(advisory.Score),
		"advisory":          advisory.ID,
	}
	if len(cves) > 0 {
		ids := make([]string, 0, len(cves))
		for _, cve := range cves {
			ids = append(ids, cve.ID)
		}
		props["cves"] = ids
	}
	if advisory.Published != "" {
		props["published"] = advisory.Published
	}
	if advisory.Modified != "" {
		props["modified"] = advisory.Modified
	}
	rb.WithProperties(props)
	rb.WithDefaultConfiguration(sarif.NewReportingConfiguration().WithLevel(riskSarifLevel(advisory.Score)))
}

// vulnResult builds the finding for one affected package. When advisory is set the
// finding is attributed to it, otherwise it reports the package on its own.
func vulnResult(ruleID string, advisory *mvd.Advisory, pkg *mvd.Package, ctx *sarifRunContext) *sarif.Result {
	score := pkg.Score
	if advisory != nil && advisory.Score > 0 {
		score = advisory.Score
	}

	update := "No fixed version is available yet."
	if pkg.Available != "" {
		update = "Update to " + pkg.Available + "."
	}

	text := pkg.Name + " " + pkg.Version + " has known vulnerabilities"
	markdown := "**" + pkg.Name + "** " + pkg.Version + " has known vulnerabilities"
	if advisory != nil {
		title := advisory.Title
		if title == "" {
			title = advisory.ID
		}
		text = pkg.Name + " " + pkg.Version + " is affected by " + advisory.ID + " (" + title + ")"
		markdown = "**" + pkg.Name + "** " + pkg.Version + " is affected by **" + advisory.ID + "** — " + title
	}
	text += " · " + reportdoc.RiskSeverityLabel(score) + " (score " + securitySeverity(score) + ") · " + update
	markdown += "\n\n" + scoreIcon(score) + " **" + reportdoc.RiskSeverityLabel(score) + "**" +
		" · score " + securitySeverity(score) + " · " + update

	logicalLocs := []*sarif.LogicalLocation{
		ctx.logicalLoc,
		sarif.NewLogicalLocation().WithName(pkg.Name).WithKind("package").
			WithFullyQualifiedName(pkg.Name + "@" + pkg.Version),
	}

	props := sarif.Properties{
		"asset":             ctx.asset.Name,
		"package":           pkg.Name,
		"installedVersion":  pkg.Version,
		"severity":          reportdoc.RiskSeverityLabel(score),
		"security-severity": securitySeverity(score),
		"status":            "fail",
	}
	if ctx.assetMrn != "" {
		props["assetMrn"] = ctx.assetMrn
	}
	if pkg.Available != "" {
		props["fixedVersion"] = pkg.Available
	}
	if pkg.Arch != "" {
		props["arch"] = pkg.Arch
	}
	if pkg.Format != "" {
		props["format"] = pkg.Format
	}
	if pkg.Namespace != "" {
		props["namespace"] = pkg.Namespace
	}
	if advisory != nil {
		props["advisory"] = advisory.ID
	}

	result := sarif.NewRuleResult(ruleID).
		WithLevel(riskSarifLevel(score)).
		WithKind("fail").
		WithMessage(sarif.NewTextMessage(text).WithMarkdown(markdown)).
		WithLocations([]*sarif.Location{
			sarif.NewLocation().WithLogicalLocations(logicalLocs),
		}).
		WithPartialFingerPrints(map[string]interface{}{
			sarifFingerprintKey: sarifFingerprint(ruleID, ctx.assetMrn, reportdoc.VulnPackageKey(pkg)),
		}).
		WithRank(float32(score))
	result.Properties = props
	return result
}

// assetLogicalLocation builds the SARIF logical location that identifies an asset.
func assetLogicalLocation(assetObj *inventory.Asset) *sarif.LogicalLocation {
	logicalLoc := sarif.NewLogicalLocation().
		WithName(assetObj.Name).
		WithKind("asset")
	if assetObj.Platform != nil {
		platformName := reportdoc.PlatformName(assetObj)
		if platformName != "" {
			logicalLoc.WithFullyQualifiedName(assetObj.Name + " (" + platformName + ")")
		}
	}
	return logicalLoc
}

// physicalLocationFromContext maps an MQL source context (path + range + content)
// to a SARIF physical location with a region and, when available, a code snippet.
func physicalLocationFromContext(ctx llx.SourceContext) *sarif.PhysicalLocation {
	pl := sarif.NewPhysicalLocation().
		WithArtifactLocation(sarif.NewSimpleArtifactLocation(ctx.Path))

	startLine, startCol, endLine, endCol, hasCols, ok := ctx.Range.Bounds()
	var region *sarif.Region
	if ok && startLine >= 1 {
		region = sarif.NewRegion().
			WithStartLine(int(startLine)).
			WithEndLine(int(endLine))
		if hasCols {
			region.WithStartColumn(int(startCol)).WithEndColumn(int(endCol))
		}
	}
	if ctx.Content != "" {
		if region == nil {
			region = sarif.NewRegion()
		}
		region.WithSnippet(sarif.NewArtifactContent().WithText(ctx.Content))
	}
	if region != nil {
		pl.WithRegion(region)
	}
	return pl
}

// sarifLocationFingerprint produces a stable fingerprint for a (rule, location)
// pair so code-scanning consumers can dedup the same finding across runs.
func sarifLocationFingerprint(ruleID string, ctx llx.SourceContext) string {
	return sarifFingerprint(ruleID, ctx.Path+"#"+ctx.Range.String())
}

// sarifFingerprint hashes the parts that identify a finding into a stable
// fingerprint. Callers must pass the parts in a fixed order.
func sarifFingerprint(parts ...string) string {
	h := sha256.Sum256([]byte(strings.Join(parts, "\n")))
	return hex.EncodeToString(h[:])
}

// scoreToSarifLevel maps a cnspec Score to a SARIF level using cnspec's
// severity rating system:
//
//	100        → "none"    (pass)
//	61 .. 99   → "note"    (Low severity)
//	31 .. 60   → "warning" (Medium severity)
//	 0 .. 30   → "error"   (High/Critical severity)
func scoreToSarifLevel(score *policy.Score) string {
	if score == nil {
		return "none"
	}

	switch score.Type {
	case policy.ScoreType_Error:
		return "error"
	case policy.ScoreType_Skip, policy.ScoreType_Unscored, policy.ScoreType_OutOfScope, policy.ScoreType_Disabled:
		return "none"
	case policy.ScoreType_Unknown:
		return "none"
	case policy.ScoreType_Result:
		if score.Value == 100 {
			return "none" // pass
		}
		return riskSarifLevel(reportdoc.ScoreRisk(score))
	default:
		return "none"
	}
}

// scoreToSarifKind maps a check outcome to a SARIF result kind. The kind tells
// consumers what the result means; the level only says how loud it is. SARIF
// requires the level to be "none" for every kind other than "fail", which
// scoreToSarifLevel guarantees.
//
// SARIF's kind enum has no member for a check that could not be evaluated, so an
// errored check is reported as "fail". That is SARIF's own lossy choice and it
// stops here: the word ERROR still reaches the reader through scoreStatusLabel,
// and the other formats map reportdoc.OutcomeError to whatever they have for it
// rather than inheriting this fold. Reading the error case back out of "fail" is
// what three of them used to do.
func scoreToSarifKind(score *policy.Score) string {
	switch reportdoc.OutcomeOf(score) {
	case reportdoc.OutcomePass:
		return "pass"
	case reportdoc.OutcomeFail, reportdoc.OutcomeError:
		return "fail"
	case reportdoc.OutcomeSkipped:
		return "notApplicable"
	case reportdoc.OutcomeUnscored:
		return "informational"
	default:
		return "review"
	}
}

// scoreStatusLabel is the human-readable outcome of a check.
func scoreStatusLabel(score *policy.Score) string {
	return reportdoc.OutcomeOf(score).Label()
}

func scoreStatusIcon(score *policy.Score) string {
	switch reportdoc.OutcomeOf(score) {
	case reportdoc.OutcomePass:
		return "✅"
	case reportdoc.OutcomeFail:
		return "❌"
	case reportdoc.OutcomeError:
		return "⚠️"
	case reportdoc.OutcomeSkipped:
		return "⏭️"
	default:
		return "ℹ️"
	}
}

func scoreIcon(risk int32) string {
	if risk >= 70 {
		return "❌"
	}
	if risk >= 40 {
		return "⚠️"
	}
	return "ℹ️"
}

// writeMarkdownSection appends a "**Title**\n\nbody" section to b.
func writeMarkdownSection(b *strings.Builder, title, body string) {
	body = strings.TrimSpace(body)
	if body == "" {
		return
	}
	if b.Len() > 0 {
		b.WriteString("\n")
	}
	b.WriteString("**" + title + "**\n\n")
	b.WriteString(body)
	b.WriteString("\n")
}

func markdownList(items []string) string {
	var b strings.Builder
	for _, item := range items {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString("- " + item)
	}
	return b.String()
}

// markdownCode wraps content in a fenced block, growing the fence if the content
// itself contains one.
func markdownCode(content string) string {
	fence := "```"
	for strings.Contains(content, fence) {
		fence += "`"
	}
	return fence + "\n" + strings.TrimSpace(content) + "\n" + fence
}

// markdownRemediation renders the platform-relevant remediation of a check, with
// each variant introduced by its platform/tool id.
func markdownRemediation(query *policy.Mquery, platformKeys map[string]bool) string {
	var b strings.Builder
	for _, item := range reportdoc.RemediationItems(query, platformKeys) {
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		if item.Id != "" && item.Id != "default" {
			b.WriteString("_" + item.Id + "_\n\n")
		}
		b.WriteString(strings.TrimSpace(item.Desc))
	}
	return b.String()
}

func markdownReferences(query *policy.Mquery) string {
	var items []string
	for _, ref := range reportdoc.QueryRefs(query) {
		title := ref.Title
		if title == "" {
			title = ref.Url
		}
		items = append(items, "["+title+"]("+ref.Url+")")
	}
	return markdownList(items)
}

func strPtr(s string) *string {
	return &s
}

func writeSarif(report *sarif.Report, out iox.OutputHelper) error {
	return report.Write(out)
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// riskSarifLevel maps a cnspec risk value to a SARIF level. The bands are the same
// ones GitHub code scanning uses for security-severity (>= 7.0 error, >= 4.0
// warning), which keeps level and security-severity consistent.
func riskSarifLevel(risk int32) string {
	switch {
	case risk >= 70:
		return "error"
	case risk >= 40:
		return "warning"
	default:
		return "note"
	}
}

// securitySeverity renders a cnspec risk value (0-100) as the 0.0-10.0 string that
// GitHub code scanning reads from the "security-severity" rule property.
func securitySeverity(risk int32) string {
	return strconv.FormatFloat(float64(risk)/10, 'f', 1, 64)
}
