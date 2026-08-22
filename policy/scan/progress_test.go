// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/cli/progress"
)

// Go test binaries run with stdout on a pipe, so these cover the non-TTY side
// of createProgressBar: the noop today, and the NDJSON stream when a parent
// process asks for one.

func TestCreateProgressBarPipedWithoutStreamIsNoop(t *testing.T) {
	t.Setenv(progress.StreamEnvVar, "")

	mp, err := createProgressBar(false)
	require.NoError(t, err)
	assert.IsType(t, progress.NoopMultiProgress{}, mp)
}

func TestCreateProgressBarStream(t *testing.T) {
	path := filepath.Join(t.TempDir(), "progress.ndjson")
	t.Setenv(progress.StreamEnvVar, path)

	mp, err := createProgressBar(false)
	require.NoError(t, err)
	require.NotNil(t, mp)
	assert.NotEqual(t, progress.NoopMultiProgress{}, mp)

	require.NoError(t, mp.Open())
	mp.AddTask("id-1", nil)
	mp.Completed("id-1")
	mp.Close()
	mp.Close() // the scanner closes explicitly and again from a defer

	raw, err := os.ReadFile(path)
	require.NoError(t, err)

	var names []string
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		var e progress.StreamEvent
		require.NoErrorf(t, json.Unmarshal([]byte(line), &e), "unparseable line: %q", line)
		names = append(names, e.Event)
	}
	assert.Equal(t, []string{
		progress.EventScanStart,
		progress.EventAssetAdded,
		progress.EventAssetDone,
		progress.EventScanDone,
	}, names)
}

// An explicitly disabled progress bar stays disabled — headless callers such as
// `cnspec serve` pass DisableProgressBar and must not start writing a stream
// just because one is configured in the environment.
func TestCreateProgressBarDisabledWinsOverStream(t *testing.T) {
	path := filepath.Join(t.TempDir(), "progress.ndjson")
	t.Setenv(progress.StreamEnvVar, path)

	mp, err := createProgressBar(true)
	require.NoError(t, err)
	assert.IsType(t, progress.NoopMultiProgress{}, mp)
	assert.NoFileExists(t, path)
}

func TestCreateProgressBarStreamTargetFails(t *testing.T) {
	t.Setenv(progress.StreamEnvVar, filepath.Join(t.TempDir(), "no-such-dir", "progress.ndjson"))

	_, err := createProgressBar(false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), progress.StreamEnvVar)
}
