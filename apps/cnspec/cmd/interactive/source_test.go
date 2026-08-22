// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	"strings"
	"testing"

	"go.mondoo.com/cnspec/cli/launcher/source"
)

// The seams between the launcher and its value pickers.
//
// cli/launcher/source cannot import this package -- this package imports it --
// so two things it needs are installed from here at init. Both are the kind of
// wiring that fails silently if it is forgotten: a picker that answers for the
// wrong cluster, and a paste box with nowhere to send what was pasted. The
// tests below are what makes forgetting loud.

// The namespace picker points its child at one cluster by writing a kubeconfig
// copy first, and that writer is kubeconfig.go's. Without it installed, a
// namespace list would come back for whichever cluster the ambient kubeconfig
// happened to name -- with no error, which is the failure the whole contract
// exists to prevent.
func TestTheKubeconfigWriterIsInstalled(t *testing.T) {
	env, cleanup, err := source.KubeEnvApply("does-not-exist")
	if cleanup != nil {
		defer cleanup()
	}
	// It either produced a KUBECONFIG or refused; what it must not do is
	// quietly return nothing, which is what the uninstalled default does for a
	// context that was actually chosen.
	if err == nil && len(env) == 0 {
		t.Fatal("a chosen context produced no environment and no error, so the picker would answer for the ambient cluster")
	}
	if err != nil && strings.Contains(err.Error(), "did not install a kubeconfig writer") {
		t.Fatalf("the launcher never installed its kubeconfig writer: %v", err)
	}

	// An empty context is nothing to do, and stays nothing to do.
	env, cleanup, err = source.KubeEnvApply("")
	if cleanup != nil {
		defer cleanup()
	}
	if err != nil || len(env) != 0 {
		t.Errorf("no context gave %v/%v, want nothing to do", env, err)
	}
}

// The paste routes cli/launcher/source used to declare into the delivery
// registry, and the test that checked they had arrived, are both gone. A pasted
// value travels the same way a typed one does now -- into the connector's own
// ParseCLI -- so there is nothing for a source to declare and nothing for a
// registry to receive. What each ambient credential still declares is which
// variables the *provider* reads when the launcher supplies nothing, which is
// what its readout row says.

// The docker sources name this field in their Needs and the form spec names it
// when it creates the field. They are two constants in two packages, and a
// mismatch is silent: the identity simply never matches, the enumeration runs
// against the default daemon, and the list looks right.
func TestTheDockerContextMarkerAgrees(t *testing.T) {
	if specialDockerContext != source.SpecialDockerContext {
		t.Fatalf("the form calls it %q, the picker %q",
			specialDockerContext, source.SpecialDockerContext)
	}
	s, ok := sourceByID(srcDockerContainer)
	if !ok {
		t.Fatal("the container source is not registered")
	}
	want := "s:" + specialDockerContext
	found := false
	for _, need := range s.Needs {
		found = found || need == want
	}
	if !found {
		t.Errorf("the container picker depends on %v, want %q", s.Needs, want)
	}
}
