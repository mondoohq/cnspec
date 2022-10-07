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
