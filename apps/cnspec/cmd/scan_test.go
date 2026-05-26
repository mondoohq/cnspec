// Copyright Mondoo, Inc. 2026
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
)

func TestApplyAutoDiscoveredInventory_MergesIdDetectorForMatchingConnection(t *testing.T) {
	target := &inventory.Asset{
		Name:        "local-cli",
		Connections: []*inventory.Config{{Type: "local"}},
	}
	inv := &inventory.Inventory{Spec: &inventory.InventorySpec{Assets: []*inventory.Asset{
		{
			Name:        "local-scan",
			Connections: []*inventory.Config{{Type: "local"}},
			IdDetector:  []string{"hostname", "machine-id"},
		},
	}}}

	applyAutoDiscoveredInventory(target, inv)

	assert.Equal(t, []string{"hostname", "machine-id"}, target.IdDetector,
		"id_detector should be lifted from the matching inventory asset onto the CLI target")
	assert.Equal(t, []*inventory.Asset{target}, inv.Spec.Assets,
		"inventory should be narrowed to the CLI target so non-CLI assets aren't scanned")
}

func TestApplyAutoDiscoveredInventory_SkipsMergeWhenConnectionTypesDiffer(t *testing.T) {
	target := &inventory.Asset{
		Name:        "aws-cli",
		Connections: []*inventory.Config{{Type: "aws"}},
	}
	inv := &inventory.Inventory{Spec: &inventory.InventorySpec{Assets: []*inventory.Asset{
		{
			Name:        "local-scan",
			Connections: []*inventory.Config{{Type: "local"}},
			IdDetector:  []string{"hostname", "machine-id"},
		},
	}}}

	applyAutoDiscoveredInventory(target, inv)

	assert.Empty(t, target.IdDetector,
		"`cnspec scan aws ...` must not inherit id_detector from a sibling local-scan inventory")
	assert.Equal(t, []*inventory.Asset{target}, inv.Spec.Assets,
		"sibling local-scan asset must be dropped so it does not replace the AWS target")
}

func TestApplyAutoDiscoveredInventory_PreservesExplicitTargetIdDetector(t *testing.T) {
	target := &inventory.Asset{
		Name:        "local-cli",
		Connections: []*inventory.Config{{Type: "local"}},
		IdDetector:  []string{"hostname"},
	}
	inv := &inventory.Inventory{Spec: &inventory.InventorySpec{Assets: []*inventory.Asset{
		{
			Name:        "local-scan",
			Connections: []*inventory.Config{{Type: "local"}},
			IdDetector:  []string{"machine-id"},
		},
	}}}

	applyAutoDiscoveredInventory(target, inv)

	assert.Equal(t, []string{"hostname"}, target.IdDetector,
		"an id_detector list already on the CLI target wins over the auto-discovered inventory")
}

func TestApplyAutoDiscoveredInventory_HandlesEmptyInventory(t *testing.T) {
	target := &inventory.Asset{
		Name:        "local-cli",
		Connections: []*inventory.Config{{Type: "local"}},
	}
	inv := &inventory.Inventory{Spec: &inventory.InventorySpec{}}

	applyAutoDiscoveredInventory(target, inv)

	assert.Empty(t, target.IdDetector)
	assert.Equal(t, []*inventory.Asset{target}, inv.Spec.Assets)
}

func TestApplyAutoDiscoveredInventory_NilSafe(t *testing.T) {
	// Should not panic on any nil argument.
	applyAutoDiscoveredInventory(nil, nil)
	applyAutoDiscoveredInventory(&inventory.Asset{}, nil)
	applyAutoDiscoveredInventory(nil, &inventory.Inventory{Spec: &inventory.InventorySpec{}})
	applyAutoDiscoveredInventory(&inventory.Asset{}, &inventory.Inventory{})
}

func TestApplyAutoDiscoveredInventory_OnlyFirstMatchingAssetIsMergedIntoTarget(t *testing.T) {
	// Locks in the documented first-match-wins semantics on the break:
	// when multiple inventory assets share the CLI target's connection
	// type, only the first one's id_detector is lifted. If this behavior
	// ever changes (e.g. to merge across matches), the test should be
	// updated together with the comment in scan.go.
	target := &inventory.Asset{
		Name:        "local-cli",
		Connections: []*inventory.Config{{Type: "local"}},
	}
	inv := &inventory.Inventory{Spec: &inventory.InventorySpec{Assets: []*inventory.Asset{
		{
			Name:        "first-local",
			Connections: []*inventory.Config{{Type: "local"}},
			IdDetector:  []string{"hostname"},
		},
		{
			Name:        "second-local",
			Connections: []*inventory.Config{{Type: "local"}},
			IdDetector:  []string{"machine-id", "bios-uuid"},
		},
	}}}

	applyAutoDiscoveredInventory(target, inv)

	assert.Equal(t, []string{"hostname"}, target.IdDetector,
		"only the first matching inventory asset's id_detector should be lifted")
	assert.Equal(t, []*inventory.Asset{target}, inv.Spec.Assets,
		"second-local and all other inventory assets should be discarded")
}
