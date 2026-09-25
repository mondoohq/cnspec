// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"sort"
	"strconv"
	"strings"

	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/mql/llx"
)

// Coverage attribution (mql ADR-46).
//
// A check that could not be assessed says why in its score's error details,
// and a check assessed on incomplete data says what it is missing. Counted over
// the scan, that answers how much of a policy was covered and which grants
// would cover the rest. Asset errors are counted the same way, so fifty assets
// failing on one expired credential read as one line.

// kindCounts counts checks, or assets, per ErrorKind. One that carries several
// kinds counts once for each; one without a classified kind counts as
// unclassified.
type kindCounts struct {
	kinds        map[llx.ErrorKind]int
	unclassified int
}

func (k *kindCounts) add(details []*llx.ErrorDetail) {
	seen := map[llx.ErrorKind]struct{}{}
	for _, d := range details {
		kind := d.GetKind()
		if kind == llx.ErrorKind_ERROR_KIND_UNSPECIFIED {
			continue
		}
		if _, ok := seen[kind]; ok {
			continue
		}
		seen[kind] = struct{}{}
		if k.kinds == nil {
			k.kinds = map[llx.ErrorKind]int{}
		}
		k.kinds[kind]++
	}
	if len(seen) == 0 {
		k.unclassified++
	}
}

// String is the breakdown, most frequent first:
// "58 access denied, 3 not applicable, 1 unclassified".
func (k *kindCounts) String() string {
	kinds := make([]llx.ErrorKind, 0, len(k.kinds))
	for kind := range k.kinds {
		kinds = append(kinds, kind)
	}
	sort.Slice(kinds, func(i, j int) bool {
		if k.kinds[kinds[i]] != k.kinds[kinds[j]] {
			return k.kinds[kinds[i]] > k.kinds[kinds[j]]
		}
		return kinds[i] < kinds[j]
	})
	parts := make([]string, 0, len(kinds)+1)
	for _, kind := range kinds {
		parts = append(parts, strconv.Itoa(k.kinds[kind])+" "+kind.Label())
	}
	if k.unclassified != 0 {
		parts = append(parts, strconv.Itoa(k.unclassified)+" unclassified")
	}
	return strings.Join(parts, ", ")
}

type coverageReport struct {
	checks int
	// Checks whose score is an error.
	unassessed      int
	unassessedKinds kindCounts
	// Checks that were assessed on incomplete data: a result with coverage gaps.
	partial      int
	partialKinds kindCounts

	assets      int
	failed      int
	failedKinds kindCounts

	permissions map[string]struct{}
	// Whether anything was classified at all. Without it the report has
	// nothing to say that the scan does not already, so nothing is printed.
	classified bool
}

func (c *coverageReport) addDetails(details []*llx.ErrorDetail) {
	for _, d := range details {
		if d.GetKind() != llx.ErrorKind_ERROR_KIND_UNSPECIFIED {
			c.classified = true
		}
		for _, p := range d.GetPermissions() {
			if c.permissions == nil {
				c.permissions = map[string]struct{}{}
			}
			c.permissions[p] = struct{}{}
		}
	}
}

func (c *coverageReport) addCheck(score *policy.Score) {
	c.checks++
	switch {
	case score.Type == policy.ScoreType_Error:
		c.unassessed++
		c.unassessedKinds.add(score.ErrorDetails)
		c.addDetails(score.ErrorDetails)
	case score.Type == policy.ScoreType_Result && len(score.ErrorDetails) != 0:
		c.partial++
		c.partialKinds.add(score.ErrorDetails)
		c.addDetails(score.ErrorDetails)
	}
}

// newCoverageReport counts the checks of every scanned asset, and the assets
// that could not be scanned.
func newCoverageReport(data *policy.ReportCollection) *coverageReport {
	res := &coverageReport{assets: len(data.Assets)}
	for mrn, report := range data.Reports {
		if report == nil {
			continue
		}
		resolved := data.ResolvedPolicies[mrn]
		if resolved == nil || resolved.CollectorJob == nil {
			continue
		}
		for _, job := range resolved.CollectorJob.ReportingJobs {
			if job.Type != policy.ReportingJob_CHECK && job.Type != policy.ReportingJob_CHECK_AND_DATA_QUERY {
				continue
			}
			if score := report.Scores[job.QrId]; score != nil {
				res.addCheck(score)
			}
		}
	}

	for mrn := range data.Errors {
		res.failed++
		var details []*llx.ErrorDetail
		if d := data.ErrorDetails[mrn]; d != nil {
			details = []*llx.ErrorDetail{d}
		}
		res.failedKinds.add(details)
		res.addDetails(details)
	}
	return res
}

// lines is the coverage summary, or nothing when no error was classified.
func (c *coverageReport) lines() []string {
	if !c.classified {
		return nil
	}
	var res []string
	if c.unassessed != 0 {
		res = append(res, strconv.Itoa(c.unassessed)+" of "+strconv.Itoa(c.checks)+" "+plural(c.checks, "check", "checks")+
			" could not be assessed: "+c.unassessedKinds.String()+".")
	}
	if c.partial != 0 {
		res = append(res, strconv.Itoa(c.partial)+" "+plural(c.partial, "check was", "checks were")+
			" assessed on incomplete data: "+c.partialKinds.String()+".")
	}
	if c.failed != 0 {
		res = append(res, strconv.Itoa(c.failed)+" of "+strconv.Itoa(c.assets)+" "+plural(c.assets, "asset", "assets")+
			" could not be scanned: "+c.failedKinds.String()+".")
	}
	if len(c.permissions) != 0 {
		perms := make([]string, 0, len(c.permissions))
		services := map[string]struct{}{}
		for p := range c.permissions {
			perms = append(perms, p)
			services[permissionService(p)] = struct{}{}
		}
		sort.Strings(perms)
		serviceList := make([]string, 0, len(services))
		for s := range services {
			serviceList = append(serviceList, s)
		}
		sort.Strings(serviceList)
		res = append(res, "Missing permissions ("+strconv.Itoa(len(perms))+" across "+strings.Join(serviceList, ", ")+"): "+
			strings.Join(perms, ", "))
	}
	return res
}

// permissionService is the service a permission belongs to: "ec2" for
// "ec2:DescribeInstances", "compute" for "compute.instances.list".
func permissionService(p string) string {
	if i := strings.IndexAny(p, ":."); i > 0 {
		return p[:i]
	}
	return p
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// describeErrorDetail names the kind, the partition, and the permissions, e.g.
// "access denied in eu-west-1 (ec2:DescribeAddresses)", the way mql prints a
// classified error.
func describeErrorDetail(d *llx.ErrorDetail) string {
	var res strings.Builder
	res.WriteString(d.GetKind().Label())
	if id := d.GetScopeId(); id != "" {
		res.WriteString(" in ")
		res.WriteString(id)
	}
	if len(d.GetPermissions()) != 0 {
		perms := append([]string(nil), d.GetPermissions()...)
		sort.Strings(perms)
		res.WriteString(" (")
		res.WriteString(strings.Join(perms, ", "))
		res.WriteString(")")
	}
	return res.String()
}

// coverageGapLines is one line per (kind, scope_id) group of what a check's
// data is missing, with the permissions of the group merged, in the words mql
// uses below a partial result: "coverage gap: access denied in eu-west-1
// (ec2:DescribeAddresses)".
func coverageGapLines(details []*llx.ErrorDetail) []string {
	type group struct {
		detail *llx.ErrorDetail
		perms  map[string]struct{}
	}
	groups := map[string]*group{}
	for _, d := range details {
		key := d.GetKind().String() + "\x00" + d.GetScopeId()
		g, ok := groups[key]
		if !ok {
			g = &group{detail: &llx.ErrorDetail{Kind: d.GetKind(), ScopeId: d.GetScopeId()}, perms: map[string]struct{}{}}
			groups[key] = g
		}
		for _, p := range d.GetPermissions() {
			g.perms[p] = struct{}{}
		}
	}
	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	res := make([]string, len(keys))
	for i, k := range keys {
		g := groups[k]
		for p := range g.perms {
			g.detail.Permissions = append(g.detail.Permissions, p)
		}
		res[i] = "coverage gap: " + describeErrorDetail(g.detail)
	}
	return res
}
