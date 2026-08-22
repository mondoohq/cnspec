// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package delivery

import (
	"strings"

	"github.com/cockroachdb/errors"
	tuiform "go.mondoo.com/cnspec/cli/tui/form"
	"go.mondoo.com/mql/llx"
	"go.mondoo.com/mql/providers"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/providers-sdk/v1/vault"
	"go.mondoo.com/mql/types"
)

// The launcher does not know what a connector's flags mean, and it never has
// to: the provider that reads them will say, over the same call `cnspec scan`
// itself makes.
//
// ParseCLI takes a connector name, the positional arguments and a map of flag
// values, and answers with the inventory.Asset it would have connected to. That
// asset *is* the mapping -- which option key each flag lands under, whether the
// credential wants a password, a bearer token, a pkcs12 bundle or a private
// key, which label cred.User has to carry for the provider's credential switch
// to route it, and what the connection type is called.
//
// A hand-written table cannot produce that answer, and the previous one did not.
// Of the connectors audited before this was written, thirteen needed a
// credential shape the launcher's own BuildInventory did not produce and seven
// needed a positional mapping no metadata carries -- github's two arguments
// become organization, user or repository depending on the first one, shodan's
// first becomes Options["search"] and its second becomes conn.Host. Every one
// of those is a fact the provider already knows and nobody else does.
//
// The secret travels to the provider in a protobuf field over the plugin's
// gRPC connection. That is the invariant this whole package exists for: it
// never becomes a word in a child process's argv, where `ps auxww` publishes
// it to every user on the machine.

// CLIParser answers what a command line means, in the provider's own terms.
//
// It is an interface with one method so that the tests can answer for a
// provider that is not installed -- and so that the launcher's own tests do not
// each spawn a plugin subprocess. See ProviderParser for the real one.
type CLIParser interface {
	ParseCLI(provider, connector string, args []string, flags map[string]*llx.Primitive) (*inventory.Asset, error)
}

// Parser is what the launcher asks. It is a variable so a test can answer
// without a provider on disk; nothing else replaces it.
var Parser CLIParser = ProviderParser{}

// ProviderParser starts the provider plugin and asks it.
type ProviderParser struct{}

// ParseCLI installs the provider if it is missing, starts it, and asks it what
// this command line means.
//
// The install is EnsureProvider, the same call the launcher already makes when
// a user opens a form, and the same one a scan would make on first use. By the
// time a credential is being delivered the provider is therefore almost always
// already on disk and already warm, because the form the credential was typed
// into could not have been drawn without it.
//
// AutoUpdate is off for the runtime because EnsureProvider has just done it.
// Leaving it on costs a second HTTPS round trip to releases.mondoo.com on every
// launch -- measured at roughly 250ms of the 400ms a cold parse takes -- to
// re-answer a question asked moments earlier.
func (ProviderParser) ParseCLI(providerName, connector string, args []string, flags map[string]*llx.Primitive) (*inventory.Asset, error) {
	p, err := providers.EnsureProvider(
		providers.ProviderLookup{ProviderName: providerName}, true, nil)
	if err != nil {
		return nil, errors.Wrap(err, "cannot load the "+providerName+" provider")
	}

	runtime := providers.Coordinator.NewRuntime()
	runtime.AutoUpdate = providers.UpdateProvidersConfig{Enabled: false}
	defer runtime.Close()

	if err := runtime.UseProvider(p.ID); err != nil {
		return nil, errors.Wrap(err, "cannot start the "+providerName+" provider")
	}
	res, err := runtime.Provider.Instance.Plugin.ParseCLI(&plugin.ParseCLIReq{
		Connector: connector,
		Args:      args,
		Flags:     flags,
	})
	if err != nil {
		return nil, err
	}
	if res == nil || res.Asset == nil {
		// The provider accepted the arguments and produced no target. mql's own
		// caller only warns about this and carries on into a connection that
		// cannot work; here it is the difference between an inventory and an
		// empty file, so it is an error.
		return nil, errors.New(connector + " did not produce a target from these settings")
	}
	return res.Asset, nil
}

// CLIRequest is one connector invocation in the shape ParseCLI takes.
type CLIRequest struct {
	Args  []string
	Flags map[string]*llx.Primitive
}

// RequestFor renders a filled-in form as a ParseCLI request.
//
// It is the same reading of the form that Args() makes -- the same visibility
// rule, the same argument order -- with one difference: secrets are included.
// See tuiform.Form.FlagFields.
//
// Only answered flags are sent. mql's own caller sends a zero value for every
// flag that declares ConfigEntry "-", and a provider that distinguishes "absent"
// from "empty" would see a difference; sending less is the safe direction,
// because an absent key is what a provider sees when a user omits the flag.
func RequestFor(f tuiform.Form, flags []plugin.Flag) CLIRequest {
	byName := make(map[string]plugin.Flag, len(flags))
	for _, fl := range flags {
		byName[fl.Long] = fl
	}

	out := CLIRequest{Args: f.Positional(), Flags: map[string]*llx.Primitive{}}
	for _, fd := range f.FlagFields() {
		out.Flags[fd.Flag] = primitiveFor(fd, byName[fd.Flag])
	}
	return out
}

// primitiveFor renders one answer as the flag's declared type.
//
// The connector's own declaration decides, not the widget, because the type is
// what the provider unmarshals on the other side. Sending the wrong one is not
// a type error anywhere: llx.Primitive carries opaque bytes, so a provider
// reading string(prim.Value) on an int gets the varint's bytes as text. That is
// not hypothetical -- `cnspec shell activedirectory --dc x --user u --password
// p --port 389` fails today with `port flag must be a valid integer:
// strconv.Atoi: parsing "\x8a\x06"`, which is 389 zigzagged, because that
// provider declares --port as an int and reads it as a string. Nothing the
// launcher does can fix that; sending what mql's own CLI sends at least means
// the launcher is not a second cause of it.
//
// plugin.FlagType counts from 1, so a zero Type means no declaration reached
// here at all, and the widget answers instead. --discover is the case that
// matters: the CLI synthesizes it rather than the provider declaring it, so it
// is absent from Connector.Flags, and it is a list.
func primitiveFor(fd tuiform.Field, fl plugin.Flag) *llx.Primitive {
	switch fl.Type {
	case plugin.FlagType_Bool:
		return llx.BoolPrimitive(fd.On())
	case plugin.FlagType_Int:
		return llx.IntPrimitive(parseInt(fd.Emitted()))
	case plugin.FlagType_String:
		return llx.StringPrimitive(stringValue(fd))
	case plugin.FlagType_List:
		return llx.ArrayPrimitiveT(listValue(fd), llx.StringPrimitive, types.String)
	case plugin.FlagType_KeyValue:
		return llx.MapPrimitiveT(keyValue(fd), llx.StringPrimitive, types.String)
	}

	switch fd.Kind {
	case tuiform.KindBool:
		return llx.BoolPrimitive(fd.On())
	case tuiform.KindMultiChoice:
		return llx.ArrayPrimitiveT(fd.Selected(), llx.StringPrimitive, types.String)
	default:
		return llx.StringPrimitive(fd.Emitted())
	}
}

// stringValue is a string flag's answer. A multi-choice answering one spells
// itself the way pflag's own StringSlice would be read back, which is how
// `--values a.yaml,b.yaml` already reaches a provider from a shell.
func stringValue(fd tuiform.Field) string {
	if fd.Kind == tuiform.KindMultiChoice {
		return strings.Join(fd.Selected(), ",")
	}
	return fd.Emitted()
}

// listValue is a list flag's answer, from whichever widget answered it.
// TypeEmptyLists turns a list with nothing to pick from into a text box, so
// both spellings reach here and pflag splits both on commas.
func listValue(fd tuiform.Field) []string {
	if fd.Kind == tuiform.KindMultiChoice {
		return fd.Selected()
	}
	return splitList(fd.Emitted())
}

func keyValue(fd tuiform.Field) map[string]string {
	out := map[string]string{}
	for _, pair := range splitList(fd.Emitted()) {
		k, v, ok := strings.Cut(pair, "=")
		if !ok {
			continue
		}
		out[strings.TrimSpace(k)] = v
	}
	return out
}

func splitList(s string) []string {
	if s == "" {
		return nil
	}
	out := strings.Split(s, ",")
	for i := range out {
		out[i] = strings.TrimSpace(out[i])
	}
	return out
}

func parseInt(s string) int64 {
	var n int64
	var neg bool
	for i, r := range s {
		if i == 0 && r == '-' {
			neg = true
			continue
		}
		if r < '0' || r > '9' {
			return 0
		}
		n = n*10 + int64(r-'0')
	}
	if neg {
		return -n
	}
	return n
}

// Placement is where a provider put a secret it was handed.
//
// It is detected rather than declared, which is the one thing a hand-written
// table could never do: the launcher looks for the value it sent in the asset
// that came back, and the answer decides whether the OS keychain can hold it.
type Placement int

const (
	// PlacedNowhere: the value is not in the asset at all. The provider read
	// the flag and dropped it, or never read it. Launching would scan
	// unauthenticated -- or, for databricks, whatever DATABRICKS_TOKEN in the
	// ambient environment names, because its credential switch has no default
	// arm and the SDK resolves the variable itself. That is a refusal.
	PlacedNowhere Placement = iota
	// PlacedCredential: the value is in conf.Credentials, so it can be
	// replaced by a reference to an OS keychain entry.
	PlacedCredential
	// PlacedOption: the value is in conf.Options, where the provider reads it
	// as a plain connection setting. A vault reference is not read from there,
	// so this credential can only travel as plaintext inside the generated
	// file.
	//
	// This is commoner than the AI connectors it was first noticed on. Measured
	// against the installed provider set, thirteen credential fields across
	// eleven connectors land here: anthropic and claude (--token and
	// --admin-token each), datadog (--app-key), elasticsearch (--api-key),
	// huggingface, ollama, openai and proxmox (--token), nutanix (--api-key),
	// openstack (--application-credential-secret) and tailscale
	// (--client-secret). Four of those connectors put a *different* credential
	// in conf.Credentials on the same form, so "reads conf.Credentials at all"
	// is not the question -- the question is where this value went, which is
	// why it is detected per value rather than declared per connector.
	//
	// The set is printed on every run by
	// TestEveryConnectorExportsSomethingTheLoaderAccepts, so nobody has to
	// trust this paragraph.
	PlacedOption
)

// Located is one secret the launcher sent, and what the provider did with it.
type Located struct {
	// Flag is the form field the value came from.
	Flag string
	// Placement is where it landed.
	Placement Placement
	// conn and index name the exact credential to swap for a keychain
	// reference, so that a form carrying two secrets replaces the one that was
	// saved and leaves the other alone.
	conn  *inventory.Config
	index int
}

// Credential is the credential the provider built to hold this secret, or nil
// when it did not build one.
func (l Located) Credential() *vault.Credential {
	if l.Placement != PlacedCredential || l.conn == nil || l.index >= len(l.conn.Credentials) {
		return nil
	}
	return l.conn.Credentials[l.index]
}

// Keychainable picks the one credential the launcher saves to the OS keychain,
// or nil when none of the form's secrets landed somewhere a vault reference is
// read from.
//
// One rather than all, because a keychain entry is referenced by a single id
// and the reference replaces exactly the credential it stands for. A form
// holding two secrets -- ssh with a password and an identity file, unifi with
// two alternative credentials -- gets the first protected and the rest left as
// the provider built them. Replacing the whole list is what the code this
// succeeded did, and it meant the second credential survived a keychain
// *failure* and vanished when the keychain *worked*: the broken behaviour lived
// on the path that succeeds.
func Keychainable(placed []Located) *Located {
	for i := range placed {
		if placed[i].Placement == PlacedCredential {
			return &placed[i]
		}
	}
	return nil
}

// Locate reports, for each secret the form holds, where the provider put it.
//
// Matching is by value: the launcher knows what it sent and looks for it in
// what came back. That is exact rather than heuristic, and it is why a provider
// that renames the option, retypes the credential or tags cred.User with a
// label needs nothing registered here.
//
// Reference secrets are skipped. A field marked Reference names a file holding
// a credential rather than holding one, so the path is not a secret to protect
// and the provider is free to read the file rather than carry the path.
func Locate(f tuiform.Form, asset *inventory.Asset) []Located {
	var out []Located
	for _, fd := range f.Secrets() {
		if fd.Reference {
			continue
		}
		out = append(out, locateValue(fd.Flag, fd.Value(), asset))
	}
	return out
}

func locateValue(flag, value string, asset *inventory.Asset) Located {
	found := Located{Flag: flag}
	if value == "" || asset == nil {
		return found
	}
	for _, conn := range asset.Connections {
		for i, cred := range conn.Credentials {
			if cred == nil {
				continue
			}
			if credentialHolds(cred, value) {
				return Located{Flag: flag, Placement: PlacedCredential, conn: conn, index: i}
			}
		}
		for _, v := range conn.Options {
			if v == value {
				// Keep looking: a provider that writes the value into both an
				// option and a credential can still be keychain-protected, and
				// the credential is the better answer.
				found = Located{Flag: flag, Placement: PlacedOption, conn: conn}
			}
		}
	}
	return found
}

// credentialHolds reports whether this credential carries the value, in
// whichever of its fields the provider chose to put it.
//
// User is one of them, and it is not a mistake. A provider is free to decide
// that the secret *is* the identity: clickhousecloud's --api-key becomes the
// user of a password credential whose password is --api-secret, which is what
// that service's key pair actually is. Reading the credential fields
// selectively is what would be wrong -- the whole credential goes to the
// keychain and comes back whole, so wherever the value sits inside it, it is
// protected.
//
// Matching User cannot collide with the labels several providers put there --
// hcp's "client-secret", stackit's option name, databricks' "token" -- because
// the comparison is against the value the launcher sent, not against a name.
func credentialHolds(cred *vault.Credential, value string) bool {
	return cred.Password == value ||
		cred.User == value ||
		string(cred.Secret) == value ||
		string(cred.PrivateKey) == value ||
		cred.PrivateKeyPath == value
}

// InventoryFor wraps the provider's own asset as a scannable inventory, with
// the saved credential swapped for a reference to it.
//
// The asset is used as it came back. Nothing is added to it and nothing is
// re-keyed, because every one of those edits would be the launcher second-
// guessing the provider that has to read the result -- which is the failure
// this replaced.
//
// secretID, when set, is a keychain entry the OS is holding, and it stands in
// for exactly one credential: the one Save picked. The rest are left as the
// provider built them.
func InventoryFor(connector string, asset *inventory.Asset, saved *Located, secretID string) *inventory.Inventory {
	if asset.Id == "" {
		asset.Id = "cnspec-ui-" + connector
	}
	if asset.Name == "" {
		asset.Name = connector
	}

	if secretID != "" && saved != nil && saved.conn != nil &&
		saved.index < len(saved.conn.Credentials) {
		// A credential reference carries the id and nothing else: the loader
		// rejects a reference that also declares a type, and silently discards
		// inline material when an id is present.
		saved.conn.Credentials[saved.index] = &vault.Credential{SecretId: secretID}
	}

	inv := inventory.New(inventory.WithAssets(asset))
	if secretID != "" {
		// There is no --vault flag; naming the vault in the inventory is the
		// only way to point cnspec at the OS keychain.
		inv.Spec.Vault = &vault.VaultConfiguration{
			Name: vaultService,
			Type: vault.VaultType_KeyRing,
		}
	}
	return inv
}
