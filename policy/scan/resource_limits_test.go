// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConnectionsForMemory(t *testing.T) {
	const mib = 1024 * 1024
	const gib = 1024 * mib
	tests := []struct {
		name  string
		limit uint64
		want  int
	}{
		{"unlimited returns default", 0, defaultMaxConnections},
		{"512MiB", 512 * mib, 512 * mib / estimatedMemoryPerConnectionBytes},
		{"1GiB", 1 * gib, 1 * gib / estimatedMemoryPerConnectionBytes},
		{"tiny clamps to 1", 10 * mib, 1},
		{"huge caps at default", 1024 * gib, defaultMaxConnections},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := connectionsForMemory(tt.limit)
			assert.Equal(t, tt.want, got)
			assert.GreaterOrEqual(t, got, 1)
			assert.LessOrEqual(t, got, defaultMaxConnections)
		})
	}
}

func TestGetMaxConnections(t *testing.T) {
	t.Run("env override wins", func(t *testing.T) {
		t.Setenv("MONDOO_MAX_PROVIDER_CONNECTIONS", "7")
		assert.Equal(t, 7, getMaxConnections())
	})

	t.Run("invalid env falls through", func(t *testing.T) {
		t.Setenv("MONDOO_MAX_PROVIDER_CONNECTIONS", "not-a-number")
		got := getMaxConnections()
		// Falls back to the detected/default value, always within bounds.
		assert.GreaterOrEqual(t, got, 1)
		assert.LessOrEqual(t, got, defaultMaxConnections)
	})

	t.Run("zero env falls through", func(t *testing.T) {
		t.Setenv("MONDOO_MAX_PROVIDER_CONNECTIONS", "0")
		got := getMaxConnections()
		assert.GreaterOrEqual(t, got, 1)
		assert.LessOrEqual(t, got, defaultMaxConnections)
	})
}
