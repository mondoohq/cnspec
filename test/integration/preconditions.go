// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build integration

package integration

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// requireAllTiers is set in CI. See skipTier.
const requireAllTiers = "CNSPEC_IT_REQUIRE_ALL"

// skipTier skips a tier whose infrastructure is missing -- unless skipping is
// forbidden.
//
// On a laptop a missing Docker daemon should skip, with the reason and the
// command that would fix it. In CI it must fail: a suite that can quietly skip
// a tier reports success for the thing it stopped testing, and a skip is far
// easier to introduce by accident than a failure is.
func skipTier(t *testing.T, tier, reason string) {
	t.Helper()
	if os.Getenv(requireAllTiers) != "" {
		t.Fatalf("tier %q cannot run and %s is set: %s", tier, requireAllTiers, reason)
	}
	t.Skipf("skipping %s tier: %s", tier, reason)
}

// requireDocker skips (or fails) when there is no usable Docker daemon.
func requireDocker(t *testing.T) {
	t.Helper()
	out, err := probe(t, 20*time.Second, "docker", "info", "--format", "{{.ServerVersion}}")
	if err != nil {
		skipTier(t, "docker", fmt.Sprintf("`docker info` failed (%v): %s", err, out))
	}
}

// requireKubernetes provides the cluster the k8s tier scans, and makes it the
// one every kubectl call and cnspec scan in the test talks to.
//
// By default it creates a k3d cluster for this test and deletes it when the
// test ends, pass or fail, so nothing is left running between runs. Its
// kubeconfig is written to a temporary file and exported as KUBECONFIG for this
// test only: the developer's kubeconfig and current context are never read or
// changed, so a scan cannot land on whatever cluster happens to be current.
//
// CNSPEC_IT_K8S_CONTEXT is the deliberate override: set it to the name of the
// current context to scan an existing cluster instead.
func requireKubernetes(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("kubectl"); err != nil {
		skipTier(t, "k8s", "kubectl is not on PATH")
	}

	if want := os.Getenv("CNSPEC_IT_K8S_CONTEXT"); want != "" {
		current, err := probe(t, 15*time.Second, "kubectl", "config", "current-context")
		if err != nil {
			skipTier(t, "k8s", fmt.Sprintf("could not read the current context: %v", err))
		}
		if current != want {
			skipTier(t, "k8s", fmt.Sprintf("current context is %q, want %q from %s",
				current, want, "CNSPEC_IT_K8S_CONTEXT"))
		}
		if out, err := probe(t, 30*time.Second, "kubectl", "cluster-info", "--request-timeout=10s"); err != nil {
			skipTier(t, "k8s", fmt.Sprintf("no reachable cluster (%v): %s", err, out))
		}
		return
	}

	startK3dCluster(t)
}

// startK3dCluster creates a single-node k3d cluster, exports its kubeconfig for
// the rest of the test, and registers its deletion.
//
// One server, no load balancer and no Traefik: the tier deploys one Deployment
// and one Pod and scans them, and anything else only costs start-up time.
func startK3dCluster(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("k3d"); err != nil {
		skipTier(t, "k8s", "k3d is not on PATH; install it (https://k3d.io) to run the k8s tier")
	}

	// Unique per run, so a cluster left behind by a killed run never collides.
	name := fmt.Sprintf("cnspec-it-%d", time.Now().UnixNano()%1_000_000_000)
	out, err := probe(t, 5*time.Minute, "k3d", "cluster", "create", name,
		"--no-lb", "--wait", "--timeout", "180s",
		"--k3s-arg", "--disable=traefik@server:0",
		"--kubeconfig-update-default=false", "--kubeconfig-switch-context=false")
	// Registered before the error check: a create that fails halfway can still
	// leave containers behind.
	t.Cleanup(func() {
		if out, err := probe(t, 3*time.Minute, "k3d", "cluster", "delete", name); err != nil {
			t.Logf("could not delete k3d cluster %s: %v\n%s", name, err, out)
		}
	})
	if err != nil {
		t.Fatalf("could not create k3d cluster %s: %v\n%s", name, err, out)
	}

	kubeconfig := filepath.Join(t.TempDir(), "kubeconfig")
	if out, err := probe(t, time.Minute, "k3d", "kubeconfig", "write", name,
		"--output", kubeconfig); err != nil {
		t.Fatalf("could not write the kubeconfig of %s: %v\n%s", name, err, out)
	}
	// Inherited by probe and by the cnspec child process (childEnv keeps
	// KUBECONFIG), and restored when the test ends.
	t.Setenv("KUBECONFIG", kubeconfig)

	if out, err := probe(t, 2*time.Minute, "kubectl", "wait", "--for=condition=Ready",
		"node", "--all", "--timeout=90s"); err != nil {
		t.Fatalf("k3d cluster %s did not become ready: %v\n%s", name, err, out)
	}
}

// probe runs a short auxiliary command and returns its trimmed combined output.
// Auxiliary, not under test: these shape the run, they are not what is being
// asserted, so their output only ever appears in a skip or failure message.
func probe(t *testing.T, timeout time.Duration, name string, args ...string) (string, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	return string(bytes.TrimSpace(out)), err
}

// pullImage fetches an image before the scan, with retries.
//
// This is the only retry in the suite, and it is deliberately not a retry of a
// scan. Docker Hub's anonymous pull limit is per-IP and shared across the
// runner fleet, so a pull is the one step that fails for reasons that have
// nothing to do with cnspec; a scan that fails intermittently is the signal the
// suite exists to produce, and retrying it would hide exactly what we want to
// see.
func pullImage(t *testing.T, image string) {
	t.Helper()
	var last error
	for attempt, backoff := range []time.Duration{0, 5 * time.Second, 15 * time.Second} {
		if backoff > 0 {
			t.Logf("retrying pull of %s in %s (attempt %d)", image, backoff, attempt+1)
			time.Sleep(backoff)
		}
		out, err := probe(t, 5*time.Minute, "docker", "pull", "--quiet", image)
		if err == nil {
			return
		}
		last = fmt.Errorf("%w: %s", err, out)
	}
	t.Fatalf("could not pull %s after 3 attempts: %v", image, last)
}
