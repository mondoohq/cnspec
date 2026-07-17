// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

func newReqWithStats(t *testing.T) *ReportUploadCompletedReq {
	t.Helper()
	stats := &ScanStatistics{Metrics: []*Metric{
		{Name: "cnspec.scan.duration", Unit: "ms", Value: &Metric_IntValue{IntValue: 4200}},
		{Name: "cnspec.scan.queries_executed", Unit: "count", Value: &Metric_IntValue{IntValue: 128}},
	}}
	details, err := anypb.New(stats)
	require.NoError(t, err)
	return &ReportUploadCompletedReq{
		UploadSessionId: "session-1",
		ScopeMrn:        "//assets/1",
		Details:         details,
	}
}

func requireStatsRoundTrip(t *testing.T, got *ReportUploadCompletedReq) {
	t.Helper()
	require.Equal(t, "session-1", got.UploadSessionId)
	require.NotNil(t, got.Details)
	var stats ScanStatistics
	require.NoError(t, got.Details.UnmarshalTo(&stats))
	require.Len(t, stats.Metrics, 2)
	require.Equal(t, "cnspec.scan.duration", stats.Metrics[0].Name)
	require.Equal(t, int64(4200), stats.Metrics[0].GetIntValue())
}

func TestReportUploadCompletedReq_AnyRoundTrip_Proto(t *testing.T) {
	req := newReqWithStats(t)
	raw, err := proto.Marshal(req)
	require.NoError(t, err)
	var got ReportUploadCompletedReq
	require.NoError(t, proto.Unmarshal(raw, &got))
	requireStatsRoundTrip(t, &got)
}

func TestReportUploadCompletedReq_AnyRoundTrip_VT(t *testing.T) {
	req := newReqWithStats(t)
	raw, err := req.MarshalVT()
	require.NoError(t, err)
	var got ReportUploadCompletedReq
	require.NoError(t, got.UnmarshalVT(raw))
	requireStatsRoundTrip(t, &got)
}
