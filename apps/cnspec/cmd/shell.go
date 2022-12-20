package cmd

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.mondoo.com/cnquery"
	cnquery_app "go.mondoo.com/cnquery/apps/cnquery/cmd"
	"go.mondoo.com/cnquery/apps/cnquery/cmd/builder"
	"go.mondoo.com/cnquery/motor/discovery/common"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnspec"
)

func init() {
	rootCmd.AddCommand(shellCmd)
}

var shellCmd = builder.NewProviderCommand(builder.CommandOpts{
	Use:   "shell",
	Short: "Interactive query shell for MQL",
	Long:  `Run the interactive shell so you can explore MQL queries.`,
	CommonFlags: func(cmd *cobra.Command) {
		cmd.Flags().StringP("password", "p", "", "Set the connection password, such as for SSH/WinRM.")
		cmd.Flags().Bool("ask-pass", false, "Prompt for connection password")

		cmd.Flags().StringP("command", "c", "", "MQL query to execute in the shell")
		cmd.Flags().StringP("identity-file", "i", "", "Select a file from which to read the identity (private key) for public key authentication.")
		cmd.Flags().Bool("insecure", false, "Disable TLS/SSL checks or SSH hostkey config.")
		cmd.Flags().Bool("sudo", false, "Elevate privileges with sudo.")
		cmd.Flags().String("platform-id", "", "Select an specific target asset by providing its platform ID.")
		cmd.Flags().Bool("instances", false, "Also scan instances. This only applies to API targets like AWS, Azure or GCP.")
		cmd.Flags().Bool("host-machines", false, "Also scan host machines like ESXi server.")
		cmd.Flags().Bool("record", false, "Record all backend calls.")
		cmd.Flags().MarkHidden("record")

		cmd.Flags().String("path", "", "Path to a local file or directory for the connection to use.")
		cmd.Flags().StringToString("option", nil, "Additional connection options. You can pass multiple options using `--option key=value`.")
		cmd.Flags().String("discover", common.DiscoveryAuto, "Enable the discovery of nested assets. Supported: 'all|auto|instances|host-instances|host-machines|container|container-images|pods|cronjobs|statefulsets|deployments|jobs|replicasets|daemonsets'")
		cmd.Flags().StringToString("discover-filter", nil, "Additional filter for asset discovery.")
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
				Short: "Connect to your local system.",
			},
			"mock": {
				Short: "Connect to mock target (a simulated asset)",
				Long: `Connect to a mock target. This connects to a simulated asset. We recorded the asset's data beforehand.
Provide the recording with mock data as an argument:

    cnspec shell container ubuntu:latest --record
    cnspec shell mock recording-20220519173543.toml
`,
			},
			"vagrant": {
				Short: "Connect to a Vagrant host.",
			},
			"terraform": {
				Short: "Connect to all Terraform files in a path (.tf files).",
			},
			"ssh": {
				Short: "Connect to an SSH target.",
			},
			"winrm": {
				Short: "Connect to a WinRM target.",
			},
			"container": {
				Short: "Connect to a container, image, or registry.",
				Long: `Connect to a container, container image, or container registry. By default cnspec tries to auto-detect the container or image from the provided ID, even
				if it's not the full ID:

    cnspec shell container b62b276baab6
    cnspec shell container b62
    cnspec shell container ubuntu:latest

You can also explicitly connect to an image or a container registry:

    cnspec shell container image ubuntu:20.04
    cnspec shell container registry harbor.lunalectric.com/project/repository
`,
			},
			"container-image": {
				Short: "Connect to a container image.",
			},
			"container-registry": {
				Short: "Connect to a container registry.",
				Long: `Connect to a container registry. Supports more parameters for different registries:

    cnspec shell container registry harbor.lunalectric.com/project/repository
    cnspec shell container registry yourname.azurecr.io
    cnspec shell container registry 123456789.dkr.ecr.us-east-1.amazonaws.com/repository
`,
			},
			"docker": {
				Short: "Connect to a Docker container or image.",
				Long: `Connect to a Docker container or image by automatically detecting the provided ID.
You can also specify a subcommand to narrow the scan to containers or images.

    cnspec shell docker b62b276baab6

    cnspec shell docker container b62b
    cnspec shell docker image ubuntu:latest
`,
			},
			"docker-container": {
				Short: "Connect to a Docker container",
				Long: `Connect to a Docker container. You can specify the container ID (such as b62b276baab6)
				or container name (such as elated_poincare).`,
			},
			"docker-image": {
				Short: "Connect to a Docker image.",
				Long: `Connect to a Docker image. You can specify the image ID (such as b6f507652425)
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
				Long:  `Connect to an AWS instance using an EBS volume scan. using an EBS volume scan. This requires that the scan execute on an instance that is running inside of AWS.`,
			},
			"aws-ec2-ebs-volume": {
				Short: "Connect to a specific AWS volume using an EBS volume scan. This requires an AWS host.",
				Long:  `Connect to a specific AWS volume using the EBS volume scan capability. This requires that the scan execute on an instance that is running inside of AWS.`,
			},
			"aws-ec2-ebs-snapshot": {
				Short: "Connect to a specific AWS snapshot using an EBS volume scan. This requires an AWS host.",
				Long:  `Connect a specific AWS snapshot using the EBS volume scan capability. This requires that the scan execute on an instance that is running inside of AWS.`,
			},
			"aws-ec2-ssm": {
				Short: "Connect to an AWS instance using the AWS Systems Manager.",
			},
			"azure": {
				Short: "Connect to a Microsoft Azure subscription or virtual machines.",
				Long: `Connect to a Microsoft Azure subscriptions or virtual machines. cnspec uses your local Azure
				configuration for the account scan. To connect to Azure virtual machines, you must
				configure your Azure credentials and have SSH access to the virtual machines.`,
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
			"vcd": {
				Short: "Connect to a VMware Virtual Cloud Director organization.",
			},
			"github": {
				Short: "Connect to a GitHub organization or repository.",
			},
			"okta": {
				Short: "Connect to an Okta organization.",
			},
			"googleworkspace": {
				Short: "Connect to a Google Workspace organization.",
			},
			"slack": {
				Short: "Connect to a Slack team.",
			},
			"github-org": {
				Short: "Connect to a GitHub organization.",
			},
			"github-repo": {
				Short: "Connect to a GitHub repository.",
			},
			"gitlab": {
				Short: "Connect to a GitLab group.",
			},
			"ms365": {
				Short: "Connect to a Microsoft 365 tenant.",
				Long: `
This command opens a shell to a Microsoft 365 tenant:

    $ cnspec shell ms365 --tenant-id {tenant id} --client-id {client id} --client-secret {client secret}

This example connects to Microsoft 365 using the PKCS #12 formatted certificate:

    $ cnspec shell ms365 --tenant-id {tenant id} --client-id {client id} --certificate-path {certificate.pfx} --certificate-secret {certificate secret}
    $ cnspec shell ms365 --tenant-id {tenant id} --client-id {client id} --certificate-path {certificate.pfx} --ask-pass
`,
			},
			"host": {
				Short: "Connect to a host endpoint.",
			},
			"arista": {
				Short: "Connect to an Arista endpoint.",
			},
		},
	},
	Run: func(cmd *cobra.Command, args []string, provider providers.ProviderType, assetType builder.AssetType) {
		conf, err := cnquery_app.GetCobraShellConfig(cmd, args, provider, assetType)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to prepare config")
		}

		conf.WelcomeMessage = cnspecLogo

		// FIXME: workaround for `mondoo.version`, remove once we call the plugin
		if cnquery.Version == "" {
			cnquery.Version = cnspec.Version
		}

		err = cnquery_app.StartShell(conf)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to run query")
		}
	},
})
