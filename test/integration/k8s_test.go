// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/inventory"
)

func TestKubernetesBundles(t *testing.T) {
	type TestCase struct {
		bundleFile string
		testDir    string
		policyMrn  string
		score      uint32
	}

	tests := []TestCase{
		{
			bundleFile: "./testdata/mondoo-kubernetes-security.mql.yaml",
			testDir:    "./testdata/k8s/pass/pod.yaml",
			policyMrn:  "//policy.api.mondoo.app/policies/mondoo-kubernetes-security",
			score:      100,
		},
		{
			bundleFile: "./testdata/mondoo-kubernetes-security.mql.yaml",
			testDir:    "./testdata/k8s/fail/pod-nonroot.yaml",
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
