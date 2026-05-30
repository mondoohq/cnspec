// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package supportbundle

import (
	"bytes"
	"context"
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
	"go.mondoo.com/cnspec/v13/internal/scandump"
	"go.mondoo.com/mql/v13/logger"
)

// resetGlobals saves and restores the package-level state that Activate
// mutates so sequential tests don't interfere.
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
	prevWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(t.TempDir()))
	t.Cleanup(func() { _ = os.Chdir(prevWd) })

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

func TestActivate_AttachesScandumpRunToContext(t *testing.T) {
	resetGlobals(t)
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	logger.DumpLocal = ""

	dir := t.TempDir()
	b, err := New(filepath.Join(dir, "bundle"))
	require.NoError(t, err)

	ctx, err := b.Activate(context.Background())
	require.NoError(t, err)
	defer b.Finalize()

	assert.True(t, scandump.Active(ctx),
		"Activate must return a ctx that carries a scandump.Run")
	assert.Equal(t, zerolog.DebugLevel, zerolog.GlobalLevel(),
		"Activate must force debug level so dump helpers fire")
	assert.True(t, strings.HasSuffix(logger.DumpLocal, "mondoo-debug-"),
		"DumpLocal should be set so cnquery-side dumps land in the bundle; got %q", logger.DumpLocal)
	assert.True(t, strings.Contains(logger.DumpLocal, "debug"),
		"DumpLocal must point inside the bundle's debug/ subdirectory; got %q", logger.DumpLocal)
}

func TestActivate_DoesNotLowerVerbosityIfTraceAlready(t *testing.T) {
	resetGlobals(t)
	zerolog.SetGlobalLevel(zerolog.TraceLevel)

	dir := t.TempDir()
	b, err := New(filepath.Join(dir, "bundle"))
	require.NoError(t, err)
	_, err = b.Activate(context.Background())
	require.NoError(t, err)
	defer b.Finalize()

	assert.Equal(t, zerolog.TraceLevel, zerolog.GlobalLevel(),
		"Activate must not downgrade from trace to debug")
}

func TestActivate_TeesLogsToFileWithTimestamps(t *testing.T) {
	resetGlobals(t)

	dir := t.TempDir()
	b, err := New(filepath.Join(dir, "bundle"))
	require.NoError(t, err)
	_, err = b.Activate(context.Background())
	require.NoError(t, err)

	log.Debug().Str("k", "v").Msg("hello support bundle")

	require.NoError(t, b.Finalize())

	raw, err := os.ReadFile(filepath.Join(b.Dir, "debug.log"))
	require.NoError(t, err)
	contents := string(raw)
	assert.Contains(t, contents, "hello support bundle")
	assert.Contains(t, contents, "k=v")
	// RFC3339Nano timestamps always include "T".
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
	_, err = b.Activate(context.Background())
	require.NoError(t, err)

	require.NoError(t, b.Finalize())
	assert.Equal(t, "/preset-prefix-", logger.DumpLocal, "DumpLocal must be restored")
	assert.Equal(t, zerolog.WarnLevel, zerolog.GlobalLevel(), "global level must be restored")
	assert.Equal(t, originalLogger, log.Logger, "log.Logger must be restored")

	// Second call is a no-op.
	require.NoError(t, b.Finalize())
}

func TestFinalize_WritesManifestAndProviders(t *testing.T) {
	resetGlobals(t)

	dir := t.TempDir()
	b, err := New(filepath.Join(dir, "bundle"))
	require.NoError(t, err)
	b.Args = []string{"cnspec", "scan", "local", "--collect-support-bundle"}
	_, err = b.Activate(context.Background())
	require.NoError(t, err)
	require.NoError(t, b.Finalize())

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
	_, err = b.Activate(context.Background())
	require.NoError(t, err)
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

func TestFinalizeAndAnnounce_NilSafe(t *testing.T) {
	var b *Bundle
	b.FinalizeAndAnnounce(io.Discard) // must not panic
}

func TestFinalizeAndAnnounce_PrintsPathOnce(t *testing.T) {
	resetGlobals(t)

	dir := t.TempDir()
	b, err := New(filepath.Join(dir, "bundle"))
	require.NoError(t, err)
	_, err = b.Activate(context.Background())
	require.NoError(t, err)

	var buf bytes.Buffer
	b.FinalizeAndAnnounce(&buf)
	b.FinalizeAndAnnounce(&buf)

	count := strings.Count(buf.String(), "support bundle written to:")
	assert.Equal(t, 1, count, "path should only be announced once across multiple FinalizeAndAnnounce calls")
}

func TestActivate_DumpsLandInDebugSubdir(t *testing.T) {
	resetGlobals(t)

	dir := t.TempDir()
	b, err := New(filepath.Join(dir, "bundle"))
	require.NoError(t, err)
	ctx, err := b.Activate(context.Background())
	require.NoError(t, err)
	defer b.Finalize()

	// Run-level dump
	scandump.JSON(ctx, "run-level", map[string]int{"x": 1})
	// Per-asset dump
	ctx2, _, err := scandump.WithAsset(ctx, "webserver")
	require.NoError(t, err)
	scandump.JSON(ctx2, "report", map[string]int{"y": 2})

	debugDir := filepath.Join(b.Dir, "debug")
	_, err = os.Stat(filepath.Join(debugDir, "run-level.json"))
	assert.NoError(t, err, "run-level dump should land under <bundle>/debug/")
	_, err = os.Stat(filepath.Join(debugDir, "webserver", "report.json"))
	assert.NoError(t, err, "per-asset dump should land under <bundle>/debug/<asset>/")
}
