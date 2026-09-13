// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	cnspec_config "go.mondoo.com/cnspec/v13/apps/cnspec/cmd/config"
	"go.mondoo.com/mql/v13/cli/execruntime"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
)

// testRuntimeEnv stands in for the runtimeEnv every caller detects once at
// startup and passes into loadInventory/reloadInventory. Its value doesn't
// matter for these tests (none rely on CI/CD auto-detection; the CI/CD test
// forces the branch with Category: "cicd" instead), only that a non-nil
// *execruntime.RuntimeEnv is supplied, matching the real call sites.
var testRuntimeEnv = execruntime.Detect()

// writeInventoryFile writes a minimal, valid inventory YAML naming a single
// local asset, and returns its path.
func writeInventoryFile(t *testing.T, dir string, assetName string) string {
	t.Helper()
	path := filepath.Join(dir, "inventory.yml")
	content := "apiVersion: v1\n" +
		"kind: Inventory\n" +
		"metadata:\n" +
		"  name: test-inventory\n" +
		"spec:\n" +
		"  assets:\n" +
		"    - name: " + assetName + "\n" +
		"      connections:\n" +
		"        - type: local\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

// withInventoryFile points the "inventory-file" viper key (the same global
// key inventoryloader.Parse reads) at path for the duration of the test.
//
// viper's global state has no synchronization of its own, so this and every
// test in this file must NOT call t.Parallel(): concurrent tests setting
// "inventory-file" to different paths would race. If a future test in this
// file needs to run in parallel, replace this viper.Set with a fake
// inventory source injected through a parameter instead of the global key.
func withInventoryFile(t *testing.T, path string) {
	t.Helper()
	previous := viper.GetString("inventory-file")
	viper.Set("inventory-file", path)
	t.Cleanup(func() {
		viper.Set("inventory-file", previous)
	})
}

func TestLoadInventory_ReloadsFromDiskOnEachCall(t *testing.T) {
	dir := t.TempDir()
	path := writeInventoryFile(t, dir, "v1")
	withInventoryFile(t, path)

	opts := &cnspec_config.CliConfig{}

	inv, err := loadInventory(opts, testRuntimeEnv, true)
	require.NoError(t, err)
	require.Len(t, inv.Spec.Assets, 1)
	assert.Equal(t, "v1", inv.Spec.Assets[0].Name,
		"first load should reflect what's on disk")

	// Overwrite the same file, as if an operator edited inventory.yml while
	// the service kept running. If loadInventory cached anything, this
	// second call would still return "v1".
	writeInventoryFile(t, dir, "v2")

	inv, err = loadInventory(opts, testRuntimeEnv, true)
	require.NoError(t, err)
	require.Len(t, inv.Spec.Assets, 1)
	assert.Equal(t, "v2", inv.Spec.Assets[0].Name,
		"loadInventory must re-read the file from disk on every call, not cache the first parse")
}

func TestLoadInventory_InvalidYAMLReturnsErrorInsteadOfPanicking(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "inventory.yml")
	require.NoError(t, os.WriteFile(path, []byte("not: valid: yaml: at: all: ["), 0o600))
	withInventoryFile(t, path)

	opts := &cnspec_config.CliConfig{}

	_, err := loadInventory(opts, testRuntimeEnv, true)
	assert.Error(t, err,
		"a malformed inventory file (e.g. mid-edit) must surface as an error so the caller can keep the last-known-good inventory, not crash the service")
}

// TestLoadInventory_CICDCategoryDoesNotPanic is a regression test: the
// pre-fix code called conf.Inventory.ApplyLabels/ApplyCategory on the
// scanConfig's Inventory field *before* it was ever assigned by
// inventoryloader.ParseOrUse, so it was always a nil *inventory.Inventory.
// Both methods dereference the receiver's Spec field without a nil check,
// so any config that set Category: "cicd" (or matched CI/CD auto-detection)
// crashed cnspec serve with a nil pointer dereference. loadInventory fixes
// this by parsing the inventory before applying CI/CD labels/category.
func TestLoadInventory_CICDCategoryDoesNotPanic(t *testing.T) {
	dir := t.TempDir()
	path := writeInventoryFile(t, dir, "ci-asset")
	withInventoryFile(t, path)

	opts := &cnspec_config.CliConfig{
		Category: "cicd",
	}

	inv, err := loadInventory(opts, testRuntimeEnv, true)
	require.NoError(t, err)
	require.Len(t, inv.Spec.Assets, 1)
	assert.Equal(t, inventory.AssetCategory_CATEGORY_CICD, inv.Spec.Assets[0].Category,
		"Category: \"cicd\" should still apply the CI/CD category once the nil-pointer order bug is fixed")
}

func TestReloadInventory_PicksUpChangesFromDisk(t *testing.T) {
	dir := t.TempDir()
	path := writeInventoryFile(t, dir, "v1")
	withInventoryFile(t, path)

	cliConfig := &cnspec_config.CliConfig{}
	scanConf := &scanConfig{Inventory: &inventory.Inventory{}}

	reloadInventory(cliConfig, scanConf, testRuntimeEnv, true)
	require.Len(t, scanConf.Inventory.Spec.Assets, 1)
	assert.Equal(t, "v1", scanConf.Inventory.Spec.Assets[0].Name)

	writeInventoryFile(t, dir, "v2")

	reloadInventory(cliConfig, scanConf, testRuntimeEnv, true)
	require.Len(t, scanConf.Inventory.Spec.Assets, 1)
	assert.Equal(t, "v2", scanConf.Inventory.Spec.Assets[0].Name,
		"reloadInventory must replace scanConf.Inventory with what's on disk now")
}

func TestReloadInventory_KeepsLastKnownGoodOnParseError(t *testing.T) {
	dir := t.TempDir()
	path := writeInventoryFile(t, dir, "good")
	withInventoryFile(t, path)

	cliConfig := &cnspec_config.CliConfig{}
	scanConf := &scanConfig{Inventory: &inventory.Inventory{}}

	reloadInventory(cliConfig, scanConf, testRuntimeEnv, true)
	require.Len(t, scanConf.Inventory.Spec.Assets, 1)

	goodInventory := scanConf.Inventory

	// Simulate the file being mid-edit: momentarily invalid YAML.
	require.NoError(t, os.WriteFile(path, []byte("not: valid: yaml: at: all: ["), 0o600))

	reloadInventory(cliConfig, scanConf, testRuntimeEnv, true)
	assert.Same(t, goodInventory, scanConf.Inventory,
		"a reload that fails to parse must not replace the last-known-good inventory")
}

// TestReloadInventory_SkipsReloadForStdinInventory guards against a
// regression: stdin can only be read once. inventoryloader.LoadDataFromPipe
// drains os.Stdin on the first read; a second read returns no data, and
// inventoryloader.ParseOrUse would then silently fall back to the
// local-only default asset, dropping every asset the piped inventory
// defined. reloadInventory must leave scanConf.Inventory untouched when the
// inventory came from stdin.
func TestReloadInventory_SkipsReloadForStdinInventory(t *testing.T) {
	withInventoryFile(t, "-")

	cliConfig := &cnspec_config.CliConfig{}
	original := &inventory.Inventory{
		Spec: &inventory.InventorySpec{
			Assets: []*inventory.Asset{{Name: "from-stdin"}},
		},
	}
	scanConf := &scanConfig{Inventory: original}

	reloadInventory(cliConfig, scanConf, testRuntimeEnv, true)

	assert.Same(t, original, scanConf.Inventory,
		"reloadInventory must not attempt to re-read stdin; doing so would silently drop the piped inventory's assets")
}

// withConfigFile points viper's "config file used" at a mondoo.yml inside
// dir for the duration of the test. That is the path discoverInventoryFile
// resolves the inventory next to, so it stands in for a real
// `--config /etc/opt/mondoo/mondoo.yml`.
//
// Like withInventoryFile, this mutates viper's global state, so tests using
// it must NOT call t.Parallel().
func withConfigFile(t *testing.T, dir string) string {
	t.Helper()
	previous := viper.ConfigFileUsed()
	path := filepath.Join(dir, "mondoo.yml")
	viper.SetConfigFile(path)
	t.Cleanup(func() {
		viper.SetConfigFile(previous)
	})
	return path
}

// TestLoadInventory_PicksUpInventoryAddedAfterStartup is the regression test
// for the workflow issue #3527 describes: extending an already-running
// service to cover an extra asset by dropping an inventory.yml next to the
// existing mondoo.yml.
//
// Discovery only reports a path when the file already exists, so probing
// once at startup leaves the "inventory-file" viper key empty for the life
// of a service that started before the inventory was written. Reloading the
// inventory on every cycle does not help on its own: with an empty key
// inventoryloader.Parse returns an empty inventory without reading any file,
// and every cycle re-derives the same local-only fallback. Discovery has to
// re-run per cycle too.
func TestLoadInventory_PicksUpInventoryAddedAfterStartup(t *testing.T) {
	dir := t.TempDir()
	withConfigFile(t, dir)
	// Startup state: no inventory.yml next to mondoo.yml yet, and the
	// operator passed no --inventory-file.
	withInventoryFile(t, "")

	opts := &cnspec_config.CliConfig{}

	inv, err := loadInventory(opts, testRuntimeEnv, false)
	require.NoError(t, err)
	require.Len(t, inv.Spec.Assets, 1)
	require.Len(t, inv.Spec.Assets[0].Connections, 1)
	assert.Equal(t, "local", inv.Spec.Assets[0].Connections[0].Type,
		"with no inventory on disk, serve scans the local asset")

	// The operator now drops an inventory.yml next to mondoo.yml, without
	// restarting the service.
	writeInventoryFile(t, dir, "added-after-startup")

	inv, err = loadInventory(opts, testRuntimeEnv, false)
	require.NoError(t, err)
	require.Len(t, inv.Spec.Assets, 1)
	assert.Equal(t, "added-after-startup", inv.Spec.Assets[0].Name,
		"an inventory.yml added after startup must be discovered on the next scan cycle, not only on restart")
}

// TestLoadInventory_ExplicitInventoryFileWinsOverDiscovery guards the other
// half of the contract: when the operator pinned a source with
// --inventory-file, per-cycle discovery must not overwrite it with whatever
// happens to sit next to mondoo.yml.
func TestLoadInventory_ExplicitInventoryFileWinsOverDiscovery(t *testing.T) {
	configDir := t.TempDir()
	withConfigFile(t, configDir)
	writeInventoryFile(t, configDir, "discovered")

	explicitDir := t.TempDir()
	explicitPath := writeInventoryFile(t, explicitDir, "explicit")
	withInventoryFile(t, explicitPath)

	opts := &cnspec_config.CliConfig{}

	inv, err := loadInventory(opts, testRuntimeEnv, true)
	require.NoError(t, err)
	require.Len(t, inv.Spec.Assets, 1)
	assert.Equal(t, "explicit", inv.Spec.Assets[0].Name,
		"--inventory-file must stay authoritative; discovery must not replace it")
	assert.Equal(t, explicitPath, viper.GetString("inventory-file"),
		"discovery must leave an operator-supplied inventory path untouched")
}

// TestLoadInventory_FallsBackToLocalWhenInventoryIsRemoved covers the
// reverse transition. Without clearing the discovered path, a removed
// inventory.yml would fail to parse every cycle and reloadInventory would
// keep the last-known-good inventory, so serve would go on scanning targets
// the operator deleted.
func TestLoadInventory_FallsBackToLocalWhenInventoryIsRemoved(t *testing.T) {
	dir := t.TempDir()
	withConfigFile(t, dir)
	withInventoryFile(t, "")
	path := writeInventoryFile(t, dir, "will-be-removed")

	opts := &cnspec_config.CliConfig{}

	inv, err := loadInventory(opts, testRuntimeEnv, false)
	require.NoError(t, err)
	require.Len(t, inv.Spec.Assets, 1)
	require.Equal(t, "will-be-removed", inv.Spec.Assets[0].Name)

	require.NoError(t, os.Remove(path))

	inv, err = loadInventory(opts, testRuntimeEnv, false)
	require.NoError(t, err)
	require.Len(t, inv.Spec.Assets, 1)
	require.Len(t, inv.Spec.Assets[0].Connections, 1)
	assert.Equal(t, "local", inv.Spec.Assets[0].Connections[0].Type,
		"removing the inventory must fall back to the local asset, not keep scanning the removed targets")
}

// TestReloadInventory_PicksUpInventoryAddedAfterStartup exercises the same
// regression through the function the scan loop actually calls.
func TestReloadInventory_PicksUpInventoryAddedAfterStartup(t *testing.T) {
	dir := t.TempDir()
	withConfigFile(t, dir)
	withInventoryFile(t, "")

	cliConfig := &cnspec_config.CliConfig{}
	scanConf := &scanConfig{Inventory: &inventory.Inventory{}}

	reloadInventory(cliConfig, scanConf, testRuntimeEnv, false)
	require.Len(t, scanConf.Inventory.Spec.Assets, 1)
	require.Equal(t, "local", scanConf.Inventory.Spec.Assets[0].Connections[0].Type)

	writeInventoryFile(t, dir, "added-after-startup")

	reloadInventory(cliConfig, scanConf, testRuntimeEnv, false)
	require.Len(t, scanConf.Inventory.Spec.Assets, 1)
	assert.Equal(t, "added-after-startup", scanConf.Inventory.Spec.Assets[0].Name,
		"the scan loop must see an inventory.yml added after the service started")
}
