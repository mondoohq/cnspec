package cmd

import (
	"context"
	"os"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.mondoo.com/cnquery"
	cnquery_cmd "go.mondoo.com/cnquery/apps/cnquery/cmd"
	"go.mondoo.com/cnquery/cli/config"
	"go.mondoo.com/cnquery/cli/execruntime"
	"go.mondoo.com/cnquery/cli/inventoryloader"
	"go.mondoo.com/cnquery/cli/prof"
	"go.mondoo.com/cnquery/cli/sysinfo"
	"go.mondoo.com/cnquery/motor/asset"
	v1 "go.mondoo.com/cnquery/motor/inventory/v1"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/upstream"
	"go.mondoo.com/cnspec"
	"go.mondoo.com/cnspec/apps/cnspec/cmd/backgroundjob"
	cnspec_config "go.mondoo.com/cnspec/apps/cnspec/cmd/config"
	"go.mondoo.com/cnspec/policy/scan"
	"go.mondoo.com/ranger-rpc"
)

// we send a 78 exit code to prevent systemd from restart
// NOTE: if we change the code here we also need to adapt the systemd service
const ConfigurationErrorCode = 78

func init() {
	rootCmd.AddCommand(serveCmd)
	// background scan flags
	serveCmd.Flags().Int("timer", 60, "scan interval in minutes")
	serveCmd.Flags().MarkHidden("timer")
	// set inventory
	serveCmd.Flags().String("inventory-file", "", "Set the path to the inventory file")
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start cnspec in background mode",

	PreRun: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("timer", cmd.Flags().Lookup("timer"))
		viper.BindPFlag("inventory-file", cmd.Flags().Lookup("inventory-file"))
	},
	Run: func(cmd *cobra.Command, args []string) {
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
			log.Error().Err(err).Msg("could not load configuration")
			// we return the specific error code to prevent systemd from restarting
			os.Exit(cnquery_cmd.ConfigurationErrorCode)
		}

		ctx := cnquery.SetFeatures(context.Background(), cnquery.DefaultFeatures)

		if conf != nil && conf.UpstreamConfig != nil {
			hc := backgroundjob.NewHealthPinger(ctx, conf.UpstreamConfig.HttpClient, conf.UpstreamConfig.ApiEndpoint, 5*time.Minute)
			hc.Start()
			defer hc.Stop()
		}

		bj, err := backgroundjob.New()
		if err != nil {
			log.Fatal().Err(err).Msg("could not start background listener")
		}

		bj.Run(func() error {
			// TODO: check in every 5 min via timer, init time in Background job
			result, err := RunScan(conf, scan.DisableProgressBar())
			if err != nil {
				log.Error().Err(err).Msg("could not successfully complete scan")
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
	},
}

func getServeConfig() (*scanConfig, error) {
	opts, optsErr := cnspec_config.ReadConfig()
	if optsErr != nil {
		log.Fatal().Err(optsErr).Msg("could not load configuration")
	}
	config.DisplayUsedConfig()

	logClientInfo(opts.SpaceMrn, opts.AgentMrn, opts.ServiceAccountMrn)

	// display activated features
	if len(opts.Features) > 0 {
		log.Info().Strs("features", opts.Features).Msg("user activated features")
	}

	conf := scanConfig{
		Features:   opts.GetFeatures(),
		DoRecord:   viper.GetBool("record"),
		ReportType: scan.ReportType_ERROR,
		Output:     "",
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
		conf.Inventory.ApplyCategory(asset.AssetCategory_CATEGORY_CICD)
	}

	serviceAccount := opts.GetServiceCredential()
	if serviceAccount != nil {
		certAuth, err := upstream.NewServiceAccountRangerPlugin(serviceAccount)
		if err != nil {
			return nil, errors.Wrap(err, errorMessageServiceAccount)
		}
		plugins := []ranger.ClientPlugin{certAuth}
		// determine information about the client
		sysInfo, err := sysinfo.GatherSystemInfo()
		if err != nil {
			log.Warn().Err(err).Msg("could not gather client information")
		}
		plugins = append(plugins, defaultRangerPlugins(sysInfo, opts.GetFeatures())...)
		log.Info().Msg("using service account credentials")
		conf.UpstreamConfig = &resources.UpstreamConfig{
			SpaceMrn:    opts.GetParentMrn(),
			ApiEndpoint: opts.UpstreamApiEndpoint(),
			Plugins:     plugins,
		}
	}

	// set up the http client to include proxy config
	httpClient, err := opts.GetHttpClient()
	if err != nil {
		log.Error().Err(err).Msg("error while setting up httpclient")
		os.Exit(ConfigurationErrorCode)
	}
	if conf.UpstreamConfig == nil {
		conf.UpstreamConfig = &resources.UpstreamConfig{}
	}
	conf.UpstreamConfig.HttpClient = httpClient

	conf.Inventory, err = inventoryloader.ParseOrUse(nil, viper.GetBool("insecure"))
	if err != nil {
		return nil, errors.Wrap(err, "could not load configuration")
	}

	// fall back to local machine if no inventory was localed
	if conf.Inventory == nil || conf.Inventory.Spec == nil || len(conf.Inventory.Spec.Assets) == 0 {
		log.Info().Msg("configure inventory to scan local operating system")
		conf.Inventory = v1.New(v1.WithAssets(&asset.Asset{
			Connections: []*providers.Config{
				{
					Backend: providers.ProviderType_LOCAL_OS,
				},
			},
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
