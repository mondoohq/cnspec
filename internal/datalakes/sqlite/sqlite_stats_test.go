// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package sqlite

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/v13/policy/scanstats"
)

func TestFileSizeBytes(t *testing.T) {
	p := filepath.Join(t.TempDir(), "scan.db")
	require.NoError(t, os.WriteFile(p, []byte("hello world"), 0o600))
	require.Equal(t, int64(11), fileSizeBytes(p))
	require.Equal(t, int64(0), fileSizeBytes(filepath.Join(t.TempDir(), "missing.db")))
}

func TestUploadSizeFromFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "scan.db")
	require.NoError(t, os.WriteFile(p, []byte("hello world"), 0o600))

	c := scanstats.New()
	c.AddInt(scanstats.MetricUploadSize, "bytes", fileSizeBytes(p))

	stats := c.ToProto()
	require.Equal(t, scanstats.MetricUploadSize, stats.Metrics[0].Name)
	require.Equal(t, "bytes", stats.Metrics[0].Unit)
	require.Equal(t, int64(11), stats.Metrics[0].GetIntValue())
}
