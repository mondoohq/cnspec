// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/mql"
)

func TestScanAdvertisesRateLimitRetry(t *testing.T) {
	f := withRateLimitRetry(mql.DefaultFeatures)
	assert.True(t, f.IsActive(mql.RateLimitRetry))
	assert.Equal(t, len(f), len(withRateLimitRetry(f)))
}
