// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/mql"
	"go.mondoo.com/mql/mqlc"
	"go.mondoo.com/mql/providers-sdk/v1/testutils"
)

func TestFilterProviderLookups(t *testing.T) {
	osSchema := testutils.MustLoadSchema(testutils.SchemaProvider{Provider: "os"})
	coreSchema := testutils.MustLoadSchema(testutils.SchemaProvider{Provider: "core"})
	schema := coreSchema.Add(osSchema)

	coreID := schema.Lookup("mondoo").Provider
	require.NotEmpty(t, coreID)
	osID := schema.Lookup("sshd.config").Provider
	require.NotEmpty(t, osID)

	t.Run("collects the providers of referenced resources", func(t *testing.T) {
		lookups := filterProviderLookups(schema, []*policy.Mquery{
			{Mql: "mondoo.version != ''"},
			{Mql: "sshd.config.params['x'] == 'y'"},
		})

		ids := make([]string, len(lookups))
		for i := range lookups {
			ids[i] = lookups[i].ID
		}
		assert.ElementsMatch(t, []string{coreID, osID}, ids)
	})

	t.Run("deduplicates providers across filters", func(t *testing.T) {
		lookups := filterProviderLookups(schema, []*policy.Mquery{
			{Mql: "mondoo.version != ''"},
			{Mql: "mondoo.build != ''"},
		})
		require.Len(t, lookups, 1)
		assert.Equal(t, coreID, lookups[0].ID)
	})

	t.Run("skips filters that do not compile", func(t *testing.T) {
		lookups := filterProviderLookups(schema, []*policy.Mquery{
			{Mql: "nonexistent.resource == 1"},
		})
		assert.Empty(t, lookups)
	})
}

func TestFiltersMatch(t *testing.T) {
	matchedCode := map[string]struct{}{"code-1": {}}
	matchedMql := map[string]struct{}{"asset.platform == 'db2'": {}}

	t.Run("nil or empty filters conservatively match", func(t *testing.T) {
		assert.True(t, filtersMatch(nil, matchedCode, matchedMql))
		assert.True(t, filtersMatch(&policy.Filters{}, matchedCode, matchedMql))
	})

	t.Run("matches by items key", func(t *testing.T) {
		filters := &policy.Filters{Items: map[string]*policy.Mquery{
			"code-1": {Mql: "something.else"},
		}}
		assert.True(t, filtersMatch(filters, matchedCode, matchedMql))
	})

	t.Run("matches by the item's code id", func(t *testing.T) {
		filters := &policy.Filters{Items: map[string]*policy.Mquery{
			"other-key": {CodeId: "code-1", Mql: "something.else"},
		}}
		assert.True(t, filtersMatch(filters, matchedCode, matchedMql))
	})

	t.Run("falls back to matching the raw mql", func(t *testing.T) {
		filters := &policy.Filters{Items: map[string]*policy.Mquery{
			"other-key": {Mql: "asset.platform == 'db2'\n"},
		}}
		assert.True(t, filtersMatch(filters, matchedCode, matchedMql))
	})

	t.Run("does not match unrelated filters", func(t *testing.T) {
		filters := &policy.Filters{Items: map[string]*policy.Mquery{
			"other-key": {CodeId: "code-2", Mql: "asset.family.contains('linux')"},
		}}
		assert.False(t, filtersMatch(filters, matchedCode, matchedMql))
	})
}

func filterSet(mqls ...string) *policy.Filters {
	res := &policy.Filters{Items: map[string]*policy.Mquery{}}
	for i, m := range mqls {
		res.Items[string(rune('a'+i))] = &policy.Mquery{Mql: m}
	}
	return res
}

func TestAssetFiltersCompile(t *testing.T) {
	// core only: a filter reaching into any other provider cannot compile
	conf := mqlc.NewConfig(
		testutils.MustLoadSchema(testutils.SchemaProvider{Provider: "core"}),
		mql.DefaultFeatures,
	)

	t.Run("a policy without filters compiles vacuously", func(t *testing.T) {
		assert.True(t, assetFiltersCompile(conf, nil))
		assert.True(t, assetFiltersCompile(conf, []*policy.Filters{nil, {}}))
	})

	t.Run("a filter on core resources compiles", func(t *testing.T) {
		assert.True(t, assetFiltersCompile(conf, []*policy.Filters{filterSet("asset.platform == 'ubuntu'")}))
	})

	t.Run("a filter needing an uninstalled provider does not", func(t *testing.T) {
		assert.False(t, assetFiltersCompile(conf, []*policy.Filters{filterSet("k8s.deployment")}))
	})

	t.Run("one failing filter is enough", func(t *testing.T) {
		assert.False(t, assetFiltersCompile(conf, []*policy.Filters{
			filterSet("asset.platform == 'ubuntu'"),
			filterSet("k8s.deployment"),
		}))
	})

	t.Run("an empty mql filter is skipped", func(t *testing.T) {
		assert.True(t, assetFiltersCompile(conf, []*policy.Filters{
			{Items: map[string]*policy.Mquery{"a": {}, "b": nil}},
		}))
	})
}

func TestPolicyAssetFilters(t *testing.T) {
	computed := filterSet("asset.platform == 'ubuntu'")
	group := filterSet("asset.family.contains('linux')")

	// nil groups are dropped; a group without filters contributes a nil entry
	// that assetFiltersCompile skips
	assert.Equal(t, []*policy.Filters{computed, group, nil}, policyAssetFilters(&policy.Policy{
		ComputedFilters: computed,
		Groups:          []*policy.PolicyGroup{nil, {Filters: group}, {}},
	}))
}

func TestPackAssetFilters(t *testing.T) {
	computed := filterSet("asset.platform == 'ubuntu'")
	own := filterSet("asset.family.contains('linux')")
	group := filterSet("asset.name != ''")

	// unlike a policy, a querypack carries authored filters on the pack itself
	assert.Equal(t, []*policy.Filters{computed, own, group}, packAssetFilters(&policy.QueryPack{
		ComputedFilters: computed,
		Filters:         own,
		Groups:          []*policy.QueryGroup{{Filters: group}},
	}))
}

func TestFilterRequirements(t *testing.T) {
	schema := testutils.MustLoadSchema(testutils.SchemaProvider{Provider: "core"})

	t.Run("nil bundle", func(t *testing.T) {
		assert.Empty(t, filterRequirements(nil, schema))
	})

	t.Run("defers requirements whose filters already compile", func(t *testing.T) {
		assert.Empty(t, filterRequirements(&policy.Bundle{
			Policies: []*policy.Policy{{
				Name:            "db2 on db2 only",
				Require:         []*policy.Requirement{{Provider: "db2"}},
				ComputedFilters: filterSet("asset.platform == 'db2'"),
			}},
		}, schema))
	})

	t.Run("pulls requirements the filters cannot compile without", func(t *testing.T) {
		assert.Equal(t, []string{"k8s"}, filterRequirements(&policy.Bundle{
			Policies: []*policy.Policy{
				{
					Name:            "needs k8s to even match",
					Require:         []*policy.Requirement{{Provider: "k8s"}},
					ComputedFilters: filterSet("k8s.deployment"),
				},
				{
					Name:            "matches on the platform alone",
					Require:         []*policy.Requirement{{Provider: "db2"}},
					ComputedFilters: filterSet("asset.platform == 'db2'"),
				},
			},
		}, schema))
	})

	t.Run("covers querypacks too", func(t *testing.T) {
		assert.Equal(t, []string{"k8s"}, filterRequirements(&policy.Bundle{
			Packs: []*policy.QueryPack{{
				Name:    "k8s pack",
				Require: []*policy.Requirement{{Provider: "k8s"}},
				Filters: filterSet("k8s.deployment"),
			}},
		}, schema))
	})

	t.Run("a policy without requirements is never considered", func(t *testing.T) {
		assert.Empty(t, filterRequirements(&policy.Bundle{
			Policies: []*policy.Policy{{Name: "no require", ComputedFilters: filterSet("k8s.deployment")}},
		}, schema))
	})
}
