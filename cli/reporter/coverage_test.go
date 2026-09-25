// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/mql/llx"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
)

func forbidden(perms ...string) *llx.ErrorDetail {
	return &llx.ErrorDetail{Kind: llx.ErrorKind_ERROR_KIND_FORBIDDEN, Permissions: perms}
}

// coverageCollection is one asset with the given check scores, keyed by check.
func coverageCollection(scores map[string]*policy.Score) *policy.ReportCollection {
	jobs := map[string]*policy.ReportingJob{}
	for mrn := range scores {
		jobs[mrn] = &policy.ReportingJob{QrId: mrn, Type: policy.ReportingJob_CHECK}
	}
	// A data query's score is not a check and is never counted.
	jobs["data"] = &policy.ReportingJob{QrId: "data", Type: policy.ReportingJob_DATA_QUERY}
	scores["data"] = &policy.Score{Type: policy.ScoreType_Error, ErrorDetails: []*llx.ErrorDetail{forbidden("s3:ListBuckets")}}

	return &policy.ReportCollection{
		Assets:           map[string]*inventory.Asset{"//a1": {Mrn: "//a1"}},
		Reports:          map[string]*policy.Report{"//a1": {Scores: scores}},
		ResolvedPolicies: map[string]*policy.ResolvedPolicy{"//a1": {CollectorJob: &policy.CollectorJob{ReportingJobs: jobs}}},
	}
}

func TestCoverageReport(t *testing.T) {
	data := coverageCollection(map[string]*policy.Score{
		"pass":  {Type: policy.ScoreType_Result, Value: 100},
		"fail":  {Type: policy.ScoreType_Result, Value: 0},
		"skip":  {Type: policy.ScoreType_Skip},
		"deny1": {Type: policy.ScoreType_Error, ErrorDetails: []*llx.ErrorDetail{forbidden("ec2:DescribeInstances"), forbidden("iam:ListUsers")}},
		"deny2": {Type: policy.ScoreType_Error, ErrorDetails: []*llx.ErrorDetail{forbidden("ec2:DescribeInstances"), {Kind: llx.ErrorKind_ERROR_KIND_TOO_MANY_REQUESTS}}},
		"na":    {Type: policy.ScoreType_Error, ErrorDetails: []*llx.ErrorDetail{{Kind: llx.ErrorKind_ERROR_KIND_NOT_APPLICABLE}}},
		"boom":  {Type: policy.ScoreType_Error},
		"partial": {Type: policy.ScoreType_Result, Value: 100, ErrorDetails: []*llx.ErrorDetail{{
			Kind: llx.ErrorKind_ERROR_KIND_FORBIDDEN, Scope: llx.ErrorScope_ERROR_SCOPE_PARTITION, ScopeId: "eu-west-1",
			Permissions: []string{"compute.instances.list"},
		}}},
	})
	data.Assets["//a2"] = &inventory.Asset{Mrn: "//a2"}
	data.Assets["//a3"] = &inventory.Asset{Mrn: "//a3"}
	data.Errors = map[string]string{"//a2": "expired", "//a3": "boom"}
	data.ErrorDetails = map[string]*llx.ErrorDetail{"//a2": {Kind: llx.ErrorKind_ERROR_KIND_UNAUTHENTICATED}}

	assert.Equal(t, []string{
		// deny2 counts once for each of its kinds; a check without a
		// classified kind is unclassified.
		"4 of 8 checks could not be assessed: 2 access denied, 1 not applicable, 1 too many requests, 1 unclassified.",
		"1 check was assessed on incomplete data: 1 access denied.",
		"2 of 3 assets could not be scanned: 1 unauthenticated, 1 unclassified.",
		"Missing permissions (3 across compute, ec2, iam): compute.instances.list, ec2:DescribeInstances, iam:ListUsers",
	}, newCoverageReport(data).lines())
}

func TestCoverageReportSaysNothingWithoutAClassifiedError(t *testing.T) {
	// Unclassified errors only, which is every scan with the flag off: the
	// output stays what it was.
	data := coverageCollection(map[string]*policy.Score{
		"pass": {Type: policy.ScoreType_Result, Value: 100},
		"boom": {Type: policy.ScoreType_Error},
	})
	data.Errors = map[string]string{"//a1": "boom"}
	assert.Nil(t, newCoverageReport(data).lines())
}

func TestCoverageGapLines(t *testing.T) {
	lines := coverageGapLines([]*llx.ErrorDetail{
		{Kind: llx.ErrorKind_ERROR_KIND_FORBIDDEN, ScopeId: "us-east-1", Permissions: []string{"ec2:DescribeVpcs"}},
		{Kind: llx.ErrorKind_ERROR_KIND_FORBIDDEN, ScopeId: "eu-west-1", Permissions: []string{"ec2:DescribeVpcs"}},
		// Same kind and partition: one line, permissions merged.
		{Kind: llx.ErrorKind_ERROR_KIND_FORBIDDEN, ScopeId: "eu-west-1", Permissions: []string{"ec2:DescribeAddresses"}},
		{Kind: llx.ErrorKind_ERROR_KIND_TOO_MANY_REQUESTS, ScopeId: "eu-west-1"},
	})
	assert.Equal(t, []string{
		"coverage gap: access denied in eu-west-1 (ec2:DescribeAddresses, ec2:DescribeVpcs)",
		"coverage gap: access denied in us-east-1 (ec2:DescribeVpcs)",
		"coverage gap: too many requests in eu-west-1",
	}, lines)
	assert.Empty(t, coverageGapLines(nil))
}

func TestPermissionService(t *testing.T) {
	assert.Equal(t, "ec2", permissionService("ec2:DescribeInstances"))
	assert.Equal(t, "compute", permissionService("compute.instances.list"))
	assert.Equal(t, "sudo", permissionService("sudo"))
}
