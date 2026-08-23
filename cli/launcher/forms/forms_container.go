// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package forms

// Containers & Kubernetes: container, docker, helm, k8s, kustomize and
// portainer.
//
// Two of the six are live systems and four are files on disk, and the split is
// what most of the curation here is about. docker, container and k8s can be
// pointed at something running -- a daemon, a cluster -- or at something
// written down, and the first question each of them asks is which. helm and
// kustomize are renderers: they take a chart or an overlay and produce
// manifests, so their form is a path plus the few values that change what gets
// rendered. portainer is neither; it is an API service reached with an address
// and a token.
//
// The path-shaped members of this family have the same problem the
// infrastructure-as-code readers do: the derived slot spells the usage string
// verbatim, so an uncurated helm asks for "PATH" where it means "a chart
// directory, a packaged chart, or a release in a cluster". Naming the argument
// in the connector's own vocabulary is most of what a spec buys here.

// containerSpec splits what used to be one opaque reference into the four
// things it can actually be. A registry image is deliberately not enumerated:
// listing a registry needs credentials and a round trip, and cnspec already
// does it properly through --discover container-images.
//
// The context comes before the reference because it decides what the reference
// can be. docker and container declare no --context -- verified against the
// installed os provider metadata, which lists only --sudo, --id-detector,
// --disable-cache and --container-proxy -- so the chosen context travels as
// DOCKER_CONTEXT, which is what the docker CLI reads and what mql's own
// dockerclient honours. It is only shown for the two kinds it can affect: a
// Dockerfile is read off this disk, and a registry reference is resolved by
// cnspec rather than by a daemon.
var containerSpec = FormSpec{
	Positional: []PositionalSpec{
		{
			Label:    "kind",
			Desc:     "what the reference points at",
			Required: true,
			Options:  []string{"running container", "local image", "registry image", "dockerfile"},
			Emit: map[string]string{
				"running container": "",
				"local image":       "",
				"registry image":    "",
				"dockerfile":        "file",
			},
		},
		{
			Label:   "docker context",
			Desc:    "which daemon to ask; leave empty for the current one",
			Special: SpecialDockerContext,
			Source:  srcDockerContext,
			ShowIf:  []string{"running container", "local image"},
		},
		{
			Label:    "reference",
			Desc:     "container name, image tag, registry reference or path",
			Required: true,
			SourceBy: map[string]string{
				"running container": srcDockerContainer,
				"local image":       srcDockerImage,
			},
			// A registry holds more than one image, and cnspec enumerates it
			// properly through discovery.
			DiscoverBy: map[string][]string{
				"registry image": {"container-images"},
			},
		},
	},
}

func init() {
	registerSpec("container", containerSpec)

	// cnspec infers what a container reference points at from the reference
	// itself: `scan docker <id>` and `scan docker ubuntu:latest` are the same
	// syntax, and the only sub-command word the connector accepts is `file`.
	// So the kind here steers which values are offered, and emits nothing --
	// except for a Dockerfile, where `file` is real.
	registerSpec("docker", containerSpec)

	// A helm chart is two different things wearing one argument. `helm ./chart`
	// reads a directory or a .tgz off disk; `helm ingress-nginx --repo <url>`
	// fetches one by name, and only then do --repo, --version and the registry
	// credentials mean anything. Both were run to confirm the second shape
	// takes a bare chart name as its positional.
	//
	// --values is shown. It used to be hidden, along with the other five
	// list-typed flags, because a FlagType_List field became a multi-choice
	// with no options and no keystroke could fill one -- a true description of
	// the field engine and a poor answer for this flag, since a chart rendered
	// without its values file is not the chart anyone deploys. typeEmptyLists
	// makes an optionless list a typed, comma-separated field, which is how
	// `--values a.yaml,b.yaml` already reaches a provider from a shell.
	//
	// The four --set variants stay hidden on their own merits. They set
	// individual template keys, with helm's own escaping rules for the dots and
	// commas that appear inside a value, which a comma-separated box would
	// mangle rather than express; --values takes the same content as a file and
	// is the shape a launcher can offer honestly. --api-versions is render
	// fidelity for one cluster rather than a description of the target.
	registerSpec("helm", FormSpec{
		Positional: []PositionalSpec{
			{
				Label: "chart source", Desc: "a chart on disk, or one fetched from a repository",
				Required: true,
				Options:  []string{"local chart", "chart repository"},
				// helm takes no sub-command word: the selector steers which
				// questions are asked and contributes nothing to the command.
				Emit: map[string]string{"local chart": "", "chart repository": ""},
			},
			{
				Label: "chart", Desc: "a directory or .tgz on disk, or the chart name to fetch",
				Required: true,
			},
		},
		// release-name and namespace belong in TARGET because a chart rendered
		// under a different release or namespace is a different set of
		// manifests -- which is the thing being scanned. --kube-version shapes
		// the render too but has a working default and is a knob rather than a
		// target, so it keeps its label and stays in OPTIONS.
		Target:     []string{"repo", "version", "release-name", "namespace", "values"},
		Credential: []string{"username", "password"},
		Labels: map[string]string{
			"repo":         "repository URL",
			"version":      "chart version",
			"release-name": "release name",
			"namespace":    "release namespace",
			"kube-version": "target Kubernetes version",
			"username":     "repository user",
			"password":     "repository password",
			"values":       "values files",
		},
		// Everything a remote fetch needs, and nothing a local chart has any
		// use for.
		ShowFlagsIf: map[string][]string{
			"repo":     {"chart repository"},
			"version":  {"chart repository"},
			"username": {"chart repository"},
			"password": {"chart repository"},
		},
		Hide: []string{
			// Per-key overrides and render fidelity; see above.
			"set", "set-string", "set-json", "set-file", "api-versions",
			// Helm's own plumbing: where the repository index is cached and
			// which repositories.yaml to read. Neither describes the target.
			"repository-config", "repository-cache",
			// Render fidelity for an upgrade rather than an install. A
			// security scan of a chart does not turn on it.
			"is-upgrade",
		},
	})

	// k8s asks what shape of thing is being scanned first, then only what that
	// shape needs: a live cluster asks for a cluster and, optionally, which
	// namespaces; a manifest asks for a path. Neither screen shows the other's
	// questions.
	registerSpec("k8s", FormSpec{
		Positional: []PositionalSpec{
			{
				Label: "what to scan", Desc: "a running cluster, or Kubernetes YAML on disk",
				Required: true,
				Options:  []string{"live cluster", "manifest file"},
				Emit:     map[string]string{"live cluster": "", "manifest file": ""},
			},
			{
				Label: "manifest path", Desc: "the file or directory to read",
				Required: true, ShowIf: []string{"manifest file"},
			},
		},
		Target:  []string{"context", "namespaces"},
		Sources: map[string]string{"context": srcKubeContext, "namespaces": srcK8sNamespace},
		Labels:  map[string]string{"context": "cluster", "namespaces": "namespaces (optional)"},
		// The cluster and its namespaces are meaningless for a manifest.
		ShowFlagsIf: map[string][]string{
			"context":    {"live cluster"},
			"namespaces": {"live cluster"},
		},
		// --context is declared and parsed, and then never reaches the client
		// config, so the chosen cluster travels as a kubeconfig copy instead.
		// The value needs a file written for it, so it is an EnvSpec in
		// kubeconfig.go rather than a plain variable here.
	})

	// kustomize declares no flags at all -- the overlay directory is the whole
	// input -- so the only thing worth saying is what the argument is. The
	// derived slot would say "PATH".
	registerSpec("kustomize", FormSpec{
		Positional: []PositionalSpec{{
			Label: "overlay path", Desc: "the Kustomize overlay directory to render",
			Required: true,
		}},
	})

	// portainer's address is both a positional and --address, and the two are
	// the same value; asking twice would be the form's own invention. The
	// positional is the shape the connector documents, so --address is hidden
	// rather than the other way round. Both routes were run against the
	// installed provider to confirm they are interchangeable.
	registerSpec("portainer", FormSpec{
		Positional: []PositionalSpec{{
			Label: "instance address", Desc: "host, host:port or URL; https is assumed",
			Required: true,
		}},
		Credential: []string{"access-token"},
		Labels: map[string]string{
			"access-token": "access token",
			"insecure":     "skip TLS verification",
		},
		Hide: []string{"address"},
	})
}
