package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/muesli/termenv"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.mondoo.com/cnquery/v9/cli/config"
	"go.mondoo.com/cnquery/v9/cli/theme"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/upstream"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/upstream/gql"
	"go.mondoo.com/cnspec/v9/policy"
	cnspec_upstream "go.mondoo.com/cnspec/v9/upstream"
	"k8s.io/utils/ptr"
)

func init() {
	rootCmd.AddCommand(policyCmd)

	policyCmd.AddCommand(policyListCmd)
	policyListCmd.Flags().StringP("file", "f", "", "list policies in a local bundle file")

	policyCmd.AddCommand(policyUploadCmd)
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
				context.Background(), mondooClient, opts.GetParentMrn(), ptr.To(true), ptr.To(true), ptr.To(true))
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

var policyUploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "Upload a policy to the connected space.",
	Args:  cobra.ExactArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		bundle, err := policy.DefaultBundleLoader().BundleFromPaths(args[0])
		if err != nil {
			log.Error().Msgf("failed to upload policies: %s", err)
			os.Exit(1)
		}

		opts, err := config.Read()
		if err != nil {
			log.Error().Msgf("failed to upload policies: %s", err)
			os.Exit(1)
		}
		config.DisplayUsedConfig()

		certAuth, err := upstream.NewServiceAccountRangerPlugin(opts.GetServiceCredential())
		if err != nil {
			log.Error().Msgf("failed to upload policies: %s", err)
			os.Exit(1)
		}

		httpClient, err := opts.GetHttpClient()
		if err != nil {
			log.Error().Msgf("failed to upload policies: %s", err)
			os.Exit(1)
		}
		policyHub, err := policy.NewPolicyHubClient(opts.UpstreamApiEndpoint(), httpClient, certAuth)
		if err != nil {
			log.Error().Msgf("failed to upload policies: %s", err)
			os.Exit(1)
		}

		ctx := context.Background()
		bundle.OwnerMrn = opts.GetParentMrn()
		_, err = policyHub.SetBundle(ctx, bundle)
		if err != nil {
			log.Error().Msgf("failed to upload policies: %s", err)
			os.Exit(1)
		}

		successMsg := fmt.Sprintf("successfully uploaded %d ", len(bundle.Policies))
		if len(bundle.Policies) > 1 {
			successMsg += "policies"
		} else {
			successMsg += "policy"
		}
		log.Info().Msg(theme.DefaultTheme.Success(successMsg))

		mondooClient, err := getGqlClient(opts)
		if err != nil {
			log.Error().Msgf("failed to create upstream client: %s", err)
			os.Exit(1)
		}

		upstreamPolicies, err := cnspec_upstream.SearchPolicy(ctx, mondooClient, opts.GetParentMrn(), nil, ptr.To(false), ptr.To(true))
		if err != nil {
			log.Error().Msgf("failed to get upstream policies: %s", err)
			os.Exit(1)
		}

		filteredPolicies := make([]*policy.Policy, 0, len(bundle.Policies))
		for _, policy := range upstreamPolicies {
			for _, p := range bundle.Policies {
				if strings.Contains(policy.Mrn, p.Uid) {
					filteredPolicies = append(filteredPolicies, policy)
					break
				}
			}
		}

		for _, p := range filteredPolicies {
			fmt.Println("  policy: " + p.Name + " " + p.Version)
			fmt.Println(termenv.String("    " + p.Mrn).Foreground(theme.DefaultTheme.Colors.Disabled))
		}

		space, err := cnspec_upstream.GetSpace(ctx, mondooClient, opts.GetParentMrn())
		if err != nil {
			log.Error().Msgf("failed to get space: %s", err)
			os.Exit(1)
		}

		fmt.Println("  space: " + space.Name)
		fmt.Println(termenv.String("    " + space.Mrn).Foreground(theme.DefaultTheme.Colors.Disabled))
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
