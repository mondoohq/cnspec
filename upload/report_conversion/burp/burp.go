// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Package burp converts a Burp Suite XML report (the <issues> export) into
// Mondoo FEX findings. Each Burp issue becomes one FEX finding (category
// SECURITY), with the affected URL from the issue's host+path. Burp Suite is a
// widely-used web-application (DAST) scanner.
package burp

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"html"
	"regexp"
	"strings"

	rc "go.mondoo.com/cnspec/v13/upload/report_conversion"
	"go.mondoo.com/mql/v13/providers-sdk/v1/upstream/fex"
)

func init() { rc.Register("burp", Convert) }

type burpReport struct {
	XMLName xml.Name    `xml:"issues"`
	Issues  []burpIssue `xml:"issue"`
}

type burpIssue struct {
	SerialNumber      string                `xml:"serialNumber"`
	Type              string                `xml:"type"`
	Name              string                `xml:"name"`
	Host              burpHost              `xml:"host"`
	Path              string                `xml:"path"`
	Location          string                `xml:"location"`
	Severity          string                `xml:"severity"`
	Confidence        string                `xml:"confidence"`
	Background        string                `xml:"issueBackground"`
	Detail            string                `xml:"issueDetail"`
	Remediation       string                `xml:"remediationBackground"`
	RemediationDetail string                `xml:"remediationDetail"`
	Classification    string                `xml:"vulnerabilityClassifications"`
	RequestResponses  []burpRequestResponse `xml:"requestresponse"`
}

type burpHost struct {
	IP    string `xml:"ip,attr"`
	Value string `xml:",chardata"`
}

type burpRequestResponse struct {
	Request  burpData `xml:"request"`
	Response burpData `xml:"response"`
}

// burpData is a request or response payload. Burp base64-encodes these when
// base64="true" (the common case); otherwise the raw bytes are inline.
type burpData struct {
	Base64 bool   `xml:"base64,attr"`
	Value  string `xml:",chardata"`
}

// decode returns the payload, base64-decoding it when Burp marked it encoded.
// If decoding fails, the raw value is returned unchanged.
func (d burpData) decode() string {
	if d.Base64 {
		if decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(d.Value)); err == nil {
			return string(decoded)
		}
	}
	return d.Value
}

var (
	tagRe    = regexp.MustCompile(`<[^>]*>`)
	cweRe    = regexp.MustCompile(`CWE-(\d+)`)
	locParam = regexp.MustCompile(`\[(.*)\]`)
)

// Convert parses a Burp Suite XML report and returns one FEX document per issue.
func Convert(data []byte) ([]*fex.FindingDocument, error) {
	var report burpReport
	if err := xml.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("parse Burp XML: %w", err)
	}
	if report.XMLName.Local != "issues" {
		return nil, fmt.Errorf("parse Burp XML: not a Burp <issues> document")
	}

	source := &fex.Source{Name: "burp"}
	docs := make([]*fex.FindingDocument, 0, len(report.Issues))
	for _, iss := range report.Issues {
		docs = append(docs, fex.FexToDocument(toFex(iss, source)))
	}
	return docs, nil
}

func toFex(iss burpIssue, source *fex.Source) *fex.FindingExchange {
	// Burp regenerates serialNumber on every scan, so it is not stable across
	// re-uploads. Derive a deterministic id from the issue's identity (type +
	// host + path + location) so the same finding keeps the same id.
	id := shortHash(strings.Join([]string{
		iss.Type,
		clean(iss.Host.Value),
		clean(iss.Path),
		clean(iss.Location),
	}, "\x00"))
	summary := clean(iss.Name)
	if summary == "" {
		summary = "Burp issue"
	}
	return &fex.FindingExchange{
		Id:      id,
		Ref:     iss.Type,
		Summary: summary,
		Source:  source,
		Status:  fex.Status_STATUS_AFFECTED,
		Details: &fex.FindingDetail{
			Category:    fex.FindingDetail_CATEGORY_SECURITY,
			Description: description(iss),
			Severity:    severity(iss.Severity),
			Confidence:  confidence(iss.Confidence),
			References:  references(iss),
		},
		Affects:      affects(iss),
		Evidences:    httpEvidence(iss),
		Remediations: remediations(iss),
	}
}

func affects(iss burpIssue) []*fex.Affects {
	host := clean(iss.Host.Value)
	path := clean(iss.Path)
	if host == "" && path == "" {
		return nil
	}
	comp := &fex.Component{Id: host + path}
	if ip := strings.TrimSpace(iss.Host.IP); ip != "" {
		comp.Identifiers = map[string]string{"ip": ip}
	}
	return []*fex.Affects{{Component: comp}}
}

// httpEvidence carries the captured HTTP request/response pairs (Burp's headline
// evidence) plus the affected URL and tested parameter (from Burp's location,
// e.g. "/x [q parameter]") as first-class HttpRequest evidence. An issue may
// carry multiple requestresponse elements; each becomes one evidence entry.
func httpEvidence(iss burpIssue) []*fex.Evidence {
	host := clean(iss.Host.Value)
	path := clean(iss.Path)
	url := host + path
	var param string
	if m := locParam.FindStringSubmatch(clean(iss.Location)); len(m) == 2 {
		param = strings.TrimSpace(m[1])
	}

	var out []*fex.Evidence
	for _, rr := range iss.RequestResponses {
		out = append(out, &fex.Evidence{Details: &fex.Evidence_HttpRequest{HttpRequest: &fex.HttpRequest{
			Url:      url,
			Param:    param,
			Request:  rr.Request.decode(),
			Response: rr.Response.decode(),
		}}})
	}
	// No captured request/response: still surface the URL context if we have one.
	if len(out) == 0 && url != "" {
		out = append(out, &fex.Evidence{Details: &fex.Evidence_HttpRequest{HttpRequest: &fex.HttpRequest{
			Url:   url,
			Param: param,
		}}})
	}
	return out
}

func references(iss burpIssue) []*fex.Reference {
	var out []*fex.Reference
	seen := map[string]bool{}
	for _, m := range cweRe.FindAllStringSubmatch(iss.Classification, -1) {
		name := "CWE-" + m[1]
		if seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, &fex.Reference{
			Type: "CWE",
			Name: name,
			Url:  "https://cwe.mitre.org/data/definitions/" + m[1] + ".html",
		})
	}
	return out
}

// remediations carries Burp's general remediation guidance (remediationBackground)
// and, when present, the instance-specific remediation (remediationDetail) as a
// separate entry.
func remediations(iss burpIssue) []*fex.Remediation {
	var out []*fex.Remediation
	if rem := clean(iss.Remediation); rem != "" {
		out = append(out, &fex.Remediation{Summary: "Remediation", Details: rem})
	}
	if detail := clean(iss.RemediationDetail); detail != "" {
		out = append(out, &fex.Remediation{Summary: "Remediation detail", Details: detail})
	}
	return out
}

func description(iss burpIssue) string {
	parts := make([]string, 0, 2)
	if d := clean(iss.Detail); d != "" {
		parts = append(parts, d)
	}
	if b := clean(iss.Background); b != "" {
		parts = append(parts, b)
	}
	return strings.Join(parts, "\n\n")
}

// severity maps Burp's severity strings to a severity rating.
func severity(s string) *fex.Severity {
	var rating fex.SeverityRating
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "high":
		rating = fex.SeverityRating_SEVERITY_RATING_HIGH
	case "medium":
		rating = fex.SeverityRating_SEVERITY_RATING_MEDIUM
	case "low":
		rating = fex.SeverityRating_SEVERITY_RATING_LOW
	case "information", "info", "informational":
		rating = fex.SeverityRating_SEVERITY_RATING_NONE
	default:
		return nil
	}
	return &fex.Severity{Rating: rating}
}

// confidence maps Burp's Certain/Firm/Tentative to the FEX confidence enum.
func confidence(c string) fex.Confidence {
	switch strings.ToLower(strings.TrimSpace(c)) {
	case "certain":
		return fex.Confidence_CONFIDENCE_HIGH
	case "firm":
		return fex.Confidence_CONFIDENCE_MEDIUM
	case "tentative":
		return fex.Confidence_CONFIDENCE_LOW
	default:
		return fex.Confidence_CONFIDENCE_UNSPECIFIED
	}
}

// clean strips the HTML Burp wraps text fields in and unescapes entities.
func clean(s string) string {
	s = tagRe.ReplaceAllString(s, "")
	return strings.TrimSpace(html.UnescapeString(s))
}

func shortHash(s string) string {
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h)[:16]
}
