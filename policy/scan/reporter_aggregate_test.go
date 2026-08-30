// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
)

func TestAggregateReport(t *testing.T) {
	b := &policy.Bundle{
		Policies: []*policy.Policy{
			{
				Uid:  "policy1",
				Name: "Policy 1",
			},
		},
	}

	r := NewAggregateReporter()
	r.AddBundle(b)
	assert.Equal(t, r.bundle, b)

	b2 := &policy.Bundle{
		Policies: []*policy.Policy{
			{
				Uid:  "policy2",
				Name: "Policy 2",
			},
		},
	}

	r.AddBundle(b2)
	assert.Equal(t, r.bundle, policy.Merge(b, b2))
}

func TestAggregateReporterAssetSnapshot(t *testing.T) {
	r := NewAggregateReporter()
	asset := &inventory.Asset{
		Mrn:         "//assets.api.mondoo.app/assets/abc",
		Name:        "target",
		PlatformIds: []string{"//platformid.api.mondoo.app/runtime/linux/host/abc"},
		Platform: &inventory.Platform{
			Name:    "linux",
			Version: "6.6",
		},
		Connections: []*inventory.Config{
			{
				Type: "ssh",
				Options: map[string]string{
					"host": "10.0.0.1",
					"key":  strings.Repeat("k", 2048),
				},
			},
		},
		RelatedAssets: []*inventory.Asset{
			{Mrn: "//assets.api.mondoo.app/assets/child", Name: "child"},
		},
		Labels: map[string]string{
			"env": "prod",
		},
		Url: "https://app.mondoo.com/assets/abc",
	}

	r.AddReport(asset, &AssetReport{
		ResolvedPolicy: &policy.ResolvedPolicy{},
		Report: &policy.Report{
			Score: &policy.Score{Value: 100},
		},
	})

	asset.Name = "mutated"
	asset.Labels["env"] = "dev"

	full := r.Reports().GetFull()
	requireAsset := full.Assets[asset.Mrn]

	assert.NotNil(t, requireAsset)
	assert.Equal(t, "target", requireAsset.Name)
	assert.Equal(t, "prod", requireAsset.Labels["env"])
	assert.Nil(t, requireAsset.Connections)
	assert.Nil(t, requireAsset.RelatedAssets)
}

func BenchmarkSnapshotAsset_TrimmedRetention(b *testing.B) {
	asset := &inventory.Asset{
		Mrn:  "//assets.api.mondoo.app/assets/abc",
		Name: "target",
		Connections: []*inventory.Config{
			{
				Type: "ssh",
				Options: map[string]string{
					"payload": strings.Repeat("x", 4096),
				},
			},
		},
		RelatedAssets: []*inventory.Asset{
			{Mrn: "//assets.api.mondoo.app/assets/child", Name: "child"},
		},
		Labels: map[string]string{"scope": "bench"},
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		s := snapshotAsset(asset)
		if i == 0 {
			if s.SizeVT() >= asset.SizeVT() {
				b.Fatalf("expected snapshot to retain less data: snapshot=%d original=%d", s.SizeVT(), asset.SizeVT())
			}
		}
		s.Mrn = fmt.Sprintf("%s-%d", s.Mrn, i)
	}
}
