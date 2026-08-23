// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reportdoc

import "go.mondoo.com/mql/providers-sdk/v1/inventory"

func PlatformName(asset *inventory.Asset) string {
	platformName := ""
	if asset.Platform != nil {
		if asset.Platform.Title == "" {
			platformName = asset.Platform.Name
		} else {
			platformName = asset.Platform.Title
		}
	}
	return platformName
}
