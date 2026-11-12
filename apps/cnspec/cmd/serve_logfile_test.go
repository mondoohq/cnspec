// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build !windows

package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenLogFileCreatesPrivateFileAndDir(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cnspec", "cnspec.log")

	f, err := openLogFile(path)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	dir, err := os.Stat(filepath.Dir(path))
	require.NoError(t, err)
	assert.Zero(t, dir.Mode().Perm()&0o007, "log directory must not be accessible to other users")

	file, err := os.Stat(path)
	require.NoError(t, err)
	assert.Zero(t, file.Mode().Perm()&0o007, "log file must not be readable by other users")
}

func TestOpenLogFileAppends(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cnspec.log")
	require.NoError(t, os.WriteFile(path, []byte("existing\n"), 0o640))

	f, err := openLogFile(path)
	require.NoError(t, err)
	_, err = f.WriteString("new\n")
	require.NoError(t, err)
	require.NoError(t, f.Close())

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "existing\nnew\n", string(got))
}
