// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/internal/reportfixture"
)

func TestFileHandler(t *testing.T) {
	yr, err := reportfixture.UbuntuScan()
	require.NoError(t, err)

	now := time.Now().Format(time.RFC3339)
	t.Run("with no prefix", func(t *testing.T) {
		fileName := fmt.Sprintf("/tmp/%s-testfilehandler.json", now)
		config := HandlerConfig{Format: "compact", OutputTarget: fileName}
		handler, err := NewOutputHandler(config)
		require.NoError(t, err)
		err = handler.WriteReport(context.Background(), yr)
		require.NoError(t, err)
		data, err := os.ReadFile(fileName)
		require.NoError(t, err)

		strData := string(data)
		assert.Contains(t, strData, "! Error:          Set")
		assert.Contains(t, strData, "✓ Ensure ")
		assert.Contains(t, strData, "✕ CRITICAL (100): Ensure")
		err = os.Remove(fileName)
		require.NoError(t, err)
	})

	t.Run("with file:// prefix", func(t *testing.T) {
		fileName := fmt.Sprintf("file:///tmp/%s-testfilehandler.json", now)
		config := HandlerConfig{Format: "compact", OutputTarget: fileName}
		handler, err := NewOutputHandler(config)
		require.NoError(t, err)
		err = handler.WriteReport(context.Background(), yr)
		require.NoError(t, err)
		trimmed := strings.TrimPrefix(fileName, "file://")
		data, err := os.ReadFile(trimmed)
		require.NoError(t, err)

		strData := string(data)
		assert.Contains(t, strData, "! Error:          Set")
		assert.Contains(t, strData, "✓ Ensure ")
		assert.Contains(t, strData, "✕ CRITICAL (100): Ensure")
		err = os.Remove(trimmed)
		require.NoError(t, err)
	})
}

// TestFileHandlerWritesPrivateFiles covers the mode of an --output-target file.
//
// os.Create would leave it 0666&^umask, so 0644 on a normal host: world-readable
// on a shared machine. The content is the same as the directory handlers write --
// account ids and ARNs, the MQL source of every check, rendered assessments
// carrying the observed values -- and those already create 0600.
func TestFileHandlerWritesPrivateFiles(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix permission bits")
	}
	yr, err := reportfixture.UbuntuScan()
	require.NoError(t, err)

	path := filepath.Join(t.TempDir(), "report.json")
	handler, err := NewOutputHandler(HandlerConfig{Format: "json", OutputTarget: path})
	require.NoError(t, err)
	require.NoError(t, handler.WriteReport(context.Background(), yr))

	fi, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), fi.Mode().Perm(),
		"the report is readable by every local account")
}

// TestFileHandlerDoesNotFollowASymlink covers a symlink pre-placed at the
// --output-target path.
//
// Following it truncates whatever it points at, so a scan -- frequently running
// as root -- overwrites a file of someone else's choosing with the report. This
// is the same exposure the directory handlers guard against; the single-file
// path carries every format, so it is the one most likely to be used.
func TestFileHandlerDoesNotFollowASymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("O_NOFOLLOW has no Windows equivalent; see internal/reportfile")
	}
	yr, err := reportfixture.UbuntuScan()
	require.NoError(t, err)

	root := t.TempDir()
	victim := filepath.Join(root, "victim")
	require.NoError(t, os.WriteFile(victim, []byte("do not overwrite me"), 0o600))

	target := filepath.Join(root, "report.json")
	require.NoError(t, os.Symlink(victim, target))

	handler, err := NewOutputHandler(HandlerConfig{Format: "json", OutputTarget: target})
	require.NoError(t, err)
	err = handler.WriteReport(context.Background(), yr)
	require.Error(t, err, "the open has to fail rather than follow the link")

	content, readErr := os.ReadFile(victim)
	require.NoError(t, readErr)
	assert.Equal(t, "do not overwrite me", string(content),
		"the symlink target must not be truncated and rewritten with the report")
}
