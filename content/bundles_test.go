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
	return k8sAssetWithTargets(dir, "pods")
}

func k8sAssetWithTargets(dir string, targets ...string) *inventory.Asset {
	config := &inventory.Config{
		Type: "k8s",
		Options: map[string]string{
			"path": dir,
		},
	}
	if len(targets) > 0 {
		config.Discover = &inventory.Discovery{
			Targets: targets,
		}
	}
	return &inventory.Asset{
		Connections: []*inventory.Config{config},
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

func TestKyvernoMappedKubernetesChecks(t *testing.T) {
	loader := policy.DefaultBundleLoader()

	securityBundle, err := loader.BundleFromPaths("./mondoo-kubernetes-security.mql.yaml")
	require.NoError(t, err)
	securityQueries := bundleQueriesByUID(t, securityBundle)

	clusterAdmin := securityQueries["mondoo-kubernetes-security-rbac-no-cluster-admin-bindings"]
	assert.Contains(t, clusterAdmin, `roleRef["kind"] == "ClusterRole" && roleRef["name"] == "cluster-admin"`)
	assert.NotContains(t, clusterAdmin, `rolebindings.none(roleRef["name"] == "cluster-admin")`)

	secretRead := securityQueries["mondoo-kubernetes-security-rbac-no-secret-read-verbs"]
	assert.Contains(t, secretRead, `_["resources"].containsNone(["secrets", "*"])`)
	assert.Contains(t, secretRead, `_["verbs"].containsNone(["get", "list", "watch", "*"])`)

	escalation := securityQueries["mondoo-kubernetes-security-rbac-no-escalation-verbs"]
	assert.Contains(t, escalation, `_["verbs"].containsNone(["bind", "escalate", "impersonate", "*"])`)
	assert.NotContains(t, escalation, `_["apiGroups"].containsNone(["rbac.authorization.k8s.io", "*"])`)
	assert.NotContains(t, escalation, `_["resources"].containsNone(["roles", "clusterroles", "*"])`)

	nodesProxy := securityQueries["mondoo-kubernetes-security-rbac-no-nodes-proxy"]
	assert.Contains(t, nodesProxy, `_["resources"].containsNone(["nodes/proxy", "*"])`)
	assert.Contains(t, nodesProxy, `_["apiGroups"].containsNone(["", "*"])`)

	kyvernoBundle, err := loader.BundleFromPaths("./mondoo-kubernetes-kyverno.mql.yaml")
	require.NoError(t, err)
	kyvernoQueries := bundleQueriesByUID(t, kyvernoBundle)

	expired := kyvernoQueries["mondoo-kubernetes-kyverno-policyexceptions-not-expired"]
	assert.Contains(t, expired, `statusReasons.any(_ == /^expired:/)`)

	orphaned := kyvernoQueries["mondoo-kubernetes-kyverno-policyexceptions-not-orphaned"]
	assert.Contains(t, orphaned, `statusReasons.any(_ == /^orphaned:/ || _ == /^invalid:/)`)

	unmappedExceptions := kyvernoQueries["mondoo-kubernetes-kyverno-policyexceptions-mapped-to-mondoo"]
	assert.Contains(t, unmappedExceptions, `statusReasons.any(_ == /^unmapped:/)`)

	broad := kyvernoQueries["mondoo-kubernetes-kyverno-policyexceptions-not-broad"]
	assert.Contains(t, broad, `statusReasons.any(_ == /^broad:/)`)

	mappedResults := kyvernoQueries["mondoo-kubernetes-kyverno-policyreports-no-mapped-failing-results"]
	assert.Contains(t, mappedResults, `result == "warn"`)

	unmappedResults := kyvernoQueries["mondoo-kubernetes-kyverno-policyreports-no-unmapped-failing-results"]
	assert.Contains(t, unmappedResults, `result == "warn"`)

	bestPracticesBundle, err := loader.BundleFromPaths("./mondoo-kubernetes-best-practices.mql.yaml")
	require.NoError(t, err)
	bestPracticesQueries := bundleQueriesByUID(t, bestPracticesBundle)

	servicePorts := bestPracticesQueries["mondoo-kubernetes-best-practices-service-ports-approved-range"]
	assert.Contains(t, servicePorts, `k8s.service.type != "NodePort"`)
	assert.Contains(t, servicePorts, `_['nodePort']`)
	assert.NotContains(t, servicePorts, `_['port'] >= 32000`)

	ingressClass := bestPracticesQueries["mondoo-kubernetes-best-practices-ingress-approved-class-annotation"]
	assert.Contains(t, ingressClass, `k8s.ingress.annotations["kubernetes.io/ingress.class"]`)
	assert.Contains(t, ingressClass, `if (k8s.ingress.ingressClassName != "")`)
	assert.Contains(t, ingressClass, `k8s.ingress.ingressClassName.in(["HAProxy", "nginx"])`)
	assert.Contains(t, ingressClass, `defaultIngressClasses = k8s.ingressClasses.where(annotations["ingressclass.kubernetes.io/is-default-class"] == "true")`)
	assert.Contains(t, ingressClass, `defaultIngressClasses.length == 1`)
}

func TestKyvernoMappedKubernetesBundlesCompile(t *testing.T) {
	loader := policy.DefaultBundleLoader()
	ctx := context.Background()

	runtime := providers.DefaultRuntime()
	for _, path := range []string{
		"./mondoo-kubernetes-kyverno.mql.yaml",
		"./mondoo-kubernetes-security.mql.yaml",
		"./mondoo-kubernetes-best-practices.mql.yaml",
	} {
		t.Run(path, func(t *testing.T) {
			bundle, err := loader.BundleFromPaths(path)
			require.NoError(t, err)

			compiled, err := bundle.Compile(ctx, runtime.Schema(), nil)
			require.NoError(t, err)
			require.NotNil(t, compiled)
		})
	}
}

func bundleQueriesByUID(t *testing.T, bundle *policy.Bundle) map[string]string {
	t.Helper()

	queries := map[string]string{}
	for _, query := range bundle.Queries {
		queries[query.Uid] = query.Mql
	}
	return queries
}

func TestKubernetesBestPracticesIngressClassUsesEffectiveValue(t *testing.T) {
	const (
		policyMrn = "//policy.api.mondoo.app/policies/mondoo-kubernetes-best-practices"
		checkUID  = "mondoo-kubernetes-best-practices-ingress-approved-class-annotation"
	)

	tests := []struct {
		name      string
		dir       string
		targets   []string
		wantScore uint32
	}{
		{
			name:      "legacy annotation accepted when spec class is absent",
			dir:       "./testdata/mondoo-kubernetes-best-practices-ingress-pass",
			targets:   []string{"ingresses"},
			wantScore: 100,
		},
		{
			name:      "spec class wins over approved legacy annotation",
			dir:       "./testdata/mondoo-kubernetes-best-practices-ingress-fail",
			targets:   []string{"ingresses"},
			wantScore: 70,
		},
	}

	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			report, err := runBundle(
				"./mondoo-kubernetes-best-practices.mql.yaml",
				policyMrn,
				k8sAssetWithTargets(test.dir, test.targets...),
			)
			require.NoError(t, err)

			score := reportScoreByUID(t, report, checkUID)
			assert.Equal(t, test.wantScore, score.Value)
		})
	}
}

func reportScoreByUID(t *testing.T, report *policy.Report, uid string) *policy.Score {
	t.Helper()

	for mrn, score := range report.Scores {
		if strings.HasSuffix(mrn, "/queries/"+uid) {
			require.NotNil(t, score)
			return score
		}
	}

	t.Fatalf("score for query %q not found", uid)
	return nil
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
