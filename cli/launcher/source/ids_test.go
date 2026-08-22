// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package source

import (
	"strings"
	"testing"
)

// declaredSourceIDs is the whole namespace from source_ids.go, listed once so
// the checks below are about the set rather than about any one id.
//
// Keeping the list here rather than deriving it is the point: a constant that
// is declared and then never registered, named or spelled consistently is
// exactly what this catches, and deriving the list from the registry would make
// the test agree with whatever happened to be there.
var declaredSourceIDs = []string{
	AWSProfile,
	KubeContext,
	SSHHost,
	DockerContainer,
	DockerImage,
	GCPProject,
	GCPProjectAll,
	GCPZone,
	AzureSubscription,
	OCIProfile,
	AlicloudProfile,
	SnowflakeConnection,
	DockerContext,
	K8sNamespace,
	GitHubToken,
	GitLabToken,
	SlackToken,
	CloudflareToken,
	DigitalOceanToken,
	HetznerToken,
	OktaToken,
	DiscoverK8sNamespaces,
	DiscoverGitHubRepos,
	DiscoverGitLabProjects,
	DiscoverGitLabGroups,
	DiscoverAzureSubscriptions,
	DiscoverGCPProjects,
	DiscoverAWSAccounts,
	DiscoverOCITenancy,
	DiscoverAlicloudAccounts,
	DiscoverDigitalOceanDBs,
	DiscoverNeonProjects,
	DiscoverNetlifySites,
	DiscoverVercelProjects,
	DiscoverAtlasProjects,
	DiscoverHCPProjects,
	DiscoverClaudeWorkspaces,
	DiscoverMSSQLDatabases,
	DiscoverMySQLDatabases,
	DiscoverPostgresDatabases,
	DiscoverSnowflakeDatabases,
}

// Two constants with the same value are two names for one cache entry, and the
// second source registered under it silently replaces the first.
func TestSourceIDsAreUnique(t *testing.T) {
	seen := map[string]int{}
	for _, id := range declaredSourceIDs {
		seen[id]++
	}
	for id, n := range seen {
		if n > 1 {
			t.Errorf("%q is declared %d times", id, n)
		}
	}
}

// A source id is a cache key prefix and appears in test failures, so it has to
// read as one thing: dotted, lowercase, no spaces.
func TestSourceIDsAreWellFormed(t *testing.T) {
	for _, id := range declaredSourceIDs {
		if id == "" {
			t.Error("an empty source id matches every unregistered lookup")
			continue
		}
		if id != strings.ToLower(id) || strings.ContainsAny(id, " \t") {
			t.Errorf("%q should be lowercase with no spaces", id)
		}
		if !strings.Contains(id, ".") {
			t.Errorf("%q should name its provider, e.g. \"aws.profile\"", id)
		}
	}
}

// The discovery constants and the builder must agree, or a source registered
// through one is invisible to a form naming the other.
func TestDiscoverSourceIDMatchesTheConstants(t *testing.T) {
	cases := []struct {
		connector, target, want string
	}{
		{"k8s", "namespaces", DiscoverK8sNamespaces},
		{"github", "repos", DiscoverGitHubRepos},
		{"gitlab", "projects", DiscoverGitLabProjects},
		{"gitlab", "groups", DiscoverGitLabGroups},
		{"azure", "subscriptions", DiscoverAzureSubscriptions},
		{"gcp", "projects", DiscoverGCPProjects},
		{"aws", "accounts", DiscoverAWSAccounts},
		{"oci", "tenancy", DiscoverOCITenancy},
		{"alicloud", "accounts", DiscoverAlicloudAccounts},
		{"digitalocean", "databases", DiscoverDigitalOceanDBs},
		{"neon", "projects", DiscoverNeonProjects},
		{"netlify", "sites", DiscoverNetlifySites},
		{"vercel", "projects", DiscoverVercelProjects},
		{"mongodbatlas", "projects", DiscoverAtlasProjects},
		{"hcp", "projects", DiscoverHCPProjects},
		{"claude", "workspaces", DiscoverClaudeWorkspaces},
		{"mssql", "databases", DiscoverMSSQLDatabases},
		{"mysqldb", "databases", DiscoverMySQLDatabases},
		{"postgresdb", "databases", DiscoverPostgresDatabases},
		{"snowflake", "databases", DiscoverSnowflakeDatabases},
	}
	for _, c := range cases {
		if got := DiscoverSourceID(c.connector, c.target); got != c.want {
			t.Errorf("DiscoverSourceID(%q, %q) = %q, want %q", c.connector, c.target, got, c.want)
		}
	}
}
