// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package policy_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/mql/providers"
)

// Providers that are neither installed nor in the binary's DefaultProviders
// fail with ProviderNotFoundError before any download is attempted, so no
// network access is needed.
func TestEnsureRequirements_ContinuesPastFailedRequirement(t *testing.T) {
	cached := providers.CachedProviders
	providers.CachedProviders = []*providers.Provider{}
	t.Cleanup(func() { providers.CachedProviders = cached })

	bundle := &policy.Bundle{
		Policies: []*policy.Policy{
			{Name: "policy-a", Require: []*policy.Requirement{{Provider: "not-a-provider-1"}}},
			{Name: "policy-b", Require: []*policy.Requirement{{Provider: "not-a-provider-2"}}},
		},
		Packs: []*policy.QueryPack{
			{Name: "pack-a", Require: []*policy.Requirement{{Provider: "not-a-provider-3"}}},
		},
	}

	err := bundle.EnsureRequirements(true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to validate policy 'policy-a': cannot find provider for provider=not-a-provider-1")
	assert.Contains(t, err.Error(), "failed to validate policy 'policy-b': cannot find provider for provider=not-a-provider-2")
	assert.Contains(t, err.Error(), "failed to validate querypack 'pack-a': cannot find provider for provider=not-a-provider-3")

	// Without auto-update, missing providers are only logged.
	assert.NoError(t, bundle.EnsureRequirements(false))
}
