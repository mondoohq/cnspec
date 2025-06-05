// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/mattn/go-isatty"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.mondoo.com/cnquery/v11/cli/components"
	"go.mondoo.com/cnquery/v11/cli/config"
	"go.mondoo.com/cnquery/v11/cli/theme"
	"go.mondoo.com/cnspec/v11/internal/onboarding"
	cnspec_upstream "go.mondoo.com/cnspec/v11/upstream"
)

const spacePrefix = "//captain.api.mondoo.app/spaces/"

func init() {
	// cnspec integrate
	rootCmd.AddCommand(integrateCmd)

	// global flags for the integrate command
	integrateCmd.PersistentFlags().String("space", "", "Set the space to create the integration")
	integrateCmd.PersistentFlags().String("output", "", "Location to write automation code")
	integrateCmd.PersistentFlags().String("integration-name", "", "The name of the integration")

	// cnspec integrate azure
	integrateCmd.AddCommand(integrateAzureCmd)
	integrateAzureCmd.Flags().String("subscription-id", "", "Azure subscription used to create resources")
	integrateAzureCmd.Flags().Bool("scan-vms", false, "Enable scanning Azure VMs using RunCommand")
	integrateAzureCmd.Flags().StringSlice("allow", []string{}, "Choose the Azure subscriptions to scan")
	integrateAzureCmd.Flags().StringSlice("deny", []string{}, "List of Azure subscriptions to avoid scanning")
	// providing both, --deny and --allow, is not allowed
	integrateAzureCmd.MarkFlagsMutuallyExclusive("allow", "deny")

	// cnspec integrate ms365
	integrateCmd.AddCommand(integrateMs365Cmd)

	integrateMs365Cmd.Flags().String("subscription-id", "", "Azure tenant ID for MS365 integration")
}

var (
	integrateCmd = &cobra.Command{
		Use:     "integrate",
		Aliases: []string{"onboard"},
		Hidden:  true,
		Short:   "Onboard integrations for continuous scanning into the Mondoo platform",
		Long:    "Run automation code to onboard your account and deploy Mondoo into various environments.",
	}
	integrateAzureCmd = &cobra.Command{
		Use:     "azure",
		Aliases: []string{"az"},
		Short:   "Onboard Microsoft Azure",
		Long: `Use this command to connect your Azure environment into the Mondoo platform.

By default, all subscriptions will be discovered and integrated for continuous scanning.

To choose the subscriptions to scan, pass the list of subscriptions using the --allow flag.

	cnspec integrate azure --allow <subscription_id_1> --allow <subscription_id_2>

To scan all subscriptions expect those you specify, pass the list of subscriptions you don't
want Mondoo to scan using the --deny flag.

	cnspec integrate azure --deny "<subscription_id_1>,<subscription_id_2>"

NOTE that --allow and --deny are mutually exclusive and can't be use together.`,
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			errs := []error{
				viper.BindPFlag("space", cmd.Flags().Lookup("space")),
				viper.BindPFlag("output", cmd.Flags().Lookup("output")),
				viper.BindPFlag("integration-name", cmd.Flags().Lookup("integration-name")),
				viper.BindPFlag("subscription-id", cmd.Flags().Lookup("subscription-id")),
				viper.BindPFlag("allow", cmd.Flags().Lookup("allow")),
				viper.BindPFlag("deny", cmd.Flags().Lookup("deny")),
				viper.BindPFlag("scan-vms", cmd.Flags().Lookup("scan-vms")),
			}
			return errors.Join(errs...)
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			var (
				space           = viper.GetString("space")
				subscriptionID  = viper.GetString("subscription-id")
				output          = viper.GetString("output")
				integrationName = viper.GetString("integration-name")
				allow           = viper.GetStringSlice("allow")
				deny            = viper.GetStringSlice("deny")
				scanVMs         = viper.GetBool("scan-vms")
			)

			// Verify if space exists, which verifies we have access to the Mondoo platform
			opts, err := config.Read()
			if err != nil {
				return err
			}
			// TODO verify that the config is a service account
			config.DisplayUsedConfig()
			mondooClient, err := getGqlClient(opts)
			if err != nil {
				return err
			}
			// by default, use the MRN from the config
			spaceMrn := opts.GetParentMrn()
			if space != "" {
				// unless it was specified via flag
				spaceMrn = spacePrefix + space
			}
			spaceInfo, err := cnspec_upstream.GetSpace(context.Background(), mondooClient, spaceMrn)
			if err != nil {
				log.Fatal().Msgf("unable to verify access to space '%s': %s", space, err)
			}
			log.Info().Msg("using space " + theme.DefaultTheme.Success(spaceInfo.Mrn))

			if space == "" {
				space = strings.Split(spaceInfo.Mrn, "/")[4] // Extract space ID from MRN
			}

			// Discover the subscription used to create resources in the cloud, if it wasn't specified. Note
			// that this will also verify that we have access to Azure. If we fail, we shouldn't try to continue.
			if subscriptionID == "" {
				azAccountJSON, err := exec.Command("az", "account", "list", "-o", "json").Output()
				if err != nil {
					return errors.Wrap(err, "unable to detect Azure subscriptions")
				}
				var azAccounts []onboarding.AzAccount
				if err := json.Unmarshal(azAccountJSON, &azAccounts); err != nil {
					return err
				}

				isTTY := isatty.IsTerminal(os.Stdout.Fd())
				if isTTY {
					selected := components.Select(
						"Select the primary subscription where resources will be created",
						azAccounts,
					)
					if selected >= 0 {
						subscriptionID = azAccounts[selected].ID
					}
				} else {
					fmt.Println(components.List(theme.OperatingSystemTheme, azAccounts))
					log.Fatal().
						Msg("cannot continue, missing subscription id, use --subscription-id to select a subscription")
				}
			}

			if subscriptionID == "" {
				log.Error().Msg("no subscription selected")
				os.Exit(1)
			}

			// Verify that the user has the right role assignments to onboard an Azure environment
			log.Info().Msg("verifying role assignments for the currently logged-in user")
			if err := onboarding.VerifyUserRoleAssignments(); err != nil {
				return errors.Wrap(err, "preflight verification failed")
			}

			// Generate HCL for azure deployment
			log.Info().Msg("generating automation code")
			hcl, err := onboarding.GenerateAzureHCL(onboarding.AzureIntegration{
				Name:    integrationName,
				Space:   space,
				Primary: subscriptionID,
				Allow:   allow,
				Deny:    deny,
				ScanVMs: scanVMs,
			})
			if err != nil {
				return errors.Wrap(err, "unable to generate automation code")
			}

			// Write generated code to disk
			dirname, err := onboarding.WriteHCL(hcl, output, "azure")
			if err != nil {
				return err
			}
			log.Info().Msgf("code stored at %s", theme.DefaultTheme.Secondary(dirname))

			// Run Terraform
			applied, err := onboarding.TerraformPlanAndExecute(dirname)
			if err != nil {
				return err
			}

			if applied {
				log.Info().Msg(theme.DefaultTheme.Success("Mondoo integration was successful!"))
				log.Info().Msgf(
					"To view integration status, visit https://console.mondoo.com/space/integrations/azure?spaceId=%s",
					space,
				)
			}
			return nil
		},
	}
	integrateMs365Cmd = &cobra.Command{
		Use:   "ms365",
		Short: "Onboard Microsoft 365",
		Long: `Use this command to connect your Microsoft 365 environment into the Mondoo platform.

Flags are optional:

	cnspec integrate ms365 --space <space_id> --subscription-id <tenant_id> --output <output_dir> --integration-name <name>

Ensure that the Azure account used for execution has the Azure AD Role "Global Reader".`,
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			errs := []error{
				viper.BindPFlag("space", cmd.Flags().Lookup("space")),
				viper.BindPFlag("output", cmd.Flags().Lookup("output")),
				viper.BindPFlag("integration-name", cmd.Flags().Lookup("integration-name")),
				viper.BindPFlag("subscription-id", cmd.Flags().Lookup("subscription-id")),
			}
			return errors.Join(errs...)
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			var (
				space           = viper.GetString("space")
				subscriptionID  = viper.GetString("subscription-id")
				output          = viper.GetString("output")
				integrationName = viper.GetString("integration-name")
			)

			// Verify if space exists, which verifies we have access to the Mondoo platform
			opts, err := config.Read()
			if err != nil {
				return err
			}
			// TODO verify that the config is a service account
			config.DisplayUsedConfig()
			mondooClient, err := getGqlClient(opts)
			if err != nil {
				return err
			}
			// by default, use the MRN from the config
			spaceMrn := opts.GetParentMrn()
			if space != "" {
				// unless it was specified via flag
				spaceMrn = spacePrefix + space
			}
			spaceInfo, err := cnspec_upstream.GetSpace(context.Background(), mondooClient, spaceMrn)
			if err != nil {
				log.Fatal().Msgf("unable to verify access to space '%s': %s", space, err)
			}
			log.Info().Msg("using space " + theme.DefaultTheme.Success(spaceInfo.Mrn))

			if space == "" {
				space = strings.Split(spaceInfo.Mrn, "/")[4] // Extract space ID from MRN
			}

			// Discover the subscription used to create resources in the cloud, if it wasn't specified. Note
			// that this will also verify that we have access to Azure. If we fail, we shouldn't try to continue.
			if subscriptionID == "" {
				azAccountJSON, err := exec.Command("az", "account", "list", "-o", "json").Output()
				if err != nil {
					return errors.Wrap(err, "unable to detect Azure subscriptions")
				}
				var azAccounts []onboarding.AzAccount
				if err := json.Unmarshal(azAccountJSON, &azAccounts); err != nil {
					return err
				}

				isTTY := isatty.IsTerminal(os.Stdout.Fd())
				if isTTY {
					selected := components.Select(
						"Select the primary subscription where resources will be created",
						azAccounts,
					)
					if selected >= 0 {
						subscriptionID = azAccounts[selected].ID
					}
				} else {
					fmt.Println(components.List(theme.OperatingSystemTheme, azAccounts))
					log.Fatal().
						Msg("cannot continue, missing subscription id, use --subscription-id to select a subscription")
				}
			}

			if subscriptionID == "" {
				log.Error().Msg("no subscription selected")
				os.Exit(1)
			}

			// Verify that the user has the right role assignments to onboard an Azure environment
			log.Info().Msg("verifying role assignments for the currently logged-in user")
			if err := onboarding.VerifyUserRoleAssignments(); err != nil {
				return errors.Wrap(err, "preflight verification failed")
			}

			// Generate HCL for MS365 deployment
			log.Info().Msg("generating automation code")
			hcl, err := onboarding.GenerateMs365HCL(onboarding.Ms365Integration{
				Name:    integrationName,
				Space:   space,
				Primary: subscriptionID,
			})
			if err != nil {
				return errors.Wrap(err, "unable to generate automation code")
			}

			// Write generated code to disk
			dirname, err := onboarding.WriteHCL(hcl, output, "ms365")
			if err != nil {
				return err
			}
			log.Info().Msgf("code stored at %s", theme.DefaultTheme.Secondary(dirname))

			// Run Terraform
			applied, err := onboarding.TerraformPlanAndExecute(dirname)
			if err != nil {
				return err
			}

			if applied {
				log.Info().Msg(theme.DefaultTheme.Success("Mondoo integration was successful!"))
				log.Info().Msgf(
					"To view integration status, visit https://console.mondoo.com/space/integrations/ms365?spaceId=%s",
					space,
				)
			}
			return nil
		},
	}
)
