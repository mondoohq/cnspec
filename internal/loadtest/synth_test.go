// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package loadtest

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/v13/policy"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
)

func newTemplate() *Template {
	return &Template{
		Path: "test.db",
		Asset: &inventory.Asset{
			Mrn:         "//assets.api.mondoo.com/spaces/x/assets/original",
			Id:          "original",
			Name:        "ubuntu-host",
			PlatformIds: []string{"//platformid.api.mondoo.app/runtime/aws/ec2/v1/accounts/123/regions/us-east-1/instances/i-abc"},
			Platform:    &inventory.Platform{Name: "ubuntu", Version: "22.04"},
		},
		Scores: []*policy.Score{
			{QrId: "q1", Value: 100, Type: 1, Weight: 1},
			{QrId: "q2", Value: 0, Type: 1, Weight: 1},
			{QrId: "q3", Value: 100, Type: 1, Weight: 1},
			{QrId: "q4", Value: 0, Type: 1, Weight: 1},
		},
	}
}

func TestSynthesizeAssetDeterministic(t *testing.T) {
	tpl := newTemplate()
	a1 := SynthesizeAsset(tpl, 7, 42)
	a2 := SynthesizeAsset(tpl, 7, 42)
	require.Equal(t, a1.PlatformIds, a2.PlatformIds, "same seed+idx must yield identical platform_ids")
	require.Equal(t, a1.Name, a2.Name)
}

func TestSynthesizeAssetUniquePerIdx(t *testing.T) {
	tpl := newTemplate()
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		a := SynthesizeAsset(tpl, i, 42)
		require.NotEmpty(t, a.PlatformIds)
		require.False(t, seen[a.PlatformIds[0]], "platform_id collision at idx %d", i)
		seen[a.PlatformIds[0]] = true
	}
}

func TestSynthesizeAssetClearsMrn(t *testing.T) {
	tpl := newTemplate()
	a := SynthesizeAsset(tpl, 0, 0)
	require.Empty(t, a.Mrn, "synthesized asset must have empty MRN so SynchronizeAssets can assign one")
}

func TestSynthesizeAssetPreservesPlatform(t *testing.T) {
	tpl := newTemplate()
	a := SynthesizeAsset(tpl, 5, 0)
	require.NotNil(t, a.Platform)
	require.Equal(t, "ubuntu", a.Platform.Name)
	require.Equal(t, "22.04", a.Platform.Version)
}

func TestSynthesizeAssetDoesNotMutateTemplate(t *testing.T) {
	tpl := newTemplate()
	originalPID := tpl.Asset.PlatformIds[0]
	originalName := tpl.Asset.Name
	_ = SynthesizeAsset(tpl, 0, 0)
	require.Equal(t, originalPID, tpl.Asset.PlatformIds[0])
	require.Equal(t, originalName, tpl.Asset.Name)
}
