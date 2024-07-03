package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/muesli/termenv"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"go.mondoo.com/cnquery/v11/cli/config"
	"go.mondoo.com/cnquery/v11/cli/theme"
	cnspec_upstream "go.mondoo.com/cnspec/v11/upstream"
	mondoogql "go.mondoo.com/mondoo-go"
)

const (
	FrameworkMrnPrefix = "//policy.api.mondoo.app/frameworks"
)

func init() {
	rootCmd.AddCommand(frameworkCmd)

	// list
	frameworkCmd.AddCommand(frameworkListCmd)
	// preview
	frameworkCmd.AddCommand(frameworkPreviewCmd)
	// active
	frameworkCmd.AddCommand(frameworkActiveCmd)
	// download
	frameworkDownloadCmd.Flags().StringP("file", "f", "", "output file")
	frameworkCmd.AddCommand(frameworkDownloadCmd)
}

var frameworkCmd = &cobra.Command{
	Use:   "framework",
	Short: "Manage local and upstream compliance frameworks",
}

var frameworkListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available compliance frameworks",
	Args:  cobra.MaximumNArgs(0),
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

		frameworks, err := cnspec_upstream.ListFrameworks(context.Background(), mondooClient, opts.GetParentMrn())
		if err != nil {
			return err
		}

		for _, framework := range frameworks {
			extraInfo := []string{}
			if framework.State == mondoogql.ComplianceFrameworkStateActive {
				extraInfo = append(extraInfo, theme.DefaultTheme.Success("active"))
			}

			extraInfoStr := ""
			if len(extraInfo) > 0 {
				extraInfoStr = " (" + strings.Join(extraInfo, ", ") + ")"
			}
			fmt.Println(framework.Name + " " + framework.Version + extraInfoStr)

			fmt.Println(termenv.String("  " + framework.Mrn).Foreground(theme.DefaultTheme.Colors.Disabled))
		}
		return nil
	},
}

var frameworkDownloadCmd = &cobra.Command{
	Use:   "download [mrn]",
	Short: "Download a compliance framework",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		outputFile, _ := cmd.Flags().GetString("file")
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
		err = cnspec_upstream.MutateFrameworkState(
			context.Background(), mondooClient, frameworkMrn,
			opts.GetParentMrn(), mondoogql.ComplianceFrameworkMutationActionPreview,
		)
		if err != nil {
			log.Error().Msgf("failed to set compliance framework to preview state in space: %s", err)
			os.Exit(1)
		}
		log.Info().Msg(theme.DefaultTheme.Success("successfully set compliance framework to preview state in space"))

		return nil
	},
}

var frameworkActiveCmd = &cobra.Command{
	Use:   "active [mrn]",
	Short: "Change a framework status to active",
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

		err = cnspec_upstream.MutateFrameworkState(
			context.Background(), mondooClient, frameworkMrn,
			opts.GetParentMrn(), mondoogql.ComplianceFrameworkMutationActionPreview,
		)
		if err != nil {
			log.Error().Msgf("failed to set compliance framework to active state in space: %s", err)
			os.Exit(1)
		}
		log.Info().Msg(theme.DefaultTheme.Success("successfully set compliance framework to active state in space"))

		return nil
	},
}
