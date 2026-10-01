// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/mql/discovery"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
)

// activityRecorder is a resolver that syncs every asset it is given and
// records ReportAssetActivityStarted.
type activityRecorder struct {
	policy.PolicyResolver
	got chan *policy.ReportAssetActivityStartedReq
	err error
}

func newActivityRecorder() *activityRecorder {
	return &activityRecorder{got: make(chan *policy.ReportAssetActivityStartedReq, 10)}
}

func (r *activityRecorder) SynchronizeAssets(_ context.Context, req *policy.SynchronizeAssetsReq) (*policy.SynchronizeAssetsResp, error) {
	resp := &policy.SynchronizeAssetsResp{Details: map[string]*policy.SynchronizeAssetsRespAssetDetail{}}
	for _, a := range req.List {
		resp.Details[a.Name] = &policy.SynchronizeAssetsRespAssetDetail{
			PlatformMrn: "//platformid/" + a.Name,
			AssetMrn:    req.SpaceMrn + "/assets/" + a.Name,
		}
	}
	return resp, nil
}

func (r *activityRecorder) ReportAssetActivityStarted(_ context.Context, req *policy.ReportAssetActivityStartedReq) (*policy.ReportAssetActivityStartedResp, error) {
	r.got <- req
	return &policy.ReportAssetActivityStartedResp{Updated: int32(len(req.AssetMrns))}, r.err
}

func (r *activityRecorder) next(t *testing.T) *policy.ReportAssetActivityStartedReq {
	t.Helper()
	select {
	case req := <-r.got:
		return req
	case <-time.After(5 * time.Second):
		t.Fatal("no ReportAssetActivityStarted call")
		return nil
	}
}

func TestReportAssetActivityStartedSendsAnAgentStart(t *testing.T) {
	r := newActivityRecorder()
	reportAssetActivityStarted(context.Background(), r, "//spaces/s", []string{"//spaces/s/assets/a"},
		policy.AssetActivityTrigger_ASSET_ACTIVITY_TRIGGER_SCHEDULED)

	req := r.next(t)
	assert.Equal(t, "//spaces/s", req.SpaceMrn)
	assert.Equal(t, []string{"//spaces/s/assets/a"}, req.AssetMrns)
	assert.Equal(t, policy.AssetActivityKind_ASSET_ACTIVITY_KIND_AGENT, req.Kind)
	assert.Equal(t, policy.AssetActivityTrigger_ASSET_ACTIVITY_TRIGGER_SCHEDULED, req.Trigger)

	// Nothing synced, nothing to report.
	reportAssetActivityStarted(context.Background(), r, "//spaces/s", nil, 0)
	assert.Empty(t, r.got)

	// A server that predates the RPC: nothing to do, and nothing panics.
	r.err = errors.New("not found")
	reportAssetActivityStarted(context.Background(), r, "//spaces/s", []string{"//spaces/s/assets/a"}, 0)
	r.next(t)
}

// The start report carries exactly the MRNs the sync handed back, with the
// scanner's trigger.
func TestSyncBatchReportsTheSyncedAssets(t *testing.T) {
	r := newActivityRecorder()
	services := &policy.Services{PolicyResolver: r}
	batch := []*discovery.TrackedAsset{
		{Asset: &inventory.Asset{Name: "a"}},
		{Asset: &inventory.Asset{Name: "b"}},
	}

	err := syncBatchWithUpstream(context.Background(), batch, services, "//spaces/s", nil,
		policy.AssetActivityTrigger_ASSET_ACTIVITY_TRIGGER_AD_HOC)
	require.NoError(t, err)

	req := r.next(t)
	assert.ElementsMatch(t, []string{"//spaces/s/assets/a", "//spaces/s/assets/b"}, req.AssetMrns)
	assert.Equal(t, policy.AssetActivityTrigger_ASSET_ACTIVITY_TRIGGER_AD_HOC, req.Trigger)
}

func TestWithActivityTriggerLastOptionWins(t *testing.T) {
	s := NewLocalScanner(
		WithActivityTrigger(policy.AssetActivityTrigger_ASSET_ACTIVITY_TRIGGER_AD_HOC),
		WithActivityTrigger(policy.AssetActivityTrigger_ASSET_ACTIVITY_TRIGGER_SCHEDULED),
	)
	assert.Equal(t, policy.AssetActivityTrigger_ASSET_ACTIVITY_TRIGGER_SCHEDULED, s.activityTrigger)
}
