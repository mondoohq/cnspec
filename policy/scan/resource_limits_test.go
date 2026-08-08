// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
