// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package loadtest

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"

	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"google.golang.org/protobuf/proto"
)

// SynthesizeAsset deep-clones the template asset and overwrites identifiers so
// the resulting asset is unique in the target space. The clone is keyed by
// (seed, assetIdx, originalPlatformId) — same inputs produce byte-identical
// platform_ids on every run, making the load test repeatable across processes
// and shards.
//
// MRN is cleared because SynchronizeAssets assigns it; everything else
// (Platform, Connections, Labels, Annotations, Options) is preserved from the
// template so the server applies the same policies it would for a real asset
// of this shape.
func SynthesizeAsset(template *Template, assetIdx int, seed int64) *inventory.Asset {
	clone := proto.Clone(template.Asset).(*inventory.Asset)

	clone.Mrn = ""
	clone.Id = ""

	if len(template.Asset.PlatformIds) == 0 {
		clone.PlatformIds = []string{deterministicPlatformID(seed, assetIdx, "")}
	} else {
		clone.PlatformIds = make([]string, len(template.Asset.PlatformIds))
		for i, orig := range template.Asset.PlatformIds {
			clone.PlatformIds[i] = deterministicPlatformID(seed, assetIdx, orig)
		}
	}

	clone.Name = fmt.Sprintf("loadtest-%d-%s", assetIdx, truncateName(template.Asset.Name, 32))
	return clone
}

func deterministicPlatformID(seed int64, assetIdx int, originalPlatformID string) string {
	var buf [16]byte
	binary.BigEndian.PutUint64(buf[0:8], uint64(seed))
	binary.BigEndian.PutUint64(buf[8:16], uint64(assetIdx))
	h := sha256.New()
	h.Write(buf[:])
	h.Write([]byte(originalPlatformID))
	sum := h.Sum(nil)
	return "//platformid.api.mondoo.app/loadtest/" + hex.EncodeToString(sum[:16])
}

func truncateName(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
