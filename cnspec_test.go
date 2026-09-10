// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cnspec

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReleaseURL(t *testing.T) {
	// One shape, whether or not updates_url is set: the install service's
	// manifest path. Trailing slashes must not produce a double separator.
	assert.Equal(t, "https://install.mondoo.com/package/cnspec/latest.json", ReleaseURL(""))
	assert.Equal(t, "https://install.example.com/package/cnspec/latest.json", ReleaseURL("https://install.example.com"))
	assert.Equal(t, "https://install.example.com/package/cnspec/latest.json", ReleaseURL("https://install.example.com/"))
}
