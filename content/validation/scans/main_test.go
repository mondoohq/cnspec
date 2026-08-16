// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Package scans holds the provider-backed suites that answer one question about
// the policy bundles in content/: when this check meets this input, does it
// reach the verdict we claim?
//
// Every suite here runs a real scan through the same scanner cnspec ships, so
// they need providers installed and are slower than a unit test. The static
// checks that only read the YAML live in ../compliance, deliberately in a
// separate package so they do not inherit this package's provider provisioning.
//
// See ../README.md for what each suite covers and how to run it.
package scans

import (
	"os"
	"path/filepath"
	"testing"

	"go.mondoo.com/mql/v13/logger"
	"go.mondoo.com/mql/v13/providers"
)

func init() {
	logger.Set("info")
}

// contentDir is the policy directory these suites validate, relative to this
// package: content/validation/scans -> content.
const contentDir = "../.."

// bundlePath resolves a policy bundle filename to a path this package can open.
// Suites carry bare filenames (mondoo-aws-security.mql.yaml) so failure messages
// name the bundle rather than a relative path, and only the code that opens a
// bundle needs to know where content/ sits.
func bundlePath(bundleFile string) string {
	return filepath.Join(contentDir, bundleFile)
}

// extraProviders lists providers that only the build-tagged IaC-variant suites
// need. It is empty in the default (main cnspec app) test build and is extended
// via an init() in iac_variants_test.go under the `iac_variants` build tag, so
// the default test run does not download IaC-only providers such as bicep.
var extraProviders []string

func TestMain(m *testing.M) {
	// ensure providers are loaded
	providerList := append([]string{"terraform", "k8s", "aws", "azure", "gcp", "cloudformation"}, extraProviders...)
	for _, p := range providerList {
		_, err := providers.EnsureProvider(providers.ProviderLookup{ProviderName: p}, true, nil)
		if err != nil {
			panic(err)
		}
	}

	// Warm the provider cache once, serially, now that every provider is
	// installed. providers.ListAll caches into an unsynchronized global and, on a
	// cache miss, sets it to an empty slice before repopulating it. The
	// iac_variants suites scan in parallel (t.Parallel + -parallel), so if the
	// cache is cold when the first scans fan out, one goroutine can observe that
	// intermediate empty slice and fail to resolve the asset's connector
	// ("cannot find provider for conn type=terraform-hcl"). The race window
	// widens with the number of installed providers, which is why it only bites
	// once the IaC suites add their extra providers. Installing leaves the cache
	// nil (the last install invalidates it), so populate it here, before any
	// parallel scan, and every scan then hits the warm read-only fast path.
	if _, err := providers.ListAll(); err != nil {
		panic(err)
	}

	// Aggregate the provider schemas once, serially, against a harder failure in
	// the same area. The coordinator takes its two locks in opposite orders on two
	// paths the parallel suites run at the same time:
	//
	//	compiling MQL         Schema.Lookup -> unsafeLoadAll -> coordinator.LoadSchema
	//	                        holds the schema lock, wants the coordinator lock
	//	starting a provider   GetRunningProvider -> unsafeStartProvider -> Schema.Add
	//	                        holds the coordinator lock, wants the schema lock
	//
	// Interleaved, that is a deadlock, and a mutex cannot be timed out: the scans
	// do not fail, they stop, and the shard runs to the test binary's timeout an
	// hour later with four checks still "running".
	//
	// Lookup only reaches LoadSchema while the aggregate is older than the last
	// provider install, so one Lookup here — after the installs above, before any
	// parallel scan — leaves the compile path on its read-only branch, and the two
	// orders stop interleaving. It has to be Lookup rather than LoadSchema:
	// LoadSchema populates the provider, but only the Lookup path refreshes the
	// aggregate and moves the timestamp that gates it.
	//
	// This is a workaround for the lock ordering in providers/coordinator.go and
	// providers/extensible_schema.go, and can go once that is fixed upstream.
	providers.Coordinator.Schema().Lookup("asset")

	// Run tests
	exitVal := m.Run()
	os.Exit(exitVal)
}
