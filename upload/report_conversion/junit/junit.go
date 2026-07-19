// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Package junit converts a JUnit XML report into Mondoo FEX findings. Each test
// case with a <failure> or <error> becomes one finding; passing and skipped
// cases are ignored. JUnit is a widely-emitted open format, so security test
// suites and CI gates that report JUnit can be imported without a tool-specific
// converter.
package junit

import (
	"crypto/sha256"
	"encoding/xml"
	"fmt"
	"strings"

	rc "go.mondoo.com/cnspec/v13/upload/report_conversion"
	"go.mondoo.com/mql/v13/providers-sdk/v1/upstream/fex"
)

func init() { rc.Register("junit", Convert) }

// testsuites is the root; some tools omit it and emit a bare <testsuite>, so both
// are handled (see Convert).
type testsuites struct {
	XMLName xml.Name    `xml:"testsuites"`
	Suites  []testsuite `xml:"testsuite"`
}

type testsuite struct {
	XMLName   xml.Name   `xml:"testsuite"`
	Name      string     `xml:"name,attr"`
	Testcases []testcase `xml:"testcase"`
}

type testcase struct {
	Name      string  `xml:"name,attr"`
	Classname string  `xml:"classname,attr"`
	Failure   *result `xml:"failure"`
	Error     *result `xml:"error"`
}

type result struct {
	Message  string `xml:"message,attr"`
	Type     string `xml:"type,attr"`
	Contents string `xml:",chardata"`
}

// Convert parses a JUnit XML report and returns one FEX document per failed or
// errored test case.
func Convert(data []byte) ([]*fex.FindingDocument, error) {
	suites, err := parse(data)
	if err != nil {
		return nil, err
	}

	var docs []*fex.FindingDocument
	for _, ts := range suites {
		source := &fex.Source{Name: sourceName(ts.Name)}
		for _, tc := range ts.Testcases {
			res, rating := failureOf(tc)
			if res == nil {
				continue // passing or skipped case — not a finding
			}
			docs = append(docs, fex.FexToDocument(toFex(tc, res, rating, source)))
		}
	}
	return docs, nil
}

// parse accepts either a <testsuites> root or a bare <testsuite>.
func parse(data []byte) ([]testsuite, error) {
	var root testsuites
	if err := xml.Unmarshal(data, &root); err == nil && len(root.Suites) > 0 {
		return root.Suites, nil
	}
	var single testsuite
	if err := xml.Unmarshal(data, &single); err != nil {
		return nil, fmt.Errorf("parse JUnit XML: %w", err)
	}
	if single.XMLName.Local != "testsuite" {
		return nil, fmt.Errorf("parse JUnit XML: no testsuite element found")
	}
	return []testsuite{single}, nil
}

// failureOf returns the failure/error result of a test case and its severity, or
// nil if the case passed or was skipped. An <error> is rated higher than a plain
// <failure>.
func failureOf(tc testcase) (*result, fex.SeverityRating) {
	if tc.Error != nil {
		return tc.Error, fex.SeverityRating_SEVERITY_RATING_HIGH
	}
	if tc.Failure != nil {
		return tc.Failure, fex.SeverityRating_SEVERITY_RATING_MEDIUM
	}
	return nil, fex.SeverityRating_SEVERITY_RATING_UNSPECIFIED
}

func toFex(tc testcase, res *result, rating fex.SeverityRating, source *fex.Source) *fex.FindingExchange {
	name := caseName(tc)
	summary := name
	if res.Message != "" {
		summary = res.Message
	}
	if summary == "" {
		summary = "Test failure"
	}
	if name == "" {
		name = summary
	}
	return &fex.FindingExchange{
		Id:      shortHash(source.Name + "\x00" + name),
		Ref:     name,
		Summary: summary,
		Source:  source,
		Status:  fex.Status_STATUS_AFFECTED,
		Details: &fex.FindingDetail{
			Category:    fex.FindingDetail_CATEGORY_SECURITY,
			Description: description(res),
			Severity:    &fex.Severity{Rating: rating},
		},
	}
}

func description(res *result) string {
	parts := make([]string, 0, 2)
	if res.Message != "" {
		parts = append(parts, res.Message)
	}
	if c := strings.TrimSpace(res.Contents); c != "" {
		parts = append(parts, c)
	}
	return strings.Join(parts, "\n\n")
}

func caseName(tc testcase) string {
	switch {
	case tc.Classname != "" && tc.Name != "":
		return tc.Classname + "." + tc.Name
	case tc.Name != "":
		return tc.Name
	default:
		return tc.Classname
	}
}

func sourceName(name string) string {
	if name == "" {
		return "junit"
	}
	return name
}

func shortHash(s string) string {
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h)[:16]
}
