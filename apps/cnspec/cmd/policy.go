// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.mondoo.com/cnquery/v9/cli/config"
	"go.mondoo.com/cnquery/v9/providers"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/upstream"
	"go.mondoo.com/cnspec/v9/internal/bundle"
	"go.mondoo.com/cnspec/v9/policy"
	"go.mondoo.com/cnspec/v9/upstream/gql"
)

const (
	defaultRegistryUrl = "https://registry.api.mondoo.com"
)

func init() {
	// policy init
	policyCmd.AddCommand(policyInitCmd)

	// policy list
	policyCmd.AddCommand(policyListCmd)
	policyListCmd.Flags().StringP("file", "f", "", "list policies in a bundle file")
	policyListCmd.Flags().BoolP("all", "a", false, "list all policies including disabled ones")
	policyListCmd.Flags().StringP("type", "t", "POLICY", "Either POLICY or QUERYPACK (defaults to POLICY)")

	// policy upload
	policyCmd.AddCommand(policyUploadCmd)
	policyUploadCmd.Flags().StringP("file", "f", "", "upload policies in a bundle file")

	// policy download
	policyCmd.AddCommand(policyDownloadCmd)
	policyDownloadCmd.Flags().StringP("file", "f", "", "save policies in a bundle file")

	// policy show
	policyCmd.AddCommand(policyShowCmd)
	policyShowCmd.Flags().StringP("file", "f", "", "show policies in a bundle file")

	// policy enable
	policyCmd.AddCommand(policyEnableCmd)

	// policy disable
	policyCmd.AddCommand(policyDisableCmd)

	// policy delete
	policyCmd.AddCommand(policyDeleteCmd)

	// validate
	policyLintCmd.Flags().StringP("output", "o", "cli", "Set output format: compact, sarif")
	policyLintCmd.Flags().String("output-file", "", "Set output file")
	policyCmd.AddCommand(policyLintCmd)

	// fmt
	policyFmtCmd.Flags().Bool("sort", false, "sort the bundle.")
	policyCmd.AddCommand(policyFmtCmd)

	// docs
	policyDocsCmd.Flags().Bool("no-code", false, "enable/disable code blocks inside of docs")
	policyDocsCmd.Flags().Bool("no-ids", false, "enable/disable the printing of ID fields")
	policyCmd.AddCommand(policyDocsCmd)

	// publish
	policyPublishCmd.Flags().Bool("no-lint", false, "Disable linting of the bundle before publishing.")
	policyPublishCmd.Flags().String("policy-version", "", "Override the version of each policy in the bundle.")
	policyCmd.AddCommand(policyPublishCmd)

	rootCmd.AddCommand(policyCmd)
}

var policyCmd = &cobra.Command{
	Use:   "policy",
	Short: "Manage policies.",
}

var policyListCmd = &cobra.Command{
	Use:   "list",
	Short: "list currently active policies in the connected space",
	Args:  cobra.MaximumNArgs(0),
	PreRun: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("file", cmd.Flags().Lookup("file"))
		viper.BindPFlag("all", cmd.Flags().Lookup("all"))
		viper.BindPFlag("type", cmd.Flags().Lookup("type"))
	},
	Run: func(cmd *cobra.Command, args []string) {
		opts, optsErr := config.Read()
		if optsErr != nil {
			log.Fatal().Err(optsErr).Msg("could not load configuration")
		}
		config.DisplayUsedConfig()

		serviceAccount := opts.GetServiceCredential()
		if serviceAccount == nil {
			log.Fatal().Msg("cnspec has no credentials. Log in with `cnspec login`")
		}

		httpClient, err := opts.GetHttpClient()
		if err != nil {
			log.Fatal().Err(err).Msg("error while creating Mondoo API client")
		}
		if err != nil {
			log.Fatal().Err(err).Msg("could not connect to the Mondoo Security Registry")
		}

		bundleFile := viper.GetString("file")

		if bundleFile != "" {
			bundleLoader := policy.DefaultBundleLoader()
			policyBundle, err := bundleLoader.BundleFromPaths(bundleFile)
			if err != nil {
				log.Fatal().Err(err)
			}
			for _, policy := range policyBundle.Policies {
				fmt.Println(policy.Name + " " + policy.Version)
				// Printing policy MRN in gray
				if policy.Mrn != "" {
					fmt.Printf("\033[90m  %s\033[0m\n", policy.Mrn)
				} else {
					fmt.Printf("\033[90m  %s\033[0m\n", policy.Uid)
				}
			}
		} else {
			upstreamConfig := upstream.UpstreamConfig{
				ApiEndpoint: opts.UpstreamApiEndpoint(),
				Creds:       serviceAccount,
			}

			mondooClient, err := gql.NewClient(upstreamConfig, httpClient)
			if err != nil {
				return
			}
			all := viper.GetBool("all")
			catalogType := viper.GetString("type")
			policiesList, err := mondooClient.SearchPolicy(opts.SpaceMrn, !all, catalogType)
			if err != nil {
				log.Fatal().Err(err).Msg("Failed to get space report")
			}
			for _, policy := range policiesList.Edges {
				fmt.Println(policy.Node.Name + " " + policy.Node.Version)
				// Printing policy MRN in gray
				if policy.Node.MRN != "" {
					fmt.Printf("\033[90m  %s\033[0m\n", policy.Node.MRN)
				} else {
					fmt.Printf("\033[90m  %s\033[0m\n", *policy.Node.UID)
				}
			}
		}
	},
}

var policyShowCmd = &cobra.Command{
	Use:   "show [UID/MRN]",
	Short: "show more info about policies, including: summary, docs, etc.",
	Args:  cobra.MinimumNArgs(1),
	PreRun: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("file", cmd.Flags().Lookup("file"))
	},
	Run: func(cmd *cobra.Command, args []string) {
		opts, optsErr := config.Read()
		if optsErr != nil {
			log.Fatal().Err(optsErr).Msg("could not load configuration")
		}
		config.DisplayUsedConfig()

		serviceAccount := opts.GetServiceCredential()
		if serviceAccount == nil {
			log.Fatal().Msg("cnspec has no credentials. Log in with `cnspec login`")
		}

		certAuth, err := upstream.NewServiceAccountRangerPlugin(serviceAccount)
		if err != nil {
			log.Error().Err(err).Msg(errorMessageServiceAccount)
			os.Exit(ConfigurationErrorCode)
		}

		httpClient, err := opts.GetHttpClient()
		if err != nil {
			log.Fatal().Err(err).Msg("error while creating Mondoo API client")
		}
		client, err := policy.NewPolicyHubClient(opts.UpstreamApiEndpoint(), httpClient, certAuth)
		if err != nil {
			log.Fatal().Err(err).Msg("could not connect to the Mondoo Security Registry")
		}

		bundleFile := viper.GetString("file")
		var policies []*policy.Policy

		if bundleFile != "" {
			bundleLoader := policy.DefaultBundleLoader()
			policyBundle, err := bundleLoader.BundleFromPaths(bundleFile)
			if err != nil {
				log.Fatal().Err(err)
			}
			policies = policyBundle.Policies
		} else {
			policyMrn := &policy.Mrn{
				Mrn: args[0],
			}
			policy, err := client.GetPolicy(context.Background(), policyMrn)
			if err != nil {
				log.Fatal().Err(err)
			}
			if policy == nil {
				log.Fatal().Msg("Failed to get policy")
			}
			if policy.Mrn == "" {
				log.Fatal().Msg("Something went wrong")
			}
			policies = append(policies, policy)
		}

		for _, p := range policies {
			var policyName string
			if p.Mrn != "" {
				index := strings.LastIndex(p.Mrn, "/")
				if index == -1 {
					policyName = p.Mrn
				} else {
					policyName = p.Mrn[index+1:]
				}
			} else {
				policyName = ""
			}
			fmt.Println("→ Name:      ", p.Name)
			fmt.Println("→ Version:   ", p.Version)
			fmt.Println("→ UID:       ", policyName)
			fmt.Println("→ MRN:       ", p.Mrn)
			fmt.Println("→ License:   ", p.License)
			fmt.Println("→ Authors:   ", p.Authors[0].Name)
			if len(p.Authors) > 1 {
				for i := range p.Authors {
					if i == 0 {
						continue
					}
					fmt.Println("             ", p.Authors[i].Name)
				}
			}
			if p.QueryCounts.TotalCount > 0 {
				fmt.Println("→ Checks:    ", p.QueryCounts.TotalCount)
			}
			if p.QueryCounts.DataCount > 0 {
				fmt.Println("→ Querys:    ", p.QueryCounts.DataCount)
			}
			if len(p.DependentPolicyMrns()) > 0 {
				fmt.Println("→ Policies:  ", len(p.DependentPolicyMrns()))
			}
			if p.Summary != "" {
				fmt.Println("→ Summary:   ", p.Summary)
			}
			if p.Docs.Desc != "" {
				fmt.Println("→ Description:")
				fmt.Println(p.Docs.Desc)
			}

			var sections []string
			for _, group := range p.Groups {
				if group.Type == policy.GroupType_CHAPTER {
					sections = append(sections, group.Title)
				}
			}
			if len(sections) > 0 {
				fmt.Println()
				fmt.Println("→ Sections:")
				for i, section := range sections {
					fmt.Printf("%d. %s", i, section)
				}

			}
		}
	},
}

var policyDeleteCmd = &cobra.Command{
	Use:   "delete [UID/MRN]",
	Short: "remove a policy from the connected space",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		opts, optsErr := config.Read()
		if optsErr != nil {
			log.Fatal().Err(optsErr).Msg("could not load configuration")
		}
		config.DisplayUsedConfig()

		serviceAccount := opts.GetServiceCredential()
		if serviceAccount == nil {
			log.Fatal().Msg("cnspec has no credentials. Log in with `cnspec login`")
		}

		certAuth, err := upstream.NewServiceAccountRangerPlugin(serviceAccount)
		if err != nil {
			log.Error().Err(err).Msg(errorMessageServiceAccount)
			os.Exit(ConfigurationErrorCode)
		}

		httpClient, err := opts.GetHttpClient()
		if err != nil {
			log.Fatal().Err(err).Msg("error while creating Mondoo API client")
		}
		client, err := policy.NewPolicyHubClient(opts.UpstreamApiEndpoint(), httpClient, certAuth)
		if err != nil {
			log.Fatal().Err(err).Msg("could not connect to the Mondoo Security Registry")
		}

		policyMrn := &policy.Mrn{
			Mrn: args[0],
		}
		policy, err := client.GetPolicy(context.Background(), policyMrn)
		if err != nil {
			log.Fatal().Err(err)
		}
		if policy.Mrn == "" {
			log.Fatal().Msg("Policy not found")
		}
		_, err = client.DeletePolicy(context.Background(), policyMrn)
		if err != nil {
			log.Fatal().Err(err)
		}

		var spaceName string
		index := strings.LastIndex(opts.SpaceMrn, "/")
		if index == -1 {
			spaceName = opts.SpaceMrn
		} else {
			spaceName = opts.SpaceMrn[index+1:]
		}

		// Success message in green
		fmt.Printf("\033[32m→ successfully removed policy from space\033[0m\n")
		fmt.Println("  policy: " + policy.Name + " " + policy.Version)
		fmt.Printf("\033[90m          %s\033[0m\n", policy.Mrn)
		fmt.Println("  space: " + spaceName)
		fmt.Printf("\033[90m          %s\033[0m\n", opts.SpaceMrn)
	},
}

var policyUploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "upload a policy to the connected space",
	Args:  cobra.MinimumNArgs(1),
	PreRun: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("file", cmd.Flags().Lookup("file"))
	},
	Run: func(cmd *cobra.Command, args []string) {
		policyFile := ""
		if len(args) == 1 {
			policyFile = args[0]
		}

		opts, optsErr := config.Read()
		if optsErr != nil {
			log.Fatal().Err(optsErr).Msg("could not load configuration")
		}
		config.DisplayUsedConfig()

		serviceAccount := opts.GetServiceCredential()
		if serviceAccount == nil {
			log.Fatal().Msg("cnspec has no credentials. Log in with `cnspec login`")
		}

		certAuth, err := upstream.NewServiceAccountRangerPlugin(serviceAccount)
		if err != nil {
			log.Error().Err(err).Msg(errorMessageServiceAccount)
			os.Exit(ConfigurationErrorCode)
		}

		httpClient, err := opts.GetHttpClient()
		if err != nil {
			log.Fatal().Err(err).Msg("error while creating Mondoo API client")
		}
		client, err := policy.NewPolicyHubClient(opts.UpstreamApiEndpoint(), httpClient, certAuth)
		if err != nil {
			log.Fatal().Err(err).Msg("could not connect to the Mondoo Security Registry")
		}

		bundleLoader := policy.DefaultBundleLoader()
		policyBundle, err := bundleLoader.BundleFromPaths(policyFile)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to load bundle from file")
		}

		_, err = client.SetBundle(context.Background(), policyBundle)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to upload policies")
		}
		// Success message in green
		fmt.Printf("\033[32m→ successfully uploaded %d policies to the space\033[0m", len(policyBundle.Policies))
		for _, policy := range policyBundle.Policies {
			fmt.Println("  policy: " + policy.Name + " " + policy.Version)
			fmt.Printf("\033[90m          %s\033[0m\n", policy.Mrn)
		}
	},
}

var policyEnableCmd = &cobra.Command{
	Use:   "enable [MRN/UID]",
	Short: "enable a policy in the connected space",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		opts, optsErr := config.Read()
		if optsErr != nil {
			log.Fatal().Err(optsErr).Msg("could not load configuration")
		}
		config.DisplayUsedConfig()

		policyMrn := ""
		if len(args) == 1 {
			policyMrn = args[0]
		}

		err := policy.IsPolicyMrn(policyMrn)
		if err != nil {
			// if the user provided a UID we must construct the MRN
			policyMrnPrefix := "//policy.api.mondoo.app/policies/"
			policyMrn = policyMrnPrefix + "/" + policyMrn
		}

		serviceAccount := opts.GetServiceCredential()
		if serviceAccount == nil {
			log.Fatal().Msg("cnspec has no credentials. Log in with `cnspec login`")
		}

		certAuth, err := upstream.NewServiceAccountRangerPlugin(serviceAccount)
		if err != nil {
			log.Error().Err(err).Msg(errorMessageServiceAccount)
			os.Exit(ConfigurationErrorCode)
		}

		httpClient, err := opts.GetHttpClient()
		if err != nil {
			log.Fatal().Err(err).Msg("error while creating Mondoo API client")
		}
		clientResolver, err := policy.NewPolicyResolverClient(opts.UpstreamApiEndpoint(), httpClient, certAuth)
		if err != nil {
			log.Fatal().Err(err).Msg("could not connect to the Mondoo Security Registry")
		}

		policyMrns := []string{policyMrn}
		_, err = clientResolver.Assign(context.Background(), &policy.PolicyAssignment{PolicyMrns: policyMrns, AssetMrn: opts.SpaceMrn})
		if err != nil {
			log.Error().Err(err).Msg("Failed to enable policy")
			return
		}

		clientHub, err := policy.NewPolicyHubClient(opts.UpstreamApiEndpoint(), httpClient, certAuth)
		if err != nil {
			log.Fatal().Err(err).Msg("could not connect to the Mondoo Security Registry")
		}
		p, err := clientHub.GetPolicy(context.Background(), &policy.Mrn{Mrn: policyMrn})
		if err != nil {
			log.Error().Err(err).Msg("Failed to get policy")
			return
		}

		// Success message in green
		fmt.Printf("\033[32m→ successfully enabled policy\033[0m\n")
		fmt.Println("  policy: " + p.Name + " " + p.Version)
		fmt.Printf("\033[90m          %s\033[0m\n", p.Mrn)
	},
}

var policyDisableCmd = &cobra.Command{
	Use:   "disable [MRN/UID]",
	Short: "disable a policy in the connected space",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		opts, optsErr := config.Read()
		if optsErr != nil {
			log.Fatal().Err(optsErr).Msg("could not load configuration")
		}
		config.DisplayUsedConfig()

		policyMrn := ""
		if len(args) == 1 {
			policyMrn = args[0]
		}

		err := policy.IsPolicyMrn(policyMrn)
		if err != nil {
			// if the user provided a UID we must construct the MRN
			policyMrnPrefix := "//policy.api.mondoo.app/policies/"
			policyMrn = policyMrnPrefix + "/" + policyMrn
		}

		serviceAccount := opts.GetServiceCredential()
		if serviceAccount == nil {
			log.Fatal().Msg("cnspec has no credentials. Log in with `cnspec login`")
		}

		certAuth, err := upstream.NewServiceAccountRangerPlugin(serviceAccount)
		if err != nil {
			log.Error().Err(err).Msg(errorMessageServiceAccount)
			os.Exit(ConfigurationErrorCode)
		}

		httpClient, err := opts.GetHttpClient()
		if err != nil {
			log.Fatal().Err(err).Msg("error while creating Mondoo API client")
		}
		clientResolver, err := policy.NewPolicyResolverClient(opts.UpstreamApiEndpoint(), httpClient, certAuth)
		if err != nil {
			log.Fatal().Err(err).Msg("could not connect to the Mondoo Security Registry")
		}

		policyMrns := []string{policyMrn}
		_, err = clientResolver.Unassign(context.Background(), &policy.PolicyAssignment{PolicyMrns: policyMrns, AssetMrn: opts.SpaceMrn})
		if err != nil {
			log.Error().Err(err).Msg("Failed to disable policy")
			return
		}

		clientHub, err := policy.NewPolicyHubClient(opts.UpstreamApiEndpoint(), httpClient, certAuth)
		if err != nil {
			log.Fatal().Err(err).Msg("could not connect to the Mondoo Security Registry")
		}
		p, err := clientHub.GetPolicy(context.Background(), &policy.Mrn{Mrn: policyMrn})
		if err != nil {
			log.Error().Err(err).Msg("Failed to get policy")
			return
		}

		// Success message in green
		fmt.Printf("\033[32m→ successfully disabled policy\033[0m\n")
		fmt.Println("  policy: " + p.Name + " " + p.Version)
		fmt.Printf("\033[90m          %s\033[0m\n", p.Mrn)
	},
}

var policyDownloadCmd = &cobra.Command{
	Use:   "download [MRN/UID]",
	Short: "download a policy from the connected space",
	Args:  cobra.MinimumNArgs(1),
	PreRun: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("file", cmd.Flags().Lookup("file"))
	},
	Run: func(cmd *cobra.Command, args []string) {
		opts, optsErr := config.Read()
		if optsErr != nil {
			log.Fatal().Err(optsErr).Msg("could not load configuration")
		}
		config.DisplayUsedConfig()

		bundleFile := viper.GetString("file")
		if bundleFile == "" {
			log.Fatal().Msg("Output file must be provided via flag --file/-f")
		}

		policyMrn := ""
		if len(args) == 1 {
			policyMrn = args[0]
		}

		err := policy.IsPolicyMrn(policyMrn)
		if err != nil {
			// if the user provided a UID we must construct the MRN
			policyMrnPrefix := "//policy.api.mondoo.app/policies/"
			policyMrn = policyMrnPrefix + "/" + policyMrn
		}

		registryEndpoint := os.Getenv("REGISTRY_URL")
		if registryEndpoint == "" {
			registryEndpoint = defaultRegistryUrl
		}

		serviceAccount := opts.GetServiceCredential()
		if serviceAccount == nil {
			log.Fatal().Msg("cnspec has no credentials. Log in with `cnspec login`")
		}

		certAuth, err := upstream.NewServiceAccountRangerPlugin(serviceAccount)
		if err != nil {
			log.Error().Err(err).Msg(errorMessageServiceAccount)
			os.Exit(ConfigurationErrorCode)
		}

		httpClient, err := opts.GetHttpClient()
		if err != nil {
			log.Fatal().Err(err).Msg("error while creating Mondoo API client")
		}
		client, err := policy.NewPolicyHubClient(opts.UpstreamApiEndpoint(), httpClient, certAuth)
		if err != nil {
			log.Fatal().Err(err).Msg("could not connect to the Mondoo Security Registry")
		}

		pb, err := client.GetBundle(context.Background(), &policy.Mrn{Mrn: policyMrn})
		if err != nil {
			log.Error().Err(err).Msg("Failed to download policy")
		}
		policyYaml, err := pb.ToYAML()
		if err != nil {
			log.Error().Err(err).Msg("Failed to convert policy to yaml")
		}
		err = os.WriteFile(bundleFile, policyYaml, 0o640)
		if err != nil {
			log.Error().Err(err).Msg("Failed to write policy bundle to file")
		}

		// Success message in green
		fmt.Printf("\033[32m→ successfully downloaded %d policies to the space\033[0m\n", len(pb.Policies))
		for _, policy := range pb.Policies {
			fmt.Println("  policy: " + policy.Name + " " + policy.Version)
			fmt.Printf("\033[90m          %s\033[0m\n", policy.Mrn)
		}
	},
}

var policyInitCmd = &cobra.Command{
	Use:   "init [path]",
	Short: "Create an example policy that you can use as a starting point. If you don't provide a filename, cnspec uses `example-policy.mql.yml`.",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := "example-policy.mql.yaml"
		if len(args) == 1 {
			name = args[0]
		}

		_, err := os.Stat(name)
		if err == nil {
			log.Fatal().Msgf("Policy '%s' already exists", name)
		}

		err = os.WriteFile(name, embedPolicyTemplate, 0o640)
		if err != nil {
			log.Fatal().Err(err).Msgf("Could not write '%s'", name)
		}
		log.Info().Msgf("Example policy file written to %s", name)
	},
}

var policyLintCmd = &cobra.Command{
	Use:     "lint [path]",
	Aliases: []string{"validate"},
	Short:   "Lint a policy.",
	Args:    cobra.ExactArgs(1),
	PreRun: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("output", cmd.Flags().Lookup("output"))
		viper.BindPFlag("output-file", cmd.Flags().Lookup("output-file"))
	},
	Run: func(cmd *cobra.Command, args []string) {
		log.Info().Str("file", args[0]).Msg("lint policy bundle")
		ensureProviders()

		files, err := policy.WalkPolicyBundleFiles(args[0])
		if err != nil {
			log.Fatal().Err(err).Msg("could not find bundle files")
		}

		runtime := providers.DefaultRuntime()
		result, err := bundle.Lint(runtime.Schema(), files...)
		if err != nil {
			log.Fatal().Err(err).Msg("could not lint bundle files")
		}

		out := os.Stdout
		if viper.GetString("output-file") != "" {
			out, err = os.Create(viper.GetString("output-file"))
			if err != nil {
				log.Fatal().Err(err).Msg("could not create output file")
			}
			defer out.Close()
		}

		switch viper.GetString("output") {
		case "cli":
			out.Write(result.ToCli())
		case "sarif":
			data, err := result.ToSarif(filepath.Dir(args[0]))
			if err != nil {
				log.Fatal().Err(err).Msg("could not generate sarif report")
			}
			out.Write(data)
		}

		if viper.GetString("output-file") == "" {
			if result.HasError() {
				log.Fatal().Msg("invalid policy bundle")
			} else {
				log.Info().Msg("valid policy bundle")
			}
		}
	},
}

var policyFmtCmd = &cobra.Command{
	Use:     "format [path]",
	Aliases: []string{"fmt"},
	Short:   "Apply style formatting to one or more policy bundles.",
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		sort, _ := cmd.Flags().GetBool("sort")
		ensureProviders()
		for _, path := range args {
			err := bundle.FormatRecursive(path, sort)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		}
		log.Info().Msg("completed formatting policy bundle(s)")
	},
}

var policyPublishCmd = &cobra.Command{
	Use: "publish [path]",
	//Aliases: []string{"upload"},
	Short: "Add a user-owned policy to the Mondoo Security Registry.",
	Args:  cobra.ExactArgs(1),
	PreRun: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("policy-version", cmd.Flags().Lookup("policy-version"))
		viper.BindPFlag("no-lint", cmd.Flags().Lookup("no-lint"))
	},
	Run: func(cmd *cobra.Command, args []string) {
		opts, optsErr := config.Read()
		if optsErr != nil {
			log.Fatal().Err(optsErr).Msg("could not load configuration")
		}
		config.DisplayUsedConfig()

		ensureProviders()

		filename := args[0]
		log.Info().Str("file", filename).Msg("load policy bundle")
		files, err := policy.WalkPolicyBundleFiles(args[0])
		if err != nil {
			log.Fatal().Err(err).Msg("could not find bundle files")
		}

		noLint := viper.GetBool("no-lint")
		if !noLint {
			runtime := providers.DefaultRuntime()
			result, err := bundle.Lint(runtime.Schema(), files...)
			if err != nil {
				log.Fatal().Err(err).Msg("could not lint bundle files")
			}

			// render cli output
			os.Stdout.Write(result.ToCli())

			if result.HasError() {
				log.Fatal().Msg("invalid policy bundle")
			} else {
				log.Info().Msg("valid policy bundle")
			}
		}

		// compile manipulates the bundle, therefore we read it again
		bundleLoader := policy.DefaultBundleLoader()
		policyBundle, err := bundleLoader.BundleFromPaths(filename)
		if err != nil {
			log.Fatal().Err(err).Msg("could not load policy bundle")
		}

		log.Info().Str("space", opts.SpaceMrn).Msg("add policy bundle to space")
		overrideVersionFlag := false
		overrideVersion := viper.GetString("policy-version")
		if len(overrideVersion) > 0 {
			overrideVersionFlag = true
		}

		serviceAccount := opts.GetServiceCredential()
		if serviceAccount == nil {
			log.Fatal().Msg("cnspec has no credentials. Log in with `cnspec login`")
		}

		certAuth, err := upstream.NewServiceAccountRangerPlugin(serviceAccount)
		if err != nil {
			log.Error().Err(err).Msg(errorMessageServiceAccount)
			os.Exit(ConfigurationErrorCode)
		}

		httpClient, err := opts.GetHttpClient()
		if err != nil {
			log.Fatal().Err(err).Msg("error while creating Mondoo API client")
		}
		queryHubServices, err := policy.NewPolicyHubClient(opts.UpstreamApiEndpoint(), httpClient, certAuth)
		if err != nil {
			log.Fatal().Err(err).Msg("could not connect to the Mondoo Security Registry")
		}

		// set the owner mrn for spaces
		policyBundle.OwnerMrn = opts.SpaceMrn
		ctx := context.Background()

		// override version and/or labels
		for i := range policyBundle.Policies {
			p := policyBundle.Policies[i]

			// override policy version
			if overrideVersionFlag {
				p.Version = overrideVersion
			}
		}

		// send data upstream
		_, err = queryHubServices.SetBundle(ctx, policyBundle)
		if err != nil {
			log.Fatal().Err(err).Msg("could not add policy bundle")
		}

		log.Info().Msg("successfully added policies")
	},
}

var policyDocsCmd = &cobra.Command{
	Use:     "docs [path]",
	Aliases: []string{},
	Short:   "Retrieve only the docs for a bundle.",
	Args:    cobra.MinimumNArgs(1),
	Hidden:  true,
	PreRun: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("no-ids", cmd.Flags().Lookup("no-ids"))
		viper.BindPFlag("no-code", cmd.Flags().Lookup("no-code"))
	},
	Run: func(cmd *cobra.Command, args []string) {
		bundleLoader := policy.DefaultBundleLoader()
		bundle, err := bundleLoader.BundleFromPaths(args...)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to load bundle")
		}

		noIDs := viper.GetBool("no-ids")
		noCode := viper.GetBool("no-code")
		bundle.ExtractDocs(os.Stdout, noIDs, noCode)
	},
}
