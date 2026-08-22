// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package source

import (
	"context"
	"os"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/cockroachdb/errors"
)

// Everything here runs against a recorded inventory document, so none of it
// needs credentials, a network, or an installed provider.
// testdata/discover_local_container.json is not written by hand: it is the
// output of a real `cnspec discover local --discover container`, with the
// machine it named scrubbed. The rest are hand-built to carry the one thing
// their connector taught -- github's owner prefix, azure's display name -- and
// a fixture that omits the catch tests nothing.

// targetByID finds a declaration to exercise.
func targetByID(t *testing.T, id string) discoverTarget {
	t.Helper()
	for _, d := range discoverTargets {
		if d.id == id {
			return d
		}
	}
	t.Fatalf("no discovery target declared for %q", id)
	return discoverTarget{}
}

// fixtureRunner stands in for the child process: it records the command line
// it was handed and writes a recorded document to whatever path --output-full
// named. stdout is what the real command prints, which is the only thing that
// tells an empty answer from a refused one.
func fixtureRunner(t *testing.T, fixture, stdout string, runErr error) (runner, *[]string) {
	t.Helper()
	var got []string
	return func(_ context.Context, _ string, args ...string) ([]byte, error) {
		got = args
		if fixture != "" {
			data, err := os.ReadFile("testdata/" + fixture)
			if err != nil {
				t.Fatalf("cannot read the fixture: %v", err)
			}
			for i, a := range args {
				if a == "--output-full" && i+1 < len(args) {
					if err := os.WriteFile(args[i+1], data, 0o600); err != nil {
						t.Fatalf("cannot stage the fixture: %v", err)
					}
				}
			}
		}
		return []byte(stdout), runErr
	}, &got
}

const summary = "Discovered assets:\nalpine: 8\n"

// The document a real run produces, parsed as it actually arrives.
//
// It carries the property every other case here depends on: the asset that was
// connected to is in the document alongside its children. A picker that took
// every asset would offer the machine it was asked about as one of the things
// inside it.
func TestTheRecordedDocumentParses(t *testing.T) {
	containers := discoverTarget{
		connector: "local", target: "container",
		value: fromPlatformID("containers"),
	}
	run, _ := fixtureRunner(t, "discover_local_container.json", summary, nil)

	got, err := containers.fetch(context.Background(), run, nil)
	if err != nil {
		t.Fatalf("parsing a real discovery document failed: %v", err)
	}
	if len(got) != 4 {
		t.Fatalf("got %d containers from the recording, want 4: %v", len(got), got)
	}
	for _, v := range got {
		if len(v) != 64 {
			t.Errorf("%q is not a container id", v)
		}
	}

	// The host the command was pointed at has no container id, so it drops out
	// of the answer rather than being offered as one of its own containers.
	byName := discoverTarget{connector: "local", target: "container", value: fromName()}
	all, err := byName.fetch(context.Background(), run, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(all, "workstation") {
		t.Fatal("the recording no longer holds the connected root; the keep/value rules below are untested")
	}
}

func TestNamespacesAreTheClustersNotTheCluster(t *testing.T) {
	d := targetByID(t, DiscoverK8sNamespaces)
	run, args := fixtureRunner(t, "discover_k8s_namespaces.json", summary, nil)

	got, err := d.fetch(context.Background(), run, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"default", "kube-system", "payments"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("namespaces = %v, want %v", got, want)
	}
	if slices.Contains(got, "kind-demo") {
		t.Error("the cluster was offered as one of its own namespaces")
	}

	// --discover is never left to default to auto, which is the expensive one.
	line := strings.Join(*args, " ")
	if !strings.Contains(line, "--discover namespaces") {
		t.Errorf("the target was not named explicitly: %s", line)
	}
	if !strings.Contains(line, "--output-format json") {
		t.Errorf("the payload format was not pinned: %s", line)
	}
}

// github names a discovered repository "<org>/<repo>" and --repos matches the
// bare name, so emitting the display name would filter to nothing at all.
func TestGitHubReposEmitTheBareName(t *testing.T) {
	d := targetByID(t, DiscoverGitHubRepos)
	run, args := fixtureRunner(t, "discover_github_repos.json", summary, nil)

	got, err := d.fetch(context.Background(), run, []string{"p:0=org", "p:1=mondoohq"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"cnquery", "cnspec", "policy-bundles"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("repos = %v, want %v", got, want)
	}
	for _, v := range got {
		if strings.Contains(v, "/") {
			t.Errorf("%q still carries its owner prefix", v)
		}
	}
	// The organization and the user in the same document are not repositories.
	if slices.Contains(got, "mondoohq") || slices.Contains(got, "octocat") {
		t.Errorf("a non-repository reached the picker: %v", got)
	}
	// The scope reaches the child as the positional pair the connector wants.
	if line := strings.Join(*args, " "); !strings.Contains(line, "discover github org mondoohq") {
		t.Errorf("the organization did not scope the discovery: %s", line)
	}
}

// A subscription is filtered by id and by nothing else, so the display name
// the asset is called by would match no subscription at all.
func TestAzureSubscriptionsEmitTheIDNotTheName(t *testing.T) {
	d := targetByID(t, DiscoverAzureSubscriptions)
	run, _ := fixtureRunner(t, "discover_azure_subscriptions.json", summary, nil)

	got, err := d.fetch(context.Background(), run, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"0d7c6b5a-4e3f-2a1b-9c8d-7e6f5a4b3c2d",
		"f1f9a0d1-9c2e-4a5b-8f3e-1c2d3e4f5a6b",
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("subscriptions = %v, want %v", got, want)
	}
	for _, v := range got {
		if strings.Contains(v, " ") {
			t.Errorf("%q is a display name, not a subscription id", v)
		}
	}
	// The second entry carries no connection option, so it is only findable
	// through the platform id -- which is the fallback this pair needs.
}

// A team picker offers teams. The projects inside them are in the same
// document and are a different flag's business.
func TestVercelOffersTeamsOnly(t *testing.T) {
	d := targetByID(t, DiscoverSourceID("vercel", "teams"))
	run, _ := fixtureRunner(t, "discover_vercel_teams.json", summary, nil)

	got, err := d.fetch(context.Background(), run, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"team_3xKd9RvBnM", "team_7fLp2QmXyZ"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("teams = %v, want %v", got, want)
	}
	for _, v := range got {
		if strings.HasPrefix(v, "prj_") {
			t.Errorf("a project reached the team picker: %q", v)
		}
	}
}

// The rule this whole file is organised around.
//
// `cnspec discover` writes no asset file for an account with nothing in it and
// no asset file for a connection it refused -- and it exits 0 either way, so
// the exit code does not separate them. Only stdout does: the per-platform
// summary is printed once discovery has actually run.
func TestEmptyAndFailedDoNotLookTheSame(t *testing.T) {
	d := targetByID(t, DiscoverAzureSubscriptions)

	// Nothing there: discovery ran and found none.
	ran, _ := fixtureRunner(t, "", "Discovered assets:\n", nil)
	values, err := d.fetch(context.Background(), ran, nil)
	if err != nil {
		t.Errorf("an empty subscription list reported an error: %v", err)
	}
	if len(values) != 0 {
		t.Errorf("values from nothing: %v", values)
	}

	// Refused: the connector printed its usage and still exited 0.
	refused, _ := fixtureRunner(t, "", "Use the azure provider to query...\n\nUsage:\n", nil)
	if _, err := d.fetch(context.Background(), refused, nil); err == nil {
		t.Fatal("a refused connection reported an empty subscription list")
	} else if got := d.explain(err).Error(); !strings.Contains(got, "cnspec discover azure") {
		t.Errorf("explained as %q, want the command the user can run", got)
	}

	// Refused loudly: the child exited non-zero and its stderr came back.
	stderr := errors.New(`! provider flag shorthand already in use, ignoring shorthand conflicts-with=user flag=username shorthand=u
x failed to parse cli arguments error="rpc error: code = Unknown desc = a valid GitHub authentication is required, pass --token '<yourtoken>', set GITHUB_TOKEN environment variable or provide GitHub App credentials"`)
	loud, _ := fixtureRunner(t, "", "", stderr)
	_, err = d.fetch(context.Background(), loud, nil)
	if err == nil {
		t.Fatal("a failed child reported success")
	}
	got := d.explain(err).Error()
	if strings.Contains(got, "\n") {
		t.Errorf("a picker gets one line, got %q", got)
	}
	if strings.HasPrefix(got, "rpc error:") || strings.Contains(got, "desc = ") {
		t.Errorf("the gRPC envelope survived: %q", got)
	}
	if strings.HasPrefix(got, "x ") || strings.Contains(got, `error="`) {
		t.Errorf("the log decoration survived: %q", got)
	}
	if !strings.Contains(got, "GITHUB_TOKEN") {
		t.Errorf("the provider's own instruction was lost: %q", got)
	}
}

// A discovery that has not been told what it is discovering inside says which
// field to fill in, rather than answering for something else.
func TestScopeIsRequiredBeforeAsking(t *testing.T) {
	d := targetByID(t, DiscoverGitHubRepos)

	if _, err := d.args([]string{"p:0=repo", "p:1=mondoohq/cnspec"}); err == nil {
		t.Error("a single-repo scan was offered a list of the org's repositories")
	} else if !strings.Contains(err.Error(), "organization") {
		t.Errorf("explained as %q, want it to name what this needs", err)
	}

	if _, err := d.args([]string{"p:0=org"}); err == nil {
		t.Error("the organization was never asked for")
	} else if !strings.Contains(err.Error(), "first") {
		t.Errorf("explained as %q, want it to say what to fill in first", err)
	}

	args, err := d.args([]string{"p:0=org", "p:1=mondoohq"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(args, " ") != "org mondoohq" {
		t.Errorf("args = %v, want the positional pair the connector takes", args)
	}

	// An optional scope narrows the answer and never blocks it.
	projects := targetByID(t, DiscoverGitLabProjects)
	if args, err := projects.args(nil); err != nil || len(args) != 0 {
		t.Errorf("unscoped gitlab projects = %v, %v; want an unscoped run", args, err)
	}
	if args, err := projects.args([]string{"f:group=acme/platform"}); err != nil ||
		strings.Join(args, " ") != "--group acme/platform" {
		t.Errorf("scoped gitlab projects = %v, %v", args, err)
	}
}

// A source depends on a field by identity, and the dependency it declares has
// to be the one it actually reads -- otherwise the model caches one cluster's
// answer under another's key.
func TestNeedsMatchTheScope(t *testing.T) {
	for _, d := range discoverTargets {
		needs := d.needs()
		for _, s := range d.scope {
			if !slices.Contains(needs, s.need) {
				t.Errorf("%s: reads %q but does not declare it", d.id, s.need)
			}
		}
		if d.envFrom != "" && !slices.Contains(needs, d.envFrom) {
			t.Errorf("%s: takes %q through the environment but does not declare it", d.id, d.envFrom)
		}
		s, ok := ByID(d.id)
		if !ok {
			t.Errorf("%s: declared but never registered", d.id)
			continue
		}
		if strings.Join(s.Needs, ",") != strings.Join(needs, ",") {
			t.Errorf("%s: registered Needs %v, declared %v", d.id, s.Needs, needs)
		}
	}

	// The namespace list belongs to one cluster, and that cluster is chosen in
	// a flag field -- the identity form, not the bare name, because a bare
	// name is only shorthand for a flag and this one has to keep working when
	// the field moves.
	d := targetByID(t, DiscoverK8sNamespaces)
	if !slices.Contains(d.needs(), "f:context") {
		t.Errorf("namespaces do not depend on the cluster: %v", d.needs())
	}
}

// Discovery connects into everything it finds, so none of these may run when a
// form opens.
func TestDiscoverySourcesWaitToBeAskedFor(t *testing.T) {
	for _, d := range discoverTargets {
		s, ok := ByID(d.id)
		if !ok {
			t.Errorf("%s is not registered", d.id)
			continue
		}
		if s.Class != ClassPostConnection {
			t.Errorf("%s: class %v, want ClassPostConnection", d.id, s.Class)
		}
		if s.Cost != CostRemote || !Deferred(d.id) {
			t.Errorf("%s: cost %v, want CostRemote so the picker opens it", d.id, s.Cost)
		}
		if s.Tool != discoverTool || !strings.Contains(s.Activity, discoverTool) {
			t.Errorf("%s: %q does not name %q", d.id, s.Activity, discoverTool)
		}
	}
}

// The pairs that were considered and left out, and why.
//
// Each of these has a discovery target that sounds like it feeds a flag, and
// none of them does. Writing them down as a test rather than as a comment is
// what stops the next pass re-deriving the same plausible mapping: an invented
// flag mapping is the documented failure mode on this branch, and it survived
// review last time because it read perfectly well.
func TestExcludedPairsStayExcluded(t *testing.T) {
	excluded := []struct {
		connector, target, why string
	}{
		{"aws", "accounts", "--filters takes regions, tags and per-service ids; the provider's prefix allowlist has no account key, so a discovered account would be silently dropped"},
		{"alicloud", "accounts", "--filters takes regions and tags only, and there is no account flag"},
		{"digitalocean", "databases", "--filters takes regions and tags only, and there is no database flag"},
		{"oci", "tenancy", "--tenancy exists, but the tenancy is what you connect with rather than what you find, and the discovered asset's shape could not be verified"},
		{"gcp", "projects", "the answer's only destination is the positional the discovery is already scoped by, so the picker would replace what it needed to run"},
		{"databricks", "workspaces", "--host takes a workspace URL, which is not the workspace id the platform id carries; the mapping could not be verified"},
		{"hcp", "projects", "--project-id exists, but the project asset's id is composed at runtime and neither the provider metadata nor the binary shows where it lands"},
		{"snowflake", "databases", "snowflake declares the databases target and no flag that takes a database name"},
		{"mssql", "databases", "--database exists, but discovery needs the password, and this launcher does not put a secret on a child's command line"},
		{"mysqldb", "databases", "as mssql"},
		{"postgresdb", "databases", "as mssql"},
	}
	for _, e := range excluded {
		id := DiscoverSourceID(e.connector, e.target)
		if _, ok := ByID(id); ok {
			t.Errorf("%s was registered; it was excluded because %s", id, e.why)
		}
		for _, d := range discoverTargets {
			if d.connector == e.connector && d.target == e.target {
				t.Errorf("%s %s was declared; it was excluded because %s", e.connector, e.target, e.why)
			}
		}
	}
}

// The chosen cluster has to reach the child, and this is the one dimension of
// it that a double cannot check: the value travels in the environment, so a
// recorded command line proves nothing about whether it arrived.
//
// It is worth a real process because composing the two existing runner helpers
// silently drops the environment -- runnerWithEnv returns its base unwrapped
// when one is set, and the production base is never nil -- which would leave
// the namespace list answering for whatever cluster the ambient kubeconfig
// named. See discoverRun.
func TestTheEnvironmentReachesTheChild(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no /bin/sh to ask")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	run := discoverRun(nil, []string{"KUBECONFIG=/tmp/picked-cluster"})
	out, err := run(ctx, "/bin/sh", "-c", "printenv KUBECONFIG")
	if err != nil {
		t.Fatalf("could not ask the child what it was given: %v", err)
	}
	if got := strings.TrimSpace(string(out)); got != "/tmp/picked-cluster" {
		t.Errorf("the child saw KUBECONFIG=%q, want the chosen cluster's copy", got)
	}

	// A double is used as it stands, so a test never spawns anything.
	fake, _ := fixtureRunner(t, "", summary, nil)
	if discoverRun(fake, []string{"KUBECONFIG=/tmp/ignored"}) == nil {
		t.Error("an injected runner was discarded")
	}
}

// The k8s namespace picker that predates this file is left where it is.
//
// It answers the same question through kubectl rather than through discovery,
// it is what the shipped k8s form names, and its behaviour is asserted in
// three test files this one does not own. Registering the generic version
// under a second id is an append; replacing the old one would have been an
// edit to a form and to tests belonging to someone else, for no behaviour the
// user can see.
func TestTheKubectlNamespaceSourceIsUntouched(t *testing.T) {
	old, ok := ByID(K8sNamespace)
	if !ok {
		t.Fatal("the kubectl namespace source is gone")
	}
	if old.Tool != "kubectl" {
		t.Errorf("the kubectl source now asks %q", old.Tool)
	}
	generic, ok := ByID(DiscoverK8sNamespaces)
	if !ok {
		t.Fatal("the discovery namespace source is not registered")
	}
	if generic.Tool == old.Tool {
		t.Error("both namespace sources claim the same tool, so a wait cannot say which is running")
	}
	if Key(K8sNamespace, nil) == Key(DiscoverK8sNamespaces, nil) {
		t.Error("the two namespace sources share a cache entry")
	}
}
