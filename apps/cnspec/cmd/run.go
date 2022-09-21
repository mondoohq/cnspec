package cmd

import (
	zlog "github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	cnquery_app "go.mondoo.com/cnquery/apps/cnquery/cmd"
	"go.mondoo.com/cnquery/apps/cnquery/cmd/builder"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnspec/internal/plugin"
)

func init() {
	rootCmd.AddCommand(execCmd)
}

var execCmd = builder.NewProviderCommand(builder.CommandOpts{
	Use:   "run",
	Short: "Run a MQL query",
	Long:  `Run a MQL query on the CLI and displays its results.`,
	CommonFlags: func(cmd *cobra.Command) {
		cmd.Flags().Bool("parse", false, "Parse the query and return the logical structure")
		cmd.Flags().Bool("ast", false, "Parse the query and return the Abstract Syntax Tree (AST)")
		cmd.Flags().BoolP("json", "j", false, "Run the query and return the object in a JSON structure")
		cmd.Flags().String("query", "", "MQL query to be executed")
		cmd.Flags().MarkHidden("query")
		cmd.Flags().StringP("command", "c", "", "MQL query to be executed")

		cmd.Flags().StringP("password", "p", "", "connection password e.g. for ssh/winrm")
		cmd.Flags().Bool("ask-pass", false, "ask for connection password")
		cmd.Flags().StringP("identity-file", "i", "", "Selects a file from which the identity (private key) for public key authentication is read.")
		cmd.Flags().Bool("insecure", false, "disables TLS/SSL checks or SSH hostkey config")
		cmd.Flags().Bool("sudo", false, "runs with sudo")
		cmd.Flags().String("platform-id", "", "select an specific asset by providing the platform id for the target")
		cmd.Flags().Bool("instances", false, "also scan instances (only applies to api targets like aws, azure or gcp)")
		cmd.Flags().Bool("host-machines", false, "also scan host machines like ESXi server")

		cmd.Flags().Bool("record", false, "records provider calls (only works for operating system providers)")
		cmd.Flags().MarkHidden("record")

		cmd.Flags().String("record-file", "", "file path to for the recorded provider calls (only works for operating system providers)")
		cmd.Flags().MarkHidden("record-file")

		cmd.Flags().String("path", "", "path to a local file or directory that the connection should use")
		cmd.Flags().StringToString("option", nil, "addition connection options, multiple options can be passed in via --option key=value")
		cmd.Flags().String("discover", "", "enables the discovery of nested assets. Supported are 'all|instances|host-instances|host-machines|container|container-images|pods|cronjobs|statefulsets|deployments|jobs|replicasets|daemonsets'")
		cmd.Flags().StringToString("discover-filter", nil, "additional filter for asset discovery")
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
