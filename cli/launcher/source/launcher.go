// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package source

import "github.com/cockroachdb/errors"

// The launcher-side seam: the two things this package needs that it cannot yet
// import.
//
// cli/launcher/source is meant to sit on top of cli/launcher/delivery, which
// owns both of them -- the kubeconfig copy that points a child at one cluster,
// and the registry saying which environment variable a pasted secret travels
// in. That package is being extracted from the same commit as this one and
// does not exist yet, so both still live in apps/cnspec/cmd/interactive, which
// imports this package. The dependency therefore cannot be written as an
// import in this direction without a cycle.
//
// What is written down instead is the shape of it: one function value the
// launcher installs, and one list the launcher drains. Neither is a second
// copy of anything, and both are one commit's work to delete once delivery
// exists -- `KubeEnvApply` becomes a direct call, and the drain becomes an
// init in this package.

// EnvApplyFunc builds the environment a child needs to reach one chosen value,
// plus the cleanup for whatever had to be written to get it there.
type EnvApplyFunc func(value string) (env []string, cleanup func(), err error)

// KubeEnvApply points a child process at one Kubernetes context.
//
// The k8s connector's --context is parsed and then never reaches the client
// config, so the only thing that selects a cluster is a KUBECONFIG whose
// current-context is the wanted one -- which means writing a copy of the
// user's kubeconfig first and removing it afterwards. That is the launcher's
// kubeconfig.go.
//
// The default refuses rather than returning nothing, which is the whole reason
// it is written out rather than left nil. A source that quietly ran with no
// environment would answer for whichever cluster the ambient kubeconfig
// happened to name: a confidently wrong answer with no error attached, which
// is the exact failure the Source contract exists to prevent. An empty context
// is still nothing to do, exactly as the real one treats it.
var KubeEnvApply EnvApplyFunc = func(context string) ([]string, func(), error) {
	if context == "" {
		return nil, nil, nil
	}
	return nil, nil, errors.New("cannot target a Kubernetes context — " +
		"the launcher did not install a kubeconfig writer")
}

// kubeEnvForContext is KubeEnvApply, read when it is called rather than when a
// source is declared.
//
// The indirection is what makes the seam safe. Sources are declared in this
// package's init, which runs before the importing package's, so a declaration
// that named KubeEnvApply directly would capture the refusal above and keep it
// for the life of the process.
func kubeEnvForContext(context string) ([]string, func(), error) {
	return KubeEnvApply(context)
}
