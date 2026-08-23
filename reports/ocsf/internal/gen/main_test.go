// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
