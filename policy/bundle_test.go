package policy

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBundleFromPaths(t *testing.T) {
	bundle, err := BundleFromPaths("./examples/example.mql.yaml")
	require.NoError(t, err)
	require.NotNil(t, bundle)
	assert.Len(t, bundle.Queries, 8)
	assert.Len(t, bundle.Policies, 2)
}
