// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	"strings"
	"testing"

	"go.mondoo.com/mql/providers-sdk/v1/plugin"
)

// Writing the kubeconfig copy that targets a cluster now lives in
// cli/launcher/delivery, and is tested there. What stays here is the launcher
// end of the same story: the picker that lists a cluster's namespaces has to
// answer for the cluster the form is pointed at, and no other.

// Namespaces are cached per cluster: serving one cluster's list for another is
// exactly the confidently-wrong answer worth avoiding.
func TestNamespaceValuesAreCachedPerCluster(t *testing.T) {
	a := sourceKey(srcK8sNamespace, []string{"KUBECONFIG=/tmp/a"})
	b := sourceKey(srcK8sNamespace, []string{"KUBECONFIG=/tmp/b"})
	if a == b {
		t.Fatal("two clusters share a cache key")
	}
	if !strings.HasPrefix(a, srcK8sNamespace) {
		t.Errorf("key %q should name its source", a)
	}
}

// A picker's cache key has to be stable, or a load and its lookup never match.
// Keying on the generated kubeconfig path did exactly that: every lookup minted
// a new path, so a field showed nothing while holding a complete answer.
func TestSourceKeyIsStableAcrossLookups(t *testing.T) {
	m := NewModel([]Connector{{
		Provider: "k8s", Name: "k8s", Use: "k8s", Category: catContainer, Installed: true,
		MaxArgs: 1,
		Flags: []plugin.Flag{
			{Long: "context", Type: plugin.FlagType_String},
			{Long: "namespaces", Type: plugin.FlagType_String},
		},
	}})
	m.syncSelection()
	for i := range m.detail.form.Fields() {
		if m.detail.form.Fields()[i].Flag == "context" {
			m.detail.form.Fields()[i].SetValue("prod-cluster")
		}
	}

	first := sourceKeyFor(m.detail.form, srcK8sNamespace)
	second := sourceKeyFor(m.detail.form, srcK8sNamespace)
	if first != second {
		t.Fatalf("key is not stable: %q then %q", first, second)
	}
	if !strings.Contains(first, "prod-cluster") {
		t.Errorf("key %q should identify the cluster it answers for", first)
	}

	// A different cluster must not read the first one's namespaces.
	for i := range m.detail.form.Fields() {
		if m.detail.form.Fields()[i].Flag == "context" {
			m.detail.form.Fields()[i].SetValue("staging-cluster")
		}
	}
	if sourceKeyFor(m.detail.form, srcK8sNamespace) == first {
		t.Fatal("two clusters share a cache key")
	}
}
