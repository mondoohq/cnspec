package cmd

import (
	"context"
	"fmt"

	"github.com/muesli/termenv"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.mondoo.com/cnquery/v9/cli/config"
	"go.mondoo.com/cnquery/v9/cli/theme"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/upstream"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/upstream/gql"
	"go.mondoo.com/cnspec/v9/policy"
	cnspec_upstream "go.mondoo.com/cnspec/v9/upstream"
)

func init() {
	rootCmd.AddCommand(policyCmd)

	policyCmd.AddCommand(policyListCmd)
	policyListCmd.Flags().StringP("file", "f", "", "list policies in a local bundle file")
}

var policyCmd = &cobra.Command{
	Use:   "policy",
	Short: "Manage local and upstream policies.",
}

var policyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List enabled policies in the connected space.",
	Args:  cobra.MaximumNArgs(0),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlag("file", cmd.Flags().Lookup("file")); err != nil {
			return err
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		bundleFile := viper.GetString("file")
		var policies []*policy.Policy
		if bundleFile != "" {
			policyBundle, err := policy.DefaultBundleLoader().BundleFromPaths(bundleFile)
			if err != nil {
				return err
			}
			policies = policyBundle.Policies
		} else {
			opts, err := config.Read()
			if err != nil {
				return err
			}
			config.DisplayUsedConfig()

			mondooClient, err := getGqlClient(opts)
			if err != nil {
				return err
			}

			policies, err = cnspec_upstream.SearchPolicy(
				context.Background(), mondooClient, opts.GetParentMrn(), true)
			if err != nil {
				return err
			}
		}
		for _, policy := range policies {
			fmt.Println(policy.Name + " " + policy.Version)
			id := policy.Uid
			if policy.Mrn != "" {
				id = policy.Mrn
			}
			fmt.Println(termenv.String("  " + id).Foreground(theme.DefaultTheme.Colors.Disabled))
		}

		return nil
	},
}

func getGqlClient(opts *config.Config) (*gql.MondooClient, error) {
	serviceAccount := opts.GetServiceCredential()
	if serviceAccount == nil {
		return nil, fmt.Errorf("cnspec has no credentials. Log in with `cnspec login`")
	}

	httpClient, err := opts.GetHttpClient()
	if err != nil {
		return nil, err
	}

	upstreamConfig := &upstream.UpstreamConfig{
		SpaceMrn:    opts.GetParentMrn(),
		ApiEndpoint: opts.UpstreamApiEndpoint(),
		ApiProxy:    opts.APIProxy,
		Creds:       serviceAccount,
	}

	mondooClient, err := gql.NewClient(upstreamConfig, httpClient)
	if err != nil {
		return nil, err
	}

	return mondooClient, nil
}
