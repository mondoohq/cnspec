package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/cockroachdb/errors"
	"github.com/mattn/go-isatty"
	"github.com/mitchellh/mapstructure"
	"github.com/muesli/termenv"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	cnquery_app "go.mondoo.com/cnquery/apps/cnquery/cmd"
	"go.mondoo.com/cnquery/apps/cnquery/cmd/builder"
	"go.mondoo.com/cnquery/cli/components"
	"go.mondoo.com/cnquery/cli/printer"
	"go.mondoo.com/cnquery/cli/shell"
	"go.mondoo.com/cnquery/cli/theme"
	"go.mondoo.com/cnquery/explorer/executor"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery"
	"go.mondoo.com/cnquery/motor/discovery/common"
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
	Short: "Scans a target for Vulnerabilities",
	CommonFlags: func(cmd *cobra.Command) {
		// inventories for multi-asset scan
		cmd.Flags().String("inventory-file", "", "path to inventory file")
		cmd.Flags().Bool("inventory-ansible", false, "set inventory format to ansible")
		cmd.Flags().Bool("inventory-domainlist", false, "set inventory format to domain list")

		// policies & incognito mode
		cmd.Flags().Bool("incognito", false, "incognito mode. do not report scan results to the Mondoo platform.")
		cmd.Flags().StringSlice("policy", nil, "list of policies to be executed (requires incognito mode), multiple policies can be passed in via --policy POLICY")
		cmd.Flags().StringSliceP("policy-bundle", "f", nil, "path to local policy bundle file")
		// flag completion command
		cmd.RegisterFlagCompletionFunc("policy", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return getPoliciesForCompletion(), cobra.ShellCompDirectiveDefault
		})

		// individual asset flags
		cmd.Flags().StringP("password", "p", "", "password e.g. for ssh/winrm")
		cmd.Flags().Bool("ask-pass", false, "ask for connection password")
		cmd.Flags().StringP("identity-file", "i", "", "selects a file from which the identity (private key) for public key authentication is read")
		cmd.Flags().String("id-detector", "", "user-override for platform id detection mechanism, supported are "+strings.Join(providers.AvailablePlatformIdDetector(), ", "))

		cmd.Flags().String("path", "", "path to a local file or directory that the connection should use")
		cmd.Flags().StringToString("option", nil, "addition connection options, multiple options can be passed in via --option key=value")
		cmd.Flags().String("discover", common.DiscoveryAuto, "enable the discovery of nested assets. Supported are 'all|instances|host-instances|host-machines|container|container-images|pods|cronjobs|statefulsets|deployments|jobs|replicasets|daemonsets'")
		cmd.Flags().StringToString("discover-filter", nil, "additional filter for asset discovery")
		cmd.Flags().StringToString("annotation", nil, "add an annotation to the asset") // user-added, editable

		// global asset flags
		cmd.Flags().Bool("insecure", false, "Disable TLS/SSL checks or SSH hostkey config")
		cmd.Flags().Bool("sudo", false, "Elevate privileges with sudo")
		cmd.Flags().Int("score-threshold", 0, "if any score falls below the threshold, exit 1")
		cmd.Flags().Bool("record", false, "Record backend calls")
		cmd.Flags().MarkHidden("record")

		// v6 should make detect-cicd and category flag public, default for "detect-cicd" should switch to true
		cmd.Flags().Bool("detect-cicd", true, "attempt to detect CI/CD environments and sets the asset category to 'cicd' if detected")
		cmd.Flags().String("category", "fleet", "sets the category for the assets 'fleet|cicd'")
		cmd.Flags().MarkHidden("category")

		// output rendering
		cmd.Flags().StringP("output", "o", "compact", "set output format: "+reporter.AllFormats())
		cmd.Flags().BoolP("json", "j", false, "set output to JSON (shorthand)")
		cmd.Flags().Bool("no-pager", false, "disable interactive scan output pagination")
		cmd.Flags().String("pager", "", "enable scan output pagination with custom pagination command. default is 'less -R'")
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
	Docs: builder.CommandsDocs{
		Entries: map[string]builder.CommandDocsEntry{
			"local": {
				Short: "Connect to a local machine",
			},
			"vagrant": {
				Short: "Scan a Vagrant host",
			},
			"ssh": {
				Short: "Scan a SSH target",
			},
			"winrm": {
				Short: "Scan a WinRM target",
			},
			"container": {
				Short: "Connect to a container, an image, or a registry",
				Long: `Connect to a container, a container image, or a container registry. By default
we will try to auto-detect the container or image from the provided ID, even
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
				Short: "Connect to a container image",
			},
			"container-registry": {
				Short: "Connect to a container registry",
				Long: `Connect to a container registry. Supports more parameters for different registries:

    cnspec vuln container registry harbor.lunalectric.com/project/repository
    cnspec vuln container registry yourname.azurecr.io
    cnspec vuln container registry 123456789.dkr.ecr.us-east-1.amazonaws.com/repository
`,
			},
			"docker": {
				Short: "Connect to a Docker container or image",
				Long: `Connect to a Docker container or image by automatically detecting the provided ID.
You can also specify a subcommand to narrow the scan to containers or images.

    cnspec vuln docker b62b276baab6

    cnspec vuln docker container b62b
    cnspec vuln docker image ubuntu:latest
`,
			},
			"docker-container": {
				Short: "Connect to a Docker container",
				Long: `Connect to a Docker container. Can be specified as the container ID (e.g. b62b276baab6)
or container name (e.g. elated_poincare).`,
			},
			"docker-image": {
				Short: "Connect to a Docker image",
				Long: `Connect to a Docker image. Can be specified as the image ID (e.g. b6f507652425)
or the image name (e.g. ubuntu:latest).`,
			},
			"kubernetes": {
				Short: "Connect to a Kubernetes cluster or manifest",
			},
			"aws-ec2": {
				Short: "Connect to an AWS instance using one of the available connectors",
			},
			"aws-ec2-connect": {
				Short: "Connect to an AWS instance using EC2 Instance Connect",
			},
			"aws-ec2-ebs-instance": {
				Short: "Connect to an AWS instance using an EBS volume scan (requires AWS host)",
				Long: `Connect to an AWS instance using an EBS volume scan. This requires that the
scan be executed on an instance that is running inside of AWS.`,
			},
			"aws-ec2-ebs-volume": {
				Short: "Connect to a specific AWS volume using the EBS volume scan functionality (requires AWS host)",
				Long: `Connect to a specific AWS volume using an EBS volume scan. This requires that the
scan be executed on an instance that is running inside of AWS.`,
			},
			"aws-ec2-ebs-snapshot": {
				Short: "Connect to a specific AWS snapshot using the EBS volume scan functionality (requires AWS host)",
				Long: `Connect to a specific AWS snapshot using an EBS volume scan. This requires that the
scan be executed on an instance that is running inside of AWS.`,
			},
			"aws-ec2-ssm": {
				Short: "Connect to an AWS instance using the AWS Systems Manager to connect",
			},
			"gcp-gcr": {
				Short: "Connect to a Google Container Registry (GCR)",
			},
			"vsphere": {
				Short: "Connect to a VMware vSphere API endpoint",
			},
		},
	},
	Run: func(cmd *cobra.Command, args []string, provider providers.ProviderType, assetType builder.AssetType) {
		conf, err := cnquery_app.GetCobraShellConfig(cmd, args, provider, assetType)
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

		backend, err := provider_resolver.OpenAssetConnection(ctx, connectAsset, im.GetCredential, false)
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

		header := fmt.Sprintf("\nTarget:     %s\n", target)
		b.WriteString(termenv.String(header).Foreground(theme.DefaultTheme.Colors.Primary).String())
		summaryDivider := strings.Repeat("=", utf8.RuneCountInString(header))
		b.WriteString(termenv.String(summaryDivider + "\n\n").Foreground(theme.DefaultTheme.Colors.Secondary).String())
		b.WriteString(reporter.RenderVulnerabilityStats(&vulnReport))
		b.WriteString(reporter.RenderVulnReport(&vulnReport))
		fmt.Println(b.String())
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
