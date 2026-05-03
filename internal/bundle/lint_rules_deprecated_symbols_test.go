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

func TestDeprecatedSymbol_CompileFailureSilent(t *testing.T) {
	overlay := &deprecateOverlay{inner: schema}

	q := &Mquery{
		Uid: "test-broken",
		Mql: "this_does_not_compile(((",
	}

	entries := walkQueryForDeprecatedSymbols(overlay, newConf(overlay), "test.mql.yaml", q)
	assert.Empty(t, entries, "compile failures should be silently skipped — bundle-compile-error reports them")
}
