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

	frameworkCmd.AddCommand(frameworkListCmd)
	frameworkCmd.AddCommand(frameworkPreviewCmd)
	frameworkCmd.AddCommand(frameworkActiveCmd)
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
