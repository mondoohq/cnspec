// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package forms

import (
	"go.mondoo.com/cnspec/cli/launcher/source"
	tuiform "go.mondoo.com/cnspec/cli/tui/form"
)

// This package's names for the value pickers, and for the two things a spec
// file does besides register a spec.
//
// The pickers themselves are cli/launcher/source: what a Source declares about
// itself, the readers behind each one, and the registry they live in. None of
// that knows what a connector is or what a screen looks like, and nothing here
// adds behaviour -- these are the identifiers the spec files below spell, kept
// as aliases so that moving the specs into this package was a move rather than
// a rewrite of three thousand curated lines.
//
// Only the ids a spec file actually names are listed. The launcher aliases the
// full namespace for its own tests; a picker nothing here attaches has no
// business being spelled here.

type (
	// form is the engine's form, which an ambient credential's Compose func
	// fills in. Nothing in this package builds one.
	form = tuiform.Form
	// envLookup is how the ambient credentials read the environment. Tests
	// substitute it rather than exporting a developer's own tokens into an
	// assertion.
	envLookup = source.EnvLookup
	// ambientCredential declares a connector whose whole credential is one
	// token from one variable.
	ambientCredential = source.AmbientCredential
)

// The source-id namespace, for the pickers a curated form attaches.
const (
	srcAWSProfile          = source.AWSProfile
	srcKubeContext         = source.KubeContext
	srcSSHHost             = source.SSHHost
	srcDockerContainer     = source.DockerContainer
	srcDockerImage         = source.DockerImage
	srcGCPProject          = source.GCPProject
	srcGCPProjectAll       = source.GCPProjectAll
	srcGCPZone             = source.GCPZone
	srcOCIProfile          = source.OCIProfile
	srcAlicloudProfile     = source.AlicloudProfile
	srcSnowflakeConnection = source.SnowflakeConnection
	srcDockerContext       = source.DockerContext
	srcK8sNamespace        = source.K8sNamespace

	srcDiscoverGitLabProjects   = source.DiscoverGitLabProjects
	srcDiscoverGitLabGroups     = source.DiscoverGitLabGroups
	srcDiscoverVercelProjects   = source.DiscoverVercelProjects
	srcDiscoverAtlasProjects    = source.DiscoverAtlasProjects
	srcDiscoverClaudeWorkspaces = source.DiscoverClaudeWorkspaces
)

// ambientWhyEnv is the note put on a field prefilled from a variable, so the
// screen says where a value it did not ask for came from.
const ambientWhyEnv = source.AmbientWhyEnv

var (
	// registerAmbient declares a connector whose credential is ambient. It is
	// the source registry's, not this package's: a spec file calls it beside
	// the spec it belongs to, which is the only reason it is spelled here.
	registerAmbient = source.RegisterAmbient
	// envValue reads one variable through the lookup a caller supplied.
	envValue = source.EnvValue
)

// discoverSourceID names a picker that runs cnspec's own discovery for a
// connector and a target. A spec calls it rather than spelling a src id when
// the target it wants has no registered id of its own.
var discoverSourceID = source.DiscoverSourceID
