// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"os"

	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.mondoo.com/cnquery/v9/cli/shell"
	"go.mondoo.com/cnquery/v9/explorer/executor"
	"go.mondoo.com/cnquery/v9/logger"
	"go.mondoo.com/cnquery/v9/providers"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/upstream/mvd"
	"go.mondoo.com/cnspec/v9/cli/reporter"
	"go.mondoo.com/ranger-rpc/codes"
	"go.mondoo.com/ranger-rpc/status"
)

func init() {
	rootCmd.AddCommand(vulnCmd)
	vulnCmd.Flags().StringP("output", "o", "compact", "Set output format: "+reporter.AllFormats())
	vulnCmd.Flags().BoolP("json", "j", false, "Run the query and return the object in a JSON structure.")
	vulnCmd.Flags().String("platform-id", "", "Select a specific target asset by providing its platform ID.")

	vulnCmd.Flags().String("inventory-file", "", "Set the path to the inventory file.")
	vulnCmd.Flags().Bool("inventory-ansible", false, "Set the inventory format to Ansible.")
	vulnCmd.Flags().Bool("inventory-domainlist", false, "Set the inventory format to domain list.")
	vulnCmd.Flags().StringToString("props", nil, "Custom values for properties")
}

var vulnCmd = &cobra.Command{
	Use:   "vuln",
	Short: "Scans a target for Vulnerabilities.",
	PreRun: func(cmd *cobra.Command, args []string) {
		// for all assets
		viper.BindPFlag("platform-id", cmd.Flags().Lookup("platform-id"))

		viper.BindPFlag("inventory-file", cmd.Flags().Lookup("inventory-file"))
		viper.BindPFlag("inventory-ansible", cmd.Flags().Lookup("inventory-ansible"))
		viper.BindPFlag("inventory-domainlist", cmd.Flags().Lookup("inventory-domainlist"))
	},
}

var vulnCmdRun = func(cmd *cobra.Command, runtime *providers.Runtime, cliRes *plugin.ParseCLIRes) {
	conf, err := getCobraScanConfig(cmd, runtime, cliRes)
	// conf := cnquery_app.ParseShellConfig(cmd, cliRes)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to prepare config")
	}

	unauthedErrorMsg := "vulnerability scan requires authentication, login with `cnspec login --token`"
	if runtime.UpstreamConfig == nil {
		log.Fatal().Msg(unauthedErrorMsg)
	}

	res, err := runtime.Provider.Instance.Plugin.Connect(&plugin.ConnectReq{
		Features: conf.Features,
		Asset:    cliRes.Asset,
		Upstream: runtime.UpstreamConfig,
	}, nil)
	if err != nil {
		log.Fatal().Err(err).Msg("could not load asset information")
	}
	runtime.Provider.Connection = res

	// when we close the shell, we need to close the backend and store the recording
	onCloseHandler := func() {
		// close backend connection
		runtime.Close()
	}

	shellOptions := []shell.ShellOption{}
	shellOptions = append(shellOptions, shell.WithOnCloseListener(onCloseHandler))
	shellOptions = append(shellOptions, shell.WithFeatures(conf.Features))

	if conf.runtime.UpstreamConfig != nil {
		shellOptions = append(shellOptions, shell.WithUpstreamConfig(conf.runtime.UpstreamConfig))
	}

	sh, err := shell.New(runtime, shellOptions...)
	if err != nil {
		log.Error().Err(err).Msg("failed to initialize cnspec shell")
	}

	vulnReportQuery := "asset.vulnerabilityReport"
	vulnReportDatapointChecksum := executor.MustGetOneDatapoint(executor.MustCompile(vulnReportQuery))
	_, results, err := sh.RunOnce(vulnReportQuery)
	if err != nil {
		log.Error().Err(err).Msg("failed to run query")
		return
	}

	// render vulnerability report
	var vulnReport mvd.VulnReport
	value, ok := results[vulnReportDatapointChecksum]
	if !ok {
		log.Error().Msg("could not find advisory report\n\n")
		return
	}

	if value == nil || value.Data == nil {
		log.Error().Msg("could not load advisory report\n\n")
		return
	}

	if value.Data.Error != nil {
		err := value.Data.Error
		if status, ok := status.FromError(err); ok {
			code := status.Code()
			switch code {
			case codes.Unauthenticated:
				log.Fatal().Msg(unauthedErrorMsg)
			default:
				log.Err(value.Data.Error).Msg("could not load advisory report")
				return
			}
		}
	}

	rawData := value.Data.Value
	cfg := &mapstructure.DecoderConfig{
		Metadata: nil,
		Result:   &vulnReport,
		TagName:  "json",
	}
	decoder, _ := mapstructure.NewDecoder(cfg)
	err = decoder.Decode(rawData)
	if err != nil {
		log.Error().Msg("could not decode advisory report\n\n")
		return
	}

	target := runtime.Provider.Connection.Asset.Name
	if target == "" {
		target = runtime.Provider.Connection.Asset.Mrn
	}

	printVulns(&vulnReport, conf, target)
}

func printVulns(report *mvd.VulnReport, conf *scanConfig, target string) {
	// print the output using the specified output format
	r, err := reporter.New("full")
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	logger.DebugDumpJSON("vulnReport", report)
	if err = r.PrintVulns(report, os.Stdout, target); err != nil {
		log.Fatal().Err(err).Msg("failed to print")
	}
}
