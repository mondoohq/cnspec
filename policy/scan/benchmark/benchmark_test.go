// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package benchmark

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v11"
	"go.mondoo.com/cnquery/v11/mqlc"
	"go.mondoo.com/cnquery/v11/providers"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/testutils"
	"go.mondoo.com/cnspec/v11/policy"
	"go.mondoo.com/cnspec/v11/policy/scan"
)

func init() {
	log.Logger = log.Logger.Level(zerolog.Disabled)
	zerolog.SetGlobalLevel(zerolog.Disabled)
}

func BenchmarkScan_SingleAsset(b *testing.B) {
	defer providers.Coordinator.Shutdown()
	ctx := context.Background()
	runtime := testutils.Local()
	conf := mqlc.NewConfig(runtime.Schema(), cnquery.DefaultFeatures)
	job := &scan.Job{
		Inventory: &inventory.Inventory{
			Spec: &inventory.InventorySpec{
				Assets: []*inventory.Asset{
					{
						Connections: []*inventory.Config{
							{
								Type: "k8s",
								Options: map[string]string{
									"path": "../testdata/1pod.yaml",
								},
								Discover: &inventory.Discovery{
									Targets: []string{"pods"},
								},
							},
						},
					},
				},
			},
		},
	}

	loader := policy.DefaultBundleLoader()
	bundle, err := loader.BundleFromPaths("../testdata/kubernetes-security.mql.yaml")
	require.NoError(b, err)

	_, err = bundle.CompileExt(context.Background(), policy.BundleCompileConf{
		CompilerConfig: conf,
		RemoveFailing:  true,
	})
	require.NoError(b, err)

	job.Bundle = bundle

	scanner := scan.NewLocalScanner(
		scan.DisableProgressBar(),
		scan.WithRuntime(runtime.(*providers.Runtime)),
	)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		res, err := scanner.RunIncognito(ctx, job)
		require.NoError(b, err)
		require.NotNil(b, res)
	}
}

func BenchmarkScan_MultipleAssets(b *testing.B) {
	defer providers.Coordinator.Shutdown()
	ctx := context.Background()
	runtime := testutils.Local()
	conf := mqlc.NewConfig(runtime.Schema(), cnquery.DefaultFeatures)
	job := &scan.Job{
		Inventory: &inventory.Inventory{
			Spec: &inventory.InventorySpec{
				Assets: []*inventory.Asset{
					{
						Connections: []*inventory.Config{
							{
								Type: "k8s",
								Options: map[string]string{
									"path": "../testdata/2pods.yaml",
								},
								Discover: &inventory.Discovery{
									Targets: []string{"pods"},
								},
							},
						},
					},
				},
			},
		},
	}

	loader := policy.DefaultBundleLoader()
	bundle, err := loader.BundleFromPaths("../testdata/kubernetes-security.mql.yaml")
	require.NoError(b, err)

	_, err = bundle.CompileExt(context.Background(), policy.BundleCompileConf{
		CompilerConfig: conf,
		RemoveFailing:  true,
	})
	require.NoError(b, err)

	job.Bundle = bundle

	scanner := scan.NewLocalScanner(
		scan.DisableProgressBar(),
		scan.WithRuntime(runtime.(*providers.Runtime)),
	)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		res, err := scanner.RunIncognito(ctx, job)
		require.NoError(b, err)
		require.NotNil(b, res)
	}
}
