package cmd

import (
	zlog "github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	cnquery_app "go.mondoo.com/cnquery/apps/cnquery/cmd"
	"go.mondoo.com/cnquery/apps/cnquery/cmd/builder"
	discovery_common "go.mondoo.com/cnquery/motor/discovery/common"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnspec/internal/plugin"
)

func init() {
	rootCmd.AddCommand(execCmd)
}

var execCmd = builder.NewProviderCommand(builder.CommandOpts{
	Use:   "run",
	Short: "Run an MQL query",
	Long:  `Run an MQL query on the CLI and displays its results.`,
	CommonFlags: func(cmd *cobra.Command) {
		cmd.Flags().Bool("parse", false, "Parse the query and return the logical structure.")
		cmd.Flags().Bool("ast", false, "Parse the query and return the abstract syntax tree (AST).")
		cmd.Flags().BoolP("json", "j", false, "Run the query and return the object in a JSON structure.")
		cmd.Flags().String("query", "", "MQL query to execute.")
		cmd.Flags().MarkHidden("query")
		cmd.Flags().StringP("command", "c", "", "MQL query to execute.")

		cmd.Flags().StringP("password", "p", "", "Connection password (such as for ssh/winrm).")
		cmd.Flags().Bool("ask-pass", false, "Prompt for connection password.")
		cmd.Flags().StringP("identity-file", "i", "", "Select the file from which to read the identity (private key) for public key authentication.")
		cmd.Flags().Bool("insecure", false, "Disable TLS/SSL checks or SSH hostkey config.")
		cmd.Flags().Bool("sudo", false, "Elevate privileges with sudo.")
		cmd.Flags().String("platform-id", "", "Select a specific target asset by providing its platform ID.")
		cmd.Flags().Bool("instances", false, "Also scan instances. This only applies to API targets like AWS, Azure or GCP.")
		cmd.Flags().Bool("host-machines", false, "Also scan host machines like ESXi servers.")

		cmd.Flags().Bool("record", false, "Record provider calls. This only works for operating system providers.")
		cmd.Flags().MarkHidden("record")

		cmd.Flags().String("record-file", "", "File path for the recorded provider calls. This only works for operating system providers.")
		cmd.Flags().MarkHidden("record-file")

		cmd.Flags().String("path", "", "Path to a local file or directory for the connection to use.")
		cmd.Flags().StringToString("option", nil, "Additional connection options. You can pass in multiple options using `--option key=value`")
		cmd.Flags().String("discover", discovery_common.DiscoveryAuto, "Enable the discovery of nested assets. Supports: 'all|auto|instances|host-instances|host-machines|container|container-images|pods|cronjobs|statefulsets|deployments|jobs|replicasets|daemonsets'")
		cmd.Flags().StringToString("discover-filter", nil, "Additional filter for asset discovery.")
	},
	CommonPreRun: func(cmd *cobra.Command, args []string) {
		// for all assets
		viper.BindPFlag("insecure", cmd.Flags().Lookup("insecure"))
		viper.BindPFlag("sudo.active", cmd.Flags().Lookup("sudo"))

		viper.BindPFlag("output", cmd.Flags().Lookup("output"))

		viper.BindPFlag("vault.name", cmd.Flags().Lookup("vault"))
		viper.BindPFlag("platform-id", cmd.Flags().Lookup("platform-id"))
		viper.BindPFlag("query", cmd.Flags().Lookup("query"))
		viper.BindPFlag("command", cmd.Flags().Lookup("command"))

		viper.BindPFlag("record", cmd.Flags().Lookup("record"))
		viper.BindPFlag("record-file", cmd.Flags().Lookup("record-file"))
	},
	Run: func(cmd *cobra.Command, args []string, provider providers.ProviderType, assetType builder.AssetType) {
		conf, err := cnquery_app.GetCobraRunConfig(cmd, args, provider, assetType)
		if err != nil {
			zlog.Fatal().Err(err).Msg("failed to prepare config")
		}

		err = plugin.RunQuery(conf)
		if err != nil {
			zlog.Fatal().Err(err).Msg("failed to run query")
		}
	},
})
