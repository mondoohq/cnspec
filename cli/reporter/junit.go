// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"

	"github.com/jstemmer/go-junit-report/v2/junit"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/cnspec/reports/reportdoc"
	"go.mondoo.com/mql/cli/printer"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/utils/iox"
)

// ConvertToJunit maps the ReportCollection to Junit. Each asset becomes its own Suite.
// When detailed is true, failed and errored check testcases carry a rich body
// (description, query, assessment, remediation, references) in their <failure>/<error>
// element; passing and skipped checks stay lean regardless.
func ConvertToJunit(r *policy.ReportCollection, out iox.OutputHelper, detailed bool) error {
	noXMLHeader := false

	suites := junit.Testsuites{}

	// render asset errors
	// r is nil if no assets were scanned
	if r != nil {
		for assetMrn, errMsg := range r.Errors {
			a := r.Assets[assetMrn]

			properties := []junit.Property{}
			ts := junit.Testsuite{
				Tests:      1,
				Failures:   1,
				Time:       "",
				Name:       "Report for " + a.Name,
				Properties: &properties,
				Testcases: []junit.Testcase{
					{
						Classname: "scan",
						Name:      "Scan " + a.Name,
						Failure: &junit.Result{
							Type:    "error",
							Message: errMsg,
						},
					},
				},
			}
			suites.Suites = append(suites.Suites, ts)
		}

		if r.Bundle == nil {
			return fmt.Errorf("no policy bundle found")
		}

		bundle := r.Bundle.ToMap()
		queries := reportdoc.QueryMap(bundle)

		// iterate over asset mrns
		for assetMrn, assetObj := range r.Assets {
			// add check results
			ts := assetPolicyTests(r, assetMrn, assetObj, queries, detailed)
			suites.Suites = append(suites.Suites, ts)

			vulnerabilityTests := assetMvdTests(r, assetMrn, assetObj)
			if vulnerabilityTests != nil {
				suites.Suites = append(suites.Suites, *vulnerabilityTests)
			}
		}
	}

	// to xml
	data, err := xml.MarshalIndent(suites, "", "\t")
	if err != nil {
		return err
	}

	if !noXMLHeader {
		_ = out.WriteString(xml.Header)
	}

	_, _ = out.Write(data)
	_ = out.WriteString("\n")

	return nil
}

// assetPolicyTests converts asset scoring queries to Junit test cases
func assetPolicyTests(r *policy.ReportCollection, assetMrn string, assetObj *inventory.Asset, queries map[string]*policy.Mquery, detailed bool) junit.Testsuite {
	ts := junit.Testsuite{
		Time:      "",
		Testcases: []junit.Testcase{},
	}
	ts.Name = "Policy Report for " + assetObj.Name

	report, ok := r.Reports[assetMrn]
	if !ok {
		// nothing to do, we get an error message in the summary code
		return ts
	}

	resolved, ok := r.ResolvedPolicies[assetMrn]
	if !ok {
		// nothing to do, we get an additional error message in the summary code
		return ts
	}

	// Data queries have no score and are not represented here. When detailed is set,
	// failed/errored checks carry their meta information (description, query, assessment,
	// remediation, references) in the failure body below.
	platformKeys := reportdoc.PlatformRemediationKeys(assetObj.Platform)
	for id, score := range report.Scores {
		_, ok := resolved.CollectorJob.ReportingQueries[id]
		if !ok {
			continue
		}

		query, ok := queries[id]
		if !ok {
			continue
		}

		ts.Tests++
		testCase := junit.Testcase{
			Classname: "score",
			Name:      query.Title,
			Time:      "",
			Failure:   nil,
		}

		if score != nil {
			if score.Type == policy.ScoreType_Skip {
				testCase.Skipped = &junit.Result{
					Message: "skipped",
				}
			}

			if score.Type == policy.ScoreType_Unknown {
				testCase.Skipped = &junit.Result{
					Message: "unknown",
				}
			}

			if score.Type == policy.ScoreType_Error {
				testCase.Failure = &junit.Result{
					Type: "error",
				}
				if detailed {
					testCase.Failure.Data = detailedCheckBody(resolved, report, query, score, platformKeys)
				}
				ts.Failures++
			}

			if score.Type == policy.ScoreType_Result && score.Value != 100 {
				testCase.Failure = &junit.Result{
					Message: "results do not match",
					Type:    "fail",
				}
				if detailed {
					if line := score.MessageLine(); line != "" {
						testCase.Failure.Message = line
					}
					testCase.Failure.Data = detailedCheckBody(resolved, report, query, score, platformKeys)
				}
				ts.Failures++
			}
		}
		ts.Testcases = append(ts.Testcases, testCase)
	}

	return ts
}

// assetMvdTests converts asset vulnerability results to Junit test cases
func assetMvdTests(r *policy.ReportCollection, assetMrn string, assetObj *inventory.Asset) *junit.Testsuite {
	// check if we have a vulnerability report
	vulnReport, ok := r.VulnReports[assetMrn]
	if !ok {
		return nil
	}

	ts := &junit.Testsuite{
		Name:      "Vulnerability Report for " + assetObj.Name,
		Tests:     0,
		Failures:  0,
		Time:      "",
		Testcases: []junit.Testcase{},
	}

	// packages advisories
	if vulnReport.Stats != nil && vulnReport.Stats.Packages != nil && vulnReport.Stats.Packages.Affected > 0 {
		ts.Tests = len(vulnReport.Packages)

		properties := []junit.Property{}
		if vulnReport.Platform != nil {
			properties = append(properties, junit.Property{"platform.name", vulnReport.Platform.Name})
			properties = append(properties, junit.Property{"platform.release", vulnReport.Platform.Release})
			properties = append(properties, junit.Property{"platform.arch", vulnReport.Platform.Arch})
		}

		if vulnReport.Stats != nil && vulnReport.Stats.Packages != nil {
			properties = append(properties, junit.Property{"report.packages.total", iota32(vulnReport.Stats.Packages.Total)})
			properties = append(properties, junit.Property{"report.packages.critical", iota32(vulnReport.Stats.Packages.Critical)})
			properties = append(properties, junit.Property{"report.packages.high", iota32(vulnReport.Stats.Packages.High)})
			properties = append(properties, junit.Property{"report.packages.medium", iota32(vulnReport.Stats.Packages.Medium)})
			properties = append(properties, junit.Property{"report.packages.low", iota32(vulnReport.Stats.Packages.Low)})
			properties = append(properties, junit.Property{"report.packages.none", iota32(vulnReport.Stats.Packages.None)})
		}

		ts.Properties = &properties

		for i := range vulnReport.Packages {
			pkg := vulnReport.Packages[i]
			if pkg == nil || !pkg.Affected {
				continue
			}

			testCase := junit.Testcase{
				Classname: "vulnerability",
				Name:      pkg.Name,
				Time:      "",
				Failure:   nil,
			}

			if pkg.Affected {
				ts.Failures++

				var content string
				content += pkg.Name + " with version " + pkg.Version + " has known vulnerabilities"
				if pkg.Score > 0 {
					content += " (score " + fmt.Sprintf("%v", float32(pkg.Score)/10) + ")"
				}

				var updateTo string
				if len(pkg.Available) > 0 {
					updateTo = " to " + pkg.Available
				}

				testCase.Failure = &junit.Result{
					Message: "Update " + pkg.Name + updateTo,
					Type:    pkg.Format,
					Data:    content,
				}
			}
			ts.Testcases = append(ts.Testcases, testCase)
		}
	}

	return ts
}

// detailedCheckBody renders rich, human-readable detail for a failed or errored
// check into the CDATA body of a JUnit <failure>/<error> element. Most JUnit
// consumers (including GitLab) surface this body only for failing tests, so it is
// only built for failures. It uses the no-color printer so no ANSI escape
// sequences leak into the XML.
func detailedCheckBody(resolved *policy.ResolvedPolicy, report *policy.Report, query *policy.Mquery, score *policy.Score, platformKeys map[string]bool) string {
	var b strings.Builder

	if desc := strings.TrimSpace(reportdoc.QueryDescription(query)); desc != "" {
		b.WriteString(desc)
		b.WriteString("\n")
	}

	if mql := reportdoc.QueryMql(query); mql != "" {
		reportdoc.WriteDetailSection(&b, "Query", mql)
	}

	// The assessment (expected vs actual) is only available for assertion checks
	// that executed. Guard exactly like the SARIF reporter: GetCodeBundle panics
	// when ExecutionJob is nil, and returns nil when the query has no code bundle.
	// report is guaranteed non-nil by the caller (assetPolicyTests ranges over
	// report.Scores) but is guarded here too, since Query2Assessment dereferences
	// it via report.Scores / report.Data.
	if report != nil && resolved != nil && resolved.ExecutionJob != nil {
		if cb := resolved.GetCodeBundle(query); cb != nil {
			if assessment := policy.Query2Assessment(cb, report); assessment != nil {
				if text := strings.TrimSpace(printer.PlainNoColorPrinter.Assessment(cb, assessment)); text != "" {
					reportdoc.WriteDetailSection(&b, "Result", text)
				}
				if locs := reportdoc.FailingResourceLocations(cb, assessment); locs != "" {
					reportdoc.WriteDetailSection(&b, "Failing resources", locs)
				}
			}
		}
	}

	// For errored checks the score message carries the failure reason.
	if score != nil && score.Type == policy.ScoreType_Error {
		if msg := score.MessageLine(); msg != "" {
			reportdoc.WriteDetailSection(&b, "Error", msg)
		}
	}

	reportdoc.WriteDetailSection(&b, "Remediation", reportdoc.QueryRemediation(query, platformKeys))
	reportdoc.WriteDetailSection(&b, "References", reportdoc.QueryReferences(query))

	return strings.TrimSpace(b.String())
}

func iota32(i int32) string {
	return strconv.FormatInt(int64(i), 10)
}
