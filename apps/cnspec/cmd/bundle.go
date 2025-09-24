// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"context"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.mondoo.com/cnquery/v12/cli/config"
	"go.mondoo.com/cnquery/v12/providers"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/upstream"
	"go.mondoo.com/cnspec/v12/internal/bundle"
	"go.mondoo.com/cnspec/v12/policy"
)

func init() {
	// policy init
	policyBundlesCmd.AddCommand(policyInitDeprecatedCmd)

	// validate
	policyLintDeprecatedCmd.Flags().StringP("output", "o", "cli", "Set output format: compact, sarif")
	policyLintDeprecatedCmd.Flags().String("output-file", "", "Set output file")
	policyBundlesCmd.AddCommand(policyLintDeprecatedCmd)

	// fmt
	policyFmtDeprecatedCmd.Flags().Bool("sort", false, "sort the bundle.")
	policyBundlesCmd.AddCommand(policyFmtDeprecatedCmd)

	// docs
	policyDocsDeprecatedCmd.Flags().Bool("no-code", false, "enable/disable code blocks inside of docs")
	policyDocsDeprecatedCmd.Flags().Bool("no-ids", false, "enable/disable the printing of ID fields")
	policyBundlesCmd.AddCommand(policyDocsDeprecatedCmd)

	// publish
	policyPublishCmd.Flags().Bool("no-lint", false, "Disable linting of the bundle before publishing.")
	policyPublishCmd.Flags().String("policy-version", "", "Override the version of each policy in the bundle.")
	policyBundlesCmd.AddCommand(policyPublishCmd)

	rootCmd.AddCommand(policyBundlesCmd)
}

var policyInitDeprecatedCmd = &cobra.Command{
	Use:        "init [path]",
	Short:      "Create an example policy bundle",
	Long:       "Create an example policy bundle that you can use as a starting point. If you don't provide a filename, cnspec uses `example-policy.mql.yml`.",
	Aliases:    []string{"new"},
	Hidden:     true,
	Deprecated: "use `cnspec policy init` instead",
	Args:       cobra.MaximumNArgs(1),
	Run:        runPolicyInit,
}

var policyFmtDeprecatedCmd = &cobra.Command{
	Use:        "format [path]",
	Aliases:    []string{"fmt"},
	Hidden:     true,
	Deprecated: "use `cnspec policy fmt` instead",
	Short:      "Apply style formatting to one or more policy bundles",
	Args:       cobra.MinimumNArgs(1),
	Run:        runPolicyFmt,
}

// ensureProviders ensures that all providers are locally installed
func ensureProviders() error {
	for _, v := range providers.DefaultProviders {
		if _, err := providers.EnsureProvider(providers.ProviderLookup{ID: v.ID}, true, nil); err != nil {
			return err
		}
	}
	return nil
}

var policyBundlesCmd = &cobra.Command{
	Use:        "bundle",
	Hidden:     true,
	Deprecated: "use `cnspec policy` instead",
	Short:      "Manage policy bundles",
}

var policyLintDeprecatedCmd = &cobra.Command{
	Use:        "lint [path]",
	Aliases:    []string{"validate"},
	Hidden:     true,
	Deprecated: "use `cnspec policy lint` instead",
	Short:      "Lint a policy bundle",
	Args:       cobra.ExactArgs(1),
	PreRun: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("output", cmd.Flags().Lookup("output"))
		viper.BindPFlag("output-file", cmd.Flags().Lookup("output-file"))
	},
	Run: runPolicyLint,
}

var policyPublishCmd = &cobra.Command{
	Use:        "publish [path]",
	Aliases:    []string{"upload"},
	Hidden:     true,
	Deprecated: "use `cnspec policy upload` instead",
	Short:      "Add a user-owned policy to the Mondoo Security Registry",
	Args:       cobra.ExactArgs(1),
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

var policyDocsDeprecatedCmd = &cobra.Command{
	Use:        "docs [path]",
	Aliases:    []string{},
	Hidden:     true,
	Deprecated: "use `cnspec policy docs` instead",
	Short:      "Retrieve only the docs for a bundle",
	Args:       cobra.MinimumNArgs(1),
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
