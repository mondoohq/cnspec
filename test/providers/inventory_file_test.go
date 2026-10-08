// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/test"
)

// These tests cover --inventory-file on `sbom`, `aibom` and `shell`: the asset
// and its credentials come from the inventory, the same way `scan` and `run`
// load them, instead of from command-line arguments.

// fakeInventoryPassword is a canary, not a credential. The tests assert it never
// shows up in the output and never print it themselves.
const fakeInventoryPassword = "canary-inventory-password-7f3c"

// inventoryCommands are the commands under test, each with the arguments that
// make it write a file per asset under dir.
func inventoryCommands(dir string) map[string][]string {
	return map[string][]string{
		"sbom":  {"sbom", "-o", "cnquery-json", "--output-target", filepath.Join(dir, "sbom.json")},
		"aibom": {"aibom", "-o", "json", "--output-target", filepath.Join(dir, "aibom.json")},
		"shell": {"shell"},
	}
}

func writeInventory(t *testing.T, dir, data string) string {
	t.Helper()
	path := filepath.Join(dir, "inventory.yml")
	require.NoError(t, os.WriteFile(path, []byte(data), 0o600))
	return path
}

// fakeLinuxRoot writes a minimal Linux filesystem the filesystem connection
// identifies by its hostname and machine ID, so two of them are two assets.
func fakeLinuxRoot(t *testing.T, hostname, machineID string) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), hostname)
	etc := filepath.Join(root, "etc")
	require.NoError(t, os.MkdirAll(etc, 0o755))
	files := map[string]string{
		"os-release": "NAME=\"Alpine Linux\"\nID=alpine\nVERSION_ID=3.19.0\nPRETTY_NAME=\"Alpine Linux v3.19\"\n",
		"hostname":   hostname + "\n",
		"machine-id": machineID + "\n",
	}
	for name, content := range files {
		require.NoError(t, os.WriteFile(filepath.Join(etc, name), []byte(content), 0o600))
	}
	return root
}

func runCnspec(t *testing.T, args ...string) test.Runner {
	t.Helper()
	r := test.NewCliTestRunner("./cnspec", args...)
	require.NoError(t, r.Run())
	return r
}

func output(r test.Runner) string {
	return string(r.Stdout()) + string(r.Stderr())
}

func TestInventoryFile_LocalAsset(t *testing.T) {
	once.Do(setup)

	dir := t.TempDir()
	inv := writeInventory(t, dir, `apiVersion: v1
kind: Inventory
metadata:
  name: local
spec:
  assets:
    - name: inventory-local
      connections:
        - type: local
`)

	t.Run("sbom", func(t *testing.T) {
		r := runCnspec(t, append(inventoryCommands(dir)["sbom"], "--inventory-file", inv)...)
		require.Equal(t, 0, r.ExitCode(), output(r))
		assert.Contains(t, string(r.Stderr()), "load inventory")

		data, err := os.ReadFile(filepath.Join(dir, "sbom.json"))
		require.NoError(t, err, "a single asset is written to --output-target itself")
		var doc map[string]any
		require.NoError(t, json.Unmarshal(data, &doc))
		assert.Contains(t, doc, "asset")
	})

	t.Run("aibom", func(t *testing.T) {
		r := runCnspec(t, append(inventoryCommands(dir)["aibom"], "--inventory-file", inv)...)
		require.Equal(t, 0, r.ExitCode(), output(r))

		data, err := os.ReadFile(filepath.Join(dir, "aibom.json"))
		require.NoError(t, err, "a single asset is written to --output-target itself")
		var doc map[string]any
		require.NoError(t, json.Unmarshal(data, &doc))
		assert.Contains(t, doc, "asset")
	})

	t.Run("shell", func(t *testing.T) {
		// The test has no terminal, so the shell stops right after it has
		// connected to the inventory's asset.
		r := runCnspec(t, "shell", "--inventory-file", inv)
		assert.Equal(t, 1, r.ExitCode())
		assert.Contains(t, string(r.Stderr()), "connected to")
		assert.Contains(t, string(r.Stderr()), "interactive terminal")
	})
}

func TestInventoryFile_SSHPasswordFromInventory(t *testing.T) {
	once.Do(setup)

	dir := t.TempDir()
	// Port 1 on the loopback address refuses the connection, so each command
	// gets as far as connecting to the inventory's host with the inventory's
	// credential, and no further.
	inv := writeInventory(t, dir, `apiVersion: v1
kind: Inventory
metadata:
  name: ssh
spec:
  assets:
    - name: inventory-ssh
      connections:
        - type: ssh
          host: 127.0.0.1
          port: 1
          credentials:
            - secret_id: ssh-password
  credentials:
    ssh-password:
      type: password
      user: tester
      password: `+fakeInventoryPassword+`
`)

	for name, args := range inventoryCommands(dir) {
		t.Run(name, func(t *testing.T) {
			r := runCnspec(t, append(args, "--inventory-file", inv)...)

			// The asset cannot be reached, so no bill of materials is written
			// and no shell opens: every command fails.
			assert.Equal(t, 1, r.ExitCode())
			assert.Contains(t, string(r.Stderr()), "127.0.0.1:1",
				"the command should try to connect to the host the inventory names")
			assert.False(t, strings.Contains(output(r), fakeInventoryPassword),
				"the inventory's password must not appear in the output")
		})
	}
}

func TestInventoryFile_MultipleAssets(t *testing.T) {
	once.Do(setup)

	dir := t.TempDir()
	inv := writeInventory(t, dir, `apiVersion: v1
kind: Inventory
metadata:
  name: multi
spec:
  assets:
    - name: host-b
      connections:
        - type: filesystem
          path: `+fakeLinuxRoot(t, "host-b", "22222222222222222222222222222222")+`
    - name: host-a
      connections:
        - type: filesystem
          path: `+fakeLinuxRoot(t, "host-a", "11111111111111111111111111111111")+`
`)
	commands := inventoryCommands(dir)

	// sbom and aibom write one document per asset, next to --output-target,
	// with the index inserted before the extension, in asset name order.
	for _, name := range []string{"sbom", "aibom"} {
		t.Run(name, func(t *testing.T) {
			r := runCnspec(t, append(commands[name], "--inventory-file", inv)...)
			require.Equal(t, 0, r.ExitCode(), output(r))

			_, err := os.Stat(filepath.Join(dir, name+".json"))
			assert.True(t, os.IsNotExist(err), "with several assets, --output-target itself is not written")

			for i, host := range []string{"host-a", "host-b"} {
				data, err := os.ReadFile(filepath.Join(dir, name+"-"+strconv.Itoa(i)+".json"))
				require.NoError(t, err)
				assert.Contains(t, string(data), host)
			}
		})
	}

	t.Run("shell", func(t *testing.T) {
		r := runCnspec(t, append(commands["shell"], "--inventory-file", inv)...)
		assert.Equal(t, 1, r.ExitCode())
		assert.Contains(t, string(r.Stderr()), "exactly one asset")
		assert.Contains(t, string(r.Stderr()), "defines 2")
	})
}

func TestInventoryFile_InvalidInventory(t *testing.T) {
	once.Do(setup)

	dir := t.TempDir()
	inv := writeInventory(t, dir, "spec: [this is not an inventory\n")

	for name, args := range inventoryCommands(dir) {
		t.Run(name, func(t *testing.T) {
			r := runCnspec(t, append(args, "--inventory-file", inv)...)
			assert.Equal(t, 1, r.ExitCode())
			assert.Contains(t, string(r.Stderr()), "could not parse inventory")
		})
	}
}
