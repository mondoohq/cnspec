// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// activityRecorder is an upstream that only records ReportAssetActivityStarted.
type activityRecorder struct {
	PolicyResolver
	got []*ReportAssetActivityStartedReq
}

func (r *activityRecorder) ReportAssetActivityStarted(_ context.Context, req *ReportAssetActivityStartedReq) (*ReportAssetActivityStartedResp, error) {
	r.got = append(r.got, req)
	return &ReportAssetActivityStartedResp{Updated: int32(len(req.AssetMrns))}, nil
}

func TestReportAssetActivityStartedGoesUpstream(t *testing.T) {
	upstream := &activityRecorder{}
	s := &LocalServices{Upstream: &Services{PolicyResolver: upstream}}

	req := &ReportAssetActivityStartedReq{
		SpaceMrn:  "//spaces/s",
		AssetMrns: []string{"//assets/a"},
		Kind:      AssetActivityKind_ASSET_ACTIVITY_KIND_AGENT,
	}
	resp, err := s.ReportAssetActivityStarted(context.Background(), req)
	require.NoError(t, err)
	require.Len(t, upstream.got, 1)
	assert.Same(t, req, upstream.got[0])
	assert.EqualValues(t, 1, resp.Updated)
}

func TestReportAssetActivityStartedStaysLocalWhenIncognito(t *testing.T) {
	upstream := &activityRecorder{}
	s := &LocalServices{Upstream: &Services{PolicyResolver: upstream}, Incognito: true}

	resp, err := s.ReportAssetActivityStarted(context.Background(), &ReportAssetActivityStartedReq{AssetMrns: []string{"//assets/a"}})
	require.NoError(t, err)
	assert.Empty(t, upstream.got)
	assert.Zero(t, resp.Updated)

	// Without an upstream there is nobody to tell, and that is not an error.
	_, err = (&LocalServices{}).ReportAssetActivityStarted(context.Background(), &ReportAssetActivityStartedReq{})
	assert.NoError(t, err)
}
