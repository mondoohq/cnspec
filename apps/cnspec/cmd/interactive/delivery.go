// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	"time"

	deliverypkg "go.mondoo.com/cnspec/cli/launcher/delivery"
	"go.mondoo.com/mql/llx"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/providers-sdk/v1/vault"
)

// How a credential gets from the form to the scan now lives in
// cli/launcher/delivery, and this file is the launcher's names for it.
//
// The rule it enforces has not moved: the launcher runs commands by
// re-executing cnspec, so a secret on that command line is world-readable
// through `ps auxww`. A credential goes to the OS keychain and is referenced by
// id; where that fails it goes into a 0600 inventory in a private directory,
// removed on every exit this process can observe.
//
// What has moved is who decides what the inventory looks like. It used to be a
// table here, one line per connector, each line a claim about a provider that
// only somebody reading that provider's source could make. Now the connector's
// own ParseCLI is asked, over the same gRPC call `cnspec scan` makes, and the
// asset it answers with is the inventory. Nothing in this package has an
// opinion about which option key a flag lands under any more, which is why
// there is no registry left to keep in step with the providers.
//
// The aliases are aliases rather than a rewrite at every use for the reason
// cli/tui/form's were: the rename would otherwise have been the change, across
// every file that launches, curates a connector, or tests one. What the
// boundary is for survives them -- the delivery package takes a connector as a
// name and a flag list, so nothing here can hand it a screen, and nothing there
// can reach back for one.

type (
	// delivery is the route a form's answers travel.
	delivery = deliverypkg.Kind
	// EnvSpec declares that one field's value reaches the child through the
	// environment, in the case where naming a variable is not enough because
	// something has to be built first.
	EnvSpec = deliverypkg.EnvSpec
	// savedEntry is one credential the launcher put in the keychain.
	savedEntry = deliverypkg.SavedEntry
)

const (
	deliverPlain     = deliverypkg.Plain
	deliverInventory = deliverypkg.Inventory

	// keychainTimeout is how long the launcher waits for the OS keychain.
	keychainTimeout = deliverypkg.KeychainTimeout
)

// The registry itself, not a copy of it: registerEnv is called from the init of
// the files that curate a connector, and the tests read back what those inits
// declared.
var envSpecs = deliverypkg.EnvSpecs

// envSpecsFor returns the environment contributors declared for a connector.
func envSpecsFor(connector string) []EnvSpec {
	return deliverypkg.EnvSpecsFor(connector)
}

// deliveryFor decides whether the form's answers need a file to travel in.
func deliveryFor(f form) delivery {
	return deliverypkg.RouteFor(f)
}

// parseCLI asks a connector's provider what a filled-in form means. It is a
// variable because the alternative is a plugin subprocess per test case: the
// launcher's own tests care what it does with the answer, not what a provider
// gives. The round trip against real providers is asserted separately, over
// whatever is installed; see parsecli_test.go.
var parseCLI = func(provider, connector string, args []string, flags map[string]*llx.Primitive) (*inventory.Asset, error) {
	return deliverypkg.Parser.ParseCLI(provider, connector, args, flags)
}

// trackTemp registers a cleanup for something the launcher wrote and returns it
// wrapped so that running it -- however it is reached -- also unregisters it.
var trackTemp = deliverypkg.TrackTemp

// cleanupTempFiles removes everything the launcher has written and not yet
// cleaned up. Every exit this process can observe calls it.
var cleanupTempFiles = deliverypkg.CleanupTempFiles

// writeInventory marshals the inventory to a private 0600 file in a private
// directory and returns its path along with a cleanup func.
var writeInventory = deliverypkg.WriteInventory

// newSecretID mints an id for a credential the launcher is about to save.
var newSecretID = deliverypkg.NewSecretID

// recordSaved adds an entry to the launcher's index of ids it created.
var recordSaved = deliverypkg.RecordSaved

// kubeEnvForContext returns the environment a child needs to target a
// Kubernetes context, having written the kubeconfig copy that carries it.
var kubeEnvForContext = deliverypkg.KubeEnvForContext

// storeCredentialFn is the keychain write, replaceable in tests: the failure
// path matters more than the success one and cannot be provoked otherwise.
var storeCredentialFn = deliverypkg.StoreCredential

// storeCredentialWithin is the keychain write with a bound on how long it may
// take, whatever the backend does with the context it is given.
//
// The variable is read here, in the caller's goroutine, and handed over: the
// write is left running when the wait gives up, so a goroutine that read the
// variable itself could still be reading it after a test had put the real one
// back.
func storeCredentialWithin(limit time.Duration, id string, cred *vault.Credential) error {
	return deliverypkg.StoreCredentialWithin(limit, storeCredentialFn, id, cred)
}
