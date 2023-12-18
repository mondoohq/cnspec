// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	_ "embed"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.mondoo.com/cnquery/v9/providers"
)

func init() {
	// policy init
	policyBundlesCmd.AddCommand(bundleInitCmd)

	// validate
	bundleLintCmd.Flags().StringP("output", "o", "cli", "Set output format: compact, sarif")
	bundleLintCmd.Flags().String("output-file", "", "Set output file")
	policyBundlesCmd.AddCommand(bundleLintCmd)

	// fmt
	bundleFmtCmd.Flags().Bool("sort", false, "sort the bundle.")
	policyBundlesCmd.AddCommand(bundleFmtCmd)

	// docs
	bundleDocsCmd.Flags().Bool("no-code", false, "enable/disable code blocks inside of docs")
	bundleDocsCmd.Flags().Bool("no-ids", false, "enable/disable the printing of ID fields")
	policyBundlesCmd.AddCommand(bundleDocsCmd)

	// publish
	bundlePublishCmd.Flags().Bool("no-lint", false, "Disable linting of the bundle before publishing.")
	bundlePublishCmd.Flags().String("policy-version", "", "Override the version of each policy in the bundle.")
	policyBundlesCmd.AddCommand(bundlePublishCmd)

	rootCmd.AddCommand(policyBundlesCmd)
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
	Use:   "bundle",
	Short: "Manage policy bundles.",
}

//go:embed policy-example.mql.yaml
var embedPolicyTemplate []byte

var bundleInitCmd = &cobra.Command{
	Use:   "init [path]",
	Short: "Create an example policy bundle that you can use as a starting point. If you don't provide a filename, cnspec uses `example-policy.mql.yml`.",
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

var bundleLintCmd = &cobra.Command{
	Use:     "lint [path]",
	Aliases: []string{"validate"},
	Short:   "Lint a policy bundle.",
	Args:    cobra.ExactArgs(1),
	PreRun: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("output", cmd.Flags().Lookup("output"))
		viper.BindPFlag("output-file", cmd.Flags().Lookup("output-file"))
	},
	Run: func(cmd *cobra.Command, args []string) {
		log.Warn().Msg("bundle command has been deprecated, use policy instead")
		policyLintCmd.Run(cmd, args)
	},
}

var bundleFmtCmd = &cobra.Command{
	Use:     "format [path]",
	Aliases: []string{"fmt"},
	Short:   "Apply style formatting to one or more policy bundles.",
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		log.Warn().Msg("bundle command has been deprecated, use policy instead")
		policyFmtCmd.Run(cmd, args)
	},
}

var bundlePublishCmd = &cobra.Command{
	Use:     "publish [path]",
	Aliases: []string{"upload"},
	Short:   "Add a user-owned policy to the Mondoo Security Registry.",
	Args:    cobra.ExactArgs(1),
	PreRun: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("policy-version", cmd.Flags().Lookup("policy-version"))
		viper.BindPFlag("no-lint", cmd.Flags().Lookup("no-lint"))
	},
	Run: func(cmd *cobra.Command, args []string) {
		log.Warn().Msg("bundle command has been deprecated, use policy instead")
		policyPublishCmd.Run(cmd, args)
	},
}

var bundleDocsCmd = &cobra.Command{
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
		log.Warn().Msg("bundle command has been deprecated, use policy instead")
		policyDocsCmd.Run(cmd, args)
	},
}
