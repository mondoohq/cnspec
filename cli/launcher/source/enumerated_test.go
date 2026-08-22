// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package source

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/cockroachdb/errors"
)

// Each fixture under testdata carries the catch its reader exists for -- a BOM,
// a capitalised DEFAULT, a sha256 directory name -- because a fixture without it
// proves only that the happy path parses, which was never in doubt.

const (
	ociFixture       = "testdata/oci_config"
	alicloudFixture  = "testdata/alicloud_credentials"
	azureFixtureDir  = "testdata/azure"
	snowflakeFixture = "testdata/snowflake"
	dockerFixtureDir = "testdata/docker"
)

// The OCI default profile is the literal DEFAULT. Reading it case-insensitively,
// or normalising it to the AWS spelling, prefills a profile the connector will
// not find.
func TestOCIProfilesKeepTheOracleSpelling(t *testing.T) {
	got, err := profileNamesFrom(ociFixture)
	if err != nil {
		t.Fatalf("reading the fixture: %v", err)
	}
	want := []string{"DEFAULT", "SANDBOX", "eu-audit"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("profiles = %v, want %v", got, want)
	}

	// The last one is there because a header with a trailing comment is what an
	// edited config file actually looks like.
	if slices.Contains(got, "eu-audit]  ; read-only auditor") {
		t.Error("a trailing comment became part of the profile name")
	}

	if v, why := PreferredValue(OCIProfile, got); v != "DEFAULT" || why != "default" {
		t.Errorf("prefilled %q/%q, want DEFAULT", v, why)
	}
	if v, _ := PreferredValue(OCIProfile, []string{"SANDBOX", "eu-audit"}); v != "" {
		t.Errorf("prefilled %q with no DEFAULT profile present", v)
	}
}

// The whole reason these are scanners and not ini libraries: the files hold live
// credentials, and answering "what are the profiles called" must not pull them
// into this process.
func TestTheIniReadersHoldNoValues(t *testing.T) {
	for _, path := range []string{ociFixture, alicloudFixture} {
		names, values, err := iniSections(iniScan{files: iniPath(path)})
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		if len(names) == 0 {
			t.Fatalf("%s: no sections read, so this proves nothing", path)
		}
		if len(values) != 0 {
			t.Errorf("%s: the reader kept %d sections' values", path, len(values))
		}
	}

	// And with an allowlist, it keeps only what is on it. The snowflake fixture
	// has a password in every table it has an account in.
	_, values, err := iniSections(iniScan{
		files: iniPath(filepath.Join(snowflakeFixture, "connections.toml")),
		want:  map[string]bool{"account": true},
	})
	if err != nil {
		t.Fatalf("reading the fixture: %v", err)
	}
	if len(values) == 0 {
		t.Fatal("no tables read, so this proves nothing")
	}
	for section, keys := range values {
		for _, kv := range keys {
			if kv.key != "account" {
				t.Errorf("[%s] kept %q, which is not on the allowlist", section, kv.key)
			}
		}
	}
}

// A missing file is not an empty list, and the difference is the only thing a
// user can act on.
func TestAMissingFileIsExplainedRatherThanShownAsEmpty(t *testing.T) {
	cases := []struct {
		id    string
		fetch func(string) ([]string, error)
		want  string
	}{
		{OCIProfile, profileNamesFrom, "oci setup config"},
		{AlicloudProfile, profileNamesFrom, "--access-key-id"},
		{AzureSubscription, azureSubscriptionsFrom, "az login"},
		{SnowflakeConnection, snowflakeAccountsFrom, "account identifier"},
	}
	for _, c := range cases {
		values, err := c.fetch(filepath.Join(t.TempDir(), "absent"))
		if err == nil {
			t.Errorf("%s: a missing file read as %v with no error", c.id, values)
			continue
		}
		if len(values) != 0 {
			t.Errorf("%s: a missing file produced values %v", c.id, values)
		}
		got := explainFailure(c.id, err).Error()
		if !strings.Contains(got, c.want) {
			t.Errorf("%s: explained as %q, want it to mention %q", c.id, got, c.want)
		}
		if strings.Contains(got, "\n") {
			t.Errorf("%s: a picker gets one line, got %q", c.id, got)
		}
		// The raw error names a syscall and a temp path; neither is useful.
		if strings.Contains(got, "no such file or directory") {
			t.Errorf("%s: the os error survived: %q", c.id, got)
		}
	}

	// Anything that is not a missing or unreadable file keeps its own words.
	own := errors.New("unexpected end of JSON input")
	if got := explainFailure(AzureSubscription, own).Error(); got != own.Error() {
		t.Errorf("an unrecognised failure was rewritten to %q", got)
	}
	if got := explainFailure(OCIProfile, fs.ErrPermission).Error(); !strings.Contains(got, "permissions") {
		t.Errorf("a permission failure explained as %q", got)
	}
}

// alicloud is the case Source.Env exists for. The claim is checked against the
// connector's own declaration rather than asserted in prose, because the whole
// failure mode is a picker wired to a flag that does not exist.
func TestAlicloudProfilesTravelInTheEnvironment(t *testing.T) {
	s, ok := ByID(AlicloudProfile)
	if !ok {
		t.Fatal("the alicloud profile source is not registered")
	}
	if s.Env != "ALIBABA_CLOUD_PROFILE" {
		t.Errorf("Env = %q, want the variable the SDK reads", s.Env)
	}

	// That alicloud really declares no --profile is checked against the
	// recorded connector metadata, which lives with the launcher: see
	// TestTheEnvBackedPickersFillNoFlag.
}

func TestAlicloudProfilesEnumerateEverySection(t *testing.T) {
	got, err := profileNamesFrom(alicloudFixture)
	if err != nil {
		t.Fatalf("reading the fixture: %v", err)
	}
	want := []string{"default", "ecs", "ram-role", "staging"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("profiles = %v, want %v", got, want)
	}

	if v, why := PreferredValue(AlicloudProfile, got); v != "default" || why != "default" {
		t.Errorf("prefilled %q/%q, want the conventional profile", v, why)
	}
	// What the environment already names wins, because that is what the child
	// would use whatever the list showed.
	t.Setenv("ALIBABA_CLOUD_PROFILE", "staging")
	if v, why := PreferredValue(AlicloudProfile, got); v != "staging" || !strings.Contains(why, "ALIBABA_CLOUD_PROFILE") {
		t.Errorf("prefilled %q/%q, want the profile the environment names", v, why)
	}
	// A variable naming something not in the file is not an opinion.
	t.Setenv("ALIBABA_CLOUD_PROFILE", "not-in-the-file")
	if v, why := PreferredValue(AlicloudProfile, got); v != "default" || why != "default" {
		t.Errorf("prefilled %q/%q for an unknown ALIBABA_CLOUD_PROFILE", v, why)
	}
}

func TestAlicloudCredentialsFileFollowsItsVariable(t *testing.T) {
	t.Setenv("ALIBABA_CLOUD_CREDENTIALS_FILE", alicloudFixture)
	got, err := alicloudProfiles()
	if err != nil {
		t.Fatalf("reading the fixture: %v", err)
	}
	if !slices.Contains(got, "ram-role") {
		t.Errorf("ALIBABA_CLOUD_CREDENTIALS_FILE was not honoured: %v", got)
	}
}

// The az CLI writes azureProfile.json with a UTF-8 BOM, and encoding/json
// refuses a document that starts with one. This asserts the fixture still has
// the BOM first, so the reader is not being congratulated on parsing a file the
// az CLI would never write.
func TestTheAzureFixtureStillHasItsBOM(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(azureFixtureDir, "azureProfile.json"))
	if err != nil {
		t.Fatalf("reading the fixture: %v", err)
	}
	if len(data) < 3 || data[0] != 0xef || data[1] != 0xbb || data[2] != 0xbf {
		t.Fatalf("the fixture no longer starts with a UTF-8 BOM: % x", data[:min(3, len(data))])
	}
	// And the catch is real: without the trim this is what happens.
	var doc azureProfileDoc
	if err := json.Unmarshal(data, &doc); err == nil {
		t.Error("encoding/json accepted a BOM, so the trim is no longer load-bearing")
	}
}

func TestAzureSubscriptionsSurviveTheBOM(t *testing.T) {
	path := filepath.Join(azureFixtureDir, "azureProfile.json")
	got, err := azureSubscriptionsFrom(path)
	if err != nil {
		t.Fatalf("reading the fixture: %v", err)
	}
	want := []string{
		"00000000-0000-0000-0000-00000000aaaa",
		"00000000-0000-0000-0000-00000000bbbb",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("subscriptions = %v, want the enabled ones %v", got, want)
	}
	// A disabled subscription is listed by az and cannot be scanned.
	if slices.Contains(got, "00000000-0000-0000-0000-00000000cccc") {
		t.Error("a Disabled subscription was offered")
	}

	// Exactly one entry is isDefault, which is the one `az account show` reports.
	if id := azureDefaultSubscriptionFrom(path); id != want[1] {
		t.Errorf("default subscription = %q, want %q", id, want[1])
	}
	t.Setenv("AZURE_CONFIG_DIR", azureFixtureDir)
	if v, why := PreferredValue(AzureSubscription, got); v != want[1] || why != "default" {
		t.Errorf("prefilled %q/%q, want the default subscription", v, why)
	}
}

// The file the az CLI stores and the JSON `az account list` prints are not the
// same document, and onboarding's AzAccount describes the second one. Decoding
// this file with it loses the cloud silently.
func TestTheAzureProfileFileNamesItsCloudDifferently(t *testing.T) {
	doc, err := azureProfileFrom(filepath.Join(azureFixtureDir, "azureProfile.json"))
	if err != nil {
		t.Fatalf("reading the fixture: %v", err)
	}
	if len(doc.Subscriptions) == 0 {
		t.Fatal("no subscriptions, so this proves nothing")
	}
	if doc.Subscriptions[0].EnvironmentName == "" {
		t.Error("environmentName did not decode; the field the file actually uses is empty")
	}

	var withCloudName struct {
		Subscriptions []struct {
			CloudName string `json:"cloudName"`
		} `json:"subscriptions"`
	}
	data, err := os.ReadFile(filepath.Join(azureFixtureDir, "azureProfile.json"))
	if err != nil {
		t.Fatalf("reading the fixture: %v", err)
	}
	if err := json.Unmarshal(data[3:], &withCloudName); err != nil {
		t.Fatalf("decoding the fixture: %v", err)
	}
	if withCloudName.Subscriptions[0].CloudName != "" {
		t.Error("this file does carry cloudName after all; reusing AzAccount would be fine")
	}
}

// The picker offers accounts, not connection names, because no flag and no
// variable this connector reads takes a connection name.
func TestSnowflakeOffersAccountsNotConnectionNames(t *testing.T) {
	path := filepath.Join(snowflakeFixture, "connections.toml")
	got, err := snowflakeAccountsFrom(path)
	if err != nil {
		t.Fatalf("reading the fixture: %v", err)
	}
	// Three tables, two distinct accounts: "default" and "legacy" share one.
	want := []string{"myorg-audit_account", "myorg-prod_account"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("accounts = %v, want %v", got, want)
	}
	for _, name := range []string{"default", "audit", "legacy"} {
		if slices.Contains(got, name) {
			t.Errorf("the connection name %q was offered as a value; --account would reject it", name)
		}
	}

	if acct := snowflakeDefaultAccountFrom(path); acct != "myorg-prod_account" {
		t.Errorf("the default connection's account = %q", acct)
	}
	t.Setenv("SNOWFLAKE_HOME", snowflakeFixture)
	if v, why := PreferredValue(SnowflakeConnection, got); v != "myorg-prod_account" || why == "" {
		t.Errorf("prefilled %q/%q, want the default connection's account", v, why)
	}

	// That --account exists and --connection does not is checked against the
	// recorded connector metadata; see TestTheFlagBackedPickersNameRealFlags.
	if s, _ := ByID(SnowflakeConnection); s.Env != "" {
		t.Errorf("Env = %q, but --account carries the value", s.Env)
	}
}

// The context store keys each context by the sha256 of its name, and the name
// itself is inside the file. A fixture whose directory name is anything else
// would still be read by a hand-rolled walk and would prove nothing about the
// store this actually goes through.
func TestTheDockerFixtureUsesRealDigests(t *testing.T) {
	metaRoot := filepath.Join(dockerFixtureDir, "contexts", "meta")
	entries, err := os.ReadDir(metaRoot)
	if err != nil {
		t.Fatalf("reading the fixture store: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("the fixture store is empty, so this proves nothing")
	}
	for _, e := range entries {
		data, err := os.ReadFile(filepath.Join(metaRoot, e.Name(), "meta.json"))
		if err != nil {
			t.Fatalf("%s: %v", e.Name(), err)
		}
		var meta struct {
			Name string `json:"Name"`
		}
		if err := json.Unmarshal(data, &meta); err != nil {
			t.Fatalf("%s: %v", e.Name(), err)
		}
		sum := sha256.Sum256([]byte(meta.Name))
		if want := hex.EncodeToString(sum[:]); want != e.Name() {
			t.Errorf("context %q lives in %s, want %s", meta.Name, e.Name(), want)
		}
	}
}

func TestDockerContextsComeFromTheStore(t *testing.T) {
	got, err := dockerContextsFrom(dockerFixtureDir)
	if err != nil {
		t.Fatalf("reading the fixture store: %v", err)
	}
	want := []string{"colima", "default", "remote-prod"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("contexts = %v, want %v", got, want)
	}

	// A machine with no context store still has the default context, and that
	// is not an error.
	only, err := dockerContextsFrom(t.TempDir())
	if err != nil {
		t.Fatalf("an absent store failed: %v", err)
	}
	if !reflect.DeepEqual(only, []string{"default"}) {
		t.Errorf("an absent store gave %v, want just the default context", only)
	}
}

func TestDockerContextPrecedence(t *testing.T) {
	// Nothing in the environment: config.json decides.
	t.Setenv("DOCKER_HOST", "")
	t.Setenv("DOCKER_CONTEXT", "")
	if got := dockerCurrentContextFrom(dockerFixtureDir); got != "colima" {
		t.Errorf("currentContext = %q, want the one in config.json", got)
	}

	// DOCKER_CONTEXT overrides the config file.
	t.Setenv("DOCKER_CONTEXT", "remote-prod")
	if got := dockerCurrentContextFrom(dockerFixtureDir); got != "remote-prod" {
		t.Errorf("with DOCKER_CONTEXT set, currentContext = %q", got)
	}

	// An explicit DOCKER_HOST pins the CLI to the default context whatever else
	// is configured, so offering another one would promise a target the child
	// will not use.
	t.Setenv("DOCKER_HOST", "tcp://127.0.0.1:2375")
	if got := dockerCurrentContextFrom(dockerFixtureDir); got != "default" {
		t.Errorf("with DOCKER_HOST set, currentContext = %q, want default", got)
	}

	t.Setenv("DOCKER_HOST", "")
	t.Setenv("DOCKER_CONTEXT", "")
	t.Setenv("DOCKER_CONFIG", dockerFixtureDir)
	values, err := dockerContexts()
	if err != nil {
		t.Fatalf("DOCKER_CONFIG was not honoured: %v", err)
	}
	if v, why := PreferredValue(DockerContext, values); v != "colima" || why != "current" {
		t.Errorf("prefilled %q/%q, want the current context", v, why)
	}
}

// Neither docker, container nor local declares --context, which is why the
// chosen context travels in DOCKER_CONTEXT.
func TestDockerContextsTravelInTheEnvironment(t *testing.T) {
	s, ok := ByID(DockerContext)
	if !ok {
		t.Fatal("the docker context source is not registered")
	}
	if s.Env != "DOCKER_CONTEXT" {
		t.Errorf("Env = %q, want the variable the docker CLI reads", s.Env)
	}
	// That none of docker, container and local declares --context is checked
	// against the recorded connector metadata; see
	// TestTheEnvBackedPickersFillNoFlag.
}

// Every one of the five reads a file on this machine, so none of them may defer
// and none of them may claim to be leaving the process. The generic contract
// tests cover the registry as a whole; this pins the five to the class and cost
// their ids were declared with.
func TestTheFiveLocalPickersAreInstantAndEnumerated(t *testing.T) {
	for _, id := range []string{
		OCIProfile, AlicloudProfile, AzureSubscription,
		SnowflakeConnection, DockerContext,
	} {
		s, ok := ByID(id)
		if !ok {
			t.Errorf("%s is declared but never registered", id)
			continue
		}
		if s.Class != ClassEnumerated {
			t.Errorf("%s: class %v, want ClassEnumerated", id, s.Class)
		}
		if s.Cost != CostInstant {
			t.Errorf("%s: cost %v, want CostInstant for a file read", id, s.Cost)
		}
		if len(s.Needs) != 0 {
			t.Errorf("%s: depends on %v, but reads a file that depends on nothing", id, s.Needs)
		}
		if s.Fetch == nil {
			t.Errorf("%s: no Fetch, so the picker is empty", id)
		}
		if s.Explain == nil {
			t.Errorf("%s: no Explain, so a missing file reads as a syscall", id)
		}
	}

	// The two with a flag must not also set a variable: a value delivered twice
	// is a value the child resolves by a precedence nobody chose.
	for _, id := range []string{OCIProfile, AzureSubscription} {
		if s, _ := ByID(id); s.Env != "" {
			t.Errorf("%s: Env = %q, but its connector declares a flag for the value", id, s.Env)
		}
	}
}
