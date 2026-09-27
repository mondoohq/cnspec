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
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/mql"
	"go.mondoo.com/mql/llx"
	"go.mondoo.com/mql/mqlc"
	"go.mondoo.com/mql/providers-sdk/v1/resources"
	"go.mondoo.com/mql/providers-sdk/v1/testutils"
	"google.golang.org/protobuf/proto"
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

	// The compiled code is reused, while mutable bundle values are isolated.
	assert.NotSame(t, first, second, "each caller gets its own map")
	assert.Equal(t, first.OwnerMrn, second.OwnerMrn)
	require.Equal(t, len(first.Queries), len(second.Queries))
	for mrn, q := range first.Queries {
		other, ok := second.Queries[mrn]
		require.True(t, ok, "query %s missing on reuse", mrn)
		assert.NotSame(t, q, other)
		assert.Equal(t, q.Mrn, other.Mrn)
	}
	require.Equal(t, len(first.Code), len(second.Code))
	for id, code := range first.Code {
		other, ok := second.Code[id]
		require.True(t, ok, "code %s missing on reuse", id)
		assert.Same(t, code, other)
	}
}

func TestBundleCompileCache_LeavesSourceBundleUntouched(t *testing.T) {
	runtime := testutils.Local()
	bundle := testBundle(t)
	conf := testCompileConf(t, runtime.Schema())

	before, err := proto.Marshal(bundle)
	require.NoError(t, err)

	_, err = newBundleCompileCache().compile(context.Background(), bundle, conf)
	require.NoError(t, err)

	after, err := proto.Marshal(bundle)
	require.NoError(t, err)
	assert.Equal(t, before, after)
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

func TestBundleCompileCache_GivesEachCallerItsOwnMaps(t *testing.T) {
	runtime := testutils.Local()
	bundle := testBundle(t)
	conf := testCompileConf(t, runtime.Schema())

	c := newBundleCompileCache()

	first, err := c.compile(context.Background(), bundle, conf)
	require.NoError(t, err)
	second, err := c.compile(context.Background(), bundle, conf)
	require.NoError(t, err)

	// The policy hub inserts into these maps while a scan runs. Each caller
	// needs its own header, otherwise concurrent assets race.
	first.Queries["//test/query"] = &policy.Mquery{Mrn: "//test/query"}
	first.Policies["//test/policy"] = &policy.Policy{Mrn: "//test/policy"}
	first.Props["//test/prop"] = &policy.Property{Mrn: "//test/prop"}
	first.Frameworks["//test/framework"] = &policy.Framework{Mrn: "//test/framework"}
	first.RiskFactors["//test/risk"] = &policy.RiskFactor{Mrn: "//test/risk"}
	first.Code["//test/code"] = &llx.CodeBundle{}

	assert.NotContains(t, second.Queries, "//test/query")
	assert.NotContains(t, second.Policies, "//test/policy")
	assert.NotContains(t, second.Props, "//test/prop")
	assert.NotContains(t, second.Frameworks, "//test/framework")
	assert.NotContains(t, second.RiskFactors, "//test/risk")
	assert.NotContains(t, second.Code, "//test/code")

	// The cached entry must stay clean too.
	third, err := c.compile(context.Background(), bundle, conf)
	require.NoError(t, err)
	assert.NotContains(t, third.Queries, "//test/query")
}

func TestBundleCompileCache_ConcurrentAssetsDoNotRace(t *testing.T) {
	runtime := testutils.Local()
	bundle := testBundle(t)
	conf := testCompileConf(t, runtime.Schema())

	c := newBundleCompileCache()

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			m, err := c.compile(context.Background(), bundle, conf)
			if err != nil {
				t.Error(err)
				return
			}
			// Mimic the inserts the policy hub does per asset.
			m.Queries["//test/query/"+strconv.Itoa(i)] = &policy.Mquery{}
			m.Policies["//test/policy/"+strconv.Itoa(i)] = &policy.Policy{}
		}(i)
	}
	wg.Wait()
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
