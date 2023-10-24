// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"context"
	"fmt"
	"os"
	"sort"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.mondoo.com/cnquery/v9"
	"go.mondoo.com/cnquery/v9/cli/config"
	"go.mondoo.com/cnquery/v9/cli/execruntime"
	"go.mondoo.com/cnquery/v9/cli/inventoryloader"
	"go.mondoo.com/cnquery/v9/cli/theme"
	"go.mondoo.com/cnquery/v9/logger"
	"go.mondoo.com/cnquery/v9/providers"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/upstream"
	"go.mondoo.com/cnspec/v9/cli/reporter"
	"go.mondoo.com/cnspec/v9/policy"
	"go.mondoo.com/cnspec/v9/policy/scan"
	policy_upstream "go.mondoo.com/cnspec/v9/policy/upstream"
)

const (
	// allow sending reports to alternative URLs
	featureReportAlternateUrlEnv = "REPORT_URL"
)

func init() {
	rootCmd.AddCommand(scanCmd)

	scanCmd.Flags().StringP("output", "o", "compact", "Set output format: "+reporter.AllFormats())
	scanCmd.Flags().BoolP("json", "j", false, "Run the query and return the object in a JSON structure.")
	scanCmd.Flags().String("platform-id", "", "Select a specific target asset by providing its platform ID.")

	scanCmd.Flags().String("inventory-file", "", "Set the path to the inventory file.")
	scanCmd.Flags().Bool("inventory-ansible", false, "Set the inventory format to Ansible.")
	scanCmd.Flags().Bool("inventory-domainlist", false, "Set the inventory format to domain list.")

	// bundles, packs & incognito mode
	scanCmd.Flags().Bool("incognito", false, "Run in incognito mode. Do not report scan results to  Mondoo Platform.")
	scanCmd.Flags().StringSlice("policy", nil, "Lists policies to execute. This requires --policy-bundle. You can pass multiple policies using --policy POLICY.")
	scanCmd.Flags().StringSliceP("policy-bundle", "f", nil, "Path to local policy file")
	// flag completion command
	scanCmd.RegisterFlagCompletionFunc("policy", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return getPoliciesForCompletion(), cobra.ShellCompDirectiveDefault
	})
	scanCmd.Flags().String("asset-name", "", "User-override for the asset name")
	scanCmd.Flags().StringToString("annotation", nil, "Add an annotation to the asset.") // user-added, editable
	scanCmd.Flags().StringToString("props", nil, "Custom values for properties")

	// v6 should make detect-cicd and category flag public
	scanCmd.Flags().Bool("detect-cicd", true, "Try to detect CI/CD environments. If detected, set the asset category to 'cicd'.")
	scanCmd.Flags().String("category", "inventory", "Set the category for the assets to 'inventory|cicd'.")
	scanCmd.Flags().MarkHidden("category")
	scanCmd.Flags().Int("score-threshold", 0, "If any score falls below the threshold, exit 1.")
	scanCmd.Flags().Bool("share", false, "create a web-based private reports when cnspec is unauthenticated. Defaults to false.")
}

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan assets with one or more policies.",
	Long: `
This command scans an asset using a policy. For example, you can scan
the local system with its pre-configured policies:

		$ cnspec scan local

To manually configure a policy, use this:

		$ cnspec scan local -f bundle.mql.yaml --incognito

`,
	PreRun: func(cmd *cobra.Command, args []string) {
		// Special handling for users that want to see what output options are
		// available. We have to do this before printing the help because we
		// don't have a target connection or provider.
		output, _ := cmd.Flags().GetString("output")
		if output == "help" {
			fmt.Println("Available output formats: " + reporter.AllFormats())
			os.Exit(0)
		}

		viper.BindPFlag("platform-id", cmd.Flags().Lookup("platform-id"))

		viper.BindPFlag("inventory-file", cmd.Flags().Lookup("inventory-file"))
		viper.BindPFlag("inventory-ansible", cmd.Flags().Lookup("inventory-ansible"))
		viper.BindPFlag("inventory-domainlist", cmd.Flags().Lookup("inventory-domainlist"))
		viper.BindPFlag("policy-bundle", cmd.Flags().Lookup("policy-bundle"))
		viper.BindPFlag("detect-cicd", cmd.Flags().Lookup("detect-cicd"))
		viper.BindPFlag("category", cmd.Flags().Lookup("category"))
		viper.BindPFlag("score-threshold", cmd.Flags().Lookup("score-threshold"))
		viper.BindPFlag("share", cmd.Flags().Lookup("share"))

		// for all assets
		viper.BindPFlag("incognito", cmd.Flags().Lookup("incognito"))
		viper.BindPFlag("insecure", cmd.Flags().Lookup("insecure"))
		viper.BindPFlag("policies", cmd.Flags().Lookup("policy"))
		viper.BindPFlag("sudo.active", cmd.Flags().Lookup("sudo"))
		viper.BindPFlag("record", cmd.Flags().Lookup("record"))

		viper.BindPFlag("output", cmd.Flags().Lookup("output"))
	},
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return []string{"yml", "yaml", "json"}, cobra.ShellCompDirectiveFilterFileExt
		}
		return []string{}, cobra.ShellCompDirectiveNoFileComp
	},
}

var scanCmdRun = func(cmd *cobra.Command, runtime *providers.Runtime, cliRes *plugin.ParseCLIRes) {
	conf, err := getCobraScanConfig(cmd, runtime, cliRes)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to prepare config")
	}

	err = conf.loadPolicies()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to resolve policies")
	}

	report, err := RunScan(conf)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to run scan")
	}

	logger.DebugDumpJSON("report", report)
	printReports(report, conf, cmd)

	var shareReport bool
	if viper.IsSet("share") {
		shareReportFlag := viper.GetBool("share")
		shareReport = shareReportFlag
	}

	// if report sharing was requested, share the report and print the URL
	if conf.IsIncognito && shareReport {
		proxy, err := config.GetAPIProxy()
		if err != nil {
			log.Error().Err(err).Msg("error getting proxy information")
		} else {
			reportId, err := policy_upstream.UploadSharedReport(report, os.Getenv(featureReportAlternateUrlEnv), proxy)
			if err != nil {
				log.Fatal().Err(err).Msg("could not upload report results")
			}
			fmt.Printf("View report at %s\n", reportId.Url)
		}
	}

	// if we had asset errors, we return a non-zero exit code
	// asset errors are only connection issues
	if report != nil {
		if len(report.Errors) > 0 {
			os.Exit(1)
		}

		if report.GetWorstScore() < uint32(conf.ScoreThreshold) {
			os.Exit(1)
		}
	}
}

// helper method to retrieve the list of policies for autocomplete
func getPoliciesForCompletion() []string {
	policyList := []string{}

	// TODO: autocompletion
	sort.Strings(policyList)

	return policyList
}

type scanConfig struct {
	Features    cnquery.Features
	Inventory   *inventory.Inventory
	ReportType  scan.ReportType
	Output      string
	PolicyPaths []string
	PolicyNames []string
	Props       map[string]string
	Bundle      *policy.Bundle
	runtime     *providers.Runtime

	IsIncognito    bool
	ScoreThreshold int

	DoRecord bool
}

func getCobraScanConfig(cmd *cobra.Command, runtime *providers.Runtime, cliRes *plugin.ParseCLIRes) (*scanConfig, error) {
	opts, err := config.Read()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}

	config.DisplayUsedConfig()

	props, err := cmd.Flags().GetStringToString("props")
	if err != nil {
		log.Fatal().Err(err).Msg("failed to parse props")
	}

	// annotations are user-added, editable labels for assets and are optional, therefore we do not need to check for err
	annotations, _ := cmd.Flags().GetStringToString("annotation")
	// merge the config and the user-provided annotations with the latter having precedence
	optAnnotations := opts.Annotations
	if optAnnotations == nil {
		optAnnotations = map[string]string{}
	}
	for k, v := range annotations {
		optAnnotations[k] = v
	}

	inv, err := inventoryloader.ParseOrUse(cliRes.Asset, viper.GetBool("insecure"), optAnnotations)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to parse inventory")
	}

	conf := scanConfig{
		Features:       opts.GetFeatures(),
		IsIncognito:    viper.GetBool("incognito"),
		Inventory:      inv,
		PolicyPaths:    viper.GetStringSlice("policy-bundle"),
		PolicyNames:    viper.GetStringSlice("policies"),
		ScoreThreshold: viper.GetInt("score-threshold"),
		Props:          props,
		runtime:        runtime,
	}

	// if users want to get more information on available output options,
	// print them before executing the scan
	output, _ := cmd.Flags().GetString("output")
	if output == "help" {
		fmt.Println("Available output formats: " + reporter.AllFormats())
		os.Exit(0)
	}

	// --json takes precedence
	if ok, _ := cmd.Flags().GetBool("json"); ok {
		output = "json"
	}
	conf.Output = output

	// detect CI/CD runs and read labels from runtime and apply them to all assets in the inventory
	runtimeEnv := execruntime.Detect()
	if opts.AutoDetectCICDCategory && runtimeEnv.IsAutomatedEnv() || opts.Category == "cicd" {
		log.Info().Msg("detected ci-cd environment")
		// NOTE: we only apply those runtime environment labels for CI/CD runs to ensure other assets from the
		// inventory are not touched, we may consider to add the data to the flagAsset
		if runtimeEnv != nil {
			runtimeLabels := runtimeEnv.Labels()
			conf.Inventory.ApplyLabels(runtimeLabels)
		}
		conf.Inventory.ApplyCategory(inventory.AssetCategory_CATEGORY_CICD)
	}

	var serviceAccount *upstream.ServiceAccountCredentials
	if !conf.IsIncognito {
		serviceAccount = opts.GetServiceCredential()
		if serviceAccount != nil {
			// TODO: determine if this needs migrating
			// // determine information about the client
			// sysInfo, err := sysinfo.GatherSystemInfo()
			// if err != nil {
			// 	log.Warn().Err(err).Msg("could not gather client information")
			// }
			// plugins = append(plugins, defaultRangerPlugins(sysInfo, opts.GetFeatures())...)

			log.Info().Msg("using service account credentials")
			conf.runtime.UpstreamConfig = &upstream.UpstreamConfig{
				SpaceMrn:    opts.GetParentMrn(),
				ApiEndpoint: opts.UpstreamApiEndpoint(),
				ApiProxy:    opts.APIProxy,
				Incognito:   conf.IsIncognito,
				Creds:       serviceAccount,
			}
		} else {
			log.Warn().Msg("No credentials provided. Switching to --incognito mode.")
			conf.IsIncognito = true
		}
	}

	if len(conf.PolicyPaths) > 0 && !conf.IsIncognito {
		log.Warn().Msg("Scanning with local bundles will switch into --incognito mode by default. Your results will not be sent upstream.")
		conf.IsIncognito = true
	}

	// print headline when its not printed to yaml
	if output == "" {
		fmt.Fprintln(os.Stdout, theme.DefaultTheme.Welcome)
	}

	return &conf, nil
}

func (c *scanConfig) loadPolicies() error {
	if c.IsIncognito {
		if len(c.PolicyPaths) == 0 {
			return nil
		}

		bundle, err := policy.BundleFromPaths(c.PolicyPaths...)
		if err != nil {
			return err
		}

		_, err = bundle.Compile(context.Background(), c.runtime.Schema(), nil)
		if err != nil {
			return errors.Wrap(err, "failed to compile bundle")
		}

		c.Bundle = bundle
		return nil
	}

	return nil
}

func RunScan(config *scanConfig, scannerOpts ...scan.ScannerOption) (*policy.ReportCollection, error) {
	opts := scannerOpts
	if config.runtime.UpstreamConfig != nil {
		opts = append(opts, scan.WithUpstream(config.runtime.UpstreamConfig))
	}
	if config.runtime.Recording != nil {
		opts = append(opts, scan.WithRecording(config.runtime.Recording))
	}

	scanner := scan.NewLocalScanner(opts...)
	ctx := cnquery.SetFeatures(context.Background(), config.Features)

	var res *scan.ScanResult
	var err error
	if config.IsIncognito {
		res, err = scanner.RunIncognito(
			ctx,
			&scan.Job{
				Inventory:     config.Inventory,
				Bundle:        config.Bundle,
				PolicyFilters: config.PolicyNames,
				Props:         config.Props,
			})
	} else {
		res, err = scanner.Run(
			ctx,
			&scan.Job{
				Inventory:     config.Inventory,
				Bundle:        config.Bundle,
				PolicyFilters: config.PolicyNames,
				Props:         config.Props,
			})
	}

	if err != nil {
		return nil, err
	}
	return res.GetFull(), nil
}

func printReports(report *policy.ReportCollection, conf *scanConfig, cmd *cobra.Command) {
	// print the output using the specified output format
	r, err := reporter.New(conf.Output)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	r.IsIncognito = conf.IsIncognito
	if err = r.Print(report, os.Stdout); err != nil {
		log.Fatal().Err(err).Msg("failed to print")
	}
}
