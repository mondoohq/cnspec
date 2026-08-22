// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Package catalog answers "what can cnspec scan on this machine", and nothing
// else. It merges the compiled-in provider list with the installed one and
// flattens the result into connectors the launcher can put in a list.
//
// It depends on no other part of the launcher on purpose: the form engine, the
// delivery layer and the scan runner all read connector metadata, and a
// catalog that reached back into any of them would make that circular.
package catalog

import (
	"sort"
	"strings"

	"go.mondoo.com/mql/providers"
	"go.mondoo.com/mql/providers-sdk/v1/plugin"
)

// Connector is one selectable target in the interactive launcher. It is a
// flattened (provider, connector) pair, e.g. the "os" provider contributes
// "local", "ssh", "docker", ... as separate connectors, because to a user
// those are distinct things to scan.
type Connector struct {
	// Provider is the provider name that ships this connector (e.g. "os").
	Provider string
	// Name is the connector command word (e.g. "ssh", "aws", "docker"). This
	// is what gets placed on the command line: `cnspec scan <Name>`.
	Name string
	// Use is the usage hint including expected args (e.g. "ssh user@host").
	Use string
	// Short is the human description (e.g. "a remote system via SSH").
	Short string
	// Aliases are alternate command words for the same connector.
	Aliases []string
	// Category groups connectors in the list (e.g. "Cloud").
	Category string
	// Installed is true when the provider binary is already present locally.
	// When false, selecting it still works: the provider is downloaded on
	// first use (auto-update), we just surface the difference to the user.
	Installed bool

	// The fields below come from the connector's own declaration and drive the
	// input form. They are only populated for installed providers:
	// providers.DefaultProviders strips them, so an uninstalled connector has
	// no form to build and falls back to a free-text argument box.

	// Long is the connector's full help text. It is the only place several
	// providers document their sub-command shapes (`github org <ORG>`).
	Long string
	// MinArgs and MaxArgs bound the positional arguments the connector takes.
	// This is authoritative, unlike reading the Use string.
	MinArgs, MaxArgs uint
	// Flags are the connector's declared CLI flags.
	Flags []plugin.Flag
	// Discovery lists the connector's valid --discover targets. "all" and
	// "auto" are always additionally valid and are not repeated here.
	Discovery []string
}

// HasFormData reports whether the connector declared enough for a real input
// form. Only installed providers do.
//
// It counts only the flags a form could actually show. The launcher's
// genericFields skips every Hidden or Deprecated flag, so counting len(c.Flags)
// answers a question nobody asked: `device` declares nine flags and every one
// of them is marked Hidden or Deprecated, so newForm(device) yields zero fields
// while this said yes -- and the launcher drew an empty box instead of falling
// back to the free-text argument entry that is the honest screen for it.
func (c Connector) HasFormData() bool {
	return len(formableFlags(c)) > 0 || c.MaxArgs > 0 || len(c.Discovery) > 0
}

// DeclaresMetadata reports whether the connector carries its own declaration at
// all, as opposed to reaching the catalog from the static DefaultProviders list
// that strips it.
//
// This is the question HasFormData used to be asked and is a different one:
// "is MinArgs authoritative here" rather than "is there a form to draw". They
// only diverge for a connector whose every flag is hidden, but that connector
// exists, and conflating the two is what made the launcher confident about a
// screen it had nothing to put on.
func (c Connector) DeclaresMetadata() bool {
	return len(c.Flags) > 0 || c.MaxArgs > 0 || len(c.Discovery) > 0
}

// formableFlags are the flags a form can render: everything the connector
// declares except what it has marked Hidden or Deprecated, which is exactly
// what the launcher's genericFields filters on. The two live in different
// packages now, so they have to agree by hand: a flag this lets through but
// that one drops puts HasFormData back to promising a form with nothing in it.
func formableFlags(c Connector) []plugin.Flag {
	out := make([]plugin.Flag, 0, len(c.Flags))
	for _, fl := range c.Flags {
		if fl.Option&plugin.FlagOption_Hidden != 0 || fl.Option&plugin.FlagOption_Deprecated != 0 {
			continue
		}
		out = append(out, fl)
	}
	return out
}

// Summary is the connector's description with its leading article removed.
//
// Ninety-four of the hundred-odd descriptions begin "a " or "an ", which in a
// list is two or three columns of nothing: every row says the same word, and
// the part that distinguishes one connector from the next is what gets
// truncated at the right edge. "a Proxmox VE hypervisor" and "an Ansible
// playbook or role" carry exactly as much meaning without it.
//
// The provider's own text is left alone -- this is a presentation choice, and
// the detail pane still shows the sentence as written.
func (c Connector) Summary() string {
	for _, article := range []string{"a ", "an ", "A ", "An "} {
		if strings.HasPrefix(c.Short, article) {
			return c.Short[len(article):]
		}
	}
	return c.Short
}

// SearchText is the lowercased haystack used for fuzzy matching.
func (c Connector) SearchText() string {
	return strings.ToLower(c.Name + " " + c.Provider + " " + c.Short + " " + strings.Join(c.Aliases, " ") + " " + c.Category)
}

// ArgHint is the argument spec a connector declares in its usage string, e.g.
// "user@host" for `ssh user@host`. Empty when the connector takes no argument.
func (c Connector) ArgHint() string {
	return strings.TrimSpace(strings.TrimPrefix(c.Use, c.Name))
}

// RequiresArg reports whether the connector needs a positional argument.
//
// MinArgs is authoritative and is used whenever it is available. It is not
// available for a provider that is not installed locally, because the static
// default list drops it -- there we fall back to reading the usage string,
// where a bracketed spec ([host], [flags]) means optional by convention.
func (c Connector) RequiresArg() bool {
	// DeclaresMetadata rather than HasFormData: the question is whether MinArgs
	// came from the connector or from the flagless static list, and a connector
	// whose flags are all hidden still declared its argument count.
	if c.DeclaresMetadata() {
		return c.MinArgs > 0
	}
	h := c.ArgHint()
	return h != "" && !strings.HasPrefix(h, "[")
}

// Category ordering and titles. Anything not mapped lands in "Other".
const (
	CatHosts     = "Hosts & Devices"
	CatContainer = "Containers & Kubernetes"
	CatCloud     = "Cloud & Virtualization"
	CatIaC       = "Infrastructure as Code"
	CatIdentity  = "Identity & Access"
	CatSaaS      = "SaaS"
	CatNetwork   = "Network & Security Devices"
	CatDatabase  = "Databases"
	CatAI        = "AI & LLM"
	CatDev       = "Developer & Supply Chain"
	CatOther     = "Other"
)

// CategoryOrder is the display order of categories in the launcher.
var CategoryOrder = []string{
	CatHosts,
	CatContainer,
	CatCloud,
	CatIaC,
	CatIdentity,
	CatSaaS,
	CatNetwork,
	CatDatabase,
	CatAI,
	CatDev,
	CatOther,
}

// providerCategory maps a provider name to a category. The "os" provider is
// special-cased per-connector in Categorize().
var providerCategory = map[string]string{
	// Cloud & virtualization
	"aws": CatCloud, "azure": CatCloud, "gcp": CatCloud, "oci": CatCloud,
	"alicloud": CatCloud, "digitalocean": CatCloud, "hetzner": CatCloud,
	"equinix": CatCloud, "stackit": CatCloud, "openstack": CatCloud,
	"proxmox": CatCloud, "nutanix": CatCloud, "vcd": CatCloud, "vsphere": CatCloud,
	"ibm": CatCloud, "hcp": CatCloud,

	// Containers & Kubernetes
	"k8s": CatContainer, "helm": CatContainer, "kustomize": CatContainer,
	"portainer": CatContainer,

	// Infrastructure as Code
	"terraform": CatIaC, "cloudformation": CatIaC, "bicep": CatIaC,
	"ansible": CatIaC,

	// Identity & Access: the systems that decide who can sign in, and where
	// credentials live. Separated from SaaS because "review our identity
	// provider" and "review the tools we subscribe to" are different jobs, and
	// twenty-six connectors in one list served neither.
	//
	// ms365 and google-workspace belong here rather than under SaaS despite
	// being productivity suites: what a scan of either actually inspects is the
	// tenant's directory, its sign-in policy and its admin roles.
	"okta": CatIdentity, "auth0": CatIdentity, "keycloak": CatIdentity,
	"jumpcloud": CatIdentity, "activedirectory": CatIdentity,
	"bitwarden": CatIdentity,
	"ms365":     CatIdentity, "google-workspace": CatIdentity,

	// SaaS: the accounts an organisation holds with someone else.
	"slack": CatSaaS, "github": CatSaaS, "gitlab": CatSaaS, "atlassian": CatSaaS,
	"snowflake": CatSaaS, "databricks": CatSaaS,
	"mondoo": CatSaaS, "datadog": CatSaaS, "grafana": CatSaaS, "vercel": CatSaaS,
	"cloudflare": CatSaaS, "nextdns": CatSaaS, "tailscale": CatSaaS,
	"mongodbatlas": CatSaaS, "netlify": CatSaaS,
	"dropbox": CatSaaS, "zoom": CatSaaS,

	// Apple device management. Iru is Kandji renamed, and sits with Jamf
	// because they are the same job.
	"jamf": CatSaaS, "iru": CatSaaS,

	// Network & security devices
	"network": CatNetwork, "networkdiscovery": CatNetwork,
	"networkdevices": CatNetwork, "fortios": CatNetwork, "panos": CatNetwork,
	"junos": CatNetwork, "arista": CatNetwork, "mikrotik": CatNetwork,
	"bigip": CatNetwork, "checkpoint": CatNetwork, "opcua": CatNetwork,
	"ipmi": CatNetwork, "redfish": CatNetwork, "nmap": CatNetwork,
	"unifi": CatNetwork, "shodan": CatNetwork, "ipinfo": CatNetwork,

	// Databases
	"mongo": CatDatabase, "mssql": CatDatabase, "mysqldb": CatDatabase,
	"postgresdb": CatDatabase, "redisdb": CatDatabase, "cassandra": CatDatabase,
	"elasticsearch": CatDatabase, "opensearch": CatDatabase, "weaviate": CatDatabase,
	"clickhouse": CatDatabase, "clickhousedb": CatDatabase,
	"clickhousecloud": CatDatabase, "oracledb": CatDatabase, "neon": CatDatabase,
	"db2": CatDatabase,

	// AI & LLM
	"ai": CatAI, "openai": CatAI, "claude": CatAI, "mistral": CatAI,
	"ollama": CatAI, "huggingface": CatAI, "together": CatAI, "vllm": CatAI,
	"anthropic": CatAI,

	// Developer & supply chain
	"depsdev": CatDev, "sbom": CatDev, "artifactory": CatDev,
}

// osConnectorCategory splits the multi-purpose "os" provider connectors into
// the right buckets.
var osConnectorCategory = map[string]string{
	"local":      CatHosts,
	"ssh":        CatHosts,
	"winrm":      CatHosts,
	"vagrant":    CatHosts,
	"device":     CatHosts,
	"filesystem": CatHosts,
	"docker":     CatContainer,
	"container":  CatContainer,
}

// ExcludedProviders ship user-visible connectors that are not scan targets:
// they replay or fake a connection for debugging. Offering them in a launcher
// whose whole premise is "pick something to secure" would be noise.
var ExcludedProviders = map[string]bool{
	"mock":      true,
	"recording": true,
}

func Categorize(provider, connector string) string {
	if provider == "os" {
		if c, ok := osConnectorCategory[connector]; ok {
			return c
		}
		return CatHosts
	}
	if c, ok := providerCategory[provider]; ok {
		return c
	}
	return CatOther
}

// BuildCatalog returns every selectable connector, merging two sources:
//
//   - providers.DefaultProviders, the static list compiled into the binary. It
//     is what cnspec can name in an air-gapped environment, so it is the floor.
//   - providers.ListActive(), the providers actually installed on this machine.
//     These win, because the static list lags behind what ships -- whole
//     categories (every database provider, for one) exist only here.
//
// It never fails hard: if the installed set can't be read we still return the
// static catalog.
func BuildCatalog() []Connector {
	type source struct {
		provider  *plugin.Provider
		installed bool
	}
	merged := make(map[string]source, len(providers.DefaultProviders))

	for name, p := range providers.DefaultProviders {
		if p.Provider == nil {
			continue
		}
		merged[name] = source{provider: p.Provider}
	}

	// ListActive is keyed by provider ID (a full Go import path), so key the
	// merge off the provider's Name instead -- that is what the catalog and the
	// command line speak.
	if active, err := providers.ListActive(); err == nil {
		for _, p := range active {
			if p.Provider == nil || p.Name == "" {
				continue
			}
			merged[p.Name] = source{provider: p.Provider, installed: true}
		}
	}

	var out []Connector
	for name, src := range merged {
		if ExcludedProviders[name] {
			continue
		}
		out = append(out, ConnectorsOf(name, src.provider, src.installed)...)
	}

	sort.Slice(out, func(i, j int) bool {
		ci, cj := categoryIndex(out[i].Category), categoryIndex(out[j].Category)
		if ci != cj {
			return ci < cj
		}
		// Keep "local" first inside Hosts so the common case is at the top.
		iLocal, jLocal := out[i].Name == "local", out[j].Name == "local"
		if iLocal != jLocal {
			return iLocal
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// ConnectorsOf flattens a provider's connectors into catalog entries. It is
// shared by the initial catalog build and by the on-demand installer, so a
// connector refreshed after an install is shaped exactly like one that was
// there from the start.
func ConnectorsOf(name string, p *plugin.Provider, installed bool) []Connector {
	if p == nil {
		return nil
	}
	// "core" and similar have no connectors; they contribute nothing.
	out := make([]Connector, 0, len(p.Connectors))
	for _, conn := range p.Connectors {
		if conn.IsHidden || conn.Name == "" {
			continue
		}
		out = append(out, Connector{
			Provider:  name,
			Name:      conn.Name,
			Use:       firstNonEmpty(conn.Use, conn.Name),
			Short:     conn.Short,
			Aliases:   conn.Aliases,
			Category:  Categorize(name, conn.Name),
			Installed: installed,
			Long:      conn.Long,
			MinArgs:   conn.MinArgs,
			MaxArgs:   conn.MaxArgs,
			Flags:     conn.Flags,
			Discovery: conn.Discovery,
		})
	}
	return out
}

func categoryIndex(cat string) int {
	for i, c := range CategoryOrder {
		if c == cat {
			return i
		}
	}
	return len(CategoryOrder)
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
