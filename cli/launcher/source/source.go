// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package source

import (
	"context"
	"sort"
	"strings"
)

// A Source produces the candidate values for a field.
//
// It exists because the launcher's pickers were written one provider at a time,
// and each one answered the same handful of questions in its own place: when to
// run, what to say while running, what to say when it fails, what it depends on,
// and when a value is obvious enough to fill in. Those answers lived as
// conditionals scattered across three files, so every new provider rediscovered
// the same bugs -- a list that shows nothing when it means "cannot reach", a
// cache key that hides a complete answer, a live list left attached to the wrong
// selector.
//
// Declaring them here is the point. A new source gets the behaviour by
// construction rather than by remembering.
type Source struct {
	// ID is what a form field names to use this source.
	ID string
	// Class is what kind of question this source can answer at all. It decides
	// which widget the field gets, not merely how it is filled.
	Class Class
	// Cost decides when the source runs. Not the mechanism: docker is a
	// subprocess like the k8s namespace list, but one asks a socket on this
	// machine and the other crosses a network.
	Cost Cost
	// Activity says what is happening while it happens, naming the tool being
	// asked -- "asking gcloud for projects". "Loading" tells the user nothing
	// they cannot see; the tool's name tells them whose credentials matter and
	// what to run themselves.
	Activity string
	// Tool is what Activity names: the command being run or the file being
	// read -- "gcloud", "kubectl", "~/.aws". It is declared rather than
	// inferred because the contract test that checks Activity actually names
	// something used to hold its own list of tool names, which meant every new
	// source failed a test in a file it did not own. Naming it here makes that
	// a per-source property instead.
	Tool string
	// Env names the environment variable this source's chosen value travels
	// in, for the targets that have no flag to carry them. Empty for the
	// common case, where the value is a command-line argument. See
	// formEnvironment: a field whose spec names a variable wins over this, and
	// the general case -- a value that has to become a file first -- is an
	// EnvSpec instead.
	Env string
	// Needs names the fields whose values this source depends on. A namespace
	// list belongs to one cluster, and serving another's is the kind of
	// confidently wrong answer worth designing against.
	Needs []string
	// Emit maps a value as this source shows it to the value a command line
	// takes. Empty for the common case, where a picker offers exactly the words
	// the connector accepts.
	//
	// A picker is free to annotate what it found -- an AWS profile is listed
	// with the account id its config names, because "which account is that?" is
	// the question a profile name rarely answers -- and the annotation is not
	// something any provider will accept. Until this existed, the form engine
	// carried the one source that does it: emitted(), the function every field
	// of every connector passes through on its way to argv, opened with a test
	// for "aws.profile". The second annotated picker would have added a second.
	Emit EmitFunc
	// Prefer picks the one obvious value, or returns "" for no opinion. A guess
	// the user has to notice and undo is worse than an empty field.
	Prefer PreferFunc
	// Explain turns whatever the underlying tool said into one actionable
	// sentence. Providers speak gRPC, CLIs speak prose; a picker has room for
	// neither.
	Explain ExplainFunc
	// Fetch returns the values and, separately, why there are none. Both:
	// "empty" and "failed" look identical on screen unless the source
	// distinguishes them.
	Fetch FetchFunc
	// FetchCtx is Fetch for a source that spawns something, and exists because
	// abandoning a result is not the same as stopping the work.
	//
	// A picker the user closes used to leave its `gcloud projects list` or its
	// `cnspec run k8s` running to completion, holding a connection and a
	// process nobody was waiting for. The context comes from the model and is
	// cancelled when the picker closes, so the child is killed rather than
	// orphaned; the source's own timeout stays as the backstop for the case
	// where the picker is left open.
	//
	// A source declares one or the other. The file-backed sources answer in
	// microseconds and have nothing to cancel, so Fetch is right for them and
	// stays the simpler shape.
	FetchCtx FetchCtxFunc
}

// Class is what kind of discovery is possible for a provider at all.
//
// Sorting providers this way is what stops the launcher building a picker for
// something that can never be enumerated. Seven of the eighteen providers
// surveyed have exactly one env var and one flag: there is no list, and offering
// an empty box was a category error rather than a missing feature.
type Class int

const (
	// ClassEnumerated: a local file lists the candidates. Cheap, offline, and
	// the answer is a set of credentials or addresses.
	ClassEnumerated Class = iota
	// ClassAmbient: one credential from the environment. Nothing to enumerate;
	// the useful thing to show is whether it is present and where it came from.
	ClassAmbient
	// ClassPostConnection: what is inside a target, once connected. Feeds the
	// connector's own filter flags.
	ClassPostConnection
	// ClassRequired: there is nothing ambient and nothing to enumerate. The
	// user has to supply the value, and the honest thing a launcher can do is
	// say so rather than offer a box that will never fill itself in. The
	// provider survey found three connectors in exactly this position, and the
	// enum could not previously express it.
	ClassRequired
)

// Cost decides when a source runs, and is the whole of that policy.
type Cost int

const (
	// CostInstant: a file read. Runs when a form opens.
	CostInstant Cost = iota
	// CostLocal: something on this machine that is not free -- a daemon, a
	// subprocess -- but answers in about a second. Also runs when a form opens,
	// asynchronously.
	CostLocal
	// CostRemote: crosses a network, needs credentials, and can fail. Runs only
	// when its picker is opened, with a spinner and a way out.
	CostRemote
)

// deferred reports whether a cost waits to be asked for.
func (c Cost) Deferred() bool { return c == CostRemote }

type (
	// PreferFunc picks the obvious value and says why, or returns "" twice.
	PreferFunc func(values []string) (value, why string)
	// ExplainFunc maps a failure to one sentence the user can act on.
	ExplainFunc func(err error) error
	// FetchFunc returns candidates for the given parameters.
	FetchFunc func(params []string) ([]string, error)
	// FetchCtxFunc is FetchFunc for a source with something to cancel.
	FetchCtxFunc func(ctx context.Context, params []string) ([]string, error)
	// EmitFunc turns one of a source's displayed values into the value a
	// command line takes.
	EmitFunc func(display string) string
)

// registry holds every declared source, keyed by ID.
var registry = map[string]Source{}

// Register adds a source, and is called from each source file's init.
func Register(sources ...Source) {
	for _, s := range sources {
		registry[s.ID] = s
	}
}

// ByID returns a declared source.
func ByID(id string) (Source, bool) {
	s, ok := registry[id]
	return s, ok
}

// Registry is every declared source keyed by id, and is the live map rather
// than a copy.
//
// It exists for the tests that install a source, drive a screen with it and
// take it away again -- a picker's behaviour is only observable through a form
// that names one, and inventing a real provider to observe it with would tie
// the test to whatever is installed. Both of those are on the launcher's side
// of this boundary, which is why this is exported and Register is not enough.
// Nothing outside a test should reach for it.
func Registry() map[string]Source { return registry }

// all returns every declared source, in id order.
//
// It is what the contract test walks, and returning a sorted slice rather than
// ranging the map is what makes a failure name the same source every run.
func all() []Source {
	out := make([]Source, 0, len(registry))
	for _, s := range registry {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// ActivityFor is what to say while a source runs.
func ActivityFor(id string) string {
	if s, ok := ByID(id); ok && s.Activity != "" {
		return s.Activity
	}
	return "looking"
}

// Deferred reports whether a source waits until its picker is opened.
func Deferred(id string) bool {
	s, ok := ByID(id)
	return ok && s.Cost.Deferred()
}

// PreferredValue asks a source for the one obvious value.
func PreferredValue(id string, values []string) (value, why string) {
	if len(values) == 0 {
		return "", ""
	}
	if s, ok := ByID(id); ok && s.Prefer != nil {
		if v, w := s.Prefer(values); v != "" {
			return v, w
		}
	}
	// A single candidate is unambiguous whatever the source, so this does not
	// need declaring nine times.
	if len(values) == 1 {
		return values[0], "only one"
	}
	return "", ""
}

// Emit is how a source's displayed values reach a command line, or nil
// when they reach it unchanged. It is what a form field is handed when a picker
// is attached to it, so the field can answer what it emits without knowing that
// pickers exist.
func Emit(id string) EmitFunc {
	if s, ok := ByID(id); ok {
		return s.Emit
	}
	return nil
}

// explainFailure turns a source's error into something worth reading.
func explainFailure(id string, err error) error {
	if err == nil {
		return nil
	}
	if s, ok := ByID(id); ok && s.Explain != nil {
		return s.Explain(err)
	}
	return stripRPCPrefix(err)
}

// Result is one source's answer: what it found, why there is nothing, and
// whether anyone is still waiting for it.
//
// The three are separate on purpose. "Empty" and "failed" look identical on
// screen unless the source distinguishes them, and a cancelled answer is
// neither -- a killed child reports "signal: killed", which is true and
// useless, because the user closed the picker and reporting that their gcloud
// died is blaming them for their own esc key.
type Result struct {
	// Values are the candidates, and may be empty for a real empty answer.
	Values []string
	// Err explains an empty list, already put through the source's own
	// Explain.
	Err error
	// Cancelled marks the answer to a question nobody is waiting for any more.
	Cancelled bool
}

// Load runs a source and returns its answer.
//
// ctx belongs to whoever asked. A source that spawns a child declares FetchCtx
// and receives it, so closing the picker kills what the picker started; the
// file-backed sources answer too quickly for it to matter and declare Fetch.
//
// This is the whole of the fetch policy, and it is deliberately not a tea.Cmd:
// the launcher wraps it in one, so the rule about which of the two fetch
// shapes runs -- and what a cancelled context means -- is testable without a
// UI.
func Load(ctx context.Context, id string, params []string) Result {
	var r Result
	s, ok := ByID(id)
	if !ok || (s.Fetch == nil && s.FetchCtx == nil) {
		return r
	}
	var err error
	if s.FetchCtx != nil {
		r.Values, err = s.FetchCtx(ctx, params)
	} else {
		r.Values, err = s.Fetch(params)
	}
	if ctx.Err() != nil {
		return Result{Cancelled: true}
	}
	r.Err = explainFailure(id, err)
	return r
}

// stripRPCPrefix removes the gRPC envelope that providers' errors arrive in.
// The useful part is what the provider said; "rpc error: code = Unknown desc ="
// is noise in a box this size.
func stripRPCPrefix(err error) error {
	text := err.Error()
	if i := strings.Index(text, "desc = "); i >= 0 && strings.HasPrefix(text, "rpc error:") {
		return errorString(text[i+len("desc = "):])
	}
	return err
}

// errorString is an error that is just its message.
type errorString string

func (e errorString) Error() string { return string(e) }

// Key identifies a source's cached values. The parameters are part of it:
// two clusters have two namespace lists, and serving one for the other is
// exactly the kind of confidently wrong answer worth avoiding. It must be
// stable -- keying on anything generated makes a load and its lookup disagree,
// and a full answer then never reaches the screen.
func Key(id string, params []string) string {
	return id + "\x00" + strings.Join(params, " ")
}

// paramValue reads one of a source's parameters.
func paramValue(params []string, name string) string {
	for _, p := range params {
		if v, ok := strings.CutPrefix(p, name+"="); ok {
			return v
		}
	}
	return ""
}
