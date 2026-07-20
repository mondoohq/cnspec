// Copyright Mondoo, Inc. 2026
// SPDX-License-Identifier: BUSL-1.1

package scanstats

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/v13/policy"
	"go.mondoo.com/mql/v13/llx"
)

func TestCountByKind(t *testing.T) {
	rp := &policy.ResolvedPolicy{
		CollectorJob: &policy.CollectorJob{
			ReportingJobs: map[string]*policy.ReportingJob{
				"u0": {QrId: "root", Type: policy.ReportingJob_POLICY}, // ignored
				"u1": {QrId: "//q/check1", Type: policy.ReportingJob_CHECK},
				"u2": {QrId: "//q/check2", Type: policy.ReportingJob_CHECK},
				"u3": {QrId: "//q/data1", Type: policy.ReportingJob_DATA_QUERY},
				"u4": {QrId: "//p/pol1", Type: policy.ReportingJob_POLICY},
				"u5": {QrId: "//c/ctrl1", Type: policy.ReportingJob_CONTROL},
				"u6": {QrId: "//f/fw1", Type: policy.ReportingJob_FRAMEWORK},
				"u7": {QrId: "//r/risk1", Type: policy.ReportingJob_RISK_FACTOR}, // ignored
			},
		},
	}
	scores := []*policy.Score{
		{QrId: "//q/check1", Type: policy.ScoreType_Error},  // errored check
		{QrId: "//q/check2", Type: policy.ScoreType_Result}, // passing check
	}
	data := []*llx.Result{
		{CodeId: "d1"},
		{CodeId: "d2", Error: "boom"}, // errored data query
	}

	c := CountByKind(rp, scores, data)
	require.Equal(t, int64(2), c.Checks)
	require.Equal(t, int64(1), c.DataQueries)
	require.Equal(t, int64(1), c.Policies)
	require.Equal(t, int64(1), c.Controls)
	require.Equal(t, int64(1), c.Frameworks)
	require.Equal(t, int64(1), c.ChecksErrored)
	require.Equal(t, int64(1), c.DataQueriesErrored)
}

func TestCountByKind_NilSafe(t *testing.T) {
	require.Equal(t, KindCounts{}, CountByKind(nil, nil, nil))
	require.Equal(t, KindCounts{}, CountByKind(&policy.ResolvedPolicy{}, nil, nil))
}
