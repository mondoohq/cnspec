// Copyright Mondoo, Inc. 2026
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
)

// Local and incognito scans never claim: the stub answers with no verdicts,
// and a missing verdict means SCAN.
func TestLocalServicesClaimAssetsIsAllScan(t *testing.T) {
	s := &LocalServices{}

	resp, err := s.ClaimAssets(context.Background(), &ClaimAssetsReq{
		SpaceMrn: "//captain.api.mondoo.app/spaces/s",
		List:     []*inventory.Asset{{Name: "a"}, {Name: "b"}},
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Empty(t, resp.Verdicts)
}

// The claim lane relies on the Claims field added to inventory.Asset (mql bump).
func TestInventoryAssetHasClaims(t *testing.T) {
	assert.Empty(t, (&inventory.Asset{}).GetClaims())
}

// SCAN is the zero value so an absent or unknown verdict fails open.
func TestClaimAssetVerdictDefaultsToScan(t *testing.T) {
	assert.Equal(t, ClaimAssetVerdict_SCAN, (&ClaimAssetVerdict{}).GetAction())
	assert.False(t, (&ScanParameters{}).GetClaimAssetsEnabled())
}
