package cmd

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.mondoo.com/cnquery"
	"go.mondoo.com/cnquery/apps/cnquery/cmd/builder"
	"go.mondoo.com/cnquery/cli/components"
	"go.mondoo.com/cnquery/cli/config"
	"go.mondoo.com/cnquery/cli/execruntime"
	"go.mondoo.com/cnquery/cli/inventoryloader"
	"go.mondoo.com/cnquery/motor/asset"
	v1 "go.mondoo.com/cnquery/motor/inventory/v1"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/upstream"
	cnspec_config "go.mondoo.com/cnspec/apps/cnspec/cmd/config"
	"go.mondoo.com/cnspec/cli/reporter"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/cnspec/policy/scan"
	"go.mondoo.com/ranger-rpc"
)

func init() {
	rootCmd.AddCommand(policyScanCmd)
}

var policyScanCmd = builder.NewProviderCommand(builder.CommandOpts{
	Use:   "scan",
	Short: "Scan assets with one or more polices",
	Long: `
This command triggers a new policy scan for an asset. By default, the local
system is scanned with its pre-configured policies:

    $ cnspec scan local

Users can also manually select a local policy to execute and run it without
storing results in the server:

    $ cnspec scan local --policy-bundle policyfile.yaml --incognito

In addition, mondoo can scan assets remotely via ssh. By default, the operating system
ssh agent and ssh config configuration is used to retrieve the credentials:

    $ cnspec scan ssh ec2-user@52.51.185.215
    $ cnspec scan ssh ec2-user@52.51.185.215:2222

Mondoo supports scanning AWS, Azure, and GCP accounts and instances.
Find out more in each sub-commands help menu. Here are a few examples:

    $ cnspec scan aws --region us-east-1
    $ cnspec scan azure --subscription ID --group NAME
    $ cnspec scan gcp --project ID

You can also access docker containers and images. This supports both local containers
and images as well as images in docker registries:

    $ cnspec scan docker container b62b276baab6
    $ cnspec scan docker image ubuntu:latest

Additionally, you can quickly scan a container registry:

    $ cnspec scan container registry harbor.yourdomain.com
    $ cnspec scan container registry 123456789.dkr.ecr.us-east-1.amazonaws.com/repository

Mondoo also support GCP's container registry, GCR:

    $ cnspec scan gcp gcr PROJECT_ID

Vagrant is supported as well:
   
    $ cnspec scan vagrant HOST

You can also leverage an inventory file:

    $ cnspec scan --inventory-file inventory.yml

You can also leverage your existing ansible inventory:

    $ ansible-inventory -i hosts.ini --list | cnspec scan --inventory-ansible

Further documentation is available at https://mondoo.com/docs/
	`,
	Docs: builder.CommandsDocs{
		Entries: map[string]builder.CommandDocsEntry{
			"local": {
				Short: "Scan a local target",
			},
			"mock": {
				Short: "Scan a mock target (a simulated asset)",
				Long: `Scan a mock target, i.e. a simulated asset, whose data was recorded beforehand.
Provide the recording with mock data as an argument:

    cnspec scan container ubuntu:latest --record
    cnspec scan mock recording-20220519173543.toml
`,
			},
			"vagrant": {
				Short: "Scan a Vagrant host",
			},
			"terraform": {
				Short: "Scan all Terraform files in a path (.tf files)",
			},
			"ssh": {
				Short: "Scan a SSH target",
			},
			"winrm": {
				Short: "Scan a WinRM target",
			},
			"container": {
				Short: "Scan a container, an image, or a registry",
				Long: `Scan a container, a container image, or a container registry. By default
we will try to auto-detect the container or image from the provided ID, even
if it's not the full ID:

    cnspec scan container b62b276baab6
    cnspec scan container b62
    cnspec scan container ubuntu:latest

You can also explicitly request the scan of an image or a container registry:

    cnspec scan container image ubuntu:20.04
    cnspec scan container registry harbor.yourdomain.com/project/repository
`,
			},
			"container-image": {
				Short: "Scan a container image",
			},
			"container-registry": {
				Short: "Scan a container registry",
				Long: `Scan a container registry. Supports more parameters for different registries:

    cnspec scan container registry harbor.yourdomain.com/project/repository
    cnspec scan container registry yourname.azurecr.io
    cnspec scan container registry 123456789.dkr.ecr.us-east-1.amazonaws.com/repository
`,
			},
			"docker": {
				Short: "Scan a Docker container or image",
				Long: `Scan a Docker container or image by automatically detecting the provided ID.
You can also specify a subcommand to narrow the scan to containers or images.

    cnspec scan docker b62b276baab6

    cnspec scan docker container b62b
    cnspec scan docker image ubuntu:latest
`,
			},
			"docker-container": {
				Short: "Scan a Docker container",
				Long: `Scan a Docker container. Can be specified as the container ID (e.g. b62b276baab6)
or container name (e.g. elated_poincare).`,
			},
			"docker-image": {
				Short: "Scan a Docker image",
				Long: `Scan a Docker image. Can be specified as the image ID (e.g. b6f507652425)
or the image name (e.g. ubuntu:latest).`,
			},
			"kubernetes": {
				Short: "Scan a Kubernetes cluster",
			},
			"aws": {
				Short: "Scan an AWS account or instance",
				Long: `Scan an AWS account or EC2 instance. It will use your local AWS configuration
for the account scan. See the subcommands to scan EC2 instances.`,
			},
			"aws-ec2": {
				Short: "Scan an AWS instance using one of the available connectors",
			},
			"aws-ec2-connect": {
				Short: "Scan an AWS instance using EC2 Instance Connect",
			},
			"aws-ec2-ebs-instance": {
				Short: "Scan an AWS instance using an EBS volume scan (requires AWS host)",
				Long: `Scan an AWS instance using an EBS volume scan. This requires that the
scan be executed on an instance that is running inside of AWS.`,
			},
			"aws-ec2-ebs-volume": {
				Short: "Scan a specific AWS volume using the EBS volume scan functionality (requires AWS host)",
				Long: `Scan a specific AWS volume using an EBS volume scan. This requires that the
scan be executed on an instance that is running inside of AWS.`,
			},
			"aws-ec2-ebs-snapshot": {
				Short: "Scan a specific AWS snapshot using the EBS volume scan functionality (requires AWS host)",
				Long: `Scan a specific AWS snapshot using an EBS volume scan. This requires that the
scan be executed on an instance that is running inside of AWS.`,
			},
			"aws-ec2-ssm": {
				Short: "Scan an AWS instance using the AWS Systems Manager to connect",
			},
			"azure": {
				Short: "Scan a Microsoft Azure account or instance",
				Long: `Scan a Microsoft Azure account or instance. It will use your local Azure
configuration for the account scan. To scan your Azure compute, you need to
configure your Azure credentials and have SSH access to your instances.`,
			},
			"gcp": {
				Short: "Scan a Google Cloud Platform (GCP) account",
			},
			"gcp-gcr": {
				Short: "Scan a Google Container Registry (GCR)",
			},
			"vsphere": {
				Short: "Scan a VMware vSphere API endpoint",
			},
			"vsphere-vm": {
				Short: "Scan a VMware vSphere VM",
			},
			"github": {
				Short: "Scan a GitHub organization or repository",
			},
			"github-org": {
				Short: "Scan a GitHub organization",
			},
			"github-repo": {
				Short: "Scan a GitHub repository",
			},
			"gitlab": {
				Short: "Scan a GitLab group",
			},
			"ms365": {
				Short: "Scan a Microsoft 365 endpoint",
				Long: `
This command triggers a new policy scan for Microsoft 365:

    $ cnspec scan ms365 --tenant-id {tennant id} --client-id {client id} --client-secret {client secret}

This example connects to Microsoft 365 using the PKCS #12 formatted certificate:

    $ cnspec scan ms365 --tenant-id {tennant id} --client-id {client id} --certificate-path {certificate.pfx} --certificate-secret {certificate secret}
    $ cnspec scan ms365 --tenant-id {tennant id} --client-id {client id} --certificate-path {certificate.pfx} --ask-pass
`,
			},
			"host": {
				Short: "Scan a host endpoint",
			},
			"arista": {
				Short: "Scan an Arista endpoint",
			},
		},
	},
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return []string{"yml", "yaml", "json"}, cobra.ShellCompDirectiveFilterFileExt
		}
		return []string{}, cobra.ShellCompDirectiveNoFileComp
	},
	CommonFlags: func(cmd *cobra.Command) {
		// inventories for multi-asset scan
		cmd.Flags().String("inventory-file", "", "path to inventory file")
		cmd.Flags().String("inventory", "", "inventory file")
		cmd.Flags().MarkDeprecated("inventory", "use new `inventory-file` flag instead")
		cmd.Flags().Bool("inventory-ansible", false, "set inventory format to ansible")
		cmd.Flags().Bool("ansible-inventory", false, "set inventory format to ansible")
		cmd.Flags().MarkDeprecated("ansible-inventory", "use the new flag `inventory-ansible` instead")
		cmd.Flags().Bool("inventory-domainlist", false, "set inventory format to domain list")
		cmd.Flags().Bool("domainlist-inventory", false, "set inventory format to domain list")
		cmd.Flags().MarkDeprecated("domainlist-inventory", "use the new flag `inventory-domainlist` instead")

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
		cmd.Flags().String("discover", "", "enable the discovery of nested assets. Supported are 'all|instances|host-instances|host-machines|container|container-images|pods|cronjobs|statefulsets|deployments|jobs|replicasets|daemonsets'")
		cmd.Flags().StringToString("discover-filter", nil, "additional filter for asset discovery")
		cmd.Flags().StringToString("annotation", nil, "add an annotation to the asset") // user-added, editable

		// global asset flags
		cmd.Flags().Bool("insecure", false, "disable TLS/SSL checks or SSH hostkey config")
		cmd.Flags().Bool("sudo", false, "run with sudo")
		cmd.Flags().Int("score-threshold", 0, "if any score falls below the threshold, exit 1")
		cmd.Flags().Bool("record", false, "record backend calls")
		cmd.Flags().MarkHidden("record")

		// v6 should make detect-cicd and category flag public, default for "detect-cicd" should switch to true
		cmd.Flags().Bool("detect-cicd", true, "attempt to detect CI/CD environments and sets the asset category to 'cicd' if detected")
		cmd.Flags().String("category", "fleet", "sets the category for the assets 'fleet|cicd'")
		cmd.Flags().MarkHidden("category")

		// output rendering
		cmd.Flags().StringP("output", "o", "compact", "set output format: "+reporter.AllFormats())
		cmd.Flags().Bool("no-pager", false, "disable interactive scan output pagination")
		cmd.Flags().String("pager", "", "enable scan output pagination with custom pagination command. default is 'less -R'")
	},
	CommonPreRun: func(cmd *cobra.Command, args []string) {
		// multiple assets mapping
		viper.BindPFlag("inventory-file", cmd.Flags().Lookup("inventory-file"))
		viper.BindPFlag("inventory-ansible", cmd.Flags().Lookup("inventory-ansible"))
		viper.BindPFlag("inventory-domainlist", cmd.Flags().Lookup("inventory-domainlist"))
		viper.BindPFlag("policy-bundle", cmd.Flags().Lookup("policy-bundle"))
		viper.BindPFlag("id-detector", cmd.Flags().Lookup("id-detector"))
		viper.BindPFlag("detect-cicd", cmd.Flags().Lookup("detect-cicd"))
		viper.BindPFlag("category", cmd.Flags().Lookup("category"))

		// deprecated flags
		viper.BindPFlag("inventory", cmd.Flags().Lookup("inventory"))
		viper.BindPFlag("ansible-inventory", cmd.Flags().Lookup("ansible-inventory"))
		viper.BindPFlag("domainlist-inventory", cmd.Flags().Lookup("domainlist-inventory"))

		// for all assets
		viper.BindPFlag("incognito", cmd.Flags().Lookup("incognito"))
		viper.BindPFlag("insecure", cmd.Flags().Lookup("insecure"))
		viper.BindPFlag("policies", cmd.Flags().Lookup("policy"))
		viper.BindPFlag("sudo.active", cmd.Flags().Lookup("sudo"))

		viper.BindPFlag("score-threshold", cmd.Flags().Lookup("score-threshold"))

		viper.BindPFlag("output", cmd.Flags().Lookup("output"))
		// the logic is that noPager takes precedence over pager if both are sent
		viper.BindPFlag("no_pager", cmd.Flags().Lookup("no-pager"))
		viper.BindPFlag("pager", cmd.Flags().Lookup("pager"))
	},
	PreRun: func(cmd *cobra.Command, args []string) {
		// Special handling for users that want to see what output options are
		// available. We have to do this before printing the help because we
		// don't have a target connection or provider.
		output, _ := cmd.Flags().GetString("output")
		if output == "help" {
			fmt.Println("Available output formats: " + reporter.AllFormats())
			os.Exit(0)
		}
	},
	Run: func(cmd *cobra.Command, args []string, provider providers.ProviderType, assetType builder.AssetType) {
		conf, err := getCobraScanConfig(cmd, args, provider, assetType)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to prepare config")
		}

		err = conf.loadPolicies()
		if err != nil {
			log.Fatal().Err(err).Msg("failed to resolve policies")
		}

		report := RunScan(conf)
		printReports(report, conf, cmd)
	},
})

// helper method to retrieve the list of policies for the policy flag
func getPoliciesForCompletion() []string {
	policyList := []string{}

	// TODO: policy autocompletion

	sort.Strings(policyList)

	return policyList
}

type scanConfig struct {
	Features    cnquery.Features
	Inventory   *v1.Inventory
	Output      string
	PolicyPaths []string
	PolicyNames []string
	Bundle      *policy.Bundle

	IsIncognito bool
	DoRecord    bool

	UpstreamConfig *resources.UpstreamConfig
}

func getCobraScanConfig(cmd *cobra.Command, args []string, provider providers.ProviderType, assetType builder.AssetType) (*scanConfig, error) {
	opts, optsErr := cnspec_config.ReadConfig()
	if optsErr != nil {
		log.Fatal().Err(optsErr).Msg("could not load configuration")
	}
	config.DisplayUsedConfig()

	// display activated features
	if len(opts.Features) > 0 {
		log.Info().Strs("features", opts.Features).Msg("user activated features")
	}

	conf := scanConfig{
		Features:    opts.GetFeatures(),
		IsIncognito: viper.GetBool("incognito"),
		DoRecord:    viper.GetBool("record"),
		PolicyPaths: viper.GetStringSlice("policy-bundle"),
		PolicyNames: viper.GetStringSlice("policies"),
	}

	// if users want to get more information on available output options,
	// print them before executing the scan
	output, _ := cmd.Flags().GetString("output")
	if output == "help" {
		fmt.Println("Available output formats: " + reporter.AllFormats())
		os.Exit(0)
	}

	// check if the user used --password without a value
	askPass, err := cmd.Flags().GetBool("ask-pass")
	if err == nil && askPass {
		pass, err := components.AskPassword("Enter password: ")
		if err != nil {
			log.Fatal().Err(err).Msg("failed to get password")
		}
		cmd.Flags().Set("password", pass)
	}

	// determine the scan config from pipe or args
	flagAsset := builder.ParseTargetAsset(cmd, args, provider, assetType)
	conf.Inventory, err = inventoryloader.ParseOrUse(flagAsset, viper.GetBool("insecure"))
	if err != nil {
		return nil, errors.Wrap(err, "could not load configuration")
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
		log.Info().Msg("using service account credentials")
		certAuth, _ := upstream.NewServiceAccountRangerPlugin(serviceAccount)

		conf.UpstreamConfig = &resources.UpstreamConfig{
			SpaceMrn:    opts.GetParentMrn(),
			ApiEndpoint: opts.UpstreamApiEndpoint(),
			Plugins:     []ranger.ClientPlugin{certAuth},
		}
	}

	if len(conf.PolicyPaths) > 0 && !conf.IsIncognito {
		log.Warn().Msg("Scanning with local policy bundles will switch into --incognito mode by default. Your results will not be sent upstream.")
		conf.IsIncognito = true
	}

	if serviceAccount == nil && !conf.IsIncognito {
		log.Warn().Msg("No credentials provided. Switching to --incogito mode.")
		conf.IsIncognito = true
	}

	// print headline when it is not printed to yaml
	if output == "" {
		fmt.Fprintln(os.Stdout, cnspecLogo)
	}

	if conf.DoRecord {
		log.Info().Msg("enable recording of platform calls")
	}

	return &conf, nil
}

func (c *scanConfig) loadPolicies() error {
	if c.IsIncognito {
		if len(c.PolicyPaths) == 0 {
			return nil
		}

		bundle, err := policy.BundleFromPaths(c.PolicyPaths...)
		if err != nil {
			return err
		}
		c.Bundle = bundle
		return nil
	}

	return errors.New("Cannot yet resolve policies other than incognito")
}

func RunScan(config *scanConfig) *policy.ReportCollection {
	scanner := scan.NewLocalScanner()
	ctx := cnquery.SetFeatures(context.Background(), config.Features)

	reports, err := scanner.RunIncognito(
		ctx,
		&scan.Job{
			DoRecord:      config.DoRecord,
			Inventory:     config.Inventory,
			Bundle:        config.Bundle,
			PolicyFilters: config.PolicyNames,
		})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to run scan")
	}

	return reports
}

func printReports(report *policy.ReportCollection, conf *scanConfig, cmd *cobra.Command) {
	// print the output using the specified output format
	r, err := reporter.New(conf.Output)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	r.UsePager, _ = cmd.Flags().GetBool("pager")
	r.Pager, _ = cmd.Flags().GetString("pager")
	r.IsIncognito = conf.IsIncognito

	if err = r.Print(report, os.Stdout); err != nil {
		log.Fatal().Err(err).Msg("failed to print")
	}
}
