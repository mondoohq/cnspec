// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

// Action is one thing cnspec can do with a target, e.g. "scan" or "shell".
type Action struct {
	// Name is the subcommand word (e.g. "scan").
	Name string
	// Short describes the action for the picker.
	Short string
	// supported, when non-empty, restricts the action to those connector
	// names. An empty set means the action applies to every connector.
	supported map[string]bool
}

// AppliesTo reports whether this action can be used with the given connector.
func (a Action) AppliesTo(connectorName string) bool {
	if len(a.supported) == 0 {
		return true
	}
	return a.supported[connectorName]
}

func set(names ...string) map[string]bool {
	m := make(map[string]bool, len(names))
	for _, n := range names {
		m[n] = true
	}
	return m
}

// Actions is the ordered list of things a user can launch from the TUI. The
// supported-connector sets mirror the SupportedConnectors declared when these
// commands are attached in apps/cnspec/cmd/root.go, so the launcher only ever
// offers an action that will actually run.
var Actions = []Action{
	{
		Name:  "scan",
		Short: "Run security & compliance policies and score the target",
	},
	{
		Name:  "shell",
		Short: "Open an interactive MQL shell to explore the target",
	},
	{
		Name:  "run",
		Short: "Run a single MQL query against the target",
	},
	{
		Name:  "discover",
		Short: "List the assets cnspec can see through this connection",
	},
	{
		Name:      "vuln",
		Short:     "Check the target for known vulnerabilities (CVEs)",
		supported: set("docker", "container", "filesystem", "local", "ssh", "vagrant", "winrm", "vsphere", "sbom"),
	},
	{
		Name:      "sbom",
		Short:     "Generate a software bill of materials (SBOM)",
		supported: set("docker", "container", "filesystem", "local", "ssh", "vagrant", "winrm", "sbom"),
	},
	{
		Name:      "aibom",
		Short:     "Generate an AI bill of materials (AIBOM)",
		supported: set("local", "docker", "container", "filesystem", "ssh", "vagrant", "winrm", "ollama", "huggingface", "aws", "gcp", "azure"),
	},
}

// ActionsFor returns the actions applicable to a connector, preserving order.
func ActionsFor(connectorName string) []Action {
	var out []Action
	for _, a := range Actions {
		if a.AppliesTo(connectorName) {
			out = append(out, a)
		}
	}
	return out
}
