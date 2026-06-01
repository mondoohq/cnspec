// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"os"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/logger"
)

// The support bundle (and its tar.gz packaging) is intentionally a scan-only
// feature. serve is a long-running daemon and must never collect one. These
// flags are registered and read only on scanCmd; guard against a future change
// accidentally wiring them into serve.
func TestServeCmd_HasNoSupportBundleFlags(t *testing.T) {
	assert.Nil(t, serveCmd.Flags().Lookup("collect-support-bundle"),
		"serve must not expose --collect-support-bundle; the support bundle is scan-only")
	assert.Nil(t, serveCmd.Flags().Lookup("support-bundle-dir"),
		"serve must not expose --support-bundle-dir; the support bundle is scan-only")
}

// disableServeDebugArtifacts must stop cnquery's logger dump helpers from
// writing mondoo-debug-* files even when debug logging is enabled via the
// DEBUG env var — the scenario a serve operator hits when troubleshooting.
func TestDisableServeDebugArtifacts_SuppressesDumps(t *testing.T) {
	prevLevel := zerolog.GlobalLevel()
	t.Cleanup(func() { zerolog.SetGlobalLevel(prevLevel) })
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	t.Setenv("DEBUG", "1")
	t.Setenv("TRACE", "1")

	// DumpLocal must be empty so the helpers take the env-driven default path
	// (a non-empty DumpLocal writes unconditionally and is not what serve hits).
	prevDump := logger.DumpLocal
	logger.DumpLocal = ""
	t.Cleanup(func() { logger.DumpLocal = prevDump })

	// Run in a temp dir so a stray "./mondoo-debug-*" write would be observable.
	tmp := t.TempDir()
	prevWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmp))
	t.Cleanup(func() { _ = os.Chdir(prevWd) })

	disableServeDebugArtifacts()

	assert.Empty(t, os.Getenv("DEBUG"), "DEBUG must be cleared so dump helpers no-op")
	assert.Empty(t, os.Getenv("TRACE"), "TRACE must be cleared so dump helpers no-op")

	// With the env cleared and DumpLocal empty, this hits the no-op path.
	logger.DebugDumpJSON("inventory-unresolved", map[string]string{"k": "v"})
	logger.DebugDumpYAML("assetBundle", map[string]string{"k": "v"})

	entries, err := os.ReadDir(tmp)
	require.NoError(t, err)
	assert.Empty(t, entries, "no mondoo-debug-* file should be written in serve mode")
}
