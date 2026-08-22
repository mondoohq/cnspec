// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package source

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cockroachdb/errors"
)

// The readers behind five of the enumerated sources: five local files that
// already name the thing the user is about to be asked for -- ~/.oci/config,
// ~/.alibabacloud/credentials, ~/.azure/azureProfile.json,
// ~/.snowflake/connections.toml and the docker context store.
//
// What they have in common is the shape, not the format. Each is a file this
// machine already has, read without a network or a credential. The Source
// declarations that use them are in declare.go with every other pre-connection
// declaration; this file is the part that differs per format, which is the
// part no amount of declaring can make uniform.
//
// Every reader takes a path so it is testable without a $HOME, and every reader
// distinguishes "nothing configured" from "could not read": a picker that shows
// an empty list for a file it never opened is the failure this contract exists
// to stop.

const (
	// The literal DEFAULT is the OCI convention, in capitals, and is what the
	// connector's own --profile help names. Lowercasing it here would prefill a
	// profile that does not exist.
	ociDefaultProfile = "DEFAULT"

	// AlicloudProfileEnv carries the chosen profile, since the connector has no
	// flag for it. Verified in aliyun/credentials-go's ProfileCredentialsProvider,
	// whose precedence is: explicit name, then this variable, then "default".
	AlicloudProfileEnv = "ALIBABA_CLOUD_PROFILE"
	// alicloudCredentialsEnv relocates the credentials file itself.
	alicloudCredentialsEnv  = "ALIBABA_CLOUD_CREDENTIALS_FILE"
	alicloudDefaultProfile  = "default"
	snowflakeHomeEnv        = "SNOWFLAKE_HOME"
	snowflakeConnectionFile = "connections.toml"
	// snowflakeDefaultConnection is the table Snowflake's tooling falls back to
	// when no connection is named.
	snowflakeDefaultConnection = "default"
	// azureConfigDirEnv relocates the az CLI's whole config directory, and so
	// the profile file inside it.
	azureConfigDirEnv = "AZURE_CONFIG_DIR"
	azureProfileFile  = "azureProfile.json"

	// The docker names, spelled as docker/cli spells them: DOCKER_CONFIG for
	// the config directory, "contexts" for the store inside it, and the context
	// named "default" for the CLI's own env-and-default-socket resolution.
	dockerConfigDirEnv    = "DOCKER_CONFIG"
	dockerConfigFile      = "config.json"
	dockerContextsDir     = "contexts"
	dockerContextMetaDir  = "meta"
	dockerContextMetaFile = "meta.json"
	DockerContextEnv      = "DOCKER_CONTEXT"
	DockerHostEnv         = "DOCKER_HOST"
	DockerDefaultContext  = "default"
)

// missingFileExplain turns a file-read failure into the one sentence that says
// what to do about it.
//
// A picker backed by a file has exactly two failures worth telling apart, and
// neither of them reads well raw: os wraps both in a path and a syscall name,
// which says where the launcher looked but not what the user should do. Anything
// else -- a malformed file, a truncated read -- keeps its own words, because a
// message of ours would say less.
func missingFileExplain(what, remedy string) ExplainFunc {
	return func(err error) error {
		switch {
		case errors.Is(err, fs.ErrNotExist):
			return errors.New("no " + what + " — " + remedy)
		case errors.Is(err, fs.ErrPermission):
			return errors.New("cannot read " + what + " — check its permissions")
		}
		return stripRPCPrefix(err)
	}
}

// profileNamesFrom reads the section names, and nothing else, out of a
// credentials file.
//
// One function for OCI and for Alibaba Cloud, because it was one function
// twice: the two readers were byte-identical, and the only thing that ever
// differed was the sentence above them explaining why each could not use a
// library. Both reasons are the same reason. ~/.oci/config is ini-*like*
// rather than ini -- Oracle's SDK hand-rolls the parse and is not in cnspec's
// module graph -- and ~/.alibabacloud/credentials is a true ini that
// gopkg.in/ini.v1 would read in a line. What settles both is what is next to
// the names: private keys and their passphrases in one, an access key secret
// in every section of the other. See ini.go.
func profileNamesFrom(path string) ([]string, error) {
	names, _, err := iniSections(iniScan{files: iniPath(path)})
	if len(names) == 0 && err != nil {
		return nil, err
	}
	sort.Strings(names)
	return names, nil
}

// ociProfiles lists the profiles in the OCI config file.
func ociProfiles() ([]string, error) { return profileNamesFrom(home(".oci", "config")) }

// alicloudProfiles lists the profiles in the Alibaba Cloud credentials file.
func alicloudProfiles() ([]string, error) {
	return profileNamesFrom(alicloudCredentialsPath())
}

// alicloudCredentialsPath is the credentials file the SDK would read.
func alicloudCredentialsPath() string {
	if path := os.Getenv(alicloudCredentialsEnv); path != "" {
		return path
	}
	return home(".alibabacloud", "credentials")
}

// snowflakeConnectionsPath is the connections file the Snowflake tooling reads.
func snowflakeConnectionsPath() string {
	if dir := os.Getenv(snowflakeHomeEnv); dir != "" {
		return filepath.Join(dir, snowflakeConnectionFile)
	}
	return home(".snowflake", snowflakeConnectionFile)
}

// snowflakeAccountKey is the one key the connections file may be read for.
var snowflakeAccountKey = map[string]bool{"account": true}

func snowflakeAccounts() ([]string, error) {
	return snowflakeAccountsFrom(snowflakeConnectionsPath())
}

// snowflakeAccountsFrom offers the account identifiers named in
// connections.toml -- not the connection names.
//
// This is the decision the id SnowflakeConnection does not make for itself.
// connections.toml is a table per connection, and the obvious picker would list
// those table names. But nothing in cnspec accepts one: the connector declares
// --user, --account, --role, --region, --token, --password and --identity-file,
// and no flag and no variable it reads takes a connection name. Offering "audit"
// as a value would put `--account audit` on the command line, which is a
// confidently wrong answer of exactly the kind this contract exists to prevent.
//
// So the picker offers the one thing in that file the connector can use. Attach
// it to --account.
//
// `account` is a locator, not a credential, which is what makes reading it
// allowable at all: the allowlist is one key wide, and the passwords, tokens and
// key paths in the same table are never held. That is the same shape as the AWS
// reader's sso_account_id, and the reason neither file goes through a TOML or
// ini library.
func snowflakeAccountsFrom(path string) ([]string, error) {
	_, values, err := iniSections(iniScan{files: iniPath(path), want: snowflakeAccountKey})
	if len(values) == 0 && err != nil {
		return nil, err
	}
	out := make([]string, 0, len(values))
	for _, keys := range values {
		out = append(out, keys.last("account"))
	}
	return SortedUnique(out), nil
}

// snowflakeDefaultAccount is the account of the connection named "default".
func snowflakeDefaultAccount() string {
	return snowflakeDefaultAccountFrom(snowflakeConnectionsPath())
}

func snowflakeDefaultAccountFrom(path string) string {
	_, values, err := iniSections(iniScan{files: iniPath(path), want: snowflakeAccountKey})
	if err != nil && len(values) == 0 {
		return ""
	}
	return values[snowflakeDefaultConnection].last("account")
}

// utf8BOM is what the az CLI writes at the start of azureProfile.json. Go's
// encoding/json refuses a document that begins with it -- "invalid character
// 'ï' looking for beginning of value" -- so it comes off before the decode.
var utf8BOM = []byte{0xef, 0xbb, 0xbf}

// azureProfileDoc is the part of the az CLI's profile this launcher reads.
//
// It is not internal/onboarding's AzAccount, which describes the JSON `az
// account list` prints rather than the JSON `az` stores. The two differ in one
// field that matters: the command emits `cloudName`, the file writes
// `environmentName`, so decoding this file into AzAccount leaves the cloud empty
// and says nothing about it.
type azureProfileDoc struct {
	Subscriptions []azureProfileSubscription `json:"subscriptions"`
}

type azureProfileSubscription struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// State is "Enabled" or "Disabled". A disabled subscription is still
	// listed, and is not scannable.
	State           string `json:"state"`
	IsDefault       bool   `json:"isDefault"`
	TenantID        string `json:"tenantId"`
	EnvironmentName string `json:"environmentName"`
}

// azureProfilePath is the az CLI's profile file.
func azureProfilePath() string {
	if dir := os.Getenv(azureConfigDirEnv); dir != "" {
		return filepath.Join(dir, azureProfileFile)
	}
	return home(".azure", azureProfileFile)
}

func azureProfileFrom(path string) (azureProfileDoc, error) {
	var doc azureProfileDoc
	data, err := os.ReadFile(path)
	if err != nil {
		return doc, err
	}
	if err := json.Unmarshal(bytes.TrimPrefix(data, utf8BOM), &doc); err != nil {
		return doc, errors.Wrap(err, "cannot read the az CLI profile")
	}
	return doc, nil
}

func azureSubscriptions() ([]string, error) { return azureSubscriptionsFrom(azureProfilePath()) }

// azureSubscriptionsFrom lists the ids of the subscriptions az knows about.
//
// Ids rather than names, because --subscription takes an id: the connector's own
// help says so and its provider reads flags["subscription"] straight through. A
// name would look better in the list and would not connect.
func azureSubscriptionsFrom(path string) ([]string, error) {
	doc, err := azureProfileFrom(path)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(doc.Subscriptions))
	for _, sub := range doc.Subscriptions {
		// A disabled subscription cannot be scanned, so offering it would only
		// produce a failure the launcher could have predicted.
		if !strings.EqualFold(sub.State, "Enabled") {
			continue
		}
		out = append(out, sub.ID)
	}
	return SortedUnique(out), nil
}

// azureDefaultSubscription is the id az would use with no --subscription.
func azureDefaultSubscription() string { return azureDefaultSubscriptionFrom(azureProfilePath()) }

func azureDefaultSubscriptionFrom(path string) string {
	doc, err := azureProfileFrom(path)
	if err != nil {
		return ""
	}
	for _, sub := range doc.Subscriptions {
		if sub.IsDefault && strings.EqualFold(sub.State, "Enabled") {
			return sub.ID
		}
	}
	return ""
}

// dockerConfigDir is the directory the docker CLI keeps its config in.
func dockerConfigDir() string {
	if dir := os.Getenv(dockerConfigDirEnv); dir != "" {
		return dir
	}
	return home(".docker")
}

func dockerContexts() ([]string, error) { return dockerContextsFrom(dockerConfigDir()) }

// dockerContextsFrom lists the contexts in a docker config directory.
//
// The store keys each context by the sha256 of its name, so the directory names
// are digests and the name itself is the `Name` field inside each meta.json --
// the one field read here. docker/cli's own store.ContextStore.List() does
// exactly this and is the reader mql's dockerclient goes through, but it is an
// indirect dependency: importing it directly promotes it in go.mod, and this
// phase owns no existing file. Reading the one field keeps the launcher's list
// and mql's connection talking about the same set of contexts without that.
//
// "default" is always offered. It is not in the store -- it is the docker CLI's
// name for its own DOCKER_HOST-and-default-socket resolution -- but it is a
// context a user can choose, and DOCKER_CONTEXT=default selects it.
func dockerContextsFrom(configDir string) ([]string, error) {
	names := []string{DockerDefaultContext}

	metaRoot := filepath.Join(configDir, dockerContextsDir, dockerContextMetaDir)
	entries, err := os.ReadDir(metaRoot)
	if err != nil {
		// A machine that has never run `docker context create` has no store,
		// and that is not a failure: it still has the default context.
		if errors.Is(err, fs.ErrNotExist) {
			return names, nil
		}
		return nil, err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(metaRoot, entry.Name(), dockerContextMetaFile))
		if err != nil {
			continue
		}
		// Capitalised, because that is how the store writes it: the docker/cli
		// type has no json tag on Name beyond ",omitempty".
		var meta struct {
			Name string `json:"Name"`
		}
		if json.Unmarshal(data, &meta) != nil {
			continue
		}
		names = append(names, meta.Name)
	}
	return SortedUnique(names), nil
}

// dockerCurrentContext reports the context the docker CLI would use.
// DockerContextEnvFrom turns a picker's source parameters into the environment
// the enumeration has to run under.
//
// A picker that ignored the chosen context would offer the default daemon's
// containers for a scan pointed at another one -- a list that looks right and
// names nothing the child can reach. The neutralisation is the same one the
// launch applies, and for the same reason: DOCKER_HOST outranks DOCKER_CONTEXT
// in the docker CLI and in mql's dockerclient, so leaving an inherited one in
// place would pin the enumeration back to the default while the variable said
// otherwise.
func DockerContextEnvFrom(params []string) []string {
	const want = "s:" + SpecialDockerContext + "="
	for _, p := range params {
		if !strings.HasPrefix(p, want) {
			continue
		}
		value := strings.TrimPrefix(p, want)
		if value == "" {
			return nil
		}
		return append([]string{DockerContextEnv + "=" + value},
			NeutralisedBy(DockerContextEnv, value)...)
	}
	return nil
}

// NeutralisedBy returns the entries that stop an inherited variable from
// overriding one that is being set deliberately.
//
// DOCKER_HOST is the case, and it is the silent kind. The docker CLI resolves
// its target as DOCKER_HOST first, then DOCKER_CONTEXT, then the config file --
// mql's own dockerclient follows the same order -- so a DOCKER_HOST already in
// the environment pins the child to the `default` context and the context the
// user picked in the launcher does nothing at all. Nothing fails: the scan runs
// against a host the user did not choose and reports on it confidently.
//
// Clearing it rather than warning about it is the choice, for two reasons. The
// launcher's whole contract is that what you pick is what gets scanned, and a
// warning that says "your choice will be ignored" is an admission that it is
// not. And the clearing is scoped as narrowly as it can be: one child process,
// one variable, only when the user actively chose a context, and only in the
// direction of honouring that choice.
//
// The `default` context is deliberately exempt. It is not a host of its own --
// it *is* the DOCKER_HOST-and-default-socket resolution, which is why
// dockerCurrentContextFrom prefills it when DOCKER_HOST is set -- so clearing
// the variable there would retarget the scan away from the daemon the user was
// already pointed at, which is the very bug this is fixing.
//
// It lives here rather than with the launch it is applied to. It is a fact
// about how docker resolves a target, spelled in the same three constants as
// the picker that offers one, and both callers -- the enumeration above and
// the launch -- need exactly the same answer. Split across the two files it
// was two halves of one rule with the constants on one side and the use on the
// other.
func NeutralisedBy(envVar, value string) []string {
	if envVar != DockerContextEnv || value == "" || value == DockerDefaultContext {
		return nil
	}
	// An empty value, not an absent one: os/exec keeps the last entry for a
	// repeated key, and every reader of DOCKER_HOST -- the docker CLI, mql's
	// dockerclient -- treats empty as unset.
	return []string{DockerHostEnv + "="}
}

func dockerCurrentContext() string { return dockerCurrentContextFrom(dockerConfigDir()) }

// dockerCurrentContextFrom mirrors the docker CLI's own precedence, which mql's
// dockerclient follows too: an explicit DOCKER_HOST pins the CLI to the default
// context whatever else is configured, then DOCKER_CONTEXT, then the
// currentContext in config.json.
//
// The DOCKER_HOST case is the one worth having: with it set, choosing any other
// context changes nothing, and a launcher that offered one anyway would be
// promising a target it cannot reach.
func dockerCurrentContextFrom(configDir string) string {
	if os.Getenv(DockerHostEnv) != "" {
		return DockerDefaultContext
	}
	if ctx := os.Getenv(DockerContextEnv); ctx != "" {
		return ctx
	}
	data, err := os.ReadFile(filepath.Join(configDir, dockerConfigFile))
	if err != nil {
		return ""
	}
	var cfg struct {
		CurrentContext string `json:"currentContext"`
	}
	if json.Unmarshal(data, &cfg) != nil {
		return ""
	}
	return cfg.CurrentContext
}
