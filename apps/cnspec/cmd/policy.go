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
	"gopkg.in/yaml.v2"
	"k8s.io/utils/ptr"
)

const (
	PolicyMrnPrefix          = "//policy.api.mondoo.app"
	PrivatePoliciesMrnPrefix = "//policy.api.mondoo.app/spaces"
)

func init() {
	rootCmd.AddCommand(policyCmd)

	policyCmd.AddCommand(policyListCmd)
	policyListCmd.Flags().StringP("file", "f", "", "a local bundle file")

	policyCmd.AddCommand(policyUploadCmd)
	policyCmd.AddCommand(policyDeleteCmd)
	policyCmd.AddCommand(policyEnableCmd)
	policyCmd.AddCommand(policyDisableCmd)

	policyCmd.AddCommand(policyInfoCmd)
	policyInfoCmd.Flags().StringP("file", "f", "", "a local bundle file")

	policyCmd.AddCommand(policyDownloadCmd)
	policyDownloadCmd.Flags().StringP("output", "o", "", "output file")
}

var policyCmd = &cobra.Command{
	Use:   "policy",
	Short: "Manage local and upstream policies.",
}

var policyListCmd = &cobra.Command{
	Use:   "list [-f bundle]",
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
	Use:   "upload my.mql.yaml",
	Short: "Upload a policy to the connected space.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		bundle, err := policy.DefaultBundleLoader().BundleFromPaths(args[0])
		if err != nil {
			log.Error().Msgf("failed to upload policies: %s", err)
			os.Exit(1)
		}

		opts, err := config.Read()
		if err != nil {
			log.Error().Msgf("failed to get config: %s", err)
			os.Exit(1)
		}
		config.DisplayUsedConfig()

		policyHub, err := getPolicyHubClient(opts)
		if err != nil {
			log.Error().Msgf("failed to create upstream client: %s", err)
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

		for _, p := range bundle.Policies {
			fmt.Println("  policy: " + p.Name + " " + p.Version)
			fmt.Println(termenv.String("    " + getPolicyMrn(opts.GetParentMrn(), p.Uid)).Foreground(theme.DefaultTheme.Colors.Disabled))
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

var policyDeleteCmd = &cobra.Command{
	Use:   "delete UID/MRN",
	Short: "Delete a policy from the connected space.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		opts, err := config.Read()
		if err != nil {
			log.Error().Msgf("failed to get config: %s", err)
			os.Exit(1)
		}
		config.DisplayUsedConfig()

		policyMrn := args[0]
		if !strings.HasPrefix(policyMrn, PolicyMrnPrefix) {
			policyMrn = getPolicyMrn(opts.GetParentMrn(), args[0])
		}

		// Add a meaningful error when trying to delete a public policy
		if !strings.HasPrefix(policyMrn, PrivatePoliciesMrnPrefix) {
			log.Error().Msgf("failed to delete policy: it is only possible to delete private policies")
			os.Exit(1)
		}

		policyHub, err := getPolicyHubClient(opts)
		if err != nil {
			log.Error().Msgf("failed to upload policies: %s", err)
			os.Exit(1)
		}

		ctx := context.Background()
		p, err := policyHub.GetPolicy(ctx, &policy.Mrn{Mrn: policyMrn})
		if err != nil {
			log.Error().Msgf("failed to get policy: %s", err)
			os.Exit(1)
		}

		_, err = policyHub.DeletePolicy(ctx, &policy.Mrn{Mrn: policyMrn})
		if err != nil {
			log.Error().Msgf("failed to delete policy: %s", err)
			os.Exit(1)
		}

		log.Info().Msg(theme.DefaultTheme.Success("successfully removed policy from space"))

		mondooClient, err := getGqlClient(opts)
		if err != nil {
			log.Error().Msgf("failed to create upstream client: %s", err)
			os.Exit(1)
		}

		space, err := cnspec_upstream.GetSpace(ctx, mondooClient, opts.GetParentMrn())
		if err != nil {
			log.Error().Msgf("failed to get space: %s", err)
			os.Exit(1)
		}

		fmt.Println("  policy: " + p.Name + " " + p.Version)
		fmt.Println(termenv.String("    " + p.Mrn).Foreground(theme.DefaultTheme.Colors.Disabled))

		fmt.Println("  space: " + space.Name)
		fmt.Println(termenv.String("    " + space.Mrn).Foreground(theme.DefaultTheme.Colors.Disabled))
	},
}

var policyInfoCmd = &cobra.Command{
	Use:     "info UID/MRN",
	Short:   "Show more info about a policy from the connected space.",
	Aliases: []string{"show"},
	Args:    cobra.MaximumNArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlag("file", cmd.Flags().Lookup("file")); err != nil {
			return err
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		bundleFile := viper.GetString("file")
		var policies []*policy.Policy
		if bundleFile != "" {
			policyBundle, err := policy.DefaultBundleLoader().BundleFromPaths(bundleFile)
			if err != nil {
				log.Error().Msgf("failed to read bundle: %s", err)
				os.Exit(1)
			}
			policies = policyBundle.Policies
		} else {
			opts, err := config.Read()
			if err != nil {
				log.Error().Msgf("failed to get config: %s", err)
				os.Exit(1)
			}
			config.DisplayUsedConfig()

			policyMrn := args[0]
			if !strings.HasPrefix(policyMrn, PolicyMrnPrefix) {
				policyMrn = getPolicyMrn(opts.GetParentMrn(), args[0])
			}

			policyHub, err := getPolicyHubClient(opts)
			if err != nil {
				log.Error().Msgf("failed to create upstream client: %s", err)
				os.Exit(1)
			}

			ctx := context.Background()
			p, err := policyHub.GetPolicy(ctx, &policy.Mrn{Mrn: policyMrn})
			if err != nil {
				log.Error().Msgf("failed to get policy: %s", err)
				os.Exit(1)
			}
			policies = append(policies, p)
		}

		for _, p := range policies {
			checks := map[string]struct{}{}
			queries := map[string]struct{}{}
			referenced := map[string]struct{}{}
			sections := []*policy.PolicyGroup{}
			for _, g := range p.Groups {
				if g.Type == policy.GroupType_CHAPTER || g.Type == policy.GroupType_UNCATEGORIZED {
					sections = append(sections, g)
				}
				for _, c := range g.Checks {
					checks[c.Mrn] = struct{}{}
				}
				for _, q := range g.Queries {
					queries[q.Mrn] = struct{}{}
				}
				for _, p := range g.Policies {
					referenced[p.Mrn] = struct{}{}
				}
			}

			fmt.Println("Name:        " + p.Name)
			fmt.Println("Version:     " + p.Version)
			if p.Uid != "" {
				fmt.Println("UID:         " + p.Uid)
			}
			if p.Mrn != "" {
				fmt.Println("MRN:         " + p.Mrn)
			}
			if len(p.Authors) > 0 {
				fmt.Printf("Authors:     %s <%s>\n", p.Authors[0].Name, p.Authors[0].Email)
				p.Authors = p.Authors[1:]
				for _, a := range p.Authors {
					fmt.Printf("          %s <%s>\n", a.Name, a.Email)
				}
			}
			if p.License != "" {
				fmt.Println("License:     " + p.License)
			} else {
				fmt.Println("License:     none")
			}
			if len(checks) > 0 {
				fmt.Printf("Checks:      %d\n", len(checks))
			}
			if len(queries) > 0 {
				fmt.Printf("Queries:     %d\n", len(queries))
			}
			if len(referenced) > 0 {
				fmt.Printf("Policies:    %d\n", len(referenced))
			}
			if p.Summary != "" {
				fmt.Println("Summary:     " + p.Summary)
			}
			if p.Docs != nil && p.Docs.Desc != "" {
				fmt.Println("Description:")
				fmt.Println(p.Docs.Desc)
			}
			if len(sections) > 0 {
				fmt.Println("Sections:")
				for i, s := range sections {
					fmt.Printf("  %d. %s\n", i+1, s.Title)
				}
			}
			fmt.Println()
			fmt.Println()
		}
	},
}

var policyDownloadCmd = &cobra.Command{
	Use:   "download UID/MRN",
	Short: "download a policy to a local bundle file.",
	Args:  cobra.ExactArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlag("output", cmd.Flags().Lookup("output")); err != nil {
			return err
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		outputFile := viper.GetString("output")
		if outputFile == "" {
			log.Error().Msgf("output file is required")
			os.Exit(1)
		}

		opts, err := config.Read()
		if err != nil {
			log.Error().Msgf("failed to get config: %s", err)
			os.Exit(1)
		}
		config.DisplayUsedConfig()

		policyMrn := args[0]
		if !strings.HasPrefix(policyMrn, PolicyMrnPrefix) {
			policyMrn = getPolicyMrn(opts.GetParentMrn(), args[0])
		}

		policyHub, err := getPolicyHubClient(opts)
		if err != nil {
			log.Error().Msgf("failed to create upstream client: %s", err)
			os.Exit(1)
		}

		ctx := context.Background()
		p, err := policyHub.GetBundle(ctx, &policy.Mrn{Mrn: policyMrn})
		if err != nil {
			log.Error().Msgf("failed to download policy: %s", err)
			os.Exit(1)
		}

		data, err := yaml.Marshal(p)
		if err != nil {
			log.Error().Msgf("failed to marshal policy: %s", err)
			os.Exit(1)
		}
		if err := os.WriteFile(outputFile, data, 0o644); err != nil {
			log.Error().Msgf("failed to store policy: %s", err)
			os.Exit(1)
		}
		log.Info().Msg(theme.DefaultTheme.Success("successfully downloaded to ", outputFile))
	},
}

var policyEnableCmd = &cobra.Command{
	Use:   "enable UID/MRN",
	Short: "Enables a policy in the connected space.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		opts, err := config.Read()
		if err != nil {
			log.Error().Msgf("failed to get config: %s", err)
			os.Exit(1)
		}
		config.DisplayUsedConfig()

		policyMrn := args[0]
		if !strings.HasPrefix(policyMrn, PolicyMrnPrefix) {
			policyMrn = getPolicyMrn(opts.GetParentMrn(), args[0])
		}

		policyResolver, err := getPolicyResolverClient(opts)
		if err != nil {
			log.Error().Msgf("failed to create upstream client: %s", err)
			os.Exit(1)
		}

		ctx := context.Background()
		_, err = policyResolver.Assign(ctx, &policy.PolicyAssignment{
			AssetMrn:   opts.GetParentMrn(),
			PolicyMrns: []string{policyMrn},
		})
		if err != nil {
			log.Error().Msgf("failed to enable policy in space: %s", err)
			os.Exit(1)
		}
		log.Info().Msg(theme.DefaultTheme.Success("successfully enabled policy in space"))
	},
}

var policyDisableCmd = &cobra.Command{
	Use:   "disable UID/MRN",
	Short: "Disables a policy in the connected space.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		opts, err := config.Read()
		if err != nil {
			log.Error().Msgf("failed to get config: %s", err)
			os.Exit(1)
		}
		config.DisplayUsedConfig()

		policyMrn := args[0]
		if !strings.HasPrefix(policyMrn, PolicyMrnPrefix) {
			policyMrn = getPolicyMrn(opts.GetParentMrn(), args[0])
		}

		policyResolver, err := getPolicyResolverClient(opts)
		if err != nil {
			log.Error().Msgf("failed to create upstream client: %s", err)
			os.Exit(1)
		}

		ctx := context.Background()
		_, err = policyResolver.Unassign(ctx, &policy.PolicyAssignment{
			AssetMrn:   opts.GetParentMrn(),
			PolicyMrns: []string{policyMrn},
		})
		if err != nil {
			log.Error().Msgf("failed to disable policy to space: %s", err)
			os.Exit(1)
		}
		log.Info().Msg(theme.DefaultTheme.Success("successfully disabled policy in space"))
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

func getPolicyMrn(spaceMrn, policyUid string) string {
	prefix := strings.Replace(spaceMrn, "captain", "policy", 1)
	return prefix + "/policies/" + policyUid
}

func getPolicyHubClient(opts *config.Config) (*policy.PolicyHubClient, error) {
	certAuth, err := upstream.NewServiceAccountRangerPlugin(opts.GetServiceCredential())
	if err != nil {
		return nil, err
	}

	httpClient, err := opts.GetHttpClient()
	if err != nil {
		return nil, err
	}
	return policy.NewPolicyHubClient(opts.UpstreamApiEndpoint(), httpClient, certAuth)
}

func getPolicyResolverClient(opts *config.Config) (*policy.PolicyResolverClient, error) {
	certAuth, err := upstream.NewServiceAccountRangerPlugin(opts.GetServiceCredential())
	if err != nil {
		return nil, err
	}

	httpClient, err := opts.GetHttpClient()
	if err != nil {
		return nil, err
	}
	return policy.NewPolicyResolverClient(opts.UpstreamApiEndpoint(), httpClient, certAuth)
}
