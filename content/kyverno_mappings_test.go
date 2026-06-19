// Copyright Mondoo, Inc. 2026
// SPDX-License-Identifier: BUSL-1.1

package content

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/v13/policy"
	"go.mondoo.com/mql/v13/providers"
	"go.mondoo.com/mql/v13/providers-sdk/v1/testutils"
	"go.yaml.in/yaml/v3"
)

type kyvernoMappingCatalog struct {
	Version  int                  `json:"version" yaml:"version"`
	Source   kyvernoMappingSource `json:"source" yaml:"source"`
	Policies []kyvernoPolicyEntry `json:"policies" yaml:"policies"`
	Mappings []kyvernoMapping     `json:"mappings" yaml:"mappings"`
}

type kyvernoMappingSource struct {
	Ref                 string `json:"ref" yaml:"ref"`
	PolicyResourceCount int    `json:"policyResourceCount" yaml:"policyResourceCount"`
	UniquePolicyCount   int    `json:"uniquePolicyCount" yaml:"uniquePolicyCount"`
}

type kyvernoPolicyEntry struct {
	KyvernoPolicy string          `json:"kyvernoPolicy" yaml:"kyvernoPolicy"`
	MappingStatus string          `json:"mappingStatus" yaml:"mappingStatus"`
	MappingRefs   []string        `json:"mappingRefs" yaml:"mappingRefs"`
	KyvernoRules  []string        `json:"kyvernoRules" yaml:"kyvernoRules"`
	Upstream      kyvernoUpstream `json:"upstream" yaml:"upstream"`
}

type kyvernoUpstream struct {
	PolicyResources []kyvernoPolicyResource `json:"policyResources" yaml:"policyResources"`
}

type kyvernoPolicyResource struct {
	Path       string `json:"path" yaml:"path"`
	APIVersion string `json:"apiVersion" yaml:"apiVersion"`
	Kind       string `json:"kind" yaml:"kind"`
}

type kyvernoMapping struct {
	Uid             string   `json:"uid" yaml:"uid"`
	KyvernoPolicy   string   `json:"kyvernoPolicy" yaml:"kyvernoPolicy"`
	KyvernoRules    []string `json:"kyvernoRules" yaml:"kyvernoRules"`
	MondooPolicyUid string   `json:"mondooPolicyUid" yaml:"mondooPolicyUid"`
	MondooChecks    []string `json:"mondooChecks" yaml:"mondooChecks"`
	Confidence      string   `json:"confidence" yaml:"confidence"`
	MappingStatus   string   `json:"mappingStatus" yaml:"mappingStatus"`
	Reason          string   `json:"reason" yaml:"reason"`
}

type mondooKubernetesSecurityContent struct {
	Queries []struct {
		Uid string `json:"uid" yaml:"uid"`
		Mql string `json:"mql" yaml:"mql"`
	} `json:"queries" yaml:"queries"`
}

type mondooKyvernoPolicyContent struct {
	Policies []struct {
		Uid     string `json:"uid" yaml:"uid"`
		Name    string `json:"name" yaml:"name"`
		Summary string `json:"summary" yaml:"summary"`
		Require []struct {
			Provider string `json:"provider" yaml:"provider"`
		} `json:"require" yaml:"require"`
		Groups []struct {
			Title   string `json:"title" yaml:"title"`
			Filters string `json:"filters" yaml:"filters"`
			Checks  []struct {
				Uid string `json:"uid" yaml:"uid"`
			} `json:"checks" yaml:"checks"`
		} `json:"groups" yaml:"groups"`
	} `json:"policies" yaml:"policies"`
	Queries []struct {
		Uid    string `json:"uid" yaml:"uid"`
		Title  string `json:"title" yaml:"title"`
		Impact int    `json:"impact" yaml:"impact"`
		Mql    string `json:"mql" yaml:"mql"`
	} `json:"queries" yaml:"queries"`
}

func TestMondooKubernetesKyvernoPolicyContent(t *testing.T) {
	data, err := os.ReadFile("./mondoo-kubernetes-kyverno.mql.yaml")
	require.NoError(t, err)

	var content mondooKyvernoPolicyContent
	require.NoError(t, yaml.Unmarshal(data, &content))
	require.Len(t, content.Policies, 1)

	policy := content.Policies[0]
	require.Equal(t, "mondoo-kubernetes-kyverno", policy.Uid)
	require.Equal(t, "Mondoo Kubernetes Kyverno Integration", policy.Name)
	require.Contains(t, policy.Summary, "Kyverno")
	require.Len(t, policy.Require, 1)
	require.Equal(t, "k8s", policy.Require[0].Provider)
	require.Len(t, policy.Groups, 2)

	expectedChecksByGroup := map[string][]string{
		"Kyverno PolicyExceptions": {
			"mondoo-kubernetes-kyverno-policyexceptions-not-expired",
			"mondoo-kubernetes-kyverno-policyexceptions-not-orphaned",
			"mondoo-kubernetes-kyverno-policyexceptions-mapped-to-mondoo",
			"mondoo-kubernetes-kyverno-policyexceptions-not-broad",
			"mondoo-kubernetes-kyverno-policyexceptions-documented",
		},
		"Kyverno Policy Reports": {
			"mondoo-kubernetes-kyverno-policyreports-no-mapped-failing-results",
			"mondoo-kubernetes-kyverno-policyreports-no-unmapped-failing-results",
		},
	}
	expectedFilter := `asset.platform == "k8s-cluster" && k8s.kyverno.installed == true`
	referencedChecks := map[string]struct{}{}
	for _, group := range policy.Groups {
		require.Equal(t, expectedFilter, group.Filters, "group %q should only run on Kyverno-enabled k8s cluster assets", group.Title)
		expectedChecks, ok := expectedChecksByGroup[group.Title]
		require.True(t, ok, "unexpected Kyverno policy group %q", group.Title)
		actualChecks := make([]string, 0, len(group.Checks))
		for _, check := range group.Checks {
			actualChecks = append(actualChecks, check.Uid)
			referencedChecks[check.Uid] = struct{}{}
		}
		require.Equal(t, expectedChecks, actualChecks)
	}

	queriesByUID := map[string]struct {
		Title  string
		Impact int
		Mql    string
	}{}
	for _, query := range content.Queries {
		require.NotEmpty(t, query.Uid)
		queriesByUID[query.Uid] = struct {
			Title  string
			Impact int
			Mql    string
		}{Title: query.Title, Impact: query.Impact, Mql: query.Mql}
	}
	require.Len(t, queriesByUID, len(referencedChecks))
	for check := range referencedChecks {
		_, ok := queriesByUID[check]
		require.True(t, ok, "group references missing query %q", check)
	}

	require.Equal(t, 80, queriesByUID["mondoo-kubernetes-kyverno-policyexceptions-not-expired"].Impact)
	require.Contains(t, queriesByUID["mondoo-kubernetes-kyverno-policyexceptions-not-expired"].Mql, "k8s.kyverno.failExpiredPolicyExceptions == false ||")
	require.Contains(t, queriesByUID["mondoo-kubernetes-kyverno-policyexceptions-not-expired"].Mql, `statusReasons.any(_ == /^expired:/)`)

	require.Equal(t, 70, queriesByUID["mondoo-kubernetes-kyverno-policyexceptions-not-orphaned"].Impact)
	require.Contains(t, queriesByUID["mondoo-kubernetes-kyverno-policyexceptions-not-orphaned"].Mql, `statusReasons.any(_ == /^orphaned:/ || _ == /^invalid:/)`)

	require.Equal(t, 60, queriesByUID["mondoo-kubernetes-kyverno-policyexceptions-mapped-to-mondoo"].Impact)
	require.Contains(t, queriesByUID["mondoo-kubernetes-kyverno-policyexceptions-mapped-to-mondoo"].Mql, "k8s.kyverno.reportUnmappedPolicyExceptions == false ||")
	require.Contains(t, queriesByUID["mondoo-kubernetes-kyverno-policyexceptions-mapped-to-mondoo"].Mql, `statusReasons.any(_ == /^unmapped:/)`)

	require.Equal(t, 50, queriesByUID["mondoo-kubernetes-kyverno-policyexceptions-not-broad"].Impact)
	require.Contains(t, queriesByUID["mondoo-kubernetes-kyverno-policyexceptions-not-broad"].Mql, `statusReasons.any(_ == /^broad:/)`)

	require.Equal(t, 40, queriesByUID["mondoo-kubernetes-kyverno-policyexceptions-documented"].Impact)
	require.Contains(t, queriesByUID["mondoo-kubernetes-kyverno-policyexceptions-documented"].Mql, `owner != "" && justification != ""`)

	require.Equal(t, 70, queriesByUID["mondoo-kubernetes-kyverno-policyreports-no-mapped-failing-results"].Impact)
	require.Contains(t, queriesByUID["mondoo-kubernetes-kyverno-policyreports-no-mapped-failing-results"].Mql, `result == "fail" || result == "warn" || result == "error"`)
	require.Contains(t, queriesByUID["mondoo-kubernetes-kyverno-policyreports-no-mapped-failing-results"].Mql, `mappedMondooCheckUids.length > 0 || mappedMondooCheckMrns.length > 0`)

	require.Equal(t, 60, queriesByUID["mondoo-kubernetes-kyverno-policyreports-no-unmapped-failing-results"].Impact)
	require.Contains(t, queriesByUID["mondoo-kubernetes-kyverno-policyreports-no-unmapped-failing-results"].Mql, "k8s.kyverno.reportUnmappedPolicyResults == false ||")
	require.Contains(t, queriesByUID["mondoo-kubernetes-kyverno-policyreports-no-unmapped-failing-results"].Mql, `mappedMondooCheckUids.length == 0 && mappedMondooCheckMrns.length == 0`)

	expectedStatusCoverage := map[string][]string{
		"mondoo-kubernetes-kyverno-policyexceptions-not-expired": {
			`failExpiredPolicyExceptions == false`,
			`statusReasons.any(_ == /^expired:/)`,
		},
		"mondoo-kubernetes-kyverno-policyexceptions-not-orphaned": {
			`statusReasons.any(_ == /^orphaned:/ || _ == /^invalid:/)`,
		},
		"mondoo-kubernetes-kyverno-policyexceptions-mapped-to-mondoo": {
			`reportUnmappedPolicyExceptions == false`,
			`statusReasons.any(_ == /^unmapped:/)`,
		},
		"mondoo-kubernetes-kyverno-policyexceptions-not-broad": {
			`statusReasons.any(_ == /^broad:/)`,
		},
		"mondoo-kubernetes-kyverno-policyexceptions-documented": {
			`owner != ""`,
			`justification != ""`,
		},
		"mondoo-kubernetes-kyverno-policyreports-no-mapped-failing-results": {
			`result == "fail" || result == "warn" || result == "error"`,
			`mappedMondooCheckUids.length > 0 || mappedMondooCheckMrns.length > 0`,
		},
		"mondoo-kubernetes-kyverno-policyreports-no-unmapped-failing-results": {
			`reportUnmappedPolicyResults == false`,
			`mappedMondooCheckUids.length == 0 && mappedMondooCheckMrns.length == 0`,
		},
	}
	for uid, substrings := range expectedStatusCoverage {
		for _, substring := range substrings {
			require.Contains(t, queriesByUID[uid].Mql, substring, "query %q should cover %q", uid, substring)
		}
	}
}

func TestKubernetesBestPracticesServiceIngressChecksUseAssetResources(t *testing.T) {
	data, err := os.ReadFile("./mondoo-kubernetes-best-practices.mql.yaml")
	require.NoError(t, err)
	content := string(data)

	require.NotContains(t, content, "k8s.services.all(")
	require.NotContains(t, content, "k8s.ingresses.all(")
	require.Contains(t, content, `k8s.service.type != "NodePort"`)
	require.Contains(t, content, `k8s.service.externalIPs.length == 0`)
	require.Contains(t, content, `k8s.ingress.rules.all(host != "")`)
	require.Contains(t, content, `tlsHosts = k8s.ingress.tls.map(hosts).flat`)
	require.Contains(t, content, `k8s.ingress.rules.map(host).where(_ != "").all(tlsHosts.contains(_))`)
}

func TestKyvernoMappedKubernetesChecksCoverEdgeCases(t *testing.T) {
	securityQueries := kubernetesContentQueries(t, "./mondoo-kubernetes-security.mql.yaml")
	bestPracticesQueries := kubernetesContentQueries(t, "./mondoo-kubernetes-best-practices.mql.yaml")

	require.Contains(t, securityQueries["mondoo-kubernetes-security-job-created-by-cronjob"], `ownerReferences.length > 0`)
	require.Contains(t, securityQueries["mondoo-kubernetes-security-job-created-by-cronjob"], `ownerReferences[0].kind == "CronJob"`)
	require.Contains(t, securityQueries["mondoo-kubernetes-security-job-created-by-cronjob"], `ownerReferences[0].controller == true`)
	require.NotContains(t, securityQueries["mondoo-kubernetes-security-job-created-by-cronjob"], `ownerReferences.any(kind == "CronJob"`)

	require.Contains(t, securityQueries["mondoo-kubernetes-security-rbac-no-escalation-verbs"], `verbs"].containsNone(["bind", "escalate", "impersonate", "*"])`)
	require.NotContains(t, securityQueries["mondoo-kubernetes-security-rbac-no-escalation-verbs"], `apiGroups"].containsNone(["rbac.authorization.k8s.io", "*"])`)

	require.Contains(t, securityQueries["mondoo-kubernetes-security-pod-apparmor-profile"], `podSpec['securityContext']['appArmorProfile']['type']`)
	require.Contains(t, securityQueries["mondoo-kubernetes-security-pod-apparmor-profile"], `containers.all(securityContext['appArmorProfile']['type']`)

	require.Contains(t, securityQueries["mondoo-kubernetes-security-pod-imagepullsecret-required-for-restricted-registries"], `imageName == /^ghcr\.io\//`)
	require.Contains(t, securityQueries["mondoo-kubernetes-security-pod-imagepullsecret-required-for-restricted-registries"], `imageName == /^quay\.io\//`)
	require.NotContains(t, securityQueries["mondoo-kubernetes-security-pod-imagepullsecret-required-for-restricted-registries"], `imageName == /^docker\.io\//`)
	require.NotContains(t, securityQueries["mondoo-kubernetes-security-pod-imagepullsecret-required-for-restricted-registries"], `imageName == /^registry\.k8s\.io\//`)

	require.NotContains(t, bestPracticesQueries["mondoo-kubernetes-best-practices-hpa-scale-target-kind"], `"DaemonSet"`)
	require.NotContains(t, bestPracticesQueries["mondoo-kubernetes-best-practices-workloads-have-hpa"], `daemonsets.map([namespace, "DaemonSet", name].join("/")).all(hpaTargets.contains(_))`)
	require.Contains(t, bestPracticesQueries["mondoo-kubernetes-best-practices-workloads-have-hpa"], `replicasets.where(ownerReferences.none(kind == "Deployment" && controller == true))`)
}

func kubernetesContentQueries(t *testing.T, path string) map[string]string {
	t.Helper()

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var content mondooKubernetesSecurityContent
	require.NoError(t, yaml.Unmarshal(data, &content))

	queries := map[string]string{}
	for _, query := range content.Queries {
		queries[query.Uid] = query.Mql
	}
	return queries
}

func TestMondooKubernetesKyvernoPolicyBundleLoads(t *testing.T) {
	bundle, err := policy.DefaultBundleLoader().BundleFromPaths("./mondoo-kubernetes-kyverno.mql.yaml")
	require.NoError(t, err)
	require.Len(t, bundle.Policies, 1)
	require.Equal(t, "mondoo-kubernetes-kyverno", bundle.Policies[0].Uid)
	require.Len(t, bundle.Queries, 7)
	for _, query := range bundle.Queries {
		require.NotEmpty(t, query.Uid)
		require.NotEmpty(t, query.Mql, "query %q should have an MQL expression", query.Uid)
	}

	requireKyvernoBundleCompiles(t, bundle, kyvernoContentRuntime(t))
}

func TestMondooKubernetesKyvernoInventoryQuerypack(t *testing.T) {
	bundle, err := policy.DefaultBundleLoader().BundleFromPaths("./querypacks/mondoo-kubernetes-inventory.mql.yaml")
	require.NoError(t, err)
	require.Len(t, bundle.Packs, 1)

	queries := map[string]string{}
	for _, group := range bundle.Packs[0].Groups {
		for _, query := range group.Queries {
			queries[query.Uid] = query.Mql
		}
	}

	kyvernoInventory, ok := queries["k8s-kyverno-inventory"]
	require.True(t, ok, "Kyverno inventory query must be included in the Kubernetes inventory querypack")
	require.Contains(t, kyvernoInventory, "k8s.kyverno")
	require.Contains(t, kyvernoInventory, "policyExceptions")
	require.Contains(t, kyvernoInventory, "mappedPolicyExceptionIds")
	require.Contains(t, kyvernoInventory, "mappedMondooExceptionIds")
	require.Contains(t, kyvernoInventory, "validUntilTime")
	require.Contains(t, kyvernoInventory, "computedStatus")
	require.Contains(t, kyvernoInventory, "statusReasons")
	require.Contains(t, kyvernoInventory, "mappings")

	bundle.ConvertQuerypacks()
	requireKyvernoBundleCompiles(t, bundle, kyvernoContentRuntime(t))
}

func kyvernoContentRuntime(t *testing.T) *providers.Runtime {
	t.Helper()

	runtime := providers.DefaultRuntime()
	schema, ok := runtime.Schema().(providers.ExtensibleSchema)
	require.True(t, ok, "provider runtime schema must support adding source schemas")
	schema.Add("core", testutils.MustLoadSchema(testutils.SchemaProvider{Path: kyvernoSchemaPath(t, "core")}))
	schema.Add("network", testutils.MustLoadSchema(testutils.SchemaProvider{Path: kyvernoSchemaPath(t, "network")}))
	schema.Add("os", testutils.MustLoadSchema(testutils.SchemaProvider{Path: kyvernoSchemaPath(t, "os")}))
	schema.Add("k8s", testutils.MustLoadSchema(testutils.SchemaProvider{Path: kyvernoSchemaPath(t, "k8s")}))
	return runtime
}

func requireKyvernoBundleCompiles(t *testing.T, bundle *policy.Bundle, runtime *providers.Runtime) {
	t.Helper()

	compiled, err := bundle.Compile(context.Background(), runtime.Schema(), nil)
	require.NoError(t, err)
	require.NotNil(t, compiled)
}

func kyvernoSchemaPath(t *testing.T, provider string) string {
	t.Helper()

	path := "./testdata/schema/providers/" + provider + "/resources/" + provider + ".lr"
	_, err := os.Stat(path)
	require.NoErrorf(t, err, "in-repo schema fixture must point to a readable %s.lr schema file", provider)
	return path
}

func TestKyvernoOfficialPolicyMappingsReferenceExistingMondooChecks(t *testing.T) {
	catalogData, err := os.ReadFile("./kyverno/official-policy-mappings.yaml")
	require.NoError(t, err)

	var catalog kyvernoMappingCatalog
	require.NoError(t, yaml.Unmarshal(catalogData, &catalog))
	require.Equal(t, 2, catalog.Version)
	require.NotEmpty(t, catalog.Source.Ref)
	require.Positive(t, catalog.Source.PolicyResourceCount)
	require.Positive(t, catalog.Source.UniquePolicyCount)
	require.GreaterOrEqual(t, catalog.Source.PolicyResourceCount, catalog.Source.UniquePolicyCount)
	require.Len(t, catalog.Policies, catalog.Source.UniquePolicyCount)
	require.NotEmpty(t, catalog.Mappings)

	k8sCheckUids := map[string]map[string]struct{}{}
	k8sPolicyData, err := os.ReadFile("./mondoo-kubernetes-security.mql.yaml")
	require.NoError(t, err)
	var k8sPolicy mondooKubernetesSecurityContent
	require.NoError(t, yaml.Unmarshal(k8sPolicyData, &k8sPolicy))
	k8sCheckUids["mondoo-kubernetes-security"] = map[string]struct{}{}
	for _, query := range k8sPolicy.Queries {
		k8sCheckUids["mondoo-kubernetes-security"][query.Uid] = struct{}{}
	}
	k8sBestPracticesData, err := os.ReadFile("./mondoo-kubernetes-best-practices.mql.yaml")
	require.NoError(t, err)
	var k8sBestPractices mondooKubernetesSecurityContent
	require.NoError(t, yaml.Unmarshal(k8sBestPracticesData, &k8sBestPractices))
	k8sCheckUids["mondoo-kubernetes-best-practices"] = map[string]struct{}{}
	for _, query := range k8sBestPractices.Queries {
		k8sCheckUids["mondoo-kubernetes-best-practices"][query.Uid] = struct{}{}
	}
	require.NotEmpty(t, k8sCheckUids)

	policiesByName := map[string]kyvernoPolicyEntry{}
	statusCounts := map[string]int{}
	resourceCount := 0
	for _, policy := range catalog.Policies {
		require.NotEmpty(t, policy.KyvernoPolicy)
		require.Contains(t, []string{"mapped", "partial", "unmapped"}, policy.MappingStatus, "policy %q has invalid mapping status", policy.KyvernoPolicy)
		statusCounts[policy.MappingStatus]++
		require.NotEmpty(t, policy.Upstream.PolicyResources, "policy %q has no upstream resources", policy.KyvernoPolicy)
		for _, resource := range policy.Upstream.PolicyResources {
			require.NotEmpty(t, resource.Path, "policy %q has an upstream resource without a path", policy.KyvernoPolicy)
			require.NotEmpty(t, resource.APIVersion, "policy %q has an upstream resource without an apiVersion", policy.KyvernoPolicy)
			require.NotEmpty(t, resource.Kind, "policy %q has an upstream resource without a kind", policy.KyvernoPolicy)
			resourceCount++
		}
		for _, rule := range policy.KyvernoRules {
			require.False(t, strings.Contains(rule, ","), "policy %q rule %q should be a single rule name", policy.KyvernoPolicy, rule)
		}
		if policy.MappingStatus == "unmapped" {
			require.Empty(t, policy.MappingRefs, "unmapped policy %q should not reference mappings", policy.KyvernoPolicy)
		} else {
			require.NotEmpty(t, policy.MappingRefs, "mapped policy %q should reference mappings", policy.KyvernoPolicy)
		}
		if _, ok := policiesByName[policy.KyvernoPolicy]; ok {
			t.Fatalf("duplicate Kyverno policy catalog entry for %q", policy.KyvernoPolicy)
		}
		policiesByName[policy.KyvernoPolicy] = policy
	}
	require.Equal(t, catalog.Source.PolicyResourceCount, resourceCount)
	require.Equal(t, len(catalog.Policies), statusCounts["mapped"]+statusCounts["partial"]+statusCounts["unmapped"])
	require.Positive(t, statusCounts["mapped"])
	require.Positive(t, statusCounts["partial"])
	require.Positive(t, statusCounts["unmapped"])

	seenMapping := map[string]struct{}{}
	mappingsByUid := map[string]kyvernoMapping{}
	for _, mapping := range catalog.Mappings {
		require.NotEmpty(t, mapping.Uid)
		require.NotEmpty(t, mapping.KyvernoPolicy)
		require.NotEmpty(t, mapping.KyvernoRules, "mapping %q has no Kyverno rules", mapping.KyvernoPolicy)
		require.NotEmpty(t, mapping.MondooChecks, "mapping %q has no Mondoo checks", mapping.KyvernoPolicy)
		require.Contains(t, []string{"high", "medium", "low"}, mapping.Confidence, "mapping %q has invalid confidence", mapping.Uid)
		require.Contains(t, []string{"mapped", "partial"}, mapping.MappingStatus, "mapping %q has invalid status", mapping.Uid)
		if _, ok := seenMapping[mapping.Uid]; ok {
			t.Fatalf("duplicate Kyverno mapping uid %q", mapping.Uid)
		}
		seenMapping[mapping.Uid] = struct{}{}
		mappingsByUid[mapping.Uid] = mapping
		policy, ok := policiesByName[mapping.KyvernoPolicy]
		require.True(t, ok, "mapping %q references unknown Kyverno policy %q", mapping.Uid, mapping.KyvernoPolicy)
		require.Contains(t, policy.MappingRefs, mapping.Uid, "policy %q does not link back to mapping %q", mapping.KyvernoPolicy, mapping.Uid)

		mondooPolicyUid := mapping.MondooPolicyUid
		if mondooPolicyUid == "" {
			mondooPolicyUid = "mondoo-kubernetes-security"
		}
		checksByPolicy, ok := k8sCheckUids[mondooPolicyUid]
		require.True(t, ok, "mapping %q references unknown Mondoo policy %q", mapping.Uid, mondooPolicyUid)
		for _, check := range mapping.MondooChecks {
			_, ok := checksByPolicy[check]
			require.True(t, ok, "mapping %q references unknown Mondoo check %q in policy %q", mapping.Uid, check, mondooPolicyUid)
		}
		for _, rule := range mapping.KyvernoRules {
			require.False(t, strings.Contains(rule, ","), "mapping %q rule %q should be a single rule name", mapping.Uid, rule)
			require.Contains(t, policy.KyvernoRules, rule, "mapping %q references rule %q that is not in the upstream policy entry", mapping.Uid, rule)
		}
	}
	for _, policy := range catalog.Policies {
		for _, ref := range policy.MappingRefs {
			_, ok := seenMapping[ref]
			require.True(t, ok, "policy %q references unknown mapping %q", policy.KyvernoPolicy, ref)
		}
	}

	require.Equal(t, "mapped", policiesByName["disallow-host-ports"].MappingStatus)
	require.Equal(t, "mapped", policiesByName["disallow-host-process"].MappingStatus)
	require.Equal(t, "mapped", policiesByName["disallow-proc-mount"].MappingStatus)
	require.Equal(t, "mapped", policiesByName["disallow-selinux"].MappingStatus)
	require.Equal(t, "mapped", policiesByName["add-emptydir-sizelimit"].MappingStatus)
	require.Equal(t, "mapped", policiesByName["add-networkpolicy-dns"].MappingStatus)
	require.Equal(t, "mapped", policiesByName["add-networkpolicy"].MappingStatus)
	require.Equal(t, "mapped", policiesByName["add-ns-quota"].MappingStatus)
	require.Equal(t, "mapped", policiesByName["add-psa-labels"].MappingStatus)
	require.Equal(t, "mapped", policiesByName["add-psa-namespace-reporting"].MappingStatus)
	require.Equal(t, "mapped", policiesByName["add-ttl-jobs"].MappingStatus)
	require.Equal(t, "mapped", policiesByName["disallow-default-namespace"].MappingStatus)
	require.Equal(t, "mapped", policiesByName["disallow-latest-tag"].MappingStatus)
	require.Equal(t, "mapped", policiesByName["disallow-ingress-nginx-custom-snippets"].MappingStatus)
	require.Equal(t, "mapped", policiesByName["block-ephemeral-containers"].MappingStatus)
	require.Equal(t, "mapped", policiesByName["restrict-binding-clusteradmin"].MappingStatus)
	require.Equal(t, "mapped", policiesByName["restrict-binding-system-groups"].MappingStatus)
	require.Equal(t, "mapped", policiesByName["restrict-clusterrole-nodesproxy"].MappingStatus)
	require.Equal(t, "partial", policiesByName["restrict-escalation-verbs-roles"].MappingStatus)
	require.Equal(t, "partial", policiesByName["restrict-secret-role-verbs"].MappingStatus)
	require.Equal(t, "mapped", policiesByName["restrict-wildcard-resources"].MappingStatus)
	require.Equal(t, "mapped", policiesByName["restrict-wildcard-verbs"].MappingStatus)
	require.Equal(t, "mapped", policiesByName["deny-commands-in-exec-probe"].MappingStatus)
	require.Equal(t, "partial", policiesByName["add-default-resources"].MappingStatus)
	require.Equal(t, "partial", policiesByName["apply-pss-restricted-profile"].MappingStatus)
	require.Equal(t, "partial", policiesByName["disallow-container-sock-mounts"].MappingStatus)
	require.Equal(t, "partial", policiesByName["deny-privileged-profile"].MappingStatus)
	require.Equal(t, "partial", policiesByName["podsecurity-subrule-restricted-capabilities"].MappingStatus)
	require.Equal(t, "partial", policiesByName["prevent-bare-pods"].MappingStatus)
	require.Equal(t, "partial", policiesByName["require-image-checksum"].MappingStatus)
	require.Equal(t, "mapped", policiesByName["require-imagepullsecrets"].MappingStatus)
	require.Equal(t, "partial", policiesByName["require-qos-guaranteed"].MappingStatus)
	require.Equal(t, "partial", policiesByName["resolve-image-to-digest"].MappingStatus)
	require.Equal(t, "partial", policiesByName["restrict-volume-types"].MappingStatus)
	require.Equal(t, "partial", policiesByName["require-pod-probes"].MappingStatus)
	require.Equal(t, "mapped", policiesByName["require-emptydir-requests-and-limits"].MappingStatus)
	require.Equal(t, "partial", policiesByName["require-requests-limits"].MappingStatus)
	require.Equal(t, "mapped", policiesByName["require-non-root-groups"].MappingStatus)
	require.Equal(t, "mapped", policiesByName["require-run-as-containeruser"].MappingStatus)
	require.Equal(t, "mapped", policiesByName["deployment-has-multiple-replicas"].MappingStatus)
	require.Equal(t, "mapped", policiesByName["forbid-cpu-limits"].MappingStatus)
	require.Equal(t, "mapped", policiesByName["generate-networkpolicy-existing"].MappingStatus)
	require.Equal(t, "partial", policiesByName["check-hpa-exists"].MappingStatus)
	require.Equal(t, "mapped", policiesByName["ingress-host-match-tls"].MappingStatus)
	require.Equal(t, "mapped", policiesByName["limit-containers-per-pod"].MappingStatus)
	require.Equal(t, "mapped", policiesByName["namespace-inventory-check"].MappingStatus)
	require.Equal(t, "mapped", policiesByName["no-secrets"].MappingStatus)
	require.Equal(t, "mapped", policiesByName["pdb-maxunavailable"].MappingStatus)
	require.Equal(t, "partial", policiesByName["pdb-maxunavailable-with-deployments"].MappingStatus)
	require.Equal(t, "mapped", policiesByName["prevent-cr8escape"].MappingStatus)
	require.Equal(t, "mapped", policiesByName["prevent-duplicate-hpa"].MappingStatus)
	require.Equal(t, "mapped", policiesByName["readwriteonce-pod"].MappingStatus)
	require.Equal(t, "mapped", policiesByName["require-labels"].MappingStatus)
	require.Equal(t, "mapped", policiesByName["require-pod-priorityclassname"].MappingStatus)
	require.Equal(t, "mapped", policiesByName["require-qos-burstable"].MappingStatus)
	require.Equal(t, "mapped", policiesByName["validate-probes"].MappingStatus)
	require.Equal(t, "mapped", policiesByName["require-storageclass"].MappingStatus)
	require.Equal(t, "mapped", policiesByName["restrict-networkpolicy-empty-podselector"].MappingStatus)
	require.Equal(t, "mapped", policiesByName["restrict-annotations"].MappingStatus)
	require.Equal(t, "mapped", policiesByName["restrict-apparmor-profiles"].MappingStatus)
	require.Equal(t, "mapped", policiesByName["restrict-deprecated-registry"].MappingStatus)
	require.Equal(t, "partial", policiesByName["restrict-ingress-classes"].MappingStatus)
	require.Equal(t, "mapped", policiesByName["restrict-ingress-paths"].MappingStatus)
	require.Equal(t, "mapped", policiesByName["restrict-jobs"].MappingStatus)
	require.Equal(t, "partial", policiesByName["restrict-node-selection"].MappingStatus)
	require.Equal(t, "mapped", policiesByName["restrict-seccomp"].MappingStatus)
	require.Equal(t, "partial", policiesByName["restrict-seccomp-strict"].MappingStatus)
	require.Equal(t, "partial", policiesByName["restrict-service-port-range"].MappingStatus)
	require.Equal(t, "mapped", policiesByName["restrict-storageclass"].MappingStatus)
	require.Equal(t, "mapped", policiesByName["restrict-sysctls"].MappingStatus)
	require.Equal(t, "mapped", policiesByName["require-network-policy"].MappingStatus)

	capabilitiesExemptMapping := mappingsByUid["podsecurity-subrule-restricted-capabilities/overlapping-security-controls"]
	require.NotEmpty(t, capabilitiesExemptMapping.MondooChecks)
	for _, check := range capabilitiesExemptMapping.MondooChecks {
		require.NotContains(t, check, "capability-", "capability-exempt PodSecurity mapping should not include capability checks")
	}
	seccompExemptMapping := mappingsByUid["podsecurity-subrule-restricted-seccomp/overlapping-security-controls"]
	require.Contains(t, seccompExemptMapping.MondooChecks, "mondoo-kubernetes-security-pod-capability-drop-all")
	require.NotContains(t, seccompExemptMapping.MondooChecks, "mondoo-kubernetes-security-pod-seccomp-profile")
	baselineMapping := mappingsByUid["podsecurity-subrule-baseline/overlapping-security-controls"]
	require.Contains(t, baselineMapping.MondooChecks, "mondoo-kubernetes-security-pod-hostprocess")
	require.Contains(t, baselineMapping.MondooChecks, "mondoo-kubernetes-security-pod-proc-mount")
	require.Contains(t, baselineMapping.MondooChecks, "mondoo-kubernetes-security-pod-safe-sysctls")
	require.Contains(t, baselineMapping.MondooChecks, "mondoo-kubernetes-security-pod-selinux-type")
	require.Contains(t, baselineMapping.MondooChecks, "mondoo-kubernetes-security-pod-selinux-user-role")
	require.Contains(t, baselineMapping.MondooChecks, "mondoo-kubernetes-security-pod-seccomp-profile")
	ephemeralContainersMapping := mappingsByUid["block-ephemeral-containers/block-ephemeral-containers"]
	require.Contains(t, ephemeralContainersMapping.MondooChecks, "mondoo-kubernetes-security-pod-no-ephemeral-containers")
	nonRootGroupsMapping := mappingsByUid["require-non-root-groups/non-root-group-ids"]
	require.Contains(t, nonRootGroupsMapping.MondooChecks, "mondoo-kubernetes-security-pod-non-root-groups")
	require.Contains(t, nonRootGroupsMapping.KyvernoRules, "check-runasgroup")
	require.Contains(t, nonRootGroupsMapping.KyvernoRules, "check-supplementalgroups")
	require.Contains(t, nonRootGroupsMapping.KyvernoRules, "check-fsgroup")
	hostProcessMapping := mappingsByUid["disallow-host-process/host-process-containers"]
	require.Contains(t, hostProcessMapping.MondooChecks, "mondoo-kubernetes-security-pod-hostprocess")
	procMountMapping := mappingsByUid["disallow-proc-mount/check-proc-mount"]
	require.Contains(t, procMountMapping.MondooChecks, "mondoo-kubernetes-security-pod-proc-mount")
	selinuxTypeMapping := mappingsByUid["disallow-selinux/selinux-type"]
	require.Contains(t, selinuxTypeMapping.MondooChecks, "mondoo-kubernetes-security-pod-selinux-type")
	selinuxUserRoleMapping := mappingsByUid["disallow-selinux/selinux-user-role"]
	require.Contains(t, selinuxUserRoleMapping.MondooChecks, "mondoo-kubernetes-security-pod-selinux-user-role")
	seccompMapping := mappingsByUid["restrict-seccomp/check-seccomp"]
	require.Contains(t, seccompMapping.MondooChecks, "mondoo-kubernetes-security-pod-seccomp-profile")
	seccompStrictMapping := mappingsByUid["restrict-seccomp-strict/check-seccomp-strict"]
	require.Contains(t, seccompStrictMapping.MondooChecks, "mondoo-kubernetes-security-pod-seccomp-profile")
	require.Equal(t, "partial", seccompStrictMapping.MappingStatus)
	sysctlsMapping := mappingsByUid["restrict-sysctls/safe-sysctls"]
	require.Contains(t, sysctlsMapping.MondooChecks, "mondoo-kubernetes-security-pod-safe-sysctls")
	psaLabelsMapping := mappingsByUid["add-psa-labels/namespace-psa-enforce-warn-labels"]
	require.Contains(t, psaLabelsMapping.MondooChecks, "mondoo-kubernetes-security-namespace-psa-enforce-warn-labels")
	psaReportingMapping := mappingsByUid["add-psa-namespace-reporting/namespace-psa-labels"]
	require.Contains(t, psaReportingMapping.MondooChecks, "mondoo-kubernetes-security-namespace-psa-labels")
	denyPrivilegedProfileMapping := mappingsByUid["deny-privileged-profile/namespace-psa-enforce-not-privileged"]
	require.Equal(t, "partial", denyPrivilegedProfileMapping.MappingStatus)
	require.Contains(t, denyPrivilegedProfileMapping.MondooChecks, "mondoo-kubernetes-security-namespace-psa-enforce-not-privileged")
	clusterAdminMapping := mappingsByUid["restrict-binding-clusteradmin/cluster-admin-bindings"]
	require.Contains(t, clusterAdminMapping.MondooChecks, "mondoo-kubernetes-security-rbac-no-cluster-admin-bindings")
	systemGroupMapping := mappingsByUid["restrict-binding-system-groups/system-group-bindings"]
	require.Contains(t, systemGroupMapping.MondooChecks, "mondoo-kubernetes-security-rbac-no-system-group-bindings")
	require.Contains(t, systemGroupMapping.KyvernoRules, "restrict-subject-groups")
	ingressClassMapping := mappingsByUid["restrict-ingress-classes/approved-class-annotation"]
	require.Equal(t, "partial", ingressClassMapping.MappingStatus)
	require.Contains(t, ingressClassMapping.Reason, "spec.ingressClassName")
	servicePortRangeMapping := mappingsByUid["restrict-service-port-range/restrict-port-range"]
	require.Equal(t, "partial", servicePortRangeMapping.MappingStatus)
	require.Contains(t, servicePortRangeMapping.Reason, "NodePort")
	nodesProxyMapping := mappingsByUid["restrict-clusterrole-nodesproxy/nodes-proxy"]
	require.Contains(t, nodesProxyMapping.MondooChecks, "mondoo-kubernetes-security-rbac-no-nodes-proxy")
	escalationMapping := mappingsByUid["restrict-escalation-verbs-roles/rbac-escalation-verbs"]
	require.Equal(t, "partial", escalationMapping.MappingStatus)
	require.Contains(t, escalationMapping.Reason, "wildcard verbs")
	require.Contains(t, escalationMapping.MondooChecks, "mondoo-kubernetes-security-rbac-no-escalation-verbs")
	secretRoleVerbsMapping := mappingsByUid["restrict-secret-role-verbs/secret-read-verbs"]
	require.Equal(t, "partial", secretRoleVerbsMapping.MappingStatus)
	require.Contains(t, secretRoleVerbsMapping.Reason, "wildcard resources")
	require.Contains(t, secretRoleVerbsMapping.MondooChecks, "mondoo-kubernetes-security-rbac-no-secret-read-verbs")
	servicePortRangeMapping = mappingsByUid["restrict-service-port-range/restrict-port-range"]
	require.Equal(t, "mondoo-kubernetes-best-practices", servicePortRangeMapping.MondooPolicyUid)
	require.Contains(t, servicePortRangeMapping.MondooChecks, "mondoo-kubernetes-best-practices-service-ports-approved-range")
	livenessExecProbeMapping := mappingsByUid["deny-commands-in-exec-probe/liveness-exec-probe-no-debug-commands"]
	require.Contains(t, livenessExecProbeMapping.MondooChecks, "mondoo-kubernetes-security-pod-liveness-exec-probe-no-debug-commands")
	require.Contains(t, livenessExecProbeMapping.MondooChecks, "mondoo-kubernetes-security-deployment-liveness-exec-probe-no-debug-commands")
	latestTagMapping := mappingsByUid["disallow-latest-tag/image-tag-not-latest"]
	require.Contains(t, latestTagMapping.MondooChecks, "mondoo-kubernetes-security-pod-image-tag-not-latest")
	require.Contains(t, latestTagMapping.MondooChecks, "mondoo-kubernetes-security-deployment-image-tag-not-latest")
	ingressNginxSnippetMapping := mappingsByUid["disallow-ingress-nginx-custom-snippets/no-custom-snippets"]
	require.Contains(t, ingressNginxSnippetMapping.MondooChecks, "mondoo-kubernetes-security-configmap-ingress-nginx-snippet-annotations-disabled")
	require.Contains(t, ingressNginxSnippetMapping.MondooChecks, "mondoo-kubernetes-security-ingress-nginx-no-custom-snippets")
	fluxAnnotationMapping := mappingsByUid["restrict-annotations/flux-v1-annotations"]
	require.Contains(t, fluxAnnotationMapping.KyvernoRules, "block-flux-v1")
	require.Contains(t, fluxAnnotationMapping.MondooChecks, "mondoo-kubernetes-security-pod-no-flux-v1-annotations")
	require.Contains(t, fluxAnnotationMapping.MondooChecks, "mondoo-kubernetes-security-deployment-no-flux-v1-annotations")
	require.NotContains(t, fluxAnnotationMapping.MondooChecks, "mondoo-kubernetes-security-replicaset-no-flux-v1-annotations")
	ingressTlsHostMapping := mappingsByUid["ingress-host-match-tls/host-match-tls"]
	require.Equal(t, "mondoo-kubernetes-best-practices", ingressTlsHostMapping.MondooPolicyUid)
	require.Contains(t, ingressTlsHostMapping.MondooChecks, "mondoo-kubernetes-best-practices-ingress-tls-hosts-match-rules")
	hpaExistsMapping := mappingsByUid["check-hpa-exists/workloads-have-hpa"]
	require.Equal(t, "mondoo-kubernetes-best-practices", hpaExistsMapping.MondooPolicyUid)
	require.Equal(t, "partial", hpaExistsMapping.MappingStatus)
	require.Contains(t, hpaExistsMapping.MondooChecks, "mondoo-kubernetes-best-practices-workloads-have-hpa")
	pdbWithDeploymentsMapping := mappingsByUid["pdb-maxunavailable-with-deployments/pdb-maxunavailable-nonzero"]
	require.Equal(t, "partial", pdbWithDeploymentsMapping.MappingStatus)
	require.Equal(t, "mondoo-kubernetes-best-practices", pdbWithDeploymentsMapping.MondooPolicyUid)
	require.Contains(t, pdbWithDeploymentsMapping.MondooChecks, "mondoo-kubernetes-best-practices-pdb-maxunavailable-nonzero")
	wildcardResourcesMapping := mappingsByUid["restrict-wildcard-resources/wildcard-resources"]
	require.Contains(t, wildcardResourcesMapping.MondooChecks, "mondoo-kubernetes-security-rbac-no-wildcard-resources")
	wildcardVerbsMapping := mappingsByUid["restrict-wildcard-verbs/wildcard-verbs"]
	require.Contains(t, wildcardVerbsMapping.MondooChecks, "mondoo-kubernetes-security-rbac-no-wildcard-verbs")
	pssMutationMapping := mappingsByUid["apply-pss-restricted-profile/restricted-security-context"]
	require.Contains(t, pssMutationMapping.MondooChecks, "mondoo-kubernetes-security-pod-allowprivilegeescalation")
	require.Contains(t, pssMutationMapping.MondooChecks, "mondoo-kubernetes-security-pod-capability-drop-all")
	qosRequestMapping := mappingsByUid["require-qos-guaranteed/resource-requests"]
	require.Equal(t, "mondoo-kubernetes-best-practices", qosRequestMapping.MondooPolicyUid)
	require.Contains(t, qosRequestMapping.MondooChecks, "mondoo-kubernetes-best-practices-pod-requestcpu")
	digestMapping := mappingsByUid["resolve-image-to-digest/image-reference-immutability"]
	require.Contains(t, digestMapping.MondooChecks, "mondoo-kubernetes-security-pod-imagepull")
	checksumMapping := mappingsByUid["require-image-checksum/image-reference-immutability"]
	require.Contains(t, checksumMapping.MondooChecks, "mondoo-kubernetes-security-pod-imagepull")
	imagePullSecretsMapping := mappingsByUid["require-imagepullsecrets/imagepullsecret-for-restricted-registries"]
	require.Contains(t, imagePullSecretsMapping.MondooChecks, "mondoo-kubernetes-security-pod-imagepullsecret-required-for-restricted-registries")
	deprecatedRegistryMapping := mappingsByUid["restrict-deprecated-registry/k8s-gcr-registry"]
	require.Contains(t, deprecatedRegistryMapping.MondooChecks, "mondoo-kubernetes-security-pod-no-deprecated-k8s-gcr-registry")
	ttlJobsMapping := mappingsByUid["add-ttl-jobs/direct-job-ttl-after-finished"]
	require.Equal(t, "mondoo-kubernetes-best-practices", ttlJobsMapping.MondooPolicyUid)
	require.Contains(t, ttlJobsMapping.MondooChecks, "mondoo-kubernetes-best-practices-job-ttl-after-finished")
	emptyDirSizeMapping := mappingsByUid["add-emptydir-sizelimit/emptydir-size-limit"]
	require.Equal(t, "mondoo-kubernetes-best-practices", emptyDirSizeMapping.MondooPolicyUid)
	require.Contains(t, emptyDirSizeMapping.MondooChecks, "mondoo-kubernetes-best-practices-pod-emptydir-size-limit")
	emptyDirResourcesMapping := mappingsByUid["require-emptydir-requests-and-limits/emptydir-ephemeral-storage-resources"]
	require.Equal(t, "mondoo-kubernetes-best-practices", emptyDirResourcesMapping.MondooPolicyUid)
	require.Contains(t, emptyDirResourcesMapping.MondooChecks, "mondoo-kubernetes-best-practices-pod-emptydir-ephemeral-storage-resources")
	defaultDenyMapping := mappingsByUid["add-networkpolicy/default-deny-networkpolicy"]
	require.Equal(t, "mondoo-kubernetes-best-practices", defaultDenyMapping.MondooPolicyUid)
	require.Contains(t, defaultDenyMapping.MondooChecks, "mondoo-kubernetes-best-practices-namespace-default-deny-networkpolicy")
	dnsNetworkPolicyMapping := mappingsByUid["add-networkpolicy-dns/allow-dns-networkpolicy"]
	require.Equal(t, "mondoo-kubernetes-best-practices", dnsNetworkPolicyMapping.MondooPolicyUid)
	require.Contains(t, dnsNetworkPolicyMapping.MondooChecks, "mondoo-kubernetes-best-practices-namespace-allow-dns-networkpolicy")
	namespaceLimitRangeMapping := mappingsByUid["add-ns-quota/namespace-limitrange"]
	require.Equal(t, "mondoo-kubernetes-best-practices", namespaceLimitRangeMapping.MondooPolicyUid)
	require.Contains(t, namespaceLimitRangeMapping.MondooChecks, "mondoo-kubernetes-best-practices-namespace-limitrange")
	namespaceResourceQuotaMapping := mappingsByUid["add-ns-quota/namespace-resourcequota"]
	require.Equal(t, "mondoo-kubernetes-best-practices", namespaceResourceQuotaMapping.MondooPolicyUid)
	require.Contains(t, namespaceResourceQuotaMapping.MondooChecks, "mondoo-kubernetes-best-practices-namespace-resourcequota")
	containerUserMapping := mappingsByUid["require-run-as-containeruser/windows-runas-containeruser"]
	require.Contains(t, containerUserMapping.MondooChecks, "mondoo-kubernetes-security-pod-windows-runas-containeruser")
	apparmorMapping := mappingsByUid["restrict-apparmor-profiles/pod-apparmor-profile"]
	require.Contains(t, apparmorMapping.MondooChecks, "mondoo-kubernetes-security-pod-apparmor-profile")
	nodeSelectorMapping := mappingsByUid["restrict-node-selection/pod-node-selector"]
	require.Equal(t, "partial", nodeSelectorMapping.MappingStatus)
	require.Equal(t, "mondoo-kubernetes-best-practices", nodeSelectorMapping.MondooPolicyUid)
	require.Contains(t, nodeSelectorMapping.MondooChecks, "mondoo-kubernetes-best-practices-pod-no-node-selector")
	require.NotContains(t, nodeSelectorMapping.KyvernoRules, "restrict-nodename")
	restrictJobsMapping := mappingsByUid["restrict-jobs/job-owned-by-cronjob"]
	require.Contains(t, restrictJobsMapping.MondooChecks, "mondoo-kubernetes-security-job-created-by-cronjob")
	egressDefaultDenyMapping := mappingsByUid["generate-networkpolicy-existing/egress-default-deny-networkpolicy"]
	require.Equal(t, "mondoo-kubernetes-best-practices", egressDefaultDenyMapping.MondooPolicyUid)
	require.Contains(t, egressDefaultDenyMapping.MondooChecks, "mondoo-kubernetes-best-practices-namespace-egress-default-deny-networkpolicy")
	barePodMapping := mappingsByUid["prevent-bare-pods/pod-owner-reference"]
	require.Equal(t, "mondoo-kubernetes-best-practices", barePodMapping.MondooPolicyUid)
	require.Contains(t, barePodMapping.MondooChecks, "mondoo-kubernetes-best-practices-pod-no-owner")
	cr8escapeMapping := mappingsByUid["prevent-cr8escape/sysctl-values-no-cr8escape-metacharacters"]
	require.Contains(t, cr8escapeMapping.MondooChecks, "mondoo-kubernetes-security-pod-sysctls-no-cr8escape-values")
	require.Contains(t, cr8escapeMapping.MondooChecks, "mondoo-kubernetes-security-deployment-sysctls-no-cr8escape-values")
	noCpuLimitsMapping := mappingsByUid["forbid-cpu-limits/no-cpu-limits"]
	require.Equal(t, "mondoo-kubernetes-best-practices", noCpuLimitsMapping.MondooPolicyUid)
	require.Contains(t, noCpuLimitsMapping.MondooChecks, "mondoo-kubernetes-best-practices-pod-no-cpu-limits")
	require.Contains(t, noCpuLimitsMapping.MondooChecks, "mondoo-kubernetes-best-practices-deployment-no-cpu-limits")
	storageClassMapping := mappingsByUid["require-storageclass/pvc-storageclass"]
	require.Equal(t, "mondoo-kubernetes-best-practices", storageClassMapping.MondooPolicyUid)
	require.Contains(t, storageClassMapping.MondooChecks, "mondoo-kubernetes-best-practices-pvc-storageclass")
	restrictStorageClassMapping := mappingsByUid["restrict-storageclass/reclaim-policy-delete"]
	require.Equal(t, "mondoo-kubernetes-best-practices", restrictStorageClassMapping.MondooPolicyUid)
	require.Contains(t, restrictStorageClassMapping.MondooChecks, "mondoo-kubernetes-best-practices-storageclass-reclaim-policy-delete")
	namespaceNetworkPolicyMapping := mappingsByUid["namespace-inventory-check/networkpolicies"]
	require.Equal(t, "mondoo-kubernetes-best-practices", namespaceNetworkPolicyMapping.MondooPolicyUid)
	require.Contains(t, namespaceNetworkPolicyMapping.MondooChecks, "mondoo-kubernetes-best-practices-namespace-networkpolicy")
	requireNetworkPolicyMapping := mappingsByUid["require-network-policy/require-network-policy"]
	require.Equal(t, "mondoo-kubernetes-best-practices", requireNetworkPolicyMapping.MondooPolicyUid)
	require.Contains(t, requireNetworkPolicyMapping.MondooChecks, "mondoo-kubernetes-best-practices-namespace-networkpolicy")
	noSecretsEnvMapping := mappingsByUid["no-secrets/no-secret-env-vars"]
	require.Contains(t, noSecretsEnvMapping.MondooChecks, "mondoo-kubernetes-security-pod-no-secret-env-vars")
	require.Contains(t, noSecretsEnvMapping.KyvernoRules, "secrets-not-from-env-envFrom-and-volumes")
	noSecretsVolumeMapping := mappingsByUid["no-secrets/no-secret-volumes"]
	require.Contains(t, noSecretsVolumeMapping.MondooChecks, "mondoo-kubernetes-security-pod-no-secret-volumes")
	require.Contains(t, noSecretsVolumeMapping.KyvernoRules, "secrets-not-from-env-envFrom-and-volumes")
	pdbMaxUnavailableMapping := mappingsByUid["pdb-maxunavailable/pdb-maxunavailable-nonzero"]
	require.Equal(t, "mondoo-kubernetes-best-practices", pdbMaxUnavailableMapping.MondooPolicyUid)
	require.Contains(t, pdbMaxUnavailableMapping.MondooChecks, "mondoo-kubernetes-best-practices-pdb-maxunavailable-nonzero")
	hpaDuplicateMapping := mappingsByUid["prevent-duplicate-hpa/no-duplicate-targets"]
	require.Equal(t, "mondoo-kubernetes-best-practices", hpaDuplicateMapping.MondooPolicyUid)
	require.Contains(t, hpaDuplicateMapping.MondooChecks, "mondoo-kubernetes-best-practices-hpa-no-duplicate-targets")
	hpaKindMapping := mappingsByUid["prevent-duplicate-hpa/scale-target-kind"]
	require.Equal(t, "mondoo-kubernetes-best-practices", hpaKindMapping.MondooPolicyUid)
	require.Contains(t, hpaKindMapping.MondooChecks, "mondoo-kubernetes-best-practices-hpa-scale-target-kind")
	readWriteOncePodMapping := mappingsByUid["readwriteonce-pod/readwrite-pvc-single-pod"]
	require.Equal(t, "mondoo-kubernetes-best-practices", readWriteOncePodMapping.MondooPolicyUid)
	require.Contains(t, readWriteOncePodMapping.MondooChecks, "mondoo-kubernetes-best-practices-pvc-readwriteoncepod")
	requireLabelsMapping := mappingsByUid["require-labels/pod-app-name-label"]
	require.Equal(t, "mondoo-kubernetes-best-practices", requireLabelsMapping.MondooPolicyUid)
	require.Contains(t, requireLabelsMapping.MondooChecks, "mondoo-kubernetes-best-practices-pod-label-app-name")
	priorityClassMapping := mappingsByUid["require-pod-priorityclassname/check-priorityclassname"]
	require.Equal(t, "mondoo-kubernetes-best-practices", priorityClassMapping.MondooPolicyUid)
	require.Contains(t, priorityClassMapping.MondooChecks, "mondoo-kubernetes-best-practices-pod-priorityclassname")
	validateProbesMapping := mappingsByUid["validate-probes/liveness-readiness-probes-different"]
	require.Equal(t, "mondoo-kubernetes-best-practices", validateProbesMapping.MondooPolicyUid)
	require.Contains(t, validateProbesMapping.MondooChecks, "mondoo-kubernetes-best-practices-deployment-probes-different")
	require.Contains(t, validateProbesMapping.MondooChecks, "mondoo-kubernetes-best-practices-daemonset-probes-different")
}
