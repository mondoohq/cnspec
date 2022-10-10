package policy

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBundleFromPaths(t *testing.T) {
	t.Run("mql bundle file with multiple queries", func(t *testing.T) {
		bundle, err := BundleFromPaths("./examples/example.mql.yaml")
		require.NoError(t, err)
		require.NotNil(t, bundle)
		assert.Len(t, bundle.Queries, 4)
		assert.Len(t, bundle.Policies, 1)
	})

	t.Run("mql bundle file with multiple policies and queries", func(t *testing.T) {
		bundle, err := BundleFromPaths("./examples/multi.mql.yaml")
		require.NoError(t, err)
		require.NotNil(t, bundle)
		assert.Len(t, bundle.Queries, 5)
		assert.Len(t, bundle.Policies, 2)
	})

	t.Run("mql bundle file with directory structure", func(t *testing.T) {
		bundle, err := BundleFromPaths("./examples/directory")
		require.NoError(t, err)
		require.NotNil(t, bundle)
		assert.Len(t, bundle.Queries, 5)
		assert.Len(t, bundle.Policies, 2)
	})
}

func TestPolicyBundleSort(t *testing.T) {
	pb, err := BundleFromPaths("./testdata/policybundle-deps.mql.yaml")
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
