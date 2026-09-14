// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"sort"
	"strings"

	"github.com/owenrumney/go-sarif/v2/sarif"
	"go.mondoo.com/cnspec"
	"go.mondoo.com/cnspec/reports/reportdoc"
	"go.mondoo.com/mql/providers-sdk/v1/upstream/fex"
	"go.mondoo.com/mql/utils/iox"
)

// VulnReportToSarif renders the VEX rows for a target as a SARIF 2.1.0 report:
// one run for the target, one rule per advisory, one result per affected
// package.
//
// It is the SARIF counterpart of VulnReportToJSON and VulnReportToCSV and reads
// the same fex.VulnRow view, so severity, fixed version and package coordinates
// cannot drift between the formats. Severity reaches SARIF as a risk value via
// reportdoc.RiskFromSeverityLabel and then through the same riskSarifLevel and
// securitySeverity helpers the scan path uses, so a VEX-sourced finding and a
// scan-sourced one of the same severity render identically.
func VulnReportToSarif(target string, rows []fex.VulnRow, out iox.OutputHelper) error {
	report, err := sarif.New(sarif.Version210)
	if err != nil {
		return err
	}

	run := newVulnRun(target)
	for _, id := range sortedVulnRuleIDs(rows) {
		registerVexRule(run, id, rows)
	}
	for i := range rows {
		run.AddResult(vexResult(target, rows[i]))
	}
	report.AddRun(run)

	return writeSarif(report, out)
}

// newVulnRun builds the run for a `cnspec vuln` target. Unlike a scan run there
// is no asset MRN, no platform and no policy score behind it: the target name is
// the whole of what is known about the asset.
func newVulnRun(target string) *sarif.Run {
	run := sarif.NewRunWithInformationURI("cnspec", sarifInformationURI)
	run.Tool.Driver.
		WithVersion(cnspec.GetVersion()).
		WithFullName("cnspec " + cnspec.GetVersion()).
		WithShortDescription(sarif.NewMultiformatMessageString(
			"cnspec is an open source, cloud-native security and policy scanner"))
	run.Tool.Driver.Organization = strPtr("Mondoo")
	run.ColumnKind = "utf16CodeUnits"

	run.AutomationDetails = sarif.NewRunAutomationDetails().
		WithID("cnspec/" + target).
		WithDescriptionText("cnspec vulnerability scan of " + target)

	run.Properties = sarif.Properties{"asset": target}
	return run
}

// sortedVulnRuleIDs lists the advisory ids in the rows, deduplicated and sorted,
// so a run's rules are emitted in a stable order regardless of row order.
func sortedVulnRuleIDs(rows []fex.VulnRow) []string {
	seen := map[string]bool{}
	ids := make([]string, 0, len(rows))
	for i := range rows {
		id := vexRuleID(rows[i])
		if seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// vexRuleID is the rule a row reports under. A row with no advisory id still has
// to report under something, and the generic package rule is what the scan path
// uses for the same case.
func vexRuleID(row fex.VulnRow) string {
	if row.ID == "" {
		return sarifVulnPackageRuleID
	}
	return row.ID
}

// registerVexRule registers one advisory as a SARIF rule. The rule carries what
// is true of the advisory itself; anything per-package belongs on the result.
// Rows are passed in so the rule can take its description and references from
// the first row reporting it -- they are properties of the advisory, so every
// such row carries the same ones.
func registerVexRule(run *sarif.Run, ruleID string, rows []fex.VulnRow) {
	var row fex.VulnRow
	for i := range rows {
		if vexRuleID(rows[i]) == ruleID {
			row = rows[i]
			break
		}
	}

	if ruleID == sarifVulnPackageRuleID && row.ID == "" {
		run.AddRule(sarifVulnPackageRuleID).
			WithName("Vulnerable package").
			WithDescription("An installed package has known vulnerabilities").
			WithFullDescription(sarif.NewMultiformatMessageString(
				"An installed package is affected by known vulnerabilities. Update it to a fixed version.")).
			WithProperties(sarif.Properties{"tags": []string{"security", "cnspec", "vulnerability"}})
		return
	}

	risk := reportdoc.RiskFromSeverityLabel(row.Severity)
	summary := strings.TrimSpace(row.Summary)

	rb := run.AddRule(ruleID).WithName(ruleID).WithDescription(firstLine(summary, ruleID))
	if summary != "" {
		rb.WithFullDescription(sarif.NewMultiformatMessageString(summary).WithMarkdown(summary))
	}

	var text, md strings.Builder
	if summary != "" {
		text.WriteString(summary + "\n")
		md.WriteString(summary + "\n")
	}

	severity := severityOrNone(row.Severity) + " (score " + securitySeverity(risk) + ")"
	reportdoc.WriteDetailSection(&text, "Severity", severity)
	writeMarkdownSection(&md, "Severity", severity)

	if hint := strings.TrimSpace(row.RemediationHint); hint != "" {
		reportdoc.WriteDetailSection(&text, "Remediation", hint)
		writeMarkdownSection(&md, "Remediation", hint)
	}

	if len(row.References) > 0 {
		mdLines := make([]string, 0, len(row.References))
		for _, ref := range row.References {
			mdLines = append(mdLines, "["+ref+"]("+ref+")")
		}
		reportdoc.WriteDetailSection(&text, "References", strings.Join(row.References, "\n"))
		writeMarkdownSection(&md, "References", markdownList(mdLines))
	}

	if help := strings.TrimSpace(text.String()); help != "" {
		rb.WithHelp(sarif.NewMultiformatMessageString(help).WithMarkdown(strings.TrimSpace(md.String())))
	}
	if len(row.References) > 0 {
		rb.WithHelpURI(row.References[0])
	}

	props := sarif.Properties{
		"tags":              []string{"security", "cnspec", "vulnerability", "advisory"},
		"security-severity": securitySeverity(risk),
		"severity":          severityOrNone(row.Severity),
		"advisory":          ruleID,
	}
	rb.WithProperties(props)
	rb.WithDefaultConfiguration(sarif.NewReportingConfiguration().WithLevel(riskSarifLevel(risk)))
}

// vexResult builds the finding for one affected package under one advisory.
func vexResult(target string, row fex.VulnRow) *sarif.Result {
	ruleID := vexRuleID(row)
	risk := reportdoc.RiskFromSeverityLabel(row.Severity)

	update := "No fixed version is available yet."
	if row.FixedVersion != "" {
		update = "Update to " + row.FixedVersion + "."
	}

	name := orUnknownPackage(row.AffectedName)
	text := name + " " + row.AffectedVersion + " has known vulnerabilities"
	markdown := "**" + name + "** " + row.AffectedVersion + " has known vulnerabilities"
	if row.ID != "" {
		text = name + " " + row.AffectedVersion + " is affected by " + row.ID
		markdown = "**" + name + "** " + row.AffectedVersion + " is affected by **" + row.ID + "**"
		if summary := firstLine(strings.TrimSpace(row.Summary), ""); summary != "" {
			text += " (" + summary + ")"
			markdown += " — " + summary
		}
	}
	text += " · " + severityOrNone(row.Severity) + " (score " + securitySeverity(risk) + ") · " + update
	markdown += "\n\n" + scoreIcon(risk) + " **" + severityOrNone(row.Severity) + "**" +
		" · score " + securitySeverity(risk) + " · " + update

	logicalLocs := []*sarif.LogicalLocation{
		sarif.NewLogicalLocation().WithName(target).WithKind("asset"),
		sarif.NewLogicalLocation().WithName(name).WithKind("package").
			WithFullyQualifiedName(name + "@" + row.AffectedVersion),
	}

	props := sarif.Properties{
		"asset":             target,
		"package":           name,
		"installedVersion":  row.AffectedVersion,
		"severity":          severityOrNone(row.Severity),
		"security-severity": securitySeverity(risk),
		"status":            "fail",
	}
	if row.FixedVersion != "" {
		props["fixedVersion"] = row.FixedVersion
	}
	if row.AffectedPurl != "" {
		props["purl"] = row.AffectedPurl
		if eco := fex.EcosystemOf(row.AffectedPurl); eco != "" {
			props["ecosystem"] = eco
		}
	}
	if row.ID != "" {
		props["advisory"] = row.ID
	}
	if len(row.References) > 0 {
		props["references"] = row.References
	}
	if hint := strings.TrimSpace(row.RemediationHint); hint != "" {
		props["remediation"] = hint
	}

	result := sarif.NewRuleResult(ruleID).
		WithLevel(riskSarifLevel(risk)).
		WithKind("fail").
		WithMessage(sarif.NewTextMessage(text).WithMarkdown(markdown)).
		WithLocations([]*sarif.Location{
			sarif.NewLocation().WithLogicalLocations(logicalLocs),
		}).
		WithPartialFingerPrints(map[string]interface{}{
			// The purl identifies the package precisely; name@version is what
			// there is when a row carries none.
			sarifFingerprintKey: sarifFingerprint(ruleID, target, vexPackageKey(row)),
		}).
		WithRank(float32(risk))
	result.Properties = props
	return result
}

// vexPackageKey identifies the affected package within a target, for fingerprints.
func vexPackageKey(row fex.VulnRow) string {
	if row.AffectedPurl != "" {
		return row.AffectedPurl
	}
	return row.AffectedName + "@" + row.AffectedVersion
}

// orUnknownPackage names a row whose component never resolved. VulnRows emits
// such a row rather than dropping the vulnerability, so it has to render.
func orUnknownPackage(name string) string {
	if strings.TrimSpace(name) == "" {
		return "unknown package"
	}
	return name
}

// firstLine reduces a summary to its first line for the one-line fields SARIF
// expects there, falling back when the summary is empty.
func firstLine(s, fallback string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	if s = strings.TrimSpace(s); s != "" {
		return s
	}
	return fallback
}
