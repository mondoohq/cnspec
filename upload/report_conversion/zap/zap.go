// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Package zap converts an OWASP ZAP XML report into Mondoo FEX findings. Each
// ZAP alert becomes one FEX finding (category SECURITY); the affected URLs and
// their request context come from the alert's instances. ZAP is a widely-used
// open-source DAST scanner.
package zap

import (
	"crypto/sha256"
	"encoding/xml"
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"

	rc "go.mondoo.com/cnspec/v13/upload/report_conversion"
	"go.mondoo.com/mql/v13/providers-sdk/v1/upstream/fex"
)

func init() { rc.Register("zap", Convert) }

type zapReport struct {
	XMLName xml.Name  `xml:"OWASPZAPReport"`
	Sites   []zapSite `xml:"site"`
}

type zapSite struct {
	Name   string     `xml:"name,attr"`
	Alerts []zapAlert `xml:"alerts>alertitem"`
}

type zapAlert struct {
	PluginID   string        `xml:"pluginid"`
	Name       string        `xml:"name"`
	RiskCode   string        `xml:"riskcode"`
	Confidence string        `xml:"confidence"`
	Desc       string        `xml:"desc"`
	Solution   string        `xml:"solution"`
	Reference  string        `xml:"reference"`
	CWEID      string        `xml:"cweid"`
	OtherInfo  string        `xml:"otherinfo"`
	Instances  []zapInstance `xml:"instances>instance"`
}

type zapInstance struct {
	URI      string `xml:"uri"`
	Method   string `xml:"method"`
	Param    string `xml:"param"`
	Attack   string `xml:"attack"`
	Evidence string `xml:"evidence"`
}

// Convert parses an OWASP ZAP XML report and returns one FEX document per alert.
func Convert(data []byte) ([]*fex.FindingDocument, error) {
	var report zapReport
	if err := xml.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("parse ZAP XML: %w", err)
	}
	if report.XMLName.Local != "OWASPZAPReport" {
		return nil, fmt.Errorf("parse ZAP XML: not an OWASPZAPReport document")
	}

	var docs []*fex.FindingDocument
	for _, site := range report.Sites {
		source := &fex.Source{Name: sourceName(site.Name)}
		for _, a := range site.Alerts {
			docs = append(docs, fex.FexToDocument(toFex(a, source)))
		}
	}
	return docs, nil
}

func toFex(a zapAlert, source *fex.Source) *fex.FindingExchange {
	id := a.PluginID
	if id == "" {
		id = shortHash(a.Name)
	}
	summary := clean(a.Name)
	if summary == "" {
		summary = "ZAP alert"
	}
	return &fex.FindingExchange{
		Id:      id,
		Ref:     a.PluginID,
		Summary: summary,
		Source:  source,
		Status:  fex.Status_STATUS_AFFECTED,
		Details: &fex.FindingDetail{
			Category:    fex.FindingDetail_CATEGORY_SECURITY,
			Description: description(a),
			Severity:    severity(a.RiskCode),
			Confidence:  confidence(a.Confidence),
			References:  references(a),
		},
		Affects:      affects(a),
		Remediations: remediations(a),
	}
}

// affects lists the vulnerable URLs. The DAST request context (method, param,
// attack, evidence) is attached as component properties for now; a first-class
// HTTP-request evidence type is a planned FEX proto extension (ADR-062).
func affects(a zapAlert) []*fex.Affects {
	var out []*fex.Affects
	seen := map[string]bool{}
	for _, in := range a.Instances {
		uri := clean(in.URI)
		if uri == "" || seen[uri] {
			continue
		}
		seen[uri] = true
		props := map[string]string{}
		putIf(props, "method", clean(in.Method))
		putIf(props, "param", clean(in.Param))
		putIf(props, "attack", clean(in.Attack))
		putIf(props, "evidence", clean(in.Evidence))
		out = append(out, &fex.Affects{Component: &fex.Component{Id: uri, Properties: props}})
	}
	return out
}

func references(a zapAlert) []*fex.Reference {
	var out []*fex.Reference
	if n, err := strconv.Atoi(strings.TrimSpace(a.CWEID)); err == nil && n > 0 {
		out = append(out, &fex.Reference{Type: "CWE", Name: fmt.Sprintf("CWE-%d", n)})
	}
	for _, url := range splitLines(clean(a.Reference)) {
		out = append(out, &fex.Reference{Name: "reference", Url: url})
	}
	return out
}

func remediations(a zapAlert) []*fex.Remediation {
	sol := clean(a.Solution)
	if sol == "" {
		return nil
	}
	return []*fex.Remediation{{Summary: "Solution", Details: sol}}
}

func description(a zapAlert) string {
	parts := make([]string, 0, 2)
	if d := clean(a.Desc); d != "" {
		parts = append(parts, d)
	}
	if o := clean(a.OtherInfo); o != "" {
		parts = append(parts, o)
	}
	return strings.Join(parts, "\n\n")
}

// severity maps ZAP riskcode (0..3) to a severity rating.
func severity(riskcode string) *fex.Severity {
	var rating fex.SeverityRating
	switch strings.TrimSpace(riskcode) {
	case "3":
		rating = fex.SeverityRating_SEVERITY_RATING_HIGH
	case "2":
		rating = fex.SeverityRating_SEVERITY_RATING_MEDIUM
	case "1":
		rating = fex.SeverityRating_SEVERITY_RATING_LOW
	case "0":
		rating = fex.SeverityRating_SEVERITY_RATING_NONE
	default:
		return nil
	}
	return &fex.Severity{Rating: rating}
}

// confidence maps ZAP confidence (0..4) to the FEX confidence enum.
func confidence(c string) fex.Confidence {
	switch strings.TrimSpace(c) {
	case "3", "4":
		return fex.Confidence_CONFIDENCE_HIGH
	case "2":
		return fex.Confidence_CONFIDENCE_MEDIUM
	case "1":
		return fex.Confidence_CONFIDENCE_LOW
	default:
		return fex.Confidence_CONFIDENCE_UNSPECIFIED
	}
}

var tagRe = regexp.MustCompile(`<[^>]*>`)

// clean strips the simple HTML ZAP wraps some fields in and unescapes entities.
func clean(s string) string {
	s = tagRe.ReplaceAllString(s, "")
	return strings.TrimSpace(html.UnescapeString(s))
}

func splitLines(s string) []string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		if l := strings.TrimSpace(line); l != "" {
			out = append(out, l)
		}
	}
	return out
}

func putIf(m map[string]string, k, v string) {
	if v != "" {
		m[k] = v
	}
}

func sourceName(name string) string {
	if name == "" {
		return "zap"
	}
	return name
}

func shortHash(s string) string {
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h)[:16]
}
