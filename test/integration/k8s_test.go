// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build integration

package integration

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/cli/reporter"
)

const (
	k8sNamespace = "cnspec-integration"
	k8sWorkload  = "testdata/k8s/workload.yaml"
)

// TestKubernetesTarget scans a live cluster.
//
// This is the suite's only multi-asset scenario, which makes it the only place
// the scan pipeline is exercised through the CLI with more than one asset in
// flight. The failure it is really watching for is an aggregation bug: assets
// discovered, scanned, and then their results dropped or collapsed onto one
// another. A single-asset scan cannot see that.
func TestKubernetesTarget(t *testing.T) {
	requireKubernetes(t)
	applyWorkload(t)

	args := []string{
		"scan", "k8s",
		"--namespaces", k8sNamespace,
		"-f", k8sSecurity,
		"--detect-cicd=false", "-o", "json",
	}
	res := run(t, 12*time.Minute, args...)
	if res.exitCode != 0 {
		res.saveArtifact(t, "k8s-cluster")
		t.Fatalf("exit code %d, want 0\n%s", res.exitCode, res.dump())
	}
	requireNoProviderPanic(t, res)

	rep := decodeReport(t, res)
	defer func() {
		if t.Failed() {
			res.saveArtifact(t, "k8s-cluster")
			t.Logf("report follows:\n%s", res.dump())
		}
	}()

	requireNoAssetErrors(t, rep)

	// The fixture deploys one Deployment (2 replicas) and one standalone Pod.
	// A floor rather than an exact count: what the provider discovers by
	// default has changed before and is not this suite's decision to pin.
	require.GreaterOrEqual(t, len(rep.GetAssets()), 2,
		"expected at least the deployment and the pod, got %d assets", len(rep.GetAssets()))

	var scoredAssets, totalPass, totalFail int
	for mrn, asset := range rep.GetAssets() {
		assert.NotEmpty(t, asset.GetName(), "asset %s has no name", mrn)

		scores := rep.GetScores()[mrn]
		if scores == nil {
			// Named explicitly: an asset that was discovered and then produced
			// no scores at all is the aggregation bug this test exists for, and
			// it is invisible in the exit code.
			t.Errorf("asset %q (%s) has no scores at all", asset.GetName(), mrn)
			continue
		}
		scoredAssets++

		checks := checkScores(t, rep, mrn)
		// observed 21 checks per asset, 2026-09-22
		if len(checks) < 10 {
			t.Errorf("asset %q scored only %d checks, want >= 10 (%s)",
				asset.GetName(), len(checks), histogram(checks))
		}
		requireNoCheckErrors(t, rep, mrn)
		counts := statusCounts(checks)
		totalPass += counts["pass"]
		totalFail += counts["fail"]
		t.Logf("asset %q (%s): %s", asset.GetName(), asset.GetPlatformName(), histogram(checks))
	}

	assert.Equal(t, len(rep.GetAssets()), scoredAssets, "some assets reported no scores")

	// Both outcomes must appear somewhere in the scan. The fixture guarantees
	// it by construction -- the deployment sets the security context fields
	// the bundle looks for and the standalone pod omits them -- so this holds
	// without pinning any individual check. It is the assertion that separates
	// "scoring works" from the two degenerate modes that look identical from
	// outside: everything passes, and everything fails.
	assert.Greater(t, totalPass, 0, "no check passed on any asset")
	assert.Greater(t, totalFail, 0,
		"no check failed on any asset, although the fixture deploys a pod with no "+
			"security context; scoring is not discriminating")

	assertAssetNamesDiffer(t, rep)
}

// applyWorkload installs the fixture and waits only for the namespace.
//
// kubectl, not client-go: the fixture is a static manifest and applying it is
// setup, not the thing under test. Adding a Kubernetes client dependency to
// cnspec's test tree to do what one kubectl call does would be paid for on
// every build.
func applyWorkload(t *testing.T) {
	t.Helper()

	if out, err := probe(t, 2*time.Minute, "kubectl", "apply", "-f", k8sWorkload); err != nil {
		t.Fatalf("could not apply %s: %v\n%s", k8sWorkload, err, out)
	}
	if out, err := probe(t, 90*time.Second, "kubectl", "wait",
		"--for=jsonpath={.status.phase}=Active",
		"namespace/"+k8sNamespace, "--timeout=60s"); err != nil {
		t.Fatalf("namespace %s did not become active: %v\n%s", k8sNamespace, err, out)
	}

	// Deliberately no readiness wait; see the comment in the fixture. The
	// objects exist in the API as soon as apply returns, which is all the
	// provider needs.
	t.Cleanup(func() {
		// On failure, record what the cluster held before anything is torn
		// down: the k3d cluster is deleted when the test ends, so there is no
		// later moment to look.
		if t.Failed() {
			state, _ := probe(t, time.Minute, "kubectl", "get", "all", "-n", k8sNamespace, "-o", "yaml")
			pods, _ := probe(t, time.Minute, "kubectl", "describe", "pods", "-n", k8sNamespace)
			path := filepath.Join(artifactDir, "k8s-cluster-state.txt")
			if err := os.WriteFile(path, []byte(state+"\n---\n"+pods), 0o644); err != nil {
				t.Logf("could not save the cluster state: %v", err)
			} else {
				t.Logf("saved the cluster state to %s", path)
			}
		}
		// Best effort, and only needed when CNSPEC_IT_K8S_CONTEXT points the
		// tier at an existing cluster: a k3d cluster is deleted as a whole.
		if out, err := probe(t, 2*time.Minute, "kubectl", "delete", "-f", k8sWorkload,
			"--ignore-not-found", "--wait=false"); err != nil {
			t.Logf("could not clean up %s: %v\n%s", k8sWorkload, err, out)
		}
	})
}

// assertAssetNamesDiffer is a guard against the naming regression where every
// discovered asset ends up with the same identity upstream. Kept separate
// because it is about identity, not scoring.
func assertAssetNamesDiffer(t *testing.T, rep *reporter.Report) {
	t.Helper()
	seen := map[string]string{}
	for mrn, asset := range rep.GetAssets() {
		name := asset.GetName()
		if prev, dup := seen[name]; dup {
			t.Errorf("assets %s and %s share the name %q", prev, mrn, name)
			continue
		}
		seen[name] = mrn
	}
}
