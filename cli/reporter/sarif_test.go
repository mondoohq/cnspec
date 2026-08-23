// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/owenrumney/go-sarif/v2/sarif"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/internal/reportfixture"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/cnspec/reports/reportdoc"
	"go.mondoo.com/mql/cli/printer"
	"go.mondoo.com/mql/llx"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/providers-sdk/v1/upstream/mvd"
	"go.mondoo.com/mql/utils/iox"
	"go.mondoo.com/mql/utils/stringx"
)

// toSarif runs the converter and parses the result back into the SARIF types.
func toSarif(t *testing.T, r *policy.ReportCollection) *sarif.Report {
	t.Helper()
	buf := bytes.Buffer{}
	writer := iox.IOWriter{Writer: &buf}
	require.NoError(t, ConvertToSarif(r, &writer))

	report, err := sarif.FromBytes(buf.Bytes())
	require.NoError(t, err)
	return report
}

func findRule(run *sarif.Run, id string) *sarif.ReportingDescriptor {
	for _, rule := range run.Tool.Driver.Rules {
		if rule.ID == id {
			return rule
		}
	}
	return nil
}

func resultsForRule(run *sarif.Run, id string) []*sarif.Result {
	var res []*sarif.Result
	for _, result := range run.Results {
		if result.RuleID != nil && *result.RuleID == id {
			res = append(res, result)
		}
	}
	return res
}

// indented renders a help section body the way reportdoc.WriteDetailSection does, so tests
// can look for it verbatim inside the rendered help.
func indented(body string) string {
	return strings.TrimRight(stringx.Indent(2, body), "\n")
}

func ruleHelpText(rule *sarif.ReportingDescriptor) string {
	if rule == nil || rule.Help == nil || rule.Help.Text == nil {
		return ""
	}
	return *rule.Help.Text
}

func TestSarifConverter(t *testing.T) {
	yr := reportfixture.Sample()
	buf := bytes.Buffer{}
	writer := iox.IOWriter{Writer: &buf}
	err := ConvertToSarif(yr, &writer)
	require.NoError(t, err)

	sarifReport := buf.String()

	// Verify it's valid JSON
	var parsed map[string]any
	err = json.Unmarshal([]byte(sarifReport), &parsed)
	require.NoError(t, err)

	// Verify SARIF version
	assert.Equal(t, "2.1.0", parsed["version"])

	// Verify schema
	assert.Contains(t, sarifReport, "https://raw.githubusercontent.com/oasis-tcs/sarif-spec")

	// Verify tool info
	assert.Contains(t, sarifReport, "\"name\":\"cnspec\"")
	assert.Contains(t, sarifReport, "https://cnspec.io")

	// Verify rules are present
	assert.Contains(t, sarifReport, "Ensure SNMP server is stopped and not enabled")
	assert.Contains(t, sarifReport, "Configure kubelet to capture all event creation")
	assert.Contains(t, sarifReport, "Set secure file permissions on the scheduler.conf file")

	// Verify results contain asset name
	assert.Contains(t, sarifReport, "X1")

	// The sample collection carries no source context, so results fall back to
	// asset logical locations and must not invent physical locations.
	assert.NotContains(t, sarifReport, "physicalLocation")
	assert.Contains(t, sarifReport, "logicalLocations")

	// Verify results contain expected levels
	// Score type 2 (Result) with value 100 -> "none" (pass)
	// Score type 4 (Error) -> "error"
	// Score type 8 (Skip) -> "none"
	// Each asset gets its own run
	runs := parsed["runs"].([]any)
	require.Len(t, runs, 1)
	run := runs[0].(map[string]any)

	// Verify run-level asset properties
	props := run["properties"].(map[string]any)
	assert.Equal(t, "X1", props["asset"])

	results := run["results"].([]any)
	require.NotEmpty(t, results)

	// Verify each result has a level and message
	for _, r := range results {
		result := r.(map[string]any)
		assert.Contains(t, result, "level")
		assert.Contains(t, result, "message")
	}
}

func TestSarifDeterministicOutput(t *testing.T) {
	yr := reportfixture.Sample()

	// Run twice and verify identical output
	buf1 := bytes.Buffer{}
	writer1 := iox.IOWriter{Writer: &buf1}
	err := ConvertToSarif(yr, &writer1)
	require.NoError(t, err)

	buf2 := bytes.Buffer{}
	writer2 := iox.IOWriter{Writer: &buf2}
	err = ConvertToSarif(yr, &writer2)
	require.NoError(t, err)

	assert.Equal(t, buf1.String(), buf2.String())
}

// TestSarifDeterministicRuleIDs guards the deterministic query lookup: the ubuntu
// fixture has several checks that share a code id, and picking an arbitrary one
// would make rule ids (and with them the fingerprints consumers dedup on) flip
// between runs of the same report.
func TestSarifDeterministicRuleIDs(t *testing.T) {
	raw := reportfixture.UbuntuScanJSON()

	var first string
	for i := 0; i < 5; i++ {
		yr := &policy.ReportCollection{}
		require.NoError(t, json.Unmarshal(raw, yr))

		buf := bytes.Buffer{}
		writer := iox.IOWriter{Writer: &buf}
		require.NoError(t, ConvertToSarif(yr, &writer))

		if i == 0 {
			first = buf.String()
			continue
		}
		assert.Equal(t, first, buf.String())
	}
}

func TestSarifNilReport(t *testing.T) {
	var yr *policy.ReportCollection

	buf := bytes.Buffer{}
	writer := iox.IOWriter{Writer: &buf}
	err := ConvertToSarif(yr, &writer)
	require.NoError(t, err)

	var parsed map[string]any
	err = json.Unmarshal(buf.Bytes(), &parsed)
	require.NoError(t, err)
	assert.Equal(t, "2.1.0", parsed["version"])
}

func TestSarifWithAssetErrors(t *testing.T) {
	yr := &policy.ReportCollection{
		Assets: map[string]*inventory.Asset{
			"asset1": {Name: "test-server"},
		},
		Bundle: &policy.Bundle{},
		Errors: map[string]string{
			"asset1": "connection refused",
		},
	}

	buf := bytes.Buffer{}
	writer := iox.IOWriter{Writer: &buf}
	err := ConvertToSarif(yr, &writer)
	require.NoError(t, err)

	sarifReport := buf.String()
	assert.Contains(t, sarifReport, "asset-error")
	assert.Contains(t, sarifReport, "connection refused")
	assert.Contains(t, sarifReport, "test-server")
}

func TestSarifWithNilCollectorJob(t *testing.T) {
	yr := &policy.ReportCollection{
		Assets: map[string]*inventory.Asset{
			"asset1": {Name: "test-server"},
		},
		Bundle: &policy.Bundle{},
		ResolvedPolicies: map[string]*policy.ResolvedPolicy{
			"asset1": {CollectorJob: nil},
		},
		Reports: map[string]*policy.Report{
			"asset1": {Scores: map[string]*policy.Score{}},
		},
	}

	buf := bytes.Buffer{}
	writer := iox.IOWriter{Writer: &buf}
	err := ConvertToSarif(yr, &writer)
	require.NoError(t, err)

	var parsed map[string]any
	err = json.Unmarshal(buf.Bytes(), &parsed)
	require.NoError(t, err)
	assert.Equal(t, "2.1.0", parsed["version"])

	// Should have one run for the single asset
	runs := parsed["runs"].([]any)
	require.Len(t, runs, 1)
}

func TestSarifMultipleAssets(t *testing.T) {
	yr := &policy.ReportCollection{
		Assets: map[string]*inventory.Asset{
			"asset1": {Name: "server-a"},
			"asset2": {Name: "server-b"},
		},
		Bundle: &policy.Bundle{},
		Errors: map[string]string{
			"asset2": "timeout",
		},
	}

	buf := bytes.Buffer{}
	writer := iox.IOWriter{Writer: &buf}
	err := ConvertToSarif(yr, &writer)
	require.NoError(t, err)

	var parsed map[string]any
	err = json.Unmarshal(buf.Bytes(), &parsed)
	require.NoError(t, err)

	// Each asset should have its own run
	runs := parsed["runs"].([]any)
	require.Len(t, runs, 2)

	// Verify each run is tagged with its asset name
	run1 := runs[0].(map[string]any)
	run2 := runs[1].(map[string]any)
	props1 := run1["properties"].(map[string]any)
	props2 := run2["properties"].(map[string]any)

	assets := []string{props1["asset"].(string), props2["asset"].(string)}
	assert.Contains(t, assets, "server-a")
	assert.Contains(t, assets, "server-b")

	// Only the errored asset's run should have the error result
	sarifReport := buf.String()
	assert.Contains(t, sarifReport, "timeout")
}

func TestSarifQueryRuleID(t *testing.T) {
	tests := []struct {
		name     string
		query    *policy.Mquery
		expected string
	}{
		{
			"prefers uid",
			&policy.Mquery{Uid: "my-check", Mrn: "//local.cnspec.io/run/local-execution/queries/my-check"},
			"my-check",
		},
		{
			"strips local MRN prefix",
			&policy.Mquery{Mrn: "//local.cnspec.io/run/local-execution/queries/sshd-01"},
			"sshd-01",
		},
		{
			"strips policy API MRN prefix",
			&policy.Mquery{Mrn: "//policy.api.mondoo.app/queries/mondoo-linux-security-snmp"},
			"mondoo-linux-security-snmp",
		},
		{
			"falls back to code ID",
			&policy.Mquery{CodeId: "abc123"},
			"abc123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, reportdoc.QueryRuleID(tt.query))
		})
	}
}

func TestPhysicalLocationFromContext(t *testing.T) {
	t.Run("line and column range with content", func(t *testing.T) {
		ctx := llx.SourceContext{
			Path:    "main.tf",
			Range:   llx.NewRange().AddLineColumnRange(12, 18, 3, 7),
			Content: "resource \"aws_s3_bucket\" \"b\" {}",
		}
		pl := physicalLocationFromContext(ctx)
		require.NotNil(t, pl.ArtifactLocation)
		require.NotNil(t, pl.ArtifactLocation.URI)
		assert.Equal(t, "main.tf", *pl.ArtifactLocation.URI)

		require.NotNil(t, pl.Region)
		require.NotNil(t, pl.Region.StartLine)
		assert.Equal(t, 12, *pl.Region.StartLine)
		assert.Equal(t, 18, *pl.Region.EndLine)
		require.NotNil(t, pl.Region.StartColumn)
		assert.Equal(t, 3, *pl.Region.StartColumn)
		assert.Equal(t, 7, *pl.Region.EndColumn)

		require.NotNil(t, pl.Region.Snippet)
		require.NotNil(t, pl.Region.Snippet.Text)
		assert.Equal(t, ctx.Content, *pl.Region.Snippet.Text)
	})

	t.Run("line-only range omits columns", func(t *testing.T) {
		ctx := llx.SourceContext{Path: "x.tf", Range: llx.NewRange().AddLine(5)}
		pl := physicalLocationFromContext(ctx)
		require.NotNil(t, pl.Region)
		assert.Equal(t, 5, *pl.Region.StartLine)
		assert.Equal(t, 5, *pl.Region.EndLine)
		assert.Nil(t, pl.Region.StartColumn)
		assert.Nil(t, pl.Region.Snippet)
	})

	t.Run("empty range and no content yields no region", func(t *testing.T) {
		ctx := llx.SourceContext{Path: "x.tf"}
		pl := physicalLocationFromContext(ctx)
		require.NotNil(t, pl.ArtifactLocation)
		assert.Nil(t, pl.Region)
	})
}

func TestSarifLocationFingerprint(t *testing.T) {
	ctx := llx.SourceContext{Path: "main.tf", Range: llx.NewRange().AddLineRange(1, 4)}

	// Stable across calls.
	assert.Equal(t, sarifLocationFingerprint("rule-a", ctx), sarifLocationFingerprint("rule-a", ctx))

	// Differs by rule, path, and range.
	other := llx.SourceContext{Path: "other.tf", Range: ctx.Range}
	assert.NotEqual(t, sarifLocationFingerprint("rule-a", ctx), sarifLocationFingerprint("rule-b", ctx))
	assert.NotEqual(t, sarifLocationFingerprint("rule-a", ctx), sarifLocationFingerprint("rule-a", other))
}

func TestSarifRunMetadata(t *testing.T) {
	report := toSarif(t, reportfixture.Sample())
	require.Len(t, report.Runs, 1)
	run := report.Runs[0]

	require.NotNil(t, run.Tool.Driver.Version)
	assert.NotEmpty(t, *run.Tool.Driver.Version)
	require.NotNil(t, run.Tool.Driver.Organization)
	assert.Equal(t, "Mondoo", *run.Tool.Driver.Organization)
	assert.Equal(t, "utf16CodeUnits", run.ColumnKind)

	require.NotNil(t, run.AutomationDetails)
	require.NotNil(t, run.AutomationDetails.ID)
	assert.Equal(t, "cnspec/X1", *run.AutomationDetails.ID)

	// the run carries the asset identity and a summary of its results
	assert.Equal(t, "X1", run.Properties["asset"])
	assert.Equal(t, "ubuntu", run.Properties["platform"])
	assert.Equal(t, "22.04", run.Properties["platformVersion"])
	assert.EqualValues(t, 3, run.Properties["checksTotal"])
	assert.EqualValues(t, 1, run.Properties["checksPassed"])
	assert.EqualValues(t, 1, run.Properties["checksErrored"])
	assert.EqualValues(t, 1, run.Properties["checksSkipped"])

	// vulnerability stats mirror the JUnit vulnerability suite properties
	assert.EqualValues(t, 1, run.Properties["packagesTotal"])
	assert.EqualValues(t, 1, run.Properties["packagesCritical"])
	assert.EqualValues(t, 1, run.Properties["packagesAffected"])
}

func TestSarifResultKinds(t *testing.T) {
	report := toSarif(t, reportfixture.Sample())
	run := report.Runs[0]

	kinds := map[string]string{}
	levels := map[string]string{}
	for _, result := range run.Results {
		if result.Kind == nil || result.RuleID == nil {
			continue
		}
		kinds[*result.RuleID] = *result.Kind
		levels[*result.RuleID] = *result.Level
	}

	// passed check
	assert.Equal(t, "pass", kinds["mondoo-linux-security-snmp-server-is-not-enabled"])
	assert.Equal(t, "none", levels["mondoo-linux-security-snmp-server-is-not-enabled"])
	// errored check
	assert.Equal(t, "fail", kinds["mondoo-kubernetes-security-kubelet-event-record-qps"])
	assert.Equal(t, "error", levels["mondoo-kubernetes-security-kubelet-event-record-qps"])
	// skipped check
	assert.Equal(t, "notApplicable", kinds["mondoo-kubernetes-security-secure-scheduler_conf"])
	assert.Equal(t, "none", levels["mondoo-kubernetes-security-secure-scheduler_conf"])
}

func TestSarifDetailedRuleContent(t *testing.T) {
	report := toSarif(t, reportfixture.Detailed())
	require.Len(t, report.Runs, 1)
	run := report.Runs[0]

	rule := findRule(run, "test-check")
	require.NotNil(t, rule)

	require.NotNil(t, rule.Name)
	assert.Equal(t, "Ensure the thing is configured", *rule.Name)
	require.NotNil(t, rule.FullDescription)
	require.NotNil(t, rule.FullDescription.Text)
	assert.Equal(t, "Root login over SSH should be disabled.", *rule.FullDescription.Text)

	// the help carries query, remediation and references, as text and markdown
	require.NotNil(t, rule.Help)
	require.NotNil(t, rule.Help.Markdown)
	help := ruleHelpText(rule)
	assert.Contains(t, help, "Query:")
	assert.Contains(t, help, "sshd.config.params['PermitRootLogin']")
	assert.Contains(t, help, "Remediation:")
	assert.Contains(t, help, "References:")
	assert.Contains(t, help, "CIS Benchmark: https://example.com/cis")

	markdown := *rule.Help.Markdown
	assert.Contains(t, markdown, "**Query**")
	assert.Contains(t, markdown, "```")
	assert.Contains(t, markdown, "**Remediation**")
	assert.Contains(t, markdown, "[CIS Benchmark](https://example.com/cis)")

	// remediation is filtered to the asset platform, like the JUnit reporter does
	assert.Contains(t, help, "Set PermitRootLogin to no in your TF config.")
	assert.NotContains(t, help, "Use the AWS console to fix it.")
	assert.NotContains(t, markdown, "Use CloudFormation to fix it.")

	require.NotNil(t, rule.HelpURI)
	assert.Equal(t, "https://example.com/cis", *rule.HelpURI)

	// severity and tags drive rendering and filtering in SARIF consumers
	assert.Contains(t, rule.Properties["tags"], "security")
	assert.Contains(t, rule.Properties["remediation"], "[terraform]")
	assert.Equal(t, "sshd.config.params['PermitRootLogin'] == \"no\"", rule.Properties["mql"])

	// the finding itself reports the outcome
	results := resultsForRule(run, "test-check")
	require.Len(t, results, 1)
	require.NotNil(t, results[0].Message.Text)
	assert.Contains(t, *results[0].Message.Text, "Ensure the thing is configured: FAIL")
	require.NotNil(t, results[0].Message.Markdown)
	assert.Contains(t, *results[0].Message.Markdown, "**FAIL**")
	assert.Equal(t, "fail", results[0].Properties["status"])
	assert.EqualValues(t, 100, results[0].Properties["risk"])
	assert.Equal(t, "CRITICAL", results[0].Properties["severity"])
	require.NotNil(t, results[0].Rank)
	assert.EqualValues(t, 100, *results[0].Rank)
}

func TestSarifRuleSeverity(t *testing.T) {
	yr := reportfixture.Detailed()
	yr.Bundle.Queries[0].Impact = &policy.Impact{Value: &policy.ImpactValue{Value: 80}}

	run := toSarif(t, yr).Runs[0]
	rule := findRule(run, "test-check")
	require.NotNil(t, rule)

	assert.EqualValues(t, 80, rule.Properties["impact"])
	assert.Equal(t, "HIGH", rule.Properties["severity"])
	// GitHub code scanning reads the alert severity from this property
	assert.Equal(t, "8.0", rule.Properties["security-severity"])
	require.NotNil(t, rule.DefaultConfiguration)
	assert.Equal(t, "error", rule.DefaultConfiguration.Level)
	assert.Contains(t, ruleHelpText(rule), "HIGH (impact 80)")
}

func TestSarifComplianceTags(t *testing.T) {
	yr := reportfixture.Detailed()
	yr.Bundle.Queries[0].Tags = map[string]string{
		"compliance/iso-27001-2022": "iso-27001-2022-a-8-24",
		"compliance/bsi-sys-1-5":    "false", // explicitly turned off, must be dropped
		"mondoo.com/platform":       "aws,cloud",
	}

	run := toSarif(t, yr).Runs[0]
	rule := findRule(run, "test-check")
	require.NotNil(t, rule)

	compliance, ok := rule.Properties["compliance"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "iso-27001-2022-a-8-24", compliance["compliance/iso-27001-2022"])
	assert.NotContains(t, compliance, "compliance/bsi-sys-1-5")
	assert.NotContains(t, compliance, "mondoo.com/platform")

	assert.Contains(t, rule.Properties["tags"], "compliance/iso-27001-2022")
	assert.Contains(t, ruleHelpText(rule), "iso-27001-2022: iso-27001-2022-a-8-24")
}

func TestSarifVulnerabilities(t *testing.T) {
	yr := reportfixture.Detailed()
	assetMrn := "//assets.api.mondoo.app/spaces/test/assets/abc"
	yr.VulnReports = map[string]*mvd.VulnReport{
		assetMrn: {
			Packages: []*mvd.Package{
				{Name: "libssl3", Version: "3.0.2-0ubuntu1.10", Available: "3.0.2-0ubuntu1.15", Affected: true, Score: 95, Arch: "amd64", Format: "deb"},
				{Name: "curl", Version: "7.81.0-1ubuntu1.4", Available: "7.81.0-1ubuntu1.15", Affected: true, Score: 50},
				{Name: "bash", Version: "5.1-6ubuntu1", Affected: false},
			},
			Advisories: []*mvd.Advisory{
				{
					ID:          "USN-1234-1",
					Title:       "OpenSSL vulnerabilities",
					Description: "Several security issues were fixed in OpenSSL.",
					Score:       95,
					Affected:    []*mvd.Package{{Name: "libssl3", Version: "3.0.2-0ubuntu1.10"}},
					Cves:        []*mvd.CVE{{ID: "CVE-2023-0286", Score: 7.4, Url: "https://nvd.nist.gov/vuln/detail/CVE-2023-0286"}},
				},
			},
		},
	}

	run := toSarif(t, yr).Runs[0]

	// the advisory becomes its own rule, with CVEs and severity
	advisoryRule := findRule(run, "USN-1234-1")
	require.NotNil(t, advisoryRule)
	assert.Equal(t, "9.5", advisoryRule.Properties["security-severity"])
	assert.Equal(t, "CRITICAL", advisoryRule.Properties["severity"])
	assert.Contains(t, advisoryRule.Properties["tags"], "vulnerability")
	assert.Contains(t, ruleHelpText(advisoryRule), "CVE-2023-0286")
	require.NotNil(t, advisoryRule.HelpURI)
	assert.Equal(t, "https://nvd.nist.gov/vuln/detail/CVE-2023-0286", *advisoryRule.HelpURI)

	advisoryResults := resultsForRule(run, "USN-1234-1")
	require.Len(t, advisoryResults, 1)
	assert.Contains(t, *advisoryResults[0].Message.Text, "libssl3 3.0.2-0ubuntu1.10 is affected by USN-1234-1")
	assert.Contains(t, *advisoryResults[0].Message.Text, "Update to 3.0.2-0ubuntu1.15.")
	assert.Equal(t, "libssl3", advisoryResults[0].Properties["package"])
	assert.Equal(t, "3.0.2-0ubuntu1.15", advisoryResults[0].Properties["fixedVersion"])
	// the package shows up as a logical location next to the asset
	require.Len(t, advisoryResults[0].Locations, 1)
	require.Len(t, advisoryResults[0].Locations[0].LogicalLocations, 2)
	assert.Equal(t, "package", *advisoryResults[0].Locations[0].LogicalLocations[1].Kind)

	// affected packages no advisory accounts for fall back to a generic rule
	pkgResults := resultsForRule(run, sarifVulnPackageRuleID)
	require.Len(t, pkgResults, 1)
	assert.Contains(t, *pkgResults[0].Message.Text, "curl 7.81.0-1ubuntu1.4 has known vulnerabilities")
	assert.Equal(t, "MEDIUM", pkgResults[0].Properties["severity"])

	// packages that are not affected produce no findings
	assert.NotContains(t, resultMessages(run), "bash")
}

func resultMessages(run *sarif.Run) string {
	var b strings.Builder
	for _, result := range run.Results {
		if result.Message.Text != nil {
			b.WriteString(*result.Message.Text)
			b.WriteString("\n")
		}
	}
	return b.String()
}

// TestSarifMatchesDetailedJunitContent verifies the promise of the SARIF reporter:
// everything the detailed JUnit failure body renders for a check is present in the
// SARIF report too - the static documentation on the rule, the dynamic outcome on
// the result.
func TestSarifMatchesDetailedJunitContent(t *testing.T) {
	raw := reportfixture.UbuntuScanJSON()
	yr := &policy.ReportCollection{}
	require.NoError(t, json.Unmarshal(raw, yr))

	report := toSarif(t, yr)
	runs := map[string]*sarif.Run{}
	for _, run := range report.Runs {
		runs[run.Properties["asset"].(string)] = run
	}

	bundle := yr.Bundle.ToMap()
	queries := reportdoc.QueryMap(bundle)

	checked := 0
	for assetMrn, asset := range yr.Assets {
		run := runs[asset.Name]
		require.NotNil(t, run)

		policyReport := yr.Reports[assetMrn]
		resolved := yr.ResolvedPolicies[assetMrn]
		require.NotNil(t, policyReport)
		require.NotNil(t, resolved)
		platformKeys := reportdoc.PlatformRemediationKeys(asset.Platform)

		for id, score := range policyReport.Scores {
			if _, ok := resolved.CollectorJob.ReportingQueries[id]; !ok {
				continue
			}
			query, ok := queries[id]
			if !ok {
				continue
			}
			// the JUnit body is only rendered for failed and errored checks
			failed := score != nil && (score.Type == policy.ScoreType_Error ||
				(score.Type == policy.ScoreType_Result && score.Value != 100))
			if !failed {
				continue
			}
			if detailedCheckBody(resolved, policyReport, query, score, platformKeys) == "" {
				continue
			}
			checked++

			ruleID := reportdoc.QueryRuleID(query)
			rule := findRule(run, ruleID)
			require.NotNil(t, rule, "no SARIF rule for %s", ruleID)
			results := resultsForRule(run, ruleID)
			require.NotEmpty(t, results, "no SARIF result for %s", ruleID)

			var messages strings.Builder
			for _, result := range results {
				if result.Message.Text != nil {
					messages.WriteString(*result.Message.Text)
				}
			}

			help := ruleHelpText(rule)
			if desc := strings.TrimSpace(reportdoc.QueryDescription(query)); desc != "" {
				require.NotNil(t, rule.FullDescription)
				assert.Equal(t, desc, *rule.FullDescription.Text, "description missing for %s", ruleID)
				assert.Contains(t, help, desc, "description missing from help of %s", ruleID)
			}
			// help sections are indented by two spaces, exactly like the JUnit body
			if mql := strings.TrimSpace(reportdoc.QueryMql(query)); mql != "" {
				assert.Contains(t, help, indented(mql), "query missing from help of %s", ruleID)
				assert.Equal(t, mql, rule.Properties["mql"])
			}
			if rem := reportdoc.QueryRemediation(query, platformKeys); rem != "" {
				assert.Contains(t, help, indented(rem), "remediation missing from help of %s", ruleID)
				assert.Equal(t, rem, rule.Properties["remediation"])
			}
			if refs := reportdoc.QueryReferences(query); refs != "" {
				assert.Contains(t, help, indented(refs), "references missing from help of %s", ruleID)
			}
			if msg := score.MessageLine(); msg != "" {
				assert.Contains(t, messages.String(), msg, "score message missing from results of %s", ruleID)
			}
			// checks that fail on several source locations get one result per
			// location and carry the offending snippet there instead of repeating
			// the whole assessment on each of them
			if assessment := checkAssessmentText(resolved, policyReport, query); assessment != "" && len(results) == 1 {
				assert.Contains(t, messages.String(), assessment, "assessment missing from results of %s", ruleID)
			}
		}
	}

	assert.Greater(t, checked, 0, "fixture has no failing checks to compare")
}

// checkAssessmentText renders the expected-vs-actual assessment the way both
// reporters do, for use in the content parity test.
func checkAssessmentText(resolved *policy.ResolvedPolicy, report *policy.Report, query *policy.Mquery) string {
	if resolved == nil || resolved.ExecutionJob == nil {
		return ""
	}
	cb := resolved.GetCodeBundle(query)
	if cb == nil {
		return ""
	}
	assessment := policy.Query2Assessment(cb, report)
	if assessment == nil {
		return ""
	}
	return strings.TrimSpace(printer.PlainNoColorPrinter.Assessment(cb, assessment))
}

func TestSarifScoreLevels(t *testing.T) {
	tests := []struct {
		name     string
		score    *policy.Score
		expected string
	}{
		{"nil score", nil, "none"},
		{"pass", &policy.Score{Type: policy.ScoreType_Result, Value: 100}, "none"},
		{"low severity 99", &policy.Score{Type: policy.ScoreType_Result, Value: 99}, "note"},
		{"low severity boundary 61", &policy.Score{Type: policy.ScoreType_Result, Value: 61}, "note"},
		{"medium severity boundary 60", &policy.Score{Type: policy.ScoreType_Result, Value: 60}, "warning"},
		{"medium severity 50", &policy.Score{Type: policy.ScoreType_Result, Value: 50}, "warning"},
		{"medium severity boundary 31", &policy.Score{Type: policy.ScoreType_Result, Value: 31}, "warning"},
		{"high severity boundary 30", &policy.Score{Type: policy.ScoreType_Result, Value: 30}, "error"},
		{"high severity 15", &policy.Score{Type: policy.ScoreType_Result, Value: 15}, "error"},
		{"critical severity 5", &policy.Score{Type: policy.ScoreType_Result, Value: 5}, "error"},
		{"critical severity zero", &policy.Score{Type: policy.ScoreType_Result, Value: 0}, "error"},
		{"error type", &policy.Score{Type: policy.ScoreType_Error}, "error"},
		{"skip", &policy.Score{Type: policy.ScoreType_Skip}, "none"},
		{"unknown", &policy.Score{Type: policy.ScoreType_Unknown}, "none"},
		{"unscored", &policy.Score{Type: policy.ScoreType_Unscored}, "none"},
		{"out of scope", &policy.Score{Type: policy.ScoreType_OutOfScope}, "none"},
		{"disabled", &policy.Score{Type: policy.ScoreType_Disabled}, "none"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, scoreToSarifLevel(tt.score))
		})
	}
}
