// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package generate

import (
	"os"
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

// agentEnviron is agentEnv over this process's environment.
func agentEnviron() []string { return agentEnv(os.Environ()) }
