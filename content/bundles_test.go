// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package content

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v12/logger"
	"go.mondoo.com/cnquery/v12/providers"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/inventory"
	"go.mondoo.com/cnspec/v12/policy"
	"go.mondoo.com/cnspec/v12/policy/scan"
)

func init() {
	logger.Set("info")
}

func TestMain(m *testing.M) {
	// ensure providers are loaded
	providers.EnsureProvider(providers.ProviderLookup{ProviderName: "terraform"}, true, nil)
	providers.EnsureProvider(providers.ProviderLookup{ProviderName: "k8s"}, true, nil)

	// Run tests
	exitVal := m.Run()
	os.Exit(exitVal)
}

func runBundle(policyBundlePath string, policyMrn string, asset *inventory.Asset) (*policy.Report, error) {
	ctx := context.Background()
	policyBundle, err := policy.BundleFromPaths(policyBundlePath)
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

func TestKubernetesBundles(t *testing.T) {
	type TestCase struct {
		bundleFile string
		testDir    string
		policyMrn  string
		score      uint32
	}

	tests := []TestCase{
		{
			bundleFile: "./mondoo-kubernetes-security.mql.yaml",
			testDir:    "./testdata/mondoo-kubernetes-security-pass",
			policyMrn:  "//policy.api.mondoo.app/policies/mondoo-kubernetes-security",
			score:      100,
		},
		{
			bundleFile: "./mondoo-kubernetes-security.mql.yaml",
			testDir:    "./testdata/mondoo-kubernetes-security-fail",
			policyMrn:  "//policy.api.mondoo.app/policies/mondoo-kubernetes-security",
			score:      0x0,
		},
	}

	for i := range tests {
		test := tests[i]
		t.Run(test.testDir, func(t *testing.T) {
			report, err := runBundle(test.bundleFile, test.policyMrn, &inventory.Asset{
				Connections: []*inventory.Config{{
					Type: "k8s",
					Options: map[string]string{
						"path": test.testDir,
					},
					Discover: &inventory.Discovery{
						Targets: []string{"pods"}, // ignore the manifest which does not return anything
					},
				}},
			})
			require.NoError(t, err)

			score := report.Scores[test.policyMrn]
			assert.Equal(t, test.score, score.Value)
		})
	}
}

func TestTerraformBundles(t *testing.T) {
	type TestCase struct {
		bundleFile string
		testDir    string
		policyMrn  string
		score      uint32
	}

	tests := []TestCase{
		{
			bundleFile: "./mondoo-aws-security.mql.yaml",
			testDir:    "./testdata/mondoo-aws-security-tf-pass",
			policyMrn:  "//policy.api.mondoo.app/policies/mondoo-aws-security",
			score:      100,
		}, {
			bundleFile: "./mondoo-aws-security.mql.yaml",
			testDir:    "./testdata/mondoo-aws-security-tf-fail",
			policyMrn:  "//policy.api.mondoo.app/policies/mondoo-aws-security",
			score:      0,
		}, {
			bundleFile: "./mondoo-gcp-security.mql.yaml",
			testDir:    "./testdata/mondoo-gcp-security-tf-pass",
			policyMrn:  "//policy.api.mondoo.app/policies/mondoo-gcp-security",
			score:      100,
		}, {
			bundleFile: "./mondoo-gcp-security.mql.yaml",
			testDir:    "./testdata/mondoo-gcp-security-tf-fail",
			policyMrn:  "//policy.api.mondoo.app/policies/mondoo-gcp-security",
			score:      0,
		},
	}

	for i := range tests {
		test := tests[i]
		t.Run(test.testDir, func(t *testing.T) {
			report, err := runBundle(test.bundleFile, test.policyMrn, &inventory.Asset{
				Connections: []*inventory.Config{
					{
						Type: "terraform-hcl",
						Options: map[string]string{
							"path": test.testDir,
						},
					},
				},
			})
			require.NoError(t, err)

			score := report.Scores[test.policyMrn]
			assert.Equal(t, test.score, score.Value)
		})
	}
}
