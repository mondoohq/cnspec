// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.mondoo.com/cnquery"
	cnquery_app "go.mondoo.com/cnquery/apps/cnquery/cmd"
	"go.mondoo.com/cnquery/providers"
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnspec"
)

func init() {
	rootCmd.AddCommand(shellCmd)

	shellCmd.Flags().StringP("command", "c", "", "MQL query to executed in the shell.")
	shellCmd.Flags().String("platform-id", "", "Select a specific target asset by providing its platform ID.")
}

var shellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Interactive query shell for MQL.",
	Long:  `Allows the interactive exploration of MQL queries.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("platform-id", cmd.Flags().Lookup("platform-id"))
	},
}

var shellRun = func(cmd *cobra.Command, runtime *providers.Runtime, cliRes *plugin.ParseCLIRes) {
	shellConf := cnquery_app.ParseShellConfig(cmd, cliRes)
	shellConf.WelcomeMessage = cnspecLogo

	// FIXME: workaround for `mondoo.version` in case of builtin providers
	// (which is how the core provider is set up by default)
	if cnquery.Version == "" {
		cnquery.Version = cnspec.Version
	}

	if err := cnquery_app.StartShell(runtime, shellConf); err != nil {
		log.Fatal().Err(err).Msg("failed to run query")
	}
}
