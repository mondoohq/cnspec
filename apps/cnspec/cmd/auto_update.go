// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cmd

import "go.mondoo.com/mql/cli/config"

// AutoUpdateEnabled reports whether automatic updates are on. One setting
// governs both cnspec's own binary and its providers.
//
// Precedence, highest first:
//  1. the --auto-update flag, when given
//  2. the MONDOO_AUTO_UPDATE environment variable
//  3. auto_update in mondoo.yml
//  4. auto-update in mondoo.yml (the spelling the documentation used to show)
//  5. on
//
// A value that does not parse as a boolean counts as off. The binary
// self-update additionally stops for MONDOO_AUTO_UPDATE=false or
// MONDOO_AUTO_UPDATE_ENGINE=false regardless of the flag (see
// shouldTrySelfUpdate in main).
//
// It delegates to mql's config.GetAutoUpdate, which reads both spellings, so
// cnspec and the provider runtime cannot disagree about the setting.
func AutoUpdateEnabled() bool {
	return config.GetAutoUpdate()
}
