// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scans

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/v13/policy"
	"go.mondoo.com/cnspec/v13/policy/scan"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
)

// bundleFixtures holds one sample project per expected whole-bundle score: a
// small Terraform or Kubernetes tree that a policy should score 100 or 0
// against. Unlike the per-check fixtures in fixtures/iac-variants, these
// exercise the bundle end to end, so they catch a new check that fires on a
// clean project.
const bundleFixtures = "./fixtures/bundles"

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
			bundleFile: "mondoo-kubernetes-security.mql.yaml",
			testDir:    "mondoo-kubernetes-security-pass",
			policyMrn:  "//policy.api.mondoo.app/policies/mondoo-kubernetes-security",
			score:      100,
		},
		{
			provider:   "k8s",
			bundleFile: "mondoo-kubernetes-security.mql.yaml",
			testDir:    "mondoo-kubernetes-security-fail",
			policyMrn:  "//policy.api.mondoo.app/policies/mondoo-kubernetes-security",
			score:      0x0,
		},
		// cnspec scan terraform hcl fixtures/bundles/mondoo-aws-security-tf-pass -f mondoo-aws-security.mql.yaml
		{
			provider:   "terraform",
			bundleFile: "mondoo-aws-security.mql.yaml",
			testDir:    "mondoo-aws-security-tf-pass",
			policyMrn:  "//policy.api.mondoo.app/policies/mondoo-aws-security",
			score:      100,
		},
		{
			provider:   "terraform",
			bundleFile: "mondoo-aws-security.mql.yaml",
			testDir:    "mondoo-aws-security-tf-fail",
			policyMrn:  "//policy.api.mondoo.app/policies/mondoo-aws-security",
			score:      0,
		},
		// cnspec scan terraform hcl fixtures/bundles/mondoo-azure-security-tf-pass -f mondoo-azure-security.mql.yaml
		// TODO: enrich azure tests with HCL test cases
		//{
		//	provider:   "terraform",
		//	bundleFile: "mondoo-azure-security.mql.yaml",
		//	testDir:    "mondoo-azure-security-tf-pass",
		//	policyMrn:  "//policy.api.mondoo.app/policies/mondoo-azure-security",
		//	score:      0x5,
		//},
		//{
		//	provider:   "terraform",
		//	bundleFile: "mondoo-azure-security.mql.yaml",
		//	testDir:    "mondoo-azure-security-tf-fail",
		//	policyMrn:  "//policy.api.mondoo.app/policies/mondoo-azure-security",
		//	score:      0,
		//},
		// cnspec scan terraform hcl fixtures/bundles/mondoo-gcp-security-tf-pass -f mondoo-gcp-security.mql.yaml
		{
			provider:   "terraform",
			bundleFile: "mondoo-gcp-security.mql.yaml",
			testDir:    "mondoo-gcp-security-tf-pass",
			policyMrn:  "//policy.api.mondoo.app/policies/mondoo-gcp-security",
			score:      100,
		},
		{
			provider:   "terraform",
			bundleFile: "mondoo-gcp-security.mql.yaml",
			testDir:    "mondoo-gcp-security-tf-fail",
			policyMrn:  "//policy.api.mondoo.app/policies/mondoo-gcp-security",
			score:      0,
		},
	}

	for i := range tests {
		test := tests[i]
		t.Run(test.testDir, func(t *testing.T) {
			fixtureDir := filepath.Join(bundleFixtures, test.testDir)

			var asset *inventory.Asset
			switch test.provider {
			case "terraform":
				asset = tfAsset(fixtureDir)
			case "k8s":
				asset = k8sAsset(fixtureDir)
			default:
				t.Fatalf("unknown provider type: %s", test.provider)
			}
			report, err := runBundle(bundlePath(test.bundleFile), test.policyMrn, asset)
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
