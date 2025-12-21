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
			score:      0x5, // TODO: remove mondoo-aws-security-root-account-mfa-enabled as this is standard now
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
			assert.Equal(t, test.score, score.Value)
		})
	}
}
