// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	"sort"
	"strings"

	"go.mondoo.com/mql/v13/providers"
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
}

// searchText is the lowercased haystack used for fuzzy matching.
func (c Connector) searchText() string {
	return strings.ToLower(c.Name + " " + c.Provider + " " + c.Short + " " + strings.Join(c.Aliases, " ") + " " + c.Category)
}

// Category ordering and titles. Anything not mapped lands in "Other".
const (
	catHosts     = "Hosts & Devices"
	catContainer = "Containers & Kubernetes"
	catCloud     = "Cloud & Virtualization"
	catIaC       = "Infrastructure as Code"
	catSaaS      = "Identity & SaaS"
	catNetwork   = "Network & Security Devices"
	catDatabase  = "Databases"
	catAI        = "AI & LLM"
	catDev       = "Developer & Supply Chain"
	catOther     = "Other"
)

// categoryOrder is the display order of categories in the launcher.
var categoryOrder = []string{
	catHosts,
	catContainer,
	catCloud,
	catIaC,
	catSaaS,
	catNetwork,
	catDatabase,
	catAI,
	catDev,
	catOther,
}

// providerCategory maps a provider name to a category. The "os" provider is
// special-cased per-connector in categorize().
var providerCategory = map[string]string{
	// Cloud & virtualization
	"aws": catCloud, "azure": catCloud, "gcp": catCloud, "oci": catCloud,
	"alicloud": catCloud, "digitalocean": catCloud, "hetzner": catCloud,
	"equinix": catCloud, "stackit": catCloud, "openstack": catCloud,
	"proxmox": catCloud, "nutanix": catCloud, "vcd": catCloud, "vsphere": catCloud,
	"ibm": catCloud, "hcp": catCloud,

	// Containers & Kubernetes
	"k8s": catContainer, "helm": catContainer, "kustomize": catContainer,
	"portainer": catContainer,

	// Infrastructure as Code
	"terraform": catIaC, "cloudformation": catIaC, "bicep": catIaC,
	"ansible": catIaC,

	// Identity & SaaS
	"okta": catSaaS, "ms365": catSaaS, "google-workspace": catSaaS,
	"slack": catSaaS, "github": catSaaS, "gitlab": catSaaS, "atlassian": catSaaS,
	"jamf": catSaaS, "snowflake": catSaaS, "databricks": catSaaS,
	"mondoo": catSaaS, "datadog": catSaaS, "grafana": catSaaS, "vercel": catSaaS,
	"cloudflare": catSaaS, "nextdns": catSaaS, "tailscale": catSaaS,
	"mongodbatlas": catSaaS,

	// Network & security devices
	"network": catNetwork, "networkdiscovery": catNetwork,
	"networkdevices": catNetwork, "fortios": catNetwork, "panos": catNetwork,
	"junos": catNetwork, "arista": catNetwork, "mikrotik": catNetwork,
	"bigip": catNetwork, "checkpoint": catNetwork, "opcua": catNetwork,
	"ipmi": catNetwork, "redfish": catNetwork, "nmap": catNetwork,
	"unifi": catNetwork, "shodan": catNetwork, "ipinfo": catNetwork,

	// Databases
	"mongo": catDatabase, "mssql": catDatabase, "mysqldb": catDatabase,
	"postgresdb": catDatabase, "redisdb": catDatabase, "cassandra": catDatabase,
	"elasticsearch": catDatabase, "opensearch": catDatabase, "weaviate": catDatabase,

	// AI & LLM
	"ai": catAI, "openai": catAI, "claude": catAI, "mistral": catAI,
	"ollama": catAI, "huggingface": catAI, "together": catAI, "vllm": catAI,

	// Developer & supply chain
	"depsdev": catDev,
}

// osConnectorCategory splits the multi-purpose "os" provider connectors into
// the right buckets.
var osConnectorCategory = map[string]string{
	"local":      catHosts,
	"ssh":        catHosts,
	"winrm":      catHosts,
	"vagrant":    catHosts,
	"device":     catHosts,
	"filesystem": catHosts,
	"docker":     catContainer,
	"container":  catContainer,
}

func categorize(provider, connector string) string {
	if provider == "os" {
		if c, ok := osConnectorCategory[connector]; ok {
			return c
		}
		return catHosts
	}
	if c, ok := providerCategory[provider]; ok {
		return c
	}
	return catOther
}

// BuildCatalog returns the full list of selectable connectors, merging the
// static catalog of everything cnspec supports (providers.DefaultProviders)
// with the set of providers already installed locally (so we can flag them).
// It never fails hard: if the installed set can't be read we still return the
// static catalog.
func BuildCatalog() []Connector {
	installed := map[string]bool{}
	if active, err := providers.ListActive(); err == nil {
		for name := range active {
			installed[name] = true
		}
	}

	var out []Connector
	for name, p := range providers.DefaultProviders {
		if p.Provider == nil {
			continue
		}
		// "core" and similar have no connectors; skip them.
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
				Category:  categorize(name, conn.Name),
				Installed: installed[name],
			})
		}
	}

	sort.Slice(out, func(i, j int) bool {
		ci, cj := categoryIndex(out[i].Category), categoryIndex(out[j].Category)
		if ci != cj {
			return ci < cj
		}
		// Keep "local" first inside Hosts so the common case is at the top.
		if out[i].Name == "local" {
			return true
		}
		if out[j].Name == "local" {
			return false
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func categoryIndex(cat string) int {
	for i, c := range categoryOrder {
		if c == cat {
			return i
		}
	}
	return len(categoryOrder)
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
