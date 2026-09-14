// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.mondoo.com/cnspec/cli/reporter"
	"go.mondoo.com/cnspec/internal/sbom/pack"
	"go.mondoo.com/cnspec/internal/scandump"
	"go.mondoo.com/cnspec/upload"
	"go.mondoo.com/mql/cli/config"
	"go.mondoo.com/mql/providers"
	"go.mondoo.com/mql/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/sbom/generator"
)

func init() {
	rootCmd.AddCommand(vulnCmd)
	vulnCmd.Flags().StringP("output", "o", "full", "Set the output format: "+reporter.AllFormats())
	vulnCmd.Flags().String("platform-id", "", "Select a specific target asset by providing its platform ID")

	// we need this for config parsing but it should not be exposed to the user
	vulnCmd.Flags().String("asset-name", "", "Override the asset name")
	vulnCmd.Flags().Lookup("asset-name").Hidden = true

	vulnCmd.Flags().String("inventory-file", "", "Set the path to the inventory file")
	vulnCmd.Flags().Bool("inventory-ansible", false, "Set the inventory format to Ansible")
	vulnCmd.Flags().Bool("inventory-domainlist", false, "Set the inventory format to domain list")
}

var vulnCmd = &cobra.Command{
	Use:   "vuln",
	Short: "Scan a target for vulnerabilities",
	PreRun: func(cmd *cobra.Command, args []string) {
		// for all assets
		_ = viper.BindPFlag("output", cmd.Flags().Lookup("output"))
		_ = viper.BindPFlag("platform-id", cmd.Flags().Lookup("platform-id"))
		_ = viper.BindPFlag("inventory-file", cmd.Flags().Lookup("inventory-file"))
		_ = viper.BindPFlag("inventory-ansible", cmd.Flags().Lookup("inventory-ansible"))
		_ = viper.BindPFlag("inventory-domainlist", cmd.Flags().Lookup("inventory-domainlist"))
	},
}

var vulnCmdRun = func(cmd *cobra.Command, runtime *providers.Runtime, cliRes *plugin.ParseCLIRes) {
	dumpCtx := setupDebugDumps(context.Background(), "cnspec-vuln-debug")

	pb, err := pack.QueryPack()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load sbom query pack")
	}

	conf, err := getCobraScanConfig(cmd, runtime, cliRes)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to gather scan config")
	}

	conf.PolicyNames = nil
	conf.PolicyPaths = nil
	conf.Bundle = pb
	conf.IsIncognito = true

	printConf, err := reporter.ParseConfig(conf.OutputFormat)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to parse config for reporter")
	}

	report, err := RunScan(dumpCtx, conf)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to run scan")
	}

	cnspecReport, err := reporter.ConvertToProto(report)
	if err == nil {
		log.Debug().Msg("converted report to proto")
		data, _ := cnspecReport.ToJSON()
		scandump.JSON(dumpCtx, "sbom-report", data)
	}

	boms := generator.GenerateBom(cnspecReport.ToCnqueryReport())

	if len(boms) != 1 {
		log.Fatal().Msg("received data for more than one asset, this is not supported yet.")
	}
	bom := boms[0]

	ctx := cmd.Context()

	// Scan the locally-generated SBOM against Mondoo Platform. This is the
	// PURL-native path: the SBOM (which carries package PURLs) is uploaded to
	// ExtendedVulnMgmt.ScanUploadedSbom, which returns VEX (Vulnerability
	// Exchange) documents. The scan is ephemeral — nothing is persisted upstream.
	// Pass the config path the CLI resolved. upload.LoadCredentials falls back to
	// the default location for an empty path, so an empty Opts silently reads
	// ~/.config/mondoo/mondoo.yml and ignores --config -- which surfaces as an
	// auth failure blaming the user's key rather than as "wrong config".
	vex, err := upload.ScanSBOM(ctx, upload.Opts{ConfigPath: config.UserProvidedPath}, bom)
	if err != nil {
		// Without credentials we can still report the local inventory; degrade to
		// a clear warning rather than failing the command.
		if upload.IsNoCredentials(err) {
			log.Warn().Msg("no Mondoo credentials found; run `cnspec login` to enable vulnerability analysis")
			return
		}
		log.Fatal().Err(err).Msg("failed to scan SBOM for vulnerabilities")
	}

	scandump.JSON(dumpCtx, "vex", vex)

	// print the output using the specified output format
	r := reporter.NewReporter(printConf, false)
	if err := r.PrintVulns(vex, bom.Asset.Name); err != nil {
		log.Fatal().Err(err).Msg("failed to print")
	}
}
