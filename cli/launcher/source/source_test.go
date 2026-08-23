// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package source

import (
	"slices"
	"strings"
	"testing"

	"github.com/cockroachdb/errors"
)

// The contract is enforced here rather than described in a comment. Every rule
// below was learned by shipping its violation for one provider; applying it to
// every registered source is the whole point of having a contract.
//
// Each of these walks all(), so a source declared tomorrow is held to the rules
// written today without either file being edited. That is deliberate and is the
// thing to preserve: the previous shape kept its own list of source ids and its
// own list of tool names here, which meant a source declared in one file failed
// a test in another, and every author of a new source edited the same line to
// get green.

// Naming the tool is what makes a wait useful: it says whose credentials matter
// and what the user could run themselves.
//
// The claim lives on the declaration rather than in a list here. This test used
// to hold its own `tools := []string{"gcloud", "kubectl", ...}`, which meant a
// source declared in one file failed a test in another, and every author of a
// new source edited the same line to get green. Source.Tool makes it a property
// of each source instead, and a new source is an append in its own file.
func TestEverySourceSaysWhatItIsDoing(t *testing.T) {
	for _, s := range all() {
		id := s.ID
		if s.Activity == "" {
			t.Errorf("%s: no Activity, so a wait on it would say nothing", id)
			continue
		}
		if strings.EqualFold(s.Activity, "loading") {
			t.Errorf("%s: %q tells the user nothing they cannot see", id, s.Activity)
		}
		if s.Tool == "" {
			t.Errorf("%s: no Tool, so nothing says whose credentials this needs", id)
			continue
		}
		if !strings.Contains(strings.ToLower(s.Activity), strings.ToLower(s.Tool)) {
			t.Errorf("%s: Activity %q does not name its Tool %q", id, s.Activity, s.Tool)
		}
	}
}

// Cost is a claim about what a source does, and a wrong claim means it runs at
// the wrong time -- eagerly blocking a form, or lazily hiding a free answer.
//
// The claim the source already makes in prose is the one checked: an Activity
// that says it is *asking* another program has left this process, and a source
// that has left this process cannot be instant. Reading a file says "reading",
// and checking an environment variable says neither. This replaces a hand-kept
// map of source ids, for the same reason as above.
//
// The other half of the contract -- that only a remote cost defers -- is
// TestOnlyRemoteSourcesAreDeferred below.
func TestSourceCostMatchesWhatItDoes(t *testing.T) {
	for _, s := range all() {
		id := s.ID
		asking := strings.HasPrefix(strings.ToLower(strings.TrimSpace(s.Activity)), "asking")
		if asking && s.Cost == CostInstant {
			t.Errorf("%s: %q leaves this process but claims CostInstant", id, s.Activity)
		}
	}
}

// Only what crosses a network waits to be asked for. A file read that deferred
// would leave a form empty for no reason; a network call that did not would
// block opening one.
func TestOnlyRemoteSourcesAreDeferred(t *testing.T) {
	for _, s := range all() {
		id := s.ID
		if got, want := Deferred(id), s.Cost == CostRemote; got != want {
			t.Errorf("%s: deferred=%v, want %v for cost %v", id, got, want, s.Cost)
		}
	}
}

// A source that can fail in a way the user could act on has to say so in one
// sentence. Anything else is either unrecognised -- where the tool's own words
// are kept -- or cannot fail.
func TestFailuresExplainThemselves(t *testing.T) {
	// Verbatim from gcloud with an expired token.
	stale := errors.New(`ERROR: (gcloud.projects.list) There was a problem refreshing your current auth tokens: Reauthentication failed.
Please run:

  $ gcloud auth login`)

	got := explainFailure(GCPProjectAll, stale).Error()
	if !strings.Contains(got, "gcloud auth login") {
		t.Errorf("explained as %q, want the command to run", got)
	}
	if strings.Contains(got, "\n") {
		t.Errorf("a picker gets one line, got %q", got)
	}

	// A source with no Explain still loses the gRPC envelope, which is the
	// shape provider errors arrive in.
	wrapped := errors.New("rpc error: code = Unknown desc = dial tcp 127.0.0.1:22: connect: connection refused")
	if got := explainFailure(SSHHost, wrapped).Error(); strings.HasPrefix(got, "rpc error:") {
		t.Errorf("the gRPC envelope survived: %q", got)
	} else if !strings.Contains(got, "connection refused") {
		t.Errorf("the provider's own words were lost: %q", got)
	}

	if explainFailure(SSHHost, nil) != nil {
		t.Error("no error should explain to nothing")
	}
}

// Prefilling a guess the user must notice and undo is worse than an empty
// field, so an opinion is only offered when the answer is unambiguous.
func TestPrefillOnlyWhenUnambiguous(t *testing.T) {
	// A sole candidate is unambiguous for every source, and is handled once
	// rather than declared nine times.
	for _, s := range all() {
		id := s.ID
		if v, why := PreferredValue(id, []string{"only"}); v != "only" || why == "" {
			t.Errorf("%s: single candidate gave %q/%q", id, v, why)
		}
		if v, _ := PreferredValue(id, nil); v != "" {
			t.Errorf("%s: prefilled %q from nothing", id, v)
		}
	}

	// Several candidates with nothing to distinguish them is not.
	if v, _ := PreferredValue(SSHHost, []string{"a", "b"}); v != "" {
		t.Errorf("two ssh hosts prefilled %q", v)
	}
	if v, why := PreferredValue(AWSProfile, []string{"prod", "default"}); v != "default" || why != "default" {
		t.Errorf("aws = %q/%q, want the conventional profile", v, why)
	}
}

// A cache key must be stable across lookups and must vary with the parameters.
// Violating the first hid a complete answer behind a key that changed every
// time it was asked for.
func TestCacheKeysAreStableAndScoped(t *testing.T) {
	a := Key(K8sNamespace, []string{"context=prod"})
	if a != Key(K8sNamespace, []string{"context=prod"}) {
		t.Fatal("the same question produced two keys")
	}
	if a == Key(K8sNamespace, []string{"context=staging"}) {
		t.Fatal("two clusters share a key")
	}
	if !strings.HasPrefix(a, K8sNamespace) {
		t.Errorf("key %q does not name its source", a)
	}
}

// A source that depends on another field says so, rather than the model
// carrying a hardcoded exception for it.
func TestDependenciesAreDeclared(t *testing.T) {
	s, ok := ByID(K8sNamespace)
	if !ok {
		t.Fatal("the namespace source is not registered")
	}
	if !slices.Contains(s.Needs, "context") {
		t.Fatalf("namespaces depend on the cluster, Needs = %v", s.Needs)
	}
	// And a source that depends on nothing declares nothing.
	if s, _ := ByID(AWSProfile); len(s.Needs) != 0 {
		t.Errorf("aws profiles depend on nothing, Needs = %v", s.Needs)
	}
}
