// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package convert

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/internal/reportfixture"
	"go.mondoo.com/cnspec/policy"
)

// A code id stands for every bundle query that compiled to it, and the queries
// resolved for the asset are the ones with a reporting job. Both halves are
// asserted here: the Linux queries of the Ubuntu fixture are reported, and the
// macOS and Windows queries that share their code are not.
func TestAssetDataResultsReportsTheResolvedQueries(t *testing.T) {
	yr, err := reportfixture.UbuntuScan()
	require.NoError(t, err)
	assetMrn := singleAssetMrn(t, yr)

	res := assetDataResults(yr, assetMrn)
	require.NotEmpty(t, res)
	assert.Contains(t, res, "//policy.api.mondoo.app/queries/mondoo-linux-mounts")
	assert.NotContains(t, res, "//policy.api.mondoo.app/queries/mondoo-macos-mounts")
	assert.NotContains(t, res, "//policy.api.mondoo.app/queries/mondoo-windows-packages")
}

// A collector job with no reporting jobs says nothing about which queries were
// resolved for the asset, so every bundle query with the code id is reported.
// Reading it as "none were resolved" instead empties the data section: the
// whole OCSF inventory event goes silent rather than losing one query. This is
// the rule cli/reporter's queryMrns applies, and the two have to agree.
func TestAssetDataResultsWithoutReportingJobs(t *testing.T) {
	yr, err := reportfixture.UbuntuScan()
	require.NoError(t, err)
	assetMrn := singleAssetMrn(t, yr)

	withJobs := assetDataResults(yr, assetMrn)
	require.NotEmpty(t, withJobs)

	for _, rp := range yr.ResolvedPolicies {
		if rp.CollectorJob != nil {
			rp.CollectorJob.ReportingJobs = map[string]*policy.ReportingJob{}
		}
	}

	res := assetDataResults(yr, assetMrn)
	assert.NotEmpty(t, res, "a collector job with no reporting jobs emptied the data section")
	assert.GreaterOrEqual(t, len(res), len(withJobs),
		"the unfiltered set cannot be smaller than the filtered one")
}

func singleAssetMrn(t *testing.T, yr *policy.ReportCollection) string {
	t.Helper()
	require.Len(t, yr.Assets, 1)
	for mrn := range yr.Assets {
		return mrn
	}
	return ""
}
