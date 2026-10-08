// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	cnquery_app "go.mondoo.com/mql/apps/mql/cmd"
	"go.mondoo.com/mql/providers"
	"go.mondoo.com/mql/providers-sdk/v1/plugin"
)

func init() {
	rootCmd.AddCommand(shellCmd)

	shellCmd.Flags().StringP("command", "c", "", "MQL query to execute in the shell")
	shellCmd.Flags().String("platform-id", "", "Select a specific target asset by providing its platform ID")
	addInventoryFlags(shellCmd, "Set the path to an inventory file that defines exactly one asset to connect to, with its credentials")
}

var shellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Interactive query shell for MQL",
	Long: `Allows the interactive exploration of MQL queries.

To open a shell on an asset defined in an inventory file, with the
credentials it defines, use --inventory-file instead of a provider
subcommand. The inventory must define exactly one asset; assets discovered
below it are offered for selection once it is connected:

  cnspec shell --inventory-file inventory.yml
`,
	PreRun: func(cmd *cobra.Command, args []string) {
		_ = viper.BindPFlag("platform-id", cmd.Flags().Lookup("platform-id"))
		bindInventoryFlags(cmd)
	},
}

var shellRun = func(cmd *cobra.Command, runtime *providers.Runtime, cliRes *plugin.ParseCLIRes) {
	shellConf := cnquery_app.ParseShellConfig(cmd, cliRes)
	shellConf.WelcomeMessage = cnspecLogo
	if err := cnquery_app.StartShell(runtime, shellConf); err != nil {
		log.Fatal().Err(err).Msg("failed to run query")
	}
}
