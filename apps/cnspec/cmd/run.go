package cmd

import (
	zlog "github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	cnquery_app "go.mondoo.com/cnquery/apps/cnquery/cmd"
	"go.mondoo.com/cnquery/apps/cnquery/cmd/builder"
	"go.mondoo.com/cnquery/apps/cnquery/cmd/builder/common"
	discovery "go.mondoo.com/cnquery/motor/discovery/common"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnspec/internal/plugin"
)

func init() {
	rootCmd.AddCommand(execCmd)
}

var execCmd = builder.NewProviderCommand(builder.CommandOpts{
	Use:   "run",
	Short: "Run an MQL query.",
	Long:  `Run an MQL query on the CLI and displays its results.`,
	Docs: common.CommandsDocs{
		Entries: map[string]common.CommandDocsEntry{
			"local": {
				Short: "Run an MQL query against your local system.",
			},
			"mock": {
				Short: "Run an MQL query against a mock target (a simulated asset).",
			},
			"vagrant": {
				Short: "Run an MQL query against a Vagrant host.",
			},
			"terraform": {
				Short: "Run an MQL query against Terraform HCL (files.tf and directories), plan files (json), and state files (json).",
			},
			"ssh": {
				Short: "Run an MQL query against an SSH target.",
			},
			"winrm": {
				Short: "Run an MQL query against a WinRM target.",
			},
			"container": {
				Short: "Run an MQL query against a container, image, or registry.",
			},
			"container-image": {
				Short: "Run an MQL query against a container image.",
			},
			"container-tar": {
				Short: "Run an MQL query against an OCI container image from a tar file.",
			},
			"container-registry": {
				Short: "Run an MQL query against a container registry.",
			},
			"docker": {
				Short: "Run an MQL query against a Docker container or image.",
			},
			"docker-container": {
				Short: "Run an MQL query against a Docker container.",
			},
			"docker-image": {
				Short: "Run an MQL query against a Docker image.",
			},
			"kubernetes": {
				Short: "Run an MQL query against a Kubernetes cluster or local manifest file(s).",
			},
			"aws": {
				Short: "Run an MQL query against an AWS account or instance.",
			},
			"aws-ec2": {
				Short: "Run an MQL query against an AWS instance using one of the available connectors.",
			},
			"aws-ec2-connect": {
				Short: "Run an MQL query against an AWS instance using EC2 Instance Connect.",
			},
			"aws-ec2-ebs-instance": {
				Short: "Run an MQL query against an AWS instance using an EBS volume scan. This requires an AWS host.",
			},
			"aws-ec2-ebs-volume": {
				Short: "Run an MQL query against a specific AWS volume using an EBS volume scan. This requires an AWS host.",
			},
			"aws-ec2-ebs-snapshot": {
				Short: "Run an MQL query against a specific AWS snapshot using an EBS volume scan. This requires an AWS host.",
			},
			"aws-ec2-ssm": {
				Short: "Run an MQL query against an AWS instance using the AWS Systems Manager to connect.",
			},
			"azure": {
				Short: "Run an MQL query against a Microsoft Azure subscription or virtual machine.",
			},
			"gcp": {
				Short: "Run an MQL query against a Google Cloud Platform (GCP) organization, project, or folder.",
			},
			"gcp-org": {
				Short: "Run an MQL query against a Google Cloud Platform (GCP) organization.",
			},
			"gcp-project": {
				Short: "Run an MQL query against a Google Cloud Platform (GCP) project.",
			},
			"gcp-folder": {
				Short: "Run an MQL query against a Google Cloud Platform (GCP) folder.",
			},
			"gcp-gcr": {
				Short: "Run an MQL query against a Google Container Registry (GCR).",
			},
			"gcp-compute-instance": {
				Short: "Run an MQL query against a Google Cloud Platform (GCP) VM instance.",
			},
			"oci": {
				Short: "Run an MQL query against an Oracle Cloud Infrastructure (OCI) tenancy.",
			},
			"vsphere": {
				Short: "Run an MQL query against a VMware vSphere API endpoint.",
			},
			"vsphere-vm": {
				Short: "Run an MQL query against a VMware vSphere VM.",
			},
			"vcd": {
				Short: "Run an MQL query against a VMware Virtual Cloud Director organization.",
			},
			"github": {
				Short: "Run an MQL query against a GitHub organization or repository.",
			},
			"okta": {
				Short: "Run an MQL query against an Okta organization.",
			},
			"googleworkspace": {
				Short: "Run an MQL query against a Google Workspace organization.",
			},
			"slack": {
				Short: "Run an MQL query against a Slack team.",
			},
			"github-org": {
				Short: "Run an MQL query against a GitHub organization.",
			},
			"github-repo": {
				Short: "Run an MQL query against a GitHub repository.",
			},
			"github-user": {
				Short: "Run an MQL query against a GitHub user.",
			},
			"gitlab": {
				Short: "Run an MQL query against a GitLab group.",
			},
			"ms365": {
				Short: "Run an MQL query against a Microsoft 365 tenant.",
			},
			"host": {
				Short: "Run an MQL query against a host endpoint (domain name).",
			},
			"arista": {
				Short: "Run an MQL query against an Arista endpoint.",
			},
			"filesystem": {
				Short: "Run an MQL query against a mounted file system target.",
			},
			"opcua": {
				Short: "Run an MQL query against an OPC UA endpoint.",
			},
		},
	},
	CommonFlags: func(cmd *cobra.Command) {
		cmd.Flags().Bool("parse", false, "Parse the query and return the logical structure.")
		cmd.Flags().Bool("ast", false, "Parse the query and return the abstract syntax tree (AST).")
		cmd.Flags().BoolP("json", "j", false, "Run the query and return the object in a JSON structure.")
		cmd.Flags().String("query", "", "MQL query to execute.")
		cmd.Flags().MarkHidden("query")
		cmd.Flags().StringP("command", "c", "", "MQL query to execute.")

		cmd.Flags().StringP("password", "p", "", "Connection password, such as for SSH/WinRM.")
		cmd.Flags().Bool("ask-pass", false, "Prompt for connection password.")
		cmd.Flags().StringP("identity-file", "i", "", "Select a file from which to read the identity (private key) for public key authentication.")
		cmd.Flags().Bool("insecure", false, "Disable TLS/SSL checks or SSH hostkey config.")
		cmd.Flags().Bool("sudo", false, "Elevate privileges with sudo.")
		cmd.Flags().String("platform-id", "", "Select a specific target asset by providing its platform ID.")
		cmd.Flags().Bool("instances", false, "Also scan instances. This only applies to API targets like AWS, Azure, or GCP.")
		cmd.Flags().Bool("host-machines", false, "Also scan host machines like ESXi servers.")

		cmd.Flags().Bool("record", false, "Record provider calls. This only works for operating system providers.")
		cmd.Flags().MarkHidden("record")

		cmd.Flags().String("record-file", "", "File path for the recorded provider calls. This only works for operating system providers.")
		cmd.Flags().MarkHidden("record-file")

		cmd.Flags().String("path", "", "Path to a local file or directory for the connection to use.")
		cmd.Flags().StringToString("option", nil, "Additional connection options. You can pass in multiple options using `--option key=value`")
		cmd.Flags().String("discover", discovery.DiscoveryAuto, "Enable the discovery of nested assets. Supported: 'all|auto|instances|host-instances|host-machines|container|container-images|pods|cronjobs|statefulsets|deployments|jobs|replicasets|daemonsets'")
		cmd.Flags().StringToString("discover-filter", nil, "Additional filter for asset discovery.")
	},
	CommonPreRun: func(cmd *cobra.Command, args []string) {
		// for all assets
		viper.BindPFlag("insecure", cmd.Flags().Lookup("insecure"))
		viper.BindPFlag("sudo.active", cmd.Flags().Lookup("sudo"))

		viper.BindPFlag("output", cmd.Flags().Lookup("output"))

		viper.BindPFlag("vault.name", cmd.Flags().Lookup("vault"))
		viper.BindPFlag("platform-id", cmd.Flags().Lookup("platform-id"))
		viper.BindPFlag("query", cmd.Flags().Lookup("query"))
		viper.BindPFlag("command", cmd.Flags().Lookup("command"))

		viper.BindPFlag("record", cmd.Flags().Lookup("record"))
		viper.BindPFlag("record-file", cmd.Flags().Lookup("record-file"))
	},
	Run: func(cmd *cobra.Command, args []string, provider providers.ProviderType, assetType builder.AssetType) {
		conf, err := cnquery_app.GetCobraRunConfig(cmd, args, provider, assetType)
		if err != nil {
			zlog.Fatal().Err(err).Msg("failed to prepare config")
		}

		err = plugin.RunQuery(conf)
		if err != nil {
			zlog.Fatal().Err(err).Msg("failed to run query")
		}
	},
})
