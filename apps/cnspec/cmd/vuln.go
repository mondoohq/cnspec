// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"encoding/json"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.mondoo.com/cnquery/v10/cli/shell"
	"go.mondoo.com/cnquery/v10/explorer/executor"
	"go.mondoo.com/cnquery/v10/logger"
	"go.mondoo.com/cnquery/v10/providers"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/upstream/gql"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/upstream/mvd"
	"go.mondoo.com/cnspec/v10/cli/reporter"
	mondoogql "go.mondoo.com/mondoo-go"
	"go.mondoo.com/ranger-rpc/codes"
	"go.mondoo.com/ranger-rpc/status"
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
	conf, err := getCobraScanConfig(cmd, runtime, cliRes)
	// conf := cnquery_app.ParseShellConfig(cmd, cliRes)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to prepare config")
	}

	unauthedErrorMsg := "vulnerability scan requires authentication, login with `cnspec login --token`"
	if runtime.UpstreamConfig == nil {
		log.Fatal().Msg(unauthedErrorMsg)
	}

	err = runtime.Connect(&plugin.ConnectReq{
		Features: conf.Features,
		Asset:    cliRes.Asset,
		Upstream: runtime.UpstreamConfig,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("could not load asset information")
	}

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

	packagesQuery := "packages { name version origin format }"
	packagesDatapointChecksum := executor.MustGetOneDatapoint(executor.MustCompile(packagesQuery))
	codeBundle, results, err := sh.RunOnce(packagesQuery)
	if err != nil {
		log.Error().Err(err).Msg("failed to run query")
		return
	}

	// render vulnerability report
	value, ok := results[packagesDatapointChecksum]
	if !ok {
		log.Error().Msg("could not find packages data\n\n")
		return
	}

	if value == nil || value.Data == nil {
		log.Error().Msg("could not load packages data\n\n")
		return
	}

	if value.Data.Error != nil {
		log.Err(value.Data.Error).Msg("could not load packages data\n\n")
		return
	}

	packagesJson := value.Data.JSON(packagesDatapointChecksum, codeBundle)

	gqlPackages := []mondoogql.PackageInput{}
	err = json.Unmarshal(packagesJson, &gqlPackages)
	if err != nil {
		log.Error().Err(err).Msg("failed to unmarshal packages")
		return
	}

	client, err := runtime.UpstreamConfig.InitClient()
	if err != nil {
		if status, ok := status.FromError(err); ok {
			code := status.Code()
			switch code {
			case codes.Unauthenticated:
				log.Fatal().Msg(unauthedErrorMsg)
			default:
				log.Err(err).Msg("could not authenticate upstream")
				return
			}
		}
	}
	mondooClient, err := gql.NewClient(runtime.UpstreamConfig, client.HttpClient)
	if err != nil {
		log.Error().Err(err).Msg("could not initialize mondoo client")
		return
	}

	platform := runtime.Provider.Connection.GetAsset().GetPlatform()
	inputPlatform := mondoogql.PlatformInput{
		Name:    mondoogql.NewStringPtr(mondoogql.String(platform.Name)),
		Release: mondoogql.NewStringPtr(mondoogql.String(platform.Version)),
		Build:   mondoogql.NewStringPtr(mondoogql.String(platform.Build)),
	}
	inputLabels := []*mondoogql.KeyValueInput{}
	for k := range platform.Labels {
		inputLabels = append(inputLabels, &mondoogql.KeyValueInput{
			Key:   mondoogql.String(k),
			Value: mondoogql.NewStringPtr(mondoogql.String(platform.Labels[k])),
		})
	}
	inputPlatform.Labels = &inputLabels
	gqlVulnReport, err := mondooClient.GetIncognitoVulnReport(mondoogql.PlatformInput{
		Name:    mondoogql.NewStringPtr(mondoogql.String(platform.Name)),
		Release: mondoogql.NewStringPtr(mondoogql.String(platform.Version)),
	}, gqlPackages)
	if err != nil {
		log.Error().Err(err).Msg("could not load advisory report")
		return
	}

	vulnReport := gql.ConvertToMvdVulnReport(gqlVulnReport)

	target := runtime.Provider.Connection.Asset.Name
	if target == "" {
		target = runtime.Provider.Connection.Asset.Mrn
	}

	printVulns(vulnReport, conf, target)
}

func printVulns(report *mvd.VulnReport, conf *scanConfig, target string) {
	// print the output using the specified output format
	r := reporter.NewReporter(reporter.Formats[strings.ToLower(conf.OutputFormat)], false)

	logger.DebugDumpJSON("vulnReport", report)
	if err := r.PrintVulns(report, target); err != nil {
		log.Fatal().Err(err).Msg("failed to print")
	}
}
