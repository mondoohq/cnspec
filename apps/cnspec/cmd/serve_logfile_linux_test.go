// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build linux

package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The test redirects a stand-in descriptor rather than the test binary's own
// stderr, so a failure here still reports normally.
func TestRedirectFDSendsDescriptorWritesToFile(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "cnspec.log")

	standIn, err := os.Create(filepath.Join(dir, "stand-in"))
	require.NoError(t, err)
	defer standIn.Close()

	require.NoError(t, redirectFD(logPath, int(standIn.Fd())))

	_, err = standIn.WriteString("after redirect\n")
	require.NoError(t, err)

	got, err := os.ReadFile(logPath)
	require.NoError(t, err)
	assert.Equal(t, "after redirect\n", string(got))

	orig, err := os.ReadFile(filepath.Join(dir, "stand-in"))
	require.NoError(t, err)
	assert.Empty(t, orig, "writes must no longer reach the original target")
}

// A truncation by an external rotator (logrotate copytruncate) must not leave
// the next write at the old offset, which would produce a sparse file with a
// run of NUL bytes ahead of it.
func TestRedirectFDSurvivesTruncation(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "cnspec.log")

	standIn, err := os.Create(filepath.Join(dir, "stand-in"))
	require.NoError(t, err)
	defer standIn.Close()

	require.NoError(t, redirectFD(logPath, int(standIn.Fd())))

	_, err = standIn.WriteString("before rotation, a reasonably long line\n")
	require.NoError(t, err)
	require.NoError(t, os.Truncate(logPath, 0))
	_, err = standIn.WriteString("after\n")
	require.NoError(t, err)

	got, err := os.ReadFile(logPath)
	require.NoError(t, err)
	assert.Equal(t, "after\n", string(got))
}

func TestRedirectFDReportsUnwritablePath(t *testing.T) {
	dir := t.TempDir()
	blocker := filepath.Join(dir, "not-a-dir")
	require.NoError(t, os.WriteFile(blocker, nil, 0o600))

	standIn, err := os.Create(filepath.Join(dir, "stand-in"))
	require.NoError(t, err)
	defer standIn.Close()

	err = redirectFD(filepath.Join(blocker, "cnspec.log"), int(standIn.Fd()))
	assert.Error(t, err)
}
