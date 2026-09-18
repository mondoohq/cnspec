// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"go.mondoo.com/mql/cli/config"
)

// migrateCmd helps to migrate user config to the latest version
var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate cnspec CLI configuration to the latest version",
	Run: func(cmd *cobra.Command, args []string) {
		migrated := false

		// providers_url was replaced by updates_url. It still works, so this is a
		// migration rather than a repair: it adds the key the current code prefers
		// and leaves the old one in place, so the rollout can drop it when every
		// agent has moved.
		res, err := config.MigrateProvidersURL()
		switch {
		case err != nil:
			log.Error().Err(err).Str("path", res.Path).
				Msg("could not write the config; providers_url still works, so nothing is broken")
		case res.Migrated:
			migrated = true
			log.Info().Str("path", res.Path).Str("updates_url", res.UpdatesURL).
				Msg("added updates_url, derived from the deprecated providers_url")
			// updates_url is also where binary updates are looked for, which
			// providers_url never governed. Say so: a mirror that carries only
			// /providers will serve providers and no release manifest, and a failed
			// update check only warns.
			log.Info().Msg("updates_url also selects where binary updates are fetched from; " +
				"make sure this host serves the release manifests, not only /providers")
			log.Info().Msg("providers_url was left in place and can be removed once every agent is on this config")
		default:
			log.Debug().Str("reason", res.Skipped).Msg("providers_url not migrated")
		}

		if !migrated {
			log.Info().Msg("No migration needed.")
		}
	},
}

func init() {
	rootCmd.AddCommand(migrateCmd)
}
