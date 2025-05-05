// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"fmt"
	"os"

	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnspec/v11/cli/components"
	"go.mondoo.com/cnspec/v11/policy"
)

func main() {
	assetsByPlatform := map[string][]*inventory.Asset{
		"Ubuntu 16.04.7 LTS": []*inventory.Asset{
			{Name: "ubuntu-1"}, {Name: "ubuntu-2"}, {Name: "ubuntu-3"},
		},
		"Alpine Linux v3.21": []*inventory.Asset{{Name: "alpine-1"}},
		"macOS":              []*inventory.Asset{{Name: "mac-1"}},
	}
	assetsByScore := map[string]int{
		policy.ScoreRatingTextCritical: 0,
		policy.ScoreRatingTextHigh:     1,
		policy.ScoreRatingTextMedium:   0,
		policy.ScoreRatingTextLow:      4,
	}
	model := components.NewDistributions(assetsByScore, assetsByPlatform)
	fmt.Fprint(os.Stdout, model.View())
}
