// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"encoding/json"
	"os"
	"testing"

	"go.mondoo.com/cnspec/policy"
)

// A scan where every asset failed to connect never resolves a policy, so the
// collection arrives with no bundle at all. That is the case stats are most
// wanted for -- it separates "nothing ran" from "nothing was found" -- so it is
// the one case that must not panic.
func TestGenerateStatsWithoutABundle(t *testing.T) {
	raw, err := os.ReadFile("testdata/report-k8s.json")
	if err != nil {
		t.Fatal(err)
	}
	var collection policy.ReportCollection
	if err := json.Unmarshal(raw, &collection); err != nil {
		t.Fatal(err)
	}
	if collection.Bundle != nil {
		t.Fatal("fixture is meant to have no bundle; it now has one")
	}

	stats := GenerateStats(&collection)

	if len(stats.AssetScores) != len(collection.Assets) {
		t.Errorf("scored %d assets, want %d", len(stats.AssetScores), len(collection.Assets))
	}
	for mrn, score := range stats.AssetScores {
		if score.Type != policy.ScoreType_Error {
			t.Errorf("%s: type %d, want error -- an asset that could not be scanned is not a clean one", mrn, score.Type)
		}
	}
}
