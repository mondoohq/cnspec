// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package bundle

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13"
	"go.mondoo.com/mql/v13/mqlc"
	"go.mondoo.com/mql/v13/providers-sdk/v1/resources"
	"google.golang.org/protobuf/proto"
)

// deprecateOverlay wraps a real schema and forces Maturity = "deprecated" on
// the specified resources and fields, returning clones from Lookup/LookupField
// so the underlying schema is not mutated. Everything else proxies through.
type deprecateOverlay struct {
	inner            resources.ResourcesSchema
	deprecatedRes    map[string]struct{}
	deprecatedFields map[string]map[string]struct{}
}

func (d *deprecateOverlay) Lookup(name string) *resources.ResourceInfo {
	r := d.inner.Lookup(name)
	if r == nil {
		return nil
	}
	if _, ok := d.deprecatedRes[name]; ok {
		clone := proto.Clone(r).(*resources.ResourceInfo)
		clone.Maturity = resources.MaturityDeprecated
		return clone
	}
	return r
}

func (d *deprecateOverlay) LookupField(resource, field string) (*resources.ResourceInfo, *resources.Field) {
	r, f := d.inner.LookupField(resource, field)
	if f == nil {
		return r, f
	}
	if fields, ok := d.deprecatedFields[resource]; ok {
		if _, ok := fields[field]; ok {
			clone := proto.Clone(f).(*resources.Field)
			clone.Maturity = resources.MaturityDeprecated
			return r, clone
		}
	}
	return r, f
}

func (d *deprecateOverlay) FindField(r *resources.ResourceInfo, field string) (resources.FieldPath, []*resources.Field, bool) {
	return d.inner.FindField(r, field)
}

func (d *deprecateOverlay) AllResources() map[string]*resources.ResourceInfo {
	return d.inner.AllResources()
}

func (d *deprecateOverlay) AllDependencies() map[string]*resources.ProviderInfo {
	return d.inner.AllDependencies()
}

func newConf(s resources.ResourcesSchema) mqlc.CompilerConfig {
	features := mql.DefaultFeatures
	features = append(features, byte(mql.FailIfNoEntryPoints))
	return mqlc.NewConfig(s, features)
}

func TestDeprecatedSymbol_DeprecatedResource(t *testing.T) {
	overlay := &deprecateOverlay{
		inner:         schema,
		deprecatedRes: map[string]struct{}{"processes": {}},
	}

	q := &Mquery{
		Uid:         "test-deprecated-resource",
		Mql:         "processes.length >= 0",
		FileContext: FileContext{Line: 5, Column: 1},
	}

	entries := walkQueryForDeprecatedSymbols(overlay, newConf(overlay), "test.mql.yaml", q)
	require.Len(t, entries, 1)
	assert.Equal(t, QueryDeprecatedSymbolRuleID, entries[0].RuleID)
	assert.Equal(t, LevelWarning, entries[0].Level)
	assert.Contains(t, entries[0].Message, "test-deprecated-resource")
	assert.Contains(t, entries[0].Message, "processes")
}

func TestDeprecatedSymbol_DeprecatedField(t *testing.T) {
	overlay := &deprecateOverlay{
		inner: schema,
		deprecatedFields: map[string]map[string]struct{}{
			"file": {"basename": {}},
		},
	}

	q := &Mquery{
		Uid:         "test-deprecated-field",
		Mql:         "file('/etc/passwd').basename == 'passwd'",
		FileContext: FileContext{Line: 7, Column: 3},
	}

	entries := walkQueryForDeprecatedSymbols(overlay, newConf(overlay), "test.mql.yaml", q)
	require.Len(t, entries, 1)
	assert.Equal(t, QueryDeprecatedSymbolRuleID, entries[0].RuleID)
	assert.Contains(t, entries[0].Message, "file.basename")
}

func TestDeprecatedSymbol_NoDeprecation(t *testing.T) {
	overlay := &deprecateOverlay{inner: schema}

	q := &Mquery{
		Uid: "test-clean",
		Mql: "file('/etc/passwd').basename == 'passwd'",
	}

	entries := walkQueryForDeprecatedSymbols(overlay, newConf(overlay), "test.mql.yaml", q)
	assert.Empty(t, entries)
}

func TestDeprecatedSymbol_DedupesRepeatedReferences(t *testing.T) {
	overlay := &deprecateOverlay{
		inner: schema,
		deprecatedFields: map[string]map[string]struct{}{
			"file": {"basename": {}},
		},
	}

	q := &Mquery{
		Uid: "test-dedupe",
		Mql: "file('/a').basename == 'a' && file('/b').basename == 'b'",
	}

	entries := walkQueryForDeprecatedSymbols(overlay, newConf(overlay), "test.mql.yaml", q)
	require.Len(t, entries, 1, "duplicate references to the same deprecated field should collapse to a single warning")
}

func TestDeprecatedSymbol_QueryFiltersOnly(t *testing.T) {
	overlay := deprecatedFieldOverlay()

	// A variant parent carries filters but no mql of its own.
	b := &Bundle{
		Queries: []*Mquery{{
			Uid:         "test-filters-only",
			FileContext: FileContext{Line: 11, Column: 3},
			Filters:     deprecatedFilters(12),
		}},
	}

	entries := lintDeprecatedSymbols(overlay, newConf(overlay), "test.mql.yaml", b)
	require.Len(t, entries, 1)
	assert.Equal(t, FilterDeprecatedSymbolRuleID, entries[0].RuleID,
		"a filter is a filter regardless of who owns it")
	assert.Equal(t, LevelWarning, entries[0].Level)
	assert.Contains(t, entries[0].Message, "test-filters-only")
	assert.Contains(t, entries[0].Message, "file.basename")
	assert.Contains(t, entries[0].Message, "filters")
	assert.Equal(t, 12, entries[0].Location[0].Line,
		"a query's filters block has its own file context too, so point at it rather than the query")
}

func TestDeprecatedSymbol_QueryMqlAndFiltersUseDistinctRules(t *testing.T) {
	overlay := &deprecateOverlay{
		inner:         schema,
		deprecatedRes: map[string]struct{}{"processes": {}},
		deprecatedFields: map[string]map[string]struct{}{
			"file": {"basename": {}},
		},
	}

	b := &Bundle{
		Queries: []*Mquery{{
			Uid:         "test-both-sites",
			Mql:         "processes.length >= 0",
			FileContext: FileContext{Line: 5},
			Filters:     deprecatedFilters(6),
		}},
	}

	entries := lintDeprecatedSymbols(overlay, newConf(overlay), "test.mql.yaml", b)
	require.Len(t, entries, 2)

	byRule := map[string]*Entry{}
	for _, e := range entries {
		byRule[e.RuleID] = e
	}
	require.Contains(t, byRule, QueryDeprecatedSymbolRuleID)
	require.Contains(t, byRule, FilterDeprecatedSymbolRuleID)
	assert.Contains(t, byRule[QueryDeprecatedSymbolRuleID].Message, "processes")
	assert.Equal(t, 5, byRule[QueryDeprecatedSymbolRuleID].Location[0].Line)
	assert.Contains(t, byRule[FilterDeprecatedSymbolRuleID].Message, "file.basename")
	assert.Equal(t, 6, byRule[FilterDeprecatedSymbolRuleID].Location[0].Line)
}

func TestDeprecatedSymbol_SameSymbolInMqlAndFiltersReportedAtBothSites(t *testing.T) {
	overlay := deprecatedFieldOverlay()

	b := &Bundle{
		Queries: []*Mquery{{
			Uid:         "test-same-symbol-both-sites",
			Mql:         "file('/etc/shadow').basename == 'shadow'",
			FileContext: FileContext{Line: 5},
			Filters:     deprecatedFilters(6),
		}},
	}

	entries := lintDeprecatedSymbols(overlay, newConf(overlay), "test.mql.yaml", b)
	require.Len(t, entries, 2,
		"the mql and the filters are two separate edits, and now two separate lines, so both are reported")

	lines := map[int]string{}
	for _, e := range entries {
		lines[e.Location[0].Line] = e.RuleID
	}
	assert.Equal(t, QueryDeprecatedSymbolRuleID, lines[5])
	assert.Equal(t, FilterDeprecatedSymbolRuleID, lines[6])
}

func TestDeprecatedSymbol_QueryMultipleFilterItems(t *testing.T) {
	overlay := &deprecateOverlay{
		inner:         schema,
		deprecatedRes: map[string]struct{}{"processes": {}},
		deprecatedFields: map[string]map[string]struct{}{
			"file": {"basename": {}},
		},
	}

	// The list form of filters: yields one Mquery per entry, keyed by index.
	b := &Bundle{
		Queries: []*Mquery{{
			Uid: "test-multi-filter-items",
			Filters: &Filters{
				FileContext: FileContext{Line: 8},
				Items: map[string]*Mquery{
					"0": {Mql: "file('/etc/passwd').basename == 'passwd'"},
					"1": {Mql: "processes.length >= 0"},
				},
			},
		}},
	}

	entries := lintDeprecatedSymbols(overlay, newConf(overlay), "test.mql.yaml", b)
	require.Len(t, entries, 2, "every filter item should be analyzed, not just the first")
	for _, e := range entries {
		assert.Equal(t, FilterDeprecatedSymbolRuleID, e.RuleID)
	}
}

func TestDeprecatedSymbol_NoFiltersIsSafe(t *testing.T) {
	overlay := &deprecateOverlay{inner: schema}

	q := &Mquery{Uid: "test-nil-filters", Mql: "file('/etc/passwd').basename == 'passwd'"}
	assert.Empty(t, walkQueryForDeprecatedSymbols(overlay, newConf(overlay), "test.mql.yaml", q))

	empty := &Mquery{Uid: "test-empty-everything"}
	assert.Empty(t, walkQueryForDeprecatedSymbols(overlay, newConf(overlay), "test.mql.yaml", empty))
}

func TestDeprecatedSymbol_CompileFailureSilent(t *testing.T) {
	overlay := &deprecateOverlay{inner: schema}

	q := &Mquery{
		Uid: "test-broken",
		Mql: "this_does_not_compile(((",
	}

	entries := walkQueryForDeprecatedSymbols(overlay, newConf(overlay), "test.mql.yaml", q)
	assert.Empty(t, entries, "compile failures should be silently skipped — bundle-compile-error reports them")
}

// deprecatedFieldOverlay is the fixture shared by the container-filter tests:
// file.basename is deprecated, and this filter references it.
func deprecatedFieldOverlay() *deprecateOverlay {
	return &deprecateOverlay{
		inner: schema,
		deprecatedFields: map[string]map[string]struct{}{
			"file": {"basename": {}},
		},
	}
}

func deprecatedFilters(line int) *Filters {
	return &Filters{
		FileContext: FileContext{Line: line, Column: 5},
		Items: map[string]*Mquery{
			"": {Mql: "file('/etc/passwd').basename == 'passwd'"},
		},
	}
}

func TestDeprecatedSymbol_PolicyGroupFilters(t *testing.T) {
	overlay := deprecatedFieldOverlay()

	b := &Bundle{
		Policies: []*Policy{{
			Uid: "test-policy",
			Groups: []*PolicyGroup{{
				Title:   "Linux hosts",
				Filters: deprecatedFilters(42),
			}},
		}},
	}

	entries := lintDeprecatedSymbols(overlay, newConf(overlay), "test.mql.yaml", b)
	require.Len(t, entries, 1)
	assert.Equal(t, FilterDeprecatedSymbolRuleID, entries[0].RuleID,
		"a policy group is not a query, so it must not report under the query rule")
	assert.Equal(t, LevelWarning, entries[0].Level)
	assert.Contains(t, entries[0].Message, "test-policy")
	assert.Contains(t, entries[0].Message, "Linux hosts")
	assert.Contains(t, entries[0].Message, "file.basename")
	assert.Equal(t, 42, entries[0].Location[0].Line,
		"a group's filters block has its own file context, so point at it directly")
}

func TestDeprecatedSymbol_PolicyGroupFiltersWithoutTitle(t *testing.T) {
	overlay := deprecatedFieldOverlay()

	b := &Bundle{
		Policies: []*Policy{{
			Uid: "test-policy",
			Groups: []*PolicyGroup{
				{Title: "first group, no deprecation"},
				{Filters: deprecatedFilters(77)},
			},
		}},
	}

	entries := lintDeprecatedSymbols(overlay, newConf(overlay), "test.mql.yaml", b)
	require.Len(t, entries, 1)
	assert.Contains(t, entries[0].Message, "group 2",
		"an untitled group still needs to be identifiable, so fall back to its position")
}

func TestDeprecatedSymbol_QueryPackFilters(t *testing.T) {
	overlay := deprecatedFieldOverlay()

	b := &Bundle{
		Packs: []*QueryPack{{
			Uid:     "test-pack",
			Filters: deprecatedFilters(13),
		}},
	}

	entries := lintDeprecatedSymbols(overlay, newConf(overlay), "test.mql.yaml", b)
	require.Len(t, entries, 1)
	assert.Equal(t, FilterDeprecatedSymbolRuleID, entries[0].RuleID)
	assert.Contains(t, entries[0].Message, "test-pack")
	assert.Contains(t, entries[0].Message, "file.basename")
	assert.Equal(t, 13, entries[0].Location[0].Line)
}

func TestDeprecatedSymbol_QueryPackGroupFilters(t *testing.T) {
	overlay := deprecatedFieldOverlay()

	b := &Bundle{
		Packs: []*QueryPack{{
			Uid: "test-pack",
			Groups: []*QueryGroup{{
				Title:   "Inventory",
				Filters: deprecatedFilters(21),
			}},
		}},
	}

	entries := lintDeprecatedSymbols(overlay, newConf(overlay), "test.mql.yaml", b)
	require.Len(t, entries, 1)
	assert.Equal(t, FilterDeprecatedSymbolRuleID, entries[0].RuleID)
	assert.Contains(t, entries[0].Message, "test-pack")
	assert.Contains(t, entries[0].Message, "Inventory")
	assert.Equal(t, 21, entries[0].Location[0].Line)
}

func TestDeprecatedSymbol_ComputedFiltersNotReported(t *testing.T) {
	overlay := deprecatedFieldOverlay()

	// ComputedFilters is derived at load time from the group and query filters,
	// so reporting it would double up on warnings the author cannot act on.
	b := &Bundle{
		Policies: []*Policy{{
			Uid:             "test-policy",
			ComputedFilters: deprecatedFilters(9),
		}},
		Packs: []*QueryPack{{
			Uid:             "test-pack",
			ComputedFilters: deprecatedFilters(11),
		}},
	}

	entries := lintDeprecatedSymbols(overlay, newConf(overlay), "test.mql.yaml", b)
	assert.Empty(t, entries)
}

func TestDeprecatedSymbol_GroupFilterAndCheckReportedIndependently(t *testing.T) {
	overlay := &deprecateOverlay{
		inner:         schema,
		deprecatedRes: map[string]struct{}{"processes": {}},
		deprecatedFields: map[string]map[string]struct{}{
			"file": {"basename": {}},
		},
	}

	b := &Bundle{
		Policies: []*Policy{{
			Uid: "test-policy",
			Groups: []*PolicyGroup{{
				Title:   "Linux hosts",
				Filters: deprecatedFilters(42),
				Checks: []*Mquery{{
					Uid:         "test-check",
					Mql:         "processes.length >= 0",
					FileContext: FileContext{Line: 50},
				}},
			}},
		}},
	}

	entries := lintDeprecatedSymbols(overlay, newConf(overlay), "test.mql.yaml", b)
	require.Len(t, entries, 2, "the group's filters and the check's mql are separate sites")

	byRule := map[string]*Entry{}
	for _, e := range entries {
		byRule[e.RuleID] = e
	}
	require.Contains(t, byRule, FilterDeprecatedSymbolRuleID)
	require.Contains(t, byRule, QueryDeprecatedSymbolRuleID)
	assert.Contains(t, byRule[FilterDeprecatedSymbolRuleID].Message, "file.basename")
	assert.Contains(t, byRule[QueryDeprecatedSymbolRuleID].Message, "processes")
}

func TestDeprecatedSymbol_ContainerFiltersDedupeAndSkipEmpties(t *testing.T) {
	overlay := deprecatedFieldOverlay()

	b := &Bundle{
		Policies: []*Policy{{
			Uid: "test-policy",
			Groups: []*PolicyGroup{
				{Title: "no filters at all"},
				{Title: "empty filters", Filters: &Filters{}},
				{
					Title: "repeated symbol",
					Filters: &Filters{
						FileContext: FileContext{Line: 31},
						Items: map[string]*Mquery{
							"0": {Mql: "file('/a').basename == 'a'"},
							"1": {Mql: "file('/b').basename == 'b'"},
						},
					},
				},
			},
		}},
	}

	entries := lintDeprecatedSymbols(overlay, newConf(overlay), "test.mql.yaml", b)
	require.Len(t, entries, 1, "one deprecated symbol in one group's filters is one warning")
	assert.Equal(t, 31, entries[0].Location[0].Line)
}
