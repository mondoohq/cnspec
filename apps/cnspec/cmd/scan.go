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
	cnquery_cmd "go.mondoo.com/cnquery/apps/cnquery/cmd"
	"go.mondoo.com/cnquery/apps/cnquery/cmd/builder"
	"go.mondoo.com/cnquery/apps/cnquery/cmd/builder/common"
	cnquery_config "go.mondoo.com/cnquery/apps/cnquery/cmd/config"
	"go.mondoo.com/cnquery/cli/components"
	"go.mondoo.com/cnquery/cli/config"
	"go.mondoo.com/cnquery/cli/execruntime"
	"go.mondoo.com/cnquery/cli/inventoryloader"
	"go.mondoo.com/cnquery/cli/prof"
	"go.mondoo.com/cnquery/cli/sysinfo"
	"go.mondoo.com/cnquery/logger"
	"go.mondoo.com/cnquery/motor/asset"
	discovery_common "go.mondoo.com/cnquery/motor/discovery/common"
	v1 "go.mondoo.com/cnquery/motor/inventory/v1"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/upstream"
	cnspec_config "go.mondoo.com/cnspec/apps/cnspec/cmd/config"
	"go.mondoo.com/cnspec/cli/reporter"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/cnspec/policy/scan"
	cnspec_upstream "go.mondoo.com/cnspec/upstream"
	"go.mondoo.com/ranger-rpc"
)

const (
	// allow sending reports to alternative URLs
	featureReportAlternateUrlEnv = "REPORT_URL"
)

func init() {
	rootCmd.AddCommand(policyScanCmd)
}

var policyScanCmd = builder.NewProviderCommand(builder.CommandOpts{
	Use:   "scan",
	Short: "Scan assets with one or more policies.",
	Long: `
This command triggers a new policy scan for an asset. By default, cnspec scans the local
system with its pre-configured policies:

    $ cnspec scan local

You can also manually select a local policy to execute and run it without
storing results in the server:

    $ cnspec scan local --policy-bundle policyfile.yaml --incognito

In addition, cnspec can scan assets remotely via SSH. By default, cnspec uses the operating system
SSH agent and SSH config to retrieve the credentials:

    $ cnspec scan ssh ec2-user@52.51.185.215
    $ cnspec scan ssh ec2-user@52.51.185.215:2222

cnspec supports scanning AWS, Azure, and GCP accounts and instances.
Find out more in each sub-commands help menu. Here are a few examples:

    $ cnspec scan aws --region us-east-1
    $ cnspec scan azure --subscription ID --group NAME
    $ cnspec scan gcp project ID

You can also access Docker containers and images. cnspec supports local containers
and images as well as images in Docker registries:

    $ cnspec scan docker container b62b276baab6
    $ cnspec scan docker image ubuntu:latest

Additionally, you can quickly scan a container registry:

    $ cnspec scan container registry harbor.lunalectric.com
    $ cnspec scan container registry 123456789.dkr.ecr.us-east-1.amazonaws.com/repository

cnspec also supports GCP's container registry, GCR:

    $ cnspec scan gcp gcr PROJECT_ID

Vagrant is supported as well:
   
    $ cnspec scan vagrant HOST

You can also use an inventory file:

    $ cnspec scan --inventory-file inventory.yml

This scan uses an existing Ansible inventory:

    $ ansible-inventory -i hosts.ini --list | cnspec scan --inventory-ansible

To learn more, read https://mondoo.com/docs/.
	`,
	Docs: common.CommandsDocs{
		Entries: map[string]common.CommandDocsEntry{
			"local": {
				Short: "Scan your local system.",
			},
			"mock": {
				Short: "Scan a mock target (a simulated asset).",
				Long: `Scan a mock target. This scans a simulated asset. We recorded the asset's data beforehand.
Provide the recording with mock data as an argument:

    cnspec scan container ubuntu:latest --record
    cnspec scan mock recording-20220519173543.toml
`,
			},
			"vagrant": {
				Short: "Scan a Vagrant host.",
			},
			"terraform": {
				Short: "Scan Terraform HCL (files.tf and directories), plan files (json), and state files (json).",
			},
			"ssh": {
				Short: "Scan an SSH target.",
			},
			"winrm": {
				Short: "Scan a WinRM target.",
			},
			"container": {
				Short: "Scan a container, image, or registry.",
				Long: `Scan a container, container image, or container registry. By default
cnspec tries to auto-detect the container or image from the provided ID, even
if it's not the full ID:

    cnspec scan container b62b276baab6
    cnspec scan container b62
    cnspec scan container ubuntu:latest

You can also explicitly request the scan of an image or a container registry:

    cnspec scan container image ubuntu:20.04
    cnspec scan container registry harbor.lunalectric.com/project/repository
`,
			},
			"container-image": {
				Short: "Scan a container image.",
			},
			"container-tar": {
				Short: "Scan an OCI container image from a tar file.",
				Long: `Scan an OCI container image by providing a path to the tar file: 

    cnspec scan container tar /path/to/image.tar
`,
			},
			"container-registry": {
				Short: "Scan a container registry.",
				Long: `Scan a container registry. This supports more parameters for different registries:

    cnspec scan container registry harbor.lunalectric.com/project/repository
    cnspec scan container registry yourname.azurecr.io
    cnspec scan container registry 123456789.dkr.ecr.us-east-1.amazonaws.com/repository
`,
			},
			"docker": {
				Short: "Scan a Docker container or image.",
				Long: `Scan a Docker container or image by automatically detecting the provided ID.
You can also specify a subcommand to narrow the scan to containers or images.

    cnspec scan docker b62b276baab6

    cnspec scan docker container b62b
    cnspec scan docker image ubuntu:latest
`,
			},
			"docker-container": {
				Short: "Scan a Docker container.",
				Long: `Scan a Docker container. You can specify the container ID (such as b62b276baab6)
or container name (such as elated_poincare).`,
			},
			"docker-image": {
				Short: "Scan a Docker image.",
				Long: `Scan a Docker image. You can specify the image ID (such as b6f507652425)
or the image name (such as ubuntu:latest).`,
			},
			"kubernetes": {
				Short: "Scan a Kubernetes cluster or local manifest file(s).",
			},
			"aws": {
				Short: "Scan an AWS account or instance.",
				Long: `Scan an AWS account or EC2 instance. cnspec uses your local AWS configuration
for the account scan. See the subcommands to scan EC2 instances.`,
			},
			"aws-ec2": {
				Short: "Scan an AWS instance using one of the available connectors.",
			},
			"aws-ec2-connect": {
				Short: "Scan an AWS instance using EC2 Instance Connect.",
			},
			"aws-ec2-ebs-instance": {
				Short: "Scan an AWS instance using an EBS volume scan. This requires an AWS host.",
				Long: `Scan an AWS instance using an EBS volume scan. This requires that the
scan execute on an instance that is running inside of AWS.`,
			},
			"aws-ec2-ebs-volume": {
				Short: "Scan a specific AWS volume using an EBS volume scan. This requires an AWS host.",
				Long: `Scan a specific AWS volume using an EBS volume scan. This requires that the
scan execute on an instance that is running inside of AWS.`,
			},
			"aws-ec2-ebs-snapshot": {
				Short: "Scan a specific AWS snapshot using an EBS volume scan. This requires an AWS host.",
				Long: `Scan a specific AWS snapshot using an EBS volume scan. This requires that the
scan execute on an instance that is running inside of AWS.`,
			},
			"aws-ec2-ssm": {
				Short: "Scan an AWS instance using the AWS Systems Manager to connect.",
			},
			"azure": {
				Short: "Scan a Microsoft Azure subscription or virtual machine.",
				Long: `Scan a Microsoft Azure subscription or virtual machine. cnspec uses your local Azure
configuration for the account scan. To scan Azure virtual machines, you must
configure your Azure credentials and have SSH access to the virtual machines.`,
			},
			"gcp": {
				Short: "Scan a Google Cloud Platform (GCP) organization, project or folder.",
			},
			"gcp-org": {
				Short: "Scan a Google Cloud Platform (GCP) organization.",
			},
			"gcp-project": {
				Short: "Scan a Google Cloud Platform (GCP) project.",
			},
			"gcp-folder": {
				Short: "Scan a Google Cloud Platform (GCP) folder.",
			},
			"gcp-gcr": {
				Short: "Scan a Google Container Registry (GCR).",
			},
			"gcp-compute-instance": {
				Short: "Scan a Google Cloud Platform (GCP) VM instance.",
			},
			"oci": {
				Short: "Scan a Oracle Cloud Infrastructure (OCI) tenancy.",
			},
			"vsphere": {
				Short: "Scan a VMware vSphere API endpoint.",
			},
			"vsphere-vm": {
				Short: "Scan a VMware vSphere VM.",
			},
			"vcd": {
				Short: "Scan a VMware Virtual Cloud Director organization.",
			},
			"github": {
				Short: "Scan a GitHub organization or repository.",
			},
			"okta": {
				Short: "Scan an Okta organization.",
			},
			"googleworkspace": {
				Short: "Scan a Google Workspace organization.",
			},
			"slack": {
				Short: "Scan a Slack team.",
			},
			"github-org": {
				Short: "Scan a GitHub organization.",
			},
			"github-repo": {
				Short: "Scan a GitHub repository.",
			},
			"github-user": {
				Short: "Scan a GitHub user.",
			},
			"gitlab": {
				Short: "Scan a GitLab group.",
			},
			"ms365": {
				Short: "Scan a Microsoft 365 tenant.",
				Long: `
This command triggers a new policy scan for Microsoft 365:

    $ cnspec scan ms365 --tenant-id {tenant id} --client-id {client id} --client-secret {client secret}

This example connects to Microsoft 365 using the PKCS #12 formatted certificate:

    $ cnspec scan ms365 --tenant-id {tenant id} --client-id {client id} --certificate-path {certificate.pfx} --certificate-secret {certificate secret}
    $ cnspec scan ms365 --tenant-id {tenant id} --client-id {client id} --certificate-path {certificate.pfx} --ask-pass
`,
			},
			"host": {
				Short: "Scan a host endpoint (domain name).",
			},
			"arista": {
				Short: "Scan an Arista endpoint.",
			},
			"filesystem": {
				Short: "Scan a mounted file system target.",
			},
			"opcua": {
				Short: "Scan a OPC UA endpoint.",
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
		cmd.Flags().String("inventory-file", "", "Set the path to the inventory file.")
		cmd.Flags().Bool("inventory-ansible", false, "Set the inventory format to Ansible.")
		cmd.Flags().Bool("inventory-domainlist", false, "Set the inventory format to domain list.")

		// policies & incognito mode
		cmd.Flags().Bool("incognito", false, "Run in incognito mode. Do not report scan results to the Mondoo platform.")
		cmd.Flags().StringSlice("policy", nil, "Lists policies to execute. This requires incognito mode. You can pass multiple policies using --policy POLICY")
		cmd.Flags().StringSliceP("policy-bundle", "f", nil, "Path to local policy bundle file.")
		// flag completion command
		cmd.RegisterFlagCompletionFunc("policy", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return getPoliciesForCompletion(), cobra.ShellCompDirectiveDefault
		})

		// individual asset flags
		cmd.Flags().StringP("password", "p", "", "Password, such as for SSH/WinRM.")
		cmd.Flags().Bool("ask-pass", false, "Ask for connection password.")
		cmd.Flags().StringP("identity-file", "i", "", "Select a file from which to read the identity (private key) for public key authentication.")
		cmd.Flags().String("id-detector", "", "User override for platform ID detection mechanism. Supported: "+strings.Join(providers.AvailablePlatformIdDetector(), ", "))
		cmd.Flags().String("asset-name", "", "User override for the asset name.")
		cmd.Flags().StringToString("props", nil, "Custom values for properties")

		cmd.Flags().String("path", "", "Path to a local file or directory for the connection to use.")
		cmd.Flags().StringToString("option", nil, "Additional connection options. You can pass multiple options using `--option key=value`.")
		cmd.Flags().String("discover", discovery_common.DiscoveryAuto, "Enable the discovery of nested assets. Supported: 'all|auto|instances|host-instances|host-machines|container|container-images|pods|cronjobs|statefulsets|deployments|jobs|replicasets|daemonsets'")
		cmd.Flags().StringToString("discover-filter", nil, "Additional filter for asset discovery.")
		cmd.Flags().StringToString("annotation", nil, "Add an annotation to the asset.") // user-added, editable

		// global asset flags
		cmd.Flags().Bool("insecure", false, "Disable TLS/SSL checks or SSH hostkey config.")
		cmd.Flags().Bool("sudo", false, "Elevate privileges with sudo.")
		cmd.Flags().Int("score-threshold", 0, "If any score falls below the threshold, exit 1.")
		cmd.Flags().Bool("record", false, "Record all backend calls.")
		cmd.Flags().MarkHidden("record")

		// v6 should make detect-cicd and category flag public, default for "detect-cicd" should switch to true
		cmd.Flags().Bool("detect-cicd", true, "Try to detect CI/CD environments and, if successful, set the asset category to 'cicd'.")
		cmd.Flags().String("category", "fleet", "Sets the category for the assets to 'fleet|cicd'.")
		cmd.Flags().MarkHidden("category")

		// output rendering
		cmd.Flags().StringP("output", "o", "compact", "Set output format: "+reporter.AllFormats())
		cmd.Flags().BoolP("json", "j", false, "Set output to JSON (shorthand).")
		cmd.Flags().Bool("share-report", false, "create a web-based private report when cnspec is unauthenticated. Defaults to false.")
		cmd.Flags().MarkHidden("share-report")
		cmd.Flags().Bool("share", false, "create a web-based private reports when cnspec is unauthenticated. Defaults to false.")
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

		// for all assets
		viper.BindPFlag("incognito", cmd.Flags().Lookup("incognito"))
		viper.BindPFlag("insecure", cmd.Flags().Lookup("insecure"))
		viper.BindPFlag("policies", cmd.Flags().Lookup("policy"))
		viper.BindPFlag("sudo.active", cmd.Flags().Lookup("sudo"))

		viper.BindPFlag("score-threshold", cmd.Flags().Lookup("score-threshold"))
		viper.BindPFlag("record", cmd.Flags().Lookup("record"))

		viper.BindPFlag("output", cmd.Flags().Lookup("output"))

		// share-report is deprecated in favor of share
		viper.BindPFlag("share-report", cmd.Flags().Lookup("share-report"))
		viper.BindPFlag("share", cmd.Flags().Lookup("share"))
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
		prof.InitProfiler()

		conf, err := getCobraScanConfig(cmd, args, provider, assetType)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to prepare config")
		}

		err = conf.loadPolicies()
		if err != nil {
			log.Fatal().Err(err).Msg("failed to resolve policies")
		}

		report, err := RunScan(conf)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to run scan")
		}

		logger.DebugDumpJSON("report", report)
		printReports(report, conf, cmd)

		// handle report sharing
		var shareReport bool

		if viper.IsSet("share-report") || viper.IsSet("share") {
			shareReportFlag := viper.GetBool("share-report") || viper.GetBool("share")
			shareReport = shareReportFlag
		}

		otherReportOptionsMsg := ""
		if conf.Output == "compact" {
			otherReportOptionsMsg += "For detailed CLI output, use `--output full`. "
		}

		if conf.IsIncognito && shareReport == false {
			otherReportOptionsMsg += "To create a web-based report with a private URL using Mondoo's reporting service, use `--share`."
		}

		if otherReportOptionsMsg != "" {
			log.Info().Msg(otherReportOptionsMsg)
		}

		// if report sharing was requested, share the report and print the URL
		if conf.IsIncognito && shareReport == true {
			proxy, err := cnquery_config.GetAPIProxy()
			if err != nil {
				log.Error().Err(err).Msg("error getting proxy information")
			} else {
				reportId, err := cnspec_upstream.UploadSharedReport(report, os.Getenv(featureReportAlternateUrlEnv), proxy)
				if err != nil {
					log.Fatal().Err(err).Msg("could not upload report results")
				}
				fmt.Printf("View report at %s\n", reportId.Url)
			}
		}

		// if we had asset errors, we return a non-zero exit code
		// asset errors are only connection issues
		if len(report.Errors) > 0 {
			os.Exit(1)
		}

		if report.GetWorstScore() < uint32(conf.ScoreThreshold) {
			os.Exit(1)
		}
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
	Features  cnquery.Features
	Inventory *v1.Inventory

	// report type, indicates if the service how much data needs to be collected
	ReportType scan.ReportType

	// output format for the rendering
	Output string

	PolicyPaths []string
	PolicyNames []string
	Props       map[string]string
	Bundle      *policy.Bundle

	IsIncognito    bool
	ScoreThreshold int
	DoRecord       bool

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

	props := map[string]string{}
	var err error
	propsFlag := cmd.Flags().Lookup("props")
	if propsFlag != nil {
		props, err = cmd.Flags().GetStringToString("props")
		if err != nil {
			log.Fatal().Err(err).Msg("failed to parse props")
		}
	}

	conf := scanConfig{
		Features:       opts.GetFeatures(),
		IsIncognito:    viper.GetBool("incognito"),
		DoRecord:       viper.GetBool("record"),
		PolicyPaths:    viper.GetStringSlice("policy-bundle"),
		PolicyNames:    viper.GetStringSlice("policies"),
		ScoreThreshold: viper.GetInt("score-threshold"),
		Props:          props,
	}

	// if users want to get more information on available output options,
	// print them before executing the scan
	output, _ := cmd.Flags().GetString("output")
	if output == "help" {
		fmt.Println("Available output formats: " + reporter.AllFormats())
		os.Exit(0)
	}

	// --json takes precedence
	if ok, _ := cmd.Flags().GetBool("json"); ok {
		output = "json"
	}
	conf.Output = output

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
		certAuth, err := upstream.NewServiceAccountRangerPlugin(serviceAccount)
		if err != nil {
			log.Error().Err(err).Msg(errorMessageServiceAccount)
			os.Exit(cnquery_cmd.ConfigurationErrorCode)
		}
		plugins := []ranger.ClientPlugin{certAuth}
		// determine information about the client
		sysInfo, err := sysinfo.GatherSystemInfo()
		if err != nil {
			log.Warn().Err(err).Msg("could not gather client information")
		}
		plugins = append(plugins, defaultRangerPlugins(sysInfo, opts.GetFeatures())...)
		httpClient, err := opts.GetHttpClient()
		if err != nil {
			log.Error().Err(err).Msg("error setting up httpclient")
			os.Exit(cnquery_cmd.ConfigurationErrorCode)

		}
		log.Info().Msg("using service account credentials")
		conf.UpstreamConfig = &resources.UpstreamConfig{
			SpaceMrn:    opts.GetParentMrn(),
			ApiEndpoint: opts.UpstreamApiEndpoint(),
			Plugins:     plugins,
			HttpClient:  httpClient,
		}
	}

	if len(conf.PolicyPaths) > 0 && !conf.IsIncognito {
		conf.IsIncognito = true
	}

	if serviceAccount == nil && !conf.IsIncognito {
		conf.IsIncognito = true
	}

	// print headline when it is not printed to yaml
	if output == "" {
		fmt.Fprintln(os.Stdout, cnspecLogo)
	}

	if conf.DoRecord {
		log.Info().Msg("enable recording of platform calls")
	}

	if opts.ShareReport != nil && (!viper.IsSet("share-report") && !viper.IsSet("share")) {
		flagValue := "false"
		if *opts.ShareReport {
			flagValue = "true"
		}
		cmd.Flags().Set("share", flagValue)
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

		_, err = bundle.Compile(context.Background(), nil)
		if err != nil {
			return errors.Wrap(err, "failed to compile bundle")
		}

		c.Bundle = bundle
		return nil
	}

	return nil
}

func RunScan(config *scanConfig, opts ...scan.ScannerOption) (*policy.ReportCollection, error) {
	scannerOpts := []scan.ScannerOption{}
	scannerOpts = append(scannerOpts, opts...)

	if config.UpstreamConfig != nil {
		scannerOpts = append(scannerOpts, scan.WithUpstream(config.UpstreamConfig.ApiEndpoint, config.UpstreamConfig.SpaceMrn, config.UpstreamConfig.HttpClient), scan.WithPlugins(config.UpstreamConfig.Plugins))
	}

	// show warning to the user of the policy filter container a bundle file name
	for i := range config.PolicyNames {
		entry := config.PolicyNames[i]
		if strings.HasSuffix(entry, ".mql.yaml") || strings.HasSuffix(entry, ".mql.yml") {
			log.Warn().Msg("You're using a bundle file as a policy. Do you mean `--policy-bundle " + entry + "`?")
		}
	}

	scanner := scan.NewLocalScanner(scannerOpts...)
	ctx := cnquery.SetFeatures(context.Background(), config.Features)

	if config.IsIncognito {
		res, err := scanner.RunIncognito(
			ctx,
			&scan.Job{
				DoRecord:      config.DoRecord,
				Inventory:     config.Inventory,
				Bundle:        config.Bundle,
				PolicyFilters: config.PolicyNames,
				Props:         config.Props,
				ReportType:    config.ReportType,
			})
		if err != nil {
			return nil, err
		}
		return res.GetFull(), nil
	}

	res, err := scanner.Run(
		ctx,
		&scan.Job{
			DoRecord:      config.DoRecord,
			Inventory:     config.Inventory,
			Bundle:        config.Bundle,
			PolicyFilters: config.PolicyNames,
			Props:         config.Props,
			ReportType:    config.ReportType,
		})
	if err != nil {
		return nil, err
	}
	return res.GetFull(), nil
}

func printReports(report *policy.ReportCollection, conf *scanConfig, cmd *cobra.Command) {
	// print the output using the specified output format
	r, err := reporter.New(conf.Output)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	r.IsIncognito = conf.IsIncognito
	if err = r.Print(report, os.Stdout); err != nil {
		log.Fatal().Err(err).Msg("failed to print")
	}
}
