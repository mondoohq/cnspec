// Copyright Mondoo, Inc. 2026
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/v13/policy"
	"go.mondoo.com/mql/v13/providers-sdk/v1/testutils"
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
