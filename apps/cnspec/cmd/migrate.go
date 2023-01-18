package cmd

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// migrateCmd helps to migrate user config to the latest version
var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate cnspec CLI configuration to the latest version",
	Run: func(cmd *cobra.Command, args []string) {
		log.Info().Msg("No migration needed.")
	},
}

func init() {
	rootCmd.AddCommand(migrateCmd)
}
