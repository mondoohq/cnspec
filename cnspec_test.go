// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cnspec

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/mql/cli/config"
)

func TestReleaseURL(t *testing.T) {
	// One shape, whether or not updates_url is set: the install service's
	// manifest path. Trailing slashes must not produce a double separator.
	assert.Equal(t, "https://install.mondoo.com/package/cnspec/latest.json", ReleaseURL("", ""))
	assert.Equal(t, "https://install.example.com/package/cnspec/latest.json", ReleaseURL("https://install.example.com", ""))
	assert.Equal(t, "https://install.example.com/package/cnspec/latest.json", ReleaseURL("https://install.example.com/", ""))
}

func TestReleaseURLChannel(t *testing.T) {
	// Stable produces exactly the URL this returned before channels existed, so
	// a client that never sets one is unaffected.
	assert.Equal(t,
		"https://install.mondoo.com/package/cnspec/latest.json",
		ReleaseURL("", config.ChannelStable))

	// The channel is a query parameter, not a different document. The install
	// service's routes are named after the package, and it answers an unmatched
	// path with the landing page as a cacheable 200 - so a path-shaped channel
	// would hand the updater HTML with a success status.
	assert.Equal(t,
		"https://install.mondoo.com/package/cnspec/latest.json?channel=preview",
		ReleaseURL("", config.ChannelPreview))

	assert.Equal(t,
		"https://install.example.com/package/cnspec/latest.json?channel=preview",
		ReleaseURL("https://install.example.com/", config.ChannelPreview))
}
