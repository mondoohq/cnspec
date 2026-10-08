// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cmd

import "github.com/spf13/viper"

// autoUpdateKeys are the spellings of the auto-update setting, in precedence
// order. auto_update is canonical: MONDOO_AUTO_UPDATE and the --auto-update
// flag bind to it. auto-update is the spelling the documentation used to show
// in mondoo.yml. Viper's key delimiter is disabled, so for keys read from a
// file the two are different settings, and every reader that consulted only
// one of them ignored a config file written with the other.
var autoUpdateKeys = []string{"auto_update", "auto-update"}

// AutoUpdateEnabled reports whether automatic updates are on. One setting
// governs both cnspec's own binary and its providers.
//
// Precedence, highest first:
//  1. the --auto-update flag, when given
//  2. the MONDOO_AUTO_UPDATE environment variable
//  3. auto_update in mondoo.yml
//  4. auto-update in mondoo.yml
//  5. on
//
// A value that does not parse as a boolean counts as off. The binary
// self-update additionally stops for MONDOO_AUTO_UPDATE=false or
// MONDOO_AUTO_UPDATE_ENGINE=false regardless of the flag (see
// shouldTrySelfUpdate in main).
//
// This matches config.GetAutoUpdate in mql once it reads both spellings, and
// can be replaced by it then.
func AutoUpdateEnabled() bool {
	for _, key := range autoUpdateKeys {
		if viper.IsSet(key) {
			return viper.GetBool(key)
		}
	}
	return true
}
