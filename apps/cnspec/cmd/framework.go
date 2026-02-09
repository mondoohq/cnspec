// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

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
	"go.mondoo.com/cnquery/v12/cli/config"
	"go.mondoo.com/cnquery/v12/cli/theme"
	"go.mondoo.com/cnspec/v12/policy"
	cnspec_upstream "go.mondoo.com/cnspec/v12/upstream"
	mondoogql "go.mondoo.com/mondoo-go"
)

const (
	FrameworkMrnPrefix = "//policy.api.mondoo.app/frameworks"
)

func init() {
	rootCmd.AddCommand(frameworkCmd)

	// list
	frameworkListCmd.Flags().StringP("file", "f", "", "Set the path to a local bundle file")
	frameworkListCmd.Flags().BoolP("all", "a", false, "List all frameworks, not only the active ones (applicable only for upstream)")
	frameworkCmd.AddCommand(frameworkListCmd)

	// preview
	frameworkCmd.AddCommand(frameworkPreviewCmd)
	// active
	frameworkCmd.AddCommand(frameworkActiveCmd)
	// disabled
	frameworkCmd.AddCommand(frameworkDisabledCmd)
	// download
	frameworkDownloadCmd.Flags().StringP("file", "f", "", "Set the output file")
	frameworkCmd.AddCommand(frameworkDownloadCmd)
	// upload
	frameworkUploadCmd.Flags().StringP("file", "f", "", "Set the input file")
	frameworkCmd.AddCommand(frameworkUploadCmd)
}

var frameworkCmd = &cobra.Command{
	Use:     "framework",
	Short:   "Manage local and Mondoo Platform hosted compliance frameworks",
	Aliases: []string{"frameworks"},
}

var frameworkListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available compliance frameworks",
	Args:  cobra.MaximumNArgs(0),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlag("file", cmd.Flags().Lookup("file")); err != nil {
			return err
		}
		if err := viper.BindPFlag("all", cmd.Flags().Lookup("all")); err != nil {
			return err
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		bundleFile := viper.GetString("file")
		var frameworks []*cnspec_upstream.UpstreamFramework

		if bundleFile != "" {
			policyBundle, err := policy.DefaultBundleLoader().BundleFromPaths(bundleFile)
			if err != nil {
				return err
			}
			for _, f := range policyBundle.Frameworks {
				frameworks = append(frameworks, &cnspec_upstream.UpstreamFramework{Framework: *f})
			}
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

			states := []mondoogql.ComplianceFrameworkState{mondoogql.ComplianceFrameworkStateActive}
			if viper.GetBool("all") {
				states = []mondoogql.ComplianceFrameworkState{}
			}

			frameworks, err = cnspec_upstream.ListFrameworks(context.Background(), mondooClient, opts.GetParentMrn(), states)
			if err != nil {
				log.Error().Msgf("failed to list compliance frameworks: %s", err)
				os.Exit(1)
				return err
			}
		}

		for _, framework := range frameworks {
			extraInfo := []string{}
			if framework.State == mondoogql.ComplianceFrameworkStateActive {
				extraInfo = append(extraInfo, theme.DefaultTheme.Success("active"))
			} else if framework.State == mondoogql.ComplianceFrameworkStatePreview {
				extraInfo = append(extraInfo, theme.DefaultTheme.PolicyPrinter.Yellow("preview"))
			} else if framework.State == mondoogql.ComplianceFrameworkStateDisabled {
				extraInfo = append(extraInfo, theme.DefaultTheme.Disabled("disabled"))
			} else if framework.State == mondoogql.ComplianceFrameworkState("") {
				extraInfo = append(extraInfo, theme.DefaultTheme.Disabled("local"))
			}

			extraInfoStr := ""
			if len(extraInfo) > 0 {
				extraInfoStr = " (" + strings.Join(extraInfo, ", ") + ")"
			}
			fmt.Println(framework.Name + " " + framework.Version + extraInfoStr)
			id := framework.Uid
			if framework.Mrn != "" {
				id = framework.Mrn
			}
			fmt.Println(termenv.String("  " + id).Foreground(theme.DefaultTheme.Colors.Disabled))
		}
		return nil
	},
}

var frameworkDownloadCmd = &cobra.Command{
	Use:   "download [mrn]",
	Short: "Download a compliance framework",
	Args:  cobra.ExactArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlag("file", cmd.Flags().Lookup("file")); err != nil {
			return err
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		outputFile := viper.GetString("file")
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

		mondooClient, err := getGqlClient(opts)
		if err != nil {
			return err
		}

		frameworkMrn := args[0]
		if !strings.HasPrefix(frameworkMrn, PolicyMrnPrefix) {
			frameworkMrn = FrameworkMrnPrefix + "/" + frameworkMrn
		}

		data, err := cnspec_upstream.DownloadFramework(context.Background(), mondooClient, frameworkMrn, opts.GetParentMrn())
		if err != nil {
			log.Error().Msgf("failed to download compliance framework: %s", err)
			os.Exit(1)
		}

		if err := os.WriteFile(outputFile, []byte(data), 0o644); err != nil {
			log.Error().Msgf("failed to store framework: %s", err)
			os.Exit(1)
		}
		log.Info().Msg(theme.DefaultTheme.Success("successfully downloaded to ", outputFile))

		return nil
	},
}

var frameworkUploadCmd = &cobra.Command{
	Use:   "upload [file]",
	Short: "Upload a compliance framework",
	Args:  cobra.ExactArgs(0),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlag("file", cmd.Flags().Lookup("file")); err != nil {
			return err
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		inputFile := viper.GetString("file")
		if inputFile == "" {
			log.Error().Msgf("input file is required")
			os.Exit(1)
		}

		opts, err := config.Read()
		if err != nil {
			log.Error().Msgf("failed to get config: %s", err)
			os.Exit(1)
		}
		config.DisplayUsedConfig()

		mondooClient, err := getGqlClient(opts)
		if err != nil {
			return err
		}

		data, err := os.ReadFile(inputFile)
		if err != nil {
			log.Error().Msgf("failed to read file: %s", err)
			os.Exit(1)
		}

		ok, err := cnspec_upstream.UploadFramework(context.Background(), mondooClient, data, opts.GetParentMrn())
		if err != nil {
			log.Error().Msgf("failed to upload compliance framework: %s", err)
			os.Exit(1)
		}
		if !ok {
			log.Error().Msgf("failed to upload compliance framework")
			os.Exit(1)
		}
		log.Info().Msg(theme.DefaultTheme.Success("successfully uploaded compliance framework"))
		return nil
	},
}

var frameworkPreviewCmd = &cobra.Command{
	Use:   "preview [mrn]",
	Short: "Change a framework status to preview",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		opts, err := config.Read()
		if err != nil {
			return err
		}
		config.DisplayUsedConfig()

		mondooClient, err := getGqlClient(opts)
		if err != nil {
			return err
		}

		frameworkMrn := args[0]
		if !strings.HasPrefix(frameworkMrn, PolicyMrnPrefix) {
			frameworkMrn = FrameworkMrnPrefix + "/" + frameworkMrn
		}
		ok, err := cnspec_upstream.MutateFrameworkState(
			context.Background(), mondooClient, frameworkMrn,
			opts.GetParentMrn(), mondoogql.ComplianceFrameworkMutationActionPreview,
		)
		if err != nil {
			log.Error().Msgf("failed to set compliance framework to preview state in space: %s", err)
			os.Exit(1)
		}
		if !ok {
			log.Error().Msgf("failed to set compliance framework to preview state in space")
			os.Exit(1)
		}
		log.Info().Msg(theme.DefaultTheme.Success("successfully set compliance framework to preview state in space"))

		return nil
	},
}

var frameworkActiveCmd = &cobra.Command{
	Use:     "active [mrn]",
	Aliases: []string{"enable", "activate"},
	Short:   "Change a framework status to active",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		opts, err := config.Read()
		if err != nil {
			return err
		}
		config.DisplayUsedConfig()

		mondooClient, err := getGqlClient(opts)
		if err != nil {
			return err
		}

		frameworkMrn := args[0]
		if !strings.HasPrefix(frameworkMrn, PolicyMrnPrefix) {
			frameworkMrn = FrameworkMrnPrefix + "/" + frameworkMrn
		}

		ok, err := cnspec_upstream.MutateFrameworkState(
			context.Background(), mondooClient, frameworkMrn,
			opts.GetParentMrn(), mondoogql.ComplianceFrameworkMutationActionEnable,
		)
		if err != nil {
			log.Error().Msgf("failed to set compliance framework to active state in space: %s", err)
			os.Exit(1)
		}
		if !ok {
			log.Error().Msgf("failed to set compliance framework to active state in space")
			os.Exit(1)
		}
		log.Info().Msg(theme.DefaultTheme.Success("successfully set compliance framework to active state in space"))

		return nil
	},
}

var frameworkDisabledCmd = &cobra.Command{
	Use:     "disabled [mrn]",
	Aliases: []string{"disable"},
	Short:   "Change a framework status to disabled",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		opts, err := config.Read()
		if err != nil {
			return err
		}
		config.DisplayUsedConfig()

		mondooClient, err := getGqlClient(opts)
		if err != nil {
			return err
		}

		frameworkMrn := args[0]
		if !strings.HasPrefix(frameworkMrn, PolicyMrnPrefix) {
			frameworkMrn = FrameworkMrnPrefix + "/" + frameworkMrn
		}

		ok, err := cnspec_upstream.MutateFrameworkState(
			context.Background(), mondooClient, frameworkMrn,
			opts.GetParentMrn(), mondoogql.ComplianceFrameworkMutationActionDisable,
		)
		if err != nil {
			log.Error().Msgf("failed to set compliance framework to disabled state in space: %s", err)
			os.Exit(1)
		}
		if !ok {
			log.Error().Msgf("failed to set compliance framework to disabled state in space")
			os.Exit(1)
		}
		log.Info().Msg(theme.DefaultTheme.Success("successfully set compliance framework to disabled state in space"))

		return nil
	},
}
