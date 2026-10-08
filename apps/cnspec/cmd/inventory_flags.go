// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// inventoryFlagNames are the inventory flags `scan` exposes that the shared
// loader (go.mondoo.com/mql/cli/inventoryloader) reads from viper. Commands
// that register them with addInventoryFlags and bind them with
// bindInventoryFlags load the inventory exactly the way `scan` does, through
// getCobraScanConfig or the shell's config, and resolve the assets'
// credentials through the inventory's credentials section and vault.
var inventoryFlagNames = []string{
	"inventory-file",
	"inventory-format-ansible",
	"inventory-format-domainlist",
}

// addInventoryFlags registers --inventory-file and the inventory format flags
// on cmd. fileUsage describes --inventory-file, since what a command does with
// several assets differs between commands.
func addInventoryFlags(cmd *cobra.Command, fileUsage string) {
	cmd.Flags().String("inventory-file", "", fileUsage)
	cmd.Flags().Bool("inventory-format-ansible", false, "Set the inventory format to Ansible")
	cmd.Flags().Bool("inventory-format-domainlist", false, "Set the inventory format to domain list")
}

// bindInventoryFlags binds the inventory flags to the viper keys the inventory
// loader reads. Call it from PreRun, like the other flag bindings, so that only
// the command that runs binds them.
func bindInventoryFlags(cmd *cobra.Command) {
	for _, name := range inventoryFlagNames {
		if f := cmd.Flags().Lookup(name); f != nil {
			_ = viper.BindPFlag(name, f)
		}
	}
}

// bomOutputFile returns the file the bill of materials at index i of n is
// written to when --output-target is target. A single document goes to target
// itself. Several documents, one per asset, go next to it with the index
// inserted before the extension: out.json becomes out-0.json, out-1.json and so
// on, in the order sortBomsByAssetName gives them.
func bomOutputFile(target string, i, n int) string {
	if n <= 1 {
		return target
	}
	ext := filepath.Ext(target)
	return fmt.Sprintf("%s-%d%s", strings.TrimSuffix(target, ext), i, ext)
}

// sortBomsByAssetName orders the per-asset documents by asset name. The
// generators build them from a map, so without sorting the index in each
// output file name would change from one run to the next.
func sortBomsByAssetName[T any](boms []T, assetName func(T) string) {
	slices.SortStableFunc(boms, func(a, b T) int {
		return strings.Compare(assetName(a), assetName(b))
	})
}
