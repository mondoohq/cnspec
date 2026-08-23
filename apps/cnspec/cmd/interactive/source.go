// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

	"go.mondoo.com/cnspec/cli/launcher/source"
)

// The launcher's names for the value pickers.
//
// The pickers themselves are cli/launcher/source: what a Source declares about
// itself, the readers behind each one, and the registry they live in. None of
// that knows what a connector is or what a screen looks like, which is the
// point of the move -- a picker that reads ~/.aws is not a fact about a
// terminal UI.
//
// These are aliases rather than a rewrite at every use, for the same reason
// the form engine's aliases exist in form.go: the rename would then have been
// the change, across 423 uses of 77 names in this package, and a diff that size
// hides whatever else is in it. Nothing here adds behaviour. Everything that
// does is on the other side of the import.

// The contract a picker declares about itself.
type (
	Source      = source.Source
	Class       = source.Class
	Cost        = source.Cost
	EmitFunc    = source.EmitFunc
	ExplainFunc = source.ExplainFunc
)

const (
	ClassEnumerated     = source.ClassEnumerated
	ClassAmbient        = source.ClassAmbient
	ClassPostConnection = source.ClassPostConnection
	ClassRequired       = source.ClassRequired

	CostInstant = source.CostInstant
	CostLocal   = source.CostLocal
	CostRemote  = source.CostRemote
)

// The source-id namespace. A curated form names its pickers by id, so every
// one of these is reached for by a curated form or by a test of one.
const (
	srcAWSProfile          = source.AWSProfile
	srcKubeContext         = source.KubeContext
	srcSSHHost             = source.SSHHost
	srcDockerContainer     = source.DockerContainer
	srcDockerImage         = source.DockerImage
	srcGCPProject          = source.GCPProject
	srcGCPProjectAll       = source.GCPProjectAll
	srcGCPZone             = source.GCPZone
	srcAzureSubscription   = source.AzureSubscription
	srcOCIProfile          = source.OCIProfile
	srcAlicloudProfile     = source.AlicloudProfile
	srcSnowflakeConnection = source.SnowflakeConnection
	srcDockerContext       = source.DockerContext
	srcK8sNamespace        = source.K8sNamespace

	srcGitHubToken       = source.GitHubToken
	srcGitLabToken       = source.GitLabToken
	srcSlackToken        = source.SlackToken
	srcCloudflareToken   = source.CloudflareToken
	srcDigitalOceanToken = source.DigitalOceanToken
	srcHetznerToken      = source.HetznerToken
	srcOktaToken         = source.OktaToken

	srcDiscoverK8sNamespaces      = source.DiscoverK8sNamespaces
	srcDiscoverGitHubRepos        = source.DiscoverGitHubRepos
	srcDiscoverGitLabProjects     = source.DiscoverGitLabProjects
	srcDiscoverGitLabGroups       = source.DiscoverGitLabGroups
	srcDiscoverAzureSubscriptions = source.DiscoverAzureSubscriptions
	srcDiscoverGCPProjects        = source.DiscoverGCPProjects
	srcDiscoverAWSAccounts        = source.DiscoverAWSAccounts
	srcDiscoverOCITenancy         = source.DiscoverOCITenancy
	srcDiscoverAlicloudAccounts   = source.DiscoverAlicloudAccounts
	srcDiscoverDigitalOceanDBs    = source.DiscoverDigitalOceanDBs
	srcDiscoverNeonProjects       = source.DiscoverNeonProjects
	srcDiscoverNetlifySites       = source.DiscoverNetlifySites
	srcDiscoverVercelProjects     = source.DiscoverVercelProjects
	srcDiscoverAtlasProjects      = source.DiscoverAtlasProjects
	srcDiscoverHCPProjects        = source.DiscoverHCPProjects
	srcDiscoverClaudeWorkspaces   = source.DiscoverClaudeWorkspaces
	srcDiscoverMSSQLDatabases     = source.DiscoverMSSQLDatabases
	srcDiscoverMySQLDatabases     = source.DiscoverMySQLDatabases
	srcDiscoverPostgresDatabases  = source.DiscoverPostgresDatabases
	srcDiscoverSnowflakeDatabases = source.DiscoverSnowflakeDatabases
)

// Names a curated form or a launch reaches for.
const (
	specialCredentialState = source.SpecialCredentialState
	ambientWhyEnv          = source.AmbientWhyEnv
	alicloudProfileEnv     = source.AlicloudProfileEnv
	dockerContextEnv       = source.DockerContextEnv
	dockerHostEnv          = source.DockerHostEnv
	dockerDefaultContext   = source.DockerDefaultContext
)

// registry is the live source registry. The tests that install a source and
// take it away again reach for it by this name; see source.Registry.
var registry = source.Registry()

var (
	register             = source.Register
	sourceByID           = source.ByID
	activityFor          = source.ActivityFor
	deferredSource       = source.Deferred
	preferredValue       = source.PreferredValue
	sourceEmit           = source.Emit
	sourceKey            = source.Key
	discoverSourceID     = source.DiscoverSourceID
	refreshAmbient       = source.RefreshAmbient
	sortedUnique         = source.SortedUnique
	gcpActiveProject     = source.GCPActiveProject
	neutralisedBy        = source.NeutralisedBy
	dockerContextEnvFrom = source.DockerContextEnvFrom
)

// applyAmbient gives a connector's ambient credentials their widgets. The
// pickers need the connector's name and nothing else about it, so this is
// where the catalog stops.
func applyAmbient(f *form, c Connector) { source.ApplyAmbient(f, c.Name) }

// sourceValuesMsg carries a picker's values back into the model.
//
// The message stays on this side of the boundary because it is model plumbing:
// bubbletea delivers it, Update reads it, and the fetch it reports on has no
// opinion about either. What it reports is source.Result, field for field.
type sourceValuesMsg struct {
	source string
	key    string
	values []string
	// err explains an empty list. A picker that cannot reach its cluster has
	// to say so rather than look like a cluster with nothing in it.
	err error
	// cancelled marks the answer to a question nobody is waiting for any more.
	// A killed child reports "signal: killed", which is true and useless: the
	// user closed the picker, and reporting that their gcloud died is blaming
	// them for their own esc key.
	cancelled bool
}

// loadSourceCmd fetches a picker's values off the UI goroutine. Docker shells
// out to a daemon that can be slow or wedged, so no source is ever resolved
// inline in Update.
//
// The key is computed once, before the command runs, and carried into every
// message it can produce. The model registers a loading key at that same point
// and deletes it when the message lands, so a key that changed in between --
// registered scoped and deleted unscoped -- would leave a spinner that can
// never stop.
func loadSourceCmd(ctx context.Context, id string, params []string) tea.Cmd {
	key := sourceKey(id, params)
	return func() tea.Msg {
		r := source.Load(ctx, id, params)
		return sourceValuesMsg{
			source:    id,
			key:       key,
			values:    r.Values,
			err:       r.Err,
			cancelled: r.Cancelled,
		}
	}
}

func init() {
	// The one thing cli/launcher/source needs from this package and cannot
	// import, because this package imports it: pointing a namespace list at the
	// cluster the user picked needs a kubeconfig copy written first, which is
	// kubeconfig.go's job.
	//
	// There used to be a second, a loop registering the environment variable
	// each ambient credential declared for its own paste box. Nothing needs it
	// now: a pasted value travels the same way a typed one does, by inventory,
	// and the variables the ambient rows name are only what the *provider*
	// reads when the launcher supplies nothing.
	source.KubeEnvApply = kubeEnvForContext
}
