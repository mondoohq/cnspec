package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/mattn/go-isatty"
	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.mondoo.com/cnquery/apps/cnquery/cmd/builder"
	"go.mondoo.com/cnquery/apps/cnquery/cmd/builder/common"
	"go.mondoo.com/cnquery/cli/components"
	"go.mondoo.com/cnquery/cli/printer"
	"go.mondoo.com/cnquery/cli/shell"
	"go.mondoo.com/cnquery/cli/theme"
	"go.mondoo.com/cnquery/explorer/executor"
	"go.mondoo.com/cnquery/logger"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery"
	discovery_common "go.mondoo.com/cnquery/motor/discovery/common"
	"go.mondoo.com/cnquery/motor/inventory"
	"go.mondoo.com/cnquery/motor/providers"
	provider_resolver "go.mondoo.com/cnquery/motor/providers/resolver"
	"go.mondoo.com/cnquery/upstream/mvd"
	"go.mondoo.com/cnspec/cli/reporter"
)

func init() {
	rootCmd.AddCommand(vulnCmd)
}

var vulnCmd = builder.NewProviderCommand(builder.CommandOpts{
	Use:   "vuln",
	Short: "Scans a target for Vulnerabilities.",
	CommonFlags: func(cmd *cobra.Command) {
		// inventories for multi-asset scan
		cmd.Flags().String("inventory-file", "", "Path to inventory file.")
		cmd.Flags().Bool("inventory-ansible", false, "Set inventory format to Ansible.")
		cmd.Flags().Bool("inventory-domainlist", false, "Set inventory format to domain list.")

		// policies & incognito mode
		cmd.Flags().Bool("incognito", false, "Incognito mode. Do not report scan results to Mondoo Platform.")
		cmd.Flags().StringSlice("policy", nil, "List policies to execute. This requires incognito mode. To scan multiple policies, pass `--policy POLICY`")
		cmd.Flags().StringSliceP("policy-bundle", "f", nil, "Path to local policy bundle file.")
		// flag completion command
		cmd.RegisterFlagCompletionFunc("policy", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return getPoliciesForCompletion(), cobra.ShellCompDirectiveDefault
		})

		// individual asset flags
		cmd.Flags().StringP("password", "p", "", "Password, such as for SSH/WinRM.")
		cmd.Flags().Bool("ask-pass", false, "Ask for connection password.")
		cmd.Flags().StringP("identity-file", "i", "", "Select a file from which too read the identity (private key) for public key authentication.")
		cmd.Flags().String("id-detector", "", "User override for platform ID detection mechanism. Supported: "+strings.Join(providers.AvailablePlatformIdDetector(), ", "))

		cmd.Flags().String("path", "", "Path to a local file or directory for the connection to use")
		cmd.Flags().StringToString("option", nil, "Additional connection options. You can pass multiple options using `--option key=value`")
		cmd.Flags().String("discover", discovery_common.DiscoveryAuto, "Enable the discovery of nested assets. Supported: 'all|instances|host-instances|host-machines|container|container-images|pods|cronjobs|statefulsets|deployments|jobs|replicasets|daemonsets'")
		cmd.Flags().StringToString("discover-filter", nil, "Additional filter for asset discovery.")
		cmd.Flags().StringToString("annotation", nil, "Add an annotation to the asset.") // user-added, editable

		// global asset flags
		cmd.Flags().Bool("insecure", false, "Disable TLS/SSL checks or SSH hostkey config.")
		cmd.Flags().Bool("sudo", false, "Elevate privileges with sudo.")
		cmd.Flags().Int("score-threshold", 0, "If any score falls below the threshold, exit 1.")
		cmd.Flags().Bool("record", false, "Record backend calls.")
		cmd.Flags().MarkHidden("record")

		// v6 should make detect-cicd and category flag public, default for "detect-cicd" should switch to true
		cmd.Flags().Bool("detect-cicd", true, "Try to detect CI/CD environments. If successful, sets the asset category to 'cicd'.")
		cmd.Flags().String("category", "fleet", "Set the category for the assets to 'fleet|cicd'.")
		cmd.Flags().MarkHidden("category")

		// output rendering
		cmd.Flags().StringP("output", "o", "compact", "Set output format: "+reporter.AllFormats())
		cmd.Flags().BoolP("json", "j", false, "Set output to JSON (shorthand).")
		cmd.Flags().Bool("no-pager", false, "Disable interactive scan output pagination.")
		cmd.Flags().String("pager", "", "Enable scan output pagination with custom pagination command. The default is 'less -R'.")
	},
	CommonPreRun: func(cmd *cobra.Command, args []string) {
		// for all assets
		viper.BindPFlag("incognito", cmd.Flags().Lookup("incognito"))
		viper.BindPFlag("insecure", cmd.Flags().Lookup("insecure"))
		viper.BindPFlag("policies", cmd.Flags().Lookup("policy"))
		viper.BindPFlag("sudo.active", cmd.Flags().Lookup("sudo"))

		viper.BindPFlag("output", cmd.Flags().Lookup("output"))

		viper.BindPFlag("vault.name", cmd.Flags().Lookup("vault"))
		viper.BindPFlag("platform-id", cmd.Flags().Lookup("platform-id"))
	},
	Docs: common.CommandsDocs{
		Entries: map[string]common.CommandDocsEntry{
			"local": {
				Short: "Scan your local system.",
			},
			"vagrant": {
				Short: "Scan a Vagrant host.",
			},
			"ssh": {
				Short: "Scan a SSH target.",
			},
			"winrm": {
				Short: "Scan a WinRM target.",
			},
			"container": {
				Short: "Connect to a container, image, or registry.",
				Long: `Connect to a container, container image, or container registry. By default cnspec tries to auto-detect the container or image from the provided ID, even
if it's not the full ID:

    cnspec vuln container b62b276baab6
    cnspec vuln container b62
    cnspec vuln container ubuntu:latest

You can also explicitly connect to an image or a container registry:

    cnspec vuln container image ubuntu:20.04
    cnspec vuln container registry harbor.lunalectric.com/project/repository
`,
			},
			"container-image": {
				Short: "Connect to a container image.",
			},
			"container-registry": {
				Short: "Connect to a container registry.",
				Long: `Connect to a container registry. Supports more parameters for different registries:

    cnspec vuln container registry harbor.lunalectric.com/project/repository
    cnspec vuln container registry yourname.azurecr.io
    cnspec vuln container registry 123456789.dkr.ecr.us-east-1.amazonaws.com/repository
`,
			},
			"docker": {
				Short: "Connect to a Docker container or image.",
				Long: `Connect to a Docker container or image by automatically detecting the provided ID.
You can also specify a subcommand to narrow the scan to containers or images.

    cnspec vuln docker b62b276baab6

    cnspec vuln docker container b62b
    cnspec vuln docker image ubuntu:latest
`,
			},
			"docker-container": {
				Short: "Connect to a Docker container.",
				Long: `Connect to a Docker container. Can be specified as the container ID (such as b62b276baab6)
or container name (such as elated_poincare).`,
			},
			"docker-image": {
				Short: "Connect to a Docker image.",
				Long: `Connect to a Docker image. Can be specified as the image ID (such as b6f507652425)
or the image name (such as ubuntu:latest).`,
			},
			"kubernetes": {
				Short: "Connect to a Kubernetes cluster or local manifest files(s).",
			},
			"aws": {
				Short: "Connect to an AWS account or instance.",
				Long: `Connect to an AWS account or EC2 instance. cnspec uses your local AWS configuration
for the account scan. See the subcommands to scan EC2 instances.`,
			},
			"aws-ec2": {
				Short: "Connect to an AWS instance using one of the available connectors.",
			},
			"aws-ec2-connect": {
				Short: "Connect to an AWS instance using EC2 Instance Connect.",
			},
			"aws-ec2-ebs-instance": {
				Short: "Connect to an AWS instance using an EBS volume scan. This requires an AWS host.",
				Long: `Connect to an AWS instance using an EBS volume scan. This requires that the
scan execute on an instance that is running inside of AWS.`,
			},
			"aws-ec2-ebs-volume": {
				Short: "Connect to a specific AWS volume using an EBS volume scan. This requires an AWS host.",
				Long: `Connect to a specific AWS volume using an EBS volume scan. This requires that the
scan execute on an instance that is running inside of AWS.`,
			},
			"aws-ec2-ebs-snapshot": {
				Short: "Connect to a specific AWS snapshot using an EBS volume scan. This requires an AWS host.",
				Long: `Connect to a specific AWS snapshot using an EBS volume scan. This requires that the
scan execute on an instance that is running inside of AWS.`,
			},
			"aws-ec2-ssm": {
				Short: "Connect to an AWS instance using the AWS Systems Manager to connect.",
			},
			"azure": {
				Short: "Connect to a Microsoft Azure subscription or virtual machines.",
				Long: `Connect to a Microsoft Azure subscriptions or virtual machines. cnspec uses your local Azure
configuration for the account scan. To scan your Azure compute, you must
configure your Azure credentials and have SSH access to your virtual machines.`,
			},
			"gcp": {
				Short: "Connect to a Google Cloud Platform (GCP) project.",
			},
			"gcp-gcr": {
				Short: "Connect to a Google Container Registry (GCR).",
			},
			"vsphere": {
				Short: "Connect to a VMware vSphere API endpoint.",
			},
			"vsphere-vm": {
				Short: "Connect to a VMware vSphere VM.",
			},
			"host": {
				Short: "Connect to a host endpoint.",
			},
		},
	},
	Run: func(cmd *cobra.Command, args []string, provider providers.ProviderType, assetType builder.AssetType) {
		conf, err := getCobraScanConfig(cmd, args, provider, assetType)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to prepare config")
		}

		if conf.UpstreamConfig == nil {
			log.Fatal().Msg("vulnerability scan requires authentication, future versions will not have this restriction, login with `cnspec login --token`")
		}

		ctx := discovery.InitCtx(context.Background())

		log.Info().Msgf("discover related assets for %d asset(s)", len(conf.Inventory.Spec.Assets))
		im, err := inventory.New(inventory.WithInventory(conf.Inventory))
		if err != nil {
			log.Fatal().Err(err).Msg("could not load asset information")
		}
		assetErrors := im.Resolve(ctx)
		if len(assetErrors) > 0 {
			for a := range assetErrors {
				log.Error().Err(assetErrors[a]).Str("asset", a.Name).Msg("could not connect to asset")
			}
			log.Fatal().Msg("could not resolve assets")
		}

		assetList := im.GetAssets()
		log.Debug().Msgf("resolved %d assets", len(assetList))

		if len(assetList) == 0 {
			log.Fatal().Msg("could not find an asset that we can connect to")
		}

		platformID := ""
		var connectAsset *asset.Asset
		if len(assetList) == 1 {
			connectAsset = assetList[0]
		} else if len(assetList) > 1 && platformID != "" {
			connectAsset, err = filterAssetByPlatformID(assetList, platformID)
			if err != nil {
				log.Fatal().Err(err).Send()
			}
		} else if len(assetList) > 1 {
			isTTY := isatty.IsTerminal(os.Stdout.Fd())
			if isTTY {
				connectAsset = components.AssetSelect(assetList)
			} else {
				fmt.Println(components.AssetList(theme.OperatingSystemTheme, assetList))
				log.Fatal().Msg("cannot connect to more than one asset, use --platform-id to select a specific asset")
			}
		}

		if connectAsset == nil {
			log.Fatal().Msg("no asset selected")
		}

		backend, err := provider_resolver.OpenAssetConnection(ctx, connectAsset, im.GetCredsResolver(), false)
		if err != nil {
			log.Fatal().Err(err).Msg("could not connect to asset")
		}

		// when we close the shell, we need to close the backend and store the recording
		onCloseHandler := func() {
			// close backend connection
			backend.Close()
		}

		shellOptions := []shell.ShellOption{}
		shellOptions = append(shellOptions, shell.WithOnCloseListener(onCloseHandler))
		shellOptions = append(shellOptions, shell.WithFeatures(conf.Features))

		if conf.UpstreamConfig != nil {
			shellOptions = append(shellOptions, shell.WithUpstreamConfig(conf.UpstreamConfig))
		}

		sh, err := shell.New(backend, shellOptions...)
		if err != nil {
			log.Error().Err(err).Msg("failed to initialize cnspec shell")
		}

		vulnReportQuery := "platform.vulnerabilityReport"
		vulnReportDatapointChecksum := executor.MustGetOneDatapoint(executor.MustCompile(vulnReportQuery))
		_, results, err := sh.RunOnce(vulnReportQuery)

		// render vulnerability report
		print := printer.DefaultPrinter
		var b bytes.Buffer
		var vulnReport mvd.VulnReport
		value, ok := results[vulnReportDatapointChecksum]
		if !ok {
			b.WriteString(print.Error("could not find advisory report\n\n"))
			b.String()
			return
		}

		if value == nil || value.Data == nil {
			b.WriteString(print.Error("could not load advisory report\n\n"))
			b.String()
			return
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
			b.WriteString(print.Error("could not decode advisory report\n\n"))
			b.String()
			return
		}

		target := connectAsset.Name
		if target == "" {
			target = connectAsset.Mrn
		}

		printVulns(&vulnReport, conf, target)
	},
})

func filterAssetByPlatformID(assetList []*asset.Asset, selectionID string) (*asset.Asset, error) {
	var foundAsset *asset.Asset
	for i := range assetList {
		assetObj := assetList[i]
		for j := range assetObj.PlatformIds {
			if assetObj.PlatformIds[j] == selectionID {
				return assetObj, nil
			}
		}
	}

	if foundAsset == nil {
		return nil, errors.New("could not find an asset with the provided identifier: " + selectionID)
	}
	return foundAsset, nil
}

func printVulns(report *mvd.VulnReport, conf *scanConfig, target string) {
	// print the output using the specified output format
	r, err := reporter.New(conf.Output)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	logger.DebugDumpJSON("vulnReport", report)
	if err = r.PrintVulns(report, os.Stdout, target); err != nil {
		log.Fatal().Err(err).Msg("failed to print")
	}
}
