// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/inventory"
)

func TestTerraformBundles(t *testing.T) {
	type TestCase struct {
		bundleFile string
		testDir    string
		policyMrn  string
		score      uint32
	}

	tests := []TestCase{
		{
			bundleFile: "./testdata/mondoo-terraform-aws-security.mql.yaml",
			testDir:    "./testdata/terraform/aws-3.xx/pass",
			policyMrn:  "//policy.api.mondoo.app/policies/mondoo-terraform-aws-security",
			score:      100,
		}, {
			bundleFile: "./testdata/mondoo-terraform-aws-security.mql.yaml",
			testDir:    "./testdata/terraform/aws-3.xx/fail",
			policyMrn:  "//policy.api.mondoo.app/policies/mondoo-terraform-aws-security",
			// NOTE: terraform-aws-security-s3-bucket-level-public-access-prohibited is not correctly implemented but needs pay the piper.
			// 3/28/2022 - Tests are passing now but not for the right reasons. We still need to revisit this query since it involves testing
			//             whether configuration was applied to a specific bucket.
			score: 0,
		}, {
			bundleFile: "./testdata/mondoo-terraform-aws-security.mql.yaml",
			testDir:    "./testdata/terraform/aws-4.xx/pass",
			policyMrn:  "//policy.api.mondoo.app/policies/mondoo-terraform-aws-security",
			score:      100,
		}, {
			bundleFile: "./testdata/mondoo-terraform-aws-security.mql.yaml",
			testDir:    "./testdata/terraform/aws-4.xx/fail",
			policyMrn:  "//policy.api.mondoo.app/policies/mondoo-terraform-aws-security",
			score:      0,
		}, {
			bundleFile: "./testdata/mondoo-terraform-gcp-security.mql.yaml",
			testDir:    "./testdata/terraform/gcp/pass",
			policyMrn:  "//policy.api.mondoo.app/policies/mondoo-terraform-gcp-security",
			score:      100,
		}, {
			bundleFile: "./testdata/mondoo-terraform-gcp-security.mql.yaml",
			testDir:    "./testdata/terraform/gcp/fail",
			policyMrn:  "//policy.api.mondoo.app/policies/mondoo-terraform-gcp-security",
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
