// Copyright Mondoo, Inc. 2026
// SPDX-License-Identifier: BUSL-1.1

package scanstats

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/v13/policy"
)

func TestNewPolicyKinds_Counts(t *testing.T) {
	rp := &policy.ResolvedPolicy{
		CollectorJob: &policy.CollectorJob{
			ReportingJobs: map[string]*policy.ReportingJob{
				"u0": {QrId: "root", Type: policy.ReportingJob_POLICY}, // ignored
				"u1": {QrId: "//q/check1", Type: policy.ReportingJob_CHECK},
				"u2": {QrId: "//q/cadq1", Type: policy.ReportingJob_CHECK_AND_DATA_QUERY},
				"u3": {QrId: "//q/data1", Type: policy.ReportingJob_DATA_QUERY},
				"u4": {QrId: "//p/pol1", Type: policy.ReportingJob_POLICY},
				"u5": {QrId: "//c/ctrl1", Type: policy.ReportingJob_CONTROL},
				"u6": {QrId: "//f/fw1", Type: policy.ReportingJob_FRAMEWORK},
				"u7": {QrId: "//r/risk1", Type: policy.ReportingJob_RISK_FACTOR}, // ignored
			},
		},
	}

	pk := NewPolicyKinds(rp)
	require.Equal(t, int64(2), pk.Counts.Checks) // CHECK + CHECK_AND_DATA_QUERY
	require.Equal(t, int64(1), pk.Counts.DataQueries)
	require.Equal(t, int64(1), pk.Counts.Policies)
	require.Equal(t, int64(1), pk.Counts.Controls)
	require.Equal(t, int64(1), pk.Counts.Frameworks)
	// errored fields are filled by the caller, not NewPolicyKinds
	require.Equal(t, int64(0), pk.Counts.ChecksErrored)
	require.Equal(t, int64(0), pk.Counts.DataQueriesErrored)
}

func TestNewPolicyKinds_IsCheckQrId(t *testing.T) {
	rp := &policy.ResolvedPolicy{
		CollectorJob: &policy.CollectorJob{
			ReportingJobs: map[string]*policy.ReportingJob{
				"u1": {QrId: "//q/check1", Type: policy.ReportingJob_CHECK},
				"u2": {QrId: "//q/cadq1", Type: policy.ReportingJob_CHECK_AND_DATA_QUERY},
				"u3": {QrId: "//q/data1", Type: policy.ReportingJob_DATA_QUERY},
				"u4": {QrId: "//p/pol1", Type: policy.ReportingJob_POLICY},
			},
		},
	}

	pk := NewPolicyKinds(rp)

	// CHECK and CHECK_AND_DATA_QUERY qr_ids are recognized as checks
	require.True(t, pk.IsCheckQrId("//q/check1"))
	require.True(t, pk.IsCheckQrId("//q/cadq1"))

	// data query, policy, root, and unknown qr_ids are not checks
	require.False(t, pk.IsCheckQrId("//q/data1"))
	require.False(t, pk.IsCheckQrId("//p/pol1"))
	require.False(t, pk.IsCheckQrId("root"))
	require.False(t, pk.IsCheckQrId("//unknown/qr"))
}

func TestNewPolicyKinds_NilSafe(t *testing.T) {
	pk := NewPolicyKinds(nil)
	require.NotNil(t, pk)
	require.Equal(t, KindCounts{}, pk.Counts)
	require.False(t, pk.IsCheckQrId("anything"))

	pk2 := NewPolicyKinds(&policy.ResolvedPolicy{})
	require.NotNil(t, pk2)
	require.Equal(t, KindCounts{}, pk2.Counts)
	require.False(t, pk2.IsCheckQrId("anything"))
}
