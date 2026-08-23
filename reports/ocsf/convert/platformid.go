// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Reading a cnspec platform id. Nothing here is OCSF: these are cnspec's own
// identifiers, and the OCSF attributes they end up in (cloud.region,
// cloud.account.uid, resource.cloud_partition) are filled in by asset.go.

package convert

import (
	"strings"

	"go.mondoo.com/mql/providers-sdk/v1/inventory"
)

type parsedARN struct {
	partition string
	region    string
	account   string
}

// awsARN pulls the partition, region and account out of the first ARN among an
// asset's platform ids. An ARN is
// arn:<partition>:<service>:<region>:<account>:<resource>, and region and account
// are empty for global and account-less resources.
func awsARN(asset *inventory.Asset) (parsedARN, bool) {
	for _, id := range asset.PlatformIds {
		if !strings.HasPrefix(id, "arn:") {
			continue
		}
		parts := strings.SplitN(id, ":", 6)
		if len(parts) < 5 {
			continue
		}
		return parsedARN{partition: parts[1], region: parts[3], account: parts[4]}, true
	}
	return parsedARN{}, false
}

// platformIDSegment returns the segment that follows a marker in a platform id,
// e.g. the project of //platformid.api.mondoo.app/runtime/gcp/projects/my-project.
func platformIDSegment(id, marker string) (string, bool) {
	idx := strings.Index(id, marker)
	if idx < 0 {
		return "", false
	}
	rest := id[idx+len(marker):]
	if end := strings.Index(rest, "/"); end >= 0 {
		rest = rest[:end]
	}
	if rest == "" {
		return "", false
	}
	return rest, true
}
