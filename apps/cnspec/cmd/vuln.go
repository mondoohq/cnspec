// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.mondoo.com/cnspec/v13/cli/reporter"
	"go.mondoo.com/cnspec/v13/internal/sbom/generator"
	"go.mondoo.com/cnspec/v13/internal/sbom/pack"
	"go.mondoo.com/cnspec/v13/internal/scandump"
	"go.mondoo.com/cnspec/v13/upload"
	cr "go.mondoo.com/mql/v13/cli/reporter"
	"go.mondoo.com/mql/v13/providers"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/upstream/fex"
	"go.mondoo.com/mql/v13/providers-sdk/v1/upstream/mvd"
	mqlsbomgen "go.mondoo.com/mql/v13/sbom/generator"
)

// vulnUploadSource identifies cnspec-produced vulnerability findings uploaded to
// Mondoo Platform.
const vulnUploadSource = "cnspec"

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

	// Experimental: upload discovered vulnerabilities to Mondoo Platform as VEX.
	vulnCmd.Flags().Bool("upload", false, "Experimental: upload discovered vulnerabilities to Mondoo Platform as findings")
	_ = vulnCmd.Flags().MarkHidden("upload")
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
	upstreamConf := conf.runtime.UpstreamConfig
	if upstreamConf == nil {
		log.Fatal().Err(err).Msg("run `cnspec login` to authenticate with Mondoo Platform")
	}
	client, err := upstreamConf.InitClient(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize authentication with Mondoo Platform")
	}

	scannerClient, err := mvd.NewAdvisoryScannerClient(client.ApiEndpoint, client.HttpClient, client.Plugins...)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize advisory scanner client")
	}

	var runningKernel string
	if bom.Asset.Labels != nil {
		runningKernel = bom.Asset.Labels[generator.LABEL_KERNEL_RUNNING]
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
	r := reporter.NewReporter(printConf, false)
	scandump.JSON(dumpCtx, "vulnReport", report)
	if err := r.PrintVulns(vulnReport, bom.Asset.Name); err != nil {
		log.Fatal().Err(err).Msg("failed to print")
	}

	if upload, _ := cmd.Flags().GetBool("upload"); upload {
		uploadVulnFindings(ctx, cnspecReport.ToCnqueryReport())
	}
}

// uploadVulnFindings emits the scanned asset's vulnerabilities to Mondoo Platform
// as VEX findings. It builds an SBOM from the scan report, scans it on the
// platform (ExtendedVulnMgmt.ScanUploadedSbom returns VEX), and uploads the VEX —
// the same SBOM-scan flow the `cnspec upload --format sbom` path and xgrep use.
func uploadVulnFindings(ctx context.Context, cnqueryReport *cr.Report) {
	boms := mqlsbomgen.GenerateBom(cnqueryReport)
	if len(boms) != 1 {
		log.Error().Msg("skipping upload: expected exactly one asset SBOM")
		return
	}

	opts := upload.Opts{}
	vex, err := upload.ScanSBOM(ctx, opts, boms[0])
	if err != nil {
		if upload.IsNoCredentials(err) {
			log.Error().Msg("skipping upload: run `cnspec login` to authenticate with Mondoo Platform")
			return
		}
		log.Error().Err(err).Msg("failed to scan SBOM for upload")
		return
	}

	docs := fex.VexToDocuments(vex)
	if len(docs) == 0 {
		log.Info().Msg("no vulnerabilities to upload")
		return
	}
	if err := upload.UploadFindings(ctx, opts, docs, vulnUploadSource); err != nil {
		log.Error().Err(err).Msg("failed to upload vulnerability findings")
		return
	}
	log.Info().Msgf("uploaded %d vulnerability finding(s) to Mondoo Platform", len(docs))
}
