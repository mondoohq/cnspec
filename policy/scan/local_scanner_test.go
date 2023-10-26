// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v9/providers"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/upstream"
)

func TestFilterPreprocess(t *testing.T) {
	// given
	filters := []string{
		"namespace1/policy1",
		"namespace2/policy2",
		"//registry.mondoo.com/namespace/namespace3/policies/policy3",
	}

	// when
	preprocessed := preprocessPolicyFilters(filters)

	// then
	assert.Equal(t, []string{
		"//registry.mondoo.com/namespace/namespace1/policies/policy1",
		"//registry.mondoo.com/namespace/namespace2/policies/policy2",
		"//registry.mondoo.com/namespace/namespace3/policies/policy3",
	}, preprocessed)
}

func TestGetUpstreamConfig(t *testing.T) {
	t.Run("with job creds", func(t *testing.T) {
		opts := []ScannerOption{
			AllowJobCredentials(),
		}

		pk, err := os.ReadFile("../testdata/private-key.p8")
		require.NoError(t, err)

		cert, err := os.ReadFile("../testdata/cert.pem")
		require.NoError(t, err)

		job := &Job{
			Inventory: &inventory.Inventory{
				Spec: &inventory.InventorySpec{
					UpstreamCredentials: &upstream.ServiceAccountCredentials{
						ApiEndpoint: "api",
						ParentMrn:   "space-mrn",
						PrivateKey:  string(pk),
						Certificate: string(cert),
					},
				},
			},
		}
		scanner := NewLocalScanner(opts...)
		_, err = scanner.getUpstreamConfig(false, job)
		require.NoError(t, err)

		_, err = scanner.getUpstreamConfig(true, &Job{})
		require.NoError(t, err)
	})
}

func TestCreateAssetList(t *testing.T) {
	t.Run("with inventory", func(t *testing.T) {
		job := &Job{
			Inventory: &inventory.Inventory{
				Spec: &inventory.InventorySpec{
					Assets: []*inventory.Asset{
						{
							Connections: []*inventory.Config{
								{
									Type: "k8s",
									Options: map[string]string{
										"path": "./testdata/2pods.yaml",
									},
									Discover: &inventory.Discovery{
										Targets: []string{"auto"},
									},
								},
							},
							ManagedBy: "mondoo-operator-123",
						},
					},
				},
			},
		}
		assetList, candidates, err := createAssetCandidateList(context.TODO(), job, nil, providers.NullRecording{})
		require.NoError(t, err)
		require.Len(t, assetList, 1)
		require.Len(t, candidates, 3)
		require.Equal(t, "mondoo-operator-123", candidates[0].asset.ManagedBy)
		require.Equal(t, "mondoo-operator-123", candidates[1].asset.ManagedBy)
		require.Equal(t, "mondoo-operator-123", candidates[2].asset.ManagedBy)
	})
}
