// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package content

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/v13/policy"
	"go.mondoo.com/cnspec/v13/policy/scan"
	"go.mondoo.com/mql/v13/logger"
	"go.mondoo.com/mql/v13/providers"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/testutils"
)

func init() {
	logger.Set("info")
}

func TestMain(m *testing.M) {
	// ensure providers are loaded
	providerList := []string{"terraform", "k8s", "aws", "azure", "gcp", "cloudformation"}
	for _, p := range providerList {
		_, err := providers.EnsureProvider(providers.ProviderLookup{ProviderName: p}, true, nil)
		if err != nil {
			panic(err)
		}
	}

	// Run tests
	exitVal := m.Run()
	os.Exit(exitVal)
}

func tfAsset(dir string) *inventory.Asset {
	return &inventory.Asset{
		Connections: []*inventory.Config{
			{
				Type: "terraform-hcl",
				Options: map[string]string{
					"path": dir,
				},
			},
		},
	}
}

func k8sAsset(dir string) *inventory.Asset {
	return &inventory.Asset{
		Connections: []*inventory.Config{{
			Type: "k8s",
			Options: map[string]string{
				"path": dir,
			},
			Discover: &inventory.Discovery{
				Targets: []string{"pods"}, // ignore the manifest which does not return anything
			},
		}},
	}
}

func runBundle(policyBundlePath string, policyMrn string, asset *inventory.Asset) (*policy.Report, error) {
	ctx := context.Background()
	policyBundle, err := policy.DefaultBundleLoader().BundleFromPaths(policyBundlePath)
	if err != nil {
		return nil, err
	}

	policyBundle.OwnerMrn = "//policy.api.mondoo.app"

	policyFilters := []string{}
	if policyMrn != "" {
		policyFilters = append(policyFilters, policyMrn)
	}

	scanner := scan.NewLocalScanner()
	result, err := scanner.RunIncognito(ctx, &scan.Job{
		Inventory: &inventory.Inventory{
			Spec: &inventory.InventorySpec{
				Assets: []*inventory.Asset{asset},
			},
		},
		Bundle:        policyBundle,
		PolicyFilters: policyFilters,
		ReportType:    scan.ReportType_FULL,
	})
	if err != nil {
		return nil, err
	}

	fullResult := result.GetFull()
	if fullResult == nil {
		return nil, errors.New("no full report generated")
	}
	if len(fullResult.Errors) > 0 {
		msg := ""
		for _, e := range fullResult.Errors {
			msg += e + "; "
		}

		return nil, errors.New("errors during scan: " + msg)
	}

	reports := fullResult.Reports
	if len(reports) > 0 {
		for _, report := range reports {
			return report, nil
		}
	}

	return nil, errors.New("no report found")
}

func TestNetworkPostureContent(t *testing.T) {
	runtime := networkPostureRuntime(t)
	loader := policy.DefaultBundleLoader()

	t.Run("policy compiles network posture controls", func(t *testing.T) {
		bundle, err := loader.BundleFromPaths("./mondoo-kubernetes-network-posture.mql.yaml")
		require.NoError(t, err)
		require.NotNil(t, bundle)
		require.Len(t, bundle.Policies, 1)
		require.Len(t, bundle.Queries, 6)

		queries := map[string]string{}
		for _, query := range bundle.Queries {
			require.NotEmpty(t, query.Uid)
			require.NotEmpty(t, query.Mql, "query %q should have an MQL expression", query.Uid)
			queries[query.Uid] = query.Mql
		}

		p := bundle.Policies[0]
		assert.Equal(t, "mondoo-kubernetes-network-posture", p.Uid)
		require.Len(t, p.Groups, 1)

		checks := map[string]bool{}
		for _, check := range p.Groups[0].Checks {
			checks[check.Uid] = true
		}
		assert.True(t, checks["mondoo-kubernetes-network-posture-internet-exposure-evidence"])
		assert.True(t, checks["mondoo-kubernetes-network-posture-public-egress-classified"])
		assert.True(t, checks["mondoo-kubernetes-network-posture-public-egress-nat-owned"])
		assert.True(t, checks["mondoo-kubernetes-network-posture-secondary-interface-covered"])
		assert.True(t, checks["mondoo-kubernetes-network-posture-primary-egress-isolated"])
		assert.True(t, checks["mondoo-kubernetes-network-posture-admin-policy-deny-semantics"])

		raw, err := os.ReadFile("./mondoo-kubernetes-network-posture.mql.yaml")
		require.NoError(t, err)
		content := string(raw)
		assert.Contains(t, queries["mondoo-kubernetes-network-posture-internet-exposure-evidence"], `k8s.networkExposures.where(internetExposed == true)`)
		assert.Contains(t, queries["mondoo-kubernetes-network-posture-internet-exposure-evidence"], `metadataClassification != "" && owner != ""`)
		assert.Contains(t, queries["mondoo-kubernetes-network-posture-public-egress-classified"], `k8s.egressRoutes.where(publicCidrs.length > 0)`)
		assert.Contains(t, queries["mondoo-kubernetes-network-posture-public-egress-classified"], `sourceRef != ""`)
		assert.Contains(t, queries["mondoo-kubernetes-network-posture-public-egress-classified"], `metadataClassification != "" && owner != ""`)
		assert.Contains(t, queries["mondoo-kubernetes-network-posture-public-egress-nat-owned"], `k8s.egressNats.where(publicCidrs.length > 0)`)
		assert.Contains(t, queries["mondoo-kubernetes-network-posture-public-egress-nat-owned"], `sourceRef != ""`)
		assert.Contains(t, queries["mondoo-kubernetes-network-posture-public-egress-nat-owned"], `metadataClassification != "" && owner != ""`)
		assert.Contains(t, queries["mondoo-kubernetes-network-posture-secondary-interface-covered"], `k8s.networkPolicyCoverages.where(interfaces.contains("secondary") && workloadRef != "").all`)
		assert.NotContains(t, queries["mondoo-kubernetes-network-posture-secondary-interface-covered"], `length > 0 &&`)
		assert.Contains(t, queries["mondoo-kubernetes-network-posture-primary-egress-isolated"], `k8s.networkPolicyCoverages.where(interfaces.contains("primary")).length > 0`)
		assert.Contains(t, queries["mondoo-kubernetes-network-posture-admin-policy-deny-semantics"], `adminNetworkPolicies.length > 0`)
		assert.Contains(t, queries["mondoo-kubernetes-network-posture-admin-policy-deny-semantics"], `adminDefaultDenyIngress == true || adminDefaultDenyEgress == true`)
		assert.NotContains(t, queries["mondoo-kubernetes-network-posture-admin-policy-deny-semantics"], `coverageGaps.none`)
		assert.NotContains(t, queries["mondoo-kubernetes-network-posture-admin-policy-deny-semantics"], `admin network policy ingress does not define catch-all deny traffic`)
		assert.NotContains(t, queries["mondoo-kubernetes-network-posture-admin-policy-deny-semantics"], `admin network policy egress does not define catch-all deny traffic`)
		for _, want := range []string{
			"Gateway API",
			"Ingress",
			"Service",
			"HBN",
			"MultiNetworkPolicy",
			"AdminNetworkPolicy",
			"BaselineAdminNetworkPolicy",
			"Calico",
			"Cilium",
		} {
			assert.Contains(t, content, want)
		}

		requireBundleCompiles(t, bundle, runtime)
	})

	t.Run("inventory querypack compiles network posture queries", func(t *testing.T) {
		bundle, err := loader.BundleFromPaths("./querypacks/mondoo-kubernetes-inventory.mql.yaml")
		require.NoError(t, err)
		require.NotNil(t, bundle)
		require.Len(t, bundle.Packs, 1)

		queries := map[string]string{}
		for _, group := range bundle.Packs[0].Groups {
			for _, query := range group.Queries {
				queries[query.Uid] = query.Mql
			}
		}
		assert.Contains(t, queries, "k8s-network-exposures")
		assert.Contains(t, queries, "k8s-gateway-api-inventory")
		assert.Contains(t, queries, "k8s-egress-routes")
		assert.Contains(t, queries, "k8s-egress-nats")
		assert.Contains(t, queries, "k8s-network-policy-coverage")
		assert.Contains(t, queries["k8s-gateway-api-inventory"], "k8s.networkExposures.where")
		assert.Contains(t, queries["k8s-gateway-api-inventory"], `sourceKind == "Gateway"`)
		assert.Contains(t, queries["k8s-gateway-api-inventory"], `sourceKind == "HTTPRoute"`)
		assert.Contains(t, queries["k8s-gateway-api-inventory"], `sourceKind == "TLSRoute"`)
		assert.Contains(t, queries["k8s-network-exposures"], "sourceKind")
		assert.Contains(t, queries["k8s-network-exposures"], "internetExposed")
		assert.Contains(t, queries["k8s-network-exposures"], "metadataClassification")
		assert.Contains(t, queries["k8s-network-exposures"], "owner")
		assert.Contains(t, queries["k8s-network-policy-coverage"], "adminNetworkPolicies")
		assert.Contains(t, queries["k8s-network-policy-coverage"], "multiNetworkPolicies")
		assert.Contains(t, queries["k8s-network-policy-coverage"], "calicoPolicies")
		assert.Contains(t, queries["k8s-network-policy-coverage"], "ciliumPolicies")
		assert.Contains(t, queries["k8s-network-policy-coverage"], "adminDefaultDenyIngress")
		assert.Contains(t, queries["k8s-network-policy-coverage"], "adminDefaultDenyEgress")
		assert.Contains(t, queries["k8s-network-policy-coverage"], "secondaryInterfaceIngressCovered")
		assert.Contains(t, queries["k8s-network-policy-coverage"], "secondaryInterfaceEgressCovered")

		bundle.ConvertQuerypacks()
		requireBundleCompiles(t, bundle, runtime)
	})
}

func networkPostureRuntime(t *testing.T) *providers.Runtime {
	t.Helper()

	runtime := providers.DefaultRuntime()
	schema, ok := runtime.Schema().(providers.ExtensibleSchema)
	require.True(t, ok, "provider runtime schema must support adding source schemas")
	schema.Add("core", testutils.MustLoadSchema(testutils.SchemaProvider{Path: networkPostureCoreSchemaPath(t)}))
	schema.Add("network", testutils.MustLoadSchema(testutils.SchemaProvider{Path: networkPostureNetworkSchemaPath(t)}))
	schema.Add("os", testutils.MustLoadSchema(testutils.SchemaProvider{Path: networkPostureOSSchemaPath(t)}))
	schema.Add("k8s", testutils.MustLoadSchema(testutils.SchemaProvider{Path: networkPostureK8sSchemaPath(t)}))
	return runtime
}

func requireBundleCompiles(t *testing.T, bundle *policy.Bundle, runtime *providers.Runtime) {
	t.Helper()

	compiled, err := bundle.Compile(context.Background(), runtime.Schema(), nil)
	require.NoError(t, err)
	require.NotNil(t, compiled)
}

func networkPostureK8sSchemaPath(t *testing.T) string {
	t.Helper()

	path := "./testdata/schema/providers/k8s/resources/k8s.lr"
	_, err := os.Stat(path)
	require.NoErrorf(t, err, "in-repo schema fixture must point to a readable k8s.lr schema file")
	return path
}

func networkPostureCoreSchemaPath(t *testing.T) string {
	t.Helper()

	path := "./testdata/schema/providers/core/resources/core.lr"
	_, err := os.Stat(path)
	require.NoErrorf(t, err, "in-repo schema fixture must point to a readable core.lr schema file")
	return path
}

func networkPostureNetworkSchemaPath(t *testing.T) string {
	t.Helper()

	path := "./testdata/schema/providers/network/resources/network.lr"
	_, err := os.Stat(path)
	require.NoErrorf(t, err, "in-repo schema fixture must point to a readable network.lr schema file")
	return path
}

func networkPostureOSSchemaPath(t *testing.T) string {
	t.Helper()

	path := "./testdata/schema/providers/os/resources/os.lr"
	_, err := os.Stat(path)
	require.NoErrorf(t, err, "in-repo schema fixture must point to a readable os.lr schema file")
	return path
}

func TestBundles(t *testing.T) {
	type TestCase struct {
		provider   string
		bundleFile string
		testDir    string
		policyMrn  string
		score      uint32
	}

	tests := []TestCase{
		{
			provider:   "k8s",
			bundleFile: "./mondoo-kubernetes-security.mql.yaml",
			testDir:    "./testdata/mondoo-kubernetes-security-pass",
			policyMrn:  "//policy.api.mondoo.app/policies/mondoo-kubernetes-security",
			score:      100,
		},
		{
			provider:   "k8s",
			bundleFile: "./mondoo-kubernetes-security.mql.yaml",
			testDir:    "./testdata/mondoo-kubernetes-security-fail",
			policyMrn:  "//policy.api.mondoo.app/policies/mondoo-kubernetes-security",
			score:      0x0,
		},
		// cnspec scan terraform hcl testdata/mondoo-aws-security-tf-pass -f mondoo-aws-security.mql.yaml
		{
			provider:   "terraform",
			bundleFile: "./mondoo-aws-security.mql.yaml",
			testDir:    "./testdata/mondoo-aws-security-tf-pass",
			policyMrn:  "//policy.api.mondoo.app/policies/mondoo-aws-security",
			score:      100,
		},
		{
			provider:   "terraform",
			bundleFile: "./mondoo-aws-security.mql.yaml",
			testDir:    "./testdata/mondoo-aws-security-tf-fail",
			policyMrn:  "//policy.api.mondoo.app/policies/mondoo-aws-security",
			score:      0,
		},
		// cnspec scan terraform hcl testdata/mondoo-azure-security-tf-pass -f mondoo-azure-security.mql.yaml
		// TODO: enrich azure tests with HCL test cases
		//{
		//	provider:   "terraform",
		//	bundleFile: "./mondoo-azure-security.mql.yaml",
		//	testDir:    "./testdata/mondoo-azure-security-tf-pass",
		//	policyMrn:  "//policy.api.mondoo.app/policies/mondoo-azure-security",
		//	score:      0x5,
		//},
		//{
		//	provider:   "terraform",
		//	bundleFile: "./mondoo-azure-security.mql.yaml",
		//	testDir:    "./testdata/mondoo-azure-security-tf-fail",
		//	policyMrn:  "//policy.api.mondoo.app/policies/mondoo-azure-security",
		//	score:      0,
		//},
		// cnspec scan terraform hcl testdata/mondoo-gcp-security-tf-pass -f mondoo-gcp-security.mql.yaml
		{
			provider:   "terraform",
			bundleFile: "./mondoo-gcp-security.mql.yaml",
			testDir:    "./testdata/mondoo-gcp-security-tf-pass",
			policyMrn:  "//policy.api.mondoo.app/policies/mondoo-gcp-security",
			score:      100,
		},
		{
			provider:   "terraform",
			bundleFile: "./mondoo-gcp-security.mql.yaml",
			testDir:    "./testdata/mondoo-gcp-security-tf-fail",
			policyMrn:  "//policy.api.mondoo.app/policies/mondoo-gcp-security",
			score:      0,
		},
	}

	for i := range tests {
		test := tests[i]
		t.Run(test.testDir, func(t *testing.T) {
			var asset *inventory.Asset
			switch test.provider {
			case "terraform":
				asset = tfAsset(test.testDir)
			case "k8s":
				asset = k8sAsset(test.testDir)
			default:
				t.Fatalf("unknown provider type: %s", test.provider)
			}
			report, err := runBundle(test.bundleFile, test.policyMrn, asset)
			require.NoError(t, err)

			score := report.Scores[test.policyMrn]
			if !assert.Equal(t, test.score, score.Value) {
				// Log all failing checks to make regressions from new checks easy to diagnose
				var failingChecks []string
				for mrn, s := range report.Scores {
					if mrn == test.policyMrn || s == nil || s.ScoreCompletion == 0 || s.Weight == 0 {
						continue
					}
					if s.Value < 100 {
						failingChecks = append(failingChecks, fmt.Sprintf("  score=%d  %s", s.Value, mrn))
					}
				}
				sort.Strings(failingChecks)
				t.Logf("Failing checks (%d):\n%s", len(failingChecks), strings.Join(failingChecks, "\n"))
			}
		})
	}
}
