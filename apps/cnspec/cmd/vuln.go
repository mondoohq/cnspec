// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.mondoo.com/cnquery/v10/logger"
	"go.mondoo.com/cnquery/v10/providers"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/upstream/mvd"
	"go.mondoo.com/cnquery/v10/sbom"
	"go.mondoo.com/cnspec/v10/cli/reporter"
	"go.mondoo.com/cnspec/v10/policy"
)

func init() {
	rootCmd.AddCommand(vulnCmd)
	vulnCmd.Flags().StringP("output", "o", "full", "Set output format: "+reporter.AllFormats())
	vulnCmd.Flags().BoolP("json", "j", false, "Run the query and return the object in a JSON structure.")
	vulnCmd.Flags().String("platform-id", "", "Select a specific target asset by providing its platform ID.")
	vulnCmd.Flags().String("asset-name", "", "User-override for the asset name")

	vulnCmd.Flags().String("inventory-file", "", "Set the path to the inventory file.")
	vulnCmd.Flags().Bool("inventory-ansible", false, "Set the inventory format to Ansible.")
	vulnCmd.Flags().Bool("inventory-domainlist", false, "Set the inventory format to domain list.")
	vulnCmd.Flags().StringToString("props", nil, "Custom values for properties")
}

var vulnCmd = &cobra.Command{
	Use:   "vuln",
	Short: "Scans a target for vulnerabilities",
	PreRun: func(cmd *cobra.Command, args []string) {
		// for all assets
		viper.BindPFlag("platform-id", cmd.Flags().Lookup("platform-id"))
		viper.BindPFlag("inventory-file", cmd.Flags().Lookup("inventory-file"))
		viper.BindPFlag("inventory-ansible", cmd.Flags().Lookup("inventory-ansible"))
		viper.BindPFlag("inventory-domainlist", cmd.Flags().Lookup("inventory-domainlist"))
	},
}

var vulnCmdRun = func(cmd *cobra.Command, runtime *providers.Runtime, cliRes *plugin.ParseCLIRes) {
	pb, err := sbom.QueryPack()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load sbom query pack")
	}

	conf, err := getCobraScanConfig(cmd, runtime, cliRes)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to gather scan config")
	}

	conf.PolicyNames = nil
	conf.PolicyPaths = nil
	conf.Bundle = policy.FromQueryPackBundle(pb)
	conf.IsIncognito = true

	report, err := RunScan(conf)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to run scan")
	}

	cnspecReport, err := reporter.ConvertToProto(report)
	if err == nil {
		log.Debug().Msg("converted report to proto")
		data, _ := cnspecReport.ToJSON()
		logger.DebugDumpJSON("mondoo-sbom-report", data)
	}

	boms, err := sbom.NewBom(cnspecReport.ToCnqueryReport())
	if err != nil {
		log.Fatal().Err(err).Msg("failed to parse sbom data")
	}

	if len(boms) != 1 {
		log.Fatal().Msg("received data for more than one asset, this is not supported yet.")
	}
	bom := boms[0]

	ctx := cmd.Context()
	upstreamConf := conf.runtime.UpstreamConfig
	if upstreamConf == nil {
		log.Fatal().Err(err).Msg("run `cnspec login` to authenticate with Mondoo platform")
	}
	client, err := upstreamConf.InitClient(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize authentication with Mondoo platform")
	}

	scannerClient, err := mvd.NewAdvisoryScannerClient(client.ApiEndpoint, client.HttpClient, client.Plugins...)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize advisory scanner client")
	}

	var runningKernel string
	if bom.Asset.Labels != nil {
		runningKernel = bom.Asset.Labels[sbom.LABEL_KERNEL_RUNNING]
	}

	req := &mvd.AnalyseAssetRequest{
		Platform: &mvd.Platform{
			Name:    bom.Asset.Platform.Name,
			Arch:    bom.Asset.Platform.Arch,
			Build:   bom.Asset.Platform.Build,
			Release: bom.Asset.Platform.Version,
			Labels:  bom.Asset.Platform.Labels,
			Title:   bom.Asset.Platform.Title,
		},
		Packages:      make([]*mvd.Package, 0),
		KernelVersion: runningKernel,
	}

	for i := range bom.Packages {
		pkg := bom.Packages[i]
		req.Packages = append(req.Packages, &mvd.Package{
			Name:    pkg.Name,
			Version: pkg.Version,
			Arch:    pkg.Architecture,
			Format:  pkg.Type,
			Origin:  pkg.Origin,
		})
	}

	vulnReport, err := scannerClient.AnalyseAsset(ctx, req)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to analyse asset")
	}

	// print the output using the specified output format
	r := reporter.NewReporter(reporter.Formats[strings.ToLower(conf.OutputFormat)], false)
	logger.DebugDumpJSON("vulnReport", report)
	if err := r.PrintVulns(vulnReport, bom.Asset.Name); err != nil {
		log.Fatal().Err(err).Msg("failed to print")
	}
}
