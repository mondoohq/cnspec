// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package test

import (
	"context"
	"go.mondoo.com/cnquery/v9/logger"
	"go.mondoo.com/cnquery/v9/providers"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/inventory"
	"go.mondoo.com/cnspec/v9/policy"
	"go.mondoo.com/cnspec/v9/policy/scan"
	"os"
	"testing"
)

func init() {
	logger.Set("info")
}

func TestMain(m *testing.M) {
	// There seems to be a small timing issue when provider installation is close to schema update.
	// The provider is registered in the init() function to make sure it is loaded early
	providers.EnsureProvider(providers.ProviderLookup{ID: "go.mondoo.com/cnquery/v9/providers/terraform"}, true, nil)
	providers.EnsureProvider(providers.ProviderLookup{ID: "go.mondoo.com/cnquery/v9/providers/k8s"}, true, nil)

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
	var results *policy.Report

	policyFilters := []string{}
	if policyMrn != "" {
		policyFilters = append(policyFilters, policyMrn)
	}

	scanner := scan.NewLocalScanner(scan.WithRecording(providers.NullRecording{})) // TODO: fix recording
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

	reports := result.GetFull().Reports
	if len(reports) > 0 {
		for _, report := range reports {
			results = report
			break
		}
	}

	return results, err
}
