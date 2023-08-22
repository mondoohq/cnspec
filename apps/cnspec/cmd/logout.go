// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"context"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	cnquery_cmd "go.mondoo.com/cnquery/apps/cnquery/cmd"
	cnquery_config "go.mondoo.com/cnquery/apps/cnquery/cmd/config"
	"go.mondoo.com/cnquery/cli/config"
	"go.mondoo.com/cnquery/cli/sysinfo"
	"go.mondoo.com/cnquery/upstream"
	"sigs.k8s.io/yaml"
)

func init() {
	rootCmd.AddCommand(logoutCmd)
	logoutCmd.Flags().Bool("force", false, "Force re-authentication")
}

var logoutCmd = &cobra.Command{
	Use:     "logout",
	Aliases: []string{"unregister"},
	Short:   "Log out from Mondoo Platform.",
	Long: `
This process also revokes the Mondoo Platform service account to 
ensure the credentials cannot be used in the future.
`,
	PreRun: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("force", cmd.Flags().Lookup("force"))
	},
	Run: func(cmd *cobra.Command, args []string) {
		var err error

		// its perfectly fine not to have a config here, therefore we ignore errors
		opts, optsErr := cnquery_config.ReadConfig()
		if optsErr != nil {
			log.Fatal().Msg("could not load configuration")
		}

		// print the used config to the user
		config.DisplayUsedConfig()

		// determine information about the client
		sysInfo, err := sysinfo.GatherSystemInfo()
		if err != nil {
			log.Fatal().Err(err).Msg("could not gather client information")
		}

		// check valid client authentication
		serviceAccount := opts.GetServiceCredential()
		if serviceAccount == nil {
			log.Error().Err(err).Msg(errorMessageServiceAccount)
			os.Exit(cnquery_cmd.ConfigurationErrorCode)
		}

		plugins := defaultRangerPlugins(sysInfo, opts.GetFeatures())
		certAuth, err := upstream.NewServiceAccountRangerPlugin(serviceAccount)
		if err != nil {
			log.Error().Err(err).Msg(errorMessageServiceAccount)
			os.Exit(cnquery_cmd.ConfigurationErrorCode)
		}
		plugins = append(plugins, certAuth)

		httpClient, err := opts.GetHttpClient()
		if err != nil {
			log.Fatal().Err(err).Msg("error while creating Mondoo API client")
		}
		client, err := upstream.NewAgentManagerClient(opts.UpstreamApiEndpoint(), httpClient, plugins...)
		if err != nil {
			log.Error().Err(err).Msg("could not initialize connection to Mondoo Platform")
			os.Exit(cnquery_cmd.ConfigurationErrorCode)
		}

		if !viper.GetBool("force") {
			log.Info().Msg("are you sure you want to revoke client access to Mondoo Platform? Use --force if you are sure")
			os.Exit(1)
			return
		}

		// try to load config into credentials struct
		credentials := opts.GetServiceCredential()

		// if we have credentials, we are going to self-destroy
		ctx := context.Background()
		if credentials != nil && len(credentials.Mrn) > 0 {
			_, err = client.PingPong(ctx, &upstream.Ping{})

			if err == nil {
				log.Info().Msgf("client %s authenticated successfully", credentials.Mrn)

				// un-register the agent
				_, err = client.UnRegisterAgent(ctx, &upstream.Mrn{
					Mrn: opts.AgentMrn,
				})
				if err != nil {
					log.Error().Err(err).Msg("failed to unregister client")
				}
			} else {
				log.Error().Err(err).Msg("communication with Mondoo Platform failed")
			}
		}

		// delete config if it exists
		path := viper.ConfigFileUsed()
		fi, err := os.Stat(path)
		if err == nil {
			log.Debug().Str("path", path).Msg("remove client information from config")

			opts.AgentMrn = ""

			data, err := yaml.Marshal(opts)
			if err != nil {
				log.Error().Err(err).Msg("could not update Mondoo config")
			}
			err = os.WriteFile(path, data, fi.Mode())
			if err != nil {
				log.Error().Err(err).Msg("could not update Mondoo config")
			}
		}

		log.Info().Msgf("Bye bye, space cat. Client %s unregistered successfully", credentials.Mrn)
	},
}
