// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package generate

import (
	"regexp"
	"strings"
)

// platformToProvider maps common asset.platform / asset.family values to the
// mql provider that models them. It is intentionally conservative: an unknown
// platform yields no provider rather than a wrong guess.
var platformToProvider = map[string]string{
	"aws":        "aws",
	"gcp":        "gcp",
	"gcp-org":    "gcp",
	"google":     "gcp",
	"azure":      "azure",
	"azurerm":    "azure",
	"kubernetes": "k8s",
	"k8s":        "k8s",
	"github":     "github",
	"gitlab":     "gitlab",
	"ms365":      "ms365",
	"microsoft":  "ms365",
	"oci":        "oci",
	"okta":       "okta",
	"slack":      "slack",
	"terraform":  "terraform",
	"vsphere":    "vsphere",
	"vmware":     "vsphere",
	"arista":     "arista",
	"atlassian":  "atlassian",
	"snowflake":  "snowflake",
	"cloudflare": "cloudflare",
	// operating-system families all resolve to the os provider
	"linux":       "os",
	"unix":        "os",
	"windows":     "os",
	"darwin":      "os",
	"macos":       "os",
	"debian":      "os",
	"ubuntu":      "os",
	"redhat":      "os",
	"rhel":        "os",
	"centos":      "os",
	"fedora":      "os",
	"suse":        "os",
	"alpine":      "os",
	"amazonlinux": "os",
	"rockylinux":  "os",
	"almalinux":   "os",
}

// knownProviders is the set of provider names we recognize as a leading token in
// a filter expression's resource path or in a query UID.
var knownProviders = func() map[string]bool {
	m := map[string]bool{}
	for _, p := range platformToProvider {
		m[p] = true
	}
	return m
}()

var quotedRe = regexp.MustCompile(`"([^"]+)"|'([^']+)'`)

// ResolveProvider determines the target provider for a check, preferring the
// authoritative filter expressions and falling back to hints in the UID. It
// returns the provider name (empty if undetermined) and the platform strings it
// observed, for reporting.
func ResolveProvider(c Check) (provider string, platforms []string) {
	seen := map[string]bool{}
	addPlatform := func(p string) {
		p = strings.ToLower(strings.TrimSpace(p))
		if p != "" && !seen[p] {
			seen[p] = true
			platforms = append(platforms, p)
		}
	}

	for _, f := range c.Filters {
		// 1. resource-path prefix, e.g. "aws.ec2.instances" in the filter
		if p := leadingProvider(f); p != "" {
			return p, platforms
		}
		// 2. quoted platform/family literals, e.g. asset.platform == "ubuntu"
		// or a hyphenated platform id like "aws-eks-cluster".
		for _, m := range quotedRe.FindAllStringSubmatch(f, -1) {
			lit := m[1]
			if lit == "" {
				lit = m[2]
			}
			addPlatform(lit)
			if p := platformForLiteral(lit); p != "" {
				provider = p
			}
		}
	}
	if provider != "" {
		return provider, platforms
	}

	// 3. UID hint, e.g. "mondoo-aws-security-..." -> aws
	for _, tok := range strings.FieldsFunc(strings.ToLower(c.UID), func(r rune) bool {
		return r == '-' || r == '_' || r == '.' || r == '/'
	}) {
		if knownProviders[tok] {
			return tok, platforms
		}
		if p, ok := platformToProvider[tok]; ok {
			return p, platforms
		}
	}

	return "", platforms
}

// platformForLiteral maps a platform/family literal to a provider. It tries an
// exact match first, then each hyphen-delimited segment, so ids like
// "aws-eks-cluster" resolve to "aws".
func platformForLiteral(lit string) string {
	lit = strings.ToLower(strings.TrimSpace(lit))
	if p, ok := platformToProvider[lit]; ok {
		return p
	}
	for _, seg := range strings.Split(lit, "-") {
		if p, ok := platformToProvider[seg]; ok {
			return p
		}
		if knownProviders[seg] {
			return seg
		}
	}
	return ""
}

// leadingProvider returns the provider name when a filter references a
// provider-scoped resource directly (e.g. "aws.ec2.instances.any(...)").
func leadingProvider(filter string) string {
	filter = strings.TrimSpace(filter)
	// grab the first identifier token
	i := 0
	for i < len(filter) {
		r := filter[i]
		if r == '.' || r == ' ' || r == '(' || r == '[' || r == '=' || r == '!' || r == '<' || r == '>' {
			break
		}
		i++
	}
	head := strings.ToLower(filter[:i])
	if head == "asset" || head == "platform" || head == "mondoo" {
		return ""
	}
	if knownProviders[head] {
		return head
	}
	return ""
}
