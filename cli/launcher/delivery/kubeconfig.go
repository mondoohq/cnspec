// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package delivery

import (
	"os"
	"path/filepath"

	"github.com/cockroachdb/errors"
	"gopkg.in/yaml.v3"
)

// Targeting a specific Kubernetes cluster needs a workaround, and it is worth
// writing down why.
//
// The k8s connector declares --context, described as "Target a specific
// Kubernetes context from your kubeconfig". It does not do that. The connection
// builds its client with the current-context override hardcoded to the empty
// string (k8s/connection/api/connection.go, buildConfigFromFlags call), so the
// kubeconfig's own current-context always wins. The flag's value is read in one
// other place, to label the asset with a cluster name. Verified by querying two
// different contexts and getting the same cluster's namespaces both times.
//
// The connection does read KUBECONFIG, but passes it whole as a single
// ExplicitPath, so the colon-separated form kubectl supports does not work
// either.
//
// What does work: point KUBECONFIG at one file whose current-context is the
// wanted one. So the launcher copies the user's kubeconfig, rewrites that single
// field, and hands the copy to the child. The rest of the document is passed
// through untouched, which keeps exec-based auth plugins, certificate paths and
// everything else working -- extracting a subset would risk dropping exactly
// the fields that make a cluster reachable.

// This is the case the environment contract in delivery.go was generalised
// from: the value the user picked has no flag that carries it, and pointing the
// child at it needs a file written first. Declaring it here rather than as an
// `if connector == "k8s"` in launchArgs is what keeps the next four connectors
// with the same problem from each adding their own branch to that function.
//
// It is also the one contributor an inventory cannot absorb, which was checked
// rather than assumed. The obvious replacement -- a k8s asset naming the
// kubeconfig and the context -- has nowhere to put either: NewConnection reads
// os.Getenv("KUBECONFIG") and nothing else (k8s/connection/api/connection.go),
// there is no option for a kubeconfig path, and Options["context"] is stored by
// ParseCLI and then only used to label the asset -- buildConfigFromFlags is
// called with an empty context, which is the bug this file already works
// around. So KUBECONFIG stays, and it is worth being precise about what that
// costs: nothing the user typed travels in it. The variable names a file, the
// file is a copy of one the user already owns, and the child inherits the
// ambient environment regardless. It is a target selector, not a credential
// route.
func init() {
	RegisterEnv(EnvSpec{
		Connector: "k8s",
		Field:     "context",
		Apply:     KubeEnvForContext,
	})
}

// kubeconfigForContext writes a copy of the user's kubeconfig whose
// current-context is ctx, and returns its path plus a cleanup func.
func kubeconfigForContext(sourcePath, ctx string) (string, func(), error) {
	if ctx == "" {
		return "", nil, errors.New("no context given")
	}
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return "", nil, errors.Wrap(err, "cannot read the kubeconfig")
	}

	// Decoded as a generic document so every field survives the round trip.
	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return "", nil, errors.Wrap(err, "cannot parse the kubeconfig")
	}
	if doc == nil {
		return "", nil, errors.New("the kubeconfig is empty")
	}
	doc["current-context"] = ctx

	out, err := yaml.Marshal(doc)
	if err != nil {
		return "", nil, errors.Wrap(err, "cannot render the kubeconfig")
	}

	dir, err := os.MkdirTemp("", "cnspec-ui-kube-")
	if err != nil {
		return "", nil, errors.Wrap(err, "cannot create a directory for the kubeconfig")
	}
	// A kubeconfig copy can hold client certificates and bearer tokens, so it
	// is tracked for the same reason the generated inventory is: every exit the
	// process can observe removes it, not only the tidy one. See TrackTemp.
	cleanup := TrackTemp(func() { _ = os.RemoveAll(dir) })

	// A kubeconfig can hold client certificates and tokens, so the copy is as
	// private as the original should be.
	path := filepath.Join(dir, "config")
	if err := os.WriteFile(path, out, 0o600); err != nil {
		cleanup()
		return "", nil, errors.Wrap(err, "cannot write the kubeconfig")
	}
	return path, cleanup, nil
}

// defaultKubeconfigPath is the file the user's contexts were read from.
func defaultKubeconfigPath() string {
	if env := os.Getenv("KUBECONFIG"); env != "" {
		// cnspec's k8s connection treats KUBECONFIG as one path, so only the
		// first entry is meaningful to it.
		if paths := filepath.SplitList(env); len(paths) > 0 {
			return paths[0]
		}
	}
	return home(".kube", "config")
}

// home is the launcher's own path-under-$HOME helper, kept here rather than
// imported: this package is downstream of nothing in the launcher, and a
// six-line join is a smaller thing to repeat than a dependency edge is to add.
func home(rel ...string) string {
	dir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(append([]string{dir}, rel...)...)
}

// KubeEnvForContext returns the environment a child needs to target ctx, or
// nothing when no context was chosen.
func KubeEnvForContext(ctx string) ([]string, func(), error) {
	if ctx == "" {
		return nil, nil, nil
	}
	path, cleanup, err := kubeconfigForContext(defaultKubeconfigPath(), ctx)
	if err != nil {
		return nil, nil, err
	}
	return []string{"KUBECONFIG=" + path}, cleanup, nil
}
