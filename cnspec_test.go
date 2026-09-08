// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cnspec

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReleaseURL(t *testing.T) {
	// The default host, an override, and an override with a trailing slash must
	// all produce exactly one separator before the manifest path.
	assert.Equal(t, "https://releases.mondoo.com/cnspec/latest.json", ReleaseURL(""))
	assert.Equal(t, "https://mirror.example.com/cnspec/latest.json", ReleaseURL("https://mirror.example.com"))
	assert.Equal(t, "https://mirror.example.com/cnspec/latest.json", ReleaseURL("https://mirror.example.com/"))
}
