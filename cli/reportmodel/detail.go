// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reportmodel

import (
	"strings"

	"go.mondoo.com/cnspec/cli/reporter"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/mql/cli/printer"
)

// RemediationItem is one way to fix a check. Id names the platform or tool the
// fix applies to ("terraform", "cli", …) and is empty or "default" for a
// platform-agnostic one. Desc is markdown source: this package does not render
// it, so a viewer either prints it plainly or runs it through a renderer of its
// choice.
type RemediationItem struct {
	Id   string
	Desc string
}

// Reference is a link a check cites.
type Reference struct {
	Title string
	Url   string
}

// CheckDetail is everything a detail view shows about one check on one asset. It
// carries the same sections, in the same order, as the JUnit reporter's
// detailedCheckBody - description, MQL, assessment, failing locations, error,
// remediation, references - but as fields rather than as one formatted string,
// so a pane can lay them out itself.
//
// Every field may be empty; a viewer skips the sections that are.
type CheckDetail struct {
	// Title, Status and Severity repeat the Check they came from, so a detail
	// pane needs nothing else to draw its header.
	Title    string
	Status   Status
	Severity string
	// Impact is the configured impact (0-100), HasImpact whether it is set.
	Impact    int32
	HasImpact bool
	// Description is the prose explaining what the check looks for.
	Description string
	// Mql is the query source.
	Mql string
	// Audit is the manual verification instructions, when the check has any.
	Audit string
	// Assessment is the rendered expected-vs-actual of the check, colour-free.
	// It is only available for assertion checks that executed.
	Assessment string
	// FailingLocations are the "path:line" locations of the resources that made
	// the check fail. It is populated for resources that carry source context
	// (Terraform/HCL) and empty for scalar checks.
	FailingLocations []string
	// Error is the reason an errored check could not run. It is empty for every
	// other status.
	Error string
	// Remediation are the fixes that apply to this asset's platform.
	Remediation []RemediationItem
	// References are the links the check cites.
	References []Reference
	// Compliance maps a framework tag to the control the check satisfies, e.g.
	// "compliance/iso-27001-2022" -> "iso-27001-2022-a-8-24".
	Compliance map[string]string
	// Policies are the policies of this asset that include the check.
	Policies []PolicyRef
}

// Detail composes the full detail of a check on its asset.
//
// It is computed on demand rather than when the model is built: rendering an
// assessment compiles nothing but does walk the raw results, and a large scan
// has thousands of checks of which a viewer opens one at a time.
func (c *Check) Detail() CheckDetail {
	if c == nil {
		return CheckDetail{Status: StatusUnknown}
	}

	res := CheckDetail{
		Title:     c.Title,
		Status:    c.Status,
		Severity:  c.Severity,
		Impact:    c.Impact,
		HasImpact: c.HasImpact,
		Policies:  c.Policies,
	}
	if c.Query == nil {
		return res
	}
	query := c.Query

	res.Description = strings.TrimSpace(normalizeNewlines(reporter.QueryDescription(query)))
	res.Mql = strings.TrimSpace(normalizeNewlines(reporter.QueryMql(query)))
	res.Audit = normalizeNewlines(reporter.QueryAudit(query))
	res.Compliance = reporter.QueryComplianceTags(query)

	platformKeys := map[string]bool{"": true, "default": true}
	if c.asset != nil {
		platformKeys = c.asset.platformKeys
	}
	for _, item := range reporter.RemediationItems(query, platformKeys) {
		res.Remediation = append(res.Remediation, RemediationItem{
			Id:   item.Id,
			Desc: strings.TrimSpace(normalizeNewlines(item.Desc)),
		})
	}

	for _, ref := range reporter.QueryRefs(query) {
		res.References = append(res.References, Reference{Title: ref.Title, Url: ref.Url})
	}

	res.Assessment, res.FailingLocations = c.assessment()

	// For an errored check the score message carries the failure reason.
	if c.Score != nil && c.Score.Type == policy.ScoreType_Error {
		res.Error = strings.TrimSpace(normalizeNewlines(c.Score.MessageLine()))
	}

	return res
}

// assessment renders the expected-vs-actual of the check and the source
// locations of the resources that failed it.
//
// The guards matter: ResolvedPolicy.GetCodeBundle dereferences ExecutionJob and
// panics when it is nil, which is the state of a resolved policy that never
// executed.
func (c *Check) assessment() (string, []string) {
	asset := c.asset
	if asset == nil || asset.report == nil || asset.resolved == nil || asset.resolved.ExecutionJob == nil {
		return "", nil
	}

	cb := asset.resolved.GetCodeBundle(c.Query)
	if cb == nil {
		return "", nil
	}

	assessment := policy.Query2Assessment(cb, asset.report)
	if assessment == nil {
		return "", nil
	}

	text := strings.TrimSpace(normalizeNewlines(printer.PlainNoColorPrinter.Assessment(cb, assessment)))

	var locations []string
	if locs := reporter.FailingResourceLocations(cb, assessment); locs != "" {
		locations = strings.Split(normalizeNewlines(locs), "\n")
	}

	return text, locations
}
