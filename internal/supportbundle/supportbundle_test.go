// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package supportbundle

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/logger"
)

// chdir switches into dir for the duration of the test and restores cwd after.
// We need this because Bundle.sweepCWD reads files from the working directory
// the bundle was created in, and we want isolation between tests.
func chdir(t *testing.T, dir string) {
	t.Helper()
	prev, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(prev) })
}

// resetGlobals saves and restores the package-level state that Activate
// mutates (logger.DumpLocal, log.Logger, zerolog.GlobalLevel) so concurrent
// or sequential tests don't interfere with each other.
func resetGlobals(t *testing.T) {
	t.Helper()
	prevDump := logger.DumpLocal
	prevLogger := log.Logger
	prevLevel := zerolog.GlobalLevel()
	t.Cleanup(func() {
		logger.DumpLocal = prevDump
		log.Logger = prevLogger
		zerolog.SetGlobalLevel(prevLevel)
	})
}

func TestNew_DefaultPath_CreatesTimestampedDir(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	b, err := New("")
	require.NoError(t, err)

	info, err := os.Stat(b.Dir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
	assert.True(t, strings.HasPrefix(filepath.Base(b.Dir), "cnspec-support-bundle-"),
		"default path should include the cnspec-support-bundle prefix; got %q", b.Dir)
}

func TestNew_ExplicitPath_IsUsedVerbatim(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "my-bundle")

	b, err := New(target)
	require.NoError(t, err)
	assert.Equal(t, target, b.Dir)

	info, err := os.Stat(target)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestActivate_SetsDumpLocalAndForcesDebugLevel(t *testing.T) {
	resetGlobals(t)
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	logger.DumpLocal = ""

	dir := t.TempDir()
	b, err := New(filepath.Join(dir, "bundle"))
	require.NoError(t, err)
	require.NoError(t, b.Activate())
	defer b.Finalize()

	assert.Equal(t, zerolog.DebugLevel, zerolog.GlobalLevel(),
		"Activate must force debug level so DebugDumpJSON/YAML helpers fire")
	assert.Equal(t, filepath.Join(b.Dir, "mondoo-debug-"), logger.DumpLocal,
		"DumpLocal should be set to the bundle prefix so existing dump calls land in the bundle dir")
}

func TestActivate_DoesNotLowerVerbosityIfTraceAlready(t *testing.T) {
	resetGlobals(t)
	zerolog.SetGlobalLevel(zerolog.TraceLevel)
	logger.DumpLocal = ""

	dir := t.TempDir()
	b, err := New(filepath.Join(dir, "bundle"))
	require.NoError(t, err)
	require.NoError(t, b.Activate())
	defer b.Finalize()

	assert.Equal(t, zerolog.TraceLevel, zerolog.GlobalLevel(),
		"Activate must not downgrade from trace to debug")
}

func TestActivate_TeesLogsToFileWithTimestamps(t *testing.T) {
	resetGlobals(t)

	dir := t.TempDir()
	b, err := New(filepath.Join(dir, "bundle"))
	require.NoError(t, err)
	require.NoError(t, b.Activate())

	log.Debug().Str("k", "v").Msg("hello support bundle")

	require.NoError(t, b.Finalize())

	raw, err := os.ReadFile(filepath.Join(b.Dir, "debug.log"))
	require.NoError(t, err)
	contents := string(raw)
	assert.Contains(t, contents, "hello support bundle")
	assert.Contains(t, contents, "k=v")
	// RFC3339Nano timestamps always include "T" and one of "Z" or a tz offset.
	assert.Contains(t, contents, "T", "expected an RFC3339Nano timestamp in the log file")
}

func TestFinalize_RestoresGlobalsAndIsIdempotent(t *testing.T) {
	resetGlobals(t)
	zerolog.SetGlobalLevel(zerolog.WarnLevel)
	logger.DumpLocal = "/preset-prefix-"
	originalLogger := log.Logger

	dir := t.TempDir()
	b, err := New(filepath.Join(dir, "bundle"))
	require.NoError(t, err)
	require.NoError(t, b.Activate())

	// Mutate globals through Activate, then verify Finalize restores them.
	require.NoError(t, b.Finalize())
	assert.Equal(t, "/preset-prefix-", logger.DumpLocal, "DumpLocal must be restored")
	assert.Equal(t, zerolog.WarnLevel, zerolog.GlobalLevel(), "global level must be restored")
	assert.Equal(t, originalLogger, log.Logger, "log.Logger must be restored")

	// Second call is a no-op and must not panic.
	require.NoError(t, b.Finalize())
}

func TestFinalize_WritesManifestAndProviders(t *testing.T) {
	resetGlobals(t)

	dir := t.TempDir()
	b, err := New(filepath.Join(dir, "bundle"))
	require.NoError(t, err)
	b.Args = []string{"cnspec", "scan", "local", "--collect-support-bundle"}
	require.NoError(t, b.Activate())
	require.NoError(t, b.Finalize())

	// Manifest contents
	raw, err := os.ReadFile(filepath.Join(b.Dir, "manifest.json"))
	require.NoError(t, err)
	var m Manifest
	require.NoError(t, json.Unmarshal(raw, &m))
	assert.NotEmpty(t, m.CnspecInfo)
	assert.NotEmpty(t, m.CnquerySDK)
	assert.NotEmpty(t, m.GoVersion)
	assert.NotEmpty(t, m.OS)
	assert.NotEmpty(t, m.Arch)
	assert.Equal(t, []string{"cnspec", "scan", "local", "--collect-support-bundle"}, m.Args)
	assert.False(t, m.CreatedAt.IsZero())

	// Providers file is present even when provider listing returns nothing.
	pRaw, err := os.ReadFile(filepath.Join(b.Dir, "providers.json"))
	require.NoError(t, err)
	assert.NotEmpty(t, pRaw, "providers.json should be written even when no providers configured")
}

func TestFinalize_DoesNotLeakCredentialEnvVars(t *testing.T) {
	resetGlobals(t)

	t.Setenv("AWS_SECRET_ACCESS_KEY", "should-not-leak")
	t.Setenv("DEBUG", "1")

	dir := t.TempDir()
	b, err := New(filepath.Join(dir, "bundle"))
	require.NoError(t, err)
	require.NoError(t, b.Activate())
	require.NoError(t, b.Finalize())

	raw, err := os.ReadFile(filepath.Join(b.Dir, "manifest.json"))
	require.NoError(t, err)
	contents := string(raw)
	assert.NotContains(t, contents, "AWS_SECRET_ACCESS_KEY",
		"manifest must NOT dump arbitrary env vars — only the curated list")
	assert.NotContains(t, contents, "should-not-leak")
	assert.Contains(t, contents, `"DEBUG": "1"`,
		"manifest should record DEBUG since it's on the curated env-var list")
}

func TestSweepCWD_MovesLeftoverDebugFiles(t *testing.T) {
	resetGlobals(t)
	dir := t.TempDir()
	chdir(t, dir)

	// Plant a file that some non-cnspec code might write to CWD (e.g. cnquery
	// graph executor) before the bundle finalizes.
	leftover := filepath.Join(dir, "mondoo-debug-resolved-policy.dot")
	require.NoError(t, os.WriteFile(leftover, []byte("digraph {}"), 0o644))

	b, err := New(filepath.Join(dir, "bundle"))
	require.NoError(t, err)
	require.NoError(t, b.Activate())
	require.NoError(t, b.Finalize())

	_, err = os.Stat(leftover)
	assert.True(t, os.IsNotExist(err), "leftover file should have been swept into bundle")

	moved := filepath.Join(b.Dir, "mondoo-debug-resolved-policy.dot")
	raw, err := os.ReadFile(moved)
	require.NoError(t, err)
	assert.Equal(t, "digraph {}", string(raw))
}

func TestSweepCWD_IgnoresUnrelatedFiles(t *testing.T) {
	resetGlobals(t)
	dir := t.TempDir()
	chdir(t, dir)

	unrelated := filepath.Join(dir, "policy-bundle.yaml")
	require.NoError(t, os.WriteFile(unrelated, []byte("policies: []"), 0o644))

	b, err := New(filepath.Join(dir, "bundle"))
	require.NoError(t, err)
	require.NoError(t, b.Activate())
	require.NoError(t, b.Finalize())

	_, err = os.Stat(unrelated)
	assert.NoError(t, err, "unrelated files must not be moved")
}

func TestFinalizeAndAnnounce_NilSafe(t *testing.T) {
	var b *Bundle
	b.FinalizeAndAnnounce(io.Discard) // must not panic
}

func TestFinalizeAndAnnounce_PrintsPathOnce(t *testing.T) {
	resetGlobals(t)

	dir := t.TempDir()
	b, err := New(filepath.Join(dir, "bundle"))
	require.NoError(t, err)
	require.NoError(t, b.Activate())

	var buf bytes.Buffer
	b.FinalizeAndAnnounce(&buf)
	b.FinalizeAndAnnounce(&buf)

	count := strings.Count(buf.String(), "support bundle written to:")
	assert.Equal(t, 1, count, "path should only be announced once across multiple FinalizeAndAnnounce calls")
}
