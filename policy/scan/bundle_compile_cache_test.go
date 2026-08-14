// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/v13/policy"
	"go.mondoo.com/mql/v13"
	"go.mondoo.com/mql/v13/mqlc"
	"go.mondoo.com/mql/v13/providers-sdk/v1/resources"
	"go.mondoo.com/mql/v13/providers-sdk/v1/testutils"
)

func testBundle(t *testing.T) *policy.Bundle {
	t.Helper()
	loader := policy.DefaultBundleLoader()
	bundle, err := loader.BundleFromPaths("./testdata/kubernetes-security.mql.yaml")
	require.NoError(t, err)
	return bundle
}

func testCompileConf(t *testing.T, schema resources.ResourcesSchema) policy.BundleCompileConf {
	t.Helper()
	return policy.BundleCompileConf{
		CompilerConfig: mqlc.NewConfig(schema, mql.DefaultFeatures),
		RemoveFailing:  true,
	}
}

func TestBundleCompileCache_ReusesResultForSameSchema(t *testing.T) {
	runtime := testutils.Local()
	bundle := testBundle(t)
	conf := testCompileConf(t, runtime.Schema())

	c := newBundleCompileCache()

	first, err := c.compile(context.Background(), bundle, conf)
	require.NoError(t, err)
	require.NotNil(t, first)

	second, err := c.compile(context.Background(), bundle, conf)
	require.NoError(t, err)
	require.NotNil(t, second)

	// The maps are reused, so the compiled code is the same object.
	assert.NotSame(t, first, second, "each caller gets its own map header")
	assert.Equal(t, first.OwnerMrn, second.OwnerMrn)
	require.Equal(t, len(first.Queries), len(second.Queries))
	for mrn, q := range first.Queries {
		other, ok := second.Queries[mrn]
		require.True(t, ok, "query %s missing on reuse", mrn)
		assert.Same(t, q, other)
	}
	require.Equal(t, len(first.Code), len(second.Code))
	for id, code := range first.Code {
		other, ok := second.Code[id]
		require.True(t, ok, "code %s missing on reuse", id)
		assert.Same(t, code, other)
	}
}

func TestBundleCompileCache_CarriesCallerLibrary(t *testing.T) {
	runtime := testutils.Local()
	bundle := testBundle(t)

	c := newBundleCompileCache()

	confA := testCompileConf(t, runtime.Schema())
	libA := &fakeLibrary{}
	confA.Library = libA

	first, err := c.compile(context.Background(), bundle, confA)
	require.NoError(t, err)
	assert.Same(t, libA, first.Library)

	confB := testCompileConf(t, runtime.Schema())
	libB := &fakeLibrary{}
	confB.Library = libB

	second, err := c.compile(context.Background(), bundle, confB)
	require.NoError(t, err)
	assert.Same(t, libB, second.Library, "reused map must carry the caller's library")
	assert.Same(t, libA, first.Library, "the earlier map must keep its own library")
}

func TestBundleCompileCache_RecompilesForNewBundle(t *testing.T) {
	runtime := testutils.Local()
	conf := testCompileConf(t, runtime.Schema())

	c := newBundleCompileCache()

	first, err := c.compile(context.Background(), testBundle(t), conf)
	require.NoError(t, err)

	// A different bundle object must not reuse the previous result.
	second, err := c.compile(context.Background(), testBundle(t), conf)
	require.NoError(t, err)

	require.NotEmpty(t, first.Queries)
	for mrn, q := range first.Queries {
		other, ok := second.Queries[mrn]
		require.True(t, ok)
		assert.NotSame(t, q, other, "a new bundle must be compiled again")
		break
	}
}

func TestSchemaKey(t *testing.T) {
	assert.Equal(t, "", schemaKey(nil))

	runtime := testutils.Local()
	key := schemaKey(runtime.Schema())
	assert.NotEmpty(t, key)
	// Stable across calls.
	assert.Equal(t, key, schemaKey(runtime.Schema()))
}

type fakeLibrary struct{}

func (f *fakeLibrary) QueryExists(ctx context.Context, mrn string) (bool, error) {
	return false, nil
}

func (f *fakeLibrary) PolicyExists(ctx context.Context, mrn string) (bool, error) {
	return false, nil
}
