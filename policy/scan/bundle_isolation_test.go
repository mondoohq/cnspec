// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"context"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/internal/datalakes/inmemory"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/providers-sdk/v1/testutils"
	"go.mondoo.com/mql/providers-sdk/v1/upstream"
	"google.golang.org/protobuf/proto"
)

func sharedTestBundle() *policy.Bundle {
	return &policy.Bundle{
		Policies: []*policy.Policy{{
			Uid:     "test-policy",
			Name:    "Test Policy",
			Version: "1.0.0",
			Groups: []*policy.PolicyGroup{{
				Filters: &policy.Filters{Items: map[string]*policy.Mquery{
					"//local/filter": {Mql: "true"},
				}},
				Checks: []*policy.Mquery{
					{Uid: "check-name", Title: "asset has a name", Mql: "asset.name != ''"},
					{Uid: "check-platform", Title: "asset has a platform", Mql: "asset.platform != ''"},
				},
			}},
		}},
	}
}

func newTestAssetScanner(t *testing.T, bundle *policy.Bundle, assetMrn string) *localAssetScanner {
	t.Helper()
	runtime := testutils.Local()
	_, services, err := inmemory.NewServices(runtime)
	require.NoError(t, err)

	return &localAssetScanner{
		services: services,
		Runtime:  runtime,
		job: &AssetJob{
			Ctx:            context.Background(),
			Asset:          &inventory.Asset{Mrn: assetMrn, Name: "test-asset"},
			Bundle:         bundle,
			UpstreamConfig: &upstream.UpstreamConfig{Incognito: true},
		},
	}
}

func TestPrepareAssetLeavesTheJobBundleUntouched(t *testing.T) {
	bundle := sharedTestBundle()
	before, err := proto.Marshal(bundle)
	require.NoError(t, err)

	scanner := newTestAssetScanner(t, bundle, "//assets/test-1")
	require.NoError(t, scanner.prepareAsset())

	after, err := proto.Marshal(bundle)
	require.NoError(t, err)
	assert.Equal(t, before, after,
		"prepareAsset must compile a copy: the job's bundle is shared by every asset in the scan")
}

// TestConcurrentPrepareAssetOnOneBundle is the regression test for the crash
// that parallel scanning used to produce: several assets sharing one job bundle
// each compiled it in place, which raced on the queries' MRNs and checksums and
// killed the process outright with a concurrent map write in ComputedFilters.
// Run under -race, this fails without the per-asset copy.
func TestConcurrentPrepareAssetOnOneBundle(t *testing.T) {
	bundle := sharedTestBundle()
	before, err := proto.Marshal(bundle)
	require.NoError(t, err)

	const assets = 8
	scanners := make([]*localAssetScanner, assets)
	for i := range scanners {
		scanners[i] = newTestAssetScanner(t, bundle, "//assets/test-"+strconv.Itoa(i))
	}

	errs := make([]error, assets)
	var wg sync.WaitGroup
	for i := range scanners {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs[i] = scanners[i].prepareAsset()
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		assert.NoError(t, err, "asset %d", i)
	}

	after, err := proto.Marshal(bundle)
	require.NoError(t, err)
	assert.Equal(t, before, after, "concurrent asset preparation must not touch the shared bundle")
}

// TestPrepareAssetMapsPropsFromTheCopy guards the one value prepareAsset reads
// back out of the compiled bundle. Property MRNs are filled in during
// compilation, so they have to be read from the asset's copy; reading them from
// the job's (now untouched) bundle would silently drop every override.
func TestPrepareAssetMapsPropsFromTheCopy(t *testing.T) {
	bundle := sharedTestBundle()
	bundle.Policies[0].Props = []*policy.Property{{
		Uid: "homeDir",
		Mql: "return '/home'",
	}}

	scanner := newTestAssetScanner(t, bundle, "//assets/test-props")
	scanner.job.Props = map[string]string{"homeDir": "return '/root'"}

	require.NoError(t, scanner.prepareAsset())

	compiled := bundle.CloneVT()
	_, err := compiled.CompileExt(context.Background(), policy.BundleCompileConf{
		CompilerConfig: scanner.services.NewCompilerConfig(),
		Library:        scanner.services.DataLake,
		RemoveFailing:  true,
	})
	require.NoError(t, err)

	req, err := scanner.mapPropOverrides(compiled)
	require.NoError(t, err)
	require.Len(t, req.Props, 1, "the override must match the property the bundle exposes")
	assert.Equal(t, "return '/root'", req.Props[0].Mql)
	require.Len(t, req.Props[0].For, 1)
	assert.Contains(t, req.Props[0].For[0].Mrn, "homeDir",
		"the override must resolve to the property MRN written during compilation")
}
