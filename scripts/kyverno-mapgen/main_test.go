// Copyright Mondoo, Inc. 2026
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildCatalogPreservesReviewedMappings(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "pod-security/disallow-host-ports.yaml", `
apiVersion: kyverno.io/v1
kind: ClusterPolicy
metadata:
  name: disallow-host-ports
  annotations:
    policies.kyverno.io/title: Disallow hostPorts
    policies.kyverno.io/category: Pod Security Standards (Baseline)
spec:
  rules:
    - name: host-ports-none
      validate:
        message: no host ports
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: ignored
`)
	writeFile(t, root, "pod-security-vpol/disallow-host-ports.yaml", `
apiVersion: policies.kyverno.io/v1alpha1
kind: ValidatingPolicy
metadata:
  name: disallow-host-ports
  annotations:
    policies.kyverno.io/title: Disallow hostPorts in ValidatingPolicy
    policies.kyverno.io/category: Pod Security Standards (Baseline) in ValidatingPolicy
spec:
  validations:
    - name: validation
      expression: "true"
`)

	existing := &catalog{
		Mappings: []mappingEntry{
			{
				UID:           "disallow-host-ports/host-ports-none",
				KyvernoPolicy: "disallow-host-ports",
				KyvernoRules:  []string{"host-ports-none"},
				MondooChecks:  []string{"mondoo-kubernetes-security-pod-ports-hostport"},
				Confidence:    "high",
				MappingStatus: "mapped",
			},
		},
	}

	generated, err := buildCatalog(root, "test-ref", existing)
	require.NoError(t, err)

	require.Equal(t, 2, generated.Source.PolicyResourceCount)
	require.Equal(t, 1, generated.Source.UniquePolicyCount)
	require.Len(t, generated.Policies, 1)
	policy := generated.Policies[0]
	require.Equal(t, "disallow-host-ports", policy.KyvernoPolicy)
	require.Equal(t, "partial", policy.MappingStatus)
	require.Equal(t, []string{"disallow-host-ports/host-ports-none"}, policy.MappingRefs)
	require.Equal(t, []string{"host-ports-none", "validation"}, policy.KyvernoRules)
	require.Equal(t, []string{"Pod Security Standards (Baseline)", "Pod Security Standards (Baseline) in ValidatingPolicy"}, policy.Upstream.Categories)
	require.Len(t, policy.Upstream.PolicyResources, 2)
}

func TestBuildCatalogMarksPoliciesWithoutMappingsUnmapped(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "other/require-labels.yaml", `
apiVersion: kyverno.io/v1
kind: Policy
metadata:
  name: require-labels
  annotations:
    policies.kyverno.io/title: Require Labels
    policies.kyverno.io/category: Other, Best Practices
spec:
  rules:
    - name: check-labels
      validate:
        message: labels required
`)

	generated, err := buildCatalog(root, "test-ref", nil)
	require.NoError(t, err)

	require.Len(t, generated.Policies, 1)
	policy := generated.Policies[0]
	require.Equal(t, "unmapped", policy.MappingStatus)
	require.Empty(t, policy.MappingRefs)
	require.Equal(t, []string{"Other, Best Practices"}, policy.Upstream.Categories)
	require.Empty(t, generated.Mappings)
}

func TestBuildCatalogExtractsSingleGenerateRule(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "other/generate-configmap.yaml", `
apiVersion: policies.kyverno.io/v1alpha1
kind: GeneratingPolicy
metadata:
  name: generate-configmap
spec:
  generate:
    name: generated-configmap
    synchronize: true
`)

	generated, err := buildCatalog(root, "test-ref", nil)
	require.NoError(t, err)

	require.Len(t, generated.Policies, 1)
	require.Equal(t, []string{"generated-configmap"}, generated.Policies[0].KyvernoRules)
}

func TestCatalogSemanticallyEqualDoesNotMutateInputs(t *testing.T) {
	actual := &catalog{
		Policies: []policyEntry{
			{
				KyvernoPolicy: "z-policy",
				MappingRefs:   []string{"z-ref", "a-ref"},
				KyvernoRules:  []string{"z-rule", "a-rule"},
				Upstream: upstreamPolicySet{
					Titles:     []string{"z-title", "a-title"},
					Categories: []string{"z-category", "a-category"},
					PolicyResources: []policyResource{
						{Path: "z.yaml", APIVersion: "kyverno.io/v1", Kind: "Policy"},
						{Path: "a.yaml", APIVersion: "kyverno.io/v1", Kind: "Policy"},
					},
				},
			},
			{KyvernoPolicy: "a-policy"},
		},
		Mappings: []mappingEntry{
			{
				UID:          "z-mapping",
				KyvernoRules: []string{"z-rule", "a-rule"},
				MondooChecks: []string{"z-check", "a-check"},
			},
			{UID: "a-mapping"},
		},
	}
	expected := &catalog{
		Policies: []policyEntry{
			{KyvernoPolicy: "a-policy"},
			{
				KyvernoPolicy: "z-policy",
				MappingRefs:   []string{"a-ref", "z-ref"},
				KyvernoRules:  []string{"a-rule", "z-rule"},
				Upstream: upstreamPolicySet{
					Titles:     []string{"a-title", "z-title"},
					Categories: []string{"a-category", "z-category"},
					PolicyResources: []policyResource{
						{Path: "a.yaml", APIVersion: "kyverno.io/v1", Kind: "Policy"},
						{Path: "z.yaml", APIVersion: "kyverno.io/v1", Kind: "Policy"},
					},
				},
			},
		},
		Mappings: []mappingEntry{
			{UID: "a-mapping"},
			{
				UID:          "z-mapping",
				KyvernoRules: []string{"a-rule", "z-rule"},
				MondooChecks: []string{"a-check", "z-check"},
			},
		},
	}

	require.True(t, catalogSemanticallyEqual(actual, expected))
	require.Equal(t, "z-policy", actual.Policies[0].KyvernoPolicy)
	require.Equal(t, []string{"z-ref", "a-ref"}, actual.Policies[0].MappingRefs)
	require.Equal(t, []string{"z-rule", "a-rule"}, actual.Policies[0].KyvernoRules)
	require.Equal(t, "z-mapping", actual.Mappings[0].UID)
	require.Equal(t, []string{"z-check", "a-check"}, actual.Mappings[0].MondooChecks)
}

func TestMarshalCatalogIncludesHeader(t *testing.T) {
	data, err := marshalCatalog(&catalog{
		Version: catalogVersion,
		Source:  catalogSource{Ref: "test-ref"},
	})
	require.NoError(t, err)
	require.Contains(t, string(data), "# Copyright Mondoo, Inc. 2026\n# SPDX-License-Identifier: BUSL-1.1\n")
	require.Contains(t, string(data), "version: 2\n")
}

func writeFile(t *testing.T, root string, rel string, content string) {
	t.Helper()

	path := filepath.Join(root, rel)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}
