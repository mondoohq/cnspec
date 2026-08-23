// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package source

import (
	"context"
	"os"
	"slices"
)

// Every pre-connection source, declared in one place: what can be offered
// before any credential is used.
//
// The readers are elsewhere -- read.go for the kubeconfig, the ssh config and
// the gcloud state, enumerated.go for the four credential files and the docker
// context store -- because the file formats differ and no amount of declaring
// makes an ini file parse like a kubeconfig. What is declared here is
// everything around them: when each runs, what it says while running, what it
// depends on, when a value is obvious, how its failures read, and where its
// chosen value has to travel. That is the part that was being reinvented per
// provider.
//
// It is one file because it used to be two idioms. Half of these sources were
// declared here and implemented in read.go; the other half declared themselves
// in the file that read their format, on the reasoning that what happens to
// the chosen value -- a flag for oci and azure, an environment variable for
// alicloud and docker, nothing at all for snowflake -- belonged next to the
// reader. It does not: that is a property of the connector, which is what
// Source.Env exists to state, and stating it in two places is how a subsystem
// ends up with no owner. Adding a pre-connection source is an entry here and a
// reader wherever its format belongs.

func init() {
	Register(
		Source{
			ID:       AWSProfile,
			Class:    ClassEnumerated,
			Cost:     CostInstant,
			Activity: "reading ~/.aws",
			Tool:     "~/.aws",
			Emit:     awsProfileValue,
			Prefer: func(values []string) (string, string) {
				// The shared config calls the conventional one "default"; the
				// picker may have labelled it with an account id.
				for _, v := range values {
					if awsProfileValue(v) == "default" {
						return v, "default"
					}
				}
				return "", ""
			},
			Fetch: func([]string) ([]string, error) { return awsProfiles(), nil },
		},
		Source{
			ID:       KubeContext,
			Class:    ClassEnumerated,
			Cost:     CostInstant,
			Activity: "reading your kubectl config",
			Tool:     "kubectl",
			Prefer: func(values []string) (string, string) {
				if cur := kubeCurrentContext(); cur != "" && slices.Contains(values, cur) {
					return cur, "current"
				}
				return "", ""
			},
			Fetch: func([]string) ([]string, error) { return kubeContexts(), nil },
		},
		Source{
			ID:       SSHHost,
			Class:    ClassEnumerated,
			Cost:     CostInstant,
			Activity: "reading ~/.ssh/config",
			Tool:     "~/.ssh/config",
			Fetch:    func([]string) ([]string, error) { return sshHosts(), nil },
		},
		Source{
			ID:       GCPProject,
			Class:    ClassEnumerated,
			Cost:     CostInstant,
			Activity: "reading gcloud config",
			Tool:     "gcloud",
			Prefer: func(values []string) (string, string) {
				if cur := GCPActiveProject(); cur != "" && slices.Contains(values, cur) {
					return cur, "active"
				}
				return "", ""
			},
			Fetch: func([]string) ([]string, error) { return gcpProjects(), nil },
		},
		Source{
			ID:       GCPZone,
			Class:    ClassEnumerated,
			Cost:     CostInstant,
			Activity: "reading gcloud config",
			Tool:     "gcloud",
			Fetch:    func([]string) ([]string, error) { return gcpZones(), nil },
		},

		// Local, but not free: a daemon on this machine, answering in about a
		// second. Cheap enough to run when a form opens.
		// Both of these ask a daemon, and which daemon is a question the form
		// can now answer: the container spec offers a docker context, and
		// listing the default daemon's containers while the child is pointed at
		// another one would be exactly the confidently wrong answer the Source
		// contract exists to prevent. Needs puts the chosen context into the
		// cache key as well as into the call, so changing it re-runs the picker
		// rather than reusing the previous daemon's list.
		Source{
			ID:       DockerContainer,
			Class:    ClassEnumerated,
			Cost:     CostLocal,
			Activity: "asking docker for containers",
			Tool:     "docker",
			Needs:    []string{"s:" + SpecialDockerContext},
			FetchCtx: func(ctx context.Context, params []string) ([]string, error) {
				return dockerContainers(ctx, runnerWithEnv(nil, DockerContextEnvFrom(params)))
			},
		},
		Source{
			ID:       DockerImage,
			Class:    ClassEnumerated,
			Cost:     CostLocal,
			Activity: "asking docker for images",
			Tool:     "docker",
			Needs:    []string{"s:" + SpecialDockerContext},
			FetchCtx: func(ctx context.Context, params []string) ([]string, error) {
				return dockerImages(ctx, runnerWithEnv(nil, DockerContextEnvFrom(params)))
			},
		},

		// Remote: crosses a network, needs credentials, and can fail. Waits to
		// be asked for.
		Source{
			ID:       GCPProjectAll,
			Class:    ClassEnumerated,
			Cost:     CostRemote,
			Activity: "asking gcloud for projects",
			Tool:     "gcloud",
			Explain:  gcloudError,
			FetchCtx: func(ctx context.Context, _ []string) ([]string, error) {
				return gcpAllProjects(ctx, execRunner)
			},
		},
		Source{
			ID:       K8sNamespace,
			Class:    ClassPostConnection,
			Cost:     CostRemote,
			Activity: "asking kubectl for namespaces",
			Tool:     "kubectl",
			// The namespaces belong to the chosen cluster. This is the
			// dependency that used to be a hardcoded `if source != ...`.
			Needs: []string{"context"},
			FetchCtx: func(ctx context.Context, params []string) ([]string, error) {
				// The connector's --context does not select a cluster, so the
				// query is pointed at a kubeconfig copy instead. See
				// kubeconfig.go.
				env, cleanup, err := kubeEnvForContext(paramValue(params, "context"))
				if cleanup != nil {
					defer cleanup()
				}
				if err != nil {
					return nil, err
				}
				return k8sNamespaces(ctx, execRunner, env)
			},
		},

		// The four credential files and the docker context store. Each is a
		// file this machine already has, read without a network or a
		// credential, so all five are CostInstant. What differs between them
		// is where the chosen value goes:
		//
		//   - oci and azure have a flag that carries it (--profile,
		//     --subscription), both verified against the installed provider
		//     metadata.
		//   - alicloud has none. Its connector declares --access-key-id and
		//     friends and no --profile at all, so the profile travels in
		//     ALIBABA_CLOUD_PROFILE -- which is what Source.Env is for, and is
		//     why guessing a flag name here would have produced a picker that
		//     silently scanned the wrong account.
		//   - snowflake has no flag that takes a *connection name* either, and
		//     unlike alicloud there is no environment variable this connector
		//     reads to accept one. So this source does not offer connection
		//     names at all; see snowflakeAccountsFrom.
		//   - docker has no --context on docker, container or local, so a
		//     context travels in DOCKER_CONTEXT the way mql's own client
		//     resolves it.
		Source{
			ID:       OCIProfile,
			Class:    ClassEnumerated,
			Cost:     CostInstant,
			Activity: "reading ~/.oci/config",
			Tool:     "~/.oci/config",
			Prefer: func(values []string) (string, string) {
				if slices.Contains(values, ociDefaultProfile) {
					return ociDefaultProfile, "default"
				}
				return "", ""
			},
			Explain: missingFileExplain("~/.oci/config", "run: oci setup config"),
			Fetch:   func([]string) ([]string, error) { return ociProfiles() },
		},
		Source{
			ID:       AlicloudProfile,
			Class:    ClassEnumerated,
			Cost:     CostInstant,
			Activity: "reading ~/.alibabacloud/credentials",
			Tool:     "~/.alibabacloud/credentials",
			// The connector declares no --profile. Verified against the
			// installed alicloud provider metadata, which lists only
			// --access-key-id, --access-key-secret, --sts-token, --role-arn,
			// --role-session-name, --region, --regions and --filters.
			Env: AlicloudProfileEnv,
			Prefer: func(values []string) (string, string) {
				// The environment already names one, and that is what the
				// child would use whatever the picker showed.
				if env := os.Getenv(AlicloudProfileEnv); env != "" && slices.Contains(values, env) {
					return env, "from " + AlicloudProfileEnv
				}
				if slices.Contains(values, alicloudDefaultProfile) {
					return alicloudDefaultProfile, "default"
				}
				return "", ""
			},
			Explain: missingFileExplain("~/.alibabacloud/credentials",
				"pass --access-key-id and --access-key-secret instead"),
			Fetch: func([]string) ([]string, error) { return alicloudProfiles() },
		},
		Source{
			ID:       AzureSubscription,
			Class:    ClassEnumerated,
			Cost:     CostInstant,
			Activity: "reading ~/.azure/azureProfile.json",
			Tool:     "~/.azure/azureProfile.json",
			Prefer: func(values []string) (string, string) {
				// az marks exactly one subscription isDefault, which is the one
				// `az account show` reports and the one a user means.
				if id := azureDefaultSubscription(); id != "" && slices.Contains(values, id) {
					return id, "default"
				}
				return "", ""
			},
			Explain: missingFileExplain("~/.azure/azureProfile.json", "run: az login"),
			Fetch:   func([]string) ([]string, error) { return azureSubscriptions() },
		},
		Source{
			ID:       SnowflakeConnection,
			Class:    ClassEnumerated,
			Cost:     CostInstant,
			Activity: "reading ~/.snowflake/connections.toml",
			Tool:     "~/.snowflake/connections.toml",
			Prefer: func(values []string) (string, string) {
				if acct := snowflakeDefaultAccount(); acct != "" && slices.Contains(values, acct) {
					return acct, "default connection"
				}
				return "", ""
			},
			Explain: missingFileExplain("~/.snowflake/connections.toml",
				"type the account identifier instead"),
			Fetch: func([]string) ([]string, error) { return snowflakeAccounts() },
		},
		Source{
			ID:       DockerContext,
			Class:    ClassEnumerated,
			Cost:     CostInstant,
			Activity: "reading ~/.docker for contexts",
			Tool:     "~/.docker",
			// No --context on docker, container or local -- verified against
			// the installed os provider metadata, which declares only --sudo,
			// --id-detector, --disable-cache and --container-proxy for them.
			// DOCKER_CONTEXT is what the docker CLI itself reads, and what
			// mql's dockerclient honours when it builds a connection.
			Env: DockerContextEnv,
			Prefer: func(values []string) (string, string) {
				if cur := dockerCurrentContext(); cur != "" && slices.Contains(values, cur) {
					return cur, "current"
				}
				return "", ""
			},
			Explain: missingFileExplain("~/.docker", "type a context name instead"),
			Fetch:   func([]string) ([]string, error) { return dockerContexts() },
		},
	)
}
