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
	"errors"
	"fmt"
	"strconv"
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
	// Aggregated reports (pytest, Maven Surefire, Jest) nest <testsuite> inside
	// <testsuite>; without this the nested cases (and their failures) are dropped.
	Suites []testsuite `xml:"testsuite"`
}

type testcase struct {
	Name      string  `xml:"name,attr"`
	Classname string  `xml:"classname,attr"`
	File      string  `xml:"file,attr"`
	Line      string  `xml:"line,attr"`
	Failure   *result `xml:"failure"`
	Error     *result `xml:"error"`
	SystemOut string  `xml:"system-out"`
	SystemErr string  `xml:"system-err"`
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

	// seen disambiguates test cases that share the same suite + name (e.g.
	// parameterized tests or merged reports), which would otherwise produce
	// identical ids and collapse into one finding downstream.
	seen := map[string]int{}

	var docs []*fex.FindingDocument
	for _, ts := range suites {
		for _, c := range flatten(ts, "") {
			res, rating := failureOf(c.tc)
			if res == nil {
				continue // passing or skipped case — not a finding
			}
			source := &fex.Source{Name: sourceName(c.suiteName)}
			f := toFex(c.tc, res, rating, source)
			key := source.Name + "\x00" + f.Ref
			if n := seen[key]; n > 0 {
				// Keep the first occurrence's id stable; suffix the rest.
				f.Id = shortHash(key + "#" + strconv.Itoa(n))
			}
			seen[key]++
			docs = append(docs, fex.FexToDocument(f))
		}
	}
	return docs, nil
}

// caseInSuite pairs a test case with the name of the suite that contains it.
type caseInSuite struct {
	tc        testcase
	suiteName string
}

// flatten walks the suite tree (aggregated reports nest <testsuite> inside
// <testsuite>) and returns every test case with its effective suite name. A
// nested suite without a name inherits its parent's name.
func flatten(ts testsuite, parentName string) []caseInSuite {
	name := ts.Name
	if name == "" {
		name = parentName
	}
	out := make([]caseInSuite, 0, len(ts.Testcases))
	for _, tc := range ts.Testcases {
		out = append(out, caseInSuite{tc: tc, suiteName: name})
	}
	for _, child := range ts.Suites {
		out = append(out, flatten(child, name)...)
	}
	return out
}

// parse accepts either a <testsuites> root or a bare <testsuite>.
func parse(data []byte) ([]testsuite, error) {
	var root testsuites
	err := xml.Unmarshal(data, &root)
	if err == nil && len(root.Suites) > 0 {
		return root.Suites, nil
	}
	// A genuine syntax error won't be fixed by trying a different root element —
	// surface it instead of falling through to a misleading structural message.
	var syntaxErr *xml.SyntaxError
	if errors.As(err, &syntaxErr) {
		return nil, fmt.Errorf("parse JUnit XML: %w", err)
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
			Description: description(tc, res),
			Severity:    &fex.Severity{Rating: rating},
		},
		Affects: affects(tc),
	}
}

// affects maps the testcase's file/line attributes (emitted by many JUnit
// producers) to a FileComponent so the finding points at the source location.
func affects(tc testcase) []*fex.Affects {
	file := strings.TrimSpace(tc.File)
	if file == "" {
		return nil
	}
	fc := &fex.FileComponent{Path: file}
	if n, err := strconv.Atoi(strings.TrimSpace(tc.Line)); err == nil && n > 0 {
		fc.StartLine = int32(n)
	}
	return []*fex.Affects{{Component: &fex.Component{
		Id:      file,
		Details: &fex.Component_File{File: fc},
	}}}
}

func description(tc testcase, res *result) string {
	parts := make([]string, 0, 4)
	if res.Message != "" {
		parts = append(parts, res.Message)
	}
	if c := strings.TrimSpace(res.Contents); c != "" {
		parts = append(parts, c)
	}
	// system-out/system-err carry extra failure context (captured stdout/stderr).
	if o := strings.TrimSpace(tc.SystemOut); o != "" {
		parts = append(parts, "system-out:\n"+o)
	}
	if e := strings.TrimSpace(tc.SystemErr); e != "" {
		parts = append(parts, "system-err:\n"+e)
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
