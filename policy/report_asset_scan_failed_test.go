// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// failedScanRecorder is an upstream that only records ReportAssetScanFailed.
type failedScanRecorder struct {
	PolicyResolver
	got []*ReportAssetScanFailedReq
}

func (r *failedScanRecorder) ReportAssetScanFailed(_ context.Context, req *ReportAssetScanFailedReq) (*Empty, error) {
	r.got = append(r.got, req)
	return &Empty{}, nil
}

func TestReportAssetScanFailedGoesUpstream(t *testing.T) {
	upstream := &failedScanRecorder{}
	s := &LocalServices{Upstream: &Services{PolicyResolver: upstream}}

	req := &ReportAssetScanFailedReq{AssetMrn: "//assets/a", Error: "boom"}
	_, err := s.ReportAssetScanFailed(context.Background(), req)
	require.NoError(t, err)
	require.Len(t, upstream.got, 1)
	assert.Same(t, req, upstream.got[0])
}

func TestReportAssetScanFailedStaysLocalWhenIncognito(t *testing.T) {
	upstream := &failedScanRecorder{}
	s := &LocalServices{Upstream: &Services{PolicyResolver: upstream}, Incognito: true}

	_, err := s.ReportAssetScanFailed(context.Background(), &ReportAssetScanFailedReq{AssetMrn: "//assets/a"})
	require.NoError(t, err)
	assert.Empty(t, upstream.got)

	// Without an upstream there is nobody to tell, and that is not an error.
	_, err = (&LocalServices{}).ReportAssetScanFailed(context.Background(), &ReportAssetScanFailedReq{AssetMrn: "//assets/a"})
	assert.NoError(t, err)
}
