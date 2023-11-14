// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"context"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.mondoo.com/cnquery/v9"
	"go.mondoo.com/cnquery/v9/cli/config"
	cli_errors "go.mondoo.com/cnquery/v9/cli/errors"
	"go.mondoo.com/cnquery/v9/cli/execruntime"
	"go.mondoo.com/cnquery/v9/cli/inventoryloader"
	"go.mondoo.com/cnquery/v9/cli/prof"
	"go.mondoo.com/cnquery/v9/providers"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/upstream"
	"go.mondoo.com/cnspec/v9"
	"go.mondoo.com/cnspec/v9/apps/cnspec/cmd/backgroundjob"
	cnspec_config "go.mondoo.com/cnspec/v9/apps/cnspec/cmd/config"
	"go.mondoo.com/cnspec/v9/policy/scan"
)

// we send a 78 exit code to prevent systemd from restart
// NOTE: if we change the code here we also need to adapt the systemd service
const ConfigurationErrorCode = 78

func init() {
	rootCmd.AddCommand(serveCmd)
	// background scan flags
	serveCmd.Flags().Int("timer", 60, "scan interval in minutes")
	serveCmd.Flags().Int("splay", 60, "randomize the timer by up to this many minutes")
	serveCmd.Flags().MarkHidden("timer")
	serveCmd.Flags().MarkHidden("splay")
	// set inventory
	serveCmd.Flags().String("inventory-file", "", "Set the path to the inventory file")
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start cnspec in background mode.",

	PreRun: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("timer", cmd.Flags().Lookup("timer"))
		viper.BindPFlag("splay", cmd.Flags().Lookup("splay"))
		viper.BindPFlag("inventory-file", cmd.Flags().Lookup("inventory-file"))
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		prof.InitProfiler()

		// prevent colors on windows
		viper.Set("color", "none")

		// check if an inventory file exists
		if viper.GetString("inventory-file") == "" {
			inventoryFilePath, ok := config.InventoryPath(viper.ConfigFileUsed())
			if ok {
				log.Info().Str("path", inventoryFilePath).Msg("found inventory file")
				viper.Set("inventory-file", inventoryFilePath)
			}
		}

		// determine the scan config from pipe or args
		conf, err := getServeConfig()
		if err != nil {
			// we return the specific error code to prevent systemd from restarting
			return cli_errors.NewCommandError(errors.Wrap(err, "could not load configuration"), ConfigurationErrorCode)
		}

		ctx := cnquery.SetFeatures(context.Background(), cnquery.DefaultFeatures)

		if conf != nil && conf.runtime.UpstreamConfig != nil {
			client, err := conf.runtime.UpstreamConfig.InitClient()
			if err != nil {
				return cli_errors.NewCommandError(errors.Wrap(err, "could not initialize upstream client"), 1)
			}

			checkin, err := backgroundjob.NewCheckinPinger(ctx, client.HttpClient, client.ApiEndpoint, conf.AgentMrn, conf.runtime.UpstreamConfig, 2*time.Hour)
			if err != nil {
				log.Error().Err(err).Msg("could not initialize upstream check-in")
			} else {
				checkin.Start()
				defer checkin.Stop()
			}
		}

		bj, err := backgroundjob.New()
		if err != nil {
			return cli_errors.NewCommandError(errors.Wrap(err, "could not start background listener"), 1)
		}

		bj.Run(func() error {
			// Try to update the os provider before each scan
			err = updateProviders()
			if err != nil {
				log.Error().Err(err).Msg("could not update providers")
			}
			// TODO: check in every 5 min via timer, init time in Background job
			result, err := RunScan(conf, scan.DisableProgressBar())
			if err != nil {
				return cli_errors.NewCommandError(errors.Wrap(err, "could not successfully complete scan"), 1)
			}

			// log errors
			if result != nil && result.GetErrors() != nil && len(result.GetErrors()) > 0 {
				assetErrors := result.GetErrors()
				for a, err := range assetErrors {
					log.Error().Err(errors.New(err)).Str("asset", a).Msg("could not connect to asset")
				}
			}
			return nil
		})
		return nil
	},
}

func getServeConfig() (*scanConfig, error) {
	opts, optsErr := cnspec_config.ReadConfig()
	if optsErr != nil {
		return nil, errors.Wrap(optsErr, "could not load configuration")
	}
	config.DisplayUsedConfig()

	logClientInfo(opts.SpaceMrn, opts.AgentMrn, opts.ServiceAccountMrn)

	if len(opts.Features) > 0 {
		log.Info().Strs("features", opts.Features).Msg("user activated features")
	}

	// Since we don't know the runtime yet, i.e. when we go into listening mode
	// we may get to a variety of actual systems that we connect to,
	// we have to create a default runtime. This will be extended for anything
	// that the job runner throws at it.

	runtime := providers.DefaultRuntime()

	conf := scanConfig{
		Features:   opts.GetFeatures(),
		DoRecord:   viper.GetBool("record"),
		ReportType: scan.ReportType_ERROR,
		Output:     "",
		runtime:    runtime,
		AgentMrn:   opts.AgentMrn,
	}

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
	if serviceAccount != nil {
		// determine information about the client
		log.Info().Msg("using service account credentials")
		runtime.UpstreamConfig = &upstream.UpstreamConfig{
			SpaceMrn:    opts.GetParentMrn(),
			ApiEndpoint: opts.UpstreamApiEndpoint(),
			ApiProxy:    opts.APIProxy,
			Creds:       serviceAccount,
		}
	}

	optAnnotations := opts.Annotations
	if optAnnotations == nil {
		optAnnotations = map[string]string{}
	}
	var err error
	conf.Inventory, err = inventoryloader.ParseOrUse(nil, viper.GetBool("insecure"), optAnnotations)
	if err != nil {
		return nil, errors.Wrap(err, "could not load configuration")
	}

	// fall back to local machine if no inventory was localed
	if conf.Inventory == nil || conf.Inventory.Spec == nil || len(conf.Inventory.Spec.Assets) == 0 {
		log.Info().Msg("configure inventory to scan local operating system")
		conf.Inventory = inventory.New(inventory.WithAssets(&inventory.Asset{
			Connections: []*inventory.Config{{
				Type: "local",
			}},
		}))
	}

	return &conf, nil
}

func logClientInfo(spaceMrn string, clientMrn string, serviceAccountMrn string) {
	if spaceMrn == "" {
		spaceMrn = "unset"
	}
	if serviceAccountMrn == "" {
		serviceAccountMrn = "unset"
	}
	if clientMrn == "" {
		clientMrn = "unset"
	}
	version := cnspec.Version
	if version == "" {
		version = "unstable"
	}
	log.Info().Str("version", version).Str("space", spaceMrn).Str("service_account", serviceAccountMrn).Str("client", clientMrn).Msg("start cnspec")
}

func updateProviders() error {
	log.Debug().Msg("checking for provider updates")
	// force re-load from disk, in case it got updated outside the serve mode
	providers.CachedProviders = nil
	allProviders, err := providers.ListActive()
	if err != nil {
		return err
	}
	updatedProviders := []*providers.Provider{}
	for _, provider := range allProviders {
		if provider.Name == "mock" || provider.Name == "core" {
			continue
		}
		latestVersion, err := providers.LatestVersion(provider.Name)
		if err != nil {
			return err
		}
		if latestVersion != provider.Version {
			installed, err := providers.Install(provider.Name, "")
			if err != nil {
				return err
			}
			updatedProviders = append(updatedProviders, installed)
		} else {
			log.Debug().Str("provider", provider.Name).Str("version", provider.Version).Msg("provider is already up to date")
		}

	}
	providers.PrintInstallResults(updatedProviders)
	return nil
}
