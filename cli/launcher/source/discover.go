// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package source

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
)

// Post-connection discovery: what is inside a target, asked through cnspec's
// own discovery rather than through each vendor's CLI.
//
// A connector that can enumerate its own contents already has the answer the
// launcher wants -- k8s knows its namespaces, github knows an organization's
// repositories, azure knows its subscriptions -- and it also declares a flag
// that consumes that answer back. `cnspec discover <connector> --discover
// <target>` is the one call that covers all of them, so this file is one
// implementation and a table, rather than a picker per provider. The bespoke
// k8s namespace reader in sources_pre.go is what this replaces the *pattern*
// of; see the note on DiscoverK8sNamespaces below for why that one source
// stays where it is.
//
// Three properties of the underlying command shape everything here, and each
// was established by running it rather than by reading about it:
//
//   - The payload flag is --output-full <path>, and it is a path, not a
//     stream: there is no stdout mode for the assets. So the source writes to
//     a temp file and reads it back. -f/--output-format json emits one
//     inventory document, {"spec":{"assets":[...]}}, credentials redacted.
//   - --discover is always passed explicitly. The default is `auto`, which is
//     the expensive one -- providers gate on the target before enumerating.
//   - `cnspec discover` connects into every discovered child, so this is slow
//     even locally: `discover local --discover container` measured 3.9s
//     against 0.6s for the bare host. Every source here is therefore
//     CostRemote, and runs only when its picker is opened.
//
// And one that decides the whole empty-versus-failed story: **a refused
// connection exits 0**. Asking github for repositories with no token prints
// the connector's usage to stdout, writes the real reason to stderr, and
// returns success. The output file is missing in that case -- but it is also
// missing when the answer is genuinely nothing, because the command skips
// writing when it discovered no assets. The two are told apart by stdout:
// printPlatformSummary runs unconditionally once discovery has actually
// happened, so its header is present for a real empty answer and absent for a
// refusal. Without that distinction a picker showing nothing cannot say
// whether the account is empty or unreachable, which is the single
// most-repeated bug in this package's history.

// discoveryMarker is what the child prints once discovery has run, including
// when it discovered nothing. Its absence means the run never got that far.
const discoveryMarker = "Discovered assets:"

// discoverTool names what the wait is waiting for. It is the command the user
// can run themselves, which is the point of naming a tool at all.
const discoverTool = "cnspec discover"

// discoverTimeout bounds the child. It is longer than the MQL query timeout
// because discovery connects into every child it finds, and a large
// organization legitimately takes minutes.
const discoverTimeout = 2 * time.Minute

// discoveredAsset is the part of an inventory asset a picker needs, and only
// that part: the document also carries platform detail, capabilities and
// relationships, none of which decides what goes on a command line. The
// encoding is protojson with UseProtoNames, so the field names are the proto
// ones -- platform_ids, not platformIds.
type discoveredAsset struct {
	Name        string   `json:"name"`
	ID          string   `json:"id"`
	PlatformIDs []string `json:"platform_ids"`
	Platform    struct {
		Name string `json:"name"`
	} `json:"platform"`
	Connections []struct {
		Options map[string]string `json:"options"`
	} `json:"connections"`
}

// assetValue pulls the value that goes on the command line out of one
// discovered asset. It returns "" for an asset that carries none, which is
// also how the root asset -- always present in the document, never a child --
// is dropped.
type assetValue func(a discoveredAsset) string

// fromOption reads a connection option.
//
// This is the most reliable of the three, because it is the provider's own
// round trip: ParseCLI maps --group to Options["group"], and discovery hands
// each child a cloned config with that same option set. Where the names line
// up, reading the option back is reading exactly what the flag would write.
// They do not always line up -- github's --repos lands in Options["repository"]
// -- so the key is declared per target rather than derived from the flag.
func fromOption(keys ...string) assetValue {
	return func(a discoveredAsset) string {
		for _, c := range a.Connections {
			for _, k := range keys {
				if v := c.Options[k]; v != "" {
					return v
				}
			}
		}
		return ""
	}
}

// fromPlatformID reads the segment following a marker in an asset's platform
// id -- the <ID> of //platformid.api.mondoo.app/runtime/neon/organization/<ID>.
//
// This is the fallback for the providers whose scoping flag takes an id: the
// platform id is where that id is guaranteed to be, whereas the asset's name
// is a display name and the connection option may not carry it at all.
func fromPlatformID(marker string) assetValue {
	sep := "/" + marker + "/"
	return func(a discoveredAsset) string {
		for _, id := range append([]string{a.ID}, a.PlatformIDs...) {
			i := strings.Index(id, sep)
			if i < 0 {
				continue
			}
			rest := id[i+len(sep):]
			if cut, _, found := strings.Cut(rest, "/"); found {
				rest = cut
			}
			if rest != "" {
				return rest
			}
		}
		return ""
	}
}

// fromName is the asset's own name, for the targets whose filter flag matches
// on it -- a Kubernetes namespace is filtered by name, not by uid.
func fromName() assetValue {
	return func(a discoveredAsset) string { return a.Name }
}

// nameAfterSlash drops an owner prefix. github names a discovered repository
// "<org>/<repo>" while --repos matches on the bare repository name.
func nameAfterSlash() assetValue {
	return func(a discoveredAsset) string {
		if i := strings.LastIndex(a.Name, "/"); i >= 0 {
			return a.Name[i+1:]
		}
		return a.Name
	}
}

// firstValue takes the first extractor that yields anything.
func firstValue(vs ...assetValue) assetValue {
	return func(a discoveredAsset) string {
		for _, v := range vs {
			if got := v(a); got != "" {
				return got
			}
		}
		return ""
	}
}

// platformNamed keeps only the assets of a declared platform.
//
// The names come from the provider's own metadata -- the Platforms block of
// ~/.config/mondoo/providers/<name>/<name>.json -- so this is a check against
// what the provider says it emits, not against a guess. It matters because the
// document always contains the asset that was connected to as well as its
// children, and a cluster is not one of its own namespaces.
func platformNamed(names ...string) func(discoveredAsset) bool {
	return func(a discoveredAsset) bool {
		for _, n := range names {
			if a.Platform.Name == n {
				return true
			}
		}
		return false
	}
}

// discoverScope is one thing the discovery has to be pointed at before it
// means anything: the cluster whose namespaces these are, the organization
// whose repositories these are.
//
// The need is a field identity -- "p:1", "f:group" -- because every
// interesting case here is keyed off a positional, which a flag name cannot
// name. A positional scope carries no flag and is emitted in declaration
// order, so it is always required: a hole in the middle of a positional list
// would silently shift the arguments after it.
type discoverScope struct {
	// need names the field whose value scopes this discovery.
	need string
	// flag is the flag the value travels in. Empty means a positional.
	flag string
	// label is how the field is described when it has to be filled in first.
	label string
	// optional marks a scope that narrows the answer rather than enabling it.
	optional bool
	// equals gates the whole source on a selector's value: github lists
	// repositories for an organization and for nothing else.
	equals string
	// unless is what to say when equals does not hold.
	unless string
}

// discoverTarget declares one (connector, discovery target) pair and where its
// answer lands.
type discoverTarget struct {
	// id is the source id; see source_ids.go.
	id string
	// connector and target are the two halves of
	// `cnspec discover <connector> --discover <target>`.
	connector, target string
	// flag is the connector flag this answer is picked into. It is recorded
	// rather than used, because the form owns the binding -- but a target
	// whose answer has nowhere to go is a picker that wastes a network call,
	// and TestEveryDiscoveryTargetHasSomewhereToPutIt checks this against the
	// connector snapshot.
	flag string
	// activity says what is happening while it happens.
	activity string
	// scope is what has to be known before the question can be asked.
	scope []discoverScope
	// envFrom names a field whose value reaches the child through the
	// environment instead of the command line, and envApply builds it. k8s is
	// the case: --context is parsed and never reaches the client config, so
	// the cluster travels as a kubeconfig copy.
	envFrom  string
	envApply func(value string) (env []string, cleanup func(), err error)
	// keep decides which discovered assets are candidates.
	keep func(discoveredAsset) bool
	// value extracts the command-line value from one of them.
	value assetValue
}

// needs is every field this target reads, derived from the declaration so the
// two cannot drift apart.
func (d discoverTarget) needs() []string {
	out := make([]string, 0, len(d.scope)+1)
	for _, s := range d.scope {
		out = append(out, s.need)
	}
	if d.envFrom != "" {
		out = append(out, d.envFrom)
	}
	return out
}

// commandLine is what the user would type to see this for themselves. It names
// no values, only the shape, because it goes into an error message.
func (d discoverTarget) commandLine() string {
	return "cnspec discover " + d.connector + " --discover " + d.target
}

// args turns the scope into the child's arguments, or explains what is still
// missing. Both halves matter: a picker that quietly discovers the wrong
// thing is worse than one that says which field to fill in first.
func (d discoverTarget) args(params []string) ([]string, error) {
	var args []string
	for _, s := range d.scope {
		v := paramValue(params, s.need)
		if s.equals != "" && v != s.equals {
			return nil, errors.New(s.unless)
		}
		if v == "" {
			if s.optional {
				continue
			}
			return nil, errors.New("pick " + s.label + " first")
		}
		if s.flag == "" {
			args = append(args, v)
			continue
		}
		args = append(args, "--"+s.flag, v)
	}
	return args, nil
}

// discoverRun picks the runner for a child that may need extra environment.
//
// It exists because composing the two existing helpers loses the environment:
// runWithEnv hands runnerWithEnv a base runner, and runnerWithEnv returns the
// base unwrapped when one is set -- so `runWithEnv(ctx, execRunner, env, ...)`
// runs the child with no env at all. execRunner is itself a runner, so the
// production path takes that branch every time, and the value a source went to
// the trouble of computing never reaches the process that needed it. For the
// namespace picker that is KUBECONFIG, which is the *only* thing that selects
// a cluster, so the answer would come back for whichever cluster the ambient
// config happened to name.
//
// A nil base here means "the real thing", which is the one case that needs the
// environment; a base is a test double, which never spawns anything and cannot
// observe it. Fixing runnerWithEnv itself is the right repair and belongs to
// the file that owns it.
func discoverRun(base runner, env []string) runner {
	if base != nil {
		return base
	}
	return runnerWithEnv(nil, env)
}

// fetch runs the discovery and returns the candidates. A nil run is the real
// child process; tests pass a double.
//
// The caller's context is what makes a picker abandonable: `cnspec discover`
// connects into every asset it finds, so closing the picker has to kill it
// rather than leave it running to completion for an answer nobody will read.
func (d discoverTarget) fetch(ctx context.Context, run runner, params []string) ([]string, error) {
	args, err := d.args(params)
	if err != nil {
		return nil, err
	}

	var env []string
	if d.envApply != nil {
		if v := paramValue(params, d.envFrom); v != "" {
			more, cleanup, err := d.envApply(v)
			if cleanup != nil {
				defer cleanup()
			}
			if err != nil {
				return nil, err
			}
			env = more
		}
	}

	// The assets have nowhere to go but a file, so one is made here and
	// removed on the way out. The child truncates it; an unwritable temp dir
	// is the only way this fails.
	out, err := os.CreateTemp("", "cnspec-discover-*.json")
	if err != nil {
		return nil, errors.Wrap(err, "cannot make room for the discovery output")
	}
	path := out.Name()
	_ = out.Close()
	defer func() { _ = os.Remove(path) }()

	self, err := os.Executable()
	if err != nil {
		self = os.Args[0]
	}

	ctx, cancel := context.WithTimeout(ctx, discoverTimeout)
	defer cancel()

	full := append([]string{"discover", d.connector}, args...)
	full = append(full,
		"--discover", d.target,
		"--output-full", path,
		"--output-format", "json",
	)

	stdout, runErr := discoverRun(run, env)(ctx, self, full...)

	data, _ := os.ReadFile(path)
	if len(bytes.TrimSpace(data)) == 0 {
		// See the file comment: no file is both "nothing there" and "never
		// got that far", and only stdout tells them apart.
		if runErr == nil && bytes.Contains(stdout, []byte(discoveryMarker)) {
			return nil, nil
		}
		if runErr != nil {
			return nil, runErr
		}
		return nil, errors.New(d.commandLine() + " did not get far enough to answer")
	}
	return d.parse(data)
}

// parse pulls the candidates out of the inventory document discovery wrote.
func (d discoverTarget) parse(data []byte) ([]string, error) {
	var doc struct {
		Spec struct {
			Assets []discoveredAsset `json:"assets"`
		} `json:"spec"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, errors.Wrap(err, "cannot read what "+discoverTool+" wrote")
	}
	var out []string
	for _, a := range doc.Spec.Assets {
		if d.keep != nil && !d.keep(a) {
			continue
		}
		if v := d.value(a); v != "" {
			out = append(out, v)
		}
	}
	if len(out) == 0 {
		// An empty list from a document that parsed is a real answer: the
		// connector has none of these. Returning nil rather than an empty
		// slice keeps that indistinguishable from the marker path above,
		// which means the same thing.
		return nil, nil
	}
	return SortedUnique(out), nil
}

// explain turns whatever the child said into one sentence worth reading.
//
// The reason arrives twice removed: the provider speaks gRPC, cnspec logs that
// through zerolog, and .Output() only surfaces stderr when the child exited
// non-zero. So this unwraps as far as it can and, when there is nothing left
// to unwrap, names the command instead -- because "could not reach it" with no
// way to find out why is the failure this whole contract exists to prevent.
func (d discoverTarget) explain(err error) error {
	if err == nil {
		return nil
	}
	if reason := discoverReason(err.Error()); reason != "" {
		return errorString(reason)
	}
	return errors.New("cannot list " + d.target + " — run: " + d.commandLine())
}

// discoverReason digs the provider's own words out of the child's stderr.
func discoverReason(text string) string {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	// The fatal line is the last thing written, and everything above it is
	// startup noise about provider shorthands and the like.
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		// zerolog's console writer renders the error as error="..." after the
		// message, and the message is ours, not the provider's.
		if _, after, ok := strings.Cut(line, `error="`); ok {
			if inner, _, ok := strings.Cut(after, `"`); ok && inner != "" {
				line = inner
			}
		}
		line = strings.TrimSpace(strings.TrimLeft(line, "x!*-> "))
		if line == "" {
			continue
		}
		return stripRPCPrefix(errorString(line)).Error()
	}
	return ""
}

// source turns the declaration into the contract the launcher consumes.
func (d discoverTarget) source() Source {
	return Source{
		ID:       d.id,
		Class:    ClassPostConnection,
		Cost:     CostRemote,
		Activity: d.activity,
		Tool:     discoverTool,
		Needs:    d.needs(),
		Explain:  d.explain,
		FetchCtx: func(ctx context.Context, params []string) ([]string, error) {
			return d.fetch(ctx, nil, params)
		},
	}
}

// discoverTargets is the table.
//
// A pair earns a row by satisfying both halves, checked against the installed
// providers rather than assumed:
//
//   - the connector declares the discovery target (its Discovery list), and
//   - it declares a flag that consumes the answer, whose description says so.
//
// Forty connectors declare discovery targets and most of them fail the second
// half, which is why this table is shorter than the count of things cnspec can
// discover. The exclusions are recorded in the test file rather than here, so
// that a future pass can see what was considered and why it was left out.
var discoverTargets = []discoverTarget{
	{
		id:        DiscoverK8sNamespaces,
		connector: "k8s",
		target:    "namespaces",
		flag:      "namespaces",
		activity:  "asking cnspec discover for the cluster's namespaces",
		// The connector's --context is parsed and then never reaches the
		// client config, so the cluster travels as a kubeconfig copy. That
		// plumbing already exists in kubeconfig.go and is not duplicated here.
		envFrom:  "f:context",
		envApply: kubeEnvForContext,
		// The document also holds the cluster that was connected to, and a
		// cluster is not one of its own namespaces.
		keep:  platformNamed("k8s-namespace"),
		value: fromName(),
	},
	{
		id:        DiscoverGitHubRepos,
		connector: "github",
		target:    "repos",
		flag:      "repos",
		activity:  "asking cnspec discover for the organization's repositories",
		scope: []discoverScope{
			{
				need: "p:0", equals: "org",
				unless: "repositories are only listed for an organization",
			},
			{need: "p:1", label: "the organization"},
		},
		keep: platformNamed("github-repo"),
		// The connector names a discovered repository "<org>/<repo>" while
		// --repos matches the bare name, which is also what the child's own
		// Options["repository"] carries.
		value: firstValue(fromOption("repository"), nameAfterSlash()),
	},
	{
		id:        DiscoverGitLabGroups,
		connector: "gitlab",
		target:    "groups",
		flag:      "group",
		activity:  "asking cnspec discover for your GitLab groups",
		keep:      platformNamed("gitlab-group"),
		// --group takes the full path, which is what discovery writes into
		// the child's own Options["group"].
		value: fromOption("group"),
	},
	{
		id:        DiscoverGitLabProjects,
		connector: "gitlab",
		target:    "projects",
		flag:      "project",
		activity:  "asking cnspec discover for the group's projects",
		// Without a group this walks every group the token can see, which is
		// slow but not wrong, so the scope narrows rather than gates.
		scope: []discoverScope{
			{need: "f:group", flag: "group", label: "a group", optional: true},
		},
		keep:  platformNamed("gitlab-project"),
		value: fromOption("project"),
	},
	{
		id:        DiscoverAzureSubscriptions,
		connector: "azure",
		target:    "subscriptions",
		flag:      "subscriptions",
		activity:  "asking cnspec discover for your Azure subscriptions",
		// --subscriptions is matched against the subscription id and nothing
		// else, so the display name the asset is called by would silently
		// match no subscription at all.
		value: firstValue(
			fromOption("subscription-id"),
			fromPlatformID("subscriptions"),
		),
	},
	{
		id:        DiscoverSourceID("neon", "organizations"),
		connector: "neon",
		target:    "organizations",
		flag:      "organization",
		activity:  "asking cnspec discover for your Neon organizations",
		keep:      platformNamed("neon-organization"),
		value:     fromPlatformID("organization"),
	},
	{
		id:        DiscoverSourceID("netlify", "accounts"),
		connector: "netlify",
		target:    "accounts",
		flag:      "account",
		activity:  "asking cnspec discover for your Netlify accounts",
		keep:      platformNamed("netlify-account"),
		value:     fromPlatformID("account"),
	},
	{
		id:        DiscoverSourceID("vercel", "teams"),
		connector: "vercel",
		target:    "teams",
		flag:      "team",
		activity:  "asking cnspec discover for your Vercel teams",
		keep:      platformNamed("vercel-team"),
		value:     fromPlatformID("team"),
	},
	{
		id:        DiscoverAtlasProjects,
		connector: "mongodbatlas",
		target:    "projects",
		flag:      "project-id",
		activity:  "asking cnspec discover for your Atlas projects",
		scope: []discoverScope{
			{need: "f:org-id", flag: "org-id", label: "an organization", optional: true},
		},
		keep:  platformNamed("mongodbatlas-project"),
		value: fromPlatformID("project"),
	},
	{
		id:        DiscoverClaudeWorkspaces,
		connector: "claude",
		target:    "workspaces",
		flag:      "workspace-id",
		activity:  "asking cnspec discover for your Claude workspaces",
		keep:      platformNamed("claude-workspace"),
		value:     fromPlatformID("workspace"),
	},
}

func init() {
	sources := make([]Source, 0, len(discoverTargets))
	for _, d := range discoverTargets {
		sources = append(sources, d.source())
	}
	Register(sources...)
}

// DiscoverPair is one row of the table above, reduced to the claims it makes
// about a connector: that the connector declares this discovery target, and
// that it declares a flag the answer can be picked into.
type DiscoverPair struct {
	ID        string
	Connector string
	Target    string
	// Flag is where the answer lands.
	Flag string
	// ScopeFlags are the flags the discovery is narrowed by, which have to
	// exist too.
	ScopeFlags []string
}

// DiscoverPairs is the table, flattened.
//
// It exists so those claims can be confronted with the recorded connector
// metadata, which lives with the launcher because that is where the catalog
// that produced it lives. Both halves have to be checked against the provider
// rather than against what a flag name suggests: aws --filters looks exactly
// like the place a discovered account goes, and is not.
func DiscoverPairs() []DiscoverPair {
	out := make([]DiscoverPair, 0, len(discoverTargets))
	for _, d := range discoverTargets {
		p := DiscoverPair{ID: d.id, Connector: d.connector, Target: d.target, Flag: d.flag}
		for _, s := range d.scope {
			if s.flag != "" {
				p.ScopeFlags = append(p.ScopeFlags, s.flag)
			}
		}
		out = append(out, p)
	}
	return out
}
