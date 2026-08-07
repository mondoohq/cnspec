// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"sort"
	"strconv"
	"strings"

	"go.mondoo.com/cnspec/v13/policy"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/utils/stringx"
)

// Helpers that extract the human-readable documentation of a check (description,
// MQL, audit steps, remediation, references, compliance mappings) and the severity
// derived from its impact. They are shared by the JUnit and SARIF reporters so both
// surface the same content.

// reporterQueryMap indexes a bundle's queries the way the reporters look them up:
// by code id, falling back to the MRN. Several queries can share a code id (e.g.
// variants that compile to the same MQL), so the one with the lowest MRN wins.
// policy.PolicyBundleMap.QueryMap picks an arbitrary one instead - it iterates a
// map - which makes SARIF rule ids and JUnit test names flip between runs of the
// same report.
func reporterQueryMap(bundle *policy.PolicyBundleMap) map[string]*policy.Mquery {
	res := make(map[string]*policy.Mquery, len(bundle.Queries))
	for _, key := range sortedKeys(bundle.Queries) {
		query := bundle.Queries[key]
		if query == nil {
			continue
		}
		id := query.CodeId
		if id == "" {
			id = query.Mrn
		}
		if id == "" {
			continue
		}
		if _, ok := res[id]; ok {
			continue
		}
		res[id] = query
	}
	return res
}

// queryDescription extracts a description from a query
func queryDescription(query *policy.Mquery) string {
	if query.Docs != nil && query.Docs.Desc != "" {
		return query.Docs.Desc
	}
	if query.Desc != "" {
		return query.Desc
	}
	return ""
}

// queryMql returns the MQL source for a query, preferring the current field and
// falling back to the deprecated one (which the compact reporter still reads).
func queryMql(query *policy.Mquery) string {
	if query.Mql != "" {
		return query.Mql
	}
	return query.Query
}

// queryAudit returns the manual audit instructions of a check, if it has any.
func queryAudit(query *policy.Mquery) string {
	if query.Docs == nil {
		return ""
	}
	return strings.TrimSpace(query.Docs.Audit)
}

// platformRemediationKeys returns the set of remediation ids relevant to an
// asset's platform: the platform name, its family entries (e.g. "terraform" for
// the "terraform-hcl" platform), and the platform-agnostic "default"/"" ids. It
// is used to filter remediation down to the platform being scanned so a Terraform
// scan shows Terraform remediation rather than every IaC/tool variant.
func platformRemediationKeys(platform *inventory.Platform) map[string]bool {
	keys := map[string]bool{"": true, "default": true}
	if platform != nil {
		if platform.Name != "" {
			keys[strings.ToLower(platform.Name)] = true
		}
		for _, f := range platform.Family {
			if f != "" {
				keys[strings.ToLower(f)] = true
			}
		}
	}
	return keys
}

// remediationItems returns the remediation entries of a query that apply to the
// asset's platform (name/family) or that are platform-agnostic. If none match, all
// items are returned so remediation is never dropped entirely.
func remediationItems(query *policy.Mquery, platformKeys map[string]bool) []*policy.TypedDoc {
	if query.Docs == nil || query.Docs.Remediation == nil {
		return nil
	}

	var matched, all []*policy.TypedDoc
	for _, item := range query.Docs.Remediation.Items {
		if item == nil || strings.TrimSpace(item.Desc) == "" {
			continue
		}
		all = append(all, item)
		if platformKeys[strings.ToLower(item.Id)] {
			matched = append(matched, item)
		}
	}

	if len(matched) == 0 {
		return all // fallback: no platform-specific match, show everything
	}
	return matched
}

// queryRemediation renders the remediation for a query as plain text, labeling
// each item with its platform/tool id (e.g. "[terraform]") when present.
func queryRemediation(query *policy.Mquery, platformKeys map[string]bool) string {
	var b strings.Builder
	for _, item := range remediationItems(query, platformKeys) {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		if item.Id != "" && item.Id != "default" {
			b.WriteString("[" + item.Id + "] ")
		}
		b.WriteString(strings.TrimSpace(item.Desc))
	}
	return b.String()
}

// queryRefs returns the references of a query. It prefers docs.refs (the canonical
// location) and falls back to the deprecated refs field.
func queryRefs(query *policy.Mquery) []*policy.MqueryRef {
	refs := query.Refs
	if query.Docs != nil && len(query.Docs.Refs) > 0 {
		refs = query.Docs.Refs
	}

	res := make([]*policy.MqueryRef, 0, len(refs))
	for _, ref := range refs {
		if ref == nil || ref.Url == "" {
			continue
		}
		res = append(res, ref)
	}
	return res
}

// queryReferences renders a query's references as "Title: URL" lines.
func queryReferences(query *policy.Mquery) string {
	var b strings.Builder
	for _, ref := range queryRefs(query) {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		if ref.Title != "" {
			b.WriteString(ref.Title + ": ")
		}
		b.WriteString(ref.Url)
	}
	return b.String()
}

// queryComplianceTags returns the compliance framework mappings of a check, keyed
// by framework (e.g. "compliance/iso-27001-2022" -> "iso-27001-2022-a-8-24").
// Entries that are explicitly turned off (value "false") are dropped.
func queryComplianceTags(query *policy.Mquery) map[string]string {
	res := map[string]string{}
	for k, v := range query.Tags {
		if !strings.HasPrefix(k, "compliance/") || v == "" || v == "false" {
			continue
		}
		res[k] = v
	}
	return res
}

// failingResourceLocations lists the source locations (path:line) of the resources
// that caused a check to fail. It is populated for resources that carry source
// context (e.g. Terraform/HCL) and empty for scalar checks.
func failingResourceLocations(cb *llx.CodeBundle, assessment *llx.Assessment) string {
	var b strings.Builder
	for _, sc := range cb.FailingResourceContexts(assessment) {
		if sc.Path == "" {
			continue
		}
		loc := sc.Path
		if startLine, _, _, _, _, ok := sc.Range.Bounds(); ok && startLine >= 1 {
			loc += ":" + strconv.FormatInt(int64(startLine), 10)
		}
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString(loc)
	}
	return b.String()
}

// writeDetailSection appends an indented "Title:\n  body" section to b. It is the
// plain-text section format used in JUnit failure bodies and SARIF rule help.
func writeDetailSection(b *strings.Builder, title, body string) {
	body = strings.TrimSpace(body)
	if body == "" {
		return
	}
	if b.Len() > 0 {
		b.WriteString("\n")
	}
	b.WriteString(title)
	b.WriteString(":\n")
	b.WriteString(stringx.Indent(2, body))
	b.WriteString("\n")
}

// policyTitlesByQuery maps a check MRN to the titles of the policies that include
// it, so a finding can be attributed to the policy it came from.
func policyTitlesByQuery(bundle *policy.PolicyBundleMap) map[string][]string {
	res := map[string][]string{}
	if bundle == nil {
		return res
	}

	for _, p := range bundle.Policies {
		if p == nil {
			continue
		}
		title := p.Name
		if title == "" {
			title = p.Mrn
		}
		if title == "" {
			continue
		}

		for _, group := range p.Groups {
			if group == nil {
				continue
			}
			for _, check := range group.Checks {
				if check == nil || check.Mrn == "" {
					continue
				}
				if !stringx.Contains(res[check.Mrn], title) {
					res[check.Mrn] = append(res[check.Mrn], title)
				}
			}
		}
	}

	for k := range res {
		sort.Strings(res[k])
	}
	return res
}

// queryImpact returns the configured impact of a check (0-100, where 100 is the
// most impactful) and whether it is set at all.
func queryImpact(query *policy.Mquery) (int32, bool) {
	if query == nil || query.Impact == nil || query.Impact.Value == nil {
		return 0, false
	}
	return query.Impact.Value.Value, true
}

// riskSeverityLabel maps a cnspec risk value (0-100, the inverse of a score value)
// to the severity label cnspec uses everywhere else. The bands mirror
// policy.ScoreRatingsText:
//
//	90 .. 100 → CRITICAL
//	70 ..  89 → HIGH
//	40 ..  69 → MEDIUM
//	 1 ..  39 → LOW
//	        0 → NONE
func riskSeverityLabel(risk int32) string {
	switch {
	case risk >= 90:
		return policy.ScoreRatingTextCritical
	case risk >= 70:
		return policy.ScoreRatingTextHigh
	case risk >= 40:
		return policy.ScoreRatingTextMedium
	case risk >= 1:
		return policy.ScoreRatingTextLow
	default:
		return policy.ScoreRatingTextNone
	}
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
