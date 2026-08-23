// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Package reportfixture holds the hand-built scan the reporter tests convert.
//
// It is a package rather than a helper in one test file because the JUnit, SARIF,
// OHDF and OCSF reporters now live in separate packages and have to be compared on
// the same scan. It sits at the module root rather than under one of them because
// an internal package is only reachable from below its own parent, and these
// readers are spread across cli/reporter and reports/...: the point of Sample is that one asset carries a passing, an
// errored and a skipped check plus an affected package, so every reporter is shown
// the same four outcomes and their treatment of them can be read side by side.
// Detailed does the same for the check documentation.
package reportfixture

import (
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/providers-sdk/v1/upstream/mvd"
)

// AssetMrn is the single asset Sample reports on.
const AssetMrn = "//assets.api.mondoo.app/spaces/dazzling-golick-767384/assets/2DRZ1cCWFyTYCArycAXHwvn1oU2"

// Sample is a scan of one Ubuntu asset with a passed, an errored and a skipped
// check, and one affected package that no advisory in the report covers.
func Sample() *policy.ReportCollection {
	return &policy.ReportCollection{
		Assets: map[string]*inventory.Asset{
			AssetMrn: {
				Name:        "X1",
				PlatformIds: []string{"//platformid.api.mondoo.app/hostname/X1"},
				State:       inventory.State_STATE_ONLINE,
				Platform: &inventory.Platform{
					Name:    "ubuntu",
					Arch:    "amd64",
					Kind:    "baremetal",
					Version: "22.04",
					Family:  []string{"debian", "linux", "unix", "os"},
				},
			},
		},
		ResolvedPolicies: map[string]*policy.ResolvedPolicy{
			AssetMrn: {
				CollectorJob: &policy.CollectorJob{
					ReportingQueries: map[string]*policy.StringArray{
						"+u6doYoYG5E=": nil,
						"057itYF8s30=": nil,
						"GyJVAziB/tU=": nil,
					},
				},
			},
		},
		Bundle: &policy.Bundle{
			Policies: nil, // not needed for this test since junit does not sort by policy
			Queries: []*policy.Mquery{
				{
					Mrn:    "//policy.api.mondoo.app/queries/mondoo-linux-security-snmp-server-is-not-enabled",
					CodeId: "+u6doYoYG5E=",
					Title:  "Ensure SNMP server is stopped and not enabled",
				},
				{
					Mrn:    "//policy.api.mondoo.app/queries/mondoo-kubernetes-security-kubelet-event-record-qps",
					CodeId: "057itYF8s30=",
					Title:  "Configure kubelet to capture all event creation",
				},
				{
					Mrn:    "//policy.api.mondoo.app/queries/mondoo-kubernetes-security-secure-scheduler_conf",
					CodeId: "GyJVAziB/tU=",
					Title:  "Set secure file permissions on the scheduler.conf file",
				},
			},
		},
		Reports: map[string]*policy.Report{
			AssetMrn: {
				ScoringMrn: AssetMrn,
				EntityMrn:  AssetMrn,
				Score: &policy.Score{
					Value:           29,
					ScoreCompletion: 100,
					DataCompletion:  100,
				},
				// add passed, failed and skipped test
				Scores: map[string]*policy.Score{
					"+u6doYoYG5E=": {
						Type:  2, // result
						Value: 100,
					},
					"057itYF8s30=": {
						Type:  4, // error
						Value: 0,
					},
					"GyJVAziB/tU=": {
						Type:  8, // skip
						Value: 0,
					},
				},
			},
		},
		VulnReports: map[string]*mvd.VulnReport{
			AssetMrn: {
				Packages: []*mvd.Package{
					{
						Name:      "libssl1.1",
						Version:   "1.1.1f-3ubuntu2.19",
						Affected:  true,
						Score:     100,
						Available: "1.1.1f-3ubuntu2.20",
					},
				},
				Stats: &mvd.ReportStats{
					Packages: &mvd.ReportStatsPackages{
						Total:    1,
						Critical: 1,
						Affected: 1,
					},
				},
			},
		},
	}
}

// Detailed is a scan of one Terraform asset with a single failing check that
// carries a description, MQL, platform-specific remediation and a reference.
// Where Sample covers the outcomes, this one covers the check documentation: it
// is what the reporters are compared on for how they render a finding.
//
// It has no ExecutionJob, so a converter's assessment path is deliberately not
// reached from here.
func Detailed() *policy.ReportCollection {
	const assetMrn = "//assets.api.mondoo.app/spaces/test/assets/abc"
	const codeID = "abc123=="
	return &policy.ReportCollection{
		Assets: map[string]*inventory.Asset{
			assetMrn: {
				Name:     "X1",
				Platform: &inventory.Platform{Name: "terraform-hcl", Family: []string{"terraform"}},
			},
		},
		ResolvedPolicies: map[string]*policy.ResolvedPolicy{
			assetMrn: {
				CollectorJob: &policy.CollectorJob{
					ReportingQueries: map[string]*policy.StringArray{
						codeID: nil,
					},
				},
			},
		},
		Bundle: &policy.Bundle{
			Queries: []*policy.Mquery{
				{
					Mrn:    "//policy.api.mondoo.app/queries/test-check",
					CodeId: codeID,
					Title:  "Ensure the thing is configured",
					Mql:    "sshd.config.params['PermitRootLogin'] == \"no\"",
					Docs: &policy.MqueryDocs{
						Desc: "Root login over SSH should be disabled.",
						Remediation: &policy.Remediation{
							Items: []*policy.TypedDoc{
								{Id: "console", Desc: "Use the AWS console to fix it."},
								{Id: "terraform", Desc: "Set PermitRootLogin to no in your TF config."},
								{Id: "cloudformation", Desc: "Use CloudFormation to fix it."},
							},
						},
						Refs: []*policy.MqueryRef{
							{Title: "CIS Benchmark", Url: "https://example.com/cis"},
						},
					},
				},
			},
		},
		Reports: map[string]*policy.Report{
			assetMrn: {
				ScoringMrn: assetMrn,
				EntityMrn:  assetMrn,
				Scores: map[string]*policy.Score{
					codeID: {Type: policy.ScoreType_Result, Value: 0},
				},
			},
		},
	}
}
