// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package delivery

import (
	"os"
	"testing"

	"gopkg.in/yaml.v3"
)

// The k8s connector's --context does not select a cluster: the connection
// hardcodes the current-context override to "". Targeting therefore goes
// through a kubeconfig copy whose current-context is the wanted one.
func TestKubeconfigForContextRewritesCurrentContext(t *testing.T) {
	path, cleanup, err := kubeconfigForContext("testdata/kubeconfig", "aks-trial")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	if got := doc["current-context"]; got != "aks-trial" {
		t.Fatalf("current-context = %v, want aks-trial", got)
	}

	// Everything else must survive: dropping a field here means dropping the
	// auth plugin or certificate that makes a cluster reachable.
	src, _ := os.ReadFile("testdata/kubeconfig")
	var orig map[string]any
	_ = yaml.Unmarshal(src, &orig)
	for _, key := range []string{"apiVersion", "kind", "contexts"} {
		if _, ok := doc[key]; !ok {
			t.Errorf("the copy dropped %q", key)
		}
	}
	if len(doc["contexts"].([]any)) != len(orig["contexts"].([]any)) {
		t.Error("the copy lost contexts")
	}
}

// A kubeconfig can hold client certificates and tokens.
func TestKubeconfigCopyIsPrivate(t *testing.T) {
	path, cleanup, err := kubeconfigForContext("testdata/kubeconfig", "aks-trial")
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("kubeconfig copy mode = %o, want 600", perm)
	}
	cleanup()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("the copy survived cleanup")
	}
}

func TestKubeconfigForContextRejectsBadInput(t *testing.T) {
	if _, _, err := kubeconfigForContext("testdata/kubeconfig", ""); err == nil {
		t.Error("expected an error with no context")
	}
	if _, _, err := kubeconfigForContext("testdata/does-not-exist", "x"); err == nil {
		t.Error("expected an error for a missing kubeconfig")
	}
}

// The copy is tracked for the same reason the generated inventory is: it can
// hold client certificates and bearer tokens, so every exit the process can
// observe removes it, not only the tidy one.
func TestTheKubeconfigCopyIsTracked(t *testing.T) {
	path, cleanup, err := kubeconfigForContext("testdata/kubeconfig", "aks-trial")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	CleanupTempFiles()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("the kubeconfig copy outlived the exit hook: %v", err)
	}
}
