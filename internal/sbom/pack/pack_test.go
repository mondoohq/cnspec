// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package pack

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueryPack(t *testing.T) {
	bundle, err := QueryPack()
	require.NoError(t, err)
	require.NotNil(t, bundle)

	// The bundle should have the pack converted to a policy
	assert.Len(t, bundle.Policies, 1, "expected one policy converted from query pack")
	assert.Equal(t, "mondoo-sbom", bundle.Policies[0].Uid)
	assert.Equal(t, "Mondoo SBOM", bundle.Policies[0].Name)

	// The original pack should still be present
	assert.Len(t, bundle.Packs, 1, "expected original query pack to be present")
	assert.Equal(t, "mondoo-sbom", bundle.Packs[0].Uid)

	// Verify the queries are available
	policy := bundle.Policies[0]
	require.NotEmpty(t, policy.Groups, "expected policy groups")

	// Collect all query UIDs from groups
	queryUIDs := map[string]bool{}
	for _, group := range policy.Groups {
		for _, query := range group.Queries {
			queryUIDs[query.Uid] = true
		}
	}

	// Check expected queries are present
	expectedQueries := []string{
		"mondoo-sbom-asset",
		"mondoo-sbom-packages",
		"mondoo-sbom-python-packages",
		"mondoo-sbom-npm-packages",
		"mondoo-sbom-kernel-installed",
	}
	for _, uid := range expectedQueries {
		assert.True(t, queryUIDs[uid], "expected query %s to be present", uid)
	}
}
