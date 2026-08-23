// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package generate

import (
	"slices"
	"strings"
	"testing"
)

// TestAgentEnvWithholdsCredentials is the whole point of the allowlist: the
// agent is a program that runs tools on text an untrusted bundle supplied, and
// it used to be handed every credential in the operator's shell.
func TestAgentEnvWithholdsCredentials(t *testing.T) {
	environ := []string{
		"PATH=/usr/bin",
		"HOME=/home/op",
		"TERM=xterm-256color",
		"LANG=en_US.UTF-8",
		"ANTHROPIC_API_KEY=agent-own-auth",
		"CNSPEC_AGENT_CLAUDE_BIN=/opt/claude",
		"HTTPS_PROXY=http://proxy:3128",
		// none of these belong to the agent
		"AWS_SECRET_ACCESS_KEY=leaked",
		"AWS_SESSION_TOKEN=leaked",
		"MONDOO_API_TOKEN=leaked",
		"SSH_AUTH_SOCK=/tmp/agent.sock",
		"GITHUB_TOKEN=leaked",
		"KUBECONFIG=/home/op/.kube/config",
		"AZURE_CLIENT_SECRET=leaked",
		"GOOGLE_APPLICATION_CREDENTIALS=/home/op/gcp.json",
		"VAULT_TOKEN=leaked",
	}

	got := agentEnv(environ)

	for _, want := range []string{
		"PATH=/usr/bin", "HOME=/home/op", "TERM=xterm-256color", "LANG=en_US.UTF-8",
		"ANTHROPIC_API_KEY=agent-own-auth", "CNSPEC_AGENT_CLAUDE_BIN=/opt/claude",
		"HTTPS_PROXY=http://proxy:3128",
	} {
		if !slices.Contains(got, want) {
			t.Errorf("the agent needs %q, but it was dropped", want)
		}
	}
	for _, kv := range got {
		name, _, _ := strings.Cut(kv, "=")
		switch name {
		case "AWS_SECRET_ACCESS_KEY", "AWS_SESSION_TOKEN", "MONDOO_API_TOKEN",
			"SSH_AUTH_SOCK", "GITHUB_TOKEN", "KUBECONFIG", "AZURE_CLIENT_SECRET",
			"GOOGLE_APPLICATION_CREDENTIALS", "VAULT_TOKEN":
			t.Errorf("credential %q was forwarded to the agent", name)
		}
	}
}

func TestAgentEnvPassthrough(t *testing.T) {
	environ := []string{"PATH=/usr/bin", "AWS_SECRET_ACCESS_KEY=leaked", "CORP_CA=/etc/ca.pem"}

	// named passthrough forwards exactly what was asked for, nothing more
	got := agentEnv(append(slices.Clone(environ), AgentEnvPassthroughVar+"=CORP_CA"))
	if !slices.Contains(got, "CORP_CA=/etc/ca.pem") {
		t.Errorf("passthrough did not forward CORP_CA: %v", got)
	}
	if slices.Contains(got, "AWS_SECRET_ACCESS_KEY=leaked") {
		t.Errorf("passthrough forwarded an unnamed variable: %v", got)
	}

	// `*` restores the old inherit-everything behavior, for whoever needs it
	all := agentEnv(append(slices.Clone(environ), AgentEnvPassthroughVar+"=*"))
	if !slices.Contains(all, "AWS_SECRET_ACCESS_KEY=leaked") {
		t.Errorf("`*` should inherit the whole environment: %v", all)
	}
}

func TestAgentEnvIsCaseInsensitive(t *testing.T) {
	// http_proxy is as common as HTTP_PROXY, and Windows names are
	// case-insensitive throughout
	got := agentEnv([]string{"http_proxy=http://proxy:3128", "Path=C:\\bin"})
	if len(got) != 2 {
		t.Fatalf("expected both variables to be forwarded, got %v", got)
	}
}
