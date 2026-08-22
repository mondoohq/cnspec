// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package source

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/cockroachdb/errors"

	"github.com/kevinburke/ssh_config"
	"gopkg.in/yaml.v3"
)

// Value pickers. These turn a text field into a list of the things actually
// present on this machine -- the AWS profiles you have, the clusters in your
// kubeconfig -- so the launcher can answer "which account?" instead of asking
// the user to remember.
//
// Every source is best-effort and non-fatal: a missing file or an absent binary
// yields no suggestions and the field stays free text. They read *names only*.
// None of them ever reads, holds or returns credential material.

// kubeCurrentContext reports the context kubectl would use.
func kubeCurrentContext() string {
	var paths []string
	if env := os.Getenv("KUBECONFIG"); env != "" {
		paths = filepath.SplitList(env)
	} else {
		paths = []string{home(".kube", "config")}
	}
	for _, path := range paths {
		if path == "" {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var cfg struct {
			CurrentContext string `yaml:"current-context"`
		}
		if yaml.Unmarshal(data, &cfg) == nil && cfg.CurrentContext != "" {
			return cfg.CurrentContext
		}
	}
	return ""
}

// home is a path under the user's home directory, or "" when there is no home
// to be under.
//
// It is exported because the readers here are not the only thing that has to
// find a dotted config directory -- the kubeconfig copy the launcher writes
// resolves the same ~/.kube/config -- and two spellings of "where the user's
// config lives" is how a picker and a launch end up talking about different
// files.
func home(rel ...string) string {
	dir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(append([]string{dir}, rel...)...)
}

// awsAccountKeys are the two spellings of the account an AWS profile names.
// They are one fact, so the file's last word on it is the answer -- see
// iniValues.last.
var awsAccountKeys = []string{"sso_account_id", "account_id"}

// awsProfiles lists the profile names in the shared AWS config files.
//
// It reads section headers and the account keys only, through the shared ini
// scanner: ~/.aws/credentials holds live access keys, and there is no reason
// for this process to pull them into memory to answer "what are the profiles
// called". See ini.go.
func awsProfiles() []string {
	return awsProfilesFrom(home(".aws", "config"), home(".aws", "credentials"))
}

func awsProfilesFrom(configPath, credentialsPath string) []string {
	profiles, accounts := awsProfileSections(configPath, credentialsPath)

	out := make([]string, 0, len(profiles))
	for _, p := range profiles {
		// Where the config already names the account -- SSO profiles carry
		// sso_account_id -- show it, because "which account is that?" is the
		// question a profile name rarely answers. Resolving the rest would
		// mean an STS call per profile, which is not something a picker should
		// do behind the user's back.
		if id := accounts[p].last(awsAccountKeys...); id != "" && !strings.Contains(p, id) {
			out = append(out, p+"  ("+id+")")
			continue
		}
		out = append(out, p)
	}
	return out
}

// awsProfileValue strips the account annotation back off, so what reaches the
// command line is the profile name AWS knows.
func awsProfileValue(display string) string {
	if i := strings.Index(display, "  ("); i > 0 {
		return display[:i]
	}
	return display
}

// awsProfileSections reads profile names, and any account id declared beside
// them, from the shared config files.
//
// The two files are one namespace: ~/.aws/config spells a profile
// `[profile prod]`, ~/.aws/credentials spells the same one `[prod]`, and a
// machine commonly has only one of the two. Both are therefore scanned into a
// single set of names, and the account id is whichever of the two keys either
// file stated last.
func awsProfileSections(configPath, credentialsPath string) ([]string, map[string]iniValues) {
	names, accounts, _ := iniSections(iniScan{
		files: []iniFile{
			{path: configPath, strip: "profile "},
			{path: credentialsPath},
		},
		want: map[string]bool{"sso_account_id": true, "account_id": true},
	})
	// A read failure is not reported: an absent ~/.aws is a machine with no
	// profiles, and the field stays free text either way.
	sort.Strings(names)
	return names, accounts
}

// kubeContexts lists the context names in the active kubeconfig, current one
// first. The file is read with a minimal schema rather than through client-go's
// loader, which would pull a very large dependency in for two field reads.
func kubeContexts() []string {
	// KUBECONFIG lists files the way PATH lists directories, and kubectl merges
	// all of them. Reading only the first hides clusters the user can see.
	var paths []string
	if env := os.Getenv("KUBECONFIG"); env != "" {
		paths = filepath.SplitList(env)
	} else {
		paths = []string{home(".kube", "config")}
	}
	return kubeContextsFrom(paths...)
}

func kubeContextsFrom(paths ...string) []string {
	var names []string
	var current string
	seen := map[string]bool{}

	for _, path := range paths {
		if path == "" {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var cfg struct {
			CurrentContext string `yaml:"current-context"`
			Contexts       []struct {
				Name string `yaml:"name"`
			} `yaml:"contexts"`
		}
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			continue
		}
		// kubectl takes current-context from the first file that sets one.
		if current == "" {
			current = cfg.CurrentContext
		}
		for _, c := range cfg.Contexts {
			if c.Name != "" && !seen[c.Name] {
				seen[c.Name] = true
				names = append(names, c.Name)
			}
		}
	}

	sort.Strings(names)
	if current != "" && seen[current] {
		// The current context is the one most likely meant, so it leads.
		out := make([]string, 0, len(names))
		out = append(out, current)
		for _, n := range names {
			if n != current {
				out = append(out, n)
			}
		}
		return out
	}
	return names
}

// sshHosts lists the Host aliases in the SSH client config. Wildcard patterns
// are skipped: `Host *` sets defaults for every host and is not itself a target.
func sshHosts() []string {
	return sshHostsFrom(home(".ssh", "config"))
}

func sshHostsFrom(path string) []string {
	if path == "" {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	cfg, err := ssh_config.Decode(f)
	if err != nil {
		return nil
	}
	var out []string
	for _, host := range cfg.Hosts {
		for _, pattern := range host.Patterns {
			p := pattern.String()
			if p == "" || strings.ContainsAny(p, "*?!") {
				continue
			}
			out = append(out, p)
		}
	}
	return SortedUnique(out)
}

// gcloud keeps its state the way kubectl does: a pointer to the active
// configuration and one file per configuration. So the project picker is a
// file read, like the kube contexts and the AWS profiles, and needs neither the
// network nor working credentials.
//
// Enumerating every project the account can see is a different matter --
// `gcloud projects list` is an API call that fails outright when the auth token
// has expired, which on a developer machine is the usual state. The launcher
// therefore offers what is configured locally and lets anything else be typed.

// gcloudDir is the directory gcloud keeps its configurations in.
//
// It is the one config path in this file that is not simply a dotted directory
// under $HOME. The aws, kubectl and ssh readers all use ~/.aws, ~/.kube and
// ~/.ssh on Windows exactly as they do elsewhere -- those tools resolve $HOME
// themselves -- so those stay as they are. gcloud does not: on Windows it puts
// its configuration under %APPDATA%, which is where `gcloud config
// configurations list` reads from and where ~/.config/gcloud does not exist.
// Looking in the wrong place there is silent, because a missing directory is
// indistinguishable from a machine with no configurations.
func gcloudDir() string { return gcloudDirFor(runtime.GOOS, os.Getenv) }

// gcloudDirFor is gcloudDir with the platform and the environment injected, so
// the Windows answer is checkable from any machine. It follows the resolution
// in the SDK's own core/config.py (_GetGlobalConfigDir).
func gcloudDirFor(goos string, getenv func(string) string) string {
	// CLOUDSDK_CONFIG relocates the whole directory, and does so on every
	// platform, so it is checked before anything else.
	if dir := getenv("CLOUDSDK_CONFIG"); dir != "" {
		return dir
	}
	if goos != "windows" {
		return home(".config", "gcloud")
	}
	if appdata := getenv("APPDATA"); appdata != "" {
		return filepath.Join(appdata, "gcloud")
	}
	// gcloud's own last resort, for a Windows profile that carries no APPDATA.
	drive := getenv("SystemDrive")
	if drive == "" {
		drive = "C:"
	}
	return filepath.Join(drive, string(filepath.Separator), "gcloud")
}

// gcpConfigurations returns each gcloud configuration's name and the settings
// the launcher cares about.
func gcpConfigurations(dir string) map[string]map[string]string {
	out := map[string]map[string]string{}
	entries, err := os.ReadDir(filepath.Join(dir, "configurations"))
	if err != nil {
		return out
	}
	for _, e := range entries {
		name, ok := strings.CutPrefix(e.Name(), "config_")
		if !ok || e.IsDir() {
			continue
		}
		f, err := os.Open(filepath.Join(dir, "configurations", e.Name()))
		if err != nil {
			continue
		}
		settings := map[string]string{}
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			key, value, ok := strings.Cut(sc.Text(), "=")
			if !ok {
				continue
			}
			settings[strings.TrimSpace(key)] = strings.TrimSpace(value)
		}
		f.Close()
		out[name] = settings
	}
	return out
}

// gcpActiveConfig is the configuration gcloud would use.
func gcpActiveConfig(dir string) string {
	data, err := os.ReadFile(filepath.Join(dir, "active_config"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func gcpProjects() []string { return gcpProjectsFrom(gcloudDir()) }

func gcpProjectsFrom(dir string) []string {
	var out []string
	for _, settings := range gcpConfigurations(dir) {
		if p := settings["project"]; p != "" {
			out = append(out, p)
		}
	}
	return SortedUnique(out)
}

// GCPActiveProject is the project of the active configuration.
func GCPActiveProject() string { return gcpActiveProjectFrom(gcloudDir()) }

func gcpActiveProjectFrom(dir string) string {
	active := gcpActiveConfig(dir)
	if active == "" {
		return ""
	}
	return gcpConfigurations(dir)[active]["project"]
}

func gcpZones() []string { return gcpZonesFrom(gcloudDir()) }

// gcpZonesFrom offers the zones the configurations name. It is a suggestion,
// not a list of valid zones: a zone gcloud has never been told about is still
// perfectly scannable, so the field stays free text.
func gcpZonesFrom(dir string) []string {
	var out []string
	for _, settings := range gcpConfigurations(dir) {
		if z := settings["zone"]; z != "" {
			out = append(out, z)
		}
	}
	return SortedUnique(out)
}

// gcpAllProjects lists every project the signed-in account can see.
//
// This is the GCP counterpart of the kubeconfig read, and it has to be a live
// call because there is no local equivalent: gcloud keeps named configurations,
// not a project list, so the config gives one project where a kubeconfig gives
// every cluster you have ever fetched credentials for.
//
// It goes through gcloud rather than MQL because `cnspec run gcp` needs a scope
// before it will connect, and `gcp.projects` comes back empty from a
// project-scoped connection -- listing projects needs an organization id, which
// is the thing the user is usually trying to avoid having to know. gcloud
// already holds the credentials for this, and the same shape is used elsewhere
// in cnspec to enumerate Azure subscriptions.
func gcpAllProjects(ctx context.Context, run runner) ([]string, error) {
	// The timeout is a backstop on a call that can hang; the context under it
	// comes from the picker, so closing the picker kills the gcloud this
	// started rather than leaving it to run to completion unwatched.
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	out, err := run(ctx, "gcloud", "projects", "list", "--format=value(projectId)")
	if err != nil {
		return nil, gcloudError(err)
	}
	var projects []string
	for _, line := range strings.Split(string(out), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			projects = append(projects, line)
		}
	}
	return SortedUnique(projects), nil
}

// gcloudError turns what gcloud printed into something worth acting on.
//
// Its own message is four lines of prose wrapped around the one instruction
// that matters, and a picker has room for a sentence. Everything it says about
// refreshing tokens and non-interactive execution amounts to: log in again.
func gcloudError(err error) error {
	text := err.Error()
	switch {
	case strings.Contains(text, "Reauthentication failed"),
		strings.Contains(text, "problem refreshing"),
		strings.Contains(text, "gcloud auth login"),
		strings.Contains(text, "invalid_grant"):
		return errors.New("gcloud needs signing in again — run: gcloud auth login")
	case strings.Contains(text, "executable file not found"),
		strings.Contains(text, "no such file"):
		return errors.New("gcloud is not installed — type a project id instead")
	case strings.Contains(text, "does not have permission"),
		strings.Contains(text, "PERMISSION_DENIED"):
		return errors.New("this account cannot list projects — type a project id instead")
	}
	// Anything unrecognised keeps gcloud's own first line, which is more use
	// than a message of ours that says nothing.
	if line, _, _ := strings.Cut(text, "\n"); line != "" {
		return errors.New(strings.TrimPrefix(line, "ERROR: "))
	}
	return err
}

// runner runs a command and returns its stdout. Injectable so the MQL-backed
// sources are testable without spawning anything.
type runner func(ctx context.Context, name string, args ...string) ([]byte, error)

// The environment a runner has to apply travels on the context rather than in
// the signature, and that is a deliberate choice rather than a shortcut.
//
// The obvious shape -- wrap the runner and let the wrapper set cmd.Env -- is
// what this code used to do, and it dropped the environment on the floor for
// every caller that supplied a base runner. Production always supplies one
// (k8sNamespaces -> mqlQuery -> runWithEnv -> runnerWithEnv(execRunner, env)),
// so the KUBECONFIG that selects the chosen cluster never reached the child
// and the namespace picker answered for whatever cluster the ambient
// kubeconfig named. Nothing failed; the answer was simply about a different
// cluster.
//
// Putting it on the context fixes that in the one place it can be fixed for
// every runner at once: whoever finally spawns the process reads it back, and
// a substituted runner -- which spawns nothing and therefore cannot be
// observed through a process -- can read the same value and assert on it. See
// TestTheEnvironmentReachesTheChild and TestASubstitutedRunnerSeesTheEnv.
type runEnvKey struct{}

// withRunEnv adds environment entries for the eventual child. Entries already
// on the context are kept, and later ones win the way os/exec resolves a
// duplicated key.
func withRunEnv(ctx context.Context, env []string) context.Context {
	if len(env) == 0 {
		return ctx
	}
	prev := runEnvFrom(ctx)
	merged := make([]string, 0, len(prev)+len(env))
	merged = append(merged, prev...)
	merged = append(merged, env...)
	return context.WithValue(ctx, runEnvKey{}, merged)
}

// runEnvFrom reports the environment a runner has been asked to apply.
func runEnvFrom(ctx context.Context) []string {
	env, _ := ctx.Value(runEnvKey{}).([]string)
	return env
}

// execRunner is the real thing: it spawns the process and applies whatever
// environment the context carries.
func execRunner(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	// Cancelling has to stop the work, not just stop waiting for it. See
	// proc_unix.go: the child gets its own process group and Cancel signals
	// the group, and WaitDelay bounds the case where something still holds
	// the output pipe after the group is gone.
	setProcessGroup(cmd)
	cmd.Cancel = func() error { return killTree(cmd) }
	cmd.WaitDelay = 2 * time.Second
	// The launcher is already the current binary. Letting the child hand
	// off to whatever release sits in the auto-update cache costs a second
	// and answers the question from a different version than the one being
	// used.
	cmd.Env = append(os.Environ(), "MONDOO_AUTO_UPDATE=false")
	// os/exec keeps the last entry for a repeated key, so this overrides an
	// inherited KUBECONFIG rather than sitting behind it.
	cmd.Env = append(cmd.Env, runEnvFrom(ctx)...)
	out, err := cmd.Output()
	if err != nil {
		// The reason a CLI refused is on stderr, and .Output() files it
		// away on the error rather than returning it. Without this a
		// picker can only report that something went wrong.
		var exit *exec.ExitError
		if errors.As(err, &exit) && len(exit.Stderr) > 0 {
			return out, errors.New(strings.TrimSpace(string(exit.Stderr)))
		}
	}
	return out, err
}

// runnerWithEnv builds a runner that applies env to whatever eventually runs.
// base is used when set, so tests can substitute their own; a nil base is the
// real child process.
func runnerWithEnv(base runner, env []string) runner {
	if base == nil {
		base = execRunner
	}
	return func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return base(withRunEnv(ctx, env), name, args...)
	}
}

// mqlQuery runs an MQL query against the local system and returns the raw JSON.
//
// Asking cnspec rather than the tool underneath it is what makes this
// generalize: the same call shape enumerates anything MQL can express, on any
// platform where cnspec already works, without the launcher growing a
// dependency on each vendor's CLI. It costs a subprocess -- about 0.7s -- so it
// only ever runs from a tea.Cmd.
func mqlQuery(ctx context.Context, run runner, connector, query string, env []string) ([]byte, error) {
	self, err := os.Executable()
	if err != nil {
		self = os.Args[0]
	}
	// A picker wants one resource, not an inventory. Without --discover none
	// the query is run against every discovered asset, which on a real cluster
	// meant 36 seconds instead of 6.
	//
	// The timeout is a backstop, not the only way out: the context comes from
	// whoever opened the picker, so pressing esc kills this child instead of
	// abandoning the answer and leaving it running.
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	// The child must not hand off to a newer installed binary mid-query.
	return runWithEnv(ctx, run, env, self,
		"run", connector, "--discover", "none", "-c", query, "-j")
}

// runWithEnv threads extra environment into a runner. Sources that target a
// specific cluster do it this way, because the connector's own --context flag
// does not select one (see kubeconfig.go).
func runWithEnv(ctx context.Context, run runner, env []string, name string, args ...string) ([]byte, error) {
	return runnerWithEnv(run, env)(ctx, name, args...)
}

// dockerContainers lists the containers on this host, running ones first,
// labelled with the image they came from.
func dockerContainers(ctx context.Context, run runner) ([]string, error) {
	data, err := mqlQuery(ctx, run, "local", "docker.containers { names image state }", nil)
	if err != nil {
		return nil, err
	}
	var rows []struct {
		Names []string `json:"names"`
		Image string   `json:"image"`
		State string   `json:"state"`
	}
	if err := decodeMQLList(data, "docker.containers", &rows); err != nil {
		return nil, err
	}

	var running, stopped []string
	for _, r := range rows {
		if len(r.Names) == 0 {
			continue
		}
		if r.State == "running" {
			running = append(running, r.Names[0])
			continue
		}
		stopped = append(stopped, r.Names[0])
	}
	sort.Strings(running)
	sort.Strings(stopped)
	return append(running, stopped...), nil
}

// dockerImages lists the image references held on this host.
func dockerImages(ctx context.Context, run runner) ([]string, error) {
	data, err := mqlQuery(ctx, run, "local", "docker.images { tags }", nil)
	if err != nil {
		return nil, err
	}
	var rows []struct {
		Tags []string `json:"tags"`
	}
	if err := decodeMQLList(data, "docker.images", &rows); err != nil {
		return nil, err
	}
	var out []string
	for _, r := range rows {
		for _, tag := range r.Tags {
			// An untagged image shows as "<none>:<none>" and cannot be scanned
			// by reference.
			if tag != "" && !strings.Contains(tag, "<none>") {
				out = append(out, tag)
			}
		}
	}
	return SortedUnique(out), nil
}

// k8sNamespaces lists the namespaces of the cluster the env targets.
func k8sNamespaces(ctx context.Context, run runner, env []string) ([]string, error) {
	data, err := mqlQuery(ctx, run, "k8s", "k8s.namespaces { name }", env)
	if err != nil {
		return nil, err
	}
	var rows []struct {
		Name string `json:"name"`
	}
	if err := decodeMQLList(data, "k8s.namespaces", &rows); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		if r.Name != "" {
			out = append(out, r.Name)
		}
	}
	return SortedUnique(out), nil
}

// decodeMQLList pulls one resource's rows out of `cnspec run -j` output, which
// is an array of result objects keyed by the queried resource.
func decodeMQLList(data []byte, resource string, out any) error {
	var envelope []map[string]json.RawMessage
	if err := json.Unmarshal(data, &envelope); err != nil {
		return err
	}
	for _, entry := range envelope {
		if raw, ok := entry[resource]; ok {
			return json.Unmarshal(raw, out)
		}
	}
	return errors.New("no " + resource + " in the query result")
}

func SortedUnique(in []string) []string {
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}
