// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package policy_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/explorer"
	"go.mondoo.com/cnspec/internal/datalakes/inmemory"
	"go.mondoo.com/cnspec/policy"
)

func TestBundleFromPaths(t *testing.T) {
	t.Run("mql bundle file with multiple queries", func(t *testing.T) {
		bundle, err := policy.BundleFromPaths("../examples/example.mql.yaml")
		require.NoError(t, err)
		require.NotNil(t, bundle)
		assert.Len(t, bundle.Queries, 1)
		require.Len(t, bundle.Policies, 1)
		require.Len(t, bundle.Policies[0].Groups, 1)
		assert.Len(t, bundle.Policies[0].Groups[0].Checks, 3)
		assert.Len(t, bundle.Policies[0].Groups[0].Queries, 2)
	})

	t.Run("mql bundle file with multiple policies and queries", func(t *testing.T) {
		bundle, err := policy.BundleFromPaths("../examples/complex.mql.yaml")
		require.NoError(t, err)
		require.NotNil(t, bundle)
		assert.Len(t, bundle.Queries, 5)
		assert.Len(t, bundle.Policies, 2)
	})

	t.Run("mql bundle file with directory structure", func(t *testing.T) {
		bundle, err := policy.BundleFromPaths("../examples/directory")
		require.NoError(t, err)
		require.NotNil(t, bundle)
		assert.Len(t, bundle.Queries, 5)
		assert.Len(t, bundle.Policies, 2)
	})
}

func TestPolicyBundleSort(t *testing.T) {
	pb, err := policy.BundleFromPaths("./testdata/policybundle-deps.mql.yaml")
	require.NoError(t, err)
	assert.Equal(t, 3, len(pb.Policies))
	pbm := pb.ToMap()

	policies, err := pbm.PoliciesSortedByDependency()
	require.NoError(t, err)
	assert.Equal(t, 3, len(policies))

	assert.Equal(t, "//policy.api.mondoo.app/policies/debian-10-level-1-server", policies[0].Mrn)
	assert.Equal(t, "//captain.api.mondoo.app/spaces/adoring-moore-542492", policies[1].Mrn)
	assert.Equal(t, "//assets.api.mondoo.app/spaces/adoring-moore-542492/assets/1dKBiOi5lkI2ov48plcowIy8WEl", policies[2].Mrn)
}

func TestBundleCompile(t *testing.T) {
	bundle, err := policy.BundleFromPaths("../examples/complex.mql.yaml")
	require.NoError(t, err)
	require.NotNil(t, bundle)

	bundlemap, err := bundle.Compile(context.Background(), nil)
	require.NoError(t, err)
	require.NotNil(t, bundlemap)

	base := bundlemap.Queries["//local.cnspec.io/run/local-execution/queries/uname"]
	require.NotNil(t, base, "variant base cannot be nil")

	variant1 := bundlemap.Queries["//local.cnspec.io/run/local-execution/queries/unix-uname"]
	require.NotNil(t, variant1, "variant cannot be nil")

	assert.Equal(t, base.Title, variant1.Title)
}

func TestBundleFrameworkGraphExecutionChecksum(t *testing.T) {
	bundleStr := `
owner_mrn: //test.sth
policies:
- uid: policy1
  groups:
  - filters: "true"
    checks:
    - uid: check-fail
      mql: 1 == 2
    - uid: check-pass-1
      mql: 1 == 1
    - uid: check-pass-2
      mql: 2 == 2
- uid: policy2
  groups:
  - filters: "true"
    checks:
    - uid: check-pass-3
      mql: 3 == 3
- uid: policy3
  groups:
  - filters: "true"
    checks:
    - uid: check-pass-4
      mql: 4 == 4
frameworks:
- uid: framework0
  name: framework0
  groups:
  - title: group0
    controls:
  - uid: control1
- uid: framework1
  name: framework1
  groups:
  - title: group1
    controls:
    - uid: control1
      title: control1
    - uid: control2
      title: control2
    - uid: control3
      title: control3
  dependencies:
  - mrn: //test.sth/frameworks/framework0
- uid: framework2
  name: framework1
  groups:
  - title: group1
    controls:
    - uid: control1
      title: control1
    - uid: control2
      title: control2
- uid: parent-framework
  dependencies:
  - mrn: //test.sth/frameworks/framework1

framework_maps:
- uid: framework-map1
  framework_owner:
    uid: framework1
  policy_dependencies:
  - uid: policy1
  controls:
  - uid: control1
    checks:
    - uid: check-pass-1
    policies:
    - uid: policy2
  - uid: control2
    checks:
    - uid: check-pass-2
    - uid: check-fail
`

	testCases := []struct {
		name   string
		modify func(bundle *policy.Bundle)
	}{
		{
			name: "no modification",
			modify: func(bundle *policy.Bundle) {
			},
		},
		{
			name: "when a control is removed",
			modify: func(bundle *policy.Bundle) {
				bundle.Frameworks[1].Groups[0].Controls = bundle.Frameworks[1].Groups[0].Controls[:2]
			},
		},
		{
			name: "when a control action is changed",
			modify: func(bundle *policy.Bundle) {
				bundle.Frameworks[1].Groups[0].Controls[0].Action = explorer.Action_DEACTIVATE
			},
		},
		{
			name: "when a framework dependency action changes",
			modify: func(bundle *policy.Bundle) {
				bundle.Frameworks[1].Dependencies[0].Action = explorer.Action_IGNORE
			},
		},
		{
			name: "when a frameworkmap control action changes",
			modify: func(bundle *policy.Bundle) {
				bundle.FrameworkMaps[0].Controls[0].Checks[0].Action = explorer.Action_DEACTIVATE
			},
		},
		{
			name: "when a frameworkmap control has a check added",
			modify: func(bundle *policy.Bundle) {
				bundle.FrameworkMaps[0].Controls[0].Checks = append(bundle.FrameworkMaps[0].Controls[0].Checks, &policy.ControlRef{
					Uid: "check-pass-2",
				})
			},
		},
		{
			name: "when a frameworkmap control has a check removed",
			modify: func(bundle *policy.Bundle) {
				bundle.FrameworkMaps[0].Controls[1].Checks = bundle.FrameworkMaps[0].Controls[1].Checks[:1]
			},
		},
		{
			name: "when a frameworkmap control policy has changed",
			modify: func(bundle *policy.Bundle) {
				bundle.FrameworkMaps[0].Controls[0].Policies = []*policy.ControlRef{
					{
						Uid: "policy3",
					},
				}
			},
		},
		{
			name: "when a frameworkmap control policy has been removed",
			modify: func(bundle *policy.Bundle) {
				bundle.FrameworkMaps[0].Controls[0].Policies = []*policy.ControlRef{}
			},
		},
		{
			name: "when a frameworkmap control has been removed",
			modify: func(bundle *policy.Bundle) {
				bundle.FrameworkMaps[0].Controls = bundle.FrameworkMaps[0].Controls[:1]
			},
		},
	}

	checksumToTestCases := map[string][]string{}

	_, srv, err := inmemory.NewServices(nil)
	require.NoError(t, err)

	t.Run("no duplicate checksums", func(t *testing.T) {
		for _, tc := range testCases {
			bundle := parseBundle(t, bundleStr)
			tc.modify(bundle)

			_, err := srv.SetBundle(context.Background(), bundle)

			checksumToTestCases[bundle.Frameworks[1].GraphExecutionChecksum] = append(checksumToTestCases[bundle.Frameworks[1].GraphExecutionChecksum], tc.name)
			require.NoError(t, err)
		}

		// There should be no duplicate checksums
		for checksum, testCases := range checksumToTestCases {
			assert.Len(t, testCases, 1, "duplicate checksum %s in test cases: %s", checksum, strings.Join(testCases, ", "))
		}
	})

	t.Run("checksums reproduceable", func(t *testing.T) {
		for _, tc := range testCases {
			checksums := []string{}

			for i := 0; i < 10; i++ {
				bundle := parseBundle(t, bundleStr)
				tc.modify(bundle)

				_, err := srv.SetBundle(context.Background(), bundle)
				require.NoError(t, err)

				checksums = append(checksums, bundle.Frameworks[1].GraphExecutionChecksum)
			}
			// All checksums should be the same
			for i := 1; i < len(checksums); i++ {
				assert.Equal(t, checksums[0], checksums[i], "checksums should be the same")
			}
		}

	})
}
