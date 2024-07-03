package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/muesli/termenv"
	"github.com/spf13/cobra"
	"go.mondoo.com/cnquery/v11/cli/config"
	"go.mondoo.com/cnquery/v11/cli/theme"
	cnspec_upstream "go.mondoo.com/cnspec/v11/upstream"
	mondoogql "go.mondoo.com/mondoo-go"
)

func init() {
	rootCmd.AddCommand(frameworkCmd)

	frameworkCmd.AddCommand(frameworkListCmd)
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
