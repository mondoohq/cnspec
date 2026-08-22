// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package source

// The whole source-ID namespace, in one place.
//
// A source id is a string, so nothing technically requires it to be declared
// before it is used. What requires it is the order the launcher gets built in:
// a curated form names its pickers by id, and the form files and the source
// files are written by different people at the same time. Declaring the names
// here first means a form can reference a picker that does not exist yet and
// still compile, and means two source files cannot quietly invent two spellings
// of the same thing.
//
// The rule for adding one: put the constant here, Register the Source in the
// file that owns its class, and name it from a form. A constant with no
// registration is caught by TestEverySourceNamedByASpecExists the moment a form
// reaches for it; an id spelled inline rather than declared here is caught by
// nothing, which is why this file exists.

// Enumerated: a local file lists the candidates.
const (
	AWSProfile      = "aws.profile"
	KubeContext     = "k8s.context"
	SSHHost         = "ssh.host"
	DockerContainer = "docker.container"
	DockerImage     = "docker.image"
	GCPProject      = "gcp.project"
	GCPProjectAll   = "gcp.project.all"
	GCPZone         = "gcp.zone"

	// AzureSubscription reads ~/.azure/azureProfile.json, which the Azure
	// CLI writes with a UTF-8 BOM.
	AzureSubscription = "azure.subscription"
	// OCIProfile reads ~/.oci/config, whose conventional first section is
	// the literal DEFAULT rather than the lowercase "default" the AWS
	// convention would suggest.
	OCIProfile = "oci.profile"
	// AlicloudProfile reads ~/.alibabacloud/credentials. The connector
	// declares no --profile, so the chosen value travels in the environment.
	AlicloudProfile = "alicloud.profile"
	// SnowflakeConnection reads ~/.snowflake/connections.toml. No flag
	// takes a connection name either.
	SnowflakeConnection = "snowflake.connection"
	// DockerContext reads ~/.docker/config.json and the context store
	// beside it. docker, container and local all lack a --context flag.
	DockerContext = "docker.context"
)

// K8sNamespace is post-connection but predates cnspec's own discovery being
// wired into the launcher: it asks kubectl directly, against a kubeconfig copy
// pointed at the chosen cluster. It sits here rather than with the discovery
// ids below because it is a different mechanism, not a different target.
const K8sNamespace = "k8s.namespace"

// Ambient: one credential from the environment. There is nothing to enumerate,
// so what these answer is "is it present, and where did it come from".
const (
	GitHubToken       = "github.token"
	GitLabToken       = "gitlab.token"
	SlackToken        = "slack.token"
	CloudflareToken   = "cloudflare.token"
	DigitalOceanToken = "digitalocean.token"
	HetznerToken      = "hetzner.token"
	// OktaToken covers okta's three-level precedence: --token, then
	// OKTA_API_TOKEN, then OKTA_TOKEN.
	OktaToken = "okta.token"
)

// Post-connection: what is inside a target, asked through cnspec's own
// discovery once a credential is in hand. Always CostRemote.
//
// The id names the connector and the discovery target together, because a
// connector has several and they feed different flags: github's repos go to
// --repos while its organization goes to a positional. DiscoverSourceID builds
// the same string for a pair not named here, so a source file can Register one
// without editing this one.
const (
	DiscoverK8sNamespaces      = "discover.k8s.namespaces"
	DiscoverGitHubRepos        = "discover.github.repos"
	DiscoverGitLabProjects     = "discover.gitlab.projects"
	DiscoverGitLabGroups       = "discover.gitlab.groups"
	DiscoverAzureSubscriptions = "discover.azure.subscriptions"
	DiscoverGCPProjects        = "discover.gcp.projects"
	DiscoverAWSAccounts        = "discover.aws.accounts"
	DiscoverOCITenancy         = "discover.oci.tenancy"
	DiscoverAlicloudAccounts   = "discover.alicloud.accounts"
	DiscoverDigitalOceanDBs    = "discover.digitalocean.databases"
	DiscoverNeonProjects       = "discover.neon.projects"
	DiscoverNetlifySites       = "discover.netlify.sites"
	DiscoverVercelProjects     = "discover.vercel.projects"
	DiscoverAtlasProjects      = "discover.mongodbatlas.projects"
	DiscoverHCPProjects        = "discover.hcp.projects"
	DiscoverClaudeWorkspaces   = "discover.claude.workspaces"
	DiscoverMSSQLDatabases     = "discover.mssql.databases"
	DiscoverMySQLDatabases     = "discover.mysqldb.databases"
	DiscoverPostgresDatabases  = "discover.postgresdb.databases"
	DiscoverSnowflakeDatabases = "discover.snowflake.databases"
)

// DiscoverSourceID names the source that asks cnspec's discovery for one
// target inside one connector. The constants above are the pairs the launcher
// already knows it wants; this is how a source file names a pair it does not
// yet have a constant for, without editing this file to get it.
func DiscoverSourceID(connector, target string) string {
	return "discover." + connector + "." + target
}

// SpecialDockerContext is the identity of the launcher-owned field holding a
// docker context.
//
// It is a field name rather than a source id, and it is here because two
// things have to spell it the same way: the form spec that creates the field,
// and the Needs of the two sources that read it. A literal in both places is a
// typo waiting to be silent -- the identity simply never matches, the
// enumeration runs against the default daemon, and the list looks right.
//
// docker, container and local declare no --context, so a chosen one travels in
// DOCKER_CONTEXT; see DockerContextEnvFrom.
const SpecialDockerContext = "docker-context"
