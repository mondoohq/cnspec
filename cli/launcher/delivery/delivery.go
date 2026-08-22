// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package delivery

import (
	tuiform "go.mondoo.com/cnspec/cli/tui/form"
)

// Package delivery is how a credential gets from the launcher's form to the
// scan.
//
// The launcher runs commands by re-executing cnspec, so a secret on that
// command line is world-readable through `ps auxww`. Everything here exists to
// avoid that, and there is now exactly one way it does: the form is handed to
// the connector's own ParseCLI, and the inventory.Asset that comes back is
// written as an inventory file the child reads. The secret reaches the provider
// in a protobuf field over the plugin's gRPC connection and reaches the child
// through the OS keychain, referenced by id. See parsecli.go.
//
// # What this replaced, and why one route rather than four
//
// There used to be four: a prompt route that made the child ask, an environment
// route over a registry of provider-native variable names, an inventory route
// gated on a hand-written list of connectors somebody had checked, and a
// refusal for everything else. Each of those was a claim about a provider that
// only a person reading that provider's source could make, and the claims went
// stale silently -- a variable a provider stops reading, or an option key
// guessed rather than read, drops the credential and surfaces as a connection
// error nowhere near its cause.
//
// Asking the provider makes all four unnecessary, because the mapping is no
// longer ours to get right. What is left here is the part that is genuinely the
// launcher's: whether a form has a credential at all, and the environment a
// chosen *target* travels in when the connector declares no flag for it.

// Kind is the route a form's answers travel.
type Kind int

const (
	// Plain: no credential in the form, so the ordinary command line. This is
	// what the user sees in the preview and can copy into a shell, and it is
	// kept for exactly that reason: a command with nothing to protect reads
	// better as a command than as a generated file.
	Plain Kind = iota
	// Inventory: the form holds a credential, so the provider's own reading of
	// it is written as an inventory file and the child is pointed at that.
	Inventory
)

// RouteFor decides how the form's answers travel.
//
// It takes only the form because that is now all it depends on, and the
// shrunken signature is the point: a route that depended on the connector was a
// route that could be wrong about the connector. Whether the child needs a file
// is decided by whether the user typed a secret; what goes in the file is
// decided by the provider.
//
// Note that a --ask-* toggle is an ordinary bool field on the form and stays on
// the Plain route. That is a deliberate reversal. The prompt route used to fire
// whenever a connector declared a working --ask-* flag and the user had typed
// exactly one secret -- so a password typed into the ssh form was dropped, the
// flag was added in its place, and the child asked for the password again.
// Verified: `cnspec scan ssh chris@10.0.0.5 --ask-pass` was the whole command.
// A user who wants to be prompted can still tick the box; what they can no
// longer do is have their typed credential silently replaced by a prompt.
func RouteFor(f tuiform.Form) Kind {
	if len(f.Secrets()) == 0 {
		return Plain
	}
	return Inventory
}

// EnvSpec declares that one field's value reaches the child process through its
// environment rather than through the command line, in the case where naming a
// variable is not enough because something has to be built first.
//
// FormSpec.Env covers the ordinary case -- ALIBABA_CLOUD_PROFILE=<the profile>
// -- in one line of the spec, and needs nothing here. This exists for the case
// k8s taught: the chosen cluster only means anything once a kubeconfig has been
// written whose current-context is that cluster, so the contribution is a
// function that returns both the variable and the cleanup for the file it made.
//
// This is about targets, not credentials, which is why it survived the
// collapse above: a connector that declares no flag for the thing the user
// picked has nothing for ParseCLI to be asked about.
type EnvSpec struct {
	// Connector is the connector this applies to.
	Connector string
	// Field is the field identity whose value travels -- "f:context", "p:1" --
	// or a bare flag name as shorthand. See tuiform.MatchesIdentity.
	Field string
	// Apply turns the chosen value into environment entries, and returns the
	// cleanup for anything it had to write. It is called only for a field that
	// is actually set.
	Apply func(value string) (env []string, cleanup func(), err error)
}

// EnvSpecs holds the declared environment contributors, keyed by connector.
var EnvSpecs = map[string][]EnvSpec{}

// RegisterEnv declares an environment contributor, and is called from the init
// of whichever file owns the connector.
func RegisterEnv(specs ...EnvSpec) {
	for _, s := range specs {
		EnvSpecs[s.Connector] = append(EnvSpecs[s.Connector], s)
	}
}

// EnvSpecsFor returns the contributors declared for a connector.
func EnvSpecsFor(connector string) []EnvSpec {
	return EnvSpecs[connector]
}
