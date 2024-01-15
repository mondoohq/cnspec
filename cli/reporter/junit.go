// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"encoding/xml"
	"fmt"
	"strconv"

	"github.com/jstemmer/go-junit-report/v2/junit"
	"github.com/mitchellh/mapstructure"
	"go.mondoo.com/cnquery/v10/explorer"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/upstream/mvd"
	"go.mondoo.com/cnquery/v10/shared"
	"go.mondoo.com/cnspec/v10/policy"
)

// ReportCollectionToJunit maps the ReportCollection to Junit. Each asset becomes its own Suite
func ReportCollectionToJunit(r *policy.ReportCollection, out shared.OutputHelper) error {
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

		bundle := r.Bundle.ToMap()
		queries := bundle.QueryMap()

		// iterate over asset mrns
		for assetMrn, assetObj := range r.Assets {
			// add check results
			ts := assetPolicyTests(r, assetMrn, assetObj, queries)
			suites.Suites = append(suites.Suites, ts)

			vulernabilityTests := assetMvdTests(r, assetMrn, assetObj)
			if vulernabilityTests != nil {
				suites.Suites = append(suites.Suites, *vulernabilityTests)
			}
		}
	}

	// to xml
	data, err := xml.MarshalIndent(suites, "", "\t")
	if err != nil {
		return err
	}

	if !noXMLHeader {
		out.WriteString(xml.Header)
	}

	out.Write(data)
	out.WriteString("\n")

	return nil
}

// assetPolicyTests converts asset scoring queries to Junit test cases
func assetPolicyTests(r *policy.ReportCollection, assetMrn string, assetObj *inventory.Asset, queries map[string]*explorer.Mquery) junit.Testsuite {
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

	// jUnit is not able to handle meta information of policies and also does not support
	// data query results.
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
				ts.Failures++
			}

			if score.Type == policy.ScoreType_Result && score.Value != 100 {
				testCase.Failure = &junit.Result{
					Message: "results do not match",
					Type:    "fail",
				}
				ts.Failures++
			}
		}
		ts.Testcases = append(ts.Testcases, testCase)
	}

	return ts
}

// assetPolicyTests converts asset vulnerability results to Junit test cases
func assetMvdTests(r *policy.ReportCollection, assetMrn string, assetObj *inventory.Asset) *junit.Testsuite {
	// check if we have a vulnerability report
	results, ok := r.Reports[assetMrn]
	if !ok {
		return nil
	}

	rawResults := results.RawResults()
	value, _ := getVulnReport(rawResults)
	if value == nil || value.Data == nil {
		return nil
	}

	ts := &junit.Testsuite{
		Name:      "Vulnerability Report for " + assetObj.Name,
		Tests:     0,
		Failures:  0,
		Time:      "",
		Testcases: []junit.Testcase{},
	}

	if value.Data.Error != nil {
		ts.Errors++
		ts.Testcases = append(ts.Testcases, junit.Testcase{
			Failure: &junit.Result{
				Message: "could not load the vulnerability report: " + value.Data.Error.Error(),
				Type:    "fail",
			},
		})
		return ts
	}

	// parse the vulnerability report
	rawData := value.Data.Value
	var vulnReport mvd.VulnReport
	cfg := &mapstructure.DecoderConfig{
		Metadata: nil,
		Result:   &vulnReport,
		TagName:  "json",
	}
	decoder, _ := mapstructure.NewDecoder(cfg)
	if err := decoder.Decode(rawData); err != nil {
		ts.Errors++
		ts.Testcases = append(ts.Testcases, junit.Testcase{
			Failure: &junit.Result{
				Message: "could not decode advisory report",
				Type:    "fail",
			},
		})
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
			if pkg == nil || pkg.Affected == false {
				continue
			}

			testCase := junit.Testcase{
				Classname: "vulnerability",
				Name:      pkg.Name,
				Time:      "",
				Failure:   nil,
			}

			if pkg.Affected == true {
				ts.Failures++

				var content string
				content += pkg.Name + "with version" + pkg.Version + " has known vulnerabilities"
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

func iota32(i int32) string {
	return strconv.FormatInt(int64(i), 10)
}
