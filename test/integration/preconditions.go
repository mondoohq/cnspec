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
	"strings"
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

// requireKubernetes skips (or fails) when no cluster is reachable, and refuses
// a cluster that is not a local one.
//
// The context guard is not paranoia: `cnspec scan k8s` reads whatever kubeconfig
// context is current, and a developer running the suite with a production
// context active would point a scan at it. The suite creates a kind cluster in
// CI, so requiring a kind context costs nothing there; CNSPEC_IT_K8S_CONTEXT is
// the deliberate override for anyone testing against something else.
func requireKubernetes(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("kubectl"); err != nil {
		skipTier(t, "k8s", "kubectl is not on PATH; the CI job installs a kind cluster, "+
			"locally run: kind create cluster --name cnspec-integration")
	}
	if out, err := probe(t, 30*time.Second, "kubectl", "cluster-info", "--request-timeout=10s"); err != nil {
		skipTier(t, "k8s", fmt.Sprintf("no reachable cluster (%v): %s; "+
			"run: kind create cluster --name cnspec-integration", err, out))
	}

	current, err := probe(t, 15*time.Second, "kubectl", "config", "current-context")
	if err != nil {
		skipTier(t, "k8s", fmt.Sprintf("could not read the current context: %v", err))
	}
	want := os.Getenv("CNSPEC_IT_K8S_CONTEXT")
	if want != "" {
		if current != want {
			skipTier(t, "k8s", fmt.Sprintf("current context is %q, want %q from %s",
				current, want, "CNSPEC_IT_K8S_CONTEXT"))
		}
		return
	}
	if !strings.HasPrefix(current, "kind-") {
		skipTier(t, "k8s", fmt.Sprintf("current context %q is not a kind cluster; "+
			"refusing to scan it. Set CNSPEC_IT_K8S_CONTEXT=%s to override", current, current))
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
