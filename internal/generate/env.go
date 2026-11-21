// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package generate

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

// AgentEnvPassthroughVar names extra environment variables to forward to the
// agent, comma-separated. It is the escape hatch for a setup the allowlist
// below does not anticipate (a corporate proxy variable, a vendor-specific auth
// var); `*` forwards the caller's entire environment, which is what cnspec used
// to do unconditionally.
const AgentEnvPassthroughVar = "CNSPEC_AGENT_ENV"

// agentEnvAllowed is the set of variables the agent subprocess inherits.
//
// It exists because the alternative is what this code used to do: hand the
// coding agent the operator's entire environment. That is every cloud
// credential the shell is carrying — AWS_SECRET_ACCESS_KEY, MONDOO_API_TOKEN,
// SSH_AUTH_SOCK and the rest — passed to a program that runs tools and acts on
// text an untrusted policy bundle supplied. The agent needs to start, find its
// own config, and reach its own API; it does not need to be able to sign an AWS
// request.
//
// Everything here is either "the process cannot run without it" or "the agent's
// own account", never a credential for a system cnspec scans.
var agentEnvAllowed = map[string]bool{
	// process basics
	"PATH": true, "HOME": true, "USER": true, "LOGNAME": true, "SHELL": true,
	"TMPDIR": true, "TMP": true, "TEMP": true, "PWD": true,
	// terminal and locale, so the agent's own output is readable
	"TERM": true, "COLORTERM": true, "NO_COLOR": true, "CLICOLOR": true,
	"LANG": true, "LC_ALL": true, "LC_CTYPE": true, "TZ": true,
	// where the agent keeps its own configuration
	"XDG_CONFIG_HOME": true, "XDG_CACHE_HOME": true, "XDG_DATA_HOME": true,
	"XDG_STATE_HOME": true, "XDG_RUNTIME_DIR": true,
	// reaching its own API through the operator's network
	"HTTP_PROXY": true, "HTTPS_PROXY": true, "ALL_PROXY": true, "NO_PROXY": true,
	"SSL_CERT_FILE": true, "SSL_CERT_DIR": true, "NODE_EXTRA_CA_CERTS": true,
	"REQUESTS_CA_BUNDLE": true, "CURL_CA_BUNDLE": true,
	// Windows: a process started without these cannot resolve system DLLs or
	// its own user profile
	"SYSTEMROOT": true, "SYSTEMDRIVE": true, "WINDIR": true, "COMSPEC": true,
	"PATHEXT": true, "APPDATA": true, "LOCALAPPDATA": true, "PROGRAMDATA": true,
	"PROGRAMFILES": true, "PROGRAMFILES(X86)": true, "USERPROFILE": true,
	"USERNAME": true, "HOMEDRIVE": true, "HOMEPATH": true,
	"PROCESSOR_ARCHITECTURE": true, "NUMBER_OF_PROCESSORS": true, "OS": true,
}

// agentEnvAllowedPrefixes covers each supported agent's own auth and settings.
// These are the agent's credentials, which is the whole premise of the design:
// cnspec ships no model access and the agent brings its own.
var agentEnvAllowedPrefixes = []string{
	"ANTHROPIC_",
	"CLAUDE_",
	"OPENAI_",
	"CODEX_",
	"KIMI_",
	"MOONSHOT_",
	"DEEPSEEK_",
	// the agent CLIs are Node programs; these decide how that runtime starts
	"NODE_OPTIONS",
	"NPM_CONFIG_",
	// the generator's own knobs (CNSPEC_AGENT_*_BIN and the passthrough list)
	"CNSPEC_AGENT_",
}

// agentEnv builds the environment for an agent invocation from environ (the
// caller's, in os.Environ form). Order is preserved so a duplicated key still
// resolves the way os/exec resolves it.
func agentEnv(environ []string) []string {
	extra, all := passthroughFrom(environ)
	if all {
		return environ
	}

	out := make([]string, 0, 32)
	for _, kv := range environ {
		name, _, ok := strings.Cut(kv, "=")
		if !ok {
			continue
		}
		if agentEnvAllows(name, extra) {
			out = append(out, kv)
		}
	}
	return out
}

// agentEnvAllows reports whether one variable is forwarded. Matching is
// case-insensitive: Windows environment names are, and `http_proxy` is as
// common as `HTTP_PROXY` everywhere else.
func agentEnvAllows(name string, extra map[string]bool) bool {
	upper := strings.ToUpper(name)
	if agentEnvAllowed[upper] || extra[upper] {
		return true
	}
	for _, prefix := range agentEnvAllowedPrefixes {
		if strings.HasPrefix(upper, prefix) {
			return true
		}
	}
	return false
}

// passthroughFrom reads the operator's additions from environ itself, rather
// than from os.Getenv, so the whole function stays testable without mutating
// the test process's environment.
func passthroughFrom(environ []string) (extra map[string]bool, all bool) {
	extra = map[string]bool{}
	for _, kv := range environ {
		name, value, ok := strings.Cut(kv, "=")
		if !ok || !strings.EqualFold(name, AgentEnvPassthroughVar) {
			continue
		}
		for _, item := range strings.Split(value, ",") {
			switch item = strings.TrimSpace(item); item {
			case "":
			case "*":
				all = true
			default:
				extra[strings.ToUpper(item)] = true
			}
		}
	}
	return extra, all
}

// agentEnvHint explains a failed agent run that may be the allowlist's doing.
//
// Claude Code on Amazon Bedrock or Google Vertex AI authenticates with the
// cloud's own credentials (AWS_PROFILE, AWS_REGION, GOOGLE_APPLICATION_CREDENTIALS,
// CLOUD_ML_REGION, ...). The CLAUDE_ prefix forwards the switch that selects
// them, but those credentials are exactly what the allowlist withholds, so the
// agent fails with an auth error that never mentions cnspec. They are not
// forwarded automatically: the same variables are often the credentials of an
// account cnspec scans, and handing those to the agent is the decision the
// allowlist leaves to the operator. The hint names the withheld variables (never
// their values) and the setting that forwards them.
func agentEnvHint(environ []string) string {
	extra, all := passthroughFrom(environ)
	if all {
		return ""
	}
	values := map[string]string{}
	for _, kv := range environ {
		if name, value, ok := strings.Cut(kv, "="); ok {
			values[strings.ToUpper(name)] = value
		}
	}
	truthy := func(name string) bool {
		v := strings.TrimSpace(values[name])
		return v != "" && v != "0" && !strings.EqualFold(v, "false")
	}

	var prefixes []string
	var via string
	switch {
	case truthy("CLAUDE_CODE_USE_BEDROCK"):
		prefixes, via = []string{"AWS_"}, "Claude Code on Bedrock"
	case truthy("CLAUDE_CODE_USE_VERTEX"):
		prefixes, via = []string{"GOOGLE_", "GCLOUD_", "CLOUDSDK_", "CLOUD_ML_", "VERTEX_"}, "Claude Code on Vertex AI"
	}

	var withheld []string
	for name := range values {
		if agentEnvAllows(name, extra) {
			continue
		}
		for _, p := range prefixes {
			if strings.HasPrefix(name, p) {
				withheld = append(withheld, name)
				break
			}
		}
	}
	sort.Strings(withheld)

	if via != "" && len(withheld) > 0 {
		return fmt.Sprintf("%s needs credentials cnspec does not pass to the agent (%s); forward them with %s=%s",
			via, strings.Join(withheld, ", "), AgentEnvPassthroughVar, strings.Join(withheld, ","))
	}
	return fmt.Sprintf("the agent runs with a restricted environment; if it needs a variable cnspec withheld, name it in %s (comma-separated)",
		AgentEnvPassthroughVar)
}

// agentEnviron is agentEnv over this process's environment.
func agentEnviron() []string { return agentEnv(os.Environ()) }
