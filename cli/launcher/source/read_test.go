// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package source

import (
	"context"
	"errors"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"
)

func eq(t *testing.T, got, want []string, what string) {
	t.Helper()
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("%s = %v, want %v", what, got, want)
	}
}

// Both AWS shared-config spellings have to work: ~/.aws/config prefixes
// sections with "profile ", ~/.aws/credentials does not. Machines commonly
// have only one of the two files.
func TestAWSProfiles(t *testing.T) {
	got := awsProfilesFrom("testdata/aws_config", "testdata/aws_credentials")
	eq(t, got, []string{"default", "prod", "sso-prod  (123456789012)", "staging"}, "profiles from both files")

	got = awsProfilesFrom("testdata/aws_config", "testdata/does-not-exist")
	eq(t, got, []string{"default", "prod", "sso-prod  (123456789012)"}, "profiles with no credentials file")

	got = awsProfilesFrom("testdata/nope", "testdata/aws_credentials")
	eq(t, got, []string{"default", "staging"}, "profiles with no config file")

	if got := awsProfilesFrom("", ""); len(got) != 0 {
		t.Errorf("expected no profiles with no paths, got %v", got)
	}
}

// The profile reader must never surface key material, whatever the file holds.
func TestAWSProfilesNeverReturnSecrets(t *testing.T) {
	for _, p := range awsProfilesFrom("testdata/aws_config", "testdata/aws_credentials") {
		if strings.Contains(p, "AKIA") || strings.Contains(p, "EXAMPLEKEY") || strings.Contains(p, "=") {
			t.Fatalf("profile list leaked credential material: %q", p)
		}
	}
}

// The current context sorts first because it is the one the user most likely
// means; the rest are alphabetical.
func TestKubeContexts(t *testing.T) {
	got := kubeContextsFrom("testdata/kubeconfig")
	want := []string{
		"arn:aws:eks:us-east-2:921877552404:cluster/patrick-container-escape-demo-azql-cluster",
		"aks-trial",
		"chris-dev",
	}
	eq(t, got, want, "kube contexts")

	if got := kubeContextsFrom("testdata/nope"); got != nil {
		t.Errorf("expected nil for a missing kubeconfig, got %v", got)
	}
	if got := kubeContextsFrom("testdata/aws_config"); len(got) != 0 {
		t.Errorf("expected nothing from a non-kubeconfig file, got %v", got)
	}
}

// `Host *` and `Host web-*` set defaults for other hosts; they are not targets.
func TestSSHHostsSkipWildcards(t *testing.T) {
	got := sshHostsFrom("testdata/ssh_config")
	eq(t, got, []string{"bastion.example.com", "jump", "prod-web"}, "ssh hosts")

	if got := sshHostsFrom("testdata/nope"); len(got) != 0 {
		t.Errorf("expected nothing for a missing ssh config, got %v", got)
	}
}

// Containers and images are enumerated by asking cnspec, not the docker CLI,
// so the launcher works wherever cnspec does and gets richer data: running
// containers sort ahead of stopped ones.
func TestDockerContainers(t *testing.T) {
	fake := func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return []byte(`[{"docker.containers":[
			{"names":["web"],"image":"nginx:1.27","state":"running"},
			{"names":["old-job"],"image":"alpine:3","state":"exited"},
			{"names":["db"],"image":"postgres:16","state":"running"}
		]}]`), nil
	}
	got, err := dockerContainers(context.Background(), fake)
	if err != nil {
		t.Fatal(err)
	}
	eq(t, got, []string{"db", "web", "old-job"}, "containers, running first")
}

func TestDockerImages(t *testing.T) {
	fake := func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return []byte(`[{"docker.images":[
			{"tags":["alpine:3","alpine:latest"]},
			{"tags":["<none>:<none>"]},
			{"tags":["ubuntu:22.04"]}
		]}]`), nil
	}
	got, err := dockerImages(context.Background(), fake)
	if err != nil {
		t.Fatal(err)
	}
	eq(t, got, []string{"alpine:3", "alpine:latest", "ubuntu:22.04"}, "image tags")
}

// A missing docker daemon, or any other query failure, must leave the field
// usable rather than error out.
func TestMQLSourcesFailSoft(t *testing.T) {
	broken := func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return nil, errors.New("connection refused")
	}
	// The error must come back too: an unreachable daemon and a host with no
	// containers are not the same thing, and the field says so differently.
	got, err := dockerContainers(context.Background(), broken)
	if len(got) != 0 || err == nil {
		t.Errorf("expected no containers and an error, got %v / %v", got, err)
	}
	if got, err := dockerImages(context.Background(), broken); len(got) != 0 || err == nil {
		t.Errorf("expected no images and an error, got %v / %v", got, err)
	}

	garbage := func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return []byte("not json"), nil
	}
	if got, err := dockerImages(context.Background(), garbage); len(got) != 0 || err == nil {
		t.Errorf("expected no images and an error from unparseable output, got %v / %v", got, err)
	}
}

// kubectl merges every file KUBECONFIG names; reading only the first hides
// clusters the user can see.
func TestKubeContextsMergeEveryFile(t *testing.T) {
	got := kubeContextsFrom("testdata/kubeconfig", "testdata/kubeconfig2")
	want := []string{
		// current-context comes from the first file that sets one, and leads.
		"arn:aws:eks:us-east-2:921877552404:cluster/patrick-container-escape-demo-azql-cluster",
		"aks-trial", "chris-dev", "staging",
	}
	eq(t, got, want, "merged contexts")

	// chris-dev appears in both files and must not be listed twice.
	seen := map[string]int{}
	for _, c := range got {
		seen[c]++
	}
	if seen["chris-dev"] != 1 {
		t.Errorf("chris-dev listed %d times, want 1", seen["chris-dev"])
	}
}

// A profile name rarely answers "which account is that?", so where the config
// already says, the picker shows it.
func TestAWSProfilesLabelTheirAccount(t *testing.T) {
	got := awsProfilesFrom("testdata/aws_config", "testdata/aws_credentials")
	if !slices.Contains(got, "sso-prod  (123456789012)") {
		t.Errorf("expected the sso profile labelled with its account, got %v", got)
	}
	// A profile with no declared account stays as it is.
	if !slices.Contains(got, "prod") {
		t.Errorf("expected plain profiles untouched, got %v", got)
	}
}

// Only the profile name is a command-line argument; the label is for reading.
func TestAWSProfileLabelIsStrippedBeforeUse(t *testing.T) {
	if got := awsProfileValue("sso-prod  (123456789012)"); got != "sso-prod" {
		t.Fatalf("awsProfileValue = %q, want %q", got, "sso-prod")
	}
	if got := awsProfileValue("plain"); got != "plain" {
		t.Fatalf("awsProfileValue = %q, want %q", got, "plain")
	}
	// The source declares the mapping, and a field attached to it emits
	// through that declaration. That the launcher's own form does so is
	// TestAWSProfileLabelNeverReachesArgv.
	s, ok := ByID(AWSProfile)
	if !ok || s.Emit == nil {
		t.Fatal("the aws profile source declares no Emit, so the label would reach argv")
	}
	if got := s.Emit("sso-prod  (123456789012)"); got != "sso-prod" {
		t.Fatalf("Emit = %q, want the bare profile name", got)
	}
}

// gcloud keeps its state the way kubectl does, so the project picker is a file
// read: no network, and no dependence on an auth token that has usually
// expired by the time anyone opens the launcher.
func TestGCPProjectsFromGcloudConfig(t *testing.T) {
	eq(t, gcpProjectsFrom("testdata/gcloud"),
		[]string{"attack-surface-scanner", "mondoo-staging"}, "configured projects")

	if got := gcpActiveProjectFrom("testdata/gcloud"); got != "attack-surface-scanner" {
		t.Errorf("active project = %q, want the one active_config names", got)
	}
	eq(t, gcpZonesFrom("testdata/gcloud"),
		[]string{"europe-west1-b", "us-central1-c"}, "configured zones")

	// No gcloud at all is not an error, just no suggestions.
	if got := gcpProjectsFrom("testdata/does-not-exist"); len(got) != 0 {
		t.Errorf("expected nothing without a gcloud config, got %v", got)
	}
	if got := gcpActiveProjectFrom("testdata/does-not-exist"); got != "" {
		t.Errorf("expected no active project, got %q", got)
	}
}

// The project list must carry no credential material, the same guarantee the
// AWS profile reader gives.
func TestGCPProjectsNeverReturnSecrets(t *testing.T) {
	for _, p := range gcpProjectsFrom("testdata/gcloud") {
		if strings.ContainsAny(p, "=@") {
			t.Fatalf("project list leaked a setting: %q", p)
		}
	}
}

// gcloud keeps configurations, not a project list, so the full list is a live
// call.
func TestGCPAllProjects(t *testing.T) {
	ok := func(ctx context.Context, name string, args ...string) ([]byte, error) {
		if name != "gcloud" {
			t.Errorf("expected gcloud, got %q", name)
		}
		return []byte("attack-surface-scanner\nmondoo-demo\n\nmondoo-staging\n"), nil
	}
	got, err := gcpAllProjects(context.Background(), ok)
	if err != nil {
		t.Fatal(err)
	}
	eq(t, got, []string{"attack-surface-scanner", "mondoo-demo", "mondoo-staging"}, "listed projects")
}

// gcloud's own message is four lines of prose around the one instruction that
// matters. A picker has room for a sentence, so it gets the instruction.
func TestGcloudErrorsSayWhatToDo(t *testing.T) {
	// Verbatim from `gcloud projects list` with an expired token.
	stale := `ERROR: (gcloud.projects.list) There was a problem refreshing your current auth tokens: Reauthentication failed. cannot prompt during non-interactive execution.
Please run:

  $ gcloud auth login

to obtain new credentials.`

	cases := []struct{ in, want string }{
		{stale, "gcloud auth login"},
		{"exec: \"gcloud\": executable file not found in $PATH", "not installed"},
		{"ERROR: (gcloud.projects.list) User does not have permission", "cannot list projects"},
	}
	for _, c := range cases {
		got := gcloudError(errors.New(c.in)).Error()
		if !strings.Contains(got, c.want) {
			t.Errorf("gcloudError(%.40q…) = %q, want it to mention %q", c.in, got, c.want)
		}
		if strings.Count(got, "\n") != 0 {
			t.Errorf("a picker gets one line, got %q", got)
		}
	}

	// Something unrecognised keeps gcloud's own first line, which beats a
	// message of ours that says nothing.
	got := gcloudError(errors.New("ERROR: (gcloud.projects.list) quota exceeded\nsee docs")).Error()
	if got != "(gcloud.projects.list) quota exceeded" {
		t.Errorf("unrecognised error = %q", got)
	}
}

// The chosen cluster has to reach the child process, and this is the one
// dimension of it a double cannot check: KUBECONFIG travels in the
// environment, so a recorded command line proves nothing about whether it
// arrived.
//
// It is worth a real process because the composition used to lose it silently.
// runnerWithEnv returned its base unwrapped whenever one was set, and the
// production path always sets one -- k8sNamespaces hands mqlQuery execRunner,
// mqlQuery hands it to runWithEnv, runWithEnv to runnerWithEnv -- so the
// KUBECONFIG that selects the cluster the user picked was dropped on every
// real call and kept on none. Nothing failed: the namespace picker simply
// answered for whatever cluster the ambient kubeconfig named.
func TestRunWithEnvReachesTheChild(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no /bin/sh to ask")
	}
	// Set in this process too, so this also proves the child's value wins over
	// an inherited one rather than sitting behind it.
	t.Setenv("KUBECONFIG", "/tmp/ambient-cluster")

	out, err := runWithEnv(context.Background(), execRunner,
		[]string{"KUBECONFIG=/tmp/chosen-cluster"},
		"/bin/sh", "-c", "printenv KUBECONFIG || echo MISSING")
	if err != nil {
		t.Fatalf("could not ask the child what it was given: %v", err)
	}
	if got := strings.TrimSpace(string(out)); got != "/tmp/chosen-cluster" {
		t.Errorf("the child saw KUBECONFIG=%q, want the chosen cluster's copy", got)
	}
}

// The seam stays a seam: a substituted runner spawns nothing, so the only way
// it can assert what it was asked to apply is to be told. It is told on the
// context, which is also how the real runner learns it.
func TestASubstitutedRunnerIsToldWhichEnvironmentToApply(t *testing.T) {
	var seen []string
	double := func(ctx context.Context, name string, args ...string) ([]byte, error) {
		seen = runEnvFrom(ctx)
		return []byte(`[{"k8s.namespaces":[{"name":"kube-system"}]}]`), nil
	}

	// Through the whole production shape, not just the last hop: this is the
	// call chain that used to drop it.
	got, err := k8sNamespaces(context.Background(), double, []string{"KUBECONFIG=/tmp/picked"})
	if err != nil {
		t.Fatal(err)
	}
	eq(t, got, []string{"kube-system"}, "namespaces")
	eq(t, seen, []string{"KUBECONFIG=/tmp/picked"}, "environment the runner was asked to apply")

	// And nothing is invented when there is nothing to apply.
	if _, err := k8sNamespaces(context.Background(), double, nil); err != nil {
		t.Fatal(err)
	}
	if len(seen) != 0 {
		t.Errorf("a query with no environment was given %v", seen)
	}
}

// gcloud is the one tool here that does not keep its configuration under a
// dotted directory in $HOME. Getting this wrong is silent: a directory that
// does not exist reads as a machine with no configurations, so the gcp pickers
// would offer nothing and say nothing.
func TestGcloudConfigDirIsPlatformAware(t *testing.T) {
	windows := func(name string) string {
		switch name {
		case "APPDATA":
			return `C:\Users\chris\AppData\Roaming`
		}
		return ""
	}
	if got, want := gcloudDirFor("windows", windows),
		filepath.Join(`C:\Users\chris\AppData\Roaming`, "gcloud"); got != want {
		t.Errorf("on Windows gcloudDir = %q, want %q", got, want)
	}

	// A Windows profile with no APPDATA falls back the way the SDK does, and
	// must not silently answer with a POSIX path that cannot exist there.
	if got := gcloudDirFor("windows", func(string) string { return "" }); strings.Contains(got, ".config") {
		t.Errorf("with no APPDATA gcloudDir = %q, which is the POSIX answer", got)
	}

	// Everywhere else is unchanged, and so are the aws, kube and ssh paths --
	// those tools do resolve ~ on Windows, and were not touched.
	if got, want := gcloudDirFor("darwin", func(string) string { return "" }),
		home(".config", "gcloud"); got != want {
		t.Errorf("gcloudDir = %q, want %q", got, want)
	}

	// CLOUDSDK_CONFIG relocates the whole thing, on every platform.
	env := func(name string) string {
		if name == "CLOUDSDK_CONFIG" {
			return "/opt/gcloud-config"
		}
		return `C:\Users\chris\AppData\Roaming`
	}
	for _, goos := range []string{"windows", "linux"} {
		if got := gcloudDirFor(goos, env); got != "/opt/gcloud-config" {
			t.Errorf("%s: CLOUDSDK_CONFIG was ignored, got %q", goos, got)
		}
	}
}

// Cancelling really does stop the work, rather than only stop waiting for it.
// This is the mechanism the picker relies on; the model's half is
// TestClosingAPickerStopsWhatItStarted.
func TestACancelledContextKillsTheChild(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no /bin/sh to ask")
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	if _, err := runnerWithEnv(nil, nil)(ctx, "/bin/sh", "-c", "sleep 30"); err == nil {
		t.Fatal("a cancelled run reported success")
	}
	if waited := time.Since(start); waited > 10*time.Second {
		t.Fatalf("the child outlived its context by %s", waited)
	}
}

// The shell above is allowed to exec the sleep, replacing itself, in which case
// killing the child is enough. A trailing command forces it to fork instead, so
// the sleep is a grandchild that survives its parent and keeps the output pipe
// open -- and Wait waits on the pipe, not on the process. macOS execs and Linux
// forks, which is why this only ever failed in CI.
func TestACancelledContextKillsAGrandchild(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no /bin/sh to ask")
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	if _, err := runnerWithEnv(nil, nil)(ctx, "/bin/sh", "-c", "sleep 30; :"); err == nil {
		t.Fatal("a cancelled run reported success")
	}
	if waited := time.Since(start); waited > 10*time.Second {
		t.Fatalf("the grandchild outlived its context by %s", waited)
	}
}
