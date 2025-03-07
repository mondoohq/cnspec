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
	"go.mondoo.com/cnquery/v11"
	"go.mondoo.com/cnquery/v11/cli/config"
	"go.mondoo.com/cnquery/v11/cli/execruntime"
	"go.mondoo.com/cnquery/v11/cli/inventoryloader"
	"go.mondoo.com/cnquery/v11/cli/theme"
	"go.mondoo.com/cnquery/v11/logger"
	"go.mondoo.com/cnquery/v11/mqlc"
	"go.mondoo.com/cnquery/v11/providers"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/upstream"
	"go.mondoo.com/cnspec/v11/cli/reporter"
	"go.mondoo.com/cnspec/v11/policy"
	"go.mondoo.com/cnspec/v11/policy/scan"
)

const (
	// allow sending reports to alternative URLs
	featureReportAlternateUrlEnv = "REPORT_URL"
)

func init() {
	rootCmd.AddCommand(scanCmd)

	_ = scanCmd.Flags().StringP("output", "o", "compact", "Set output format: "+reporter.AllFormats())
	_ = scanCmd.Flags().BoolP("json", "j", false, "Run the query and return the object in a JSON structure.")
	_ = scanCmd.Flags().String("platform-id", "", "Select a specific target asset by providing its platform ID.")

	_ = scanCmd.Flags().String("inventory-file", "", "Set the path to the inventory file.")
	_ = scanCmd.Flags().String("inventory-template", "", "Set the path to the inventory template.")
	_ = scanCmd.Flags().MarkHidden("inventory-template")

	_ = scanCmd.Flags().Bool("inventory-format-ansible", false, "Set the inventory format to Ansible.")
	// "inventory-ansible" is deprecated, use "inventory-format-ansible" instead
	_ = scanCmd.Flags().Bool("inventory-ansible", false, "Set the inventory format to Ansible.")
	_ = scanCmd.Flags().MarkDeprecated("inventory-ansible", "use --inventory-format-ansible")
	_ = scanCmd.Flags().MarkHidden("inventory-ansible")

	_ = scanCmd.Flags().Bool("inventory-format-domainlist", false, "Set the inventory format to domain list.")
	// "inventory-domainlist" is deprecated, use "inventory-format-domainlist" instead
	_ = scanCmd.Flags().Bool("inventory-domainlist", false, "Set the inventory format to domain list.")
	_ = scanCmd.Flags().MarkDeprecated("inventory-domainlist", "use --inventory-format-domainlist")
	_ = scanCmd.Flags().MarkHidden("inventory-domainlist")

	// bundles, packs & incognito mode
	_ = scanCmd.Flags().Bool("incognito", false, "Run in incognito mode. Do not report scan results to Mondoo Platform.")
	_ = scanCmd.Flags().StringSlice("policy", nil, "Lists policies to execute. This requires --policy-bundle. You can pass multiple policies using --policy POLICY.")
	_ = scanCmd.Flags().StringSliceP("policy-bundle", "f", nil, "Path to local policy file")
	// flag completion command
	_ = scanCmd.RegisterFlagCompletionFunc("policy", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return getPoliciesForCompletion(), cobra.ShellCompDirectiveDefault
	})
	_ = scanCmd.Flags().String("asset-name", "", "User-override for the asset name")
	_ = scanCmd.Flags().StringToString("annotation", nil, "Add an annotation to the asset in this format: key=value.") // user-added, editable
	_ = scanCmd.Flags().StringToString("props", nil, "Custom values for properties")
	_ = scanCmd.Flags().String("trace-id", "", "Trace identifier")

	// v6 should make detect-cicd and category flag public
	_ = scanCmd.Flags().Bool("detect-cicd", true, "Try to detect CI/CD environments. If detected, set the asset category to 'cicd'.")
	_ = scanCmd.Flags().String("category", "inventory", "Set the category for the assets to 'inventory|cicd'.")
	_ = scanCmd.Flags().MarkHidden("category")
	_ = scanCmd.Flags().Int("score-threshold", 0, "If any score falls below the threshold, exit 1.")
	_ = scanCmd.Flags().String("output-target", "", "Set output target to which the asset report will be sent. Currently only supports AWS SQS topic URLs and local files")
}

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan assets with one or more policies",
	Long: `
This command scans an asset using a policy. For example, you can scan
the local system with its pre-configured policies:

		$ cnspec scan local

To manually configure a policy, use this:

		$ cnspec scan local -f bundle.mql.yaml --incognito

`,
	PreRun: func(cmd *cobra.Command, _ []string) {
		// Special handling for users that want to see what output options are
		// available. We have to do this before printing the help because we
		// don't have a target connection or provider.
		output, _ := cmd.Flags().GetString("output")
		if output == "help" {
			fmt.Println(reporter.AllAvailableOptions())
			os.Exit(0)
		}

		_ = viper.BindPFlag("platform-id", cmd.Flags().Lookup("platform-id"))

		_ = viper.BindPFlag("inventory-file", cmd.Flags().Lookup("inventory-file"))
		_ = viper.BindPFlag("inventory-template", cmd.Flags().Lookup("inventory-template"))
		_ = viper.BindPFlag("inventory-format-ansible", cmd.Flags().Lookup("inventory-format-ansible"))
		// inventory-ansible is deprecated
		_ = viper.BindPFlag("inventory-ansible", cmd.Flags().Lookup("inventory-ansible"))
		_ = viper.BindPFlag("inventory-format-domainlist", cmd.Flags().Lookup("inventory-format-domainlist"))
		// inventory-domainlist is deprecated
		_ = viper.BindPFlag("inventory-domainlist", cmd.Flags().Lookup("inventory-domainlist"))

		_ = viper.BindPFlag("policy-bundle", cmd.Flags().Lookup("policy-bundle"))
		_ = viper.BindPFlag("detect-cicd", cmd.Flags().Lookup("detect-cicd"))
		_ = viper.BindPFlag("asset-name", cmd.Flags().Lookup("asset-name"))
		_ = viper.BindPFlag("trace-id", cmd.Flags().Lookup("trace-id"))
		_ = viper.BindPFlag("category", cmd.Flags().Lookup("category"))
		_ = viper.BindPFlag("score-threshold", cmd.Flags().Lookup("score-threshold"))

		// for all assets
		_ = viper.BindPFlag("incognito", cmd.Flags().Lookup("incognito"))
		_ = viper.BindPFlag("insecure", cmd.Flags().Lookup("insecure"))
		_ = viper.BindPFlag("policies", cmd.Flags().Lookup("policy"))
		_ = viper.BindPFlag("sudo.active", cmd.Flags().Lookup("sudo"))
		_ = viper.BindPFlag("record", cmd.Flags().Lookup("record"))
		_ = viper.BindPFlag("annotations", cmd.Flags().Lookup("annotation"))
		_ = viper.BindPFlag("props", cmd.Flags().Lookup("props"))

		_ = viper.BindPFlag("json", cmd.Flags().Lookup("json"))
		_ = viper.BindPFlag("output", cmd.Flags().Lookup("output"))
		_ = viper.BindPFlag("output-target", cmd.Flags().Lookup("output-target"))
	},
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return []string{"yml", "yaml", "json"}, cobra.ShellCompDirectiveFilterFileExt
		}
		return []string{}, cobra.ShellCompDirectiveNoFileComp
	},
	// we have to initialize an empty run so it shows up as a runnable command in --help
	Run: func(cmd *cobra.Command, args []string) {},
}

var scanCmdRun = func(cmd *cobra.Command, runtime *providers.Runtime, cliRes *plugin.ParseCLIRes) {
	ctx := context.Background()
	conf, err := getCobraScanConfig(cmd, runtime, cliRes)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to prepare config")
	}

	err = conf.loadPolicies(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to resolve policies")
	}

	report, err := RunScan(conf)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to run scan")
	}

	logger.DebugDumpJSON("report", report)

	handlerConf := reporter.HandlerConfig{
		Format:       conf.OutputFormat,
		OutputTarget: conf.OutputTarget,
		Incognito:    conf.IsIncognito,
	}
	outputHandler, err := reporter.NewOutputHandler(handlerConf)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create an output handler")
	}

	fCtx := cnquery.SetFeatures(ctx, conf.Features)
	if err := outputHandler.WriteReport(fCtx, report); err != nil {
		log.Fatal().Err(err).Msg("failed to write report to output target")
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
	Features     cnquery.Features
	Inventory    *inventory.Inventory
	ReportType   scan.ReportType
	OutputTarget string
	OutputFormat string
	PolicyPaths  []string
	PolicyNames  []string
	Props        map[string]string
	Bundle       *policy.Bundle
	runtime      *providers.Runtime

	IsIncognito    bool
	ScoreThreshold int

	DoRecord bool
	AgentMrn string
}

func getCobraScanConfig(cmd *cobra.Command, runtime *providers.Runtime, cliRes *plugin.ParseCLIRes) (*scanConfig, error) {
	opts, err := config.Read()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}

	config.DisplayUsedConfig()

	props := viper.GetStringMapString("props")

	// merge the config and the user-provided annotations with the latter having precedence
	optAnnotations := opts.Annotations
	if optAnnotations == nil {
		optAnnotations = map[string]string{}
	}

	assetName, err := cmd.Flags().GetString("asset-name")
	if err != nil {
		log.Fatal().Err(err).Msg("failed to parse asset-name")
	}
	if assetName != "" && cliRes.Asset != nil {
		cliRes.Asset.Name = assetName
	}

	traceId := viper.GetString("trace-id")
	if traceId != "" && cliRes.Asset != nil {
		cliRes.Asset.TraceId = traceId
	}

	inv, err := inventoryloader.ParseOrUse(cliRes.Asset, viper.GetBool("insecure"), optAnnotations)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to parse inventory")
	}

	conf := scanConfig{
		Features:       opts.GetFeatures(),
		IsIncognito:    viper.GetBool("incognito"),
		Inventory:      inv,
		PolicyPaths:    dedupe(viper.GetStringSlice("policy-bundle")),
		PolicyNames:    viper.GetStringSlice("policies"),
		ScoreThreshold: viper.GetInt("score-threshold"),
		Props:          props,
		runtime:        runtime,
		AgentMrn:       opts.AgentMrn,
		OutputTarget:   viper.GetString("output-target"),
	}

	// FIXME: DEPRECATED, remove in v12.0 and make this the default for all
	// use-cases where we have upstream recording enabled vv
	// Instead of depending on the feature-flag, we look at the config
	if conf.Features.IsActive(cnquery.StoreResourcesData) {
		if err = runtime.EnableResourcesRecording(); err != nil {
			log.Fatal().Err(err).Msg("failed to enable resources recording")
		}
	}
	// ^^

	// if users want to get more information on available output options,
	// print them before executing the scan
	output := viper.GetString("output")
	if output == "help" {
		fmt.Println(reporter.AllAvailableOptions())
		os.Exit(0)
	}

	// --json takes precedence
	if ok := viper.GetBool("json"); ok {
		output = "json"
	}
	conf.OutputFormat = output

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

	serviceAccount := opts.GetServiceCredential()

	// NOTE: even if we have incognito, we want to set the upstream config. Otherwise we would not be able to
	// use the policies that are defined in Mondoo Platform
	if serviceAccount != nil {
		log.Info().Msg("using service account credentials")
		conf.runtime.UpstreamConfig = &upstream.UpstreamConfig{
			SpaceMrn:    opts.GetParentMrn(),
			ApiEndpoint: opts.UpstreamApiEndpoint(),
			ApiProxy:    opts.APIProxy,
			Incognito:   conf.IsIncognito,
			Creds:       serviceAccount,
		}
		providers.DefaultRuntime().UpstreamConfig = conf.runtime.UpstreamConfig
	} else {
		log.Warn().Msg("No credentials provided. Switching to --incognito mode.")
		conf.IsIncognito = true
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

func (c *scanConfig) loadPolicies(ctx context.Context) error {
	if c.IsIncognito {
		if len(c.PolicyPaths) == 0 {
			return nil
		}

		bundleLoader := policy.DefaultBundleLoader()
		bundle, err := bundleLoader.BundleFromPaths(c.PolicyPaths...)
		if err != nil {
			return err
		}

		// prepare the bundle for compilation
		bundle.Prepare()
		conf := mqlc.NewConfig(c.runtime.Schema(), cnquery.DefaultFeatures)

		_, err = bundle.CompileExt(ctx, policy.BundleCompileConf{
			CompilerConfig: conf,
			// We don't care about failing queries for local runs. We may only
			// process a subset of all the queries in the bundle. When we receive
			// things from the server, upstream can filter things for us. But running
			// them locally requires us to do it in here.
			RemoveFailing: true,
		})
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
	opts = append(opts, scan.WithRecording(config.runtime.Recording()))

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

func dedupe[T string | int](sliceList []T) []T {
	allKeys := make(map[T]bool)
	list := []T{}
	for _, item := range sliceList {
		if _, value := allKeys[item]; !value {
			allKeys[item] = true
			list = append(list, item)
		}
	}
	return list
}
