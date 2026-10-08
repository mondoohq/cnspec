// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestBomOutputFile(t *testing.T) {
	dir := filepath.Join("some", "dir")
	cases := []struct {
		name   string
		target string
		i, n   int
		want   string
	}{
		{"single asset keeps the target", filepath.Join(dir, "sbom.json"), 0, 1, filepath.Join(dir, "sbom.json")},
		{"first of several", filepath.Join(dir, "sbom.json"), 0, 3, filepath.Join(dir, "sbom-0.json")},
		{"last of several", filepath.Join(dir, "sbom.json"), 2, 3, filepath.Join(dir, "sbom-2.json")},
		{"no extension", filepath.Join(dir, "sbom"), 1, 2, filepath.Join(dir, "sbom-1")},
		{"only the last extension moves", "bom.cdx.json", 1, 2, "bom.cdx-1.json"},
		{"relative file in the working directory", "out.xml", 0, 2, "out-0.xml"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, bomOutputFile(tc.target, tc.i, tc.n))
		})
	}
}

func TestSortBomsByAssetName(t *testing.T) {
	type bom struct{ name, id string }
	boms := []*bom{{"web", "1"}, {"db", "2"}, {"", "3"}, {"db", "4"}}

	sortBomsByAssetName(boms, func(b *bom) string { return b.name })

	ids := []string{}
	for _, b := range boms {
		ids = append(ids, b.id)
	}
	// Ordered by name; equal names keep the order they came in.
	assert.Equal(t, []string{"3", "2", "4", "1"}, ids)
}

func TestInventoryFlagsAreRegistered(t *testing.T) {
	commands := map[string]*cobra.Command{"sbom": sbomCmd, "aibom": aibomCmd, "shell": shellCmd}
	for name, cmd := range commands {
		t.Run(name, func(t *testing.T) {
			for _, flag := range inventoryFlagNames {
				assert.NotNil(t, cmd.Flags().Lookup(flag), "--%s", flag)
			}
		})
	}
}
