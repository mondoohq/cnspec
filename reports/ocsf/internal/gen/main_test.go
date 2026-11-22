// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSchemaOrderIsBySemver pins the order the schemas are walked in.
//
// lookup takes the caption and documentation of the first schema that has an
// attribute, so this order decides which version's wording every generated field
// carries -- and the whole file is regenerated from it. Sorting the filenames as
// strings puts "schema-1.10.0.json.gz" ahead of "schema-1.3.0.json.gz", so
// adding a 1.10 schema would make the newest version the source of truth for the
// oldest, rewriting every doc comment in types.gen.go with no line of the
// generator having changed.
func TestSchemaOrderIsBySemver(t *testing.T) {
	paths := []string{
		"schemas/schema-1.10.0.json.gz",
		"schemas/schema-1.3.0.json.gz",
		"schemas/schema-1.9.0.json.gz",
		"schemas/schema-2.0.0.json.gz",
	}
	sortSchemaPaths(paths)

	assert.Equal(t, []string{
		"schemas/schema-1.3.0.json.gz",
		"schemas/schema-1.9.0.json.gz",
		"schemas/schema-1.10.0.json.gz",
		"schemas/schema-2.0.0.json.gz",
	}, paths, "oldest first, by version number rather than by filename")
}

func TestCompareVersions(t *testing.T) {
	assert.Negative(t, compareVersions("1.9.0", "1.10.0"), "9 is before 10")
	assert.Positive(t, compareVersions("1.10.0", "1.9.0"))
	assert.Equal(t, 0, compareVersions("1.3.0", "1.3.0"))
	assert.Negative(t, compareVersions("1.3", "1.3.1"), "a shorter version is the earlier one")
	assert.Negative(t, compareVersions("1.3.0", "1.3.0-rc1"),
		"a non-numeric component sorts after the numbers, so the order stays total")
}

func TestSchemaVersion(t *testing.T) {
	assert.Equal(t, "1.9.0", schemaVersion("/x/schemas/schema-1.9.0.json.gz"))
}

// selfReferencingSchema has one object whose parent attribute is the object
// itself, the shape of OCSF's process.parent_process.
func selfReferencingSchema() []schema {
	return []schema{{
		Version: "1.3.0",
		Objects: map[string]entity{
			"node": {Caption: "Node", Attributes: map[string]attr{
				"name":   {Caption: "Name", Type: "string_t"},
				"parent": {Caption: "Parent", Type: "object_t", ObjectType: "node"},
			}},
		},
	}}
}

// TestVariantCutsSelfReference: a variant types the self-referencing attribute
// with a reduced copy of the object, so the graph ends one level down and the
// Parquet schema derivation terminates.
func TestVariantCutsSelfReference(t *testing.T) {
	g := &generator{
		spec: spec{
			Objects:  map[string][]string{"node": {"name", "parent"}},
			Variants: map[string]variantSpec{"parent": {Go: "ParentNode", Object: "node", Attributes: []string{"name"}}},
		},
		schemas: selfReferencingSchema(),
	}
	out, err := g.types()
	require.NoError(t, err)
	src := string(out)

	assert.Contains(t, src, "type ParentNode struct")
	assert.Regexp(t, `Parent \*ParentNode `, src, "the attribute takes the variant, not the object")
	variant := src[strings.Index(src, "type ParentNode struct"):]
	assert.NotContains(t, variant[:strings.Index(variant, "}")], "Parent ", "the variant has no parent of its own")
}

func TestVariantMayNotListItself(t *testing.T) {
	g := &generator{
		spec: spec{
			Objects:  map[string][]string{"node": {"name", "parent"}},
			Variants: map[string]variantSpec{"parent": {Go: "ParentNode", Object: "node", Attributes: []string{"name", "parent"}}},
		},
		schemas: selfReferencingSchema(),
	}
	_, err := g.types()
	assert.ErrorContains(t, err, "lists itself")
}

func enumSchema() []schema {
	return []schema{{
		Version: "1.3.0",
		Objects: map[string]entity{
			"node": {Caption: "Node", Attributes: map[string]attr{
				"level_id": {Caption: "Level ID", Type: "integer_t", Requirement: "recommended", Enum: map[string]enumValue{
					"0": {Caption: "Unknown"}, "1": {Caption: "Low"},
				}},
				"kind_id": {Caption: "Kind ID", Type: "integer_t", Requirement: "recommended", Enum: map[string]enumValue{
					"0": {Caption: "Info"}, "1": {Caption: "Warning"},
				}},
				"rank_id": {Caption: "Rank ID", Type: "integer_t", Requirement: "required", Enum: map[string]enumValue{
					"0": {Caption: "Unknown"}, "1": {Caption: "First"},
				}},
				"name": {Caption: "Name", Type: "string_t"},
			}},
		},
	}}
}

// TestOmitUnknown: an enum listed in omit_unknown takes omitempty, every other
// enum id keeps writing its zero.
func TestOmitUnknown(t *testing.T) {
	g := &generator{
		spec: spec{
			Objects:     map[string][]string{"node": {"level_id", "kind_id"}},
			OmitUnknown: []string{"level_id"},
		},
		schemas: enumSchema(),
	}
	out, err := g.types()
	require.NoError(t, err)
	src := string(out)
	assert.Contains(t, src, `json:"level_id,omitempty"`)
	assert.Contains(t, src, `json:"kind_id"`)
}

func TestOmitUnknownRejects(t *testing.T) {
	for attribute, want := range map[string]string{
		"kind_id": "zero value is not Unknown",
		"rank_id": "not an optional enum",
		"name":    "not an optional enum",
	} {
		t.Run(attribute, func(t *testing.T) {
			g := &generator{
				spec: spec{
					Objects:     map[string][]string{"node": {attribute}},
					OmitUnknown: []string{attribute},
				},
				schemas: enumSchema(),
			}
			_, err := g.types()
			assert.ErrorContains(t, err, want)
		})
	}
}
